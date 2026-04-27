package facade

import (
	"encoding/hex"

	"github.com/tx3-lang/go-sdk/sdk/core"
	"github.com/tx3-lang/go-sdk/sdk/signer"
	"github.com/tx3-lang/go-sdk/sdk/trp"
)

// ResolvedTx represents a transaction that has been resolved by TRP
// and is ready for signing.
type ResolvedTx struct {
	trp              *trp.Client
	Hash             string // Transaction hash (hex)
	TxHex            string // Raw transaction bytes (hex)
	signers          []signerParty
	manualWitnesses  []trp.TxWitness
}

// SigningHash returns the hash that signers should sign.
func (r *ResolvedTx) SigningHash() string {
	return r.Hash
}

// AddWitness attaches a pre-computed witness produced outside any registered
// signer (e.g. from a wallet app, hardware device, or remote signer service).
//
// The witness is appended to the TRP SubmitParams.witnesses array after any
// witnesses produced by registered signer parties, in attach order. May be
// called any number of times. Sign succeeds with zero registered signers when
// at least one witness has been manually attached.
//
// The SDK does not verify the witness against the tx hash; that binding is
// enforced by TRP at submit time.
func (r *ResolvedTx) AddWitness(w trp.TxWitness) *ResolvedTx {
	r.manualWitnesses = append(r.manualWitnesses, w)
	return r
}

// Sign signs the resolved transaction with all attached signers,
// producing a SignedTx ready for submission. Manually attached witnesses
// (see AddWitness) are appended after the registered-signer witnesses, in
// attach order.
func (r *ResolvedTx) Sign() (*SignedTx, error) {
	var witnesses []trp.TxWitness
	var witnessInfos []trp.WitnessInfo

	request := signer.SignRequest{TxHashHex: r.Hash, TxCborHex: r.TxHex}

	for _, sp := range r.signers {
		w, err := sp.signer.Sign(request)
		if err != nil {
			return nil, err
		}

		witnesses = append(witnesses, trp.TxWitness{
			Key:         w.Key,
			Signature:   w.Signature,
			WitnessType: w.WitnessType,
		})

		pubKeyBytes, _ := hex.DecodeString(w.Key.Content)
		_ = pubKeyBytes
		witnessInfos = append(witnessInfos, trp.WitnessInfo{
			PublicKey: w.Key.Content,
			Address:   sp.signer.Address(),
		})
	}

	for _, w := range r.manualWitnesses {
		witnesses = append(witnesses, w)
		witnessInfos = append(witnessInfos, trp.WitnessInfo{
			PublicKey: w.Key.Content,
			Address:   "",
		})
	}

	submitParams := trp.SubmitParams{
		Tx:        core.NewHexEnvelope(r.TxHex),
		Witnesses: witnesses,
	}

	return &SignedTx{
		trp:          r.trp,
		Hash:         r.Hash,
		SubmitParams: submitParams,
		witnesses:    witnessInfos,
	}, nil
}
