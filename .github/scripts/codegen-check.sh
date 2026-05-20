#!/usr/bin/env bash
#
# CI artifact — not part of the SDK.
#
# Renders the .trix/client-lib codegen plugin against the shared transfer
# fixture and verifies the result. The subject under test is the Handlebars
# templates + tx3c integration, not the SDK runtime.
#
# Steps: invoke `tx3c codegen`, assert the expected files exist, smoke-check
# the generated surface, and compile the output against this repo's SDK.
#
# Requires `tx3c` and `go` on PATH.
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
gen="$(mktemp -d)"
trap 'rm -rf "$gen"' EXIT

tx3c codegen \
  --tii "$repo_root/sdk/testdata/transfer.tii" \
  --template "$repo_root/.trix/client-lib" \
  --output "$gen"

for f in protocol.go go.mod; do
  test -f "$gen/$f" || { echo "missing generated file: $f"; exit 1; }
done

for sym in \
  'TargetTIIVersion' \
  'type TransferParams struct' \
  'TRANSFER_TIR' \
  'func (c *Client) Transfer(' \
  'var Profiles'; do
  grep -qF "$sym" "$gen/protocol.go" || { echo "generated protocol.go missing: $sym"; exit 1; }
done

# Build the rendered package against this repo's SDK, not a published release.
printf 'module codegentest\n\ngo 1.24.2\n\nrequire github.com/tx3-lang/go-sdk/sdk v0.0.0\n\nreplace github.com/tx3-lang/go-sdk/sdk => %s/sdk\n' \
  "$repo_root" > "$gen/go.mod"
cp "$repo_root/sdk/go.sum" "$gen/go.sum"
( cd "$gen" && GOFLAGS=-mod=mod go build ./... )

echo "codegen check passed"
