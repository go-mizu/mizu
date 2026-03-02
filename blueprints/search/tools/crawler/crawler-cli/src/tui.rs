//! Modern crawler dashboard — alternate screen, no log corruption, pass-aware.
//!
//! Layout:
//!  ┌ HN Recrawl ──────────── reqwest · binary · 16,000w · 1s ┐
//!  │ Requests             │ Throughput                        │
//!  │  OK      92,431 46.2%│ ▁▂▄▇█▆▃▁▂▄▆▇█▇▅▃▁▂▄▆▇█▇▅▃▁▂▄▆ │
//!  │  Failed   8,124  4.1%│                                   │
//!  │    dns   6,012       │  Avg  3,486 /s   Peak 10,444 /s  │
//!  │    conn  1,841       │  Elapsed 1m26s   ETA    2m08s    │
//!  │    tls     271       │                                   │
//!  │  Timeout 56,781 28.4%│ Domains  42,501 / 75,492 (16 ab) │
//!  │  Skipped 42,664 21.3%│ HTTP 2xx 91k 3xx 1k 4xx 2k 5xx 0│
//!  │  Total  200,000      │ RAM 245MB  FDs 12,401  Net 48MB/s│
//!  │──────────────────────────────────────────────────────────│
//!  │ ████████████████████░░░░  46.2%  92,431 / 200,000        │
//!  │ Log                                                      │
//!  │  engine: 200000 seeds, 75492 domains, 16000 workers      │
//!  │  dns: dead.example.com (NXDOMAIN)                        │
//!  │  abandoned slow.example.com (3 timeouts, 0 ok)           │
//!  │  pass 2: 43,560 retry URLs, timeout=15000ms              │
//!  └─ q quit ──────────────────────────── 275.4 MB downloaded ┘

use std::collections::VecDeque;
use std::io::{self, IsTerminal};
use std::sync::atomic::{AtomicBool, Ordering};
use std::sync::Arc;
use std::time::{Duration, Instant};

use crossterm::{
    event::{self, Event, KeyCode},
    execute,
    terminal::{disable_raw_mode, enable_raw_mode, EnterAlternateScreen, LeaveAlternateScreen},
};
use ratatui::{
    backend::CrosstermBackend,
    layout::{Constraint, Layout, Rect},
    style::{Color, Modifier, Style},
    symbols,
    text::{Line, Span},
    widgets::{Block, Borders, LineGauge, List, ListItem, Paragraph, Sparkline},
    Terminal,
};

use crawler_lib::stats::Stats;

// ---------------------------------------------------------------------------
// Public API
// ---------------------------------------------------------------------------

/// Configuration passed from the CLI so the TUI can display engine/writer/workers.
pub struct TuiConfig {
    pub title: String,
    pub engine: String,
    pub writer: String,
    pub workers: String,
    pub timeout_ms: u64,
}

/// Handle to a running TUI thread.
pub struct TuiHandle {
    stop: Arc<AtomicBool>,
    thread: Option<std::thread::JoinHandle<()>>,
}

impl TuiHandle {
    pub fn stop_and_join(mut self) {
        self.stop.store(true, Ordering::Relaxed);
        if let Some(h) = self.thread.take() {
            let _ = h.join();
        }
    }
}

/// Spawn the TUI dashboard on a dedicated OS thread.
/// Returns `None` when stdout is not a terminal.
pub fn spawn(stats: Arc<Stats>, cfg: TuiConfig) -> Option<TuiHandle> {
    if !io::stdout().is_terminal() {
        return None;
    }

    let stop = Arc::new(AtomicBool::new(false));
    let stop2 = stop.clone();

    let thread = std::thread::spawn(move || {
        if let Err(e) = run_dashboard(stats, stop2, cfg) {
            // TUI failed — restore terminal and print error.
            let _ = disable_raw_mode();
            let _ = execute!(io::stdout(), LeaveAlternateScreen);
            eprintln!("[tui] failed: {e}");
        }
    });

    Some(TuiHandle { stop, thread: Some(thread) })
}

// ---------------------------------------------------------------------------
// Render state
// ---------------------------------------------------------------------------

struct RenderState {
    rps_history: VecDeque<u64>,
    prev_total: u64,
    prev_ts: Instant,
    first_tick: bool,
    last_pass: u8,
}

impl RenderState {
    fn new() -> Self {
        Self {
            rps_history: VecDeque::with_capacity(256),
            prev_total: 0,
            prev_ts: Instant::now(),
            first_tick: true,
            last_pass: 1,
        }
    }

    fn tick(&mut self, total: u64, max_samples: usize) {
        let now = Instant::now();
        let dt = now.duration_since(self.prev_ts).as_secs_f64();

        // First tick: sync to current total to avoid initial spike.
        if self.first_tick {
            self.prev_total = total;
            self.prev_ts = now;
            self.first_tick = false;
            return;
        }

        if dt >= 0.05 {
            let delta = total.saturating_sub(self.prev_total);
            let rps = (delta as f64 / dt).round() as u64;
            self.rps_history.push_back(rps);
            // Trim to panel width so every bar maps to a real data point.
            let cap = max_samples.max(30);
            while self.rps_history.len() > cap {
                self.rps_history.pop_front();
            }
            self.prev_total = total;
            self.prev_ts = now;
        }
    }
}

// ---------------------------------------------------------------------------
// Dashboard loop
// ---------------------------------------------------------------------------

fn run_dashboard(
    stats: Arc<Stats>,
    stop: Arc<AtomicBool>,
    cfg: TuiConfig,
) -> anyhow::Result<()> {
    // Restore terminal on panic.
    let original_hook = std::panic::take_hook();
    std::panic::set_hook(Box::new(move |info| {
        let _ = disable_raw_mode();
        let _ = execute!(io::stdout(), LeaveAlternateScreen);
        original_hook(info);
    }));

    enable_raw_mode()?;
    execute!(io::stdout(), EnterAlternateScreen)?;

    let backend = CrosstermBackend::new(io::stdout());
    let mut terminal = Terminal::new(backend)?;

    let mut state = RenderState::new();

    loop {
        let total = stats.total.load(Ordering::Relaxed);
        // Use terminal width to size sparkline samples.
        let spark_width = terminal.size().map(|s| s.width as usize / 2).unwrap_or(40);
        state.tick(total, spark_width);

        // Detect pass transition.
        let cur_pass = stats.pass.load(Ordering::Relaxed);
        if cur_pass != state.last_pass {
            state.rps_history.clear();
            state.first_tick = true;
            state.last_pass = cur_pass;
        }

        terminal.draw(|f| render(f, &stats, &cfg, &state))?;

        if event::poll(Duration::from_millis(80))? {
            if let Event::Key(key) = event::read()? {
                match key.code {
                    KeyCode::Char('q') | KeyCode::Esc => break,
                    _ => {}
                }
            }
        }

        if stop.load(Ordering::Relaxed) {
            // Final render.
            let total = stats.total.load(Ordering::Relaxed);
            state.tick(total, spark_width);
            terminal.draw(|f| render(f, &stats, &cfg, &state))?;
            std::thread::sleep(Duration::from_millis(100));
            break;
        }
    }

    disable_raw_mode()?;
    execute!(terminal.backend_mut(), LeaveAlternateScreen)?;
    terminal.show_cursor()?;

    // Restore the original panic hook.
    let _ = std::panic::take_hook();

    Ok(())
}

// ---------------------------------------------------------------------------
// Top-level render
// ---------------------------------------------------------------------------

fn render(frame: &mut ratatui::Frame, stats: &Stats, cfg: &TuiConfig, state: &RenderState) {
    let area = frame.area();

    let ok = stats.ok.load(Ordering::Relaxed);
    let failed = stats.failed.load(Ordering::Relaxed);
    let timeout = stats.timeout.load(Ordering::Relaxed);
    let skipped = stats.skipped.load(Ordering::Relaxed);
    let total = stats.total.load(Ordering::Relaxed);
    let total_seeds = stats.total_seeds.load(Ordering::Relaxed);
    let peak_rps = stats.peak_rps.load(Ordering::Relaxed);
    let bytes = stats.bytes_downloaded.load(Ordering::Relaxed);
    let elapsed = stats.start.elapsed();
    let pass = stats.pass.load(Ordering::Relaxed);

    // Error breakdown
    let err_inv = stats.err_invalid_url.load(Ordering::Relaxed);
    let err_dns = stats.err_dns.load(Ordering::Relaxed);
    let err_conn = stats.err_conn.load(Ordering::Relaxed);
    let err_tls = stats.err_tls.load(Ordering::Relaxed);
    let err_other = stats.err_other.load(Ordering::Relaxed);

    // Status codes
    let s2xx = stats.status_2xx.load(Ordering::Relaxed);
    let s3xx = stats.status_3xx.load(Ordering::Relaxed);
    let s4xx = stats.status_4xx.load(Ordering::Relaxed);
    let s5xx = stats.status_5xx.load(Ordering::Relaxed);

    // Domains
    let dom_total = stats.domains_total.load(Ordering::Relaxed);
    let dom_abandoned = stats.domains_abandoned.load(Ordering::Relaxed);

    // System
    let mem_rss = stats.mem_rss_mb.load(Ordering::Relaxed);
    let net_rx = stats.net_rx_bps.load(Ordering::Relaxed);
    let net_tx = stats.net_tx_bps.load(Ordering::Relaxed);
    let open_fds = stats.open_fds.load(Ordering::Relaxed);

    let avg_rps = if elapsed.as_secs_f64() > 0.1 {
        total as f64 / elapsed.as_secs_f64()
    } else {
        0.0
    };

    let ratio = if total_seeds > 0 {
        (total as f64 / total_seeds as f64).clamp(0.0, 1.0)
    } else {
        0.0
    };

    let eta: Option<Duration> = if avg_rps > 1.0 && total_seeds > total {
        let remaining = total_seeds - total;
        Some(Duration::from_secs_f64(remaining as f64 / avg_rps))
    } else {
        None
    };

    // Outer block with header title.
    let pass_label = if pass > 1 { format!(" [Pass {}]", pass) } else { String::new() };
    let header_right = format!(
        " {} · {} · {}w · {}ms{} ",
        cfg.engine, cfg.writer, cfg.workers, cfg.timeout_ms, pass_label,
    );

    let outer_block = Block::default()
        .borders(Borders::ALL)
        .border_style(Style::default().fg(Color::DarkGray))
        .title(Line::from(vec![
            Span::styled(
                format!(" {} ", cfg.title),
                Style::default().fg(Color::Cyan).add_modifier(Modifier::BOLD),
            ),
        ]))
        .title_bottom(Line::from(vec![
            Span::styled(" q quit ", Style::default().fg(Color::DarkGray)),
            Span::raw(format!("{:>width$}", fmt_bytes(bytes), width = area.width.saturating_sub(12) as usize)),
        ]));

    let inner = outer_block.inner(area);
    frame.render_widget(outer_block, area);

    // Right-align config info on top border.
    if area.width > header_right.len() as u16 + 4 {
        let x = area.x + area.width - header_right.len() as u16 - 1;
        let hdr_area = Rect::new(x, area.y, header_right.len() as u16, 1);
        frame.render_widget(
            Paragraph::new(Span::styled(
                &header_right,
                Style::default().fg(Color::DarkGray),
            )),
            hdr_area,
        );
    }

    // Inner layout: main | progress | log
    let [main_area, progress_area, log_area] = Layout::vertical([
        Constraint::Length(10),
        Constraint::Length(1),
        Constraint::Min(2),
    ]).areas(inner);

    render_main(
        frame, main_area,
        ok, failed, timeout, skipped, total,
        err_inv, err_dns, err_conn, err_tls, err_other,
        avg_rps, peak_rps, elapsed, eta,
        s2xx, s3xx, s4xx, s5xx,
        dom_total, dom_abandoned,
        mem_rss, net_rx, net_tx, open_fds,
        state,
    );
    render_progress(frame, progress_area, ratio, total, total_seeds, eta);
    render_log(frame, log_area, stats);
}

// ---------------------------------------------------------------------------
// Main: counters + throughput
// ---------------------------------------------------------------------------

#[allow(clippy::too_many_arguments)]
fn render_main(
    frame: &mut ratatui::Frame,
    area: Rect,
    ok: u64, failed: u64, timeout: u64, skipped: u64, total: u64,
    err_inv: u64, err_dns: u64, err_conn: u64, err_tls: u64, err_other: u64,
    avg_rps: f64, peak_rps: u64, elapsed: Duration, eta: Option<Duration>,
    s2xx: u64, s3xx: u64, s4xx: u64, s5xx: u64,
    dom_total: u64, dom_abandoned: u64,
    mem_rss: u64, net_rx: u64, net_tx: u64, open_fds: u64,
    state: &RenderState,
) {
    let [left, right] = Layout::horizontal([
        Constraint::Length(28),
        Constraint::Min(24),
    ]).areas(area);

    render_counters(frame, left, ok, failed, timeout, skipped, total, err_inv, err_dns, err_conn, err_tls, err_other);
    render_throughput(
        frame, right, avg_rps, peak_rps, elapsed, eta,
        s2xx, s3xx, s4xx, s5xx,
        dom_total, dom_abandoned,
        mem_rss, net_rx, net_tx, open_fds,
        state,
    );
}

fn render_counters(
    frame: &mut ratatui::Frame,
    area: Rect,
    ok: u64, failed: u64, timeout: u64, skipped: u64, total: u64,
    err_inv: u64, err_dns: u64, err_conn: u64, err_tls: u64, err_other: u64,
) {
    let dim = Style::default().fg(Color::DarkGray);
    let bold = Style::default().add_modifier(Modifier::BOLD);

    let mut lines: Vec<Line> = vec![
        Span::styled(" Requests", dim).into(),
        counter_line(" OK     ", Color::Green, ok, total),
        counter_line(" Failed ", Color::Red, failed, total),
    ];

    // Error breakdown (sub-lines under Failed, only when there are failures)
    if failed > 0 {
        lines.push(Line::from(vec![
            Span::styled("   inv ", Style::default().fg(Color::Rgb(180, 180, 180))),
            Span::styled(format!("{:>6}", fmt_count(err_inv)), dim),
            Span::styled(" dns ", Style::default().fg(Color::Rgb(255, 100, 100))),
            Span::styled(format!("{}", fmt_count(err_dns)), dim),
        ]));
        lines.push(Line::from(vec![
            Span::styled("   conn", Style::default().fg(Color::Rgb(255, 140, 100))),
            Span::styled(format!("{:>6}", fmt_count(err_conn)), dim),
            Span::styled(" tls ", Style::default().fg(Color::Rgb(255, 180, 100))),
            Span::styled(format!("{}", fmt_count(err_tls)), dim),
            if err_other > 0 {
                Span::styled(format!(" ?{}", fmt_count(err_other)), Style::default().fg(Color::Rgb(120, 120, 120)))
            } else {
                Span::raw("")
            },
        ]));
    } else {
        lines.push(Line::from(Span::raw("")));
        lines.push(Line::from(Span::raw("")));
    }

    lines.push(counter_line(" Timeout", Color::Yellow, timeout, total));
    lines.push(counter_line(" Skipped", Color::DarkGray, skipped, total));
    lines.push(Line::from(vec![
        Span::styled(" Total  ", Style::default().fg(Color::White)),
        Span::styled(format!("{:>9}", fmt_count(total)), bold.fg(Color::White)),
    ]));

    // Pad remaining lines if area is taller
    while lines.len() < area.height as usize {
        lines.push(Line::from(Span::raw("")));
    }

    frame.render_widget(Paragraph::new(lines), area);
}

#[allow(clippy::too_many_arguments)]
fn render_throughput(
    frame: &mut ratatui::Frame,
    area: Rect,
    avg_rps: f64, peak_rps: u64, elapsed: Duration, eta: Option<Duration>,
    s2xx: u64, s3xx: u64, s4xx: u64, s5xx: u64,
    dom_total: u64, dom_abandoned: u64,
    mem_rss: u64, net_rx: u64, net_tx: u64, open_fds: u64,
    state: &RenderState,
) {
    let dim = Style::default().fg(Color::DarkGray);
    let accent = Style::default().fg(Color::Cyan).add_modifier(Modifier::BOLD);
    let green = Style::default().fg(Color::Green).add_modifier(Modifier::BOLD);

    // Split: sparkline on top, metrics below.
    let [spark_area, metrics_area] = Layout::vertical([
        Constraint::Min(3),
        Constraint::Length(6),
    ]).areas(area);

    // Sparkline.
    let spark_data: Vec<u64> = state.rps_history.iter().copied().collect();
    if !spark_data.is_empty() {
        let sparkline = Sparkline::default()
            .data(&spark_data)
            .bar_set(symbols::bar::NINE_LEVELS)
            .style(Style::default().fg(Color::Green));
        frame.render_widget(sparkline, spark_area);
    } else {
        frame.render_widget(
            Paragraph::new(Span::styled(" Waiting for data...", dim)),
            spark_area,
        );
    }

    // Metrics lines.
    let eta_str = eta.map(fmt_elapsed).unwrap_or("--".into());

    let mut metrics: Vec<Line> = vec![
        Span::styled(" Throughput", dim).into(),
        Line::from(vec![
            Span::styled(" Avg ", dim),
            Span::styled(format!("{:>7.0} /s", avg_rps), accent),
            Span::styled("  Peak ", dim),
            Span::styled(format!("{:>7} /s", fmt_count(peak_rps)), green),
        ]),
        Line::from(vec![
            Span::styled(" Elapsed ", dim),
            Span::styled(format!("{:>6}", fmt_elapsed(elapsed)), Style::default().fg(Color::White)),
            Span::styled("  ETA ", dim),
            Span::styled(format!("{:>6}", eta_str), Style::default().fg(Color::Yellow)),
        ]),
    ];

    // Domains line
    let dom_str = if dom_total > 0 {
        if dom_abandoned > 0 {
            format!(" Domains {}/{} ({} abn)", fmt_count(dom_total - dom_abandoned), fmt_count(dom_total), dom_abandoned)
        } else {
            format!(" Domains {}", fmt_count(dom_total))
        }
    } else {
        " Domains --".to_string()
    };
    metrics.push(Line::from(Span::styled(dom_str, dim)));

    // HTTP status line
    let http_total = s2xx + s3xx + s4xx + s5xx;
    if http_total > 0 {
        metrics.push(Line::from(vec![
            Span::styled(" HTTP ", dim),
            Span::styled(format!("2xx {}", fmt_short(s2xx)), Style::default().fg(Color::Green)),
            Span::styled(format!(" 3xx {}", fmt_short(s3xx)), Style::default().fg(Color::Blue)),
            Span::styled(format!(" 4xx {}", fmt_short(s4xx)), Style::default().fg(Color::Yellow)),
            Span::styled(format!(" 5xx {}", fmt_short(s5xx)), Style::default().fg(Color::Red)),
        ]));
    } else {
        metrics.push(Line::from(Span::styled(" HTTP --", dim)));
    }

    // System resources line
    let net_str = fmt_bytes_rate(net_rx + net_tx);
    let sys_line = if mem_rss > 0 || open_fds > 0 {
        let mut parts: Vec<Span> = vec![Span::styled(" Sys ", dim)];
        if mem_rss > 0 {
            parts.push(Span::styled(format!("{}MB", mem_rss), Style::default().fg(Color::Magenta)));
            parts.push(Span::styled("  ", dim));
        }
        if open_fds > 0 {
            parts.push(Span::styled(format!("fd {}", fmt_count(open_fds)), Style::default().fg(Color::Cyan)));
            parts.push(Span::styled("  ", dim));
        }
        if net_rx + net_tx > 0 {
            parts.push(Span::styled(format!("net {}", net_str), Style::default().fg(Color::Blue)));
        }
        Line::from(parts)
    } else {
        Line::from(Span::styled(" Sys --", dim))
    };
    metrics.push(sys_line);

    frame.render_widget(Paragraph::new(metrics), metrics_area);
}

// ---------------------------------------------------------------------------
// Progress gauge (single line)
// ---------------------------------------------------------------------------

fn render_progress(
    frame: &mut ratatui::Frame,
    area: Rect,
    ratio: f64,
    total: u64,
    total_seeds: u64,
    eta: Option<Duration>,
) {
    if total_seeds == 0 {
        let label = if total == 0 {
            " Waiting...".to_string()
        } else {
            format!(" {} fetched", fmt_count(total))
        };
        frame.render_widget(
            Paragraph::new(Span::styled(label, Style::default().fg(Color::DarkGray))),
            area,
        );
        return;
    }

    let eta_part = eta.map(|d| format!("  ETA {}", fmt_elapsed(d))).unwrap_or_default();
    let label = format!(
        " {}%  {} / {}{}",
        format!("{:.1}", ratio * 100.0),
        fmt_count(total),
        fmt_count(total_seeds),
        eta_part,
    );

    let gauge = LineGauge::default()
        .ratio(ratio)
        .filled_style(Style::default().fg(Color::Cyan))
        .unfilled_style(Style::default().fg(Color::DarkGray))
        .label(Span::styled(label, Style::default().fg(Color::White)));
    frame.render_widget(gauge, area);
}

// ---------------------------------------------------------------------------
// Log panel
// ---------------------------------------------------------------------------

fn render_log(frame: &mut ratatui::Frame, area: Rect, stats: &Stats) {
    let max_items = area.height.saturating_sub(1) as usize; // -1 for title border
    let warnings: Vec<String> = if let Ok(w) = stats.warnings.lock() {
        // Show newest at bottom.
        let skip = w.len().saturating_sub(max_items);
        w.iter().skip(skip).cloned().collect()
    } else {
        vec![]
    };

    let items: Vec<ListItem> = if warnings.is_empty() {
        vec![ListItem::new(Span::styled(
            " (no events)",
            Style::default().fg(Color::DarkGray),
        ))]
    } else {
        warnings
            .iter()
            .map(|s| {
                // Color-code by event type
                let (prefix_style, text) = if s.starts_with("dns:") {
                    (Style::default().fg(Color::Red), s.as_str())
                } else if s.starts_with("conn:") {
                    (Style::default().fg(Color::Rgb(255, 140, 100)), s.as_str())
                } else if s.starts_with("tls:") {
                    (Style::default().fg(Color::Yellow), s.as_str())
                } else if s.starts_with("abandoned") {
                    (Style::default().fg(Color::Rgb(255, 100, 100)), s.as_str())
                } else if s.starts_with("engine:") || s.starts_with("done:") {
                    (Style::default().fg(Color::Cyan), s.as_str())
                } else if s.starts_with("pass") {
                    (Style::default().fg(Color::Magenta), s.as_str())
                } else {
                    (Style::default().fg(Color::White), s.as_str())
                };
                ListItem::new(Line::from(vec![
                    Span::styled(" > ", Style::default().fg(Color::DarkGray)),
                    Span::styled(text.to_string(), prefix_style),
                ]))
            })
            .collect()
    };

    let title = Span::styled(" Log ", Style::default().fg(Color::DarkGray));
    let list = List::new(items).block(
        Block::default()
            .borders(Borders::TOP)
            .border_style(Style::default().fg(Color::DarkGray))
            .title(title),
    );
    frame.render_widget(list, area);
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

fn counter_line(label: &str, color: Color, value: u64, total: u64) -> Line<'static> {
    Line::from(vec![
        Span::styled(label.to_string(), Style::default().fg(color)),
        Span::styled(
            format!("{:>9}", fmt_count(value)),
            Style::default().fg(Color::White).add_modifier(Modifier::BOLD),
        ),
        Span::styled(
            format!(" {:>5.1}%", pct(value, total)),
            Style::default().fg(Color::DarkGray),
        ),
    ])
}

fn pct(n: u64, d: u64) -> f64 {
    if d == 0 { 0.0 } else { n as f64 * 100.0 / d as f64 }
}

fn fmt_count(n: u64) -> String {
    let s = n.to_string();
    let mut out = String::with_capacity(s.len() + s.len() / 3);
    for (i, c) in s.chars().rev().enumerate() {
        if i > 0 && i % 3 == 0 {
            out.push(',');
        }
        out.push(c);
    }
    out.chars().rev().collect()
}

/// Short number format: 1,234 → "1.2k", 1,234,567 → "1.2M"
fn fmt_short(n: u64) -> String {
    if n >= 1_000_000 {
        format!("{:.1}M", n as f64 / 1_000_000.0)
    } else if n >= 10_000 {
        format!("{:.0}k", n as f64 / 1_000.0)
    } else if n >= 1_000 {
        format!("{:.1}k", n as f64 / 1_000.0)
    } else {
        n.to_string()
    }
}

fn fmt_elapsed(d: Duration) -> String {
    let s = d.as_secs();
    let h = s / 3600;
    let m = (s % 3600) / 60;
    let s = s % 60;
    if h > 0 {
        format!("{h}h{m:02}m{s:02}s")
    } else if m > 0 {
        format!("{m}m{s:02}s")
    } else {
        format!("{s}s")
    }
}

fn fmt_bytes(b: u64) -> String {
    if b >= 1_000_000_000 {
        format!("{:.1} GB", b as f64 / 1_000_000_000.0)
    } else if b >= 1_000_000 {
        format!("{:.1} MB", b as f64 / 1_000_000.0)
    } else if b >= 1_000 {
        format!("{:.1} KB", b as f64 / 1_000.0)
    } else {
        format!("{b} B")
    }
}

fn fmt_bytes_rate(bps: u64) -> String {
    if bps >= 1_000_000_000 {
        format!("{:.1}GB/s", bps as f64 / 1_000_000_000.0)
    } else if bps >= 1_000_000 {
        format!("{:.0}MB/s", bps as f64 / 1_000_000.0)
    } else if bps >= 1_000 {
        format!("{:.0}KB/s", bps as f64 / 1_000.0)
    } else if bps > 0 {
        format!("{bps}B/s")
    } else {
        "0".to_string()
    }
}
