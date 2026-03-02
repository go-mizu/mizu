use std::fs::File;
use std::path::Path;
use std::sync::{Arc, Mutex};

use anyhow::{Context, Result};
use arrow::array::{ArrayRef, Int32Builder, Int64Builder, StringBuilder};
use arrow::datatypes::{DataType, Field, Schema};
use arrow::record_batch::RecordBatch;
use parquet::arrow::ArrowWriter;
use parquet::basic::Compression;
use parquet::file::properties::WriterProperties;

use super::{FailureWriter, ResultWriter};
use crate::types::{CrawlResult, FailedDomain, FailedURL};

// ---------------------------------------------------------------------------
// Result writer
// ---------------------------------------------------------------------------

pub struct ParquetResultWriter {
    inner: Mutex<ParquetResultInner>,
}

struct ParquetResultInner {
    writer: Option<ArrowWriter<File>>,
    batch: Vec<CrawlResult>,
    batch_size: usize,
}

fn result_schema() -> Arc<Schema> {
    Arc::new(Schema::new(vec![
        Field::new("url", DataType::Utf8, false),
        Field::new("status_code", DataType::Int32, false),
        Field::new("content_type", DataType::Utf8, false),
        Field::new("content_length", DataType::Int64, false),
        Field::new("body", DataType::Utf8, false),
        Field::new("title", DataType::Utf8, false),
        Field::new("description", DataType::Utf8, false),
        Field::new("language", DataType::Utf8, false),
        Field::new("domain", DataType::Utf8, false),
        Field::new("redirect_url", DataType::Utf8, false),
        Field::new("fetch_time_ms", DataType::Int64, false),
        Field::new("crawled_at", DataType::Utf8, false),
        Field::new("error", DataType::Utf8, false),
    ]))
}

impl ParquetResultWriter {
    pub fn new(output_dir: &Path, batch_size: usize) -> Result<Self> {
        std::fs::create_dir_all(output_dir)?;
        let path = output_dir.join("results.parquet");
        let file = File::create(&path)
            .with_context(|| format!("create parquet file: {}", path.display()))?;

        let schema = result_schema();
        let props = WriterProperties::builder()
            .set_compression(Compression::SNAPPY)
            .build();
        let writer = ArrowWriter::try_new(file, schema, Some(props))
            .context("create ArrowWriter for results")?;

        Ok(Self {
            inner: Mutex::new(ParquetResultInner {
                writer: Some(writer),
                batch: Vec::with_capacity(batch_size),
                batch_size,
            }),
        })
    }
}

/// Build a `RecordBatch` from a slice of `CrawlResult`.
fn build_result_batch(rows: &[CrawlResult]) -> Result<RecordBatch> {
    let len = rows.len();
    let schema = result_schema();

    let mut url = StringBuilder::with_capacity(len, len * 64);
    let mut status_code = Int32Builder::with_capacity(len);
    let mut content_type = StringBuilder::with_capacity(len, len * 16);
    let mut content_length = Int64Builder::with_capacity(len);
    let mut body = StringBuilder::with_capacity(len, 0);
    let mut title = StringBuilder::with_capacity(len, len * 64);
    let mut description = StringBuilder::with_capacity(len, len * 128);
    let mut language = StringBuilder::with_capacity(len, len * 4);
    let mut domain = StringBuilder::with_capacity(len, len * 32);
    let mut redirect_url = StringBuilder::with_capacity(len, len * 32);
    let mut fetch_time_ms = Int64Builder::with_capacity(len);
    let mut crawled_at = StringBuilder::with_capacity(len, len * 20);
    let mut error = StringBuilder::with_capacity(len, len * 32);

    for r in rows {
        url.append_value(&r.url);
        status_code.append_value(r.status_code as i32);
        content_type.append_value(&r.content_type);
        content_length.append_value(r.content_length);
        body.append_value(&r.body);
        title.append_value(&r.title);
        description.append_value(&r.description);
        language.append_value(&r.language);
        domain.append_value(&r.domain);
        redirect_url.append_value(&r.redirect_url);
        fetch_time_ms.append_value(r.fetch_time_ms);
        crawled_at.append_value(r.crawled_at.format("%Y-%m-%d %H:%M:%S").to_string());
        error.append_value(&r.error);
    }

    let columns: Vec<ArrayRef> = vec![
        Arc::new(url.finish()),
        Arc::new(status_code.finish()),
        Arc::new(content_type.finish()),
        Arc::new(content_length.finish()),
        Arc::new(body.finish()),
        Arc::new(title.finish()),
        Arc::new(description.finish()),
        Arc::new(language.finish()),
        Arc::new(domain.finish()),
        Arc::new(redirect_url.finish()),
        Arc::new(fetch_time_ms.finish()),
        Arc::new(crawled_at.finish()),
        Arc::new(error.finish()),
    ];

    RecordBatch::try_new(schema, columns).context("build result RecordBatch")
}

/// Flush buffered results as a row group. Caller must hold the lock.
fn flush_result_batch(inner: &mut ParquetResultInner) -> Result<()> {
    if inner.batch.is_empty() {
        return Ok(());
    }
    let batch = build_result_batch(&inner.batch)?;
    if let Some(ref mut writer) = inner.writer {
        writer.write(&batch).context("write result row group")?;
    }
    inner.batch.clear();
    Ok(())
}

impl ResultWriter for ParquetResultWriter {
    fn write(&self, result: CrawlResult) -> Result<()> {
        let mut inner = self.inner.lock().unwrap();
        inner.batch.push(result);
        if inner.batch.len() >= inner.batch_size {
            flush_result_batch(&mut inner)?;
        }
        Ok(())
    }

    fn flush(&self) -> Result<()> {
        let mut inner = self.inner.lock().unwrap();
        flush_result_batch(&mut inner)?;
        if let Some(ref mut writer) = inner.writer {
            writer.flush().context("flush ArrowWriter for results")?;
        }
        Ok(())
    }

    fn close(&self) -> Result<()> {
        let mut inner = self.inner.lock().unwrap();
        flush_result_batch(&mut inner)?;
        if let Some(writer) = inner.writer.take() {
            writer.close().context("close ArrowWriter for results")?;
        }
        Ok(())
    }
}

// ---------------------------------------------------------------------------
// Failure writer
// ---------------------------------------------------------------------------

pub struct ParquetFailureWriter {
    inner: Mutex<ParquetFailureInner>,
}

struct ParquetFailureInner {
    writer: Option<ArrowWriter<File>>,
    batch: Vec<FailedURL>,
    batch_size: usize,
}

fn failure_schema() -> Arc<Schema> {
    Arc::new(Schema::new(vec![
        Field::new("url", DataType::Utf8, false),
        Field::new("domain", DataType::Utf8, false),
        Field::new("reason", DataType::Utf8, false),
        Field::new("subcategory", DataType::Utf8, false),
        Field::new("error", DataType::Utf8, false),
        Field::new("status_code", DataType::Int32, false),
        Field::new("fetch_time_ms", DataType::Int64, false),
        Field::new("detected_at", DataType::Utf8, false),
    ]))
}

impl ParquetFailureWriter {
    pub fn new(output_dir: &Path, batch_size: usize) -> Result<Self> {
        std::fs::create_dir_all(output_dir)?;
        let path = output_dir.join("failed_urls.parquet");
        let file = File::create(&path)
            .with_context(|| format!("create parquet file: {}", path.display()))?;

        let schema = failure_schema();
        let props = WriterProperties::builder()
            .set_compression(Compression::SNAPPY)
            .build();
        let writer = ArrowWriter::try_new(file, schema, Some(props))
            .context("create ArrowWriter for failures")?;

        Ok(Self {
            inner: Mutex::new(ParquetFailureInner {
                writer: Some(writer),
                batch: Vec::with_capacity(batch_size),
                batch_size,
            }),
        })
    }
}

/// Build a `RecordBatch` from a slice of `FailedURL`.
fn build_failure_batch(rows: &[FailedURL]) -> Result<RecordBatch> {
    let len = rows.len();
    let schema = failure_schema();

    let mut url = StringBuilder::with_capacity(len, len * 64);
    let mut domain = StringBuilder::with_capacity(len, len * 32);
    let mut reason = StringBuilder::with_capacity(len, len * 24);
    let mut subcategory = StringBuilder::with_capacity(len, len * 16);
    let mut error = StringBuilder::with_capacity(len, len * 64);
    let mut status_code = Int32Builder::with_capacity(len);
    let mut fetch_time_ms = Int64Builder::with_capacity(len);
    let mut detected_at = StringBuilder::with_capacity(len, len * 20);

    for f in rows {
        url.append_value(&f.url);
        domain.append_value(&f.domain);
        reason.append_value(&f.reason);
        subcategory.append_value(&f.subcategory);
        error.append_value(&f.error);
        status_code.append_value(f.status_code as i32);
        fetch_time_ms.append_value(f.fetch_time_ms);
        detected_at.append_value(f.detected_at.format("%Y-%m-%d %H:%M:%S").to_string());
    }

    let columns: Vec<ArrayRef> = vec![
        Arc::new(url.finish()),
        Arc::new(domain.finish()),
        Arc::new(reason.finish()),
        Arc::new(subcategory.finish()),
        Arc::new(error.finish()),
        Arc::new(status_code.finish()),
        Arc::new(fetch_time_ms.finish()),
        Arc::new(detected_at.finish()),
    ];

    RecordBatch::try_new(schema, columns).context("build failure RecordBatch")
}

/// Flush buffered failures as a row group. Caller must hold the lock.
fn flush_failure_batch(inner: &mut ParquetFailureInner) -> Result<()> {
    if inner.batch.is_empty() {
        return Ok(());
    }
    let batch = build_failure_batch(&inner.batch)?;
    if let Some(ref mut writer) = inner.writer {
        writer.write(&batch).context("write failure row group")?;
    }
    inner.batch.clear();
    Ok(())
}

impl FailureWriter for ParquetFailureWriter {
    fn write_url(&self, failed: FailedURL) -> Result<()> {
        let mut inner = self.inner.lock().unwrap();
        inner.batch.push(failed);
        if inner.batch.len() >= inner.batch_size {
            flush_failure_batch(&mut inner)?;
        }
        Ok(())
    }

    fn write_domain(&self, _failed: FailedDomain) -> Result<()> {
        // FailedDomain records are small metadata; skip for parquet output.
        Ok(())
    }

    fn flush(&self) -> Result<()> {
        let mut inner = self.inner.lock().unwrap();
        flush_failure_batch(&mut inner)?;
        if let Some(ref mut writer) = inner.writer {
            writer.flush().context("flush ArrowWriter for failures")?;
        }
        Ok(())
    }

    fn close(&self) -> Result<()> {
        let mut inner = self.inner.lock().unwrap();
        flush_failure_batch(&mut inner)?;
        if let Some(writer) = inner.writer.take() {
            writer.close().context("close ArrowWriter for failures")?;
        }
        Ok(())
    }
}
