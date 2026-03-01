use std::collections::VecDeque;
use std::sync::atomic::{AtomicBool, AtomicU64, Ordering};
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
    /// Recent warning messages (domain timeouts, abandonments). Cap 100.
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
            warnings: Mutex::new(VecDeque::with_capacity(100)),
        }
    }

    /// Push a warning message into the ring buffer (max 100 entries).
    pub fn push_warning(&self, msg: String) {
        if let Ok(mut w) = self.warnings.lock() {
            if w.len() >= 100 {
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
        }
    }
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
}

impl StatsSnapshot {
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
