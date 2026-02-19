#!/usr/bin/env bash
# Run benchmark comparison: LiteIO (horse driver via S3) vs MinIO vs RustFS vs SeaweedFS vs Garage.
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
SEAWEEDFS_DATA=""
GARAGE_DATA=""
MINIO_PID=""
RUSTFS_PID=""
LITEIO_PID=""
SEAWEEDFS_MASTER_PID=""
SEAWEEDFS_VOLUME_PID=""
SEAWEEDFS_FILER_PID=""
SEAWEEDFS_S3_PID=""
GARAGE_PID=""
LITEIO_BIN=""

# Ports
MINIO_PORT=9000
RUSTFS_PORT=9100
LITEIO_PORT=9200
SEAWEEDFS_S3_PORT=8333
SEAWEEDFS_MASTER_PORT=9333
SEAWEEDFS_VOLUME_PORT=8080
SEAWEEDFS_FILER_PORT=8888
GARAGE_S3_PORT=3900
GARAGE_RPC_PORT=3901
GARAGE_ADMIN_PORT=3903

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

    # SeaweedFS: stop in reverse order (S3 → Filer → Volume → Master)
    if [[ -n "$SEAWEEDFS_S3_PID" ]] && kill -0 "$SEAWEEDFS_S3_PID" 2>/dev/null; then
        echo "Stopping SeaweedFS S3 (PID $SEAWEEDFS_S3_PID)..."
        kill "$SEAWEEDFS_S3_PID" 2>/dev/null || true
        wait "$SEAWEEDFS_S3_PID" 2>/dev/null || true
    fi
    if [[ -n "$SEAWEEDFS_FILER_PID" ]] && kill -0 "$SEAWEEDFS_FILER_PID" 2>/dev/null; then
        echo "Stopping SeaweedFS Filer (PID $SEAWEEDFS_FILER_PID)..."
        kill "$SEAWEEDFS_FILER_PID" 2>/dev/null || true
        wait "$SEAWEEDFS_FILER_PID" 2>/dev/null || true
    fi
    if [[ -n "$SEAWEEDFS_VOLUME_PID" ]] && kill -0 "$SEAWEEDFS_VOLUME_PID" 2>/dev/null; then
        echo "Stopping SeaweedFS Volume (PID $SEAWEEDFS_VOLUME_PID)..."
        kill "$SEAWEEDFS_VOLUME_PID" 2>/dev/null || true
        wait "$SEAWEEDFS_VOLUME_PID" 2>/dev/null || true
    fi
    if [[ -n "$SEAWEEDFS_MASTER_PID" ]] && kill -0 "$SEAWEEDFS_MASTER_PID" 2>/dev/null; then
        echo "Stopping SeaweedFS Master (PID $SEAWEEDFS_MASTER_PID)..."
        kill "$SEAWEEDFS_MASTER_PID" 2>/dev/null || true
        wait "$SEAWEEDFS_MASTER_PID" 2>/dev/null || true
    fi

    if [[ -n "$GARAGE_PID" ]] && kill -0 "$GARAGE_PID" 2>/dev/null; then
        echo "Stopping Garage (PID $GARAGE_PID)..."
        kill "$GARAGE_PID" 2>/dev/null || true
        wait "$GARAGE_PID" 2>/dev/null || true
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

    if [[ -n "$SEAWEEDFS_DATA" && -d "$SEAWEEDFS_DATA" ]]; then
        echo "Removing SeaweedFS temp dir: $SEAWEEDFS_DATA"
        rm -rf "$SEAWEEDFS_DATA"
    fi

    if [[ -n "$GARAGE_DATA" && -d "$GARAGE_DATA" ]]; then
        echo "Removing Garage temp dir: $GARAGE_DATA"
        rm -rf "$GARAGE_DATA"
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

# Parse which drivers to run (default: all five via S3)
DRIVERS="liteio,minio,rustfs,seaweedfs,garage"
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
RUN_SEAWEEDFS=false
RUN_GARAGE=false
if [[ "$DRIVERS" == *"liteio"* ]]; then RUN_LITEIO=true; fi
if [[ "$DRIVERS" == *"minio"* ]]; then RUN_MINIO=true; fi
if [[ "$DRIVERS" == *"rustfs"* ]]; then RUN_RUSTFS=true; fi
if [[ "$DRIVERS" == *"seaweedfs"* ]]; then RUN_SEAWEEDFS=true; fi
if [[ "$DRIVERS" == *"garage"* ]]; then RUN_GARAGE=true; fi

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

if $RUN_SEAWEEDFS; then
    if ! command -v weed &>/dev/null; then
        echo "SeaweedFS not found. Installing..."
        "$SCRIPT_DIR/install-seaweedfs.sh"
    else
        echo "  SeaweedFS: $(weed version 2>&1 | head -1)"
    fi
fi

if $RUN_GARAGE; then
    if ! command -v garage &>/dev/null; then
        echo "Garage not found. Installing..."
        "$SCRIPT_DIR/install-garage.sh"
    else
        echo "  Garage: $(garage --version 2>&1 | head -1)"
    fi
fi

if $RUN_MINIO || $RUN_RUSTFS || $RUN_LITEIO || $RUN_SEAWEEDFS || $RUN_GARAGE; then
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

if $RUN_SEAWEEDFS; then
    SEAWEEDFS_DATA="$(mktemp -d /tmp/seaweedfs-bench.XXXXXX)"
    mkdir -p "$SEAWEEDFS_DATA"/{master,volume,filer}
    echo "  SeaweedFS data dir: $SEAWEEDFS_DATA"

    # Write S3 config
    cat > "$SEAWEEDFS_DATA/s3.json" <<'SEAWEEDFS_S3_CONFIG'
{
  "identities": [
    {
      "name": "admin",
      "credentials": [
        {
          "accessKey": "admin",
          "secretKey": "adminpassword"
        }
      ],
      "actions": ["*"]
    }
  ]
}
SEAWEEDFS_S3_CONFIG

    # Start Master
    weed master \
        -ip=127.0.0.1 \
        -ip.bind=127.0.0.1 \
        -port="$SEAWEEDFS_MASTER_PORT" \
        -mdir="$SEAWEEDFS_DATA/master" \
        -volumeSizeLimitMB=100 \
        >"$SEAWEEDFS_DATA/master.log" 2>&1 &
    SEAWEEDFS_MASTER_PID=$!
    echo "  SeaweedFS Master started (PID $SEAWEEDFS_MASTER_PID, port $SEAWEEDFS_MASTER_PORT)"

    # Wait for master
    wait_for_server "SeaweedFS Master" "http://localhost:${SEAWEEDFS_MASTER_PORT}/cluster/status" 30

    # Start Volume
    weed volume \
        -mserver="127.0.0.1:${SEAWEEDFS_MASTER_PORT}" \
        -ip=127.0.0.1 \
        -ip.bind=127.0.0.1 \
        -port="$SEAWEEDFS_VOLUME_PORT" \
        -dir="$SEAWEEDFS_DATA/volume" \
        >"$SEAWEEDFS_DATA/volume.log" 2>&1 &
    SEAWEEDFS_VOLUME_PID=$!
    echo "  SeaweedFS Volume started (PID $SEAWEEDFS_VOLUME_PID, port $SEAWEEDFS_VOLUME_PORT)"

    # Wait for volume
    wait_for_server "SeaweedFS Volume" "http://localhost:${SEAWEEDFS_VOLUME_PORT}/status" 30

    # Start Filer
    weed filer \
        -master="127.0.0.1:${SEAWEEDFS_MASTER_PORT}" \
        -ip=127.0.0.1 \
        -ip.bind=127.0.0.1 \
        -port="$SEAWEEDFS_FILER_PORT" \
        >"$SEAWEEDFS_DATA/filer.log" 2>&1 &
    SEAWEEDFS_FILER_PID=$!
    echo "  SeaweedFS Filer started (PID $SEAWEEDFS_FILER_PID, port $SEAWEEDFS_FILER_PORT)"

    # Wait for filer
    wait_for_server "SeaweedFS Filer" "http://localhost:${SEAWEEDFS_FILER_PORT}/" 30

    # Start S3 gateway
    weed s3 \
        -filer="127.0.0.1:${SEAWEEDFS_FILER_PORT}" \
        -ip.bind=127.0.0.1 \
        -port="$SEAWEEDFS_S3_PORT" \
        -config="$SEAWEEDFS_DATA/s3.json" \
        >"$SEAWEEDFS_DATA/s3.log" 2>&1 &
    SEAWEEDFS_S3_PID=$!
    echo "  SeaweedFS S3 started (PID $SEAWEEDFS_S3_PID, port $SEAWEEDFS_S3_PORT)"
fi

if $RUN_GARAGE; then
    GARAGE_DATA="$(mktemp -d /tmp/garage-bench.XXXXXX)"
    mkdir -p "$GARAGE_DATA"/{meta,data}
    echo "  Garage data dir: $GARAGE_DATA"

    # Generate secrets
    GARAGE_RPC_SECRET="$(openssl rand -hex 32)"
    GARAGE_ADMIN_TOKEN_VAL="benchadmintoken"

    # Write config
    cat > "$GARAGE_DATA/garage.toml" <<GARAGE_CONFIG
metadata_dir = "$GARAGE_DATA/meta"
data_dir = "$GARAGE_DATA/data"
db_engine = "sqlite"

replication_factor = 1

rpc_bind_addr = "127.0.0.1:${GARAGE_RPC_PORT}"
rpc_public_addr = "127.0.0.1:${GARAGE_RPC_PORT}"
rpc_secret = "$GARAGE_RPC_SECRET"

[s3_api]
s3_region = "us-east-1"
api_bind_addr = "127.0.0.1:${GARAGE_S3_PORT}"

[admin]
api_bind_addr = "127.0.0.1:${GARAGE_ADMIN_PORT}"
admin_token = "$GARAGE_ADMIN_TOKEN_VAL"
GARAGE_CONFIG

    # Start Garage server
    garage -c "$GARAGE_DATA/garage.toml" server \
        >"$GARAGE_DATA/garage.log" 2>&1 &
    GARAGE_PID=$!
    echo "  Garage started (PID $GARAGE_PID, S3 port $GARAGE_S3_PORT)"

    # Wait for Garage admin API
    wait_for_server "Garage" "http://localhost:${GARAGE_ADMIN_PORT}/health" 30

    # Configure node layout
    echo "  Configuring Garage node layout..."
    GARAGE_NODE_ID=$(garage -c "$GARAGE_DATA/garage.toml" status 2>&1 | grep -oE '[0-9a-f]{16}' | head -1)
    if [[ -n "$GARAGE_NODE_ID" ]]; then
        garage -c "$GARAGE_DATA/garage.toml" layout assign -z dc1 -c 1G "$GARAGE_NODE_ID" 2>/dev/null || true
        garage -c "$GARAGE_DATA/garage.toml" layout apply --version 1 2>/dev/null || true
        echo "  Garage node layout configured (node $GARAGE_NODE_ID)"
    else
        echo "  Warning: could not get Garage node ID"
    fi

    # Create API key
    echo "  Creating Garage API key..."
    GARAGE_KEY_OUTPUT=$(garage -c "$GARAGE_DATA/garage.toml" key create bench-key 2>&1)
    GARAGE_ACCESS_KEY=$(echo "$GARAGE_KEY_OUTPUT" | grep "Key ID:" | awk '{print $NF}')
    GARAGE_SECRET_KEY=$(echo "$GARAGE_KEY_OUTPUT" | grep "Secret key:" | awk '{print $NF}')

    if [[ -z "$GARAGE_ACCESS_KEY" || -z "$GARAGE_SECRET_KEY" ]]; then
        echo "  Warning: could not parse Garage key. Full output:"
        echo "$GARAGE_KEY_OUTPUT"
        echo "  Skipping Garage benchmark."
        RUN_GARAGE=false
    else
        echo "  Garage key created: $GARAGE_ACCESS_KEY"

        # Create bucket and allow key
        garage -c "$GARAGE_DATA/garage.toml" bucket create test-bucket 2>/dev/null || true
        garage -c "$GARAGE_DATA/garage.toml" bucket allow --read --write --owner test-bucket --key bench-key 2>/dev/null || true
        echo "  Garage bucket test-bucket created and permissions granted"

        # Save credentials for later use
        echo "$GARAGE_ACCESS_KEY" > "$GARAGE_DATA/access_key"
        echo "$GARAGE_SECRET_KEY" > "$GARAGE_DATA/secret_key"
    fi
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

if $RUN_SEAWEEDFS; then
    if ! wait_for_server "SeaweedFS S3" "http://localhost:${SEAWEEDFS_S3_PORT}/" 30; then
        echo "SeaweedFS S3 failed to start. Log:"
        tail -20 "$SEAWEEDFS_DATA/s3.log" 2>/dev/null || true
        exit 1
    fi
fi

if $RUN_GARAGE; then
    if ! wait_for_server "Garage S3" "http://localhost:${GARAGE_S3_PORT}/" 30; then
        echo "Garage failed to start. Log:"
        tail -20 "$GARAGE_DATA/garage.log" 2>/dev/null || true
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

if $RUN_SEAWEEDFS; then
    create_bucket "SeaweedFS" "http://localhost:${SEAWEEDFS_S3_PORT}" admin adminpassword
fi

# Garage bucket already created during setup (via garage CLI)
if $RUN_GARAGE; then
    echo "  Garage test-bucket already created via CLI"
fi

echo ""

# Run benchmark
echo "=== Running benchmark ==="
echo "Drivers: $DRIVERS (all via S3 transport)"
echo ""

REPO_ROOT="${REPO_ROOT:-$(cd "$PROJECT_DIR/../.." && pwd)}"
cd "$REPO_ROOT"

# Build driver list for benchmark, injecting Garage credentials into DSN
BENCH_DRIVERS="$DRIVERS"

# If running Garage, we need to pass its dynamic credentials via environment
if $RUN_GARAGE && [[ -f "$GARAGE_DATA/access_key" ]]; then
    export GARAGE_BENCH_ACCESS_KEY="$(cat "$GARAGE_DATA/access_key")"
    export GARAGE_BENCH_SECRET_KEY="$(cat "$GARAGE_DATA/secret_key")"
fi

go run ./blueprints/liteio/cmd/bench \
    --drivers "$BENCH_DRIVERS" \
    --docker-stats=false \
    --cleanup-data=true \
    --output "$PROJECT_DIR/report" \
    --formats markdown,json,csv \
    "${PARSED_ARGS[@]}"

echo ""
echo "=== Benchmark complete ==="
echo "Reports saved to: $PROJECT_DIR/report/"
