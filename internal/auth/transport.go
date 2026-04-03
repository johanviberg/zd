package auth

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"sync"
)

// RefreshFunc is the signature for a function that refreshes an OAuth token.
type RefreshFunc func(subdomain, clientID, refreshToken string) (*OAuthResult, error)

type AuthTransport struct {
	Credentials *ProfileCredentials
	Profile     string
	Base        http.RoundTripper

	// RefreshFunc is called to refresh expired tokens. Defaults to RefreshAccessToken.
	RefreshFunc RefreshFunc

	mu sync.Mutex
}

func (t *AuthTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.Credentials == nil {
		return nil, fmt.Errorf("no credentials configured")
	}

	// Proactive refresh: if OAuth token is expired and we have a refresh token, refresh before the request
	if t.Credentials.Method == "oauth" && t.Credentials.RefreshToken != "" && t.Credentials.IsTokenExpired() {
		if err := t.tryRefresh(); err != nil {
			return nil, fmt.Errorf("token refresh failed: %w", err)
		}
	}

	reqClone := req.Clone(req.Context())

	switch t.Credentials.Method {
	case "oauth":
		reqClone.Header.Set("Authorization", "Bearer "+t.Credentials.OAuthToken)
	case "token":
		auth := fmt.Sprintf("%s/token:%s", t.Credentials.Email, t.Credentials.APIToken)
		encoded := base64.StdEncoding.EncodeToString([]byte(auth))
		reqClone.Header.Set("Authorization", "Basic "+encoded)
	default:
		return nil, fmt.Errorf("unknown auth method: %s", t.Credentials.Method)
	}

	base := t.Base
	if base == nil {
		base = http.DefaultTransport
	}

	resp, err := base.RoundTrip(reqClone)
	if err != nil {
		return nil, err
	}

	// Reactive refresh: if we got 401 and have a refresh token, try once
	if resp.StatusCode == 401 && t.Credentials.Method == "oauth" && t.Credentials.RefreshToken != "" {
		resp.Body.Close()

		if err := t.forceRefresh(); err != nil {
			return nil, fmt.Errorf("token refresh failed: %w", err)
		}

		// Retry the request with the new token
		retryReq := req.Clone(req.Context())
		retryReq.Header.Set("Authorization", "Bearer "+t.Credentials.OAuthToken)
		return base.RoundTrip(retryReq)
	}

	return resp, nil
}

// tryRefresh refreshes proactively, skipping if the token is no longer expired
// (another goroutine may have refreshed it already).
func (t *AuthTransport) tryRefresh() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Double-check: another goroutine may have already refreshed
	if !t.Credentials.IsTokenExpired() {
		return nil
	}

	return t.refreshLocked()
}

// forceRefresh refreshes unconditionally (e.g., after a 401 response).
func (t *AuthTransport) forceRefresh() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.refreshLocked()
}

func (t *AuthTransport) refreshLocked() error {
	refreshFn := t.RefreshFunc
	if refreshFn == nil {
		refreshFn = RefreshAccessToken
	}

	result, err := refreshFn(t.Credentials.Subdomain, t.Credentials.OAuthClientID, t.Credentials.RefreshToken)
	if err != nil {
		return err
	}

	t.Credentials.OAuthToken = result.AccessToken
	t.Credentials.RefreshToken = result.RefreshToken
	t.Credentials.TokenExpiresAt = result.ExpiresAt

	// Persist the new tokens so they survive process restarts
	if t.Profile != "" {
		if err := SaveCredentials(t.Profile, t.Credentials); err != nil {
			return fmt.Errorf("saving refreshed credentials: %w", err)
		}
	}

	return nil
}
