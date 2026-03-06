package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_Empty(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	// Clear env vars that could interfere
	t.Setenv("ZENDESK_SUBDOMAIN", "")

	cfg, err := Load("default")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Subdomain != "" {
		t.Errorf("expected empty subdomain, got %q", cfg.Subdomain)
	}
	if cfg.Profile != "default" {
		t.Errorf("expected profile 'default', got %q", cfg.Profile)
	}
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
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Subdomain != "mycompany" {
		t.Errorf("expected subdomain 'mycompany', got %q", cfg.Subdomain)
	}
	if cfg.OAuthClientID != "client123" {
		t.Errorf("expected client ID 'client123', got %q", cfg.OAuthClientID)
	}
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
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Subdomain != "env-override" {
		t.Errorf("expected env override 'env-override', got %q", cfg.Subdomain)
	}
}

func TestSave(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	t.Setenv("ZENDESK_SUBDOMAIN", "")

	cfg := &Config{
		Subdomain:     "saved-co",
		OAuthClientID: "saved-id",
	}

	if err := Save("test", cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := Load("test")
	if err != nil {
		t.Fatalf("Load after save: %v", err)
	}
	if loaded.Subdomain != "saved-co" {
		t.Errorf("expected subdomain 'saved-co', got %q", loaded.Subdomain)
	}
}

func TestSetValue(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	t.Setenv("ZENDESK_SUBDOMAIN", "")

	if err := SetValue("default", "subdomain", "set-value"); err != nil {
		t.Fatalf("SetValue: %v", err)
	}

	cfg, err := Load("default")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Subdomain != "set-value" {
		t.Errorf("expected subdomain 'set-value', got %q", cfg.Subdomain)
	}
}
