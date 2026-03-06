package auth

import (
	"encoding/base64"
	"fmt"
	"net/http"
)

type AuthTransport struct {
	Credentials *ProfileCredentials
	Base        http.RoundTripper
}

func (t *AuthTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.Credentials == nil {
		return nil, fmt.Errorf("no credentials configured")
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
	return base.RoundTrip(reqClone)
}
