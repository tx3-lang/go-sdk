package facade

import (
	"context"
	"strings"

	"github.com/tx3-lang/go-sdk/sdk/core"
	"github.com/tx3-lang/go-sdk/sdk/signer"
	"github.com/tx3-lang/go-sdk/sdk/tii"
	"github.com/tx3-lang/go-sdk/sdk/trp"
)

// TxBuilder collects transaction arguments and resolves the transaction via TRP.
type TxBuilder struct {
	protocol *tii.Protocol
	trp      *trp.Client
	txName   string
	args     map[string]interface{}
	parties  map[string]Party
	profile  *string
}

// Arg sets a single transaction argument. The key is matched case-insensitively
// against protocol-declared parameter names.
func (b *TxBuilder) Arg(name string, value interface{}) *TxBuilder {
	coerced, err := core.CoerceArg(value)
	if err != nil {
		// Store raw value; validation happens at resolve time
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
	// Create invocation from protocol
	inv, err := b.protocol.Invoke(b.txName, b.profile)
	if err != nil {
		return nil, err
	}

	// Validate parties and inject their addresses
	protocolParties := b.protocol.Parties()
	var signers []signerParty
	for name, party := range b.parties {
		// Check that the party is known to the protocol
		found := false
		for pName := range protocolParties {
			if strings.EqualFold(pName, name) {
				found = true
				break
			}
		}
		if !found {
			return nil, &UnknownPartyError{Name: name}
		}

		// Inject party address as an argument
		inv.SetArg(name, party.partyAddress())

		// Collect signers
		if party.isSigner {
			signers = append(signers, signerParty{name: name, signer: party.signer})
		}
	}

	// Set user-provided args (these override party-injected ones)
	for k, v := range b.args {
		inv.SetArg(k, v)
	}

	// Convert to resolve request
	tirEnvelope, args, err := inv.IntoResolveRequest()
	if err != nil {
		return nil, err
	}

	// Call TRP resolve
	resolveParams := trp.ResolveParams{
		Tir:  tirEnvelope,
		Args: args,
	}

	envelope, err := b.trp.Resolve(ctx, resolveParams)
	if err != nil {
		return nil, err
	}

	return &ResolvedTx{
		trp:     b.trp,
		Hash:    envelope.Hash,
		TxHex:   envelope.Tx,
		signers: signers,
	}, nil
}
