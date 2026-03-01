use crawler_lib::stats::StatsSnapshot;
use std::time::Duration;

/// Format a duration as a human-readable string (e.g., "1h23m45s", "5m02s", "30s").
pub fn format_duration(d: Duration) -> String {
    let total_secs = d.as_secs();
    let h = total_secs / 3600;
    let m = (total_secs % 3600) / 60;
    let s = total_secs % 60;
    if h > 0 {
        format!("{}h{:02}m{:02}s", h, m, s)
    } else if m > 0 {
        format!("{}m{:02}s", m, s)
    } else {
        format!("{}s", s)
    }
}

/// Print a formatted progress line to stdout (no newline suppression — one line per tick).
#[allow(dead_code)]
pub fn print_progress(elapsed: Duration, snap: &StatsSnapshot) {
    let elapsed_str = format_duration(elapsed);
    println!(
        "[{}] ok={} | timeout={} | failed={} | total={} | avg={:.0} rps | peak={} rps",
        elapsed_str,
        snap.ok,
        snap.timeout,
        snap.failed,
        snap.total,
        snap.avg_rps(),
        snap.peak_rps,
    );
}

/// Print the final summary for a completed two-pass job.
///
/// Matches the format described in the task spec (Go's printFinalSummary).
pub fn print_summary(
    pass1: &StatsSnapshot,
    pass2: Option<&StatsSnapshot>,
    total: &StatsSnapshot,
    workers: usize,
) {
    let pct = |n: u64, d: u64| -> f64 {
        if d == 0 {
            0.0
        } else {
            n as f64 * 100.0 / d as f64
        }
    };

    println!();
    println!("=== Pass 1 ===");
    println!("  OK:       {:>8}  ({:.1}%)", pass1.ok, pct(pass1.ok, pass1.total));
    println!("  Timeout:  {:>8}  ({:.1}%)", pass1.timeout, pct(pass1.timeout, pass1.total));
    println!("  Failed:   {:>8}  ({:.1}%)", pass1.failed, pct(pass1.failed, pass1.total));
    if pass1.failed > 0 {
        println!("    dns:    {:>8}  conn: {}  tls: {}  other: {}",
            pass1.err_dns, pass1.err_conn, pass1.err_tls, pass1.err_other);
    }
    println!("  Skipped:  {:>8}  ({:.1}%)", pass1.skipped, pct(pass1.skipped, pass1.total));
    println!("  Total:    {:>8}", pass1.total);
    if pass1.status_2xx + pass1.status_3xx + pass1.status_4xx + pass1.status_5xx > 0 {
        println!("  HTTP:     2xx={}  3xx={}  4xx={}  5xx={}",
            pass1.status_2xx, pass1.status_3xx, pass1.status_4xx, pass1.status_5xx);
    }
    println!(
        "  Avg RPS:  {:>8.0}    Peak: {} rps",
        pass1.avg_rps(),
        pass1.peak_rps
    );
    println!("  Duration: {}", format_duration(pass1.duration));

    if let Some(p2) = pass2 {
        println!();
        println!("=== Pass 2 ===");
        println!("  Rescued:  {:>8}  ({:.1}%)", p2.ok, pct(p2.ok, p2.total));
        println!("  Timeout:  {:>8}  ({:.1}%)", p2.timeout, pct(p2.timeout, p2.total));
        println!("  Failed:   {:>8}  ({:.1}%)", p2.failed, pct(p2.failed, p2.total));
        println!("  Skipped:  {:>8}  ({:.1}%)", p2.skipped, pct(p2.skipped, p2.total));
        println!("  Total:    {:>8}", p2.total);
        println!(
            "  Avg RPS:  {:>8.0}    Peak: {} rps",
            p2.avg_rps(),
            p2.peak_rps
        );
        println!("  Duration: {}", format_duration(p2.duration));
    }

    println!();
    println!("=== Total ===");
    println!(
        "  OK:       {:>8} / {} ({:.1}%)",
        total.ok,
        total.total,
        pct(total.ok, total.total)
    );
    println!(
        "  Timeout:  {:>8}  ({:.1}%)",
        total.timeout,
        pct(total.timeout, total.total)
    );
    println!(
        "  Failed:   {:>8}  ({:.1}%)",
        total.failed,
        pct(total.failed, total.total)
    );
    if total.failed > 0 {
        println!("    dns:    {:>8}  conn: {}  tls: {}  other: {}",
            total.err_dns, total.err_conn, total.err_tls, total.err_other);
    }
    println!("  Workers:  {}", workers);
    println!("  Duration: {}", format_duration(total.duration));
    println!();
}
