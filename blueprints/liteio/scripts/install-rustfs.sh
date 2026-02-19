#!/usr/bin/env bash
# Install RustFS for native benchmarking.
# Usage: ./scripts/install-rustfs.sh
set -euo pipefail

if command -v rustfs &>/dev/null; then
    echo "RustFS already installed: $(rustfs --version 2>&1 | head -1)"
    exit 0
fi

echo "Installing RustFS..."

if command -v brew &>/dev/null; then
    brew tap rustfs/homebrew-tap
    brew install rustfs
else
    echo "Error: brew not found. Install Homebrew first."
    echo "Alternatively, download from https://github.com/rustfs/rustfs/releases"
    exit 1
fi

echo "RustFS installed: $(rustfs --version 2>&1 | head -1)"
