#!/usr/bin/env bash
# Build and release storage CLI binaries to R2.
#
# Usage:
#   ./scripts/release-cli.sh           # build + upload
#   ./scripts/release-cli.sh --build   # build only (no upload)
#   ./scripts/release-cli.sh --upload  # upload only (skip build)
#
# Environment:
#   CLI_UPLOAD_KEY    Admin key for uploads (required for --upload)
#
# Requires: go, curl

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
WORKER_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
REPO_ROOT="$(cd "$WORKER_DIR/../../../../.." && pwd)"
CMD_DIR="$REPO_ROOT/cmd/storage"
BUILD_DIR="$WORKER_DIR/build"
UPLOAD_URL="https://storage.liteio.dev/cli/upload"

# Extract version from main.go
VERSION=$(grep 'version.*=' "$CMD_DIR/main.go" | head -1 | sed 's/.*"\([0-9][^"]*\)".*/\1/')

BUILD=1
UPLOAD=1
for arg in "$@"; do
  case "$arg" in
    --build)  UPLOAD=0 ;;
    --upload) BUILD=0 ;;
  esac
done

echo ""
echo "  Storage CLI release — v$VERSION"
echo ""

TARGETS=(
  "darwin  arm64"
  "darwin  amd64"
  "linux   arm64"
  "linux   amd64"
  "windows amd64"
  "windows arm64"
)

# ── Build ───────────────────────────────────────────────────────────

if [[ $BUILD -eq 1 ]]; then
  mkdir -p "$BUILD_DIR"

  for target in "${TARGETS[@]}"; do
    os=$(echo "$target" | awk '{print $1}')
    arch=$(echo "$target" | awk '{print $2}')
    output="storage-${os}-${arch}"
    [[ "$os" == "windows" ]] && output="${output}.exe"

    echo "  build  $output"
    cd "$REPO_ROOT"
    CGO_ENABLED=0 GOOS=$os GOARCH=$arch \
      go build -ldflags="-s -w" -o "$BUILD_DIR/$output" ./cmd/storage
  done

  echo ""
  echo "  Built $(ls "$BUILD_DIR"/storage-* 2>/dev/null | wc -l | tr -d ' ') binaries in $BUILD_DIR"
  echo ""

  # Show sizes
  for f in "$BUILD_DIR"/storage-*; do
    size=$(wc -c < "$f" | tr -d ' ')
    name=$(basename "$f")
    if [[ $size -ge 1048576 ]]; then
      hr=$(awk "BEGIN {printf \"%.1f MB\", $size/1048576}")
    elif [[ $size -ge 1024 ]]; then
      hr=$(awk "BEGIN {printf \"%.1f KB\", $size/1024}")
    else
      hr="${size} B"
    fi
    printf "    %-32s %8s\n" "$name" "$hr"
  done
  echo ""
fi

# ── Upload to R2 (via Worker) ─────────────────────────────────────

if [[ $UPLOAD -eq 1 ]]; then
  if [[ -z "${CLI_UPLOAD_KEY:-}" ]]; then
    echo "  error: CLI_UPLOAD_KEY environment variable is required for upload"
    echo "  Set it: export CLI_UPLOAD_KEY=<your-key>"
    exit 1
  fi

  for target in "${TARGETS[@]}"; do
    os=$(echo "$target" | awk '{print $1}')
    arch=$(echo "$target" | awk '{print $2}')
    output="storage-${os}-${arch}"
    [[ "$os" == "windows" ]] && output="${output}.exe"

    if [[ ! -f "$BUILD_DIR/$output" ]]; then
      echo "  skip   $output (not found)"
      continue
    fi

    echo -n "  upload $output ... "
    resp=$(curl -sS -X PUT "${UPLOAD_URL}/${VERSION}/${output}" \
      -H "X-Admin-Key: ${CLI_UPLOAD_KEY}" \
      -H "Content-Type: application/octet-stream" \
      --data-binary @"$BUILD_DIR/$output" 2>&1)

    if echo "$resp" | grep -q '"ok":true'; then
      echo "ok"
    else
      echo "FAILED: $resp"
    fi
  done

  echo ""
  echo "  Released v$VERSION"
  echo ""
  echo "  Download:"
  echo "    https://storage.liteio.dev/cli/releases/latest/storage-{os}-{arch}"
  echo ""
  echo "  Install:"
  echo "    curl -fsSL https://storage.liteio.dev/cli/install.sh | sh"
  echo ""
fi
