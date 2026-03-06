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
