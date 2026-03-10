"""Gmail account registration orchestrator."""

from __future__ import annotations

import time
from dataclasses import dataclass

from .browser_register import RegisteredAccount, register as browser_register
from .identity import Identity, generate as generate_identity
from .proxy import ProxyManager, Proxy, load_proxy_file
from .sms import make_client as make_sms_client
from .store import Account, make_account, save


@dataclass
class Config:
    sms_service: str = "manual"   # "smsactivate" | "5sim" | "manual"
    sms_key: str = ""
    sms_phone: str = ""           # for manual mode
    sms_country: str = "any"
    headless: bool = True
    verbose: bool = False
    proxy_file: str = ""          # path to proxy list file
    use_proxies: bool = True


def register_one(cfg: Config) -> Account:
    """Register a single Gmail account. Returns saved Account."""
    identity = generate_identity()
    sms_client = make_sms_client(
        service=cfg.sms_service,
        api_key=cfg.sms_key,
        phone=cfg.sms_phone,
        verbose=cfg.verbose,
    )

    proxy_config: dict | None = None
    proxy_mgr: ProxyManager | None = None

    if cfg.use_proxies:
        extra: list[Proxy] = []
        if cfg.proxy_file:
            extra = load_proxy_file(cfg.proxy_file)
        proxy_mgr = ProxyManager(extra_proxies=extra, verbose=cfg.verbose)
        for proxy in proxy_mgr.iter_candidates():
            proxy_config = proxy.to_playwright_proxy()
            proxy_mgr.mark_good(proxy)
            break  # use first reachable proxy

    try:
        result: RegisteredAccount = browser_register(
            identity=identity,
            sms_client=sms_client,
            proxy_config=proxy_config,
            headless=cfg.headless,
            verbose=cfg.verbose,
            country=cfg.sms_country,
            sms_service=cfg.sms_service,
        )
    finally:
        sms_client.close()

    account = make_account(
        email=result.email,
        first_name=identity.first_name,
        last_name=identity.last_name,
        password=identity.password,
        phone=result.phone,
        birth_year=identity.birth_year,
        birth_month=identity.birth_month,
        birth_day=identity.birth_day,
    )
    save(account)
    return account


def register_n(n: int, cfg: Config) -> list[Account]:
    """Register N Gmail accounts sequentially."""
    accounts: list[Account] = []
    for i in range(n):
        print(f"\n[{i+1}/{n}] Starting Gmail registration...", flush=True)
        try:
            account = register_one(cfg)
            accounts.append(account)
            print(f"[{i+1}/{n}] Registered: {account.email}", flush=True)
        except Exception as e:
            print(f"[{i+1}/{n}] Failed: {e}", flush=True)
        if i < n - 1:
            time.sleep(2)
    print(f"\nDone: {len(accounts)}/{n} accounts registered.", flush=True)
    return accounts
