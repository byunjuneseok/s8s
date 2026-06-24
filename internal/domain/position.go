package domain

import "github.com/shopspring/decimal"

// PnL is a profit-and-loss figure: an absolute amount and a rate of return.
type PnL struct {
	// Amount is the absolute gain or loss.
	Amount Money
	// Rate is the return as a fraction (0.05 means +5%).
	Rate decimal.Decimal
}

// PnLSet groups the cumulative and single-day profit-and-loss for a holding.
type PnLSet struct {
	// Total is the cumulative profit and loss since purchase.
	Total PnL
	// Daily is the profit and loss over the current day.
	Daily PnL
}

// Position is a single holding within an account, valued in its trading
// currency.
type Position struct {
	Symbol      string
	Name        string
	Market      Market
	Currency    Currency
	Quantity    decimal.Decimal
	AvgCost     Money
	LastPrice   Money
	MarketValue Money
	PnL         PnLSet
}

// CurrencyTotals aggregates a portfolio's figures for a single currency, since
// holdings across markets cannot be summed into one currency.
type CurrencyTotals struct {
	Currency       Currency
	PurchaseAmount Money
	MarketValue    Money
	PnL            PnLSet
}

// HoldingsOverview is the full picture of an account's holdings: per-currency
// totals plus the individual positions.
type HoldingsOverview struct {
	Totals    []CurrencyTotals
	Positions []Position
}
