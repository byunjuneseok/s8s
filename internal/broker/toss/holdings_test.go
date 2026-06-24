package toss

import (
	"context"
	"net/http"
	"strconv"
	"testing"

	"github.com/byunjuneseok/s8s/internal/domain"
	"github.com/shopspring/decimal"
)

const holdingsBody = `{"result":{
	"totalPurchaseAmount":{"krw":"1000000","usd":"500.00"},
	"marketValue":{"amount":{"krw":"1100000","usd":"550.00"}},
	"profitLoss":{"amount":{"krw":"100000","usd":"50.00"}},
	"dailyProfitLoss":{"amount":{"krw":"5000","usd":"2.00"}},
	"items":[
		{"symbol":"005930","name":"삼성전자","marketCountry":"KR","currency":"KRW",
		 "quantity":"10","lastPrice":"70000","averagePurchasePrice":"65000",
		 "marketValue":{"amount":"700000"},
		 "profitLoss":{"amount":"50000","rate":"0.0769"},
		 "dailyProfitLoss":{"amount":"3000","rate":"0.0043"}},
		{"symbol":"AAPL","name":"Apple","marketCountry":"US","currency":"USD",
		 "quantity":"2.5","lastPrice":"185.50","averagePurchasePrice":"180.00",
		 "marketValue":{"amount":"463.75"},
		 "profitLoss":{"amount":"13.75","rate":"0.0305"},
		 "dailyProfitLoss":{"amount":"1.20","rate":"0.0026"}}
	]
}}`

func TestHoldings(t *testing.T) {
	var gotAccountHeader string
	srv := apiServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != holdingsPath {
			t.Errorf("path = %q, want %q", r.URL.Path, holdingsPath)
		}
		gotAccountHeader = r.Header.Get(accountHeader)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(holdingsBody))
	})
	defer srv.Close()

	c := newTestClient(t, srv)
	ov, err := c.Holdings(context.Background(), domain.Account{Seq: 7})
	if err != nil {
		t.Fatalf("Holdings: %v", err)
	}

	if gotAccountHeader != strconv.Itoa(7) {
		t.Errorf("%s header = %q, want 7", accountHeader, gotAccountHeader)
	}

	// Two currency totals (KRW + USD).
	if len(ov.Totals) != 2 {
		t.Fatalf("got %d currency totals, want 2", len(ov.Totals))
	}
	krw := ov.Totals[0]
	if krw.Currency != domain.KRW || !krw.PurchaseAmount.Amount.Equal(decimal.RequireFromString("1000000")) {
		t.Errorf("KRW totals = %+v", krw)
	}
	// Derived total rate = 100000 / 1000000 = 0.1
	if !krw.PnL.Total.Rate.Equal(decimal.RequireFromString("0.1")) {
		t.Errorf("KRW total rate = %s, want 0.1", krw.PnL.Total.Rate)
	}
	if ov.Totals[1].Currency != domain.USD {
		t.Errorf("second total currency = %s, want USD", ov.Totals[1].Currency)
	}

	// Two positions; check the fractional US one.
	if len(ov.Positions) != 2 {
		t.Fatalf("got %d positions, want 2", len(ov.Positions))
	}
	aapl := ov.Positions[1]
	if aapl.Symbol != "AAPL" || aapl.Market != domain.MarketUS || aapl.Currency != domain.USD {
		t.Errorf("AAPL position = %+v", aapl)
	}
	if !aapl.Quantity.Equal(decimal.RequireFromString("2.5")) {
		t.Errorf("AAPL quantity = %s, want 2.5", aapl.Quantity)
	}
	if !aapl.PnL.Total.Rate.Equal(decimal.RequireFromString("0.0305")) {
		t.Errorf("AAPL total rate = %s, want 0.0305", aapl.PnL.Total.Rate)
	}
	if aapl.LastPrice.Currency != domain.USD {
		t.Errorf("AAPL last price currency = %s, want USD", aapl.LastPrice.Currency)
	}
}

func TestHoldingsKRWOnly(t *testing.T) {
	body := `{"result":{
		"totalPurchaseAmount":{"krw":"1000000","usd":null},
		"marketValue":{"amount":{"krw":"1000000","usd":null}},
		"profitLoss":{"amount":{"krw":"0","usd":null}},
		"dailyProfitLoss":{"amount":{"krw":"0","usd":null}},
		"items":[]
	}}`
	srv := apiServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	})
	defer srv.Close()

	c := newTestClient(t, srv)
	ov, err := c.Holdings(context.Background(), domain.Account{Seq: 1})
	if err != nil {
		t.Fatalf("Holdings: %v", err)
	}
	if len(ov.Totals) != 1 || ov.Totals[0].Currency != domain.KRW {
		t.Errorf("totals = %+v, want single KRW", ov.Totals)
	}
	if len(ov.Positions) != 0 {
		t.Errorf("positions = %d, want 0", len(ov.Positions))
	}
}
