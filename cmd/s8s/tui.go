package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/byunjuneseok/s8s/internal/broker"
	"github.com/byunjuneseok/s8s/internal/broker/toss"
	"github.com/byunjuneseok/s8s/internal/config"
	"github.com/byunjuneseok/s8s/internal/domain"
	"github.com/byunjuneseok/s8s/internal/poll"
	"github.com/byunjuneseok/s8s/internal/session"
	"github.com/byunjuneseok/s8s/internal/tui"
)

// runTUI launches the interactive terminal UI.
func runTUI(stderr io.Writer) int {
	path, err := config.DefaultPath()
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "s8s: %v\n", err)
		return 1
	}
	cfg, err := loadOrEmpty(path)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "s8s: %v\n", err)
		return 1
	}

	mgr, err := session.NewManager(cfg, brokerFactory)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "s8s: %v\n", err)
		return 1
	}

	app := tui.NewApp()
	app.SetContextLabel(func() string { return managerLabel(mgr) })

	lv := &live{mgr: mgr}

	// The home screen is the default view only when there is no active context.
	current := "holdings"
	if mgr.Current() == "" {
		app.AddMessageScreen("home",
			"No active context.\n\nRun  s8s configure  to add one,\nor  :ctx use <name>  if you already have contexts.")
		current = "home"
	}

	holdings := app.AddHoldingsScreen("holdings", lv)
	watch := app.AddWatchlistScreen("watch", lv, cfg.Watchlist)
	book := app.AddOrderbookScreen("orderbook", lv)
	orders := app.AddOrdersScreen("orders", lv)
	ctxScreen := app.AddCtxListScreen("ctx", mgr.Contexts(), mgr.Current())
	acctScreen := app.AddAccountsScreen("account", nil, 0)
	orderModal := app.NewOrderModal("order")
	modifyModal := app.NewModifyModal("modify")

	// Polling: one job per live screen; only the visible screen polls.
	sched := poll.New()
	pollNames := []string{"holdings", "watch", "orderbook", "orders"}
	register := func(name string, interval time.Duration, refresh func()) {
		sched.Register(name, interval, func(context.Context) error {
			refresh()
			if lv.RateLimited() {
				return poll.ErrThrottled
			}
			return nil
		})
		sched.Pause(name)
	}
	register("holdings", 5*time.Second, holdings.Refresh)
	register("watch", 2*time.Second, watch.Refresh)
	register("orderbook", time.Second, book.Refresh)
	register("orders", 3*time.Second, orders.Refresh)

	activate := func(name string) {
		for _, n := range pollNames {
			if n == name {
				sched.Resume(n)
			} else {
				sched.Pause(n)
			}
		}
	}
	show := func(name string) {
		app.Show(name)
		current = name
		activate(name)
	}
	refreshCurrent := func() {
		switch current {
		case "holdings":
			holdings.Refresh()
		case "watch":
			watch.Refresh()
		case "orderbook":
			book.Refresh()
		case "orders":
			orders.Refresh()
		}
	}

	persistWatch := func(syms []string) {
		cfg.Watchlist = syms
		if err := config.Save(path, cfg); err != nil {
			app.Flash(fmt.Sprintf("save watchlist: %v", err))
		}
	}

	applyAccount := func(acct domain.Account) {
		lv.SetActiveSeq(acct.Seq)
		orders.SetAccount(acct)
		holdings.Refresh()
		orders.Refresh()
	}

	switchContext := func(name string) {
		if err := mgr.Use(name); err != nil {
			app.Flash(fmt.Sprintf("ctx: %v", err))
			return
		}
		lv.SetActiveSeq(0)
		_ = cfg.SetCurrentContext(name) // keep in-memory config consistent
		if err := config.SaveCurrentContext(path, name); err != nil {
			app.Flash(fmt.Sprintf("persist context: %v", err))
		}
		ctxScreen.SetContexts(mgr.Contexts(), mgr.Current())
		initActiveAccount(app, lv, orders)
		refreshCurrent()
	}

	ctxScreen.OnSelect = func(name string) {
		switchContext(name)
		show("holdings")
	}
	acctScreen.OnSelect = func(acct domain.Account) {
		applyAccount(acct)
		app.Flash("active account: " + acct.No)
		show("holdings")
	}

	orderModal.Estimate = estimateOrder
	orderModal.OnConfirm = func(req domain.OrderRequest) {
		if mgr.ReadOnly() {
			app.Flash("context is read-only — order blocked")
			return
		}
		go func() {
			ctx := context.Background()
			acct, err := lv.ActiveAccount(ctx)
			var ord domain.Order
			if err == nil {
				ord, err = lv.PlaceOrder(ctx, acct, req)
			}
			app.QueueUpdate(func() {
				if err != nil {
					app.Flash(fmt.Sprintf("order failed: %v", err))
					return
				}
				app.Flash("order placed: " + ord.ID)
				orders.Refresh()
			})
		}()
	}

	modifyModal.OnConfirm = func(orderID string, mod domain.OrderModification) {
		if mgr.ReadOnly() {
			app.Flash("context is read-only — modify blocked")
			return
		}
		go func() {
			ctx := context.Background()
			acct, err := lv.ActiveAccount(ctx)
			if err == nil {
				_, err = lv.ModifyOrder(ctx, acct, orderID, mod)
			}
			app.QueueUpdate(func() {
				if err != nil {
					app.Flash(fmt.Sprintf("modify failed: %v", err))
					return
				}
				app.Flash("order modified: " + orderID)
				orders.Refresh()
			})
		}()
	}

	orders.OnModify = func(orderID string) {
		if mgr.ReadOnly() {
			app.Flash("context is read-only — modify blocked")
			return
		}
		modifyModal.Show(orderID, domain.KRW)
	}
	orders.OnCancel = func(orderID string) {
		if mgr.ReadOnly() {
			app.Flash("context is read-only — cancel blocked")
			return
		}
		go func() {
			ctx := context.Background()
			acct, err := lv.ActiveAccount(ctx)
			if err == nil {
				_, err = lv.CancelOrder(ctx, acct, orderID)
			}
			app.QueueUpdate(func() {
				if err != nil {
					app.Flash(fmt.Sprintf("cancel failed: %v", err))
					return
				}
				app.Flash("order canceled: " + orderID)
				orders.Refresh()
			})
		}()
	}

	// Commands.
	app.RegisterCommand("holdings", func([]string) { show("holdings"); holdings.Refresh() })
	app.RegisterCommand("refresh", func([]string) { refreshCurrent() })
	app.RegisterCommand("watch", func(args []string) {
		switch {
		case len(args) == 0:
			show("watch")
			watch.Refresh()
		case args[0] == "add" && len(args) > 1:
			persistWatch(watch.AddSymbol(strings.ToUpper(args[1])))
			show("watch")
			watch.Refresh()
		case args[0] == "rm" && len(args) > 1:
			persistWatch(watch.RemoveSymbol(strings.ToUpper(args[1])))
			show("watch")
			watch.Refresh()
		default:
			app.Flash("usage: :watch [add|rm <symbol>]")
		}
	})
	orderbookCmd := func(args []string) {
		if len(args) == 0 {
			app.Flash("usage: :orderbook <symbol>")
			return
		}
		book.SetSymbol(strings.ToUpper(args[0]))
		show("orderbook")
		book.Refresh()
	}
	app.RegisterCommand("orderbook", orderbookCmd)
	app.RegisterCommand("ob", orderbookCmd)
	app.RegisterCommand("orders", func([]string) { show("orders"); orders.Refresh() })
	app.RegisterCommand("order", func([]string) {
		if mgr.ReadOnly() {
			app.Flash("context is read-only — orders disabled")
			return
		}
		orderModal.Show()
	})
	app.RegisterCommand("ctx", func(args []string) {
		if len(args) >= 2 && args[0] == "use" {
			switchContext(args[1])
			show("holdings")
			return
		}
		ctxScreen.SetContexts(mgr.Contexts(), mgr.Current())
		show("ctx")
	})
	accountCmd := func([]string) {
		show("account")
		go func() {
			accs, err := lv.Accounts(context.Background())
			app.QueueUpdate(func() {
				if err != nil {
					app.Flash(fmt.Sprintf("accounts: %v", err))
					return
				}
				seq := lv.ActiveSeq()
				if seq == 0 && len(accs) > 0 {
					seq = accs[0].Seq
				}
				acctScreen.SetAccounts(accs, seq)
			})
		}()
	}
	app.RegisterCommand("account", accountCmd)
	app.RegisterCommand("acct", accountCmd)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sched.Start(ctx)
	defer sched.Stop()

	if current == "holdings" {
		activate("holdings")
		holdings.Refresh()
		initActiveAccount(app, lv, orders)
	}

	if err := app.Run(); err != nil {
		_, _ = fmt.Fprintf(stderr, "s8s: %v\n", err)
		return 1
	}
	return 0
}

// initActiveAccount resolves the active account in the background and points the
// orders screen at it, so order listing works without a manual :account.
func initActiveAccount(app *tui.App, lv *live, orders *tui.OrdersScreen) {
	go func() {
		acct, err := lv.ActiveAccount(context.Background())
		if err != nil {
			return
		}
		app.QueueUpdate(func() { orders.SetAccount(acct) })
	}()
}

// estimateOrder returns a local pre-trade estimate for the confirmation modal.
// It does no network I/O so the confirm step never blocks the UI.
func estimateOrder(req domain.OrderRequest) string {
	switch req.Basis {
	case domain.AmountBased:
		return "est. amount: " + req.Amount.Amount.String() + " " + string(req.Amount.Currency)
	case domain.QuantityBased:
		if req.Type == domain.LimitOrder {
			notional := req.Quantity.Mul(req.Price.Amount)
			return "est. amount: " + notional.String() + " " + string(req.Price.Currency)
		}
	}
	return ""
}

// brokerFactory builds a concrete broker for a resolved context, resolving any
// ${env:...}/keychain secret references in the credential.
func brokerFactory(r config.Resolved) (broker.Broker, error) {
	switch r.Broker.Type {
	case "toss", "":
		id, _, err := config.ResolveSecret(r.Credential.ClientID)
		if err != nil {
			return nil, fmt.Errorf("client id: %w", err)
		}
		secret, _, err := config.ResolveSecret(r.Credential.ClientSecret)
		if err != nil {
			return nil, fmt.Errorf("client secret: %w", err)
		}
		return toss.NewClient(id, secret), nil
	default:
		return nil, fmt.Errorf("unsupported broker type %q", r.Broker.Type)
	}
}

// managerLabel renders the active context for the status bar.
func managerLabel(mgr *session.Manager) string {
	cur := mgr.Current()
	if cur == "" {
		return "no context (run: s8s configure)"
	}
	if mgr.ReadOnly() {
		return cur + " (read-only)"
	}
	return cur
}

// live is a context-aware broker adapter shared by every screen. It always
// delegates to the session manager's currently active broker, so a :ctx switch
// instantly repoints all screens, and it applies the runtime-selected active
// account by ordering it first in Accounts.
type live struct {
	mgr *session.Manager

	mu        sync.Mutex
	activeSeq int64
}

// Compile-time proof that live satisfies every screen provider interface.
var (
	_ tui.HoldingsProvider  = (*live)(nil)
	_ tui.WatchlistProvider = (*live)(nil)
	_ tui.OrderbookProvider = (*live)(nil)
	_ tui.OrdersProvider    = (*live)(nil)
)

func (l *live) broker() (broker.Broker, error) {
	b := l.mgr.Broker()
	if b == nil {
		return nil, errors.New("no active context — run s8s configure or :ctx use <name>")
	}
	return b, nil
}

// SetActiveSeq selects the active account by its sequence (0 clears it).
func (l *live) SetActiveSeq(seq int64) {
	l.mu.Lock()
	l.activeSeq = seq
	l.mu.Unlock()
}

// ActiveSeq returns the selected active account sequence (0 if none).
func (l *live) ActiveSeq() int64 {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.activeSeq
}

func (l *live) Accounts(ctx context.Context) ([]domain.Account, error) {
	b, err := l.broker()
	if err != nil {
		return nil, err
	}
	accs, err := b.Accounts(ctx)
	if err != nil {
		return nil, err
	}
	if seq := l.ActiveSeq(); seq != 0 {
		var head, tail []domain.Account
		for _, a := range accs {
			if a.Seq == seq {
				head = append(head, a)
			} else {
				tail = append(tail, a)
			}
		}
		accs = append(head, tail...)
	}
	return accs, nil
}

// ActiveAccount returns the currently selected account, or the first account
// when none has been selected.
func (l *live) ActiveAccount(ctx context.Context) (domain.Account, error) {
	accs, err := l.Accounts(ctx)
	if err != nil {
		return domain.Account{}, err
	}
	if len(accs) == 0 {
		return domain.Account{}, errors.New("no accounts on this context")
	}
	return accs[0], nil
}

func (l *live) Holdings(ctx context.Context, acct domain.Account) (domain.HoldingsOverview, error) {
	b, err := l.broker()
	if err != nil {
		return domain.HoldingsOverview{}, err
	}
	return b.Holdings(ctx, acct)
}

func (l *live) Prices(ctx context.Context, symbols []string) ([]domain.Quote, error) {
	b, err := l.broker()
	if err != nil {
		return nil, err
	}
	return b.Prices(ctx, symbols)
}

func (l *live) Orderbook(ctx context.Context, symbol string) (domain.Orderbook, error) {
	b, err := l.broker()
	if err != nil {
		return domain.Orderbook{}, err
	}
	return b.Orderbook(ctx, symbol)
}

func (l *live) Orders(ctx context.Context, acct domain.Account) ([]domain.Order, error) {
	b, err := l.broker()
	if err != nil {
		return nil, err
	}
	return b.Orders(ctx, acct)
}

func (l *live) Commission(ctx context.Context, acct domain.Account, req domain.OrderRequest) (domain.Commission, error) {
	b, err := l.broker()
	if err != nil {
		return domain.Commission{}, err
	}
	return b.Commission(ctx, acct, req)
}

func (l *live) PlaceOrder(ctx context.Context, acct domain.Account, req domain.OrderRequest) (domain.Order, error) {
	b, err := l.broker()
	if err != nil {
		return domain.Order{}, err
	}
	return b.PlaceOrder(ctx, acct, req)
}

func (l *live) ModifyOrder(ctx context.Context, acct domain.Account, orderID string, mod domain.OrderModification) (domain.Order, error) {
	b, err := l.broker()
	if err != nil {
		return domain.Order{}, err
	}
	return b.ModifyOrder(ctx, acct, orderID, mod)
}

func (l *live) CancelOrder(ctx context.Context, acct domain.Account, orderID string) (domain.Order, error) {
	b, err := l.broker()
	if err != nil {
		return domain.Order{}, err
	}
	return b.CancelOrder(ctx, acct, orderID)
}

// RateLimited reports whether the active broker's most recent responses show an
// exhausted rate-limit bucket, so the poller can back off.
func (l *live) RateLimited() bool {
	rl, ok := l.mgr.Broker().(interface{ RateLimits() map[string]string })
	if !ok {
		return false
	}
	for k, v := range rl.RateLimits() {
		if !strings.Contains(strings.ToLower(k), "remaining") {
			continue
		}
		if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil && n <= 0 {
			return true
		}
	}
	return false
}
