# mizu middleware - Explore available middlewares

## Synopsis

```
mizu middleware <subcommand> [options]
```

## Description

Browse and explore the catalog of available middlewares for Mizu applications.
Middlewares are organized by category.

## Subcommands

### ls, list

List all middlewares, optionally filtered by category.

```bash
mizu middleware ls [-c category]
```

| Flag           | Description           |
|----------------|-----------------------|
| `-c, --category` | Filter by category  |

### show

Show detailed information about a specific middleware.

```bash
mizu middleware show <name>
```

## Categories

| Category     | Description                               |
|--------------|-------------------------------------------|
| `security`   | Authentication, authorization, CORS, CSP  |
| `logging`    | Request logging, metrics, tracing         |
| `performance`| Compression, caching, rate limiting       |
| `validation` | Request validation, sanitization          |
| `utilities`  | Recovery, timeout, request ID             |

## Examples

List all middlewares:

```bash
mizu middleware ls
```

Filter by category:

```bash
mizu middleware ls -c security
```

Show middleware details:

```bash
mizu middleware show helmet
mizu middleware show cors
```

JSON output:

```bash
mizu middleware ls --json
mizu middleware show ratelimit --json
```

## Output

The `show` command displays:

- **Name**: Middleware identifier
- **Description**: What the middleware does
- **Category**: Which category it belongs to
- **Import**: Go import path
- **Quick Start**: Example usage code
- **Related**: Related middlewares

## See Also

- `mizu new` - Create projects with middlewares
- `mizu --help` - Main help
