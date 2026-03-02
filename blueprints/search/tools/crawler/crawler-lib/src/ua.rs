use std::hash::{DefaultHasher, Hash, Hasher};

/// A complete browser profile — User-Agent + all matching headers that a real
/// browser would send on a top-level navigation request.
pub struct BrowserProfile {
    pub user_agent: &'static str,
    pub accept: &'static str,
    pub accept_language: &'static str,
    pub accept_encoding: &'static str,
    /// Sec-CH-UA (Chrome/Edge only — None for Firefox/Safari).
    pub sec_ch_ua: Option<&'static str>,
    pub sec_ch_ua_mobile: Option<&'static str>,
    pub sec_ch_ua_platform: Option<&'static str>,
    /// Sec-Fetch-* headers (Chrome + Firefox — None for Safari).
    pub sec_fetch_dest: Option<&'static str>,
    pub sec_fetch_mode: Option<&'static str>,
    pub sec_fetch_site: Option<&'static str>,
    pub sec_fetch_user: Option<&'static str>,
}

pub const BROWSER_PROFILES: &[BrowserProfile] = &[
    // Profile 0: Chrome 133 / Windows
    BrowserProfile {
        user_agent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36",
        accept: "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7",
        accept_language: "en-US,en;q=0.9",
        accept_encoding: "gzip, deflate, br, zstd",
        sec_ch_ua: Some("\"Chromium\";v=\"133\", \"Not(A:Brand\";v=\"99\", \"Google Chrome\";v=\"133\""),
        sec_ch_ua_mobile: Some("?0"),
        sec_ch_ua_platform: Some("\"Windows\""),
        sec_fetch_dest: Some("document"),
        sec_fetch_mode: Some("navigate"),
        sec_fetch_site: Some("none"),
        sec_fetch_user: Some("?1"),
    },
    // Profile 1: Chrome 133 / macOS
    BrowserProfile {
        user_agent: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36",
        accept: "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7",
        accept_language: "en-US,en;q=0.9",
        accept_encoding: "gzip, deflate, br, zstd",
        sec_ch_ua: Some("\"Chromium\";v=\"133\", \"Not(A:Brand\";v=\"99\", \"Google Chrome\";v=\"133\""),
        sec_ch_ua_mobile: Some("?0"),
        sec_ch_ua_platform: Some("\"macOS\""),
        sec_fetch_dest: Some("document"),
        sec_fetch_mode: Some("navigate"),
        sec_fetch_site: Some("none"),
        sec_fetch_user: Some("?1"),
    },
    // Profile 2: Chrome 133 / Linux
    BrowserProfile {
        user_agent: "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36",
        accept: "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7",
        accept_language: "en-US,en;q=0.9",
        accept_encoding: "gzip, deflate, br, zstd",
        sec_ch_ua: Some("\"Chromium\";v=\"133\", \"Not(A:Brand\";v=\"99\", \"Google Chrome\";v=\"133\""),
        sec_ch_ua_mobile: Some("?0"),
        sec_ch_ua_platform: Some("\"Linux\""),
        sec_fetch_dest: Some("document"),
        sec_fetch_mode: Some("navigate"),
        sec_fetch_site: Some("none"),
        sec_fetch_user: Some("?1"),
    },
    // Profile 3: Firefox 133 / Windows
    BrowserProfile {
        user_agent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:133.0) Gecko/20100101 Firefox/133.0",
        accept: "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
        accept_language: "en-US,en;q=0.5",
        accept_encoding: "gzip, deflate, br, zstd",
        sec_ch_ua: None,
        sec_ch_ua_mobile: None,
        sec_ch_ua_platform: None,
        sec_fetch_dest: Some("document"),
        sec_fetch_mode: Some("navigate"),
        sec_fetch_site: Some("none"),
        sec_fetch_user: Some("?1"),
    },
    // Profile 4: Firefox 133 / macOS
    BrowserProfile {
        user_agent: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:133.0) Gecko/20100101 Firefox/133.0",
        accept: "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
        accept_language: "en-US,en;q=0.5",
        accept_encoding: "gzip, deflate, br, zstd",
        sec_ch_ua: None,
        sec_ch_ua_mobile: None,
        sec_ch_ua_platform: None,
        sec_fetch_dest: Some("document"),
        sec_fetch_mode: Some("navigate"),
        sec_fetch_site: Some("none"),
        sec_fetch_user: Some("?1"),
    },
];

/// Pick a browser profile deterministically by domain hash.
/// Same domain always gets the same profile (session consistency).
pub fn pick_profile(domain: &str) -> &'static BrowserProfile {
    let mut hasher = DefaultHasher::new();
    domain.hash(&mut hasher);
    let idx = (hasher.finish() as usize) % BROWSER_PROFILES.len();
    &BROWSER_PROFILES[idx]
}
