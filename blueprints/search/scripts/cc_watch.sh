#!/usr/bin/env bash
# cc_watch.sh — Keep the HF publish watcher running, restarting on failure.
#
# Run inside a dedicated screen session on each server:
#   server1: screen -dmS g_watch bash -c "bash ~/scripts/cc_watch.sh; exec bash"
#   server2: screen -dmS g_watch bash -c "bash ~/scripts/cc_watch.sh; exec bash"
#
# Or via Makefile:
#   make deploy-cc-watch          # deploy script + start on server1
#   make deploy-cc-watch SERVER=2 # deploy script + start on server2
#
# HuggingFace rate limit: 128 commits/hour per token across ALL repos and servers.
# With 2 servers + other repos also committing, keep well under that budget.
# COMMIT_INTERVAL defaults to 180s → 20/hour per server, 40/hour total, leaving
# 88 commits/hour headroom for arctic, goodreads, and any other HF repos.
# Lower only if you're not running other HF publish jobs on the same token.
#
# Environment:
#   CRAWL           — crawl ID (default: CC-MAIN-2026-08)
#   HF_TOKEN        — Hugging Face token (required; also checked in ~/.hf_token)
#   SEARCH_BIN      — path to search binary (default: auto-detected)
#   RESTART_DELAY   — seconds to wait before restarting after failure (default: 10)
#   COMMIT_INTERVAL — minimum seconds between HF commits (default: 180)

set -uo pipefail

CRAWL=${CRAWL:-CC-MAIN-2026-12}
RESTART_DELAY=${RESTART_DELAY:-10}
COMMIT_INTERVAL=${COMMIT_INTERVAL:-180}
LOG_DIR="$HOME/log"
LOG="$LOG_DIR/cc_watch.log"
mkdir -p "$LOG_DIR"

# Resolve HF_TOKEN from env or ~/.hf_token
if [[ -z "${HF_TOKEN:-}" ]]; then
    if [[ -f "$HOME/.hf_token" ]]; then
        HF_TOKEN=$(cat "$HOME/.hf_token")
    else
        echo "ERROR: HF_TOKEN not set and ~/.hf_token not found" >&2
        echo "  Write your token: echo 'hf_...' > ~/.hf_token" >&2
        exit 1
    fi
fi
export HF_TOKEN

# Auto-detect search binary
if [[ -n "${SEARCH_BIN:-}" ]]; then
    SEARCH="$SEARCH_BIN"
elif command -v search &>/dev/null; then
    SEARCH="search"
elif [[ -x "$HOME/bin/search" ]]; then
    SEARCH="$HOME/bin/search"
else
    echo "ERROR: search binary not found; set SEARCH_BIN" >&2
    exit 1
fi

export PATH="$HOME/bin:$PATH"

log() {
    local msg="[$(date '+%Y-%m-%d %H:%M:%S')] $*"
    echo "$msg"
    echo "$msg" >> "$LOG"
}

log "=== CC Watch starting ==="
log "  Crawl:          $CRAWL"
log "  Binary:         $SEARCH"
log "  Restart delay:  ${RESTART_DELAY}s"
log "  Commit interval: ${COMMIT_INTERVAL}s (≤$(( 3600 / COMMIT_INTERVAL ))/hour; HF limit 128/hour across all servers)"
log ""

ATTEMPT=0
while true; do
    ATTEMPT=$(( ATTEMPT + 1 ))
    log "  [watch] attempt $ATTEMPT — launching: $SEARCH cc publish --watch --commit-interval ${COMMIT_INTERVAL}"

    "$SEARCH" cc publish --watch --commit-interval "$COMMIT_INTERVAL" 2>&1 | tee -a "$LOG"
    EXIT_CODE=${PIPESTATUS[0]}

    log "  [watch] exited with code $EXIT_CODE"

    if (( EXIT_CODE == 0 )); then
        log "  [watch] clean exit — restarting in ${RESTART_DELAY}s..."
    else
        log "  [watch] abnormal exit ($EXIT_CODE) — restarting in ${RESTART_DELAY}s..."
    fi

    sleep "$RESTART_DELAY"
    log ""
done
