package toss

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/byunjuneseok/s8s/internal/domain"
	"github.com/shopspring/decimal"
)

func TestPrices(t *testing.T) {
	const body = `{"result":[
		{"symbol":"005930","lastPrice":"70000","currency":"KRW","timestamp":"2026-06-25T09:30:00Z"},
		{"symbol":"AAPL","lastPrice":"185.50","currency":"USD","timestamp":null}
	]}`

	var gotQuery string
	srv := apiServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != pricesPath {
			t.Errorf("path = %q, want %q", r.URL.Path, pricesPath)
		}
		gotQuery = r.URL.Query().Get("symbols")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	})
	defer srv.Close()

	c := newTestClient(t, srv)
	quotes, err := c.Prices(context.Background(), []string{"005930", "AAPL"})
	if err != nil {
		t.Fatalf("Prices: %v", err)
	}

	if gotQuery != "005930,AAPL" {
		t.Errorf("symbols query = %q, want 005930,AAPL", gotQuery)
	}
	if len(quotes) != 2 {
		t.Fatalf("got %d quotes, want 2", len(quotes))
	}

	kr := quotes[0]
	if kr.Symbol != "005930" || kr.LastPrice.Currency != domain.KRW {
		t.Errorf("KR quote = %+v", kr)
	}
	if !kr.LastPrice.Amount.Equal(decimal.RequireFromString("70000")) {
		t.Errorf("KR last price = %s, want 70000", kr.LastPrice.Amount)
	}
	wantTS := time.Date(2026, 6, 25, 9, 30, 0, 0, time.UTC)
	if kr.Timestamp == nil || !kr.Timestamp.Equal(wantTS) {
		t.Errorf("KR timestamp = %v, want %v", kr.Timestamp, wantTS)
	}

	us := quotes[1]
	if us.Symbol != "AAPL" || us.LastPrice.Currency != domain.USD {
		t.Errorf("US quote = %+v", us)
	}
	if !us.LastPrice.Amount.Equal(decimal.RequireFromString("185.50")) {
		t.Errorf("US last price = %s, want 185.50", us.LastPrice.Amount)
	}
	if us.Timestamp != nil {
		t.Errorf("US timestamp = %v, want nil", us.Timestamp)
	}
}
