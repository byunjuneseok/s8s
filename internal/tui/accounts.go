package tui

import (
	"fmt"

	"github.com/byunjuneseok/s8s/internal/domain"
	"github.com/rivo/tview"
)

// AccountsScreen lists the broker accounts on the active context, highlighting
// the active one. Pressing Enter on a row fires OnSelect with that account.
type AccountsScreen struct {
	*tview.Flex

	list     *tview.List
	accounts []domain.Account

	// OnSelect, if set, is called with the chosen account on Enter.
	OnSelect func(domain.Account)
}

func newAccountsScreen(accts []domain.Account, activeSeq int64) *AccountsScreen {
	s := &AccountsScreen{
		list: tview.NewList().ShowSecondaryText(false),
	}
	s.list.SetBorder(true).SetTitle(" accounts ")

	s.Flex = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(s.list, 0, 1, true)

	s.SetAccounts(accts, activeSeq)
	return s
}

// SetAccounts repopulates the list, marking the account whose Seq equals
// activeSeq as active.
func (s *AccountsScreen) SetAccounts(accts []domain.Account, activeSeq int64) {
	s.accounts = append([]domain.Account(nil), accts...)
	s.list.Clear()
	for i, acct := range s.accounts {
		row := fmt.Sprintf("%s  %s  seq=%d", acct.No, acct.Type, acct.Seq)
		if acct.Seq == activeSeq {
			row = fmt.Sprintf("[green]* %s (active)[-]", row)
		} else {
			row = "  " + row
		}
		idx := i // capture for closure
		s.list.AddItem(row, "", 0, func() {
			if s.OnSelect != nil {
				s.OnSelect(s.accounts[idx])
			}
		})
	}
	if len(s.accounts) == 0 {
		s.list.AddItem("no accounts on this context", "", 0, nil)
	}
}

// AddAccountsScreen creates an accounts screen seeded with accts (the account
// matching activeSeq highlighted), registers it under name, and returns it.
func (a *App) AddAccountsScreen(name string, accts []domain.Account, activeSeq int64) *AccountsScreen {
	s := newAccountsScreen(accts, activeSeq)
	a.AddScreen(name, s)
	return s
}
