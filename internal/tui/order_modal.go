package tui

import (
	"fmt"
	"strings"

	"github.com/byunjuneseok/s8s/internal/domain"
	"github.com/rivo/tview"
	"github.com/shopspring/decimal"
)

// orderFormValues holds the raw strings collected by the order form, before
// validation and conversion into a domain.OrderRequest.
type orderFormValues struct {
	Symbol   string
	Side     string // "BUY" | "SELL"
	Type     string // "LIMIT" | "MARKET"
	Basis    string // "QUANTITY" | "AMOUNT"
	Quantity string // used when Basis == QUANTITY
	Amount   string // used when Basis == AMOUNT
	Currency string // currency for Amount and Price (e.g. "KRW")
	Price    string // used when Type == LIMIT
	TIF      string // "", "DAY", "CLS"
}

// Option lists for the form's drop-downs.
var (
	orderSideOptions  = []string{string(domain.Buy), string(domain.Sell)}
	orderTypeOptions  = []string{string(domain.LimitOrder), string(domain.MarketOrder)}
	orderBasisOptions = []string{string(domain.QuantityBased), string(domain.AmountBased)}
	orderTIFOptions   = []string{"DEFAULT", string(domain.Day), string(domain.Closing)}
	orderCurrencyOpts = []string{string(domain.KRW), string(domain.USD)}
)

// buildOrderRequest converts raw form values into a validated domain.OrderRequest.
// It is pure and side-effect free so it can be unit-tested directly.
func buildOrderRequest(v orderFormValues) (domain.OrderRequest, error) {
	req := domain.OrderRequest{
		Symbol: strings.TrimSpace(v.Symbol),
		Side:   domain.Side(v.Side),
		Type:   domain.OrderType(v.Type),
		Basis:  domain.OrderBasis(v.Basis),
	}

	currency := domain.Currency(v.Currency)
	if currency == "" {
		currency = domain.KRW
	}

	switch v.TIF {
	case "", "DEFAULT":
		req.TimeInForce = ""
	default:
		req.TimeInForce = domain.TimeInForce(v.TIF)
	}

	switch req.Basis {
	case domain.QuantityBased:
		qty, err := decimal.NewFromString(strings.TrimSpace(v.Quantity))
		if err != nil {
			return domain.OrderRequest{}, fmt.Errorf("order: invalid quantity %q: %w", v.Quantity, err)
		}
		req.Quantity = qty
	case domain.AmountBased:
		amt, err := decimal.NewFromString(strings.TrimSpace(v.Amount))
		if err != nil {
			return domain.OrderRequest{}, fmt.Errorf("order: invalid amount %q: %w", v.Amount, err)
		}
		req.Amount = domain.NewMoney(amt, currency)
	default:
		return domain.OrderRequest{}, fmt.Errorf("order: invalid basis %q", v.Basis)
	}

	if req.Type == domain.LimitOrder {
		price, err := decimal.NewFromString(strings.TrimSpace(v.Price))
		if err != nil {
			return domain.OrderRequest{}, fmt.Errorf("order: invalid price %q: %w", v.Price, err)
		}
		req.Price = domain.NewMoney(price, currency)
	}

	if err := req.Validate(); err != nil {
		return domain.OrderRequest{}, err
	}
	return req, nil
}

// orderSummary renders a human-readable one-block summary of a request for the
// confirmation modal.
func orderSummary(req domain.OrderRequest) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s %s %s\n", req.Side, req.Type, req.Symbol)
	switch req.Basis {
	case domain.QuantityBased:
		fmt.Fprintf(&b, "quantity: %s\n", formatDecimal(req.Quantity))
	case domain.AmountBased:
		fmt.Fprintf(&b, "amount: %s\n", formatMoney(req.Amount))
	}
	if req.Type == domain.LimitOrder {
		fmt.Fprintf(&b, "price: %s\n", formatMoney(req.Price))
	}
	if req.TimeInForce != "" {
		fmt.Fprintf(&b, "tif: %s\n", req.TimeInForce)
	}
	return strings.TrimRight(b.String(), "\n")
}

// OrderModal is a two-step order entry: a form that builds a domain.OrderRequest,
// then a confirmation modal showing the request (and an injected estimate)
// before OnConfirm fires.
type OrderModal struct {
	app   *App
	pages *tview.Pages
	form  *tview.Form

	// OnConfirm, if set, receives the request once the user confirms.
	OnConfirm func(domain.OrderRequest)
	// Estimate, if set, returns a one-line cost/fee estimate for the request,
	// shown on the confirmation modal.
	Estimate func(domain.OrderRequest) string

	// pageName is the root page the modal is mounted under.
	pageName string
}

// newOrderModal builds an order modal mounted on app's root pages under pageName.
func newOrderModal(app *App, pageName string) *OrderModal {
	m := &OrderModal{
		app:      app,
		pages:    tview.NewPages(),
		pageName: pageName,
	}
	m.form = tview.NewForm().
		AddInputField("Symbol", "", 20, nil, nil).
		AddDropDown("Side", orderSideOptions, 0, nil).
		AddDropDown("Type", orderTypeOptions, 0, nil).
		AddDropDown("Basis", orderBasisOptions, 0, nil).
		AddInputField("Quantity", "", 20, nil, nil).
		AddInputField("Amount", "", 20, nil, nil).
		AddDropDown("Currency", orderCurrencyOpts, 0, nil).
		AddInputField("Price", "", 20, nil, nil).
		AddDropDown("TIF", orderTIFOptions, 0, nil)
	m.form.SetBorder(true).SetTitle(" new order ")

	m.form.AddButton("Review", func() { m.review() })
	m.form.AddButton("Cancel", func() { m.close() })

	m.pages.AddPage("form", center(m.form, 50, 17), true, true)
	return m
}

// formValues reads the current widget state into an orderFormValues.
func (m *OrderModal) formValues() orderFormValues {
	getInput := func(label string) string {
		item := m.form.GetFormItemByLabel(label)
		if f, ok := item.(*tview.InputField); ok {
			return f.GetText()
		}
		return ""
	}
	getDrop := func(label string) string {
		item := m.form.GetFormItemByLabel(label)
		if d, ok := item.(*tview.DropDown); ok {
			_, opt := d.GetCurrentOption()
			return opt
		}
		return ""
	}
	return orderFormValues{
		Symbol:   getInput("Symbol"),
		Side:     getDrop("Side"),
		Type:     getDrop("Type"),
		Basis:    getDrop("Basis"),
		Quantity: getInput("Quantity"),
		Amount:   getInput("Amount"),
		Currency: getDrop("Currency"),
		Price:    getInput("Price"),
		TIF:      getDrop("TIF"),
	}
}

func (m *OrderModal) review() {
	req, err := buildOrderRequest(m.formValues())
	if err != nil {
		m.showMessage(fmt.Sprintf("[red]%v[-]", err))
		return
	}
	m.showConfirm(req)
}

// confirmText builds the body of the confirmation modal for req, including the
// estimate line and a market-order warning when applicable.
func (m *OrderModal) confirmText(req domain.OrderRequest) string {
	var b strings.Builder
	b.WriteString(orderSummary(req))
	if m.Estimate != nil {
		if est := m.Estimate(req); est != "" {
			fmt.Fprintf(&b, "\n%s", est)
		}
	}
	if req.Type == domain.MarketOrder {
		b.WriteString("\n\n[yellow]warning:[-] market order — fills at the prevailing price, which may differ from the quote.")
	}
	return b.String()
}

func (m *OrderModal) showConfirm(req domain.OrderRequest) {
	modal := tview.NewModal().
		SetText("Confirm order?\n\n" + m.confirmText(req)).
		AddButtons([]string{"Confirm", "Back"}).
		SetDoneFunc(func(_ int, label string) {
			if label == "Confirm" {
				if m.OnConfirm != nil {
					m.OnConfirm(req)
				}
				m.close()
				return
			}
			m.pages.RemovePage("confirm")
			m.app.app.SetFocus(m.form)
		})
	m.pages.AddPage("confirm", modal, true, true)
	m.app.app.SetFocus(modal)
}

func (m *OrderModal) showMessage(text string) {
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

// Show mounts the modal on the app root and focuses the form.
func (m *OrderModal) Show() {
	m.app.root.AddPage(m.pageName, m.pages, true, true)
	m.app.app.SetFocus(m.form)
}

func (m *OrderModal) close() {
	m.app.root.RemovePage(m.pageName)
	m.app.app.SetFocus(m.app.body)
}

// center wraps a primitive in a fixed-size centered flex, leaving the
// surrounding area transparent.
func center(p tview.Primitive, width, height int) tview.Primitive {
	return tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(p, height, 0, true).
			AddItem(nil, 0, 1, false), width, 0, true).
		AddItem(nil, 0, 1, false)
}

// NewOrderModal creates an order entry modal mounted under name on the app's
// root pages. Call Show to display it.
func (a *App) NewOrderModal(name string) *OrderModal {
	return newOrderModal(a, name)
}
