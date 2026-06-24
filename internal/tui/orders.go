package tui

import (
	"context"
	"fmt"

	"github.com/byunjuneseok/s8s/internal/domain"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// OrdersProvider is the slice of broker behaviour the orders screen needs.
type OrdersProvider interface {
	Orders(ctx context.Context, acct domain.Account) ([]domain.Order, error)
}

// OrdersScreen lists open and recent orders for an account. It performs no
// network calls of its own for modify/cancel: instead it invokes the OnModify
// and OnCancel callbacks with the selected order's ID.
type OrdersScreen struct {
	*tview.Flex
	app      *tview.Application
	provider OrdersProvider

	table *tview.Table
	acct  domain.Account

	// OnModify, if set, is called with the selected order ID when 'm' is pressed.
	OnModify func(orderID string)
	// OnCancel, if set, is called with the selected order ID when 'c' is pressed.
	OnCancel func(orderID string)

	// orderIDs maps table row (1-based) to order ID for the rendered orders.
	orderIDs []string
}

func newOrdersScreen(app *tview.Application, provider OrdersProvider) *OrdersScreen {
	s := &OrdersScreen{
		app:      app,
		provider: provider,
		table:    tview.NewTable().SetBorders(false).SetFixed(1, 0).SetSelectable(true, false),
	}
	s.table.SetBorder(true).SetTitle(" orders ")

	s.Flex = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(s.table, 0, 1, true)

	s.table.SetInputCapture(func(ev *tcell.EventKey) *tcell.EventKey {
		switch ev.Rune() {
		case 'r':
			s.Refresh()
			return nil
		case 'm':
			if id := s.selectedOrderID(); id != "" && s.OnModify != nil {
				s.OnModify(id)
			}
			return nil
		case 'c':
			if id := s.selectedOrderID(); id != "" && s.OnCancel != nil {
				s.OnCancel(id)
			}
			return nil
		}
		return ev
	})
	s.setLoading()
	return s
}

// SetAccount points the screen at acct; the next Refresh fetches its orders.
func (s *OrdersScreen) SetAccount(acct domain.Account) { s.acct = acct }

func (s *OrdersScreen) setLoading() {
	s.table.Clear()
	s.table.SetCell(0, 0, tview.NewTableCell("loading…").SetSelectable(false))
}

// selectedOrderID returns the order ID for the currently selected row, or "".
func (s *OrdersScreen) selectedOrderID() string {
	row, _ := s.table.GetSelection()
	idx := row - 1
	if idx < 0 || idx >= len(s.orderIDs) {
		return ""
	}
	return s.orderIDs[idx]
}

// Refresh fetches the account's orders in the background and redraws.
func (s *OrdersScreen) Refresh() {
	acct := s.acct
	go func() {
		orders, err := s.provider.Orders(context.Background(), acct)
		if err != nil {
			s.postError(fmt.Errorf("orders: %w", err))
			return
		}
		s.app.QueueUpdateDraw(func() { s.render(orders) })
	}()
}

func (s *OrdersScreen) postError(err error) {
	s.app.QueueUpdateDraw(func() {
		s.table.Clear()
		s.orderIDs = nil
		s.table.SetCell(0, 0, tview.NewTableCell(fmt.Sprintf("[red]error:[-] %v", err)).SetSelectable(false))
	})
}

func (s *OrdersScreen) render(orders []domain.Order) {
	s.table.Clear()
	s.orderIDs = s.orderIDs[:0]

	headers := []string{"ID", "SYMBOL", "SIDE", "TYPE", "STATUS", "QTY", "FILLED", "PRICE"}
	for c, h := range headers {
		s.table.SetCell(0, c, tview.NewTableCell(h).
			SetTextColor(tcell.ColorYellow).
			SetSelectable(false))
	}

	for i, o := range orders {
		row := i + 1
		s.orderIDs = append(s.orderIDs, o.ID)
		cells := []string{
			o.ID,
			o.Symbol,
			sideColor(o.Side) + string(o.Side) + "[-]",
			string(o.Type),
			string(o.Status),
			formatDecimal(o.Quantity),
			formatDecimal(o.FilledQuantity),
			formatDecimal(o.Price.Amount),
		}
		for c, v := range cells {
			cell := tview.NewTableCell(v)
			if c >= 5 {
				cell.SetAlign(tview.AlignRight)
			}
			s.table.SetCell(row, c, cell)
		}
	}

	if len(orders) == 0 {
		s.table.SetCell(1, 0, tview.NewTableCell("no orders").SetSelectable(false))
	}
}

// sideColor returns a tview color tag for an order side: buy red, sell blue
// (Korean market convention).
func sideColor(side domain.Side) string {
	switch side {
	case domain.Buy:
		return "[red]"
	case domain.Sell:
		return "[blue]"
	default:
		return "[white]"
	}
}

// AddOrdersScreen creates an orders screen backed by provider, registers it
// under name, and returns it.
func (a *App) AddOrdersScreen(name string, provider OrdersProvider) *OrdersScreen {
	s := newOrdersScreen(a.app, provider)
	a.AddScreen(name, s)
	return s
}
