package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_Empty(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	// Clear env vars that could interfere
	t.Setenv("ZENDESK_SUBDOMAIN", "")

	cfg, err := Load("default")
	require.NoError(t, err)
	assert.Equal(t, "", cfg.Subdomain)
	assert.Equal(t, "default", cfg.Profile)
}

func TestLoad_FromFile(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	t.Setenv("ZENDESK_SUBDOMAIN", "")

	dir := filepath.Join(tmpDir, "zd")
	os.MkdirAll(dir, 0700)

	configYAML := `
profiles:
  default:
    subdomain: mycompany
    oauth_client_id: "client123"
`
	os.WriteFile(filepath.Join(dir, "config.yaml"), []byte(configYAML), 0600)

	cfg, err := Load("default")
	require.NoError(t, err)
	assert.Equal(t, "mycompany", cfg.Subdomain)
	assert.Equal(t, "client123", cfg.OAuthClientID)
}

func TestLoad_EnvOverride(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	t.Setenv("ZENDESK_SUBDOMAIN", "env-override")

	dir := filepath.Join(tmpDir, "zd")
	os.MkdirAll(dir, 0700)
	os.WriteFile(filepath.Join(dir, "config.yaml"), []byte(`
profiles:
  default:
    subdomain: file-value
`), 0600)

	cfg, err := Load("default")
	require.NoError(t, err)
	assert.Equal(t, "env-override", cfg.Subdomain)
}

func TestSave(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	t.Setenv("ZENDESK_SUBDOMAIN", "")

	cfg := &Config{
		Subdomain:     "saved-co",
		OAuthClientID: "saved-id",
	}

	err := Save("test", cfg)
	require.NoError(t, err)

	loaded, err := Load("test")
	require.NoError(t, err)
	assert.Equal(t, "saved-co", loaded.Subdomain)
}

func TestSetValue(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	t.Setenv("ZENDESK_SUBDOMAIN", "")

	err := SetValue("default", "subdomain", "set-value")
	require.NoError(t, err)

	cfg, err := Load("default")
	require.NoError(t, err)
	assert.Equal(t, "set-value", cfg.Subdomain)
}

func TestValidateSubdomain(t *testing.T) {
	tests := []struct {
		name      string
		subdomain string
		wantErr   bool
	}{
		{name: "single char", subdomain: "a", wantErr: false},
		{name: "alphanumeric only", subdomain: "mycompany", wantErr: false},
		{name: "with hyphen", subdomain: "my-company", wantErr: false},
		{name: "numeric only", subdomain: "123", wantErr: false},
		{name: "mixed alphanumeric", subdomain: "a1b2c3", wantErr: false},
		{name: "empty string", subdomain: "", wantErr: true},
		{name: "leading hyphen", subdomain: "-bad", wantErr: true},
		{name: "trailing hyphen", subdomain: "bad-", wantErr: true},
		{name: "contains dot", subdomain: "bad.com", wantErr: true},
		{name: "contains underscore", subdomain: "bad_sub", wantErr: true},
		{name: "contains space", subdomain: " space", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateSubdomain(tc.subdomain)
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidateProfileName(t *testing.T) {
	tests := []struct {
		name    string
		profile string
		wantErr bool
	}{
		{name: "default", profile: "default", wantErr: false},
		{name: "with hyphen", profile: "my-profile", wantErr: false},
		{name: "with underscore", profile: "prod_env", wantErr: false},
		{name: "single char", profile: "a", wantErr: false},
		{name: "starts with digit", profile: "123abc", wantErr: false},
		{name: "empty string", profile: "", wantErr: true},
		{name: "leading hyphen", profile: "-bad", wantErr: true},
		{name: "leading underscore", profile: "_bad", wantErr: true},
		{name: "contains space", profile: "bad name", wantErr: true},
		{name: "contains dot", profile: "bad.name", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateProfileName(tc.profile)
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
