#!/usr/bin/env bash
# Install MinIO for native benchmarking.
# Usage: ./scripts/install-minio.sh
set -euo pipefail

if command -v minio &>/dev/null; then
    echo "MinIO already installed: $(minio --version 2>&1 | head -1)"
    exit 0
fi

echo "Installing MinIO..."

if command -v brew &>/dev/null; then
    brew install minio/stable/minio
elif command -v go &>/dev/null; then
    echo "Homebrew not found, installing via go install..."
    go install github.com/minio/minio@latest
else
    echo "Error: neither brew nor go found. Install Homebrew or Go first."
    exit 1
fi

echo "MinIO installed: $(minio --version 2>&1 | head -1)"
