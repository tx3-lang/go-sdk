package signer

import (
	"crypto/ed25519"
	"encoding/hex"
	"fmt"

	"github.com/tyler-smith/go-bip39"
)

// Ed25519Signer is a generic raw-key signer using Ed25519.
// Constructed from a 32-byte private key seed and an address.
type Ed25519Signer struct {
	address    string
	privateKey ed25519.PrivateKey
}

// NewEd25519Signer creates an Ed25519Signer from a 32-byte private key seed and address.
func NewEd25519Signer(address string, privateKey [32]byte) *Ed25519Signer {
	return &Ed25519Signer{
		address:    address,
		privateKey: ed25519.NewKeyFromSeed(privateKey[:]),
	}
}

// Ed25519SignerFromHex creates an Ed25519Signer from a hex-encoded private key seed.
func Ed25519SignerFromHex(address, privateKeyHex string) (*Ed25519Signer, error) {
	keyBytes, err := hex.DecodeString(privateKeyHex)
	if err != nil {
		return nil, &InvalidPrivateKeyError{Detail: "hex decode failed", Cause: err}
	}
	if len(keyBytes) != 32 {
		return nil, &InvalidPrivateKeyError{
			Detail: fmt.Sprintf("expected 32 bytes, got %d", len(keyBytes)),
		}
	}
	var key [32]byte
	copy(key[:], keyBytes)
	return NewEd25519Signer(address, key), nil
}

// Ed25519SignerFromMnemonic creates an Ed25519Signer from a BIP39 mnemonic phrase.
// Uses the first 32 bytes of the BIP39 seed as the Ed25519 private key seed.
func Ed25519SignerFromMnemonic(address, phrase string) (*Ed25519Signer, error) {
	if !bip39.IsMnemonicValid(phrase) {
		return nil, &InvalidMnemonicError{Cause: fmt.Errorf("invalid BIP39 mnemonic")}
	}
	seed := bip39.NewSeed(phrase, "")
	var key [32]byte
	copy(key[:], seed[:32])
	return NewEd25519Signer(address, key), nil
}

// Address returns the bech32 address associated with this signer.
func (s *Ed25519Signer) Address() string {
	return s.address
}

// Sign produces a VKey witness for the given transaction hash (hex-encoded).
func (s *Ed25519Signer) Sign(txHashHex string) (*TxWitness, error) {
	hashBytes, err := hex.DecodeString(txHashHex)
	if err != nil {
		return nil, &InvalidHashError{Detail: "hex decode failed", Cause: err}
	}
	if len(hashBytes) != 32 {
		return nil, &InvalidHashError{
			Detail: fmt.Sprintf("expected 32 bytes, got %d", len(hashBytes)),
		}
	}

	signature := ed25519.Sign(s.privateKey, hashBytes)
	publicKey := s.privateKey.Public().(ed25519.PublicKey)

	return NewVKeyWitness(
		hex.EncodeToString(publicKey),
		hex.EncodeToString(signature),
	), nil
}
