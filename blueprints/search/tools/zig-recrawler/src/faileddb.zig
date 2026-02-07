const std = @import("std");
const types = @import("types.zig");
const zuckdb = @import("zuckdb");

/// Failed domain/URL logging with DuckDB persistence via zuckdb.
///
/// Thread-safe: separate mutexes for domain and URL lists.
/// Accumulates failures in memory during crawl, writes to DuckDB on save/close.
pub const FailedDB = struct {
    domains: std.ArrayList(FailedDomain),
    urls: std.ArrayList(FailedURL),
    domain_mutex: std.Thread.Mutex,
    url_mutex: std.Thread.Mutex,
    db_path: []const u8,
    allocator: std.mem.Allocator,

    pub const FailedDomain = struct {
        domain: []const u8, // borrowed, lives for program duration
        reason: []const u8, // comptime string
        ips: [64]u8,
        ips_len: u8,
        url_count: u32,
        stage: []const u8, // comptime string
    };

    pub const FailedURL = struct {
        url: []const u8, // borrowed, lives for program duration
        domain: []const u8, // borrowed
        reason: []const u8, // comptime string
        status_code: u16,
        fetch_time_ms: u32,
    };

    pub fn init(allocator: std.mem.Allocator, db_path: []const u8) FailedDB {
        return .{
            .domains = .{},
            .urls = .{},
            .domain_mutex = .{},
            .url_mutex = .{},
            .db_path = db_path,
            .allocator = allocator,
        };
    }

    /// Record a failed domain. Thread-safe.
    pub fn addDomain(self: *FailedDB, fd: FailedDomain) void {
        self.domain_mutex.lock();
        defer self.domain_mutex.unlock();
        self.domains.append(self.allocator, fd) catch {};
    }

    /// Record a failed URL. Thread-safe.
    pub fn addURL(self: *FailedDB, fu: FailedURL) void {
        self.url_mutex.lock();
        defer self.url_mutex.unlock();
        self.urls.append(self.allocator, fu) catch {};
    }

    /// Helper: record a DNS-dead domain.
    pub fn addDNSDead(self: *FailedDB, domain: *const types.DomainInfo, reason: []const u8) void {
        var fd = FailedDomain{
            .domain = domain.name,
            .reason = reason,
            .ips = undefined,
            .ips_len = 0,
            .url_count = domain.url_count,
            .stage = "dns_batch",
        };
        @memset(&fd.ips, 0);

        // Format IPs if any
        if (domain.ip_count > 0) {
            var ip_buf: [64]u8 = undefined;
            var ip_pos: u8 = 0;
            for (0..domain.ip_count) |i| {
                if (i > 0 and ip_pos < 63) {
                    ip_buf[ip_pos] = ',';
                    ip_pos += 1;
                }
                const ip_str = std.fmt.bufPrint(ip_buf[ip_pos..], "{d}.{d}.{d}.{d}", .{
                    domain.ips[i][0], domain.ips[i][1], domain.ips[i][2], domain.ips[i][3],
                }) catch break;
                ip_pos += @intCast(ip_str.len);
            }
            fd.ips_len = ip_pos;
            @memcpy(fd.ips[0..ip_pos], ip_buf[0..ip_pos]);
        }

        self.addDomain(fd);
    }

    /// Helper: record a domain killed by HTTP timeouts.
    pub fn addHTTPDead(self: *FailedDB, domain: *const types.DomainInfo, reason: []const u8) void {
        var fd = FailedDomain{
            .domain = domain.name,
            .reason = reason,
            .ips = undefined,
            .ips_len = 0,
            .url_count = domain.url_count,
            .stage = "http_worker",
        };
        @memset(&fd.ips, 0);
        self.addDomain(fd);
    }

    /// Helper: record a failed URL from HTTP fetch.
    pub fn addHTTPFailedURL(self: *FailedDB, url: []const u8, domain: []const u8, reason: []const u8, status_code: u16, fetch_time_ms: u32) void {
        self.addURL(.{
            .url = url,
            .domain = domain,
            .reason = reason,
            .status_code = status_code,
            .fetch_time_ms = fetch_time_ms,
        });
    }

    /// Save all accumulated failures to DuckDB via zuckdb.
    pub fn save(self: *FailedDB) void {
        const db = zuckdb.DB.init(self.allocator, self.db_path, .{}) catch return;
        defer db.deinit();
        var conn = db.conn() catch return;
        defer conn.deinit();

        // Create tables (drop + recreate for clean state)
        _ = conn.exec("DROP TABLE IF EXISTS failed_domains", .{}) catch {};
        _ = conn.exec("DROP TABLE IF EXISTS failed_urls", .{}) catch {};
        _ = conn.exec(
            \\CREATE TABLE failed_domains (
            \\  domain VARCHAR,
            \\  reason VARCHAR,
            \\  ips VARCHAR,
            \\  url_count INTEGER,
            \\  stage VARCHAR
            \\)
        , .{}) catch return;
        _ = conn.exec(
            \\CREATE TABLE failed_urls (
            \\  url VARCHAR,
            \\  domain VARCHAR,
            \\  reason VARCHAR,
            \\  status_code INTEGER,
            \\  fetch_time_ms BIGINT
            \\)
        , .{}) catch return;

        // Append domains
        self.domain_mutex.lock();
        if (self.domains.items.len > 0) {
            var appender = conn.appender(null, "failed_domains") catch {
                self.domain_mutex.unlock();
                return;
            };
            for (self.domains.items) |d| {
                appender.appendRow(.{
                    d.domain,
                    d.reason,
                    d.ips[0..d.ips_len],
                    @as(i32, @intCast(d.url_count)),
                    d.stage,
                }) catch continue;
            }
            appender.flush() catch {};
            appender.deinit();
        }
        self.domain_mutex.unlock();

        // Append URLs
        self.url_mutex.lock();
        if (self.urls.items.len > 0) {
            var appender = conn.appender(null, "failed_urls") catch {
                self.url_mutex.unlock();
                return;
            };
            for (self.urls.items) |u| {
                appender.appendRow(.{
                    u.url,
                    u.domain,
                    u.reason,
                    @as(i32, @intCast(u.status_code)),
                    @as(i64, @intCast(u.fetch_time_ms)),
                }) catch continue;
            }
            appender.flush() catch {};
            appender.deinit();
        }
        self.url_mutex.unlock();
    }

    /// Save and clean up.
    pub fn close(self: *FailedDB) void {
        self.save();
        self.domains.deinit(self.allocator);
        self.urls.deinit(self.allocator);
    }

    pub fn domainCount(self: *const FailedDB) usize {
        return self.domains.items.len;
    }

    pub fn urlCount(self: *const FailedDB) usize {
        return self.urls.items.len;
    }
};

test "FailedDB basic" {
    const allocator = std.testing.allocator;
    var fdb = FailedDB.init(allocator, "/tmp/test_failed.duckdb");
    defer {
        fdb.domains.deinit(allocator);
        fdb.urls.deinit(allocator);
    }

    fdb.addURL(.{
        .url = "https://example.com/page",
        .domain = "example.com",
        .reason = "http_timeout",
        .status_code = 0,
        .fetch_time_ms = 5000,
    });

    try std.testing.expectEqual(@as(usize, 1), fdb.urlCount());
    try std.testing.expectEqual(@as(usize, 0), fdb.domainCount());
}
