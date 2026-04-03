package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPostTokenRequest_ConfidentialClient(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		r.ParseForm()
		assert.Equal(t, "authorization_code", r.FormValue("grant_type"))
		assert.Equal(t, "test-code", r.FormValue("code"))
		assert.Equal(t, "test-client-id", r.FormValue("client_id"))
		assert.Equal(t, "test-secret", r.FormValue("client_secret"), "confidential client should include client_secret")
		assert.NotEmpty(t, r.FormValue("code_verifier"))

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tokenResponse{
			AccessToken:  "returned-token",
			TokenType:    "bearer",
			RefreshToken: "refresh-abc",
			ExpiresIn:    3600,
		})
	}))
	t.Cleanup(server.Close)

	result, err := postTokenRequest(server.URL, url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {"test-code"},
		"client_id":     {"test-client-id"},
		"client_secret": {"test-secret"},
		"code_verifier": {"verifier123"},
	})
	require.NoError(t, err)
	assert.Equal(t, "returned-token", result.AccessToken)
	assert.Equal(t, "refresh-abc", result.RefreshToken)
	assert.NotNil(t, result.ExpiresAt)
}

func TestPostTokenRequest_PublicClient(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		assert.Empty(t, r.FormValue("client_secret"), "public client should NOT include client_secret")
		assert.Equal(t, "test-client-id", r.FormValue("client_id"))

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tokenResponse{
			AccessToken:  "public-token",
			TokenType:    "bearer",
			RefreshToken: "public-refresh",
			ExpiresIn:    7200,
		})
	}))
	t.Cleanup(server.Close)

	// Simulate what exchangeCode does for a public client (no secret)
	data := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {"test-code"},
		"client_id":     {"test-client-id"},
		"code_verifier": {"verifier123"},
	}
	// No client_secret set — this is the public client path

	result, err := postTokenRequest(server.URL, data)
	require.NoError(t, err)
	assert.Equal(t, "public-token", result.AccessToken)
	assert.Equal(t, "public-refresh", result.RefreshToken)
}

func TestPostTokenRequest_NoRefreshToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tokenResponse{
			AccessToken: "access-only",
			TokenType:   "bearer",
		})
	}))
	t.Cleanup(server.Close)

	result, err := postTokenRequest(server.URL, url.Values{"grant_type": {"authorization_code"}})
	require.NoError(t, err)
	assert.Equal(t, "access-only", result.AccessToken)
	assert.Empty(t, result.RefreshToken)
	assert.Nil(t, result.ExpiresAt)
}

func TestPostTokenRequest_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	t.Cleanup(server.Close)

	_, err := postTokenRequest(server.URL, url.Values{"grant_type": {"authorization_code"}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "status 400")
}

func TestRefreshAccessToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		assert.Equal(t, "refresh_token", r.FormValue("grant_type"))
		assert.Equal(t, "old-refresh", r.FormValue("refresh_token"))
		assert.Equal(t, "my-client", r.FormValue("client_id"))

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tokenResponse{
			AccessToken:  "new-access",
			TokenType:    "bearer",
			RefreshToken: "new-refresh",
			ExpiresIn:    3600,
		})
	}))
	t.Cleanup(server.Close)

	// Override the URL by calling postTokenRequest directly since RefreshAccessToken
	// builds the URL from subdomain. We test postTokenRequest for the actual HTTP behavior,
	// and test RefreshAccessToken's URL construction separately.
	result, err := postTokenRequest(server.URL, url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {"old-refresh"},
		"client_id":     {"my-client"},
	})
	require.NoError(t, err)
	assert.Equal(t, "new-access", result.AccessToken)
	assert.Equal(t, "new-refresh", result.RefreshToken)
	assert.NotNil(t, result.ExpiresAt)
}

func TestGenerateState(t *testing.T) {
	state, err := generateState()
	require.NoError(t, err, "generateState")
	assert.Len(t, state, 32)

	// Test uniqueness
	state2, _ := generateState()
	assert.NotEqual(t, state, state2, "expected unique states")
}

func TestGenerateCodeVerifier(t *testing.T) {
	v, err := generateCodeVerifier()
	require.NoError(t, err, "generateCodeVerifier")
	// 32 bytes base64url-encoded = 43 chars
	assert.Len(t, v, 43)

	// Uniqueness
	v2, _ := generateCodeVerifier()
	assert.NotEqual(t, v, v2, "expected unique verifiers")
}

func TestCodeChallenge(t *testing.T) {
	verifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	challenge := codeChallenge(verifier)
	assert.NotEmpty(t, challenge, "expected non-empty challenge")
	assert.NotEqual(t, verifier, challenge, "challenge should differ from verifier")
	// S256 challenge is base64url(sha256(verifier)) = 43 chars
	assert.Len(t, challenge, 43)
}

func TestGenerateState_Uniqueness(t *testing.T) {
	states := make(map[string]bool)
	for i := 0; i < 100; i++ {
		s, err := generateState()
		require.NoError(t, err, "generateState")
		assert.False(t, states[s], "duplicate state generated: %s", s)
		states[s] = true
	}
}
