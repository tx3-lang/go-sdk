// Package tx3sdk is the main package for TX3 SDK functionality
package tx3sdk

import (
	"github.com/tx3-lang/go-sdk/sdk/trp"
)

// NewTRPClient creates a new TRP client with the given options
func NewTRPClient(options trp.ClientOptions) *trp.Client {
	return trp.NewClient(options)
}
