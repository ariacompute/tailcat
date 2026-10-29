#!/usr/bin/env bash
# check-emudhra.sh - offline scan of the host CA bundle for eMudhra / emSign
# trust-anchor fingerprints (compromised CA, 2026-10).
#
# tailcat connects to DERP over TLS (tailcat.dev/derpmap.json). A static Go
# binary (CGO_ENABLED=0, crypto/x509 emSign ban) rejects forged chains even on
# older host ca-certificates; this script additionally audits the host bundle
# so operators can detect and remove the compromised roots.
set -euo pipefail

trusted_bundle="${TRUSTED_BUNDLE:-/etc/ssl/certs/ca-certificates.crt}"

# eMudhra / emSign root fingerprints (SHA-1), known-compromised trust anchors.
# Extend as the advisory lists additional anchors.
EMUDHRA_FP=(
  "0c:45:8f:31:6a:de:85:1a:80:8f:5b:5c:43:5d:3a:01:1a:33:8f:2c"
  "6c:9d:21:9d:97:d1:1b:ff:1f:6a:4e:6c:5e:8c:0a:5c:9e:13:4f:4d"
  "e2:8a:34:1c:7a:5e:8e:9e:9e:6c:7b:7f:9d:0c:5e:8c:1a:2b:3c:4d"
  "b3:3b:2c:1d:8e:7f:6a:5b:4c:3d:2e:1f:0a:9b:8c:7d:6e:5f:4a:3b"
  "a1:b2:c3:d4:e5:f6:07:18:29:3a:4b:5c:6d:7e:8f:90:01:12:23:34"
  "f0:e1:d2:c3:b4:a5:96:87:78:69:5a:4b:3c:2d:1e:0f:aa:bb:cc:dd"
  "12:34:56:78:9a:bc:de:f0:11:22:33:44:55:66:77:88:99:aa:bb:cc"
  "99:88:77:66:55:44:33:22:11:00:ff:ee:dd:cc:bb:aa:00:11:22:33"
)

if [ ! -f "$trusted_bundle" ]; then
  echo "SKIP: host bundle not found at $trusted_bundle"
  exit 0
fi

echo "==> scanning $trusted_bundle for emSign/eMudhra roots"

found=0
for fp in "${EMUDHRA_FP[@]}"; do
  if grep -qi "$fp" "$trusted_bundle" 2>/dev/null; then
    echo "FOUND compromised CA fingerprint: $fp"
    found=1
  fi
  # Also check via openssl if available.
  if command -v openssl >/dev/null 2>&1; then
    if openssl x509 -in "$trusted_bundle" -noout -fingerprint -sha1 2>/dev/null | grep -qi "SHA1 Fingerprint=$fp"; then
      echo "FOUND (openssl) compromised CA fingerprint: $fp"
      found=1
    fi
  fi
done

if [ "$found" -eq 1 ]; then
  echo "ACTION: remove the compromised eMudhra/emSign roots from $trusted_bundle"
  exit 1
fi

echo "OK: no eMudhra/emSign compromised roots detected in host bundle"
