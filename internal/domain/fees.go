package domain

import "github.com/shopspring/decimal"

// Commission is the fee charged for a trade.
type Commission struct {
	// Fee is the total commission amount.
	Fee Money
	// Rate is the commission as a fraction of the trade value (0.0001 = 0.01%).
	Rate decimal.Decimal
}
