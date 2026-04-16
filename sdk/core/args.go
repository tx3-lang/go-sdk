package core

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
)

// ArgValueType identifies the kind of value an ArgValue holds.
type ArgValueType int

const (
	ArgTypeInt     ArgValueType = iota // Integer value (arbitrary precision)
	ArgTypeBool                        // Boolean value
	ArgTypeString                      // String value
	ArgTypeBytes                       // Raw byte array
	ArgTypeAddress                     // Blockchain address (as bytes)
	ArgTypeUtxoRef                     // UTxO reference
)

// ArgValue represents a typed argument value for TRP marshalling.
type ArgValue struct {
	Type       ArgValueType
	IntVal     *big.Int
	BoolVal    bool
	StringVal  string
	BytesVal   []byte
	UtxoRefVal string
}

// IntArg creates an integer ArgValue.
func IntArg(v int64) ArgValue {
	return ArgValue{Type: ArgTypeInt, IntVal: big.NewInt(v)}
}

// BigIntArg creates an integer ArgValue from a big.Int.
func BigIntArg(v *big.Int) ArgValue {
	return ArgValue{Type: ArgTypeInt, IntVal: new(big.Int).Set(v)}
}

// BoolArg creates a boolean ArgValue.
func BoolArg(v bool) ArgValue {
	return ArgValue{Type: ArgTypeBool, BoolVal: v}
}

// StringArg creates a string ArgValue.
func StringArg(v string) ArgValue {
	return ArgValue{Type: ArgTypeString, StringVal: v}
}

// BytesArg creates a byte-array ArgValue.
func BytesArg(v []byte) ArgValue {
	return ArgValue{Type: ArgTypeBytes, BytesVal: v}
}

// AddressArg creates an address ArgValue from a bech32 address string.
func AddressArg(v string) ArgValue {
	return ArgValue{Type: ArgTypeAddress, StringVal: v}
}

// UtxoRefArg creates a UTxO reference ArgValue (format: "txid#index").
func UtxoRefArg(v string) ArgValue {
	return ArgValue{Type: ArgTypeUtxoRef, UtxoRefVal: v}
}

// ToJSON converts an ArgValue to its TRP wire-format representation.
func (a ArgValue) ToJSON() interface{} {
	switch a.Type {
	case ArgTypeInt:
		if a.IntVal.IsInt64() {
			v := a.IntVal.Int64()
			if v >= -(1<<53) && v <= (1<<53) {
				return v
			}
		}
		// Large integers: encode as hex string
		return "0x" + hex.EncodeToString(a.IntVal.Bytes())
	case ArgTypeBool:
		return a.BoolVal
	case ArgTypeString:
		return a.StringVal
	case ArgTypeBytes:
		return "0x" + hex.EncodeToString(a.BytesVal)
	case ArgTypeAddress:
		return a.StringVal
	case ArgTypeUtxoRef:
		return a.UtxoRefVal
	default:
		return nil
	}
}

// CoerceArg converts a native Go value to a JSON-compatible wire format value
// suitable for inclusion in an ArgMap. Supported input types:
// int, int64, float64, string, bool, []byte, ArgValue, *big.Int.
func CoerceArg(v interface{}) (interface{}, error) {
	switch val := v.(type) {
	case ArgValue:
		return val.ToJSON(), nil
	case int:
		return int64(val), nil
	case int64:
		return val, nil
	case float64:
		return val, nil
	case string:
		return val, nil
	case bool:
		return val, nil
	case []byte:
		return "0x" + hex.EncodeToString(val), nil
	case *big.Int:
		return BigIntArg(val).ToJSON(), nil
	default:
		return nil, fmt.Errorf("unsupported arg type: %T", v)
	}
}

// NormalizeArgKey lowercases an argument key for case-insensitive matching.
func NormalizeArgKey(key string) string {
	return strings.ToLower(key)
}
