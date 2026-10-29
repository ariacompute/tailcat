#!/usr/bin/env bash
# Build the aria-tailcat .deb using dpkg-buildpackage.
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

# Ensure the module graph is complete before building.
go mod download

dpkg-buildpackage -us -uc -b

echo "==> built:"
ls -1 ../aria-tailcat_*.deb 2>/dev/null || true
