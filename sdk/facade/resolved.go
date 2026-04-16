package facade

import (
	"encoding/hex"

	"github.com/tx3-lang/go-sdk/sdk/core"
	"github.com/tx3-lang/go-sdk/sdk/trp"
)

// ResolvedTx represents a transaction that has been resolved by TRP
// and is ready for signing.
type ResolvedTx struct {
	trp     *trp.Client
	Hash    string // Transaction hash (hex)
	TxHex   string // Raw transaction bytes (hex)
	signers []signerParty
}

// SigningHash returns the hash that signers should sign.
func (r *ResolvedTx) SigningHash() string {
	return r.Hash
}

// Sign signs the resolved transaction with all attached signers,
// producing a SignedTx ready for submission.
func (r *ResolvedTx) Sign() (*SignedTx, error) {
	var witnesses []trp.TxWitness
	var witnessInfos []trp.WitnessInfo

	for _, sp := range r.signers {
		w, err := sp.signer.Sign(r.Hash)
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
