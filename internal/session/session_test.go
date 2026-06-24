package session

import (
	"errors"
	"testing"

	"github.com/byunjuneseok/s8s/internal/broker"
	"github.com/byunjuneseok/s8s/internal/broker/brokertest"
	"github.com/byunjuneseok/s8s/internal/config"
)

func testConfig() *config.Config {
	return &config.Config{
		CurrentContext: "main",
		Contexts: []config.Context{
			{Name: "main", Broker: "real", ReadOnly: false},
			{Name: "paper", Broker: "paper", ReadOnly: true},
		},
		Brokers: []config.BrokerConfig{
			{Name: "real", Type: "toss", Credentials: "prod"},
			{Name: "paper", Type: "toss", Credentials: "paper"},
		},
		Credentials: []config.Credential{
			{Name: "prod", ClientID: "id-prod"},
			{Name: "paper", ClientID: "id-paper"},
		},
	}
}

// recordingFactory returns a stub broker and records which credential it was
// asked to build for.
func recordingFactory(seen *[]string) BrokerFactory {
	return func(r config.Resolved) (broker.Broker, error) {
		*seen = append(*seen, r.Credential.ClientID)
		return brokertest.Stub{}, nil
	}
}

func TestNewManagerActivatesCurrentContext(t *testing.T) {
	var seen []string
	m, err := NewManager(testConfig(), recordingFactory(&seen))
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	if m.Current() != "main" {
		t.Errorf("Current() = %q, want main", m.Current())
	}
	if m.Broker() == nil {
		t.Error("Broker() = nil, want active broker")
	}
	if len(seen) != 1 || seen[0] != "id-prod" {
		t.Errorf("factory built for %v, want [id-prod]", seen)
	}
}

func TestUseSwitchesContext(t *testing.T) {
	var seen []string
	m, err := NewManager(testConfig(), recordingFactory(&seen))
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}

	if err := m.Use("paper"); err != nil {
		t.Fatalf("Use(paper): %v", err)
	}
	if m.Current() != "paper" {
		t.Errorf("Current() = %q, want paper", m.Current())
	}
	if !m.ReadOnly() {
		t.Error("ReadOnly() = false, want true for paper context")
	}
	if got := seen[len(seen)-1]; got != "id-paper" {
		t.Errorf("last factory build = %q, want id-paper", got)
	}
}

func TestContexts(t *testing.T) {
	m, err := NewManager(testConfig(), recordingFactory(new([]string)))
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	got := m.Contexts()
	if len(got) != 2 || got[0] != "main" || got[1] != "paper" {
		t.Errorf("Contexts() = %v, want [main paper]", got)
	}
}

func TestUseUnknownContextKeepsCurrent(t *testing.T) {
	var seen []string
	m, err := NewManager(testConfig(), recordingFactory(&seen))
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	if err := m.Use("ghost"); err == nil {
		t.Error("Use(ghost) = nil error, want error")
	}
	if m.Current() != "main" {
		t.Errorf("Current() = %q after failed switch, want main", m.Current())
	}
}

func TestFactoryErrorDoesNotSwap(t *testing.T) {
	failing := func(config.Resolved) (broker.Broker, error) {
		return nil, errors.New("boom")
	}
	cfg := testConfig()
	cfg.CurrentContext = "" // start inactive so NewManager succeeds
	m, err := NewManager(cfg, failing)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	if err := m.Use("main"); err == nil {
		t.Error("Use with failing factory = nil error, want error")
	}
	if m.Current() != "" || m.Broker() != nil {
		t.Errorf("state changed after factory failure: current=%q broker=%v", m.Current(), m.Broker())
	}
}
