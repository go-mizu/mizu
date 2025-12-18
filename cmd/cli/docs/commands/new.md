# mizu new - Create a new project

## Synopsis

```
mizu new [path] [options]
```

## Description

Create a new Mizu project from a template. If no path is specified, the
current directory is used.

## Options

| Flag               | Description                                  |
|--------------------|----------------------------------------------|
| `-t, --template`   | Template to use (required unless --list)     |
| `--list`           | List available templates                     |
| `--force`          | Overwrite existing files                     |
| `--dry-run`        | Preview changes without writing files        |
| `--name`           | Project name (default: derived from path)    |
| `--module`         | Go module path (default: example.com/name)   |
| `--license`        | License identifier (default: MIT)            |
| `--var key=value`  | Template variable (repeatable)               |

## Templates

| Template   | Description                                |
|------------|--------------------------------------------|
| `minimal`  | Bare-bones single-file app                 |
| `api`      | REST API with feature-based layout         |
| `web`      | Server-rendered HTML application           |
| `live`     | Real-time app with SSE                     |
| `sync`     | CRDT-based collaborative app               |
| `contract` | Contract-first JSON-RPC service            |

## Examples

Create minimal project:

```bash
mizu new . --template minimal
```

Create API project in new directory:

```bash
mizu new ./myapp --template api
```

Preview template output:

```bash
mizu new ./myapp --template api --dry-run
```

List templates:

```bash
mizu new --list
```

Custom module path:

```bash
mizu new ./myapp --template api --module github.com/myorg/myapp
```

## See Also

- `mizu dev` - Run the project
- `mizu --help` - Main help
