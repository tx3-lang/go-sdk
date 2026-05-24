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
# Last verified against fleet v0.12.0 (unified Tx3ClientBuilder).
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
  'type Profile string' \
  'ProfileLocal' \
  'ProfilePreprod' \
  '_LOCAL_PROFILE' \
  'facade.FromParts' \
  'WithPartyUnchecked'; do
  grep -qF "$sym" "$gen/protocol.go" || { echo "generated protocol.go missing: $sym"; exit 1; }
done

( cd "$gen" && go mod tidy && go build ./... )

echo "codegen check passed"
