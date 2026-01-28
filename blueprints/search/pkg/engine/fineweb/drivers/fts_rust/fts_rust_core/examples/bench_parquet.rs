//! Benchmark: Read parquet file and measure pure indexing throughput
//!
//! Usage: cargo run --release --example bench_parquet --features bench -- <parquet_path> [profile]
//!
//! This benchmark:
//! 1. Reads all documents from parquet into memory (not measured)
//! 2. Indexes documents using specified profile (measured)
//! 3. Reports throughput in docs/sec

use std::env;
use std::fs::File;
use std::path::Path;
use std::time::Instant;

use arrow::array::StringArray;
use arrow::datatypes::FieldRef;
use arrow::record_batch::RecordBatch;
use parquet::arrow::arrow_reader::ParquetRecordBatchReaderBuilder;

use fts_rust_core::{Document, FtsIndex};

fn main() {
    let args: Vec<String> = env::args().collect();

    if args.len() < 2 {
        eprintln!("Usage: {} <parquet_path> [profile]", args[0]);
        eprintln!("  parquet_path: Path to parquet file or directory");
        eprintln!("  profile: ultra (default), tantivy, turbo, etc.");
        std::process::exit(1);
    }

    let parquet_path = &args[1];
    let profile = args.get(2).map(|s| s.as_str()).unwrap_or("ultra");

    println!("=== FTS Rust Parquet Benchmark ===");
    println!("Parquet: {}", parquet_path);
    println!("Profile: {}", profile);
    println!();

    // Phase 1: Read parquet into memory (not measured)
    println!("[Phase 1] Reading parquet into memory...");
    let read_start = Instant::now();
    let documents = read_parquet(parquet_path);
    let read_duration = read_start.elapsed();
    println!(
        "  Read {} documents in {:.2}s ({:.0} docs/sec)",
        documents.len(),
        read_duration.as_secs_f64(),
        documents.len() as f64 / read_duration.as_secs_f64()
    );
    println!();

    // Phase 2: Index documents (measured)
    println!("[Phase 2] Indexing documents (MEASURED)...");
    let temp_dir = tempfile::tempdir().expect("Failed to create temp dir");
    let index_path = temp_dir.path().to_str().unwrap();

    let index = FtsIndex::create(index_path, profile).expect("Failed to create index");

    // Warm up (small batch)
    let warmup_size = 1000.min(documents.len());
    let _ = index.index_batch(&documents[..warmup_size]);
    index.clear();

    // Benchmark indexing
    let batch_size = 100_000;
    let index_start = Instant::now();
    let mut indexed = 0;

    for chunk in documents.chunks(batch_size) {
        let n = index.index_batch(chunk).expect("Index batch failed");
        indexed += n;

        let elapsed = index_start.elapsed().as_secs_f64();
        if elapsed > 0.0 {
            let rate = indexed as f64 / elapsed;
            print!(
                "\r  Progress: {}/{} docs ({:.0} docs/sec)    ",
                indexed,
                documents.len(),
                rate
            );
        }
    }

    // Commit
    index.commit().expect("Commit failed");
    let index_duration = index_start.elapsed();

    println!();
    println!();

    // Results
    let throughput = documents.len() as f64 / index_duration.as_secs_f64();
    println!("=== RESULTS ===");
    println!("Profile:      {}", profile);
    println!("Documents:    {}", documents.len());
    println!("Duration:     {:.3}s", index_duration.as_secs_f64());
    println!("Throughput:   {:.0} docs/sec", throughput);
    println!();

    // Memory stats
    let stats = index.memory_stats();
    println!("Memory Stats:");
    println!("  Index bytes:     {} MB", stats.index_bytes / 1024 / 1024);
    println!("  Term dict bytes: {} MB", stats.term_dict_bytes / 1024 / 1024);
    println!("  Postings bytes:  {} MB", stats.postings_bytes / 1024 / 1024);
    println!("  Docs indexed:    {}", stats.docs_indexed);
}

fn read_parquet(path: &str) -> Vec<Document> {
    let path = Path::new(path);
    let mut documents = Vec::new();

    if path.is_dir() {
        // Read all parquet files in directory
        for entry in std::fs::read_dir(path).expect("Failed to read directory") {
            let entry = entry.expect("Failed to read entry");
            let file_path = entry.path();
            if file_path
                .extension()
                .map(|e| e == "parquet")
                .unwrap_or(false)
            {
                documents.extend(read_single_parquet(&file_path));
            }
        }
    } else {
        documents = read_single_parquet(path);
    }

    documents
}

fn read_single_parquet(path: &Path) -> Vec<Document> {
    let file = File::open(path).expect("Failed to open parquet file");
    let builder =
        ParquetRecordBatchReaderBuilder::try_new(file).expect("Failed to create parquet reader");

    let reader = builder.build().expect("Failed to build reader");
    let mut documents = Vec::new();

    for batch_result in reader {
        let batch: RecordBatch = batch_result.expect("Failed to read batch");

        // Find id and text columns
        let schema = batch.schema();
        let id_idx = schema
            .fields()
            .iter()
            .position(|f: &FieldRef| f.name() == "id")
            .expect("No 'id' column found");
        let text_idx = schema
            .fields()
            .iter()
            .position(|f: &FieldRef| f.name() == "text")
            .expect("No 'text' column found");

        let id_col = batch.column(id_idx);
        let text_col = batch.column(text_idx);

        // Get as string arrays
        let ids = id_col
            .as_any()
            .downcast_ref::<StringArray>()
            .expect("id is not string");
        let texts = text_col
            .as_any()
            .downcast_ref::<StringArray>()
            .expect("text is not string");

        for i in 0..batch.num_rows() {
            let id = ids.value(i);
            let text = texts.value(i);
            documents.push(Document::new(id.to_string(), text.to_string()));
        }
    }

    documents
}
