package toss

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type sample struct {
	Symbol string `json:"symbol"`
	Price  string `json:"price"`
}

// apiServer serves the token endpoint plus a handler for everything else.
func apiServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == tokenPath {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"access_token":"tok","token_type":"Bearer","expires_in":3600}`))
			return
		}
		handler(w, r)
	}))
}

func newTestClient(t *testing.T, srv *httptest.Server) *Client {
	t.Helper()
	return NewClient("id", "secret", WithHTTPClient(srv.Client()), WithBaseURL(srv.URL))
}

func TestGetJSONUnwrapsEnvelopeAndAuth(t *testing.T) {
	var gotAuth string
	srv := apiServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("X-RateLimit-Remaining-MARKET_DATA", "42")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"result":{"symbol":"005930","price":"70000"}}`))
	})
	defer srv.Close()

	c := newTestClient(t, srv)
	got, err := getJSON[sample](context.Background(), c, "/api/v1/prices", nil)
	if err != nil {
		t.Fatalf("getJSON: %v", err)
	}
	if got.Symbol != "005930" || got.Price != "70000" {
		t.Errorf("result = %+v, want 005930/70000", got)
	}
	if gotAuth != "Bearer tok" {
		t.Errorf("Authorization = %q, want 'Bearer tok'", gotAuth)
	}
	// Go canonicalizes header keys, so look up the canonical form.
	rlKey := http.CanonicalHeaderKey("X-RateLimit-Remaining-MARKET_DATA")
	if rl := c.RateLimits(); rl[rlKey] != "42" {
		t.Errorf("rate limits = %v, want remaining 42 captured under %q", rl, rlKey)
	}
}

func TestGetJSONAPIError(t *testing.T) {
	srv := apiServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":"rate_limited"}`))
	})
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := getJSON[sample](context.Background(), c, "/api/v1/prices", nil)
	if err == nil {
		t.Fatal("getJSON = nil error, want APIError")
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error = %v, want *APIError", err)
	}
	if apiErr.StatusCode != http.StatusTooManyRequests {
		t.Errorf("status = %d, want 429", apiErr.StatusCode)
	}
}

func TestPostJSONRoundTrip(t *testing.T) {
	srv := apiServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("content-type = %q, want application/json", ct)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"result":{"symbol":"AAPL","price":"185.5"}}`))
	})
	defer srv.Close()

	c := newTestClient(t, srv)
	got, err := postJSON[sample](context.Background(), c, "/api/v1/orders", map[string]string{"symbol": "AAPL"})
	if err != nil {
		t.Fatalf("postJSON: %v", err)
	}
	if got.Symbol != "AAPL" {
		t.Errorf("result = %+v, want AAPL", got)
	}
}
