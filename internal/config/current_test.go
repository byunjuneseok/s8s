package config

import "testing"

func TestSaveCurrentContextSwitchesAndPreservesData(t *testing.T) {
	path := writeTemp(t, sampleYAML)

	if err := SaveCurrentContext(path, "toss-paper"); err != nil {
		t.Fatalf("SaveCurrentContext: %v", err)
	}

	c, err := Load(path)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if c.CurrentContext != "toss-paper" {
		t.Errorf("current-context = %q, want toss-paper", c.CurrentContext)
	}

	// All other data must survive the round-trip.
	if len(c.Contexts) != 2 || len(c.Brokers) != 2 || len(c.Credentials) != 2 {
		t.Fatalf("counts changed: %d ctx, %d brokers, %d creds", len(c.Contexts), len(c.Brokers), len(c.Credentials))
	}

	// Credentials (including secrets) intact.
	cred, ok := findCredential(c.Credentials, "toss-prod")
	if !ok {
		t.Fatal("credential toss-prod missing after save")
	}
	if cred.ClientID != "id-prod" || cred.ClientSecret != "secret-prod" {
		t.Errorf("credential = %+v, want id-prod/secret-prod", cred)
	}

	// Brokers intact, still resolving through to the right credential.
	r, err := c.Resolve("")
	if err != nil {
		t.Fatalf("Resolve(current): %v", err)
	}
	if r.Context.Name != "toss-paper" || r.Broker.Name != "paper" || r.Credential.ClientSecret != "secret-paper" {
		t.Errorf("resolved = %+v, want toss-paper/paper/secret-paper", r)
	}
}

func TestSaveCurrentContextUnknown(t *testing.T) {
	path := writeTemp(t, sampleYAML)

	if err := SaveCurrentContext(path, "ghost"); err == nil {
		t.Fatal("SaveCurrentContext(ghost) = nil error, want error")
	}

	// The file must be untouched after a failed switch.
	c, err := Load(path)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if c.CurrentContext != "toss-main" {
		t.Errorf("current-context = %q, want unchanged toss-main", c.CurrentContext)
	}
}

func TestSetCurrentContext(t *testing.T) {
	c := &Config{
		Contexts: []Context{{Name: "a", Broker: "b"}},
	}
	if err := c.SetCurrentContext("a"); err != nil {
		t.Fatalf("SetCurrentContext(a): %v", err)
	}
	if c.CurrentContext != "a" {
		t.Errorf("current-context = %q, want a", c.CurrentContext)
	}

	if err := c.SetCurrentContext("missing"); err == nil {
		t.Error("SetCurrentContext(missing) = nil error, want error")
	}
	if c.CurrentContext != "a" {
		t.Errorf("current-context = %q, want unchanged a after failed set", c.CurrentContext)
	}
}
