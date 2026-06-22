package tii

import (
	"fmt"
	"sort"
)

// Type-directed encoding of argument values into the TRP `TaggedArg` wire form.
//
// A TRP resolve request carries an untyped TIR, so the resolver cannot recover
// the structure of an aggregate argument (a record, list, tuple, or map) on its
// own. The full type lives in the .tii, a client-side artifact — so the SDK is
// authoritative: it walks the resolved ParamType alongside the user value and
// emits the deterministic, self-describing TaggedArg (single-key tagged,
// recursive — see the TaggedArg schema in core/trp/v1beta0/trp.json and the SDK
// spec's api-surface/args.md). The resolver then decodes it structurally,
// without a schema.
//
// This is a single recursive walk over (type, value); scalars are just the leaf
// cases. A scalar leaf at the top level renders bare (the resolver coerces it via
// the flat TIR type); the same scalar nested inside an aggregate renders tagged,
// where the resolver has no element/field type to coerce against.

// EncodeError reports an argument value whose shape does not match its declared
// ParamType. It is surfaced before the request is sent (the SDK is authoritative
// for complex types), so a malformed complex arg fails fast at the client rather
// than as an opaque resolver error.
type EncodeError struct {
	msg string
}

func (e *EncodeError) Error() string { return e.msg }
func (e *EncodeError) isTiiError()   {}

func wrongShape(kind, expected string, got interface{}) error {
	return &EncodeError{msg: fmt.Sprintf("expected %s for a `%s` argument, got `%s`", expected, kind, shapeOf(got))}
}

// shapeOf names the JSON shape of a decoded value, for EncodeError messages.
func shapeOf(value interface{}) string {
	switch value.(type) {
	case nil:
		return "null"
	case bool:
		return "bool"
	case float64, int, int64:
		return "number"
	case string:
		return "string"
	case []interface{}:
		return "array"
	case map[string]interface{}:
		return "object"
	default:
		return "unknown"
	}
}

// Encode marshals an argument value to its TRP wire form, directed by param.
//
// One recursive walk over (type, value). A scalar leaf renders bare at the top
// level — the resolver coerces it via the param's flat type — and tagged when it
// sits inside an aggregate, where the resolver has no element type. Aggregates
// always render to their tagged structural form. Returns an error if value's
// shape cannot match param.
func Encode(param ParamType, value interface{}) (interface{}, error) {
	return marshal(param, value, false)
}

// marshal is the single recursive walk. nested is true when value sits inside an
// aggregate, where scalar leaves must be tagged for the schema-less resolver.
func marshal(param ParamType, value interface{}, nested bool) (interface{}, error) {
	switch param.Kind {
	// Scalar leaves: bare at the top level, tagged when nested. Shape checks
	// here are the "reject before sending" pass; the resolver still performs the
	// authoritative coercion.
	case KindInteger:
		switch value.(type) {
		case float64, int, int64, string:
			return leaf("int", value, nested), nil
		default:
			return nil, wrongShape("integer", "number or decimal/hex string", value)
		}
	case KindBoolean:
		// Accept the same lenient forms the resolver coerces (bool, 0/1,
		// "true"/"false").
		switch value.(type) {
		case bool, float64, int, int64, string:
			return leaf("bool", value, nested), nil
		default:
			return nil, wrongShape("boolean", "bool", value)
		}
	case KindBytes:
		// Hex string or a bytes-envelope object.
		switch value.(type) {
		case string, map[string]interface{}:
			return leaf("bytes", value, nested), nil
		default:
			return nil, wrongShape("bytes", "hex string or bytes envelope", value)
		}
	case KindAddress:
		switch value.(type) {
		case string:
			return leaf("address", value, nested), nil
		default:
			return nil, wrongShape("address", "bech32 or hex string", value)
		}
	case KindUtxoRef:
		switch value.(type) {
		case string:
			return leaf("utxoRef", value, nested), nil
		default:
			return nil, wrongShape("utxoRef", "txid#index string", value)
		}

	// A unit field has no payload; it lowers to a nullary struct.
	case KindUnit:
		return structValue(0, []interface{}{}), nil

	case KindList:
		items, ok := value.([]interface{})
		if !ok {
			return nil, wrongShape("list", "array", value)
		}
		encoded := make([]interface{}, len(items))
		for i, v := range items {
			e, err := marshal(*param.Inner, v, true)
			if err != nil {
				return nil, err
			}
			encoded[i] = e
		}
		return map[string]interface{}{"list": encoded}, nil

	case KindTuple:
		items, ok := value.([]interface{})
		if !ok {
			return nil, wrongShape("tuple", "array", value)
		}
		if len(items) != len(param.Elements) {
			return nil, &EncodeError{msg: fmt.Sprintf("tuple arity mismatch: expected %d element(s), got %d", len(param.Elements), len(items))}
		}
		encoded := make([]interface{}, len(items))
		for i, v := range items {
			e, err := marshal(param.Elements[i], v, true)
			if err != nil {
				return nil, err
			}
			encoded[i] = e
		}
		return map[string]interface{}{"tuple": encoded}, nil

	case KindMap:
		obj, ok := value.(map[string]interface{})
		if !ok {
			return nil, wrongShape("map", "object", value)
		}
		// The .tii erases the Tx3 key type (JSON object keys are strings), so
		// keys are carried as `string` leaves. Sort by key for a deterministic,
		// language-neutral pair order.
		keys := make([]string, 0, len(obj))
		for k := range obj {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		pairs := make([]interface{}, 0, len(keys))
		for _, k := range keys {
			v, err := marshal(*param.Inner, obj[k], true)
			if err != nil {
				return nil, err
			}
			pairs = append(pairs, []interface{}{map[string]interface{}{"string": k}, v})
		}
		return map[string]interface{}{"map": pairs}, nil

	// A record is constructor 0; a variant resolves its case index. Both emit
	// the same positional `struct` form.
	case KindRecord:
		fields, err := encodeRecordFields(param.Fields, value)
		if err != nil {
			return nil, err
		}
		return structValue(0, fields), nil

	case KindVariant:
		return encodeVariant(param.Cases, value)

	// No wire-leaf form and no element types to drive encoding: pass the value
	// through and let the resolver coerce it via the flat type.
	default: // KindUtxo, KindAnyAsset, KindUnknown
		return value, nil
	}
}

// leaf renders a scalar leaf: bare at the top level (the resolver knows the
// param's flat type), tagged when nested inside an aggregate (it doesn't).
func leaf(tag string, value interface{}, nested bool) interface{} {
	if nested {
		return map[string]interface{}{tag: value}
	}
	return value
}

// structValue builds a `{ "struct": { "constructor": c, "fields": [...] } }`
// TaggedArg node.
func structValue(constructor int, fields []interface{}) map[string]interface{} {
	return map[string]interface{}{
		"struct": map[string]interface{}{
			"constructor": constructor,
			"fields":      fields,
		},
	}
}

// encodeRecordFields encodes a record's fields positionally in declared order,
// mapping the user's by-name object. Rejects missing or extra fields up front.
func encodeRecordFields(fields []RecordField, value interface{}) ([]interface{}, error) {
	obj, ok := value.(map[string]interface{})
	if !ok {
		return nil, wrongShape("record", "object", value)
	}

	// Reject any field the record does not declare.
	for key := range obj {
		found := false
		for i := range fields {
			if fields[i].Name == key {
				found = true
				break
			}
		}
		if !found {
			return nil, &EncodeError{msg: fmt.Sprintf("unknown record field `%s`", key)}
		}
	}

	encoded := make([]interface{}, len(fields))
	for i := range fields {
		fieldValue, present := obj[fields[i].Name]
		if !present {
			return nil, &EncodeError{msg: fmt.Sprintf("missing record field `%s`", fields[i].Name)}
		}
		e, err := marshal(fields[i].Type, fieldValue, true)
		if err != nil {
			return nil, err
		}
		encoded[i] = e
	}
	return encoded, nil
}

// encodeVariant encodes an externally-tagged variant value
// `{ "<Case>": <payload> }` into a struct whose constructor is the case index
// from the .tii oneOf order.
func encodeVariant(cases []VariantCase, value interface{}) (interface{}, error) {
	obj, ok := value.(map[string]interface{})
	if !ok || len(obj) != 1 {
		return nil, &EncodeError{msg: "variant value must be a single-key object naming the case"}
	}

	var tag string
	var payload interface{}
	for k, v := range obj {
		tag, payload = k, v
	}

	index := -1
	for i := range cases {
		if cases[i].Tag == tag {
			index = i
			break
		}
	}
	if index < 0 {
		return nil, &EncodeError{msg: fmt.Sprintf("unknown variant case `%s`", tag)}
	}

	// A case payload is a record (possibly empty). Encode its fields positionally
	// and stamp the case index as the constructor.
	caseType := cases[index].Fields
	var fields []interface{}
	if caseType.Kind == KindRecord {
		var err error
		fields, err = encodeRecordFields(caseType.Fields, payload)
		if err != nil {
			return nil, err
		}
	} else {
		// Defensive: a non-record payload encodes as the single field.
		e, err := marshal(caseType, payload, true)
		if err != nil {
			return nil, err
		}
		fields = []interface{}{e}
	}

	return structValue(index, fields), nil
}
