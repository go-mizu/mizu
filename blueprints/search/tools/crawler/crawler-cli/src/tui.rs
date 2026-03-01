//! Real-time crawler dashboard using ratatui.
//!
//! Spawns a background std::thread that renders the TUI in the alternate screen.
//! The main tokio runtime and the TUI thread share `Arc<Stats>` for lock-free reads.
//! Warnings (domain timeouts, abandonments) are read from `stats.warnings` ring buffer.
//!
//! Only activates when stdout is a terminal (IsTerminal check). When spawned,
//! tracing logs should be redirected to stderr so they don't corrupt the display.

use std::io::{self, IsTerminal};
use std::sync::atomic::{AtomicBool, Ordering};
use std::sync::Arc;
use std::time::Duration;

use crossterm::{
    event::{self, Event, KeyCode},
    execute,
    terminal::{disable_raw_mode, enable_raw_mode, EnterAlternateScreen, LeaveAlternateScreen},
};
use ratatui::{
    backend::CrosstermBackend,
    layout::{Constraint, Direction, Layout},
    style::{Color, Modifier, Style},
    text::{Line, Span},
    widgets::{Block, Borders, Gauge, List, ListItem, Paragraph},
    Terminal,
};

use crawler_lib::stats::Stats;

// ---------------------------------------------------------------------------
// Public API
// ---------------------------------------------------------------------------

/// Handle to a running TUI thread. Call `stop_and_join` after the crawl ends.
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
/// When `Some`, the terminal is taken into alternate-screen mode for the lifetime
/// of the TuiHandle.
pub fn spawn(stats: Arc<Stats>, title: String) -> Option<TuiHandle> {
    if !io::stdout().is_terminal() {
        return None;
    }

    let stop = Arc::new(AtomicBool::new(false));
    let stop2 = stop.clone();

    let thread = std::thread::spawn(move || {
        if let Err(e) = run_dashboard(stats, stop2, title) {
            // If TUI init fails, silently fall back (plain println still works)
            eprintln!("[tui] failed: {e}");
        }
    });

    Some(TuiHandle { stop, thread: Some(thread) })
}

// ---------------------------------------------------------------------------
// Internal render loop
// ---------------------------------------------------------------------------

fn run_dashboard(
    stats: Arc<Stats>,
    stop: Arc<AtomicBool>,
    title: String,
) -> anyhow::Result<()> {
    // Install a panic hook so the terminal is always restored on panic.
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

    loop {
        terminal.draw(|f| render(f, &stats, &title))?;

        // Poll for key events (200 ms window so the stop-flag check is responsive).
        if event::poll(Duration::from_millis(200))? {
            if let Event::Key(key) = event::read()? {
                match key.code {
                    KeyCode::Char('q') | KeyCode::Esc => break,
                    _ => {}
                }
            }
        }

        if stop.load(Ordering::Relaxed) {
            // One final render with completed stats before exiting.
            terminal.draw(|f| render(f, &stats, &title))?;
            break;
        }
    }

    // Restore terminal
    disable_raw_mode()?;
    execute!(terminal.backend_mut(), LeaveAlternateScreen)?;
    terminal.show_cursor()?;
    Ok(())
}

// ---------------------------------------------------------------------------
// Rendering
// ---------------------------------------------------------------------------

fn render(frame: &mut ratatui::Frame, stats: &Stats, title: &str) {
    let area = frame.area();

    // Vertical layout: progress (3) | stats (7) | warnings (rest)
    let chunks = Layout::default()
        .direction(Direction::Vertical)
        .constraints([
            Constraint::Length(3),
            Constraint::Length(7),
            Constraint::Min(3),
        ])
        .split(area);

    // Read stats atomically (no locking — all AtomicU64)
    let ok = stats.ok.load(Ordering::Relaxed);
    let failed = stats.failed.load(Ordering::Relaxed);
    let timeout = stats.timeout.load(Ordering::Relaxed);
    let skipped = stats.skipped.load(Ordering::Relaxed);
    let total = stats.total.load(Ordering::Relaxed);
    let total_seeds = stats.total_seeds.load(Ordering::Relaxed);
    let peak_rps = stats.peak_rps.load(Ordering::Relaxed);
    let elapsed = stats.start.elapsed();
    let avg_rps = if elapsed.as_secs_f64() > 0.01 {
        total as f64 / elapsed.as_secs_f64()
    } else {
        0.0
    };

    // --- Progress gauge ---
    let ratio = if total_seeds > 0 {
        (total as f64 / total_seeds as f64).clamp(0.0, 1.0)
    } else {
        0.0
    };
    let label = if total_seeds > 0 {
        format!("{total} / {total_seeds}  ({:.1}%)", ratio * 100.0)
    } else {
        format!("{total} fetched")
    };
    let gauge = Gauge::default()
        .block(
            Block::default()
                .title(format!(" {title} "))
                .borders(Borders::ALL),
        )
        .gauge_style(Style::default().fg(Color::Cyan))
        .ratio(ratio)
        .label(label);
    frame.render_widget(gauge, chunks[0]);

    // --- Stats panel ---
    let bold_cyan = Style::default()
        .fg(Color::Cyan)
        .add_modifier(Modifier::BOLD);
    let stats_lines = vec![
        stats_row(
            "  ✓  OK      ",
            Color::Green,
            ok,
            total,
            "  Avg RPS ",
            format!("{avg_rps:>7.0}"),
            bold_cyan,
        ),
        stats_row(
            "  ⏱  Timeout ",
            Color::Yellow,
            timeout,
            total,
            "  Peak RPS",
            format!("{peak_rps:>7}"),
            bold_cyan,
        ),
        stats_row(
            "  ✗  Failed  ",
            Color::Red,
            failed,
            total,
            "  Elapsed  ",
            format!("{:>7}", fmt_elapsed(elapsed)),
            Style::default().fg(Color::White),
        ),
        stats_row(
            "  ⊘  Skipped ",
            Color::DarkGray,
            skipped,
            total,
            "",
            String::new(),
            Style::default(),
        ),
        Line::from(vec![
            Span::raw("  "),
            Span::styled(
                format!("Total: {total}"),
                Style::default().fg(Color::White).add_modifier(Modifier::DIM),
            ),
        ]),
    ];
    let stats_para = Paragraph::new(stats_lines)
        .block(Block::default().title(" Stats ").borders(Borders::ALL));
    frame.render_widget(stats_para, chunks[1]);

    // --- Warnings list ---
    let max_items = chunks[2].height.saturating_sub(2) as usize;
    // Collect strings first (before MutexGuard drops) then build ListItems from owned data.
    let warning_strings: Vec<String> = if let Ok(w) = stats.warnings.lock() {
        w.iter().rev().take(max_items).cloned().collect()
    } else {
        vec![]
    };
    let items: Vec<ListItem> = warning_strings
        .iter()
        .map(|s| {
            ListItem::new(Line::from(vec![
                Span::styled("  › ", Style::default().fg(Color::DarkGray)),
                Span::styled(s.as_str(), Style::default().fg(Color::Yellow)),
            ]))
        })
        .collect();
    let warn_list = List::new(items)
        .block(Block::default().title(" Warnings (q to quit) ").borders(Borders::ALL));
    frame.render_widget(warn_list, chunks[2]);
}

fn stats_row<'a>(
    label: &'a str,
    label_color: Color,
    value: u64,
    total: u64,
    right_label: &'a str,
    right_value: String,
    right_style: Style,
) -> Line<'a> {
    let mut spans = vec![
        Span::styled(label, Style::default().fg(label_color)),
        Span::raw(format!("{value:>9}  ({:.1}%)", pct(value, total))),
    ];
    if !right_label.is_empty() {
        spans.push(Span::styled(right_label, Style::default().fg(Color::White).add_modifier(Modifier::DIM)));
        spans.push(Span::styled(right_value, right_style));
    }
    Line::from(spans)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

fn pct(n: u64, d: u64) -> f64 {
    if d == 0 { 0.0 } else { n as f64 * 100.0 / d as f64 }
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
