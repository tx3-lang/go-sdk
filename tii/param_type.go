package tii

import "strings"

// ParamType represents the type of a transaction parameter.
type ParamType int

const (
	ParamTypeBytes   ParamType = iota // Hex-encoded byte array
	ParamTypeInteger                  // Signed/unsigned integer
	ParamTypeBoolean                  // Boolean
	ParamTypeUtxoRef                  // UTxO reference "txid#index"
	ParamTypeAddress                  // Bech32 address
	ParamTypeList                     // List of another type
	ParamTypeCustom                   // Custom JSON schema
)

// ParamInfo holds a parameter's name and type.
type ParamInfo struct {
	Name string
	Type ParamType
}

// tx3CoreTypePrefix is the base URI for Tx3 core type references.
const tx3CoreTypePrefix = "https://tx3.land/specs/v1beta0/core#"

// ParamTypeFromSchema derives a ParamType from a JSON schema definition.
func ParamTypeFromSchema(schema Schema) (ParamType, error) {
	// Check for $ref to Tx3 core types
	if schema.Ref != "" {
		return paramTypeFromRef(schema.Ref)
	}

	switch schema.Type {
	case "integer":
		return ParamTypeInteger, nil
	case "boolean":
		return ParamTypeBoolean, nil
	case "string":
		return ParamTypeAddress, nil // default string type maps to address
	case "array":
		return ParamTypeList, nil
	case "object":
		return ParamTypeCustom, nil
	default:
		if schema.Type != "" {
			return 0, &InvalidParamTypeError{Detail: "unknown type: " + schema.Type}
		}
		return ParamTypeCustom, nil
	}
}

func paramTypeFromRef(ref string) (ParamType, error) {
	if !strings.HasPrefix(ref, tx3CoreTypePrefix) {
		return ParamTypeCustom, nil
	}

	typeName := strings.TrimPrefix(ref, tx3CoreTypePrefix)
	switch typeName {
	case "Bytes":
		return ParamTypeBytes, nil
	case "Address":
		return ParamTypeAddress, nil
	case "UtxoRef":
		return ParamTypeUtxoRef, nil
	default:
		return 0, &InvalidParamTypeError{Detail: "unknown core type ref: " + typeName}
	}
}
