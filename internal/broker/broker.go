package broker

import (
	"context"

	"github.com/byunjuneseok/s8s/internal/domain"
	"github.com/shopspring/decimal"
)

// Broker is the single abstraction the application depends on for all
// securities operations. Implementations translate a specific brokerage API
// into the broker-neutral domain models; no provider-specific type ever appears
// in this interface.
type Broker interface {
	// Accounts lists the accounts available to the authenticated user.
	Accounts(ctx context.Context) ([]domain.Account, error)
	// Holdings returns the positions and per-currency totals for an account.
	Holdings(ctx context.Context, acct domain.Account) (domain.HoldingsOverview, error)

	// Prices returns the latest quote for each requested symbol.
	Prices(ctx context.Context, symbols []string) ([]domain.Quote, error)
	// Orderbook returns the current bid/ask depth for a symbol.
	Orderbook(ctx context.Context, symbol string) (domain.Orderbook, error)

	// BuyingPower returns the cash available to buy in the given currency.
	BuyingPower(ctx context.Context, acct domain.Account, currency domain.Currency) (domain.Money, error)
	// SellableQuantity returns the quantity of a symbol that can be sold.
	SellableQuantity(ctx context.Context, acct domain.Account, symbol string) (decimal.Decimal, error)
	// Commission estimates the fee for the given order.
	Commission(ctx context.Context, acct domain.Account, req domain.OrderRequest) (domain.Commission, error)

	// PlaceOrder submits a new order and returns the resulting order.
	PlaceOrder(ctx context.Context, acct domain.Account, req domain.OrderRequest) (domain.Order, error)
	// ModifyOrder changes an existing order and returns its updated state.
	ModifyOrder(ctx context.Context, acct domain.Account, orderID string, mod domain.OrderModification) (domain.Order, error)
	// CancelOrder cancels an existing order and returns its updated state.
	CancelOrder(ctx context.Context, acct domain.Account, orderID string) (domain.Order, error)
	// Orders lists the account's orders.
	Orders(ctx context.Context, acct domain.Account) ([]domain.Order, error)
	// OrderDetail returns a single order by id.
	OrderDetail(ctx context.Context, acct domain.Account, orderID string) (domain.Order, error)
}
