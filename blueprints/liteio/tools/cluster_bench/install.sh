#!/usr/bin/env bash
# Install all dependencies for cluster benchmarking.
# Usage: ./tools/cluster_bench/install.sh
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"

echo "=== Installing Cluster Benchmark Dependencies ==="
echo

# 1. MinIO
echo "--- MinIO ---"
if command -v minio &>/dev/null; then
    echo "  Already installed: $(minio --version 2>&1 | head -1)"
else
    echo "  Installing..."
    if command -v brew &>/dev/null; then
        brew install minio/stable/minio
    elif command -v go &>/dev/null; then
        go install github.com/minio/minio@latest
    else
        echo "  Error: neither brew nor go found"
        exit 1
    fi
    echo "  Installed: $(minio --version 2>&1 | head -1)"
fi
echo

# 2. RustFS
echo "--- RustFS ---"
if command -v rustfs &>/dev/null; then
    echo "  Already installed: $(rustfs --version 2>&1 | head -1)"
else
    echo "  Installing..."
    if command -v brew &>/dev/null; then
        brew tap rustfs/homebrew-tap 2>/dev/null || true
        brew install rustfs
    else
        echo "  Error: brew not found. Download from https://github.com/rustfs/rustfs/releases"
        exit 1
    fi
    echo "  Installed: $(rustfs --version 2>&1 | head -1)"
fi
echo

# 3. SeaweedFS
echo "--- SeaweedFS ---"
if command -v weed &>/dev/null; then
    echo "  Already installed: $(weed version 2>&1 | head -1)"
else
    echo "  Installing..."
    if command -v brew &>/dev/null; then
        brew install seaweedfs
    elif command -v go &>/dev/null; then
        go install github.com/seaweedfs/seaweedfs/weed@latest
    else
        echo "  Error: neither brew nor go found"
        exit 1
    fi
    echo "  Installed: $(weed version 2>&1 | head -1)"
fi
echo

# 4. Herd (build from source)
echo "--- Herd ---"
echo "  Building from source..."
cd "$ROOT_DIR"
GOWORK=off go build -o "${HOME}/bin/herd" ./cmd/herd/
echo "  Built: ${HOME}/bin/herd"
echo

echo "=== All dependencies installed ==="
