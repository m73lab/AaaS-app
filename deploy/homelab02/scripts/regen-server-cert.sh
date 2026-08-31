#!/bin/bash
# Regenerate server certificate with SANs for IP and hostname.
set -e

CERT_DIR="/opt/aaas/certs"
CA_CERT="$CERT_DIR/ca.crt"
CA_KEY="$CERT_DIR/ca.key"

# Get the server IP
SERVER_IP=$(hostname -I | awk '{print $1}')
SERVER_HOST=$(hostname)

echo "Server IP: $SERVER_IP"
echo "Server hostname: $SERVER_HOST"

# Create SAN config
cat > /tmp/san.cnf <<EOF
[req]
distinguished_name = req_dn
req_extensions = v3_req
prompt = no

[req_dn]
CN = presidio

[v3_req]
subjectAltName = @alt_names

[alt_names]
DNS.1 = presidio
DNS.2 = $SERVER_HOST
IP.1 = $SERVER_IP
IP.2 = 127.0.0.1
EOF

# Generate new key
openssl genrsa -out "$CERT_DIR/presidio-server.key" 2048 2>/dev/null

# Generate CSR
openssl req -new -key "$CERT_DIR/presidio-server.key" \
  -out /tmp/server.csr \
  -config /tmp/san.cnf 2>/dev/null

# Sign with CA
openssl x509 -req -in /tmp/server.csr \
  -CA "$CA_CERT" \
  -CAkey "$CA_KEY" \
  -CAcreateserial \
  -out "$CERT_DIR/presidio-server.crt" \
  -days 365 \
  -sha256 \
  -extensions v3_req \
  -extfile /tmp/san.cnf 2>/dev/null

rm -f /tmp/server.csr /tmp/san.cnf

echo "Server certificate regenerated with SANs:"
openssl x509 -in "$CERT_DIR/presidio-server.crt" -noout -text 2>/dev/null | grep -A3 "Subject Alternative Name"
