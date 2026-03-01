use crate::config::Config;
use crate::domain::{group_by_domain, DomainBatch, DomainState};
use crate::stats::{AdaptiveTimeout, PeakTracker, Stats, StatsSnapshot};
use crate::types::{CrawlResult, FailedURL, SeedURL};
use crate::ua;
use crate::writer::{FailureWriter, ResultWriter};
use anyhow::Result;
use std::sync::atomic::{AtomicBool, Ordering};
use std::sync::Arc;
use std::time::{Duration, Instant};
use tracing::{debug, info, warn};

/// Reqwest-based crawl engine with domain-grouped batch processing.
///
/// Architecture:
/// 1. Seeds are sorted and grouped by domain into DomainBatches.
/// 2. A producer feeds batches into a bounded channel (cap 4096).
/// 3. N worker tasks drain from the channel, each processing one domain at a time.
/// 4. Per-domain: inner_n fetch tasks share a reqwest::Client with connection pooling.
/// 5. Adaptive timeout, domain abandonment, and peak RPS tracking are all lock-free.
pub struct ReqwestEngine;

impl ReqwestEngine {
    pub fn new() -> Self {
        Self
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

        // Group seeds by domain
        let batches = group_by_domain(seeds);
        let domain_count = batches.len();
        info!("grouped into {} domains", domain_count);

        // Shared stats
        let stats = Arc::new(Stats::new());
        let adaptive = Arc::new(AdaptiveTimeout::new());
        let peak = Arc::new(PeakTracker::new());

        // Work channel: producer feeds domain batches, workers consume
        let (batch_tx, batch_rx) = async_channel::bounded::<DomainBatch>(4096);

        // Producer: feed all domain batches into the channel
        let producer = tokio::spawn(async move {
            for batch in batches {
                if batch_tx.send(batch).await.is_err() {
                    break; // receivers dropped
                }
            }
            // Channel closes when batch_tx is dropped
        });

        // Worker tasks
        let workers = cfg.workers.max(1);
        let inner_n = cfg.inner_n.max(1);
        let mut worker_handles = Vec::with_capacity(workers);

        for _ in 0..workers {
            let rx = batch_rx.clone();
            let cfg = cfg.clone();
            let results = Arc::clone(&results);
            let failures = Arc::clone(&failures);
            let stats = Arc::clone(&stats);
            let adaptive = Arc::clone(&adaptive);
            let peak = Arc::clone(&peak);

            let handle = tokio::spawn(async move {
                while let Ok(batch) = rx.recv().await {
                    process_one_domain(
                        batch.domain,
                        batch.urls,
                        &cfg,
                        &adaptive,
                        inner_n,
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

        // Wait for producer to finish sending
        let _ = producer.await;
        // Close the channel so workers see EOF
        batch_rx.close();

        // Wait for all workers to finish
        for h in worker_handles {
            let _ = h.await;
        }

        // Update peak RPS in stats
        stats
            .peak_rps
            .store(peak.peak(), Ordering::Relaxed);

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

/// Process all URLs for a single domain.
///
/// Creates a reqwest::Client with connection pooling (pool_max_idle_per_host = inner_n),
/// spawns inner_n fetch tasks sharing the client, and tracks domain health for abandonment.
async fn process_one_domain(
    domain: String,
    urls: Vec<SeedURL>,
    cfg: &Config,
    adaptive: &Arc<AdaptiveTimeout>,
    inner_n: usize,
    results: &Arc<dyn ResultWriter>,
    failures: &Arc<dyn FailureWriter>,
    stats: &Arc<Stats>,
    peak: &Arc<PeakTracker>,
) {
    let url_count = urls.len();
    if url_count == 0 {
        return;
    }

    // Calculate effective domain timeout
    let effective_domain_timeout = compute_domain_timeout(cfg, url_count, inner_n);

    // Build reqwest client for this domain
    let client = match reqwest::Client::builder()
        .pool_max_idle_per_host(inner_n)
        .timeout(cfg.timeout)
        .danger_accept_invalid_certs(true)
        .redirect(reqwest::redirect::Policy::limited(7))
        .build()
    {
        Ok(c) => Arc::new(c),
        Err(e) => {
            warn!("failed to build client for {}: {}", domain, e);
            // Mark all URLs as failed
            for seed in &urls {
                let _ = failures.write_url(FailedURL {
                    url: seed.url.clone(),
                    domain: seed.domain.clone(),
                    reason: "client_build_error".to_string(),
                    error: e.to_string(),
                    status_code: 0,
                    fetch_time_ms: 0,
                    detected_at: chrono::Utc::now().naive_utc(),
                });
                stats.failed.fetch_add(1, Ordering::Relaxed);
            }
            return;
        }
    };

    // URL channel: bounded by URL count so send never blocks
    let (url_tx, url_rx) = async_channel::bounded::<SeedURL>(url_count);
    for u in urls {
        let _ = url_tx.send(u).await;
    }
    url_tx.close();

    // Shared domain state for abandonment
    let abandoned = Arc::new(AtomicBool::new(false));
    let domain_successes = Arc::new(std::sync::atomic::AtomicU64::new(0));
    let domain_timeouts = Arc::new(std::sync::atomic::AtomicU64::new(0));

    // Spawn inner_n fetch tasks (capped by url count)
    let n = inner_n.min(url_count);
    let mut handles = Vec::with_capacity(n);

    for _ in 0..n {
        let rx = url_rx.clone();
        let client = Arc::clone(&client);
        let cfg_timeout = cfg.timeout;
        let cfg_adaptive_max = cfg.adaptive_timeout_max;
        let disable_adaptive = cfg.disable_adaptive_timeout;
        let max_body_bytes = cfg.max_body_bytes;
        let domain_fail_threshold = cfg.domain_fail_threshold;
        let domain_dead_probe = cfg.domain_dead_probe;
        let domain_stall_ratio = cfg.domain_stall_ratio;
        let inner_n_copy = inner_n;
        let adaptive = Arc::clone(adaptive);
        let results = Arc::clone(results);
        let failures = Arc::clone(failures);
        let stats = Arc::clone(stats);
        let peak = Arc::clone(peak);
        let abandoned = Arc::clone(&abandoned);
        let domain_successes = Arc::clone(&domain_successes);
        let domain_timeouts = Arc::clone(&domain_timeouts);
        let domain_name = domain.clone();

        let handle = tokio::spawn(async move {
            while let Ok(seed) = rx.recv().await {
                // Check if domain has been abandoned
                if abandoned.load(Ordering::Relaxed) {
                    stats.skipped.fetch_add(1, Ordering::Relaxed);
                    let _ = failures.write_url(FailedURL::new(
                        &seed.url,
                        &seed.domain,
                        "domain_http_timeout_killed",
                    ));
                    continue;
                }

                // Compute effective timeout (adaptive or fixed)
                let effective_timeout = if !disable_adaptive {
                    adaptive
                        .timeout(cfg_adaptive_max)
                        .unwrap_or(cfg_timeout)
                } else {
                    cfg_timeout
                };

                // Fetch the URL
                let result = fetch_one(
                    &client,
                    &seed,
                    effective_timeout,
                    max_body_bytes,
                )
                .await;

                stats.total.fetch_add(1, Ordering::Relaxed);
                peak.record();

                // Classify result
                if !result.error.is_empty() {
                    let is_timeout = result.error.contains("timeout")
                        || result.error.contains("Timeout")
                        || result.error.contains("deadline")
                        || result.error.contains("timed out");

                    if is_timeout {
                        stats.timeout.fetch_add(1, Ordering::Relaxed);
                        let t = domain_timeouts.fetch_add(1, Ordering::Relaxed) + 1;

                        // Check abandonment
                        let ds = DomainState {
                            successes: domain_successes.load(Ordering::Relaxed),
                            timeouts: t,
                        };
                        if ds.should_abandon(
                            domain_fail_threshold,
                            domain_dead_probe,
                            domain_stall_ratio,
                            inner_n_copy,
                        ) {
                            abandoned.store(true, Ordering::Relaxed);
                            debug!(
                                "abandoning domain {} (timeouts={}, successes={})",
                                domain_name, t, ds.successes
                            );
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
                    domain_successes.fetch_add(1, Ordering::Relaxed);
                    adaptive.record(result.fetch_time_ms);
                }

                stats
                    .bytes_downloaded
                    .fetch_add(result.content_length as u64, Ordering::Relaxed);

                let _ = results.write(result);
            }
        });
        handles.push(handle);
    }

    // Optionally wrap with domain timeout
    if let Some(dt) = effective_domain_timeout {
        let domain_name = domain.clone();
        let abandoned_outer = Arc::clone(&abandoned);
        let stats_outer = Arc::clone(stats);
        let failures_outer = Arc::clone(failures);

        let wait_fut = async {
            for h in handles {
                let _ = h.await;
            }
        };

        match tokio::time::timeout(dt, wait_fut).await {
            Ok(()) => {
                // All tasks completed within domain timeout
            }
            Err(_) => {
                // Domain timeout exceeded — abandon remaining URLs
                abandoned_outer.store(true, Ordering::Relaxed);
                warn!(
                    "domain {} exceeded timeout ({:.1}s), abandoning remaining URLs",
                    domain_name,
                    dt.as_secs_f64()
                );

                // Drain remaining URLs and mark them as deadline exceeded
                while let Ok(seed) = url_rx.try_recv() {
                    stats_outer.skipped.fetch_add(1, Ordering::Relaxed);
                    let _ = failures_outer.write_url(FailedURL::new(
                        &seed.url,
                        &seed.domain,
                        "domain_deadline_exceeded",
                    ));
                }
            }
        }
    } else {
        // No domain timeout — just wait for all tasks
        for h in handles {
            let _ = h.await;
        }
    }
}

/// Calculate effective domain timeout.
///
/// - domain_timeout_ms < 0 (adaptive): len(urls) * timeout / inner_n * 2, clamped [30s, max]
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
    // Formula: urls * timeout_ms / inner_n * 2, clamped [30s, adaptive_timeout_max]
    let timeout_ms = cfg.timeout.as_millis() as u64;
    let estimated_ms = url_count as u64 * timeout_ms / inner_n.max(1) as u64 * 2;
    let min_ms = 30_000u64;
    let max_ms = cfg.adaptive_timeout_max.as_millis() as u64;
    let clamped_ms = estimated_ms.max(min_ms).min(max_ms);

    Some(Duration::from_millis(clamped_ms))
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

/// Read response body with a size limit to avoid OOM on large responses.
async fn read_body_limited(
    resp: reqwest::Response,
    max_bytes: usize,
) -> Result<bytes::Bytes, reqwest::Error> {
    // reqwest does not have a built-in body size limit, so we stream chunks
    // For simplicity and performance, use bytes() which reads the full body.
    // The max_body_bytes is enforced by truncation after read.
    let full = resp.bytes().await?;
    if full.len() > max_bytes {
        Ok(full.slice(..max_bytes))
    } else {
        Ok(full)
    }
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

    #[test]
    fn test_compute_domain_timeout_disabled() {
        let mut cfg = Config::default();
        cfg.domain_timeout_ms = 0;
        assert!(compute_domain_timeout(&cfg, 100, 4).is_none());
    }

    #[test]
    fn test_compute_domain_timeout_explicit() {
        let mut cfg = Config::default();
        cfg.domain_timeout_ms = 5000;
        let dt = compute_domain_timeout(&cfg, 100, 4);
        assert_eq!(dt, Some(Duration::from_millis(5000)));
    }

    #[test]
    fn test_compute_domain_timeout_adaptive() {
        let mut cfg = Config::default();
        cfg.domain_timeout_ms = -1;
        cfg.timeout = Duration::from_millis(1000);
        cfg.adaptive_timeout_max = Duration::from_secs(600);
        // 100 urls * 1000ms / 4 inner * 2 = 50000ms
        let dt = compute_domain_timeout(&cfg, 100, 4);
        assert_eq!(dt, Some(Duration::from_millis(50000)));
    }

    #[test]
    fn test_compute_domain_timeout_adaptive_clamped_min() {
        let mut cfg = Config::default();
        cfg.domain_timeout_ms = -1;
        cfg.timeout = Duration::from_millis(1000);
        cfg.adaptive_timeout_max = Duration::from_secs(600);
        // 2 urls * 1000ms / 4 inner * 2 = 1000ms -> clamped to 30000ms min
        let dt = compute_domain_timeout(&cfg, 2, 4);
        assert_eq!(dt, Some(Duration::from_millis(30000)));
    }

    #[test]
    fn test_compute_domain_timeout_adaptive_clamped_max() {
        let mut cfg = Config::default();
        cfg.domain_timeout_ms = -1;
        cfg.timeout = Duration::from_millis(10000);
        cfg.adaptive_timeout_max = Duration::from_secs(120);
        // 1000 urls * 10000ms / 4 inner * 2 = 5_000_000ms -> clamped to 120_000ms max
        let dt = compute_domain_timeout(&cfg, 1000, 4);
        assert_eq!(dt, Some(Duration::from_millis(120_000)));
    }
}
