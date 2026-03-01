pub mod reqwest_engine;

use crate::config::Config;
use crate::stats::StatsSnapshot;
use crate::types::SeedURL;
use crate::writer::{FailureWriter, ResultWriter};
use anyhow::Result;
use std::sync::Arc;

#[async_trait::async_trait]
pub trait Engine: Send + Sync {
    async fn run(
        &self,
        seeds: Vec<SeedURL>,
        cfg: &Config,
        results: Arc<dyn ResultWriter>,
        failures: Arc<dyn FailureWriter>,
    ) -> Result<StatsSnapshot>;
}
