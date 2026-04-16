# tx3-sdk (Go)

[![Go Reference](https://pkg.go.dev/badge/github.com/tx3-lang/go-sdk/sdk.svg)](https://pkg.go.dev/github.com/tx3-lang/go-sdk/sdk)
[![License: Apache-2.0](https://img.shields.io/badge/license-Apache--2.0-blue.svg)](LICENSE)

The official Go SDK for [Tx3](https://tx3.land) — a DSL and protocol suite for defining and executing UTxO-based blockchain transactions declaratively. Load a compiled `.tii` protocol, bind parties and signers, and drive the full transaction lifecycle (resolve, sign, submit, confirm) via the Transaction Resolve Protocol (TRP).

This repository is organized as a monorepo. The publishable Go SDK module lives in `sdk/`.

## Installation

```bash
go get github.com/tx3-lang/go-sdk/sdk
```

## Quick start

```go
package main

import (
    "context"
    "fmt"
    "log"

    tx3 "github.com/tx3-lang/go-sdk/sdk"
    "github.com/tx3-lang/go-sdk/sdk/signer"
    "github.com/tx3-lang/go-sdk/sdk/trp"
)

func main() {
    // Load a compiled .tii protocol
    protocol, err := tx3.ProtocolFromFile("transfer.tii")
    if err != nil {
        log.Fatal(err)
    }

    // Connect to a TRP server
    trpClient := tx3.NewTRPClient(trp.ClientOptions{
        Endpoint: "http://localhost:3000",
    })

    // Create a Cardano signer from a mnemonic
    mySigner, err := signer.CardanoSignerFromMnemonic(
        "addr_test1qz...",
        "word1 word2 ... word24",
    )
    if err != nil {
        log.Fatal(err)
    }

    // Configure client with profile and parties
    client := tx3.NewClient(protocol, trpClient).
        WithProfile("preprod").
        WithParty("sender", tx3.SignerParty(mySigner)).
        WithParty("receiver", tx3.AddressParty("addr_test1qz..."))

    ctx := context.Background()

    // Build, resolve, sign, submit, and wait for confirmation
    status, err := client.Tx("transfer").
        Arg("quantity", 10_000_000).
        Resolve(ctx)         // -> ResolvedTx
        .Sign()              // -> SignedTx
        .Submit(ctx)         // -> SubmittedTx
        .WaitForConfirmed(ctx, tx3.DefaultPollConfig())
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Confirmed at stage %s\n", status.Stage)
}
```

> **Note:** The quick-start example above shows the conceptual flow. In real code, check errors at each step since Go doesn't chain errors across method calls.

## Concepts

| SDK Type | Glossary Term | Description |
|---|---|---|
| `Protocol` | TII / Protocol | Loaded `.tii` file exposing transactions, parties, and profiles |
| `Tx3Client` | Facade | Entry point holding protocol, TRP client, and party bindings |
| `TxBuilder` | Invocation builder | Collects args, resolves via TRP |
| `Party` | Party | Named participant — `AddressParty` (read-only) or `SignerParty` (signing) |
| `Signer` | Signer | Interface producing a `TxWitness` for a tx hash |
| `CardanoSigner` | Cardano Signer | BIP32-Ed25519 signer at `m/1852'/1815'/0'/0/0` |
| `Ed25519Signer` | Ed25519 Signer | Generic raw-key Ed25519 signer |
| `ResolvedTx` | Resolved transaction | Output of `Resolve()`, ready for signing |
| `SignedTx` | Signed transaction | Output of `Sign()`, ready for submission |
| `SubmittedTx` | Submitted transaction | Output of `Submit()`, pollable for status |
| `PollConfig` | Poll configuration | Controls `WaitForConfirmed` / `WaitForFinalized` polling |

## Advanced usage

### Low-level TRP client

```go
import "github.com/tx3-lang/go-sdk/sdk/trp"

client := trp.NewClient(trp.ClientOptions{
    Endpoint: "http://localhost:3000",
    Headers:  map[string]string{"Authorization": "Bearer token"},
})

envelope, err := client.Resolve(ctx, trp.ResolveParams{...})
resp, err := client.Submit(ctx, trp.SubmitParams{...})
status, err := client.CheckStatus(ctx, []string{txHash})
```

### Custom Signer

```go
type MySigner struct { /* ... */ }

func (s *MySigner) Address() string { return "addr_test1..." }
func (s *MySigner) Sign(txHashHex string) (*signer.TxWitness, error) {
    // Your signing logic here
    return signer.NewVKeyWitness(pubKeyHex, signatureHex), nil
}

client.WithParty("sender", tx3.SignerParty(&MySigner{}))
```

### Error handling

All errors are discriminable via `errors.As()` — no string matching needed:

```go
import "github.com/tx3-lang/go-sdk/sdk/facade"

_, err := client.Tx("transfer").Resolve(ctx)
var unknownParty *facade.UnknownPartyError
if errors.As(err, &unknownParty) {
    fmt.Printf("Party %q not found in protocol\n", unknownParty.Name)
}
```

## Tx3 protocol compatibility

- **TRP protocol version:** v1beta0
- **TII schema version:** v1beta0

## License

Apache-2.0 — see [LICENSE](LICENSE).
