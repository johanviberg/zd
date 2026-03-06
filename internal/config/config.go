package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/adrg/xdg"
	"github.com/spf13/viper"
)

type Config struct {
	Subdomain     string `mapstructure:"subdomain"`
	OAuthClientID string `mapstructure:"oauth_client_id"`
	Profile       string `mapstructure:"-"`
}

var (
	subdomainRe       = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?$`)
	profileNameRe     = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]*$`)
	allowedConfigKeys = map[string]bool{
		"subdomain":       true,
		"oauth_client_id": true,
	}
)

func ValidateSubdomain(subdomain string) error {
	if !subdomainRe.MatchString(subdomain) {
		return fmt.Errorf("invalid subdomain %q: must contain only alphanumeric characters and hyphens", subdomain)
	}
	return nil
}

func ValidateProfileName(profile string) error {
	if !profileNameRe.MatchString(profile) {
		return fmt.Errorf("invalid profile name %q: must contain only alphanumeric characters, hyphens, and underscores", profile)
	}
	return nil
}

func ConfigDir() string {
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		return filepath.Join(dir, "zd")
	}
	return filepath.Join(xdg.ConfigHome, "zd")
}

func ConfigPath() string {
	return filepath.Join(ConfigDir(), "config.yaml")
}

func CredentialsPath() string {
	return filepath.Join(ConfigDir(), "credentials.json")
}

func Load(profile string) (*Config, error) {
	if profile == "" {
		profile = "default"
	}

	if err := ValidateProfileName(profile); err != nil {
		return nil, err
	}

	v := viper.New()
	v.SetConfigFile(ConfigPath())
	v.SetConfigType("yaml")

	v.SetEnvPrefix("ZENDESK")
	v.BindEnv("subdomain")

	cfg := &Config{Profile: profile}

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// No config file — continue with defaults/env
		} else if os.IsNotExist(err) {
			// File doesn't exist yet — continue with defaults/env
		} else {
			return nil, fmt.Errorf("reading config: %w", err)
		}
	}

	profileKey := fmt.Sprintf("profiles.%s", profile)
	sub := v.Sub(profileKey)
	if sub != nil {
		if err := sub.Unmarshal(cfg); err != nil {
			return nil, fmt.Errorf("parsing profile %q: %w", profile, err)
		}
	}

	// Env vars override config file
	if envSub := os.Getenv("ZENDESK_SUBDOMAIN"); envSub != "" {
		cfg.Subdomain = envSub
	}

	cfg.Profile = profile
	return cfg, nil
}

func Save(profile string, cfg *Config) error {
	dir := ConfigDir()
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}

	v := viper.New()
	v.SetConfigFile(ConfigPath())
	v.SetConfigType("yaml")

	_ = v.ReadInConfig()

	profileKey := fmt.Sprintf("profiles.%s", profile)
	v.Set(profileKey+".subdomain", cfg.Subdomain)
	if cfg.OAuthClientID != "" {
		v.Set(profileKey+".oauth_client_id", cfg.OAuthClientID)
	}

	return writeConfigAtomically(v, ConfigPath())
}

func SetValue(profile, key, value string) error {
	if err := ValidateProfileName(profile); err != nil {
		return err
	}
	if !allowedConfigKeys[key] {
		return fmt.Errorf("unknown config key %q: allowed keys are subdomain, oauth_client_id", key)
	}

	v := viper.New()
	v.SetConfigFile(ConfigPath())
	v.SetConfigType("yaml")

	_ = v.ReadInConfig()

	fullKey := fmt.Sprintf("profiles.%s.%s", profile, key)
	v.Set(fullKey, value)

	dir := ConfigDir()
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}

	return writeConfigAtomically(v, ConfigPath())
}

// writeConfigAtomically writes config to a temp file with restricted permissions,
// then atomically renames it to the target path to avoid TOCTOU race conditions.
func writeConfigAtomically(v *viper.Viper, targetPath string) error {
	dir := filepath.Dir(targetPath)
	tmp, err := os.CreateTemp(dir, ".config-*.yaml")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	tmp.Close()

	if err := v.WriteConfigAs(tmpPath); err != nil {
		os.Remove(tmpPath)
		return err
	}
	if err := os.Chmod(tmpPath, 0600); err != nil {
		os.Remove(tmpPath)
		return err
	}
	return os.Rename(tmpPath, targetPath)
}
