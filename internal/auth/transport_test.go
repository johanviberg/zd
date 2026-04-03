package auth

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthTransport_NilCredentials(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	transport := &AuthTransport{
		Credentials: nil,
		Base:        http.DefaultTransport,
	}

	req, err := http.NewRequest(http.MethodGet, server.URL, nil)
	require.NoError(t, err)

	resp, err := transport.RoundTrip(req)
	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "no credentials")
}

func TestAuthTransport_UnknownMethod(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	transport := &AuthTransport{
		Credentials: &ProfileCredentials{
			Method:    "saml",
			Subdomain: "testcompany",
		},
		Base: http.DefaultTransport,
	}

	req, err := http.NewRequest(http.MethodGet, server.URL, nil)
	require.NoError(t, err)

	resp, err := transport.RoundTrip(req)
	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "unknown auth method")
}

func TestAuthTransport_ProactiveRefresh(t *testing.T) {
	expired := time.Now().Add(-5 * time.Minute)
	newExpiry := time.Now().Add(1 * time.Hour)

	var refreshCalled atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer new-access-token", r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	transport := &AuthTransport{
		Credentials: &ProfileCredentials{
			Method:         "oauth",
			Subdomain:      "test",
			OAuthToken:     "expired-token",
			OAuthClientID:  "client-123",
			RefreshToken:   "refresh-tok",
			TokenExpiresAt: &expired,
		},
		Base: http.DefaultTransport,
		RefreshFunc: func(subdomain, clientID, refreshToken string) (*OAuthResult, error) {
			refreshCalled.Add(1)
			return &OAuthResult{
				AccessToken:  "new-access-token",
				RefreshToken: "new-refresh-tok",
				ExpiresAt:    &newExpiry,
			}, nil
		},
	}

	req, err := http.NewRequest(http.MethodGet, server.URL, nil)
	require.NoError(t, err)

	resp, err := transport.RoundTrip(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	assert.Equal(t, int32(1), refreshCalled.Load(), "refresh should be called exactly once")
	assert.Equal(t, "new-access-token", transport.Credentials.OAuthToken)
	assert.Equal(t, "new-refresh-tok", transport.Credentials.RefreshToken)
}

func TestAuthTransport_ReactiveRefreshOn401(t *testing.T) {
	future := time.Now().Add(1 * time.Hour)
	newExpiry := time.Now().Add(2 * time.Hour)

	var requestCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := requestCount.Add(1)
		if count == 1 {
			// First request returns 401
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		// Second request (after refresh) should have new token
		assert.Equal(t, "Bearer refreshed-token", r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	transport := &AuthTransport{
		Credentials: &ProfileCredentials{
			Method:         "oauth",
			Subdomain:      "test",
			OAuthToken:     "stale-token",
			OAuthClientID:  "client-123",
			RefreshToken:   "refresh-tok",
			TokenExpiresAt: &future, // Not expired yet, but will get 401
		},
		Base: http.DefaultTransport,
		RefreshFunc: func(subdomain, clientID, refreshToken string) (*OAuthResult, error) {
			return &OAuthResult{
				AccessToken:  "refreshed-token",
				RefreshToken: "new-refresh",
				ExpiresAt:    &newExpiry,
			}, nil
		},
	}

	req, err := http.NewRequest(http.MethodGet, server.URL, nil)
	require.NoError(t, err)

	resp, err := transport.RoundTrip(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	assert.Equal(t, int32(2), requestCount.Load(), "should have made 2 requests (original + retry)")
	assert.Equal(t, "refreshed-token", transport.Credentials.OAuthToken)
}

func TestAuthTransport_NoRefreshWithoutRefreshToken(t *testing.T) {
	expired := time.Now().Add(-5 * time.Minute)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer expired-token", r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusUnauthorized)
	}))
	t.Cleanup(server.Close)

	transport := &AuthTransport{
		Credentials: &ProfileCredentials{
			Method:         "oauth",
			Subdomain:      "test",
			OAuthToken:     "expired-token",
			TokenExpiresAt: &expired,
			// No RefreshToken
		},
		Base: http.DefaultTransport,
		RefreshFunc: func(subdomain, clientID, refreshToken string) (*OAuthResult, error) {
			t.Fatal("refresh should not be called without a refresh token")
			return nil, nil
		},
	}

	req, err := http.NewRequest(http.MethodGet, server.URL, nil)
	require.NoError(t, err)

	resp, err := transport.RoundTrip(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	resp.Body.Close()
}

func TestAuthTransport_TokenAuthUnchanged(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		assert.Contains(t, auth, "Basic ")
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	transport := &AuthTransport{
		Credentials: &ProfileCredentials{
			Method:    "token",
			Subdomain: "test",
			Email:     "agent@co.com",
			APIToken:  "api-tok",
		},
		Base: http.DefaultTransport,
	}

	req, err := http.NewRequest(http.MethodGet, server.URL, nil)
	require.NoError(t, err)

	resp, err := transport.RoundTrip(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()
}
