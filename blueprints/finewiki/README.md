# FineWiki

A fast, read-only wiki viewer built on the FineWiki dataset. FineWiki serves Wikipedia articles directly from Parquet files using DuckDB, with server-side rendered pages and instant title search.

## Quick Start

```bash
# Install the CLI
make build

# Download Vietnamese Wikipedia data (~800 MB)
finewiki import vi

# Start the server
finewiki serve vi

# Open http://localhost:8080
```

## Commands

### `finewiki import <lang>`

Download the Parquet data for a language from HuggingFace.

```bash
finewiki import vi        # Vietnamese
finewiki import en        # English
finewiki import ja        # Japanese
finewiki import de        # German
```

**Options:**
- `--data <dir>` - Custom data directory (default: `$HOME/data/blueprint/finewiki`)

The import fetches data from [HuggingFace FineWiki dataset](https://huggingface.co/datasets/HuggingFaceFW/finewiki) and saves it to `<data>/<lang>/data.parquet`.

For private or rate-limited access, set `HF_TOKEN`:

```bash
export HF_TOKEN=your_token
finewiki import en
```

### `finewiki serve <lang>`

Start the web server for a specific language.

```bash
finewiki serve vi              # Vietnamese on :8080
finewiki serve en --addr :3000 # English on port 3000
```

**Options:**
- `--addr <addr>` - HTTP listen address (default: `:8080`)
- `--data <dir>` - Custom data directory (default: `$HOME/data/blueprint/finewiki`)

The server creates a DuckDB index at `<data>/<lang>/wiki.duckdb` on first run.

### `finewiki list`

List available or installed languages.

```bash
finewiki list              # All available languages from HuggingFace
finewiki list --installed  # Locally installed languages
```

## Data Structure

Data is organized per language in the data directory:

```
$HOME/data/blueprint/finewiki/
├── vi/
│   ├── data.parquet    # Vietnamese Wikipedia articles
│   └── wiki.duckdb     # Title search index
├── en/
│   ├── data.parquet
│   └── wiki.duckdb
└── ...
```

## Architecture

```
finewiki/
├── cmd/finewiki/     # CLI entry point
├── cli/              # Command implementations
│   ├── serve.go      # Web server command
│   ├── import.go     # Data import command
│   ├── list.go       # List languages command
│   └── views/        # HTML templates
├── app/web/          # HTTP handlers
├── feature/
│   ├── search/       # Title search service
│   └── view/         # Page view service
└── store/duckdb/     # DuckDB storage layer
```

### Design Principles

- **Parquet as source of truth**: No data transformation or ETL pipelines
- **Title-only search index**: Minimal storage, instant results
- **Per-language isolation**: Each language operates independently
- **Server-side rendering**: No JavaScript frameworks required
- **Single binary**: Everything embedded, zero runtime dependencies

## Development

```bash
# Run with hot reload
make run ARGS="serve vi"

# Run tests
make test

# Build binary to $HOME/bin
make build

# Clean data directory
make clean-data
```

## Requirements

- Go 1.22+
- CGO enabled (required for DuckDB)

## Performance

- Cold start: 1-3 seconds (index creation)
- Warm start: < 100ms
- Search latency: < 10ms (title prefix match)
- Memory: ~50 MB base + index size

## Dataset

FineWiki is based on the [FineWiki dataset](https://huggingface.co/datasets/HuggingFaceFW/finewiki) from HuggingFace, which provides cleaned Wikipedia dumps in Parquet format for 325 languages.

## License

MIT
