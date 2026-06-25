package tui

import (
	"context"
	"fmt"

	"github.com/byunjuneseok/s8s/internal/domain"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// WatchlistProvider is the slice of broker behaviour the watchlist screen needs.
type WatchlistProvider interface {
	Prices(ctx context.Context, symbols []string) ([]domain.Quote, error)
}

// WatchlistScreen shows a user-curated list of symbols and their last prices.
type WatchlistScreen struct {
	*tview.Flex
	app      *tview.Application
	provider WatchlistProvider

	table   *tview.Table
	symbols []string
}

func newWatchlistScreen(app *tview.Application, provider WatchlistProvider, symbols []string) *WatchlistScreen {
	s := &WatchlistScreen{
		app:      app,
		provider: provider,
		table:    tview.NewTable().SetBorders(false).SetFixed(1, 0).SetSelectable(true, false),
		symbols:  append([]string(nil), symbols...),
	}
	s.table.SetBorder(true).SetTitle(" watchlist ")

	s.Flex = tview.NewFlex().SetDirection(tview.FlexRow).
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

func (s *WatchlistScreen) setLoading() {
	s.table.Clear()
	s.table.SetCell(0, 0, tview.NewTableCell("loading…").SetSelectable(false))
}

// Symbols returns a copy of the current watchlist symbols.
func (s *WatchlistScreen) Symbols() []string {
	return append([]string(nil), s.symbols...)
}

// AddSymbol appends sym to the watchlist if it is not already present and
// returns the updated list (so callers can persist it to config).
func (s *WatchlistScreen) AddSymbol(sym string) []string {
	for _, existing := range s.symbols {
		if existing == sym {
			return s.Symbols()
		}
	}
	s.symbols = append(s.symbols, sym)
	return s.Symbols()
}

// RemoveSymbol drops sym from the watchlist if present and returns the updated
// list (so callers can persist it to config).
func (s *WatchlistScreen) RemoveSymbol(sym string) []string {
	out := s.symbols[:0]
	for _, existing := range s.symbols {
		if existing != sym {
			out = append(out, existing)
		}
	}
	s.symbols = out
	return s.Symbols()
}

// Refresh fetches prices for the current symbols in the background and redraws.
func (s *WatchlistScreen) Refresh() {
	symbols := s.Symbols()
	go func() {
		if len(symbols) == 0 {
			s.postMessage("watchlist is empty")
			return
		}
		quotes, err := s.provider.Prices(context.Background(), symbols)
		if err != nil {
			s.postError(fmt.Errorf("prices: %w", err))
			return
		}
		s.app.QueueUpdateDraw(func() { s.render(quotes) })
	}()
}

func (s *WatchlistScreen) postError(err error) {
	s.app.QueueUpdateDraw(func() {
		s.table.Clear()
		s.table.SetCell(0, 0, tview.NewTableCell(fmt.Sprintf("[red]error:[-] %v", err)).SetSelectable(false))
	})
}

func (s *WatchlistScreen) postMessage(msg string) {
	s.app.QueueUpdateDraw(func() {
		s.table.Clear()
		s.table.SetCell(0, 0, tview.NewTableCell(msg).SetSelectable(false))
	})
}

func (s *WatchlistScreen) render(quotes []domain.Quote) {
	s.table.Clear()
	headers := []string{"SYMBOL", "LAST"}
	for c, h := range headers {
		s.table.SetCell(0, c, tview.NewTableCell(h).
			SetTextColor(tcell.ColorYellow).
			SetSelectable(false))
	}

	for i, q := range quotes {
		row := i + 1
		s.table.SetCell(row, 0, tview.NewTableCell(q.Symbol))
		s.table.SetCell(row, 1, tview.NewTableCell(formatMoney(q.LastPrice)).SetAlign(tview.AlignRight))
	}

	if len(quotes) == 0 {
		s.table.SetCell(1, 0, tview.NewTableCell("no quotes").SetSelectable(false))
	}
}

// AddWatchlistScreen creates a watchlist screen seeded with symbols, registers
// it under name, and returns it.
func (a *App) AddWatchlistScreen(name string, provider WatchlistProvider, symbols []string) *WatchlistScreen {
	s := newWatchlistScreen(a.app, provider, symbols)
	a.AddScreen(name, s)
	return s
}
