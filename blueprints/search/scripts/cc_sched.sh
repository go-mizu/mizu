#!/usr/bin/env bash
# cc_sched.sh — Run the CC publish scheduler with logging and auto-restart on failure.
#
# Run inside a dedicated screen session:
#   screen -dmS cc_sched bash -c "bash ~/scripts/cc_sched.sh; exec bash"
#
# Environment:
#   CRAWL           — crawl ID (default: CC-MAIN-2026-12)
#   HF_TOKEN        — HuggingFace token (required; also checked in ~/.hf_token)
#   SEARCH_BIN      — path to search binary (default: auto-detected)
#   RESTART_DELAY   — seconds to wait before restarting after failure (default: 30)
#   START           — first file index (default: 0)
#   END             — last file index (default: 9999)
#   MAX_SESSIONS    — max concurrent screen sessions (default: 0 = auto)
#   RAM_PER_SESSION — GB of RAM budgeted per session (default: 0 = use binary default)

set -uo pipefail

CRAWL=${CRAWL:-CC-MAIN-2026-12}
RESTART_DELAY=${RESTART_DELAY:-15}
START=${START:-0}
END=${END:-9999}
MAX_SESSIONS=${MAX_SESSIONS:-0}
RAM_PER_SESSION=${RAM_PER_SESSION:-0}
LOG_DIR="$HOME/log"
LOG="$LOG_DIR/cc_sched.log"
mkdir -p "$LOG_DIR"

# Resolve HF_TOKEN from env or ~/.hf_token
if [[ -z "${HF_TOKEN:-}" ]]; then
    if [[ -f "$HOME/.hf_token" ]]; then
        HF_TOKEN=$(cat "$HOME/.hf_token")
    else
        echo "ERROR: HF_TOKEN not set and ~/.hf_token not found" >&2
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

SCHED_FLAGS=""
if [[ "$MAX_SESSIONS" -gt 0 ]]; then
    SCHED_FLAGS="--max-sessions $MAX_SESSIONS"
fi
if [[ "$(echo "$RAM_PER_SESSION > 0" | bc -l 2>/dev/null)" == "1" ]]; then
    SCHED_FLAGS="$SCHED_FLAGS --ram-per-session $RAM_PER_SESSION"
fi

log "=== CC Scheduler starting ==="
log "  Crawl:         $CRAWL"
log "  Range:         ${START}–${END}"
log "  Max sessions:  ${MAX_SESSIONS} (0=auto)"
log "  Binary:        $SEARCH"
log "  Restart delay: ${RESTART_DELAY}s"
log ""

ATTEMPT=0
while true; do
    ATTEMPT=$(( ATTEMPT + 1 ))
    log "  [sched] attempt $ATTEMPT — launching scheduler for ${CRAWL} ${START}–${END}"

    "$SEARCH" cc publish --schedule \
        --crawl "$CRAWL" \
        --start "$START" --end "$END" \
        $SCHED_FLAGS \
        2>&1 | tee -a "$LOG"
    EXIT_CODE=${PIPESTATUS[0]}

    log "  [sched] exited with code $EXIT_CODE"

    if (( EXIT_CODE == 0 )); then
        log "  [sched] clean exit (all gaps filled) — done."
        break
    fi

    log "  [sched] abnormal exit ($EXIT_CODE) — restarting in ${RESTART_DELAY}s..."
    sleep "$RESTART_DELAY"
    log ""
done
