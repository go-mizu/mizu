#!/usr/bin/env bash
# Install Garage for native benchmarking.
# Usage: ./scripts/install-garage.sh
set -euo pipefail

if command -v garage &>/dev/null; then
    echo "Garage already installed: $(garage --version 2>&1 | head -1)"
    exit 0
fi

echo "Installing Garage..."

if command -v brew &>/dev/null; then
    brew install garage
else
    echo "Error: brew not found. Install Homebrew first."
    echo "Alternatively, download from https://garagehq.deuxfleurs.fr/download/"
    exit 1
fi

echo "Garage installed: $(garage --version 2>&1 | head -1)"
