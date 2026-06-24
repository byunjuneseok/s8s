package domain

import (
	"time"

	"github.com/shopspring/decimal"
)

// Quote is the latest traded price for a symbol.
type Quote struct {
	Symbol string
	// LastPrice is the most recent traded price.
	LastPrice Money
	// Timestamp is when the price was observed, or nil if no trade has occurred.
	Timestamp *time.Time
}

// OrderbookEntry is a single price level in the order book.
type OrderbookEntry struct {
	Price    Money
	Quantity decimal.Decimal
}

// Orderbook is the current bid/ask depth for a symbol.
type Orderbook struct {
	Symbol string
	// Asks are sell levels ordered from lowest price to highest.
	Asks []OrderbookEntry
	// Bids are buy levels ordered from highest price to lowest.
	Bids []OrderbookEntry
	// Timestamp is when the book was observed, or nil if unavailable.
	Timestamp *time.Time
}
