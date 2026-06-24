package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"

	"github.com/byunjuneseok/s8s/internal/config"
	"golang.org/x/term"
)

type configureOptions struct {
	context        string
	brokerType     string
	clientID       string
	clientSecret   string
	readOnly       bool
	setCurrent     bool
	nonInteractive bool
}

// runConfigure implements `s8s configure`: it creates or updates a context (with
// its broker and credential) in the config file and writes it back atomically.
func runConfigure(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("configure", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var (
		ctxName  = fs.String("context", "", "context name to create or update")
		typ      = fs.String("type", "toss", "broker type")
		clientID = fs.String("client-id", "", "OAuth client id")
		secret   = fs.String("client-secret", "", "OAuth client secret")
		readOnly = fs.Bool("read-only", false, "block order operations in this context")
		setCur   = fs.Bool("set-current", false, "make this the current context")
		noInt    = fs.Bool("non-interactive", false, "do not prompt; require flags")
	)
	if err := fs.Parse(args); err != nil {
		return 2
	}

	opts := configureOptions{
		context:        strings.TrimSpace(*ctxName),
		brokerType:     strings.TrimSpace(*typ),
		clientID:       strings.TrimSpace(*clientID),
		clientSecret:   *secret,
		readOnly:       *readOnly,
		setCurrent:     *setCur,
		nonInteractive: *noInt,
	}

	reader := bufio.NewReader(stdin)
	if !opts.nonInteractive {
		if opts.context == "" {
			opts.context = promptLine(reader, stdout, "Context name", "toss-main")
		}
		if opts.brokerType == "" {
			opts.brokerType = promptLine(reader, stdout, "Broker type", "toss")
		}
		if opts.clientID == "" {
			opts.clientID = promptLine(reader, stdout, "Client ID", "")
		}
		if opts.clientSecret == "" {
			opts.clientSecret = promptSecret(reader, stdin, stdout, "Client secret")
		}
	}

	if opts.brokerType == "" {
		opts.brokerType = "toss"
	}
	if opts.context == "" || opts.clientID == "" || opts.clientSecret == "" {
		_, _ = fmt.Fprintln(stderr, "configure: --context, --client-id and --client-secret are required")
		return 2
	}

	path, err := config.DefaultPath()
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "configure: %v\n", err)
		return 1
	}
	cfg, err := loadOrEmpty(path)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "configure: %v\n", err)
		return 1
	}

	applyConfigure(cfg, opts)

	if err := config.Save(path, cfg); err != nil {
		_, _ = fmt.Fprintf(stderr, "configure: %v\n", err)
		return 1
	}

	_, _ = fmt.Fprintf(stdout, "Saved context %q to %s\n", opts.context, path)
	if cfg.CurrentContext == opts.context {
		_, _ = fmt.Fprintf(stdout, "current-context is now %q\n", opts.context)
	}
	_, _ = fmt.Fprintln(stdout, "Note: the client secret is stored in plaintext for now; secure storage comes later.")
	return 0
}

// applyConfigure upserts the context, broker, and credential implied by opts.
// The three entities share the context name, mirroring the kubeconfig style.
func applyConfigure(cfg *config.Config, o configureOptions) {
	name := o.context
	cfg.SetCredential(config.Credential{Name: name, ClientID: o.clientID, ClientSecret: o.clientSecret})
	cfg.SetBroker(config.BrokerConfig{Name: name, Type: o.brokerType, Credentials: name})
	cfg.SetContext(config.Context{Name: name, Broker: name, ReadOnly: o.readOnly})
	if o.setCurrent || cfg.CurrentContext == "" {
		cfg.CurrentContext = name
	}
}

// loadOrEmpty loads the config at path, returning an empty config if the file
// does not exist yet.
func loadOrEmpty(path string) (*config.Config, error) {
	if _, err := os.Stat(path); errors.Is(err, fs.ErrNotExist) {
		return &config.Config{}, nil
	}
	return config.Load(path)
}

func promptLine(reader *bufio.Reader, stdout io.Writer, label, def string) string {
	if def != "" {
		_, _ = fmt.Fprintf(stdout, "%s [%s]: ", label, def)
	} else {
		_, _ = fmt.Fprintf(stdout, "%s: ", label)
	}
	line, _ := reader.ReadString('\n')
	line = strings.TrimSpace(line)
	if line == "" {
		return def
	}
	return line
}

func promptSecret(reader *bufio.Reader, stdin io.Reader, stdout io.Writer, label string) string {
	_, _ = fmt.Fprintf(stdout, "%s: ", label)
	if f, ok := stdin.(*os.File); ok && term.IsTerminal(int(f.Fd())) {
		b, err := term.ReadPassword(int(f.Fd()))
		_, _ = fmt.Fprintln(stdout)
		if err == nil {
			return strings.TrimSpace(string(b))
		}
	}
	line, _ := reader.ReadString('\n')
	return strings.TrimSpace(line)
}
