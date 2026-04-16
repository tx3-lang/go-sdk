// Package trp implements the low-level TRP (Transaction Resolve Protocol) JSON-RPC client.
package trp

import (
	"github.com/tx3-lang/go-sdk/sdk/core"
)

// ResolveParams is the request body for the trp.resolve JSON-RPC method.
type ResolveParams struct {
	Tir  core.TirEnvelope       `json:"tir"`
	Args map[string]interface{} `json:"args"`
	Env  map[string]interface{} `json:"env,omitempty"`
}

// TxEnvelope is the response from trp.resolve, containing the resolved transaction.
type TxEnvelope struct {
	Hash string `json:"hash"` // Transaction hash (hex)
	Tx   string `json:"tx"`   // CBOR transaction bytes (hex)
}

// SubmitParams is the request body for the trp.submit JSON-RPC method.
type SubmitParams struct {
	Tx        core.BytesEnvelope `json:"tx"`
	Witnesses []TxWitness        `json:"witnesses"`
}

// TxWitness represents a cryptographic witness for a transaction.
type TxWitness struct {
	Key         core.BytesEnvelope `json:"key"`
	Signature   core.BytesEnvelope `json:"signature"`
	WitnessType string             `json:"type"`
}

// SubmitResponse is the response from trp.submit.
type SubmitResponse struct {
	Hash string `json:"hash"`
}

// CheckStatusParams is the request body for the trp.checkStatus JSON-RPC method.
type CheckStatusParams struct {
	Hashes []string `json:"hashes"`
}

// CheckStatusResponse is the response from trp.checkStatus.
type CheckStatusResponse struct {
	Statuses map[string]TxStatus `json:"statuses"`
}

// TxStatus represents the on-chain status of a submitted transaction.
type TxStatus struct {
	Stage             TxStage     `json:"stage"`
	Confirmations     uint64      `json:"confirmations"`
	NonConfirmations  uint64      `json:"nonConfirmations"`
	ConfirmedAt       *ChainPoint `json:"confirmedAt,omitempty"`
}

// ChainPoint identifies a specific point on the blockchain.
type ChainPoint struct {
	Slot      uint64 `json:"slot"`
	BlockHash string `json:"blockHash"`
}

// TxStage represents the lifecycle stage of a transaction.
type TxStage string

const (
	StagePending      TxStage = "pending"
	StagePropagated   TxStage = "propagated"
	StageAcknowledged TxStage = "acknowledged"
	StageConfirmed    TxStage = "confirmed"
	StageFinalized    TxStage = "finalized"
	StageDropped      TxStage = "dropped"
	StageRolledBack   TxStage = "rolledBack"
	StageUnknown      TxStage = "unknown"
)

// IsTerminalFailure returns true if the stage represents a terminal failure state.
func (s TxStage) IsTerminalFailure() bool {
	return s == StageDropped || s == StageRolledBack
}

// WitnessInfo holds diagnostic information about a witness produced during signing.
type WitnessInfo struct {
	PublicKey string
	Address   string
}
