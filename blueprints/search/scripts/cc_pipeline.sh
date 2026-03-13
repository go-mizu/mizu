#!/usr/bin/env bash
# CC WARC pipeline: download → pack → delete raw WARC → export → publish
# Usage: cc_pipeline.sh <start_idx> <end_idx>
# Example: cc_pipeline.sh 11 90
set -euo pipefail

START=${1:-11}
END=${2:-90}

SEARCH=/root/bin/search
WARC_DIR=/root/data/common-crawl/CC-MAIN-2026-08/warc
HF_TOKEN=${HF_TOKEN:-}

if [ -z "$HF_TOKEN" ]; then
  echo "ERROR: HF_TOKEN not set"
  exit 1
fi

echo "=== CC Pipeline: files $START to $END ==="
echo ""

for i in $(seq $START $END); do
  FILE=$(printf "%05d" $i)
  echo "--- [$i/$END] $FILE ---"

  # 1. Download raw WARC (skip if md.warc.gz already exists)
  WARC_MD=/root/data/common-crawl/CC-MAIN-2026-08/warc_md/${FILE}.md.warc.gz
  if [ -f "$WARC_MD" ]; then
    echo "  pack: $FILE already packed, skipping download+pack"
  else
    echo "  download: $FILE"
    $SEARCH cc warc download --file $i

    # 2. Pack: WARC → md.warc.gz
    echo "  pack: $FILE"
    $SEARCH cc warc pack --file $i

    # 3. Delete raw WARC to free disk
    RAW=$(ls $WARC_DIR/*-$(printf "%05d" $i).warc.gz 2>/dev/null || true)
    if [ -n "$RAW" ]; then
      echo "  cleanup: removing $RAW"
      rm -f "$RAW"
    fi
  fi

  # 4. Export → parquet
  echo "  export: $FILE"
  $SEARCH cc warc export --file $i

  # 5. Publish to HuggingFace
  echo "  publish: $FILE"
  $SEARCH cc publish --file $i

  echo "  done: $FILE"
  echo ""
done

echo "=== Pipeline complete: files $START to $END ==="
