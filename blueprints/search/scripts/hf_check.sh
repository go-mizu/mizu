#!/usr/bin/env bash
# hf_check.sh — Verify recent commits exist on HuggingFace repos.
# Designed to run from cron every 30 minutes.
#
# Checks:
#   open-index/draft   — CC publish watcher; expects a commit within 3 hours
#   open-index/arctic  — Arctic backward sweep; expects a commit within 8 hours
#
# Exits 0 always (cron-safe). Logs to $HOME/log/hf_check.log.
# Writes $HOME/log/hf_check_status.json for easy machine parsing.

set -uo pipefail

LOG_DIR="$HOME/log"
LOG="$LOG_DIR/hf_check.log"
STATUS_JSON="$LOG_DIR/hf_check_status.json"
mkdir -p "$LOG_DIR"

HF_TOKEN=${HF_TOKEN:-$(cat "$HOME/.hf_token" 2>/dev/null || echo "")}
NOW=$(date +%s)

log() {
    local level=$1; shift
    local msg="[$(date '+%Y-%m-%d %H:%M:%S')] [$level] $*"
    echo "$msg" >> "$LOG"
}

# Returns the Unix timestamp of the latest commit on a HF dataset repo's main branch.
# Exits non-zero on API error.
latest_commit_ts() {
    local repo=$1
    local url="https://huggingface.co/api/datasets/${repo}/commits/main?limit=1"
    local resp
    resp=$(curl -sf --max-time 15 \
        -H "Authorization: Bearer $HF_TOKEN" \
        "$url") || { echo 0; return 1; }
    # Response: [{"id":..., "date":"2026-03-21T03:41:08.000Z", ...}]
    echo "$resp" | python3 -c "
import sys, json, datetime
data = json.load(sys.stdin)
if not data: sys.exit(1)
d = data[0]['date'].replace('Z','+00:00')
dt = datetime.datetime.fromisoformat(d)
print(int(dt.timestamp()))
" 2>/dev/null || echo 0
}

check_repo() {
    local repo=$1
    local max_stale_s=$2   # alert if no commit within this many seconds
    local label=$3

    local ts
    ts=$(latest_commit_ts "$repo")
    if [[ "$ts" == "0" ]]; then
        log "WARN" "$label ($repo): could not fetch latest commit"
        echo "\"$repo\": {\"status\": \"error\", \"message\": \"api_failure\"}"
        return
    fi

    local age=$(( NOW - ts ))
    local age_min=$(( age / 60 ))
    local commit_time
    commit_time=$(date -d "@$ts" '+%Y-%m-%d %H:%M:%S UTC' 2>/dev/null || date -r "$ts" '+%Y-%m-%d %H:%M:%S' 2>/dev/null || echo "unknown")

    if (( age > max_stale_s )); then
        local stale_h=$(( max_stale_s / 3600 ))
        log "STALE" "$label ($repo): last commit ${age_min}m ago ($commit_time) — exceeded ${stale_h}h threshold"
        echo "\"$repo\": {\"status\": \"stale\", \"age_min\": $age_min, \"last_commit\": \"$commit_time\", \"threshold_h\": $stale_h}"
    else
        log "OK" "$label ($repo): last commit ${age_min}m ago ($commit_time)"
        echo "\"$repo\": {\"status\": \"ok\", \"age_min\": $age_min, \"last_commit\": \"$commit_time\"}"
    fi
}

log "INFO" "=== HF commit check ==="

# Collect results
draft_result=$(check_repo "open-index/draft"  10800 "CC watcher")    # 3h
arctic_result=$(check_repo "open-index/arctic" 28800 "Arctic backward") # 8h

# Write status JSON
cat > "$STATUS_JSON" <<EOF
{
  "checked_at": "$(date -u '+%Y-%m-%dT%H:%M:%SZ')",
  $draft_result,
  $arctic_result
}
EOF

log "INFO" "Status written to $STATUS_JSON"
