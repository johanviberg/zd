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
