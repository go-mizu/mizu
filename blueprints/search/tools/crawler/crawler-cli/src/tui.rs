//! Real-time crawler dashboard using ratatui.
//!
//! Layout:
//!   ┌ Title ─────────────────────────────────────────────────────┐
//!   │ ┌ Requests ──────────────────┐ ┌ RPS ─────────────────────┐│
//!   │ │  ✓  OK   92,431   46.2%   │ │ ▁▂▄▇█▆▃▁▂▄▆▇█▇▅▃▁▂▄▆▇█  ││
//!   │ │  ✗  Failed  8,124   4.1%  │ │  Avg   3,486 /s           ││
//!   │ │  ⏱  Timeout 56,781  28.4% │ │  Peak 10,444 /s           ││
//!   │ │  ⊘  Skipped 42,664  21.3% │ │  Elapsed   26s  ETA   38s ││
//!   │ │  ─  Total  200,000        │ │  Downloaded  12.4 MB      ││
//!   │ └────────────────────────────┘ └───────────────────────────┘│
//!   │ ████████████████████░░░░░░░░░  46.2%  92,431 / 200,000      │
//!   │ ┌ Warnings ─────────────────────────────────────────────────┐│
//!   │ │  › abandoned: slow.example.com (5 timeouts)               ││
//!   └─────────────────────────────────────────────────────────────┘

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
    layout::{Constraint, Direction, Layout, Rect},
    style::{Color, Modifier, Style},
    text::{Line, Span},
    widgets::{Block, Borders, Gauge, List, ListItem, Paragraph, Sparkline},
    Terminal,
};

use crawler_lib::stats::Stats;

/// Number of RPS samples kept for the sparkline (sampled every ~80ms → ~8s of history).
const SPARKLINE_LEN: usize = 100;

// ---------------------------------------------------------------------------
// Public API
// ---------------------------------------------------------------------------

/// Handle to a running TUI thread.
pub struct TuiHandle {
    stop: Arc<AtomicBool>,
    thread: Option<std::thread::JoinHandle<()>>,
}

impl TuiHandle {
    /// Signal the TUI to do one final render, then exit and restore the terminal.
    pub fn stop_and_join(mut self) {
        self.stop.store(true, Ordering::Relaxed);
        if let Some(h) = self.thread.take() {
            let _ = h.join();
        }
    }
}

/// Spawn the TUI dashboard thread.
///
/// Returns `None` when stdout is not a terminal (CI, piped, non-interactive SSH).
pub fn spawn(stats: Arc<Stats>, title: String) -> Option<TuiHandle> {
    if !io::stdout().is_terminal() {
        return None;
    }

    let stop = Arc::new(AtomicBool::new(false));
    let stop2 = stop.clone();

    let thread = std::thread::spawn(move || {
        if let Err(e) = run_dashboard(stats, stop2, title) {
            eprintln!("[tui] failed: {e}");
        }
    });

    Some(TuiHandle { stop, thread: Some(thread) })
}

// ---------------------------------------------------------------------------
// Render state (lives on the TUI thread, not in Stats)
// ---------------------------------------------------------------------------

struct RenderState {
    /// Ring buffer of instantaneous RPS samples (one per poll interval).
    rps_history: VecDeque<u64>,
    /// Total processed count from the previous sample.
    prev_total: u64,
    /// Timestamp of the previous sample.
    prev_ts: Instant,
}

impl RenderState {
    fn new() -> Self {
        Self {
            rps_history: VecDeque::with_capacity(SPARKLINE_LEN + 1),
            prev_total: 0,
            prev_ts: Instant::now(),
        }
    }

    /// Record a new sample using the current total. Call once per render loop tick.
    fn tick(&mut self, total: u64) {
        let now = Instant::now();
        let dt = now.duration_since(self.prev_ts).as_secs_f64();
        // Only sample when enough time has elapsed (avoids division by tiny dt).
        if dt >= 0.05 {
            let delta = total.saturating_sub(self.prev_total);
            let rps = (delta as f64 / dt).round() as u64;
            if self.rps_history.len() >= SPARKLINE_LEN {
                self.rps_history.pop_front();
            }
            self.rps_history.push_back(rps);
            self.prev_total = total;
            self.prev_ts = now;
        }
    }
}

// ---------------------------------------------------------------------------
// Render loop
// ---------------------------------------------------------------------------

fn run_dashboard(
    stats: Arc<Stats>,
    stop: Arc<AtomicBool>,
    title: String,
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
        // Sample before rendering so the sparkline has up-to-date data.
        let total = stats.total.load(Ordering::Relaxed);
        state.tick(total);

        terminal.draw(|f| render(f, &stats, &title, &state))?;

        // Poll for input with a short timeout (80ms) for responsive updates.
        if event::poll(Duration::from_millis(80))? {
            if let Event::Key(key) = event::read()? {
                match key.code {
                    KeyCode::Char('q') | KeyCode::Esc => break,
                    _ => {}
                }
            }
        }

        if stop.load(Ordering::Relaxed) {
            // Final render with completed stats.
            let total = stats.total.load(Ordering::Relaxed);
            state.tick(total);
            terminal.draw(|f| render(f, &stats, &title, &state))?;
            break;
        }
    }

    disable_raw_mode()?;
    execute!(terminal.backend_mut(), LeaveAlternateScreen)?;
    terminal.show_cursor()?;
    Ok(())
}

// ---------------------------------------------------------------------------
// Top-level render
// ---------------------------------------------------------------------------

fn render(frame: &mut ratatui::Frame, stats: &Stats, title: &str, state: &RenderState) {
    let area = frame.area();

    // Read all stats atomically.
    let ok = stats.ok.load(Ordering::Relaxed);
    let failed = stats.failed.load(Ordering::Relaxed);
    let timeout = stats.timeout.load(Ordering::Relaxed);
    let skipped = stats.skipped.load(Ordering::Relaxed);
    let total = stats.total.load(Ordering::Relaxed);
    let total_seeds = stats.total_seeds.load(Ordering::Relaxed);
    let peak_rps = stats.peak_rps.load(Ordering::Relaxed);
    let bytes = stats.bytes_downloaded.load(Ordering::Relaxed);
    let elapsed = stats.start.elapsed();

    let avg_rps: f64 = if elapsed.as_secs_f64() > 0.1 {
        total as f64 / elapsed.as_secs_f64()
    } else {
        0.0
    };

    let ratio: f64 = if total_seeds > 0 {
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

    // Outer layout: header(3) | main(9) | progress(3) | warnings(rest)
    let outer = Layout::default()
        .direction(Direction::Vertical)
        .constraints([
            Constraint::Length(3),  // title header
            Constraint::Length(9),  // counters + sparkline
            Constraint::Length(3),  // progress gauge
            Constraint::Min(3),     // warnings log
        ])
        .split(area);

    render_header(frame, outer[0], title, total_seeds);
    render_main(
        frame, outer[1],
        ok, failed, timeout, skipped, total,
        avg_rps, peak_rps, elapsed, eta, bytes, state,
    );
    render_progress(frame, outer[2], ratio, total, total_seeds, eta);
    render_warnings(frame, outer[3], stats);
}

// ---------------------------------------------------------------------------
// Header
// ---------------------------------------------------------------------------

fn render_header(frame: &mut ratatui::Frame, area: Rect, title: &str, total_seeds: u64) {
    let seeds_part = if total_seeds > 0 {
        format!("  ·  {} seeds", fmt_count(total_seeds))
    } else {
        String::new()
    };
    let header = Paragraph::new(Line::from(vec![
        Span::styled(
            format!("  {}{}", title, seeds_part),
            Style::default().fg(Color::Cyan).add_modifier(Modifier::BOLD),
        ),
    ]))
    .block(
        Block::default()
            .borders(Borders::ALL)
            .border_style(Style::default().fg(Color::Cyan)),
    );
    frame.render_widget(header, area);
}

// ---------------------------------------------------------------------------
// Main (counters left + RPS sparkline right)
// ---------------------------------------------------------------------------

fn render_main(
    frame: &mut ratatui::Frame,
    area: Rect,
    ok: u64,
    failed: u64,
    timeout: u64,
    skipped: u64,
    total: u64,
    avg_rps: f64,
    peak_rps: u64,
    elapsed: Duration,
    eta: Option<Duration>,
    bytes: u64,
    state: &RenderState,
) {
    // Two columns: counters (fixed 38 chars) | RPS panel (rest).
    let cols = Layout::default()
        .direction(Direction::Horizontal)
        .constraints([Constraint::Length(38), Constraint::Min(22)])
        .split(area);

    render_counters(frame, cols[0], ok, failed, timeout, skipped, total);
    render_rps_panel(frame, cols[1], avg_rps, peak_rps, elapsed, eta, bytes, state);
}

fn render_counters(
    frame: &mut ratatui::Frame,
    area: Rect,
    ok: u64,
    failed: u64,
    timeout: u64,
    skipped: u64,
    total: u64,
) {
    let block = Block::default()
        .borders(Borders::ALL)
        .border_style(Style::default().fg(Color::DarkGray))
        .title(Span::styled(
            " Requests ",
            Style::default().fg(Color::White).add_modifier(Modifier::DIM),
        ));
    let inner = block.inner(area);
    frame.render_widget(block, area);

    let lines = vec![
        counter_line("  ✓  OK      ", Color::Green, ok, total),
        counter_line("  ✗  Failed  ", Color::Red, failed, total),
        counter_line("  ⏱  Timeout ", Color::Yellow, timeout, total),
        counter_line("  ⊘  Skipped ", Color::DarkGray, skipped, total),
        Line::from(vec![
            Span::raw("  ─  Total   "),
            Span::styled(
                format!("{:>9}", fmt_count(total)),
                Style::default().fg(Color::White).add_modifier(Modifier::BOLD),
            ),
        ]),
    ];
    frame.render_widget(Paragraph::new(lines), inner);
}

fn render_rps_panel(
    frame: &mut ratatui::Frame,
    area: Rect,
    avg_rps: f64,
    peak_rps: u64,
    elapsed: Duration,
    eta: Option<Duration>,
    bytes: u64,
    state: &RenderState,
) {
    let block = Block::default()
        .borders(Borders::ALL)
        .border_style(Style::default().fg(Color::DarkGray))
        .title(Span::styled(
            " RPS ",
            Style::default().fg(Color::Cyan).add_modifier(Modifier::BOLD),
        ));
    let inner = block.inner(area);
    frame.render_widget(block, area);

    // Split inner: top rows for sparkline, bottom rows for metrics.
    // inner height is 7 (9 total - 2 borders). Sparkline=3, metrics=4.
    let sparkline_height = (inner.height / 2).max(2).min(4);
    let rows = Layout::default()
        .direction(Direction::Vertical)
        .constraints([
            Constraint::Length(sparkline_height),
            Constraint::Min(2),
        ])
        .split(inner);

    // Sparkline.
    let spark_data: Vec<u64> = state.rps_history.iter().cloned().collect();
    let sparkline = Sparkline::default()
        .data(&spark_data)
        .style(Style::default().fg(Color::Cyan));
    frame.render_widget(sparkline, rows[0]);

    // Metrics.
    let eta_str = eta.map(fmt_elapsed).unwrap_or_else(|| "—".into());
    let dim = Style::default().fg(Color::DarkGray);
    let val = Style::default().fg(Color::White).add_modifier(Modifier::BOLD);
    let accent = Style::default().fg(Color::Cyan).add_modifier(Modifier::BOLD);
    let green = Style::default().fg(Color::Green).add_modifier(Modifier::BOLD);

    let metrics_lines = vec![
        Line::from(vec![
            Span::styled("  Avg  ", dim),
            Span::styled(format!("{avg_rps:>6.0} /s"), accent),
            Span::styled("  Peak  ", dim),
            Span::styled(format!("{peak_rps:>6} /s"), green),
        ]),
        Line::from(vec![
            Span::styled("  Elapsed  ", dim),
            Span::styled(format!("{:>8}", fmt_elapsed(elapsed)), val),
            Span::styled("  ETA  ", dim),
            Span::styled(format!("{:>8}", eta_str), Style::default().fg(Color::Yellow)),
        ]),
        Line::from(vec![
            Span::styled("  Downloaded  ", dim),
            Span::styled(fmt_bytes(bytes), val),
        ]),
    ];
    frame.render_widget(Paragraph::new(metrics_lines), rows[1]);
}

// ---------------------------------------------------------------------------
// Progress gauge
// ---------------------------------------------------------------------------

fn render_progress(
    frame: &mut ratatui::Frame,
    area: Rect,
    ratio: f64,
    total: u64,
    total_seeds: u64,
    eta: Option<Duration>,
) {
    let label = if total == 0 {
        " Initializing... ".to_string()
    } else if total_seeds > 0 {
        let eta_part = eta
            .map(|d| format!("  ETA {}", fmt_elapsed(d)))
            .unwrap_or_default();
        format!(
            " {} / {}  ({:.1}%){} ",
            fmt_count(total),
            fmt_count(total_seeds),
            ratio * 100.0,
            eta_part,
        )
    } else {
        format!(" {} fetched ", fmt_count(total))
    };

    let gauge = Gauge::default()
        .block(
            Block::default()
                .borders(Borders::ALL)
                .border_style(Style::default().fg(Color::DarkGray)),
        )
        .gauge_style(Style::default().fg(Color::Cyan).bg(Color::Black))
        .ratio(ratio)
        .label(label);
    frame.render_widget(gauge, area);
}

// ---------------------------------------------------------------------------
// Warnings log
// ---------------------------------------------------------------------------

fn render_warnings(frame: &mut ratatui::Frame, area: Rect, stats: &Stats) {
    let max_items = area.height.saturating_sub(2) as usize;
    let strings: Vec<String> = if let Ok(w) = stats.warnings.lock() {
        w.iter().rev().take(max_items).cloned().collect()
    } else {
        vec![]
    };

    let items: Vec<ListItem> = if strings.is_empty() {
        vec![ListItem::new(Line::from(vec![Span::styled(
            "  (no warnings)",
            Style::default().fg(Color::DarkGray),
        )]))]
    } else {
        strings
            .iter()
            .map(|s| {
                ListItem::new(Line::from(vec![
                    Span::styled("  › ", Style::default().fg(Color::DarkGray)),
                    Span::styled(s.as_str(), Style::default().fg(Color::Yellow)),
                ]))
            })
            .collect()
    };

    let list = List::new(items).block(
        Block::default()
            .borders(Borders::ALL)
            .border_style(Style::default().fg(Color::DarkGray))
            .title(Span::styled(
                " Warnings  (q to quit) ",
                Style::default()
                    .fg(Color::White)
                    .add_modifier(Modifier::DIM),
            )),
    );
    frame.render_widget(list, area);
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

fn counter_line(label: &str, color: Color, value: u64, total: u64) -> Line<'_> {
    Line::from(vec![
        Span::styled(label, Style::default().fg(color)),
        Span::styled(
            format!("{:>9}", fmt_count(value)),
            Style::default().fg(Color::White).add_modifier(Modifier::BOLD),
        ),
        Span::styled(
            format!("  {:>5.1}%", pct(value, total)),
            Style::default().fg(Color::DarkGray),
        ),
    ])
}

fn pct(n: u64, d: u64) -> f64 {
    if d == 0 { 0.0 } else { n as f64 * 100.0 / d as f64 }
}

fn fmt_count(n: u64) -> String {
    // Manual thousands separator for readability.
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
