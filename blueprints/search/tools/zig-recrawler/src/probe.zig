const std = @import("std");
const posix = std.posix;
const types = @import("types.zig");
const stats_mod = @import("stats.zig");

/// Concurrent TCP probe for domain reachability.
///
/// Conservative strategy matching Go recrawler:
///   - Connection refused/reset -> dead (definitive failure)
///   - Timeout/other -> alive (may be slow but reachable)
///   - Only probes domains with DNS resolved (ip_count > 0, status == alive)
pub fn probeDomains(
    allocator: std.mem.Allocator,
    domains: []types.DomainInfo,
    stats: *stats_mod.Stats,
    worker_count: u32,
    timeout_ms: u32,
) void {
    // Count domains that need probing
    var to_probe: u64 = 0;
    for (domains) |d| {
        if (!d.isDead() and d.ip_count > 0) to_probe += 1;
    }
    if (to_probe == 0) return;

    var work_idx = std.atomic.Value(u64).init(0);
    const total: u64 = @intCast(domains.len);

    const Context = struct {
        domains: []types.DomainInfo,
        work_idx: *std.atomic.Value(u64),
        total: u64,
        stats: *stats_mod.Stats,
        timeout_ms: u32,
    };

    var ctx = Context{
        .domains = domains,
        .work_idx = &work_idx,
        .total = total,
        .stats = stats,
        .timeout_ms = timeout_ms,
    };

    const actual_workers = @min(worker_count, @as(u32, @intCast(@max(to_probe, 1))));
    var threads = allocator.alloc(std.Thread, actual_workers) catch return;
    defer allocator.free(threads);

    var spawned: u32 = 0;
    for (threads) |*t| {
        t.* = std.Thread.spawn(.{ .stack_size = 128 * 1024 }, probeWorker, .{&ctx}) catch continue;
        spawned += 1;
    }

    for (threads[0..spawned]) |t| {
        t.join();
    }
}

fn probeWorker(ctx: anytype) void {
    while (true) {
        const idx = ctx.work_idx.fetchAdd(1, .monotonic);
        if (idx >= ctx.total) return;

        const domain = &ctx.domains[@intCast(idx)];

        // Skip dead domains or domains without DNS resolution
        if (domain.isDead() or domain.ip_count == 0) continue;

        probeSingle(domain, ctx.timeout_ms, ctx.stats);
    }
}

/// Probe a single domain via TCP connect.
/// Conservative: timeout = reachable. Only connection refused/reset = dead.
fn probeSingle(domain: *types.DomainInfo, timeout_ms: u32, stats: *stats_mod.Stats) void {
    // Try port 443 first (most URLs are HTTPS), fallback to 80
    if (tcpProbe(domain, 443, timeout_ms)) {
        stats.probe_reachable.add(1);
        return;
    }

    // If port 443 failed with definitive error, try port 80
    if (domain.getStatus() == .dead_probe) {
        // Reset status for second attempt
        domain.setStatus(.alive);
        if (tcpProbe(domain, 80, timeout_ms)) {
            stats.probe_reachable.add(1);
            return;
        }
    }

    // Both failed with definitive errors
    if (domain.getStatus() == .dead_probe) {
        stats.probe_unreachable.add(1);
    } else {
        // Timeout or other non-definitive error - consider reachable
        stats.probe_reachable.add(1);
    }
}

/// Attempt TCP connect to a domain:port using non-blocking connect + poll.
/// SO_SNDTIMEO does NOT enforce connect timeout on macOS — we must use poll().
/// Returns true if connection succeeded or timed out (conservative).
/// Returns false and sets status=dead_probe if connection was refused/reset.
fn tcpProbe(domain: *types.DomainInfo, port: u16, timeout_ms: u32) bool {
    const sock = posix.socket(posix.AF.INET, posix.SOCK.STREAM, posix.IPPROTO.TCP) catch return true;
    defer posix.close(sock);

    // Set non-blocking mode for connect
    const fl_flags = posix.fcntl(sock, posix.F.GETFL, 0) catch return true;
    _ = posix.fcntl(sock, posix.F.SETFL, fl_flags | (1 << @bitOffsetOf(posix.O, "NONBLOCK"))) catch return true;

    const addr = posix.sockaddr.in{
        .family = posix.AF.INET,
        .port = std.mem.nativeToBig(u16, port),
        .addr = @bitCast(domain.ips[0]),
        .zero = [_]u8{0} ** 8,
    };

    // Non-blocking connect — returns WouldBlock (EINPROGRESS)
    posix.connect(sock, @ptrCast(&addr), @sizeOf(posix.sockaddr.in)) catch |err| {
        switch (err) {
            error.WouldBlock => {
                // Expected for non-blocking connect — wait with poll
            },
            error.ConnectionRefused => {
                domain.setStatus(.dead_probe);
                return false;
            },
            error.ConnectionResetByPeer => {
                domain.setStatus(.dead_probe);
                return false;
            },
            else => return true, // Conservative: assume reachable
        }

        // Poll for connect completion with timeout
        var pfds = [1]posix.pollfd{.{
            .fd = sock,
            .events = posix.POLL.OUT,
            .revents = 0,
        }};
        const ready = posix.poll(&pfds, @intCast(timeout_ms)) catch return true;
        if (ready == 0) return true; // Timeout = assume reachable (conservative)

        // Check for connect error via SO_ERROR
        posix.getsockoptError(sock) catch |gerr| {
            switch (gerr) {
                error.ConnectionRefused => {
                    domain.setStatus(.dead_probe);
                    return false;
                },
                error.ConnectionResetByPeer => {
                    domain.setStatus(.dead_probe);
                    return false;
                },
                else => return true,
            }
        };
    };

    return true;
}

test "probeSingle alive domain" {
    // Test with a domain that has no IPs (should be skipped)
    var domain = types.DomainInfo.init("example.com");
    var stats = stats_mod.Stats.init(1, 1);

    // No IPs - probeWorker would skip, but probeSingle would try connect
    // This test just verifies the function doesn't crash
    probeSingle(&domain, 1000, &stats);
}
