package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/byunjuneseok/s8s/internal/domain"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/shopspring/decimal"
)

// OrderbookProvider is the slice of broker behaviour the order book screen needs.
type OrderbookProvider interface {
	Orderbook(ctx context.Context, symbol string) (domain.Orderbook, error)
}

// OrderbookScreen renders the bid/ask depth for a single symbol, with a
// proportional depth bar per level.
type OrderbookScreen struct {
	*tview.Flex
	app      *tview.Application
	provider OrderbookProvider

	view   *tview.TextView
	symbol string
}

func newOrderbookScreen(app *tview.Application, provider OrderbookProvider) *OrderbookScreen {
	s := &OrderbookScreen{
		app:      app,
		provider: provider,
		view:     tview.NewTextView().SetDynamicColors(true),
	}
	s.view.SetBorder(true).SetTitle(" orderbook ")

	s.Flex = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(s.view, 0, 1, true)

	s.view.SetInputCapture(func(ev *tcell.EventKey) *tcell.EventKey {
		if ev.Rune() == 'r' {
			s.Refresh()
			return nil
		}
		return ev
	})
	s.view.SetText("no symbol selected")
	return s
}

// SetSymbol points the screen at symbol; the next Refresh fetches its book.
func (s *OrderbookScreen) SetSymbol(symbol string) {
	s.symbol = symbol
	s.view.SetTitle(fmt.Sprintf(" orderbook · %s ", symbol))
}

// Refresh fetches the current order book in the background and redraws.
func (s *OrderbookScreen) Refresh() {
	symbol := s.symbol
	go func() {
		if symbol == "" {
			s.postMessage("no symbol selected")
			return
		}
		ob, err := s.provider.Orderbook(context.Background(), symbol)
		if err != nil {
			s.postError(fmt.Errorf("orderbook: %w", err))
			return
		}
		s.app.QueueUpdateDraw(func() { s.render(ob) })
	}()
}

func (s *OrderbookScreen) postError(err error) {
	s.app.QueueUpdateDraw(func() {
		s.view.SetText(fmt.Sprintf("[red]error:[-] %v", err))
	})
}

func (s *OrderbookScreen) postMessage(msg string) {
	s.app.QueueUpdateDraw(func() { s.view.SetText(msg) })
}

// depthBarWidth is the number of columns the proportional depth bar spans.
const depthBarWidth = 20

func (s *OrderbookScreen) render(ob domain.Orderbook) {
	s.view.SetText(renderOrderbook(ob, depthBarWidth))
}

// renderOrderbook builds the order-book text: asks first (best ask nearest the
// spread, i.e. descending price) then bids (best bid first). Each row carries a
// depth bar scaled against the largest quantity across both sides.
func renderOrderbook(ob domain.Orderbook, barWidth int) string {
	maxQty := maxQuantity(ob)

	var b strings.Builder
	// Asks: domain orders them ascending; reverse so the best (lowest) ask sits
	// just above the spread.
	for i := len(ob.Asks) - 1; i >= 0; i-- {
		e := ob.Asks[i]
		fmt.Fprintf(&b, "[blue]%12s  %10s  %s[-]\n",
			formatDecimal(e.Price.Amount),
			formatDecimal(e.Quantity),
			depthBar(e.Quantity, maxQty, barWidth),
		)
	}

	b.WriteString(strings.Repeat("─", 12) + "  spread\n")

	for _, e := range ob.Bids {
		fmt.Fprintf(&b, "[red]%12s  %10s  %s[-]\n",
			formatDecimal(e.Price.Amount),
			formatDecimal(e.Quantity),
			depthBar(e.Quantity, maxQty, barWidth),
		)
	}

	if len(ob.Asks) == 0 && len(ob.Bids) == 0 {
		return "empty order book"
	}
	return strings.TrimRight(b.String(), "\n")
}

func maxQuantity(ob domain.Orderbook) decimal.Decimal {
	maxQty := decimal.Zero
	for _, e := range ob.Asks {
		if e.Quantity.GreaterThan(maxQty) {
			maxQty = e.Quantity
		}
	}
	for _, e := range ob.Bids {
		if e.Quantity.GreaterThan(maxQty) {
			maxQty = e.Quantity
		}
	}
	return maxQty
}

// depthBar returns a bar of "█" runes whose length is proportional to qty
// relative to max, capped at width. A non-positive max or width yields "".
func depthBar(qty, maxQty decimal.Decimal, width int) string {
	if width <= 0 || !maxQty.IsPositive() || !qty.IsPositive() {
		return ""
	}
	ratio := qty.Div(maxQty)
	n := ratio.Mul(decimal.NewFromInt(int64(width))).IntPart()
	if n < 1 {
		n = 1
	}
	if n > int64(width) {
		n = int64(width)
	}
	return strings.Repeat("█", int(n))
}

// AddOrderbookScreen creates an order book screen backed by provider, registers
// it under name, and returns it.
func (a *App) AddOrderbookScreen(name string, provider OrderbookProvider) *OrderbookScreen {
	s := newOrderbookScreen(a.app, provider)
	a.AddScreen(name, s)
	return s
}
