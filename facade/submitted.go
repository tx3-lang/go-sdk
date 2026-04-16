package facade

import (
	"context"
	"time"

	"github.com/tx3-lang/go-sdk/trp"
)

// SubmittedTx represents a transaction that has been submitted to the chain.
// It provides polling methods to wait for confirmation or finalization.
type SubmittedTx struct {
	trp  *trp.Client
	Hash string
}

// WaitForConfirmed polls the TRP server until the transaction reaches the
// Confirmed or Finalized stage. Fails fast on Dropped or RolledBack.
func (s *SubmittedTx) WaitForConfirmed(ctx context.Context, config PollConfig) (*trp.TxStatus, error) {
	return s.waitForStage(ctx, config, trp.StageConfirmed)
}

// WaitForFinalized polls the TRP server until the transaction reaches the
// Finalized stage. Fails fast on Dropped or RolledBack.
func (s *SubmittedTx) WaitForFinalized(ctx context.Context, config PollConfig) (*trp.TxStatus, error) {
	return s.waitForStage(ctx, config, trp.StageFinalized)
}

func (s *SubmittedTx) waitForStage(ctx context.Context, config PollConfig, target trp.TxStage) (*trp.TxStatus, error) {
	for attempt := uint(0); attempt < config.Attempts; attempt++ {
		// Check status
		resp, err := s.trp.CheckStatus(ctx, []string{s.Hash})
		if err != nil {
			return nil, err
		}

		if status, ok := resp.Statuses[s.Hash]; ok {
			// Fail fast on terminal failures
			if status.Stage.IsTerminalFailure() {
				return nil, &FinalizedFailedError{
					Hash:  s.Hash,
					Stage: string(status.Stage),
				}
			}

			// Check if we've reached the target stage
			if stageReached(status.Stage, target) {
				return &status, nil
			}
		}

		// Wait before next poll (unless this is the last attempt)
		if attempt < config.Attempts-1 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(config.Delay):
			}
		}
	}

	return nil, &FinalizedTimeoutError{
		Hash:     s.Hash,
		Attempts: config.Attempts,
		Delay:    config.Delay,
	}
}

// stageReached returns true if the current stage has reached the target.
// Confirmed is reached by Confirmed or Finalized.
// Finalized is reached only by Finalized.
func stageReached(current, target trp.TxStage) bool {
	if target == trp.StageConfirmed {
		return current == trp.StageConfirmed || current == trp.StageFinalized
	}
	return current == target
}
