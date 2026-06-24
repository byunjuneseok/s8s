package main

import (
	"context"
	"testing"

	"github.com/byunjuneseok/s8s/internal/broker"
	"github.com/byunjuneseok/s8s/internal/broker/brokertest"
	"github.com/byunjuneseok/s8s/internal/config"
	"github.com/byunjuneseok/s8s/internal/domain"
	"github.com/byunjuneseok/s8s/internal/session"
	"github.com/shopspring/decimal"
)

// fakeBroker is a Stub whose Accounts returns a fixed list.
type fakeBroker struct {
	brokertest.Stub
	accounts []domain.Account
}

func (f fakeBroker) Accounts(context.Context) ([]domain.Account, error) { return f.accounts, nil }

func newTestManager(t *testing.T, readOnly bool, accts []domain.Account) *session.Manager {
	t.Helper()
	cfg := &config.Config{
		CurrentContext: "c",
		Contexts:       []config.Context{{Name: "c", Broker: "b", ReadOnly: readOnly}},
		Brokers:        []config.BrokerConfig{{Name: "b", Type: "toss", Credentials: "cr"}},
		Credentials:    []config.Credential{{Name: "cr", ClientID: "x", ClientSecret: "y"}},
	}
	mgr, err := session.NewManager(cfg, func(config.Resolved) (broker.Broker, error) {
		return fakeBroker{accounts: accts}, nil
	})
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	return mgr
}

func TestManagerLabel(t *testing.T) {
	rw := newTestManager(t, false, nil)
	if got := managerLabel(rw); got != "c" {
		t.Errorf("label = %q, want c", got)
	}
	ro := newTestManager(t, true, nil)
	if got := managerLabel(ro); got != "c (read-only)" {
		t.Errorf("read-only label = %q", got)
	}

	empty, err := session.NewManager(&config.Config{}, func(config.Resolved) (broker.Broker, error) {
		return brokertest.Stub{}, nil
	})
	if err != nil {
		t.Fatalf("NewManager empty: %v", err)
	}
	if got := managerLabel(empty); got == "" || got == "c" {
		t.Errorf("no-context label = %q, want a configure hint", got)
	}
}

func TestLiveAccountsReorder(t *testing.T) {
	accts := []domain.Account{{No: "A", Seq: 1}, {No: "B", Seq: 2}, {No: "C", Seq: 3}}
	lv := &live{mgr: newTestManager(t, false, accts)}

	// No active selection: order is preserved.
	got, err := lv.Accounts(context.Background())
	if err != nil {
		t.Fatalf("Accounts: %v", err)
	}
	if got[0].Seq != 1 {
		t.Errorf("default first = %d, want 1", got[0].Seq)
	}

	// Selecting seq 2 moves it to the front, others keep their relative order.
	lv.SetActiveSeq(2)
	got, err = lv.Accounts(context.Background())
	if err != nil {
		t.Fatalf("Accounts after select: %v", err)
	}
	if got[0].Seq != 2 || got[1].Seq != 1 || got[2].Seq != 3 {
		t.Errorf("reordered = %+v, want seq order [2 1 3]", []int64{got[0].Seq, got[1].Seq, got[2].Seq})
	}

	active, err := lv.ActiveAccount(context.Background())
	if err != nil {
		t.Fatalf("ActiveAccount: %v", err)
	}
	if active.Seq != 2 {
		t.Errorf("ActiveAccount = %d, want 2", active.Seq)
	}
}

func TestLiveNoContext(t *testing.T) {
	mgr, err := session.NewManager(&config.Config{}, func(config.Resolved) (broker.Broker, error) {
		return brokertest.Stub{}, nil
	})
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	lv := &live{mgr: mgr}
	if _, err := lv.Accounts(context.Background()); err == nil {
		t.Error("expected error with no active context")
	}
}

func TestEstimateOrder(t *testing.T) {
	amount := domain.OrderRequest{
		Basis:  domain.AmountBased,
		Amount: domain.NewMoney(decimal.RequireFromString("100000"), domain.KRW),
	}
	if got := estimateOrder(amount); got != "est. amount: 100000 KRW" {
		t.Errorf("amount estimate = %q", got)
	}

	limit := domain.OrderRequest{
		Basis:    domain.QuantityBased,
		Type:     domain.LimitOrder,
		Quantity: decimal.RequireFromString("10"),
		Price:    domain.NewMoney(decimal.RequireFromString("70000"), domain.KRW),
	}
	if got := estimateOrder(limit); got != "est. amount: 700000 KRW" {
		t.Errorf("limit estimate = %q", got)
	}

	market := domain.OrderRequest{Basis: domain.QuantityBased, Type: domain.MarketOrder, Quantity: decimal.RequireFromString("10")}
	if got := estimateOrder(market); got != "" {
		t.Errorf("market estimate = %q, want empty (no price to estimate)", got)
	}
}

func TestBrokerFactory(t *testing.T) {
	r := config.Resolved{
		Broker:     config.BrokerConfig{Type: "toss"},
		Credential: config.Credential{ClientID: "id", ClientSecret: "secret"},
	}
	if _, err := brokerFactory(r); err != nil {
		t.Errorf("toss factory: %v", err)
	}

	bad := config.Resolved{Broker: config.BrokerConfig{Type: "etrade"}}
	if _, err := brokerFactory(bad); err == nil {
		t.Error("expected error for unsupported broker type")
	}
}
