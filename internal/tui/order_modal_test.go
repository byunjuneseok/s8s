package tui

import (
	"strings"
	"testing"

	"github.com/byunjuneseok/s8s/internal/domain"
)

func TestBuildOrderRequestLimitQuantity(t *testing.T) {
	req, err := buildOrderRequest(orderFormValues{
		Symbol: "005930", Side: "BUY", Type: "LIMIT", Basis: "QUANTITY",
		Quantity: "10", Currency: "KRW", Price: "70000", TIF: "DAY",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.Symbol != "005930" || req.Side != domain.Buy || req.Type != domain.LimitOrder {
		t.Errorf("header fields wrong: %+v", req)
	}
	if !req.Quantity.Equal(dec("10")) {
		t.Errorf("quantity = %s, want 10", req.Quantity)
	}
	if !req.Price.Amount.Equal(dec("70000")) || req.Price.Currency != domain.KRW {
		t.Errorf("price = %v, want 70000 KRW", req.Price)
	}
	if req.TimeInForce != domain.Day {
		t.Errorf("tif = %q, want DAY", req.TimeInForce)
	}
}

func TestBuildOrderRequestMarketAmount(t *testing.T) {
	req, err := buildOrderRequest(orderFormValues{
		Symbol: "AAPL", Side: "SELL", Type: "MARKET", Basis: "AMOUNT",
		Amount: "1000", Currency: "USD", TIF: "DEFAULT",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.Basis != domain.AmountBased || !req.Amount.Amount.Equal(dec("1000")) {
		t.Errorf("amount fields wrong: %+v", req)
	}
	if req.Amount.Currency != domain.USD {
		t.Errorf("amount currency = %q, want USD", req.Amount.Currency)
	}
	if req.TimeInForce != "" {
		t.Errorf("DEFAULT tif should map to empty, got %q", req.TimeInForce)
	}
	// Market order leaves price zero — must still validate.
	if req.Type != domain.MarketOrder {
		t.Errorf("type = %q, want MARKET", req.Type)
	}
}

func TestBuildOrderRequestTIFEmptyIsDefault(t *testing.T) {
	req, err := buildOrderRequest(orderFormValues{
		Symbol: "X", Side: "BUY", Type: "MARKET", Basis: "QUANTITY", Quantity: "1", TIF: "",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.TimeInForce != "" {
		t.Errorf("empty TIF = %q, want empty", req.TimeInForce)
	}
}

func TestBuildOrderRequestErrors(t *testing.T) {
	cases := []struct {
		name string
		in   orderFormValues
	}{
		{"missing symbol", orderFormValues{Symbol: "", Side: "BUY", Type: "MARKET", Basis: "QUANTITY", Quantity: "1"}},
		{"bad quantity", orderFormValues{Symbol: "X", Side: "BUY", Type: "MARKET", Basis: "QUANTITY", Quantity: "abc"}},
		{"non-positive quantity", orderFormValues{Symbol: "X", Side: "BUY", Type: "MARKET", Basis: "QUANTITY", Quantity: "0"}},
		{"bad amount", orderFormValues{Symbol: "X", Side: "BUY", Type: "MARKET", Basis: "AMOUNT", Amount: "xyz", Currency: "KRW"}},
		{"non-positive amount", orderFormValues{Symbol: "X", Side: "BUY", Type: "MARKET", Basis: "AMOUNT", Amount: "-5", Currency: "KRW"}},
		{"bad price on limit", orderFormValues{Symbol: "X", Side: "BUY", Type: "LIMIT", Basis: "QUANTITY", Quantity: "1", Price: "oops", Currency: "KRW"}},
		{"limit needs positive price", orderFormValues{Symbol: "X", Side: "BUY", Type: "LIMIT", Basis: "QUANTITY", Quantity: "1", Price: "0", Currency: "KRW"}},
		{"invalid side", orderFormValues{Symbol: "X", Side: "HODL", Type: "MARKET", Basis: "QUANTITY", Quantity: "1"}},
		{"invalid basis", orderFormValues{Symbol: "X", Side: "BUY", Type: "MARKET", Basis: "WAT", Quantity: "1"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := buildOrderRequest(c.in); err == nil {
				t.Errorf("expected error for %s, got nil", c.name)
			}
		})
	}
}

func TestBuildOrderRequestDefaultCurrency(t *testing.T) {
	req, err := buildOrderRequest(orderFormValues{
		Symbol: "X", Side: "BUY", Type: "LIMIT", Basis: "QUANTITY", Quantity: "1", Price: "100",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.Price.Currency != domain.KRW {
		t.Errorf("default currency = %q, want KRW", req.Price.Currency)
	}
}

func TestOrderSummary(t *testing.T) {
	req := domain.OrderRequest{
		Symbol: "X", Side: domain.Buy, Type: domain.LimitOrder, Basis: domain.QuantityBased,
		Quantity: dec("10"), Price: domain.NewMoney(dec("100"), domain.KRW), TimeInForce: domain.Day,
	}
	out := orderSummary(req)
	for _, want := range []string{"BUY", "LIMIT", "quantity: 10", "price: 100 KRW", "tif: DAY"} {
		if !strings.Contains(out, want) {
			t.Errorf("summary missing %q:\n%s", want, out)
		}
	}
}

func TestOrderModalConfirmText(t *testing.T) {
	a := NewApp()
	m := a.NewOrderModal("order")
	m.Estimate = func(domain.OrderRequest) string { return "est. cost 1,000 KRW" }

	market := domain.OrderRequest{
		Symbol: "X", Side: domain.Buy, Type: domain.MarketOrder, Basis: domain.AmountBased,
		Amount: domain.NewMoney(dec("1000"), domain.KRW),
	}
	out := m.confirmText(market)
	if !strings.Contains(out, "est. cost 1,000 KRW") {
		t.Errorf("confirm text missing estimate:\n%s", out)
	}
	if !strings.Contains(out, "warning") {
		t.Errorf("market order should show a warning:\n%s", out)
	}

	limit := domain.OrderRequest{
		Symbol: "X", Side: domain.Buy, Type: domain.LimitOrder, Basis: domain.QuantityBased,
		Quantity: dec("1"), Price: domain.NewMoney(dec("100"), domain.KRW),
	}
	if strings.Contains(m.confirmText(limit), "warning") {
		t.Error("limit order must not show a market warning")
	}
}

func TestOrderModalFormValuesAndReview(t *testing.T) {
	a := NewApp()
	m := a.NewOrderModal("order")

	// formValues reads widget state without panicking.
	v := m.formValues()
	if v.Side != string(domain.Buy) {
		t.Errorf("default side = %q, want BUY", v.Side)
	}
	if v.Type != string(domain.LimitOrder) {
		t.Errorf("default type = %q, want LIMIT", v.Type)
	}
	// review with empty symbol must not panic (it shows a message modal).
	m.review()
}

func TestNewOrderModal(t *testing.T) {
	a := NewApp()
	m := a.NewOrderModal("order")
	if m == nil || m.form == nil {
		t.Fatal("NewOrderModal returned incomplete modal")
	}
}
