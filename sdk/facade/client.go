// Package facade provides the high-level ergonomic API for the Tx3 SDK.
// It wraps protocol loading, TRP communication, party management, signing,
// and submission into a fluent builder chain.
package facade

import (
	"github.com/tx3-lang/go-sdk/sdk/tii"
	"github.com/tx3-lang/go-sdk/sdk/trp"
)

// Tx3Client is the main entry point for the Tx3 SDK.
// It holds a loaded protocol, a TRP client, and party bindings,
// and creates TxBuilder instances for building transactions.
type Tx3Client struct {
	protocol *tii.Protocol
	trp      *trp.Client
	parties  map[string]Party
	profile  *string
}

// NewClient creates a new Tx3Client with the given protocol and TRP client.
func NewClient(protocol *tii.Protocol, trpClient *trp.Client) *Tx3Client {
	return &Tx3Client{
		protocol: protocol,
		trp:      trpClient,
		parties:  make(map[string]Party),
	}
}

// WithProfile returns a new Tx3Client with the given profile set.
// The profile is applied to every invocation created by this client.
func (c *Tx3Client) WithProfile(name string) *Tx3Client {
	cp := c.clone()
	cp.profile = &name
	return cp
}

// WithParty returns a new Tx3Client with the named party attached.
func (c *Tx3Client) WithParty(name string, party Party) *Tx3Client {
	cp := c.clone()
	cp.parties[name] = party
	return cp
}

// WithParties returns a new Tx3Client with multiple parties attached.
func (c *Tx3Client) WithParties(parties map[string]Party) *Tx3Client {
	cp := c.clone()
	for k, v := range parties {
		cp.parties[k] = v
	}
	return cp
}

// Tx begins building a transaction by name, returning a TxBuilder.
func (c *Tx3Client) Tx(name string) *TxBuilder {
	return &TxBuilder{
		protocol: c.protocol,
		trp:      c.trp,
		txName:   name,
		args:     make(map[string]interface{}),
		parties:  c.cloneParties(),
		profile:  c.profile,
	}
}

func (c *Tx3Client) clone() *Tx3Client {
	return &Tx3Client{
		protocol: c.protocol,
		trp:      c.trp,
		parties:  c.cloneParties(),
		profile:  c.profile,
	}
}

func (c *Tx3Client) cloneParties() map[string]Party {
	cp := make(map[string]Party, len(c.parties))
	for k, v := range c.parties {
		cp[k] = v
	}
	return cp
}
