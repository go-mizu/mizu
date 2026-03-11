# ClickHouse Go CLI Integration

## Goal

Integrate ClickHouse Cloud account management into the `search` CLI as `search clickhouse` subcommands. The Python tool handles browser automation; everything else (state management, query execution) is pure Go.

## Architecture

```
search clickhouse register      → exec clickhouse-tool register --json
                                   → parse JSON → store in pkg/clickhouse/store (DuckDB)

search clickhouse account ls    → pure Go, reads DuckDB
search clickhouse account rm    → pure Go, writes DuckDB

search clickhouse service ls    → pure Go, reads DuckDB
search clickhouse service use   → pure Go, writes DuckDB
search clickhouse service rm    → pure Go, writes DuckDB

search clickhouse query <sql>   → pure Go, HTTPS POST to ClickHouse Cloud
```

## File Structure

```
blueprints/search/
  cli/
    clickhouse.go              # Cobra commands: NewClickHouse()
  pkg/clickhouse/
    store.go                   # DuckDB state (accounts, services, query_log)
    client.go                  # HTTPS query client
    types.go                   # Account, Service structs
```

## DuckDB Schema

Mirrors the Python store exactly (same file path: `~/data/clickhouse/clickhouse.duckdb`).

**accounts table:**
```sql
CREATE TABLE IF NOT EXISTS accounts (
    id             VARCHAR PRIMARY KEY DEFAULT gen_random_uuid()::VARCHAR,
    email          VARCHAR NOT NULL UNIQUE,
    password       VARCHAR NOT NULL,
    org_id         VARCHAR DEFAULT '',
    api_key_id     VARCHAR DEFAULT '',
    api_key_secret VARCHAR DEFAULT '',
    created_at     TIMESTAMP DEFAULT now(),
    is_active      BOOLEAN DEFAULT true
);
```

**services table:**
```sql
CREATE TABLE IF NOT EXISTS services (
    id           VARCHAR PRIMARY KEY DEFAULT gen_random_uuid()::VARCHAR,
    account_id   VARCHAR NOT NULL REFERENCES accounts(id),
    cloud_id     VARCHAR DEFAULT '',
    name         VARCHAR NOT NULL,
    alias        VARCHAR NOT NULL UNIQUE,
    host         VARCHAR DEFAULT '',
    port         INTEGER DEFAULT 8443,
    db_user      VARCHAR DEFAULT 'default',
    db_password  VARCHAR DEFAULT '',
    provider     VARCHAR DEFAULT 'aws',
    region       VARCHAR DEFAULT 'us-east-1',
    is_default   BOOLEAN DEFAULT false,
    created_at   TIMESTAMP DEFAULT now(),
    last_used_at TIMESTAMP,
    notes        VARCHAR DEFAULT ''
);
```

**query_log table:**
```sql
CREATE TABLE IF NOT EXISTS query_log (
    id            VARCHAR PRIMARY KEY DEFAULT gen_random_uuid()::VARCHAR,
    service_id    VARCHAR REFERENCES services(id),
    sql           VARCHAR NOT NULL,
    rows_returned INTEGER,
    duration_ms   INTEGER,
    ran_at        TIMESTAMP DEFAULT now()
);
```

## pkg/clickhouse/types.go

```go
package clickhouse

type Account struct {
    ID           string
    Email        string
    Password     string
    OrgID        string
    APIKeyID     string
    APIKeySecret string
    IsActive     bool
    CreatedAt    string
    SvcCount     int
}

type Service struct {
    ID         string
    AccountID  string
    CloudID    string
    Name       string
    Alias      string
    Host       string
    Port       int
    DBUser     string
    DBPassword string
    Provider   string
    Region     string
    IsDefault  bool
    CreatedAt  string
    LastUsedAt string
    QueryCount int
    Email      string // joined from accounts
}

// RegisterResult is the JSON output from clickhouse-tool register --json
type RegisterResult struct {
    Email        string `json:"email"`
    Password     string `json:"password"`
    OrgID        string `json:"org_id"`
    APIKeyID     string `json:"api_key_id"`
    APIKeySecret string `json:"api_key_secret"`
    ServiceID    string `json:"service_id"`
    Host         string `json:"host"`
    Port         int    `json:"port"`
    DBPassword   string `json:"db_password"`
}
```

## pkg/clickhouse/store.go

Uses `database/sql` with `github.com/duckdb/duckdb-go/v2` driver. Single connection (`SetMaxOpenConns(1)`).

Key methods:
- `NewStore(path string) (*Store, error)` — open DuckDB, run schema migrations
- `Close() error`
- `AddAccount(r RegisterResult) (string, error)` — insert account + service atomically
- `ListAccounts() ([]Account, error)`
- `DeactivateAccount(email string) error`
- `ListServices() ([]Service, error)`
- `GetServiceByAlias(alias string) (*Service, error)`
- `GetDefaultService() (*Service, error)`
- `SetDefault(alias string) error` — transaction: clear all, set one
- `RemoveService(alias string) error`
- `TouchLastUsed(alias string) error`
- `LogQuery(serviceID, sql string, rows, durationMS int) error`

## pkg/clickhouse/client.go

HTTPS POST to ClickHouse Cloud. No external Go library needed — uses `net/http` with Basic auth.

ClickHouse HTTP interface: `POST https://{host}:{port}/?query={sql}` with Basic auth header. Response body is TSV rows (or JSON with `FORMAT JSONEachRow`).

We use `FORMAT JSONEachRow` to get structured output:

```go
func (c *Client) Query(sql string) ([]map[string]any, error) {
    fullSQL := sql + " FORMAT JSONEachRow"
    req, _ := http.NewRequest("POST", c.url+"/?query="+url.QueryEscape(fullSQL), nil)
    req.SetBasicAuth(c.user, c.password)
    resp, err := c.http.Do(req)
    // parse NDJSON (one JSON object per line)
}
```

Key methods:
- `NewClient(host string, port int, user, password string) *Client`
- `Query(sql string) ([]map[string]any, []string, error)` — returns rows + column order
- `Ping() error`

## cli/clickhouse.go

### Command tree

```
search clickhouse
  register [--no-headless] [--verbose]  exec binary, store result
  account
    ls                                  list accounts table
    rm <email>                          deactivate account
  service
    ls                                  list services table
    use <alias>                         set default
    rm <alias>                          remove from local state
  query <sql> [--service <alias>] [--json]  run SQL, display table
```

### register implementation

```go
func runClickHouseRegister(cmd *cobra.Command, args []string) error {
    // 1. Find binary
    binaryPath := findClickHouseToolBinary()  // ~/bin/clickhouse-tool

    // 2. Build exec args
    execArgs := []string{"register", "--json"}
    if noHeadless { execArgs = append(execArgs, "--no-headless") }
    if verbose    { execArgs = append(execArgs, "--verbose") }

    // 3. Run subprocess
    //    - stderr → os.Stderr (live progress output)
    //    - stdout → captured
    var stdout bytes.Buffer
    proc := exec.Command(binaryPath, execArgs...)
    proc.Stderr = os.Stderr
    proc.Stdout = &stdout
    if err := proc.Run(); err != nil {
        return fmt.Errorf("registration failed: %w", err)
    }

    // 4. Parse JSON
    var result clickhouse.RegisterResult
    if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
        return fmt.Errorf("bad JSON from clickhouse-tool: %w\n%s", err, stdout.String())
    }

    // 5. Store in DuckDB
    store := openStore()
    defer store.Close()
    if err := store.AddAccount(result); err != nil {
        return fmt.Errorf("store failed: %w", err)
    }

    // 6. Print summary
    fmt.Printf("Registered: %s\n", result.Email)
    if result.Host != "" {
        fmt.Printf("Service:    %s\n", result.Host)
    }
    return nil
}
```

### query implementation

Uses `pkg/clickhouse/client.go` HTTPS client directly. Displays results as a table (using `tabwriter`) or JSON with `--json`.

### Binary discovery

```go
func findClickHouseToolBinary() string {
    // 1. $CLICKHOUSE_TOOL env var
    if p := os.Getenv("CLICKHOUSE_TOOL"); p != "" { return p }
    // 2. ~/bin/clickhouse-tool
    home, _ := os.UserHomeDir()
    p := filepath.Join(home, "bin", "clickhouse-tool")
    if _, err := os.Stat(p); err == nil { return p }
    // 3. PATH lookup
    if p, err := exec.LookPath("clickhouse-tool"); err == nil { return p }
    return ""
}
```

If binary not found, print helpful error: `"clickhouse-tool binary not found. Build it: cd tools/clickhouse && make install"`.

## DuckDB Path

`~/data/clickhouse/clickhouse.duckdb` — same as Python tool. Go and Python can both read/write this file but **not simultaneously**. The Go CLI never runs concurrently with the Python tool.

## Testing

No tests required for CLI glue code. The store and client packages have straightforward CRUD and HTTP logic that's testable via integration (not mocked unit tests, given the complexity of DuckDB mocking).

## Registration in root.go

```go
root.AddCommand(NewClickHouse())
```

## Output Styling

Use existing `cli/ui.go` styles: `titleStyle`, `infoStyle`, `successStyle`, `errorStyle`, `warningStyle`.

For table output, use `text/tabwriter` (no external dependency). Columns: aligned with tab stops.
