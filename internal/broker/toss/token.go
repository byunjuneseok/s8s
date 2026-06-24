// Package toss implements the broker.Broker interface against the Toss
// Securities Open API.
package toss

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// DefaultBaseURL is the Toss Securities Open API base URL.
const DefaultBaseURL = "https://openapi.tossinvest.com"

const tokenPath = "/oauth2/token"

// refreshSkew is how long before the reported expiry a token is refreshed, to
// avoid using a token that expires mid-request.
const refreshSkew = 30 * time.Second

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

// tokenSource issues and caches an OAuth2 client-credentials access token,
// refreshing it shortly before expiry. It is safe for concurrent use.
type tokenSource struct {
	httpClient   *http.Client
	baseURL      string
	clientID     string
	clientSecret string
	now          func() time.Time

	mu     sync.Mutex
	token  string
	expiry time.Time
}

func newTokenSource(httpClient *http.Client, baseURL, clientID, clientSecret string) *tokenSource {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &tokenSource{
		httpClient:   httpClient,
		baseURL:      strings.TrimRight(baseURL, "/"),
		clientID:     clientID,
		clientSecret: clientSecret,
		now:          time.Now,
	}
}

// Token returns a valid access token, fetching or refreshing it as needed.
func (s *tokenSource) Token(ctx context.Context) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.token != "" && s.now().Before(s.expiry) {
		return s.token, nil
	}
	if err := s.fetch(ctx); err != nil {
		return "", err
	}
	return s.token, nil
}

func (s *tokenSource) fetch(ctx context.Context) error {
	form := url.Values{
		"grant_type":    {"client_credentials"},
		"client_id":     {s.clientID},
		"client_secret": {s.clientSecret},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.baseURL+tokenPath, strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("toss: build token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("toss: token request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return fmt.Errorf("toss: read token response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("toss: token request failed: status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var tr tokenResponse
	if err := json.Unmarshal(body, &tr); err != nil {
		return fmt.Errorf("toss: decode token response: %w", err)
	}
	if tr.AccessToken == "" {
		return fmt.Errorf("toss: token response missing access_token")
	}

	s.token = tr.AccessToken
	s.expiry = s.now().Add(time.Duration(tr.ExpiresIn)*time.Second - refreshSkew)
	return nil
}
