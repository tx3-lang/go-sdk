package facade

import (
	"strings"

	"github.com/tx3-lang/go-sdk/sdk/core"
	"github.com/tx3-lang/go-sdk/sdk/tii"
	"github.com/tx3-lang/go-sdk/sdk/trp"
)

// Tx3ClientBuilder builds a Tx3Client.
//
// Obtained via Protocol.Client() for the dynamic flow or FromParts for the
// codegen flow. All fallible validation — TRP endpoint present, selected
// profile declared, every bound party declared — happens in Build(). Optional
// setters never fail, so chains stay fluent.
//
// Example:
//
//	client, err := protocol.Client().
//	    TRPEndpoint("https://trp.example").
//	    WithProfile("preprod").
//	    WithParty("sender", facade.SignerParty(signer)).
//	    Build()
type Tx3ClientBuilder struct {
	transactions      map[string]core.TirEnvelope
	profiles          map[string]Profile
	knownParties      map[string]bool
	trpOptions        *trp.ClientOptions
	trpClientOverride *trp.Client
	profile           *string
	parties           map[string]Party
	uncheckedParties  map[string]Party
	envOverrides      core.EnvMap
}

// NewBuilder creates an empty Tx3ClientBuilder seeded with the supplied parts.
// Prefer Protocol.Client() or FromParts.
func newBuilder(
	transactions map[string]core.TirEnvelope,
	profiles map[string]Profile,
	knownParties map[string]bool,
) *Tx3ClientBuilder {
	return &Tx3ClientBuilder{
		transactions:     transactions,
		profiles:         profiles,
		knownParties:     knownParties,
		parties:          make(map[string]Party),
		uncheckedParties: make(map[string]Party),
		envOverrides:     core.EnvMap{},
	}
}

// FromParts seeds a builder with already-deconstructed protocol fragments.
//
// Codegen-generated bindings call this with embedded per-transaction TIR
// envelopes, per-profile environment + party-address maps, and (typically)
// an empty known-parties set — the typed With<Party> wrapper methods bake
// party names in at codegen time so runtime name validation is unnecessary.
func FromParts(
	transactions map[string]core.TirEnvelope,
	profiles map[string]Profile,
	knownParties []string,
) *Tx3ClientBuilder {
	knownSet := make(map[string]bool, len(knownParties))
	for _, name := range knownParties {
		knownSet[strings.ToLower(name)] = true
	}
	return newBuilder(transactions, profiles, knownSet)
}

// FromProtocol seeds a builder from a runtime-loaded protocol. Entry point for
// Protocol.Client().
func FromProtocol(p *tii.Protocol) *Tx3ClientBuilder {
	transactions := make(map[string]core.TirEnvelope)
	for name, tx := range p.Transactions() {
		transactions[name] = tx.Tir
	}

	profiles := make(map[string]Profile)
	for name, spec := range p.Profiles() {
		parties := make(map[string]string, len(spec.Parties))
		for k, v := range spec.Parties {
			parties[k] = v
		}
		env := core.EnvMap{}
		for k, v := range spec.Environment {
			env[k] = v
		}
		profiles[name] = Profile{Environment: env, Parties: parties}
	}

	known := make(map[string]bool)
	for name := range p.Parties() {
		known[strings.ToLower(name)] = true
	}

	return newBuilder(transactions, profiles, known)
}

// TRP sets the full TRP client options.
func (b *Tx3ClientBuilder) TRP(options trp.ClientOptions) *Tx3ClientBuilder {
	b.trpOptions = &options
	return b
}

// TRPEndpoint is a shorthand for TRP(trp.ClientOptions{Endpoint: url}).
func (b *Tx3ClientBuilder) TRPEndpoint(url string) *Tx3ClientBuilder {
	b.trpOptions = &trp.ClientOptions{Endpoint: url}
	return b
}

// WithHeader adds a single TRP request header. Initializes options to an empty
// endpoint if not set — callers must still supply the endpoint via TRP() or
// TRPEndpoint().
func (b *Tx3ClientBuilder) WithHeader(key, value string) *Tx3ClientBuilder {
	if b.trpOptions == nil {
		b.trpOptions = &trp.ClientOptions{}
	}
	if b.trpOptions.Headers == nil {
		b.trpOptions.Headers = map[string]string{}
	}
	b.trpOptions.Headers[key] = value
	return b
}

// WithProfile selects a profile by name. Validated in Build().
func (b *Tx3ClientBuilder) WithProfile(name string) *Tx3ClientBuilder {
	b.profile = &name
	return b
}

// WithParty binds a party by name. The name is validated against the
// protocol's declared parties in Build().
func (b *Tx3ClientBuilder) WithParty(name string, party Party) *Tx3ClientBuilder {
	b.parties[strings.ToLower(name)] = party
	return b
}

// WithPartyUnchecked binds a party without validating the name against the
// protocol's declared parties. Intended for codegen-generated wrappers; hand-
// written code SHOULD prefer WithParty.
func (b *Tx3ClientBuilder) WithPartyUnchecked(name string, party Party) *Tx3ClientBuilder {
	b.uncheckedParties[strings.ToLower(name)] = party
	return b
}

// WithParties binds multiple parties at once. See WithParty.
func (b *Tx3ClientBuilder) WithParties(parties map[string]Party) *Tx3ClientBuilder {
	for name, party := range parties {
		b.WithParty(name, party)
	}
	return b
}

// WithEnvValue sets a single environment value, merged on top of the selected
// profile's environment at resolve time (override wins).
func (b *Tx3ClientBuilder) WithEnvValue(key string, value interface{}) *Tx3ClientBuilder {
	b.envOverrides[key] = value
	return b
}

// trpClient is an internal escape hatch that lets tests inject a pre-built /
// fake TRP client without going through the ClientOptions construction path.
// Not part of the public API.
func (b *Tx3ClientBuilder) trpClient(client *trp.Client) *Tx3ClientBuilder {
	b.trpClientOverride = client
	return b
}

// Build validates the builder state and materializes the Tx3Client.
//
// Returns *MissingTrpEndpointError if no TRP endpoint was supplied,
// *tii.UnknownProfileError if the selected profile is not declared, and
// *UnknownPartyError if any bound party is not declared.
func (b *Tx3ClientBuilder) Build() (*Tx3Client, error) {
	var trpClient *trp.Client
	if b.trpClientOverride != nil {
		trpClient = b.trpClientOverride
	} else {
		if b.trpOptions == nil || b.trpOptions.Endpoint == "" {
			return nil, &MissingTrpEndpointError{}
		}
		trpClient = trp.NewClient(*b.trpOptions)
	}

	var selectedProfile *Profile
	if b.profile != nil {
		profile, ok := b.profiles[*b.profile]
		if !ok {
			return nil, &tii.UnknownProfileError{Name: *b.profile}
		}
		selectedProfile = &profile
	}

	for name := range b.parties {
		if !b.knownParties[name] {
			return nil, &UnknownPartyError{Name: name}
		}
	}

	bound := make(map[string]Party, len(b.parties)+len(b.uncheckedParties))
	for name, party := range b.parties {
		bound[name] = party
	}
	for name, party := range b.uncheckedParties {
		bound[name] = party
	}

	return newClient(b.transactions, b.knownParties, trpClient, bound, selectedProfile, b.envOverrides), nil
}
