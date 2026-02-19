#!/usr/bin/env bash
# Run fair S3-mode benchmark comparison for horse vs bee clusters.
#
# Starts:
# - 8 bee data nodes (3-node and 5-node pools)
# - 4 LiteIO S3 gateways:
#   - local_s3 on :9213
#   - horse_s3 on :9210
#   - bee3net_s3 on :9211
#   - bee5net_s3 on :9212
#
# Then runs cmd/bench with S3 drivers only.
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"

LOG_DIR="/tmp/bee-s3-bench"
mkdir -p "$LOG_DIR"

PIDS=()
ALL_PORTS=(9401 9402 9403 9501 9502 9503 9504 9505 9210 9211 9212 9213)

kill_port() {
  local port="$1"
  if command -v lsof >/dev/null 2>&1; then
    local pids
    pids="$(lsof -ti tcp:"$port" || true)"
    if [[ -n "$pids" ]]; then
      kill $pids 2>/dev/null || true
      sleep 0.2
    fi
  fi
}

wait_http() {
  local name="$1"
  local url="$2"
  local timeout_sec="${3:-30}"
  local elapsed=0
  while (( elapsed < timeout_sec )); do
    if curl -fsS "$url" >/dev/null 2>&1; then
      return 0
    fi
    sleep 1
    ((elapsed+=1))
  done
  echo "[ERROR] timeout waiting for ${name}: ${url}" >&2
  return 1
}

start_bg() {
  local name="$1"
  shift
  local log_file="$LOG_DIR/${name}.log"
  "$@" >"$log_file" 2>&1 &
  local pid=$!
  PIDS+=("$pid")
  echo "[start] ${name} pid=${pid} log=${log_file}"
}

cleanup() {
  set +e
  for pid in "${PIDS[@]:-}"; do
    if kill -0 "$pid" >/dev/null 2>&1; then
      kill "$pid" >/dev/null 2>&1 || true
    fi
  done
  sleep 0.5
  for pid in "${PIDS[@]:-}"; do
    if kill -0 "$pid" >/dev/null 2>&1; then
      kill -9 "$pid" >/dev/null 2>&1 || true
    fi
  done
}
trap cleanup EXIT INT TERM

for p in "${ALL_PORTS[@]}"; do
  kill_port "$p"
done

echo "[setup] preparing bee node directories"
for p in 9401 9402 9403 9501 9502 9503 9504 9505; do
  rm -rf "/tmp/bee-net/node-${p}"
  mkdir -p "/tmp/bee-net/node-${p}"
  start_bg "bee-node-${p}" go run ./cmd/bee --listen ":${p}" --data-dir "/tmp/bee-net/node-${p}" --sync none --inline-kb 64
done

echo "[wait] bee node health"
for p in 9401 9402 9403 9501 9502 9503 9504 9505; do
  wait_http "bee-node-${p}" "http://127.0.0.1:${p}/v1/ping" 30
  echo "  bee-node-${p}: ok"
done

rm -rf /tmp/liteio-local-s3 /tmp/liteio-horse-s3 /tmp/liteio-bee3-s3 /tmp/liteio-bee5-s3
mkdir -p /tmp/liteio-local-s3 /tmp/liteio-horse-s3 /tmp/liteio-bee3-s3 /tmp/liteio-bee5-s3

start_bg "liteio-local-s3" go run ./cmd/liteio \
  --host 127.0.0.1 --port 9213 \
  --driver "local:///tmp/liteio-local-s3" \
  --access-key local --secret-key local123 --no-auth --no-log

start_bg "liteio-horse-s3" go run ./cmd/liteio \
  --host 127.0.0.1 --port 9210 \
  --driver "horse:///tmp/liteio-horse-s3?sync=none&prealloc=2048" \
  --access-key horse --secret-key horse123 --no-auth --no-log

start_bg "liteio-bee3net-s3" go run ./cmd/liteio \
  --host 127.0.0.1 --port 9211 \
  --driver "bee:///?peers=http://127.0.0.1:9401,http://127.0.0.1:9402,http://127.0.0.1:9403&replicas=3&w=1&r=1&repair=true&repair_workers=6&repair_max_kb=256" \
  --access-key bee3 --secret-key bee3123 --no-auth --no-log

start_bg "liteio-bee5net-s3" go run ./cmd/liteio \
  --host 127.0.0.1 --port 9212 \
  --driver "bee:///?peers=http://127.0.0.1:9501,http://127.0.0.1:9502,http://127.0.0.1:9503,http://127.0.0.1:9504,http://127.0.0.1:9505&replicas=3&w=1&r=1&repair=true&repair_workers=10&repair_max_kb=256" \
  --access-key bee5 --secret-key bee5123 --no-auth --no-log

echo "[wait] liteio gateway health"
wait_http "liteio-local-s3" "http://127.0.0.1:9213/healthz/ready" 30
wait_http "liteio-horse-s3" "http://127.0.0.1:9210/healthz/ready" 30
wait_http "liteio-bee3net-s3" "http://127.0.0.1:9211/healthz/ready" 30
wait_http "liteio-bee5net-s3" "http://127.0.0.1:9212/healthz/ready" 30

echo "[bench] running fair S3 benchmark"
go run ./cmd/bench \
  --quick \
  --drivers local_s3,horse_s3,bee3net_s3,bee5net_s3 \
  --docker-stats=false \
  --cleanup-data=false \
  --formats markdown,json,csv \
  --output ./report/bee_s3_fair \
  "$@"

echo "[done] reports at ./report/bee_s3_fair"
