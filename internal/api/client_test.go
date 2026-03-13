package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/johanviberg/zd/internal/auth"
	"github.com/johanviberg/zd/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	require.NoError(t, err)
	assert.Equal(t, int64(1), result.Ticket.ID)
	assert.Equal(t, "Test", result.Ticket.Subject)
}

func TestDoJSON_NotFound(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		w.Write([]byte(`{"error":"RecordNotFound"}`))
	})

	client := testClient(t, handler)

	var result interface{}
	err := client.doJSON(context.Background(), "GET", "/api/v2/tickets/999", nil, &result)
	require.Error(t, err)

	appErr, ok := err.(*types.AppError)
	require.True(t, ok, "expected AppError, got %T", err)
	assert.Equal(t, types.ExitNotFound, appErr.ExitCode)
}

func TestDoJSON_AuthError(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		w.Write([]byte(`{"error":"Unauthorized"}`))
	})

	client := testClient(t, handler)

	var result interface{}
	err := client.doJSON(context.Background(), "GET", "/api/v2/tickets/1", nil, &result)
	require.Error(t, err)

	appErr, ok := err.(*types.AppError)
	require.True(t, ok, "expected AppError, got %T", err)
	assert.Equal(t, types.ExitAuthError, appErr.ExitCode)
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
	require.NoError(t, err)
	assert.GreaterOrEqual(t, attempts, 2, "expected at least 2 attempts")
}

func TestRetryTransport_POST_NotRetried(t *testing.T) {
	attempts := 0
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(500)
		w.Write([]byte(`{"error":"InternalError"}`))
	})

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	client := &Client{
		HTTPClient: &http.Client{
			Transport: &RetryTransport{
				Base:       server.Client().Transport,
				MaxRetries: 3,
			},
		},
		BaseURL: server.URL,
	}

	var result interface{}
	_ = client.doJSON(context.Background(), "POST", "/api/v2/tickets", nil, &result)
	assert.Equal(t, 1, attempts, "expected exactly 1 attempt for POST on 5xx")
}

func TestRetryTransport_GET_Retried(t *testing.T) {
	attempts := 0
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts <= 2 {
			w.WriteHeader(500)
			w.Write([]byte(`{"error":"InternalError"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ticket":{"id":1}}`))
	})

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

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
	require.NoError(t, err)
	assert.Equal(t, 3, attempts, "expected 3 attempts for GET on 5xx")
}

func TestSanitizeErrorBody(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{"json error", `{"error":"RecordNotFound","description":"Ticket not found"}`, "RecordNotFound: Ticket not found"},
		{"json error no desc", `{"error":"Unauthorized"}`, "Unauthorized"},
		{"raw body short", `Something went wrong`, "Something went wrong"},
		{"raw body truncated", string(make([]byte, 300)), string(make([]byte, 200)) + "…"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeErrorBody([]byte(tt.body))
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestAuthTransport_TokenAuth(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		assert.NotEmpty(t, authHeader, "missing Authorization header")
		assert.True(t, len(authHeader) >= 6 && authHeader[:6] == "Basic ", "expected Basic auth, got %q", authHeader)
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
	require.NoError(t, err)
}

func TestAuthTransport_OAuthAuth(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		assert.Equal(t, "Bearer test-oauth-token", authHeader)
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
	require.NoError(t, err)
}
