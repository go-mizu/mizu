#!/usr/bin/env python3
# /// script
# requires-python = ">=3.10"
# dependencies = ["huggingface_hub[hf_transfer]", "hf-xet"]
# ///
"""
Minimal HuggingFace commit helper used by the search CLI.
Reads a JSON payload from stdin, performs the commit, and prints the commit URL.

Input JSON (stdin):
{
  "token":    "hf_...",
  "repo_id":  "open-index/draft",
  "message":  "Publish shard ...",
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

from huggingface_hub import HfApi, CommitOperationAdd, CommitOperationDelete
from huggingface_hub.errors import HfHubHTTPError


def main() -> None:
    payload = json.load(sys.stdin)
    token    = payload["token"]
    repo_id  = payload["repo_id"]
    message  = payload["message"]
    ops_raw  = payload["ops"]

    api = HfApi(token=token)

    operations = []
    for op in ops_raw:
        if op.get("delete", False):
            operations.append(CommitOperationDelete(path_in_repo=op["path_in_repo"]))
            continue
        local = op["local_path"]
        repo_path = op["path_in_repo"]
        if not os.path.isfile(local):
            print(f"[hf_commit.py] WARNING: file not found: {local}", file=sys.stderr)
            continue
        operations.append(CommitOperationAdd(path_in_repo=repo_path, path_or_fileobj=local))

    if not operations:
        print(json.dumps({"commit_url": "", "error": "no files to commit"}))
        sys.exit(1)

    try:
        commit_info = api.create_commit(
            repo_id=repo_id,
            repo_type="dataset",
            operations=operations,
            commit_message=message,
        )
        print(json.dumps({"commit_url": commit_info.commit_url}))
    except HfHubHTTPError as e:
        retry_after = 0
        if getattr(e, "response", None) is not None and e.response.status_code == 429:
            ra = e.response.headers.get("Retry-After", "")
            try:
                retry_after = int(ra)
            except (ValueError, TypeError):
                pass
        print(json.dumps({"error": str(e), "retry_after": retry_after}))
        sys.exit(1)
    except Exception as e:
        print(json.dumps({"error": str(e)}))
        sys.exit(1)


if __name__ == "__main__":
    main()
