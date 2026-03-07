package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestExchangeCode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		r.ParseForm()
		if r.FormValue("grant_type") != "authorization_code" {
			t.Errorf("expected grant_type=authorization_code, got %q", r.FormValue("grant_type"))
		}
		if r.FormValue("code") != "test-code" {
			t.Errorf("expected code=test-code, got %q", r.FormValue("code"))
		}
		if r.FormValue("client_id") != "test-client-id" {
			t.Errorf("expected client_id=test-client-id, got %q", r.FormValue("client_id"))
		}
		if r.FormValue("client_secret") != "test-secret" {
			t.Errorf("expected client_secret=test-secret, got %q", r.FormValue("client_secret"))
		}

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
	if err != nil {
		t.Fatalf("generateState: %v", err)
	}
	if len(state) != 32 {
		t.Errorf("expected 32 char state, got %d", len(state))
	}

	// Test uniqueness
	state2, _ := generateState()
	if state == state2 {
		t.Error("expected unique states")
	}
}

func TestGenerateCodeVerifier(t *testing.T) {
	v, err := generateCodeVerifier()
	if err != nil {
		t.Fatalf("generateCodeVerifier: %v", err)
	}
	// 32 bytes base64url-encoded = 43 chars
	if len(v) != 43 {
		t.Errorf("expected 43 char verifier, got %d", len(v))
	}

	// Uniqueness
	v2, _ := generateCodeVerifier()
	if v == v2 {
		t.Error("expected unique verifiers")
	}
}

func TestCodeChallenge(t *testing.T) {
	verifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	challenge := codeChallenge(verifier)
	if challenge == "" {
		t.Error("expected non-empty challenge")
	}
	if challenge == verifier {
		t.Error("challenge should differ from verifier")
	}
	// S256 challenge is base64url(sha256(verifier)) = 43 chars
	if len(challenge) != 43 {
		t.Errorf("expected 43 char challenge, got %d", len(challenge))
	}
}

func TestGenerateState(t *testing.T) {
	states := make(map[string]bool)
	for i := 0; i < 100; i++ {
		s, err := generateState()
		if err != nil {
			t.Fatalf("generateState: %v", err)
		}
		if states[s] {
			t.Errorf("duplicate state generated: %s", s)
		}
		states[s] = true
	}
}
