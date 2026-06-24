package domain

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestParseMoney(t *testing.T) {
	m, err := ParseMoney("70000.50", KRW)
	if err != nil {
		t.Fatalf("ParseMoney returned error: %v", err)
	}
	if !m.Amount.Equal(decimal.RequireFromString("70000.50")) {
		t.Errorf("amount = %s, want 70000.50", m.Amount)
	}
	if m.Currency != KRW {
		t.Errorf("currency = %s, want KRW", m.Currency)
	}

	if _, err := ParseMoney("not-a-number", USD); err == nil {
		t.Error("ParseMoney(invalid) = nil error, want error")
	}
}

func TestMoneyAdd(t *testing.T) {
	a := NewMoney(decimal.RequireFromString("100.10"), USD)
	b := NewMoney(decimal.RequireFromString("0.90"), USD)

	sum, err := a.Add(b)
	if err != nil {
		t.Fatalf("Add returned error: %v", err)
	}
	if !sum.Amount.Equal(decimal.RequireFromString("101.0")) {
		t.Errorf("sum = %s, want 101.0", sum.Amount)
	}

	if _, err := a.Add(NewMoney(decimal.NewFromInt(1), KRW)); err == nil {
		t.Error("Add(mismatched currency) = nil error, want error")
	}
}

func TestMoneyString(t *testing.T) {
	m := NewMoney(decimal.NewFromInt(42), USD)
	if got, want := m.String(), "42 USD"; got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}
