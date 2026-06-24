package tui

import (
	"fmt"
	"strings"

	"github.com/byunjuneseok/s8s/internal/domain"
	"github.com/rivo/tview"
	"github.com/shopspring/decimal"
)

// ModifyModal is a small two-field form (new price, new quantity) for amending
// an existing order. Blank fields are left unchanged. On confirm it fires
// OnConfirm with the order ID and the requested modification.
type ModifyModal struct {
	app      *App
	pages    *tview.Pages
	form     *tview.Form
	pageName string

	orderID  string
	currency domain.Currency

	// OnConfirm, if set, receives the order ID and modification on submit.
	OnConfirm func(orderID string, mod domain.OrderModification)
}

func newModifyModal(app *App, pageName string) *ModifyModal {
	return &ModifyModal{app: app, pageName: pageName}
}

// Show builds the form for orderID (prices interpreted in cur, defaulting to
// KRW) and displays it over the current screen.
func (m *ModifyModal) Show(orderID string, cur domain.Currency) {
	if cur == "" {
		cur = domain.KRW
	}
	m.orderID = orderID
	m.currency = cur

	m.form = tview.NewForm().
		AddInputField("New price", "", 20, nil, nil).
		AddInputField("New quantity", "", 20, nil, nil)
	m.form.SetBorder(true).SetTitle(fmt.Sprintf(" modify %s ", orderID))
	m.form.AddButton("Apply", func() { m.apply() })
	m.form.AddButton("Cancel", func() { m.close() })

	m.pages = tview.NewPages()
	m.pages.AddPage("form", center(m.form, 50, 9), true, true)
	m.app.root.AddPage(m.pageName, m.pages, true, true)
	m.app.app.SetFocus(m.form)
}

func (m *ModifyModal) inputText(label string) string {
	if f, ok := m.form.GetFormItemByLabel(label).(*tview.InputField); ok {
		return strings.TrimSpace(f.GetText())
	}
	return ""
}

func (m *ModifyModal) apply() {
	var mod domain.OrderModification

	if priceStr := m.inputText("New price"); priceStr != "" {
		p, err := decimal.NewFromString(priceStr)
		if err != nil {
			m.message(fmt.Sprintf("[red]invalid price %q: %v[-]", priceStr, err))
			return
		}
		mod.Price = domain.NewMoney(p, m.currency)
	}
	if qtyStr := m.inputText("New quantity"); qtyStr != "" {
		q, err := decimal.NewFromString(qtyStr)
		if err != nil {
			m.message(fmt.Sprintf("[red]invalid quantity %q: %v[-]", qtyStr, err))
			return
		}
		mod.Quantity = q
	}
	if mod.Price.Amount.IsZero() && mod.Quantity.IsZero() {
		m.message("[red]enter a new price and/or quantity[-]")
		return
	}

	if m.OnConfirm != nil {
		m.OnConfirm(m.orderID, mod)
	}
	m.close()
}

func (m *ModifyModal) message(text string) {
	modal := tview.NewModal().
		SetText(text).
		AddButtons([]string{"OK"}).
		SetDoneFunc(func(int, string) {
			m.pages.RemovePage("message")
			m.app.app.SetFocus(m.form)
		})
	m.pages.AddPage("message", modal, true, true)
	m.app.app.SetFocus(modal)
}

func (m *ModifyModal) close() {
	m.app.root.RemovePage(m.pageName)
	m.app.app.SetFocus(m.app.body)
}

// NewModifyModal creates an order-modify modal mounted under name on the app's
// root pages. Call Show to display it for a specific order.
func (a *App) NewModifyModal(name string) *ModifyModal {
	return newModifyModal(a, name)
}
