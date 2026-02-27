# 0608 — `pkg/qlocal` Port + `search local` Integration Plan (with Current Status)

## Goal
Add a Go-native, qmd-inspired local markdown search system to the Search blueprint as:
- package: `blueprints/search/pkg/qlocal`
- CLI surface: `search local ...`

Target outcome (full parity end-state) includes:
- collections/context management
- update/indexing
- BM25 search
- vector search
- hybrid query with structured query support
- get / multi-get / ls
- status / cleanup
- model pull
- MCP server (stdio + HTTP + daemon stop/status)

## Current Turn Status (Implemented Now)
### Implemented in `pkg/qlocal`
- SQLite-backed store and schema
- YAML config for collections + contexts
- collection commands (add/list/remove/rename/show/include/exclude/update-cmd)
- context commands (add/list/rm) with path/virtual path resolution
- `update` indexing pipeline (content hashing, titles, soft deactivation)
- FTS5 search (`search`) with qmd-like lex parsing (phrases + negation + prefix terms)
- embeddings (`embed`) with pluggable backend:
  - deterministic hash-vector fallback (no external model required)
  - optional llama.cpp-backed embeddings via existing `pkg/llm`
- vector search (`vsearch`) over stored chunk embeddings (fallback or LLM embeddings)
- hybrid `query` with:
  - structured query support (`lex`/`vec`/`hyde`)
  - qmd parser parity for single-line typed queries and explicit `expand:` syntax
  - optional LLM query expansion + rerank (cached in `llm_cache`)
  - heuristic fallback when no LLM backend is configured
- `get`, `multi-get`, `ls`
- `status`, `cleanup`
- qmd-style model pull/cache (`pull`) for `hf:` GGUF URIs with etag-aware caching
- MCP-compatible JSON-RPC server surface (stdio framing + HTTP `/mcp` + `/health`) with qmd-style tools/resources
  - qmd-like MCP HTTP compatibility additions:
    - `mcp-session-id` header on `initialize` and echo on subsequent requests
    - `GET/HEAD /mcp` streamable-http compatibility subset endpoint
    - `POST /query` structured-search convenience endpoint
    - URL-encoded `qmd://...` resource URI decoding
    - `get` tool `path` argument alias
- output formatting helpers (`cli/json/csv/md/xml/files`) for search and multi-get
- comprehensive tests (core integration, LLM expansion/rerank cache path, model pull cache, MCP HTTP/stdio, parser/glob, qmd structured-query parser parity)

### Integrated in CLI (`search local`)
Added new Cobra subtree in `blueprints/search/cli/local.go` and registered in `blueprints/search/cli/root.go`:
- `search local status`
- `search local collection ...`
- `search local context ...`
- `search local update`
- `search local embed`
- `search local search`
- `search local vsearch`
- `search local vector-search` (alias of `vsearch`)
- `search local query`
- `search local deep-search` (alias of `query`)
- `search local get`
- `search local multi-get`
- `search local ls`
- `search local cleanup`
- `search local pull`
- `search local mcp` (stdio / HTTP / daemon / stop)
- `search local collection set-update` (alias of `collection update-cmd`)

### Smoke-tested sample data
Used:
- `/Users/apple/github/rui314/8cc`
- `/Users/apple/Downloads/latex-theme-macos` (subfolder of `~/Downloads` as requested)

Observed successful flows:
- collection add
- context add
- update indexing
- status
- FTS search (`compiler`)
- get (`8cc/README.md`)
- multi-get (`8cc/*.md`)
- embed (direct package smoke runner)
- vector search (direct package smoke runner)
- hybrid query (direct package smoke runner)
- cleanup (direct package smoke runner)
- MCP HTTP/stdio protocol and tool/resource calls (automated tests)
- model pull etag cache behavior (automated tests with local HTTP stub)

## Remaining Differences vs qmd (Exact Behavior / Runtime Parity)
### 1. LLM runtime and quality parity is configuration-dependent
`pkg/qlocal` now supports optional llama.cpp-backed embeddings/query-expansion/reranking via the existing `pkg/llm` client, but exact qmd quality still depends on:
- running a compatible llama.cpp server
- providing equivalent models/prompts
- prompt/JSON parsing robustness vs qmd’s node-llama-cpp integration

Without an LLM backend configured, qlocal falls back to deterministic hash embeddings and heuristic reranking.

### 2. Vector backend differs (sqlite-vec vs BLOB + cosine in Go)
qmd uses `sqlite-vec` for ANN-style vector retrieval. qlocal currently stores vectors in SQLite BLOBs and performs cosine scoring in Go.

Impact:
- feature parity is present (`embed`/`vsearch`/`query`)
- performance/scaling parity with qmd is not guaranteed on larger corpora

### 3. MCP transport parity is implemented but not full Streamable HTTP
qlocal now exposes:
- stdio JSON-RPC with `Content-Length` framing
- HTTP `POST /mcp` JSON-RPC
- HTTP `GET/HEAD /mcp` compatibility subset
- `GET /health`
- `POST /query` structured-search convenience endpoint (qmd-style)

It does not yet implement qmd’s full Streamable HTTP semantics or every MCP SDK convenience behavior.

### 4. CLI/binary validation is intermittently blocked by unrelated repo state
During this implementation, unrelated in-progress code in other blueprint packages (`cli`, then later `pkg/recrawler`) intermittently blocked `go build ./cmd/search` / `go test ./cli`. `pkg/qlocal` itself compiles/tests cleanly, and protocol/search validation was completed at the package level.

## Architecture Chosen for Go Port (Current)
### Why this shape
A pragmatic staged port was used to make the qmd workflows usable immediately while isolating the hardest parity pieces (LLM + MCP).

### `pkg/qlocal` responsibilities
- config and collection/context management
- schema initialization and database operations
- indexing/update logic
- FTS/vector/hybrid search pipelines
- retrieval and formatting helpers

### CLI responsibilities (`search local`)
- command routing and flags
- output mode selection
- user-facing text/errors
- qmd-like command naming and aliases where practical

## Detailed Plan / Status Toward Full Parity
### Phase 1 (done): Core local retrieval baseline
- Implement `pkg/qlocal` schema + YAML config
- Add `search local` command family
- Ship FTS/get/multi-get/ls/status/update/cleanup
- Add deterministic local embeddings + vector/hybrid fallback
- Validate on real local sample directories

### Phase 2 (implemented): Optional real embedding backend
#### Option A (recommended in this repo)
Reuse existing `blueprints/search/pkg/llm` with llama.cpp server compatibility:
- add qlocal embedding provider config (URL/model)
- implement batch embedding for docs and queries
- store model name and embedding dimension in `content_vectors`
- expose clear error messages when embedding backend is unavailable

#### Option B
Integrate a Go-native embedding runtime/library (higher complexity/risk).

Status:
- Implemented pluggable backend with llama.cpp client integration + fallback.
- `embed`/`vsearch`/`query` now call backend embeddings when configured.
- Fallback remains enabled for offline/test use.

### Phase 3 (implemented): Query expansion + reranker hooks with cache
- Implement pluggable expansion backend (`lex`, `vec`, `hyde` variants)
- Implement reranker interface and cache table usage
- Add qmd-like `llm_cache` keying for expansion/rerank responses
- Preserve current heuristic fallback when LLM backend is not configured

Status:
- Implemented backend expansion/rerank hooks + `llm_cache` reuse.
- `query` supports both fallback and LLM-enhanced modes.
- Tests cover expansion/rerank calls + cache hit behavior.

### Phase 4 (implemented, compatibility subset): MCP server
- Implemented stdio JSON-RPC server with `Content-Length` framing
- Implemented HTTP `/mcp` JSON-RPC and `/health`
- Implemented `search local mcp --http --daemon` and `search local mcp stop` command paths
- Exposed qmd-style tools/resources:
  - `qmd_search`
  - `qmd_vector_search`
  - `qmd_deep_search`
  - `qmd_get`
  - `qmd_multi_get`
  - `qmd_status`
  - aliases (`query`, `get`, `multi_get`, `status`)
  - `qmd://...` resource reads

Remaining:
- full Streamable HTTP semantics and broader MCP method coverage

### Phase 5: Output and CLI polish parity pass
- Align help text/examples with qmd semantics
- Expand format parity coverage for `get` (if needed)
- improve `ls` rendering / filtering UX
- add richer status tips (stale index, missing embeddings)

## Feature Parity Matrix (Current)
| Feature | qmd | `pkg/qlocal` current | Status |
|---|---|---|---|
| Collections YAML config | Yes | Yes | Implemented |
| Context tree (global + path prefix) | Yes | Yes | Implemented |
| Content-addressable storage | Yes | Yes | Implemented |
| FTS5 BM25 search | Yes | Yes | Implemented |
| Lex query syntax (phrase/negation/prefix) | Yes | Yes (core support) | Implemented |
| `get` by path/docid | Yes | Yes | Implemented |
| `multi-get` glob/list | Yes | Yes | Implemented |
| `ls` | Yes | Yes | Implemented |
| `status` | Yes | Yes | Implemented |
| `update` | Yes | Yes | Implemented |
| `cleanup` | Yes | Yes | Implemented |
| Embeddings | Yes (LLM/GGUF) | Yes (llama.cpp optional + hash fallback) | Implemented / quality varies |
| Vector search | Yes | Yes (Go cosine over stored vectors) | Implemented / backend differs |
| Hybrid query (`query`) | Yes (expansion + rerank) | Yes (optional LLM expansion+rerank + fallback) | Implemented / quality varies |
| Structured query (`lex/vec/hyde`) | Yes | Yes (parser + execution) | Implemented |
| Model pull (`pull`) | Yes | Yes (`hf:` download + etag cache) | Implemented |
| MCP stdio/HTTP/daemon | Yes | Yes (JSON-RPC subset + daemon lifecycle) | Implemented / transport subset |

## Validation Plan (Remaining)
1. Add CLI command tests for `search local` once unrelated `cli` build break is resolved.
2. Add golden tests for JSON/CSV/XML/files outputs at CLI level (package formatters are already exercised indirectly).
3. Add benchmark coverage for indexing/query on larger corpora.
4. Add compatibility tests against real MCP clients (Claude/inspector) and a llama.cpp server.

## Validation Results From This Turn
### Automated
- `go test ./pkg/qlocal` ✅
- Added tests covering:
  - qmd structured-query parser conformance (single-line typed, explicit `expand:`, mixed-line errors, case-insensitive prefixes)
  - core integration/index/search/get/multi-get/ls/embed/query/cleanup
  - update deactivation behavior
  - LLM expansion+rerank cache flow (fake backend)
  - model pull etag cache behavior (httptest)
  - MCP HTTP `/mcp` + `/health` + `/query`, session header, encoded resource URIs, and stdio framing/tool/resource calls
- `go test ./cli` ✅ (alias/subcommand parity test coverage added in `cli/local_test.go`)

### Smoke results (summary)
- Indexed 8 markdown docs across `8cc` + `latex-theme-macos`
- FTS search returned `8cc/README.md` and `8cc/HACKING.md` for `compiler`
- `get` returned correct file content and context annotation
- `multi-get '8cc/*.md'` returned both markdown files
- Embedding pass completed (`8 docs`, `15 chunks`, `0 errors`) after fixing SQLite read/write deadlock
- Vector search and hybrid query returned sensible results with current hash-embedding fallback
- qlocal package smoke runner still passes after LLM/MCP/pull additions (`EMBED`, `STATUS`, `VSEARCH`, `QUERY`, `MULTIGET`, `CLEANUP`)

## Notes for Next Engineer
- `pull` and `mcp` are now implemented in `pkg/qlocal`, but qmd-exact transport/runtime behavior still differs (see “Remaining Differences vs qmd”).
- Resolve unrelated blueprint compile breaks (currently `pkg/recrawler`) before treating `go build ./cmd/search` as the final validation gate for `search local`.
- For best parity/quality, run a llama.cpp server and set `QLOCAL_LLAMACPP_URL` (plus optional model env vars).
