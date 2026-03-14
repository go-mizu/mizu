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

    commit_info = api.create_commit(
        repo_id=repo_id,
        repo_type="dataset",
        operations=operations,
        commit_message=message,
    )
    print(json.dumps({"commit_url": commit_info.commit_url}))


if __name__ == "__main__":
    main()
