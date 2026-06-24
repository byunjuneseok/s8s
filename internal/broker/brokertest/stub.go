// Package brokertest provides test doubles for the broker.Broker interface.
package brokertest

import (
	"context"

	"github.com/byunjuneseok/s8s/internal/broker"
	"github.com/byunjuneseok/s8s/internal/domain"
	"github.com/shopspring/decimal"
)

// Stub is a no-op broker.Broker implementation that returns zero values and nil
// errors. It is a convenient placeholder while wiring up code that depends on a
// Broker.
type Stub struct{}

// Stub must satisfy the Broker interface.
var _ broker.Broker = Stub{}

// Accounts implements broker.Broker.
func (Stub) Accounts(context.Context) ([]domain.Account, error) { return nil, nil }

// Holdings implements broker.Broker.
func (Stub) Holdings(context.Context, domain.Account) (domain.HoldingsOverview, error) {
	return domain.HoldingsOverview{}, nil
}

// Prices implements broker.Broker.
func (Stub) Prices(context.Context, []string) ([]domain.Quote, error) { return nil, nil }

// Orderbook implements broker.Broker.
func (Stub) Orderbook(context.Context, string) (domain.Orderbook, error) {
	return domain.Orderbook{}, nil
}

// BuyingPower implements broker.Broker.
func (Stub) BuyingPower(context.Context, domain.Account, domain.Currency) (domain.Money, error) {
	return domain.Money{}, nil
}

// SellableQuantity implements broker.Broker.
func (Stub) SellableQuantity(context.Context, domain.Account, string) (decimal.Decimal, error) {
	return decimal.Decimal{}, nil
}

// Commission implements broker.Broker.
func (Stub) Commission(context.Context, domain.Account, domain.OrderRequest) (domain.Commission, error) {
	return domain.Commission{}, nil
}

// PlaceOrder implements broker.Broker.
func (Stub) PlaceOrder(context.Context, domain.Account, domain.OrderRequest) (domain.Order, error) {
	return domain.Order{}, nil
}

// ModifyOrder implements broker.Broker.
func (Stub) ModifyOrder(context.Context, domain.Account, string, domain.OrderModification) (domain.Order, error) {
	return domain.Order{}, nil
}

// CancelOrder implements broker.Broker.
func (Stub) CancelOrder(context.Context, domain.Account, string) (domain.Order, error) {
	return domain.Order{}, nil
}

// Orders implements broker.Broker.
func (Stub) Orders(context.Context, domain.Account) ([]domain.Order, error) { return nil, nil }

// OrderDetail implements broker.Broker.
func (Stub) OrderDetail(context.Context, domain.Account, string) (domain.Order, error) {
	return domain.Order{}, nil
}
