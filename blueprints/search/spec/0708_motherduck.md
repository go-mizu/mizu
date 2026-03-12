# MotherDuck Go CLI Integration + Binary Pack

## Goal

Mirror the clickhouse tool pattern for MotherDuck:
1. Pack `tools/motherduck` as a self-contained binary (`motherduck-tool`) via PyInstaller
2. Add `--json` to Python `register` command (stdout JSON, skip DuckDB)
3. Implement `search motherduck` Go CLI — Go handles state + queries; Python handles registration only

## Architecture

```
search motherduck register      → exec motherduck-tool register --json
                                   → parse JSON → store in pkg/motherduck/store (DuckDB)

search motherduck account ls    → pure Go
search motherduck account rm    → pure Go

search motherduck db create     → pure Go (DuckDB md: via duckdb-go driver)
search motherduck db ls         → pure Go
search motherduck db use        → pure Go
search motherduck db rm         → pure Go

search motherduck query <sql>   → pure Go (DuckDB md: connection)
```

## Python Side

### --json flag for register

Register with `--json` outputs to stdout:
```json
{"email": "...", "password": "...", "token": "..."}
```
Rich output goes to stderr. No DuckDB write. Same pattern as clickhouse.

### PyInstaller binary

Entry point: `motherduck_entry.py` (top-level, absolute imports — avoids relative import error under PyInstaller)

Binary: `dist/motherduck-tool` → installed to `~/bin/motherduck-tool`

### Makefile targets: `build`, `install`, `clean`, `test`

## Go Side

### File Structure

```
blueprints/search/
  cli/motherduck.go             # Cobra commands: NewMotherDuck()
  pkg/motherduck/
    store.go                   # DuckDB state (accounts, databases, query_log)
    client.go                  # DuckDB md: query client (duckdb-go driver)
    types.go                   # Account, Database, RegisterResult structs
```

### DuckDB Schema (same as Python store)

Path: `~/data/motherduck/mother.duckdb`

accounts: `id, email, password, token, created_at, is_active`
databases: `id, account_id, name, alias, is_default, created_at, last_used_at, notes`
query_log: `id, db_id, sql, rows_returned, duration_ms, ran_at`

### pkg/motherduck/client.go

Uses `duckdb-go/v2` driver with the `md:` MotherDuck protocol:

```go
db, err := sql.Open("duckdb", "md:"+dbName+"?motherduck_token="+token)
rows, err := db.QueryContext(ctx, sql)
```

MotherDuck auto-downloads and caches the `motherduck` extension at `~/.duckdb/extensions/` on first connect. Subsequent connects reuse the cached extension.

For `db create`: connects to `md:?motherduck_token=TOKEN` and runs `CREATE DATABASE IF NOT EXISTS name`.

For `query`: connects to `md:dbName?motherduck_token=TOKEN` and runs the SQL.

### pkg/motherduck/types.go

```go
type Account struct { ID, Email, Password, Token, CreatedAt string; IsActive bool; DBCount int }
type Database struct { ID, AccountID, Name, Alias, Email, Token, CreatedAt, LastUsedAt string; IsDefault bool; QueryCount int }
type RegisterResult struct { Email, Password, Token string }
```

### cli/motherduck.go Command Tree

```
search motherduck
  register [--no-headless] [--verbose]
  account
    ls
    rm <email>
  db
    create <name> [--alias a] [--account email] [--default]
    ls
    use <alias>
    rm <alias>
  query <sql> [--db alias] [--json]
```

Binary discovery: `$MOTHERDUCK_TOOL` → `~/bin/motherduck-tool` → PATH.

## Binary Size Estimate

~65-70MB (no clickhouse-connect; same python/duckdb/patchright/faker base).

## Differences from ClickHouse

| Aspect | ClickHouse | MotherDuck |
|--------|-----------|------------|
| Query protocol | HTTPS (net/http) | DuckDB md: (duckdb-go driver) |
| Auth | host+port+user+password | token |
| Resource unit | service | database |
| State file | ~/data/clickhouse/clickhouse.duckdb | ~/data/motherduck/mother.duckdb |
| Binary | ~/bin/clickhouse-tool | ~/bin/motherduck-tool |
