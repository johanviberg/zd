package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/johanviberg/zd/internal/browser"
	"github.com/johanviberg/zd/internal/config"
)

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
}

func OAuthFlow(subdomain, clientID, clientSecret, scope string) (string, error) {
	if err := config.ValidateSubdomain(subdomain); err != nil {
		return "", err
	}

	// Start local server on random port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", fmt.Errorf("starting local server: %w", err)
	}
	defer listener.Close()

	port := listener.Addr().(*net.TCPAddr).Port
	redirectURI := fmt.Sprintf("http://127.0.0.1:%d/callback", port)

	state, err := generateState()
	if err != nil {
		return "", fmt.Errorf("generating state: %w", err)
	}

	codeVerifier, err := generateCodeVerifier()
	if err != nil {
		return "", fmt.Errorf("generating PKCE verifier: %w", err)
	}

	authURL := fmt.Sprintf("https://%s.zendesk.com/oauth/authorizations/new?%s",
		subdomain,
		url.Values{
			"response_type":         {"code"},
			"client_id":             {clientID},
			"redirect_uri":          {redirectURI},
			"scope":                 {scope},
			"state":                 {state},
			"code_challenge":        {codeChallenge(codeVerifier)},
			"code_challenge_method": {"S256"},
		}.Encode(),
	)

	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("X-Content-Type-Options", "nosniff")

		if r.URL.Query().Get("state") != state {
			errCh <- fmt.Errorf("state mismatch: possible CSRF attack")
			http.Error(w, "State mismatch", http.StatusBadRequest)
			return
		}

		if errParam := r.URL.Query().Get("error"); errParam != "" {
			errCh <- fmt.Errorf("OAuth error: %s — %s", errParam, r.URL.Query().Get("error_description"))
			fmt.Fprintf(w, "<html><body><h1>Authentication Failed</h1><p>%s</p><p>You may close this window.</p></body></html>", html.EscapeString(r.URL.Query().Get("error_description")))
			return
		}

		code := r.URL.Query().Get("code")
		if code == "" {
			errCh <- fmt.Errorf("no code in callback")
			http.Error(w, "No code received", http.StatusBadRequest)
			return
		}

		codeCh <- code
		fmt.Fprint(w, "<html><body><h1>Authentication Successful</h1><p>You may close this window and return to the terminal.</p></body></html>")
	})

	server := &http.Server{
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  5 * time.Second,
	}
	go server.Serve(listener)
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.Shutdown(shutdownCtx)
	}()

	fmt.Printf("Opening browser for authorization...\n")
	fmt.Printf("If the browser doesn't open, visit:\n%s\n\n", authURL)
	browser.Open(authURL)

	var code string
	select {
	case code = <-codeCh:
	case err := <-errCh:
		return "", err
	case <-time.After(5 * time.Minute):
		return "", fmt.Errorf("OAuth flow timed out after 5 minutes")
	}

	// Exchange code for token
	token, err := exchangeCode(subdomain, clientID, clientSecret, code, redirectURI, scope, codeVerifier)
	if err != nil {
		return "", fmt.Errorf("exchanging code: %w", err)
	}

	return token, nil
}

func exchangeCode(subdomain, clientID, clientSecret, code, redirectURI, scope, codeVerifier string) (string, error) {
	tokenURL := fmt.Sprintf("https://%s.zendesk.com/oauth/tokens", subdomain)

	data := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"client_id":     {clientID},
		"client_secret": {clientSecret},
		"redirect_uri":  {redirectURI},
		"scope":         {scope},
		"code_verifier": {codeVerifier},
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Post(tokenURL, "application/x-www-form-urlencoded", strings.NewReader(data.Encode()))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("token exchange failed with status %d", resp.StatusCode)
	}

	var tokenResp tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", fmt.Errorf("parsing token response: %w", err)
	}

	if tokenResp.AccessToken == "" {
		return "", fmt.Errorf("empty access token in response")
	}

	return tokenResp.AccessToken, nil
}

func generateCodeVerifier() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func codeChallenge(verifier string) string {
	h := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(h[:])
}

func generateState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
