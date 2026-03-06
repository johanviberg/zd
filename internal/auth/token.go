package auth

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/johanviberg/zd/internal/config"
)

type Credentials struct {
	Profiles map[string]ProfileCredentials `json:"profiles"`
}

type ProfileCredentials struct {
	Method            string `json:"method"`
	Subdomain         string `json:"subdomain"`
	Email             string `json:"email,omitempty"`
	APIToken          string `json:"api_token,omitempty"`
	OAuthToken        string `json:"oauth_token,omitempty"`
	OAuthClientID     string `json:"oauth_client_id,omitempty"`
	OAuthClientSecret string `json:"oauth_client_secret,omitempty"`
}

func LoadCredentials(profile string) (*ProfileCredentials, error) {
	path := config.CredentialsPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
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

	return os.WriteFile(path, out, 0600)
}

func DeleteCredentials(profile string) error {
	path := config.CredentialsPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
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

	return os.WriteFile(path, out, 0600)
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
