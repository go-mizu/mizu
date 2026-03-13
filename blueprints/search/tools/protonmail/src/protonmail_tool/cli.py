"""protonmail-tool CLI — register Proton Mail accounts for use with Discord."""
from __future__ import annotations

from pathlib import Path
from typing import Annotated, Optional

import typer
from rich.console import Console
from rich.table import Table

from .store import Store, DEFAULT_DB_PATH

app = typer.Typer(
    name="protonmail-tool",
    help="Register Proton Mail accounts (@proton.me) for Discord registration.",
    no_args_is_help=True,
)
console = Console()
err_console = Console(stderr=True, style="red")

account_app = typer.Typer(help="Manage stored Proton Mail accounts", no_args_is_help=True)
app.add_typer(account_app, name="account")

DB_OPT = Annotated[Path, typer.Option("--db", help="accounts DuckDB path", envvar="PROTONMAIL_TOOL_DB")]


def _store(db: Path) -> Store:
    return Store(db)


# ---------------------------------------------------------------------------
# register
# ---------------------------------------------------------------------------

@app.command()
def register(
    db: DB_OPT = DEFAULT_DB_PATH,
    headless: bool = typer.Option(False, help="Run browser headless (not recommended — captcha needs manual solve)"),
    verbose: bool = typer.Option(True, "--verbose/--no-verbose", "-v"),
) -> None:
    """Register a new Proton Mail account via browser automation.

    A browser window opens at account.proton.me/signup.
    Everything is filled automatically — you only need to solve the captcha.
    """
    from .identity import generate
    from .browser import register_via_browser

    identity = generate()

    console.print("[bold]Registering Proton Mail account[/bold]")
    console.print(f"  Username:     [cyan]{identity.username}[/cyan]")
    console.print(f"  Password:     [cyan]{identity.password}[/cyan]")
    console.print(f"  Email:        [cyan]{identity.username}@proton.me[/cyan]")
    console.print(f"\n  [yellow]Browser will open — solve the captcha when shown.[/yellow]\n")

    try:
        email = register_via_browser(
            username=identity.username,
            password=identity.password,
            display_name=identity.display_name,
            headless=headless,
            verbose=verbose,
        )
    except Exception as e:
        err_console.print(f"Registration failed: {e}")
        raise typer.Exit(1)

    store = _store(db)
    store.add(username=identity.username, password=identity.password,
               display_name=identity.display_name)
    store.close()

    console.print(f"\n[green]✓ Account registered:[/green]  {email}")
    console.print(f"  Password: {identity.password}")
    console.print(f"\n[dim]Use with discord-tool:[/dim]")
    console.print(f"  cd ../discord && uv run python discord_entry.py register --no-headless --verbose")
    console.print(f"  [dim](when asked for email, use: {email})[/dim]")


# ---------------------------------------------------------------------------
# wait-link  (poll Proton Mail inbox for verification link)
# ---------------------------------------------------------------------------

@app.command("wait-link")
def wait_link(
    email_or_username: Annotated[Optional[str], typer.Argument(help="Proton Mail email or username")] = None,
    keyword: str = typer.Option("discord", help="Keyword to match in the link URL"),
    timeout: int = typer.Option(120, help="Seconds to wait for email"),
    db: DB_OPT = DEFAULT_DB_PATH,
    headless: bool = typer.Option(False),
    verbose: bool = typer.Option(True, "--verbose/--no-verbose"),
) -> None:
    """Poll Proton Mail inbox and print the first verification link found.

    Opens a browser, logs in, watches for new email containing keyword in the link.
    Prints the URL so you can pipe it or open it manually.
    """
    from .browser import wait_for_link

    store = _store(db)
    if email_or_username:
        acct = store.get(email_or_username)
    else:
        acct = store.get_first_active()
    store.close()

    if not acct:
        err_console.print("No account found. Run [bold]protonmail-tool register[/bold] first.")
        raise typer.Exit(1)

    console.print(f"[bold]Waiting for verification email[/bold] in {acct['email']}")
    console.print(f"  Keyword: [cyan]{keyword}[/cyan]  |  Timeout: {timeout}s\n")

    try:
        link = wait_for_link(
            username=acct["username"],
            password=acct["password"],
            keyword=keyword,
            timeout=timeout,
            headless=headless,
            verbose=verbose,
        )
    except TimeoutError as e:
        err_console.print(str(e))
        raise typer.Exit(1)
    except Exception as e:
        err_console.print(f"Failed: {e}")
        raise typer.Exit(1)

    console.print(f"\n[green]✓ Link found:[/green]")
    print(link)   # bare print so it can be piped


# ---------------------------------------------------------------------------
# wait-otp  (poll Proton Mail inbox for numeric OTP code)
# ---------------------------------------------------------------------------

@app.command("wait-otp")
def wait_otp(
    email_or_username: Annotated[Optional[str], typer.Argument(help="Proton Mail email or username")] = None,
    timeout: int = typer.Option(120, help="Seconds to wait for email"),
    db: DB_OPT = DEFAULT_DB_PATH,
    headless: bool = typer.Option(False),
    verbose: bool = typer.Option(True, "--verbose/--no-verbose"),
) -> None:
    """Poll Proton Mail inbox and print the first numeric OTP code found.

    Prints the bare code to stdout so it can be captured by other tools.
    """
    from .browser import wait_for_otp

    store = _store(db)
    if email_or_username:
        acct = store.get(email_or_username)
    else:
        acct = store.get_first_active()
    store.close()

    if not acct:
        err_console.print("No account found. Run [bold]protonmail-tool register[/bold] first.")
        raise typer.Exit(1)

    console.print(f"[bold]Waiting for OTP email[/bold] in {acct['email']}")
    console.print(f"  Timeout: {timeout}s\n")

    try:
        code = wait_for_otp(
            username=acct["username"],
            password=acct["password"],
            timeout=timeout,
            headless=headless,
            verbose=verbose,
        )
    except TimeoutError as e:
        err_console.print(str(e))
        raise typer.Exit(1)
    except Exception as e:
        err_console.print(f"Failed: {e}")
        raise typer.Exit(1)

    console.print(f"\n[green]✓ OTP found:[/green]")
    print(code)   # bare print so it can be piped/captured


# ---------------------------------------------------------------------------
# account subcommands
# ---------------------------------------------------------------------------

@account_app.command("ls")
def account_ls(db: DB_OPT = DEFAULT_DB_PATH) -> None:
    """List all stored Proton Mail accounts."""
    store = _store(db)
    accounts = store.list_all()
    store.close()

    if not accounts:
        console.print("[dim]No accounts yet. Run [bold]protonmail-tool register[/bold].[/dim]")
        return

    table = Table(show_header=True, header_style="bold")
    table.add_column("Email")
    table.add_column("Password")
    table.add_column("Active")
    table.add_column("Created")

    for a in accounts:
        active = "[green]✓[/green]" if a["is_active"] else "[red]✗[/red]"
        created = str(a["created_at"])[:16]
        table.add_row(a["email"], a["password"], active, created)

    console.print(table)


@account_app.command("rm")
def account_rm(
    email_or_username: Annotated[str, typer.Argument()],
    db: DB_OPT = DEFAULT_DB_PATH,
) -> None:
    """Remove a stored account."""
    store = _store(db)
    store.remove(email_or_username)
    store.close()
    console.print(f"[green]✓ Removed[/green] {email_or_username}")


# ---------------------------------------------------------------------------
# env  — print email address for use in other scripts
# ---------------------------------------------------------------------------

@app.command()
def env(
    email_or_username: Annotated[Optional[str], typer.Argument()] = None,
    db: DB_OPT = DEFAULT_DB_PATH,
) -> None:
    """Print the Proton Mail email address (for use in Discord registration)."""
    store = _store(db)
    acct = store.get(email_or_username) if email_or_username else store.get_first_active()
    store.close()

    if not acct:
        err_console.print("No account found.")
        raise typer.Exit(1)

    print(acct["email"])


# ---------------------------------------------------------------------------
# Entry point
# ---------------------------------------------------------------------------

def app_entry() -> None:
    app()
