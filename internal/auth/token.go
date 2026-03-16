package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/johanviberg/zd/internal/config"
)

type Credentials struct {
	Profiles map[string]ProfileCredentials `json:"profiles"`
}

type ProfileCredentials struct {
	Method        string `json:"method"`
	Subdomain     string `json:"subdomain"`
	Email         string `json:"email,omitempty"`
	APIToken      string `json:"api_token,omitempty"`
	OAuthToken    string `json:"oauth_token,omitempty"`
	OAuthClientID string `json:"oauth_client_id,omitempty"`
}

func checkCredentialFile(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("credentials file is a symlink: %s", path)
	}
	return checkCredentialPermissions(path, info)
}

func writeCredentialsAtomically(path string, data []byte) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".credentials-*.json")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	tmpPath := tmp.Name()

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("writing temp file: %w", err)
	}
	if err := tmp.Chmod(0600); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return err
	}
	// Re-check target is not a symlink before rename (narrows TOCTOU window)
	if info, err := os.Lstat(path); err == nil && info.Mode()&os.ModeSymlink != 0 {
		os.Remove(tmpPath)
		return fmt.Errorf("credentials file is a symlink: %s", path)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return err
	}
	return nil
}

func LoadCredentials(profile string) (*ProfileCredentials, error) {
	path := config.CredentialsPath()

	if err := checkCredentialFile(path); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("credentials security check: %w", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading credentials: %w", err)
	}

	var creds Credentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return nil, fmt.Errorf("parsing credentials: %w", err)
	}

	if pc, ok := creds.Profiles[profile]; ok {
		return &pc, nil
	}
	return nil, nil
}

func SaveCredentials(profile string, pc *ProfileCredentials) error {
	path := config.CredentialsPath()
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}

	// Check existing file for symlinks/permissions (ignore if not exists)
	if err := checkCredentialFile(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("credentials security check: %w", err)
	}

	var creds Credentials
	data, err := os.ReadFile(path)
	if err == nil {
		if err := json.Unmarshal(data, &creds); err != nil {
			return fmt.Errorf("credentials file is corrupt — %s: %w", path, err)
		}
	}
	if creds.Profiles == nil {
		creds.Profiles = make(map[string]ProfileCredentials)
	}
	creds.Profiles[profile] = *pc

	out, err := json.MarshalIndent(creds, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling credentials: %w", err)
	}

	return writeCredentialsAtomically(path, out)
}

func DeleteCredentials(profile string) error {
	path := config.CredentialsPath()

	if err := checkCredentialFile(path); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("credentials security check: %w", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var creds Credentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return err
	}

	delete(creds.Profiles, profile)

	out, err := json.MarshalIndent(creds, "", "  ")
	if err != nil {
		return err
	}

	return writeCredentialsAtomically(path, out)
}

func ResolveCredentials(profile string) (*ProfileCredentials, error) {
	// Check env vars first
	if token := os.Getenv("ZENDESK_OAUTH_TOKEN"); token != "" {
		return &ProfileCredentials{
			Method:     "oauth",
			Subdomain:  os.Getenv("ZENDESK_SUBDOMAIN"),
			OAuthToken: token,
		}, nil
	}
	if email := os.Getenv("ZENDESK_EMAIL"); email != "" {
		if token := os.Getenv("ZENDESK_API_TOKEN"); token != "" {
			return &ProfileCredentials{
				Method:    "token",
				Subdomain: os.Getenv("ZENDESK_SUBDOMAIN"),
				Email:     email,
				APIToken:  token,
			}, nil
		}
	}

	return LoadCredentials(profile)
}
