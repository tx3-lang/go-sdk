#!/usr/bin/env bash
#
# CI artifact — not part of the SDK.
#
# Renders the .trix/client-lib codegen plugin against the shared transfer
# fixture and verifies the result the way a consumer would: the rendered module
# resolves the published go-sdk at the version its generated go.mod pins — no
# replace directives.
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

( cd "$gen" && go mod tidy && go build ./... )

echo "codegen check passed"
