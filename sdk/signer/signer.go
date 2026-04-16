// Package signer defines the Signer interface and built-in implementations
// for Ed25519 and Cardano (BIP32-Ed25519) key signing.
package signer

// Signer produces a cryptographic witness for a transaction hash.
// Users may implement custom signers by satisfying this interface.
type Signer interface {
	// Address returns the bech32 address associated with this signer.
	Address() string

	// Sign produces a TxWitness for the given transaction hash (hex-encoded, 32 bytes).
	Sign(txHashHex string) (*TxWitness, error)
}
