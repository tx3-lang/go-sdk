package tii

import "strings"

// ParamKind identifies the category of a transaction parameter type.
type ParamKind int

const (
	KindBytes    ParamKind = iota // Hex-encoded byte array
	KindInteger                   // Signed/unsigned integer
	KindBoolean                   // Boolean
	KindUnit                      // Unit ({"type":"null"})
	KindUtxoRef                   // UTxO reference "txid#index"
	KindAddress                   // Bech32 address
	KindUtxo                      // Resolved UTxO object
	KindAnyAsset                  // Asset identified by policy + name
	KindList                      // Homogeneous sequence (array + items)
	KindTuple                     // Positional sequence (array + prefixItems)
	KindMap                       // String-keyed map (object + additionalProperties)
	KindRecord                    // User record (object + properties)
	KindVariant                   // User tagged union (oneOf)
	KindUnknown                   // Unrecognized schema; carries the raw schema
)

// ParamType describes a transaction parameter's type. Compound kinds carry their
// element/field types: List/Map in Inner, Tuple in Elements, Record in Fields,
// Variant in Cases. Unknown carries the raw Schema.
type ParamType struct {
	Kind     ParamKind
	Inner    *ParamType           // List element / Map value
	Elements []ParamType          // Tuple positional types
	Fields   map[string]ParamType // Record field name → type
	Cases    []VariantCase        // Variant cases
	Schema   *Schema              // Unknown fallback (raw schema)
}

// VariantCase is one case of a KindVariant param.
type VariantCase struct {
	Tag    string
	Fields ParamType
}

// ParamInfo pairs a parameter name with its type.
type ParamInfo struct {
	Name string
	Type ParamType
}

// ParamTypeFromSchema derives a ParamType from a JSON schema node. It never
// fails: any shape it does not recognize — a bare string, an unresolved object,
// an unknown $ref — becomes KindUnknown carrying the raw schema. components is
// the TII's components.schemas table, used to resolve "#/components/schemas/<Name>"
// references to user-defined record / variant types.
func ParamTypeFromSchema(schema Schema, components map[string]Schema) ParamType {
	if schema.Ref != "" {
		if name, ok := strings.CutPrefix(schema.Ref, "#/components/schemas/"); ok {
			if resolved, found := components[name]; found {
				return ParamTypeFromSchema(resolved, components)
			}
			return unknown(schema)
		}
		if kind, ok := coreKindFromRef(schema.Ref); ok {
			return ParamType{Kind: kind}
		}
		return unknown(schema)
	}

	if len(schema.OneOf) > 0 {
		cases := make([]VariantCase, 0, len(schema.OneOf))
		for _, c := range schema.OneOf {
			cases = append(cases, variantCase(c, components))
		}
		return ParamType{Kind: KindVariant, Cases: cases}
	}

	switch schema.Type {
	case "integer":
		return ParamType{Kind: KindInteger}
	case "boolean":
		return ParamType{Kind: KindBoolean}
	case "null":
		return ParamType{Kind: KindUnit}
	case "array":
		if len(schema.PrefixItems) > 0 {
			elements := make([]ParamType, 0, len(schema.PrefixItems))
			for _, el := range schema.PrefixItems {
				elements = append(elements, ParamTypeFromSchema(el, components))
			}
			return ParamType{Kind: KindTuple, Elements: elements}
		}
		if schema.Items != nil {
			inner := ParamTypeFromSchema(*schema.Items, components)
			return ParamType{Kind: KindList, Inner: &inner}
		}
		return unknown(schema)
	case "object":
		if schema.AdditionalProperties != nil {
			value := ParamTypeFromSchema(*schema.AdditionalProperties, components)
			return ParamType{Kind: KindMap, Inner: &value}
		}
		if len(schema.Properties) > 0 {
			fields := make(map[string]ParamType, len(schema.Properties))
			for k, v := range schema.Properties {
				fields[k] = ParamTypeFromSchema(v, components)
			}
			return ParamType{Kind: KindRecord, Fields: fields}
		}
		return unknown(schema)
	default:
		return unknown(schema)
	}
}

// variantCase interprets one externally-tagged oneOf branch:
// {required: ["Tag"], properties: {"Tag": <fields>}}.
func variantCase(c Schema, components map[string]Schema) VariantCase {
	tag := ""
	if len(c.Required) > 0 {
		tag = c.Required[0]
	}
	fields := unknown(c)
	if f, ok := c.Properties[tag]; ok {
		fields = ParamTypeFromSchema(f, components)
	}
	return VariantCase{Tag: tag, Fields: fields}
}

// coreKindFromRef matches a built-in core type by the trailing name of its $ref,
// so both the canonical ".../tii#/$defs/<Name>" and legacy ".../core#<Name>"
// forms resolve.
func coreKindFromRef(ref string) (ParamKind, bool) {
	name := ref
	if i := strings.LastIndexAny(ref, "#/"); i >= 0 {
		name = ref[i+1:]
	}
	switch name {
	case "Bytes":
		return KindBytes, true
	case "Address":
		return KindAddress, true
	case "UtxoRef":
		return KindUtxoRef, true
	case "Utxo":
		return KindUtxo, true
	case "AnyAsset":
		return KindAnyAsset, true
	default:
		return 0, false
	}
}

func unknown(schema Schema) ParamType {
	s := schema
	return ParamType{Kind: KindUnknown, Schema: &s}
}
