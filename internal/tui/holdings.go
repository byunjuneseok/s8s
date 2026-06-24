package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/byunjuneseok/s8s/internal/domain"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// HoldingsProvider is the slice of broker behaviour the holdings screen needs.
type HoldingsProvider interface {
	Accounts(ctx context.Context) ([]domain.Account, error)
	Holdings(ctx context.Context, acct domain.Account) (domain.HoldingsOverview, error)
}

// HoldingsScreen shows an account's positions and per-currency totals.
type HoldingsScreen struct {
	*tview.Flex
	app      *tview.Application
	provider HoldingsProvider

	overview *tview.TextView
	table    *tview.Table
}

func newHoldingsScreen(app *tview.Application, provider HoldingsProvider) *HoldingsScreen {
	s := &HoldingsScreen{
		app:      app,
		provider: provider,
		overview: tview.NewTextView().SetDynamicColors(true),
		table:    tview.NewTable().SetBorders(false).SetFixed(1, 0).SetSelectable(true, false),
	}
	s.overview.SetBorder(true).SetTitle(" overview ")
	s.table.SetBorder(true).SetTitle(" holdings ")

	s.Flex = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(s.overview, 4, 0, false).
		AddItem(s.table, 0, 1, true)

	s.table.SetInputCapture(func(ev *tcell.EventKey) *tcell.EventKey {
		if ev.Rune() == 'r' {
			s.Refresh()
			return nil
		}
		return ev
	})
	s.setLoading()
	return s
}

func (s *HoldingsScreen) setLoading() {
	s.overview.SetText("loading…")
	s.table.Clear()
}

// Refresh fetches accounts and holdings in the background and redraws.
func (s *HoldingsScreen) Refresh() {
	go func() {
		ctx := context.Background()
		accts, err := s.provider.Accounts(ctx)
		if err != nil {
			s.postError(fmt.Errorf("accounts: %w", err))
			return
		}
		if len(accts) == 0 {
			s.postMessage("no accounts on this context")
			return
		}
		ov, err := s.provider.Holdings(ctx, accts[0])
		if err != nil {
			s.postError(fmt.Errorf("holdings: %w", err))
			return
		}
		s.app.QueueUpdateDraw(func() { s.render(accts[0], ov) })
	}()
}

func (s *HoldingsScreen) postError(err error) {
	s.app.QueueUpdateDraw(func() {
		s.overview.SetText(fmt.Sprintf("[red]error:[-] %v", err))
		s.table.Clear()
	})
}

func (s *HoldingsScreen) postMessage(msg string) {
	s.app.QueueUpdateDraw(func() {
		s.overview.SetText(msg)
		s.table.Clear()
	})
}

func (s *HoldingsScreen) render(acct domain.Account, ov domain.HoldingsOverview) {
	s.renderOverview(acct, ov)
	s.renderTable(ov.Positions)
}

func (s *HoldingsScreen) renderOverview(acct domain.Account, ov domain.HoldingsOverview) {
	var b strings.Builder
	fmt.Fprintf(&b, "account %s\n", acct.No)
	for _, t := range ov.Totals {
		color := pnlColor(t.PnL.Total.Amount.Amount)
		fmt.Fprintf(&b, "%s  invested %s  value %s  P/L %s%s (%s)[-]\n",
			t.Currency,
			formatMoney(t.PurchaseAmount),
			formatMoney(t.MarketValue),
			color, formatMoney(t.PnL.Total.Amount), formatRate(t.PnL.Total.Rate),
		)
	}
	s.overview.SetText(strings.TrimRight(b.String(), "\n"))
}

func (s *HoldingsScreen) renderTable(positions []domain.Position) {
	s.table.Clear()
	headers := []string{"SYMBOL", "NAME", "QTY", "AVG", "LAST", "VALUE", "P/L", "P/L%", "DAY%"}
	for c, h := range headers {
		s.table.SetCell(0, c, tview.NewTableCell(h).
			SetTextColor(tcell.ColorYellow).
			SetSelectable(false).
			SetExpansion(boolToInt(c == 1)))
	}

	for i, p := range positions {
		row := i + 1
		plColor := pnlColor(p.PnL.Total.Amount.Amount)
		dayColor := pnlColor(p.PnL.Daily.Amount.Amount)
		cells := []string{
			p.Symbol,
			p.Name,
			formatDecimal(p.Quantity),
			formatDecimal(p.AvgCost.Amount),
			formatDecimal(p.LastPrice.Amount),
			formatDecimal(p.MarketValue.Amount),
			plColor + formatDecimal(p.PnL.Total.Amount.Amount) + "[-]",
			plColor + formatRate(p.PnL.Total.Rate) + "[-]",
			dayColor + formatRate(p.PnL.Daily.Rate) + "[-]",
		}
		for c, v := range cells {
			cell := tview.NewTableCell(v).SetExpansion(boolToInt(c == 1))
			if c >= 2 {
				cell.SetAlign(tview.AlignRight)
			}
			s.table.SetCell(row, c, cell)
		}
	}

	if len(positions) == 0 {
		s.table.SetCell(1, 0, tview.NewTableCell("no holdings").SetSelectable(false))
	}
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// AddHoldingsScreen creates a holdings screen backed by provider, registers it
// under name, and returns it.
func (a *App) AddHoldingsScreen(name string, provider HoldingsProvider) *HoldingsScreen {
	s := newHoldingsScreen(a.app, provider)
	a.AddScreen(name, s)
	return s
}
