# 0637: Crawler GUI Dashboard

Modern web-based dashboard for the Rust crawler, replacing the ratatui TUI when `--gui` is active. Serves a real-time monitoring page via an embedded axum server with SSE-powered live updates.

## Architecture

```
┌───────────────────────────────────────────────────────┐
│  crawler cc recrawl --file p:0 --gui                  │
│                                                       │
│  ┌──────────┐    Arc<Stats>    ┌────────────────────┐ │
│  │  Engine   │────────────────▶│  GUI Server (axum) │ │
│  │ (tokio)   │   (atomics)     │                    │ │
│  └──────────┘                  │  GET /             │ │
│                                │  → embedded HTML   │ │
│  stdout:                       │                    │ │
│  "Dashboard: http://0:9111"    │  GET /api/stats    │ │
│  critical errors to stderr     │  → SSE stream      │ │
│                                └────────────────────┘ │
└───────────────────────────────────────────────────────┘
```

- **axum** web server spawned as a tokio task (shares runtime with crawler engine)
- **SSE** (`GET /api/stats`) pushes JSON stat snapshots every 200ms
- **Single HTML file** embedded via `include_str!("../assets/dashboard.html")` — all CSS/JS inline
- **Zero external dependencies** on the frontend — no npm, no CDN, no build step
- **Port 9111** default, `--gui-port` to override
- **0.0.0.0** binding for remote server access

## CLI Changes

### New flags (both `hn recrawl` and `cc recrawl`)

```
--gui              Enable web GUI dashboard (disables TUI)
--gui-port <PORT>  GUI server port (default: 9111)
```

### Behavior when `--gui` is active

1. Suppress ratatui TUI (`no_tui = true` internally)
2. Start axum server on `0.0.0.0:{port}` before crawl begins
3. Print `Dashboard: http://{hostname}:{port}` to stdout (use system hostname)
4. Critical errors logged to stderr via tracing (WARN+ level)
5. On crawl completion: keep server alive 30s for viewing final stats, then exit
6. `--gui` and `--no-tui` are independent: `--gui` implies no TUI; `--no-tui` alone just disables TUI without starting GUI

## HTTP Endpoints

### `GET /` — Dashboard page

Returns the embedded HTML file (`Content-Type: text/html`). Single file containing all CSS and JS inline.

### `GET /api/stats` — SSE event stream

```
Content-Type: text/event-stream
Cache-Control: no-cache
Connection: keep-alive

data: {"ok":92431,"failed":8124,"timeout":56781,"skipped":42664,"total":200000,...}

data: {"ok":92500,...}
```

Pushes a full `StatsPayload` JSON object every 200ms. Multiple SSE clients supported (broadcast pattern via tokio watch channel).

### `GET /api/config` — Crawl configuration (one-shot)

Returns static crawl config JSON (engine, writer, workers, timeout, title, etc.). Called once on page load.

## SSE Payload (`StatsPayload`)

```rust
#[derive(Serialize)]
struct StatsPayload {
    // Core counters
    ok: u64,
    failed: u64,
    timeout: u64,
    skipped: u64,
    total: u64,
    total_seeds: u64,
    bytes_downloaded: u64,
    peak_rps: u64,
    elapsed_ms: u64,
    pass: u8,
    pass2_seeds: u64,

    // Error breakdown
    err_invalid_url: u64,
    err_dns: u64,
    err_conn: u64,
    err_tls: u64,
    err_other: u64,

    // Error sub-categories
    dns_nxdomain: u64,
    dns_malformed: u64,
    dns_other: u64,
    conn_refused: u64,
    conn_reset: u64,
    conn_eof: u64,
    conn_other: u64,
    timeout_connect: u64,
    timeout_response: u64,

    // HTTP status distribution
    status_2xx: u64,
    status_3xx: u64,
    status_4xx: u64,
    status_5xx: u64,

    // Domains
    domains_total: u64,
    domains_done: u64,
    domains_abandoned: u64,

    // System resources
    mem_rss_mb: u64,
    net_rx_bps: u64,
    net_tx_bps: u64,
    open_fds: u64,

    // Warnings (last 50)
    warnings: Vec<String>,

    // Done flag
    done: bool,
}
```

## Config Payload

```rust
#[derive(Serialize)]
struct ConfigPayload {
    title: String,
    engine: String,
    writer: String,
    workers: String, // "auto" or number
    timeout_ms: u64,
    retry_timeout_ms: u64,
    no_retry: bool,
}
```

## GUI Design — shadcn-Inspired

### Theme System

CSS custom properties on `[data-theme="light"]` and `[data-theme="dark"]`:

**Light mode** (default follows system preference):
- Background: `hsl(0, 0%, 100%)` (white)
- Card background: `hsl(0, 0%, 98%)` (gray-50)
- Card border: `hsl(0, 0%, 90%)` (gray-200)
- Text primary: `hsl(0, 0%, 9%)` (gray-900)
- Text muted: `hsl(0, 0%, 45%)` (gray-500)

**Dark mode**:
- Background: `hsl(0, 0%, 4%)` (gray-950)
- Card background: `hsl(0, 0%, 7%)` (gray-900)
- Card border: `hsl(0, 0%, 15%)` (gray-800)
- Text primary: `hsl(0, 0%, 95%)` (gray-50)
- Text muted: `hsl(0, 0%, 55%)` (gray-400)

**Accent colors** (same in both themes):
- OK/Success: `hsl(142, 71%, 45%)` (green-500)
- Failed/Error: `hsl(0, 84%, 60%)` (red-500)
- Timeout/Warning: `hsl(48, 96%, 53%)` (yellow-400)
- Skipped: `hsl(0, 0%, 55%)` (gray)
- Info/Throughput: `hsl(199, 89%, 48%)` (cyan-500)
- 3xx: `hsl(221, 83%, 53%)` (blue-500)

**Toggle**: Sun/moon icon button in header, persisted to `localStorage`.
**System default**: `prefers-color-scheme` media query on first visit.

### Layout

Desktop-first responsive grid. Cards use `border-radius: 0.75rem`, `border: 1px solid var(--border)`, subtle `box-shadow`.

```
┌─────────────────────────────────────────────────────────┐
│  Header: title (left), theme toggle + config (right)    │
├─────────────────────────────────────────────────────────┤
│                                                         │
│  ┌─ Requests ─────────┐  ┌─ Throughput ──────────────┐ │
│  │ OK      92,431 46.2%│  │  ▁▂▄▇█▆▃▁ (canvas chart) │ │
│  │ Failed   8,124  4.1%│  │                           │ │
│  │ Timeout 56,781 28.4%│  │  Avg 3,486/s  Peak 10,444│ │
│  │ Skipped 42,664 21.3%│  │  Elapsed 1m26s ETA 2m08s │ │
│  │ Total  200,000      │  └───────────────────────────┘ │
│  └─────────────────────┘                                │
│                                                         │
│  ┌─ Errors ─────┐ ┌─ HTTP ──────┐ ┌─ System ────────┐ │
│  │ DNS    6,012  │ │ 2xx  91,200 │ │ RAM  245 MB     │ │
│  │ Conn   1,841  │ │ 3xx   1,231 │ │ FDs  12,401     │ │
│  │ TLS      271  │ │ 4xx   2,100 │ │ Net  48 MB/s    │ │
│  │ Other      0  │ │ 5xx       0 │ │ Down 1.2 GB     │ │
│  └───────────────┘ └─────────────┘ └─────────────────┘ │
│                                                         │
│  ┌─ Progress ───────────────────────────────────────┐   │
│  │ ████████████████████░░░░░  46.2%  92k / 200k     │   │
│  └──────────────────────────────────────────────────┘   │
│                                                         │
│  ┌─ Event Log ──────────────────────────────────────┐   │
│  │ > engine: 200000 seeds, 75492 domains            │   │
│  │ > dns: dead.example.com (NXDOMAIN)               │   │
│  │ > abandoned slow.example.com (3 timeouts, 0 ok)  │   │
│  │ > pass 2: 43,560 retry URLs, timeout=15000ms     │   │
│  └──────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────┘
```

### Card Details

**Requests card**: Vertical list of counters with colored dots, count, and percentage. Percentage computed client-side from total. Numbers use thousands separators and animate on change (CSS `transition` on opacity for digit swap).

**Throughput card**: Canvas-based sparkline chart (last 120 samples at 200ms = 24s window). Avg RPS and Peak RPS below the chart. Elapsed and ETA with live countdown. Chart color uses accent gradient.

**Errors card**: Expandable sub-categories. Click "DNS 6,012" to see nxdomain/malformed/other breakdown. Uses `<details>` element for no-JS progressive enhancement.

**HTTP Status card**: Horizontal stacked bar showing 2xx/3xx/4xx/5xx proportions with colored segments. Counts below.

**System card**: RAM, FDs, network bandwidth (combined rx+tx), total bytes downloaded.

**Progress bar**: Full-width bar with percentage text, animated fill using CSS transitions. Shows pass indicator: "[Pass 1]" or "[Pass 2]" label when pass > 1.

**Event Log**: Scrollable container, auto-scroll to bottom on new events. Color-coded by prefix: dns (red), conn (orange), tls (yellow), engine/done (cyan), pass (purple), abandoned (red). Max 50 visible, newest at bottom.

### Typography

```css
font-family: ui-sans-serif, system-ui, -apple-system, "Segoe UI", sans-serif;
```

Monospace for numbers:
```css
font-family: ui-monospace, "SF Mono", "Cascadia Code", "Segoe UI Mono", monospace;
font-variant-numeric: tabular-nums;
```

### Responsive

- Desktop (>768px): 2-column grid for main cards, 3-column for errors/http/system
- Mobile (<768px): single-column stack (useful for checking from phone on same network)

## File Structure

```
crawler-cli/
  src/
    main.rs          # Add --gui, --gui-port flags; route to gui::spawn()
    gui.rs           # axum server, SSE handler, routes, stats serialization
    common.rs        # CrawlJobParams gets gui/gui_port fields
    cc.rs            # RecrawlArgs gets --gui, --gui-port
    hn.rs            # RecrawlArgs gets --gui, --gui-port
  assets/
    dashboard.html   # Single-file HTML+CSS+JS (~600-800 lines)
```

## New Dependencies (crawler-cli/Cargo.toml)

```toml
axum = { version = "0.8", features = ["macros"] }
axum-extra = { version = "0.10", features = ["typed-header"] }
tower-http = { version = "0.6", features = ["cors"] }
serde_json = "1"  # already transitive, make explicit
tokio = { version = "1", features = ["full", "signal"] }  # already present
hostname = "0.4"  # for printing dashboard URL
```

## Implementation Plan

### 1. Add CLI flags

Both `cc::RecrawlArgs` and `hn::RecrawlArgs`:
```rust
/// Enable web GUI dashboard (disables TUI)
#[arg(long)]
pub gui: bool,

/// GUI server port (default: 9111)
#[arg(long, default_value_t = 9111)]
pub gui_port: u16,
```

`CrawlJobParams`:
```rust
pub gui: bool,
pub gui_port: u16,
```

### 2. Create `gui.rs`

```rust
use axum::{Router, response::{Html, Sse, Json}};
use axum::extract::State;
use futures_util::stream;
use std::sync::Arc;
use crawler_lib::stats::Stats;

struct GuiState {
    stats: Arc<Stats>,
    config: ConfigPayload,
}

pub struct GuiConfig {
    pub title: String,
    pub engine: String,
    pub writer: String,
    pub workers: String,
    pub timeout_ms: u64,
    pub retry_timeout_ms: u64,
    pub no_retry: bool,
}

/// Spawn the GUI server as a tokio task. Returns the bound address.
pub async fn spawn(
    stats: Arc<Stats>,
    cfg: GuiConfig,
    port: u16,
) -> anyhow::Result<std::net::SocketAddr> {
    let state = Arc::new(GuiState { stats, config: cfg.into() });

    let app = Router::new()
        .route("/", get(index_handler))
        .route("/api/stats", get(sse_handler))
        .route("/api/config", get(config_handler))
        .with_state(state);

    let addr = std::net::SocketAddr::from(([0, 0, 0, 0], port));
    let listener = tokio::net::TcpListener::bind(addr).await?;
    let bound = listener.local_addr()?;

    tokio::spawn(async move {
        axum::serve(listener, app).await.ok();
    });

    Ok(bound)
}

async fn index_handler() -> Html<&'static str> {
    Html(include_str!("../assets/dashboard.html"))
}

async fn sse_handler(State(state): State<Arc<GuiState>>) -> Sse<impl Stream<Item = ...>> {
    // Interval stream at 200ms, reads Arc<Stats>, serializes to JSON
}

async fn config_handler(State(state): State<Arc<GuiState>>) -> Json<ConfigPayload> {
    Json(state.config.clone())
}
```

### 3. Wire into `common.rs`

In `run_crawl_job()`, before the TUI spawn block:

```rust
// GUI server (mutually exclusive with TUI)
if params.gui {
    let gui_cfg = gui::GuiConfig { ... };
    let addr = gui::spawn(live_stats.clone(), gui_cfg, params.gui_port).await?;
    let host = hostname::get()
        .map(|h| h.to_string_lossy().to_string())
        .unwrap_or_else(|_| "localhost".to_string());
    println!("Dashboard: http://{}:{}", host, addr.port());
}

// TUI (only if not GUI and not no_tui)
let tui_handle = if !params.gui && !params.no_tui {
    // ... existing TUI spawn ...
} else {
    None
};
```

After crawl completes (before exit):
```rust
if params.gui {
    println!("Crawl complete. Dashboard available for 30s...");
    tokio::time::sleep(Duration::from_secs(30)).await;
}
```

### 4. Create `assets/dashboard.html`

Single HTML file with inline `<style>` and `<script>`. Key JavaScript:

```javascript
// Theme toggle
const toggle = () => {
  const next = document.documentElement.dataset.theme === 'dark' ? 'light' : 'dark';
  document.documentElement.dataset.theme = next;
  localStorage.setItem('theme', next);
};

// SSE connection
const es = new EventSource('/api/stats');
let prev = null;

es.onmessage = (e) => {
  const s = JSON.parse(e.data);
  updateCounters(s);
  updateSparkline(s, prev);
  updateProgress(s);
  updateErrors(s);
  updateHttp(s);
  updateSystem(s);
  updateLog(s);
  prev = s;
};

// Sparkline: Canvas 2D, stores last 120 RPS samples
// RPS computed client-side: delta(total) / delta(elapsed_ms)
```

### 5. Build and deploy

```bash
# Build on server2 (includes new assets/dashboard.html in binary)
make build-on-server SERVER=2

# Deploy
make deploy-server2

# Verify
ssh server2 "~/bin/crawler cc recrawl --file p:0 --gui"
# Then open http://server2:9111 in browser
```

## Testing

1. **Unit test**: `gui.rs` — spawn server, GET `/`, verify 200 + HTML content
2. **Unit test**: `gui.rs` — GET `/api/config`, verify JSON shape
3. **Integration test**: spawn server with mock Stats, connect SSE, verify events arrive
4. **Manual test**: `crawler cc recrawl --file p:0 --gui` on server2, open in browser
5. **Theme test**: toggle dark/light, verify all cards readable
6. **Mobile test**: resize browser to <768px, verify single-column layout

## Edge Cases

- **Port conflict**: axum bind fails → print error to stderr, continue with no GUI (crawl still runs)
- **SSE reconnect**: browser auto-reconnects on disconnect (EventSource default behavior)
- **Multiple clients**: each SSE handler independently reads `Arc<Stats>` — no broadcast overhead needed since stats are atomic reads
- **Crawl completion**: `done` field in payload triggers client-side "Completed" state (disable ETA, show final stats, stop sparkline)
- **Pass transition**: client detects `pass` field change (1→2), clears sparkline history, shows "[Pass 2]" badge
