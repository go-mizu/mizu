#!/usr/bin/env bash
# arctic_backward.sh — Run arctic publish backward sweep with auto-restart.
# Used by watchdog.sh to keep the arctic pipeline running.
#
# Processes ALL months from 2005-12 to current; stats.csv-committed months
# are automatically skipped. The restart loop handles OOM kills, stall
# exits (exit code 75), and transient failures.
#
# Key environment tuning for bundle torrent (≤2023-12) on limited-RAM servers:
#   TORRENT_STORAGE_DEFAULT_FILE_IO=classic  — avoid mmap memory pressure
#   GOMEMLIMIT=6GiB                           — aggressive Go GC
#
# Logs: $HOME/log/arctic_backward_N.log  (per-attempt)
#        $HOME/log/arctic_restart.log     (restart history)

set -uo pipefail

LOG_DIR="$HOME/log"
mkdir -p "$LOG_DIR"

# Load HF token
if [[ -f "$HOME/.hf_token" ]]; then
    HF_TOKEN=$(cat "$HOME/.hf_token")
    export HF_TOKEN
fi

if [[ -z "${HF_TOKEN:-}" ]]; then
    echo "ERROR: HF_TOKEN not set and $HOME/.hf_token not found" >&2
    exit 1
fi

# Use classic file IO to avoid mmap memory issues with the large bundle torrent.
# The bundle torrent (~2700 files) uses mmap by default; on servers with ≤12 GB
# RAM, the mmap address space and page cache pressure causes OOM when combined
# with the 2 GB zstd decoder window.
export TORRENT_STORAGE_DEFAULT_FILE_IO=classic

# Limit Go memory to leave headroom for C allocations (DuckDB, torrent client).
# On an 11 GB server: 6 GiB Go + ~3 GB C/OS = ~9 GB total, leaving 2 GB free.
export GOMEMLIMIT=${GOMEMLIMIT:-6GiB}

# Xet upload tuning: limit concurrency to reduce memory per HF commit.
export HF_XET_FIXED_UPLOAD_CONCURRENCY=${HF_XET_FIXED_UPLOAD_CONCURRENCY:-4}

# Pipeline config
FROM=${ARCTIC_FROM:-2005-12}
TYPE=${ARCTIC_TYPE:-submissions}
STALL=${ARCTIC_STALL:-90m}
MAX_ATTEMPTS=${ARCTIC_MAX_ATTEMPTS:-100}

echo "[$(date)] arctic_backward starting: --from $FROM --type $TYPE --max-commit-stall $STALL"

n=0
while true; do
    n=$((n + 1))
    if (( n > MAX_ATTEMPTS )); then
        echo "[$(date)] arctic_backward reached max attempts ($MAX_ATTEMPTS) — stopping" >> "$LOG_DIR/arctic_restart.log"
        break
    fi

    LOG="$LOG_DIR/arctic_backward_${n}.log"
    echo "[$(date)] arctic_backward attempt $n starting (--from $FROM --type $TYPE)" >> "$LOG_DIR/arctic_restart.log"

    "$HOME/bin/search-linux-noble" arctic publish \
        --from "$FROM" \
        --type "$TYPE" \
        --max-commit-stall "$STALL" \
        2>&1 | tee "$LOG"

    rc=$?
    echo "[$(date)] arctic_backward exited $rc (attempt $n)" >> "$LOG_DIR/arctic_restart.log"

    if [ $rc -eq 0 ]; then
        echo "[$(date)] arctic_backward done — stopping" >> "$LOG_DIR/arctic_restart.log"
        break
    fi

    # Exit code 75 (EX_TEMPFAIL) = stall detected, restart is expected.
    # Any other non-zero exit is crash/OOM — also restart after cooldown.
    if [ $rc -eq 75 ]; then
        echo "[$(date)] stall exit — restarting in 30s" >> "$LOG_DIR/arctic_restart.log"
        sleep 30
    else
        echo "[$(date)] crash (rc=$rc) — restarting in 60s" >> "$LOG_DIR/arctic_restart.log"
        sleep 60
    fi
done
