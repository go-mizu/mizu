const std = @import("std");
const posix = std.posix;
const types = @import("types.zig");
const stats_mod = @import("stats.zig");
const results_mod = @import("results.zig");

/// High-throughput HTTP fetcher.
/// Uses std.http.Client for HTTPS (with TLS), raw TCP for plain HTTP.
/// Each worker thread does blocking I/O on one connection at a time.
pub const HttpFetcher = struct {
    domains: []types.DomainInfo,
    stats: *stats_mod.Stats,
    writer: *results_mod.ResultWriter,
    config: Config,

    pub const Config = struct {
        workers: u32 = 1024,
        timeout_ms: u32 = 5000,
        max_conns_per_domain: u8 = 8,
        fail_threshold: u16 = 2,
        status_only: bool = true,
        head_only: bool = false,
        user_agent: []const u8 = "MizuCrawler/2.0",
    };

    pub fn init(
        domains: []types.DomainInfo,
        stats: *stats_mod.Stats,
        writer: *results_mod.ResultWriter,
        config: Config,
    ) HttpFetcher {
        return .{
            .domains = domains,
            .stats = stats,
            .writer = writer,
            .config = config,
        };
    }

    /// Fetch all URLs using a thread pool. URLs are fed in round-robin domain order.
    pub fn fetchAll(self: *HttpFetcher, allocator: std.mem.Allocator, seeds: []const types.SeedUrl) void {
        // Build interleaved URL order (round-robin across domains)
        const interleaved = self.interleaveUrls(allocator, seeds) catch {
            // Fallback to sequential order
            self.fetchSequential(allocator, seeds);
            return;
        };
        defer allocator.free(interleaved);

        self.fetchSequential(allocator, interleaved);
    }

    fn fetchSequential(self: *HttpFetcher, allocator: std.mem.Allocator, seeds: []const types.SeedUrl) void {
        var work_idx = std.atomic.Value(u64).init(0);
        const total: u64 = @intCast(seeds.len);

        const actual_workers = @min(self.config.workers, @as(u32, @intCast(@max(seeds.len, 1))));

        const Context = struct {
            seeds: []const types.SeedUrl,
            domains: []types.DomainInfo,
            work_idx: *std.atomic.Value(u64),
            total: u64,
            stats: *stats_mod.Stats,
            writer: *results_mod.ResultWriter,
            config: *const Config,
            allocator: std.mem.Allocator,
        };

        var ctx = Context{
            .seeds = seeds,
            .domains = self.domains,
            .work_idx = &work_idx,
            .total = total,
            .stats = self.stats,
            .writer = self.writer,
            .config = &self.config,
            .allocator = allocator,
        };

        var threads = allocator.alloc(std.Thread, actual_workers) catch return;
        defer allocator.free(threads);

        var spawned: u32 = 0;
        for (threads) |*t| {
            t.* = std.Thread.spawn(.{ .stack_size = 512 * 1024 }, httpWorkerFn, .{&ctx}) catch continue;
            spawned += 1;
        }

        for (threads[0..spawned]) |t| {
            t.join();
        }
    }

    fn interleaveUrls(self: *HttpFetcher, allocator: std.mem.Allocator, seeds: []const types.SeedUrl) ![]types.SeedUrl {
        // Group URLs by domain, skip dead domains
        const DomainUrls = struct {
            urls: std.ArrayList(types.SeedUrl),
        };
        var domain_groups = std.AutoHashMap(u32, DomainUrls).init(allocator);
        defer {
            var iter = domain_groups.valueIterator();
            while (iter.next()) |v| {
                v.urls.deinit(allocator);
            }
            domain_groups.deinit();
        }

        for (seeds) |seed| {
            if (seed.domain_id < self.domains.len and self.domains[seed.domain_id].isDead()) {
                self.stats.recordDomainSkip(1);
                continue;
            }
            const entry = try domain_groups.getOrPut(seed.domain_id);
            if (!entry.found_existing) {
                entry.value_ptr.* = .{ .urls = .{} };
            }
            try entry.value_ptr.urls.append(allocator, seed);
        }

        // Round-robin interleave
        var result = try allocator.alloc(types.SeedUrl, seeds.len);
        var result_idx: usize = 0;

        var groups = try allocator.alloc(*DomainUrls, domain_groups.count());
        defer allocator.free(groups);
        var gi: usize = 0;
        var iter = domain_groups.valueIterator();
        while (iter.next()) |v| {
            groups[gi] = v;
            gi += 1;
        }

        var cursors = try allocator.alloc(usize, groups.len);
        defer allocator.free(cursors);
        @memset(cursors, 0);

        var remaining = groups.len;
        while (remaining > 0) {
            remaining = 0;
            for (groups, 0..) |group, idx| {
                if (cursors[idx] < group.urls.items.len) {
                    result[result_idx] = group.urls.items[cursors[idx]];
                    result_idx += 1;
                    cursors[idx] += 1;
                    if (cursors[idx] < group.urls.items.len) {
                        remaining += 1;
                    }
                }
            }
        }

        return try allocator.realloc(result, result_idx);
    }
};

fn httpWorkerFn(ctx: anytype) void {
    // Each worker creates its own http.Client for TLS support
    var client = std.http.Client{ .allocator = ctx.allocator };
    defer client.deinit();

    while (true) {
        const idx = ctx.work_idx.fetchAdd(1, .monotonic);
        if (idx >= ctx.total) return;

        const seed = ctx.seeds[@intCast(idx)];

        // Check if domain is dead
        if (seed.domain_id < ctx.domains.len) {
            const domain = &ctx.domains[seed.domain_id];
            if (domain.isDead()) {
                ctx.stats.recordDomainSkip(1);
                continue;
            }

            // Per-domain connection limit
            if (!domain.acquireConn(ctx.config.max_conns_per_domain)) {
                var retries: u8 = 0;
                while (retries < 50) : (retries += 1) {
                    std.Thread.sleep(10 * std.time.ns_per_ms);
                    if (domain.acquireConn(ctx.config.max_conns_per_domain)) break;
                    if (domain.isDead()) break;
                }
                if (retries >= 50 or domain.isDead()) {
                    ctx.stats.recordDomainSkip(1);
                    continue;
                }
            }

            defer domain.releaseConn();

            // Fetch the URL
            fetchOne(&client, seed, domain, ctx.stats, ctx.writer, ctx.config, ctx.allocator);
        }
    }
}

fn fetchOne(
    client: *std.http.Client,
    seed: types.SeedUrl,
    domain: *types.DomainInfo,
    stats: *stats_mod.Stats,
    writer: *results_mod.ResultWriter,
    config: *const HttpFetcher.Config,
    allocator: std.mem.Allocator,
) void {
    const start = std.time.nanoTimestamp();
    var result = types.FetchResult{
        .url = seed.url,
        .domain = seed.domain,
        .status_code = 0,
        .content_type = undefined,
        .content_type_len = 0,
        .content_length = -1,
        .fetch_time_ms = 0,
        .error_msg = undefined,
        .error_len = 0,
        .redirect_url = undefined,
        .redirect_len = 0,
    };
    @memset(&result.content_type, 0);
    @memset(&result.error_msg, 0);
    @memset(&result.redirect_url, 0);

    const use_tls = types.isHttps(seed.url);

    if (use_tls) {
        // Use std.http.Client for HTTPS (handles TLS)
        fetchWithStdClient(client, seed, &result, config, allocator);
    } else {
        // Use raw TCP for plain HTTP (faster, no TLS overhead)
        const port: u16 = 80;
        const sock = connectToDomain(domain, port, config.timeout_ms) catch |err| {
            recordError(&result, start, err, domain, config.fail_threshold);
            stats.recordFailure();
            writer.addResult(&result);
            return;
        };
        defer posix.close(sock);

        var req_buf: [2048]u8 = undefined;
        const method = if (config.head_only) "HEAD" else "GET";
        const path = types.extractPath(seed.url);
        const host = types.extractDomain(seed.url);

        const req_slice = std.fmt.bufPrint(&req_buf, "{s} {s} HTTP/1.1\r\nHost: {s}\r\nUser-Agent: {s}\r\nAccept: text/html,*/*;q=0.8\r\nConnection: close\r\n\r\n", .{ method, path, host, config.user_agent }) catch {
            setError(&result, "request too large");
            stats.recordFailure();
            writer.addResult(&result);
            return;
        };

        fetchPlain(sock, req_slice, &result, config.status_only) catch |err| {
            recordError(&result, start, err, domain, config.fail_threshold);
            stats.recordFailure();
            writer.addResult(&result);
            return;
        };
    }

    // Calculate fetch time
    const end = std.time.nanoTimestamp();
    result.fetch_time_ms = @intCast(@max(0, @divFloor(end - start, std.time.ns_per_ms)));

    // Record stats
    if (result.status_code >= 200 and result.status_code < 400) {
        domain.recordSuccess();
        const bytes: u64 = if (result.content_length >= 0) @intCast(result.content_length) else 0;
        stats.recordSuccess(bytes, result.fetch_time_ms);
    } else if (result.error_len > 0) {
        // Already recorded as failure
    } else {
        stats.recordFailure();
    }

    writer.addResult(&result);
}

/// Fetch URL using std.http.Client (handles TLS)
fn fetchWithStdClient(
    client: *std.http.Client,
    seed: types.SeedUrl,
    result: *types.FetchResult,
    config: *const HttpFetcher.Config,
    allocator: std.mem.Allocator,
) void {
    _ = config;
    _ = allocator;
    const fetch_result = client.fetch(.{
        .location = .{ .url = seed.url },
        .method = .GET,
        .keep_alive = false,
        .redirect_behavior = .unhandled,
    }) catch |err| {
        const err_name = @errorName(err);
        const copy_len = @min(err_name.len, result.error_msg.len);
        @memcpy(result.error_msg[0..copy_len], err_name[0..copy_len]);
        result.error_len = @intCast(copy_len);
        return;
    };

    result.status_code = @intFromEnum(fetch_result.status);
}

fn connectToDomain(domain: *const types.DomainInfo, port: u16, timeout_ms: u32) !posix.socket_t {
    const sock = try posix.socket(posix.AF.INET, posix.SOCK.STREAM, posix.IPPROTO.TCP);
    errdefer posix.close(sock);

    // Use pre-resolved IP
    var addr: posix.sockaddr.in = undefined;
    if (domain.ip_count > 0) {
        addr = .{
            .family = posix.AF.INET,
            .port = std.mem.nativeToBig(u16, port),
            .addr = @bitCast(domain.ips[0]),
            .zero = [_]u8{0} ** 8,
        };
    } else {
        return error.NoDnsRecord;
    }

    // Set connect timeout
    const tv = posix.timeval{
        .sec = @intCast(@min(timeout_ms / 2000, 2)),
        .usec = @intCast(if (timeout_ms < 2000) (timeout_ms % 1000) * 1000 else 0),
    };
    posix.setsockopt(sock, posix.SOL.SOCKET, posix.SO.RCVTIMEO, std.mem.asBytes(&tv)) catch {};
    posix.setsockopt(sock, posix.SOL.SOCKET, posix.SO.SNDTIMEO, std.mem.asBytes(&tv)) catch {};

    try posix.connect(sock, @ptrCast(&addr), @sizeOf(posix.sockaddr.in));

    return sock;
}

fn fetchPlain(sock: posix.socket_t, request: []const u8, result: *types.FetchResult, status_only: bool) !void {
    // Send request
    _ = try posix.send(sock, request, 0);

    // Receive response
    var resp_buf: [8192]u8 = undefined;
    const n = try posix.recv(sock, &resp_buf, 0);
    if (n == 0) return error.ConnectionClosed;

    parseHttpResponse(resp_buf[0..n], result, status_only);
}

/// Parse HTTP response status line and headers
fn parseHttpResponse(data: []const u8, result: *types.FetchResult, status_only: bool) void {
    _ = status_only;

    // Find "HTTP/1.x NNN" status line
    if (data.len < 12) return;

    // Parse status code
    if (std.mem.startsWith(u8, data, "HTTP/")) {
        // Find space after version
        var pos: usize = 5;
        while (pos < data.len and data[pos] != ' ') : (pos += 1) {}
        pos += 1; // skip space

        if (pos + 3 <= data.len) {
            result.status_code = parseU16(data[pos .. pos + 3]) orelse 0;
        }
    }

    // Parse headers for Content-Type and Content-Length
    var line_start: usize = 0;
    // Skip status line
    while (line_start < data.len) : (line_start += 1) {
        if (data[line_start] == '\n') {
            line_start += 1;
            break;
        }
    }

    while (line_start < data.len) {
        // Find end of line
        var line_end = line_start;
        while (line_end < data.len and data[line_end] != '\r' and data[line_end] != '\n') : (line_end += 1) {}

        const line = data[line_start..line_end];
        if (line.len == 0) break; // empty line = end of headers

        // Parse header
        if (std.ascii.startsWithIgnoreCase(line, "content-type:")) {
            const value = std.mem.trimLeft(u8, line["content-type:".len..], " ");
            const copy_len = @min(value.len, result.content_type.len);
            @memcpy(result.content_type[0..copy_len], value[0..copy_len]);
            result.content_type_len = @intCast(copy_len);
        } else if (std.ascii.startsWithIgnoreCase(line, "content-length:")) {
            const value = std.mem.trimLeft(u8, line["content-length:".len..], " ");
            result.content_length = std.fmt.parseInt(i64, value, 10) catch -1;
        } else if (std.ascii.startsWithIgnoreCase(line, "location:")) {
            const value = std.mem.trimLeft(u8, line["location:".len..], " ");
            const copy_len = @min(value.len, result.redirect_url.len);
            @memcpy(result.redirect_url[0..copy_len], value[0..copy_len]);
            result.redirect_len = @intCast(copy_len);
        }

        // Move to next line
        while (line_end < data.len and (data[line_end] == '\r' or data[line_end] == '\n')) : (line_end += 1) {}
        line_start = line_end;
    }
}

fn parseU16(s: []const u8) ?u16 {
    var result: u16 = 0;
    for (s) |c| {
        if (c < '0' or c > '9') return null;
        result = result * 10 + (c - '0');
    }
    return result;
}

fn recordError(result: *types.FetchResult, start: i128, err: anyerror, domain: *types.DomainInfo, threshold: u16) void {
    const end = std.time.nanoTimestamp();
    result.fetch_time_ms = @intCast(@max(0, @divFloor(end - start, std.time.ns_per_ms)));

    const err_name = @errorName(err);
    const copy_len = @min(err_name.len, result.error_msg.len);
    @memcpy(result.error_msg[0..copy_len], err_name[0..copy_len]);
    result.error_len = @intCast(copy_len);

    // Classify error for domain tracking
    switch (err) {
        error.ConnectionRefused, error.ConnectionResetByPeer => {
            domain.setStatus(.dead_http);
        },
        error.WouldBlock, error.ConnectionTimedOut => {
            domain.recordTimeout(threshold);
        },
        else => {},
    }
}

fn setError(result: *types.FetchResult, msg: []const u8) void {
    const copy_len = @min(msg.len, result.error_msg.len);
    @memcpy(result.error_msg[0..copy_len], msg[0..copy_len]);
    result.error_len = @intCast(copy_len);
}

test "parseHttpResponse" {
    var result = types.FetchResult{
        .url = "https://example.com",
        .domain = "example.com",
        .status_code = 0,
        .content_type = undefined,
        .content_type_len = 0,
        .content_length = -1,
        .fetch_time_ms = 0,
        .error_msg = undefined,
        .error_len = 0,
        .redirect_url = undefined,
        .redirect_len = 0,
    };
    @memset(&result.content_type, 0);
    @memset(&result.error_msg, 0);
    @memset(&result.redirect_url, 0);

    const response = "HTTP/1.1 200 OK\r\nContent-Type: text/html\r\nContent-Length: 1234\r\n\r\n<html>";
    parseHttpResponse(response, &result, true);

    try std.testing.expectEqual(@as(u16, 200), result.status_code);
    try std.testing.expectEqual(@as(i64, 1234), result.content_length);
    try std.testing.expectEqualStrings("text/html", result.contentTypeSlice());
}
