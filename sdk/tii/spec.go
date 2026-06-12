package tii

import (
	"bytes"
	"encoding/json"

	"github.com/tx3-lang/go-sdk/sdk/core"
)

// TiiFile represents the top-level structure of a .tii file.
type TiiFile struct {
	Tii          TiiInfo                `json:"tii"`
	Protocol     ProtocolInfo           `json:"protocol"`
	Environment  *Schema                `json:"environment,omitempty"`
	Parties      map[string]PartySpec   `json:"parties"`
	Transactions map[string]Transaction `json:"transactions"`
	Profiles     map[string]Profile     `json:"profiles"`
	Components   *Components            `json:"components,omitempty"`
}

// Components holds the reusable schemas referenced by
// "#/components/schemas/<Name>" param refs (user-defined records / variants).
type Components struct {
	Schemas map[string]Schema `json:"schemas"`
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
	Type                 string            `json:"type,omitempty"`
	Properties           map[string]Schema `json:"properties,omitempty"`
	Required             []string          `json:"required,omitempty"`
	Ref                  string            `json:"$ref,omitempty"`
	Items                *Schema           `json:"items,omitempty"`
	PrefixItems          []Schema          `json:"prefixItems,omitempty"`
	AdditionalProperties *Schema           `json:"additionalProperties,omitempty"`
	OneOf                []Schema          `json:"oneOf,omitempty"`
}

// UnmarshalJSON decodes a Schema, tolerating the boolean forms `tx3c` emits for
// `items` (a tuple sets `"items": false`) and `additionalProperties` (a variant
// case sets `"additionalProperties": false`). A `*Schema` field cannot decode a
// bare `false`, so those keys are only populated when they hold a schema object.
func (s *Schema) UnmarshalJSON(data []byte) error {
	type rawSchema struct {
		Type                 string            `json:"type"`
		Properties           map[string]Schema `json:"properties"`
		Required             []string          `json:"required"`
		Ref                  string            `json:"$ref"`
		Items                json.RawMessage   `json:"items"`
		PrefixItems          []Schema          `json:"prefixItems"`
		AdditionalProperties json.RawMessage   `json:"additionalProperties"`
		OneOf                []Schema          `json:"oneOf"`
	}

	var raw rawSchema
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	s.Type = raw.Type
	s.Properties = raw.Properties
	s.Required = raw.Required
	s.Ref = raw.Ref
	s.PrefixItems = raw.PrefixItems
	s.OneOf = raw.OneOf
	s.Items = objectSchema(raw.Items)
	s.AdditionalProperties = objectSchema(raw.AdditionalProperties)
	return nil
}

// objectSchema decodes a JSON value into a *Schema only when it is an object,
// returning nil for the boolean / absent forms (e.g. `items: false`).
func objectSchema(raw json.RawMessage) *Schema {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || trimmed[0] != '{' {
		return nil
	}
	var sub Schema
	if err := json.Unmarshal(trimmed, &sub); err != nil {
		return nil
	}
	return &sub
}
