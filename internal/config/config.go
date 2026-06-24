package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// EnvPath is the environment variable that overrides the config file location.
const EnvPath = "S8S_CONFIG"

// Config is the on-disk s8s configuration. It follows a kubeconfig-style layout
// where a context references a broker, and a broker references a credential.
type Config struct {
	CurrentContext string         `yaml:"current-context"`
	Contexts       []Context      `yaml:"contexts"`
	Brokers        []BrokerConfig `yaml:"brokers"`
	Credentials    []Credential   `yaml:"credentials"`
	// Watchlist is the set of symbols shown on the :watch screen.
	Watchlist []string `yaml:"watchlist,omitempty"`
}

// Context names a broker to use and how to treat it.
type Context struct {
	Name string `yaml:"name"`
	// Broker references a BrokerConfig by name.
	Broker string `yaml:"broker"`
	// ReadOnly, when true, blocks all order operations in this context.
	ReadOnly bool `yaml:"read-only"`
}

// BrokerConfig configures a brokerage adapter instance.
type BrokerConfig struct {
	Name string `yaml:"name"`
	// Type selects the adapter implementation (e.g. "toss").
	Type string `yaml:"type"`
	// Credentials references a Credential by name.
	Credentials string `yaml:"credentials"`
}

// Credential holds the secrets used to authenticate with a brokerage.
type Credential struct {
	Name         string `yaml:"name"`
	ClientID     string `yaml:"client-id"`
	ClientSecret string `yaml:"client-secret"`
}

// Resolved is a fully dereferenced context: the context together with the
// broker and credential it points at.
type Resolved struct {
	Context    Context
	Broker     BrokerConfig
	Credential Credential
}

// DefaultPath returns the config file path, honoring the S8S_CONFIG override
// and otherwise falling back to ~/.s8s/config.yaml.
func DefaultPath() (string, error) {
	if p := os.Getenv(EnvPath); p != "" {
		return p, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("config: locate home dir: %w", err)
	}
	return filepath.Join(home, ".s8s", "config.yaml"), nil
}

// Load reads and validates the config at path.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("config: read %s: %w", path, err)
	}
	var c Config
	if err := yaml.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("config: parse %s: %w", path, err)
	}
	if err := c.Validate(); err != nil {
		return nil, fmt.Errorf("config: %s: %w", path, err)
	}
	return &c, nil
}

// Validate checks that names are unique and that every reference resolves.
func (c *Config) Validate() error {
	credNames := make(map[string]bool, len(c.Credentials))
	for _, cr := range c.Credentials {
		if cr.Name == "" {
			return errors.New("credential with empty name")
		}
		if credNames[cr.Name] {
			return fmt.Errorf("duplicate credential %q", cr.Name)
		}
		credNames[cr.Name] = true
	}

	brokerNames := make(map[string]bool, len(c.Brokers))
	for _, b := range c.Brokers {
		if b.Name == "" {
			return errors.New("broker with empty name")
		}
		if brokerNames[b.Name] {
			return fmt.Errorf("duplicate broker %q", b.Name)
		}
		brokerNames[b.Name] = true
		if b.Credentials != "" && !credNames[b.Credentials] {
			return fmt.Errorf("broker %q references unknown credential %q", b.Name, b.Credentials)
		}
	}

	ctxNames := make(map[string]bool, len(c.Contexts))
	for _, ctx := range c.Contexts {
		if ctx.Name == "" {
			return errors.New("context with empty name")
		}
		if ctxNames[ctx.Name] {
			return fmt.Errorf("duplicate context %q", ctx.Name)
		}
		ctxNames[ctx.Name] = true
		if !brokerNames[ctx.Broker] {
			return fmt.Errorf("context %q references unknown broker %q", ctx.Name, ctx.Broker)
		}
	}

	if c.CurrentContext != "" && !ctxNames[c.CurrentContext] {
		return fmt.Errorf("current-context %q does not exist", c.CurrentContext)
	}
	return nil
}

// Resolve dereferences the context with the given name into a Resolved. An empty
// name selects current-context.
func (c *Config) Resolve(name string) (Resolved, error) {
	if name == "" {
		name = c.CurrentContext
	}
	if name == "" {
		return Resolved{}, errors.New("config: no context selected (set current-context)")
	}

	ctx, ok := findContext(c.Contexts, name)
	if !ok {
		return Resolved{}, fmt.Errorf("config: unknown context %q", name)
	}
	broker, ok := findBroker(c.Brokers, ctx.Broker)
	if !ok {
		return Resolved{}, fmt.Errorf("config: context %q references unknown broker %q", name, ctx.Broker)
	}
	cred, ok := findCredential(c.Credentials, broker.Credentials)
	if broker.Credentials != "" && !ok {
		return Resolved{}, fmt.Errorf("config: broker %q references unknown credential %q", broker.Name, broker.Credentials)
	}
	return Resolved{Context: ctx, Broker: broker, Credential: cred}, nil
}

func findContext(s []Context, name string) (Context, bool) {
	for _, v := range s {
		if v.Name == name {
			return v, true
		}
	}
	return Context{}, false
}

func findBroker(s []BrokerConfig, name string) (BrokerConfig, bool) {
	for _, v := range s {
		if v.Name == name {
			return v, true
		}
	}
	return BrokerConfig{}, false
}

func findCredential(s []Credential, name string) (Credential, bool) {
	for _, v := range s {
		if v.Name == name {
			return v, true
		}
	}
	return Credential{}, false
}
