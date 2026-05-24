package facade

import "github.com/tx3-lang/go-sdk/sdk/core"

// Profile is a named profile baked into a client: environment values and
// party-address overrides keyed by name.
//
// Produced either by deconstructing a loaded *tii.Protocol inside
// Tx3ClientBuilder.FromProtocol, or by feeding the per-profile JSON blob a
// generated codegen client embeds through Tx3ClientBuilder.FromParts.
type Profile struct {
	Environment core.EnvMap
	Parties     map[string]string
}
