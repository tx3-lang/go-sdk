package tii

import (
	"github.com/tx3-lang/go-sdk/core"
)

// TiiFile represents the top-level structure of a .tii file.
type TiiFile struct {
	Tii          TiiInfo                `json:"tii"`
	Protocol     ProtocolInfo           `json:"protocol"`
	Environment  *Schema                `json:"environment,omitempty"`
	Parties      map[string]PartySpec   `json:"parties"`
	Transactions map[string]Transaction `json:"transactions"`
	Profiles     map[string]Profile     `json:"profiles"`
}

// TiiInfo holds TII format version metadata.
type TiiInfo struct {
	Version string `json:"version"`
}

// ProtocolInfo holds protocol identification metadata.
type ProtocolInfo struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Scope       string `json:"scope"`
	Description string `json:"description,omitempty"`
}

// PartySpec represents a party definition in the TII file.
// Currently an empty struct; may gain fields in future TII versions.
type PartySpec struct{}

// Transaction represents a single transaction definition in the TII file.
type Transaction struct {
	Tir         core.TirEnvelope `json:"tir"`
	Params      Schema           `json:"params"`
	Description string           `json:"description,omitempty"`
}

// Profile represents a named environment configuration (e.g., "preprod", "mainnet").
type Profile struct {
	Description string                 `json:"description,omitempty"`
	Environment map[string]interface{} `json:"environment"`
	Parties     map[string]string      `json:"parties"`
}

// Schema is a JSON Schema object used for parameter definitions.
// We use a thin wrapper to support type introspection.
type Schema struct {
	Type       string            `json:"type,omitempty"`
	Properties map[string]Schema `json:"properties,omitempty"`
	Required   []string          `json:"required,omitempty"`
	Ref        string            `json:"$ref,omitempty"`
	Items      *Schema           `json:"items,omitempty"`
}
