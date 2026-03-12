"""End-to-end account registration orchestrator."""

from __future__ import annotations

import random
import string
import time
from dataclasses import dataclass, field

from curl_cffi.requests.exceptions import RequestException as CurlRequestException

from . import browser_register, email, email_imap, identity, proxy, store, twitter_api


@dataclass
class ImapConfig:
    provider: str    # "outlook" | "gmail" | "yahoo"
    address: str     # base account address
    password: str    # IMAP password


@dataclass
class Config:
    solver_key: str = ""
    solver_service: str = "capsolver"
    headless: bool = True
    verbose: bool = False
    imap: ImapConfig | None = None


class RegistrationError(Exception):
    """Non-fatal registration failure (try next account)."""


def _log(msg: str, verbose: bool = False) -> None:
    if verbose:
        ts = time.strftime("%H:%M:%S")
        print(f"[{ts}] {msg}", flush=True)
    else:
        print(msg, flush=True)


def _random_tag(n: int = 8) -> str:
    return "".join(random.choices(string.ascii_lowercase + string.digits, k=n))


def register_one(
    proxy_mgr: proxy.ProxyManager,
    config: Config,
) -> store.Account:
    """Register a single X account via browser automation. Returns saved Account.

    Raises RegistrationError on recoverable failures.
    """
    v = config.verbose

    # 1. Generate identity
    ident = identity.generate()
    _log(f"identity: {ident.display_name!r} @{ident.username}", v)

    # 2. Set up email (IMAP+alias or mail.tm)
    if config.imap:
        imap_client = email_imap.ImapMailClient(
            provider=config.imap.provider,
            address=config.imap.address,
            password=config.imap.password,
            verbose=v,
        )
        tag = _random_tag()
        mailbox_imap = imap_client.create_alias(tag)
        _log(f"imap alias: {mailbox_imap.address}", v)

        # Wrap into objects compatible with browser_register
        reg_email = mailbox_imap.address
        email_password = config.imap.password

        try:
            result = browser_register.sign_up_via_browser_imap(
                identity=ident,
                email_address=reg_email,
                imap_client=imap_client,
                imap_mailbox=mailbox_imap,
                headless=config.headless,
                verbose=v,
            )
        except RuntimeError as e:
            raise RegistrationError(str(e)) from e

    else:
        # mail.tm mode (may fail for X due to domain blocks)
        mail = email.MailTmClient(verbose=v)
        try:
            mailbox = mail.create_mailbox(ident.email_local)
            _log(f"mailbox: {mailbox.address}", v)
        except email.MailTmError as e:
            mail.close()
            raise RegistrationError(f"mail.tm: {e}") from e

        reg_email = mailbox.address
        email_password = mailbox.password

        try:
            result = browser_register.sign_up_via_browser(
                identity=ident,
                mailbox=mailbox,
                mail_client=mail,
                headless=config.headless,
                verbose=v,
            )
        except email.MailTmError as e:
            raise RegistrationError(f"mail.tm OTP: {e}") from e
        except RuntimeError as e:
            raise RegistrationError(str(e)) from e
        finally:
            mail.close()

    # 3. Tweet "hello, world!" via API
    _log('tweeting "hello, world!"...', v)
    session = twitter_api.TwitterSession(verbose=v)
    tweet_id = ""
    try:
        session.activate()
        tweet_id = session.create_tweet(result.auth_token, result.ct0, "hello, world!")
        _log(f"tweeted! tweet_id={tweet_id}", v)
    except (twitter_api.TwitterError, CurlRequestException) as e:
        _log(f"[warn] tweet failed: {e}", v)
    finally:
        session.close()

    # 4. Save account
    account = store.make_account(
        email=reg_email,
        email_password=email_password,
        display_name=ident.display_name,
        username=result.screen_name,
        password=ident.password,
        auth_token=result.auth_token,
        ct0=result.ct0,
        user_id=result.user_id,
        tweet_id=tweet_id,
    )
    store.save(account)
    _log(f"saved: @{result.screen_name} ({reg_email})", v)
    return account


def register_n(
    count: int,
    proxy_mgr: proxy.ProxyManager,
    config: Config,
) -> list[store.Account]:
    """Register N accounts sequentially. Skips and logs errors; continues to next."""
    accounts: list[store.Account] = []
    failures = 0

    for i in range(count):
        print(f"\n=== Registration {i + 1}/{count} ===", flush=True)
        try:
            account = register_one(proxy_mgr, config)
            accounts.append(account)
            print(f"  OK: @{account.username} | {account.email}", flush=True)
            if account.tweet_id:
                print(f"  tweet: https://x.com/{account.username}/status/{account.tweet_id}", flush=True)
        except RegistrationError as e:
            failures += 1
            print(f"  SKIP: {e}", flush=True)
        except KeyboardInterrupt:
            print("\nInterrupted.", flush=True)
            break

    print(f"\nDone: {len(accounts)} registered, {failures} failed.", flush=True)
    print(f"Accounts saved to: {store.ACCOUNTS_FILE}", flush=True)
    return accounts
