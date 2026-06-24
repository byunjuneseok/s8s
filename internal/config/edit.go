package config

// SetCredential inserts cr, or replaces the existing credential with the same
// name. Lookup and replacement are by Name.
func (c *Config) SetCredential(cr Credential) {
	for i := range c.Credentials {
		if c.Credentials[i].Name == cr.Name {
			c.Credentials[i] = cr
			return
		}
	}
	c.Credentials = append(c.Credentials, cr)
}

// SetBroker inserts b, or replaces the existing broker with the same name.
func (c *Config) SetBroker(b BrokerConfig) {
	for i := range c.Brokers {
		if c.Brokers[i].Name == b.Name {
			c.Brokers[i] = b
			return
		}
	}
	c.Brokers = append(c.Brokers, b)
}

// SetContext inserts ctx, or replaces the existing context with the same name.
func (c *Config) SetContext(ctx Context) {
	for i := range c.Contexts {
		if c.Contexts[i].Name == ctx.Name {
			c.Contexts[i] = ctx
			return
		}
	}
	c.Contexts = append(c.Contexts, ctx)
}
