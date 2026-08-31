#!/usr/bin/env bash
# Generates a private CA plus Redis server and client certificates for
# mutual TLS between the AaaS proxy and Redis. Run once on your workstation:
#   bash scripts/gen-certs.sh
# Output lands in deploy/certs/:
#   ca.crt, ca.key                 -> CA (proxy uses ca.crt to verify server)
#   redis-server.{crt,key}        -> deploy to homelab02 (Redis side)
#   redis-client.{crt,key}        -> keep local (proxy presents this)
set -euo pipefail

OUT="deploy/certs"
DAYS=365
mkdir -p "$OUT"

echo "==> Generating CA"
openssl genrsa -out "$OUT/ca.key" 4096
openssl req -x509 -new -nodes -key "$OUT/ca.key" -sha256 -days "$DAYS" \
  -out "$OUT/ca.crt" -subj "/CN=aaS-CA"

echo "==> Redis SERVER cert (SAN includes localhost for the SSH tunnel)"
openssl genrsa -out "$OUT/redis-server.key" 2048
openssl req -new -key "$OUT/redis-server.key" -out "$OUT/redis-server.csr" \
  -subj "/CN=redis"
cat > "$OUT/redis-server.ext" <<'EOF'
subjectAltName=DNS:localhost,IP:127.0.0.1,DNS:redis
extendedKeyUsage=serverAuth
EOF
openssl x509 -req -in "$OUT/redis-server.csr" -CA "$OUT/ca.crt" -CAkey "$OUT/ca.key" \
  -CAcreateserial -out "$OUT/redis-server.crt" -days "$DAYS" -sha256 \
  -extfile "$OUT/redis-server.ext"

echo "==> Redis CLIENT cert (proxy side)"
openssl genrsa -out "$OUT/redis-client.key" 2048
openssl req -new -key "$OUT/redis-client.key" -out "$OUT/redis-client.csr" \
  -subj "/CN=aaS-proxy"
cat > "$OUT/redis-client.ext" <<'EOF'
extendedKeyUsage=clientAuth
EOF
openssl x509 -req -in "$OUT/redis-client.csr" -CA "$OUT/ca.crt" -CAkey "$OUT/ca.key" \
  -CAcreateserial -out "$OUT/redis-client.crt" -days "$DAYS" -sha256 \
  -extfile "$OUT/redis-client.ext"

echo "==> Presidio SERVER cert (mTLS edge: SAN covers presidio-tls, localhost, 127.0.0.1)"
openssl genrsa -out "$OUT/presidio-server.key" 2048
openssl req -new -key "$OUT/presidio-server.key" -out "$OUT/presidio-server.csr" \
  -subj "/CN=presidio"
cat > "$OUT/presidio-server.ext" <<'EOF'
subjectAltName=DNS:presidio-tls,DNS:localhost,IP:127.0.0.1
extendedKeyUsage=serverAuth
EOF
openssl x509 -req -in "$OUT/presidio-server.csr" -CA "$OUT/ca.crt" -CAkey "$OUT/ca.key" \
  -CAcreateserial -out "$OUT/presidio-server.crt" -days "$DAYS" -sha256 \
  -extfile "$OUT/presidio-server.ext"

echo "==> Done. Keep redis-client.* + ca.crt locally; ship ca.crt + redis-server.* + presidio-server.* to the edge."

