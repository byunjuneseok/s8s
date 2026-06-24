package toss

import (
	"context"
	"net/url"
	"strings"
	"time"

	"github.com/byunjuneseok/s8s/internal/domain"
)

const pricesPath = "/api/v1/prices"

// priceItemDTO is the Toss quote for a single symbol. Timestamp is null when no
// trade has occurred yet.
type priceItemDTO struct {
	Symbol    string  `json:"symbol"`
	LastPrice string  `json:"lastPrice"`
	Currency  string  `json:"currency"`
	Timestamp *string `json:"timestamp"`
}

func (it priceItemDTO) toDomain() (domain.Quote, error) {
	last, err := domain.ParseMoney(it.LastPrice, domain.Currency(it.Currency))
	if err != nil {
		return domain.Quote{}, err
	}
	ts, err := parseTimestamp(it.Timestamp)
	if err != nil {
		return domain.Quote{}, err
	}
	return domain.Quote{
		Symbol:    it.Symbol,
		LastPrice: last,
		Timestamp: ts,
	}, nil
}

// Prices implements broker.Broker.
func (c *Client) Prices(ctx context.Context, symbols []string) ([]domain.Quote, error) {
	query := url.Values{"symbols": {strings.Join(symbols, ",")}}
	items, err := getJSON[[]priceItemDTO](ctx, c, pricesPath, query, nil)
	if err != nil {
		return nil, err
	}
	quotes := make([]domain.Quote, 0, len(items))
	for i := range items {
		q, err := items[i].toDomain()
		if err != nil {
			return nil, err
		}
		quotes = append(quotes, q)
	}
	return quotes, nil
}

// parseTimestamp parses a nullable RFC3339 timestamp. A nil or empty input
// yields a nil *time.Time.
func parseTimestamp(s *string) (*time.Time, error) {
	if s == nil || *s == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, *s)
	if err != nil {
		return nil, err
	}
	return &t, nil
}
