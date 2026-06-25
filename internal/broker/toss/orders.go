package toss

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/url"

	"github.com/byunjuneseok/s8s/internal/broker"
	"github.com/byunjuneseok/s8s/internal/domain"
	"github.com/shopspring/decimal"
)

// Client satisfies the full broker.Broker interface.
var _ broker.Broker = (*Client)(nil)

const ordersPath = "/api/v1/orders"

// orderDTO is the Toss representation of a placed order, shared by the place,
// modify, cancel, list, and detail endpoints.
type orderDTO struct {
	OrderID        string  `json:"orderId"`
	ClientOrderID  string  `json:"clientOrderId"`
	Symbol         string  `json:"symbol"`
	Side           string  `json:"side"`
	OrderType      string  `json:"orderType"`
	Status         string  `json:"status"`
	Quantity       string  `json:"quantity"`
	FilledQuantity string  `json:"filledQuantity"`
	Price          string  `json:"price"`
	Currency       string  `json:"currency"`
	CreatedAt      *string `json:"createdAt"`
}

func (d orderDTO) toDomain() (domain.Order, error) {
	quantity, err := decimalOrZero(d.Quantity)
	if err != nil {
		return domain.Order{}, fmt.Errorf("orders: parse quantity %q: %w", d.Quantity, err)
	}
	filled, err := decimalOrZero(d.FilledQuantity)
	if err != nil {
		return domain.Order{}, fmt.Errorf("orders: parse filled quantity %q: %w", d.FilledQuantity, err)
	}
	price, err := moneyOrZero(d.Price, domain.Currency(d.Currency))
	if err != nil {
		return domain.Order{}, err
	}
	createdAt, err := parseTimestamp(d.CreatedAt)
	if err != nil {
		return domain.Order{}, err
	}
	return domain.Order{
		ID:             d.OrderID,
		ClientOrderID:  d.ClientOrderID,
		Symbol:         d.Symbol,
		Side:           domain.Side(d.Side),
		Type:           domain.OrderType(d.OrderType),
		Status:         domain.OrderStatus(d.Status),
		Quantity:       quantity,
		FilledQuantity: filled,
		Price:          price,
		CreatedAt:      createdAt,
	}, nil
}

// placeOrderPayload is the request body for placing an order. The quantity vs.
// orderAmount fields are mutually exclusive, selected by the order's basis.
type placeOrderPayload struct {
	Symbol         string  `json:"symbol"`
	Side           string  `json:"side"`
	OrderType      string  `json:"orderType"`
	Quantity       *string `json:"quantity,omitempty"`
	OrderAmount    *string `json:"orderAmount,omitempty"`
	Price          *string `json:"price,omitempty"`
	TimeInForce    string  `json:"timeInForce,omitempty"`
	IdempotencyKey string  `json:"idempotencyKey"`
}

// PlaceOrder implements broker.Broker.
func (c *Client) PlaceOrder(ctx context.Context, acct domain.Account, req domain.OrderRequest) (domain.Order, error) {
	if err := req.Validate(); err != nil {
		return domain.Order{}, err
	}

	key := req.ClientOrderID
	if key == "" {
		var err error
		key, err = newIdempotencyKey()
		if err != nil {
			return domain.Order{}, err
		}
	}

	payload := placeOrderPayload{
		Symbol:         req.Symbol,
		Side:           string(req.Side),
		OrderType:      string(req.Type),
		TimeInForce:    string(req.TimeInForce),
		IdempotencyKey: key,
	}
	switch req.Basis {
	case domain.QuantityBased:
		q := req.Quantity.String()
		payload.Quantity = &q
	case domain.AmountBased:
		a := req.Amount.Amount.String()
		payload.OrderAmount = &a
	}
	if req.Type == domain.LimitOrder {
		p := req.Price.Amount.String()
		payload.Price = &p
	}

	dto, err := postJSON[orderDTO](ctx, c, ordersPath, accountHeaders(acct), payload)
	if err != nil {
		return domain.Order{}, err
	}
	return dto.toDomain()
}

// modifyOrderPayload is the request body for modifying an order.
type modifyOrderPayload struct {
	Price    string `json:"price"`
	Quantity string `json:"quantity"`
}

// ModifyOrder implements broker.Broker.
func (c *Client) ModifyOrder(ctx context.Context, acct domain.Account, orderID string, mod domain.OrderModification) (domain.Order, error) {
	payload := modifyOrderPayload{
		Price:    mod.Price.Amount.String(),
		Quantity: mod.Quantity.String(),
	}
	path := ordersPath + "/" + url.PathEscape(orderID) + "/modify"
	dto, err := postJSON[orderDTO](ctx, c, path, accountHeaders(acct), payload)
	if err != nil {
		return domain.Order{}, err
	}
	return dto.toDomain()
}

// CancelOrder implements broker.Broker.
func (c *Client) CancelOrder(ctx context.Context, acct domain.Account, orderID string) (domain.Order, error) {
	path := ordersPath + "/" + url.PathEscape(orderID) + "/cancel"
	dto, err := postJSON[orderDTO](ctx, c, path, accountHeaders(acct), struct{}{})
	if err != nil {
		return domain.Order{}, err
	}
	return dto.toDomain()
}

// Orders implements broker.Broker.
func (c *Client) Orders(ctx context.Context, acct domain.Account) ([]domain.Order, error) {
	dtos, err := getJSON[[]orderDTO](ctx, c, ordersPath, nil, accountHeaders(acct))
	if err != nil {
		return nil, err
	}
	orders := make([]domain.Order, 0, len(dtos))
	for i := range dtos {
		o, err := dtos[i].toDomain()
		if err != nil {
			return nil, err
		}
		orders = append(orders, o)
	}
	return orders, nil
}

// OrderDetail implements broker.Broker.
func (c *Client) OrderDetail(ctx context.Context, acct domain.Account, orderID string) (domain.Order, error) {
	path := ordersPath + "/" + url.PathEscape(orderID)
	dto, err := getJSON[orderDTO](ctx, c, path, nil, accountHeaders(acct))
	if err != nil {
		return domain.Order{}, err
	}
	return dto.toDomain()
}

// newIdempotencyKey generates a random hex idempotency key for orders placed
// without a caller-supplied ClientOrderID.
func newIdempotencyKey() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("orders: generate idempotency key: %w", err)
	}
	return hex.EncodeToString(b[:]), nil
}

// decimalOrZero parses a decimal string, treating empty as zero.
func decimalOrZero(s string) (decimal.Decimal, error) {
	if s == "" {
		return decimal.Zero, nil
	}
	return decimal.NewFromString(s)
}

// moneyOrZero parses a money string, treating empty as zero in the given currency.
func moneyOrZero(s string, cur domain.Currency) (domain.Money, error) {
	if s == "" {
		return domain.NewMoney(decimal.Zero, cur), nil
	}
	return domain.ParseMoney(s, cur)
}
