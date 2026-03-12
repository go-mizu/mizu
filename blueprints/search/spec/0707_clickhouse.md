# ClickHouse Cloud Tool — Spec

**Date:** 2026-03-10
**Location:** `tools/clickhouse/`

---

## Overview

A UV + Python CLI tool that auto-registers ClickHouse Cloud accounts (via Patchright browser automation + mail.tm disposable email), manages cloud services, and queries data. State persisted in a local DuckDB file. Mirrors the `tools/motherduck/` architecture.

---

## Stack

| Concern | Choice | Reason |
|---|---|---|
| Python | UV (script/project mode) | Matches project convention |
| Browser automation | Patchright (real Chrome) | Matches motherduck; bypasses bot detection |
| Email | mail.tm API | Free disposable; already used in motherduck |
| CLI framework | Typer + Rich | Matches motherduck |
| State store | DuckDB (`$HOME/data/clickhouse/clickhouse.duckdb`) | Matches motherduck |
| Cloud DB client | `clickhouse-connect` | Official ClickHouse Python HTTP driver |
| Cloud API | ClickHouse Cloud REST API (`api.clickhouse.cloud/v1`) | Service management, API key generation |
| Fake identity | Faker | Matches motherduck |

---

## ClickHouse Cloud Architecture

### Authentication Model

ClickHouse Cloud has TWO layers of auth:

1. **Console auth** — email/password login to `console.clickhouse.cloud` (the web UI). Signup supports email/password, Google, Microsoft.
2. **API keys** — key_id + key_secret pairs created in Console → Settings → API Keys. Used as HTTP Basic Auth for the REST API (`api.clickhouse.cloud/v1`).
3. **Service credentials** — each ClickHouse service has a host, port, and `default` user password. These are returned when a service is created via API.

### Registration Flow

1. Open `console.clickhouse.cloud/signUp` in Patchright
2. Fill email + password (use `?with=email` to force email/password form)
3. Handle email verification if required (poll mail.tm)
4. Click through onboarding (org name, region, etc.)
5. Navigate to Settings → API Keys → create API key
6. Extract key_id + key_secret from the UI
7. Use the REST API from that point on (no more browser needed)

### Service Management (REST API)

Base: `https://api.clickhouse.cloud/v1`
Auth: HTTP Basic Auth — `key_id:key_secret`

Key endpoints:
- `GET /v1/organizations` → list orgs, get `organizationId`
- `POST /v1/organizations/{orgId}/services` → create service (returns host, port, password)
- `GET /v1/organizations/{orgId}/services` → list services
- `GET /v1/organizations/{orgId}/services/{serviceId}` → service details
- `DELETE /v1/organizations/{orgId}/services/{serviceId}` → delete service

Create service request:
```json
{
  "name": "my-service",
  "provider": "aws",
  "region": "us-east-1",
  "tier": "development"
}
```

Create service response (key fields):
```json
{
  "result": {
    "service": {
      "id": "...",
      "endpoints": [{"protocol": "nativesecure", "host": "xxx.clickhouse.cloud", "port": 9440}]
    },
    "password": "generated-password"
  }
}
```

---

## CLI Command Structure

Docker-style grouped subcommands (mirrors motherduck):

```
clickhouse register              # Browser signup → store account + API keys
clickhouse account ls            # Rich table: email, org_id, api_keys, services
clickhouse account rm <email>    # Soft-delete (set is_active=false)

clickhouse service create <name> # Create ClickHouse Cloud service via REST API
    --provider TEXT  (aws|gcp|azure; default: aws)
    --region TEXT    (default: us-east-1)
    --tier TEXT      (default: development)
    --alias TEXT     (local alias; default: name)
    --default        (set as default immediately)
clickhouse service ls            # Rich table: alias, name, host, account, default, queries
clickhouse service use <alias>   # Set default service
clickhouse service rm <alias>    # Remove from local state (optionally delete from cloud)

clickhouse query <sql>           # Run SQL, print Rich table
    --service TEXT  (alias; omit = use default)
    --json          (output raw JSON)
```

### Why "service" not "db"

ClickHouse Cloud provisions **services** (each is a full ClickHouse instance with its own host/port). You can create multiple databases within a single service. The primary resource to manage is the service.

---

## File Structure

```
tools/clickhouse/
  pyproject.toml
  src/clickhouse_tool/           # "clickhouse" is taken on PyPI; use clickhouse_tool
    __init__.py
    cli.py                       # Typer app
    email.py                     # Shared mail.tm client (copy from motherduck, or symlink)
    browser.py                   # Patchright signup: console.clickhouse.cloud flow
    store.py                     # Local DuckDB state
    cloud_api.py                 # ClickHouse Cloud REST API wrapper
    client.py                    # clickhouse-connect wrapper for queries
    identity.py                  # Faker identity generation
```

---

## Local State Schema (`clickhouse.duckdb`)

```sql
CREATE TABLE IF NOT EXISTS accounts (
    id          VARCHAR PRIMARY KEY DEFAULT gen_random_uuid()::VARCHAR,
    email       VARCHAR NOT NULL UNIQUE,
    password    VARCHAR NOT NULL,       -- console password
    org_id      VARCHAR DEFAULT '',     -- ClickHouse Cloud organization ID
    api_key_id  VARCHAR DEFAULT '',     -- API key ID
    api_key_secret VARCHAR DEFAULT '',  -- API key secret
    created_at  TIMESTAMP DEFAULT now(),
    is_active   BOOLEAN DEFAULT true
);

CREATE TABLE IF NOT EXISTS services (
    id           VARCHAR PRIMARY KEY DEFAULT gen_random_uuid()::VARCHAR,
    account_id   VARCHAR NOT NULL REFERENCES accounts(id),
    cloud_id     VARCHAR DEFAULT '',          -- ClickHouse Cloud service ID
    name         VARCHAR NOT NULL,            -- service name on ClickHouse Cloud
    alias        VARCHAR NOT NULL UNIQUE,     -- local alias
    host         VARCHAR DEFAULT '',          -- connection host
    port         INTEGER DEFAULT 8443,        -- connection port (HTTPS)
    db_user      VARCHAR DEFAULT 'default',   -- ClickHouse username
    db_password  VARCHAR DEFAULT '',          -- service password
    provider     VARCHAR DEFAULT 'aws',
    region       VARCHAR DEFAULT 'us-east-1',
    is_default   BOOLEAN DEFAULT false,
    created_at   TIMESTAMP DEFAULT now(),
    last_used_at TIMESTAMP,
    notes        VARCHAR DEFAULT ''
);

CREATE TABLE IF NOT EXISTS query_log (
    id            VARCHAR PRIMARY KEY DEFAULT gen_random_uuid()::VARCHAR,
    service_id    VARCHAR REFERENCES services(id),
    sql           VARCHAR NOT NULL,
    rows_returned INTEGER,
    duration_ms   INTEGER,
    ran_at        TIMESTAMP DEFAULT now()
);
```

---

## Registration Flow (`browser.py`)

1. Create `mail.tm` mailbox via API
2. Open `https://console.clickhouse.cloud/signUp?with=email` in Patchright
3. Fill email + password fields → submit
4. Handle email verification if needed (poll mail.tm for verification link)
5. Click through onboarding prompts (org setup, region, etc.)
6. Navigate to Settings → API Keys
7. Click "New API key" → fill name → generate
8. Extract key_id + key_secret from the displayed dialog
9. Use REST API: `GET /v1/organizations` to get org_id
10. Store `(email, password, org_id, api_key_id, api_key_secret)` in accounts table

---

## Cloud API Wrapper (`cloud_api.py`)

```python
class ClickHouseCloudAPI:
    BASE = "https://api.clickhouse.cloud/v1"

    def __init__(self, key_id: str, key_secret: str):
        self._auth = (key_id, key_secret)
        self._client = httpx.Client(base_url=self.BASE, auth=self._auth, timeout=30)

    def get_organizations(self) -> list[dict]: ...
    def create_service(self, org_id, name, provider, region, tier) -> dict: ...
    def list_services(self, org_id) -> list[dict]: ...
    def get_service(self, org_id, service_id) -> dict: ...
    def delete_service(self, org_id, service_id) -> None: ...
    def close(self): ...
```

---

## Query Client (`client.py`)

Uses `clickhouse-connect` for HTTP-based queries:

```python
import clickhouse_connect

class ClickHouseClient:
    def __init__(self, host, port, username, password):
        self._client = clickhouse_connect.get_client(
            host=host, port=port,
            username=username, password=password,
            secure=True,
        )

    def run_query(self, sql) -> tuple[list, list]:
        result = self._client.query(sql)
        return result.result_rows, result.column_names

    def create_db(self, name):
        self._client.command(f"CREATE DATABASE IF NOT EXISTS {name}")

    def close(self):
        self._client.close()
```

---

## Dependencies (`pyproject.toml`)

```toml
dependencies = [
    "typer>=0.12",
    "rich>=13.0",
    "patchright>=1.50",
    "duckdb>=1.2,<1.5",
    "httpx>=0.27",
    "faker>=33.0",
    "clickhouse-connect>=0.7",
]
```

---

## Testing

1. `test_store.py` — unit test schema init, account/service CRUD, default rotation (in-memory DuckDB)
2. `test_cloud_api.py` — unit test REST API wrapper with mocked httpx
3. `test_client.py` — unit test query client with mocked clickhouse-connect
4. Integration: `clickhouse register --no-headless` → `clickhouse service create test` → `clickhouse query "SELECT 1"`
