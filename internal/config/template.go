package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// templateBody is the commented starter config written for a fresh install.
const templateBody = `# s8s configuration
# Edit by hand or run: s8s configure
#
# A context references a broker; a broker references a credential.

current-context: ""

contexts: []
  # - name: toss-main
  #   broker: real-account
  #   read-only: false

brokers: []
  # - name: real-account
  #   type: toss
  #   credentials: toss-prod

credentials: []
  # - name: toss-prod
  #   client-id: "..."
  #   client-secret: "..."
`

// EnsureDefault writes the starter template at path if no file exists there,
// creating parent directories as needed. It reports whether a file was created.
// An existing file is left untouched.
func EnsureDefault(path string) (created bool, err error) {
	if _, err := os.Stat(path); err == nil {
		return false, nil
	} else if !errors.Is(err, fs.ErrNotExist) {
		return false, fmt.Errorf("config: stat %s: %w", path, err)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return false, fmt.Errorf("config: create dir for %s: %w", path, err)
	}
	if err := os.WriteFile(path, []byte(templateBody), 0o600); err != nil {
		return false, fmt.Errorf("config: write template %s: %w", path, err)
	}
	return true, nil
}
