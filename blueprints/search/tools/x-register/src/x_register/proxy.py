"""Proxy management: public lists, good/bad JSON cache.

Ported and refactored from tools/jina-key/api_key.py.
"""

from __future__ import annotations

import json
import random
import socket
import threading
import time
import urllib.request
from pathlib import Path
from typing import NamedTuple

DATA_DIR = Path.home() / "data" / "x"
BAD_FILE = DATA_DIR / "bad_proxies.json"
GOOD_FILE = DATA_DIR / "good_proxies.json"

# Public free proxy sources
PROXY_SOURCES: list[tuple[str, str]] = [
    ("https://raw.githubusercontent.com/proxifly/free-proxy-list/main/proxies/protocols/https/data.txt", "https"),
    ("https://raw.githubusercontent.com/TheSpeedX/PROXY-List/master/http.txt", "http"),
    ("https://raw.githubusercontent.com/monosans/proxy-list/main/proxies/socks5.txt", "socks5"),
    ("https://raw.githubusercontent.com/hookzof/socks5_list/master/proxy.txt", "socks5"),
]

MAX_USES = 3          # retire a good proxy after this many successful uses
BAD_TTL = 86400       # seconds before bad proxy entry expires
PARALLEL_WORKERS = 16
MAX_CANDIDATES = 200


class Proxy(NamedTuple):
    scheme: str   # "http", "https", "socks5"
    host: str
    port: int

    @property
    def key(self) -> str:
        return f"{self.scheme}://{self.host}:{self.port}"

    def to_curl_proxies(self) -> dict[str, str]:
        """Return proxies dict for curl_cffi / requests."""
        url = self.key if self.scheme == "socks5" else f"http://{self.host}:{self.port}"
        return {"http": url, "https": url}


def _load_json(path: Path) -> dict:
    try:
        if path.exists():
            return json.loads(path.read_text())
    except Exception:
        pass
    return {}


def _save_json(path: Path, data: dict) -> None:
    try:
        DATA_DIR.mkdir(parents=True, exist_ok=True)
        path.write_text(json.dumps(data, indent=2))
    except Exception:
        pass


class ProxyManager:
    def __init__(self, extra_proxies: list[Proxy] | None = None, verbose: bool = False):
        self._extra = extra_proxies or []
        self._verbose = verbose
        self._lock = threading.Lock()
        self._fresh: list[Proxy] = []
        self._fresh_loaded = False

    def _log(self, msg: str) -> None:
        if self._verbose:
            ts = time.strftime("%H:%M:%S")
            print(f"[{ts}] [proxy] {msg}", flush=True)

    # ------------------------------------------------------------------
    # Cache read/write
    # ------------------------------------------------------------------

    def _load_bad(self) -> set[str]:
        data = _load_json(BAD_FILE)
        cutoff = time.time() - BAD_TTL
        return {k for k, v in data.items() if v.get("ts", 0) > cutoff}

    def mark_bad(self, proxy: Proxy, reason: str = "") -> None:
        with self._lock:
            data = _load_json(BAD_FILE)
            data[proxy.key] = {"ts": time.time(), "reason": reason[:80]}
            _save_json(BAD_FILE, data)
            self._log(f"bad: {proxy.key} ({reason[:40]})")

    def mark_good(self, proxy: Proxy) -> None:
        with self._lock:
            data = _load_json(GOOD_FILE)
            prev = data.get(proxy.key, {})
            data[proxy.key] = {"ts": time.time(), "uses": prev.get("uses", 0) + 1}
            _save_json(GOOD_FILE, data)
            self._log(f"good: {proxy.key}")

    def _load_good(self) -> list[Proxy]:
        data = _load_json(GOOD_FILE)
        result: list[Proxy] = []
        for key, meta in sorted(data.items(), key=lambda x: -x[1].get("ts", 0)):
            if meta.get("uses", 0) >= MAX_USES:
                continue
            try:
                scheme, rest = key.split("://", 1)
                host, port_s = rest.rsplit(":", 1)
                result.append(Proxy(scheme, host, int(port_s)))
            except Exception:
                pass
        return result

    # ------------------------------------------------------------------
    # Fetching
    # ------------------------------------------------------------------

    def _fetch_fresh(self) -> list[Proxy]:
        results: list[Proxy] = []
        for url, scheme in PROXY_SOURCES:
            self._log(f"fetching [{scheme}] {url}")
            try:
                resp = urllib.request.urlopen(url, timeout=10)
                lines = resp.read().decode("utf-8", errors="replace").strip().split("\n")
                before = len(results)
                for line in lines:
                    line = line.strip()
                    if not line or line.startswith("#"):
                        continue
                    if ":" in line:
                        parts = line.rsplit(":", 1)
                        try:
                            results.append(Proxy(scheme, parts[0], int(parts[1])))
                        except ValueError:
                            pass
                self._log(f"  -> {len(results) - before} proxies")
            except Exception as e:
                self._log(f"  source error: {e}")
        self._log(f"total fresh: {len(results)}")
        return results

    def _ensure_fresh(self) -> None:
        if not self._fresh_loaded:
            self._fresh = self._fetch_fresh()
            self._fresh_loaded = True

    # ------------------------------------------------------------------
    # Connectivity test
    # ------------------------------------------------------------------

    @staticmethod
    def _reachable(proxy: Proxy, timeout: float = 3.0) -> bool:
        try:
            s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
            s.settimeout(timeout)
            s.connect((proxy.host, proxy.port))
            s.close()
            return True
        except Exception:
            return False

    # ------------------------------------------------------------------
    # Public API
    # ------------------------------------------------------------------

    def iter_candidates(self):
        """Yield proxies: good cache first, then fresh list, then file proxies.

        Skips known-bad and unreachable proxies.
        Caller should call mark_good / mark_bad based on outcome.
        """
        bad_set = self._load_bad()
        good = [p for p in self._load_good() if p.key not in bad_set]
        self._log(f"good cache: {len(good)}, bad cache: {len(bad_set)}")

        # 1. Good proxies (sequential — few, high success rate)
        for proxy in good:
            self._log(f"[good] checking {proxy.key}")
            if self._reachable(proxy):
                yield proxy
            else:
                self.mark_bad(proxy, "unreachable")

        # 2. Fresh + file proxies (parallel reachability pre-check)
        self._ensure_fresh()
        good_keys = {p.key for p in good}
        candidates = [
            p for p in (self._fresh + self._extra)
            if p.key not in good_keys and p.key not in bad_set
        ]
        random.shuffle(candidates)
        candidates = candidates[:MAX_CANDIDATES]

        self._log(f"fresh candidates: {len(candidates)} ({PARALLEL_WORKERS} workers)")

        # Pre-check reachability in parallel and yield in order found
        ready: list[Proxy] = []
        done = threading.Event()
        lock = threading.Lock()

        def check(p: Proxy) -> None:
            if self._reachable(p):
                with lock:
                    ready.append(p)

        threads = [threading.Thread(target=check, args=(p,), daemon=True) for p in candidates]
        for i in range(0, len(threads), PARALLEL_WORKERS):
            batch = threads[i:i + PARALLEL_WORKERS]
            for t in batch:
                t.start()
            for t in batch:
                t.join(timeout=5)
            with lock:
                for p in ready:
                    yield p
                ready.clear()


def load_proxy_file(path: str) -> list[Proxy]:
    """Parse a proxy file: one per line, format scheme://host:port or host:port."""
    result: list[Proxy] = []
    try:
        for line in Path(path).read_text().splitlines():
            line = line.strip()
            if not line or line.startswith("#"):
                continue
            if "://" in line:
                scheme, rest = line.split("://", 1)
                host, port_s = rest.rsplit(":", 1)
            else:
                scheme = "http"
                host, port_s = line.rsplit(":", 1)
            result.append(Proxy(scheme.lower(), host, int(port_s)))
    except Exception:
        pass
    return result
