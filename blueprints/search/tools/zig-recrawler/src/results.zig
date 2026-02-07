const std = @import("std");
const types = @import("types.zig");
const zuckdb = @import("zuckdb");

/// Sharded result database using native DuckDB via zuckdb.
///
/// Architecture:
///   Each shard: DuckDB file + Connection + Appender + Mutex
///   addResult() -> mutex lock -> appender.appendRow() -> unlock
///   Appender auto-flushes every 2048 rows.
///   close() flushes all appenders and closes all connections.
pub const ResultDB = struct {
    shards: []Shard,
    dir_path: []const u8,
    allocator: std.mem.Allocator,
    total_written: std.atomic.Value(u64),

    const Shard = struct {
        db: zuckdb.DB,
        conn: zuckdb.Conn,
        appender: zuckdb.Appender,
        mutex: std.Thread.Mutex,
        written: u64,
    };

    pub fn init(allocator: std.mem.Allocator, dir_path: []const u8, shard_count: u32) !ResultDB {
        std.fs.cwd().makePath(dir_path) catch {};

        const shards = try allocator.alloc(Shard, shard_count);
        var initialized: u32 = 0;
        errdefer {
            for (shards[0..initialized]) |*shard| {
                shard.appender.deinit();
                shard.conn.deinit();
                shard.db.deinit();
            }
            allocator.free(shards);
        }

        for (shards, 0..) |*shard, i| {
            var name_buf: [512]u8 = undefined;
            const name = std.fmt.bufPrint(&name_buf, "{s}/results_{d:0>3}.duckdb", .{ dir_path, i }) catch return error.PathTooLong;

            shard.db = try zuckdb.DB.init(allocator, name, .{});
            errdefer shard.db.deinit();

            shard.conn = try shard.db.conn();
            errdefer shard.conn.deinit();

            // Create table (IF NOT EXISTS for resume support)
            _ = shard.conn.exec(
                \\CREATE TABLE IF NOT EXISTS results (
                \\  url VARCHAR,
                \\  status_code INTEGER,
                \\  content_type VARCHAR,
                \\  content_length BIGINT,
                \\  body_len INTEGER,
                \\  domain VARCHAR,
                \\  redirect_url VARCHAR,
                \\  fetch_time_ms BIGINT,
                \\  connect_ms INTEGER,
                \\  tls_ms INTEGER,
                \\  ttfb_ms INTEGER,
                \\  error VARCHAR
                \\)
            , .{}) catch return error.CreateTableFailed;

            shard.appender = shard.conn.appender(null, "results") catch return error.AppenderFailed;
            shard.mutex = .{};
            shard.written = 0;
            initialized += 1;
        }

        return .{
            .shards = shards,
            .dir_path = dir_path,
            .allocator = allocator,
            .total_written = std.atomic.Value(u64).init(0),
        };
    }

    /// Add a fetch result. Thread-safe via per-shard mutex.
    pub fn addResult(self: *ResultDB, result: *const types.FetchResult) void {
        const shard_idx = types.fnv1a(result.url) % @as(u32, @intCast(self.shards.len));
        const shard = &self.shards[shard_idx];

        shard.mutex.lock();
        defer shard.mutex.unlock();

        shard.appender.appendRow(.{
            result.url,
            @as(i32, @intCast(result.status_code)),
            result.contentTypeSlice(),
            result.content_length,
            @as(i32, @intCast(result.body_len)),
            result.domain,
            result.redirectSlice(),
            @as(i64, @intCast(result.fetch_time_ms)),
            @as(i32, @intCast(result.connect_ms)),
            @as(i32, @intCast(result.tls_ms)),
            @as(i32, @intCast(result.ttfb_ms)),
            result.errorSlice(),
        }) catch {
            // If append fails, try flushing and retrying
            shard.appender.flush() catch {};
            shard.appender.appendRow(.{
                result.url,
                @as(i32, @intCast(result.status_code)),
                result.contentTypeSlice(),
                result.content_length,
                @as(i32, @intCast(result.body_len)),
                result.domain,
                result.redirectSlice(),
                @as(i64, @intCast(result.fetch_time_ms)),
                @as(i32, @intCast(result.connect_ms)),
                @as(i32, @intCast(result.tls_ms)),
                @as(i32, @intCast(result.ttfb_ms)),
                result.errorSlice(),
            }) catch return;
        };

        shard.written += 1;
        _ = self.total_written.fetchAdd(1, .monotonic);
    }

    /// Flush all appender buffers to DuckDB.
    pub fn flush(self: *ResultDB) void {
        for (self.shards) |*shard| {
            shard.mutex.lock();
            shard.appender.flush() catch {};
            shard.mutex.unlock();
        }
    }

    pub fn totalWritten(self: *const ResultDB) u64 {
        return self.total_written.load(.monotonic);
    }

    /// Close: flush all appenders first (data safety), then close shards sequentially.
    /// DuckDB can SEGFAULT/SIGABRT during concurrent shard close — we flush first
    /// to ensure data is persisted, then close one at a time with error recovery.
    pub fn close(self: *ResultDB) void {
        // Phase 1: Flush all appenders (data safety — ensures everything is on disk)
        for (self.shards) |*shard| {
            shard.mutex.lock();
            shard.appender.flush() catch {};
            shard.mutex.unlock();
        }

        // Phase 2: Sequential close with small delay between shards
        for (self.shards, 0..) |*shard, i| {
            shard.appender.deinit();
            shard.conn.deinit();
            shard.db.deinit();
            // Small delay between shards to avoid concurrent DuckDB internal cleanup
            if (i + 1 < self.shards.len) {
                std.Thread.sleep(5 * std.time.ns_per_ms);
            }
        }
        self.allocator.free(self.shards);
    }
};

test "ResultDB in-memory" {
    const allocator = std.testing.allocator;

    var db = try ResultDB.init(allocator, "/tmp/zig-recrawler-test-results", 2);
    defer db.close();

    var result = types.FetchResult{
        .url = "https://example.com",
        .domain = "example.com",
        .status_code = 200,
        .content_type = undefined,
        .content_type_len = 0,
        .content_length = 1024,
        .fetch_time_ms = 150,
        .error_msg = undefined,
        .error_len = 0,
        .redirect_url = undefined,
        .redirect_len = 0,
    };
    @memset(&result.content_type, 0);
    @memset(&result.error_msg, 0);
    @memset(&result.redirect_url, 0);
    @memcpy(result.content_type[0..9], "text/html");
    result.content_type_len = 9;

    db.addResult(&result);
    try std.testing.expectEqual(@as(u64, 1), db.totalWritten());
}
