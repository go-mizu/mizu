#!/usr/bin/env bash
# Run benchmark comparison: LiteIO (horse driver via S3) vs MinIO vs RustFS.
# All drivers go through HTTP/S3 transport for a fair comparison.
# Uses temp directories with automatic cleanup.
#
# Usage:
#   ./scripts/bench-compare.sh                    # Default: 1s per benchmark
#   ./scripts/bench-compare.sh --quick            # Quick mode: 500ms per benchmark
#   ./scripts/bench-compare.sh --drivers liteio,minio  # Specific drivers
#   ./scripts/bench-compare.sh --benchtime 2s     # Custom bench time
#   ./scripts/bench-compare.sh --progress         # Live progress
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

# Temp directories (created per run, cleaned up on exit)
MINIO_DATA=""
RUSTFS_DATA=""
LITEIO_DATA=""
MINIO_PID=""
RUSTFS_PID=""
LITEIO_PID=""
LITEIO_BIN=""

# Ports
MINIO_PORT=9000
RUSTFS_PORT=9100
LITEIO_PORT=9200

cleanup() {
    echo ""
    echo "=== Cleaning up ==="

    if [[ -n "$LITEIO_PID" ]] && kill -0 "$LITEIO_PID" 2>/dev/null; then
        echo "Stopping LiteIO (PID $LITEIO_PID)..."
        kill "$LITEIO_PID" 2>/dev/null || true
        wait "$LITEIO_PID" 2>/dev/null || true
    fi

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

    if [[ -n "$LITEIO_DATA" && -d "$LITEIO_DATA" ]]; then
        echo "Removing LiteIO temp dir: $LITEIO_DATA"
        rm -rf "$LITEIO_DATA"
    fi

    if [[ -n "$MINIO_DATA" && -d "$MINIO_DATA" ]]; then
        echo "Removing MinIO temp dir: $MINIO_DATA"
        rm -rf "$MINIO_DATA"
    fi

    if [[ -n "$RUSTFS_DATA" && -d "$RUSTFS_DATA" ]]; then
        echo "Removing RustFS temp dir: $RUSTFS_DATA"
        rm -rf "$RUSTFS_DATA"
    fi

    if [[ -n "$LITEIO_BIN" && -f "$LITEIO_BIN" ]]; then
        echo "Removing LiteIO binary: $LITEIO_BIN"
        rm -f "$LITEIO_BIN"
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

# Parse which drivers to run (default: all three via S3)
DRIVERS="liteio,minio,rustfs"
EXTRA_ARGS=()
for arg in "$@"; do
    if [[ "$arg" == --drivers=* ]]; then
        DRIVERS="${arg#--drivers=}"
    elif [[ "$arg" == "--drivers" ]]; then
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

RUN_LITEIO=false
RUN_MINIO=false
RUN_RUSTFS=false
if [[ "$DRIVERS" == *"liteio"* ]]; then RUN_LITEIO=true; fi
if [[ "$DRIVERS" == *"minio"* ]]; then RUN_MINIO=true; fi
if [[ "$DRIVERS" == *"rustfs"* ]]; then RUN_RUSTFS=true; fi

echo "=== LiteIO Native Benchmark Comparison (Fair: All S3) ==="
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

if $RUN_MINIO || $RUN_RUSTFS || $RUN_LITEIO; then
    if ! command -v aws &>/dev/null; then
        echo "Error: aws CLI not found. Install with: brew install awscli"
        exit 1
    fi
    echo "  AWS CLI: installed"
fi
echo ""

# Build LiteIO binary if needed
if $RUN_LITEIO; then
    echo "=== Building LiteIO ==="
    LITEIO_BIN="$(mktemp /tmp/liteio-bench.XXXXXX)"
    # Build from repo root to use go.work workspace (resolves local mizu dependency)
    REPO_ROOT="$(cd "$PROJECT_DIR/../.." && pwd)"
    echo "  Building cmd/liteio..."
    (cd "$REPO_ROOT" && go build -o "$LITEIO_BIN" ./blueprints/liteio/cmd/liteio)
    echo "  Built: $LITEIO_BIN"
    echo ""
fi

# Start servers
echo "=== Starting servers ==="

if $RUN_LITEIO; then
    LITEIO_DATA="$(mktemp -d /tmp/liteio-bench-data.XXXXXX)"
    echo "  LiteIO data dir: $LITEIO_DATA"

    "$LITEIO_BIN" \
        --driver "horse://$LITEIO_DATA?sync=none" \
        --port "$LITEIO_PORT" \
        --access-key liteio \
        --secret-key liteio123 \
        --no-log \
        >"$LITEIO_DATA/liteio.log" 2>&1 &
    LITEIO_PID=$!
    echo "  LiteIO started (PID $LITEIO_PID, port $LITEIO_PORT, horse driver)"
fi

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

if $RUN_LITEIO; then
    if ! wait_for_server "LiteIO" "http://localhost:${LITEIO_PORT}/healthz/ready" 30; then
        echo "LiteIO failed to start. Log:"
        tail -20 "$LITEIO_DATA/liteio.log" 2>/dev/null || true
        exit 1
    fi
fi

if $RUN_MINIO; then
    if ! wait_for_server "MinIO" "http://localhost:${MINIO_PORT}/minio/health/live" 30; then
        echo "MinIO failed to start. Log:"
        tail -20 "$MINIO_DATA/minio.log" 2>/dev/null || true
        exit 1
    fi
fi

if $RUN_RUSTFS; then
    if ! wait_for_server "RustFS" "http://localhost:${RUSTFS_PORT}/" 30; then
        echo "RustFS failed to start. Log:"
        tail -20 "$RUSTFS_DATA/rustfs.log" 2>/dev/null || true
        exit 1
    fi
fi

echo ""

# Create buckets
echo "=== Creating test buckets ==="

if $RUN_LITEIO; then
    create_bucket "LiteIO" "http://localhost:${LITEIO_PORT}" liteio liteio123
fi

if $RUN_MINIO; then
    create_bucket "MinIO" "http://localhost:${MINIO_PORT}" minioadmin minioadmin
fi

if $RUN_RUSTFS; then
    create_bucket "RustFS" "http://localhost:${RUSTFS_PORT}" rustfsadmin rustfsadmin
fi

echo ""

# Run benchmark
echo "=== Running benchmark ==="
echo "Drivers: $DRIVERS (all via S3 transport)"
echo ""

REPO_ROOT="${REPO_ROOT:-$(cd "$PROJECT_DIR/../.." && pwd)}"
cd "$REPO_ROOT"

go run ./blueprints/liteio/cmd/bench \
    --drivers "$DRIVERS" \
    --docker-stats=false \
    --cleanup-data=true \
    --output "$PROJECT_DIR/report" \
    --formats markdown,json,csv \
    "${PARSED_ARGS[@]}"

echo ""
echo "=== Benchmark complete ==="
echo "Reports saved to: $PROJECT_DIR/report/"
