//! Web GUI dashboard — axum server with SSE-powered real-time stats.
//!
//! When `--gui` is active, an axum server is spawned as a tokio task serving:
//!   GET /              — embedded HTML dashboard
//!   GET /api/stats     — SSE stream (200ms interval, JSON stats snapshots)
//!   GET /api/config    — one-shot JSON with crawl configuration

use std::convert::Infallible;
use std::sync::atomic::Ordering;
use std::sync::Arc;
use std::time::Duration;

use axum::extract::State;
use axum::response::sse::{Event, KeepAlive, Sse};
use axum::response::{Html, Json};
use axum::routing::get;
use axum::Router;
use futures_util::stream::Stream;
use serde::Serialize;
use tower_http::cors::CorsLayer;

use crawler_lib::stats::Stats;

// ---------------------------------------------------------------------------
// Public API
// ---------------------------------------------------------------------------

/// Configuration passed from the CLI so the GUI can display engine/writer/workers.
pub struct GuiConfig {
    pub title: String,
    pub engine: String,
    pub writer: String,
    pub workers: String,
    pub timeout_ms: u64,
    pub retry_timeout_ms: u64,
    pub no_retry: bool,
}

struct GuiState {
    stats: Arc<Stats>,
    config: ConfigPayload,
}

/// Spawn the GUI server as a tokio task. Returns the bound address.
pub async fn spawn(
    stats: Arc<Stats>,
    cfg: GuiConfig,
    port: u16,
) -> anyhow::Result<std::net::SocketAddr> {
    let state = Arc::new(GuiState {
        stats,
        config: ConfigPayload {
            title: cfg.title,
            engine: cfg.engine,
            writer: cfg.writer,
            workers: cfg.workers,
            timeout_ms: cfg.timeout_ms,
            retry_timeout_ms: cfg.retry_timeout_ms,
            no_retry: cfg.no_retry,
        },
    });

    let app = Router::new()
        .route("/", get(index_handler))
        .route("/api/stats", get(sse_handler))
        .route("/api/config", get(config_handler))
        .layer(CorsLayer::permissive())
        .with_state(state);

    let addr = std::net::SocketAddr::from(([0, 0, 0, 0], port));
    let listener = tokio::net::TcpListener::bind(addr).await?;
    let bound = listener.local_addr()?;

    tokio::spawn(async move {
        axum::serve(listener, app).await.ok();
    });

    Ok(bound)
}

// ---------------------------------------------------------------------------
// Handlers
// ---------------------------------------------------------------------------

async fn index_handler() -> Html<&'static str> {
    Html(include_str!("../assets/dashboard.html"))
}

async fn config_handler(State(state): State<Arc<GuiState>>) -> Json<ConfigPayload> {
    Json(state.config.clone())
}

async fn sse_handler(
    State(state): State<Arc<GuiState>>,
) -> Sse<impl Stream<Item = Result<Event, Infallible>>> {
    let stats = state.stats.clone();

    let stream = async_stream::stream! {
        let mut interval = tokio::time::interval(Duration::from_millis(200));
        loop {
            interval.tick().await;
            let payload = snapshot_stats(&stats);
            let json = serde_json::to_string(&payload).unwrap_or_default();
            yield Ok(Event::default().data(json));

            if payload.done {
                // Keep sending for a bit after done so late-connecting clients see final state
                tokio::time::sleep(Duration::from_secs(1)).await;
            }
        }
    };

    Sse::new(stream).keep_alive(KeepAlive::default())
}

// ---------------------------------------------------------------------------
// Payloads
// ---------------------------------------------------------------------------

#[derive(Serialize, Clone)]
pub struct ConfigPayload {
    pub title: String,
    pub engine: String,
    pub writer: String,
    pub workers: String,
    pub timeout_ms: u64,
    pub retry_timeout_ms: u64,
    pub no_retry: bool,
}

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

fn snapshot_stats(stats: &Stats) -> StatsPayload {
    let warnings: Vec<String> = if let Ok(w) = stats.warnings.lock() {
        let skip = w.len().saturating_sub(50);
        w.iter().skip(skip).cloned().collect()
    } else {
        vec![]
    };

    StatsPayload {
        ok: stats.ok.load(Ordering::Relaxed),
        failed: stats.failed.load(Ordering::Relaxed),
        timeout: stats.timeout.load(Ordering::Relaxed),
        skipped: stats.skipped.load(Ordering::Relaxed),
        total: stats.total.load(Ordering::Relaxed),
        total_seeds: stats.total_seeds.load(Ordering::Relaxed),
        bytes_downloaded: stats.bytes_downloaded.load(Ordering::Relaxed),
        peak_rps: stats.peak_rps.load(Ordering::Relaxed),
        elapsed_ms: stats.start.elapsed().as_millis() as u64,
        pass: stats.pass.load(Ordering::Relaxed),
        pass2_seeds: stats.pass2_seeds.load(Ordering::Relaxed),
        err_invalid_url: stats.err_invalid_url.load(Ordering::Relaxed),
        err_dns: stats.err_dns.load(Ordering::Relaxed),
        err_conn: stats.err_conn.load(Ordering::Relaxed),
        err_tls: stats.err_tls.load(Ordering::Relaxed),
        err_other: stats.err_other.load(Ordering::Relaxed),
        dns_nxdomain: stats.dns_nxdomain.load(Ordering::Relaxed),
        dns_malformed: stats.dns_malformed.load(Ordering::Relaxed),
        dns_other: stats.dns_other.load(Ordering::Relaxed),
        conn_refused: stats.conn_refused.load(Ordering::Relaxed),
        conn_reset: stats.conn_reset.load(Ordering::Relaxed),
        conn_eof: stats.conn_eof.load(Ordering::Relaxed),
        conn_other: stats.conn_other.load(Ordering::Relaxed),
        timeout_connect: stats.timeout_connect.load(Ordering::Relaxed),
        timeout_response: stats.timeout_response.load(Ordering::Relaxed),
        status_2xx: stats.status_2xx.load(Ordering::Relaxed),
        status_3xx: stats.status_3xx.load(Ordering::Relaxed),
        status_4xx: stats.status_4xx.load(Ordering::Relaxed),
        status_5xx: stats.status_5xx.load(Ordering::Relaxed),
        domains_total: stats.domains_total.load(Ordering::Relaxed),
        domains_done: stats.domains_done.load(Ordering::Relaxed),
        domains_abandoned: stats.domains_abandoned.load(Ordering::Relaxed),
        mem_rss_mb: stats.mem_rss_mb.load(Ordering::Relaxed),
        net_rx_bps: stats.net_rx_bps.load(Ordering::Relaxed),
        net_tx_bps: stats.net_tx_bps.load(Ordering::Relaxed),
        open_fds: stats.open_fds.load(Ordering::Relaxed),
        warnings,
        done: stats.done.load(Ordering::Relaxed),
    }
}
