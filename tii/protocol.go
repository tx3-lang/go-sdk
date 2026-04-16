// Package tii handles loading and introspecting TII (Transaction Invoke Interface) protocol files.
package tii

import (
	"encoding/json"
	"os"
)

// Protocol is the in-memory representation of a loaded TII file.
// It exposes the protocol's transactions, parties, and profiles, and
// creates Invocation objects for building transaction requests.
type Protocol struct {
	spec TiiFile
}

// FromFile loads a Protocol from a .tii file at the given path.
func FromFile(path string) (*Protocol, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, &IOError{Path: path, Cause: err}
	}
	return FromBytes(data)
}

// FromString loads a Protocol from a JSON string.
func FromString(jsonStr string) (*Protocol, error) {
	return FromBytes([]byte(jsonStr))
}

// FromBytes loads a Protocol from raw JSON bytes.
func FromBytes(data []byte) (*Protocol, error) {
	var spec TiiFile
	if err := json.Unmarshal(data, &spec); err != nil {
		return nil, &InvalidJSONError{Cause: err}
	}
	return &Protocol{spec: spec}, nil
}

// Transactions returns the map of transaction names to their definitions.
func (p *Protocol) Transactions() map[string]Transaction {
	return p.spec.Transactions
}

// Parties returns the map of party names to their definitions.
func (p *Protocol) Parties() map[string]PartySpec {
	return p.spec.Parties
}

// Profiles returns the map of profile names to their definitions.
func (p *Protocol) Profiles() map[string]Profile {
	return p.spec.Profiles
}

// ProtocolInfo returns the protocol identification metadata.
func (p *Protocol) ProtocolInfo() ProtocolInfo {
	return p.spec.Protocol
}

// TiiVersion returns the TII format version.
func (p *Protocol) TiiVersion() string {
	return p.spec.Tii.Version
}

// Invoke creates a new Invocation for the named transaction, optionally
// pre-populated with environment values from the given profile.
func (p *Protocol) Invoke(txName string, profile *string) (*Invocation, error) {
	tx, ok := p.spec.Transactions[txName]
	if !ok {
		return nil, &UnknownTxError{Name: txName}
	}

	// Extract param types from the transaction schema
	params := make(map[string]ParamType)
	for name, schema := range tx.Params.Properties {
		pt, err := ParamTypeFromSchema(schema)
		if err != nil {
			return nil, err
		}
		params[name] = pt
	}

	// Build required set
	required := make(map[string]bool)
	for _, r := range tx.Params.Required {
		required[r] = true
	}

	inv := &Invocation{
		tir:      tx.Tir,
		params:   params,
		required: required,
		args:     make(map[string]interface{}),
	}

	// Pre-populate from profile if provided
	if profile != nil {
		prof, ok := p.spec.Profiles[*profile]
		if !ok {
			return nil, &UnknownProfileError{Name: *profile}
		}
		// Inject environment values as args
		for k, v := range prof.Environment {
			inv.args[k] = v
		}
	}

	return inv, nil
}
