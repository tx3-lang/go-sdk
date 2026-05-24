# tx3-sdk (Go)

[![Go Reference](https://pkg.go.dev/badge/github.com/tx3-lang/go-sdk/sdk.svg)](https://pkg.go.dev/github.com/tx3-lang/go-sdk/sdk)
[![CI](https://github.com/tx3-lang/go-sdk/actions/workflows/ci.yml/badge.svg)](https://github.com/tx3-lang/go-sdk/actions/workflows/ci.yml)
[![Tx3 docs](https://img.shields.io/badge/Tx3-docs-blue.svg)](https://docs.txpipe.io/tx3)
[![License: Apache-2.0](https://img.shields.io/badge/license-Apache--2.0-blue.svg)](LICENSE)

The official Go SDK for [Tx3](https://docs.txpipe.io/tx3) — a DSL and protocol suite for defining and executing UTxO-based blockchain transactions declaratively. Load a compiled `.tii` protocol, bind parties and signers, and drive the full transaction lifecycle (resolve, sign, submit, confirm) via the Transaction Resolve Protocol (TRP).

This repository is organized as a monorepo. The publishable Go SDK module lives in `sdk/`.

## What is Tx3

Tx3 is a domain-specific language and protocol suite for declarative, type-safe UTxO transactions. Authors write `.tx3` files describing parties, environment, and transactions; the toolchain compiles them to `.tii` artifacts that this SDK loads at runtime to drive the resolve → sign → submit → wait lifecycle through a TRP server. See the [Tx3 docs](https://docs.txpipe.io/tx3) for project context.

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
    // 1. Load a compiled .tii protocol
    protocol, err := tx3.ProtocolFromFile("transfer.tii")
    if err != nil {
        log.Fatal(err)
    }

    // 2. Build a Cardano signer from a mnemonic
    mySigner, err := signer.CardanoSignerFromMnemonic(
        "addr_test1qz...",
        "word1 word2 ... word24",
    )
    if err != nil {
        log.Fatal(err)
    }

    // 3. Build a client: configure TRP, profile, and parties on the builder
    client, err := tx3.ProtocolClient(protocol).
        TRP(trp.ClientOptions{Endpoint: "http://localhost:3000"}).
        WithProfile("preprod").
        WithParty("sender", tx3.SignerParty(mySigner)).
        WithParty("receiver", tx3.AddressParty("addr_test1qz...")).
        Build()
    if err != nil {
        log.Fatal(err)
    }

    ctx := context.Background()

    // 4. Build, resolve, sign, submit, and wait for confirmation
    txb, err := client.Tx("transfer")
    if err != nil {
        log.Fatal(err)
    }
    resolved, err := txb.Arg("quantity", 10_000_000).Resolve(ctx)
    if err != nil {
        log.Fatal(err)
    }
    signed, err := resolved.Sign()
    if err != nil {
        log.Fatal(err)
    }
    submitted, err := signed.Submit(ctx)
    if err != nil {
        log.Fatal(err)
    }
    status, err := submitted.WaitForConfirmed(ctx, tx3.DefaultPollConfig())
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Confirmed at stage %s\n", status.Stage)
}
```

All fallible validation — TRP endpoint present, profile declared, every bound
party declared — happens inside `Build()`, which returns `*MissingTrpEndpointError`,
`*tii.UnknownProfileError`, or `*facade.UnknownPartyError` (all discriminable via
`errors.As`). Optional setters never fail, so chains stay fluent. Profile selection
is **builder-only**: there is no profile-switching method on the built client.
Switching profiles requires a new builder.

> **Note on naming:** Go uses the cross-package factory `tx3.ProtocolClient(p)`
> (equivalently `facade.FromProtocol(p)`) instead of a `Protocol.Client()` method
> to avoid a `tii → facade → tii` import cycle. Other SDKs (web, python) do
> expose `Protocol.client()`.

## Concepts

| SDK Type | Glossary Term | Description |
|---|---|---|
| `Protocol` | TII / Protocol | Loaded `.tii` file exposing transactions, parties, and profiles |
| `Tx3ClientBuilder` | Client builder | Fluent builder seeded by `tx3.ProtocolClient(p)` or `facade.FromParts(...)`; absorbs all fallible validation in `Build()` |
| `Tx3Client` | Facade | Output of `Tx3ClientBuilder.Build()` — owns the deconstructed protocol parts, TRP client, profile, and party bindings |
| `TxBuilder` | Invocation builder | Source-agnostic; collects args, resolves via TRP |
| `Party` | Party | Named participant — `AddressParty` (read-only) or `SignerParty` (signing) |
| `Profile` | Profile | `{Environment, Parties}` value baked into the client; embedded by codegen plugins, decomposed from `Protocol` by `FromProtocol` |
| `MissingTrpEndpointError` / `UnknownPartyError` | Builder errors | Returned by `Build()`; implement the `facade.FacadeError` marker |
| `Signer` | Signer | Interface producing a `TxWitness` for a `SignRequest` |
| `SignRequest` | SignRequest | Input passed to `Signer.Sign`: `TxHashHex` + `TxCborHex` |
| `CardanoSigner` | Cardano Signer | BIP32-Ed25519 signer at `m/1852'/1815'/0'/0/0` |
| `Ed25519Signer` | Ed25519 Signer | Generic raw-key Ed25519 signer |
| `ResolvedTx` | Resolved transaction | Output of `Resolve()`, ready for signing |
| `SignedTx` | Signed transaction | Output of `Sign()`, ready for submission |
| `SubmittedTx` | Submitted transaction | Output of `Submit()`, pollable for status |
| `PollConfig` | Poll configuration | Controls `WaitForConfirmed` / `WaitForFinalized` polling |

## Advanced usage

### Skipping the runtime `.tii` (codegen flow)

If you've run `trix codegen` to generate typed bindings, your generated `Client`
embeds the per-transaction TIR envelopes and per-profile data at codegen time —
no `.tii` artifact at runtime. Under the hood it seeds the same builder via
`facade.FromParts(transactions, profiles, knownParties)` and routes typed
per-party setters through `WithPartyUnchecked`. You can also call `FromParts`
directly from hand-written code (`tx3.FromParts` is the same factory re-exported
at the package root):

```go
import tx3 "github.com/tx3-lang/go-sdk/sdk"

client, err := tx3.FromParts(transactions, profiles, []string{"sender", "receiver"}).
    TRPEndpoint("http://localhost:3000").
    WithPartyUnchecked("sender", tx3.SignerParty(mySigner)).
    Build()
```

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

Implement the `Signer` interface. `Sign` receives a `SignRequest` carrying both
the tx hash and the full tx CBOR; hash-based signers read `TxHashHex`, tx-based
signers (e.g. wallet bridges) read `TxCborHex`.

```go
import "github.com/tx3-lang/go-sdk/sdk/signer"

type MySigner struct { /* ... */ }

func (s *MySigner) Address() string { return "addr_test1..." }

func (s *MySigner) Sign(request signer.SignRequest) (*signer.TxWitness, error) {
    // sign request.TxHashHex with your key
    return signer.NewVKeyWitness(pubKeyHex, signatureHex), nil
}

client.WithParty("sender", tx3.SignerParty(&MySigner{}))
```

### Manual witness attachment

When a witness is produced outside any registered signer — for example by an
external wallet app or a remote signing service — resolve the transaction
first, hand the resolved hash (or full tx CBOR) to the wallet, then attach
the returned witness before `Sign()`:

```go
import "github.com/tx3-lang/go-sdk/sdk/trp"

resolved, err := client.Tx("transfer").Arg("quantity", 10_000_000).Resolve(ctx)
if err != nil { /* ... */ }

// Hand resolved.Hash (or resolved.TxHex) to the external wallet and get
// back a witness. The wallet needs the resolved tx to sign.
var witness trp.TxWitness // sign resolved.Hash with external wallet

signed, err := resolved.AddWitness(witness).Sign()
if err != nil { /* ... */ }

submitted, err := signed.Submit(ctx)
```

`AddWitness` may be called any number of times; manual witnesses are appended after registered-signer witnesses in attach order.

### Error handling

All errors are discriminable via `errors.As()` — no string matching needed:

```go
import (
    "github.com/tx3-lang/go-sdk/sdk/facade"
    "github.com/tx3-lang/go-sdk/sdk/tii"
)

_, err := tx3.ProtocolClient(protocol).Build()
var missingTRP *facade.MissingTrpEndpointError
if errors.As(err, &missingTRP) {
    // no TRP endpoint supplied via TRP() / TRPEndpoint()
}

_, err = client.Tx("transfer")
var unknownTx *tii.UnknownTxError
if errors.As(err, &unknownTx) {
    fmt.Printf("Tx %q not declared by protocol\n", unknownTx.Name)
}
```

## Tx3 protocol compatibility

- **TRP protocol version:** v1beta0
- **TII schema version:** v1beta0

## Testing

- Unit tests are co-located with the packages they exercise (`*_test.go` next to source files).
- End-to-end (e2e) tests live under `sdk/e2e/` and are gated by the `e2e` build tag.

```bash
# from go-sdk/sdk
go test ./... -count=1
go test -tags=e2e ./e2e -count=1
```

## License

Apache-2.0 — see [LICENSE](LICENSE).
