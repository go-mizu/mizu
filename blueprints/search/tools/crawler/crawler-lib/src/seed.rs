use crate::types::SeedURL;
use anyhow::{Context, Result};
use chrono::NaiveDateTime;
use duckdb::Connection;

/// Load seed URLs from a DuckDB database.
/// Tries table names in order: docs, pages, urls, seeds.
pub fn load_seeds_duckdb(path: &str, limit: usize) -> Result<Vec<SeedURL>> {
    let config = duckdb::Config::default()
        .access_mode(duckdb::AccessMode::ReadOnly)?;
    let conn = Connection::open_with_flags(path, config)
        .with_context(|| format!("opening seed db: {}", path))?;

    // Discover which table holds the seed URLs.
    let table_names = ["docs", "pages", "urls", "seeds"];
    let mut table = None;
    for name in &table_names {
        let check = format!(
            "SELECT COUNT(*) FROM information_schema.tables WHERE table_name = '{}'",
            name
        );
        if let Ok(mut stmt) = conn.prepare(&check) {
            if let Ok(mut rows) = stmt.query([]) {
                if let Ok(Some(row)) = rows.next() {
                    let count: i64 = row.get(0).unwrap_or(0);
                    if count > 0 {
                        table = Some(*name);
                        break;
                    }
                }
            }
        }
    }
    let table = table.ok_or_else(|| {
        anyhow::anyhow!(
            "no recognised seed table in {}: tried {:?}",
            path,
            table_names
        )
    })?;

    let query = if limit > 0 {
        format!(
            "SELECT url, COALESCE(domain, '') as domain FROM {} LIMIT {}",
            table, limit
        )
    } else {
        format!("SELECT url, COALESCE(domain, '') as domain FROM {}", table)
    };

    let mut stmt = conn.prepare(&query)?;
    let seeds: Vec<SeedURL> = stmt
        .query_map([], |row| {
            Ok(SeedURL {
                url: row.get(0)?,
                domain: row.get(1)?,
            })
        })?
        .filter_map(|r| r.ok())
        .collect();

    Ok(seeds)
}

/// Load seed URLs from a parquet file using DuckDB's read_parquet.
pub fn load_seeds_parquet(path: &str, limit: usize) -> Result<Vec<SeedURL>> {
    let conn = Connection::open_in_memory()?;

    let escaped = path.replace('\'', "''");
    let query = if limit > 0 {
        format!(
            "SELECT url, COALESCE(domain, '') as domain FROM read_parquet('{}') LIMIT {}",
            escaped, limit
        )
    } else {
        format!(
            "SELECT url, COALESCE(domain, '') as domain FROM read_parquet('{}')",
            escaped
        )
    };

    let mut stmt = conn.prepare(&query)?;
    let seeds: Vec<SeedURL> = stmt
        .query_map([], |row| {
            Ok(SeedURL {
                url: row.get(0)?,
                domain: row.get(1)?,
            })
        })?
        .filter_map(|r| r.ok())
        .collect();

    Ok(seeds)
}

/// Load timeout URLs from failed DB for pass-2 retry.
/// Only loads URLs with reason='http_timeout' detected after `since`.
pub fn load_retry_seeds(path: &str, since: NaiveDateTime) -> Result<Vec<SeedURL>> {
    let config = duckdb::Config::default()
        .access_mode(duckdb::AccessMode::ReadOnly)?;
    let conn = Connection::open_with_flags(path, config)
        .with_context(|| format!("opening failed db for retry: {}", path))?;

    let mut stmt = conn.prepare(
        "SELECT url, COALESCE(domain, '') as domain FROM failed_urls \
         WHERE reason = 'http_timeout' AND detected_at >= ?"
    )?;

    let since_str = since.format("%Y-%m-%d %H:%M:%S").to_string();
    let seeds: Vec<SeedURL> = stmt
        .query_map([since_str], |row| {
            Ok(SeedURL {
                url: row.get(0)?,
                domain: row.get(1)?,
            })
        })?
        .filter_map(|r| r.ok())
        .collect();

    Ok(seeds)
}
