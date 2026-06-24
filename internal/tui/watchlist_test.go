package tui

import (
	"context"
	"testing"

	"github.com/byunjuneseok/s8s/internal/domain"
)

type fakeWatchlistProvider struct {
	quotes []domain.Quote
	err    error
	gotSym []string
}

func (f *fakeWatchlistProvider) Prices(_ context.Context, symbols []string) ([]domain.Quote, error) {
	f.gotSym = symbols
	return f.quotes, f.err
}

func newTestWatchlist(symbols []string) *WatchlistScreen {
	return newWatchlistScreen(NewApp().app, &fakeWatchlistProvider{}, symbols)
}

func TestWatchlistRender(t *testing.T) {
	s := newTestWatchlist([]string{"005930"})
	quotes := []domain.Quote{
		{Symbol: "005930", LastPrice: domain.NewMoney(dec("70000"), domain.KRW)},
	}
	s.render(quotes) // must not panic
	if got := s.table.GetCell(1, 0).Text; got != "005930" {
		t.Errorf("symbol cell = %q, want 005930", got)
	}
	// Empty render path.
	s.render(nil)
}

func TestWatchlistAddRemoveSymbol(t *testing.T) {
	s := newTestWatchlist([]string{"A", "B"})

	got := s.AddSymbol("C")
	if len(got) != 3 || got[2] != "C" {
		t.Errorf("AddSymbol = %v, want [A B C]", got)
	}
	// Duplicate is ignored.
	if got := s.AddSymbol("A"); len(got) != 3 {
		t.Errorf("duplicate AddSymbol changed list: %v", got)
	}

	got = s.RemoveSymbol("B")
	if len(got) != 2 || got[0] != "A" || got[1] != "C" {
		t.Errorf("RemoveSymbol = %v, want [A C]", got)
	}
	// Removing absent is a no-op.
	if got := s.RemoveSymbol("Z"); len(got) != 2 {
		t.Errorf("removing absent changed list: %v", got)
	}
}

func TestWatchlistSymbolsIsCopy(t *testing.T) {
	s := newTestWatchlist([]string{"A"})
	got := s.Symbols()
	got[0] = "MUTATED"
	if s.Symbols()[0] != "A" {
		t.Error("Symbols() returned a slice sharing backing array")
	}
}

func TestAddWatchlistScreen(t *testing.T) {
	a := NewApp()
	s := a.AddWatchlistScreen("watchlist", &fakeWatchlistProvider{}, []string{"A"})
	if s == nil {
		t.Fatal("AddWatchlistScreen returned nil")
	}
	if !a.body.HasPage("watchlist") {
		t.Error("watchlist screen not registered")
	}
}
