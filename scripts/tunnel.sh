#!/usr/bin/env bash
# Tunnels the AaaS proxy (running server-side on homelab02) to your workstation
# so your apps can call it as if local. Redis/Presidio are NOT exposed — only
# the proxy's 8080 is forwarded. Keep this running, then point your LLM client
# at http://localhost:8080/v1/chat/completions.
set -euo pipefail

HOMELAB02="${HOMELAB02:?set HOMELAB02=user@host}"
echo "==> Tunneling localhost:8080 (proxy) and localhost:5444 (admin mTLS) -> $HOMELAB02 (Ctrl-C to stop)"
exec ssh -N \
  -L 8080:localhost:8080 \
  -L 5444:localhost:5444 \
  "$HOMELAB02"
