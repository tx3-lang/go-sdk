package tii

import "fmt"

// TiiError is the marker interface for all TII-related errors.
// Use errors.As() to discriminate concrete error types.
type TiiError interface {
	error
	isTiiError()
}

// InvalidJSONError indicates malformed TII JSON.
type InvalidJSONError struct {
	Cause error
}

func (e *InvalidJSONError) Error() string {
	return fmt.Sprintf("invalid TII JSON: %v", e.Cause)
}
func (e *InvalidJSONError) Unwrap() error { return e.Cause }
func (e *InvalidJSONError) isTiiError()   {}

// IOError indicates a file read failure during protocol loading.
type IOError struct {
	Path  string
	Cause error
}

func (e *IOError) Error() string {
	return fmt.Sprintf("failed to read TII file %q: %v", e.Path, e.Cause)
}
func (e *IOError) Unwrap() error { return e.Cause }
func (e *IOError) isTiiError()   {}

// UnknownTxError indicates a reference to a transaction not defined in the protocol.
type UnknownTxError struct {
	Name string
}

func (e *UnknownTxError) Error() string {
	return fmt.Sprintf("unknown transaction: %q", e.Name)
}
func (e *UnknownTxError) isTiiError() {}

// UnknownProfileError indicates a reference to a profile not defined in the protocol.
type UnknownProfileError struct {
	Name string
}

func (e *UnknownProfileError) Error() string {
	return fmt.Sprintf("unknown profile: %q", e.Name)
}
func (e *UnknownProfileError) isTiiError() {}
