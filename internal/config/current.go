package config

import "fmt"

// SetCurrentContext sets CurrentContext to name after verifying a context with
// that name exists. It returns an error and leaves CurrentContext unchanged if
// no such context is defined.
func (c *Config) SetCurrentContext(name string) error {
	if _, ok := findContext(c.Contexts, name); !ok {
		return fmt.Errorf("config: unknown context %q", name)
	}
	c.CurrentContext = name
	return nil
}

// SaveCurrentContext loads the config at path, switches its current-context to
// name (which must be an existing context), and writes it back atomically via
// Save. It returns an error if the config cannot be loaded, name does not name a
// known context, or the write fails.
//
// Because the config is round-tripped through the full Config struct, all data —
// contexts, brokers, and credentials (including any referenced or inline
// secrets) — is preserved. Exact byte-level formatting and comments in the
// original file may not be preserved, but no configuration data is lost.
func SaveCurrentContext(path string, name string) error {
	c, err := Load(path)
	if err != nil {
		return err
	}
	if err := c.SetCurrentContext(name); err != nil {
		return err
	}
	return Save(path, c)
}
