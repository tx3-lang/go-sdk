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

// InvalidParamsSchemaError indicates the transaction's params schema is malformed.
type InvalidParamsSchemaError struct {
	Detail string
}

func (e *InvalidParamsSchemaError) Error() string {
	return fmt.Sprintf("invalid params schema: %s", e.Detail)
}
func (e *InvalidParamsSchemaError) isTiiError() {}

// InvalidParamTypeError indicates an unrecognized parameter type in the schema.
type InvalidParamTypeError struct {
	Detail string
}

func (e *InvalidParamTypeError) Error() string {
	return fmt.Sprintf("invalid param type: %s", e.Detail)
}
func (e *InvalidParamTypeError) isTiiError() {}
