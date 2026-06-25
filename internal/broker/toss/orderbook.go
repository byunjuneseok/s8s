package toss

import (
	"context"
	"net/url"

	"github.com/byunjuneseok/s8s/internal/domain"
	"github.com/shopspring/decimal"
)

const orderbookPath = "/api/v1/orderbook"

// orderbookEntryDTO is one price level in the Toss order book.
type orderbookEntryDTO struct {
	Price    string `json:"price"`
	Quantity string `json:"quantity"`
}

// orderbookDTO is the Toss order book for a symbol. Asks are ordered lowest to
// highest price, bids highest to lowest; the ordering is preserved as given.
type orderbookDTO struct {
	Asks      []orderbookEntryDTO `json:"asks"`
	Bids      []orderbookEntryDTO `json:"bids"`
	Currency  string              `json:"currency"`
	Timestamp *string             `json:"timestamp"`
}

func (e orderbookEntryDTO) toDomain(cur domain.Currency) (domain.OrderbookEntry, error) {
	price, err := domain.ParseMoney(e.Price, cur)
	if err != nil {
		return domain.OrderbookEntry{}, err
	}
	qty, err := decimal.NewFromString(e.Quantity)
	if err != nil {
		return domain.OrderbookEntry{}, err
	}
	return domain.OrderbookEntry{Price: price, Quantity: qty}, nil
}

func (o orderbookDTO) toDomain(symbol string) (domain.Orderbook, error) {
	cur := domain.Currency(o.Currency)
	asks, err := orderbookEntries(o.Asks, cur)
	if err != nil {
		return domain.Orderbook{}, err
	}
	bids, err := orderbookEntries(o.Bids, cur)
	if err != nil {
		return domain.Orderbook{}, err
	}
	ts, err := parseTimestamp(o.Timestamp)
	if err != nil {
		return domain.Orderbook{}, err
	}
	return domain.Orderbook{
		Symbol:    symbol,
		Asks:      asks,
		Bids:      bids,
		Timestamp: ts,
	}, nil
}

func orderbookEntries(dtos []orderbookEntryDTO, cur domain.Currency) ([]domain.OrderbookEntry, error) {
	entries := make([]domain.OrderbookEntry, 0, len(dtos))
	for i := range dtos {
		e, err := dtos[i].toDomain(cur)
		if err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, nil
}

// Orderbook implements broker.Broker.
func (c *Client) Orderbook(ctx context.Context, symbol string) (domain.Orderbook, error) {
	query := url.Values{"symbol": {symbol}}
	dto, err := getJSON[orderbookDTO](ctx, c, orderbookPath, query, nil)
	if err != nil {
		return domain.Orderbook{}, err
	}
	return dto.toDomain(symbol)
}
