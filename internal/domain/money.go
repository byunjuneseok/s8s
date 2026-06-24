package domain

import (
	"fmt"

	"github.com/shopspring/decimal"
)

// Currency identifies the currency of a monetary amount.
type Currency string

// Supported currencies.
const (
	KRW Currency = "KRW"
	USD Currency = "USD"
)

// Money is a decimal amount paired with its currency. Amounts use
// decimal.Decimal rather than a float so that prices and balances are exact.
type Money struct {
	Amount   decimal.Decimal
	Currency Currency
}

// NewMoney returns a Money with the given amount and currency.
func NewMoney(amount decimal.Decimal, c Currency) Money {
	return Money{Amount: amount, Currency: c}
}

// ParseMoney parses a decimal string (the form brokerage APIs return amounts in)
// into Money with the given currency.
func ParseMoney(s string, c Currency) (Money, error) {
	d, err := decimal.NewFromString(s)
	if err != nil {
		return Money{}, fmt.Errorf("parse money %q: %w", s, err)
	}
	return Money{Amount: d, Currency: c}, nil
}

// IsZero reports whether the amount is zero.
func (m Money) IsZero() bool { return m.Amount.IsZero() }

// IsNegative reports whether the amount is less than zero.
func (m Money) IsNegative() bool { return m.Amount.IsNegative() }

// Add returns the sum of m and other. The currencies must match.
func (m Money) Add(other Money) (Money, error) {
	if m.Currency != other.Currency {
		return Money{}, fmt.Errorf("money: currency mismatch %s vs %s", m.Currency, other.Currency)
	}
	return Money{Amount: m.Amount.Add(other.Amount), Currency: m.Currency}, nil
}

// String renders the amount and currency, e.g. "70000 KRW".
func (m Money) String() string {
	return fmt.Sprintf("%s %s", m.Amount.String(), m.Currency)
}
