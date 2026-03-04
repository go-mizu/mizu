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

/// Print detailed error breakdown for a pass.
fn print_error_breakdown(s: &StatsSnapshot) {
    if s.failed > 0 {
        println!("    inv:    {:>8}  dns: {}  conn: {}  tls: {}  other: {}",
            s.err_invalid_url, s.err_dns, s.err_conn, s.err_tls, s.err_other);
    }
    // Timeout sub-categories
    if s.timeout > 0 {
        println!("    timeout: connect={} response={}",
            s.timeout_connect, s.timeout_response);
    }
    // DNS sub-categories
    if s.err_dns > 0 {
        println!("    dns:     nxdomain={} malformed={} other={}",
            s.dns_nxdomain, s.dns_malformed, s.dns_other);
    }
    // Connection sub-categories
    if s.err_conn > 0 {
        println!("    conn:    refused={} reset={} eof={} other={}",
            s.conn_refused, s.conn_reset, s.conn_eof, s.conn_other);
    }
}

/// Print the final summary for a completed two-pass job.
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
    let p1_all = pass1.total + pass1.skipped;
    println!("  OK:       {:>8}  ({:.1}%)", pass1.ok, pct(pass1.ok, p1_all));
    println!("  Timeout:  {:>8}  ({:.1}%)", pass1.timeout, pct(pass1.timeout, p1_all));
    println!("  Failed:   {:>8}  ({:.1}%)", pass1.failed, pct(pass1.failed, p1_all));
    print_error_breakdown(pass1);
    println!("  Skipped:  {:>8}  ({:.1}%)", pass1.skipped, pct(pass1.skipped, p1_all));
    println!("  Total:    {:>8}", p1_all);
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
        let p2_all = p2.total + p2.skipped;
        println!("  Rescued:  {:>8}  ({:.1}%)", p2.ok, pct(p2.ok, p2_all));
        println!("  Timeout:  {:>8}  ({:.1}%)", p2.timeout, pct(p2.timeout, p2_all));
        println!("  Failed:   {:>8}  ({:.1}%)", p2.failed, pct(p2.failed, p2_all));
        print_error_breakdown(p2);
        println!("  Skipped:  {:>8}  ({:.1}%)", p2.skipped, pct(p2.skipped, p2_all));
        println!("  Total:    {:>8}", p2_all);
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
    print_error_breakdown(total);
    println!("  Workers:  {}", workers);
    println!("  Duration: {}", format_duration(total.duration));
    println!();
}
