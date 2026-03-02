use super::{FailureWriter, ResultWriter};
use crate::types::{CrawlResult, FailedDomain, FailedURL};
use anyhow::Result;

/// No-op result writer. Discards all results (for benchmarking baseline).
pub struct DevNullResultWriter;

/// No-op failure writer. Discards all failures (for benchmarking baseline).
pub struct DevNullFailureWriter;

impl ResultWriter for DevNullResultWriter {
    fn write(&self, _result: CrawlResult) -> Result<()> {
        Ok(())
    }
    fn flush(&self) -> Result<()> {
        Ok(())
    }
    fn close(&self) -> Result<()> {
        Ok(())
    }
}

impl FailureWriter for DevNullFailureWriter {
    fn write_url(&self, _failed: FailedURL) -> Result<()> {
        Ok(())
    }
    fn write_domain(&self, _failed: FailedDomain) -> Result<()> {
        Ok(())
    }
    fn flush(&self) -> Result<()> {
        Ok(())
    }
    fn close(&self) -> Result<()> {
        Ok(())
    }
}
