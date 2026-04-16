package facade

import (
	"fmt"
	"time"
)

// FacadeError is the marker interface for all facade-related errors.
type FacadeError interface {
	error
	isFacadeError()
}

// UnknownPartyError indicates a party name not defined in the protocol.
type UnknownPartyError struct {
	Name string
}

func (e *UnknownPartyError) Error() string {
	return fmt.Sprintf("unknown party: %q", e.Name)
}
func (e *UnknownPartyError) isFacadeError() {}

// MissingParamsError indicates required parameters were not provided.
type MissingParamsError struct {
	Params []string
}

func (e *MissingParamsError) Error() string {
	return fmt.Sprintf("missing required params: %v", e.Params)
}
func (e *MissingParamsError) isFacadeError() {}

// SubmitHashMismatchError indicates the server returned a different hash than expected.
type SubmitHashMismatchError struct {
	Expected string
	Received string
}

func (e *SubmitHashMismatchError) Error() string {
	return fmt.Sprintf("submit hash mismatch: expected %s, got %s", e.Expected, e.Received)
}
func (e *SubmitHashMismatchError) isFacadeError() {}

// FinalizedFailedError indicates the transaction reached a terminal failure stage.
type FinalizedFailedError struct {
	Hash  string
	Stage string
}

func (e *FinalizedFailedError) Error() string {
	return fmt.Sprintf("tx %s failed with stage %s", e.Hash, e.Stage)
}
func (e *FinalizedFailedError) isFacadeError() {}

// FinalizedTimeoutError indicates the transaction was not confirmed within the polling window.
type FinalizedTimeoutError struct {
	Hash     string
	Attempts uint
	Delay    time.Duration
}

func (e *FinalizedTimeoutError) Error() string {
	return fmt.Sprintf("tx %s not confirmed after %d attempts (delay %s)", e.Hash, e.Attempts, e.Delay)
}
func (e *FinalizedTimeoutError) isFacadeError() {}
