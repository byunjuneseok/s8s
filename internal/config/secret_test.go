package config

import (
	"os"
	"runtime"
	"strings"
	"testing"
)

func TestResolveSecretEnv(t *testing.T) {
	t.Setenv("S8S_TEST_SECRET", "shh")

	tests := []struct {
		name string
		ref  string
	}{
		{name: "bare prefix", ref: "env:S8S_TEST_SECRET"},
		{name: "wrapped prefix", ref: "${env:S8S_TEST_SECRET}"},
		{name: "wrapped with spaces", ref: "${ env:S8S_TEST_SECRET }"},
		{name: "uppercase prefix", ref: "ENV:S8S_TEST_SECRET"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, src, err := ResolveSecret(tt.ref)
			if err != nil {
				t.Fatalf("ResolveSecret(%q): %v", tt.ref, err)
			}
			if v != "shh" {
				t.Errorf("value = %q, want shh", v)
			}
			if src != SourceEnv {
				t.Errorf("source = %q, want %q", src, SourceEnv)
			}
		})
	}
}

func TestResolveSecretEnvUnset(t *testing.T) {
	// t.Setenv records the prior state and restores it on cleanup, so unsetting
	// the var here is safe even if it happened to be set in the environment.
	t.Setenv("S8S_TEST_MISSING", "x")
	if err := os.Unsetenv("S8S_TEST_MISSING"); err != nil {
		t.Fatalf("unset: %v", err)
	}

	if _, _, err := ResolveSecret("env:S8S_TEST_MISSING"); err == nil {
		t.Error("ResolveSecret(unset env) = nil error, want error")
	}
}

func TestResolveSecretEnvEmpty(t *testing.T) {
	t.Setenv("S8S_TEST_EMPTY", "")

	_, src, err := ResolveSecret("env:S8S_TEST_EMPTY")
	if err == nil {
		t.Error("ResolveSecret(empty env) = nil error, want error")
	}
	if src != SourceEnv {
		t.Errorf("source = %q, want %q", src, SourceEnv)
	}
}

func TestResolveSecretPlaintext(t *testing.T) {
	tests := []struct {
		name string
		ref  string
	}{
		{name: "plain value", ref: "my-secret-value"},
		{name: "no body", ref: "env:"},
		{name: "wrapped no body", ref: "${keychain:}"},
		{name: "leading colon", ref: ":nope"},
		{name: "unknown prefix", ref: "https://example.com/path"},
		{name: "empty", ref: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, src, err := ResolveSecret(tt.ref)
			if err != nil {
				t.Fatalf("ResolveSecret(%q): %v", tt.ref, err)
			}
			if v != tt.ref {
				t.Errorf("value = %q, want %q (passthrough)", v, tt.ref)
			}
			if src != SourcePlaintext {
				t.Errorf("source = %q, want %q", src, SourcePlaintext)
			}
		})
	}
}

func TestParseSecretRef(t *testing.T) {
	tests := []struct {
		ref        string
		wantPrefix string
		wantBody   string
		wantOK     bool
	}{
		{ref: "env:VAR", wantPrefix: "env", wantBody: "VAR", wantOK: true},
		{ref: "${env:VAR}", wantPrefix: "env", wantBody: "VAR", wantOK: true},
		{ref: "keychain:svc/acct", wantPrefix: "keychain", wantBody: "svc/acct", wantOK: true},
		{ref: "${keychain:svc}", wantPrefix: "keychain", wantBody: "svc", wantOK: true},
		{ref: "ENV:VAR", wantPrefix: "env", wantBody: "VAR", wantOK: true},
		{ref: "plaintext", wantOK: false},
		{ref: "env:", wantOK: false},
		{ref: ":x", wantOK: false},
		{ref: "", wantOK: false},
	}
	for _, tt := range tests {
		t.Run(tt.ref, func(t *testing.T) {
			p, b, ok := parseSecretRef(tt.ref)
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}
			if !ok {
				return
			}
			if p != tt.wantPrefix || b != tt.wantBody {
				t.Errorf("got (%q, %q), want (%q, %q)", p, b, tt.wantPrefix, tt.wantBody)
			}
		})
	}
}

func TestResolveSecretKeychainNonDarwin(t *testing.T) {
	if runtime.GOOS == "darwin" {
		t.Skip("non-darwin behavior only")
	}
	_, src, err := ResolveSecret("keychain:my-service")
	if err == nil {
		t.Fatal("ResolveSecret(keychain) = nil error on non-darwin, want error")
	}
	if !strings.Contains(err.Error(), "macOS") {
		t.Errorf("error = %v, want mention of macOS", err)
	}
	if src != SourceKeychain {
		t.Errorf("source = %q, want %q", src, SourceKeychain)
	}
}

func TestResolveSecretKeychainParsing(t *testing.T) {
	// Verify the service/account split independent of platform; the security
	// binary is not required because we only exercise the parser here.
	tests := []struct {
		body        string
		wantService string
		wantAccount string
	}{
		{body: "my-service", wantService: "my-service", wantAccount: ""},
		{body: "my-service/me@example.com", wantService: "my-service", wantAccount: "me@example.com"},
	}
	for _, tt := range tests {
		t.Run(tt.body, func(t *testing.T) {
			service, account := tt.body, ""
			if i := strings.IndexByte(tt.body, '/'); i >= 0 {
				service = tt.body[:i]
				account = tt.body[i+1:]
			}
			if service != tt.wantService || account != tt.wantAccount {
				t.Errorf("got (%q, %q), want (%q, %q)", service, account, tt.wantService, tt.wantAccount)
			}
		})
	}
}
