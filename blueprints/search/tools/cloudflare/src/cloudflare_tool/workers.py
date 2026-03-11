"""Worker operations: deploy/tail via wrangler subprocess, invoke via httpx."""
from __future__ import annotations

import os
import re
import subprocess
import sys
from typing import Any

import httpx


def _wrangler_env(account_id: str, token: str) -> dict[str, str]:
    """Build environment variables for wrangler subprocess."""
    env = os.environ.copy()
    env["CLOUDFLARE_ACCOUNT_ID"] = account_id
    env["CLOUDFLARE_API_TOKEN"] = token
    return env


def deploy(
    account_id: str,
    token: str,
    name: str,
    path: str,
    subdomain: str = "",
) -> str:
    """Deploy a Worker via wrangler. Returns the public URL."""
    cmd = ["npx", "wrangler", "deploy", "--name", name, path]
    result = subprocess.run(
        cmd,
        env=_wrangler_env(account_id, token),
        capture_output=True,
        text=True,
    )
    if result.returncode != 0:
        raise RuntimeError(
            f"wrangler deploy failed (exit {result.returncode}):\n{result.stderr}"
        )

    # Extract URL from wrangler output
    output = result.stdout + result.stderr
    m = re.search(r"https://[\w\-]+\.[\w\-]+\.workers\.dev", output)
    if m:
        return m.group(0)

    # Fallback: construct URL from name + subdomain
    if subdomain:
        return f"https://{name}.{subdomain}.workers.dev"
    return f"https://{name}.workers.dev"


def tail(account_id: str, token: str, name: str) -> None:
    """Stream Worker logs via wrangler tail (runs until interrupted)."""
    cmd = ["npx", "wrangler", "tail", name]
    try:
        subprocess.run(
            cmd,
            env=_wrangler_env(account_id, token),
        )
    except KeyboardInterrupt:
        pass


def invoke(
    url: str,
    method: str = "GET",
    path: str = "/",
    body: str = "",
    headers: dict[str, str] | None = None,
    timeout: float = 30.0,
) -> tuple[int, str]:
    """Send a request to a Worker URL. Returns (status_code, response_body)."""
    full_url = url.rstrip("/") + (path if path.startswith("/") else f"/{path}")
    req_headers = headers or {}

    with httpx.Client(timeout=timeout) as client:
        resp = client.request(
            method=method.upper(),
            url=full_url,
            headers=req_headers,
            content=body if body else None,
        )

    return resp.status_code, resp.text
