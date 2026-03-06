package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/johanviberg/zd/internal/auth"
	"github.com/johanviberg/zd/internal/types"
)

func testClient(t *testing.T, handler http.Handler) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	return &Client{
		HTTPClient: server.Client(),
		BaseURL:    server.URL,
	}
}

func TestDoJSON_Success(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ticket":{"id":1,"subject":"Test"}}`))
	})

	client := testClient(t, handler)

	var result struct {
		Ticket struct {
			ID      int64  `json:"id"`
			Subject string `json:"subject"`
		} `json:"ticket"`
	}

	err := client.doJSON(context.Background(), "GET", "/api/v2/tickets/1", nil, &result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Ticket.ID != 1 {
		t.Errorf("expected ID 1, got %d", result.Ticket.ID)
	}
	if result.Ticket.Subject != "Test" {
		t.Errorf("expected subject 'Test', got %q", result.Ticket.Subject)
	}
}

func TestDoJSON_NotFound(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		w.Write([]byte(`{"error":"RecordNotFound"}`))
	})

	client := testClient(t, handler)

	var result interface{}
	err := client.doJSON(context.Background(), "GET", "/api/v2/tickets/999", nil, &result)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	appErr, ok := err.(*types.AppError)
	if !ok {
		t.Fatalf("expected AppError, got %T", err)
	}
	if appErr.ExitCode != types.ExitNotFound {
		t.Errorf("expected exit code %d, got %d", types.ExitNotFound, appErr.ExitCode)
	}
}

func TestDoJSON_AuthError(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		w.Write([]byte(`{"error":"Unauthorized"}`))
	})

	client := testClient(t, handler)

	var result interface{}
	err := client.doJSON(context.Background(), "GET", "/api/v2/tickets/1", nil, &result)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	appErr, ok := err.(*types.AppError)
	if !ok {
		t.Fatalf("expected AppError, got %T", err)
	}
	if appErr.ExitCode != types.ExitAuthError {
		t.Errorf("expected exit code %d, got %d", types.ExitAuthError, appErr.ExitCode)
	}
}

func TestDoJSON_RateLimited(t *testing.T) {
	attempts := 0
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts <= 1 {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(429)
			w.Write([]byte(`{"error":"TooManyRequests"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ticket":{"id":1}}`))
	})

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	// Client with retry transport directly (no auth needed for test)
	client := &Client{
		HTTPClient: &http.Client{
			Transport: &RetryTransport{
				Base:       server.Client().Transport,
				MaxRetries: 3,
			},
		},
		BaseURL: server.URL,
	}

	var result struct {
		Ticket struct {
			ID int64 `json:"id"`
		} `json:"ticket"`
	}

	err := client.doJSON(context.Background(), "GET", "/api/v2/tickets/1", nil, &result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if attempts < 2 {
		t.Errorf("expected at least 2 attempts, got %d", attempts)
	}
}

func TestAuthTransport_TokenAuth(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			t.Error("missing Authorization header")
		}
		if len(authHeader) < 6 || authHeader[:6] != "Basic " {
			t.Errorf("expected Basic auth, got %q", authHeader)
		}
		w.WriteHeader(200)
		w.Write([]byte(`{}`))
	})

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	creds := &auth.ProfileCredentials{
		Method:   "token",
		Email:    "test@example.com",
		APIToken: "abc123",
	}

	client := &Client{
		HTTPClient: &http.Client{
			Transport: &auth.AuthTransport{
				Credentials: creds,
				Base:        server.Client().Transport,
			},
		},
		BaseURL: server.URL,
	}

	_, err := client.do(context.Background(), "GET", "/api/v2/tickets", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAuthTransport_OAuthAuth(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		expected := "Bearer test-oauth-token"
		if authHeader != expected {
			t.Errorf("expected %q, got %q", expected, authHeader)
		}
		w.WriteHeader(200)
		w.Write([]byte(`{}`))
	})

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	creds := &auth.ProfileCredentials{
		Method:     "oauth",
		OAuthToken: "test-oauth-token",
	}

	client := &Client{
		HTTPClient: &http.Client{
			Transport: &auth.AuthTransport{
				Credentials: creds,
				Base:        server.Client().Transport,
			},
		},
		BaseURL: server.URL,
	}

	_, err := client.do(context.Background(), "GET", "/api/v2/tickets", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
