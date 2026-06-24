package toss

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/byunjuneseok/s8s/internal/domain"
	"github.com/shopspring/decimal"
)

func TestPlaceOrderQuantityBased(t *testing.T) {
	var gotAccount string
	var gotBody placeOrderPayload
	srv := apiServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != ordersPath {
			t.Errorf("path = %q, want %q", r.URL.Path, ordersPath)
		}
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		gotAccount = r.Header.Get(accountHeader)
		data, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(data, &gotBody); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"result":{
			"orderId":"ord-1","clientOrderId":"cli-1","symbol":"005930","side":"BUY",
			"orderType":"LIMIT","status":"OPEN","quantity":"10","filledQuantity":"0",
			"price":"70000","currency":"KRW","createdAt":"2026-06-25T09:00:00Z"
		}}`))
	})
	defer srv.Close()

	c := newTestClient(t, srv)
	req := domain.OrderRequest{
		Symbol:        "005930",
		Side:          domain.Buy,
		Type:          domain.LimitOrder,
		TimeInForce:   domain.Day,
		Basis:         domain.QuantityBased,
		Quantity:      decimal.RequireFromString("10"),
		Price:         domain.NewMoney(decimal.RequireFromString("70000"), domain.KRW),
		ClientOrderID: "cli-1",
	}
	ord, err := c.PlaceOrder(context.Background(), domain.Account{Seq: 7}, req)
	if err != nil {
		t.Fatalf("PlaceOrder: %v", err)
	}

	if gotAccount != strconv.Itoa(7) {
		t.Errorf("%s header = %q, want 7", accountHeader, gotAccount)
	}
	if gotBody.Quantity == nil || *gotBody.Quantity != "10" {
		t.Errorf("body quantity = %v, want 10", gotBody.Quantity)
	}
	if gotBody.OrderAmount != nil {
		t.Errorf("body orderAmount = %v, want nil for quantity-based", gotBody.OrderAmount)
	}
	if gotBody.Price == nil || *gotBody.Price != "70000" {
		t.Errorf("body price = %v, want 70000", gotBody.Price)
	}
	if gotBody.TimeInForce != "DAY" {
		t.Errorf("body timeInForce = %q, want DAY", gotBody.TimeInForce)
	}
	if gotBody.IdempotencyKey != "cli-1" {
		t.Errorf("idempotencyKey = %q, want cli-1 (reuses ClientOrderID)", gotBody.IdempotencyKey)
	}

	if ord.ID != "ord-1" || ord.Status != domain.StatusOpen || ord.Side != domain.Buy {
		t.Errorf("order = %+v", ord)
	}
	if !ord.Quantity.Equal(decimal.RequireFromString("10")) || !ord.FilledQuantity.IsZero() {
		t.Errorf("order qty/filled = %s/%s", ord.Quantity, ord.FilledQuantity)
	}
	if ord.Price.Currency != domain.KRW || !ord.Price.Amount.Equal(decimal.RequireFromString("70000")) {
		t.Errorf("order price = %+v", ord.Price)
	}
	if ord.CreatedAt == nil {
		t.Error("createdAt = nil, want a time")
	}
}

func TestPlaceOrderAmountBasedAutoKey(t *testing.T) {
	var gotBody placeOrderPayload
	srv := apiServer(t, func(w http.ResponseWriter, r *http.Request) {
		data, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(data, &gotBody); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"result":{
			"orderId":"ord-2","symbol":"AAPL","side":"BUY","orderType":"MARKET",
			"status":"PENDING","quantity":"0","filledQuantity":"0","currency":"USD"
		}}`))
	})
	defer srv.Close()

	c := newTestClient(t, srv)
	req := domain.OrderRequest{
		Symbol: "AAPL",
		Side:   domain.Buy,
		Type:   domain.MarketOrder,
		Basis:  domain.AmountBased,
		Amount: domain.NewMoney(decimal.RequireFromString("500"), domain.USD),
	}
	ord, err := c.PlaceOrder(context.Background(), domain.Account{Seq: 2}, req)
	if err != nil {
		t.Fatalf("PlaceOrder: %v", err)
	}

	if gotBody.OrderAmount == nil || *gotBody.OrderAmount != "500" {
		t.Errorf("body orderAmount = %v, want 500", gotBody.OrderAmount)
	}
	if gotBody.Quantity != nil {
		t.Errorf("body quantity = %v, want nil for amount-based", gotBody.Quantity)
	}
	if gotBody.Price != nil {
		t.Errorf("body price = %v, want nil for market order", gotBody.Price)
	}
	if gotBody.TimeInForce != "" {
		t.Errorf("body timeInForce = %q, want empty (omitted)", gotBody.TimeInForce)
	}
	// Auto-generated 16-byte hex idempotency key.
	if len(gotBody.IdempotencyKey) != 32 {
		t.Errorf("idempotencyKey = %q (len %d), want 32-char hex", gotBody.IdempotencyKey, len(gotBody.IdempotencyKey))
	}

	if ord.ID != "ord-2" || ord.Status != domain.StatusPending {
		t.Errorf("order = %+v", ord)
	}
	if ord.Price.Currency != domain.USD || !ord.Price.Amount.IsZero() {
		t.Errorf("order price = %+v, want zero USD", ord.Price)
	}
	if ord.CreatedAt != nil {
		t.Errorf("createdAt = %v, want nil (omitted in response)", ord.CreatedAt)
	}
}

func TestPlaceOrderValidationError(t *testing.T) {
	srv := apiServer(t, func(http.ResponseWriter, *http.Request) {
		t.Error("server should not be called for an invalid request")
	})
	defer srv.Close()

	c := newTestClient(t, srv)
	// Missing symbol fails OrderRequest.Validate before any HTTP call.
	_, err := c.PlaceOrder(context.Background(), domain.Account{Seq: 1}, domain.OrderRequest{
		Side:     domain.Buy,
		Type:     domain.MarketOrder,
		Basis:    domain.QuantityBased,
		Quantity: decimal.RequireFromString("1"),
	})
	if err == nil {
		t.Fatal("PlaceOrder = nil error, want validation error")
	}
}

func TestModifyOrder(t *testing.T) {
	var gotBody modifyOrderPayload
	srv := apiServer(t, func(w http.ResponseWriter, r *http.Request) {
		wantPath := ordersPath + "/ord-1/modify"
		if r.URL.Path != wantPath {
			t.Errorf("path = %q, want %q", r.URL.Path, wantPath)
		}
		data, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(data, &gotBody); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"result":{
			"orderId":"ord-1","symbol":"005930","side":"BUY","orderType":"LIMIT",
			"status":"OPEN","quantity":"5","filledQuantity":"0","price":"71000","currency":"KRW"
		}}`))
	})
	defer srv.Close()

	c := newTestClient(t, srv)
	ord, err := c.ModifyOrder(context.Background(), domain.Account{Seq: 1}, "ord-1", domain.OrderModification{
		Price:    domain.NewMoney(decimal.RequireFromString("71000"), domain.KRW),
		Quantity: decimal.RequireFromString("5"),
	})
	if err != nil {
		t.Fatalf("ModifyOrder: %v", err)
	}

	if gotBody.Price != "71000" || gotBody.Quantity != "5" {
		t.Errorf("body = %+v, want price 71000 / quantity 5", gotBody)
	}
	if !ord.Quantity.Equal(decimal.RequireFromString("5")) {
		t.Errorf("order quantity = %s, want 5", ord.Quantity)
	}
}

func TestCancelOrder(t *testing.T) {
	srv := apiServer(t, func(w http.ResponseWriter, r *http.Request) {
		wantPath := ordersPath + "/ord-1/cancel"
		if r.URL.Path != wantPath {
			t.Errorf("path = %q, want %q", r.URL.Path, wantPath)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"result":{
			"orderId":"ord-1","symbol":"005930","side":"BUY","orderType":"LIMIT",
			"status":"CANCELED","quantity":"10","filledQuantity":"0","price":"70000","currency":"KRW"
		}}`))
	})
	defer srv.Close()

	c := newTestClient(t, srv)
	ord, err := c.CancelOrder(context.Background(), domain.Account{Seq: 1}, "ord-1")
	if err != nil {
		t.Fatalf("CancelOrder: %v", err)
	}
	if ord.Status != domain.StatusCanceled {
		t.Errorf("status = %s, want CANCELED", ord.Status)
	}
}

func TestOrders(t *testing.T) {
	var gotAccount string
	srv := apiServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != ordersPath {
			t.Errorf("path = %q, want %q", r.URL.Path, ordersPath)
		}
		gotAccount = r.Header.Get(accountHeader)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"result":[
			{"orderId":"ord-1","symbol":"005930","side":"BUY","orderType":"LIMIT",
			 "status":"PARTIALLY_FILLED","quantity":"10","filledQuantity":"4","price":"70000","currency":"KRW"},
			{"orderId":"ord-2","symbol":"AAPL","side":"SELL","orderType":"MARKET",
			 "status":"FILLED","quantity":"2","filledQuantity":"2","price":"185.50","currency":"USD"}
		]}`))
	})
	defer srv.Close()

	c := newTestClient(t, srv)
	orders, err := c.Orders(context.Background(), domain.Account{Seq: 9})
	if err != nil {
		t.Fatalf("Orders: %v", err)
	}

	if gotAccount != strconv.Itoa(9) {
		t.Errorf("%s header = %q, want 9", accountHeader, gotAccount)
	}
	if len(orders) != 2 {
		t.Fatalf("got %d orders, want 2", len(orders))
	}
	if orders[0].Status != domain.StatusPartiallyFilled || !orders[0].FilledQuantity.Equal(decimal.RequireFromString("4")) {
		t.Errorf("order[0] = %+v", orders[0])
	}
	if orders[1].Side != domain.Sell || orders[1].Status != domain.StatusFilled {
		t.Errorf("order[1] = %+v", orders[1])
	}
}

func TestOrderDetail(t *testing.T) {
	srv := apiServer(t, func(w http.ResponseWriter, r *http.Request) {
		wantPath := ordersPath + "/ord-1"
		if r.URL.Path != wantPath {
			t.Errorf("path = %q, want %q", r.URL.Path, wantPath)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"result":{
			"orderId":"ord-1","symbol":"005930","side":"BUY","orderType":"LIMIT",
			"status":"REJECTED","quantity":"10","filledQuantity":"0","price":"70000","currency":"KRW"
		}}`))
	})
	defer srv.Close()

	c := newTestClient(t, srv)
	ord, err := c.OrderDetail(context.Background(), domain.Account{Seq: 1}, "ord-1")
	if err != nil {
		t.Fatalf("OrderDetail: %v", err)
	}
	if ord.ID != "ord-1" || ord.Status != domain.StatusRejected {
		t.Errorf("order = %+v", ord)
	}
}

func TestNewIdempotencyKeyUnique(t *testing.T) {
	a, err := newIdempotencyKey()
	if err != nil {
		t.Fatalf("newIdempotencyKey: %v", err)
	}
	b, err := newIdempotencyKey()
	if err != nil {
		t.Fatalf("newIdempotencyKey: %v", err)
	}
	if a == b {
		t.Errorf("keys are not unique: %q == %q", a, b)
	}
	if len(a) != 32 || strings.TrimLeft(a, "0123456789abcdef") != "" {
		t.Errorf("key = %q, want 32-char lowercase hex", a)
	}
}
