package tests

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/tx3-lang/go-sdk/sdk/facade"
	"github.com/tx3-lang/go-sdk/sdk/signer"
	"github.com/tx3-lang/go-sdk/sdk/tii"
	"github.com/tx3-lang/go-sdk/sdk/trp"
)

// TestErrorDiscrimination verifies that every §3.8 error category is
// discriminable via errors.As() without string matching.
func TestErrorDiscriminationTII(t *testing.T) {
	// Wrap each error and verify errors.As can extract the concrete type
	assertDiscriminable[*tii.InvalidJSONError](t, &tii.InvalidJSONError{Cause: fmt.Errorf("bad")})
	assertDiscriminable[*tii.UnknownTxError](t, &tii.UnknownTxError{Name: "foo"})
	assertDiscriminable[*tii.UnknownProfileError](t, &tii.UnknownProfileError{Name: "bar"})
	assertDiscriminable[*tii.InvalidParamsSchemaError](t, &tii.InvalidParamsSchemaError{Detail: "bad"})
	assertDiscriminable[*tii.InvalidParamTypeError](t, &tii.InvalidParamTypeError{Detail: "bad"})
}

func TestErrorDiscriminationTRP(t *testing.T) {
	assertDiscriminable[*trp.NetworkError](t, &trp.NetworkError{})
	assertDiscriminable[*trp.HttpError](t, &trp.HttpError{Status: 500, StatusText: "Internal Server Error"})
	assertDiscriminable[*trp.DeserializationError](t, &trp.DeserializationError{})
	assertDiscriminable[*trp.MalformedResponseError](t, &trp.MalformedResponseError{Detail: "no result"})
	assertDiscriminable[*trp.GenericRpcError](t, &trp.GenericRpcError{Code: -1, Message: "err"})
	assertDiscriminable[*trp.MissingTxArgError](t, &trp.MissingTxArgError{Key: "x"})
}

func TestErrorDiscriminationSigner(t *testing.T) {
	assertDiscriminable[*signer.InvalidMnemonicError](t, &signer.InvalidMnemonicError{})
	assertDiscriminable[*signer.InvalidPrivateKeyError](t, &signer.InvalidPrivateKeyError{Detail: "bad"})
	assertDiscriminable[*signer.InvalidHashError](t, &signer.InvalidHashError{Detail: "bad"})
	assertDiscriminable[*signer.InvalidAddressError](t, &signer.InvalidAddressError{Detail: "bad"})
	assertDiscriminable[*signer.AddressMismatchError](t, &signer.AddressMismatchError{})
}

func TestErrorDiscriminationFacade(t *testing.T) {
	assertDiscriminable[*facade.UnknownPartyError](t, &facade.UnknownPartyError{Name: "x"})
	assertDiscriminable[*facade.MissingParamsError](t, &facade.MissingParamsError{Params: []string{"a"}})
	assertDiscriminable[*facade.SubmitHashMismatchError](t, &facade.SubmitHashMismatchError{Expected: "a", Received: "b"})
	assertDiscriminable[*facade.FinalizedFailedError](t, &facade.FinalizedFailedError{Hash: "x", Stage: "dropped"})
	assertDiscriminable[*facade.FinalizedTimeoutError](t, &facade.FinalizedTimeoutError{Hash: "x", Attempts: 1, Delay: time.Second})
}

func TestErrorMessagesIncludeContext(t *testing.T) {
	// Verify error messages include relevant context (not just generic strings)
	err := &facade.UnknownPartyError{Name: "sender"}
	if msg := err.Error(); msg == "" || !contains(msg, "sender") {
		t.Errorf("error message should include party name, got: %s", msg)
	}

	err2 := &trp.HttpError{Status: 404, StatusText: "Not Found"}
	if msg := err2.Error(); !contains(msg, "404") {
		t.Errorf("error message should include HTTP status, got: %s", msg)
	}

	err3 := &facade.FinalizedTimeoutError{Hash: "abc", Attempts: 5, Delay: time.Second}
	if msg := err3.Error(); !contains(msg, "abc") || !contains(msg, "5") {
		t.Errorf("error message should include hash and attempts, got: %s", msg)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsImpl(s, substr))
}

func containsImpl(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// assertDiscriminable verifies that errors.As can extract a concrete error type
// from a wrapped error, proving discrimination without string matching.
func assertDiscriminable[T error](t *testing.T, original T) {
	t.Helper()
	wrapped := fmt.Errorf("wrapped: %w", original)

	// Verify non-empty error message
	if original.Error() == "" {
		t.Errorf("expected non-empty error message for %T", original)
	}

	// Verify errors.As can find the concrete type through wrapping
	var target T
	if !errors.As(wrapped, &target) {
		t.Errorf("errors.As failed to extract %T from wrapped error", original)
	}
}

// TestTiiErrorMarker verifies TII errors implement TiiError interface.
func TestTiiErrorMarker(t *testing.T) {
	var _ tii.TiiError = &tii.InvalidJSONError{}
	var _ tii.TiiError = &tii.UnknownTxError{}
	var _ tii.TiiError = &tii.UnknownProfileError{}
	var _ tii.TiiError = &tii.InvalidParamsSchemaError{}
	var _ tii.TiiError = &tii.InvalidParamTypeError{}
}

// TestTrpErrorMarker verifies TRP errors implement TrpError interface.
func TestTrpErrorMarker(t *testing.T) {
	var _ trp.TrpError = &trp.NetworkError{}
	var _ trp.TrpError = &trp.HttpError{}
	var _ trp.TrpError = &trp.DeserializationError{}
	var _ trp.TrpError = &trp.GenericRpcError{}
}

// TestSignerErrorMarker verifies signer errors implement SignerError interface.
func TestSignerErrorMarker(t *testing.T) {
	var _ signer.SignerError = &signer.InvalidMnemonicError{}
	var _ signer.SignerError = &signer.InvalidPrivateKeyError{}
	var _ signer.SignerError = &signer.InvalidHashError{}
	var _ signer.SignerError = &signer.InvalidAddressError{}
	var _ signer.SignerError = &signer.UnsupportedPaymentCredentialError{}
	var _ signer.SignerError = &signer.AddressMismatchError{}
}

// TestFacadeErrorMarker verifies facade errors implement FacadeError interface.
func TestFacadeErrorMarker(t *testing.T) {
	var _ facade.FacadeError = &facade.UnknownPartyError{}
	var _ facade.FacadeError = &facade.MissingParamsError{}
	var _ facade.FacadeError = &facade.SubmitHashMismatchError{}
	var _ facade.FacadeError = &facade.FinalizedFailedError{}
	var _ facade.FacadeError = &facade.FinalizedTimeoutError{}
}
