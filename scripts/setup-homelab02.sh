#!/usr/bin/env bash
# Deploys the entire AaaS stack (Redis+mTLS, Presidio, proxy) to homelab02 and
# brings it up with Docker Compose. The proxy itself runs server-side; you only
# reach it via an SSH tunnel to :8080.
#
# Prerequisites (run on your workstation):
#   export HOMELAB02=user@homelab02
#   export REDIS_PASSWORD='a-strong-password'
#   export AaaS_MASTER_KEY='32-byte-master-key-for-envelope-encryption'
#   export AaaS_KEK_VERSION=1            # optional
# Usage:
#   bash scripts/setup-homelab02.sh
set -euo pipefail

HOMELAB02="${HOMELAB02:?set HOMELAB02=user@host}"
REDIS_PASSWORD="${REDIS_PASSWORD:?set REDIS_PASSWORD}"
AaaS_MASTER_KEY="${AaaS_MASTER_KEY:?set AaaS_MASTER_KEY}"
AaaS_KEK_VERSION="${AaaS_KEK_VERSION:-1}"
REMOTE_DIR="/opt/aaas"

echo "==> Rendering config"
bash scripts/render-config.sh

echo "==> Writing .env for compose (secrets stay on the server)"
printf 'AaaS_MASTER_KEY=%s\nAaaS_KEK_VERSION=%s\nREDIS_PASSWORD=%s\n' \
  "$AaaS_MASTER_KEY" "$AaaS_KEK_VERSION" "$REDIS_PASSWORD" > .deploy.env
mv .deploy.env .env

echo "==> Shipping project to $HOMELAB02:$REMOTE_DIR"
rsync -az --exclude .git --exclude bin --exclude '*.log' ./ "$HOMELAB02:$REMOTE_DIR/"

echo "==> Rendering redis.conf and starting stack"
ssh "$HOMELAB02" bash -s <<EOF
set -e
cd $REMOTE_DIR
sed -i 's/__REDIS_PASSWORD__/$REDIS_PASSWORD/' redis.conf.template && mv -f redis.conf.template redis.conf
cd $REMOTE_DIR && docker compose up -d --build
EOF

echo "==> Done. Run 'make tunnel' to reach the proxy at http://localhost:8080."
