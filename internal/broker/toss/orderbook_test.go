package toss

import (
	"context"
	"net/http"
	"testing"

	"github.com/byunjuneseok/s8s/internal/domain"
	"github.com/shopspring/decimal"
)

func TestOrderbook(t *testing.T) {
	const body = `{"result":{
		"asks":[
			{"price":"70100","quantity":"5"},
			{"price":"70200","quantity":"10"},
			{"price":"70300","quantity":"15"}
		],
		"bids":[
			{"price":"70000","quantity":"8"},
			{"price":"69900","quantity":"12"}
		],
		"currency":"KRW",
		"timestamp":null
	}}`

	var gotSymbol string
	srv := apiServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != orderbookPath {
			t.Errorf("path = %q, want %q", r.URL.Path, orderbookPath)
		}
		gotSymbol = r.URL.Query().Get("symbol")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	})
	defer srv.Close()

	c := newTestClient(t, srv)
	ob, err := c.Orderbook(context.Background(), "005930")
	if err != nil {
		t.Fatalf("Orderbook: %v", err)
	}

	if gotSymbol != "005930" {
		t.Errorf("symbol query = %q, want 005930", gotSymbol)
	}
	if ob.Symbol != "005930" {
		t.Errorf("orderbook symbol = %q, want 005930", ob.Symbol)
	}
	if ob.Timestamp != nil {
		t.Errorf("timestamp = %v, want nil", ob.Timestamp)
	}

	if len(ob.Asks) != 3 || len(ob.Bids) != 2 {
		t.Fatalf("got %d asks / %d bids, want 3 / 2", len(ob.Asks), len(ob.Bids))
	}

	// Asks preserve low -> high ordering.
	wantAsks := []string{"70100", "70200", "70300"}
	for i, want := range wantAsks {
		if !ob.Asks[i].Price.Amount.Equal(decimal.RequireFromString(want)) {
			t.Errorf("ask[%d] price = %s, want %s", i, ob.Asks[i].Price.Amount, want)
		}
		if ob.Asks[i].Price.Currency != domain.KRW {
			t.Errorf("ask[%d] currency = %s, want KRW", i, ob.Asks[i].Price.Currency)
		}
	}
	if !ob.Asks[0].Quantity.Equal(decimal.RequireFromString("5")) {
		t.Errorf("ask[0] quantity = %s, want 5", ob.Asks[0].Quantity)
	}

	// Bids preserve high -> low ordering.
	wantBids := []string{"70000", "69900"}
	for i, want := range wantBids {
		if !ob.Bids[i].Price.Amount.Equal(decimal.RequireFromString(want)) {
			t.Errorf("bid[%d] price = %s, want %s", i, ob.Bids[i].Price.Amount, want)
		}
	}
}
