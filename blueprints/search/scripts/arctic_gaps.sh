#!/usr/bin/env bash
# arctic_gaps.sh — Monitor arctic publish progress and detect gaps.
# Designed to run from cron every 30 minutes.
#
# Reads stats.csv to find missing months, checks failures.csv for recent
# errors, and logs a summary. Exits 0 always (cron-safe).
#
# Logs to $HOME/log/arctic_gaps.log
# Writes $HOME/log/arctic_gaps_status.json for machine parsing.

set -uo pipefail

REPO_ROOT="${MIZU_ARCTIC_REPO_ROOT:-$HOME/data/arctic/repo}"
STATS="$REPO_ROOT/stats.csv"
FAILURES="$REPO_ROOT/failures.csv"
LOG_DIR="$HOME/log"
LOG="$LOG_DIR/arctic_gaps.log"
STATUS_JSON="$LOG_DIR/arctic_gaps_status.json"
mkdir -p "$LOG_DIR"

log() {
    local level=$1; shift
    local msg="[$(date '+%Y-%m-%d %H:%M:%S')] [$level] $*"
    echo "$msg" >> "$LOG"
}

# ── Compute expected vs committed months ──────────────────────────────────
# Arctic Shift covers 2005-12 to current month.
FROM_YEAR=2005
FROM_MONTH=12
TO_YEAR=$(date -u '+%Y')
TO_MONTH=$(date -u '+%-m')

# Parse stats.csv and collect committed year-month/type keys.
declare -A committed
total_committed=0
total_rows=0
total_bytes=0

if [[ -f "$STATS" ]]; then
    # Skip header, read CSV fields.
    while IFS=, read -r year month type shards count size_bytes rest; do
        [[ "$year" == "year" ]] && continue  # skip header
        key="${year}-$(printf '%02d' "$month")/${type}"
        committed[$key]=1
        total_committed=$((total_committed + 1))
        total_rows=$((total_rows + count))
        total_bytes=$((total_bytes + size_bytes))
    done < "$STATS"
fi

# Build expected set and find gaps.
gap_count=0
gap_list=""
y=$FROM_YEAR
m=$FROM_MONTH
while (( y < TO_YEAR || (y == TO_YEAR && m <= TO_MONTH) )); do
    ym=$(printf '%04d-%02d' "$y" "$m")
    for type in comments submissions; do
        key="${ym}/${type}"
        if [[ -z "${committed[$key]:-}" ]]; then
            gap_count=$((gap_count + 1))
            gap_list="${gap_list}  ${ym} ${type}\n"
        fi
    done
    m=$((m + 1))
    if (( m > 12 )); then
        m=1
        y=$((y + 1))
    fi
done

# ── Compute expected total ────────────────────────────────────────────────
expected=0
ey=$FROM_YEAR
em=$FROM_MONTH
while (( ey < TO_YEAR || (ey == TO_YEAR && em <= TO_MONTH) )); do
    expected=$((expected + 2))  # comments + submissions
    em=$((em + 1))
    if (( em > 12 )); then
        em=1
        ey=$((ey + 1))
    fi
done

pct=0
if (( expected > 0 )); then
    pct=$((total_committed * 100 / expected))
fi

# ── Recent failures ───────────────────────────────────────────────────────
recent_failures=0
recent_failure_summary=""
if [[ -f "$FAILURES" ]]; then
    # Count failures in last 24 hours.
    cutoff=$(date -u -d "24 hours ago" '+%Y-%m-%dT%H:%M:%S' 2>/dev/null || \
             date -u -v-24H '+%Y-%m-%dT%H:%M:%S' 2>/dev/null || echo "")
    if [[ -n "$cutoff" ]]; then
        while IFS=, read -r year month type stage error failed_at; do
            [[ "$year" == "year" ]] && continue
            if [[ "$failed_at" > "$cutoff" ]]; then
                recent_failures=$((recent_failures + 1))
                ym=$(printf '%04d-%02d' "$year" "$month")
                recent_failure_summary="${recent_failure_summary}  ${ym} ${type} [${stage}]: ${error}\n"
            fi
        done < "$FAILURES"
    fi
fi

# ── Pipeline process status ───────────────────────────────────────────────
pipeline_running=false
pipeline_pid=""
if pgrep -f "arctic publish" > /dev/null 2>&1; then
    pipeline_running=true
    pipeline_pid=$(pgrep -f "arctic publish" | head -1)
fi

# ── Log summary ───────────────────────────────────────────────────────────
log "INFO" "=== Arctic gaps check ==="
log "INFO" "Progress: ${total_committed}/${expected} committed (${pct}%)"
log "INFO" "Gaps: ${gap_count} missing month×type pairs"
log "INFO" "Recent failures (24h): ${recent_failures}"
log "INFO" "Pipeline running: ${pipeline_running} (pid: ${pipeline_pid:-N/A})"

if (( gap_count > 0 && gap_count <= 50 )); then
    log "INFO" "Gap list:"
    echo -e "$gap_list" | while read -r line; do
        [[ -n "$line" ]] && log "INFO" "  $line"
    done
elif (( gap_count > 50 )); then
    # Too many gaps to list individually — show year ranges.
    log "WARN" "Too many gaps ($gap_count) — showing year summary:"
    echo -e "$gap_list" | awk '{print substr($1,1,4)}' | sort -u | while read -r year; do
        count=$(echo -e "$gap_list" | grep "^  ${year}-" | wc -l)
        log "WARN" "  $year: $count missing"
    done
fi

if (( recent_failures > 0 )); then
    log "WARN" "Recent failure details:"
    echo -e "$recent_failure_summary" | head -20 | while read -r line; do
        [[ -n "$line" ]] && log "WARN" "  $line"
    done
fi

# ── Write status JSON ────────────────────────────────────────────────────
size_gb=$(echo "scale=1; $total_bytes / 1073741824" | bc 2>/dev/null || echo "0")
cat > "$STATUS_JSON" << EOF
{
  "checked_at": "$(date -u '+%Y-%m-%dT%H:%M:%SZ')",
  "committed": $total_committed,
  "expected": $expected,
  "pct": $pct,
  "gaps": $gap_count,
  "recent_failures_24h": $recent_failures,
  "total_rows": $total_rows,
  "total_size_gb": $size_gb,
  "pipeline_running": $pipeline_running,
  "pipeline_pid": "${pipeline_pid:-null}"
}
EOF

log "INFO" "Status written to $STATUS_JSON"
