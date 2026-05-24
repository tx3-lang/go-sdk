package facade

import (
	"context"
	"strings"

	"github.com/tx3-lang/go-sdk/sdk/core"
	"github.com/tx3-lang/go-sdk/sdk/signer"
	"github.com/tx3-lang/go-sdk/sdk/trp"
)

// TxBuilder collects transaction arguments and resolves the transaction via TRP.
//
// Source-agnostic: holds the TIR envelope directly, plus the environment
// values from the selected profile (with builder-supplied overrides already
// folded in), the bound parties, and the typed args. Drives a single Resolve()
// path regardless of whether the upstream was a runtime-loaded *tii.Protocol
// or codegen-embedded fragments.
type TxBuilder struct {
	tir     core.TirEnvelope
	trp     *trp.Client
	env     core.EnvMap
	parties map[string]Party
	args    map[string]interface{}
}

// newTxBuilder is the internal constructor used by Tx3Client.Tx().
func newTxBuilder(trpClient *trp.Client, tir core.TirEnvelope) *TxBuilder {
	return &TxBuilder{
		tir:     tir,
		trp:     trpClient,
		env:     core.EnvMap{},
		parties: make(map[string]Party),
		args:    make(map[string]interface{}),
	}
}

// Env sets the environment values applied to this transaction.
func (b *TxBuilder) Env(env core.EnvMap) *TxBuilder {
	b.env = core.EnvMap{}
	for k, v := range env {
		b.env[k] = v
	}
	return b
}

// Parties attaches party definitions (case-insensitive names).
func (b *TxBuilder) Parties(parties map[string]Party) *TxBuilder {
	for name, party := range parties {
		b.parties[strings.ToLower(name)] = party
	}
	return b
}

// Arg sets a single transaction argument. The key is matched case-
// insensitively against protocol-declared parameter names.
func (b *TxBuilder) Arg(name string, value interface{}) *TxBuilder {
	coerced, err := core.CoerceArg(value)
	if err != nil {
		// Store raw value; validation happens server-side.
		b.args[core.NormalizeArgKey(name)] = value
	} else {
		b.args[core.NormalizeArgKey(name)] = coerced
	}
	return b
}

// Args sets multiple transaction arguments at once.
func (b *TxBuilder) Args(args map[string]interface{}) *TxBuilder {
	for k, v := range args {
		b.Arg(k, v)
	}
	return b
}

// signerParty pairs a party name with its signer for the signing step.
type signerParty struct {
	name   string
	signer signer.Signer
}

// Resolve invokes the TRP server to resolve this transaction.
// Returns a ResolvedTx ready for signing.
func (b *TxBuilder) Resolve(ctx context.Context) (*ResolvedTx, error) {
	merged := map[string]interface{}{}
	for k, v := range b.env {
		merged[k] = v
	}
	for name, party := range b.parties {
		merged[name] = party.partyAddress()
	}
	for k, v := range b.args {
		merged[k] = v
	}

	resolveParams := trp.ResolveParams{
		Tir:  b.tir,
		Args: merged,
	}

	envelope, err := b.trp.Resolve(ctx, resolveParams)
	if err != nil {
		return nil, err
	}

	var signers []signerParty
	for name, party := range b.parties {
		if party.isSigner {
			signers = append(signers, signerParty{name: name, signer: party.signer})
		}
	}

	return &ResolvedTx{
		trp:     b.trp,
		Hash:    envelope.Hash,
		TxHex:   envelope.Tx,
		signers: signers,
	}, nil
}
