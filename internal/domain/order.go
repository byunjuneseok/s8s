package domain

import (
	"errors"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
)

// OrderBasis indicates whether an order specifies a quantity of shares or a
// cash amount to spend.
type OrderBasis string

// Order bases.
const (
	// QuantityBased orders specify a number of shares (OrderRequest.Quantity).
	QuantityBased OrderBasis = "QUANTITY"
	// AmountBased orders specify a cash amount to spend (OrderRequest.Amount).
	AmountBased OrderBasis = "AMOUNT"
)

// OrderRequest is a request to place an order. Exactly one of Quantity or
// Amount is meaningful, selected by Basis.
type OrderRequest struct {
	Symbol      string
	Side        Side
	Type        OrderType
	TimeInForce TimeInForce // optional; empty means broker default
	Basis       OrderBasis

	// Quantity is the number of shares when Basis is QuantityBased. Fractional
	// quantities are allowed for markets that support them (e.g. US).
	Quantity decimal.Decimal
	// Amount is the cash to spend when Basis is AmountBased.
	Amount Money
	// Price is the limit price; required for LimitOrder, ignored for MarketOrder.
	Price Money
	// ClientOrderID is an optional idempotency key supplied by the caller.
	ClientOrderID string
}

// Validate checks that the request is internally consistent.
func (r OrderRequest) Validate() error {
	if r.Symbol == "" {
		return errors.New("order: symbol is required")
	}
	switch r.Side {
	case Buy, Sell:
	default:
		return fmt.Errorf("order: invalid side %q", r.Side)
	}
	switch r.Type {
	case LimitOrder, MarketOrder:
	default:
		return fmt.Errorf("order: invalid type %q", r.Type)
	}
	switch r.Basis {
	case QuantityBased:
		if !r.Quantity.IsPositive() {
			return errors.New("order: quantity must be positive")
		}
	case AmountBased:
		if !r.Amount.Amount.IsPositive() {
			return errors.New("order: amount must be positive")
		}
	default:
		return fmt.Errorf("order: invalid basis %q", r.Basis)
	}
	if r.Type == LimitOrder && !r.Price.Amount.IsPositive() {
		return errors.New("order: limit order requires a positive price")
	}
	return nil
}

// Order is a placed order as reported by the broker.
type Order struct {
	ID             string
	ClientOrderID  string
	Symbol         string
	Side           Side
	Type           OrderType
	Status         OrderStatus
	Quantity       decimal.Decimal
	FilledQuantity decimal.Decimal
	Price          Money
	CreatedAt      *time.Time
}
