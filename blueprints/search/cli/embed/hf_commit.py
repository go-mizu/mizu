#!/usr/bin/env python3
# /// script
# requires-python = ">=3.10"
# dependencies = ["huggingface_hub>=1.7.1", "hf-xet>=1.4.2"]
# ///
"""
Minimal HuggingFace commit helper used by the search CLI.
Reads a JSON payload from stdin, performs the commit, and prints the commit URL.

Input JSON (stdin):
{
  "token":    "hf_...",
  "repo_id":  "open-index/draft",
  "message":  "Publish shard ...",
  "num_threads": 10,
  "ops": [
    {"local_path": "/abs/path/to/file.parquet", "path_in_repo": "data/CC-MAIN-2026-08/00000.parquet"},
    ...
  ]
}

Output JSON (stdout):
{"commit_url": "https://huggingface.co/datasets/open-index/draft/commit/abc123"}
"""

import json
import sys
import os
import logging
import time

# DO NOT set HF_XET_HIGH_PERFORMANCE here — it requires 64+ GB RAM and causes
# upload stalls on smaller machines. Xet env vars are set by the Go caller
# (cc_publish_hf.go) which tailors them to the server's resources.

from huggingface_hub import HfApi, CommitOperationAdd, CommitOperationDelete
from huggingface_hub.errors import HfHubHTTPError

# Suppress verbose httpx/urllib3 request logs — they flood stderr with
# every HTTP call and make it hard to read the actual progress.
# Keep huggingface_hub at WARNING so create_commit progress still shows.
logging.basicConfig(
    level=logging.WARNING,
    format="[hf_commit.py] %(asctime)s %(levelname)s %(name)s: %(message)s",
    datefmt="%H:%M:%S",
    stream=sys.stderr,
)
for _logger_name in ("httpx", "urllib3", "huggingface_hub", "hf_xet"):
    _lg = logging.getLogger(_logger_name)
    _lg.setLevel(logging.WARNING)
    _lg.handlers.clear()        # remove any pre-configured handlers
    _lg.propagate = True        # fall through to root (which is WARNING)


def main() -> None:
    payload = json.load(sys.stdin)
    token    = payload["token"]
    repo_id  = payload["repo_id"]
    message  = payload["message"]
    ops_raw  = payload["ops"]
    num_threads = payload.get("num_threads", 10)

    api = HfApi(token=token)

    operations = []
    total_size = 0
    skipped = 0
    for op in ops_raw:
        if op.get("delete", False):
            operations.append(CommitOperationDelete(path_in_repo=op["path_in_repo"]))
            continue
        local = op["local_path"]
        repo_path = op["path_in_repo"]
        if not os.path.isfile(local):
            print(f"[hf_commit.py] WARNING: file not found: {local}", file=sys.stderr)
            skipped += 1
            continue
        fsize = os.path.getsize(local)
        total_size += fsize
        print(f"[hf_commit.py] add: {repo_path} ({fsize / 1024 / 1024:.1f} MB)", file=sys.stderr)
        operations.append(CommitOperationAdd(path_in_repo=repo_path, path_or_fileobj=local))

    if skipped > 0:
        print(f"[hf_commit.py] WARNING: {skipped} file(s) missing", file=sys.stderr)
        data_ops = [o for o in operations if isinstance(o, CommitOperationAdd) and o.path_in_repo.endswith(".parquet")]
        if len(data_ops) == 0:
            message = f"Update charts/README (no data — {skipped} parquet(s) missing locally)"

    if not operations:
        print(json.dumps({"commit_url": "", "error": "no files to commit", "uploaded": 0}))
        sys.exit(1)

    print(f"[hf_commit.py] committing {len(operations)} ops ({total_size / 1024 / 1024:.1f} MB total) to {repo_id}", file=sys.stderr)
    t0 = time.monotonic()

    uploaded = sum(1 for o in operations if isinstance(o, CommitOperationAdd))
    try:
        commit_info = api.create_commit(
            repo_id=repo_id,
            repo_type="dataset",
            operations=operations,
            commit_message=message,
            num_threads=num_threads,
        )
        elapsed = time.monotonic() - t0
        print(f"[hf_commit.py] committed in {elapsed:.1f}s: {commit_info.commit_url}", file=sys.stderr)
        print(json.dumps({"commit_url": commit_info.commit_url, "uploaded": uploaded}))
    except HfHubHTTPError as e:
        elapsed = time.monotonic() - t0
        retry_after = 0
        if getattr(e, "response", None) is not None and e.response.status_code == 429:
            ra = e.response.headers.get("Retry-After", "")
            try:
                retry_after = int(ra)
            except (ValueError, TypeError):
                pass
        print(f"[hf_commit.py] HF error after {elapsed:.1f}s: {e}", file=sys.stderr)
        print(json.dumps({"error": str(e), "retry_after": retry_after}))
        sys.exit(1)
    except (OSError, ConnectionError) as e:
        elapsed = time.monotonic() - t0
        print(f"[hf_commit.py] network error after {elapsed:.1f}s: {e}", file=sys.stderr)
        # Signal retryable error — Go caller will retry the commit.
        print(json.dumps({"error": f"network: {e}"}))
        sys.exit(1)
    except Exception as e:
        elapsed = time.monotonic() - t0
        print(f"[hf_commit.py] error after {elapsed:.1f}s: {e}", file=sys.stderr)
        print(json.dumps({"error": str(e)}))
        sys.exit(1)


if __name__ == "__main__":
    main()
