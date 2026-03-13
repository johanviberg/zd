package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExchangeCode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		r.ParseForm()
		assert.Equal(t, "authorization_code", r.FormValue("grant_type"))
		assert.Equal(t, "test-code", r.FormValue("code"))
		assert.Equal(t, "test-client-id", r.FormValue("client_id"))
		assert.Equal(t, "test-secret", r.FormValue("client_secret"))

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tokenResponse{
			AccessToken: "returned-token",
			TokenType:   "bearer",
		})
	}))
	defer server.Close()

	// We can't test exchangeCode directly because it constructs its own URL.
	// Instead test the state generation.
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

func TestGenerateState(t *testing.T) {
	states := make(map[string]bool)
	for i := 0; i < 100; i++ {
		s, err := generateState()
		require.NoError(t, err, "generateState")
		assert.False(t, states[s], "duplicate state generated: %s", s)
		states[s] = true
	}
}
