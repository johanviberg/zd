package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

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
