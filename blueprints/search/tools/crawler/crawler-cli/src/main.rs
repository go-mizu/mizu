use clap::{Parser, Subcommand};
use std::io::IsTerminal;
use tracing_subscriber::EnvFilter;

mod cc;
mod common;
mod display;
mod gui;
mod hn;
mod tui;

/// Long version shown by `crawler --version` (verbose form).
/// Built from env vars set by build.rs at compile time.
const LONG_VERSION: &str = concat!(
    env!("CRAWLER_GIT_VERSION"),
    "\ncommit: ", env!("CRAWLER_GIT_COMMIT"),
    "\nbuilt:  ", env!("CRAWLER_BUILD_TIME"),
);

#[derive(Parser)]
#[command(
    name = "crawler",
    about = "High-throughput multi-domain recrawler",
    version = env!("CRAWLER_GIT_VERSION"),
    long_version = LONG_VERSION,
)]
struct Cli {
    #[command(subcommand)]
    command: Commands,
}

#[derive(Subcommand)]
enum Commands {
    /// HN recrawl commands
    Hn {
        #[command(subcommand)]
        action: HnAction,
    },
    /// Common Crawl recrawl commands
    Cc {
        #[command(subcommand)]
        action: CcAction,
    },
}

#[derive(Subcommand)]
enum HnAction {
    /// Recrawl HN seed URLs
    Recrawl(Box<hn::RecrawlArgs>),
}

#[derive(Subcommand)]
enum CcAction {
    /// Recrawl Common Crawl URLs from CC index parquet
    Recrawl(Box<cc::RecrawlArgs>),
}

#[tokio::main]
async fn main() -> anyhow::Result<()> {
    // When stdout is a TTY, the TUI uses alternate screen — tracing to stderr
    // would corrupt the display. Suppress INFO logs; WARN+ still visible.
    // When piped/non-TTY, full INFO logging to stderr.
    let filter = if std::io::stdout().is_terminal() {
        EnvFilter::from_default_env()
            .add_directive("crawler_cli=warn".parse().unwrap())
            .add_directive("crawler_lib=warn".parse().unwrap())
    } else {
        EnvFilter::from_default_env()
            .add_directive("crawler_cli=info".parse().unwrap())
            .add_directive("crawler_lib=info".parse().unwrap())
    };

    tracing_subscriber::fmt()
        .with_env_filter(filter)
        .with_target(false)
        .with_writer(std::io::stderr)
        .init();

    let cli = Cli::parse();

    match cli.command {
        Commands::Hn { action } => match action {
            HnAction::Recrawl(args) => {
                hn::run_recrawl(*args).await?;
            }
        },
        Commands::Cc { action } => match action {
            CcAction::Recrawl(args) => {
                cc::run_recrawl(*args).await?;
            }
        },
    }

    Ok(())
}
