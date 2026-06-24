package tui

import (
	"context"
	"strings"
	"testing"

	"github.com/byunjuneseok/s8s/internal/domain"
	"github.com/shopspring/decimal"
)

type fakeOrderbookProvider struct {
	ob  domain.Orderbook
	err error
}

func (f *fakeOrderbookProvider) Orderbook(_ context.Context, _ string) (domain.Orderbook, error) {
	return f.ob, f.err
}

func TestDepthBar(t *testing.T) {
	cases := []struct {
		name      string
		qty, max  string
		width     int
		wantRunes int // expected count of █
	}{
		{"full", "10", "10", 20, 20},
		{"half", "5", "10", 20, 10},
		{"tiny rounds up to 1", "1", "1000", 20, 1},
		{"quarter", "5", "20", 20, 5},
		{"caps at width", "100", "10", 20, 20},
		{"zero qty", "0", "10", 20, 0},
		{"zero max", "5", "0", 20, 0},
		{"zero width", "5", "10", 0, 0},
		{"negative qty", "-5", "10", 20, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := depthBar(dec(c.qty), dec(c.max), c.width)
			if n := len([]rune(got)); n != c.wantRunes {
				t.Errorf("depthBar(%s,%s,%d) = %d runes, want %d", c.qty, c.max, c.width, n, c.wantRunes)
			}
		})
	}
}

func TestMaxQuantity(t *testing.T) {
	ob := domain.Orderbook{
		Asks: []domain.OrderbookEntry{{Quantity: dec("3")}, {Quantity: dec("9")}},
		Bids: []domain.OrderbookEntry{{Quantity: dec("7")}},
	}
	if got := maxQuantity(ob); !got.Equal(dec("9")) {
		t.Errorf("maxQuantity = %s, want 9", got)
	}
	if got := maxQuantity(domain.Orderbook{}); !got.Equal(decimal.Zero) {
		t.Errorf("maxQuantity(empty) = %s, want 0", got)
	}
}

func mkEntry(price, qty string) domain.OrderbookEntry {
	return domain.OrderbookEntry{
		Price:    domain.NewMoney(dec(price), domain.KRW),
		Quantity: dec(qty),
	}
}

func TestRenderOrderbookAsksOrderedBestNearestSpread(t *testing.T) {
	ob := domain.Orderbook{
		Symbol: "X",
		// domain asks ascending: 100 is best ask.
		Asks: []domain.OrderbookEntry{mkEntry("100", "1"), mkEntry("101", "2"), mkEntry("102", "3")},
		Bids: []domain.OrderbookEntry{mkEntry("99", "4"), mkEntry("98", "5")},
	}
	out := renderOrderbook(ob, depthBarWidth)
	lines := strings.Split(out, "\n")

	// Locate spread row.
	spreadIdx := -1
	for i, l := range lines {
		if strings.Contains(l, "spread") {
			spreadIdx = i
		}
	}
	if spreadIdx < 0 {
		t.Fatalf("no spread line in:\n%s", out)
	}
	// The ask line immediately above the spread must be the best (lowest) ask, 100.
	bestAskLine := lines[spreadIdx-1]
	if !strings.Contains(bestAskLine, "100") {
		t.Errorf("ask nearest spread = %q, want it to contain best ask 100", bestAskLine)
	}
	// The bid line immediately below the spread must be the best (highest) bid, 99.
	bestBidLine := lines[spreadIdx+1]
	if !strings.Contains(bestBidLine, "99") {
		t.Errorf("bid nearest spread = %q, want it to contain best bid 99", bestBidLine)
	}
}

func TestRenderOrderbookEmpty(t *testing.T) {
	if got := renderOrderbook(domain.Orderbook{}, depthBarWidth); got != "empty order book" {
		t.Errorf("empty render = %q", got)
	}
}

func TestOrderbookScreenRender(_ *testing.T) {
	s := newOrderbookScreen(NewApp().app, &fakeOrderbookProvider{})
	s.SetSymbol("005930")
	s.render(domain.Orderbook{Symbol: "005930", Asks: []domain.OrderbookEntry{mkEntry("100", "1")}}) // no panic
}

func TestAddOrderbookScreen(t *testing.T) {
	a := NewApp()
	s := a.AddOrderbookScreen("orderbook", &fakeOrderbookProvider{})
	if s == nil || !a.body.HasPage("orderbook") {
		t.Fatal("AddOrderbookScreen did not register screen")
	}
}
