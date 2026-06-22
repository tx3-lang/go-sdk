package tii

import (
	"encoding/json"
	"testing"
)

// parse decodes a JSON schema literal into a Schema (exercising the custom
// UnmarshalJSON, incl. the `items: false` / `additionalProperties: false` forms).
func parse(t *testing.T, raw string) Schema {
	t.Helper()
	var s Schema
	if err := json.Unmarshal([]byte(raw), &s); err != nil {
		t.Fatalf("unmarshal schema %q: %v", raw, err)
	}
	return s
}

func kindOf(t *testing.T, raw string) ParamType {
	return ParamTypeFromSchema(parse(t, raw), map[string]Schema{})
}

func TestParamPrimitivesAndUnit(t *testing.T) {
	cases := map[string]ParamKind{
		`{"type":"integer"}`: KindInteger,
		`{"type":"boolean"}`: KindBoolean,
		`{"type":"null"}`:    KindUnit,
	}
	for raw, want := range cases {
		if got := kindOf(t, raw).Kind; got != want {
			t.Errorf("%s: got kind %d, want %d", raw, got, want)
		}
	}
}

func TestParamCoreRefsBothForms(t *testing.T) {
	forms := []func(string) string{
		func(n string) string { return "https://tx3.land/specs/v1beta0/tii#/$defs/" + n },
		func(n string) string { return "https://tx3.land/specs/v1beta0/core#" + n },
	}
	want := map[string]ParamKind{
		"Bytes":    KindBytes,
		"Address":  KindAddress,
		"UtxoRef":  KindUtxoRef,
		"Utxo":     KindUtxo,
		"AnyAsset": KindAnyAsset,
	}
	for _, form := range forms {
		for name, wantKind := range want {
			schema := Schema{Ref: form(name)}
			if got := ParamTypeFromSchema(schema, nil).Kind; got != wantKind {
				t.Errorf("$ref %s: got kind %d, want %d", form(name), got, wantKind)
			}
		}
	}
}

func TestParamListNested(t *testing.T) {
	pt := kindOf(t, `{"type":"array","items":{"type":"array","items":{"type":"boolean"}}}`)
	if pt.Kind != KindList || pt.Inner == nil {
		t.Fatalf("expected list, got %+v", pt)
	}
	if pt.Inner.Kind != KindList || pt.Inner.Inner == nil || pt.Inner.Inner.Kind != KindBoolean {
		t.Fatalf("expected list(list(bool)), got %+v", pt)
	}
}

func TestParamTupleWithItemsFalse(t *testing.T) {
	pt := kindOf(t, `{"type":"array","prefixItems":[{"type":"integer"},{"$ref":"https://tx3.land/specs/v1beta0/tii#/$defs/Bytes"}],"items":false}`)
	if pt.Kind != KindTuple {
		t.Fatalf("expected tuple, got %+v", pt)
	}
	if len(pt.Elements) != 2 || pt.Elements[0].Kind != KindInteger || pt.Elements[1].Kind != KindBytes {
		t.Fatalf("unexpected tuple elements: %+v", pt.Elements)
	}
}

func TestParamMap(t *testing.T) {
	pt := kindOf(t, `{"type":"object","additionalProperties":{"type":"integer"}}`)
	if pt.Kind != KindMap || pt.Inner == nil || pt.Inner.Kind != KindInteger {
		t.Fatalf("expected map(int), got %+v", pt)
	}
}

func TestParamRecord(t *testing.T) {
	pt := kindOf(t, `{"type":"object","properties":{"price":{"type":"integer"},"live":{"type":"boolean"}},"required":["price","live"]}`)
	if pt.Kind != KindRecord {
		t.Fatalf("expected record, got %+v", pt)
	}
	if pt.Field("price").Kind != KindInteger || pt.Field("live").Kind != KindBoolean {
		t.Fatalf("unexpected record fields: %+v", pt.Fields)
	}
}

func TestParamVariant(t *testing.T) {
	pt := kindOf(t, `{"oneOf":[
		{"type":"object","additionalProperties":false,"required":["Buy"],"properties":{"Buy":{"type":"object","properties":{},"required":[]}}},
		{"type":"object","additionalProperties":false,"required":["Sell"],"properties":{"Sell":{"type":"object","properties":{"price":{"type":"integer"}},"required":["price"]}}}
	]}`)
	if pt.Kind != KindVariant || len(pt.Cases) != 2 {
		t.Fatalf("expected 2-case variant, got %+v", pt)
	}
	if pt.Cases[0].Tag != "Buy" || pt.Cases[1].Tag != "Sell" {
		t.Fatalf("unexpected variant tags: %q, %q", pt.Cases[0].Tag, pt.Cases[1].Tag)
	}
	if pt.Cases[1].Fields.Kind != KindRecord || pt.Cases[1].Fields.Field("price").Kind != KindInteger {
		t.Fatalf("unexpected Sell fields: %+v", pt.Cases[1].Fields)
	}
}

func TestParamComponentRefResolves(t *testing.T) {
	components := map[string]Schema{
		"AssetClass": parse(t, `{"type":"object","properties":{"policy":{"$ref":"https://tx3.land/specs/v1beta0/tii#/$defs/Bytes"}},"required":["policy"]}`),
	}
	pt := ParamTypeFromSchema(Schema{Ref: "#/components/schemas/AssetClass"}, components)
	if pt.Kind != KindRecord || pt.Field("policy").Kind != KindBytes {
		t.Fatalf("expected record(policy:bytes), got %+v", pt)
	}
	// Missing component → Unknown, never errors.
	miss := ParamTypeFromSchema(Schema{Ref: "#/components/schemas/Nope"}, components)
	if miss.Kind != KindUnknown {
		t.Fatalf("expected unknown for missing component, got %+v", miss)
	}
}

func TestParamUnknownNeverFails(t *testing.T) {
	for _, raw := range []string{
		`{"type":"string"}`,
		`{}`,
		`{"type":"array"}`,
		`{"$ref":"https://example.com/Weird"}`,
	} {
		if got := kindOf(t, raw).Kind; got != KindUnknown {
			t.Errorf("%s: expected unknown, got kind %d", raw, got)
		}
	}
}
