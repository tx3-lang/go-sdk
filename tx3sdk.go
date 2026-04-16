// Package tx3sdk is the Tx3 SDK for Go — a client library for defining and
// executing UTxO-based blockchain transactions declaratively using the Tx3 protocol.
//
// Quick start:
//
//	protocol, _ := tx3sdk.ProtocolFromFile("transfer.tii")
//	trpClient := tx3sdk.NewTRPClient(tx3sdk.TRPClientOptions{Endpoint: "http://localhost:3000"})
//
//	client := tx3sdk.NewClient(protocol, trpClient).
//	    WithProfile("preprod").
//	    WithParty("sender", tx3sdk.SignerParty(mySigner)).
//	    WithParty("receiver", tx3sdk.AddressParty("addr_test1..."))
//
//	status, _ := client.Tx("transfer").
//	    Arg("quantity", 10_000_000).
//	    Resolve(ctx).
//	    Sign().
//	    Submit(ctx).
//	    WaitForConfirmed(ctx, tx3sdk.DefaultPollConfig())
package tx3sdk

import (
	"github.com/tx3-lang/go-sdk/facade"
	"github.com/tx3-lang/go-sdk/signer"
	"github.com/tx3-lang/go-sdk/tii"
	"github.com/tx3-lang/go-sdk/trp"
)

// --- Protocol Loading (§3.1) ---

// ProtocolFromFile loads a Protocol from a .tii file at the given path.
func ProtocolFromFile(path string) (*tii.Protocol, error) {
	return tii.FromFile(path)
}

// ProtocolFromString loads a Protocol from a JSON string.
func ProtocolFromString(jsonStr string) (*tii.Protocol, error) {
	return tii.FromString(jsonStr)
}

// ProtocolFromBytes loads a Protocol from raw JSON bytes.
func ProtocolFromBytes(data []byte) (*tii.Protocol, error) {
	return tii.FromBytes(data)
}

// --- TRP Client (§3.2) ---

// TRPClientOptions configures a TRP client.
type TRPClientOptions = trp.ClientOptions

// NewTRPClient creates a new low-level TRP JSON-RPC client.
func NewTRPClient(options TRPClientOptions) *trp.Client {
	return trp.NewClient(options)
}

// --- Facade (§3.3) ---

// Tx3Client is the high-level entry point for building and submitting transactions.
type Tx3Client = facade.Tx3Client

// NewClient creates a new Tx3Client with the given protocol and TRP client.
func NewClient(protocol *tii.Protocol, trpClient *trp.Client) *Tx3Client {
	return facade.NewClient(protocol, trpClient)
}

// TxBuilder collects transaction arguments and resolves the transaction.
type TxBuilder = facade.TxBuilder

// ResolvedTx is a transaction resolved by TRP, ready for signing.
type ResolvedTx = facade.ResolvedTx

// SignedTx is a signed transaction ready for submission.
type SignedTx = facade.SignedTx

// SubmittedTx is a submitted transaction that can be polled for status.
type SubmittedTx = facade.SubmittedTx

// --- Parties (§3.4) ---

// Party represents a named participant in a transaction.
type Party = facade.Party

// AddressParty creates a read-only party that provides only an address.
func AddressParty(address string) Party {
	return facade.AddressParty(address)
}

// SignerParty creates a signer party. The address is read from the signer.
func SignerParty(s signer.Signer) Party {
	return facade.SignerParty(s)
}

// --- Signers (§3.5) ---

// Signer is the interface for transaction signing.
type Signer = signer.Signer

// Ed25519Signer is a generic raw-key signer using Ed25519.
type Ed25519Signer = signer.Ed25519Signer

// CardanoSigner is a Cardano-specific signer using BIP32-Ed25519 key derivation.
type CardanoSigner = signer.CardanoSigner

// NewEd25519Signer creates an Ed25519Signer from a 32-byte private key seed and address.
func NewEd25519Signer(address string, privateKey [32]byte) *Ed25519Signer {
	return signer.NewEd25519Signer(address, privateKey)
}

// Ed25519SignerFromHex creates an Ed25519Signer from a hex-encoded private key.
func Ed25519SignerFromHex(address, privateKeyHex string) (*Ed25519Signer, error) {
	return signer.Ed25519SignerFromHex(address, privateKeyHex)
}

// Ed25519SignerFromMnemonic creates an Ed25519Signer from a BIP39 mnemonic phrase.
func Ed25519SignerFromMnemonic(address, phrase string) (*Ed25519Signer, error) {
	return signer.Ed25519SignerFromMnemonic(address, phrase)
}

// CardanoSignerFromMnemonic creates a CardanoSigner from a BIP39 mnemonic phrase.
// Keys are derived at the standard Cardano path m/1852'/1815'/0'/0/0.
func CardanoSignerFromMnemonic(address, phrase string) (*CardanoSigner, error) {
	return signer.CardanoSignerFromMnemonic(address, phrase)
}

// CardanoSignerFromHex creates a CardanoSigner from a hex-encoded extended private key.
func CardanoSignerFromHex(address, privateKeyHex string) (*CardanoSigner, error) {
	return signer.CardanoSignerFromHex(address, privateKeyHex)
}

// --- Poll Config (§3.7) ---

// PollConfig controls the polling behaviour of wait methods.
type PollConfig = facade.PollConfig

// DefaultPollConfig returns a PollConfig with sensible defaults (20 attempts, 5s delay).
func DefaultPollConfig() PollConfig {
	return facade.DefaultPollConfig()
}
