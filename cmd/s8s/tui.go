package main

import (
	"fmt"
	"io"

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

	if err := app.Run(); err != nil {
		_, _ = fmt.Fprintf(stderr, "s8s: %v\n", err)
		return 1
	}
	return 0
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
