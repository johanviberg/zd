package auth

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

	err := SaveCredentials("default", creds)
	require.NoError(t, err, "SaveCredentials")

	// Verify file permissions (Unix only; Windows always reports 0666)
	path := filepath.Join(tmpDir, "zd", "credentials.json")
	if runtime.GOOS != "windows" {
		info, err := os.Stat(path)
		require.NoError(t, err, "stat credentials")
		assert.Equal(t, os.FileMode(0600), info.Mode().Perm())
	}

	loaded, err := LoadCredentials("default")
	require.NoError(t, err, "LoadCredentials")
	require.NotNil(t, loaded)
	assert.Equal(t, "test@example.com", loaded.Email)
	assert.Equal(t, "abc123", loaded.APIToken)
}

func TestLoadCredentials_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	loaded, err := LoadCredentials("nonexistent")
	require.NoError(t, err, "unexpected error")
	assert.Nil(t, loaded)
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

	err := DeleteCredentials("default")
	require.NoError(t, err, "DeleteCredentials")

	loaded, err := LoadCredentials("default")
	require.NoError(t, err, "LoadCredentials after delete")
	assert.Nil(t, loaded)
}

func TestResolveCredentials_EnvVars(t *testing.T) {
	t.Setenv("ZENDESK_OAUTH_TOKEN", "env-oauth-token")
	t.Setenv("ZENDESK_SUBDOMAIN", "env-subdomain")

	creds, err := ResolveCredentials("default")
	require.NoError(t, err, "ResolveCredentials")
	require.NotNil(t, creds, "expected credentials from env vars")
	assert.Equal(t, "env-oauth-token", creds.OAuthToken)
	assert.Equal(t, "env-subdomain", creds.Subdomain)
}

func TestResolveCredentials_APITokenEnv(t *testing.T) {
	// Clear OAuth token to test API token path
	t.Setenv("ZENDESK_OAUTH_TOKEN", "")
	t.Setenv("ZENDESK_EMAIL", "env@example.com")
	t.Setenv("ZENDESK_API_TOKEN", "env-api-token")
	t.Setenv("ZENDESK_SUBDOMAIN", "env-sub")

	creds, err := ResolveCredentials("default")
	require.NoError(t, err, "ResolveCredentials")
	require.NotNil(t, creds, "expected credentials from env vars")
	assert.Equal(t, "token", creds.Method)
	assert.Equal(t, "env@example.com", creds.Email)
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
	require.Error(t, err, "expected error for symlink, got nil")
	assert.Contains(t, err.Error(), "symlink")
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
	require.Error(t, err, "expected error for insecure permissions, got nil")
	assert.Contains(t, err.Error(), "insecure permissions")
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

	err := SaveCredentials("default", creds)
	require.NoError(t, err, "SaveCredentials")

	// Verify file permissions are 0600 (Unix only)
	path := filepath.Join(tmpDir, "zd", "credentials.json")
	if runtime.GOOS != "windows" {
		info, err := os.Stat(path)
		require.NoError(t, err, "stat")
		assert.Equal(t, os.FileMode(0600), info.Mode().Perm())
	}

	// Verify no temp files left behind
	entries, _ := os.ReadDir(filepath.Join(tmpDir, "zd"))
	for _, e := range entries {
		assert.False(t, strings.HasPrefix(e.Name(), ".credentials-"), "temp file left behind: %s", e.Name())
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
	require.Error(t, err, "expected error for symlink, got nil")
	assert.Contains(t, err.Error(), "symlink")
}

func TestIsTokenExpired_NilExpiry(t *testing.T) {
	pc := &ProfileCredentials{Method: "oauth", OAuthToken: "tok"}
	assert.False(t, pc.IsTokenExpired(), "nil expiry should not be considered expired")
}

func TestIsTokenExpired_FutureExpiry(t *testing.T) {
	future := time.Now().Add(10 * time.Minute)
	pc := &ProfileCredentials{Method: "oauth", OAuthToken: "tok", TokenExpiresAt: &future}
	assert.False(t, pc.IsTokenExpired(), "token with 10min remaining should not be expired")
}

func TestIsTokenExpired_Within60Seconds(t *testing.T) {
	almostExpired := time.Now().Add(30 * time.Second)
	pc := &ProfileCredentials{Method: "oauth", OAuthToken: "tok", TokenExpiresAt: &almostExpired}
	assert.True(t, pc.IsTokenExpired(), "token within 60s of expiry should be considered expired")
}

func TestIsTokenExpired_PastExpiry(t *testing.T) {
	past := time.Now().Add(-5 * time.Minute)
	pc := &ProfileCredentials{Method: "oauth", OAuthToken: "tok", TokenExpiresAt: &past}
	assert.True(t, pc.IsTokenExpired(), "token past expiry should be expired")
}

func TestSaveAndLoadCredentials_WithRefreshToken(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	expires := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	creds := &ProfileCredentials{
		Method:         "oauth",
		Subdomain:      "testcompany",
		OAuthToken:     "access-tok",
		OAuthClientID:  "client-123",
		RefreshToken:   "refresh-tok",
		TokenExpiresAt: &expires,
	}

	err := SaveCredentials("default", creds)
	require.NoError(t, err)

	loaded, err := LoadCredentials("default")
	require.NoError(t, err)
	require.NotNil(t, loaded)
	assert.Equal(t, "access-tok", loaded.OAuthToken)
	assert.Equal(t, "refresh-tok", loaded.RefreshToken)
	require.NotNil(t, loaded.TokenExpiresAt)
	assert.True(t, loaded.TokenExpiresAt.Equal(expires))
}

func TestLoadCredentials_BackwardCompat_NoRefreshFields(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	zdDir := filepath.Join(tmpDir, "zd")
	os.MkdirAll(zdDir, 0700)

	// Old-format credentials without refresh_token or token_expires_at
	old := Credentials{
		Profiles: map[string]ProfileCredentials{
			"default": {
				Method:        "oauth",
				Subdomain:     "testco",
				OAuthToken:    "old-token",
				OAuthClientID: "old-client",
			},
		},
	}
	data, _ := json.MarshalIndent(old, "", "  ")
	os.WriteFile(filepath.Join(zdDir, "credentials.json"), data, 0600)

	loaded, err := LoadCredentials("default")
	require.NoError(t, err)
	require.NotNil(t, loaded)
	assert.Equal(t, "old-token", loaded.OAuthToken)
	assert.Empty(t, loaded.RefreshToken)
	assert.Nil(t, loaded.TokenExpiresAt)
	assert.False(t, loaded.IsTokenExpired(), "legacy token without expiry should not be expired")
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

	assert.Equal(t, "prod-co", prod.Subdomain)
	assert.Equal(t, "staging-co", staging.Subdomain)
}
