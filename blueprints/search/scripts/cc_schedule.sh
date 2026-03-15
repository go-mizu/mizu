#!/usr/bin/env bash
# cc_schedule.sh — Drive CC pipeline sessions to cover a file index range.
#
# Splits the range into chunks and keeps MAX_SESS screen sessions running
# until every chunk reaches DONE_PCT% committed shards. The watcher (g_watch)
# handles HF commits independently; this script only manages --pipeline jobs.
#
# Usage:
#   bash cc_schedule.sh <start> <end> [max_sessions] [chunk_size]
#
# Run in a dedicated screen session on each server with non-overlapping ranges:
#   server1: screen -dmS g_sched bash -c "bash ~/scripts/cc_schedule.sh 0 4999; exec bash"
#   server2: screen -dmS g_sched bash -c "bash ~/scripts/cc_schedule.sh 5000 9999; exec bash"
#
# Environment:
#   CRAWL        — crawl ID (default: CC-MAIN-2026-08)
#   DONE_PCT     — % of shards committed before chunk is considered done (default: 95)
#   STALL_ROUNDS — rounds with no new commits before a running session is killed (default: 15 = 30 min)
#   SEARCH_BIN   — path to search binary (default: auto-detected)

set -uo pipefail

START=${1:?"Usage: $0 <start> <end> [max_sessions] [chunk_size]"}
END=${2:?"Usage: $0 <start> <end> [max_sessions] [chunk_size]"}
MAX_SESS=${3:-6}
CHUNK=${4:-250}

CRAWL=${CRAWL:-CC-MAIN-2026-08}
DONE_PCT=${DONE_PCT:-95}
STALL_ROUNDS=${STALL_ROUNDS:-15}   # 15 rounds × 120s = 30 min with no progress → restart

STATS="$HOME/data/common-crawl/$CRAWL/export/repo/stats.csv"
LOG_DIR="$HOME/log"
LOG="$LOG_DIR/cc_schedule_${START}_${END}.log"
mkdir -p "$LOG_DIR"

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

log() {
    local msg="[$(date '+%Y-%m-%d %H:%M:%S')] $*"
    echo "$msg"
    echo "$msg" >> "$LOG"
}

# Print committed count for this crawl
committed_count() {
    grep -c "^${CRAWL}," "$STATS" 2>/dev/null || echo 0
}

# Build committed lookup file (tmpfile, refreshed each iteration)
COMM_TMP=$(mktemp)
trap 'rm -f "$COMM_TMP"' EXIT

refresh_committed() {
    grep "^${CRAWL}," "$STATS" 2>/dev/null | cut -d, -f2 | sort -n > "$COMM_TMP"
}

# Count how many indices in [s, e] are in the committed set
committed_in_range() {
    local s=$1 e=$2
    awk -v s="$s" -v e="$e" '$1>=s && $1<=e' "$COMM_TMP" | wc -l
}

# True if the pipeline process for a chunk is running
chunk_running() {
    local s=$1 e=$2
    pgrep -f "publish.*--file ${s}-${e}$" > /dev/null 2>&1
}

# Kill the pipeline process for a chunk (SIGKILL the search binary)
kill_chunk() {
    local s=$1 e=$2
    pgrep -f "publish.*--file ${s}-${e}$" | xargs -r kill -9 2>/dev/null || true
}

# Start a new screen session for a chunk
start_chunk() {
    local s=$1 e=$2
    local name="g${s}_${e}"
    screen -S "$name" -X quit 2>/dev/null || true
    sleep 0.2
    screen -dmS "$name" bash -c \
        "export PATH=$HOME/bin:\$PATH; $SEARCH cc publish --pipeline --cleanup --skip-errors --file ${s}-${e}; exec bash"
    log "  started $name  (files $s–$e)"
}

# Build chunk list
chunks=()
for ((s=START; s<=END; s+=CHUNK)); do
    e=$(( s + CHUNK - 1 ))
    (( e > END )) && e=$END
    chunks+=("$s:$e")
done

log "=== CC Schedule starting ==="
log "  Crawl:       $CRAWL"
log "  Range:       $START–$END"
log "  Chunks:      ${#chunks[@]}  (size=$CHUNK)"
log "  Sessions:    $MAX_SESS max"
log "  Done pct:    $DONE_PCT%"
log "  Stall kill:  after $STALL_ROUNDS rounds (~$(( STALL_ROUNDS * 2 ))m) with no new commits"
log "  Binary:      $SEARCH"
log ""

# Stall tracking — map from "s:e" → last committed count and stall round counter
declare -A stall_last_comm=()
declare -A stall_rounds=()

ROUND=0
while true; do
    ROUND=$(( ROUND + 1 ))
    refresh_committed

    n_running=0
    n_done=0
    n_todo=0
    running_names=()
    todo_chunks=()

    for chunk in "${chunks[@]}"; do
        s=${chunk%%:*}
        e=${chunk##*:}
        total=$(( e - s + 1 ))
        name="g${s}_${e}"
        key="${s}:${e}"

        if chunk_running "$s" "$e"; then
            n_comm=$(committed_in_range "$s" "$e")

            # Stall detection: track rounds with no new commits for this chunk.
            prev=${stall_last_comm[$key]:-0}
            if (( n_comm > prev )); then
                # Progress made — reset stall counter.
                stall_last_comm[$key]=$n_comm
                stall_rounds[$key]=0
            else
                stall_rounds[$key]=$(( ${stall_rounds[$key]:-0} + 1 ))
            fi

            sr=${stall_rounds[$key]:-0}
            if (( sr >= STALL_ROUNDS )); then
                log "  STALL: $name stuck at ${n_comm}/${total} for $sr rounds — killing and restarting"
                kill_chunk "$s" "$e"
                stall_rounds[$key]=0
                # Move to todo so it gets restarted this round.
                (( n_todo++ )) || true
                todo_chunks+=("$chunk")
            else
                (( n_running++ )) || true
                stall_label=""
                (( sr > 0 )) && stall_label="stall=${sr}"
                running_names+=("$name(${n_comm}/${total}${stall_label:+ $stall_label})")
            fi
        else
            # Reset stall state when not running.
            stall_rounds[$key]=0
            n_comm=$(committed_in_range "$s" "$e")
            pct=$(( n_comm * 100 / total ))
            if (( pct >= DONE_PCT )); then
                (( n_done++ )) || true
            else
                (( n_todo++ )) || true
                todo_chunks+=("$chunk")
            fi
        fi
    done

    total_committed=$(committed_count)
    slots=$(( MAX_SESS - n_running ))

    log "Round $ROUND | committed=$total_committed | done=${n_done}/${#chunks[@]} chunks | running=$n_running | todo=$n_todo | slots=$slots"
    if (( ${#running_names[@]} > 0 )); then
        log "  running: ${running_names[*]}"
    fi

    if (( n_running == 0 && n_todo == 0 )); then
        log ""
        log "=== All chunks complete for range $START–$END ==="
        log "Total committed: $(committed_count)"
        break
    fi

    # Fill free slots
    started=0
    for chunk in "${todo_chunks[@]}"; do
        (( slots <= 0 )) && break
        s=${chunk%%:*}
        e=${chunk##*:}
        start_chunk "$s" "$e"
        (( slots-- )) || true
        (( started++ )) || true
    done

    if (( started > 0 )); then
        log "  launched $started new session(s)"
    fi

    log ""
    sleep 120
done

log "Schedule finished. Run: $SEARCH cc publish --list"
