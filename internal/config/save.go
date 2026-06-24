package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Save writes the config to path atomically: it marshals to a temp file in the
// same directory and renames it into place. Parent directories are created as
// needed and the file is written 0600 since it may contain secrets. The config
// is validated before writing.
func Save(path string, c *Config) error {
	if err := c.Validate(); err != nil {
		return fmt.Errorf("config: refusing to save invalid config: %w", err)
	}
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("config: marshal: %w", err)
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("config: create dir %s: %w", dir, err)
	}

	tmp, err := os.CreateTemp(dir, ".config-*.yaml.tmp")
	if err != nil {
		return fmt.Errorf("config: create temp file: %w", err)
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()

	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("config: chmod temp file: %w", err)
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("config: write temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("config: close temp file: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("config: replace %s: %w", path, err)
	}
	return nil
}
