package config

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// Secret reference sources returned as the second value of ResolveSecret.
const (
	// SourceEnv indicates the value came from an environment variable.
	SourceEnv = "env"
	// SourceKeychain indicates the value came from the macOS Keychain.
	SourceKeychain = "keychain"
	// SourcePlaintext indicates the value was stored inline, not as a reference.
	SourcePlaintext = "plaintext"
)

// ResolveSecret resolves a credential field that may be either an inline secret
// or a reference to an external secret store. It returns the resolved value, the
// source it was read from (one of SourceEnv, SourceKeychain, SourcePlaintext),
// and an error.
//
// Reference syntax. A reference is "PREFIX:BODY", optionally wrapped in "${...}".
// Both forms below are equivalent:
//
//	env:VAR            ${env:VAR}
//	keychain:SERVICE   ${keychain:SERVICE}
//
// Recognized prefixes, in the order they are matched:
//
//   - env:VAR — reads os.Getenv("VAR"). It is an error if VAR is unset or empty.
//     source is SourceEnv.
//   - keychain:SERVICE or keychain:SERVICE/ACCOUNT — on macOS, reads the generic
//     password for SERVICE (and optional ACCOUNT) via
//     /usr/bin/security find-generic-password. On non-macOS platforms this
//     returns an error. source is SourceKeychain.
//
// Any value without a recognized prefix is treated as a plaintext secret and
// returned unchanged with source SourcePlaintext. Callers can use the returned
// source to warn when a secret is stored inline rather than referenced.
func ResolveSecret(ref string) (value string, source string, err error) {
	prefix, body, ok := parseSecretRef(ref)
	if !ok {
		return ref, SourcePlaintext, nil
	}
	switch prefix {
	case "env":
		return resolveEnvSecret(body)
	case "keychain":
		return resolveKeychainSecret(body)
	default:
		// Unknown prefix: treat the whole reference as a plaintext value rather
		// than guessing. This keeps unrelated values (e.g. URLs with a scheme)
		// from being misinterpreted as secret references.
		return ref, SourcePlaintext, nil
	}
}

// parseSecretRef splits a reference into its prefix and body, unwrapping an
// optional ${...} wrapper. It reports ok=false when ref is not of the form
// "prefix:body" (and so should be treated as plaintext).
func parseSecretRef(ref string) (prefix, body string, ok bool) {
	s := strings.TrimSpace(ref)
	// Unwrap a single ${...} layer if present.
	if strings.HasPrefix(s, "${") && strings.HasSuffix(s, "}") {
		s = s[2 : len(s)-1]
		s = strings.TrimSpace(s)
	}
	idx := strings.IndexByte(s, ':')
	if idx <= 0 {
		return "", "", false
	}
	prefix = strings.ToLower(strings.TrimSpace(s[:idx]))
	body = strings.TrimSpace(s[idx+1:])
	if body == "" {
		return "", "", false
	}
	return prefix, body, true
}

func resolveEnvSecret(name string) (string, string, error) {
	v := os.Getenv(name)
	if v == "" {
		return "", SourceEnv, fmt.Errorf("config: secret env reference %q: environment variable %s is unset or empty", "env:"+name, name)
	}
	return v, SourceEnv, nil
}

func resolveKeychainSecret(body string) (string, string, error) {
	if runtime.GOOS != "darwin" {
		return "", SourceKeychain, fmt.Errorf("config: keychain references are only supported on macOS (current platform %s)", runtime.GOOS)
	}
	service, account := body, ""
	if i := strings.IndexByte(body, '/'); i >= 0 {
		service = strings.TrimSpace(body[:i])
		account = strings.TrimSpace(body[i+1:])
	}
	if service == "" {
		return "", SourceKeychain, fmt.Errorf("config: keychain reference %q: empty service name", "keychain:"+body)
	}

	args := []string{"find-generic-password", "-s", service}
	if account != "" {
		args = append(args, "-a", account)
	}
	args = append(args, "-w")

	out, err := exec.Command("/usr/bin/security", args...).Output()
	if err != nil {
		return "", SourceKeychain, fmt.Errorf("config: keychain reference %q: %w", "keychain:"+body, err)
	}
	return strings.TrimRight(string(out), "\n"), SourceKeychain, nil
}
