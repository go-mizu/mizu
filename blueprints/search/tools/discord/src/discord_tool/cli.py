"""discord-tool CLI — register Discord accounts and extract user tokens."""
from __future__ import annotations

import os
import sys
from pathlib import Path
from typing import Annotated, Optional

import typer
from rich.console import Console
from rich.table import Table

from .store import Store, DEFAULT_DB_PATH

app = typer.Typer(
    name="discord-tool",
    help="Register Discord accounts and extract user tokens for the search discord crawler.",
    no_args_is_help=True,
)
console = Console()
err_console = Console(stderr=True, style="red")

account_app = typer.Typer(help="Manage stored Discord accounts", no_args_is_help=True)
app.add_typer(account_app, name="account")

DB_OPT = Annotated[Path, typer.Option("--db", help="Path to accounts DuckDB", envvar="DISCORD_TOOL_DB")]


def _store(db: Path) -> Store:
    return Store(db)


# ---------------------------------------------------------------------------
# register
# ---------------------------------------------------------------------------

@app.command()
def register(
    db: DB_OPT = DEFAULT_DB_PATH,
    headless: bool = typer.Option(True, help="Run browser in headless mode"),
    verbose: bool = typer.Option(False, "--verbose", "-v"),
) -> None:
    """Register a new Discord account via browser automation and store the token."""
    from .identity import generate
    from .email import MailTmClient
    from .browser import register_via_browser

    identity = generate()
    console.print(f"[bold]Registering Discord account[/bold]")
    console.print(f"  Email:    {identity.email_local}@mail.tm (via mail.tm)")
    console.print(f"  Username: {identity.username}")
    console.print(f"  DOB:      {identity.birth_year}-{identity.birth_month:02d}-{identity.birth_day:02d}")

    mail_client = MailTmClient()
    console.print("  Creating mail.tm mailbox...")
    mailbox = mail_client.create_mailbox(identity.email_local, identity.password)
    console.print(f"  Mailbox:  {mailbox.address}")

    console.print("  Launching browser...")
    try:
        token = register_via_browser(
            mailbox=mailbox,
            mail_client=mail_client,
            username=identity.username,
            password=identity.password,
            birth_year=identity.birth_year,
            birth_month=identity.birth_month,
            birth_day=identity.birth_day,
            headless=headless,
            verbose=verbose,
        )
    except Exception as e:
        err_console.print(f"Registration failed: {e}")
        raise typer.Exit(1)

    store = _store(db)
    store.add_account(
        email=mailbox.address,
        username=identity.username,
        password=identity.password,
        token=token,
    )
    store.close()

    console.print(f"\n[green]✓ Account registered[/green]")
    console.print(f"  Email:  {mailbox.address}")
    console.print(f"  Token:  {token[:20]}...")
    console.print(f"\n[dim]Export token:[/dim]  export DISCORD_TOKEN='{token}'")


# ---------------------------------------------------------------------------
# login
# ---------------------------------------------------------------------------

@app.command()
def login(
    email: Annotated[str, typer.Argument(help="Discord account email")],
    password: Annotated[Optional[str], typer.Option("--password", "-p", help="Password (prompt if omitted)")] = None,
    db: DB_OPT = DEFAULT_DB_PATH,
    headless: bool = typer.Option(True, help="Run browser in headless mode"),
    verbose: bool = typer.Option(False, "--verbose", "-v"),
) -> None:
    """Login to an existing Discord account and extract/update the token."""
    from .browser import login_via_browser

    if not password:
        # Try to get password from store
        store = _store(db)
        acct = store.get_account(email)
        store.close()
        if acct:
            password = acct["password"]
        else:
            password = typer.prompt("Password", hide_input=True)

    console.print(f"[bold]Logging in[/bold] {email}")
    try:
        token = login_via_browser(email=email, password=password, headless=headless, verbose=verbose)
    except Exception as e:
        err_console.print(f"Login failed: {e}")
        raise typer.Exit(1)

    store = _store(db)
    acct = store.get_account(email)
    if acct:
        store.update_token(email, token)
        console.print(f"[green]✓ Token updated[/green] for {email}")
    else:
        store.add_account(email=email, username=email.split("@")[0], password=password, token=token)
        console.print(f"[green]✓ Account saved[/green] for {email}")
    store.close()

    console.print(f"  Token:  {token[:20]}...")
    console.print(f"\n[dim]Export token:[/dim]  export DISCORD_TOKEN='{token}'")


# ---------------------------------------------------------------------------
# account subcommands
# ---------------------------------------------------------------------------

@account_app.command("ls")
def account_ls(db: DB_OPT = DEFAULT_DB_PATH) -> None:
    """List all stored accounts."""
    store = _store(db)
    accounts = store.list_accounts()
    store.close()

    if not accounts:
        console.print("[dim]No accounts stored yet. Run [bold]discord-tool register[/bold] first.[/dim]")
        return

    table = Table(show_header=True, header_style="bold")
    table.add_column("Email")
    table.add_column("Username")
    table.add_column("Token (preview)")
    table.add_column("Active")
    table.add_column("Created")

    for a in accounts:
        token_preview = a["token"][:20] + "..." if a["token"] else ""
        active = "[green]✓[/green]" if a["is_active"] else "[red]✗[/red]"
        created = str(a["created_at"])[:16] if a["created_at"] else ""
        table.add_row(a["email"], a["username"], token_preview, active, created)

    console.print(table)


@account_app.command("token")
def account_token(
    email: Annotated[Optional[str], typer.Argument(help="Email (default: first active)")] = None,
    db: DB_OPT = DEFAULT_DB_PATH,
) -> None:
    """Print the stored token for an account (for piping into env vars)."""
    store = _store(db)
    if email:
        acct = store.get_account(email)
    else:
        acct = store.get_first_active()
    store.close()

    if not acct:
        err_console.print("No account found.")
        raise typer.Exit(1)

    print(acct["token"])


@account_app.command("rm")
def account_rm(
    email: Annotated[str, typer.Argument(help="Email of account to remove")],
    db: DB_OPT = DEFAULT_DB_PATH,
) -> None:
    """Remove a stored account."""
    store = _store(db)
    store.remove(email)
    store.close()
    console.print(f"[green]✓ Removed[/green] {email}")


# ---------------------------------------------------------------------------
# env
# ---------------------------------------------------------------------------

@app.command()
def env(
    email: Annotated[Optional[str], typer.Argument(help="Email (default: first active)")] = None,
    db: DB_OPT = DEFAULT_DB_PATH,
) -> None:
    """Print shell export statement for DISCORD_TOKEN."""
    store = _store(db)
    if email:
        acct = store.get_account(email)
    else:
        acct = store.get_first_active()
    store.close()

    if not acct:
        err_console.print("No account found. Run [bold]discord-tool register[/bold] first.")
        raise typer.Exit(1)

    print(f"export DISCORD_TOKEN='{acct['token']}'")


# ---------------------------------------------------------------------------
# Entry point
# ---------------------------------------------------------------------------

def app_entry() -> None:
    app()
