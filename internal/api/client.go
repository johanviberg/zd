package api

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"github.com/johanviberg/zd/internal/auth"
	"github.com/johanviberg/zd/internal/config"
	"github.com/johanviberg/zd/internal/types"
)

type Client struct {
	HTTPClient *http.Client
	BaseURL    string
	TraceID    string
}

type APIError struct {
	StatusCode int
	Body       string
	RetryAfter int
}

func (e *APIError) Error() string {
	return fmt.Sprintf("zendesk API error (status %d): %s", e.StatusCode, e.Body)
}

func NewClient(subdomain string, creds *auth.ProfileCredentials, profile, traceID string) (*Client, error) {
	if err := config.ValidateSubdomain(subdomain); err != nil {
		return nil, err
	}

	transport := buildTransport(creds, profile)

	return &Client{
		HTTPClient: &http.Client{
			Transport: transport,
			Timeout:   30 * time.Second,
		},
		BaseURL: fmt.Sprintf("https://%s.zendesk.com", subdomain),
		TraceID: traceID,
	}, nil
}

func buildTransport(creds *auth.ProfileCredentials, profile string) http.RoundTripper {
	base := http.DefaultTransport.(*http.Transport).Clone()
	tlsCfg := base.TLSClientConfig
	if tlsCfg == nil {
		tlsCfg = &tls.Config{}
	} else {
		tlsCfg = tlsCfg.Clone()
	}
	tlsCfg.MinVersion = tls.VersionTLS12
	base.TLSClientConfig = tlsCfg

	authTransport := &auth.AuthTransport{
		Credentials: creds,
		Profile:     profile,
		Base:        base,
	}

	retryTransport := &RetryTransport{
		Base:       authTransport,
		MaxRetries: 3,
	}

	return retryTransport
}

func (c *Client) do(ctx context.Context, method, path string, body io.Reader) (*http.Response, error) {
	url := c.BaseURL + path
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if c.TraceID != "" {
		req.Header.Set("X-Trace-ID", c.TraceID)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		defer resp.Body.Close()
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))

		apiErr := &APIError{
			StatusCode: resp.StatusCode,
			Body:       sanitizeErrorBody(respBody),
		}

		if resp.StatusCode == 429 {
			if ra := resp.Header.Get("Retry-After"); ra != "" {
				apiErr.RetryAfter, _ = strconv.Atoi(ra)
			}
			return nil, types.NewRetryableError(apiErr.Error(), apiErr.RetryAfter)
		}
		if resp.StatusCode == 401 || resp.StatusCode == 403 {
			return nil, types.NewAuthError(apiErr.Error())
		}
		if resp.StatusCode == 404 {
			return nil, types.NewNotFoundError(apiErr.Error())
		}
		return nil, types.NewGeneralError(apiErr.Error())
	}

	return resp, nil
}

func (c *Client) doJSON(ctx context.Context, method, path string, body io.Reader, result interface{}) error {
	resp, err := c.do(ctx, method, path, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if result != nil {
		return json.NewDecoder(io.LimitReader(resp.Body, 10<<20)).Decode(result)
	}
	return nil
}

// RetryTransport implements exponential backoff with jitter for 429/5xx.
type RetryTransport struct {
	Base       http.RoundTripper
	MaxRetries int
}

func safeMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace:
		return true
	}
	return false
}

func (t *RetryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Don't retry non-idempotent methods
	if !safeMethod(req.Method) {
		return t.Base.RoundTrip(req)
	}

	var resp *http.Response
	var err error

	for attempt := 0; attempt <= t.MaxRetries; attempt++ {
		if attempt > 0 {
			if req.GetBody != nil {
				var bodyErr error
				req.Body, bodyErr = req.GetBody()
				if bodyErr != nil {
					return nil, fmt.Errorf("retry: rewinding request body: %w", bodyErr)
				}
			}
		}

		resp, err = t.Base.RoundTrip(req)
		if err != nil {
			if resp != nil {
				resp.Body.Close()
			}
			return nil, err
		}

		if resp.StatusCode != 429 && resp.StatusCode < 500 {
			return resp, nil
		}

		if attempt == t.MaxRetries {
			return resp, nil
		}

		// Calculate backoff with cap
		wait := time.Duration(math.Pow(2, float64(attempt))) * time.Second
		jitter := time.Duration(rand.Int63n(int64(time.Second)))
		wait += jitter
		if wait > 30*time.Second {
			wait = 30 * time.Second
		}

		// Check Retry-After header for 429, capped at 120 seconds
		if resp.StatusCode == 429 {
			if ra := resp.Header.Get("Retry-After"); ra != "" {
				if seconds, err := strconv.Atoi(ra); err == nil && seconds > 0 {
					if seconds > 120 {
						seconds = 120
					}
					wait = time.Duration(seconds) * time.Second
				}
			}
		}

		io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<20))
		resp.Body.Close()

		select {
		case <-req.Context().Done():
			return nil, req.Context().Err()
		case <-time.After(wait):
		}
	}

	return resp, err
}

func sanitizeErrorBody(raw []byte) string {
	var parsed struct {
		Error       string `json:"error"`
		Description string `json:"description"`
	}
	if json.Unmarshal(raw, &parsed) == nil && parsed.Error != "" {
		if parsed.Description != "" {
			return parsed.Error + ": " + parsed.Description
		}
		return parsed.Error
	}
	s := string(raw)
	if len(s) > 200 {
		s = s[:200] + "…"
	}
	return s
}
