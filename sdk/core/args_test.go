package core_test

import (
	"math/big"
	"testing"

	"github.com/tx3-lang/go-sdk/sdk/core"
)

func TestIntArgSmall(t *testing.T) {
	v := core.IntArg(42)
	result := v.ToJSON()
	if result != int64(42) {
		t.Errorf("expected 42, got %v (%T)", result, result)
	}
}

func TestIntArgLarge(t *testing.T) {
	// Value larger than 2^53
	big := new(big.Int)
	big.SetString("99999999999999999999", 10)
	v := core.BigIntArg(big)
	result := v.ToJSON()
	s, ok := result.(string)
	if !ok {
		t.Fatalf("expected string for large int, got %T", result)
	}
	if s[:2] != "0x" {
		t.Errorf("expected hex prefix, got %s", s)
	}
}

func TestBoolArg(t *testing.T) {
	v := core.BoolArg(true)
	if v.ToJSON() != true {
		t.Error("expected true")
	}
}

func TestStringArg(t *testing.T) {
	v := core.StringArg("hello")
	if v.ToJSON() != "hello" {
		t.Error("expected 'hello'")
	}
}

func TestBytesArg(t *testing.T) {
	v := core.BytesArg([]byte{0xDE, 0xAD})
	result := v.ToJSON()
	if result != "0xdead" {
		t.Errorf("expected '0xdead', got %v", result)
	}
}

func TestAddressArg(t *testing.T) {
	v := core.AddressArg("addr_test1abc")
	if v.ToJSON() != "addr_test1abc" {
		t.Error("expected address pass-through")
	}
}

func TestUtxoRefArg(t *testing.T) {
	v := core.UtxoRefArg("abc123#0")
	if v.ToJSON() != "abc123#0" {
		t.Error("expected utxo ref pass-through")
	}
}

func TestCoerceArgTypes(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected interface{}
	}{
		{"int", 42, int64(42)},
		{"int64", int64(100), int64(100)},
		{"float64", 3.14, 3.14},
		{"string", "hello", "hello"},
		{"bool", true, true},
		{"bytes", []byte{0xAB}, "0xab"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := core.CoerceArg(tt.input)
			if err != nil {
				t.Fatalf("CoerceArg failed: %v", err)
			}
			if result != tt.expected {
				t.Errorf("expected %v (%T), got %v (%T)", tt.expected, tt.expected, result, result)
			}
		})
	}
}

func TestCoerceArgUnsupportedType(t *testing.T) {
	_, err := core.CoerceArg(struct{}{})
	if err == nil {
		t.Error("expected error for unsupported type")
	}
}

func TestNormalizeArgKey(t *testing.T) {
	if core.NormalizeArgKey("Quantity") != "quantity" {
		t.Error("expected lowercase")
	}
	if core.NormalizeArgKey("SENDER") != "sender" {
		t.Error("expected lowercase")
	}
}
