package auth

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestSaveAndLoadCredentials(t *testing.T) {
	tmpDir := t.TempDir()

	// Override XDG config dir for testing
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	creds := &ProfileCredentials{
		Method:    "token",
		Subdomain: "testcompany",
		Email:     "test@example.com",
		APIToken:  "abc123",
	}

	if err := SaveCredentials("default", creds); err != nil {
		t.Fatalf("SaveCredentials: %v", err)
	}

	// Verify file permissions (Unix only; Windows always reports 0666)
	path := filepath.Join(tmpDir, "zd", "credentials.json")
	if runtime.GOOS != "windows" {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("stat credentials: %v", err)
		}
		perm := info.Mode().Perm()
		if perm != 0600 {
			t.Errorf("expected permissions 0600, got %o", perm)
		}
	}

	loaded, err := LoadCredentials("default")
	if err != nil {
		t.Fatalf("LoadCredentials: %v", err)
	}
	if loaded == nil {
		t.Fatal("expected credentials, got nil")
	}
	if loaded.Email != "test@example.com" {
		t.Errorf("expected email 'test@example.com', got %q", loaded.Email)
	}
	if loaded.APIToken != "abc123" {
		t.Errorf("expected token 'abc123', got %q", loaded.APIToken)
	}
}

func TestLoadCredentials_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	loaded, err := LoadCredentials("nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if loaded != nil {
		t.Errorf("expected nil, got %+v", loaded)
	}
}

func TestDeleteCredentials(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	creds := &ProfileCredentials{
		Method:    "token",
		Subdomain: "testcompany",
		Email:     "test@example.com",
		APIToken:  "abc123",
	}

	SaveCredentials("default", creds)

	if err := DeleteCredentials("default"); err != nil {
		t.Fatalf("DeleteCredentials: %v", err)
	}

	loaded, err := LoadCredentials("default")
	if err != nil {
		t.Fatalf("LoadCredentials after delete: %v", err)
	}
	if loaded != nil {
		t.Errorf("expected nil after delete, got %+v", loaded)
	}
}

func TestResolveCredentials_EnvVars(t *testing.T) {
	t.Setenv("ZENDESK_OAUTH_TOKEN", "env-oauth-token")
	t.Setenv("ZENDESK_SUBDOMAIN", "env-subdomain")

	creds, err := ResolveCredentials("default")
	if err != nil {
		t.Fatalf("ResolveCredentials: %v", err)
	}
	if creds == nil {
		t.Fatal("expected credentials from env vars")
	}
	if creds.OAuthToken != "env-oauth-token" {
		t.Errorf("expected oauth token 'env-oauth-token', got %q", creds.OAuthToken)
	}
	if creds.Subdomain != "env-subdomain" {
		t.Errorf("expected subdomain 'env-subdomain', got %q", creds.Subdomain)
	}
}

func TestResolveCredentials_APITokenEnv(t *testing.T) {
	// Clear OAuth token to test API token path
	t.Setenv("ZENDESK_OAUTH_TOKEN", "")
	t.Setenv("ZENDESK_EMAIL", "env@example.com")
	t.Setenv("ZENDESK_API_TOKEN", "env-api-token")
	t.Setenv("ZENDESK_SUBDOMAIN", "env-sub")

	creds, err := ResolveCredentials("default")
	if err != nil {
		t.Fatalf("ResolveCredentials: %v", err)
	}
	if creds == nil {
		t.Fatal("expected credentials from env vars")
	}
	if creds.Method != "token" {
		t.Errorf("expected method 'token', got %q", creds.Method)
	}
	if creds.Email != "env@example.com" {
		t.Errorf("expected email 'env@example.com', got %q", creds.Email)
	}
}

func TestLoadCredentials_SymlinkRejected(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink tests require Unix filesystem semantics")
	}
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	zdDir := filepath.Join(tmpDir, "zd")
	os.MkdirAll(zdDir, 0700)

	// Create a real credentials file
	realPath := filepath.Join(tmpDir, "real-credentials.json")
	os.WriteFile(realPath, []byte(`{"profiles":{"default":{"method":"token","subdomain":"test","email":"a@b.com","api_token":"x"}}}`), 0600)

	// Create a symlink at the credentials path
	credPath := filepath.Join(zdDir, "credentials.json")
	os.Symlink(realPath, credPath)

	_, err := LoadCredentials("default")
	if err == nil {
		t.Fatal("expected error for symlink, got nil")
	}
	if !strings.Contains(err.Error(), "symlink") {
		t.Errorf("expected symlink error, got: %v", err)
	}
}

func TestLoadCredentials_InsecurePermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission checks require Unix filesystem semantics")
	}
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	zdDir := filepath.Join(tmpDir, "zd")
	os.MkdirAll(zdDir, 0700)

	credPath := filepath.Join(zdDir, "credentials.json")
	os.WriteFile(credPath, []byte(`{"profiles":{}}`), 0644)

	_, err := LoadCredentials("default")
	if err == nil {
		t.Fatal("expected error for insecure permissions, got nil")
	}
	if !strings.Contains(err.Error(), "insecure permissions") {
		t.Errorf("expected permissions error, got: %v", err)
	}
}

func TestSaveCredentials_AtomicWrite(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	creds := &ProfileCredentials{
		Method:    "token",
		Subdomain: "testcompany",
		Email:     "test@example.com",
		APIToken:  "abc123",
	}

	if err := SaveCredentials("default", creds); err != nil {
		t.Fatalf("SaveCredentials: %v", err)
	}

	// Verify file permissions are 0600 (Unix only)
	path := filepath.Join(tmpDir, "zd", "credentials.json")
	if runtime.GOOS != "windows" {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("stat: %v", err)
		}
		if info.Mode().Perm() != 0600 {
			t.Errorf("expected permissions 0600, got %o", info.Mode().Perm())
		}
	}

	// Verify no temp files left behind
	entries, _ := os.ReadDir(filepath.Join(tmpDir, "zd"))
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".credentials-") {
			t.Errorf("temp file left behind: %s", e.Name())
		}
	}
}

func TestSaveCredentials_SymlinkRejected(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink tests require Unix filesystem semantics")
	}
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	zdDir := filepath.Join(tmpDir, "zd")
	os.MkdirAll(zdDir, 0700)

	realPath := filepath.Join(tmpDir, "real-credentials.json")
	os.WriteFile(realPath, []byte(`{"profiles":{}}`), 0600)

	credPath := filepath.Join(zdDir, "credentials.json")
	os.Symlink(realPath, credPath)

	err := SaveCredentials("default", &ProfileCredentials{Method: "token"})
	if err == nil {
		t.Fatal("expected error for symlink, got nil")
	}
	if !strings.Contains(err.Error(), "symlink") {
		t.Errorf("expected symlink error, got: %v", err)
	}
}

func TestMultipleProfiles(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	SaveCredentials("prod", &ProfileCredentials{
		Method: "token", Subdomain: "prod-co", Email: "prod@example.com", APIToken: "prod-token",
	})
	SaveCredentials("staging", &ProfileCredentials{
		Method: "token", Subdomain: "staging-co", Email: "staging@example.com", APIToken: "staging-token",
	})

	prod, _ := LoadCredentials("prod")
	staging, _ := LoadCredentials("staging")

	if prod.Subdomain != "prod-co" {
		t.Errorf("expected prod subdomain 'prod-co', got %q", prod.Subdomain)
	}
	if staging.Subdomain != "staging-co" {
		t.Errorf("expected staging subdomain 'staging-co', got %q", staging.Subdomain)
	}
}
