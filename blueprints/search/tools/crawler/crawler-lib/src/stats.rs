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
    /// Cumulative total processed URL count when pass-2 began (0 = still in pass-1).
    /// Used by GUI to compute pass-2 progress independently of cumulative total.
    pub pass1_total: AtomicU64,
    /// Elapsed ms when pass-2 began (0 = still in pass-1).
    /// Used by GUI to compute accurate pass-2 avg RPS (not diluted by pass-1 duration).
    pub pass2_start_elapsed_ms: AtomicU64,

    // --- Error breakdown (sub-categories of `failed`) ---
    pub err_invalid_url: AtomicU64, // garbage/unparseable URLs (builder error)
    pub err_dns: AtomicU64,
    pub err_conn: AtomicU64,
    pub err_tls: AtomicU64,
    pub err_http_status: AtomicU64, // 4xx/5xx counted as errors
    pub err_other: AtomicU64,

    // --- DNS sub-categories ---
    pub dns_nxdomain: AtomicU64,    // no records found / NXDOMAIN
    pub dns_malformed: AtomicU64,   // malformed label, invalid chars, too long
    pub dns_other: AtomicU64,       // servfail, network error, etc

    // --- Connection sub-categories ---
    pub conn_refused: AtomicU64,    // connection refused (os error 111)
    pub conn_reset: AtomicU64,      // reset by peer (os error 104)
    pub conn_eof: AtomicU64,        // unexpected EOF / connection closed
    pub conn_other: AtomicU64,

    // --- Timeout sub-categories ---
    pub timeout_connect: AtomicU64, // TCP/TLS connect timeout (< cfg.timeout)
    pub timeout_response: AtomicU64,// full HTTP timeout (>= cfg.timeout)

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
    /// Total system RAM in MB (set once at startup)
    pub mem_total_mb: AtomicU64,
    /// Network bytes sent since last sample (per second)
    pub net_tx_bps: AtomicU64,
    /// Network bytes received since last sample (per second)
    pub net_rx_bps: AtomicU64,
    /// Open file descriptors (this process)
    pub open_fds: AtomicU64,

    // --- Disk stats (updated by disk_sampler every 10s) ---
    /// Number of live seg_*.bin files
    pub disk_seg_files: AtomicU64,
    /// Total MB of seg_*.bin files
    pub disk_seg_mb: AtomicU64,
    /// Total MB of results_*.duckdb shards
    pub disk_duckdb_mb: AtomicU64,
    /// Row count in result DuckDB shards (set once, post-drain)
    pub disk_results_rows: AtomicU64,
    /// Total MB of failures/ dir
    pub disk_failures_mb: AtomicU64,
    /// Row count in failed.duckdb (set once, post-drain)
    pub disk_failed_rows: AtomicU64,
    /// File count in bodies/ CAS dir
    pub disk_bodies_count: AtomicU64,
    /// Total MB of bodies/ dir
    pub disk_bodies_mb: AtomicU64,
    /// Grand total disk MB (seg + duckdb + failures + bodies)
    pub disk_total_mb: AtomicU64,
    /// Unix seconds of last disk scan
    pub disk_last_updated: AtomicU64,

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
            pass1_total: AtomicU64::new(0),
            pass2_start_elapsed_ms: AtomicU64::new(0),
            err_invalid_url: AtomicU64::new(0),
            err_dns: AtomicU64::new(0),
            err_conn: AtomicU64::new(0),
            err_tls: AtomicU64::new(0),
            err_http_status: AtomicU64::new(0),
            err_other: AtomicU64::new(0),
            dns_nxdomain: AtomicU64::new(0),
            dns_malformed: AtomicU64::new(0),
            dns_other: AtomicU64::new(0),
            conn_refused: AtomicU64::new(0),
            conn_reset: AtomicU64::new(0),
            conn_eof: AtomicU64::new(0),
            conn_other: AtomicU64::new(0),
            timeout_connect: AtomicU64::new(0),
            timeout_response: AtomicU64::new(0),
            status_2xx: AtomicU64::new(0),
            status_3xx: AtomicU64::new(0),
            status_4xx: AtomicU64::new(0),
            status_5xx: AtomicU64::new(0),
            domains_total: AtomicU64::new(0),
            domains_done: AtomicU64::new(0),
            domains_abandoned: AtomicU64::new(0),
            mem_rss_mb: AtomicU64::new(0),
            mem_total_mb: AtomicU64::new(0),
            net_tx_bps: AtomicU64::new(0),
            net_rx_bps: AtomicU64::new(0),
            open_fds: AtomicU64::new(0),
            disk_seg_files: AtomicU64::new(0),
            disk_seg_mb: AtomicU64::new(0),
            disk_duckdb_mb: AtomicU64::new(0),
            disk_results_rows: AtomicU64::new(0),
            disk_failures_mb: AtomicU64::new(0),
            disk_failed_rows: AtomicU64::new(0),
            disk_bodies_count: AtomicU64::new(0),
            disk_bodies_mb: AtomicU64::new(0),
            disk_total_mb: AtomicU64::new(0),
            disk_last_updated: AtomicU64::new(0),
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
            err_invalid_url: self.err_invalid_url.load(Ordering::Relaxed),
            err_dns: self.err_dns.load(Ordering::Relaxed),
            err_conn: self.err_conn.load(Ordering::Relaxed),
            err_tls: self.err_tls.load(Ordering::Relaxed),
            err_http_status: self.err_http_status.load(Ordering::Relaxed),
            err_other: self.err_other.load(Ordering::Relaxed),
            dns_nxdomain: self.dns_nxdomain.load(Ordering::Relaxed),
            dns_malformed: self.dns_malformed.load(Ordering::Relaxed),
            dns_other: self.dns_other.load(Ordering::Relaxed),
            conn_refused: self.conn_refused.load(Ordering::Relaxed),
            conn_reset: self.conn_reset.load(Ordering::Relaxed),
            conn_eof: self.conn_eof.load(Ordering::Relaxed),
            conn_other: self.conn_other.load(Ordering::Relaxed),
            timeout_connect: self.timeout_connect.load(Ordering::Relaxed),
            timeout_response: self.timeout_response.load(Ordering::Relaxed),
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
    InvalidUrl,
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
    pub err_invalid_url: u64,
    pub err_dns: u64,
    pub err_conn: u64,
    pub err_tls: u64,
    pub err_http_status: u64,
    pub err_other: u64,
    // Sub-categories
    pub dns_nxdomain: u64,
    pub dns_malformed: u64,
    pub dns_other: u64,
    pub conn_refused: u64,
    pub conn_reset: u64,
    pub conn_eof: u64,
    pub conn_other: u64,
    pub timeout_connect: u64,
    pub timeout_response: u64,
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
            err_invalid_url: 0, err_dns: 0, err_conn: 0, err_tls: 0, err_http_status: 0, err_other: 0,
            dns_nxdomain: 0, dns_malformed: 0, dns_other: 0,
            conn_refused: 0, conn_reset: 0, conn_eof: 0, conn_other: 0,
            timeout_connect: 0, timeout_response: 0,
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

    /// Compute delta of `self` relative to a `base` snapshot taken earlier.
    ///
    /// Use this when `live_stats` is shared across multiple engine passes: the
    /// second pass snapshot is cumulative (base + new work), so subtracting the
    /// first-pass snapshot yields only the new work done in the second pass.
    /// All additive counters are saturating-subtracted; `peak_rps` is kept as-is.
    pub fn delta(&self, base: &StatsSnapshot) -> StatsSnapshot {
        StatsSnapshot {
            ok:               self.ok.saturating_sub(base.ok),
            failed:           self.failed.saturating_sub(base.failed),
            timeout:          self.timeout.saturating_sub(base.timeout),
            skipped:          self.skipped.saturating_sub(base.skipped),
            bytes_downloaded: self.bytes_downloaded.saturating_sub(base.bytes_downloaded),
            total:            self.total.saturating_sub(base.total),
            duration:         self.duration.saturating_sub(base.duration),
            peak_rps:         self.peak_rps,
            err_invalid_url:  self.err_invalid_url.saturating_sub(base.err_invalid_url),
            err_dns:          self.err_dns.saturating_sub(base.err_dns),
            err_conn:         self.err_conn.saturating_sub(base.err_conn),
            err_tls:          self.err_tls.saturating_sub(base.err_tls),
            err_http_status:  self.err_http_status.saturating_sub(base.err_http_status),
            err_other:        self.err_other.saturating_sub(base.err_other),
            dns_nxdomain:     self.dns_nxdomain.saturating_sub(base.dns_nxdomain),
            dns_malformed:    self.dns_malformed.saturating_sub(base.dns_malformed),
            dns_other:        self.dns_other.saturating_sub(base.dns_other),
            conn_refused:     self.conn_refused.saturating_sub(base.conn_refused),
            conn_reset:       self.conn_reset.saturating_sub(base.conn_reset),
            conn_eof:         self.conn_eof.saturating_sub(base.conn_eof),
            conn_other:       self.conn_other.saturating_sub(base.conn_other),
            timeout_connect:  self.timeout_connect.saturating_sub(base.timeout_connect),
            timeout_response: self.timeout_response.saturating_sub(base.timeout_response),
            status_2xx:       self.status_2xx.saturating_sub(base.status_2xx),
            status_3xx:       self.status_3xx.saturating_sub(base.status_3xx),
            status_4xx:       self.status_4xx.saturating_sub(base.status_4xx),
            status_5xx:       self.status_5xx.saturating_sub(base.status_5xx),
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
            err_invalid_url: a.err_invalid_url + b.err_invalid_url,
            err_dns: a.err_dns + b.err_dns,
            err_conn: a.err_conn + b.err_conn,
            err_tls: a.err_tls + b.err_tls,
            err_http_status: a.err_http_status + b.err_http_status,
            err_other: a.err_other + b.err_other,
            dns_nxdomain: a.dns_nxdomain + b.dns_nxdomain,
            dns_malformed: a.dns_malformed + b.dns_malformed,
            dns_other: a.dns_other + b.dns_other,
            conn_refused: a.conn_refused + b.conn_refused,
            conn_reset: a.conn_reset + b.conn_reset,
            conn_eof: a.conn_eof + b.conn_eof,
            conn_other: a.conn_other + b.conn_other,
            timeout_connect: a.timeout_connect + b.timeout_connect,
            timeout_response: a.timeout_response + b.timeout_response,
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

#[cfg(test)]
mod adaptive_tests {
    use super::*;
    use std::time::Duration;

    #[test]
    fn adaptive_floor_never_drops_below_cfg_timeout() {
        let adaptive = AdaptiveTimeout::new();
        // Record 20 very fast responses (100 ms) — P95 = 100 ms → raw adaptive = 200 ms
        for _ in 0..20 {
            adaptive.record(100);
        }

        let cfg_timeout = Duration::from_millis(1000);
        let ceiling = Duration::from_secs(600);
        let adaptive_val = adaptive.timeout(ceiling).unwrap_or(cfg_timeout);

        // With floor: max(200ms, 1000ms) = 1000ms
        let effective = adaptive_val
            .max(cfg_timeout)
            .min(cfg_timeout.saturating_mul(3));

        assert!(
            effective >= cfg_timeout,
            "effective {}ms should be >= cfg {}ms",
            effective.as_millis(),
            cfg_timeout.as_millis()
        );
    }

    #[test]
    fn adaptive_extends_for_slow_domains() {
        let adaptive = AdaptiveTimeout::new();
        // Record 20 slow responses (3000 ms) — P95 = 3000 ms → raw adaptive = 6000 ms
        for _ in 0..20 {
            adaptive.record(3000);
        }

        let cfg_timeout = Duration::from_millis(1000);
        let ceiling = Duration::from_secs(600);
        let adaptive_val = adaptive.timeout(ceiling).unwrap_or(cfg_timeout);

        let effective = adaptive_val
            .max(cfg_timeout)
            .min(cfg_timeout.saturating_mul(3));

        // Should be capped at 3× = 3000ms
        assert_eq!(effective, Duration::from_millis(3000));
    }
}
