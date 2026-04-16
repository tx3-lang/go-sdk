package signer

import "github.com/tx3-lang/go-sdk/sdk/core"

// WitnessType identifies the kind of witness.
const WitnessTypeVKey = "vkey"

// TxWitness is a cryptographic witness binding a signature to a transaction hash.
type TxWitness struct {
	Key         core.BytesEnvelope `json:"key"`
	Signature   core.BytesEnvelope `json:"signature"`
	WitnessType string             `json:"type"`
}

// NewVKeyWitness creates a VKey witness from hex-encoded public key and signature.
func NewVKeyWitness(publicKeyHex, signatureHex string) *TxWitness {
	return &TxWitness{
		Key:         core.NewHexEnvelope(publicKeyHex),
		Signature:   core.NewHexEnvelope(signatureHex),
		WitnessType: WitnessTypeVKey,
	}
}
