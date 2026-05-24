// Package facade provides the high-level ergonomic API for the Tx3 SDK.
// It wraps protocol loading, TRP communication, party management, signing,
// and submission into a fluent builder chain.
package facade

import (
	"strings"

	"github.com/tx3-lang/go-sdk/sdk/core"
	"github.com/tx3-lang/go-sdk/sdk/tii"
	"github.com/tx3-lang/go-sdk/sdk/trp"
)

// Tx3Client is the main entry point for the Tx3 SDK.
//
// Holds the deconstructed protocol parts — per-transaction TIR envelopes, the
// set of declared party names, the selected profile — plus the runtime state
// (TRP client, bound parties, env overrides). Built through Tx3ClientBuilder
// (obtained via Protocol.Client() or facade.FromParts). Profile selection is
// locked in at build time: there is no profile-switching method on the built
// client.
type Tx3Client struct {
	transactions    map[string]core.TirEnvelope
	knownParties    map[string]bool
	trp             *trp.Client
	boundParties    map[string]Party
	selectedProfile *Profile
	envOverrides    core.EnvMap
}

// newClient is the internal constructor used by Tx3ClientBuilder.Build().
func newClient(
	transactions map[string]core.TirEnvelope,
	knownParties map[string]bool,
	trpClient *trp.Client,
	boundParties map[string]Party,
	selectedProfile *Profile,
	envOverrides core.EnvMap,
) *Tx3Client {
	return &Tx3Client{
		transactions:    transactions,
		knownParties:    knownParties,
		trp:             trpClient,
		boundParties:    boundParties,
		selectedProfile: selectedProfile,
		envOverrides:    envOverrides,
	}
}

// WithParty returns a new Tx3Client with the named party bound. Validated
// against the protocol's declared parties.
//
// Returns *UnknownPartyError if name is not declared by the protocol.
func (c *Tx3Client) WithParty(name string, party Party) (*Tx3Client, error) {
	lower := strings.ToLower(name)
	if !c.knownParties[lower] {
		return nil, &UnknownPartyError{Name: lower}
	}
	next := c.cloneParties()
	next[lower] = party
	return c.withBoundParties(next), nil
}

// WithPartyUnchecked returns a new Tx3Client with the named party bound,
// skipping the declared-party lookup. Intended for codegen-generated
// wrappers; hand-written code SHOULD prefer WithParty.
func (c *Tx3Client) WithPartyUnchecked(name string, party Party) *Tx3Client {
	next := c.cloneParties()
	next[strings.ToLower(name)] = party
	return c.withBoundParties(next)
}

// WithParties returns a new Tx3Client with multiple parties bound at once.
// See WithParty.
func (c *Tx3Client) WithParties(parties map[string]Party) (*Tx3Client, error) {
	next := c.cloneParties()
	for name, party := range parties {
		lower := strings.ToLower(name)
		if !c.knownParties[lower] {
			return nil, &UnknownPartyError{Name: lower}
		}
		next[lower] = party
	}
	return c.withBoundParties(next), nil
}

// Tx starts building a transaction invocation.
//
// Returns *tii.UnknownTxError if name is not a transaction declared by the
// protocol.
func (c *Tx3Client) Tx(name string) (*TxBuilder, error) {
	tirEnvelope, ok := c.transactions[name]
	if !ok {
		return nil, &tii.UnknownTxError{Name: name}
	}
	env := c.mergedEnv()
	parties := c.mergedParties()
	return newTxBuilder(c.trp, tirEnvelope).Env(env).Parties(parties), nil
}

func (c *Tx3Client) withBoundParties(next map[string]Party) *Tx3Client {
	return newClient(c.transactions, c.knownParties, c.trp, next, c.selectedProfile, c.envOverrides)
}

func (c *Tx3Client) cloneParties() map[string]Party {
	cp := make(map[string]Party, len(c.boundParties))
	for k, v := range c.boundParties {
		cp[k] = v
	}
	return cp
}

func (c *Tx3Client) mergedEnv() core.EnvMap {
	env := core.EnvMap{}
	if c.selectedProfile != nil {
		for k, v := range c.selectedProfile.Environment {
			env[k] = v
		}
	}
	for k, v := range c.envOverrides {
		env[k] = v
	}
	return env
}

func (c *Tx3Client) mergedParties() map[string]Party {
	merged := make(map[string]Party)
	if c.selectedProfile != nil {
		for name, address := range c.selectedProfile.Parties {
			merged[strings.ToLower(name)] = AddressParty(address)
		}
	}
	for name, party := range c.boundParties {
		merged[name] = party
	}
	return merged
}
