use crate::config::{auto_config, Config, EngineType, SysInfo};
use crate::engine::hyper_engine::HyperEngine;
use crate::engine::reqwest_engine::ReqwestEngine;
#[cfg(feature = "wreq-engine")]
use crate::engine::wreq_engine::WreqEngine;
use crate::engine::Engine;
use crate::seed::vec_to_receiver;
use crate::stats::StatsSnapshot;
use crate::types::SeedURL;
use crate::writer::{FailureWriter, ResultWriter};
use anyhow::Result;
use chrono::NaiveDateTime;
use std::sync::Arc;
use tracing;

/// Configuration for a crawl job (seeds + engine config).
pub struct JobConfig {
    pub config: Config,
    pub seeds: Vec<SeedURL>,
}

/// Result of a completed crawl job with per-pass and total stats.
#[derive(Debug, Clone)]
pub struct JobResult {
    pub pass1: StatsSnapshot,
    pub pass2: Option<StatsSnapshot>,
    pub total: StatsSnapshot,
    pub start: NaiveDateTime,
    pub end: NaiveDateTime,
    pub workers: usize,
}

/// Run a two-pass recrawl job.
///
/// Pass 1: run engine with `cfg.timeout`.
/// Pass 2: if enabled (`!cfg.no_retry` and `retry_timeout > 0`), load timeout URLs
///          from the failed DB via `load_retry_seeds`, re-run with `retry_timeout`,
///          `disable_adaptive_timeout=true`, `domain_dead_probe=2`.
///
/// # Arguments
///
/// * `seeds` - Seed URLs for pass 1
/// * `cfg` - Crawl configuration (may be mutated for auto-config)
/// * `result_writer` - Shared writer for successful crawl results
/// * `open_failure_writer` - Factory closure to create a new failure writer per pass
/// * `load_retry_seeds` - Optional closure to load timeout URLs for pass 2;
///   receives the job start time so only current-run failures are retried
pub async fn run_job(
    seed_rx: async_channel::Receiver<SeedURL>,
    seed_count: u64,
    mut cfg: Config,
    result_writer: Arc<dyn ResultWriter>,
    open_failure_writer: &dyn Fn() -> Result<Arc<dyn FailureWriter>>,
    load_retry_seeds: Option<&dyn Fn(NaiveDateTime) -> Result<Vec<SeedURL>>>,
) -> Result<JobResult> {
    let start = chrono::Utc::now().naive_utc();

    // Auto-config if workers=0 (auto mode)
    if cfg.workers == 0 {
        let si = SysInfo::gather();
        let auto = auto_config(&si, true);
        cfg.workers = auto.workers;
        if cfg.inner_n == 0 {
            cfg.inner_n = auto.inner_n;
        }
        if cfg.db_shards == 0 {
            cfg.db_shards = auto.db_shards;
        }
        if cfg.db_mem_mb == 0 {
            cfg.db_mem_mb = auto.db_mem_mb;
        }
        tracing::info!(
            workers = cfg.workers,
            inner_n = cfg.inner_n,
            db_shards = cfg.db_shards,
            db_mem_mb = cfg.db_mem_mb,
            "auto-configured from hardware"
        );
    } else if cfg.inner_n == 0 {
        let si = SysInfo::gather();
        cfg.inner_n = (si.cpu_count * 2).max(4).min(16);
    }

    // Create engine based on config
    let engine: Box<dyn Engine> = match cfg.engine {
        EngineType::Reqwest => Box::new(ReqwestEngine::new()),
        EngineType::Hyper => Box::new(HyperEngine::new()),
        #[cfg(feature = "wreq-engine")]
        EngineType::Wreq => Box::new(WreqEngine::new()),
    };

    // --- Pass 1 ---
    // Signal TUI that pass 1 is starting.
    if let Some(ref stats) = cfg.live_stats {
        stats.pass.store(1, std::sync::atomic::Ordering::Relaxed);
    }
    let failure_writer1 = open_failure_writer()?;
    tracing::info!(
        seed_count,
        timeout_ms = cfg.timeout.as_millis() as u64,
        workers = cfg.workers,
        inner_n = cfg.inner_n,
        "pass 1 starting"
    );

    let pass1 = engine
        .run(seed_rx, seed_count, &cfg, result_writer.clone(), failure_writer1.clone())
        .await?;

    // Close failure writer 1 — release DuckDB lock before loading retry seeds
    failure_writer1.close()?;

    tracing::info!(
        ok = pass1.ok,
        failed = pass1.failed,
        timeout = pass1.timeout,
        skipped = pass1.skipped,
        avg_rps = format!("{:.0}", pass1.avg_rps()),
        peak_rps = pass1.peak_rps,
        duration_s = format!("{:.1}", pass1.duration.as_secs_f64()),
        "pass 1 complete"
    );

    let mut result = JobResult {
        pass1: pass1.clone(),
        pass2: None,
        total: pass1.clone(),
        start,
        end: chrono::Utc::now().naive_utc(),
        workers: cfg.workers,
    };

    // --- Pass 2 (retry timeouts) ---
    let do_retry = !cfg.no_retry
        && cfg.retry_timeout.as_millis() > 0
        && load_retry_seeds.is_some()
        && pass1.timeout > 0;

    if do_retry {
        let loader = load_retry_seeds.unwrap();
        match loader(start) {
            Ok(retry_seeds) if !retry_seeds.is_empty() => {
                // Signal TUI: pass 2 starting.
                if let Some(ref stats) = cfg.live_stats {
                    stats.pass.store(2, std::sync::atomic::Ordering::Relaxed);
                    stats.pass2_seeds.store(retry_seeds.len() as u64, std::sync::atomic::Ordering::Relaxed);
                    stats.push_warning(format!(
                        "pass 2: {} retry URLs, timeout={}ms",
                        retry_seeds.len(),
                        cfg.retry_timeout.as_millis(),
                    ));
                }
                tracing::info!(
                    retry_seeds = retry_seeds.len(),
                    retry_timeout_ms = cfg.retry_timeout.as_millis() as u64,
                    "pass 2 starting"
                );

                let failure_writer2 = open_failure_writer()?;

                // Pass-2 config overrides:
                // - Use retry_timeout as the per-request timeout
                // - Disable adaptive timeout (fast domains would skew P95 down)
                // - Set domain_dead_probe=2 (dead domains fail fast: 1 batch then abandon)
                // - Domain timeout = 3x retry_timeout
                // - No domain_fail_threshold (let domain_dead_probe handle abandonment)
                let mut retry_cfg = cfg.clone();
                retry_cfg.timeout = cfg.retry_timeout;
                retry_cfg.domain_fail_threshold = 0;
                retry_cfg.domain_timeout_ms = (cfg.retry_timeout.as_millis() as i64) * 3;
                retry_cfg.disable_adaptive_timeout = true;
                retry_cfg.domain_dead_probe = 2;
                if cfg.pass2_workers > 0 {
                    retry_cfg.workers = cfg.pass2_workers;
                }

                let (retry_rx, retry_count) = vec_to_receiver(retry_seeds);
                let pass2_cumulative = engine
                    .run(
                        retry_rx,
                        retry_count,
                        &retry_cfg,
                        result_writer.clone(),
                        failure_writer2.clone(),
                    )
                    .await?;

                failure_writer2.close()?;

                // The engine reuses the shared live_stats, so pass2_cumulative is the
                // cumulative snapshot (pass1 + pass2). Subtract pass1 to get the delta.
                let pass2 = pass2_cumulative.delta(&pass1);

                tracing::info!(
                    ok = pass2.ok,
                    rescued = pass2.ok,
                    failed = pass2.failed,
                    timeout = pass2.timeout,
                    skipped = pass2.skipped,
                    avg_rps = format!("{:.0}", pass2.avg_rps()),
                    duration_s = format!("{:.1}", pass2.duration.as_secs_f64()),
                    "pass 2 complete"
                );

                result.total = StatsSnapshot::merge(&result.pass1, &pass2);
                result.pass2 = Some(pass2);
            }
            Ok(_) => {
                tracing::info!("pass 2 skipped: no timeout URLs to retry");
            }
            Err(e) => {
                tracing::warn!(error = %e, "failed to load retry seeds, skipping pass 2");
            }
        }
    }

    result.end = chrono::Utc::now().naive_utc();

    // Log combined summary
    tracing::info!(
        total_ok = result.total.ok,
        total_failed = result.total.failed,
        total_timeout = result.total.timeout,
        total_skipped = result.total.skipped,
        total_duration_s = format!("{:.1}", result.total.duration.as_secs_f64()),
        pass2_rescued = result.pass2.as_ref().map(|p| p.ok).unwrap_or(0),
        "job complete"
    );

    Ok(result)
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::types::{CrawlResult, FailedDomain, FailedURL};
    use std::sync::atomic::{AtomicU64, Ordering};
    use std::time::Duration;

    /// A no-op result writer for testing.
    struct NoopResultWriter;
    impl ResultWriter for NoopResultWriter {
        fn write(&self, _result: CrawlResult) -> Result<()> {
            Ok(())
        }
        fn flush(&self) -> Result<()> {
            Ok(())
        }
        fn close(&self) -> Result<()> {
            Ok(())
        }
    }

    /// A no-op failure writer that counts close() calls.
    struct CountingFailureWriter {
        close_count: AtomicU64,
    }
    impl CountingFailureWriter {
        fn new() -> Self {
            Self {
                close_count: AtomicU64::new(0),
            }
        }
    }
    impl FailureWriter for CountingFailureWriter {
        fn write_url(&self, _failed: FailedURL) -> Result<()> {
            Ok(())
        }
        fn write_domain(&self, _failed: FailedDomain) -> Result<()> {
            Ok(())
        }
        fn flush(&self) -> Result<()> {
            Ok(())
        }
        fn close(&self) -> Result<()> {
            self.close_count.fetch_add(1, Ordering::Relaxed);
            Ok(())
        }
    }

    #[tokio::test]
    async fn test_run_job_empty_seeds() {
        let result_writer: Arc<dyn ResultWriter> = Arc::new(NoopResultWriter);
        let open_failure = || -> Result<Arc<dyn FailureWriter>> {
            Ok(Arc::new(CountingFailureWriter::new()))
        };

        let mut cfg = Config::default();
        cfg.workers = 1;
        cfg.inner_n = 1;

        let (seed_rx, seed_count) = vec_to_receiver(vec![]);
        let result = run_job(seed_rx, seed_count, cfg, result_writer, &open_failure, None)
            .await
            .unwrap();

        assert_eq!(result.pass1.ok, 0);
        assert_eq!(result.pass1.total, 0);
        assert!(result.pass2.is_none());
        assert_eq!(result.total.ok, 0);
    }

    #[tokio::test]
    async fn test_run_job_no_retry_when_disabled() {
        let result_writer: Arc<dyn ResultWriter> = Arc::new(NoopResultWriter);
        let open_failure = || -> Result<Arc<dyn FailureWriter>> {
            Ok(Arc::new(CountingFailureWriter::new()))
        };

        let mut cfg = Config::default();
        cfg.workers = 1;
        cfg.inner_n = 1;
        cfg.no_retry = true;

        let (seed_rx, seed_count) = vec_to_receiver(vec![]);
        let result = run_job(seed_rx, seed_count, cfg, result_writer, &open_failure, None)
            .await
            .unwrap();

        assert!(result.pass2.is_none());
    }

    #[test]
    fn test_job_result_fields() {
        let mut snap = StatsSnapshot::empty();
        snap.ok = 100;
        snap.failed = 10;
        snap.timeout = 5;
        snap.skipped = 2;
        snap.bytes_downloaded = 1024;
        snap.total = 117;
        snap.duration = Duration::from_secs(10);
        snap.peak_rps = 50;

        let result = JobResult {
            pass1: snap.clone(),
            pass2: None,
            total: snap.clone(),
            start: chrono::Utc::now().naive_utc(),
            end: chrono::Utc::now().naive_utc(),
            workers: 100,
        };

        assert_eq!(result.pass1.ok, 100);
        assert_eq!(result.workers, 100);
        assert!(result.pass2.is_none());
    }
}
