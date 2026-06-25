package toss

import (
	"context"
	"net/url"

	"github.com/byunjuneseok/s8s/internal/domain"
	"github.com/shopspring/decimal"
)

const (
	buyingPowerPath      = "/api/v1/buying-power"
	sellableQuantityPath = "/api/v1/sellable-quantity"
	commissionsPath      = "/api/v1/commissions"
)

// buyingPowerDTO is the cash available to buy in a given currency.
type buyingPowerDTO struct {
	Amount   string `json:"amount"`
	Currency string `json:"currency"`
}

// sellableQuantityDTO is the quantity of a symbol that may be sold.
type sellableQuantityDTO struct {
	Symbol   string `json:"symbol"`
	Quantity string `json:"quantity"`
}

// commissionDTO is the estimated fee for an order.
type commissionDTO struct {
	Fee      string `json:"fee"`
	Rate     string `json:"rate"`
	Currency string `json:"currency"`
}

// BuyingPower implements broker.Broker.
func (c *Client) BuyingPower(ctx context.Context, acct domain.Account, currency domain.Currency) (domain.Money, error) {
	query := url.Values{"currency": {string(currency)}}
	dto, err := getJSON[buyingPowerDTO](ctx, c, buyingPowerPath, query, accountHeaders(acct))
	if err != nil {
		return domain.Money{}, err
	}
	return domain.ParseMoney(dto.Amount, domain.Currency(dto.Currency))
}

// SellableQuantity implements broker.Broker.
func (c *Client) SellableQuantity(ctx context.Context, acct domain.Account, symbol string) (decimal.Decimal, error) {
	query := url.Values{"symbol": {symbol}}
	dto, err := getJSON[sellableQuantityDTO](ctx, c, sellableQuantityPath, query, accountHeaders(acct))
	if err != nil {
		return decimal.Decimal{}, err
	}
	return decimal.NewFromString(dto.Quantity)
}

// Commission implements broker.Broker.
func (c *Client) Commission(ctx context.Context, acct domain.Account, req domain.OrderRequest) (domain.Commission, error) {
	query := url.Values{
		"symbol": {req.Symbol},
		"side":   {string(req.Side)},
	}
	if req.Basis == domain.QuantityBased {
		query.Set("quantity", req.Quantity.String())
	}
	if req.Type == domain.LimitOrder {
		query.Set("price", req.Price.Amount.String())
	}
	dto, err := getJSON[commissionDTO](ctx, c, commissionsPath, query, accountHeaders(acct))
	if err != nil {
		return domain.Commission{}, err
	}
	fee, err := domain.ParseMoney(dto.Fee, domain.Currency(dto.Currency))
	if err != nil {
		return domain.Commission{}, err
	}
	rate, err := decimal.NewFromString(dto.Rate)
	if err != nil {
		return domain.Commission{}, err
	}
	return domain.Commission{Fee: fee, Rate: rate}, nil
}
