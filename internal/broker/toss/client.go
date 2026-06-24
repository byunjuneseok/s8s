package toss

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

// Client is the shared HTTP layer for the Toss Open API. It injects the bearer
// token, unwraps the common response envelope, maps non-2xx responses to
// APIError, and records rate-limit headers.
type Client struct {
	httpClient *http.Client
	baseURL    string
	tokens     *tokenSource

	mu         sync.Mutex
	rateLimits map[string]string
}

// ClientOption customizes a Client.
type ClientOption func(*Client)

// WithHTTPClient sets the underlying *http.Client.
func WithHTTPClient(h *http.Client) ClientOption {
	return func(c *Client) {
		if h != nil {
			c.httpClient = h
		}
	}
}

// WithBaseURL overrides the API base URL (useful for tests).
func WithBaseURL(u string) ClientOption {
	return func(c *Client) {
		if u != "" {
			c.baseURL = strings.TrimRight(u, "/")
		}
	}
}

// NewClient builds a Client authenticating with the given client credentials.
func NewClient(clientID, clientSecret string, opts ...ClientOption) *Client {
	c := &Client{httpClient: http.DefaultClient, baseURL: DefaultBaseURL}
	for _, o := range opts {
		o(c)
	}
	c.tokens = newTokenSource(c.httpClient, c.baseURL, clientID, clientSecret)
	return c
}

// RateLimits returns a copy of the most recently observed rate-limit headers.
func (c *Client) RateLimits() map[string]string {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make(map[string]string, len(c.rateLimits))
	for k, v := range c.rateLimits {
		out[k] = v
	}
	return out
}

// APIError is returned for non-2xx responses from the API.
type APIError struct {
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("toss: api error: status %d: %s", e.StatusCode, e.Body)
}

// apiResponse is the common BFF envelope wrapping every successful payload.
type apiResponse[T any] struct {
	Result T `json:"result"`
}

func (c *Client) doRaw(ctx context.Context, method, path string, query url.Values, body []byte, contentType string) ([]byte, error) {
	tok, err := c.tokens.Token(ctx)
	if err != nil {
		return nil, err
	}

	u := c.baseURL + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, u, reader)
	if err != nil {
		return nil, fmt.Errorf("toss: build request %s %s: %w", method, path, err)
	}
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Accept", "application/json")
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("toss: request %s %s: %w", method, path, err)
	}
	defer func() { _ = resp.Body.Close() }()

	c.captureRateLimits(resp.Header)

	data, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, fmt.Errorf("toss: read response %s %s: %w", method, path, err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &APIError{StatusCode: resp.StatusCode, Body: strings.TrimSpace(string(data))}
	}
	return data, nil
}

func (c *Client) captureRateLimits(h http.Header) {
	var limits map[string]string
	for k, v := range h {
		if len(v) > 0 && strings.Contains(strings.ToLower(k), "ratelimit") {
			if limits == nil {
				limits = make(map[string]string)
			}
			limits[k] = v[0]
		}
	}
	if limits == nil {
		return
	}
	c.mu.Lock()
	c.rateLimits = limits
	c.mu.Unlock()
}

// getJSON performs a GET and unwraps the envelope's result into T.
func getJSON[T any](ctx context.Context, c *Client, path string, query url.Values) (T, error) {
	var zero T
	data, err := c.doRaw(ctx, http.MethodGet, path, query, nil, "")
	if err != nil {
		return zero, err
	}
	var env apiResponse[T]
	if err := json.Unmarshal(data, &env); err != nil {
		return zero, fmt.Errorf("toss: decode %s: %w", path, err)
	}
	return env.Result, nil
}

// postJSON performs a POST with a JSON body and unwraps the result into T.
func postJSON[T any](ctx context.Context, c *Client, path string, payload any) (T, error) {
	var zero T
	body, err := json.Marshal(payload)
	if err != nil {
		return zero, fmt.Errorf("toss: encode %s payload: %w", path, err)
	}
	data, err := c.doRaw(ctx, http.MethodPost, path, nil, body, "application/json")
	if err != nil {
		return zero, err
	}
	var env apiResponse[T]
	if err := json.Unmarshal(data, &env); err != nil {
		return zero, fmt.Errorf("toss: decode %s: %w", path, err)
	}
	return env.Result, nil
}
