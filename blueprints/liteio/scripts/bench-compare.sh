#!/usr/bin/env bash
# Run benchmark comparison: horse (in-process) vs MinIO vs RustFS (native servers).
# Uses temp directories with automatic cleanup.
#
# Usage:
#   ./scripts/bench-compare.sh                    # Default: 1s per benchmark
#   ./scripts/bench-compare.sh --quick            # Quick mode: 500ms per benchmark
#   ./scripts/bench-compare.sh --drivers horse,minio  # Specific drivers
#   ./scripts/bench-compare.sh --benchtime 2s     # Custom bench time
#   ./scripts/bench-compare.sh --progress         # Live progress
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

# Temp directories (created per run, cleaned up on exit)
MINIO_DATA=""
RUSTFS_DATA=""
MINIO_PID=""
RUSTFS_PID=""

# Ports
MINIO_PORT=9000
RUSTFS_PORT=9100

cleanup() {
    echo ""
    echo "=== Cleaning up ==="

    if [[ -n "$MINIO_PID" ]] && kill -0 "$MINIO_PID" 2>/dev/null; then
        echo "Stopping MinIO (PID $MINIO_PID)..."
        kill "$MINIO_PID" 2>/dev/null || true
        wait "$MINIO_PID" 2>/dev/null || true
    fi

    if [[ -n "$RUSTFS_PID" ]] && kill -0 "$RUSTFS_PID" 2>/dev/null; then
        echo "Stopping RustFS (PID $RUSTFS_PID)..."
        kill "$RUSTFS_PID" 2>/dev/null || true
        wait "$RUSTFS_PID" 2>/dev/null || true
    fi

    if [[ -n "$MINIO_DATA" && -d "$MINIO_DATA" ]]; then
        echo "Removing MinIO temp dir: $MINIO_DATA"
        rm -rf "$MINIO_DATA"
    fi

    if [[ -n "$RUSTFS_DATA" && -d "$RUSTFS_DATA" ]]; then
        echo "Removing RustFS temp dir: $RUSTFS_DATA"
        rm -rf "$RUSTFS_DATA"
    fi

    # Horse driver data is cleaned by --cleanup-data flag
    if [[ -d /tmp/horse-bench ]]; then
        echo "Removing horse temp dir: /tmp/horse-bench"
        rm -rf /tmp/horse-bench
    fi

    echo "Cleanup complete."
}

trap cleanup EXIT INT TERM

wait_for_server() {
    local name="$1"
    local url="$2"
    local max_wait="${3:-30}"
    local elapsed=0

    echo -n "  Waiting for $name..."
    while true; do
        # Accept any HTTP response (200, 403, etc.) as "server is running"
        local code
        code=$(curl -so /dev/null -w '%{http_code}' "$url" 2>/dev/null) || code="000"
        if [[ "$code" != "000" ]]; then
            echo " ready (${elapsed}s, HTTP $code)"
            return 0
        fi
        sleep 1
        elapsed=$((elapsed + 1))
        if [[ $elapsed -ge $max_wait ]]; then
            echo " TIMEOUT after ${max_wait}s"
            return 1
        fi
        echo -n "."
    done
}

create_bucket() {
    local name="$1"
    local endpoint="$2"
    local access_key="$3"
    local secret_key="$4"

    echo "  Creating test-bucket on $name..."
    AWS_ACCESS_KEY_ID="$access_key" \
    AWS_SECRET_ACCESS_KEY="$secret_key" \
    AWS_DEFAULT_REGION=us-east-1 \
    aws --endpoint-url="$endpoint" s3 mb s3://test-bucket 2>/dev/null || true
}

# Parse which drivers to run (default: all three)
DRIVERS="horse,minio,rustfs"
EXTRA_ARGS=()
for arg in "$@"; do
    if [[ "$arg" == --drivers=* ]]; then
        DRIVERS="${arg#--drivers=}"
    elif [[ "$arg" == "--drivers" ]]; then
        # Next arg will be the drivers value, handled by shift below
        :
    else
        EXTRA_ARGS+=("$arg")
    fi
done

# Re-parse to handle --drivers VALUE (two-arg form)
PARSED_ARGS=()
skip_next=false
for i in "$@"; do
    if $skip_next; then
        DRIVERS="$i"
        skip_next=false
        continue
    fi
    if [[ "$i" == "--drivers" ]]; then
        skip_next=true
        continue
    fi
    if [[ "$i" == --drivers=* ]]; then
        DRIVERS="${i#--drivers=}"
        continue
    fi
    PARSED_ARGS+=("$i")
done

RUN_MINIO=false
RUN_RUSTFS=false
if [[ "$DRIVERS" == *"minio"* ]]; then RUN_MINIO=true; fi
if [[ "$DRIVERS" == *"rustfs"* ]]; then RUN_RUSTFS=true; fi

echo "=== LiteIO Native Benchmark Comparison ==="
echo "Drivers: $DRIVERS"
echo ""

# Check prerequisites
echo "=== Checking prerequisites ==="

if $RUN_MINIO; then
    if ! command -v minio &>/dev/null; then
        echo "MinIO not found. Installing..."
        "$SCRIPT_DIR/install-minio.sh"
    else
        echo "  MinIO: $(minio --version 2>&1 | head -1)"
    fi
fi

if $RUN_RUSTFS; then
    if ! command -v rustfs &>/dev/null; then
        echo "RustFS not found. Installing..."
        "$SCRIPT_DIR/install-rustfs.sh"
    else
        echo "  RustFS: $(rustfs --version 2>&1 | head -1)"
    fi
fi

if ! command -v aws &>/dev/null; then
    echo "Error: aws CLI not found. Install with: brew install awscli"
    exit 1
fi
echo "  AWS CLI: installed"
echo ""

# Start servers
echo "=== Starting servers ==="

if $RUN_MINIO; then
    MINIO_DATA="$(mktemp -d /tmp/minio-bench.XXXXXX)"
    echo "  MinIO data dir: $MINIO_DATA"

    MINIO_ROOT_USER=minioadmin \
    MINIO_ROOT_PASSWORD=minioadmin \
    minio server "$MINIO_DATA" --address ":${MINIO_PORT}" --console-address ":9001" \
        >"$MINIO_DATA/minio.log" 2>&1 &
    MINIO_PID=$!
    echo "  MinIO started (PID $MINIO_PID, port $MINIO_PORT)"
fi

if $RUN_RUSTFS; then
    RUSTFS_DATA="$(mktemp -d /tmp/rustfs-bench.XXXXXX)"
    echo "  RustFS data dir: $RUSTFS_DATA"

    RUSTFS_VOLUMES="$RUSTFS_DATA" \
    RUSTFS_ACCESS_KEY=rustfsadmin \
    RUSTFS_SECRET_KEY=rustfsadmin \
    RUSTFS_ADDRESS=":${RUSTFS_PORT}" \
    RUSTFS_CONSOLE_ENABLE=false \
    RUST_LOG=error \
    rustfs "$RUSTFS_DATA" \
        >"$RUSTFS_DATA/rustfs.log" 2>&1 &
    RUSTFS_PID=$!
    echo "  RustFS started (PID $RUSTFS_PID, port $RUSTFS_PORT)"
fi

echo ""

# Wait for servers to be ready
echo "=== Waiting for servers ==="

if $RUN_MINIO; then
    if ! wait_for_server "MinIO" "http://localhost:${MINIO_PORT}/minio/health/live" 30; then
        echo "MinIO failed to start. Log:"
        cat "$MINIO_DATA/minio.log" 2>/dev/null | tail -20
        exit 1
    fi
fi

if $RUN_RUSTFS; then
    if ! wait_for_server "RustFS" "http://localhost:${RUSTFS_PORT}/" 30; then
        echo "RustFS failed to start. Log:"
        cat "$RUSTFS_DATA/rustfs.log" 2>/dev/null | tail -20
        exit 1
    fi
fi

echo ""

# Create buckets
echo "=== Creating test buckets ==="

if $RUN_MINIO; then
    create_bucket "MinIO" "http://localhost:${MINIO_PORT}" minioadmin minioadmin
fi

if $RUN_RUSTFS; then
    create_bucket "RustFS" "http://localhost:${RUSTFS_PORT}" rustfsadmin rustfsadmin
fi

echo ""

# Run benchmark
echo "=== Running benchmark ==="
echo "Drivers: $DRIVERS"
echo ""

cd "$PROJECT_DIR"

GOWORK=off go run ./cmd/bench \
    --drivers "$DRIVERS" \
    --docker-stats=false \
    --cleanup-data=true \
    --output ./report \
    --formats markdown,json,csv \
    "${PARSED_ARGS[@]}"

echo ""
echo "=== Benchmark complete ==="
echo "Reports saved to: $PROJECT_DIR/report/"
