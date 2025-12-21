# FineWiki CLI Enhancement Specification

## Overview

This specification outlines the refactoring of the FineWiki CLI to improve developer experience, simplify configuration, and enable per-language data management.

## Goals

1. Simplify configuration by removing environment variables
2. Default data directory at `$HOME/data/blueprint/finewiki`
3. Per-language data organization (parquet + duckdb per language)
4. Clean CLI architecture with commands in `cli/` package
5. Move views to project root for cleaner structure
6. Enhanced import command for fetching per-language parquet files

## Data Directory Structure

```
$HOME/data/blueprint/finewiki/
├── vi/
│   ├── data.parquet    # Vietnamese parquet data
│   └── wiki.duckdb     # Vietnamese DuckDB index
├── en/
│   ├── data.parquet    # English parquet data
│   └── wiki.duckdb     # English DuckDB index
├── ja/
│   ├── data.parquet
│   └── wiki.duckdb
└── ...
```

### Rationale

- **Per-language isolation**: Each language operates independently
- **Easy management**: Add/remove languages by adding/removing directories
- **Fast startup**: Only load the language you need
- **Predictable paths**: No glob patterns needed, just `{lang}/data.parquet`

## CLI Commands

### `finewiki` (no args)
Shows help message with available commands.

### `finewiki serve [lang]`
Start the web server for a specific language.

```bash
finewiki serve vi              # Serve Vietnamese wiki on :8080
finewiki serve en --addr :3000 # Serve English wiki on port 3000
finewiki serve vi --data /custom/path  # Use custom data directory
```

**Flags:**
- `--addr` (default: `:8080`): HTTP listen address
- `--data` (default: `$HOME/data/blueprint/finewiki`): Base data directory

### `finewiki import <lang>`
Download parquet file for a specific language from HuggingFace.

```bash
finewiki import vi        # Download Vietnamese parquet
finewiki import en        # Download English parquet
finewiki import ja --data /custom/path
```

**Flags:**
- `--data` (default: `$HOME/data/blueprint/finewiki`): Base data directory

**HuggingFace URL Pattern:**
```
https://huggingface.co/datasets/HuggingFaceFW/finewiki/resolve/main/data/{lang}/train-00000-of-00001.parquet
```

Note: Larger languages may have multiple shards. The import command will detect and download all shards.

### `finewiki list`
List available languages from the HuggingFace dataset.

```bash
finewiki list                  # Show all available languages
finewiki list --installed      # Show only installed languages
```

## Environment Variables

### Removed
- `FINEWIKI_DUCKDB` - replaced by per-language structure
- `FINEWIKI_PARQUET` - replaced by per-language structure
- `FINEWIKI_DATA` - replaced by `--data` flag
- `FINEWIKI_FTS` - FTS always enabled

### Kept
- `HF_TOKEN` - HuggingFace authentication token (optional)

## File Structure Changes

### Before
```
cmd/finewiki/
├── main.go
└── views/
    ├── layout/
    ├── component/
    └── page/
```

### After
```
cmd/finewiki/
└── main.go          # Minimal entry point

cli/
├── root.go          # Root command with help
├── serve.go         # Serve command
├── import.go        # Import command
├── list.go          # List command
├── config.go        # Shared configuration
└── templates.go     # Template loading

views/               # Moved to project root
├── layout/
├── component/
└── page/
```

## Files to Remove

- `app/web/middleware.go` - Logging middleware inline in server
- `app/web/middleware_test.go` - No longer needed
- `feature/search/service_test.go` - Mock-based tests not valuable
- `feature/view/service_test.go` - Mock-based tests not valuable
- `store/duckdb/store_test.go` - Mock-based tests not valuable

## Implementation Steps

### Phase 1: File Structure
1. Move `cmd/finewiki/views/` to `views/`
2. Create `cli/` package
3. Remove unnecessary files

### Phase 2: CLI Refactoring
4. Create `cli/config.go` with shared config
5. Create `cli/root.go` with root command
6. Create `cli/serve.go` with serve command
7. Create `cli/import.go` with import command
8. Create `cli/list.go` with list command
9. Create `cli/templates.go` for template loading
10. Refactor `main.go` to use cli package

### Phase 3: Store Updates
11. Update `store/duckdb/store.go` for per-language paths
12. Update `store/duckdb/import.go` for HuggingFace URL patterns

### Phase 4: Inline Middleware
13. Move logging middleware inline to `app/web/server.go`

### Phase 5: Documentation
14. Update `Makefile`
15. Update `README.md`

## API Compatibility

The web handlers and store interfaces remain unchanged. Only the CLI layer and configuration are affected.

## Testing

Manual testing workflow:
```bash
# Import Vietnamese data
finewiki import vi

# Start server
finewiki serve vi

# Open browser
open http://localhost:8080
```
