use crate::config::Config;
use crate::domain::{group_by_domain, DomainState};
use crate::stats::{AdaptiveTimeout, ErrorCategory, Stats, StatsSnapshot};
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
            return Ok(StatsSnapshot::empty());
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
        // Reset done flag (may be set from a previous pass)
        stats.done.store(false, Ordering::Relaxed);
        stats.domains_total.store(domain_count as u64, Ordering::Relaxed);

        // Push a start event to the TUI log.
        stats.push_warning(format!(
            "engine: {} seeds, {} domains, {} workers, inner_n={}",
            total_seeds, domain_count, cfg.workers, cfg.inner_n,
        ));

        let adaptive = Arc::new(AdaptiveTimeout::new());

        // Spawn a peak-RPS tracker task: samples total every 100ms, writes live peak_rps.
        // This replaces PeakTracker (which used try_lock and almost never fired at 16K workers).
        let peak_stats = Arc::clone(&stats);
        tokio::spawn(async move {
            // Initialize from current total so we don't count prior-pass history on restart.
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
                // delta/100ms → per-second rate
                let rps = delta * 10;
                peak_stats.peak_rps.fetch_max(rps, Ordering::Relaxed);
                prev_total = cur;
            }
        });

        // Spawn a system-monitor task: samples RSS, FDs, network every 500ms.
        let sysmon_stats = Arc::clone(&stats);
        tokio::spawn(async move {
            spawn_sysmon(sysmon_stats).await;
        });

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

        // Producer: send URLs in round-robin domain order to prevent semaphore clustering.
        //
        // Sorted domain order [A1,A2,A3..An, B1,B2..Bn, ...] causes workers to cluster on
        // the same domain's semaphore simultaneously. Round-robin order [A1,B1,C1,...,A2,B2,...]
        // ensures each "wave" of workers hits different domains, eliminating semaphore contention
        // and spreading DNS resolution across many nameservers simultaneously.
        let dm = Arc::clone(&domain_map);
        let producer = tokio::spawn(async move {
            // Pair each batch's URL list with its pre-created domain entry.
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

            // Send slot[i] from every domain before sending slot[i+1].
            // Result: [A_slot0, B_slot0, C_slot0, ..., A_slot1, B_slot1, ...]
            for slot in 0..max_len {
                for (urls, entry) in &domain_batches {
                    if let Some(url) = urls.get(slot) {
                        if url_tx
                            .send((url.clone(), Arc::clone(entry)))
                            .await
                            .is_err()
                        {
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

        // Wait for producer to finish sending all URLs.
        let _ = producer.await;
        // Close channel so workers see EOF when all URLs are consumed.
        url_rx.close();

        // Wait for all workers to finish.
        for h in worker_handles {
            let _ = h.await;
        }

        // Signal the peak tracker task to stop.
        stats.done.store(true, Ordering::Relaxed);

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

        // Push final summary event.
        stats.push_warning(format!(
            "done: {} ok, {} failed (inv={} dns={} conn={} tls={}), {} timeout, {:.0} avg rps",
            snapshot.ok,
            snapshot.failed,
            snapshot.err_invalid_url,
            snapshot.err_dns,
            snapshot.err_conn,
            snapshot.err_tls,
            snapshot.timeout,
            snapshot.avg_rps(),
        ));

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
    let fetch_result = fetch_one(
        client,
        &seed,
        effective_timeout,
        cfg.max_body_bytes,
        cfg.body_store.as_deref(),
    )
    .await;
    stats.total.fetch_add(1, Ordering::Relaxed);

    match fetch_result {
        Err((reqwest_err, fetch_ms)) => {
            // Classify using reqwest's typed error methods + full error chain.
            let category = classify_reqwest_error(&reqwest_err);
            let error_str = error_chain_string(&reqwest_err);

            if category == ErrorCategory::Timeout {
                stats.timeout.fetch_add(1, Ordering::Relaxed);
                // Sub-classify: connect timeout (< cfg.timeout) vs response timeout (full timeout)
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
                        stats.domains_abandoned.fetch_add(1, Ordering::Relaxed);
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
                    subcategory: subcat.to_string(),
                    error: error_str,
                    status_code: 0,
                    fetch_time_ms: fetch_ms,
                    detected_at: chrono::Utc::now().naive_utc(),
                });
            } else {
                stats.failed.fetch_add(1, Ordering::Relaxed);
                let lower = error_str.to_lowercase();

                // Track error sub-category and compute subcategory string.
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
                            || lower.contains("name or service not known")
                            || lower.contains("no address associated")
                        {
                            stats.dns_nxdomain.fetch_add(1, Ordering::Relaxed);
                            "nxdomain"
                        } else if lower.contains("malformed") || lower.contains("invalid character")
                            || lower.contains("label bytes exceed")
                        {
                            stats.dns_malformed.fetch_add(1, Ordering::Relaxed);
                            "malformed"
                        } else {
                            stats.dns_other.fetch_add(1, Ordering::Relaxed);
                            "other"
                        };
                        if n <= 5 || (n <= 100 && n % 20 == 0) || n % 500 == 0 {
                            stats.push_warning(format!("dns: {} ({})", seed.domain, short_error(&error_str)));
                        }
                        sub
                    }
                    ErrorCategory::Connection => {
                        let n = stats.err_conn.fetch_add(1, Ordering::Relaxed) + 1;
                        let sub = if lower.contains("connection refused") {
                            stats.conn_refused.fetch_add(1, Ordering::Relaxed);
                            "refused"
                        } else if lower.contains("reset by peer") || lower.contains("connection reset") {
                            stats.conn_reset.fetch_add(1, Ordering::Relaxed);
                            "reset"
                        } else if lower.contains("unexpected eof") || lower.contains("connection closed")
                            || lower.contains("broken pipe")
                        {
                            stats.conn_eof.fetch_add(1, Ordering::Relaxed);
                            "eof"
                        } else {
                            stats.conn_other.fetch_add(1, Ordering::Relaxed);
                            "other"
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
                        if n <= 50 {
                            tracing::warn!(n, domain = %seed.domain, fetch_ms, chain = %error_str,
                                is_timeout = reqwest_err.is_timeout(), is_connect = reqwest_err.is_connect(),
                                is_builder = reqwest_err.is_builder(), is_request = reqwest_err.is_request(),
                                is_body = reqwest_err.is_body(), is_decode = reqwest_err.is_decode(),
                                "other_sample");
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

            // Track HTTP status code distribution.
            match result.status_code {
                200..=299 => { stats.status_2xx.fetch_add(1, Ordering::Relaxed); }
                300..=399 => { stats.status_3xx.fetch_add(1, Ordering::Relaxed); }
                400..=499 => { stats.status_4xx.fetch_add(1, Ordering::Relaxed); }
                500..=599 => { stats.status_5xx.fetch_add(1, Ordering::Relaxed); }
                _ => {}
            }

            stats
                .bytes_downloaded
                .fetch_add(result.content_length as u64, Ordering::Relaxed);
            let _ = results.write(result);
        }
    }
}

/// Sanitize a URL for reqwest:
/// - Trim whitespace
/// - Fix "http:// domain" → "http://domain" (space after scheme)
/// - Fix double schemes "http:// http://..." → "http://..."
pub(crate) fn sanitize_url(url: &str) -> String {
    let url = url.trim();

    // Fix "http:// http://..." or "http:// https://..." (double scheme with space)
    if let Some(rest) = url.strip_prefix("http:// http://") {
        return format!("http://{}", rest.trim_start());
    }
    if let Some(rest) = url.strip_prefix("http:// https://") {
        return format!("https://{}", rest.trim_start());
    }

    // Fix "http:// domain" → "http://domain"
    if let Some(rest) = url.strip_prefix("http:// ") {
        return format!("http://{}", rest.trim_start());
    }
    if let Some(rest) = url.strip_prefix("https:// ") {
        return format!("https://{}", rest.trim_start());
    }

    url.to_string()
}

/// Background system monitor: updates RSS, FDs, network stats every 500ms.
pub(crate) async fn spawn_sysmon(stats: Arc<Stats>) {
    let pid = sysinfo::Pid::from_u32(std::process::id());
    let mut sys = sysinfo::System::new();
    let mut nets = sysinfo::Networks::new_with_refreshed_list();
    let mut interval = tokio::time::interval(Duration::from_millis(500));
    interval.set_missed_tick_behavior(tokio::time::MissedTickBehavior::Skip);

    // Set total RAM once at startup.
    sys.refresh_memory();
    stats.mem_total_mb.store(sys.total_memory() / (1024 * 1024), Ordering::Relaxed);

    loop {
        interval.tick().await;
        if stats.done.load(Ordering::Relaxed) {
            break;
        }

        // RSS memory
        sys.refresh_processes(sysinfo::ProcessesToUpdate::Some(&[pid]), true);
        if let Some(proc) = sys.process(pid) {
            stats.mem_rss_mb.store(proc.memory() / (1024 * 1024), Ordering::Relaxed);
        }

        // Open FDs (Linux only: count entries in /proc/self/fd)
        #[cfg(target_os = "linux")]
        {
            if let Ok(entries) = std::fs::read_dir("/proc/self/fd") {
                stats.open_fds.store(entries.count() as u64, Ordering::Relaxed);
            }
        }

        // Network I/O
        nets.refresh(true);
        let mut total_rx = 0u64;
        let mut total_tx = 0u64;
        for (_name, data) in nets.iter() {
            total_rx += data.received();
            total_tx += data.transmitted();
        }
        // received()/transmitted() return bytes since last refresh → /0.5s = per-second.
        stats.net_rx_bps.store(total_rx * 2, Ordering::Relaxed);
        stats.net_tx_bps.store(total_tx * 2, Ordering::Relaxed);
    }
}

/// Shorten an error message for log display (max 60 chars).
fn short_error(error: &str) -> String {
    // Extract the most useful part from reqwest error chains.
    // e.g. "error sending request for url... : dns error: ..." → "dns error: ..."
    let s = if let Some(idx) = error.rfind(": ") {
        &error[idx + 2..]
    } else {
        error
    };
    if s.len() > 60 {
        format!("{}...", &s[..57])
    } else {
        s.to_string()
    }
}

/// Classify a reqwest::Error using its typed methods (not string matching).
/// This is reliable because reqwest sets internal flags for each error kind.
fn classify_reqwest_error(e: &reqwest::Error) -> ErrorCategory {
    if e.is_timeout() {
        ErrorCategory::Timeout
    } else if e.is_builder() {
        ErrorCategory::InvalidUrl
    } else if e.is_connect() {
        // Connection includes TCP errors, but the inner source might reveal DNS/TLS.
        // Walk the error chain to check for DNS or TLS errors inside a connect error.
        let chain = error_chain_string(e);
        let lower = chain.to_lowercase();
        if lower.contains("dns") || lower.contains("resolve") || lower.contains("no record found")
            || lower.contains("nxdomain") || lower.contains("name or service not known")
            || lower.contains("failed to lookup")
        {
            ErrorCategory::Dns
        } else if lower.contains("tls") || lower.contains("ssl") || lower.contains("certificate")
            || lower.contains("handshake alert")
        {
            ErrorCategory::Tls
        } else {
            ErrorCategory::Connection
        }
    } else if e.is_request() {
        // "error sending request" — walk chain to find the real cause.
        let chain = error_chain_string(e);
        let lower = chain.to_lowercase();
        if lower.contains("dns") || lower.contains("resolve") || lower.contains("nxdomain")
            || lower.contains("no record found") || lower.contains("failed to lookup")
        {
            ErrorCategory::Dns
        } else if lower.contains("tls") || lower.contains("ssl") || lower.contains("certificate") {
            ErrorCategory::Tls
        } else if lower.contains("timeout") || lower.contains("timed out") || lower.contains("deadline") {
            ErrorCategory::Timeout
        } else {
            ErrorCategory::Connection
        }
    } else if e.is_redirect() {
        ErrorCategory::Connection
    } else if e.is_decode() || e.is_body() {
        // Server responded but body decompression or read failed (e.g. invalid gzip/deflate).
        // Count as a connection-category failure rather than an unknown "other".
        ErrorCategory::Connection
    } else {
        ErrorCategory::Other
    }
}

/// Walk the std::error::Error source() chain, building a full error message.
/// e.g. "error sending request: connection error: tcp connect error: Connection refused"
fn error_chain_string(e: &(dyn std::error::Error + 'static)) -> String {
    let mut parts = vec![e.to_string()];
    let mut current: &dyn std::error::Error = e;
    while let Some(source) = current.source() {
        parts.push(source.to_string());
        current = source;
    }
    parts.join(": ")
}

/// Fetch a single URL using the shared reqwest client.
///
/// Returns Ok(CrawlResult) on success (any HTTP status), Err on network/parse failure.
/// The caller uses the reqwest::Error for proper classification.
async fn fetch_one(
    client: &reqwest::Client,
    seed: &SeedURL,
    timeout: Duration,
    max_body_bytes: usize,
    body_store: Option<&crate::bodystore::BodyStore>,
) -> Result<CrawlResult, (reqwest::Error, i64)> {
    let start = Instant::now();

    // Sanitize URL: fix "http:// domain" → "http://domain", trim whitespace.
    let url = sanitize_url(&seed.url);

    let profile = ua::pick_profile(&seed.domain);
    let mut req = client
        .get(&url)
        .header("User-Agent", profile.user_agent)
        .header("Accept", profile.accept)
        .header("Accept-Language", profile.accept_language)
        .header("Accept-Encoding", profile.accept_encoding)
        .header("Upgrade-Insecure-Requests", "1")
        .timeout(timeout);

    if let Some(v) = profile.sec_ch_ua {
        req = req.header("Sec-CH-UA", v);
    }
    if let Some(v) = profile.sec_ch_ua_mobile {
        req = req.header("Sec-CH-UA-Mobile", v);
    }
    if let Some(v) = profile.sec_ch_ua_platform {
        req = req.header("Sec-CH-UA-Platform", v);
    }
    if let Some(v) = profile.sec_fetch_dest {
        req = req.header("Sec-Fetch-Dest", v);
    }
    if let Some(v) = profile.sec_fetch_mode {
        req = req.header("Sec-Fetch-Mode", v);
    }
    if let Some(v) = profile.sec_fetch_site {
        req = req.header("Sec-Fetch-Site", v);
    }
    if let Some(v) = profile.sec_fetch_user {
        req = req.header("Sec-Fetch-User", v);
    }

    let response = req.send().await;

    let resp = match response {
        Ok(r) => r,
        Err(e) => {
            let elapsed = start.elapsed().as_millis() as i64;
            return Err((e, elapsed));
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

    let is_html =
        content_type.contains("text/html") || content_type.contains("application/xhtml");

    // Only read the body for 200 HTML responses where we can extract metadata.
    // For all other responses (4xx, 5xx, non-HTML, non-200), drop the response
    // immediately so the connection returns to the pool and bandwidth is saved.
    let should_read_body = status == 200 && is_html;
    let body_bytes = if should_read_body {
        match read_body_limited(resp, max_body_bytes).await {
            Ok(b) => b,
            Err(e) => {
                let elapsed = start.elapsed().as_millis() as i64;
                return Err((e, elapsed));
            }
        }
    } else {
        // Drop the response body — reqwest handles connection cleanup.
        drop(resp);
        bytes::Bytes::new()
    };

    let body_len = body_bytes.len() as i64;

    let (title, description, language) = if should_read_body && !body_bytes.is_empty() {
        extract_metadata(&body_bytes)
    } else {
        (String::new(), String::new(), String::new())
    };

    // Store body in CAS when a body store is configured and body was read.
    let body_cid = if let Some(store) = body_store {
        if should_read_body && !body_bytes.is_empty() {
            store.put(&body_bytes).unwrap_or_default()
        } else {
            String::new()
        }
    } else {
        String::new()
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
        body: String::new(), // always empty — avoids DuckDB overflow blocks
        body_cid,
    })
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

/// ASCII case-insensitive substring search.
///
/// `needle` must be pure lowercase ASCII. Searches `haystack` byte-by-byte,
/// treating 'A'-'Z' as equivalent to 'a'-'z'. Returns the byte offset of
/// the first match in `haystack`, or `None`.
///
/// Since the needle starts with an ASCII byte, the returned position is always
/// a valid UTF-8 char boundary in `haystack` — safe to slice at directly.
fn ascii_find(haystack: &str, needle: &str) -> Option<usize> {
    let h = haystack.as_bytes();
    let n = needle.as_bytes();
    if n.is_empty() {
        return Some(0);
    }
    if h.len() < n.len() {
        return None;
    }
    'outer: for i in 0..=(h.len() - n.len()) {
        for (j, &nb) in n.iter().enumerate() {
            let hb = h[i + j];
            let hb_ci = if hb.is_ascii_uppercase() { hb | 0x20 } else { hb };
            if hb_ci != nb {
                continue 'outer;
            }
        }
        return Some(i);
    }
    None
}

/// Extract text content between an opening tag and its closing tag.
/// e.g. `<title>Hello World</title>` -> "Hello World"
fn extract_tag_content(html: &str, open_tag: &str, close_tag: &str) -> String {
    // Use ascii_find instead of to_lowercase().find() to avoid char-boundary mismatches:
    // to_lowercase() can change byte lengths for non-ASCII chars (e.g. Turkish İ),
    // causing the position from `lower` to be invalid in the original `html`.
    if let Some(start) = ascii_find(html, open_tag) {
        let rest = &html[start..];
        if let Some(gt) = rest.find('>') {
            let after = &rest[gt + 1..];
            if let Some(end) = ascii_find(after, close_tag) {
                return html_decode(after[..end].trim());
            }
        }
    }
    String::new()
}

/// Extract the `content` attribute from a `<meta name="..." content="...">` tag.
fn extract_meta_content(html: &str, name: &str) -> String {
    let search = format!("name=\"{}\"", name);

    if let Some(pos) = ascii_find(html, &search) {
        // Search in a window around the match for the content attribute.
        // The meta tag could have name before or after content.
        // floor_char_boundary ensures window boundaries are valid UTF-8 char boundaries.
        let window_start = html.floor_char_boundary(pos.saturating_sub(200));
        let window_end = html.floor_char_boundary(html.len().min(pos + 500));
        let window = &html[window_start..window_end];

        if let Some(content_pos) = ascii_find(window, "content=\"") {
            let after = &window[content_pos + 9..];
            if let Some(end) = after.find('"') {
                return html_decode(&after[..end]);
            }
        }
        // Also try content='...' (single quotes)
        if let Some(content_pos) = ascii_find(window, "content='") {
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
    // Find lang="..." (double quotes)
    if let Some(pos) = ascii_find(html, "lang=\"") {
        let after = &html[pos + 6..];
        if let Some(end) = after.find('"') {
            return after[..end].to_string();
        }
    }
    // Try lang='...' (single quotes)
    if let Some(pos) = ascii_find(html, "lang='") {
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
