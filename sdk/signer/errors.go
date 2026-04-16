package signer

import "fmt"

// SignerError is the marker interface for all signer-related errors.
type SignerError interface {
	error
	isSignerError()
}

// InvalidMnemonicError indicates a malformed BIP39 mnemonic phrase.
type InvalidMnemonicError struct {
	Cause error
}

func (e *InvalidMnemonicError) Error() string {
	return fmt.Sprintf("invalid mnemonic: %v", e.Cause)
}
func (e *InvalidMnemonicError) Unwrap() error  { return e.Cause }
func (e *InvalidMnemonicError) isSignerError() {}

// InvalidPrivateKeyError indicates a problem with the private key format or length.
type InvalidPrivateKeyError struct {
	Detail string
	Cause  error
}

func (e *InvalidPrivateKeyError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("invalid private key: %s: %v", e.Detail, e.Cause)
	}
	return fmt.Sprintf("invalid private key: %s", e.Detail)
}
func (e *InvalidPrivateKeyError) Unwrap() error  { return e.Cause }
func (e *InvalidPrivateKeyError) isSignerError() {}

// InvalidHashError indicates a problem with the transaction hash format or length.
type InvalidHashError struct {
	Detail string
	Cause  error
}

func (e *InvalidHashError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("invalid hash: %s: %v", e.Detail, e.Cause)
	}
	return fmt.Sprintf("invalid hash: %s", e.Detail)
}
func (e *InvalidHashError) Unwrap() error  { return e.Cause }
func (e *InvalidHashError) isSignerError() {}

// InvalidAddressError indicates a problem decoding a blockchain address.
type InvalidAddressError struct {
	Detail string
	Cause  error
}

func (e *InvalidAddressError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("invalid address: %s: %v", e.Detail, e.Cause)
	}
	return fmt.Sprintf("invalid address: %s", e.Detail)
}
func (e *InvalidAddressError) Unwrap() error  { return e.Cause }
func (e *InvalidAddressError) isSignerError() {}

// UnsupportedPaymentCredentialError indicates the address uses a script hash
// instead of a key hash, which is not supported for signing.
type UnsupportedPaymentCredentialError struct{}

func (e *UnsupportedPaymentCredentialError) Error() string {
	return "unsupported payment credential: script hash addresses cannot be used with signers"
}
func (e *UnsupportedPaymentCredentialError) isSignerError() {}

// AddressMismatchError indicates the signer's public key does not match
// the payment key hash in the provided address.
type AddressMismatchError struct{}

func (e *AddressMismatchError) Error() string {
	return "address mismatch: signer public key does not match address payment key hash"
}
func (e *AddressMismatchError) isSignerError() {}
