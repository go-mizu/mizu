const std = @import("std");
const posix = std.posix;
const types = @import("types.zig");
const stats_mod = @import("stats.zig");

/// High-throughput DNS resolver.
///
/// Architecture:
/// - Pre-created pool of connected UDP sockets to DNS servers
/// - Workers take a socket, send query, recv response, return socket
/// - Zero-copy DNS packet construction and parsing
/// - Multi-server: Cloudflare 1.1.1.1, Google 8.8.8.8, system fallback
pub const DnsResolver = struct {
    allocator: std.mem.Allocator,
    timeout_ms: u32,

    // Pre-created UDP socket pools (one per DNS server)
    cf_pool: SocketPool, // Cloudflare 1.1.1.1:53
    google_pool: SocketPool, // Google 8.8.8.8:53

    // Results
    resolved: std.atomic.Value(u64),
    dead: std.atomic.Value(u64),
    timed_out: std.atomic.Value(u64),
    duration_ms: u64,

    const POOL_SIZE = 256; // connections per DNS server

    pub fn init(allocator: std.mem.Allocator, timeout_ms: u32) DnsResolver {
        const timeout = if (timeout_ms == 0) 2000 else timeout_ms;
        return .{
            .allocator = allocator,
            .timeout_ms = timeout,
            .cf_pool = SocketPool.init(
                &[4]u8{ 1, 1, 1, 1 },
                53,
                POOL_SIZE,
                timeout,
            ),
            .google_pool = SocketPool.init(
                &[4]u8{ 8, 8, 8, 8 },
                53,
                POOL_SIZE,
                timeout,
            ),
            .resolved = std.atomic.Value(u64).init(0),
            .dead = std.atomic.Value(u64).init(0),
            .timed_out = std.atomic.Value(u64).init(0),
            .duration_ms = 0,
        };
    }

    pub fn deinit(self: *DnsResolver) void {
        self.cf_pool.deinit();
        self.google_pool.deinit();
    }

    /// Resolve all domains using worker threads with raw UDP DNS.
    pub fn resolveBatch(
        self: *DnsResolver,
        domains: []types.DomainInfo,
        worker_count: u32,
        stats: *stats_mod.Stats,
        progress_cb: ?*const fn (done: u64, total: u64, live: u64, dead: u64, timeout: u64) void,
    ) void {
        const start = std.time.nanoTimestamp();
        const total: u64 = @intCast(domains.len);

        if (total == 0) return;

        // Skip already-resolved domains
        var to_resolve: u64 = 0;
        for (domains) |d| {
            if (!d.isDead() and d.ip_count == 0) to_resolve += 1;
        }

        const actual_workers = @min(worker_count, @as(u32, @intCast(@max(to_resolve, 1))));

        // Shared work index
        var work_idx = std.atomic.Value(u64).init(0);

        const Context = struct {
            domains: []types.DomainInfo,
            work_idx: *std.atomic.Value(u64),
            resolver: *DnsResolver,
            total: u64,
        };

        var ctx = Context{
            .domains = domains,
            .work_idx = &work_idx,
            .resolver = self,
            .total = total,
        };

        // Spawn worker threads
        var threads = self.allocator.alloc(std.Thread, actual_workers) catch return;
        defer self.allocator.free(threads);

        var spawned: u32 = 0;
        for (threads) |*t| {
            t.* = std.Thread.spawn(.{ .stack_size = 256 * 1024 }, dnsWorkerFn, .{&ctx}) catch continue;
            spawned += 1;
        }

        // Progress reporting in main thread
        if (progress_cb) |cb| {
            while (true) {
                const done = self.resolved.load(.acquire) + self.dead.load(.acquire) + self.timed_out.load(.acquire);
                cb(done, total, self.resolved.load(.acquire), self.dead.load(.acquire), self.timed_out.load(.acquire));
                if (done >= to_resolve) break;
                std.Thread.sleep(500 * std.time.ns_per_ms);
            }
        }

        for (threads[0..spawned]) |t| {
            t.join();
        }

        stats.dns_live.store(self.resolved.load(.acquire));
        stats.dns_dead.store(self.dead.load(.acquire));
        stats.dns_timeout.store(self.timed_out.load(.acquire));

        const end = std.time.nanoTimestamp();
        self.duration_ms = @intCast(@divFloor(end - start, std.time.ns_per_ms));
    }
};

fn dnsWorkerFn(ctx: anytype) void {
    while (true) {
        const idx = ctx.work_idx.fetchAdd(1, .monotonic);
        if (idx >= ctx.total) return;

        const domain = &ctx.domains[@intCast(idx)];

        // Skip already resolved or dead
        if (domain.isDead() or domain.ip_count > 0) continue;

        // Try Cloudflare first, then Google, then system fallback
        if (resolveRawUdp(&ctx.resolver.cf_pool, domain, ctx.resolver.timeout_ms)) {
            _ = ctx.resolver.resolved.fetchAdd(1, .release);
            continue;
        }

        if (resolveRawUdp(&ctx.resolver.google_pool, domain, ctx.resolver.timeout_ms)) {
            _ = ctx.resolver.resolved.fetchAdd(1, .release);
            continue;
        }

        // System fallback via getaddrinfo
        if (resolveViaSystem(domain)) {
            _ = ctx.resolver.resolved.fetchAdd(1, .release);
            continue;
        }

        // All failed
        domain.setStatus(.dead_dns);
        _ = ctx.resolver.dead.fetchAdd(1, .release);
    }
}

/// Resolve a domain using a raw UDP DNS query through the socket pool.
fn resolveRawUdp(pool: *SocketPool, domain: *types.DomainInfo, timeout_ms: u32) bool {
    // Build DNS query packet
    var query_buf: [512]u8 = undefined;
    const query_len = buildDnsQuery(domain.name, &query_buf) catch return false;

    // Get socket from pool
    const sock = pool.acquire() orelse return false;
    defer pool.release(sock);

    // Set timeout
    const timeout_tv = posix.timeval{
        .sec = @intCast(timeout_ms / 1000),
        .usec = @intCast((timeout_ms % 1000) * 1000),
    };
    posix.setsockopt(sock, posix.SOL.SOCKET, posix.SO.RCVTIMEO, std.mem.asBytes(&timeout_tv)) catch {};
    posix.setsockopt(sock, posix.SOL.SOCKET, posix.SO.SNDTIMEO, std.mem.asBytes(&timeout_tv)) catch {};

    // Send query
    _ = posix.send(sock, query_buf[0..query_len], 0) catch return false;

    // Receive response
    var resp_buf: [512]u8 = undefined;
    const resp_len = posix.recv(sock, &resp_buf, 0) catch return false;
    if (resp_len < 12) return false;

    // Parse response
    return parseDnsResponse(resp_buf[0..resp_len], domain);
}

/// Build a DNS A query packet (RFC 1035)
fn buildDnsQuery(name: []const u8, buf: *[512]u8) !usize {
    var pos: usize = 0;

    // Transaction ID (random-ish)
    const tid = @as(u16, @truncate(@as(u128, @bitCast(std.time.nanoTimestamp()))));
    buf[0] = @intCast(tid >> 8);
    buf[1] = @intCast(tid & 0xFF);
    pos = 2;

    // Flags: standard query, recursion desired
    buf[2] = 0x01;
    buf[3] = 0x00;
    pos = 4;

    // QDCOUNT = 1
    buf[4] = 0;
    buf[5] = 1;
    // ANCOUNT, NSCOUNT, ARCOUNT = 0
    @memset(buf[6..12], 0);
    pos = 12;

    // Question section: encode domain name
    var label_start = pos;
    pos += 1; // reserve length byte

    for (name) |c| {
        if (c == '.') {
            // Write label length
            const label_len = pos - label_start - 1;
            buf[label_start] = @intCast(label_len);
            label_start = pos;
            pos += 1;
        } else {
            buf[pos] = c;
            pos += 1;
        }
    }
    // Last label
    const last_len = pos - label_start - 1;
    buf[label_start] = @intCast(last_len);

    // Terminating zero
    buf[pos] = 0;
    pos += 1;

    // QTYPE = A (1)
    buf[pos] = 0;
    buf[pos + 1] = 1;
    pos += 2;

    // QCLASS = IN (1)
    buf[pos] = 0;
    buf[pos + 1] = 1;
    pos += 2;

    return pos;
}

/// Parse a DNS response and extract A records into domain info.
fn parseDnsResponse(resp: []const u8, domain: *types.DomainInfo) bool {
    if (resp.len < 12) return false;

    // Check RCODE (bits 0-3 of byte 3)
    const rcode = resp[3] & 0x0F;
    if (rcode == 3) {
        // NXDOMAIN
        domain.setStatus(.dead_dns);
        return false;
    }
    if (rcode != 0) return false; // other error

    // ANCOUNT
    const ancount = (@as(u16, resp[6]) << 8) | @as(u16, resp[7]);
    if (ancount == 0) return false;

    // Skip question section
    var pos: usize = 12;
    // Skip QNAME
    while (pos < resp.len and resp[pos] != 0) {
        if (resp[pos] & 0xC0 == 0xC0) {
            pos += 2; // pointer
            break;
        }
        pos += @as(usize, resp[pos]) + 1;
    }
    if (pos < resp.len and resp[pos] == 0) pos += 1;
    pos += 4; // skip QTYPE + QCLASS

    // Parse answer records
    var ip_count: u8 = 0;
    var answers: u16 = 0;
    while (answers < ancount and pos + 10 < resp.len and ip_count < 4) {
        // Skip NAME (may be pointer)
        if (resp[pos] & 0xC0 == 0xC0) {
            pos += 2;
        } else {
            while (pos < resp.len and resp[pos] != 0) {
                pos += @as(usize, resp[pos]) + 1;
            }
            pos += 1;
        }

        if (pos + 10 > resp.len) break;

        const rtype = (@as(u16, resp[pos]) << 8) | @as(u16, resp[pos + 1]);
        // const rclass = (@as(u16, resp[pos + 2]) << 8) | @as(u16, resp[pos + 3]);
        // skip TTL (4 bytes)
        const rdlength = (@as(u16, resp[pos + 8]) << 8) | @as(u16, resp[pos + 9]);
        pos += 10;

        if (rtype == 1 and rdlength == 4 and pos + 4 <= resp.len) {
            // A record - extract IPv4 address
            domain.ips[ip_count] = .{ resp[pos], resp[pos + 1], resp[pos + 2], resp[pos + 3] };
            ip_count += 1;
        }

        pos += rdlength;
        answers += 1;
    }

    if (ip_count > 0) {
        domain.ip_count = ip_count;
        domain.setStatus(.alive);
        return true;
    }

    return false;
}

/// Resolve via system getaddrinfo (fallback)
fn resolveViaSystem(domain: *types.DomainInfo) bool {
    var name_buf: [256]u8 = undefined;
    if (domain.name.len >= name_buf.len) return false;
    @memcpy(name_buf[0..domain.name.len], domain.name);
    name_buf[domain.name.len] = 0;

    const hints = posix.addrinfo{
        .flags = .{},
        .family = posix.AF.INET,
        .socktype = posix.SOCK.STREAM,
        .protocol = posix.IPPROTO.TCP,
        .addrlen = 0,
        .addr = null,
        .canonname = null,
        .next = null,
    };

    const name_z: [*:0]const u8 = name_buf[0..domain.name.len :0];
    var res: ?*posix.addrinfo = null;
    switch (posix.system.getaddrinfo(name_z, null, &hints, &res)) {
        @as(posix.system.EAI, @enumFromInt(0)) => {},
        else => return false,
    }
    defer if (res) |some| posix.system.freeaddrinfo(some);

    var info: ?*posix.addrinfo = res;
    var ip_count: u8 = 0;
    while (info) |ai| : (info = ai.next) {
        if (ai.family == posix.AF.INET and ip_count < 4) {
            if (ai.addr) |addr| {
                const sa: *const posix.sockaddr.in = @ptrCast(@alignCast(addr));
                const bytes: [4]u8 = @bitCast(sa.addr);
                domain.ips[ip_count] = bytes;
                ip_count += 1;
            }
        }
    }

    if (ip_count > 0) {
        domain.ip_count = ip_count;
        domain.setStatus(.alive);
        return true;
    }

    return false;
}

/// Pre-created pool of connected UDP sockets to a DNS server.
/// Inspired by phuslu/fastdns UDPDialer: pre-dial N connections,
/// workers acquire/release from the pool for zero-overhead DNS queries.
const SocketPool = struct {
    sockets: [MAX_POOL]posix.socket_t,
    available: [MAX_POOL]std.atomic.Value(u8), // 1=available, 0=in-use
    pool_size: u32,
    server_ip: [4]u8,
    server_port: u16,

    const MAX_POOL = 256;

    fn init(ip: *const [4]u8, port: u16, pool_size: u32, timeout_ms: u32) SocketPool {
        var pool = SocketPool{
            .sockets = undefined,
            .available = undefined,
            .pool_size = @min(pool_size, MAX_POOL),
            .server_ip = ip.*,
            .server_port = port,
        };

        // Pre-create connected UDP sockets
        const actual_size = @min(pool_size, MAX_POOL);
        for (0..actual_size) |i| {
            pool.available[i] = std.atomic.Value(u8).init(1);
            pool.sockets[i] = createUdpSocket(ip, port, timeout_ms) catch {
                pool.available[i] = std.atomic.Value(u8).init(0);
                pool.sockets[i] = 0;
                continue;
            };
        }

        return pool;
    }

    fn deinit(self: *SocketPool) void {
        for (0..self.pool_size) |i| {
            if (self.sockets[i] != 0) {
                posix.close(self.sockets[i]);
            }
        }
    }

    fn acquire(self: *SocketPool) ?posix.socket_t {
        // Try to find an available socket
        for (0..self.pool_size) |i| {
            if (self.available[i].cmpxchgWeak(1, 0, .acquire, .monotonic) == null) {
                if (self.sockets[i] != 0) return self.sockets[i];
                // Socket was invalid, release and continue
                self.available[i].store(1, .release);
            }
        }
        return null;
    }

    fn release(self: *SocketPool, sock: posix.socket_t) void {
        for (0..self.pool_size) |i| {
            if (self.sockets[i] == sock) {
                self.available[i].store(1, .release);
                return;
            }
        }
    }
};

fn createUdpSocket(ip: *const [4]u8, port: u16, timeout_ms: u32) !posix.socket_t {
    const sock = try posix.socket(posix.AF.INET, posix.SOCK.DGRAM, 0);
    errdefer posix.close(sock);

    // Connect to DNS server (makes send/recv work without specifying addr)
    const addr = posix.sockaddr.in{
        .family = posix.AF.INET,
        .port = std.mem.nativeToBig(u16, port),
        .addr = @bitCast(ip.*),
        .zero = [_]u8{0} ** 8,
    };
    try posix.connect(sock, @ptrCast(&addr), @sizeOf(posix.sockaddr.in));

    // Set timeouts
    const tv = posix.timeval{
        .sec = @intCast(timeout_ms / 1000),
        .usec = @intCast((timeout_ms % 1000) * 1000),
    };
    posix.setsockopt(sock, posix.SOL.SOCKET, posix.SO.RCVTIMEO, std.mem.asBytes(&tv)) catch {};
    posix.setsockopt(sock, posix.SOL.SOCKET, posix.SO.SNDTIMEO, std.mem.asBytes(&tv)) catch {};

    return sock;
}

/// Save DNS cache to a TSV file
pub fn saveDnsCache(domains: []const types.DomainInfo, path: []const u8) !void {
    const file = try std.fs.cwd().createFile(path, .{});
    defer file.close();

    try file.writeAll("domain\tips\tstatus\n");
    for (domains) |d| {
        const s = d.getStatus();
        try file.writeAll(d.name);
        try file.writeAll("\t");

        for (0..d.ip_count) |i| {
            if (i > 0) try file.writeAll(",");
            var ip_buf: [16]u8 = undefined;
            const ip_str = std.fmt.bufPrint(&ip_buf, "{d}.{d}.{d}.{d}", .{ d.ips[i][0], d.ips[i][1], d.ips[i][2], d.ips[i][3] }) catch continue;
            try file.writeAll(ip_str);
        }

        try file.writeAll("\t");
        try file.writeAll(switch (s) {
            .alive => "alive",
            .dead_dns => "dead_dns",
            .dead_probe => "dead_probe",
            .dead_http => "dead_http",
            .dead_timeout => "dead_timeout",
            .unknown => "unknown",
        });
        try file.writeAll("\n");
    }
}

/// Load DNS cache from a TSV file
pub fn loadDnsCache(domains: []types.DomainInfo, path: []const u8) !u32 {
    const file = std.fs.cwd().openFile(path, .{}) catch return 0;
    defer file.close();

    const stat = try file.stat();
    var arena = std.heap.ArenaAllocator.init(std.heap.page_allocator);
    defer arena.deinit();
    const content = try arena.allocator().alloc(u8, stat.size);
    const bytes_read = try file.readAll(content);
    const data = content[0..bytes_read];

    var name_map = std.StringHashMap(usize).init(arena.allocator());
    for (domains, 0..) |d, i| {
        try name_map.put(d.name, i);
    }

    var loaded: u32 = 0;
    var line_iter = std.mem.splitScalar(u8, data, '\n');
    _ = line_iter.next(); // skip header

    while (line_iter.next()) |line| {
        if (line.len == 0) continue;
        var col_iter = std.mem.splitScalar(u8, line, '\t');
        const name = col_iter.next() orelse continue;
        const ips_str = col_iter.next() orelse continue;
        const status_str = col_iter.next() orelse continue;

        const idx = name_map.get(name) orelse continue;
        const domain = &domains[idx];

        if (std.mem.eql(u8, status_str, "alive")) {
            var ip_iter = std.mem.splitScalar(u8, ips_str, ',');
            var ip_count: u8 = 0;
            while (ip_iter.next()) |ip_str| {
                if (ip_count >= 4) break;
                if (parseIpv4(ip_str)) |ip_bytes| {
                    domain.ips[ip_count] = ip_bytes;
                    ip_count += 1;
                }
            }
            domain.ip_count = ip_count;
            if (ip_count > 0) domain.setStatus(.alive);
        } else if (std.mem.eql(u8, status_str, "dead_dns")) {
            domain.setStatus(.dead_dns);
        }
        loaded += 1;
    }

    return loaded;
}

fn parseIpv4(s: []const u8) ?[4]u8 {
    var parts: [4]u8 = undefined;
    var part_idx: u8 = 0;
    var current: u16 = 0;

    for (s) |c| {
        if (c == '.') {
            if (part_idx >= 3) return null;
            if (current > 255) return null;
            parts[part_idx] = @intCast(current);
            part_idx += 1;
            current = 0;
        } else if (c >= '0' and c <= '9') {
            current = current * 10 + (c - '0');
        } else {
            return null;
        }
    }
    if (part_idx != 3 or current > 255) return null;
    parts[3] = @intCast(current);
    return parts;
}

test "buildDnsQuery" {
    var buf: [512]u8 = undefined;
    const len = try buildDnsQuery("example.com", &buf);
    // Header(12) + 7("example") + 1 + 3("com") + 1 + 1(null) + 4(type+class) = 29
    try std.testing.expect(len > 12);
    // Check QDCOUNT = 1
    try std.testing.expectEqual(@as(u8, 0), buf[4]);
    try std.testing.expectEqual(@as(u8, 1), buf[5]);
}

test "parseIpv4" {
    const result = parseIpv4("1.2.3.4");
    try std.testing.expect(result != null);
    try std.testing.expectEqual([4]u8{ 1, 2, 3, 4 }, result.?);
    try std.testing.expect(parseIpv4("256.0.0.1") == null);
}
