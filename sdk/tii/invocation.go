package tii

import (
	"strings"

	"github.com/tx3-lang/go-sdk/sdk/core"
)

// Invocation represents a partially-built transaction: the transaction name,
// its TIR, expected parameters, and the arguments collected so far.
type Invocation struct {
	tir      core.TirEnvelope
	params   map[string]ParamType   // declared param name → type
	required map[string]bool        // set of required param names
	args     map[string]interface{} // collected args (original case keys)
}

// Params returns the declared parameters and their types.
func (inv *Invocation) Params() map[string]ParamType {
	return inv.params
}

// UnspecifiedParams returns the names of required parameters that have not been
// set via SetArg. Useful for validation before resolution.
func (inv *Invocation) UnspecifiedParams() []string {
	var missing []string
	for name := range inv.required {
		if _, ok := inv.findArg(name); !ok {
			missing = append(missing, name)
		}
	}
	return missing
}

// SetArg sets a single argument value. The key is matched case-insensitively.
func (inv *Invocation) SetArg(name string, value interface{}) {
	// Store with the canonical (lowercased) key
	inv.args[strings.ToLower(name)] = value
}

// SetArgs sets multiple arguments at once.
func (inv *Invocation) SetArgs(args map[string]interface{}) {
	for k, v := range args {
		inv.SetArg(k, v)
	}
}

// WithArg returns the invocation with an additional argument (fluent API).
func (inv *Invocation) WithArg(name string, value interface{}) *Invocation {
	inv.SetArg(name, value)
	return inv
}

// WithArgs returns the invocation with additional arguments (fluent API).
func (inv *Invocation) WithArgs(args map[string]interface{}) *Invocation {
	inv.SetArgs(args)
	return inv
}

// IntoResolveRequest converts the invocation into TRP ResolveParams.
// Returns the TIR envelope and the collected argument map.
//
// Each arg is marshalled type-directed by its .tii ParamType: top-level scalars
// come back bare (the resolver coerces them via the flat TIR type), aggregates
// (record/list/tuple/map/variant) come back as the self-describing TaggedArg
// wire form. An unmapped arg has no type, so it passes through untouched. Arg
// keys are lowercased on set; params keep their original case, so match
// case-insensitively.
func (inv *Invocation) IntoResolveRequest() (core.TirEnvelope, core.ArgMap, error) {
	// Check for missing required params
	missing := inv.UnspecifiedParams()
	if len(missing) > 0 {
		return core.TirEnvelope{}, nil, &MissingParamsError{Params: missing}
	}

	args := make(core.ArgMap, len(inv.args))
	for key, value := range inv.args {
		if param, ok := inv.findParam(key); ok {
			encoded, err := Encode(param, value)
			if err != nil {
				return core.TirEnvelope{}, nil, err
			}
			args[key] = encoded
		} else {
			args[key] = value
		}
	}

	return inv.tir, args, nil
}

// findParam looks up a parameter type by name, case-insensitively. Arg keys are
// lowercased on set, while declared params keep their original case.
func (inv *Invocation) findParam(key string) (ParamType, bool) {
	for name, ty := range inv.params {
		if strings.EqualFold(name, key) {
			return ty, true
		}
	}
	return ParamType{}, false
}

// MissingParamsError is returned when required params are not set.
type MissingParamsError struct {
	Params []string
}

func (e *MissingParamsError) Error() string {
	return "missing required params: " + strings.Join(e.Params, ", ")
}
func (e *MissingParamsError) isTiiError() {}

// findArg looks up an argument by name, case-insensitively.
func (inv *Invocation) findArg(name string) (interface{}, bool) {
	v, ok := inv.args[strings.ToLower(name)]
	return v, ok
}
