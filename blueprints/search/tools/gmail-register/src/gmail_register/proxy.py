"""Proxy management: public lists, good/bad JSON cache.

Identical pattern to tools/x-register/src/x_register/proxy.py.
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

DATA_DIR = Path.home() / "data" / "gmail"
BAD_FILE = DATA_DIR / "bad_proxies.json"
GOOD_FILE = DATA_DIR / "good_proxies.json"

PROXY_SOURCES: list[tuple[str, str]] = [
    ("https://raw.githubusercontent.com/proxifly/free-proxy-list/main/proxies/protocols/https/data.txt", "https"),
    ("https://raw.githubusercontent.com/TheSpeedX/PROXY-List/master/http.txt", "http"),
    ("https://raw.githubusercontent.com/monosans/proxy-list/main/proxies/socks5.txt", "socks5"),
    ("https://raw.githubusercontent.com/hookzof/socks5_list/master/proxy.txt", "socks5"),
]

MAX_USES = 3
BAD_TTL = 86400
PARALLEL_WORKERS = 16
MAX_CANDIDATES = 200


class Proxy(NamedTuple):
    scheme: str
    host: str
    port: int

    @property
    def key(self) -> str:
        return f"{self.scheme}://{self.host}:{self.port}"

    def to_playwright_proxy(self) -> dict:
        """Return proxy config dict for patchright."""
        server = self.key if self.scheme == "socks5" else f"http://{self.host}:{self.port}"
        return {"server": server}


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

    def iter_candidates(self):
        """Yield proxies: good cache first, then fresh list, then file proxies."""
        bad_set = self._load_bad()
        good = [p for p in self._load_good() if p.key not in bad_set]
        self._log(f"good cache: {len(good)}, bad cache: {len(bad_set)}")

        for proxy in good:
            if self._reachable(proxy):
                yield proxy
            else:
                self.mark_bad(proxy, "unreachable")

        self._ensure_fresh()
        good_keys = {p.key for p in good}
        candidates = [
            p for p in (self._fresh + self._extra)
            if p.key not in good_keys and p.key not in bad_set
        ]
        random.shuffle(candidates)
        candidates = candidates[:MAX_CANDIDATES]

        ready: list[Proxy] = []
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
