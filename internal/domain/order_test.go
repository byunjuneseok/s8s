package domain

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestOrderRequestValidate(t *testing.T) {
	qty := func(s string) decimal.Decimal { return decimal.RequireFromString(s) }
	krw := func(s string) Money { return NewMoney(qty(s), KRW) }

	tests := []struct {
		name    string
		req     OrderRequest
		wantErr bool
	}{
		{
			name: "valid quantity limit buy",
			req:  OrderRequest{Symbol: "005930", Side: Buy, Type: LimitOrder, Basis: QuantityBased, Quantity: qty("10"), Price: krw("70000")},
		},
		{
			name: "valid amount market buy",
			req:  OrderRequest{Symbol: "AAPL", Side: Buy, Type: MarketOrder, Basis: AmountBased, Amount: NewMoney(qty("100.5"), USD)},
		},
		{
			name:    "missing symbol",
			req:     OrderRequest{Side: Buy, Type: MarketOrder, Basis: QuantityBased, Quantity: qty("1")},
			wantErr: true,
		},
		{
			name:    "invalid side",
			req:     OrderRequest{Symbol: "X", Side: "HOLD", Type: MarketOrder, Basis: QuantityBased, Quantity: qty("1")},
			wantErr: true,
		},
		{
			name:    "limit without price",
			req:     OrderRequest{Symbol: "X", Side: Buy, Type: LimitOrder, Basis: QuantityBased, Quantity: qty("1")},
			wantErr: true,
		},
		{
			name:    "non-positive quantity",
			req:     OrderRequest{Symbol: "X", Side: Sell, Type: MarketOrder, Basis: QuantityBased, Quantity: qty("0")},
			wantErr: true,
		},
		{
			name:    "missing basis",
			req:     OrderRequest{Symbol: "X", Side: Buy, Type: MarketOrder},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if tt.wantErr && err == nil {
				t.Error("Validate() = nil, want error")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Validate() = %v, want nil", err)
			}
		})
	}
}
