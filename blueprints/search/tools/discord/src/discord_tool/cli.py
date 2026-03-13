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
    email: Annotated[Optional[str], typer.Option("--email", help="Email to use. If omitted, auto-generates a yopmail.com address.")] = None,
    proton_username: Annotated[Optional[str], typer.Option("--proton-user", help="Proton Mail username for auto-verification", envvar="PROTON_USERNAME")] = None,
    proton_password: Annotated[Optional[str], typer.Option("--proton-pass", help="Proton Mail password for auto-verification", envvar="PROTON_PASSWORD")] = None,
    yopmail: bool = typer.Option(True, "--yopmail/--no-yopmail", help="Use yopmail.com for auto email verification (default: True)"),
    db: DB_OPT = DEFAULT_DB_PATH,
    headless: bool = typer.Option(False, help="Run browser in headless mode"),
    verbose: bool = typer.Option(True, "--verbose/--no-verbose", "-v"),
) -> None:
    """Register a new Discord account via browser automation and store the token.

    By default uses yopmail.com for instant email verification (no registration needed).
    Use --email to provide a specific email, --proton-user/--proton-pass for Proton Mail.

    Examples:
      discord-tool register                          # auto yopmail (recommended)
      discord-tool register --email user@proton.me --proton-user user --proton-pass 'pass'
    """
    from .identity import generate
    from .browser import register_via_browser

    identity = generate()
    yopmail_user_local = ""

    if email:
        reg_email = email
        # Auto-load Proton creds from protonmail-tool store if it's a proton.me address
        if not proton_username and reg_email.endswith("@proton.me"):
            _pm_user = reg_email.split("@")[0]
            _pm_pass = proton_password or ""
            if not _pm_pass:
                try:
                    import sys, os as _os
                    _pm_src = _os.path.join(_os.path.dirname(__file__),
                                             "../../../protonmail/src")
                    sys.path.insert(0, _os.path.abspath(_pm_src))
                    from protonmail_tool.store import Store as PmStore, DEFAULT_DB_PATH as PM_DB
                    ps = PmStore(PM_DB)
                    acct = ps.get(_pm_user)
                    ps.close()
                    if acct:
                        _pm_pass = acct["password"]
                        proton_username = _pm_user
                        proton_password = _pm_pass
                        console.print(f"  [dim]Loaded Proton creds from store for {reg_email}[/dim]")
                except Exception:
                    pass
    elif yopmail:
        yopmail_user_local = identity.email_local
        reg_email = f"{yopmail_user_local}@yopmail.com"
        console.print(f"  [dim]Using yopmail for auto-verification[/dim]")
    else:
        reg_email = f"{identity.email_local}@proton.me"

    console.print(f"[bold]Registering Discord account[/bold]")
    console.print(f"  Email:    [cyan]{reg_email}[/cyan]")
    console.print(f"  Username: [cyan]{identity.username}[/cyan]")
    console.print(f"  Password: [cyan]{identity.password}[/cyan]")
    console.print(f"  DOB:      {identity.birth_year}-{identity.birth_month:02d}-{identity.birth_day:02d}")
    if proton_username:
        console.print(f"  Proton:   will auto-verify via {proton_username}@proton.me inbox")
    elif yopmail_user_local:
        console.print(f"  Yopmail:  will auto-verify via {reg_email}")
    console.print()

    console.print("  Launching browser...")
    try:
        token = register_via_browser(
            email=reg_email,
            username=identity.username,
            password=identity.password,
            birth_year=identity.birth_year,
            birth_month=identity.birth_month,
            birth_day=identity.birth_day,
            headless=headless,
            verbose=verbose,
            proton_username=proton_username or "",
            proton_password=proton_password or "",
            yopmail_user=yopmail_user_local,
        )
    except Exception as e:
        err_console.print(f"Registration failed: {e}")
        raise typer.Exit(1)

    store = _store(db)
    store.add_account(
        email=reg_email,
        username=identity.username,
        password=identity.password,
        token=token,
    )
    store.close()

    console.print(f"\n[green]✓ Account registered[/green]")
    console.print(f"  Email:  {reg_email}")
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
# extract-token
# ---------------------------------------------------------------------------

@app.command("extract-token")
def extract_token(
    login_email: Annotated[Optional[str], typer.Option("--login-email", help="Discord account email to pre-fill")] = None,
    login_password: Annotated[Optional[str], typer.Option("--login-password", help="Discord account password to pre-fill")] = None,
    save_email: Annotated[Optional[str], typer.Option("--email", help="Email to save in DB (optional)")] = None,
    db: DB_OPT = DEFAULT_DB_PATH,
    wait: int = typer.Option(300, help="Seconds to wait for login"),
    verbose: bool = typer.Option(True, "--verbose/--no-verbose", "-v"),
) -> None:
    """Open Discord login in a browser, pre-fill credentials, capture token automatically.

    Provide --login-email and --login-password to auto-fill the form.
    You only need to solve the captcha (if shown) — everything else is automatic.
    """
    from .browser import extract_token_manual

    console.print("[bold]Token extraction via browser[/bold]")
    if login_email:
        console.print(f"  Pre-filling: [cyan]{login_email}[/cyan]")
        console.print("  Solve captcha if prompted, then token is captured automatically.")
    else:
        console.print("  Browser opens at discord.com/login — fill credentials manually.")
    console.print(f"  Waiting up to [bold]{wait}s[/bold]...\n")

    try:
        token = extract_token_manual(
            headless=False,
            verbose=verbose,
            wait=wait,
            email=login_email or "",
            password=login_password or "",
        )
    except Exception as e:
        err_console.print(f"Token extraction failed: {e}")
        raise typer.Exit(1)

    console.print(f"\n[green]✓ Token captured![/green]  {token[:20]}...")

    store = _store(db)
    if save_email:
        acct = store.get_account(save_email)
        if acct:
            store.update_token(save_email, token)
            console.print(f"  Updated token for {save_email}")
        else:
            store.add_account(email=save_email, username=save_email.split("@")[0], password="", token=token)
            console.print(f"  Saved new account for {save_email}")
    store.close()

    console.print(f"\n[dim]Export token:[/dim]  export DISCORD_TOKEN='{token}'")


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
