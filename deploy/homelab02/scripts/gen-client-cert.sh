#!/bin/bash
# Generate a client certificate for AaaS mTLS access.
# Run on homelab02 as root (needs access to CA key).
set -e

CERT_DIR="/opt/aaas/certs"
CA_CERT="$CERT_DIR/ca.crt"
CA_KEY="$CERT_DIR/ca.key"
CLIENT_NAME="${1:-client}"

if [ ! -f "$CA_KEY" ]; then
  echo "ERROR: CA key not found at $CA_KEY"
  exit 1
fi

# Generate client key
openssl genrsa -out "$CERT_DIR/$CLIENT_NAME.key" 2048 2>/dev/null

# Generate client CSR
openssl req -new -key "$CERT_DIR/$CLIENT_NAME.key" \
  -out "$CERT_DIR/$CLIENT_NAME.csr" \
  -subj "/CN=$CLIENT_NAME/O=AaaS Clients" 2>/dev/null

# Sign with CA
openssl x509 -req -in "$CERT_DIR/$CLIENT_NAME.csr" \
  -CA "$CA_CERT" \
  -CAkey "$CA_KEY" \
  -CAcreateserial \
  -out "$CERT_DIR/$CLIENT_NAME.crt" \
  -days 365 \
  -sha256 2>/dev/null

# Cleanup CSR
rm -f "$CERT_DIR/$CLIENT_NAME.csr"

echo "Client certificate generated:"
echo "  Certificate: $CERT_DIR/$CLIENT_NAME.crt"
echo "  Private key:  $CERT_DIR/$CLIENT_NAME.key"
echo "  CA certificate (for verification): $CA_CERT"
