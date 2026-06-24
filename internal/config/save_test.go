package config

import (
	"path/filepath"
	"testing"
)

func TestSaveRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "config.yaml")

	c := &Config{
		CurrentContext: "main",
		Contexts:       []Context{{Name: "main", Broker: "b"}},
		Brokers:        []BrokerConfig{{Name: "b", Type: "toss", Credentials: "cr"}},
		Credentials:    []Credential{{Name: "cr", ClientID: "id", ClientSecret: "secret"}},
	}
	if err := Save(path, c); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load after save: %v", err)
	}
	if got.CurrentContext != "main" || len(got.Contexts) != 1 || got.Credentials[0].ClientSecret != "secret" {
		t.Errorf("round-trip mismatch: %+v", got)
	}
}

func TestSaveRejectsInvalid(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	// context references a broker that does not exist
	c := &Config{Contexts: []Context{{Name: "x", Broker: "ghost"}}}
	if err := Save(path, c); err == nil {
		t.Error("Save(invalid) = nil error, want error")
	}
}

func TestUpsert(t *testing.T) {
	c := &Config{}
	c.SetCredential(Credential{Name: "cr", ClientID: "a"})
	c.SetCredential(Credential{Name: "cr", ClientID: "b"}) // replace
	if len(c.Credentials) != 1 || c.Credentials[0].ClientID != "b" {
		t.Errorf("SetCredential upsert failed: %+v", c.Credentials)
	}

	c.SetBroker(BrokerConfig{Name: "b1", Type: "toss"})
	c.SetBroker(BrokerConfig{Name: "b1", Type: "toss2"}) // replace
	if len(c.Brokers) != 1 || c.Brokers[0].Type != "toss2" {
		t.Errorf("SetBroker upsert failed: %+v", c.Brokers)
	}

	c.SetContext(Context{Name: "c1", Broker: "b1"})
	c.SetContext(Context{Name: "c2", Broker: "b1"}) // new
	if len(c.Contexts) != 2 {
		t.Errorf("SetContext should have 2 entries: %+v", c.Contexts)
	}
}
