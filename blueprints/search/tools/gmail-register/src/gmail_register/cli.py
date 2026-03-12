"""CLI entry point for gmail-register."""

from __future__ import annotations

import argparse
import sys

from .registrar import Config, register_n


def main() -> None:
    parser = argparse.ArgumentParser(
        prog="gmail-register",
        description="Auto-register Gmail accounts using patchright + SMS verification.",
    )
    parser.add_argument(
        "-n", "--count",
        type=int,
        default=1,
        metavar="N",
        help="Number of accounts to register (default: 1)",
    )
    parser.add_argument(
        "--sms-service",
        choices=["smspool", "5sim", "manual"],
        default="manual",
        help="SMS verification service (default: manual)",
    )
    parser.add_argument(
        "--sms-key",
        default="",
        metavar="KEY",
        help="API key for smsactivate or 5sim",
    )
    parser.add_argument(
        "--phone",
        default="",
        metavar="PHONE",
        help="Phone number for manual SMS mode (e.g. +14155552671)",
    )
    parser.add_argument(
        "--sms-country",
        default="any",
        metavar="COUNTRY",
        help="Country for SMS number (any|us|uk|ru|in; default: any)",
    )
    parser.add_argument(
        "--proxies",
        default="",
        metavar="FILE",
        help="Path to proxy list file (one per line, scheme://host:port)",
    )
    parser.add_argument(
        "--no-proxy",
        action="store_true",
        help="Disable proxy usage entirely",
    )
    parser.add_argument(
        "--no-headless",
        action="store_true",
        help="Show browser window (useful for debugging)",
    )
    parser.add_argument(
        "-v", "--verbose",
        action="store_true",
        help="Verbose logging",
    )

    args = parser.parse_args()

    if args.sms_service in ("smspool", "5sim") and not args.sms_key:
        parser.error(f"--sms-key is required when --sms-service={args.sms_service}")

    if args.sms_service == "manual" and not args.phone:
        print(
            "Warning: --sms-service=manual requires --phone; "
            "you will be prompted interactively.",
            file=sys.stderr,
        )

    cfg = Config(
        sms_service=args.sms_service,
        sms_key=args.sms_key,
        sms_phone=args.phone,
        sms_country=args.sms_country,
        headless=not args.no_headless,
        verbose=args.verbose,
        proxy_file=args.proxies,
        use_proxies=not args.no_proxy,
    )

    register_n(args.count, cfg)


if __name__ == "__main__":
    main()
