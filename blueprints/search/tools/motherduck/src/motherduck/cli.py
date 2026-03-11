"""Typer CLI: motherduck register / account / db / query."""
from __future__ import annotations

import json
import sys
from pathlib import Path
from typing import Annotated, Optional

import typer
from rich.console import Console
from rich.table import Table

from .store import Store, DEFAULT_DB_PATH

app = typer.Typer(
    name="motherduck",
    help="Manage MotherDuck accounts and cloud DuckDB databases.",
    no_args_is_help=True,
)
account_app = typer.Typer(help="Manage accounts.", no_args_is_help=True)
db_app = typer.Typer(help="Manage databases.", no_args_is_help=True)
app.add_typer(account_app, name="account")
app.add_typer(db_app, name="db")

console = Console()
err_console = Console(stderr=True)


def _store() -> Store:
    return Store(DEFAULT_DB_PATH)


# ---------------------------------------------------------------------------
# register
# ---------------------------------------------------------------------------

@app.command()
def register(
    no_headless: Annotated[bool, typer.Option("--no-headless", help="Show browser window")] = False,
    verbose: Annotated[bool, typer.Option("--verbose", "-v")] = False,
    json_out: Annotated[bool, typer.Option("--json", help="Print JSON to stdout, skip DuckDB")] = False,
) -> None:
    """Auto-register a new MotherDuck account via browser + mail.tm."""
    from .email import MailTmClient
    from .identity import generate
    from .browser import register_via_browser

    # When --json: redirect rich output to stderr so stdout stays clean
    status_console = Console(stderr=True) if json_out else console

    identity = generate()
    mail_client = MailTmClient(verbose=verbose)

    with status_console.status("[bold green]Creating mail.tm mailbox..."):
        mailbox = mail_client.create_mailbox(identity.email_local)

    status_console.print(f"[green]Mailbox:[/green] {mailbox.address}")
    status_console.print("[bold green]Opening browser for MotherDuck signup...[/bold green]")

    try:
        token = register_via_browser(
            mailbox=mailbox,
            mail_client=mail_client,
            password=identity.password,
            headless=not no_headless,
            verbose=verbose,
        )
    except Exception as e:
        err_console.print(f"[bold red]Registration failed:[/bold red] {e}")
        raise typer.Exit(1)
    finally:
        mail_client.close()

    if json_out:
        # Output structured result; caller (Go CLI) stores in its own DuckDB
        output = {
            "email": mailbox.address,
            "password": identity.password,
            "token": token,
        }
        print(json.dumps(output))
        return

    store = _store()
    store.add_account(
        email=mailbox.address,
        password=identity.password,
        token=token,
    )

    console.print(f"\n[bold green]✓ Registered:[/bold green] {mailbox.address}")
    console.print(f"[dim]Token:[/dim] {token[:20]}...")
    console.print(f"[dim]Stored in:[/dim] {DEFAULT_DB_PATH}")


# ---------------------------------------------------------------------------
# account
# ---------------------------------------------------------------------------

@account_app.command("ls")
def account_ls() -> None:
    """List all accounts."""
    store = _store()
    rows = store.list_accounts()
    if not rows:
        console.print("[yellow]No accounts registered.[/yellow]")
        return

    table = Table(title="Accounts", show_lines=True)
    table.add_column("Email", style="cyan")
    table.add_column("Databases", justify="right")
    table.add_column("Active", justify="center")
    table.add_column("Created At")

    for r in rows:
        active = "[green]✓[/green]" if r["is_active"] else "[red]✗[/red]"
        created = str(r["created_at"])[:16] if r["created_at"] else "-"
        table.add_row(r["email"], str(r["db_count"]), active, created)

    console.print(table)


@account_app.command("rm")
def account_rm(
    email: Annotated[str, typer.Argument(help="Account email to deactivate")],
) -> None:
    """Deactivate an account (local only — does not delete from MotherDuck)."""
    store = _store()
    if not store.get_account_by_email(email):
        err_console.print(f"[bold red]Account not found:[/bold red] {email}")
        raise typer.Exit(1)
    store.deactivate_account(email)
    console.print(f"[yellow]Deactivated:[/yellow] {email}")


# ---------------------------------------------------------------------------
# db
# ---------------------------------------------------------------------------

@db_app.command("create")
def db_create(
    name: Annotated[str, typer.Argument(help="Database name on MotherDuck")],
    alias: Annotated[Optional[str], typer.Option("--alias", help="Local alias (default: same as name)")] = None,
    account: Annotated[Optional[str], typer.Option("--account", help="Account email to use")] = None,
    set_default: Annotated[bool, typer.Option("--default", help="Set as default database")] = False,
) -> None:
    """Create a new database on MotherDuck."""
    from .client import MotherDuckClient

    alias = alias or name
    store = _store()

    # Resolve account
    if account:
        row = store.get_account_by_email(account)
        if not row:
            err_console.print(f"[bold red]Account not found:[/bold red] {account}")
            raise typer.Exit(1)
        acc_id, token = row["id"], row["token"]
    else:
        acc = store.get_first_active_account()
        if not acc:
            err_console.print("[bold red]No active accounts. Run:[/bold red] motherduck register")
            raise typer.Exit(1)
        acc_id, token = acc["id"], acc["token"]

    with console.status(f"[green]Creating database '{name}' on MotherDuck..."):
        try:
            client = MotherDuckClient(token=token)
            client.create_db(name)
            client.close()
        except Exception as e:
            err_console.print(f"[bold red]Failed to create database:[/bold red] {e}")
            raise typer.Exit(1)

    store.add_database(account_id=acc_id, name=name, alias=alias)
    if set_default:
        store.set_default(alias)

    console.print(f"[bold green]✓ Created:[/bold green] {name} [dim](alias: {alias})[/dim]")
    if set_default:
        console.print(f"[green]Set as default.[/green]")


@db_app.command("ls")
def db_ls() -> None:
    """List all databases."""
    store = _store()
    rows = store.list_databases()
    if not rows:
        console.print("[yellow]No databases. Run:[/yellow] motherduck db create <name>")
        return

    table = Table(title="Databases", show_lines=True)
    table.add_column("Alias", style="cyan")
    table.add_column("Name")
    table.add_column("Account")
    table.add_column("Default", justify="center")
    table.add_column("Queries", justify="right")
    table.add_column("Last Used")
    table.add_column("Created")

    for r in rows:
        default = "[bold green]●[/bold green]" if r["is_default"] else ""
        last = str(r["last_used_at"])[:16] if r["last_used_at"] else "-"
        created = str(r["created_at"])[:16] if r["created_at"] else "-"
        alias_str = f"[bold]{r['alias']}[/bold]" if r["is_default"] else r["alias"]
        table.add_row(
            alias_str, r["name"], r["email"], default,
            str(r["query_count"]), last, created,
        )

    console.print(table)


@db_app.command("use")
def db_use(
    alias: Annotated[str, typer.Argument(help="Alias to set as default")],
) -> None:
    """Set the default database."""
    store = _store()
    if not store.get_db(alias):
        err_console.print(f"[bold red]No database with alias:[/bold red] {alias}")
        raise typer.Exit(1)
    store.set_default(alias)
    console.print(f"[green]Default set to:[/green] {alias}")


@db_app.command("rm")
def db_rm(
    alias: Annotated[str, typer.Argument(help="Alias to remove")],
) -> None:
    """Remove a database from local state (does not delete from MotherDuck cloud)."""
    store = _store()
    if not store.get_db(alias):
        err_console.print(f"[bold red]No database with alias:[/bold red] {alias}")
        raise typer.Exit(1)
    store.remove_database(alias)
    console.print(f"[yellow]Removed:[/yellow] {alias} [dim](local state only)[/dim]")


# ---------------------------------------------------------------------------
# query
# ---------------------------------------------------------------------------

@app.command()
def query(
    sql: Annotated[str, typer.Argument(help="SQL to run")],
    db: Annotated[Optional[str], typer.Option("--db", help="DB alias or name (default: use default)")] = None,
    json_out: Annotated[bool, typer.Option("--json", help="Output raw JSON")] = False,
) -> None:
    """Run SQL against a MotherDuck database."""
    from .client import MotherDuckClient
    import time as _time

    store = _store()

    # Resolve DB
    if db:
        db_row = store.get_db(db)
        if not db_row:
            err_console.print(f"[bold red]No database with alias:[/bold red] {db}")
            raise typer.Exit(1)
    else:
        db_row = store.get_default_db()
        if not db_row:
            err_console.print(
                "[bold red]No default database set. Use:[/bold red] motherduck db use <alias>"
            )
            raise typer.Exit(1)

    token = db_row["token"]
    db_name = db_row["name"]
    db_id = db_row["id"]

    t0 = _time.monotonic()
    try:
        client = MotherDuckClient(token=token)
        rows, cols = client.run_query(db_name, sql)
        client.close()
    except Exception as e:
        err_console.print(f"[bold red]Query failed:[/bold red] {e}")
        raise typer.Exit(1)

    duration_ms = int((_time.monotonic() - t0) * 1000)
    store.log_query(db_id=db_id, sql=sql, rows_returned=len(rows), duration_ms=duration_ms)
    store.touch_last_used(db_row["alias"])

    if json_out:
        data = [dict(zip(cols, r)) for r in rows]
        print(json.dumps(data, indent=2, default=str))
        return

    if not rows:
        console.print("[dim]No rows returned.[/dim]")
        return

    table = Table(show_lines=True)
    for col in cols:
        table.add_column(col)
    for row in rows:
        table.add_row(*[str(v) if v is not None else "[dim]NULL[/dim]" for v in row])

    console.print(table)
    console.print(
        f"[dim]{len(rows)} row(s) · {duration_ms}ms · db: {db_name}[/dim]"
    )


# ---------------------------------------------------------------------------
# Entry point
# ---------------------------------------------------------------------------

def app_entry() -> None:
    app()


if __name__ == "__main__":
    app_entry()
