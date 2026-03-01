use crate::config::Config;
use crate::domain::{group_by_domain, DomainState};
use crate::stats::{AdaptiveTimeout, Stats, StatsSnapshot};
use crate::types::{CrawlResult, FailedURL, SeedURL};
use crate::ua;
use crate::writer::{FailureWriter, ResultWriter};
use anyhow::Result;
use bytes::Bytes;
use dashmap::DashMap;
use http_body_util::{BodyExt, Empty};
use hyper_rustls::HttpsConnector;
use hyper_util::client::legacy::connect::HttpConnector;
use hyper_util::client::legacy::Client as HyperClient;
use hyper_util::rt::{TokioExecutor, TokioTimer};
use std::sync::atomic::{AtomicBool, AtomicU64, Ordering};
use std::sync::Arc;
use std::time::{Duration, Instant};
use tracing::{debug, info};

use super::reqwest_engine::extract_metadata;

/// Type alias for the hyper HTTPS client.
type HttpsClient = HyperClient<HttpsConnector<HttpConnector>, Empty<Bytes>>;

/// Maximum number of HTTP redirects to follow per request.
const MAX_REDIRECTS: usize = 7;

/// Per-domain state shared across all workers fetching from the same domain.
/// Mirrors DomainEntry in reqwest_engine.
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

/// Hyper+rustls crawl engine with flat URL task queue.
///
/// Identical architecture to ReqwestEngine but uses:
/// - hyper + hyper-rustls (rustls/ring backend) instead of reqwest/OpenSSL
/// - HTTP/2 multiplexing via ALPN negotiation
/// - Shared single client for all workers (connection pool reused globally)
///
/// On modern x86-64 (AES-NI, SHA extensions), ring is 10-20% faster than
/// OpenSSL for TLS handshakes and bulk crypto operations.
pub struct HyperEngine;

impl HyperEngine {
    pub fn new() -> Self {
        Self
    }
}

#[async_trait::async_trait]
impl super::Engine for HyperEngine {
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
            "hyper engine: {} seeds, {} workers, inner_n={}",
            total_seeds, cfg.workers, cfg.inner_n
        );

        let batches = group_by_domain(seeds);
        let domain_count = batches.len();
        info!("grouped into {} domains", domain_count);

        // Shared stats — use caller-provided Arc for live TUI display.
        let stats = cfg.live_stats.clone().unwrap_or_else(|| Arc::new(Stats::new()));
        if stats.total_seeds.load(Ordering::Relaxed) == 0 {
            stats.total_seeds.store(total_seeds as u64, Ordering::Relaxed);
        }
        stats.done.store(false, Ordering::Relaxed);

        let adaptive = Arc::new(AdaptiveTimeout::new());

        // Spawn live peak-RPS tracker (same as reqwest engine).
        let peak_stats = Arc::clone(&stats);
        tokio::spawn(async move {
            let mut prev_total = 0u64;
            let mut interval = tokio::time::interval(Duration::from_millis(100));
            interval.set_missed_tick_behavior(tokio::time::MissedTickBehavior::Skip);
            loop {
                interval.tick().await;
                if peak_stats.done.load(Ordering::Relaxed) {
                    break;
                }
                let cur = peak_stats.total.load(Ordering::Relaxed);
                let delta = cur.saturating_sub(prev_total);
                peak_stats.peak_rps.fetch_max(delta * 10, Ordering::Relaxed);
                prev_total = cur;
            }
        });

        let workers = cfg.workers.max(1);
        let inner_n = cfg.inner_n.max(1);

        // Pre-create per-domain entries.
        let domain_map: Arc<DashMap<String, Arc<DomainEntry>>> =
            Arc::new(DashMap::with_capacity(domain_count));
        for batch in &batches {
            domain_map.insert(
                batch.domain.clone(),
                Arc::new(DomainEntry::new(inner_n)),
            );
        }

        // Flat URL channel.
        let (url_tx, url_rx) =
            async_channel::bounded::<(SeedURL, Arc<DomainEntry>)>(workers * 4);

        // Round-robin producer.
        let dm = Arc::clone(&domain_map);
        let producer = tokio::spawn(async move {
            let domain_batches: Vec<(Vec<SeedURL>, Arc<DomainEntry>)> = batches
                .into_iter()
                .filter_map(|batch| {
                    dm.get(&batch.domain)
                        .map(|e| (batch.urls, Arc::clone(e.value())))
                })
                .collect();

            let max_len = domain_batches
                .iter()
                .map(|(urls, _)| urls.len())
                .max()
                .unwrap_or(0);

            for slot in 0..max_len {
                for (urls, entry) in &domain_batches {
                    if let Some(url) = urls.get(slot) {
                        if url_tx
                            .send((url.clone(), Arc::clone(entry)))
                            .await
                            .is_err()
                        {
                            return;
                        }
                    }
                }
            }
        });

        // Build ONE shared hyper client with rustls+ring and HTTP/2.
        let shared_client = match build_hyper_client(Duration::from_secs(60)) {
            Ok(c) => Arc::new(c),
            Err(e) => return Err(anyhow::anyhow!("failed to build hyper client: {}", e)),
        };

        // Spawn N worker tasks.
        let mut worker_handles = Vec::with_capacity(workers);
        for _ in 0..workers {
            let rx = url_rx.clone();
            let cfg = cfg.clone();
            let results = Arc::clone(&results);
            let failures = Arc::clone(&failures);
            let stats = Arc::clone(&stats);
            let adaptive = Arc::clone(&adaptive);
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
                    )
                    .await;
                }
            });
            worker_handles.push(handle);
        }

        let _ = producer.await;
        url_rx.close();

        for h in worker_handles {
            let _ = h.await;
        }

        stats.done.store(true, Ordering::Relaxed);

        let snapshot = stats.snapshot();
        info!(
            "hyper engine done: total={} ok={} failed={} timeout={} skipped={} peak_rps={} duration={:.1}s",
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

/// Build a shared hyper client with:
/// - rustls + ring backend (faster TLS than OpenSSL on AES-NI hardware)
/// - HTTP/2 via ALPN negotiation
/// - HTTP/1.1 fallback
/// - Connection pool with configurable idle timeout
fn build_hyper_client(pool_idle: Duration) -> Result<HttpsClient> {
    let https = hyper_rustls::HttpsConnectorBuilder::new()
        .with_webpki_roots()
        .https_or_http()
        .enable_http1()
        .enable_http2()
        .build();

    let client = HyperClient::builder(TokioExecutor::new())
        .pool_idle_timeout(pool_idle)
        .pool_timer(TokioTimer::new())
        .build(https);

    Ok(client)
}

/// Process a single URL using the shared hyper client.
async fn process_one_url(
    seed: SeedURL,
    domain_entry: &Arc<DomainEntry>,
    cfg: &Config,
    adaptive: &Arc<AdaptiveTimeout>,
    inner_n: usize,
    client: &Arc<HttpsClient>,
    results: &Arc<dyn ResultWriter>,
    failures: &Arc<dyn FailureWriter>,
    stats: &Arc<Stats>,
) {
    if domain_entry.abandoned.load(Ordering::Relaxed) {
        stats.skipped.fetch_add(1, Ordering::Relaxed);
        let _ = failures.write_url(FailedURL::new(
            &seed.url,
            &seed.domain,
            "domain_http_timeout_killed",
        ));
        return;
    }

    let _permit = match domain_entry.semaphore.acquire().await {
        Ok(p) => p,
        Err(_) => return,
    };

    let effective_timeout = if !cfg.disable_adaptive_timeout {
        adaptive
            .timeout(cfg.adaptive_timeout_max)
            .unwrap_or(cfg.timeout)
            .min(cfg.timeout.saturating_mul(5))
    } else {
        cfg.timeout
    };

    let result = hyper_fetch_one(client, &seed, effective_timeout, cfg.max_body_bytes).await;
    stats.total.fetch_add(1, Ordering::Relaxed);

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

/// Fetch a single URL using the shared hyper client with redirect following.
async fn hyper_fetch_one(
    client: &HttpsClient,
    seed: &SeedURL,
    timeout: Duration,
    max_body_bytes: usize,
) -> CrawlResult {
    let start = Instant::now();
    let mut current_url = seed.url.clone();

    for _redirect in 0..=MAX_REDIRECTS {
        let uri = match current_url.parse::<hyper::Uri>() {
            Ok(u) => u,
            Err(e) => {
                return CrawlResult::error_result(
                    &seed.url,
                    &seed.domain,
                    format!("invalid URI: {}", e),
                    start.elapsed().as_millis() as i64,
                );
            }
        };

        let req = match hyper::Request::builder()
            .method(hyper::Method::GET)
            .uri(&uri)
            .header("user-agent", ua::pick_user_agent())
            .body(Empty::<Bytes>::new())
        {
            Ok(r) => r,
            Err(e) => {
                return CrawlResult::error_result(
                    &seed.url,
                    &seed.domain,
                    format!("request build error: {}", e),
                    start.elapsed().as_millis() as i64,
                );
            }
        };

        let resp = match tokio::time::timeout(timeout, client.request(req)).await {
            Ok(Ok(r)) => r,
            Ok(Err(e)) => {
                return CrawlResult::error_result(
                    &seed.url,
                    &seed.domain,
                    e.to_string(),
                    start.elapsed().as_millis() as i64,
                );
            }
            Err(_) => {
                return CrawlResult::error_result(
                    &seed.url,
                    &seed.domain,
                    "timeout".to_string(),
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
        let content_length_header = resp
            .headers()
            .get("content-length")
            .and_then(|v| v.to_str().ok())
            .and_then(|v| v.parse::<i64>().ok())
            .unwrap_or(0);
        let location = resp
            .headers()
            .get("location")
            .and_then(|v| v.to_str().ok())
            .map(|s| s.to_string());

        // Follow redirects.
        if matches!(status, 301 | 302 | 307 | 308) {
            if let Some(ref loc) = location {
                // Drop the response body before following the redirect.
                drop(resp);
                current_url = resolve_redirect(&current_url, loc);
                continue;
            }
        }

        let is_html =
            content_type.contains("text/html") || content_type.contains("application/xhtml");

        // Only read body for 200 HTML responses where metadata can be extracted.
        let should_read_body = status == 200 && is_html;
        let body_bytes = if should_read_body {
            match tokio::time::timeout(timeout, collect_body(resp, max_body_bytes)).await {
                Ok(Ok(b)) => b,
                Ok(Err(e)) => {
                    return CrawlResult::error_result(
                        &seed.url,
                        &seed.domain,
                        format!("body read error: {}", e),
                        start.elapsed().as_millis() as i64,
                    );
                }
                Err(_) => {
                    return CrawlResult::error_result(
                        &seed.url,
                        &seed.domain,
                        "timeout reading body".to_string(),
                        start.elapsed().as_millis() as i64,
                    );
                }
            }
        } else {
            drop(resp);
            Bytes::new()
        };

        let body_len = body_bytes.len() as i64;
        let (title, description, language) = if should_read_body && !body_bytes.is_empty() {
            extract_metadata(&body_bytes)
        } else {
            (String::new(), String::new(), String::new())
        };

        return CrawlResult {
            url: seed.url.clone(),
            domain: seed.domain.clone(),
            status_code: status,
            content_type,
            content_length: content_length_header.max(body_len),
            title,
            description,
            language,
            redirect_url: location.unwrap_or_default(),
            fetch_time_ms: start.elapsed().as_millis() as i64,
            crawled_at: chrono::Utc::now().naive_utc(),
            error: String::new(),
            body: String::new(),
        };
    }

    CrawlResult::error_result(
        &seed.url,
        &seed.domain,
        format!("too many redirects (max {})", MAX_REDIRECTS),
        start.elapsed().as_millis() as i64,
    )
}

/// Collect the response body up to max_bytes.
async fn collect_body(
    resp: hyper::Response<hyper::body::Incoming>,
    max_bytes: usize,
) -> Result<Bytes, String> {
    use http_body_util::Limited;
    let limited = Limited::new(resp.into_body(), max_bytes);
    match limited.collect().await {
        Ok(collected) => Ok(collected.to_bytes()),
        Err(e) => {
            let err_str = e.to_string();
            if err_str.contains("length limit exceeded") {
                Ok(Bytes::new())
            } else {
                Err(err_str)
            }
        }
    }
}

/// Resolve a redirect location against the current URL.
fn resolve_redirect(base_url: &str, location: &str) -> String {
    if location.starts_with("http://") || location.starts_with("https://") {
        return location.to_string();
    }

    if let Ok(base_uri) = base_url.parse::<hyper::Uri>() {
        let scheme = base_uri.scheme_str().unwrap_or("https");
        let authority = base_uri.authority().map(|a| a.as_str()).unwrap_or("");

        if location.starts_with('/') {
            format!("{}://{}{}", scheme, authority, location)
        } else {
            let base_path = base_uri.path();
            let parent = if let Some(pos) = base_path.rfind('/') {
                &base_path[..=pos]
            } else {
                "/"
            };
            format!("{}://{}{}{}", scheme, authority, parent, location)
        }
    } else {
        location.to_string()
    }
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_resolve_redirect_absolute() {
        assert_eq!(
            resolve_redirect("https://example.com/page", "https://other.com/new"),
            "https://other.com/new"
        );
    }

    #[test]
    fn test_resolve_redirect_absolute_path() {
        assert_eq!(
            resolve_redirect("https://example.com/old/page", "/new/page"),
            "https://example.com/new/page"
        );
    }

    #[test]
    fn test_resolve_redirect_relative() {
        assert_eq!(
            resolve_redirect("https://example.com/dir/page", "other"),
            "https://example.com/dir/other"
        );
    }

    #[test]
    fn test_resolve_redirect_http() {
        assert_eq!(
            resolve_redirect("http://example.com/page", "/new"),
            "http://example.com/new"
        );
    }
}
