"""CLI entry point for x-register."""

from __future__ import annotations

import argparse
import sys

from . import proxy as proxy_mod
from . import registrar


def main() -> None:
    parser = argparse.ArgumentParser(
        prog="x-register",
        description="Auto-register X (Twitter) accounts using patchright + mail.tm",
    )
    parser.add_argument(
        "--count", "-n",
        type=int,
        default=1,
        metavar="N",
        help="Number of accounts to register (default: 1)",
    )
    parser.add_argument(
        "--proxies",
        metavar="FILE",
        help="Path to proxy file (one per line: scheme://host:port or host:port)",
    )
    parser.add_argument(
        "--solver-key",
        metavar="KEY",
        default="",
        help="Captcha solver API key for Arkose FunCaptcha (CapSolver or 2captcha)",
    )
    parser.add_argument(
        "--solver-service",
        choices=["capsolver", "2captcha"],
        default="capsolver",
        help="Captcha solver service (default: capsolver)",
    )
    parser.add_argument(
        "--no-headless",
        dest="headless",
        action="store_false",
        default=True,
        help="Show browser window (useful for debugging)",
    )
    parser.add_argument(
        "--verbose", "-v",
        action="store_true",
        help="Verbose step-by-step logging",
    )

    # IMAP mode: use a real email account with +alias addressing
    imap_group = parser.add_argument_group(
        "IMAP mode",
        "Use a real email account (Outlook/Gmail) with +alias addressing. "
        "Bypasses throwaway email blocks. Example: --imap-email user@outlook.com --imap-password secret"
    )
    imap_group.add_argument("--imap-email", metavar="EMAIL", help="IMAP account email (e.g. user@outlook.com)")
    imap_group.add_argument("--imap-password", metavar="PASS", help="IMAP account password (or App Password for Gmail)")
    imap_group.add_argument("--imap-provider", choices=["outlook", "gmail", "yahoo"], default="outlook",
                            help="IMAP provider (default: outlook)")

    args = parser.parse_args()

    # Load extra proxies from file
    extra: list[proxy_mod.Proxy] = []
    if args.proxies:
        extra = proxy_mod.load_proxy_file(args.proxies)
        print(f"Loaded {len(extra)} proxies from {args.proxies}", flush=True)

    proxy_mgr = proxy_mod.ProxyManager(extra_proxies=extra, verbose=args.verbose)

    imap_config = None
    if args.imap_email and args.imap_password:
        imap_config = registrar.ImapConfig(
            provider=args.imap_provider,
            address=args.imap_email,
            password=args.imap_password,
        )
        print(f"IMAP mode: {args.imap_email} ({args.imap_provider})", flush=True)

    config = registrar.Config(
        solver_key=args.solver_key,
        solver_service=args.solver_service,
        headless=args.headless,
        verbose=args.verbose,
        imap=imap_config,
    )

    try:
        registrar.register_n(args.count, proxy_mgr, config)
    except KeyboardInterrupt:
        print("\nInterrupted.", flush=True)
        sys.exit(1)


if __name__ == "__main__":
    main()
