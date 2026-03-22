#!/usr/bin/env bash
# watchdog.sh — Check that key screen sessions are alive and restart if dead.
# Designed to run from cron every 5 minutes.
#
# Sessions monitored:
#   cc_watcher    — HF commit watcher (cc_watch.sh, CC-MAIN-2026-12)
#   cc_sched      — CC gap scheduler (cc_sched.sh, CC-MAIN-2026-12)
#   arctic_backward — Arctic submissions backward sweep
#
# Logs to $HOME/log/watchdog.log

set -uo pipefail

CRAWL=${CRAWL:-CC-MAIN-2026-12}
LOG_DIR="$HOME/log"
LOG="$LOG_DIR/watchdog.log"
mkdir -p "$LOG_DIR"

log() {
    local msg="[$(date '+%Y-%m-%d %H:%M:%S')] [watchdog] $*"
    echo "$msg" >> "$LOG"
}

restart_count=0

# ── cc_watcher ──────────────────────────────────────────────────────────────
watcher_alive() {
    # Screen session must exist AND have an active search process inside
    screen -ls | grep -q "\.cc_watcher" || return 1
    pgrep -f "search.*cc publish --watch" > /dev/null 2>&1
}

if watcher_alive; then
    log "cc_watcher OK (pid=$(pgrep -f 'search.*cc publish --watch' | head -1))"
else
    log "cc_watcher DEAD — restarting"
    screen -S cc_watcher -X quit 2>/dev/null || true
    sleep 1
    HF_TOKEN=$(cat "$HOME/.hf_token")
    screen -dmS cc_watcher bash -c \
        "CRAWL=$CRAWL HF_TOKEN=$HF_TOKEN bash $HOME/scripts/cc_watch.sh; exec bash"
    log "cc_watcher restarted"
    restart_count=$(( restart_count + 1 ))
fi

# ── cc_sched ─────────────────────────────────────────────────────────────────
STATS="$HOME/data/common-crawl/$CRAWL/export/repo/stats.csv"
committed_count() {
    grep -c "^${CRAWL}," "$STATS" 2>/dev/null || echo 0
}

sched_alive() {
    screen -ls | grep -q "\.cc_sched" || return 1
    pgrep -f "search.*cc publish.*(--schedule|--pipeline)" > /dev/null 2>&1
}

sched_done() {
    # Consider done if >= 9800 shards committed (allows for intentional skips)
    local n
    n=$(committed_count)
    (( n >= 9800 ))
}

if sched_done; then
    log "cc_sched DONE ($(committed_count)/10000 committed) — not restarting"
elif sched_alive; then
    log "cc_sched OK"
else
    log "cc_sched DEAD ($(committed_count)/10000 committed) — restarting"
    screen -S cc_sched -X quit 2>/dev/null || true
    sleep 1
    HF_TOKEN=$(cat "$HOME/.hf_token")
    screen -dmS cc_sched bash -c \
        "CRAWL=$CRAWL HF_TOKEN=$HF_TOKEN bash $HOME/scripts/cc_sched.sh; exec bash"
    log "cc_sched restarted"
    restart_count=$(( restart_count + 1 ))
fi

# ── arctic_backward ───────────────────────────────────────────────────────────
arctic_alive() {
    screen -ls | grep -q "\.arctic_backward" || return 1
    pgrep -f "(arctic publish|arctic_backward)" > /dev/null 2>&1
}

arctic_done() {
    # Done if script wrote "stopping" to restart log
    grep -q "stopping" "$HOME/log/arctic_restart.log" 2>/dev/null && \
        tail -3 "$HOME/log/arctic_restart.log" | grep -q "stopping"
}

if arctic_done; then
    log "arctic_backward DONE (reached stop point) — not restarting"
elif arctic_alive; then
    log "arctic_backward OK (pid=$(pgrep -f 'arctic publish' | head -1))"
else
    log "arctic_backward DEAD — restarting"
    screen -S arctic_backward -X quit 2>/dev/null || true
    sleep 1
    # Try repo-tracked script first, fall back to $HOME/bin copy.
    ARCTIC_SCRIPT="$HOME/scripts/arctic_backward.sh"
    if [[ ! -f "$ARCTIC_SCRIPT" ]]; then
        ARCTIC_SCRIPT="$HOME/bin/arctic_backward.sh"
    fi
    HF_TOKEN=$(cat "$HOME/.hf_token")
    screen -dmS arctic_backward bash -c \
        "HF_TOKEN=$HF_TOKEN bash $ARCTIC_SCRIPT; exec bash"
    log "arctic_backward restarted (script: $ARCTIC_SCRIPT)"
    restart_count=$(( restart_count + 1 ))
fi

# ── Summary ──────────────────────────────────────────────────────────────────
if (( restart_count > 0 )); then
    log "SUMMARY: restarted $restart_count session(s)"
else
    log "SUMMARY: all sessions healthy"
fi
