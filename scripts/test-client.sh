#!/usr/bin/env bash
# test-client.sh - client-side connectivity test for an aria-tailcat node using
# the upstream tailcat CLI as the peer.
#
# aria-tailcat runs the tailcat server IN-PROCESS (github.com/tailscale/tailcat);
# this script obtains the upstream tailcat CLI and uses it as a client to verify
# DERP reachability and (optionally) end-to-end TCP reachability to the server's
# forwarded port.
#
# The upstream CLI is obtained without a local Go toolchain:
#   - linux:   download the release tarball (tailcat_<ver>_linux_<arch>.tar.gz)
#   - darwin:  no release asset exists, so fall back to `go build` if `go` is present
# The downloaded binary is cached at build/tailcat (gitignored) for reuse.
#
# Usage:
#   TC_ADDR=tcp... [PORT=8080] [BIND=0.0.0.0] [KEEP=1] ./scripts/test-client.sh
#   ./scripts/test-client.sh tcp...                 # ping only (omit PORT)
#   PORT= KEEP=0 ./scripts/test-client.sh tcp...    # one-shot ping + probe, then stop
#   ./scripts/test-client.sh tcp... 8080           # ping + keep a forward alive on :18080
#
# Defaults: PORT=8080, BIND=0.0.0.0 (listen on all interfaces), KEEP=1 (keep the
# forward running for browsing; Ctrl-C to stop). Set KEEP=0 for a one-shot smoke
# test that tears down the forward after probing.
#
# The server's address is printed by `aria-tailcat run` as:
#   aria-tailcat: serving tailcat address: tcp...
# (the prefix is "tcp", not "tc")
#
# Notes:
# - The client also needs DERP map access. By default it uses
#   https://tailcat.dev/derpmap.json; override with TAILCAT_DERPMAP_URL if your
#   server uses a custom DERP map.
# - In the default `forward` config the server forwards to localhost:$PORT on the
#   server; the TCP probe just checks that a tunnel can be established end-to-end.
# - With KEEP=1 (default) the script blocks until Ctrl-C, leaving the forwarded
#   port up so you can browse it. Set KEEP=0 (or run under `timeout`) for CI.
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

TC_ADDR="${1:-${TC_ADDR:-}}"
PORT="${2:-${PORT:-8080}}"
BIND="${BIND:-0.0.0.0}"
KEEP="${KEEP:-1}"
TAILCAT_VERSION="${TAILCAT_VERSION:-0.7.0}"
FWD_PID=""

if [[ -z "$TC_ADDR" ]]; then
  echo "error: no tailcat address. Set TC_ADDR (or pass it as the first arg)." >&2
  echo "       It is printed by 'aria-tailcat run' as:" >&2
  echo "         aria-tailcat: serving tailcat address: tcp..." >&2
  exit 2
fi

download() {
  local url="$1" out="$2"
  if command -v curl >/dev/null 2>&1; then
    curl -fsSL "$url" -o "$out"
  elif command -v wget >/dev/null 2>&1; then
    wget -qO "$out" "$url"
  else
    echo "error: need curl or wget to download tailcat" >&2
    exit 1
  fi
}

# --- Obtain the upstream tailcat CLI -----------------------------------------
WORK="$(mktemp -d)"
cleanup() {
  [[ -n "$FWD_PID" ]] && kill "$FWD_PID" 2>/dev/null || true
  rm -rf "$WORK"
}
trap cleanup EXIT INT TERM

TAILCAT_BIN="$ROOT_DIR/build/tailcat"
if [[ -x "$TAILCAT_BIN" ]]; then
  echo "==> reusing cached tailcat CLI ($TAILCAT_BIN)"
else
  OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
  ARCH="$(uname -m)"
  case "$ARCH" in
    x86_64|amd64)  ARCH=amd64 ;;
    aarch64|arm64) ARCH=arm64 ;;
    armv7l|armv7)  ARCH=armv7 ;;
    *) echo "error: unsupported architecture: $ARCH" >&2; exit 2 ;;
  esac

  if [[ "$OS" == "linux" ]]; then
    ASSET="tailcat_${TAILCAT_VERSION}_${OS}_${ARCH}.tar.gz"
    URL="https://github.com/tailscale/tailcat/releases/download/v${TAILCAT_VERSION}/${ASSET}"
    echo "==> downloading upstream tailcat $TAILCAT_VERSION ($ASSET)"
    if ! download "$URL" "$WORK/tailcat.tar.gz"; then
      echo "error: failed to download $URL" >&2
      exit 1
    fi
    tar -xzf "$WORK/tailcat.tar.gz" -C "$WORK"
  elif [[ "$OS" == "darwin" ]]; then
    echo "==> no macOS release asset for tailcat $TAILCAT_VERSION; building from source"
    if ! command -v go >/dev/null 2>&1; then
      echo "error: 'go' not found; install Go or place a linux tailcat binary at build/tailcat" >&2
      exit 1
    fi
    go build -trimpath -buildvcs=false -ldflags "-s -w" -o "$WORK/tailcat" github.com/tailscale/tailcat/cmd/tailcat
  else
    echo "error: unsupported OS: $OS" >&2
    exit 2
  fi

  TAILCAT_SRC="$(find "$WORK" -type f -name tailcat | head -1)"
  if [[ -z "$TAILCAT_SRC" ]]; then
    echo "error: tailcat binary not found in downloaded archive" >&2
    exit 1
  fi
  mkdir -p "$(dirname "$TAILCAT_BIN")"
  mv "$TAILCAT_SRC" "$TAILCAT_BIN"
  chmod +x "$TAILCAT_BIN"
fi

echo "==> using tailcat: $("$TAILCAT_BIN" version 2>&1 | head -1)"

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
  FWD_ARGS=(forward)
  if [[ -n "$BIND" ]]; then
    FWD_ARGS+=(--bind="$BIND")
  fi
  FWD_ARGS+=("$TC_ADDR" "$LP:$PORT")
  echo "==> forwarding local :$LP -> $TC_ADDR:$PORT (background${BIND:+ bind=$BIND})"
  "$TAILCAT_BIN" "${FWD_ARGS[@]}" &
  FWD_PID=$!
  sleep 2
  # Probe target: if bound to a specific non-wildcard address, hit that; else loopback.
  PROBE_HOST=127.0.0.1
  if [[ -n "$BIND" && "$BIND" != "0.0.0.0" ]]; then
    PROBE_HOST="$BIND"
  fi
  if command -v nc >/dev/null 2>&1; then
    if printf 'ping' | nc -w 3 "$PROBE_HOST" "$LP" >/dev/null 2>&1; then
      echo "PASS: $PROBE_HOST:$LP reached the server's :$PORT over the tunnel"
    else
      echo "NOTE: TCP probe inconclusive — nothing may be listening on the server's :$PORT (expected for a bare forward target)"
    fi
  else
    echo "NOTE: nc not installed; skipping TCP probe"
  fi
  if [[ "$KEEP" == "1" ]]; then
    echo "==> tunnel kept alive (KEEP=1). Press Ctrl-C to stop."
    echo "==> browse: http://127.0.0.1:$LP/ (or <this-host-ip>:$LP if the firewall allows)"
    wait "$FWD_PID" || true
  else
    kill "$FWD_PID" 2>/dev/null || true
  fi
fi

echo "==> client connectivity OK"
