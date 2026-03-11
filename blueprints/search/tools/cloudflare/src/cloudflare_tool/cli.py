"""Typer CLI: cloudflare register / account / token / worker."""
from __future__ import annotations

import json
import time as _time
from pathlib import Path
from typing import Annotated, Optional

import typer
from rich.console import Console
from rich.table import Table

from .store import Store, DEFAULT_DB_PATH

app = typer.Typer(
    name="cloudflare-tool",
    help="Manage Cloudflare accounts, API tokens, and Workers.",
    no_args_is_help=True,
)
account_app = typer.Typer(help="Manage accounts.", no_args_is_help=True)
token_app = typer.Typer(help="Manage API tokens.", no_args_is_help=True)
worker_app = typer.Typer(help="Manage Workers.", no_args_is_help=True)
app.add_typer(account_app, name="account")
app.add_typer(token_app, name="token")
app.add_typer(worker_app, name="worker")

console = Console()
err_console = Console(stderr=True)

_CF_JSON = Path.home() / "data" / "cloudflare" / "cloudflare.json"


def _store() -> Store:
    return Store(DEFAULT_DB_PATH)


# ---------------------------------------------------------------------------
# register
# ---------------------------------------------------------------------------

@app.command()
def register(
    no_headless: Annotated[bool, typer.Option("--no-headless")] = False,
    verbose: Annotated[bool, typer.Option("--verbose", "-v")] = False,
    json_out: Annotated[bool, typer.Option("--json")] = False,
) -> None:
    """Auto-register a new Cloudflare account via browser + mail.tm."""
    from .email import MailTmClient
    from .identity import generate
    from .browser import register_via_browser

    status_console = Console(stderr=True) if json_out else console

    identity = generate()
    mail_client = MailTmClient(verbose=verbose)

    with status_console.status("[bold green]Creating mail.tm mailbox..."):
        mailbox = mail_client.create_mailbox(identity.email_local)

    status_console.print(f"[green]Mailbox:[/green] {mailbox.address}")
    status_console.print("[bold green]Opening browser for Cloudflare signup...[/bold green]")

    try:
        account_id = register_via_browser(
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
        print(json.dumps({
            "email": mailbox.address,
            "password": identity.password,
            "account_id": account_id,
        }))
        return

    store = _store()
    store.add_account(
        email=mailbox.address,
        password=identity.password,
        account_id=account_id,
    )

    console.print(f"\n[bold green]✓ Registered:[/bold green] {mailbox.address}")
    console.print(f"[dim]Account ID:[/dim] {account_id}")
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
    table.add_column("Account ID")
    table.add_column("Tokens", justify="right")
    table.add_column("Workers", justify="right")
    table.add_column("Active", justify="center")
    table.add_column("Created")

    for r in rows:
        active = "[green]✓[/green]" if r["is_active"] else "[red]✗[/red]"
        created = str(r["created_at"])[:16] if r["created_at"] else "-"
        acc_short = r["account_id"][:16] + "..." if len(r["account_id"]) > 16 else r["account_id"]
        table.add_row(
            r["email"], acc_short,
            str(r["token_count"]), str(r["worker_count"]),
            active, created,
        )
    console.print(table)


@account_app.command("rm")
def account_rm(
    email: Annotated[str, typer.Argument()],
) -> None:
    """Deactivate an account (local only)."""
    store = _store()
    if not store.get_account_by_email(email):
        err_console.print(f"[bold red]Account not found:[/bold red] {email}")
        raise typer.Exit(1)
    store.deactivate_account(email)
    console.print(f"[yellow]Deactivated:[/yellow] {email}")


# ---------------------------------------------------------------------------
# token
# ---------------------------------------------------------------------------

@token_app.command("create")
def token_create(
    name: Annotated[str, typer.Argument(help="Token name")],
    preset: Annotated[str, typer.Option("--preset", help="Permission preset")] = "all",
    account: Annotated[Optional[str], typer.Option("--account")] = None,
    no_headless: Annotated[bool, typer.Option("--no-headless")] = False,
    verbose: Annotated[bool, typer.Option("--verbose", "-v")] = False,
    set_default: Annotated[bool, typer.Option("--default")] = False,
) -> None:
    """Create a named API token via CF dashboard browser automation."""
    from .browser import create_token_via_browser, PRESETS

    if preset not in PRESETS:
        err_console.print(
            f"[bold red]Unknown preset:[/bold red] {preset}. "
            f"Choose: {', '.join(PRESETS)}"
        )
        raise typer.Exit(1)

    store = _store()
    if account:
        acc = store.get_account_by_email(account)
        if not acc:
            err_console.print(f"[bold red]Account not found:[/bold red] {account}")
            raise typer.Exit(1)
    else:
        acc = store.get_first_active_account()
        if not acc:
            err_console.print("[bold red]No active accounts. Run:[/bold red] cloudflare-tool register")
            raise typer.Exit(1)

    console.print(f"[bold green]Creating token '{name}' (preset: {preset})...[/bold green]")
    console.print("[dim]Opening browser to Cloudflare API Tokens page...[/dim]")

    try:
        token_value = create_token_via_browser(
            email=acc["email"],
            password=acc["password"],
            token_name=name,
            preset=preset,
            headless=not no_headless,
            verbose=verbose,
        )
    except Exception as e:
        err_console.print(f"[bold red]Token creation failed:[/bold red] {e}")
        raise typer.Exit(1)

    store.add_token(
        account_id=acc["id"],
        name=name,
        token_value=token_value,
        preset=preset,
    )
    if set_default:
        store.set_default_token(name)
        _write_cf_json(store, name)

    console.print(f"[bold green]✓ Token created:[/bold green] {name}")
    console.print(f"[dim]Preset:[/dim] {preset}")
    if set_default:
        console.print(f"[green]Set as default. cloudflare.json updated.[/green]")


@token_app.command("ls")
def token_ls() -> None:
    """List all tokens."""
    store = _store()
    rows = store.list_tokens()
    if not rows:
        console.print("[yellow]No tokens. Run:[/yellow] cloudflare-tool token create <name>")
        return

    table = Table(title="API Tokens", show_lines=True)
    table.add_column("Name", style="cyan")
    table.add_column("Preset")
    table.add_column("Account")
    table.add_column("Default", justify="center")
    table.add_column("Created")

    for r in rows:
        default = "[bold green]●[/bold green]" if r["is_default"] else ""
        created = str(r["created_at"])[:16] if r["created_at"] else "-"
        name_str = f"[bold]{r['name']}[/bold]" if r["is_default"] else r["name"]
        table.add_row(name_str, r["preset"], r["email"], default, created)
    console.print(table)


@token_app.command("rm")
def token_rm(
    name: Annotated[str, typer.Argument()],
) -> None:
    """Remove a token from local state."""
    store = _store()
    if not store.get_token_by_name(name):
        err_console.print(f"[bold red]Token not found:[/bold red] {name}")
        raise typer.Exit(1)
    store.remove_token(name)
    console.print(f"[yellow]Removed:[/yellow] {name} [dim](local only)[/dim]")


@token_app.command("use")
def token_use(
    name: Annotated[str, typer.Argument()],
) -> None:
    """Set default token and write ~/data/cloudflare/cloudflare.json."""
    store = _store()
    tok = store.get_token_by_name(name)
    if not tok:
        err_console.print(f"[bold red]Token not found:[/bold red] {name}")
        raise typer.Exit(1)
    store.set_default_token(name)
    _write_cf_json(store, name)
    console.print(f"[green]Default set to:[/green] {name}")
    console.print(f"[dim]cloudflare.json written to:[/dim] {_CF_JSON}")


def _write_cf_json(store: Store, token_name: str) -> None:
    """Write ~/data/cloudflare/cloudflare.json for pkg/scrape compatibility."""
    tok = store.get_token_by_name(token_name)
    if not tok:
        return
    _CF_JSON.parent.mkdir(parents=True, exist_ok=True)
    _CF_JSON.write_text(json.dumps({
        "account_id": tok["account_id"],
        "api_token": tok["token_value"],
    }, indent=2))


# ---------------------------------------------------------------------------
# worker
# ---------------------------------------------------------------------------

@worker_app.command("deploy")
def worker_deploy(
    path: Annotated[str, typer.Argument(help="Path to Worker source or wrangler.toml dir")],
    name: Annotated[Optional[str], typer.Option("--name")] = None,
    alias: Annotated[Optional[str], typer.Option("--alias")] = None,
    token_name: Annotated[Optional[str], typer.Option("--token")] = None,
    set_default: Annotated[bool, typer.Option("--default")] = False,
) -> None:
    """Deploy a Worker via wrangler."""
    from .workers import deploy as _deploy

    # Derive name from path if not provided
    worker_name = name or Path(path).resolve().name
    worker_alias = alias or worker_name

    store = _store()

    # Resolve token
    if token_name:
        tok = store.get_token_by_name(token_name)
        if not tok:
            err_console.print(f"[bold red]Token not found:[/bold red] {token_name}")
            raise typer.Exit(1)
    else:
        tok = store.get_default_token()
        if not tok:
            err_console.print(
                "[bold red]No default token. Run:[/bold red] cloudflare-tool token use <name>"
            )
            raise typer.Exit(1)

    console.print(f"[bold green]Deploying '{worker_name}' from {path}...[/bold green]")

    t0 = _time.monotonic()
    try:
        url = _deploy(
            account_id=tok["account_id"],
            token=tok["token_value"],
            name=worker_name,
            path=path,
            subdomain=tok.get("subdomain", ""),
        )
    except Exception as e:
        err_console.print(f"[bold red]Deploy failed:[/bold red] {e}")
        raise typer.Exit(1)

    duration_ms = int((_time.monotonic() - t0) * 1000)

    # Store or update worker
    existing = store.get_worker(worker_alias)
    if existing:
        store.update_worker_url(worker_alias, url)
        w_id = existing["id"]
    else:
        # Get token DB id
        tok_row = store.get_token_by_name(tok["name"])
        w_id = store.add_worker(
            account_id=tok["account_db_id"],
            token_id=tok_row["id"] if tok_row else None,
            name=worker_name,
            alias=worker_alias,
            url=url,
        )

    store.log_op(worker_id=w_id, operation="deploy", detail=path, duration_ms=duration_ms)

    if set_default:
        store.set_default_worker(worker_alias)

    console.print(f"[bold green]✓ Deployed:[/bold green] {worker_name}")
    console.print(f"[dim]URL:[/dim] {url}")
    console.print(f"[dim]Alias:[/dim] {worker_alias}")
    if set_default:
        console.print("[green]Set as default.[/green]")


@worker_app.command("ls")
def worker_ls() -> None:
    """List all Workers."""
    store = _store()
    rows = store.list_workers()
    if not rows:
        console.print("[yellow]No workers. Run:[/yellow] cloudflare-tool worker deploy <path>")
        return

    table = Table(title="Workers", show_lines=True)
    table.add_column("Alias", style="cyan")
    table.add_column("Name")
    table.add_column("URL")
    table.add_column("Account")
    table.add_column("Default", justify="center")
    table.add_column("Ops", justify="right")
    table.add_column("Deployed")

    for r in rows:
        default = "[bold green]●[/bold green]" if r["is_default"] else ""
        deployed = str(r["deployed_at"])[:16] if r["deployed_at"] else "-"
        url_short = r["url"][:40] + "..." if len(r.get("url", "")) > 40 else r.get("url", "")
        alias_str = f"[bold]{r['alias']}[/bold]" if r["is_default"] else r["alias"]
        table.add_row(
            alias_str, r["name"], url_short, r["email"],
            default, str(r["op_count"]), deployed,
        )
    console.print(table)


@worker_app.command("rm")
def worker_rm(
    alias: Annotated[str, typer.Argument()],
) -> None:
    """Delete a Worker from Cloudflare and remove from local state."""
    from .client import CloudflareClient

    store = _store()
    w = store.get_worker(alias)
    if not w:
        err_console.print(f"[bold red]Worker not found:[/bold red] {alias}")
        raise typer.Exit(1)

    tok = store.get_default_token()
    if tok:
        try:
            client = CloudflareClient(
                account_id=tok["account_id"],
                api_token=tok["token_value"],
            )
            client.delete_worker(w["name"])
            client.close()
            console.print(f"[dim]Deleted from Cloudflare:[/dim] {w['name']}")
        except Exception as e:
            err_console.print(f"[yellow]CF delete warning:[/yellow] {e}")

    store.remove_worker(alias)
    console.print(f"[yellow]Removed:[/yellow] {alias}")


@worker_app.command("tail")
def worker_tail(
    alias: Annotated[str, typer.Argument()],
) -> None:
    """Stream real-time Worker logs via wrangler tail."""
    from .workers import tail as _tail

    store = _store()
    w = store.get_worker(alias)
    if not w:
        err_console.print(f"[bold red]Worker not found:[/bold red] {alias}")
        raise typer.Exit(1)

    tok = store.get_default_token()
    if not tok:
        err_console.print(
            "[bold red]No default token. Run:[/bold red] cloudflare-tool token use <name>"
        )
        raise typer.Exit(1)

    console.print(f"[bold green]Tailing '{w['name']}' (Ctrl+C to stop)...[/bold green]")
    _tail(
        account_id=tok["account_id"],
        token=tok["token_value"],
        name=w["name"],
    )


@worker_app.command("invoke")
def worker_invoke(
    alias: Annotated[str, typer.Argument()],
    path: Annotated[str, typer.Option("--path")] = "/",
    method: Annotated[str, typer.Option("--method")] = "GET",
    body: Annotated[Optional[str], typer.Option("--body")] = None,
    header: Annotated[Optional[list[str]], typer.Option("--header")] = None,
    json_out: Annotated[bool, typer.Option("--json")] = False,
) -> None:
    """Send an HTTP request to a Worker."""
    from .workers import invoke as _invoke

    store = _store()
    w = store.get_worker(alias)
    if not w:
        # Try default worker
        w = store.get_default_worker()
        if not w:
            err_console.print(f"[bold red]Worker not found:[/bold red] {alias}")
            raise typer.Exit(1)

    if not w.get("url"):
        err_console.print(f"[bold red]Worker has no URL:[/bold red] {alias}")
        raise typer.Exit(1)

    headers: dict[str, str] = {}
    for h in (header or []):
        if ":" in h:
            k, v = h.split(":", 1)
            headers[k.strip()] = v.strip()

    t0 = _time.monotonic()
    status, response_body = _invoke(
        url=w["url"], method=method,
        path=path, body=body or "",
        headers=headers,
    )
    duration_ms = int((_time.monotonic() - t0) * 1000)

    store.log_op(
        worker_id=w["id"], operation="invoke",
        detail=f"{method} {path} → {status}",
        duration_ms=duration_ms,
    )

    if json_out:
        print(json.dumps({
            "status": status, "body": response_body, "duration_ms": duration_ms
        }))
        return

    color = "green" if status < 400 else "red"
    from rich.panel import Panel
    console.print(Panel(
        response_body,
        title=f"[{color}]{status} {method} {path}[/{color}]  "
              f"[dim]{duration_ms}ms[/dim]",
        border_style=color,
    ))


@worker_app.command("use")
def worker_use(
    alias: Annotated[str, typer.Argument()],
) -> None:
    """Set the default Worker for invoke/tail."""
    store = _store()
    if not store.get_worker(alias):
        err_console.print(f"[bold red]Worker not found:[/bold red] {alias}")
        raise typer.Exit(1)
    store.set_default_worker(alias)
    console.print(f"[green]Default worker set to:[/green] {alias}")


# ---------------------------------------------------------------------------
# Entry point
# ---------------------------------------------------------------------------

def app_entry() -> None:
    app()


if __name__ == "__main__":
    app_entry()
