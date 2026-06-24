package tui

import (
	"github.com/byunjuneseok/s8s/internal/domain"
	"github.com/shopspring/decimal"
)

// pnlColor returns a tview color tag for a profit/loss value, following the
// Korean market convention: gains are red, losses are blue.
func pnlColor(d decimal.Decimal) string {
	switch {
	case d.IsPositive():
		return "[red]"
	case d.IsNegative():
		return "[blue]"
	default:
		return "[white]"
	}
}

// formatMoney renders an amount with its currency, e.g. "70,000 KRW".
func formatMoney(m domain.Money) string {
	return formatDecimal(m.Amount) + " " + string(m.Currency)
}

// formatRate renders a fractional rate as a percentage, e.g. 0.1077 -> "+10.77%".
func formatRate(d decimal.Decimal) string {
	pct := d.Mul(decimal.NewFromInt(100)).StringFixed(2)
	if d.IsPositive() {
		return "+" + pct + "%"
	}
	return pct + "%"
}

// formatDecimal renders a decimal with thousands separators on the integer part.
func formatDecimal(d decimal.Decimal) string {
	neg := d.IsNegative()
	s := d.Abs().String()

	intPart, frac := s, ""
	if i := indexByte(s, '.'); i >= 0 {
		intPart, frac = s[:i], s[i:]
	}

	grouped := groupThousands(intPart)
	if neg {
		grouped = "-" + grouped
	}
	return grouped + frac
}

func groupThousands(digits string) string {
	n := len(digits)
	if n <= 3 {
		return digits
	}
	lead := n % 3
	out := make([]byte, 0, n+n/3)
	if lead > 0 {
		out = append(out, digits[:lead]...)
	}
	for i := lead; i < n; i += 3 {
		if len(out) > 0 {
			out = append(out, ',')
		}
		out = append(out, digits[i:i+3]...)
	}
	return string(out)
}

func indexByte(s string, b byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == b {
			return i
		}
	}
	return -1
}
