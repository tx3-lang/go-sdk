package facade

import (
	"context"

	"github.com/tx3-lang/go-sdk/trp"
)

// SignedTx represents a signed transaction ready for submission to the chain.
type SignedTx struct {
	trp          *trp.Client
	Hash         string
	SubmitParams trp.SubmitParams
	witnesses    []trp.WitnessInfo
}

// Witnesses returns diagnostic information about the witnesses that signed this transaction.
func (s *SignedTx) Witnesses() []trp.WitnessInfo {
	return s.witnesses
}

// Submit sends the signed transaction to the TRP server for chain submission.
// Returns a SubmittedTx that can be polled for confirmation.
func (s *SignedTx) Submit(ctx context.Context) (*SubmittedTx, error) {
	resp, err := s.trp.Submit(ctx, s.SubmitParams)
	if err != nil {
		return nil, err
	}

	if resp.Hash != s.Hash {
		return nil, &SubmitHashMismatchError{
			Expected: s.Hash,
			Received: resp.Hash,
		}
	}

	return &SubmittedTx{
		trp:  s.trp,
		Hash: resp.Hash,
	}, nil
}
