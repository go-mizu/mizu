use std::collections::VecDeque;
use std::sync::atomic::{AtomicBool, AtomicU64, AtomicU8, Ordering};
use std::sync::Mutex;
use std::time::{Duration, Instant};

#[derive(Debug)]
pub struct Stats {
    pub ok: AtomicU64,
    pub failed: AtomicU64,
    pub timeout: AtomicU64,
    pub skipped: AtomicU64,
    pub bytes_downloaded: AtomicU64,
    pub total: AtomicU64,
    /// Set by engine before crawl starts; used by TUI for progress %.
    pub total_seeds: AtomicU64,
    pub start: Instant,
    /// Live peak RPS — updated every ~100ms by the engine's peak tracker task.
    pub peak_rps: AtomicU64,
    /// Set to true when the crawl completes; used to stop the peak tracker task.
    pub done: AtomicBool,
    /// Current pass (1 or 2). Set by job.rs before each engine run.
    pub pass: AtomicU8,
    /// Seed count for pass 2 (set when pass 2 begins).
    pub pass2_seeds: AtomicU64,

    // --- Error breakdown (sub-categories of `failed`) ---
    pub err_dns: AtomicU64,
    pub err_conn: AtomicU64,
    pub err_tls: AtomicU64,
    pub err_http_status: AtomicU64, // 4xx/5xx counted as errors
    pub err_other: AtomicU64,

    // --- HTTP status code distribution (for successful responses) ---
    pub status_2xx: AtomicU64,
    pub status_3xx: AtomicU64,
    pub status_4xx: AtomicU64,
    pub status_5xx: AtomicU64,

    // --- Domain tracking ---
    pub domains_total: AtomicU64,
    pub domains_done: AtomicU64,
    pub domains_abandoned: AtomicU64,

    // --- System resources (updated by sysmon task) ---
    /// RSS memory in MB (of this process)
    pub mem_rss_mb: AtomicU64,
    /// Network bytes sent since last sample (per second)
    pub net_tx_bps: AtomicU64,
    /// Network bytes received since last sample (per second)
    pub net_rx_bps: AtomicU64,
    /// Open file descriptors (this process)
    pub open_fds: AtomicU64,

    /// Recent warning messages (domain timeouts, abandonments). Cap 200.
    pub warnings: Mutex<VecDeque<String>>,
}

impl Stats {
    pub fn new() -> Self {
        Self {
            ok: AtomicU64::new(0),
            failed: AtomicU64::new(0),
            timeout: AtomicU64::new(0),
            skipped: AtomicU64::new(0),
            bytes_downloaded: AtomicU64::new(0),
            total: AtomicU64::new(0),
            total_seeds: AtomicU64::new(0),
            start: Instant::now(),
            peak_rps: AtomicU64::new(0),
            done: AtomicBool::new(false),
            pass: AtomicU8::new(1),
            pass2_seeds: AtomicU64::new(0),
            err_dns: AtomicU64::new(0),
            err_conn: AtomicU64::new(0),
            err_tls: AtomicU64::new(0),
            err_http_status: AtomicU64::new(0),
            err_other: AtomicU64::new(0),
            status_2xx: AtomicU64::new(0),
            status_3xx: AtomicU64::new(0),
            status_4xx: AtomicU64::new(0),
            status_5xx: AtomicU64::new(0),
            domains_total: AtomicU64::new(0),
            domains_done: AtomicU64::new(0),
            domains_abandoned: AtomicU64::new(0),
            mem_rss_mb: AtomicU64::new(0),
            net_tx_bps: AtomicU64::new(0),
            net_rx_bps: AtomicU64::new(0),
            open_fds: AtomicU64::new(0),
            warnings: Mutex::new(VecDeque::with_capacity(200)),
        }
    }

    /// Push a warning message into the ring buffer (max 200 entries).
    pub fn push_warning(&self, msg: String) {
        if let Ok(mut w) = self.warnings.lock() {
            if w.len() >= 200 {
                w.pop_front();
            }
            w.push_back(msg);
        }
    }

    pub fn snapshot(&self) -> StatsSnapshot {
        StatsSnapshot {
            ok: self.ok.load(Ordering::Relaxed),
            failed: self.failed.load(Ordering::Relaxed),
            timeout: self.timeout.load(Ordering::Relaxed),
            skipped: self.skipped.load(Ordering::Relaxed),
            bytes_downloaded: self.bytes_downloaded.load(Ordering::Relaxed),
            total: self.total.load(Ordering::Relaxed),
            duration: self.start.elapsed(),
            peak_rps: self.peak_rps.load(Ordering::Relaxed),
            err_dns: self.err_dns.load(Ordering::Relaxed),
            err_conn: self.err_conn.load(Ordering::Relaxed),
            err_tls: self.err_tls.load(Ordering::Relaxed),
            err_http_status: self.err_http_status.load(Ordering::Relaxed),
            err_other: self.err_other.load(Ordering::Relaxed),
            status_2xx: self.status_2xx.load(Ordering::Relaxed),
            status_3xx: self.status_3xx.load(Ordering::Relaxed),
            status_4xx: self.status_4xx.load(Ordering::Relaxed),
            status_5xx: self.status_5xx.load(Ordering::Relaxed),
        }
    }
}

/// Error category for classification.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum ErrorCategory {
    Dns,
    Connection,
    Tls,
    Timeout,
    Other,
}

/// Classify a reqwest error string into a category.
pub fn classify_error(error: &str) -> ErrorCategory {
    let lower = error.to_lowercase();

    // Timeout first (already handled separately in engine, but useful for standalone use)
    if lower.contains("timeout")
        || lower.contains("deadline")
        || lower.contains("timed out")
    {
        return ErrorCategory::Timeout;
    }

    // DNS resolution failures
    if lower.contains("dns error")
        || lower.contains("resolve")
        || lower.contains("name or service not known")
        || lower.contains("no address associated")
        || lower.contains("nxdomain")
        || lower.contains("no record found")
        || lower.contains("failed to lookup")
        || lower.contains("dns")
    {
        return ErrorCategory::Dns;
    }

    // TLS / SSL errors
    if lower.contains("tls")
        || lower.contains("ssl")
        || lower.contains("certificate")
        || lower.contains("handshake")
        || lower.contains("alert")
        || lower.contains("crypto")
    {
        return ErrorCategory::Tls;
    }

    // Connection errors
    if lower.contains("connect")
        || lower.contains("connection refused")
        || lower.contains("connection reset")
        || lower.contains("broken pipe")
        || lower.contains("network is unreachable")
        || lower.contains("no route to host")
        || lower.contains("connection aborted")
        || lower.contains("builder error")
        || lower.contains("error sending request")
        || lower.contains("tcp")
        || lower.contains("socket")
        || lower.contains("eof")
        || lower.contains("peer")
        || lower.contains("refused")
        || lower.contains("reset")
        || lower.contains("closed")
    {
        return ErrorCategory::Connection;
    }

    ErrorCategory::Other
}

#[derive(Debug, Clone)]
pub struct StatsSnapshot {
    pub ok: u64,
    pub failed: u64,
    pub timeout: u64,
    pub skipped: u64,
    pub bytes_downloaded: u64,
    pub total: u64,
    pub duration: Duration,
    pub peak_rps: u64,
    pub err_dns: u64,
    pub err_conn: u64,
    pub err_tls: u64,
    pub err_http_status: u64,
    pub err_other: u64,
    pub status_2xx: u64,
    pub status_3xx: u64,
    pub status_4xx: u64,
    pub status_5xx: u64,
}

impl StatsSnapshot {
    pub fn empty() -> Self {
        Self {
            ok: 0, failed: 0, timeout: 0, skipped: 0,
            bytes_downloaded: 0, total: 0,
            duration: Duration::ZERO, peak_rps: 0,
            err_dns: 0, err_conn: 0, err_tls: 0, err_http_status: 0, err_other: 0,
            status_2xx: 0, status_3xx: 0, status_4xx: 0, status_5xx: 0,
        }
    }

    pub fn avg_rps(&self) -> f64 {
        let secs = self.duration.as_secs_f64();
        if secs > 0.0 {
            self.total as f64 / secs
        } else {
            0.0
        }
    }

    pub fn merge(a: &StatsSnapshot, b: &StatsSnapshot) -> StatsSnapshot {
        StatsSnapshot {
            ok: a.ok + b.ok,
            failed: a.failed + b.failed,
            timeout: a.timeout + b.timeout,
            skipped: a.skipped + b.skipped,
            bytes_downloaded: a.bytes_downloaded + b.bytes_downloaded,
            total: a.total + b.total,
            duration: a.duration + b.duration,
            peak_rps: a.peak_rps.max(b.peak_rps),
            err_dns: a.err_dns + b.err_dns,
            err_conn: a.err_conn + b.err_conn,
            err_tls: a.err_tls + b.err_tls,
            err_http_status: a.err_http_status + b.err_http_status,
            err_other: a.err_other + b.err_other,
            status_2xx: a.status_2xx + b.status_2xx,
            status_3xx: a.status_3xx + b.status_3xx,
            status_4xx: a.status_4xx + b.status_4xx,
            status_5xx: a.status_5xx + b.status_5xx,
        }
    }
}

/// Lock-free latency histogram for P95-based adaptive timeout.
/// 8 buckets matching Go's adaptiveEdgesKA.
const ADAPTIVE_EDGES: [i64; 8] = [100, 250, 500, 1000, 2000, 3500, 5000, 10000];

pub struct AdaptiveTimeout {
    buckets: [AtomicU64; 8],
    total: AtomicU64,
}

impl AdaptiveTimeout {
    pub fn new() -> Self {
        Self {
            buckets: std::array::from_fn(|_| AtomicU64::new(0)),
            total: AtomicU64::new(0),
        }
    }

    pub fn record(&self, ms: i64) {
        self.total.fetch_add(1, Ordering::Relaxed);
        for (i, &edge) in ADAPTIVE_EDGES.iter().enumerate() {
            if ms < edge {
                self.buckets[i].fetch_add(1, Ordering::Relaxed);
                return;
            }
        }
        self.buckets[7].fetch_add(1, Ordering::Relaxed);
    }

    /// Returns P95x2 clamped to [500ms, ceiling]. Returns None if <5 samples.
    pub fn timeout(&self, ceiling: Duration) -> Option<Duration> {
        let n = self.total.load(Ordering::Relaxed);
        if n < 5 {
            return None;
        }
        let target = (n as f64 * 0.95) as u64;
        let mut cum = 0u64;
        for (i, &edge) in ADAPTIVE_EDGES.iter().enumerate() {
            cum += self.buckets[i].load(Ordering::Relaxed);
            if cum >= target {
                let ms = (edge * 2).max(500);
                let ceil_ms = ceiling.as_millis() as i64;
                let result_ms = ms.min(ceil_ms);
                return Some(Duration::from_millis(result_ms as u64));
            }
        }
        Some(ceiling)
    }

    pub fn p95_ms(&self) -> Option<i64> {
        let n = self.total.load(Ordering::Relaxed);
        if n < 10 {
            return None;
        }
        let target = (n as f64 * 0.95) as u64;
        let mut cum = 0u64;
        for (i, &edge) in ADAPTIVE_EDGES.iter().enumerate() {
            cum += self.buckets[i].load(Ordering::Relaxed);
            if cum >= target {
                return Some(edge);
            }
        }
        Some(ADAPTIVE_EDGES[7])
    }
}

/// Tracks peak RPS using a sliding 1-second window.
pub struct PeakTracker {
    count: AtomicU64,
    last_reset: std::sync::Mutex<Instant>,
    peak: AtomicU64,
}

impl PeakTracker {
    pub fn new() -> Self {
        Self {
            count: AtomicU64::new(0),
            last_reset: std::sync::Mutex::new(Instant::now()),
            peak: AtomicU64::new(0),
        }
    }

    pub fn record(&self) {
        let c = self.count.fetch_add(1, Ordering::Relaxed) + 1;
        if let Ok(mut last) = self.last_reset.try_lock() {
            let elapsed = last.elapsed();
            if elapsed >= Duration::from_secs(1) {
                let rps = (c as f64 / elapsed.as_secs_f64()) as u64;
                self.peak.fetch_max(rps, Ordering::Relaxed);
                self.count.store(0, Ordering::Relaxed);
                *last = Instant::now();
            }
        }
    }

    pub fn peak(&self) -> u64 {
        self.peak.load(Ordering::Relaxed)
    }
}
