package config

import (
	"os"
	"path/filepath"
	"testing"
)

const sampleYAML = `current-context: toss-main
contexts:
  - name: toss-main
    broker: real-account
    read-only: false
  - name: toss-paper
    broker: paper
    read-only: true
brokers:
  - name: real-account
    type: toss
    credentials: toss-prod
  - name: paper
    type: toss
    credentials: toss-paper
credentials:
  - name: toss-prod
    client-id: id-prod
    client-secret: secret-prod
  - name: toss-paper
    client-id: id-paper
    client-secret: secret-paper
`

func writeTemp(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write temp config: %v", err)
	}
	return path
}

func TestLoadAndResolve(t *testing.T) {
	path := writeTemp(t, sampleYAML)

	c, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if c.CurrentContext != "toss-main" {
		t.Errorf("current-context = %q, want toss-main", c.CurrentContext)
	}
	if len(c.Contexts) != 2 || len(c.Brokers) != 2 || len(c.Credentials) != 2 {
		t.Fatalf("unexpected counts: %d ctx, %d brokers, %d creds", len(c.Contexts), len(c.Brokers), len(c.Credentials))
	}

	// Empty name resolves current-context.
	r, err := c.Resolve("")
	if err != nil {
		t.Fatalf("Resolve(current): %v", err)
	}
	if r.Context.Name != "toss-main" || r.Broker.Type != "toss" || r.Credential.ClientID != "id-prod" {
		t.Errorf("resolved = %+v, want toss-main/toss/id-prod", r)
	}

	// Named resolve picks the right credential through the broker.
	r2, err := c.Resolve("toss-paper")
	if err != nil {
		t.Fatalf("Resolve(toss-paper): %v", err)
	}
	if r2.Credential.ClientSecret != "secret-paper" || !r2.Context.ReadOnly {
		t.Errorf("resolved = %+v, want secret-paper and read-only", r2)
	}

	if _, err := c.Resolve("nope"); err == nil {
		t.Error("Resolve(unknown) = nil error, want error")
	}
}

func TestValidateReferentialIntegrity(t *testing.T) {
	tests := []struct {
		name string
		yaml string
	}{
		{
			name: "context references unknown broker",
			yaml: "contexts:\n  - name: c\n    broker: ghost\n",
		},
		{
			name: "broker references unknown credential",
			yaml: "brokers:\n  - name: b\n    type: toss\n    credentials: ghost\n",
		},
		{
			name: "current-context does not exist",
			yaml: "current-context: ghost\n",
		},
		{
			name: "duplicate context",
			yaml: "contexts:\n  - name: c\n    broker: b\n  - name: c\n    broker: b\nbrokers:\n  - name: b\n    type: toss\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := writeTemp(t, tt.yaml)
			if _, err := Load(path); err == nil {
				t.Error("Load = nil error, want validation error")
			}
		})
	}
}

func TestEnsureDefault(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "config.yaml")

	created, err := EnsureDefault(path)
	if err != nil {
		t.Fatalf("EnsureDefault: %v", err)
	}
	if !created {
		t.Error("created = false, want true on first call")
	}

	// The written template must be valid and loadable.
	if _, err := Load(path); err != nil {
		t.Fatalf("Load(template): %v", err)
	}

	// Second call is a no-op.
	created2, err := EnsureDefault(path)
	if err != nil {
		t.Fatalf("EnsureDefault (2nd): %v", err)
	}
	if created2 {
		t.Error("created = true on second call, want false")
	}
}
