use clap::{Parser, Subcommand};
use tracing_subscriber::EnvFilter;

mod cc;
mod display;
mod hn;
mod tui;

#[derive(Parser)]
#[command(
    name = "crawler",
    about = "High-throughput multi-domain recrawler",
    version
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
    /// Recrawl Common Crawl URLs (stub — coming soon)
    Recrawl(Box<cc::RecrawlArgs>),
}

#[tokio::main]
async fn main() -> anyhow::Result<()> {
    // Always write tracing output to stderr so the TUI (stdout) is unaffected.
    tracing_subscriber::fmt()
        .with_env_filter(
            EnvFilter::from_default_env()
                .add_directive("crawler_cli=info".parse().unwrap())
                .add_directive("crawler_lib=info".parse().unwrap()),
        )
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
