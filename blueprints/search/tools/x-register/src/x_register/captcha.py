"""Optional Arkose FunCaptcha solver integration.

Supported services: capsolver (default), 2captcha.
If no solver key is configured, raises ArkoseRequired.

X email signup public key: 2CB16598-CB82-4CF7-B332-5990DB66F3AB
"""

from __future__ import annotations

import time

import httpx

ARKOSE_PUBLIC_KEY = "2CB16598-CB82-4CF7-B332-5990DB66F3AB"
ARKOSE_PAGE_URL = "https://x.com"
POLL_INTERVAL = 5
POLL_TIMEOUT = 120


class ArkoseRequired(Exception):
    """Raised when Arkose challenge appears but no solver is configured."""


class CaptchaError(Exception):
    """Solver API returned an error."""


def solve_arkose(
    blob: str,
    api_key: str,
    service: str = "capsolver",
    verbose: bool = False,
) -> str:
    """Submit Arkose challenge to solver service and return the token.

    Args:
        blob: The base64 data blob from the ArkoseEmail subtask URL's `data=` param.
        api_key: Solver service API key.
        service: "capsolver" or "2captcha".
        verbose: Log progress.
    """
    def log(msg: str) -> None:
        if verbose:
            ts = time.strftime("%H:%M:%S")
            print(f"[{ts}] [captcha] {msg}", flush=True)

    if service == "capsolver":
        return _solve_capsolver(blob, api_key, log)
    elif service == "2captcha":
        return _solve_2captcha(blob, api_key, log)
    else:
        raise CaptchaError(f"Unknown solver service: {service!r}")


def _solve_capsolver(blob: str, api_key: str, log) -> str:
    client = httpx.Client(timeout=30)
    try:
        log("submitting to CapSolver...")
        resp = client.post(
            "https://api.capsolver.com/createTask",
            json={
                "clientKey": api_key,
                "task": {
                    "type": "FunCaptchaTask",
                    "websiteURL": ARKOSE_PAGE_URL,
                    "websitePublicKey": ARKOSE_PUBLIC_KEY,
                    "data": blob,
                },
            },
        )
        resp.raise_for_status()
        data = resp.json()
        if data.get("errorId"):
            raise CaptchaError(f"CapSolver error: {data.get('errorDescription')}")
        task_id = data["taskId"]
        log(f"task_id={task_id}, polling...")

        deadline = time.time() + POLL_TIMEOUT
        while time.time() < deadline:
            time.sleep(POLL_INTERVAL)
            result = client.post(
                "https://api.capsolver.com/getTaskResult",
                json={"clientKey": api_key, "taskId": task_id},
            )
            result.raise_for_status()
            rdata = result.json()
            status = rdata.get("status")
            log(f"  status={status}")
            if status == "ready":
                token = rdata["solution"]["token"]
                log(f"  solved! token={token[:20]}...")
                return token
            if status == "failed":
                raise CaptchaError(f"CapSolver task failed: {rdata.get('errorDescription')}")

        raise CaptchaError("CapSolver timed out")
    finally:
        client.close()


def _solve_2captcha(blob: str, api_key: str, log) -> str:
    client = httpx.Client(timeout=30)
    try:
        log("submitting to 2captcha...")
        resp = client.post(
            "https://2captcha.com/in.php",
            data={
                "key": api_key,
                "method": "funcaptcha",
                "publickey": ARKOSE_PUBLIC_KEY,
                "pageurl": ARKOSE_PAGE_URL,
                "data[blob]": blob,
                "json": "1",
            },
        )
        resp.raise_for_status()
        data = resp.json()
        if data.get("status") != 1:
            raise CaptchaError(f"2captcha submit failed: {data.get('request')}")
        task_id = data["request"]
        log(f"task_id={task_id}, polling...")

        deadline = time.time() + POLL_TIMEOUT
        while time.time() < deadline:
            time.sleep(POLL_INTERVAL)
            result = client.get(
                "https://2captcha.com/res.php",
                params={"key": api_key, "action": "get", "id": task_id, "json": "1"},
            )
            result.raise_for_status()
            rdata = result.json()
            log(f"  status={rdata.get('status')} request={str(rdata.get('request',''))[:30]}")
            if rdata.get("status") == 1:
                token = rdata["request"]
                log(f"  solved! token={token[:20]}...")
                return token
            if rdata.get("request") == "ERROR_CAPTCHA_UNSOLVABLE":
                raise CaptchaError("2captcha: unsolvable")
        raise CaptchaError("2captcha timed out")
    finally:
        client.close()
