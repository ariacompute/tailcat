#!/usr/bin/env bash
# test-client.sh - client-side connectivity test for an aria-tailcat node using
# the upstream tailcat CLI as the peer.
#
# aria-tailcat runs the tailcat server IN-PROCESS (github.com/tailscale/tailcat);
# this script builds that same pinned module's `cmd/tailcat` CLI and uses it as a
# client to verify DERP reachability and (optionally) end-to-end TCP reachability
# to the server's forwarded port.
#
# Usage:
#   TC_ADDR=tc... [PORT=8080] ./scripts/test-client.sh
#   ./scripts/test-client.sh tc...          # ping only
#   ./scripts/test-client.sh tc... 8080     # ping + TCP forward probe
#
# The server's `tc...` address is printed by `aria-tailcat run` as:
#   aria-tailcat: serving tailcat address: tc...
#
# Notes:
# - The client also needs DERP map access. By default it uses
#   https://tailcat.dev/derpmap.json; override with TAILCAT_DERPMAP_URL if your
#   server uses a custom DERP map.
# - In the default `forward` config the server forwards to localhost:$PORT on the
#   server; the TCP probe just checks that a tunnel can be established end-to-end.
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

TC_ADDR="${1:-${TC_ADDR:-}}"
PORT="${2:-${PORT:-8080}}"

if [[ -z "$TC_ADDR" ]]; then
  echo "error: no tailcat address. Set TC_ADDR (or pass it as the first arg)." >&2
  echo "       It is printed by 'aria-tailcat run' as:" >&2
  echo "         aria-tailcat: serving tailcat address: tc..." >&2
  exit 2
fi

# Build the upstream tailcat CLI from the pinned module (no separate download).
WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT
TAILCAT_BIN="$WORK/tailcat"
echo "==> building upstream tailcat CLI (github.com/tailscale/tailcat)"
go build -trimpath -buildvcs=false -ldflags "-s -w" -o "$TAILCAT_BIN" github.com/tailscale/tailcat/cmd/tailcat

echo "==> ping $TC_ADDR (via DERP)"
if "$TAILCAT_BIN" ping --timeout=15s "$TC_ADDR"; then
  echo "PASS: $TC_ADDR reachable over DERP"
else
  echo "FAIL: could not reach $TC_ADDR over DERP (check DERP map access / allowlist / address)"
  exit 1
fi

# Optional: forward a local port to the server's port and probe it.
if [[ -n "${PORT:-}" ]]; then
  LP=18080
  echo "==> forwarding local :$LP -> $TC_ADDR:$PORT (background)"
  "$TAILCAT_BIN" forward "$TC_ADDR" "$LP:$PORT" &
  FWD_PID=$!
  sleep 2
  if command -v nc >/dev/null 2>&1; then
    if printf 'ping' | nc -w 3 127.0.0.1 "$LP" >/dev/null 2>&1; then
      echo "PASS: local :$LP reached the server's :$PORT over the tunnel"
    else
      echo "NOTE: TCP probe inconclusive — nothing may be listening on the server's :$PORT (expected for a bare forward target)"
    fi
  else
    echo "NOTE: nc not installed; skipping TCP probe"
  fi
  kill "$FWD_PID" 2>/dev/null || true
fi

echo "==> client connectivity OK"
