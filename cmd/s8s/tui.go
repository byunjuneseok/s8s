package main

import (
	"fmt"
	"io"

	"github.com/byunjuneseok/s8s/internal/broker/toss"
	"github.com/byunjuneseok/s8s/internal/config"
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

	app := tui.NewApp()
	label := contextLabel(cfg)
	app.SetContextLabel(func() string { return label })

	provider, perr := buildHoldingsProvider(cfg)
	switch {
	case perr != nil:
		app.AddMessageScreen("holdings", fmt.Sprintf("config error: %v", perr))
	case provider == nil:
		app.AddMessageScreen("holdings", "No context configured.\n\nRun:  s8s configure")
	default:
		hs := app.AddHoldingsScreen("holdings", provider)
		app.RegisterCommand("holdings", func([]string) { app.Show("holdings") })
		app.RegisterCommand("refresh", func([]string) { hs.Refresh() })
		hs.Refresh()
	}

	if err := app.Run(); err != nil {
		_, _ = fmt.Fprintf(stderr, "s8s: %v\n", err)
		return 1
	}
	return 0
}

// buildHoldingsProvider constructs a broker for the current context, or returns
// (nil, nil) when no context is configured yet.
func buildHoldingsProvider(cfg *config.Config) (tui.HoldingsProvider, error) {
	if cfg.CurrentContext == "" {
		return nil, nil
	}
	r, err := cfg.Resolve("")
	if err != nil {
		return nil, err
	}
	switch r.Broker.Type {
	case "toss", "":
		return toss.NewClient(r.Credential.ClientID, r.Credential.ClientSecret), nil
	default:
		return nil, fmt.Errorf("unsupported broker type %q", r.Broker.Type)
	}
}

// contextLabel renders the current context for the status bar.
func contextLabel(cfg *config.Config) string {
	if cfg.CurrentContext == "" {
		return "no context (run: s8s configure)"
	}
	for _, c := range cfg.Contexts {
		if c.Name == cfg.CurrentContext {
			if c.ReadOnly {
				return c.Name + " (read-only)"
			}
			return c.Name
		}
	}
	return cfg.CurrentContext
}
