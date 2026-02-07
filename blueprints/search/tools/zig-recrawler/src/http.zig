const std = @import("std");
const posix = std.posix;
const tls = std.crypto.tls;
const types = @import("types.zig");
const stats_mod = @import("stats.zig");
const results_mod = @import("results.zig");
const faileddb_mod = @import("faileddb.zig");

/// High-throughput HTTP fetcher using raw TCP + TLS with per-URL dispatch.
///
/// Architecture (Phase 3 — per-URL workers, no batching):
///   - URLs interleaved round-robin across domains for fair distribution
///   - Workers pick individual URLs atomically from global queue
///   - Per-domain semaphore (atomic CAS) limits concurrent connections
///   - If domain is at max conns, worker skips to next URL (non-blocking)
///   - Pre-resolved IPs from DNS phase (skip system DNS)
///   - Per-URL timing breakdown: connect_ms, tls_ms, ttfb_ms
///   - No CA verification (we only need status codes for recrawling)
///
/// Domain death logic (matches Go recrawler):
///   - Only connection refused/reset kills domains immediately
///   - Timeouts call recordTimeout (threshold-based, immune after success)
///   - ANY valid HTTP response (even 4xx/5xx) records domain success
pub const HttpFetcher = struct {
    domains: []types.DomainInfo,
    stats: *stats_mod.Stats,
    writer: *results_mod.ResultDB,
    failed_db: ?*faileddb_mod.FailedDB,
    config: Config,

    pub const Config = struct {
        workers: u32 = 1024,
        timeout_ms: u32 = 3000,
        max_conns_per_domain: u8 = 16,
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

    /// Interleave URLs across domains, then dispatch to per-URL worker threads.
    pub fn fetchAll(self: *HttpFetcher, allocator: std.mem.Allocator, seeds: []const types.SeedUrl) void {
        if (seeds.len == 0) return;

        // Build interleaved URL order (round-robin across domains)
        const url_order = interleaveUrls(allocator, seeds, self.domains) catch return;
        defer allocator.free(url_order);

        var url_idx = std.atomic.Value(u64).init(0);
        const total: u64 = @intCast(url_order.len);

        const actual_workers = @min(self.config.workers, @as(u32, @intCast(url_order.len)));

        const Context = struct {
            seeds: []const types.SeedUrl,
            domains: []types.DomainInfo,
            url_order: []const u32,
            url_idx: *std.atomic.Value(u64),
            total: u64,
            stats: *stats_mod.Stats,
            writer: *results_mod.ResultDB,
            failed_db: ?*faileddb_mod.FailedDB,
            config: *const Config,
        };

        var ctx = Context{
            .seeds = seeds,
            .domains = self.domains,
            .url_order = url_order,
            .url_idx = &url_idx,
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
            t.* = std.Thread.spawn(.{ .stack_size = 512 * 1024 }, urlWorkerFn, .{&ctx}) catch continue;
            spawned += 1;
        }

        for (threads[0..spawned]) |t| {
            t.join();
        }
    }
};

/// Interleave URLs round-robin across domains for fair distribution.
/// Input: seeds may be clustered by domain [A,A,A,B,B,C,C,C,C]
/// Output: indices interleaved [A,B,C,A,B,C,A,C,C]
fn interleaveUrls(
    allocator: std.mem.Allocator,
    seeds: []const types.SeedUrl,
    domains: []const types.DomainInfo,
) ![]u32 {
    // Group seed indices by domain_id
    const IndexList = struct { items: std.ArrayList(u32) };
    var domain_groups = std.AutoHashMap(u32, IndexList).init(allocator);
    defer {
        var iter = domain_groups.valueIterator();
        while (iter.next()) |v| {
            v.items.deinit(allocator);
        }
        domain_groups.deinit();
    }

    for (seeds, 0..) |seed, i| {
        // Skip dead domains during interleaving
        if (seed.domain_id < domains.len and domains[seed.domain_id].isDead()) {
            continue;
        }
        const entry = try domain_groups.getOrPut(seed.domain_id);
        if (!entry.found_existing) {
            entry.value_ptr.* = .{ .items = .{} };
        }
        try entry.value_ptr.items.append(allocator, @intCast(i));
    }

    // Collect domain groups into a list for round-robin
    var groups = std.ArrayList([]const u32){};
    defer groups.deinit(allocator);

    var iter = domain_groups.valueIterator();
    while (iter.next()) |v| {
        try groups.append(allocator, v.items.items);
    }

    if (groups.items.len == 0) return try allocator.alloc(u32, 0);

    // Round-robin interleave
    var result = std.ArrayList(u32){};
    errdefer result.deinit(allocator);

    var positions = try allocator.alloc(usize, groups.items.len);
    defer allocator.free(positions);
    @memset(positions, 0);

    var remaining = groups.items.len;
    while (remaining > 0) {
        for (groups.items, 0..) |group, gi| {
            if (positions[gi] < group.len) {
                try result.append(allocator, group[positions[gi]]);
                positions[gi] += 1;
                if (positions[gi] >= group.len) {
                    remaining -= 1;
                }
            }
        }
    }

    return try result.toOwnedSlice(allocator);
}

/// Per-URL worker: picks individual URLs from the interleaved queue,
/// blocks on per-domain connection slot (like Go's buffered channel semaphore),
/// fetches, and releases.
fn urlWorkerFn(ctx: anytype) void {
    while (true) {
        const idx = ctx.url_idx.fetchAdd(1, .monotonic);
        if (idx >= ctx.total) return;

        const url_i = ctx.url_order[@intCast(idx)];
        const seed = ctx.seeds[url_i];

        if (seed.domain_id >= ctx.domains.len) continue;
        const domain = &ctx.domains[seed.domain_id];

        // Check domain alive
        if (domain.isDead()) {
            ctx.stats.recordDomainSkip(1);
            continue;
        }

        // Block until a connection slot is available (like Go's buffered channel)
        // Sleep 5ms between retries — minimal CPU waste, fast response to freed slots
        while (!domain.acquireConn(ctx.config.max_conns_per_domain)) {
            if (domain.isDead()) break;
            std.Thread.sleep(5 * std.time.ns_per_ms);
        }

        // Re-check domain alive after waiting
        if (domain.isDead()) {
            ctx.stats.recordDomainSkip(1);
            continue;
        }

        // Got a slot — fetch this URL
        fetchOneUrl(seed, domain, ctx.stats, ctx.writer, ctx.failed_db, ctx.config);
        domain.releaseConn();
    }
}

/// Fetch a single URL with full timing breakdown.
fn fetchOneUrl(
    seed: types.SeedUrl,
    domain: *types.DomainInfo,
    stats: *stats_mod.Stats,
    writer: *results_mod.ResultDB,
    failed_db: ?*faileddb_mod.FailedDB,
    config: *const HttpFetcher.Config,
) void {
    var result = initResult(seed);
    const overall_start = std.time.nanoTimestamp();

    const use_tls = types.isHttps(seed.url);
    const port: u16 = if (use_tls) 443 else 80;

    // ── Phase 1: TCP Connect ──
    const connect_start = std.time.nanoTimestamp();
    const sock = connectToDomain(domain, port, config.timeout_ms) catch |err| {
        recordError(&result, overall_start, err, domain, config.fail_threshold);
        stats.recordFailure();
        writer.addResult(&result);
        if (failed_db) |fdb| {
            fdb.addHTTPFailedURL(seed.url, seed.domain, classifyError(err), 0, result.fetch_time_ms);
        }
        return;
    };
    defer posix.close(sock);
    const connect_end = std.time.nanoTimestamp();
    result.connect_ms = nanosToMs(connect_end - connect_start);

    if (use_tls) {
        fetchOneTls(sock, seed, domain, stats, writer, failed_db, config, &result, overall_start);
    } else {
        fetchOnePlain(sock, seed, domain, stats, writer, failed_db, config, &result, overall_start);
    }
}

/// Fetch a single URL over TLS with timing breakdown.
fn fetchOneTls(
    sock: posix.socket_t,
    seed: types.SeedUrl,
    domain: *types.DomainInfo,
    stats: *stats_mod.Stats,
    writer: *results_mod.ResultDB,
    failed_db: ?*faileddb_mod.FailedDB,
    config: *const HttpFetcher.Config,
    result: *types.FetchResult,
    overall_start: i128,
) void {
    const hostname = types.extractDomain(seed.url);
    const stream = std.net.Stream{ .handle = sock };

    // Stack-allocated buffers for TLS I/O (4 x 16KB = 64KB)
    var socket_read_buf: [tls.max_ciphertext_record_len]u8 = undefined;
    var tls_write_buf: [tls.max_ciphertext_record_len]u8 = undefined;
    var tls_read_buf: [tls.max_ciphertext_record_len]u8 = undefined;
    var socket_write_buf: [tls.max_ciphertext_record_len]u8 = undefined;

    var stream_reader = stream.reader(&socket_read_buf);
    var stream_writer = stream.writer(&tls_write_buf);

    // ── Phase 2: TLS Handshake ──
    const tls_start = std.time.nanoTimestamp();
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
        const tls_end = std.time.nanoTimestamp();
        result.tls_ms = nanosToMs(tls_end - tls_start);
        setError(result, "tls_handshake");
        finalizeResult(result, overall_start);
        stats.recordFailure();
        writer.addResult(result);
        if (failed_db) |fdb| {
            fdb.addHTTPFailedURL(seed.url, seed.domain, "tls_handshake", 0, result.fetch_time_ms);
        }
        return;
    };
    const tls_end = std.time.nanoTimestamp();
    result.tls_ms = nanosToMs(tls_end - tls_start);

    // Bump recv timeout for response
    const resp_timeout_ms = config.timeout_ms *| 3;
    const resp_tv = posix.timeval{
        .sec = @intCast(resp_timeout_ms / 1000),
        .usec = @intCast((resp_timeout_ms % 1000) * 1000),
    };
    posix.setsockopt(sock, posix.SOL.SOCKET, posix.SO.RCVTIMEO, std.mem.asBytes(&resp_tv)) catch {};

    // ── Phase 3: Send Request ──
    var req_buf: [2048]u8 = undefined;
    const method = if (config.head_only) "HEAD" else "GET";
    const path = types.extractPath(seed.url);
    const host = types.extractDomain(seed.url);

    const req_slice = std.fmt.bufPrint(&req_buf, "{s} {s} HTTP/1.1\r\nHost: {s}\r\nUser-Agent: {s}\r\nAccept: text/html,*/*;q=0.8\r\nConnection: close\r\n\r\n", .{ method, path, host, config.user_agent }) catch {
        setError(result, "request too large");
        finalizeResult(result, overall_start);
        stats.recordFailure();
        writer.addResult(result);
        return;
    };

    const write_ok = blk: {
        tls_client.writer.writeAll(req_slice) catch break :blk false;
        tls_client.writer.flush() catch break :blk false;
        stream_writer.interface.flush() catch break :blk false;
        break :blk true;
    };

    if (!write_ok) {
        setError(result, "tls_write");
        finalizeResult(result, overall_start);
        domain.recordTimeout(config.fail_threshold);
        stats.recordFailure();
        writer.addResult(result);
        if (failed_db) |fdb| {
            fdb.addHTTPFailedURL(seed.url, seed.domain, "tls_write", 0, result.fetch_time_ms);
        }
        return;
    }

    const req_sent_time = std.time.nanoTimestamp();

    // ── Phase 4: Read Response ──
    const resp_ok = readHttpResponse(&tls_client.reader, result, config.status_only, stats);

    // Calculate TTFB (from request sent to first response byte)
    const resp_time = std.time.nanoTimestamp();
    result.ttfb_ms = nanosToMs(resp_time - req_sent_time);

    finalizeResult(result, overall_start);

    if (!resp_ok) {
        if (result.error_len == 0) {
            setError(result, "read_failed");
        }
        classifyAndRecordError(result, domain, config.fail_threshold, stats, failed_db, seed);
        writer.addResult(result);
        return;
    }

    // Record stats
    recordResultStats(result, seed, domain, stats, writer, failed_db);
}

/// Fetch a single URL over plain HTTP with timing.
fn fetchOnePlain(
    sock: posix.socket_t,
    seed: types.SeedUrl,
    domain: *types.DomainInfo,
    stats: *stats_mod.Stats,
    writer: *results_mod.ResultDB,
    failed_db: ?*faileddb_mod.FailedDB,
    config: *const HttpFetcher.Config,
    result: *types.FetchResult,
    overall_start: i128,
) void {
    result.tls_ms = 0; // no TLS

    var req_buf: [2048]u8 = undefined;
    const method = if (config.head_only) "HEAD" else "GET";
    const path = types.extractPath(seed.url);
    const host = types.extractDomain(seed.url);

    const req_slice = std.fmt.bufPrint(&req_buf, "{s} {s} HTTP/1.1\r\nHost: {s}\r\nUser-Agent: {s}\r\nAccept: text/html,*/*;q=0.8\r\nConnection: close\r\n\r\n", .{ method, path, host, config.user_agent }) catch {
        setError(result, "request too large");
        finalizeResult(result, overall_start);
        stats.recordFailure();
        writer.addResult(result);
        return;
    };

    _ = posix.send(sock, req_slice, 0) catch {
        setError(result, "write_failed");
        finalizeResult(result, overall_start);
        domain.recordTimeout(config.fail_threshold);
        stats.recordFailure();
        writer.addResult(result);
        return;
    };

    const req_sent_time = std.time.nanoTimestamp();

    // Read response
    var plain_reader = PlainSocketReader{ .sock = sock };
    const resp_ok = readHttpResponse(&plain_reader, result, config.status_only, stats);

    const resp_time = std.time.nanoTimestamp();
    result.ttfb_ms = nanosToMs(resp_time - req_sent_time);

    finalizeResult(result, overall_start);

    if (!resp_ok) {
        if (result.error_len == 0) {
            setError(result, "read_failed");
        }
        classifyAndRecordError(result, domain, config.fail_threshold, stats, failed_db, seed);
        writer.addResult(result);
        return;
    }

    recordResultStats(result, seed, domain, stats, writer, failed_db);
}

/// Plain socket reader that provides the same interface as TLS reader.
const PlainSocketReader = struct {
    sock: posix.socket_t,

    fn readSliceShort(self: *PlainSocketReader, buf: []u8) !usize {
        const n = posix.recv(self.sock, buf, 0) catch return error.ReadFailed;
        return n;
    }

    fn readAtLeast(self: *PlainSocketReader, buf: []u8, min_bytes: usize) !usize {
        var total: usize = 0;
        while (total < min_bytes) {
            const n = posix.recv(self.sock, buf[total..], 0) catch return error.ReadFailed;
            if (n == 0) break;
            total += n;
        }
        return total;
    }
};

/// Read a full HTTP response (headers + body) for connection framing.
/// Returns true if response was successfully read, false on error.
fn readHttpResponse(reader: anytype, result: *types.FetchResult, status_only: bool, stats: *stats_mod.Stats) bool {
    // Read headers into stack buffer
    var header_buf: [4096]u8 = undefined;
    var header_len: usize = 0;
    var header_end: usize = 0;
    var found_header_end = false;

    while (header_len < header_buf.len) {
        const n = reader.readSliceShort(header_buf[header_len..]) catch return false;
        if (n == 0) {
            if (header_len == 0) return false;
            break;
        }
        header_len += n;

        if (findHeaderEnd(header_buf[0..header_len])) |end_pos| {
            header_end = end_pos;
            found_header_end = true;
            break;
        }
    }

    if (header_len == 0) return false;

    // Parse headers
    var conn_close = false;
    var chunked = false;
    var content_length: i64 = -1;
    parseHttpHeaders(header_buf[0..header_len], result, &conn_close, &chunked, &content_length);

    if (result.status_code == 0) return false;

    // Body bytes already in header_buf
    const body_in_buf: usize = if (found_header_end and header_end < header_len)
        header_len - header_end
    else
        0;

    // Drain body
    var body_bytes: u64 = body_in_buf;

    if (content_length >= 0) {
        const cl: u64 = @intCast(content_length);
        if (body_bytes < cl) {
            body_bytes += drainBytes(reader, cl - body_bytes);
        }
    } else if (chunked) {
        body_bytes += drainChunked(reader);
    } else if (conn_close or !found_header_end) {
        body_bytes += drainUntilEof(reader);
    }

    result.body_len = @intCast(@min(body_bytes, std.math.maxInt(u32)));
    if (body_bytes > 0) {
        stats.bytes_recv.add(body_bytes);
    }

    _ = status_only;

    return true;
}

/// Find the end of HTTP headers (\r\n\r\n), returns offset of body start.
fn findHeaderEnd(data: []const u8) ?usize {
    if (data.len < 4) return null;
    for (0..data.len - 3) |i| {
        if (data[i] == '\r' and data[i + 1] == '\n' and data[i + 2] == '\r' and data[i + 3] == '\n') {
            return i + 4;
        }
    }
    return null;
}

/// Parse HTTP response headers.
fn parseHttpHeaders(
    data: []const u8,
    result: *types.FetchResult,
    conn_close: *bool,
    chunked: *bool,
    content_length: *i64,
) void {
    if (data.len < 12) return;

    // Parse status line
    if (std.mem.startsWith(u8, data, "HTTP/")) {
        var pos: usize = 5;
        while (pos < data.len and data[pos] != ' ') : (pos += 1) {}
        pos += 1;
        if (pos + 3 <= data.len) {
            result.status_code = parseU16(data[pos .. pos + 3]) orelse 0;
        }
    }

    // Parse headers
    var line_start: usize = 0;
    while (line_start < data.len) : (line_start += 1) {
        if (data[line_start] == '\n') {
            line_start += 1;
            break;
        }
    }

    while (line_start < data.len) {
        var line_end = line_start;
        while (line_end < data.len and data[line_end] != '\r' and data[line_end] != '\n') : (line_end += 1) {}

        const line = data[line_start..line_end];
        if (line.len == 0) break;

        if (std.ascii.startsWithIgnoreCase(line, "content-type:")) {
            const value = std.mem.trimLeft(u8, line["content-type:".len..], " ");
            const copy_len = @min(value.len, result.content_type.len);
            @memcpy(result.content_type[0..copy_len], value[0..copy_len]);
            result.content_type_len = @intCast(copy_len);
        } else if (std.ascii.startsWithIgnoreCase(line, "content-length:")) {
            const value = std.mem.trimLeft(u8, line["content-length:".len..], " ");
            const parsed = std.fmt.parseInt(i64, value, 10) catch -1;
            content_length.* = parsed;
            result.content_length = parsed;
        } else if (std.ascii.startsWithIgnoreCase(line, "location:")) {
            const value = std.mem.trimLeft(u8, line["location:".len..], " ");
            const copy_len = @min(value.len, result.redirect_url.len);
            @memcpy(result.redirect_url[0..copy_len], value[0..copy_len]);
            result.redirect_len = @intCast(copy_len);
        } else if (std.ascii.startsWithIgnoreCase(line, "connection:")) {
            const value = std.mem.trimLeft(u8, line["connection:".len..], " ");
            if (std.ascii.startsWithIgnoreCase(value, "close")) {
                conn_close.* = true;
            }
        } else if (std.ascii.startsWithIgnoreCase(line, "transfer-encoding:")) {
            const value = std.mem.trimLeft(u8, line["transfer-encoding:".len..], " ");
            if (std.ascii.indexOfIgnoreCase(value, "chunked") != null) {
                chunked.* = true;
            }
        }

        while (line_end < data.len and (data[line_end] == '\r' or data[line_end] == '\n')) : (line_end += 1) {}
        line_start = line_end;
    }
}

/// Drain exactly `remaining` bytes from reader.
fn drainBytes(reader: anytype, remaining: u64) u64 {
    var drained: u64 = 0;
    var buf: [8192]u8 = undefined;
    while (drained < remaining) {
        const want = @min(remaining - drained, buf.len);
        const n = reader.readSliceShort(buf[0..@intCast(want)]) catch break;
        if (n == 0) break;
        drained += n;
    }
    return drained;
}

/// Drain chunked transfer encoding.
fn drainChunked(reader: anytype) u64 {
    var total: u64 = 0;
    var buf: [8192]u8 = undefined;
    while (true) {
        const n = reader.readSliceShort(&buf) catch break;
        if (n == 0) break;
        total += n;
        // Check for end marker
        if (n >= 5) {
            const tail = buf[n - 5 .. n];
            if (std.mem.eql(u8, tail, "0\r\n\r\n")) break;
        }
        if (n >= 7) {
            for (0..n -| 6) |i| {
                if (buf[i] == '\r' and buf[i + 1] == '\n' and
                    buf[i + 2] == '0' and buf[i + 3] == '\r' and
                    buf[i + 4] == '\n' and buf[i + 5] == '\r' and
                    buf[i + 6] == '\n')
                {
                    total -= (n - i - 7);
                    return total;
                }
            }
        }
    }
    return total;
}

/// Drain until EOF.
fn drainUntilEof(reader: anytype) u64 {
    var total: u64 = 0;
    var buf: [8192]u8 = undefined;
    while (true) {
        const n = reader.readSliceShort(&buf) catch break;
        if (n == 0) break;
        total += n;
    }
    return total;
}

/// Connect to domain's pre-resolved IP using non-blocking connect + poll.
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

    posix.connect(sock, @ptrCast(&addr), @sizeOf(posix.sockaddr.in)) catch |err| {
        if (err != error.WouldBlock) return err;

        var pfds = [1]posix.pollfd{.{
            .fd = sock,
            .events = posix.POLL.OUT,
            .revents = 0,
        }};
        const ready = posix.poll(&pfds, @intCast(timeout_ms)) catch return error.ConnectionTimedOut;
        if (ready == 0) return error.ConnectionTimedOut;

        try posix.getsockoptError(sock);
    };

    // Set back to blocking mode
    _ = posix.fcntl(sock, posix.F.SETFL, fl_flags) catch {};

    // Set I/O timeouts
    const tv = posix.timeval{
        .sec = @intCast(timeout_ms / 1000),
        .usec = @intCast((timeout_ms % 1000) * 1000),
    };
    posix.setsockopt(sock, posix.SOL.SOCKET, posix.SO.RCVTIMEO, std.mem.asBytes(&tv)) catch {};
    posix.setsockopt(sock, posix.SOL.SOCKET, posix.SO.SNDTIMEO, std.mem.asBytes(&tv)) catch {};

    return sock;
}

fn initResult(seed: types.SeedUrl) types.FetchResult {
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
    return result;
}

fn nanosToMs(nanos: i128) u32 {
    return @intCast(@max(0, @divFloor(nanos, std.time.ns_per_ms)));
}

fn finalizeResult(result: *types.FetchResult, start: i128) void {
    const end = std.time.nanoTimestamp();
    result.fetch_time_ms = nanosToMs(end - start);
}

fn recordError(result: *types.FetchResult, start: i128, err: anyerror, domain: *types.DomainInfo, threshold: u16) void {
    finalizeResult(result, start);

    const err_name = @errorName(err);
    const copy_len = @min(err_name.len, result.error_msg.len);
    @memcpy(result.error_msg[0..copy_len], err_name[0..copy_len]);
    result.error_len = @intCast(copy_len);

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

/// Classify error from result and record failure stats.
fn classifyAndRecordError(
    result: *types.FetchResult,
    domain: *types.DomainInfo,
    threshold: u16,
    stats: *stats_mod.Stats,
    failed_db: ?*faileddb_mod.FailedDB,
    seed: types.SeedUrl,
) void {
    if (result.status_code > 0) {
        domain.recordSuccess();
    } else {
        const err_str = result.error_msg[0..result.error_len];
        if (std.mem.eql(u8, err_str, "read_failed") or std.mem.eql(u8, err_str, "WouldBlock")) {
            domain.recordTimeout(threshold);
        }
    }
    stats.recordFailure();
    if (failed_db) |fdb| {
        fdb.addHTTPFailedURL(seed.url, seed.domain, "http_error", result.status_code, result.fetch_time_ms);
    }
}

/// Record stats for a successfully-read response.
fn recordResultStats(
    result: *types.FetchResult,
    seed: types.SeedUrl,
    domain: *types.DomainInfo,
    stats: *stats_mod.Stats,
    writer: *results_mod.ResultDB,
    failed_db: ?*faileddb_mod.FailedDB,
) void {
    if (result.status_code > 0) {
        domain.recordSuccess();
        if (result.status_code >= 200 and result.status_code < 400) {
            const bytes: u64 = if (result.content_length >= 0) @intCast(result.content_length) else 0;
            stats.recordSuccess(bytes, result.fetch_time_ms);
        } else {
            stats.recordFailure();
            if (failed_db) |fdb| {
                fdb.addHTTPFailedURL(seed.url, seed.domain, "http_error", result.status_code, result.fetch_time_ms);
            }
        }
    } else {
        stats.recordFailure();
        if (failed_db) |fdb| {
            fdb.addHTTPFailedURL(seed.url, seed.domain, "http_error", result.status_code, result.fetch_time_ms);
        }
    }

    writer.addResult(result);
}

fn parseU16(s: []const u8) ?u16 {
    var result: u16 = 0;
    for (s) |c| {
        if (c < '0' or c > '9') return null;
        result = result * 10 + (c - '0');
    }
    return result;
}

// ── Tests ──

test "parseHttpHeaders" {
    var result = initResult(.{ .url = "https://example.com", .domain = "example.com", .domain_id = 0 });

    const response = "HTTP/1.1 200 OK\r\nContent-Type: text/html\r\nContent-Length: 1234\r\nConnection: keep-alive\r\n\r\n<html>";
    var conn_close = false;
    var chunked = false;
    var content_length: i64 = -1;
    parseHttpHeaders(response, &result, &conn_close, &chunked, &content_length);

    try std.testing.expectEqual(@as(u16, 200), result.status_code);
    try std.testing.expectEqual(@as(i64, 1234), content_length);
    try std.testing.expectEqualStrings("text/html", result.contentTypeSlice());
    try std.testing.expect(!conn_close);
    try std.testing.expect(!chunked);
}

test "parseHttpHeaders connection close" {
    var result = initResult(.{ .url = "https://example.com", .domain = "example.com", .domain_id = 0 });

    const response = "HTTP/1.1 301 Moved\r\nLocation: https://www.example.com/\r\nConnection: close\r\n\r\n";
    var conn_close = false;
    var chunked = false;
    var content_length: i64 = -1;
    parseHttpHeaders(response, &result, &conn_close, &chunked, &content_length);

    try std.testing.expectEqual(@as(u16, 301), result.status_code);
    try std.testing.expect(conn_close);
    try std.testing.expectEqualStrings("https://www.example.com/", result.redirectSlice());
}

test "parseHttpHeaders chunked" {
    var result = initResult(.{ .url = "https://example.com", .domain = "example.com", .domain_id = 0 });

    const response = "HTTP/1.1 200 OK\r\nTransfer-Encoding: chunked\r\nContent-Type: text/html\r\n\r\n";
    var conn_close = false;
    var chunked = false;
    var content_length: i64 = -1;
    parseHttpHeaders(response, &result, &conn_close, &chunked, &content_length);

    try std.testing.expectEqual(@as(u16, 200), result.status_code);
    try std.testing.expect(chunked);
    try std.testing.expectEqual(@as(i64, -1), content_length);
}

test "findHeaderEnd" {
    try std.testing.expectEqual(@as(?usize, 19), findHeaderEnd("HTTP/1.1 200 OK\r\n\r\nbody"));
    try std.testing.expectEqual(@as(?usize, null), findHeaderEnd("HTTP/1.1 200 OK\r\n"));
    try std.testing.expectEqual(@as(?usize, 4), findHeaderEnd("\r\n\r\nrest"));
}

test "interleaveUrls" {
    const allocator = std.testing.allocator;

    var domains = [_]types.DomainInfo{
        types.DomainInfo.init("a.com"),
        types.DomainInfo.init("b.com"),
        types.DomainInfo.init("c.com"),
    };

    const seeds = [_]types.SeedUrl{
        .{ .url = "https://a.com/1", .domain = "a.com", .domain_id = 0 },
        .{ .url = "https://a.com/2", .domain = "a.com", .domain_id = 0 },
        .{ .url = "https://a.com/3", .domain = "a.com", .domain_id = 0 },
        .{ .url = "https://b.com/1", .domain = "b.com", .domain_id = 1 },
        .{ .url = "https://b.com/2", .domain = "b.com", .domain_id = 1 },
        .{ .url = "https://c.com/1", .domain = "c.com", .domain_id = 2 },
    };

    const order = try interleaveUrls(allocator, &seeds, &domains);
    defer allocator.free(order);

    // Should contain all 6 URLs
    try std.testing.expectEqual(@as(usize, 6), order.len);

    // Verify interleaving: first 3 should be from different domains
    var first_domains = [_]u32{ seeds[order[0]].domain_id, seeds[order[1]].domain_id, seeds[order[2]].domain_id };
    // Sort to check they're all different
    std.mem.sort(u32, &first_domains, {}, std.sort.asc(u32));
    try std.testing.expectEqual(@as(u32, 0), first_domains[0]);
    try std.testing.expectEqual(@as(u32, 1), first_domains[1]);
    try std.testing.expectEqual(@as(u32, 2), first_domains[2]);
}
