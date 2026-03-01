use chrono::NaiveDateTime;
use serde::{Deserialize, Serialize};

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SeedURL {
    pub url: String,
    pub domain: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CrawlResult {
    pub url: String,
    pub domain: String,
    pub status_code: u16,
    pub content_type: String,
    pub content_length: i64,
    pub title: String,
    pub description: String,
    pub language: String,
    pub redirect_url: String,
    pub fetch_time_ms: i64,
    pub crawled_at: NaiveDateTime,
    pub error: String,
    pub body: String, // always empty (DuckDB overflow block fix)
}

impl CrawlResult {
    pub fn error_result(url: &str, domain: &str, error: String, fetch_time_ms: i64) -> Self {
        Self {
            url: url.to_string(),
            domain: domain.to_string(),
            status_code: 0,
            content_type: String::new(),
            content_length: 0,
            title: String::new(),
            description: String::new(),
            language: String::new(),
            redirect_url: String::new(),
            fetch_time_ms,
            crawled_at: chrono::Utc::now().naive_utc(),
            error,
            body: String::new(),
        }
    }
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct FailedURL {
    pub url: String,
    pub domain: String,
    pub reason: String, // http_timeout, dns_timeout, domain_killed, http_error, domain_dead, domain_deadline_exceeded, domain_http_timeout_killed
    pub error: String,
    pub status_code: u16,
    pub fetch_time_ms: i64,
    pub detected_at: NaiveDateTime,
}

impl FailedURL {
    pub fn new(url: &str, domain: &str, reason: &str) -> Self {
        Self {
            url: url.to_string(),
            domain: domain.to_string(),
            reason: reason.to_string(),
            error: String::new(),
            status_code: 0,
            fetch_time_ms: 0,
            detected_at: chrono::Utc::now().naive_utc(),
        }
    }
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct FailedDomain {
    pub domain: String,
    pub reason: String,
    pub error: String,
    pub url_count: i64,
    pub detected_at: NaiveDateTime,
}
