use anyhow::{Context, Result};
use clap::Parser;
use fastembed::{EmbeddingModel, InitOptions, TextEmbedding};
use serde::{Deserialize, Serialize};
use std::io::{BufRead, BufReader};
use std::time::Instant;

#[derive(Parser)]
#[command(name = "embedder-fastembed")]
enum Cli {
    /// Run embedding benchmark
    Bench {
        /// Input JSONL file (one {"text":"..."} per line)
        #[arg(long)]
        input: String,

        /// Batch size
        #[arg(long, default_value = "64")]
        batch_size: usize,
    },
}

#[derive(Deserialize)]
struct InputLine {
    text: String,
}

#[derive(Serialize)]
struct BenchResult {
    approach: String,
    backend: String,
    model: String,
    batch_size: usize,
    total_vecs: usize,
    warmup_vecs: usize,
    vecs_per_sec: f64,
    p50_ms: f64,
    p99_ms: f64,
    peak_rss_mb: f64,
    elapsed_sec: f64,
}

fn get_peak_rss_mb() -> f64 {
    #[cfg(target_os = "macos")]
    {
        use std::mem;
        unsafe {
            let mut rusage: libc::rusage = mem::zeroed();
            libc::getrusage(libc::RUSAGE_SELF, &mut rusage);
            rusage.ru_maxrss as f64 / (1024.0 * 1024.0)
        }
    }
    #[cfg(target_os = "linux")]
    {
        use std::mem;
        unsafe {
            let mut rusage: libc::rusage = mem::zeroed();
            libc::getrusage(libc::RUSAGE_SELF, &mut rusage);
            rusage.ru_maxrss as f64 / 1024.0
        }
    }
    #[cfg(not(any(target_os = "macos", target_os = "linux")))]
    {
        0.0
    }
}

fn main() -> Result<()> {
    let cli = Cli::parse();

    match cli {
        Cli::Bench { input, batch_size } => {
            eprintln!("Loading model: all-MiniLM-L6-v2 (ONNX Runtime)");
            let mut model = TextEmbedding::try_new(
                InitOptions::new(EmbeddingModel::AllMiniLML6V2).with_show_download_progress(true),
            )
            .context("Failed to load fastembed model")?;

            // Load input chunks
            let file = std::fs::File::open(&input)
                .with_context(|| format!("Cannot open {input}"))?;
            let reader = BufReader::new(file);
            let mut texts: Vec<String> = Vec::new();
            for line in reader.lines() {
                let line = line?;
                if line.trim().is_empty() {
                    continue;
                }
                let item: InputLine = serde_json::from_str(&line)?;
                texts.push(item.text);
            }
            eprintln!("Loaded {} chunks", texts.len());

            // Warmup
            let warmup_n = 100.min(texts.len());
            let warmup_texts: Vec<String> = texts[..warmup_n].to_vec();
            for chunk in warmup_texts.chunks(batch_size) {
                let _ = model.embed(chunk.to_vec(), Some(batch_size))
                    .context("warmup embed failed")?;
            }
            eprintln!("Warmup done ({} vectors)", warmup_n);

            // Benchmark
            let mut batch_latencies: Vec<f64> = Vec::new();
            let start = Instant::now();
            let mut total_vecs = 0usize;

            for chunk in texts.chunks(batch_size) {
                let batch_start = Instant::now();
                let result = model
                    .embed(chunk.to_vec(), Some(batch_size))
                    .context("embed failed")?;
                let batch_elapsed = batch_start.elapsed().as_secs_f64() * 1000.0;
                batch_latencies.push(batch_elapsed);
                total_vecs += result.len();
            }

            let elapsed = start.elapsed().as_secs_f64();
            let vecs_per_sec = total_vecs as f64 / elapsed;

            batch_latencies.sort_by(|a, b| a.partial_cmp(b).unwrap());
            let p50 = batch_latencies[batch_latencies.len() / 2];
            let p99_idx = (batch_latencies.len() as f64 * 0.99) as usize;
            let p99 = batch_latencies[p99_idx.min(batch_latencies.len() - 1)];

            let peak_rss = get_peak_rss_mb();

            let result = BenchResult {
                approach: "fastembed".into(),
                backend: "onnx-cpu".into(),
                model: "all-MiniLM-L6-v2".into(),
                batch_size,
                total_vecs,
                warmup_vecs: warmup_n,
                vecs_per_sec,
                p50_ms: (p50 * 100.0).round() / 100.0,
                p99_ms: (p99 * 100.0).round() / 100.0,
                peak_rss_mb: (peak_rss * 10.0).round() / 10.0,
                elapsed_sec: (elapsed * 1000.0).round() / 1000.0,
            };

            println!("{}", serde_json::to_string_pretty(&result)?);
        }
    }

    Ok(())
}
