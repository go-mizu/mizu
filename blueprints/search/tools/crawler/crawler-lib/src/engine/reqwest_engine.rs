use crate::config::Config;
use crate::domain::{group_by_domain, DomainState};
use crate::stats::{AdaptiveTimeout, PeakTracker, Stats, StatsSnapshot};
use crate::types::{CrawlResult, FailedURL, SeedURL};
use crate::ua;
use crate::writer::{FailureWriter, ResultWriter};
use anyhow::Result;
use dashmap::DashMap;
use std::sync::atomic::{AtomicBool, AtomicU64, Ordering};
use std::sync::Arc;
use std::time::{Duration, Instant};
use tracing::{debug, info};

/// Reqwest-based crawl engine with flat URL task queue.
///
/// Architecture:
/// 1. Seeds are sorted and grouped by domain to create per-domain state entries.
/// 2. A flat URL channel is populated with (SeedURL, Arc<DomainEntry>) pairs.
/// 3. N worker tasks drain from the URL channel, each processing one URL at a time.
/// 4. Per-domain semaphore (capacity=inner_n) limits concurrency per domain without
///    blocking the worker — tokio suspends the task, not the thread.
/// 5. Workers are never idle between domain boundaries (no domain-batch gaps).
pub struct ReqwestEngine;

impl ReqwestEngine {
    pub fn new() -> Self {
        Self
    }
}

/// Per-domain state shared across all workers fetching from the same domain.
/// The semaphore limits concurrency to inner_n; abandoned flag short-circuits remaining URLs.
struct DomainEntry {
    semaphore: tokio::sync::Semaphore,
    abandoned: AtomicBool,
    ok: AtomicU64,
    timeouts: AtomicU64,
}

impl DomainEntry {
    fn new(inner_n: usize) -> Self {
        Self {
            semaphore: tokio::sync::Semaphore::new(inner_n.max(1)),
            abandoned: AtomicBool::new(false),
            ok: AtomicU64::new(0),
            timeouts: AtomicU64::new(0),
        }
    }
}

#[async_trait::async_trait]
impl super::Engine for ReqwestEngine {
    async fn run(
        &self,
        seeds: Vec<SeedURL>,
        cfg: &Config,
        results: Arc<dyn ResultWriter>,
        failures: Arc<dyn FailureWriter>,
    ) -> Result<StatsSnapshot> {
        let total_seeds = seeds.len();
        if total_seeds == 0 {
            return Ok(StatsSnapshot {
                ok: 0,
                failed: 0,
                timeout: 0,
                skipped: 0,
                bytes_downloaded: 0,
                total: 0,
                duration: Duration::ZERO,
                peak_rps: 0,
            });
        }

        info!(
            "reqwest engine: {} seeds, {} workers, inner_n={}",
            total_seeds, cfg.workers, cfg.inner_n
        );

        // Group seeds by domain to create per-domain state entries.
        let batches = group_by_domain(seeds);
        let domain_count = batches.len();
        info!("grouped into {} domains", domain_count);

        // Shared stats — use caller-provided Arc for live TUI display, or create fresh.
        // Only set total_seeds if not already populated (pass 1 sets it; pass 2 reuses).
        let stats = cfg.live_stats.clone().unwrap_or_else(|| Arc::new(Stats::new()));
        if stats.total_seeds.load(Ordering::Relaxed) == 0 {
            stats.total_seeds.store(total_seeds as u64, Ordering::Relaxed);
        }
        let adaptive = Arc::new(AdaptiveTimeout::new());
        let peak = Arc::new(PeakTracker::new());

        let workers = cfg.workers.max(1);
        let inner_n = cfg.inner_n.max(1);

        // Pre-create per-domain entries (semaphore + abandonment state).
        let domain_map: Arc<DashMap<String, Arc<DomainEntry>>> =
            Arc::new(DashMap::with_capacity(domain_count));
        for batch in &batches {
            domain_map.insert(
                batch.domain.clone(),
                Arc::new(DomainEntry::new(inner_n)),
            );
        }

        // Flat URL channel: capacity = workers * 4 so producer never blocks on startup.
        // Each item carries the URL and a reference to its domain's shared state.
        let (url_tx, url_rx) =
            async_channel::bounded::<(SeedURL, Arc<DomainEntry>)>(workers * 4);

        // Producer: flatten all domain batches into the URL channel.
        let dm = Arc::clone(&domain_map);
        let producer = tokio::spawn(async move {
            for batch in batches {
                if let Some(entry_ref) = dm.get(&batch.domain) {
                    let entry = Arc::clone(entry_ref.value());
                    for url in batch.urls {
                        if url_tx.send((url, Arc::clone(&entry))).await.is_err() {
                            return; // receivers all dropped
                        }
                    }
                }
            }
            // url_tx dropped here → channel closes when all receivers see EOF
        });

        // Build ONE shared reqwest::Client for all workers.
        // reqwest::Client is Arc-backed internally; cloning just bumps the refcount.
        let max_timeout = cfg.timeout.saturating_mul(5); // adaptive cap ceiling
        let shared_client = match reqwest::Client::builder()
            .pool_max_idle_per_host(inner_n)
            .timeout(max_timeout)
            .danger_accept_invalid_certs(true)
            .redirect(reqwest::redirect::Policy::limited(7))
            .tcp_keepalive(std::time::Duration::from_secs(60))
            .build()
        {
            Ok(c) => Arc::new(c),
            Err(e) => return Err(anyhow::anyhow!("failed to build reqwest client: {}", e)),
        };

        // Spawn N worker tasks.
        // Each worker loops: pop URL → acquire domain semaphore → fetch → update state.
        // Workers are never idle between URLs (no domain-batch boundaries).
        let mut worker_handles = Vec::with_capacity(workers);

        for _ in 0..workers {
            let rx = url_rx.clone();
            let cfg = cfg.clone();
            let results = Arc::clone(&results);
            let failures = Arc::clone(&failures);
            let stats = Arc::clone(&stats);
            let adaptive = Arc::clone(&adaptive);
            let peak = Arc::clone(&peak);
            let client = Arc::clone(&shared_client);

            let handle = tokio::spawn(async move {
                while let Ok((seed, domain_entry)) = rx.recv().await {
                    process_one_url(
                        seed,
                        &domain_entry,
                        &cfg,
                        &adaptive,
                        inner_n,
                        &client,
                        &results,
                        &failures,
                        &stats,
                        &peak,
                    )
                    .await;
                }
            });
            worker_handles.push(handle);
        }

        // Wait for producer to finish sending all URLs.
        let _ = producer.await;
        // Close channel so workers see EOF when all URLs are consumed.
        url_rx.close();

        // Wait for all workers to finish.
        for h in worker_handles {
            let _ = h.await;
        }

        // Update peak RPS in stats.
        stats.peak_rps.store(peak.peak(), Ordering::Relaxed);

        let snapshot = stats.snapshot();
        info!(
            "reqwest engine done: total={} ok={} failed={} timeout={} skipped={} peak_rps={} duration={:.1}s",
            snapshot.total,
            snapshot.ok,
            snapshot.failed,
            snapshot.timeout,
            snapshot.skipped,
            snapshot.peak_rps,
            snapshot.duration.as_secs_f64()
        );

        Ok(snapshot)
    }
}

/// Process a single URL using the shared reqwest client.
///
/// Acquires a per-domain semaphore permit before fetching (limits inner_n concurrency
/// per domain without blocking the worker task itself — tokio suspends and schedules
/// other tasks while waiting for the permit).
async fn process_one_url(
    seed: SeedURL,
    domain_entry: &Arc<DomainEntry>,
    cfg: &Config,
    adaptive: &Arc<AdaptiveTimeout>,
    inner_n: usize,
    client: &Arc<reqwest::Client>,
    results: &Arc<dyn ResultWriter>,
    failures: &Arc<dyn FailureWriter>,
    stats: &Arc<Stats>,
    peak: &Arc<PeakTracker>,
) {
    // Skip if domain has been abandoned (dead/stalling).
    if domain_entry.abandoned.load(Ordering::Relaxed) {
        stats.skipped.fetch_add(1, Ordering::Relaxed);
        let _ = failures.write_url(FailedURL::new(
            &seed.url,
            &seed.domain,
            "domain_http_timeout_killed",
        ));
        return;
    }

    // Acquire per-domain concurrency permit.
    // tokio suspends this task (not the thread) if inner_n fetches are already in flight.
    let _permit = match domain_entry.semaphore.acquire().await {
        Ok(p) => p,
        Err(_) => return, // semaphore closed (should not happen)
    };

    // Compute effective timeout (adaptive or fixed, capped at 5× base).
    let effective_timeout = if !cfg.disable_adaptive_timeout {
        adaptive
            .timeout(cfg.adaptive_timeout_max)
            .unwrap_or(cfg.timeout)
            .min(cfg.timeout.saturating_mul(5))
    } else {
        cfg.timeout
    };

    // Fetch the URL.
    let result = fetch_one(client, &seed, effective_timeout, cfg.max_body_bytes).await;
    stats.total.fetch_add(1, Ordering::Relaxed);
    peak.record();

    // Classify result and update domain state.
    if !result.error.is_empty() {
        let is_timeout = result.error.contains("timeout")
            || result.error.contains("Timeout")
            || result.error.contains("deadline")
            || result.error.contains("timed out");

        if is_timeout {
            stats.timeout.fetch_add(1, Ordering::Relaxed);
            let t = domain_entry.timeouts.fetch_add(1, Ordering::Relaxed) + 1;
            let s = domain_entry.ok.load(Ordering::Relaxed);

            let ds = DomainState { successes: s, timeouts: t };
            if ds.should_abandon(
                cfg.domain_fail_threshold,
                cfg.domain_dead_probe,
                cfg.domain_stall_ratio,
                inner_n,
            ) {
                // Only emit warning on the first abandonment (swap returns old value).
                if !domain_entry.abandoned.swap(true, Ordering::Relaxed) {
                    debug!(
                        "abandoning domain {} (timeouts={}, ok={})",
                        seed.domain, t, s
                    );
                    stats.push_warning(format!(
                        "abandoned {} (timeouts={}, ok={})",
                        seed.domain, t, s
                    ));
                }
            }

            let _ = failures.write_url(FailedURL {
                url: seed.url.clone(),
                domain: seed.domain.clone(),
                reason: "http_timeout".to_string(),
                error: result.error.clone(),
                status_code: 0,
                fetch_time_ms: result.fetch_time_ms,
                detected_at: chrono::Utc::now().naive_utc(),
            });
        } else {
            stats.failed.fetch_add(1, Ordering::Relaxed);
            let _ = failures.write_url(FailedURL {
                url: seed.url.clone(),
                domain: seed.domain.clone(),
                reason: "http_error".to_string(),
                error: result.error.clone(),
                status_code: result.status_code,
                fetch_time_ms: result.fetch_time_ms,
                detected_at: chrono::Utc::now().naive_utc(),
            });
        }
    } else {
        stats.ok.fetch_add(1, Ordering::Relaxed);
        domain_entry.ok.fetch_add(1, Ordering::Relaxed);
        adaptive.record(result.fetch_time_ms);
    }

    stats
        .bytes_downloaded
        .fetch_add(result.content_length as u64, Ordering::Relaxed);
    let _ = results.write(result);
}

/// Fetch a single URL using the shared reqwest client.
///
/// Returns a CrawlResult with metadata extracted from HTML responses.
/// On error, returns an error result with the error message.
async fn fetch_one(
    client: &reqwest::Client,
    seed: &SeedURL,
    timeout: Duration,
    max_body_bytes: usize,
) -> CrawlResult {
    let start = Instant::now();

    let response = client
        .get(&seed.url)
        .header("User-Agent", ua::pick_user_agent())
        .timeout(timeout)
        .send()
        .await;

    let resp = match response {
        Ok(r) => r,
        Err(e) => {
            return CrawlResult::error_result(
                &seed.url,
                &seed.domain,
                e.to_string(),
                start.elapsed().as_millis() as i64,
            );
        }
    };

    let status = resp.status().as_u16();
    let content_type = resp
        .headers()
        .get("content-type")
        .and_then(|v| v.to_str().ok())
        .unwrap_or("")
        .to_string();
    let content_length = resp.content_length().unwrap_or(0) as i64;
    let redirect_url = resp
        .headers()
        .get("location")
        .and_then(|v| v.to_str().ok())
        .unwrap_or("")
        .to_string();

    // Read body (up to max_body_bytes)
    let body_bytes = match read_body_limited(resp, max_body_bytes).await {
        Ok(b) => b,
        Err(e) => {
            return CrawlResult::error_result(
                &seed.url,
                &seed.domain,
                e.to_string(),
                start.elapsed().as_millis() as i64,
            );
        }
    };

    let body_len = body_bytes.len() as i64;
    let is_html =
        content_type.contains("text/html") || content_type.contains("application/xhtml");

    let (title, description, language) = if status == 200 && is_html && !body_bytes.is_empty() {
        extract_metadata(&body_bytes)
    } else {
        (String::new(), String::new(), String::new())
    };

    CrawlResult {
        url: seed.url.clone(),
        domain: seed.domain.clone(),
        status_code: status,
        content_type,
        content_length: content_length.max(body_len),
        title,
        description,
        language,
        redirect_url,
        fetch_time_ms: start.elapsed().as_millis() as i64,
        crawled_at: chrono::Utc::now().naive_utc(),
        error: String::new(),
        body: String::new(), // always empty — avoids DuckDB overflow blocks
    }
}

/// Read response body up to `max_bytes`, stopping the download early.
///
/// Unlike `resp.bytes()` (which buffers the full body), this streams chunks
/// and stops reading once `max_bytes` are accumulated. This avoids downloading
/// multi-MB responses when we only need the first 256 KB for metadata extraction.
async fn read_body_limited(
    resp: reqwest::Response,
    max_bytes: usize,
) -> Result<bytes::Bytes, reqwest::Error> {
    use bytes::BytesMut;
    use futures_util::StreamExt;

    // Fast path: if Content-Length hints that body fits, use the simple path.
    if let Some(len) = resp.content_length() {
        if len as usize <= max_bytes {
            return resp.bytes().await;
        }
    }

    // Stream chunks, stop once we have enough.
    let mut stream = resp.bytes_stream();
    let mut buf = BytesMut::with_capacity(max_bytes.min(64 * 1024));
    while let Some(chunk) = stream.next().await {
        let chunk: bytes::Bytes = chunk?;
        let remaining = max_bytes.saturating_sub(buf.len());
        if remaining == 0 {
            break; // Drop remaining stream data (connection may be reused by keep-alive)
        }
        let take = chunk.len().min(remaining);
        buf.extend_from_slice(&chunk[..take]);
    }
    Ok(buf.freeze())
}

// ---------------------------------------------------------------------------
// Domain timeout helper (used by hyper_engine.rs too)
// ---------------------------------------------------------------------------

/// Calculate effective domain timeout.
///
/// - domain_timeout_ms < 0 (adaptive): len(urls) * timeout / inner_n * 2, clamped [5s, max]
/// - domain_timeout_ms > 0 (explicit): use as-is
/// - domain_timeout_ms == 0 (disabled): None
pub(crate) fn compute_domain_timeout(
    cfg: &Config,
    url_count: usize,
    inner_n: usize,
) -> Option<Duration> {
    if cfg.domain_timeout_ms == 0 {
        return None;
    }

    if cfg.domain_timeout_ms > 0 {
        return Some(Duration::from_millis(cfg.domain_timeout_ms as u64));
    }

    // Adaptive: estimate how long this domain should take
    // Formula: urls * timeout_ms / inner_n * 2, clamped [5s, adaptive_timeout_max]
    let timeout_ms = cfg.timeout.as_millis() as u64;
    let estimated_ms = url_count as u64 * timeout_ms / inner_n.max(1) as u64 * 2;
    let min_ms = 5_000u64;
    let max_ms = cfg.adaptive_timeout_max.as_millis() as u64;
    let clamped_ms = estimated_ms.max(min_ms).min(max_ms);

    Some(Duration::from_millis(clamped_ms))
}

// ---------------------------------------------------------------------------
// HTML metadata extraction (simple, no regex, no external parser)
// ---------------------------------------------------------------------------

/// Extract title, description, and language from an HTML body.
/// Only scans the first 64KB for performance.
pub(crate) fn extract_metadata(body: &[u8]) -> (String, String, String) {
    let html = String::from_utf8_lossy(body);
    let scan_limit = html.floor_char_boundary(html.len().min(64 * 1024));
    let html = &html[..scan_limit];

    let title = extract_tag_content(html, "<title", "</title>");
    let description = extract_meta_content(html, "description");
    let language = extract_lang_attr(html);

    (
        truncate_string(title, 512),
        truncate_string(description, 1024),
        truncate_string(language, 16),
    )
}

/// Extract text content between an opening tag and its closing tag.
/// e.g. `<title>Hello World</title>` -> "Hello World"
fn extract_tag_content(html: &str, open_tag: &str, close_tag: &str) -> String {
    let lower = html.to_lowercase();
    if let Some(start) = lower.find(open_tag) {
        let rest = &html[start..];
        if let Some(gt) = rest.find('>') {
            let after = &rest[gt + 1..];
            let lower_after = after.to_lowercase();
            if let Some(end) = lower_after.find(close_tag) {
                return html_decode(after[..end].trim());
            }
        }
    }
    String::new()
}

/// Extract the `content` attribute from a `<meta name="..." content="...">` tag.
fn extract_meta_content(html: &str, name: &str) -> String {
    let lower = html.to_lowercase();
    let search = format!("name=\"{}\"", name);

    if let Some(pos) = lower.find(&search) {
        // Search in a window around the match for the content attribute.
        // The meta tag could have name before or after content.
        // Use floor_char_boundary to avoid slicing in the middle of a multi-byte char.
        let window_start = html.floor_char_boundary(pos.saturating_sub(200));
        let window_end = html.floor_char_boundary(html.len().min(pos + 500));
        let window = &html[window_start..window_end];
        let window_lower = window.to_lowercase();

        if let Some(content_pos) = window_lower.find("content=\"") {
            let after = &window[content_pos + 9..];
            if let Some(end) = after.find('"') {
                return html_decode(&after[..end]);
            }
        }
        // Also try content='...' (single quotes)
        if let Some(content_pos) = window_lower.find("content='") {
            let after = &window[content_pos + 9..];
            if let Some(end) = after.find('\'') {
                return html_decode(&after[..end]);
            }
        }
    }
    String::new()
}

/// Extract the `lang` attribute from the `<html>` tag.
fn extract_lang_attr(html: &str) -> String {
    let lower = html.to_lowercase();

    // Find lang="..." (double quotes)
    if let Some(pos) = lower.find("lang=\"") {
        let after = &html[pos + 6..];
        if let Some(end) = after.find('"') {
            return after[..end].to_string();
        }
    }
    // Try lang='...' (single quotes)
    if let Some(pos) = lower.find("lang='") {
        let after = &html[pos + 6..];
        if let Some(end) = after.find('\'') {
            return after[..end].to_string();
        }
    }
    String::new()
}

/// Basic HTML entity decoding for the most common entities.
fn html_decode(s: &str) -> String {
    s.replace("&amp;", "&")
        .replace("&lt;", "<")
        .replace("&gt;", ">")
        .replace("&quot;", "\"")
        .replace("&#39;", "'")
        .replace("&apos;", "'")
}

/// Truncate a string to at most `max_len` bytes, respecting char boundaries.
fn truncate_string(s: String, max_len: usize) -> String {
    if s.len() <= max_len {
        return s;
    }
    let mut end = max_len;
    while end > 0 && !s.is_char_boundary(end) {
        end -= 1;
    }
    s[..end].to_string()
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_extract_title() {
        let html = b"<html><head><title>Hello World</title></head></html>";
        let (title, _, _) = extract_metadata(html);
        assert_eq!(title, "Hello World");
    }

    #[test]
    fn test_extract_title_case_insensitive() {
        let html = b"<HTML><HEAD><TITLE>Case Test</TITLE></HEAD></HTML>";
        let (title, _, _) = extract_metadata(html);
        assert_eq!(title, "Case Test");
    }

    #[test]
    fn test_extract_description() {
        let html = b"<html><head><meta name=\"description\" content=\"A test page\"></head></html>";
        let (_, desc, _) = extract_metadata(html);
        assert_eq!(desc, "A test page");
    }

    #[test]
    fn test_extract_description_reversed_attrs() {
        let html =
            b"<html><head><meta content=\"Reversed\" name=\"description\"></head></html>";
        let (_, desc, _) = extract_metadata(html);
        assert_eq!(desc, "Reversed");
    }

    #[test]
    fn test_extract_language() {
        let html = b"<html lang=\"en-US\"><head></head></html>";
        let (_, _, lang) = extract_metadata(html);
        assert_eq!(lang, "en-US");
    }

    #[test]
    fn test_extract_empty() {
        let html = b"<html><head></head><body>no metadata</body></html>";
        let (title, desc, lang) = extract_metadata(html);
        assert_eq!(title, "");
        assert_eq!(desc, "");
        assert_eq!(lang, "");
    }

    #[test]
    fn test_html_decode() {
        assert_eq!(html_decode("AT&amp;T"), "AT&T");
        assert_eq!(html_decode("a &lt; b &gt; c"), "a < b > c");
        assert_eq!(html_decode("&quot;hello&quot;"), "\"hello\"");
    }

    #[test]
    fn test_truncate_string() {
        assert_eq!(truncate_string("hello".to_string(), 10), "hello");
        assert_eq!(truncate_string("hello world".to_string(), 5), "hello");
        // Multi-byte: euro sign is 3 bytes
        let s = "a\u{20AC}b".to_string(); // "a€b" = 5 bytes
        let truncated = truncate_string(s, 3);
        // Should truncate at char boundary: "a" (1 byte) + "€" (3 bytes) = 4 bytes > 3
        // So just "a"
        assert_eq!(truncated, "a");
    }
}
