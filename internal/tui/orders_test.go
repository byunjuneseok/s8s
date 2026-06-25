package tui

import (
	"context"
	"testing"

	"github.com/byunjuneseok/s8s/internal/domain"
)

type fakeOrdersProvider struct {
	orders []domain.Order
	err    error
}

func (f *fakeOrdersProvider) Orders(_ context.Context, _ domain.Account) ([]domain.Order, error) {
	return f.orders, f.err
}

func sampleOrders() []domain.Order {
	return []domain.Order{
		{
			ID: "o1", Symbol: "005930", Side: domain.Buy, Type: domain.LimitOrder,
			Status: domain.StatusOpen, Quantity: dec("10"), FilledQuantity: dec("3"),
			Price: domain.NewMoney(dec("70000"), domain.KRW),
		},
		{
			ID: "o2", Symbol: "AAPL", Side: domain.Sell, Type: domain.MarketOrder,
			Status: domain.StatusFilled, Quantity: dec("5"), FilledQuantity: dec("5"),
			Price: domain.NewMoney(dec("190"), domain.USD),
		},
	}
}

func TestOrdersRender(t *testing.T) {
	s := newOrdersScreen(NewApp().app, &fakeOrdersProvider{})
	s.render(sampleOrders())

	if got := s.table.GetCell(1, 0).Text; got != "o1" {
		t.Errorf("row1 id = %q, want o1", got)
	}
	if len(s.orderIDs) != 2 || s.orderIDs[1] != "o2" {
		t.Errorf("orderIDs = %v, want [o1 o2]", s.orderIDs)
	}
	// Re-render resets the ID mapping rather than appending.
	s.render(sampleOrders()[:1])
	if len(s.orderIDs) != 1 {
		t.Errorf("orderIDs after re-render = %v, want len 1", s.orderIDs)
	}
	// Empty path.
	s.render(nil)
	if len(s.orderIDs) != 0 {
		t.Errorf("orderIDs after empty render = %v", s.orderIDs)
	}
}

func TestOrdersSelectedOrderID(t *testing.T) {
	s := newOrdersScreen(NewApp().app, &fakeOrdersProvider{})
	s.render(sampleOrders())

	s.table.Select(1, 0)
	if got := s.selectedOrderID(); got != "o1" {
		t.Errorf("selected = %q, want o1", got)
	}
	s.table.Select(2, 0)
	if got := s.selectedOrderID(); got != "o2" {
		t.Errorf("selected = %q, want o2", got)
	}
	// Out-of-range row (header) yields "".
	s.table.Select(0, 0)
	if got := s.selectedOrderID(); got != "" {
		t.Errorf("selected on header = %q, want empty", got)
	}
}

func TestOrdersCallbacks(t *testing.T) {
	s := newOrdersScreen(NewApp().app, &fakeOrdersProvider{})
	s.render(sampleOrders())
	s.table.Select(2, 0)

	var modified, canceled string
	s.OnModify = func(id string) { modified = id }
	s.OnCancel = func(id string) { canceled = id }

	// Invoke the same logic the input capture uses.
	if id := s.selectedOrderID(); id != "" && s.OnModify != nil {
		s.OnModify(id)
	}
	if id := s.selectedOrderID(); id != "" && s.OnCancel != nil {
		s.OnCancel(id)
	}
	if modified != "o2" {
		t.Errorf("OnModify got %q, want o2", modified)
	}
	if canceled != "o2" {
		t.Errorf("OnCancel got %q, want o2", canceled)
	}
}

func TestSideColor(t *testing.T) {
	if sideColor(domain.Buy) != "[red]" {
		t.Error("buy should be red")
	}
	if sideColor(domain.Sell) != "[blue]" {
		t.Error("sell should be blue")
	}
}

func TestAddOrdersScreen(t *testing.T) {
	a := NewApp()
	s := a.AddOrdersScreen("orders", &fakeOrdersProvider{})
	if s == nil || !a.body.HasPage("orders") {
		t.Fatal("AddOrdersScreen did not register screen")
	}
}
