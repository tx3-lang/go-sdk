package tii

import (
	"encoding/json"
	"os"
	"testing"
)

// wireVectors mirrors the shared oracle at
// sdk-spec/test-vectors/complex-types/wire-vectors.json (copied into testdata).
type wireVectors struct {
	Components map[string]Schema `json:"components"`
	Accept     []acceptVector    `json:"accept"`
	Reject     []rejectVector    `json:"reject"`
}

type acceptVector struct {
	Name   string          `json:"name"`
	Schema Schema          `json:"schema"`
	Value  json.RawMessage `json:"value"`
	Tagged json.RawMessage `json:"tagged"`
}

type rejectVector struct {
	Name   string          `json:"name"`
	Schema Schema          `json:"schema"`
	Value  json.RawMessage `json:"value"`
	Reason string          `json:"reason"`
}

func loadVectors(t *testing.T) wireVectors {
	t.Helper()
	data, err := os.ReadFile("../testdata/wire-vectors.json")
	if err != nil {
		t.Fatalf("read wire-vectors.json: %v", err)
	}
	var v wireVectors
	if err := json.Unmarshal(data, &v); err != nil {
		t.Fatalf("parse wire-vectors.json: %v", err)
	}
	return v
}

// decodeJSON unmarshals a raw JSON message into a generic interface{} tree, so
// values can be compared structurally regardless of map ordering.
func decodeJSON(t *testing.T, raw json.RawMessage) interface{} {
	t.Helper()
	var v interface{}
	if err := json.Unmarshal(raw, &v); err != nil {
		t.Fatalf("decode value %s: %v", raw, err)
	}
	return v
}

// normalize round-trips a value through JSON so that the encoder output (with
// Go-typed ints and map[string]interface{}) compares equal to the oracle's
// decoded JSON (float64 numbers). Avoids map-ordering pitfalls.
func normalize(t *testing.T, v interface{}) interface{} {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal %#v: %v", v, err)
	}
	var out interface{}
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("unmarshal %s: %v", data, err)
	}
	return out
}

func TestEncodeAcceptVectors(t *testing.T) {
	vectors := loadVectors(t)
	for _, vec := range vectors.Accept {
		t.Run(vec.Name, func(t *testing.T) {
			param := ParamTypeFromSchema(vec.Schema, vectors.Components)
			got, err := Encode(param, decodeJSON(t, vec.Value))
			if err != nil {
				t.Fatalf("encode failed: %v", err)
			}
			want := decodeJSON(t, vec.Tagged)
			if !jsonEqual(t, got, want) {
				gotJSON, _ := json.Marshal(got)
				t.Errorf("wire mismatch:\n got: %s\nwant: %s", gotJSON, vec.Tagged)
			}
		})
	}
}

func TestEncodeRejectVectors(t *testing.T) {
	vectors := loadVectors(t)
	for _, vec := range vectors.Reject {
		t.Run(vec.Name, func(t *testing.T) {
			param := ParamTypeFromSchema(vec.Schema, vectors.Components)
			_, err := Encode(param, decodeJSON(t, vec.Value))
			if err == nil {
				t.Fatalf("expected error (%s), got nil", vec.Reason)
			}
		})
	}
}

// jsonEqual compares two decoded JSON trees by re-marshalling the encoder output
// to the same shape as the oracle (float64 numbers, sorted map keys).
func jsonEqual(t *testing.T, got, want interface{}) bool {
	gotData, err := json.Marshal(normalize(t, got))
	if err != nil {
		return false
	}
	wantData, err := json.Marshal(normalize(t, want))
	if err != nil {
		return false
	}
	return string(gotData) == string(wantData)
}

// TestRecordFieldOrderFollowsRequired locks the subtle invariant: struct field
// order follows the schema's `required` array, not alphabetized `properties`.
// Meta { tags: List<Int>, level: Int } has required = [tags, level], while
// properties alphabetizes to [level, tags]; the fields MUST be [list, int].
func TestRecordFieldOrderFollowsRequired(t *testing.T) {
	schema := parse(t, `{
		"type": "object",
		"properties": {
			"level": { "type": "integer" },
			"tags": { "type": "array", "items": { "type": "integer" } }
		},
		"required": ["tags", "level"]
	}`)
	param := ParamTypeFromSchema(schema, nil)

	value := map[string]interface{}{"level": float64(7), "tags": []interface{}{float64(1), float64(2), float64(3)}}
	got, err := Encode(param, value)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	want := map[string]interface{}{
		"struct": map[string]interface{}{
			"constructor": 0,
			"fields": []interface{}{
				map[string]interface{}{"list": []interface{}{
					map[string]interface{}{"int": float64(1)},
					map[string]interface{}{"int": float64(2)},
					map[string]interface{}{"int": float64(3)},
				}},
				map[string]interface{}{"int": float64(7)},
			},
		},
	}
	if !jsonEqual(t, got, want) {
		gotJSON, _ := json.Marshal(got)
		wantJSON, _ := json.Marshal(want)
		t.Errorf("field order mismatch:\n got: %s\nwant: %s", gotJSON, wantJSON)
	}
}

// TestTopLevelScalarRendersBare verifies the position-dependent leaf rendering:
// a scalar at the top level is sent bare for the resolver to coerce via the flat
// type.
func TestTopLevelScalarRendersBare(t *testing.T) {
	intParam := ParamTypeFromSchema(parse(t, `{"type":"integer"}`), nil)
	got, err := Encode(intParam, float64(5))
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}
	if !jsonEqual(t, got, float64(5)) {
		t.Errorf("top-level int: got %#v, want bare 5", got)
	}

	bytesParam := ParamTypeFromSchema(parse(t, `{"$ref":"https://tx3.land/specs/v1beta0/tii#/$defs/Bytes"}`), nil)
	got, err = Encode(bytesParam, "cafe")
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}
	if !jsonEqual(t, got, "cafe") {
		t.Errorf("top-level bytes: got %#v, want bare \"cafe\"", got)
	}
}

// TestNestedScalarIsTagged verifies that the same scalar, nested inside an
// aggregate, is tagged because the resolver has no element type to coerce
// against.
func TestNestedScalarIsTagged(t *testing.T) {
	listParam := ParamTypeFromSchema(parse(t, `{"type":"array","items":{"type":"integer"}}`), nil)
	got, err := Encode(listParam, []interface{}{float64(5)})
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}
	want := map[string]interface{}{"list": []interface{}{
		map[string]interface{}{"int": float64(5)},
	}}
	if !jsonEqual(t, got, want) {
		gotJSON, _ := json.Marshal(got)
		t.Errorf("nested int: got %s, want tagged {\"list\":[{\"int\":5}]}", gotJSON)
	}
}

const bytesSchema = `{"$ref":"https://tx3.land/specs/v1beta0/tii#/$defs/Bytes"}`
const listOfBytesSchema = `{"type":"array","items":{"$ref":"https://tx3.land/specs/v1beta0/tii#/$defs/Bytes"}}`

// TestNativeByteArraysCanonicalizeToHex verifies that a native byte array
// (`[]byte`, or an integer array — the JSON shape other SDKs' native byte
// arrays serialize to) canonicalizes to 0x-prefixed hex, per SDK spec §3.9
// (regression: TRP `(-32005) value is not bytes: [1,1]`).
func TestNativeByteArraysCanonicalizeToHex(t *testing.T) {
	bytesParam := ParamTypeFromSchema(parse(t, bytesSchema), nil)

	got, err := Encode(bytesParam, []byte{1, 1})
	if err != nil {
		t.Fatalf("encode []byte failed: %v", err)
	}
	if !jsonEqual(t, got, "0x0101") {
		t.Errorf("[]byte: got %#v, want \"0x0101\"", got)
	}

	got, err = Encode(bytesParam, []interface{}{float64(1), float64(1)})
	if err != nil {
		t.Fatalf("encode integer array failed: %v", err)
	}
	if !jsonEqual(t, got, "0x0101") {
		t.Errorf("integer array: got %#v, want \"0x0101\"", got)
	}

	listParam := ParamTypeFromSchema(parse(t, listOfBytesSchema), nil)
	got, err = Encode(listParam, []interface{}{[]byte{1, 2}})
	if err != nil {
		t.Fatalf("encode list of []byte failed: %v", err)
	}
	want := map[string]interface{}{"list": []interface{}{
		map[string]interface{}{"bytes": "0x0102"},
	}}
	if !jsonEqual(t, got, want) {
		gotJSON, _ := json.Marshal(got)
		t.Errorf("nested []byte: got %s, want {\"list\":[{\"bytes\":\"0x0102\"}]}", gotJSON)
	}
}

// TestRejectsNonByteArraysForBytes pins the reject pass for byte params.
func TestRejectsNonByteArraysForBytes(t *testing.T) {
	bytesParam := ParamTypeFromSchema(parse(t, bytesSchema), nil)
	for _, bad := range []interface{}{
		[]interface{}{float64(1), float64(256)},
		[]interface{}{float64(1), float64(-1)},
		[]interface{}{"aa", float64(1)},
		[]interface{}{true},
		true,
	} {
		if _, err := Encode(bytesParam, bad); err == nil {
			t.Errorf("value %#v should have been rejected", bad)
		}
	}
}

// TestHydraInitArgShapes covers the Hydra `init` argument shapes:
// `participants` / `parties` are `List<Bytes>`, `head_id` is `Bytes`
// (regression: `(-32005) target type not supported: List` /
// `value is not bytes: [1,2]`).
func TestHydraInitArgShapes(t *testing.T) {
	listParam := ParamTypeFromSchema(parse(t, listOfBytesSchema), nil)

	got, err := Encode(listParam, []interface{}{"0102", "0304"})
	if err != nil {
		t.Fatalf("encode hex strings failed: %v", err)
	}
	want := map[string]interface{}{"list": []interface{}{
		map[string]interface{}{"bytes": "0102"},
		map[string]interface{}{"bytes": "0304"},
	}}
	if !jsonEqual(t, got, want) {
		gotJSON, _ := json.Marshal(got)
		t.Errorf("participants (hex): got %s, want tagged list of bytes", gotJSON)
	}

	got, err = Encode(listParam, []interface{}{[]byte{1, 2}})
	if err != nil {
		t.Fatalf("encode native byte arrays failed: %v", err)
	}
	want = map[string]interface{}{"list": []interface{}{
		map[string]interface{}{"bytes": "0x0102"},
	}}
	if !jsonEqual(t, got, want) {
		gotJSON, _ := json.Marshal(got)
		t.Errorf("participants (native): got %s, want tagged list of bytes", gotJSON)
	}

	bytesParam := ParamTypeFromSchema(parse(t, bytesSchema), nil)
	got, err = Encode(bytesParam, "abcd0123")
	if err != nil {
		t.Fatalf("encode head_id failed: %v", err)
	}
	if !jsonEqual(t, got, "abcd0123") {
		t.Errorf("head_id: got %#v, want bare \"abcd0123\"", got)
	}
}

// TestAsteriaNameArgShapes covers the Asteria `create_ship` `ship_name` /
// `pilot_name` `Bytes` params (regression: `(-32005) value is not bytes: [1,1]`).
func TestAsteriaNameArgShapes(t *testing.T) {
	bytesParam := ParamTypeFromSchema(parse(t, bytesSchema), nil)

	got, err := Encode(bytesParam, "53484950313233")
	if err != nil {
		t.Fatalf("encode hex name failed: %v", err)
	}
	if !jsonEqual(t, got, "53484950313233") {
		t.Errorf("hex name: got %#v, want bare passthrough", got)
	}

	got, err = Encode(bytesParam, []byte("SHIP"))
	if err != nil {
		t.Fatalf("encode native name failed: %v", err)
	}
	if !jsonEqual(t, got, "0x53484950") {
		t.Errorf("native name: got %#v, want \"0x53484950\"", got)
	}
}
