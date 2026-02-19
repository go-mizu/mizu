#!/usr/bin/env bash
# One-command cluster benchmark: install → start → bench → stop → report.
# Usage:
#   ./tools/cluster_bench/run.sh                          # Full benchmark
#   ./tools/cluster_bench/run.sh --quick                  # Quick mode
#   ./tools/cluster_bench/run.sh --systems minio,herd     # Specific systems
#   ./tools/cluster_bench/run.sh --benchtime 2s           # Custom bench time
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"

echo "=== Cluster S3 Benchmark Runner ==="
echo "Root: $ROOT_DIR"
echo

# Step 1: Install dependencies
echo "--- Step 1: Installing dependencies ---"
bash "$SCRIPT_DIR/install.sh"
echo

# Step 2: Build and run the benchmark tool
echo "--- Step 2: Running cluster benchmarks ---"
cd "$ROOT_DIR"
GOWORK=off go run ./tools/cluster_bench/ "$@"
echo

echo "=== Done ==="
echo "Reports: ./report/cluster/"
