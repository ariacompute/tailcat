#!/usr/bin/env bash
# test-cat.sh - local smoke test for aria-tailcat.
# Builds aria-tailcat and runs it in-process (the tailcat server runs inside the
# aria-tailcat process via the imported github.com/tailscale/tailcat module), in
# listen mode, and verifies a tailcat address (tc...) is printed.
#
# Requires outbound access to a DERP map (https://tailcat.dev/derpmap.json) so
# tailcat can obtain its address.
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

echo "==> building aria-tailcat"
go build -o build/aria-tailcat ./cmd/aria-tailcat

cat > build/test-cat.yaml <<'EOF'
serve:
  mode: listen
  ports:
    - "8080"
EOF

echo "==> running aria-tailcat (listen :8080) — expecting a tc... address"
if timeout 60 ./build/aria-tailcat run --config build/test-cat.yaml 2>&1 | grep -qiE 'tc[0-9a-z]+'; then
  echo "PASS: tailcat address printed"
else
  echo "WARN: no tailcat address observed (likely no DERP/network access)"
  exit 1
fi
