// Package signer defines the Signer interface and built-in implementations
// for Ed25519 and Cardano (BIP32-Ed25519) key signing.
package signer

// SignRequest carries the inputs passed to a Signer for each sign call.
//
// Hash-based signers (CardanoSigner, Ed25519Signer) read TxHashHex; tx-based
// signers (e.g. wallet adapters that need the full tx body) read TxCborHex.
// The SDK always populates both fields.
type SignRequest struct {
	// TxHashHex is the hex-encoded tx hash bound to this signing call.
	TxHashHex string
	// TxCborHex is the hex-encoded full tx CBOR.
	TxCborHex string
}

// Signer produces a cryptographic witness for a SignRequest.
// Users may implement custom signers by satisfying this interface.
type Signer interface {
	// Address returns the bech32 address associated with this signer.
	Address() string

	// Sign produces a TxWitness for the given SignRequest.
	Sign(request SignRequest) (*TxWitness, error)
}
