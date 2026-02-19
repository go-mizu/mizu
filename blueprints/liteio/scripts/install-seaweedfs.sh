#!/usr/bin/env bash
# Install SeaweedFS for native benchmarking.
# Usage: ./scripts/install-seaweedfs.sh
set -euo pipefail

if command -v weed &>/dev/null; then
    echo "SeaweedFS already installed: $(weed version 2>&1 | head -1)"
    exit 0
fi

echo "Installing SeaweedFS..."

if command -v brew &>/dev/null; then
    brew install seaweedfs
elif command -v go &>/dev/null; then
    echo "Homebrew not found, installing via go install..."
    go install github.com/seaweedfs/seaweedfs/weed@latest
else
    echo "Error: neither brew nor go found. Install Homebrew or Go first."
    exit 1
fi

echo "SeaweedFS installed: $(weed version 2>&1 | head -1)"
