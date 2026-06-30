#!/bin/bash
# Generate development TLS certificates for service mesh
set -e

CERT_DIR="${1:-/root/stackrun/certs}"
mkdir -p "$CERT_DIR"

echo "Generating dev certificates in $CERT_DIR..."

# Root CA
openssl req -x509 -newkey rsa:4096 -days 365 -nodes \
    -keyout "$CERT_DIR/ca-key.pem" -out "$CERT_DIR/ca-cert.pem" \
    -subj "/CN=StackRun Dev CA" 2>/dev/null

# Server cert
openssl req -newkey rsa:4096 -nodes \
    -keyout "$CERT_DIR/server-key.pem" -out "$CERT_DIR/server-req.pem" \
    -subj "/CN=stackrun.local" 2>/dev/null

openssl x509 -req -in "$CERT_DIR/server-req.pem" -days 60 \
    -CA "$CERT_DIR/ca-cert.pem" -CAkey "$CERT_DIR/ca-key.pem" -CAcreateserial \
    -out "$CERT_DIR/server-cert.pem" 2>/dev/null

rm "$CERT_DIR/server-req.pem"

echo "✓ CA cert:     $CERT_DIR/ca-cert.pem"
echo "✓ Server cert: $CERT_DIR/server-cert.pem"
echo "✓ Server key:  $CERT_DIR/server-key.pem"
