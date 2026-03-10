"""Typer CLI: clickhouse-tool register / account / service / query."""
from __future__ import annotations

import json
import sys
from typing import Annotated, Optional

import typer
from rich.console import Console
from rich.table import Table

from .store import Store, DEFAULT_DB_PATH

app = typer.Typer(
    name="clickhouse-tool",
    help="Manage ClickHouse Cloud accounts and services.",
    no_args_is_help=True,
)
account_app = typer.Typer(help="Manage accounts.", no_args_is_help=True)
service_app = typer.Typer(help="Manage services.", no_args_is_help=True)
app.add_typer(account_app, name="account")
app.add_typer(service_app, name="service")

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
) -> None:
    """Auto-register a new ClickHouse Cloud account via browser + mail.tm."""
    from .email import MailTmClient
    from .identity import generate
    from .browser import register_via_browser

    identity = generate()
    mail_client = MailTmClient(verbose=verbose)

    with console.status("[bold green]Creating mail.tm mailbox..."):
        mailbox = mail_client.create_mailbox(identity.email_local)

    console.print(f"[green]Mailbox:[/green] {mailbox.address}")
    console.print("[bold green]Opening browser for ClickHouse Cloud signup...[/bold green]")

    try:
        result = register_via_browser(
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

    store = _store()
    store.add_account(
        email=mailbox.address,
        password=identity.password,
    )

    # If a service was created during onboarding, store it
    service_id = result.get("service_id", "")
    host = result.get("host", "")
    if service_id and host:
        acc = store.get_account_by_email(mailbox.address)
        if acc:
            # Use a unique alias based on email prefix
            alias = mailbox.address.split("@")[0][:20]
            # Remove existing service with same alias if any
            if store.get_service(alias):
                store.remove_service(alias)
            store.add_service(
                account_id=acc["id"],
                name="default-service",
                alias=alias,
                cloud_id=service_id,
                host=host,
                port=result.get("port", 8443),
                db_password=result.get("db_password", ""),
            )
            store.set_default(alias)

    console.print(f"\n[bold green]Registered:[/bold green] {mailbox.address}")
    if host:
        console.print(f"[dim]Service host:[/dim] {host}")
        console.print(f"[dim]Service ID:[/dim] {service_id}")
    else:
        console.print("[yellow]No service host found. Create one with: clickhouse-tool service create[/yellow]")
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
    table.add_column("Org ID")
    table.add_column("Services", justify="right")
    table.add_column("Active", justify="center")
    table.add_column("Created At")

    for r in rows:
        active = "[green]\u2713[/green]" if r["is_active"] else "[red]\u2717[/red]"
        created = str(r["created_at"])[:16] if r["created_at"] else "-"
        org = r["org_id"][:12] + "..." if len(r.get("org_id", "")) > 12 else r.get("org_id", "")
        table.add_row(r["email"], org, str(r["svc_count"]), active, created)

    console.print(table)


@account_app.command("rm")
def account_rm(
    email: Annotated[str, typer.Argument(help="Account email to deactivate")],
) -> None:
    """Deactivate an account (local only)."""
    store = _store()
    if not store.get_account_by_email(email):
        err_console.print(f"[bold red]Account not found:[/bold red] {email}")
        raise typer.Exit(1)
    store.deactivate_account(email)
    console.print(f"[yellow]Deactivated:[/yellow] {email}")


# ---------------------------------------------------------------------------
# service
# ---------------------------------------------------------------------------

@service_app.command("create")
def service_create(
    name: Annotated[str, typer.Argument(help="Service name on ClickHouse Cloud")],
    provider: Annotated[str, typer.Option("--provider", help="Cloud provider")] = "aws",
    region: Annotated[str, typer.Option("--region", help="Cloud region")] = "us-east-1",
    tier: Annotated[str, typer.Option("--tier", help="Service tier")] = "development",
    alias: Annotated[Optional[str], typer.Option("--alias", help="Local alias")] = None,
    account: Annotated[Optional[str], typer.Option("--account", help="Account email")] = None,
    set_default: Annotated[bool, typer.Option("--default", help="Set as default")] = False,
) -> None:
    """Create a new service on ClickHouse Cloud."""
    from .cloud_api import ClickHouseCloudAPI

    alias = alias or name
    store = _store()

    if account:
        acc = store.get_account_by_email(account)
        if not acc:
            err_console.print(f"[bold red]Account not found:[/bold red] {account}")
            raise typer.Exit(1)
    else:
        acc = store.get_first_active_account()
        if not acc:
            err_console.print("[bold red]No active accounts. Run:[/bold red] clickhouse-tool register")
            raise typer.Exit(1)

    if not acc["org_id"]:
        err_console.print("[bold red]Account has no org_id. Re-register.[/bold red]")
        raise typer.Exit(1)

    with console.status(f"[green]Creating service '{name}' on ClickHouse Cloud..."):
        try:
            api = ClickHouseCloudAPI(acc["api_key_id"], acc["api_key_secret"])
            result = api.create_service(
                org_id=acc["org_id"], name=name,
                provider=provider, region=region, tier=tier,
            )
            api.close()
        except Exception as e:
            err_console.print(f"[bold red]Failed to create service:[/bold red] {e}")
            raise typer.Exit(1)

    svc = result.get("service", {})
    svc_password = result.get("password", "")
    endpoints = svc.get("endpoints", [])
    host = endpoints[0]["host"] if endpoints else ""
    port = endpoints[0].get("port", 8443) if endpoints else 8443

    store.add_service(
        account_id=acc["id"], name=name, alias=alias,
        cloud_id=svc.get("id", ""), host=host, port=port,
        db_password=svc_password, provider=provider, region=region,
    )
    if set_default:
        store.set_default(alias)

    console.print(f"[bold green]Created:[/bold green] {name} [dim](alias: {alias})[/dim]")
    console.print(f"[dim]Host:[/dim] {host}")
    console.print(f"[dim]Password:[/dim] {svc_password[:10]}...")
    if set_default:
        console.print("[green]Set as default.[/green]")


@service_app.command("ls")
def service_ls() -> None:
    """List all services."""
    store = _store()
    rows = store.list_services()
    if not rows:
        console.print("[yellow]No services. Run:[/yellow] clickhouse-tool service create <name>")
        return

    table = Table(title="Services", show_lines=True)
    table.add_column("Alias", style="cyan")
    table.add_column("Name")
    table.add_column("Host")
    table.add_column("Account")
    table.add_column("Default", justify="center")
    table.add_column("Queries", justify="right")
    table.add_column("Last Used")
    table.add_column("Created")

    for r in rows:
        default = "[bold green]\u25cf[/bold green]" if r["is_default"] else ""
        last = str(r["last_used_at"])[:16] if r["last_used_at"] else "-"
        created = str(r["created_at"])[:16] if r["created_at"] else "-"
        host_short = r["host"][:30] + "..." if len(r.get("host", "")) > 30 else r.get("host", "")
        alias_str = f"[bold]{r['alias']}[/bold]" if r["is_default"] else r["alias"]
        table.add_row(
            alias_str, r["name"], host_short, r["email"], default,
            str(r["query_count"]), last, created,
        )

    console.print(table)


@service_app.command("use")
def service_use(
    alias: Annotated[str, typer.Argument(help="Alias to set as default")],
) -> None:
    """Set the default service."""
    store = _store()
    if not store.get_service(alias):
        err_console.print(f"[bold red]No service with alias:[/bold red] {alias}")
        raise typer.Exit(1)
    store.set_default(alias)
    console.print(f"[green]Default set to:[/green] {alias}")


@service_app.command("rm")
def service_rm(
    alias: Annotated[str, typer.Argument(help="Alias to remove")],
) -> None:
    """Remove a service from local state."""
    store = _store()
    if not store.get_service(alias):
        err_console.print(f"[bold red]No service with alias:[/bold red] {alias}")
        raise typer.Exit(1)
    store.remove_service(alias)
    console.print(f"[yellow]Removed:[/yellow] {alias} [dim](local state only)[/dim]")


# ---------------------------------------------------------------------------
# query
# ---------------------------------------------------------------------------

@app.command()
def query(
    sql: Annotated[str, typer.Argument(help="SQL to run")],
    service: Annotated[Optional[str], typer.Option("--service", help="Service alias")] = None,
    json_out: Annotated[bool, typer.Option("--json", help="Output raw JSON")] = False,
) -> None:
    """Run SQL against a ClickHouse Cloud service."""
    from .client import ClickHouseClient
    import time as _time

    store = _store()

    if service:
        svc = store.get_service(service)
        if not svc:
            err_console.print(f"[bold red]No service with alias:[/bold red] {service}")
            raise typer.Exit(1)
    else:
        svc = store.get_default_service()
        if not svc:
            err_console.print(
                "[bold red]No default service. Use:[/bold red] clickhouse-tool service use <alias>"
            )
            raise typer.Exit(1)

    t0 = _time.monotonic()
    try:
        client = ClickHouseClient(
            host=svc["host"], port=svc["port"],
            username=svc["db_user"], password=svc["db_password"],
        )
        rows, cols = client.run_query(sql)
        client.close()
    except Exception as e:
        err_console.print(f"[bold red]Query failed:[/bold red] {e}")
        raise typer.Exit(1)

    duration_ms = int((_time.monotonic() - t0) * 1000)
    store.log_query(service_id=svc["id"], sql=sql, rows_returned=len(rows), duration_ms=duration_ms)
    store.touch_last_used(svc["alias"])

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
        f"[dim]{len(rows)} row(s) \u00b7 {duration_ms}ms \u00b7 service: {svc['name']}[/dim]"
    )


# ---------------------------------------------------------------------------
# Entry point
# ---------------------------------------------------------------------------

def app_entry() -> None:
    app()
