package signer_test

import (
	"crypto/ed25519"
	"encoding/hex"
	"testing"

	"github.com/tx3-lang/go-sdk/sdk/signer"
)

func TestEd25519SignerFromKnownKey(t *testing.T) {
	// Known 32-byte seed
	seedHex := "9d61b19deffd5a60ba844af492ec2cc44449c5697b326919703bac031cae7f60"
	address := "addr_test1qz..."

	s, err := signer.Ed25519SignerFromHex(address, seedHex)
	if err != nil {
		t.Fatalf("Ed25519SignerFromHex failed: %v", err)
	}

	if s.Address() != address {
		t.Errorf("expected address %q, got %q", address, s.Address())
	}

	// Sign a known 32-byte hash
	hashHex := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	w, err := s.Sign(hashHex)
	if err != nil {
		t.Fatalf("Sign failed: %v", err)
	}

	// Verify witness structure
	if w.WitnessType != "vkey" {
		t.Errorf("expected witness type 'vkey', got %q", w.WitnessType)
	}
	if w.Key.ContentType != "hex" {
		t.Errorf("expected key content type 'hex', got %q", w.Key.ContentType)
	}
	if w.Signature.ContentType != "hex" {
		t.Errorf("expected signature content type 'hex', got %q", w.Signature.ContentType)
	}

	// Verify the signature is valid using Go's ed25519
	seedBytes, _ := hex.DecodeString(seedHex)
	privKey := ed25519.NewKeyFromSeed(seedBytes)
	pubKey := privKey.Public().(ed25519.PublicKey)
	hashBytes, _ := hex.DecodeString(hashHex)
	sigBytes, _ := hex.DecodeString(w.Signature.Content)

	if !ed25519.Verify(pubKey, hashBytes, sigBytes) {
		t.Error("signature verification failed")
	}

	// Verify public key matches
	expectedPubHex := hex.EncodeToString(pubKey)
	if w.Key.Content != expectedPubHex {
		t.Errorf("public key mismatch: expected %s, got %s", expectedPubHex, w.Key.Content)
	}
}

func TestEd25519SignerInvalidKeyLength(t *testing.T) {
	_, err := signer.Ed25519SignerFromHex("addr", "aabb") // 2 bytes, not 32
	if err == nil {
		t.Fatal("expected error for invalid key length")
	}
	var keyErr *signer.InvalidPrivateKeyError
	if !isErrType(err, &keyErr) {
		t.Fatalf("expected InvalidPrivateKeyError, got %T: %v", err, err)
	}
}

func TestEd25519SignerInvalidHashLength(t *testing.T) {
	seedHex := "9d61b19deffd5a60ba844af492ec2cc44449c5697b326919703bac031cae7f60"
	s, _ := signer.Ed25519SignerFromHex("addr", seedHex)

	_, err := s.Sign("aabb") // 2 bytes, not 32
	if err == nil {
		t.Fatal("expected error for invalid hash length")
	}
	var hashErr *signer.InvalidHashError
	if !isErrType(err, &hashErr) {
		t.Fatalf("expected InvalidHashError, got %T: %v", err, err)
	}
}

func TestEd25519SignerFromMnemonic(t *testing.T) {
	phrase := "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"
	address := "addr_test1qz..."

	s, err := signer.Ed25519SignerFromMnemonic(address, phrase)
	if err != nil {
		t.Fatalf("Ed25519SignerFromMnemonic failed: %v", err)
	}

	if s.Address() != address {
		t.Errorf("expected address %q, got %q", address, s.Address())
	}

	// Should be able to sign
	hashHex := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	w, err := s.Sign(hashHex)
	if err != nil {
		t.Fatalf("Sign failed: %v", err)
	}
	if w.WitnessType != "vkey" {
		t.Error("expected vkey witness type")
	}
}

func TestCustomSignerIntegration(t *testing.T) {
	// Implement a custom signer
	custom := &mockSigner{
		address: "addr_mock",
		pubKey:  "aabbccdd",
		sig:     "11223344",
	}

	// Verify it satisfies the Signer interface
	var s signer.Signer = custom
	if s.Address() != "addr_mock" {
		t.Errorf("expected address 'addr_mock', got %q", s.Address())
	}

	w, err := s.Sign("e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855")
	if err != nil {
		t.Fatalf("Sign failed: %v", err)
	}
	if w.Key.Content != "aabbccdd" {
		t.Errorf("expected public key 'aabbccdd', got %q", w.Key.Content)
	}
}

// mockSigner is a test helper implementing the Signer interface.
type mockSigner struct {
	address string
	pubKey  string
	sig     string
}

func (m *mockSigner) Address() string { return m.address }
func (m *mockSigner) Sign(txHashHex string) (*signer.TxWitness, error) {
	return signer.NewVKeyWitness(m.pubKey, m.sig), nil
}

// isErrType is a helper for errors.As with a pointer target.
func isErrType[T any](err error, target *T) bool {
	return err != nil && func() bool {
		var t T
		return asErr(err, &t)
	}()
}

func asErr[T any](err error, target *T) bool {
	return err != nil && func() bool {
		switch e := err.(type) {
		case T:
			*target = e
			return true
		default:
			return false
		}
	}()
}
