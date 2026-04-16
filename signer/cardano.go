package signer

import (
	"crypto/hmac"
	"crypto/sha512"
	"encoding/binary"
	"encoding/hex"
	"fmt"

	"filippo.io/edwards25519"
	"github.com/tyler-smith/go-bip39"
	"golang.org/x/crypto/blake2b"
	"golang.org/x/crypto/pbkdf2"
)

// CardanoSigner implements the Signer interface for Cardano wallets using
// BIP32-Ed25519 key derivation following the CIP-3 (Icarus) scheme.
// Keys are derived at path m/1852'/1815'/0'/0/0.
type CardanoSigner struct {
	address        string
	extendedSecret [64]byte // 64-byte extended secret key (clamp-modified seed)
	publicKey      [32]byte
}

// CardanoSignerFromMnemonic creates a CardanoSigner by deriving keys from a BIP39
// mnemonic using the Cardano Icarus (CIP-3) derivation scheme at m/1852'/1815'/0'/0/0.
func CardanoSignerFromMnemonic(address, phrase string) (*CardanoSigner, error) {
	if !bip39.IsMnemonicValid(phrase) {
		return nil, &InvalidMnemonicError{Cause: fmt.Errorf("invalid BIP39 mnemonic")}
	}

	entropy, err := bip39.EntropyFromMnemonic(phrase)
	if err != nil {
		return nil, &InvalidMnemonicError{Cause: err}
	}

	// Icarus master key derivation: PBKDF2-HMAC-SHA512 with empty password
	masterKey := pbkdf2.Key([]byte(""), entropy, 4096, 96, sha512.New)

	// Split into extended secret (64 bytes) and chain code (32 bytes)
	var extSecret [64]byte
	var chainCode [32]byte
	copy(extSecret[:], masterKey[:64])
	copy(chainCode[:], masterKey[64:96])

	// Clamp the secret key (Ed25519-BIP32 requirement)
	extSecret[0] &= 0xF8
	extSecret[31] &= 0x1F
	extSecret[31] |= 0x40

	// Derive m/1852'/1815'/0'/0/0
	path := []uint32{
		0x80000000 | 1852, // 1852' (CIP-1852 purpose)
		0x80000000 | 1815, // 1815' (Cardano coin type)
		0x80000000 | 0,    // 0' (account)
		0,                 // 0 (external chain)
		0,                 // 0 (first address)
	}

	currentSecret := extSecret
	currentChain := chainCode
	for _, idx := range path {
		currentSecret, currentChain = deriveChild(currentSecret, currentChain, idx)
	}

	// Compute public key from the derived extended secret
	pubKey := extendedSecretToPublic(currentSecret)

	signer := &CardanoSigner{
		address:        address,
		extendedSecret: currentSecret,
		publicKey:      pubKey,
	}

	// Verify address binding
	if err := signer.verifyAddressBinding(); err != nil {
		return nil, err
	}

	return signer, nil
}

// CardanoSignerFromHex creates a CardanoSigner from a hex-encoded extended private key
// (64 bytes) or a 32-byte seed.
func CardanoSignerFromHex(address, privateKeyHex string) (*CardanoSigner, error) {
	keyBytes, err := hex.DecodeString(privateKeyHex)
	if err != nil {
		return nil, &InvalidPrivateKeyError{Detail: "hex decode failed", Cause: err}
	}

	var extSecret [64]byte
	switch len(keyBytes) {
	case 64:
		copy(extSecret[:], keyBytes)
	case 32:
		// Treat as seed: hash with SHA-512 and clamp
		h := sha512.Sum512(keyBytes)
		copy(extSecret[:], h[:])
		extSecret[0] &= 0xF8
		extSecret[31] &= 0x1F
		extSecret[31] |= 0x40
	default:
		return nil, &InvalidPrivateKeyError{
			Detail: fmt.Sprintf("expected 32 or 64 bytes, got %d", len(keyBytes)),
		}
	}

	pubKey := extendedSecretToPublic(extSecret)

	signer := &CardanoSigner{
		address:        address,
		extendedSecret: extSecret,
		publicKey:      pubKey,
	}

	if err := signer.verifyAddressBinding(); err != nil {
		return nil, err
	}

	return signer, nil
}

// Address returns the bech32 address associated with this signer.
func (s *CardanoSigner) Address() string {
	return s.address
}

// Sign produces a VKey witness for the given transaction hash (hex-encoded).
func (s *CardanoSigner) Sign(txHashHex string) (*TxWitness, error) {
	hashBytes, err := hex.DecodeString(txHashHex)
	if err != nil {
		return nil, &InvalidHashError{Detail: "hex decode failed", Cause: err}
	}
	if len(hashBytes) != 32 {
		return nil, &InvalidHashError{
			Detail: fmt.Sprintf("expected 32 bytes, got %d", len(hashBytes)),
		}
	}

	signature := extendedSign(s.extendedSecret, s.publicKey, hashBytes)

	return NewVKeyWitness(
		hex.EncodeToString(s.publicKey[:]),
		hex.EncodeToString(signature),
	), nil
}

// verifyAddressBinding checks that the signer's public key matches the
// payment key hash in the provided Cardano address.
func (s *CardanoSigner) verifyAddressBinding() error {
	paymentKeyHash, err := extractPaymentKeyHash(s.address)
	if err != nil {
		return err
	}

	// Hash our public key with blake2b-224
	hasher, _ := blake2b.New(28, nil) // 224 bits = 28 bytes
	hasher.Write(s.publicKey[:])
	ourHash := hasher.Sum(nil)

	if !bytesEqual(ourHash, paymentKeyHash) {
		return &AddressMismatchError{}
	}
	return nil
}

// extractPaymentKeyHash decodes a Cardano bech32 address and extracts
// the payment key hash (28 bytes).
func extractPaymentKeyHash(addr string) ([]byte, error) {
	_, data, err := bech32Decode(addr)
	if err != nil {
		return nil, &InvalidAddressError{Detail: "bech32 decode failed", Cause: err}
	}
	if len(data) < 1 {
		return nil, &InvalidAddressError{Detail: "address too short"}
	}

	header := data[0]
	addrType := header >> 4

	// Shelley base addresses (types 0x00-0x03) and enterprise (0x06-0x07)
	switch addrType {
	case 0x00, 0x01, 0x02, 0x03:
		// Base addresses: header + 28-byte payment key hash + 28-byte stake key hash
		if header&0x10 != 0 {
			// Bit 4 set = script hash payment credential
			return nil, &UnsupportedPaymentCredentialError{}
		}
		if len(data) < 29 {
			return nil, &InvalidAddressError{Detail: "address too short for base address"}
		}
		return data[1:29], nil
	case 0x06, 0x07:
		// Enterprise addresses: header + 28-byte payment key hash
		if header&0x10 != 0 {
			return nil, &UnsupportedPaymentCredentialError{}
		}
		if len(data) < 29 {
			return nil, &InvalidAddressError{Detail: "address too short for enterprise address"}
		}
		return data[1:29], nil
	default:
		return nil, &InvalidAddressError{
			Detail: fmt.Sprintf("unsupported address type: 0x%02x", addrType),
		}
	}
}

// deriveChild derives a child key from a parent extended secret and chain code
// using Ed25519-BIP32 (Icarus/CIP-3).
func deriveChild(parentSecret [64]byte, parentChain [32]byte, index uint32) ([64]byte, [32]byte) {
	isHardened := index >= 0x80000000

	var data []byte
	if isHardened {
		// Hardened: 0x00 || parent_secret (64) || index_le (4)
		data = make([]byte, 1+64+4)
		data[0] = 0x00
		copy(data[1:], parentSecret[:])
	} else {
		// Normal: 0x02 || parent_public (32) || index_le (4)
		pubKey := extendedSecretToPublic(parentSecret)
		data = make([]byte, 1+32+4)
		data[0] = 0x02
		copy(data[1:], pubKey[:])
	}
	binary.LittleEndian.PutUint32(data[len(data)-4:], index)

	// HMAC-SHA512
	mac := hmac.New(sha512.New, parentChain[:])
	mac.Write(data)
	z := mac.Sum(nil) // 64 bytes

	// For chain code derivation, change tag byte
	if isHardened {
		data[0] = 0x01
	} else {
		data[0] = 0x03
	}
	mac2 := hmac.New(sha512.New, parentChain[:])
	mac2.Write(data)
	cc := mac2.Sum(nil)

	// Derive child secret
	var childSecret [64]byte

	// zl (first 28 bytes of z, multiplied by 8, added to parent kl)
	zl := z[:32]
	// Multiply zl by 8 and add to parent secret left half
	var carry uint32
	for i := 0; i < 28; i++ {
		sum := uint32(parentSecret[i]) + uint32(zl[i])*8 + carry
		childSecret[i] = byte(sum & 0xFF)
		carry = sum >> 8
	}
	// Remaining bytes of the left half (28-31): just add with carry
	for i := 28; i < 32; i++ {
		sum := uint32(parentSecret[i]) + carry
		childSecret[i] = byte(sum & 0xFF)
		carry = sum >> 8
	}

	// zr (last 32 bytes of z, added to parent secret right half)
	zr := z[32:]
	carry = 0
	for i := 0; i < 32; i++ {
		sum := uint32(parentSecret[32+i]) + uint32(zr[i]) + carry
		childSecret[32+i] = byte(sum & 0xFF)
		carry = sum >> 8
	}

	var childChain [32]byte
	copy(childChain[:], cc[32:64])

	return childSecret, childChain
}

// extendedSecretToPublic computes the Ed25519 public key from a 64-byte extended secret.
func extendedSecretToPublic(secret [64]byte) [32]byte {
	// The first 32 bytes of the extended secret (already clamped) are the scalar
	scalar, _ := edwards25519.NewScalar().SetBytesWithClamping(secret[:32])
	point := edwards25519.NewGeneratorPoint().ScalarBaseMult(scalar)
	var pub [32]byte
	copy(pub[:], point.Bytes())
	return pub
}

// extendedSign signs a message using the extended Ed25519 secret key.
// This follows the Ed25519-BIP32 signing algorithm.
func extendedSign(secret [64]byte, publicKey [32]byte, message []byte) []byte {
	// r = SHA-512(secret_right || message)
	rHash := sha512.New()
	rHash.Write(secret[32:64])
	rHash.Write(message)
	rDigest := rHash.Sum(nil)

	// Reduce r mod L (Ed25519 group order)
	rScalar, _ := edwards25519.NewScalar().SetUniformBytes(rDigest)
	R := edwards25519.NewGeneratorPoint().ScalarBaseMult(rScalar)

	// S = r + SHA-512(R || public_key || message) * secret_left
	hramHash := sha512.New()
	hramHash.Write(R.Bytes())
	hramHash.Write(publicKey[:])
	hramHash.Write(message)
	hramDigest := hramHash.Sum(nil)

	hramScalar, _ := edwards25519.NewScalar().SetUniformBytes(hramDigest)

	// a = secret_left (the scalar, not clamped again since it's already clamped in the extended secret)
	aScalar, _ := edwards25519.NewScalar().SetBytesWithClamping(secret[:32])

	// S = r + hram * a
	S := edwards25519.NewScalar().MultiplyAdd(hramScalar, aScalar, rScalar)

	// Signature = R || S
	sig := make([]byte, 64)
	copy(sig[:32], R.Bytes())
	copy(sig[32:], S.Bytes())
	return sig
}

func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
