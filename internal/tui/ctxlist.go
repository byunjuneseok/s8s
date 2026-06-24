package tui

import (
	"fmt"

	"github.com/rivo/tview"
)

// CtxListScreen lists the configured context names, highlighting the current
// one. Pressing Enter on a row fires OnSelect with that context's name.
type CtxListScreen struct {
	*tview.Flex

	list *tview.List

	// OnSelect, if set, is called with the chosen context name on Enter.
	OnSelect func(name string)
}

func newCtxListScreen(names []string, current string) *CtxListScreen {
	s := &CtxListScreen{
		list: tview.NewList().ShowSecondaryText(false),
	}
	s.list.SetBorder(true).SetTitle(" contexts ")

	s.Flex = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(s.list, 0, 1, true)

	s.SetContexts(names, current)
	return s
}

// SetContexts repopulates the list with names, marking current as active.
func (s *CtxListScreen) SetContexts(names []string, current string) {
	s.list.Clear()
	for _, name := range names {
		var label string
		if name == current {
			label = fmt.Sprintf("[green]* %s (current)[-]", name)
		} else {
			label = "  " + name
		}
		selected := name // capture for closure
		s.list.AddItem(label, "", 0, func() {
			if s.OnSelect != nil {
				s.OnSelect(selected)
			}
		})
	}
	if len(names) == 0 {
		s.list.AddItem("no contexts configured", "", 0, nil)
	}
}

// AddCtxListScreen creates a context-list screen seeded with names (current
// highlighted), registers it under screenName, and returns it.
func (a *App) AddCtxListScreen(screenName string, names []string, current string) *CtxListScreen {
	s := newCtxListScreen(names, current)
	a.AddScreen(screenName, s)
	return s
}
