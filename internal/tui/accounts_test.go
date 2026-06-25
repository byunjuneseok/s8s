package tui

import (
	"strings"
	"testing"

	"github.com/byunjuneseok/s8s/internal/domain"
)

func sampleAccounts() []domain.Account {
	return []domain.Account{
		{No: "111-11", Seq: 1, Type: domain.Brokerage},
		{No: "222-22", Seq: 2, Type: domain.Brokerage},
	}
}

func TestAccountsSetAccounts(t *testing.T) {
	s := newAccountsScreen(sampleAccounts(), 2)
	if got := s.list.GetItemCount(); got != 2 {
		t.Errorf("item count = %d, want 2", got)
	}
	// The active account (Seq 2, row 1) is highlighted.
	main, _ := s.list.GetItemText(1)
	if !strings.Contains(main, "active") {
		t.Errorf("active row = %q, want it marked active", main)
	}
	// The inactive one is not.
	other, _ := s.list.GetItemText(0)
	if strings.Contains(other, "active") {
		t.Errorf("inactive row = %q, should not be marked active", other)
	}

	// Updater repopulates with a new active seq.
	s.SetAccounts(sampleAccounts()[:1], 1)
	if got := s.list.GetItemCount(); got != 1 {
		t.Errorf("after update count = %d, want 1", got)
	}
}

func TestAccountsEmpty(t *testing.T) {
	s := newAccountsScreen(nil, 0)
	if got := s.list.GetItemCount(); got != 1 {
		t.Errorf("empty count = %d, want 1 placeholder", got)
	}
}

func TestAccountsOnSelect(t *testing.T) {
	s := newAccountsScreen(sampleAccounts(), 1)
	var picked domain.Account
	s.OnSelect = func(a domain.Account) { picked = a }

	if fn := s.list.GetItemSelectedFunc(1); fn != nil {
		fn()
	}
	if picked.Seq != 2 {
		t.Errorf("OnSelect got seq %d, want 2", picked.Seq)
	}
}

func TestAddAccountsScreen(t *testing.T) {
	a := NewApp()
	s := a.AddAccountsScreen("accounts", sampleAccounts(), 1)
	if s == nil || !a.body.HasPage("accounts") {
		t.Fatal("AddAccountsScreen did not register screen")
	}
}
