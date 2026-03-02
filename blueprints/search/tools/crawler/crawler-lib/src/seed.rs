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

/// Load seed URLs from a CC index parquet file.
/// Filters for `warc_filename IS NOT NULL` and extracts `url` + `url_host_registered_domain`.
pub fn load_seeds_cc_parquet(path: &str, limit: usize, filters: &CcSeedFilter) -> Result<Vec<SeedURL>> {
    let conn = Connection::open_in_memory()?;

    // Cap DuckDB buffer pool to avoid OOM when scanning large CC parquet files
    // (a single partition is typically 500MB gzipped / 5–15M rows).
    // Use 40% of total RAM, clamped to [512 MB, 4 GB].
    {
        use sysinfo::System;
        let sys = System::new_all();
        let total_mb = sys.total_memory() / (1024 * 1024);
        let limit_mb = ((total_mb * 40 / 100) as usize).max(512).min(4096);
        conn.execute_batch(&format!("SET memory_limit='{limit_mb}MB'"))?;
    }

    let escaped = path.replace('\'', "''");
    let mut conditions = vec!["warc_filename IS NOT NULL".to_string()];

    if !filters.status_codes.is_empty() {
        let codes: Vec<String> = filters.status_codes.iter().map(|c| c.to_string()).collect();
        conditions.push(format!("fetch_status IN ({})", codes.join(",")));
    }
    if !filters.mime_types.is_empty() {
        let quoted: Vec<String> = filters.mime_types.iter().map(|m| format!("'{}'", m.replace('\'', "''"))).collect();
        conditions.push(format!("content_mime_detected IN ({})", quoted.join(",")));
    }
    if !filters.languages.is_empty() {
        for lang in &filters.languages {
            conditions.push(format!("content_languages LIKE '%{}%'", lang.replace('\'', "''")));
        }
    }

    let where_clause = conditions.join(" AND ");
    let limit_clause = if limit > 0 { format!(" LIMIT {}", limit) } else { String::new() };

    let query = format!(
        "SELECT url, COALESCE(url_host_registered_domain, '') as domain \
         FROM read_parquet('{}') WHERE {}{}",
        escaped, where_clause, limit_clause
    );

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

/// Filters for CC seed loading.
#[derive(Default, Clone)]
pub struct CcSeedFilter {
    pub status_codes: Vec<i32>,
    pub mime_types: Vec<String>,
    pub languages: Vec<String>,
}

/// Load timeout URLs from failed DB for pass-2 retry.
/// Convert a Vec<SeedURL> into an already-closed async_channel::Receiver.
///
/// All seeds are sent immediately (bounded by seeds.len()), then the sender
/// is dropped so the receiver sees EOF. Used by HN and direct --seed callers
/// that load the full seed list upfront.
pub fn vec_to_receiver(seeds: Vec<SeedURL>) -> (async_channel::Receiver<SeedURL>, u64) {
    let total = seeds.len() as u64;
    if seeds.is_empty() {
        let (_tx, rx) = async_channel::bounded(1);
        // tx dropped immediately → rx sees empty closed channel
        return (rx, 0);
    }
    let (tx, rx) = async_channel::bounded(seeds.len());
    for seed in seeds {
        // bounded by seeds.len() — never blocks
        let _ = tx.try_send(seed);
    }
    drop(tx); // close sender → receiver returns Err(Closed) after last item
    (rx, total)
}

/// Loads URLs with reason='http_timeout' or 'domain_http_timeout_killed' detected after `since`.
pub fn load_retry_seeds(path: &str, since: NaiveDateTime) -> Result<Vec<SeedURL>> {
    let config = duckdb::Config::default()
        .access_mode(duckdb::AccessMode::ReadOnly)?;
    let conn = Connection::open_with_flags(path, config)
        .with_context(|| format!("opening failed db for retry: {}", path))?;

    let mut stmt = conn.prepare(
        "SELECT url, COALESCE(domain, '') as domain FROM failed_urls \
         WHERE reason IN ('http_timeout', 'domain_http_timeout_killed') \
           AND detected_at >= ?"
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

#[cfg(test)]
mod tests {
    use super::*;
    use chrono::NaiveDateTime;

    fn make_failed_db(path: &str) -> duckdb::Connection {
        let conn = duckdb::Connection::open(path).unwrap();
        conn.execute_batch(
            "CREATE TABLE failed_urls (
                url TEXT, domain TEXT, reason TEXT,
                subcategory TEXT, error TEXT, status_code INTEGER,
                fetch_time_ms INTEGER, detected_at TIMESTAMP
             )",
        )
        .unwrap();
        conn
    }

    fn insert_failed(conn: &duckdb::Connection, url: &str, domain: &str, reason: &str, ts: NaiveDateTime) {
        conn.execute(
            "INSERT INTO failed_urls VALUES (?,?,?,'','',0,0,?)",
            duckdb::params![url, domain, reason, ts.format("%Y-%m-%d %H:%M:%S").to_string()],
        )
        .unwrap();
    }

    #[test]
    fn vec_to_receiver_delivers_all_seeds_then_closes() {
        let seeds = vec![
            SeedURL { url: "https://a.com/1".into(), domain: "a.com".into() },
            SeedURL { url: "https://b.com/1".into(), domain: "b.com".into() },
        ];
        let (rx, count) = vec_to_receiver(seeds);
        assert_eq!(count, 2);
        assert_eq!(rx.recv_blocking().unwrap().url, "https://a.com/1");
        assert_eq!(rx.recv_blocking().unwrap().url, "https://b.com/1");
        assert!(rx.recv_blocking().is_err(), "channel should be closed after last seed");
    }

    #[test]
    fn load_retry_seeds_includes_killed_urls() {
        let dir = tempfile::tempdir().unwrap();
        let db_path = dir.path().join("failed.duckdb").to_string_lossy().to_string();
        let conn = make_failed_db(&db_path);
        let ts = chrono::Utc::now().naive_utc();

        insert_failed(&conn, "https://a.com/1", "a.com", "http_timeout", ts);
        insert_failed(&conn, "https://b.com/1", "b.com", "domain_http_timeout_killed", ts);
        drop(conn);

        let since = ts - chrono::Duration::seconds(1);
        let seeds = load_retry_seeds(&db_path, since).unwrap();

        let urls: Vec<&str> = seeds.iter().map(|s| s.url.as_str()).collect();
        assert!(urls.contains(&"https://a.com/1"), "should include http_timeout URL");
        assert!(urls.contains(&"https://b.com/1"), "should include domain_http_timeout_killed URL");
        assert_eq!(seeds.len(), 2);
    }

    #[test]
    fn load_retry_seeds_excludes_before_since() {
        let dir = tempfile::tempdir().unwrap();
        let db_path = dir.path().join("failed2.duckdb").to_string_lossy().to_string();
        let conn = make_failed_db(&db_path);
        let old_ts = chrono::NaiveDateTime::parse_from_str("2020-01-01 00:00:00", "%Y-%m-%d %H:%M:%S").unwrap();
        let new_ts = chrono::Utc::now().naive_utc();

        insert_failed(&conn, "https://old.com/1", "old.com", "http_timeout", old_ts);
        insert_failed(&conn, "https://new.com/1", "new.com", "http_timeout", new_ts);
        drop(conn);

        let since = new_ts - chrono::Duration::seconds(1);
        let seeds = load_retry_seeds(&db_path, since).unwrap();
        assert_eq!(seeds.len(), 1);
        assert_eq!(seeds[0].url, "https://new.com/1");
    }
}
