package domain

// Market identifies the market a symbol is listed on.
type Market string

// Supported markets.
const (
	MarketKR Market = "KR"
	MarketUS Market = "US"
)

// Side is the direction of an order.
type Side string

// Order sides.
const (
	Buy  Side = "BUY"
	Sell Side = "SELL"
)

// OrderType is the execution type of an order.
type OrderType string

// Order types.
const (
	LimitOrder  OrderType = "LIMIT"
	MarketOrder OrderType = "MARKET"
)

// TimeInForce specifies how long an order remains active.
type TimeInForce string

// Time-in-force values. The empty value means the broker default.
const (
	Day     TimeInForce = "DAY"
	Closing TimeInForce = "CLS"
)

// OrderStatus is the lifecycle state of a placed order.
type OrderStatus string

// Order statuses.
const (
	StatusPending         OrderStatus = "PENDING"
	StatusOpen            OrderStatus = "OPEN"
	StatusPartiallyFilled OrderStatus = "PARTIALLY_FILLED"
	StatusFilled          OrderStatus = "FILLED"
	StatusCanceled        OrderStatus = "CANCELED"
	StatusRejected        OrderStatus = "REJECTED"
)
