package tui

import (
	"strings"
	"testing"
)

func TestCtxListSetContexts(t *testing.T) {
	s := newCtxListScreen([]string{"a", "b", "c"}, "b")
	if got := s.list.GetItemCount(); got != 3 {
		t.Errorf("item count = %d, want 3", got)
	}
	// Current context is highlighted.
	main, _ := s.list.GetItemText(1)
	if !strings.Contains(main, "current") {
		t.Errorf("current item = %q, want it marked current", main)
	}

	// Updater repopulates.
	s.SetContexts([]string{"x"}, "x")
	if got := s.list.GetItemCount(); got != 1 {
		t.Errorf("after update count = %d, want 1", got)
	}
}

func TestCtxListEmpty(t *testing.T) {
	s := newCtxListScreen(nil, "")
	if got := s.list.GetItemCount(); got != 1 {
		t.Errorf("empty list count = %d, want 1 placeholder", got)
	}
}

func TestCtxListOnSelect(t *testing.T) {
	s := newCtxListScreen([]string{"a", "b"}, "a")
	var picked string
	s.OnSelect = func(name string) { picked = name }

	// Invoke the per-item handler for row 1, the way Enter would.
	if fn := s.list.GetItemSelectedFunc(1); fn != nil {
		fn()
	}
	if picked != "b" {
		t.Errorf("OnSelect got %q, want b", picked)
	}
}

func TestAddCtxListScreen(t *testing.T) {
	a := NewApp()
	s := a.AddCtxListScreen("contexts", []string{"a"}, "a")
	if s == nil || !a.body.HasPage("contexts") {
		t.Fatal("AddCtxListScreen did not register screen")
	}
}
