// Package session manages the active configuration context and the broker
// instance bound to it. It is pure logic: it depends only on config and the
// broker interface, never on the UI or any network client. Concrete brokers are
// supplied through a BrokerFactory so this package stays adapter-agnostic.
package session

import (
	"errors"
	"fmt"

	"github.com/byunjuneseok/s8s/internal/broker"
	"github.com/byunjuneseok/s8s/internal/config"
)

// BrokerFactory builds a broker.Broker for a resolved context. It is injected so
// that session has no dependency on any concrete adapter.
type BrokerFactory func(config.Resolved) (broker.Broker, error)

// Manager holds the loaded config and the currently active context together
// with its broker.
type Manager struct {
	cfg     *config.Config
	factory BrokerFactory

	current  string
	resolved config.Resolved
	broker   broker.Broker
}

// NewManager creates a Manager. If the config has a current-context set, that
// context is activated immediately; otherwise no context is active until Use is
// called.
func NewManager(cfg *config.Config, factory BrokerFactory) (*Manager, error) {
	if cfg == nil {
		return nil, errors.New("session: nil config")
	}
	if factory == nil {
		return nil, errors.New("session: nil broker factory")
	}
	m := &Manager{cfg: cfg, factory: factory}
	if cfg.CurrentContext != "" {
		if err := m.Use(cfg.CurrentContext); err != nil {
			return nil, err
		}
	}
	return m, nil
}

// Contexts returns the names of all configured contexts, in file order.
func (m *Manager) Contexts() []string {
	names := make([]string, 0, len(m.cfg.Contexts))
	for _, c := range m.cfg.Contexts {
		names = append(names, c.Name)
	}
	return names
}

// Current returns the active context name, or "" if none is active.
func (m *Manager) Current() string { return m.current }

// Use switches the active context to name, building a fresh broker through the
// factory. If resolution or broker construction fails, the previously active
// context is left unchanged.
func (m *Manager) Use(name string) error {
	resolved, err := m.cfg.Resolve(name)
	if err != nil {
		return err
	}
	b, err := m.factory(resolved)
	if err != nil {
		return fmt.Errorf("session: build broker for context %q: %w", resolved.Context.Name, err)
	}
	m.current = resolved.Context.Name
	m.resolved = resolved
	m.broker = b
	return nil
}

// Broker returns the broker for the active context, or nil if none is active.
func (m *Manager) Broker() broker.Broker { return m.broker }

// Resolved returns the resolved active context.
func (m *Manager) Resolved() config.Resolved { return m.resolved }

// ReadOnly reports whether the active context forbids order operations.
func (m *Manager) ReadOnly() bool { return m.resolved.Context.ReadOnly }
