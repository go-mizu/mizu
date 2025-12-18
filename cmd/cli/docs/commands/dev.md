# mizu dev - Run development server

## Synopsis

```
mizu dev [options] [-- args...]
```

## Description

Run the current project in development mode. Automatically discovers the
main package in `cmd/*` or the current directory, builds it, and executes
until interrupted with Ctrl+C.

## Options

| Flag     | Description                               |
|----------|-------------------------------------------|
| `--cmd`  | Explicit main package path                |
| `--json` | Emit lifecycle events as JSON             |

## Discovery

The command searches for a runnable main package in this order:

1. Explicit `--cmd` path if provided
2. First directory in `cmd/` containing a main package
3. Current directory if it contains a main package

## Signals

| Signal               | Behavior                                  |
|----------------------|-------------------------------------------|
| `SIGINT` (Ctrl+C)    | Graceful shutdown with 15s timeout        |
| `SIGTERM`            | Graceful shutdown with 15s timeout        |

## Examples

Auto-discover and run:

```bash
mizu dev
```

Explicit main package:

```bash
mizu dev --cmd ./cmd/server
```

Pass arguments to application:

```bash
mizu dev -- --port 3000 --debug
```

JSON lifecycle events:

```bash
mizu dev --json
```

## JSON Output

When using `--json`, the command emits lifecycle events:

| Event      | Description                    |
|------------|--------------------------------|
| `starting` | About to run the package       |
| `started`  | Process started with PID       |
| `signal`   | Signal received                |
| `stopped`  | Process exited                 |
| `error`    | Error occurred                 |

Example output:

```json
{"event":"starting","timestamp":"2025-01-15T10:30:00Z","message":"running ./cmd/api"}
{"event":"started","timestamp":"2025-01-15T10:30:01Z","message":"pid 12345"}
{"event":"signal","timestamp":"2025-01-15T10:35:00Z","message":"interrupt"}
{"event":"stopped","timestamp":"2025-01-15T10:35:01Z","message":"clean exit"}
```

## See Also

- `mizu new` - Create new projects
- `mizu --help` - Main help
