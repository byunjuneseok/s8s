package tui

import (
	"testing"

	"github.com/byunjuneseok/s8s/internal/domain"
	"github.com/shopspring/decimal"
)

func dec(s string) decimal.Decimal { return decimal.RequireFromString(s) }

func TestFormatDecimal(t *testing.T) {
	cases := map[string]string{
		"0":       "0",
		"12":      "12",
		"100":     "100",
		"1000":    "1,000",
		"70000":   "70,000",
		"1234567": "1,234,567",
		"-5000":   "-5,000",
		"185.5":   "185.5",
		"1234.56": "1,234.56",
	}
	for in, want := range cases {
		if got := formatDecimal(dec(in)); got != want {
			t.Errorf("formatDecimal(%s) = %q, want %q", in, got, want)
		}
	}
}

func TestFormatRate(t *testing.T) {
	cases := map[string]string{
		"0.1077":  "+10.77%",
		"-0.05":   "-5.00%",
		"0":       "0.00%",
		"0.00005": "+0.01%",
	}
	for in, want := range cases {
		if got := formatRate(dec(in)); got != want {
			t.Errorf("formatRate(%s) = %q, want %q", in, got, want)
		}
	}
}

func TestFormatMoney(t *testing.T) {
	m := domain.NewMoney(dec("70000"), domain.KRW)
	if got, want := formatMoney(m), "70,000 KRW"; got != want {
		t.Errorf("formatMoney = %q, want %q", got, want)
	}
}

func TestPnlColor(t *testing.T) {
	if pnlColor(dec("1")) != "[red]" {
		t.Error("positive should be red (KR convention)")
	}
	if pnlColor(dec("-1")) != "[blue]" {
		t.Error("negative should be blue (KR convention)")
	}
	if pnlColor(dec("0")) != "[white]" {
		t.Error("zero should be white")
	}
}
