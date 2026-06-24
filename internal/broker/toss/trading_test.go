package toss

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"testing"

	"github.com/byunjuneseok/s8s/internal/domain"
	"github.com/shopspring/decimal"
)

func TestBuyingPower(t *testing.T) {
	var gotAccount, gotCurrency string
	srv := apiServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != buyingPowerPath {
			t.Errorf("path = %q, want %q", r.URL.Path, buyingPowerPath)
		}
		gotAccount = r.Header.Get(accountHeader)
		gotCurrency = r.URL.Query().Get("currency")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"result":{"amount":"1500000","currency":"KRW"}}`))
	})
	defer srv.Close()

	c := newTestClient(t, srv)
	bp, err := c.BuyingPower(context.Background(), domain.Account{Seq: 7}, domain.KRW)
	if err != nil {
		t.Fatalf("BuyingPower: %v", err)
	}

	if gotAccount != strconv.Itoa(7) {
		t.Errorf("%s header = %q, want 7", accountHeader, gotAccount)
	}
	if gotCurrency != "KRW" {
		t.Errorf("currency query = %q, want KRW", gotCurrency)
	}
	if bp.Currency != domain.KRW || !bp.Amount.Equal(decimal.RequireFromString("1500000")) {
		t.Errorf("buying power = %+v, want 1500000 KRW", bp)
	}
}

func TestSellableQuantity(t *testing.T) {
	var gotAccount, gotSymbol string
	srv := apiServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != sellableQuantityPath {
			t.Errorf("path = %q, want %q", r.URL.Path, sellableQuantityPath)
		}
		gotAccount = r.Header.Get(accountHeader)
		gotSymbol = r.URL.Query().Get("symbol")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"result":{"symbol":"AAPL","quantity":"2.5"}}`))
	})
	defer srv.Close()

	c := newTestClient(t, srv)
	qty, err := c.SellableQuantity(context.Background(), domain.Account{Seq: 3}, "AAPL")
	if err != nil {
		t.Fatalf("SellableQuantity: %v", err)
	}

	if gotAccount != strconv.Itoa(3) {
		t.Errorf("%s header = %q, want 3", accountHeader, gotAccount)
	}
	if gotSymbol != "AAPL" {
		t.Errorf("symbol query = %q, want AAPL", gotSymbol)
	}
	if !qty.Equal(decimal.RequireFromString("2.5")) {
		t.Errorf("quantity = %s, want 2.5", qty)
	}
}

func TestCommission(t *testing.T) {
	var gotQuery url.Values
	srv := apiServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != commissionsPath {
			t.Errorf("path = %q, want %q", r.URL.Path, commissionsPath)
		}
		gotQuery = r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"result":{"fee":"150","rate":"0.00015","currency":"KRW"}}`))
	})
	defer srv.Close()

	c := newTestClient(t, srv)
	req := domain.OrderRequest{
		Symbol:   "005930",
		Side:     domain.Buy,
		Type:     domain.LimitOrder,
		Basis:    domain.QuantityBased,
		Quantity: decimal.RequireFromString("10"),
		Price:    domain.NewMoney(decimal.RequireFromString("70000"), domain.KRW),
	}
	comm, err := c.Commission(context.Background(), domain.Account{Seq: 1}, req)
	if err != nil {
		t.Fatalf("Commission: %v", err)
	}

	if gotQuery.Get("symbol") != "005930" {
		t.Errorf("symbol = %q, want 005930", gotQuery.Get("symbol"))
	}
	if gotQuery.Get("side") != "BUY" {
		t.Errorf("side = %q, want BUY", gotQuery.Get("side"))
	}
	if gotQuery.Get("quantity") != "10" {
		t.Errorf("quantity = %q, want 10", gotQuery.Get("quantity"))
	}
	if gotQuery.Get("price") != "70000" {
		t.Errorf("price = %q, want 70000", gotQuery.Get("price"))
	}
	if comm.Fee.Currency != domain.KRW || !comm.Fee.Amount.Equal(decimal.RequireFromString("150")) {
		t.Errorf("fee = %+v, want 150 KRW", comm.Fee)
	}
	if !comm.Rate.Equal(decimal.RequireFromString("0.00015")) {
		t.Errorf("rate = %s, want 0.00015", comm.Rate)
	}
}
