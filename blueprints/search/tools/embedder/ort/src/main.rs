use anyhow::{Context, Result};
use clap::Parser;
use ndarray::{Array2, Axis};
use ort::session::builder::GraphOptimizationLevel;
use ort::session::Session;
use ort::value::TensorRef;
use serde::{Deserialize, Serialize};
use std::io::{BufRead, BufReader, Write as IoWrite};
use std::path::{Path, PathBuf};
use std::time::Instant;
use tokenizers::{PaddingParams, PaddingStrategy, TruncationParams, Tokenizer};

#[derive(Parser)]
#[command(name = "embedder-ort")]
enum Cli {
    /// Benchmark on JSONL input
    Bench {
        #[arg(long)]
        input: String,
        #[arg(long, default_value = "64")]
        batch_size: usize,
        /// Use CoreML execution provider (Apple Neural Engine)
        #[arg(long)]
        coreml: bool,
        /// ONNX model file path
        #[arg(long)]
        model: String,
        /// Tokenizer JSON path
        #[arg(long)]
        tokenizer: String,
        /// Number of intra-op threads (0 = auto)
        #[arg(long, default_value = "0")]
        threads: usize,
    },

    /// Embed markdown files (like "search cc fts embed run")
    Embed {
        #[arg(long)]
        input: String,
        #[arg(long)]
        output: String,
        #[arg(long, default_value = "64")]
        batch_size: usize,
        #[arg(long, default_value = "500")]
        max_chars: usize,
        #[arg(long, default_value = "200")]
        overlap: usize,
        /// Use CoreML EP
        #[arg(long)]
        coreml: bool,
        /// ONNX model file
        #[arg(long)]
        model: String,
        /// Tokenizer JSON
        #[arg(long)]
        tokenizer: String,
        #[arg(long, default_value = "128")]
        max_tokens: usize,
        #[arg(long, default_value = "0")]
        threads: usize,
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

fn build_session(model_path: &str, coreml: bool, threads: usize) -> Result<Session> {
    let mut builder = Session::builder()?
        .with_optimization_level(GraphOptimizationLevel::Level3)?;

    if threads > 0 {
        builder = builder.with_intra_threads(threads)?;
    }

    if coreml {
        #[cfg(target_os = "macos")]
        {
            use ort::execution_providers::CoreMLExecutionProvider;
            builder = builder.with_execution_providers([
                CoreMLExecutionProvider::default()
                    .with_subgraphs(true)
                    .build(),
            ])?;
            eprintln!("CoreML EP requested");
        }
        #[cfg(not(target_os = "macos"))]
        {
            anyhow::bail!("CoreML only available on macOS");
        }
    }

    let session = builder.commit_from_file(model_path)
        .with_context(|| format!("Failed to load ONNX model: {model_path}"))?;

    eprintln!("Model loaded: {}", model_path);

    Ok(session)
}

fn build_tokenizer(path: &str, max_tokens: usize) -> Result<Tokenizer> {
    let mut tokenizer = Tokenizer::from_file(path)
        .map_err(|e| anyhow::anyhow!("tokenizer: {e}"))?;

    tokenizer.with_padding(Some(PaddingParams {
        strategy: PaddingStrategy::BatchLongest,
        ..Default::default()
    }));

    tokenizer.with_truncation(Some(TruncationParams {
        max_length: max_tokens,
        ..Default::default()
    })).map_err(|e| anyhow::anyhow!("{e}"))?;

    Ok(tokenizer)
}

fn embed_batch(
    session: &mut Session,
    tokenizer: &Tokenizer,
    texts: &[String],
) -> Result<Vec<Vec<f32>>> {
    let encodings = tokenizer
        .encode_batch(texts.to_vec(), true)
        .map_err(|e| anyhow::anyhow!("{e}"))?;

    let seq_len = encodings[0].len();
    let batch_size = encodings.len();

    let ids: Vec<i64> = encodings.iter()
        .flat_map(|e| e.get_ids().iter().map(|&i| i as i64))
        .collect();
    let mask: Vec<i64> = encodings.iter()
        .flat_map(|e| e.get_attention_mask().iter().map(|&i| i as i64))
        .collect();
    let type_ids: Vec<i64> = vec![0i64; batch_size * seq_len];

    let t_ids = TensorRef::from_array_view(([batch_size, seq_len], &*ids))?;
    let t_mask = TensorRef::from_array_view(([batch_size, seq_len], &*mask))?;
    let t_type = TensorRef::from_array_view(([batch_size, seq_len], &*type_ids))?;

    // Run with 3 inputs: input_ids, attention_mask, token_type_ids
    let outputs = session.run(ort::inputs![t_ids, t_mask, t_type])?;

    // Try output[1] (sentence_embedding, already pooled), fallback to output[0]
    let n_outputs = outputs.len();
    let embeddings: Array2<f32> = if n_outputs > 1 {
        outputs[1].try_extract_array::<f32>()?
            .into_dimensionality::<ndarray::Ix2>()
            .map_err(|e| anyhow::anyhow!("shape: {e}"))?
            .to_owned()
    } else {
        // last_hidden_state [batch, seq, dim] — need mean pooling
        let hidden = outputs[0].try_extract_array::<f32>()?
            .into_dimensionality::<ndarray::Ix3>()
            .map_err(|e| anyhow::anyhow!("shape: {e}"))?;
        mean_pool(&hidden.view(), &mask, batch_size, seq_len)
    };

    // L2 normalize
    let mut results = Vec::with_capacity(batch_size);
    for row in embeddings.axis_iter(Axis(0)) {
        let norm: f32 = row.iter().map(|x| x * x).sum::<f32>().sqrt();
        let normalized: Vec<f32> = if norm > 0.0 {
            row.iter().map(|x| x / norm).collect()
        } else {
            row.to_vec()
        };
        results.push(normalized);
    }

    Ok(results)
}

fn mean_pool(
    hidden: &ndarray::ArrayView3<f32>,
    mask: &[i64],
    batch_size: usize,
    seq_len: usize,
) -> Array2<f32> {
    let dim = hidden.shape()[2];
    let mut pooled = Array2::<f32>::zeros((batch_size, dim));

    for b in 0..batch_size {
        let mut count = 0.0f32;
        for s in 0..seq_len {
            let m = mask[b * seq_len + s] as f32;
            if m > 0.0 {
                for d in 0..dim {
                    pooled[[b, d]] += hidden[[b, s, d]] * m;
                }
                count += m;
            }
        }
        if count > 0.0 {
            for d in 0..dim {
                pooled[[b, d]] /= count;
            }
        }
    }
    pooled
}

fn chunk_text(text: &str, max_chars: usize, overlap: usize) -> Vec<String> {
    let text = text.trim();
    if text.is_empty() { return vec![]; }
    let chars: Vec<(usize, char)> = text.char_indices().collect();
    let n = chars.len();
    if n <= max_chars { return vec![text.to_string()]; }

    let step = max_chars.saturating_sub(overlap).max(1);
    let mut chunks = Vec::new();
    let mut i = 0;
    while i < n {
        let end = (i + max_chars).min(n);
        let bs = chars[i].0;
        let be = if end < n { chars[end].0 } else { text.len() };
        let c = text[bs..be].trim();
        if !c.is_empty() { chunks.push(c.to_string()); }
        if end >= n { break; }
        i += step;
    }
    chunks
}

fn collect_markdown_files(dir: &Path) -> Result<Vec<PathBuf>> {
    let mut files = Vec::new();
    fn walk(dir: &Path, files: &mut Vec<PathBuf>) -> Result<()> {
        for entry in std::fs::read_dir(dir)? {
            let entry = entry?;
            let p = entry.path();
            if p.is_dir() { walk(&p, files)?; }
            else if let Some(n) = p.file_name().and_then(|n| n.to_str()) {
                if n.ends_with(".md") || n.ends_with(".md.gz") { files.push(p); }
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
        let f = std::fs::File::open(path)?;
        let mut dec = flate2::read::GzDecoder::new(f);
        let mut s = String::new();
        dec.read_to_string(&mut s)?;
        Ok(s)
    } else {
        Ok(std::fs::read_to_string(path)?)
    }
}

fn get_peak_rss_mb() -> f64 {
    #[cfg(target_os = "macos")]
    unsafe {
        let mut ru: libc::rusage = std::mem::zeroed();
        libc::getrusage(libc::RUSAGE_SELF, &mut ru);
        ru.ru_maxrss as f64 / (1024.0 * 1024.0)
    }
    #[cfg(target_os = "linux")]
    unsafe {
        let mut ru: libc::rusage = std::mem::zeroed();
        libc::getrusage(libc::RUSAGE_SELF, &mut ru);
        ru.ru_maxrss as f64 / 1024.0
    }
    #[cfg(not(any(target_os = "macos", target_os = "linux")))]
    { 0.0 }
}

fn main() -> Result<()> {
    let cli = Cli::parse();

    match cli {
        Cli::Bench { input, batch_size, coreml, model, tokenizer, threads } => {
            let mut session = build_session(&model, coreml, threads)?;
            let tok = build_tokenizer(&tokenizer, 128)?;

            let file = std::fs::File::open(&input)?;
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
                let _ = embed_batch(&mut session, &tok, &chunk.to_vec())?;
            }
            eprintln!("Warmup done");

            let mut latencies: Vec<f64> = Vec::new();
            let start = Instant::now();
            let mut total = 0usize;

            for chunk in texts.chunks(batch_size) {
                let t0 = Instant::now();
                let vecs = embed_batch(&mut session, &tok, &chunk.to_vec())?;
                latencies.push(t0.elapsed().as_secs_f64() * 1000.0);
                total += vecs.len();
            }

            let elapsed = start.elapsed().as_secs_f64();
            latencies.sort_by(|a, b| a.partial_cmp(b).unwrap());
            let p50 = latencies[latencies.len() / 2];
            let p99i = (latencies.len() as f64 * 0.99) as usize;
            let p99 = latencies[p99i.min(latencies.len() - 1)];

            let backend = if coreml { "coreml" } else { "cpu" };
            let result = BenchResult {
                approach: "ort".into(),
                backend: backend.into(),
                model,
                batch_size,
                total_vecs: total,
                warmup_vecs: warmup_n,
                vecs_per_sec: total as f64 / elapsed,
                p50_ms: (p50 * 100.0).round() / 100.0,
                p99_ms: (p99 * 100.0).round() / 100.0,
                peak_rss_mb: (get_peak_rss_mb() * 10.0).round() / 10.0,
                elapsed_sec: (elapsed * 1000.0).round() / 1000.0,
            };
            println!("{}", serde_json::to_string_pretty(&result)?);
        }

        Cli::Embed { input, output, batch_size, max_chars, overlap, coreml, model, tokenizer, max_tokens, threads } => {
            let mut session = build_session(&model, coreml, threads)?;
            let tok = build_tokenizer(&tokenizer, max_tokens)?;

            let files = collect_markdown_files(Path::new(&input))?;
            eprintln!("Found {} markdown files", files.len());

            struct ChunkItem { file: String, chunk_idx: usize, text: String }
            let mut all_chunks: Vec<ChunkItem> = Vec::new();
            let (mut file_count, mut errors) = (0usize, 0usize);

            for path in &files {
                match read_markdown(path) {
                    Ok(content) => {
                        let rel = path.strip_prefix(&input).unwrap_or(path);
                        let rel_str = rel.to_string_lossy().to_string();
                        for (i, text) in chunk_text(&content, max_chars, overlap).into_iter().enumerate() {
                            all_chunks.push(ChunkItem { file: rel_str.clone(), chunk_idx: i, text });
                        }
                        file_count += 1;
                    }
                    Err(_) => { errors += 1; }
                }
            }

            let total_chunks = all_chunks.len();
            eprintln!("{} files -> {} chunks ({} errors)", file_count, total_chunks, errors);

            std::fs::create_dir_all(&output)?;
            let mut vec_file = std::fs::File::create(Path::new(&output).join("vectors.bin"))?;
            let mut meta_file = std::fs::File::create(Path::new(&output).join("meta.jsonl"))?;

            let start = Instant::now();
            let mut total_vecs = 0usize;

            // Detect dim from first batch
            let first: Vec<String> = all_chunks.iter().take(1).map(|c| c.text.clone()).collect();
            let first_vecs = embed_batch(&mut session, &tok, &first)?;
            let dim = first_vecs[0].len();
            eprintln!("Embedding dim: {}", dim);

            for (bi, batch) in all_chunks.chunks(batch_size).enumerate() {
                let texts: Vec<String> = batch.iter().map(|c| c.text.clone()).collect();
                let vectors = embed_batch(&mut session, &tok, &texts)?;

                for (i, vec) in vectors.iter().enumerate() {
                    let item = &batch[i];
                    let bytes: &[u8] = unsafe {
                        std::slice::from_raw_parts(vec.as_ptr() as *const u8, vec.len() * 4)
                    };
                    vec_file.write_all(bytes)?;
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

                if bi % 10 == 0 {
                    let el = start.elapsed().as_secs_f64();
                    let vps = if el > 0.0 { total_vecs as f64 / el } else { 0.0 };
                    eprint!("\r{}/{} ({:.0} vec/s)", total_vecs, total_chunks, vps);
                }
            }

            let elapsed = start.elapsed();
            let vps = total_vecs as f64 / elapsed.as_secs_f64();
            eprintln!("\r{}/{} done ({:.0} vec/s)     ", total_vecs, total_chunks, vps);

            let stats = EmbedStats {
                files: file_count, chunks: total_chunks, vectors: total_vecs, errors,
                dim, driver: if coreml { "ort-coreml" } else { "ort-cpu" }.into(),
                batch_size, elapsed_ms: elapsed.as_millis(),
                vec_per_sec: (vps * 10.0).round() / 10.0,
            };
            std::fs::write(Path::new(&output).join("stats.json"), serde_json::to_string_pretty(&stats)?)?;
            eprintln!("{}", serde_json::to_string_pretty(&stats)?);
            eprintln!("Peak RSS: {:.0} MB", get_peak_rss_mb());
        }
    }

    Ok(())
}
