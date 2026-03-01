use clap::{Parser, Subcommand};
use tracing_subscriber::EnvFilter;

mod display;
mod hn;

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
}

#[derive(Subcommand)]
enum HnAction {
    /// Recrawl HN seed URLs
    Recrawl(Box<hn::RecrawlArgs>),
}

#[tokio::main]
async fn main() -> anyhow::Result<()> {
    tracing_subscriber::fmt()
        .with_env_filter(
            EnvFilter::from_default_env()
                .add_directive("crawler_cli=info".parse().unwrap())
                .add_directive("crawler_lib=info".parse().unwrap()),
        )
        .with_target(false)
        .init();

    let cli = Cli::parse();

    match cli.command {
        Commands::Hn { action } => match action {
            HnAction::Recrawl(args) => {
                hn::run_recrawl(*args).await?;
            }
        },
    }

    Ok(())
}
