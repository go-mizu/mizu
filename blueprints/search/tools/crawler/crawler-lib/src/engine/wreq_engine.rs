//! Wreq-based crawl engine with Chrome TLS fingerprint impersonation.
//!
//! Architecture identical to reqwest_engine — flat URL queue, per-domain semaphores,
//! adaptive timeout. The key difference: wreq::Client with Chrome133 impersonation
//! produces authentic TLS (JA3/JA4) and HTTP/2 fingerprints, bypassing bot-detection
//! that keys on TLS fingerprints rather than just User-Agent headers.

#![cfg(feature = "wreq-engine")]

use crate::config::Config;
use crate::domain::{group_by_domain, DomainState};
use crate::stats::{AdaptiveTimeout, ErrorCategory, Stats, StatsSnapshot};
use crate::types::{CrawlResult, FailedURL, SeedURL};
use crate::writer::{FailureWriter, ResultWriter};
use anyhow::Result;
use dashmap::DashMap;
use std::sync::atomic::{AtomicBool, AtomicU64, Ordering};
use std::sync::Arc;
use std::time::{Duration, Instant};
use tracing::{debug, info};
use wreq_util::Emulation;

pub struct WreqEngine;

impl WreqEngine {
    pub fn new() -> Self {
        Self
    }
}

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
impl super::Engine for WreqEngine {
    async fn run(
        &self,
        seed_rx: async_channel::Receiver<SeedURL>,
        seed_count: u64,
        cfg: &Config,
        results: Arc<dyn ResultWriter>,
        failures: Arc<dyn FailureWriter>,
    ) -> Result<StatsSnapshot> {
        info!(
            "wreq engine (Chrome133 TLS): workers={} inner_n={} timeout={}ms seed_hint={}",
            cfg.workers, cfg.inner_n, cfg.timeout.as_millis(), seed_count,
        );

        let stats = cfg.live_stats.clone().unwrap_or_else(|| Arc::new(Stats::new()));
        if seed_count > 0 && stats.total_seeds.load(Ordering::Relaxed) == 0 {
            stats.total_seeds.store(seed_count, Ordering::Relaxed);
        }
        stats.done.store(false, Ordering::Relaxed);

        let adaptive = Arc::new(AdaptiveTimeout::new());

        // Peak-RPS tracker
        let peak_stats = Arc::clone(&stats);
        tokio::spawn(async move {
            let mut prev_total = peak_stats.total.load(Ordering::Relaxed);
            let mut interval = tokio::time::interval(Duration::from_millis(100));
            interval.set_missed_tick_behavior(tokio::time::MissedTickBehavior::Skip);
            loop {
                interval.tick().await;
                if peak_stats.done.load(Ordering::Relaxed) {
                    break;
                }
                let cur = peak_stats.total.load(Ordering::Relaxed);
                let delta = cur.saturating_sub(prev_total);
                let rps = delta * 10;
                peak_stats.peak_rps.fetch_max(rps, Ordering::Relaxed);
                prev_total = cur;
            }
        });

        // System monitor
        let sysmon_stats = Arc::clone(&stats);
        tokio::spawn(async move {
            super::reqwest_engine::spawn_sysmon(sysmon_stats).await;
        });

        let workers = cfg.workers.max(1);
        let inner_n = cfg.inner_n.max(1);

        // Domain map: lazily populated as new domains arrive from the streaming channel.
        let domain_map: Arc<DashMap<String, Arc<DomainEntry>>> =
            Arc::new(DashMap::with_capacity(32_768));

        let (url_tx, url_rx) =
            async_channel::bounded::<(SeedURL, Arc<DomainEntry>)>(workers * 4);

        // Batch-streaming producer (mirrors reqwest_engine / hyper_engine).
        const BATCH_SIZE: usize = 100_000;
        let dm = Arc::clone(&domain_map);
        let stats_prod = Arc::clone(&stats);
        let producer = tokio::spawn(async move {
            use std::collections::VecDeque;
            let mut batch: Vec<SeedURL> = Vec::with_capacity(BATCH_SIZE);
            let mut total_seen = 0u64;

            loop {
                batch.clear();
                match seed_rx.recv().await {
                    Ok(seed) => batch.push(seed),
                    Err(_) => break,
                }
                while batch.len() < BATCH_SIZE {
                    match seed_rx.try_recv() {
                        Ok(seed) => batch.push(seed),
                        Err(_) => break,
                    }
                }

                total_seen += batch.len() as u64;
                if stats_prod.total_seeds.load(Ordering::Relaxed) == 0 {
                    stats_prod.total_seeds.store(total_seen, Ordering::Relaxed);
                }

                let sub_batches = group_by_domain(std::mem::take(&mut batch));
                stats_prod
                    .domains_total
                    .fetch_add(sub_batches.len() as u64, Ordering::Relaxed);

                let mut queue: VecDeque<(VecDeque<SeedURL>, Arc<DomainEntry>)> = sub_batches
                    .into_iter()
                    .map(|b| {
                        let entry = dm
                            .entry(b.domain.clone())
                            .or_insert_with(|| Arc::new(DomainEntry::new(inner_n)));
                        (VecDeque::from(b.urls), Arc::clone(entry.value()))
                    })
                    .collect();

                while let Some((mut urls, entry)) = queue.pop_front() {
                    let url = match urls.pop_front() {
                        Some(u) => u,
                        None => continue,
                    };
                    if url_tx.send((url, Arc::clone(&entry))).await.is_err() {
                        return;
                    }
                    if !urls.is_empty() {
                        queue.push_back((urls, entry));
                    }
                }
            }

            info!(
                "wreq producer done: {} seeds dispatched across {} domains",
                total_seen,
                dm.len(),
            );
        });

        // Build wreq client with Chrome133 TLS impersonation
        let max_timeout = cfg.timeout.saturating_mul(5);
        let shared_client = match wreq::Client::builder()
            .emulation(Emulation::Chrome133)
            .pool_max_idle_per_host(inner_n)
            .timeout(max_timeout)
            .cert_verification(false)
            .redirect(wreq::redirect::Policy::limited(7))
            .tcp_keepalive(Duration::from_secs(60))
            .build()
        {
            Ok(c) => Arc::new(c),
            Err(e) => return Err(anyhow::anyhow!("failed to build wreq client: {}", e)),
        };

        // Spawn workers
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
                        seed, &domain_entry, &cfg, &adaptive, inner_n,
                        &client, &results, &failures, &stats,
                    ).await;
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
            "wreq engine done: total={} ok={} failed={} timeout={} skipped={} peak_rps={} duration={:.1}s",
            snapshot.total, snapshot.ok, snapshot.failed, snapshot.timeout,
            snapshot.skipped, snapshot.peak_rps, snapshot.duration.as_secs_f64()
        );
        stats.push_warning(format!(
            "done: {} ok, {} failed (inv={} dns={} conn={} tls={}), {} timeout, {:.0} avg rps",
            snapshot.ok, snapshot.failed, snapshot.err_invalid_url, snapshot.err_dns,
            snapshot.err_conn, snapshot.err_tls, snapshot.timeout, snapshot.avg_rps(),
        ));
        Ok(snapshot)
    }
}

// ---------------------------------------------------------------------------
// URL processing (mirrors reqwest_engine logic)
// ---------------------------------------------------------------------------

async fn process_one_url(
    seed: SeedURL,
    domain_entry: &Arc<DomainEntry>,
    cfg: &Config,
    adaptive: &Arc<AdaptiveTimeout>,
    inner_n: usize,
    client: &Arc<wreq::Client>,
    results: &Arc<dyn ResultWriter>,
    failures: &Arc<dyn FailureWriter>,
    stats: &Arc<Stats>,
) {
    if domain_entry.abandoned.load(Ordering::Relaxed) {
        stats.skipped.fetch_add(1, Ordering::Relaxed);
        let _ = failures.write_url(FailedURL::new(&seed.url, &seed.domain, "domain_http_timeout_killed"));
        return;
    }

    let _permit = match domain_entry.semaphore.acquire().await {
        Ok(p) => p,
        Err(_) => return,
    };

    // Floor: never drop below cfg.timeout; ceiling: ×3 (tight vs old ×5).
    let effective_timeout = if !cfg.disable_adaptive_timeout {
        adaptive
            .timeout(cfg.adaptive_timeout_max)
            .unwrap_or(cfg.timeout)
            .max(cfg.timeout)
            .min(cfg.timeout.saturating_mul(3))
    } else {
        cfg.timeout
    };

    let fetch_result = fetch_one(client, &seed, effective_timeout, cfg.max_body_bytes).await;
    stats.total.fetch_add(1, Ordering::Relaxed);

    // Release semaphore before any writer call (avoids fetch goroutines blocking behind
    // binary writer backpressure while still holding the inner_n permit).
    drop(_permit);

    match fetch_result {
        Err((wreq_err, fetch_ms)) => {
            let category = classify_wreq_error(&wreq_err);
            let error_str = error_chain_string(&wreq_err);

            if category == ErrorCategory::Timeout {
                stats.timeout.fetch_add(1, Ordering::Relaxed);
                let timeout_threshold_ms = (cfg.timeout.as_millis() as i64) * 90 / 100;
                let subcat = if fetch_ms < timeout_threshold_ms {
                    stats.timeout_connect.fetch_add(1, Ordering::Relaxed);
                    "connect"
                } else {
                    stats.timeout_response.fetch_add(1, Ordering::Relaxed);
                    "response"
                };
                let t = domain_entry.timeouts.fetch_add(1, Ordering::Relaxed) + 1;
                let s = domain_entry.ok.load(Ordering::Relaxed);
                let ds = DomainState { successes: s, timeouts: t };
                if ds.should_abandon(cfg.domain_fail_threshold, cfg.domain_dead_probe, cfg.domain_stall_ratio, inner_n) {
                    if !domain_entry.abandoned.swap(true, Ordering::Relaxed) {
                        debug!("abandoning domain {} (timeouts={}, ok={})", seed.domain, t, s);
                        stats.domains_abandoned.fetch_add(1, Ordering::Relaxed);
                        stats.push_warning(format!("abandoned {} (timeouts={}, ok={})", seed.domain, t, s));
                    }
                }
                let _ = failures.write_url(FailedURL {
                    url: seed.url.clone(),
                    domain: seed.domain.clone(),
                    reason: "http_timeout".to_string(),
                    subcategory: subcat.to_string(),
                    error: error_str,
                    status_code: 0,
                    fetch_time_ms: fetch_ms,
                    detected_at: chrono::Utc::now().naive_utc(),
                });
            } else {
                stats.failed.fetch_add(1, Ordering::Relaxed);
                let lower = error_str.to_lowercase();

                let subcat: &str = match category {
                    ErrorCategory::InvalidUrl => {
                        let n = stats.err_invalid_url.fetch_add(1, Ordering::Relaxed) + 1;
                        if n <= 5 || n % 200 == 0 {
                            stats.push_warning(format!("invalid: {} ({})", seed.domain, short_error(&error_str)));
                        }
                        "invalid"
                    }
                    ErrorCategory::Dns => {
                        let n = stats.err_dns.fetch_add(1, Ordering::Relaxed) + 1;
                        let sub = if lower.contains("no records found") || lower.contains("nxdomain")
                            || lower.contains("name or service not known") || lower.contains("no address associated") {
                            stats.dns_nxdomain.fetch_add(1, Ordering::Relaxed); "nxdomain"
                        } else if lower.contains("malformed") || lower.contains("invalid character")
                            || lower.contains("label bytes exceed") {
                            stats.dns_malformed.fetch_add(1, Ordering::Relaxed); "malformed"
                        } else {
                            stats.dns_other.fetch_add(1, Ordering::Relaxed); "other"
                        };
                        if n <= 5 || (n <= 100 && n % 20 == 0) || n % 500 == 0 {
                            stats.push_warning(format!("dns: {} ({})", seed.domain, short_error(&error_str)));
                        }
                        sub
                    }
                    ErrorCategory::Connection => {
                        let n = stats.err_conn.fetch_add(1, Ordering::Relaxed) + 1;
                        let sub = if lower.contains("connection refused") {
                            stats.conn_refused.fetch_add(1, Ordering::Relaxed); "refused"
                        } else if lower.contains("reset by peer") || lower.contains("connection reset") {
                            stats.conn_reset.fetch_add(1, Ordering::Relaxed); "reset"
                        } else if lower.contains("unexpected eof") || lower.contains("connection closed")
                            || lower.contains("broken pipe") {
                            stats.conn_eof.fetch_add(1, Ordering::Relaxed); "eof"
                        } else {
                            stats.conn_other.fetch_add(1, Ordering::Relaxed); "other"
                        };
                        if n <= 3 || n % 500 == 0 {
                            stats.push_warning(format!("conn: {} ({})", seed.domain, short_error(&error_str)));
                        }
                        sub
                    }
                    ErrorCategory::Tls => {
                        let n = stats.err_tls.fetch_add(1, Ordering::Relaxed) + 1;
                        if n <= 3 || n % 200 == 0 {
                            stats.push_warning(format!("tls: {} ({})", seed.domain, short_error(&error_str)));
                        }
                        "tls"
                    }
                    _ => {
                        let n = stats.err_other.fetch_add(1, Ordering::Relaxed) + 1;
                        if n <= 10 || n % 200 == 0 {
                            stats.push_warning(format!("other: {} ({})", seed.domain, short_error(&error_str)));
                        }
                        "other"
                    }
                };

                let _ = failures.write_url(FailedURL {
                    url: seed.url.clone(),
                    domain: seed.domain.clone(),
                    reason: match category {
                        ErrorCategory::InvalidUrl => "invalid_url",
                        ErrorCategory::Dns => "dns_error",
                        ErrorCategory::Connection => "conn_error",
                        ErrorCategory::Tls => "tls_error",
                        _ => "http_error",
                    }.to_string(),
                    subcategory: subcat.to_string(),
                    error: error_str,
                    status_code: 0,
                    fetch_time_ms: fetch_ms,
                    detected_at: chrono::Utc::now().naive_utc(),
                });
            }
        }
        Ok(result) => {
            stats.ok.fetch_add(1, Ordering::Relaxed);
            domain_entry.ok.fetch_add(1, Ordering::Relaxed);
            adaptive.record(result.fetch_time_ms);
            match result.status_code {
                200..=299 => { stats.status_2xx.fetch_add(1, Ordering::Relaxed); }
                300..=399 => { stats.status_3xx.fetch_add(1, Ordering::Relaxed); }
                400..=499 => { stats.status_4xx.fetch_add(1, Ordering::Relaxed); }
                500..=599 => { stats.status_5xx.fetch_add(1, Ordering::Relaxed); }
                _ => {}
            }
            stats.bytes_downloaded.fetch_add(result.content_length as u64, Ordering::Relaxed);
            let _ = results.write(result);
        }
    }
}

// ---------------------------------------------------------------------------
// wreq error classification (mirrors reqwest logic — wreq has same typed methods)
// ---------------------------------------------------------------------------

fn classify_wreq_error(e: &wreq::Error) -> ErrorCategory {
    if e.is_timeout() {
        ErrorCategory::Timeout
    } else if e.is_builder() {
        ErrorCategory::InvalidUrl
    } else if e.is_connect() {
        let chain = error_chain_string(e);
        let lower = chain.to_lowercase();
        if lower.contains("dns") || lower.contains("resolve") || lower.contains("no record")
            || lower.contains("nxdomain") || lower.contains("name or service not known") {
            ErrorCategory::Dns
        } else if lower.contains("tls") || lower.contains("ssl") || lower.contains("certificate")
            || lower.contains("handshake alert") {
            ErrorCategory::Tls
        } else {
            ErrorCategory::Connection
        }
    } else if e.is_request() {
        let chain = error_chain_string(e);
        let lower = chain.to_lowercase();
        if lower.contains("dns") || lower.contains("resolve") {
            ErrorCategory::Dns
        } else if lower.contains("tls") || lower.contains("ssl") {
            ErrorCategory::Tls
        } else if lower.contains("timed out") || lower.contains("deadline has elapsed") {
            ErrorCategory::Timeout
        } else {
            ErrorCategory::Connection
        }
    } else if e.is_decode() || e.is_body() {
        // Server responded but body decompression or read failed.
        ErrorCategory::Connection
    } else {
        ErrorCategory::Other
    }
}

fn error_chain_string(e: &(dyn std::error::Error + 'static)) -> String {
    let mut parts = vec![e.to_string()];
    let mut current: &dyn std::error::Error = e;
    while let Some(source) = current.source() {
        parts.push(source.to_string());
        current = source;
    }
    parts.join(": ")
}

fn short_error(s: &str) -> String {
    if s.len() <= 80 { s.to_string() } else { format!("{}...", &s[..77]) }
}

// ---------------------------------------------------------------------------
// Fetch
// ---------------------------------------------------------------------------

async fn fetch_one(
    client: &wreq::Client,
    seed: &SeedURL,
    timeout: Duration,
    max_body_bytes: usize,
) -> Result<CrawlResult, (wreq::Error, i64)> {
    let start = Instant::now();
    let url = super::reqwest_engine::sanitize_url(&seed.url);

    // wreq with Chrome133 impersonation already sends correct TLS fingerprint + HTTP/2 settings.
    // We still set browser headers for header-level fingerprint consistency.
    let profile = crate::ua::pick_profile(&seed.domain);
    let mut req = client
        .get(&url)
        .header("User-Agent", profile.user_agent)
        .header("Accept", profile.accept)
        .header("Accept-Language", profile.accept_language)
        .header("Accept-Encoding", profile.accept_encoding)
        .header("Upgrade-Insecure-Requests", "1")
        .timeout(timeout);

    if let Some(v) = profile.sec_ch_ua { req = req.header("Sec-CH-UA", v); }
    if let Some(v) = profile.sec_ch_ua_mobile { req = req.header("Sec-CH-UA-Mobile", v); }
    if let Some(v) = profile.sec_ch_ua_platform { req = req.header("Sec-CH-UA-Platform", v); }
    if let Some(v) = profile.sec_fetch_dest { req = req.header("Sec-Fetch-Dest", v); }
    if let Some(v) = profile.sec_fetch_mode { req = req.header("Sec-Fetch-Mode", v); }
    if let Some(v) = profile.sec_fetch_site { req = req.header("Sec-Fetch-Site", v); }
    if let Some(v) = profile.sec_fetch_user { req = req.header("Sec-Fetch-User", v); }

    let response = req.send().await;
    let resp = match response {
        Ok(r) => r,
        Err(e) => {
            return Err((e, start.elapsed().as_millis() as i64));
        }
    };

    let status = resp.status().as_u16();
    let content_type = resp.headers().get("content-type")
        .and_then(|v| v.to_str().ok()).unwrap_or("").to_string();
    let content_length = resp.content_length().unwrap_or(0) as i64;
    let redirect_url = resp.headers().get("location")
        .and_then(|v| v.to_str().ok()).unwrap_or("").to_string();

    let is_html = content_type.contains("text/html") || content_type.contains("application/xhtml");
    let should_read_body = status == 200 && is_html;

    let body_bytes = if should_read_body {
        match read_body_limited(resp, max_body_bytes).await {
            Ok(b) => b,
            Err(e) => {
                return Err((e, start.elapsed().as_millis() as i64));
            }
        }
    } else {
        drop(resp);
        bytes::Bytes::new()
    };

    let body_len = body_bytes.len() as i64;
    let (title, description, language) = if should_read_body && !body_bytes.is_empty() {
        super::reqwest_engine::extract_metadata(&body_bytes)
    } else {
        (String::new(), String::new(), String::new())
    };

    Ok(CrawlResult {
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
        body: String::new(),
        body_cid: String::new(), // wreq engine: body store not wired (reqwest engine only)
    })
}

async fn read_body_limited(
    resp: wreq::Response,
    max_bytes: usize,
) -> Result<bytes::Bytes, wreq::Error> {
    use bytes::BytesMut;
    use futures_util::StreamExt;

    if let Some(len) = resp.content_length() {
        if len as usize <= max_bytes {
            return resp.bytes().await;
        }
    }

    let mut stream = resp.bytes_stream();
    let mut buf = BytesMut::with_capacity(max_bytes.min(64 * 1024));
    while let Some(chunk) = stream.next().await {
        let chunk: bytes::Bytes = chunk?;
        let remaining = max_bytes.saturating_sub(buf.len());
        if remaining == 0 { break; }
        let take = chunk.len().min(remaining);
        buf.extend_from_slice(&chunk[..take]);
    }
    Ok(buf.freeze())
}
