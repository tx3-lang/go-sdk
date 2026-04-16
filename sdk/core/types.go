// Package core provides fundamental data types shared across all Tx3 SDK packages.
package core

// ArgMap holds transaction arguments as a string-keyed map of arbitrary values.
// Keys are matched case-insensitively against protocol-declared parameter names.
type ArgMap map[string]interface{}

// EnvMap holds environment variables for a TRP invocation.
type EnvMap map[string]interface{}

// Address is a bech32-encoded blockchain address.
type Address = string

// UtxoRef is a UTxO reference in the format "txid#index".
type UtxoRef = string

// BytesEnvelope wraps binary data with its encoding metadata for JSON transport.
type BytesEnvelope struct {
	Content     string `json:"content"`
	ContentType string `json:"contentType"`
}

// NewHexEnvelope creates a BytesEnvelope with hex content type.
func NewHexEnvelope(hexContent string) BytesEnvelope {
	return BytesEnvelope{
		Content:     hexContent,
		ContentType: "hex",
	}
}

// TirEnvelope wraps a Transaction Intermediate Representation with encoding and version metadata.
type TirEnvelope struct {
	Content  string `json:"content"`
	Encoding string `json:"encoding"`
	Version  string `json:"version"`
}
