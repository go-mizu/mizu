#!/bin/bash
# ── Generate self-signed TLS certificates for PostgreSQL ─────────────
# Runs once via docker-entrypoint-initdb.d on first container start.
# Certificates are stored in /certs (Docker volume, persisted).

set -euo pipefail

CERT_DIR="/certs"
CERT_FILE="$CERT_DIR/server.crt"
KEY_FILE="$CERT_DIR/server.key"

if [ -f "$CERT_FILE" ] && [ -f "$KEY_FILE" ]; then
  echo "init-certs: certificates already exist, skipping generation."
  exit 0
fi

echo "init-certs: generating self-signed TLS certificate..."

openssl req -new -x509 -days 3650 -nodes \
  -subj "/CN=storage-postgres/O=storage/C=US" \
  -addext "subjectAltName=DNS:storage-postgres,DNS:localhost,IP:127.0.0.1" \
  -keyout "$KEY_FILE" \
  -out "$CERT_FILE" \
  2>/dev/null

# PostgreSQL requires key file to be owned by postgres and mode 600
chmod 600 "$KEY_FILE"
chmod 644 "$CERT_FILE"
chown 999:999 "$KEY_FILE" "$CERT_FILE" 2>/dev/null || true

echo "init-certs: TLS certificate generated (valid 10 years)."
echo "init-certs:   cert = $CERT_FILE"
echo "init-certs:   key  = $KEY_FILE"
