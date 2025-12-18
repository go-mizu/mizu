# mizu - Project Toolkit for go-mizu Framework

## Synopsis

```
mizu [global options] <command> [command options] [arguments...]
```

## Description

**mizu** is the official CLI toolkit for the go-mizu web framework. It provides
commands for scaffolding new projects, running development servers, working with
service contracts, and exploring available middlewares.

## Commands

| Command      | Description                              |
|--------------|------------------------------------------|
| `new`        | Create a new project from a template     |
| `dev`        | Run the current project in development   |
| `contract`   | Work with service contracts              |
| `middleware` | Explore available middlewares            |
| `version`    | Print version information                |
| `completion` | Generate shell completion scripts        |

## Global Options

| Flag          | Description                        |
|---------------|------------------------------------|
| `--json`      | Emit machine-readable JSON output  |
| `--no-color`  | Disable colored output             |
| `-q, --quiet` | Reduce output to errors only       |
| `-v`          | Increase verbosity (repeatable)    |
| `--md`        | Print help as rendered markdown    |
| `-h, --help`  | Show help for any command          |

## Quick Start

Create a new API project:

```bash
mizu new ./myapp --template api
cd myapp
mizu dev
```

## Examples

```bash
# Create minimal project
mizu new . --template minimal

# Run development server
mizu dev

# List contract methods
mizu contract ls

# Show middleware details
mizu middleware show cors

# Generate shell completions
mizu completion bash > ~/.bash_completion.d/mizu
```

## Environment Variables

| Variable   | Description                                          |
|------------|------------------------------------------------------|
| `MIZU_URL` | Default server URL for contract commands             |
| `NO_COLOR` | Disable colors when set to any value                 |

## See Also

- `mizu new --help` - Project scaffolding
- `mizu dev --help` - Development server
- `mizu contract --help` - Contract operations
- `mizu middleware --help` - Middleware catalog

## Authors

The go-mizu contributors <https://github.com/go-mizu/mizu>

## Reporting Bugs

Report bugs at <https://github.com/go-mizu/mizu/issues>
