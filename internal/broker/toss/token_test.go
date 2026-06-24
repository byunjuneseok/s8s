package toss

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

// tokenServer returns a server that issues incrementing tokens and counts hits.
func tokenServer(t *testing.T, calls *int32) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(calls, 1)
		if r.URL.Path != tokenPath {
			t.Errorf("path = %q, want %q", r.URL.Path, tokenPath)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse form: %v", err)
		}
		if got := r.FormValue("grant_type"); got != "client_credentials" {
			t.Errorf("grant_type = %q, want client_credentials", got)
		}
		if got := r.FormValue("client_id"); got != "id" {
			t.Errorf("client_id = %q, want id", got)
		}
		if got := r.FormValue("client_secret"); got != "secret" {
			t.Errorf("client_secret = %q, want secret", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"access_token":"tok-%d","token_type":"Bearer","expires_in":3600}`, n)
	}))
}

func TestTokenFetchAndCache(t *testing.T) {
	var calls int32
	srv := tokenServer(t, &calls)
	defer srv.Close()

	ts := newTokenSource(srv.Client(), srv.URL, "id", "secret")

	tok, err := ts.Token(context.Background())
	if err != nil {
		t.Fatalf("Token: %v", err)
	}
	if tok != "tok-1" {
		t.Errorf("token = %q, want tok-1", tok)
	}

	// Within expiry the cached token is reused; no new request.
	tok2, err := ts.Token(context.Background())
	if err != nil {
		t.Fatalf("Token (cached): %v", err)
	}
	if tok2 != "tok-1" {
		t.Errorf("cached token = %q, want tok-1", tok2)
	}
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Errorf("server calls = %d, want 1 (cached)", got)
	}
}

func TestTokenRefreshAfterExpiry(t *testing.T) {
	var calls int32
	srv := tokenServer(t, &calls)
	defer srv.Close()

	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	current := base
	ts := newTokenSource(srv.Client(), srv.URL, "id", "secret")
	ts.now = func() time.Time { return current }

	if _, err := ts.Token(context.Background()); err != nil {
		t.Fatalf("Token: %v", err)
	}

	// Advance past expiry (expires_in 3600s, minus skew); a new token is fetched.
	current = base.Add(2 * time.Hour)
	tok, err := ts.Token(context.Background())
	if err != nil {
		t.Fatalf("Token (after expiry): %v", err)
	}
	if tok != "tok-2" {
		t.Errorf("refreshed token = %q, want tok-2", tok)
	}
	if got := atomic.LoadInt32(&calls); got != 2 {
		t.Errorf("server calls = %d, want 2 (refreshed)", got)
	}
}

func TestTokenErrorOnNon200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = fmt.Fprint(w, `{"error":"invalid_client"}`)
	}))
	defer srv.Close()

	ts := newTokenSource(srv.Client(), srv.URL, "id", "secret")
	if _, err := ts.Token(context.Background()); err == nil {
		t.Error("Token = nil error on 401, want error")
	}
}
