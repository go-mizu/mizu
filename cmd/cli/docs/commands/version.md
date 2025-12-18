# mizu version - Print version information

## Synopsis

```
mizu version [options]
```

## Description

Print version information including the Mizu CLI version, Go version used
to build it, git commit hash, and build timestamp.

## Options

| Flag     | Description                      |
|----------|----------------------------------|
| `--json` | Output version information as JSON |

## Examples

Print version:

```bash
mizu version
```

JSON output:

```bash
mizu version --json
```

## Output

### Normal Mode

```
mizu version v0.3.0
go version: go1.24.11
commit: abc1234
built: 2025-01-15T10:30:00Z
```

### JSON Mode

```json
{
  "version": "v0.3.0",
  "go_version": "go1.24.11",
  "commit": "abc1234",
  "built_at": "2025-01-15T10:30:00Z"
}
```

## See Also

- `mizu --help` - Main help
