use anyhow::{Context, Result};
use candle_core::{Device, Tensor};
use candle_nn::VarBuilder;
use candle_transformers::models::bert::{BertModel, Config, DTYPE};
use clap::Parser;
use hf_hub::{api::sync::Api, Repo, RepoType};
use serde::{Deserialize, Serialize};
use std::io::{BufRead, BufReader, Write as IoWrite};
use std::path::{Path, PathBuf};
use std::time::Instant;
use tokenizers::{PaddingParams, PaddingStrategy, TruncationParams, Tokenizer};

#[derive(Parser)]
#[command(name = "embedder-candle")]
enum Cli {
    /// Run embedding benchmark on JSONL input
    Bench {
        /// Input JSONL file (one {"text":"..."} per line)
        #[arg(long)]
        input: String,

        /// Batch size
        #[arg(long, default_value = "64")]
        batch_size: usize,

        /// Use Metal GPU (macOS)
        #[arg(long)]
        metal: bool,

        /// Model name on HuggingFace
        #[arg(long, default_value = "sentence-transformers/all-MiniLM-L6-v2")]
        model: String,

        /// Local model directory (skip HF download)
        #[arg(long)]
        model_dir: Option<String>,
    },

    /// Embed markdown files (like "search cc fts embed run")
    Embed {
        /// Input markdown directory
        #[arg(long)]
        input: String,

        /// Output directory (vectors.bin + meta.jsonl)
        #[arg(long)]
        output: String,

        /// Batch size
        #[arg(long, default_value = "64")]
        batch_size: usize,

        /// Max chars per chunk
        #[arg(long, default_value = "500")]
        max_chars: usize,

        /// Chunk overlap chars
        #[arg(long, default_value = "200")]
        overlap: usize,

        /// Use Metal GPU (macOS)
        #[arg(long)]
        metal: bool,

        /// Local model directory
        #[arg(long, default_value_t = default_model_dir())]
        model_dir: String,

        /// Max token sequence length
        #[arg(long, default_value = "128")]
        max_tokens: usize,
    },
}

fn default_model_dir() -> String {
    let home = std::env::var("HOME").unwrap_or_else(|_| ".".into());
    format!("{}/data/models/all-MiniLM-L6-v2", home)
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

#[derive(Serialize)]
struct EmbedMeta {
    id: String,
    file: String,
    chunk_idx: usize,
    text_len: usize,
    dim: usize,
}

#[derive(Serialize)]
struct EmbedStats {
    files: usize,
    chunks: usize,
    vectors: usize,
    errors: usize,
    dim: usize,
    driver: String,
    batch_size: usize,
    elapsed_ms: u128,
    vec_per_sec: f64,
}

fn resolve_model_files(
    model_name: &str,
    model_dir: Option<&str>,
) -> Result<(PathBuf, PathBuf, PathBuf)> {
    if let Some(dir) = model_dir {
        let dir = Path::new(dir);
        let config = dir.join("config.json");
        let tokenizer = dir.join("tokenizer.json");
        let weights = dir.join("model.safetensors");
        anyhow::ensure!(config.exists(), "config.json not found in {}", dir.display());
        anyhow::ensure!(tokenizer.exists(), "tokenizer.json not found in {}", dir.display());
        anyhow::ensure!(weights.exists(), "model.safetensors not found in {}", dir.display());
        eprintln!("Loading model from: {}", dir.display());
        return Ok((config, tokenizer, weights));
    }

    let repo = Repo::with_revision(
        model_name.to_string(),
        RepoType::Model,
        "main".to_string(),
    );
    let api = Api::new()?.repo(repo);
    eprintln!("Downloading model: {}", model_name);
    let config = api.get("config.json").context("config.json")?;
    let tokenizer = api.get("tokenizer.json").context("tokenizer.json")?;
    let weights = api.get("model.safetensors").context("model.safetensors")?;
    Ok((config, tokenizer, weights))
}

fn load_model(
    model_name: &str,
    model_dir: Option<&str>,
    device: &Device,
    max_tokens: usize,
) -> Result<(BertModel, Tokenizer, usize)> {
    let (config_path, tokenizer_path, weights_path) =
        resolve_model_files(model_name, model_dir)?;

    let config: Config =
        serde_json::from_str(&std::fs::read_to_string(&config_path)?)?;
    let dim = config.hidden_size;

    let mut tokenizer =
        Tokenizer::from_file(&tokenizer_path).map_err(|e| anyhow::anyhow!("{e}"))?;

    // Padding: pad to batch longest
    tokenizer.with_padding(Some(PaddingParams {
        strategy: PaddingStrategy::BatchLongest,
        ..Default::default()
    }));

    // Truncation: limit token length
    tokenizer.with_truncation(Some(TruncationParams {
        max_length: max_tokens,
        ..Default::default()
    })).map_err(|e| anyhow::anyhow!("{e}"))?;

    let vb = unsafe {
        VarBuilder::from_mmaped_safetensors(&[weights_path], DTYPE, device)?
    };
    let model = BertModel::load(vb, &config)?;

    eprintln!("Model loaded: dim={}, max_tokens={}", dim, max_tokens);
    Ok((model, tokenizer, dim))
}

fn embed_batch(
    model: &BertModel,
    tokenizer: &Tokenizer,
    texts: &[String],
    device: &Device,
) -> Result<Vec<Vec<f32>>> {
    let tokens = tokenizer
        .encode_batch(texts.to_vec(), true)
        .map_err(|e| anyhow::anyhow!("{e}"))?;

    let token_ids: Vec<Tensor> = tokens
        .iter()
        .map(|t| {
            let ids: Vec<u32> = t.get_ids().to_vec();
            Tensor::new(ids.as_slice(), device)
        })
        .collect::<candle_core::Result<_>>()?;

    let masks: Vec<Tensor> = tokens
        .iter()
        .map(|t| {
            let m: Vec<u32> = t.get_attention_mask().to_vec();
            Tensor::new(m.as_slice(), device)
        })
        .collect::<candle_core::Result<_>>()?;

    let token_ids = Tensor::stack(&token_ids, 0)?;
    let attention_mask = Tensor::stack(&masks, 0)?;
    let token_type_ids = token_ids.zeros_like()?;

    // Forward pass
    let embeddings = model.forward(&token_ids, &token_type_ids, Some(&attention_mask))?;

    // Mean pooling
    let mask_f = attention_mask.to_dtype(DTYPE)?.unsqueeze(2)?;
    let sum_mask = mask_f.sum(1)?;
    let pooled = embeddings.broadcast_mul(&mask_f)?.sum(1)?;
    let pooled = pooled.broadcast_div(&sum_mask)?;

    // L2 normalize
    let norm = pooled.sqr()?.sum_keepdim(1)?.sqrt()?;
    let normalized = pooled.broadcast_div(&norm)?;

    // Convert to Vec<Vec<f32>>
    let n = normalized.dim(0)?;
    let mut results = Vec::with_capacity(n);
    for i in 0..n {
        let row = normalized.get(i)?.to_vec1::<f32>()?;
        results.push(row);
    }
    Ok(results)
}

/// Chunk text into overlapping segments (char-boundary safe)
fn chunk_text(text: &str, max_chars: usize, overlap: usize) -> Vec<String> {
    let text = text.trim();
    if text.is_empty() {
        return vec![];
    }

    // Work with char indices to avoid UTF-8 boundary issues
    let char_indices: Vec<(usize, char)> = text.char_indices().collect();
    let total_chars = char_indices.len();

    if total_chars <= max_chars {
        return vec![text.to_string()];
    }

    let mut chunks = Vec::new();
    let step = max_chars.saturating_sub(overlap).max(1);
    let mut char_start = 0;

    while char_start < total_chars {
        let char_end = (char_start + max_chars).min(total_chars);
        let byte_start = char_indices[char_start].0;
        let byte_end = if char_end < total_chars {
            char_indices[char_end].0
        } else {
            text.len()
        };
        let chunk = text[byte_start..byte_end].trim();
        if !chunk.is_empty() {
            chunks.push(chunk.to_string());
        }
        if char_end >= total_chars {
            break;
        }
        char_start += step;
    }
    chunks
}

fn get_peak_rss_mb() -> f64 {
    #[cfg(target_os = "macos")]
    {
        unsafe {
            let mut rusage: libc::rusage = std::mem::zeroed();
            libc::getrusage(libc::RUSAGE_SELF, &mut rusage);
            rusage.ru_maxrss as f64 / (1024.0 * 1024.0)
        }
    }
    #[cfg(target_os = "linux")]
    {
        unsafe {
            let mut rusage: libc::rusage = std::mem::zeroed();
            libc::getrusage(libc::RUSAGE_SELF, &mut rusage);
            rusage.ru_maxrss as f64 / 1024.0
        }
    }
    #[cfg(not(any(target_os = "macos", target_os = "linux")))]
    {
        0.0
    }
}

fn collect_markdown_files(dir: &Path) -> Result<Vec<PathBuf>> {
    let mut files = Vec::new();
    fn walk(dir: &Path, files: &mut Vec<PathBuf>) -> Result<()> {
        for entry in std::fs::read_dir(dir)? {
            let entry = entry?;
            let path = entry.path();
            if path.is_dir() {
                walk(&path, files)?;
            } else if let Some(name) = path.file_name().and_then(|n| n.to_str()) {
                if name.ends_with(".md") || name.ends_with(".md.gz") {
                    files.push(path);
                }
            }
        }
        Ok(())
    }
    walk(dir, &mut files)?;
    files.sort();
    Ok(files)
}

fn read_markdown(path: &Path) -> Result<String> {
    if path.extension().map(|e| e == "gz").unwrap_or(false) {
        use std::io::Read;
        let file = std::fs::File::open(path)?;
        let mut decoder = flate2::read::GzDecoder::new(file);
        let mut content = String::new();
        decoder.read_to_string(&mut content)?;
        Ok(content)
    } else {
        Ok(std::fs::read_to_string(path)?)
    }
}

fn run_embed(
    input_dir: &str,
    output_dir: &str,
    batch_size: usize,
    max_chars: usize,
    overlap: usize,
    metal: bool,
    model_dir: &str,
    max_tokens: usize,
) -> Result<()> {
    let device = if metal {
        #[cfg(feature = "metal")]
        { Device::new_metal(0)? }
        #[cfg(not(feature = "metal"))]
        { anyhow::bail!("Metal not enabled"); }
    } else {
        Device::Cpu
    };

    let (model, tokenizer, dim) =
        load_model("all-MiniLM-L6-v2", Some(model_dir), &device, max_tokens)?;

    // Collect files
    let files = collect_markdown_files(Path::new(input_dir))?;
    eprintln!("Found {} markdown files", files.len());

    // Chunk all files
    struct ChunkItem {
        file: String,
        chunk_idx: usize,
        text: String,
    }

    let mut all_chunks: Vec<ChunkItem> = Vec::new();
    let mut file_count = 0;
    let mut errors = 0;

    for path in &files {
        match read_markdown(path) {
            Ok(content) => {
                let rel = path.strip_prefix(input_dir).unwrap_or(path);
                let rel_str = rel.to_string_lossy().to_string();
                let chunks = chunk_text(&content, max_chars, overlap);
                for (i, text) in chunks.into_iter().enumerate() {
                    all_chunks.push(ChunkItem {
                        file: rel_str.clone(),
                        chunk_idx: i,
                        text,
                    });
                }
                file_count += 1;
            }
            Err(_) => {
                errors += 1;
            }
        }
    }

    eprintln!("{} files → {} chunks ({} errors)", file_count, all_chunks.len(), errors);

    // Create output dir
    std::fs::create_dir_all(output_dir)?;
    let vec_path = Path::new(output_dir).join("vectors.bin");
    let meta_path = Path::new(output_dir).join("meta.jsonl");
    let stats_path = Path::new(output_dir).join("stats.json");

    let mut vec_file = std::fs::File::create(&vec_path)?;
    let mut meta_file = std::fs::File::create(&meta_path)?;

    let start = Instant::now();
    let mut total_vecs = 0usize;
    let total_chunks = all_chunks.len();

    // Process in batches
    for (batch_idx, batch) in all_chunks.chunks(batch_size).enumerate() {
        let texts: Vec<String> = batch.iter().map(|c| c.text.clone()).collect();
        let vectors = embed_batch(&model, &tokenizer, &texts, &device)?;

        for (i, vec) in vectors.iter().enumerate() {
            let item = &batch[i];

            // Write raw f32 vector
            let bytes: &[u8] = unsafe {
                std::slice::from_raw_parts(
                    vec.as_ptr() as *const u8,
                    vec.len() * 4,
                )
            };
            vec_file.write_all(bytes)?;

            // Write metadata
            let meta = EmbedMeta {
                id: format!("{}:{}", item.file, item.chunk_idx),
                file: item.file.clone(),
                chunk_idx: item.chunk_idx,
                text_len: item.text.len(),
                dim,
            };
            serde_json::to_writer(&mut meta_file, &meta)?;
            meta_file.write_all(b"\n")?;

            total_vecs += 1;
        }

        // Progress every 10 batches
        if batch_idx % 10 == 0 {
            let elapsed = start.elapsed().as_secs_f64();
            let vps = if elapsed > 0.0 { total_vecs as f64 / elapsed } else { 0.0 };
            eprint!("\r{}/{} chunks ({:.0} vec/s)", total_vecs, total_chunks, vps);
        }
    }

    let elapsed = start.elapsed();
    let vps = total_vecs as f64 / elapsed.as_secs_f64();
    eprintln!("\r{}/{} chunks done ({:.0} vec/s)     ", total_vecs, total_chunks, vps);

    // Write stats
    let stats = EmbedStats {
        files: file_count,
        chunks: total_chunks,
        vectors: total_vecs,
        errors,
        dim,
        driver: "candle".into(),
        batch_size,
        elapsed_ms: elapsed.as_millis(),
        vec_per_sec: (vps * 10.0).round() / 10.0,
    };
    let stats_json = serde_json::to_string_pretty(&stats)?;
    std::fs::write(&stats_path, &stats_json)?;
    eprintln!("{}", stats_json);
    eprintln!("Output: {}", output_dir);
    eprintln!("Peak RSS: {:.0} MB", get_peak_rss_mb());

    Ok(())
}

fn main() -> Result<()> {
    let cli = Cli::parse();

    match cli {
        Cli::Bench {
            input,
            batch_size,
            metal,
            model: model_name,
            model_dir,
        } => {
            let device = if metal {
                #[cfg(feature = "metal")]
                { Device::new_metal(0).context("Metal device")? }
                #[cfg(not(feature = "metal"))]
                { anyhow::bail!("Metal not enabled"); }
            } else {
                Device::Cpu
            };

            let backend = if metal { "metal" } else { "cpu" };
            eprintln!("Device: {}", backend);

            let (model, tokenizer, _dim) =
                load_model(&model_name, model_dir.as_deref(), &device, 128)?;

            let file = std::fs::File::open(&input)
                .with_context(|| format!("Cannot open {input}"))?;
            let reader = BufReader::new(file);
            let mut texts: Vec<String> = Vec::new();
            for line in reader.lines() {
                let line = line?;
                if line.trim().is_empty() { continue; }
                let item: InputLine = serde_json::from_str(&line)?;
                texts.push(item.text);
            }
            eprintln!("Loaded {} chunks", texts.len());

            // Warmup
            let warmup_n = 100.min(texts.len());
            for chunk in texts[..warmup_n].chunks(batch_size) {
                let chunk_vec: Vec<String> = chunk.to_vec();
                let _ = embed_batch(&model, &tokenizer, &chunk_vec, &device)?;
            }
            eprintln!("Warmup done ({} vectors)", warmup_n);

            // Benchmark
            let mut batch_latencies: Vec<f64> = Vec::new();
            let start = Instant::now();
            let mut total_vecs = 0usize;

            for chunk in texts.chunks(batch_size) {
                let chunk_vec: Vec<String> = chunk.to_vec();
                let batch_start = Instant::now();
                let result = embed_batch(&model, &tokenizer, &chunk_vec, &device)?;
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

            let result = BenchResult {
                approach: "candle".into(),
                backend: backend.into(),
                model: model_name,
                batch_size,
                total_vecs,
                warmup_vecs: warmup_n,
                vecs_per_sec,
                p50_ms: (p50 * 100.0).round() / 100.0,
                p99_ms: (p99 * 100.0).round() / 100.0,
                peak_rss_mb: (get_peak_rss_mb() * 10.0).round() / 10.0,
                elapsed_sec: (elapsed * 1000.0).round() / 1000.0,
            };

            println!("{}", serde_json::to_string_pretty(&result)?);
        }

        Cli::Embed {
            input,
            output,
            batch_size,
            max_chars,
            overlap,
            metal,
            model_dir,
            max_tokens,
        } => {
            run_embed(&input, &output, batch_size, max_chars, overlap, metal, &model_dir, max_tokens)?;
        }
    }

    Ok(())
}
