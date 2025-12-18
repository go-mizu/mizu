# mizu contract - Work with service contracts

## Synopsis

```
mizu contract <subcommand> [options]
```

## Description

Discover, inspect, and call methods on Mizu contract-based services.
Supports multiple discovery protocols including JSON-RPC, OpenAPI, and OpenRPC.

## Subcommands

### ls, list

List all services and methods from a running server.

```bash
mizu contract ls [url] [--all]
```

| Flag    | Description                    |
|---------|--------------------------------|
| `--all` | Include deprecated methods     |

### show

Show detailed information about a specific method.

```bash
mizu contract show <method> [url] [--schema]
```

| Flag       | Description             |
|------------|-------------------------|
| `--schema` | Show full JSON schema   |

### call

Call a method with optional input data.

```bash
mizu contract call <method> [input] [url] [options]
```

| Flag         | Description                    |
|--------------|--------------------------------|
| `--id`       | Path parameter ID (for REST)   |
| `--timeout`  | Request timeout (default 30s)  |
| `-H`         | Add header (key:value)         |
| `--raw`      | Output raw response            |

### spec

Export the API specification (OpenAPI or OpenRPC).

```bash
mizu contract spec [url] [--format] [--pretty] [-o file]
```

| Flag       | Description                      |
|------------|----------------------------------|
| `--format` | Output format (openapi, openrpc) |
| `--pretty` | Pretty print JSON                |
| `-o`       | Write to file                    |

### types

List types and their JSON schemas.

```bash
mizu contract types [type] [url] [--schema]
```

| Flag       | Description             |
|------------|-------------------------|
| `--schema` | Show full JSON schema   |

## URL Resolution

Server URL is resolved in this order:

1. Positional argument (if starts with `http://` or `https://`)
2. `--url` flag
3. `MIZU_URL` environment variable
4. Default: `http://localhost:8080`

## Examples

List methods from local server:

```bash
mizu contract ls
```

Call a method:

```bash
mizu contract call todo.Create '{"title":"Buy milk"}'
```

Call with file input:

```bash
mizu contract call todo.Create @input.json
```

Call with stdin:

```bash
echo '{"title":"test"}' | mizu contract call todo.Create -
```

Export OpenAPI spec:

```bash
mizu contract spec --pretty > openapi.json
```

Use different server:

```bash
mizu contract ls http://api.example.com
MIZU_URL=http://api.example.com mizu contract ls
```

## See Also

- `mizu middleware` - Explore middlewares
- `mizu --help` - Main help
