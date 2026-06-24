package main

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/byunjuneseok/s8s/internal/config"
)

func TestApplyConfigure(t *testing.T) {
	cfg := &config.Config{}
	applyConfigure(cfg, configureOptions{context: "main", brokerType: "toss", clientID: "id1", clientSecret: "s1"})

	if cfg.CurrentContext != "main" {
		t.Errorf("current-context = %q, want main (first context)", cfg.CurrentContext)
	}
	if len(cfg.Contexts) != 1 || len(cfg.Brokers) != 1 || len(cfg.Credentials) != 1 {
		t.Fatalf("unexpected counts: %+v", cfg)
	}

	// Re-running with the same name updates in place (idempotent upsert).
	applyConfigure(cfg, configureOptions{context: "main", brokerType: "toss", clientID: "id2", clientSecret: "s2"})
	if len(cfg.Credentials) != 1 || cfg.Credentials[0].ClientID != "id2" {
		t.Errorf("re-configure did not update credential: %+v", cfg.Credentials)
	}

	// A second context does not steal current-context.
	applyConfigure(cfg, configureOptions{context: "paper", brokerType: "toss", clientID: "id3", clientSecret: "s3"})
	if cfg.CurrentContext != "main" {
		t.Errorf("current-context = %q, want main (unchanged)", cfg.CurrentContext)
	}
	if len(cfg.Contexts) != 2 {
		t.Errorf("want 2 contexts, got %d", len(cfg.Contexts))
	}
}

func TestRunConfigureNonInteractive(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	t.Setenv(config.EnvPath, path)

	args := []string{
		"--context", "toss-main",
		"--type", "toss",
		"--api-key", "my-id",
		"--secret-key", "my-secret",
		"--non-interactive",
	}
	var stdout, stderr bytes.Buffer
	if code := runConfigure(args, strings.NewReader(""), &stdout, &stderr); code != 0 {
		t.Fatalf("runConfigure exit = %d, stderr=%q", code, stderr.String())
	}

	c, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load written config: %v", err)
	}
	r, err := c.Resolve("")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if r.Credential.ClientID != "my-id" || r.Credential.ClientSecret != "my-secret" || r.Broker.Type != "toss" {
		t.Errorf("resolved = %+v, want my-id/my-secret/toss", r)
	}
}

func TestRunConfigureMissingRequired(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runConfigure([]string{"--non-interactive", "--context", "x"}, strings.NewReader(""), &stdout, &stderr)
	if code != 2 {
		t.Errorf("exit = %d, want 2 for missing required flags", code)
	}
}

func TestRunConfigureInteractive(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	t.Setenv(config.EnvPath, path)

	// type defaults to "toss" via the flag, so only context (blank => default),
	// client id, and secret are prompted.
	stdin := strings.NewReader("\nthe-id\nthe-secret\n")
	var stdout, stderr bytes.Buffer
	if code := runConfigure(nil, stdin, &stdout, &stderr); code != 0 {
		t.Fatalf("runConfigure exit = %d, stderr=%q", code, stderr.String())
	}

	c, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	r, err := c.Resolve("")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if r.Context.Name != "default" || r.Credential.ClientID != "the-id" || r.Credential.ClientSecret != "the-secret" {
		t.Errorf("resolved = %+v, want default/the-id/the-secret", r)
	}
}
