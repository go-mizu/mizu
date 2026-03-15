"""Typer CLI: goodread-tool register / account / test / cookies."""
from __future__ import annotations

import json
import sys
from pathlib import Path
from typing import Annotated, Optional

import typer
from rich.console import Console
from rich.table import Table

from .store import Store, DEFAULT_DB_PATH

DEFAULT_COOKIES_PATH = Path.home() / "data" / "goodread" / "cookies.json"

app = typer.Typer(
    name="goodread-tool",
    help="Manage Goodreads accounts and export session cookies.",
    no_args_is_help=True,
)
account_app = typer.Typer(help="Manage accounts.", no_args_is_help=True)
cookies_app = typer.Typer(help="Manage cookies.", no_args_is_help=True)
app.add_typer(account_app, name="account")
app.add_typer(cookies_app, name="cookies")

console = Console()
err_console = Console(stderr=True)

DB_OPT = Annotated[Path, typer.Option("--db", help="accounts DuckDB path", envvar="GOODREAD_TOOL_DB")]


def _store(db: Path) -> Store:
    return Store(db)


# ---------------------------------------------------------------------------
# register
# ---------------------------------------------------------------------------

_PROTONMAIL_DB = Path.home() / "data" / "protonmail" / "accounts.duckdb"
_OLD_PROTONMAIL_DB = Path.home() / "data" / "protonmail" / "protonmail.duckdb"


def _get_protonmail_account(pm_email: str | None) -> dict:
    """Load a Proton Mail account from the protonmail store."""
    import duckdb
    db_path = _PROTONMAIL_DB if _PROTONMAIL_DB.exists() else _OLD_PROTONMAIL_DB
    if not db_path.exists():
        raise RuntimeError(
            f"Proton Mail store not found at {db_path}. "
            "Run: protonmail-tool register"
        )
    con = duckdb.connect(str(db_path), read_only=True)
    try:
        if pm_email:
            val = pm_email if "@" in pm_email else f"{pm_email}@proton.me"
            row = con.execute(
                "SELECT email, username, password FROM accounts WHERE email = ?", [val]
            ).fetchone()
        else:
            row = con.execute(
                "SELECT email, username, password FROM accounts "
                "WHERE is_active = true ORDER BY created_at LIMIT 1"
            ).fetchone()
    finally:
        con.close()
    if not row:
        raise RuntimeError("No active Proton Mail account found. Run: protonmail-tool register")
    return {"email": row[0], "username": row[1], "password": row[2]}


@app.command()
def register(
    db: DB_OPT = DEFAULT_DB_PATH,
    headless: Annotated[bool, typer.Option("--headless", help="Run browser headless (WARNING: Goodreads blocks headless Chrome)")] = False,
    verbose: Annotated[bool, typer.Option("--verbose", "-v")] = False,
) -> None:
    """Auto-register a new Goodreads account via browser + mail.tm OTP.

    Uses a temporary mail.tm address to receive the Amazon OTP email.
    A browser window will open (headed mode) — Goodreads blocks headless Chrome.
    """
    from .email import MailTmClient
    from .identity import generate
    from .browser import register_via_browser

    identity = generate()
    mail_client = MailTmClient(verbose=verbose)

    with console.status("[bold green]Creating mail.tm mailbox..."):
        mailbox = mail_client.create_mailbox(identity.email_local)

    console.print(f"[green]Mailbox:[/green] {mailbox.address}")
    console.print(f"[green]Name:[/green] {identity.name}")
    console.print("[bold green]Opening browser for Goodreads signup...[/bold green]")

    def poll_otp() -> str:
        return mail_client.poll_for_otp(mailbox, timeout=220)

    try:
        cookies = register_via_browser(
            name=identity.name,
            email=mailbox.address,
            password=identity.password,
            poll_otp_fn=poll_otp,
            headless=headless,
            verbose=verbose,
        )
    except Exception as e:
        err_console.print(f"[bold red]Registration failed:[/bold red] {e}")
        raise typer.Exit(1)
    finally:
        mail_client.close()

    store = _store(db)
    try:
        store.add_account(email=mailbox.address, password=identity.password)
        store.update_cookies(mailbox.address, cookies)
    finally:
        store.close()

    console.print(f"\n[bold green]✓ Registered:[/bold green] {mailbox.address}")
    console.print(f"[dim]Cookies:[/dim] {len(cookies)} extracted")
    console.print(f"[dim]Stored in:[/dim] {db}")
    console.print(f"\nExport cookies: [cyan]goodread-tool cookies export[/cyan]")


# ---------------------------------------------------------------------------
# login (manual — reliable, no bot detection risk)
# ---------------------------------------------------------------------------

@app.command()
def login(
    email: Annotated[Optional[str], typer.Argument(help="Account email to store cookies under (optional)")] = None,
    password: Annotated[Optional[str], typer.Option("--password", "-p", help="Account password (optional)")] = None,
    db: DB_OPT = DEFAULT_DB_PATH,
    timeout: Annotated[int, typer.Option("--timeout", help="Seconds to wait for login")] = 300,
    verbose: Annotated[bool, typer.Option("--verbose", "-v")] = False,
) -> None:
    """Open a browser window — log in manually, cookies are saved automatically.

    This is the most reliable approach (no bot detection risk).
    If --email is provided, cookies are stored under that account.
    Otherwise a new account entry is created from whatever email you log in with.
    """
    from .browser import login_via_browser

    console.print("[bold green]Opening browser for manual Goodreads login...[/bold green]")

    try:
        cookies = login_via_browser(verbose=verbose, timeout=timeout)
    except Exception as e:
        err_console.print(f"[bold red]Login failed:[/bold red] {e}")
        raise typer.Exit(1)

    store = _store(db)
    try:
        used_email = email or "manual@goodreads.com"
        acct = store.get_by_email(used_email)
        if not acct:
            store.add_account(email=used_email, password=password or "")
        store.update_cookies(used_email, cookies)
    finally:
        store.close()

    console.print(f"\n[bold green]✓ Logged in:[/bold green] {used_email}")
    console.print(f"[dim]Cookies:[/dim] {len(cookies)} extracted")
    console.print(f"[dim]Stored in:[/dim] {db}")
    console.print(f"\nExport cookies: [cyan]goodread-tool cookies export[/cyan]")


# ---------------------------------------------------------------------------
# test
# ---------------------------------------------------------------------------

@app.command()
def test(
    email: Annotated[Optional[str], typer.Argument(help="Account email (default: most recent active)")] = None,
    db: DB_OPT = DEFAULT_DB_PATH,
    verbose: Annotated[bool, typer.Option("--verbose", "-v")] = False,
) -> None:
    """Test that stored cookies can authenticate with Goodreads."""
    from .browser import test_cookies

    store = _store(db)
    try:
        acct = store.get_by_email(email) if email else store.get_first_active()
    finally:
        store.close()

    if not acct:
        err_console.print("[bold red]No active account found.[/bold red] Run: goodread-tool register")
        raise typer.Exit(1)

    cookies_json = acct.get("cookies", "[]") or "[]"
    try:
        cookies = json.loads(cookies_json)
    except Exception:
        cookies = []

    if not cookies:
        err_console.print(f"[bold red]No cookies stored for:[/bold red] {acct['email']}")
        raise typer.Exit(1)

    console.print(f"Testing {len(cookies)} cookies for [cyan]{acct['email']}[/cyan] ...")
    user_id = test_cookies(cookies, verbose=verbose)

    if user_id:
        console.print(f"[bold green]✓ Authenticated[/bold green] (user_id={user_id})")
    else:
        err_console.print("[bold red]✗ Authentication failed[/bold red] — cookies may be expired")
        raise typer.Exit(1)


# ---------------------------------------------------------------------------
# account subcommands
# ---------------------------------------------------------------------------

@account_app.command("ls")
def account_ls(db: DB_OPT = DEFAULT_DB_PATH) -> None:
    """List all accounts."""
    store = _store(db)
    try:
        rows = store.list_accounts()
    finally:
        store.close()

    if not rows:
        console.print("[yellow]No accounts registered.[/yellow]")
        return

    table = Table(title="Accounts", show_lines=True)
    table.add_column("Email", style="cyan")
    table.add_column("User ID")
    table.add_column("Active", justify="center")
    table.add_column("Created At")

    for r in rows:
        active = "[green]✓[/green]" if r["is_active"] else "[red]✗[/red]"
        created = str(r["created_at"])[:16] if r["created_at"] else "-"
        table.add_row(r["email"], r["user_id"] or "-", active, created)

    console.print(table)


@account_app.command("rm")
def account_rm(
    email: Annotated[str, typer.Argument(help="Account email to deactivate")],
    db: DB_OPT = DEFAULT_DB_PATH,
) -> None:
    """Deactivate an account (local only — does not delete from Goodreads)."""
    store = _store(db)
    try:
        if not store.get_by_email(email):
            err_console.print(f"[bold red]Account not found:[/bold red] {email}")
            raise typer.Exit(1)
        store.deactivate(email)
    finally:
        store.close()
    console.print(f"[yellow]Deactivated:[/yellow] {email}")


# ---------------------------------------------------------------------------
# cookies subcommands
# ---------------------------------------------------------------------------

@cookies_app.command("export")
def cookies_export(
    email: Annotated[Optional[str], typer.Argument(help="Account email (default: most recent active)")] = None,
    output: Annotated[Path, typer.Option("--output", "-o", help="Output path for cookies.json")] = DEFAULT_COOKIES_PATH,
    db: DB_OPT = DEFAULT_DB_PATH,
) -> None:
    """Export session cookies to a JSON file for use by the Go scraper."""
    store = _store(db)
    try:
        used_email = store.export_cookies_file(email, output)
    except ValueError as e:
        err_console.print(f"[bold red]Error:[/bold red] {e}")
        raise typer.Exit(1)
    finally:
        store.close()

    console.print(f"[bold green]✓ Cookies exported:[/bold green] {output}")
    console.print(f"[dim]Account:[/dim] {used_email}")
    console.print(f"\nNow use with Go scraper:")
    console.print(f"  search goodread search \"Dune\" [bold cyan]--auth[/bold cyan]")
    console.print(f"  search goodread shelf <user_id> [bold cyan]--auth[/bold cyan]")


@cookies_app.command("show")
def cookies_show(
    email: Annotated[Optional[str], typer.Argument(help="Account email (default: most recent active)")] = None,
    db: DB_OPT = DEFAULT_DB_PATH,
) -> None:
    """Print stored cookies as JSON (to stdout)."""
    store = _store(db)
    try:
        acct = store.get_by_email(email) if email else store.get_first_active()
    finally:
        store.close()

    if not acct:
        err_console.print("[bold red]No active account found.[/bold red]")
        raise typer.Exit(1)

    cookies_json = acct.get("cookies", "[]") or "[]"
    try:
        cookies = json.loads(cookies_json)
    except Exception:
        cookies = []

    print(json.dumps(cookies, indent=2))


# ---------------------------------------------------------------------------
# Entry point
# ---------------------------------------------------------------------------

def app_entry() -> None:
    app()


if __name__ == "__main__":
    app_entry()
