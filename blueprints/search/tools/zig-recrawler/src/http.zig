const std = @import("std");
const posix = std.posix;
const tls = std.crypto.tls;
const types = @import("types.zig");
const stats_mod = @import("stats.zig");
const results_mod = @import("results.zig");
const faileddb_mod = @import("faileddb.zig");

/// High-throughput HTTP fetcher using raw TCP + TLS.
///
/// Architecture:
///   - Raw TCP sockets with SO_RCVTIMEO/SO_SNDTIMEO for enforced timeouts
///   - std.crypto.tls.Client over raw sockets (no std.http.Client)
///   - Pre-resolved IPs from DNS phase (skip system DNS)
///   - No CA verification (we only need status codes for recrawling)
///   - Stack-allocated buffers per worker (no heap allocation per request)
///
/// Domain death logic (matches Go recrawler):
///   - Only connection refused/reset kills domains immediately
///   - Timeouts call recordTimeout (threshold-based, immune after success)
///   - ANY valid HTTP response (even 4xx/5xx) records domain success
///   - acquireConn blocks up to 5s to match Go's blocking semaphore
pub const HttpFetcher = struct {
    domains: []types.DomainInfo,
    stats: *stats_mod.Stats,
    writer: *results_mod.ResultDB,
    failed_db: ?*faileddb_mod.FailedDB,
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
        writer: *results_mod.ResultDB,
        failed_db: ?*faileddb_mod.FailedDB,
        config: Config,
    ) HttpFetcher {
        return .{
            .domains = domains,
            .stats = stats,
            .writer = writer,
            .failed_db = failed_db,
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
            writer: *results_mod.ResultDB,
            failed_db: ?*faileddb_mod.FailedDB,
            config: *const Config,
        };

        var ctx = Context{
            .seeds = seeds,
            .domains = self.domains,
            .work_idx = &work_idx,
            .total = total,
            .stats = self.stats,
            .writer = self.writer,
            .failed_db = self.failed_db,
            .config = &self.config,
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
    // No per-worker std.http.Client — we use raw TCP + TLS with stack buffers
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

            // Per-domain connection limit with longer wait (5s, matching Go's blocking semaphore)
            if (!domain.acquireConn(ctx.config.max_conns_per_domain)) {
                var retries: u16 = 0;
                while (retries < 500) : (retries += 1) {
                    std.Thread.sleep(10 * std.time.ns_per_ms);
                    if (domain.acquireConn(ctx.config.max_conns_per_domain)) break;
                    if (domain.isDead()) break;
                }
                if (domain.isDead()) {
                    ctx.stats.recordDomainSkip(1);
                    continue;
                }
                if (retries >= 500) {
                    // acquireConn timed out - just skip, don't mark domain dead
                    ctx.stats.recordDomainSkip(1);
                    continue;
                }
            }

            defer domain.releaseConn();

            // Fetch the URL using raw TCP (+ TLS for HTTPS)
            fetchOne(seed, domain, ctx.stats, ctx.writer, ctx.failed_db, ctx.config);
        }
    }
}

fn fetchOne(
    seed: types.SeedUrl,
    domain: *types.DomainInfo,
    stats: *stats_mod.Stats,
    writer: *results_mod.ResultDB,
    failed_db: ?*faileddb_mod.FailedDB,
    config: *const HttpFetcher.Config,
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
    const port: u16 = if (use_tls) 443 else 80;

    // Step 1: Connect raw TCP socket to pre-resolved IP
    const sock = connectToDomain(domain, port, config.timeout_ms) catch |err| {
        recordError(&result, start, err, domain, config.fail_threshold);
        stats.recordFailure();
        writer.addResult(&result);
        if (failed_db) |fdb| {
            fdb.addHTTPFailedURL(seed.url, seed.domain, classifyError(err), 0, result.fetch_time_ms);
        }
        return;
    };
    defer posix.close(sock);

    // Build HTTP request
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

    if (use_tls) {
        // Step 2: TLS handshake + send/recv over raw socket
        fetchWithRawTls(sock, host, req_slice, &result, domain, config) catch |err| {
            recordError(&result, start, err, domain, config.fail_threshold);
            stats.recordFailure();
            writer.addResult(&result);
            if (failed_db) |fdb| {
                fdb.addHTTPFailedURL(seed.url, seed.domain, classifyError(err), 0, result.fetch_time_ms);
            }
            return;
        };
    } else {
        // Plain HTTP: send/recv directly on TCP socket
        fetchPlain(sock, req_slice, &result, config.status_only) catch |err| {
            recordError(&result, start, err, domain, config.fail_threshold);
            stats.recordFailure();
            writer.addResult(&result);
            if (failed_db) |fdb| {
                fdb.addHTTPFailedURL(seed.url, seed.domain, classifyError(err), 0, result.fetch_time_ms);
            }
            return;
        };
    }

    // Calculate fetch time
    const end = std.time.nanoTimestamp();
    result.fetch_time_ms = @intCast(@max(0, @divFloor(end - start, std.time.ns_per_ms)));

    // Record stats — ANY valid HTTP response proves domain alive (matches Go behavior)
    if (result.status_code > 0) {
        domain.recordSuccess();
        if (result.status_code >= 200 and result.status_code < 400) {
            const bytes: u64 = if (result.content_length >= 0) @intCast(result.content_length) else 0;
            stats.recordSuccess(bytes, result.fetch_time_ms);
        } else {
            // 4xx/5xx — domain is alive but URL failed
            stats.recordFailure();
            if (failed_db) |fdb| {
                fdb.addHTTPFailedURL(seed.url, seed.domain, "http_error", result.status_code, result.fetch_time_ms);
            }
        }
    } else if (result.error_len > 0) {
        stats.recordFailure();
        if (failed_db) |fdb| {
            fdb.addHTTPFailedURL(seed.url, seed.domain, "http_error", result.status_code, result.fetch_time_ms);
        }
    }

    writer.addResult(&result);
}

/// Fetch HTTPS URL using raw TCP socket + std.crypto.tls.Client.
/// Uses stack-allocated buffers. Socket timeouts enforced via SO_RCVTIMEO/SO_SNDTIMEO.
/// No CA verification — we only need status codes for recrawling.
fn fetchWithRawTls(
    sock: posix.socket_t,
    hostname: []const u8,
    request: []const u8,
    result: *types.FetchResult,
    domain: *types.DomainInfo,
    config: *const HttpFetcher.Config,
) !void {
    _ = domain;

    // Create std.net.Stream over raw socket for TLS
    const stream = std.net.Stream{ .handle = sock };

    // Stack-allocated buffers for I/O (min_buffer_len = 16645 bytes each)
    // 4 buffers × ~16KB = ~64KB on stack (well within 512KB stack)
    var socket_read_buf: [tls.max_ciphertext_record_len]u8 = undefined;
    var tls_write_buf: [tls.max_ciphertext_record_len]u8 = undefined;
    var tls_read_buf: [tls.max_ciphertext_record_len]u8 = undefined;
    var socket_write_buf: [tls.max_ciphertext_record_len]u8 = undefined;

    // Create buffered Reader/Writer over socket
    var stream_reader = stream.reader(&socket_read_buf);
    var stream_writer = stream.writer(&tls_write_buf);

    // TLS handshake — no CA verification, SNI set for hostname
    var tls_client = tls.Client.init(
        stream_reader.interface(),
        &stream_writer.interface,
        .{
            .host = .{ .explicit = hostname },
            .ca = .no_verification,
            .read_buffer = &tls_read_buf,
            .write_buffer = &socket_write_buf,
            .allow_truncation_attacks = true,
        },
    ) catch {
        // TLS handshake failed — don't kill domain (cert issues are per-URL)
        return error.TlsInitializationFailed;
    };

    // After TLS handshake: bump recv timeout to 3x for response waiting.
    // The connect/handshake used timeout_ms, but slow servers may need more time
    // to generate responses. Go's net/http has no read timeout, so we use 3x.
    const resp_timeout_ms = config.timeout_ms *| 3;
    const resp_tv = posix.timeval{
        .sec = @intCast(resp_timeout_ms / 1000),
        .usec = @intCast((resp_timeout_ms % 1000) * 1000),
    };
    posix.setsockopt(sock, posix.SOL.SOCKET, posix.SO.RCVTIMEO, std.mem.asBytes(&resp_tv)) catch {};

    // Send HTTP request over TLS
    tls_client.writer.writeAll(request) catch {
        return error.WriteFailed;
    };
    // Flush TLS buffer → encrypts and pushes to socket writer buffer
    tls_client.writer.flush() catch {
        return error.WriteFailed;
    };
    // Flush socket writer → sends encrypted data to the actual socket
    stream_writer.interface.flush() catch {
        return error.WriteFailed;
    };

    // Read HTTP response over TLS.
    // Use a small buffer (1024) so readSliceShort fills it from a single TLS record
    // without blocking for more data. HTTP status + essential headers fit in ~300 bytes.
    var resp_buf: [1024]u8 = undefined;
    const n = tls_client.reader.readSliceShort(&resp_buf) catch {
        return error.ReadFailed;
    };
    if (n == 0) return error.ConnectionClosed;

    parseHttpResponse(resp_buf[0..n], result, true);
}

/// Connect to domain's pre-resolved IP using non-blocking connect + poll.
/// SO_SNDTIMEO does NOT enforce connect timeout on macOS — we must use poll().
/// After connect, socket is set back to blocking with SO_RCVTIMEO/SO_SNDTIMEO for I/O.
fn connectToDomain(domain: *const types.DomainInfo, port: u16, timeout_ms: u32) !posix.socket_t {
    const sock = try posix.socket(posix.AF.INET, posix.SOCK.STREAM, posix.IPPROTO.TCP);
    errdefer posix.close(sock);

    if (domain.ip_count == 0) return error.NoDnsRecord;

    const addr = posix.sockaddr.in{
        .family = posix.AF.INET,
        .port = std.mem.nativeToBig(u16, port),
        .addr = @bitCast(domain.ips[0]),
        .zero = [_]u8{0} ** 8,
    };

    // Set non-blocking for connect
    const fl_flags = try posix.fcntl(sock, posix.F.GETFL, 0);
    _ = try posix.fcntl(sock, posix.F.SETFL, fl_flags | (1 << @bitOffsetOf(posix.O, "NONBLOCK")));

    // Non-blocking connect — returns WouldBlock (EINPROGRESS)
    posix.connect(sock, @ptrCast(&addr), @sizeOf(posix.sockaddr.in)) catch |err| {
        if (err != error.WouldBlock) return err;

        // Poll for connect completion with timeout
        var pfds = [1]posix.pollfd{.{
            .fd = sock,
            .events = posix.POLL.OUT,
            .revents = 0,
        }};
        const ready = posix.poll(&pfds, @intCast(timeout_ms)) catch return error.ConnectionTimedOut;
        if (ready == 0) return error.ConnectionTimedOut;

        // Check for connect error via SO_ERROR
        try posix.getsockoptError(sock);
    };

    // Set back to blocking mode for TLS/recv
    _ = posix.fcntl(sock, posix.F.SETFL, fl_flags) catch {};

    // Set I/O timeouts for subsequent operations
    const tv = posix.timeval{
        .sec = @intCast(timeout_ms / 1000),
        .usec = @intCast((timeout_ms % 1000) * 1000),
    };
    posix.setsockopt(sock, posix.SOL.SOCKET, posix.SO.RCVTIMEO, std.mem.asBytes(&tv)) catch {};
    posix.setsockopt(sock, posix.SOL.SOCKET, posix.SO.SNDTIMEO, std.mem.asBytes(&tv)) catch {};

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

    // Classify error for domain tracking (conservative, matching Go)
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

fn classifyError(err: anyerror) []const u8 {
    return switch (err) {
        error.ConnectionRefused => "http_refused",
        error.ConnectionResetByPeer => "http_reset",
        error.WouldBlock, error.ConnectionTimedOut => "http_timeout",
        error.TlsInitializationFailed => "tls_handshake",
        error.WriteFailed => "tls_write",
        error.ReadFailed => "tls_read",
        else => "http_error",
    };
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
