pub mod binary;
pub mod devnull;
pub mod duckdb_writer;
pub mod parquet_writer;

use crate::types::{CrawlResult, FailedDomain, FailedURL};
use anyhow::Result;

/// Trait for writing crawl results to storage.
pub trait ResultWriter: Send + Sync {
    fn write(&self, result: CrawlResult) -> Result<()>;
    fn flush(&self) -> Result<()>;
    fn close(&self) -> Result<()>;
}

/// Trait for writing crawl failures to storage.
pub trait FailureWriter: Send + Sync {
    fn write_url(&self, failed: FailedURL) -> Result<()>;
    fn write_domain(&self, failed: FailedDomain) -> Result<()>;
    fn flush(&self) -> Result<()>;
    fn close(&self) -> Result<()>;
}
