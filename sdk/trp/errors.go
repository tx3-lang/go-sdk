package trp

import (
	"encoding/json"
	"fmt"
)

// TrpError is the marker interface for all TRP-related errors.
type TrpError interface {
	error
	isTrpError()
}

// NetworkError indicates a connection or transport failure.
type NetworkError struct {
	Cause error
}

func (e *NetworkError) Error() string  { return fmt.Sprintf("TRP network error: %v", e.Cause) }
func (e *NetworkError) Unwrap() error  { return e.Cause }
func (e *NetworkError) isTrpError()    {}

// HttpError indicates a non-200 HTTP response from the TRP server.
type HttpError struct {
	Status     int
	StatusText string
	Body       string
}

func (e *HttpError) Error() string {
	return fmt.Sprintf("TRP HTTP error %d %s: %s", e.Status, e.StatusText, e.Body)
}
func (e *HttpError) isTrpError() {}

// DeserializationError indicates failure to parse a TRP response.
type DeserializationError struct {
	Cause error
	Raw   string
}

func (e *DeserializationError) Error() string {
	return fmt.Sprintf("TRP deserialization error: %v", e.Cause)
}
func (e *DeserializationError) Unwrap() error { return e.Cause }
func (e *DeserializationError) isTrpError()   {}

// MalformedResponseError indicates a JSON-RPC response that lacks expected fields.
type MalformedResponseError struct {
	Detail string
}

func (e *MalformedResponseError) Error() string {
	return fmt.Sprintf("TRP malformed response: %s", e.Detail)
}
func (e *MalformedResponseError) isTrpError() {}

// GenericRpcError represents a JSON-RPC error object returned by the server.
type GenericRpcError struct {
	Code    int
	Message string
	Data    json.RawMessage
}

func (e *GenericRpcError) Error() string {
	return fmt.Sprintf("TRP RPC error %d: %s", e.Code, e.Message)
}
func (e *GenericRpcError) isTrpError() {}

// UnsupportedTirError indicates TIR version mismatch.
type UnsupportedTirError struct {
	Expected string
	Provided string
}

func (e *UnsupportedTirError) Error() string {
	return fmt.Sprintf("unsupported TIR: expected %s, got %s", e.Expected, e.Provided)
}
func (e *UnsupportedTirError) isTrpError() {}

// MissingTxArgError indicates a required transaction argument was not provided.
type MissingTxArgError struct {
	Key     string
	ArgType string
}

func (e *MissingTxArgError) Error() string {
	return fmt.Sprintf("missing tx arg %q (type %s)", e.Key, e.ArgType)
}
func (e *MissingTxArgError) isTrpError() {}

// InputNotResolvedError indicates a transaction input could not be resolved.
type InputNotResolvedError struct {
	Name string
}

func (e *InputNotResolvedError) Error() string {
	return fmt.Sprintf("input not resolved: %q", e.Name)
}
func (e *InputNotResolvedError) isTrpError() {}

// TxScriptFailureError indicates a transaction script evaluation failure.
type TxScriptFailureError struct {
	Logs []string
}

func (e *TxScriptFailureError) Error() string {
	return fmt.Sprintf("tx script failure: %v", e.Logs)
}
func (e *TxScriptFailureError) isTrpError() {}

// InvalidTirEnvelopeError indicates a malformed TIR envelope.
type InvalidTirEnvelopeError struct{}

func (e *InvalidTirEnvelopeError) Error() string { return "invalid TIR envelope" }
func (e *InvalidTirEnvelopeError) isTrpError()   {}

// InvalidTirBytesError indicates TIR content could not be decoded.
type InvalidTirBytesError struct{}

func (e *InvalidTirBytesError) Error() string { return "invalid TIR bytes" }
func (e *InvalidTirBytesError) isTrpError()   {}

// UnsupportedEraError indicates the transaction targets an unsupported ledger era.
type UnsupportedEraError struct {
	Era string
}

func (e *UnsupportedEraError) Error() string {
	return fmt.Sprintf("unsupported era: %s", e.Era)
}
func (e *UnsupportedEraError) isTrpError() {}
