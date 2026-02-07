const std = @import("std");

/// Seed URL loaded from TSV file. Points into mmap'd memory (zero-copy).
pub const SeedUrl = struct {
    url: []const u8,
    domain: []const u8,
    domain_id: u32,
};

/// Domain status categories
pub const DomainStatus = enum(u8) {
    unknown = 0,
    alive = 1,
    dead_dns = 2,
    dead_probe = 3,
    dead_http = 4,
    dead_timeout = 5,
};

/// Domain info with atomic fields for concurrent access
pub const DomainInfo = struct {
    name: []const u8,
    ips: [4][4]u8, // up to 4 IPv4 addresses
    ip_count: u8,
    status: std.atomic.Value(u8),
    fail_count: std.atomic.Value(u16),
    success_count: std.atomic.Value(u16),
    active_conns: std.atomic.Value(u8),
    url_start: u32,
    url_count: u32,

    pub fn init(name: []const u8) DomainInfo {
        return .{
            .name = name,
            .ips = std.mem.zeroes([4][4]u8),
            .ip_count = 0,
            .status = std.atomic.Value(u8).init(@intFromEnum(DomainStatus.unknown)),
            .fail_count = std.atomic.Value(u16).init(0),
            .success_count = std.atomic.Value(u16).init(0),
            .active_conns = std.atomic.Value(u8).init(0),
            .url_start = 0,
            .url_count = 0,
        };
    }

    pub fn getStatus(self: *const DomainInfo) DomainStatus {
        return @enumFromInt(self.status.load(.acquire));
    }

    pub fn setStatus(self: *DomainInfo, s: DomainStatus) void {
        self.status.store(@intFromEnum(s), .release);
    }

    pub fn isDead(self: *const DomainInfo) bool {
        const s = self.getStatus();
        return s == .dead_dns or s == .dead_probe or s == .dead_http or s == .dead_timeout;
    }

    pub fn acquireConn(self: *DomainInfo, max: u8) bool {
        while (true) {
            const current = self.active_conns.load(.acquire);
            if (current >= max) return false;
            if (self.active_conns.cmpxchgWeak(current, current + 1, .acq_rel, .acquire)) |_| {
                continue;
            } else {
                return true;
            }
        }
    }

    pub fn releaseConn(self: *DomainInfo) void {
        _ = self.active_conns.fetchSub(1, .release);
    }

    pub fn recordTimeout(self: *DomainInfo, threshold: u16) void {
        if (self.success_count.load(.acquire) > 0) return; // immune
        const fails = self.fail_count.fetchAdd(1, .acq_rel) + 1;
        if (fails >= threshold) {
            self.setStatus(.dead_timeout);
        }
    }

    pub fn recordSuccess(self: *DomainInfo) void {
        _ = self.success_count.fetchAdd(1, .release);
        self.setStatus(.alive);
    }
};

/// Result of fetching a single URL
pub const FetchResult = struct {
    url: []const u8,
    domain: []const u8,
    status_code: u16,
    content_type: [128]u8,
    content_type_len: u8,
    content_length: i64,
    fetch_time_ms: u32,
    error_msg: [200]u8,
    error_len: u8,
    redirect_url: [256]u8,
    redirect_len: u16,

    pub fn contentTypeSlice(self: *const FetchResult) []const u8 {
        return self.content_type[0..self.content_type_len];
    }

    pub fn errorSlice(self: *const FetchResult) []const u8 {
        return self.error_msg[0..self.error_len];
    }

    pub fn redirectSlice(self: *const FetchResult) []const u8 {
        return self.redirect_url[0..self.redirect_len];
    }
};

/// FNV-1a hash for consistent sharding (same as Go version)
pub fn fnv1a(data: []const u8) u32 {
    var h: u32 = 2166136261;
    for (data) |byte| {
        h ^= @as(u32, byte);
        h *%= 16777619;
    }
    return h;
}

/// Extract domain from URL (zero-copy)
pub fn extractDomain(url: []const u8) []const u8 {
    // Skip scheme
    var start: usize = 0;
    if (std.mem.startsWith(u8, url, "https://")) {
        start = 8;
    } else if (std.mem.startsWith(u8, url, "http://")) {
        start = 7;
    }

    // Find end of host (first / or : or end)
    var end = start;
    while (end < url.len) : (end += 1) {
        if (url[end] == '/' or url[end] == ':' or url[end] == '?') break;
    }

    return url[start..end];
}

/// Extract path from URL (zero-copy)
pub fn extractPath(url: []const u8) []const u8 {
    // Skip scheme
    var start: usize = 0;
    if (std.mem.startsWith(u8, url, "https://")) {
        start = 8;
    } else if (std.mem.startsWith(u8, url, "http://")) {
        start = 7;
    }

    // Find start of path
    while (start < url.len) : (start += 1) {
        if (url[start] == '/') return url[start..];
    }

    return "/";
}

/// Check if URL uses HTTPS
pub fn isHttps(url: []const u8) bool {
    return std.mem.startsWith(u8, url, "https://");
}

test "extractDomain" {
    try std.testing.expectEqualStrings("example.com", extractDomain("https://example.com/path"));
    try std.testing.expectEqualStrings("example.com", extractDomain("http://example.com/"));
    try std.testing.expectEqualStrings("example.com", extractDomain("https://example.com:443/path"));
}

test "extractPath" {
    try std.testing.expectEqualStrings("/path", extractPath("https://example.com/path"));
    try std.testing.expectEqualStrings("/", extractPath("https://example.com"));
    try std.testing.expectEqualStrings("/a/b?q=1", extractPath("http://example.com/a/b?q=1"));
}

test "fnv1a" {
    const h = fnv1a("test");
    try std.testing.expect(h != 0);
}
