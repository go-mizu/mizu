const std = @import("std");

/// Cache-line-aligned atomic counter to prevent false sharing.
fn AlignedAtomic(comptime T: type) type {
    return struct {
        value: std.atomic.Value(T) align(64) = std.atomic.Value(T).init(0),

        const Self = @This();

        pub fn load(self: *const Self) T {
            return self.value.load(.monotonic);
        }

        pub fn add(self: *Self, val: T) void {
            _ = self.value.fetchAdd(val, .monotonic);
        }

        pub fn store(self: *Self, val: T) void {
            self.value.store(val, .release);
        }
    };
}

/// Atomic statistics counters with cache-line padding to prevent false sharing
/// across 1024+ worker threads.
pub const Stats = struct {
    success: AlignedAtomic(u64) = .{},
    failed: AlignedAtomic(u64) = .{},
    timeout: AlignedAtomic(u64) = .{},
    skipped: AlignedAtomic(u64) = .{},
    domain_skip: AlignedAtomic(u64) = .{},
    bytes_recv: AlignedAtomic(u64) = .{},
    fetch_ms_sum: AlignedAtomic(u64) = .{},

    // DNS counters
    dns_live: AlignedAtomic(u64) = .{},
    dns_dead: AlignedAtomic(u64) = .{},
    dns_timeout: AlignedAtomic(u64) = .{},

    // Probe counters
    probe_reachable: AlignedAtomic(u64) = .{},
    probe_unreachable: AlignedAtomic(u64) = .{},

    // Config (immutable after init)
    total_urls: u64 = 0,
    unique_domains: u64 = 0,
    start_time: i128 = 0, // nanosecond timestamp

    // Speed tracking
    peak_speed: AlignedAtomic(u64) = .{}, // stored as speed * 10 for 1 decimal

    // Frozen state
    frozen: bool = false,
    frozen_elapsed_ms: u64 = 0,

    pub fn init(total_urls: u64, unique_domains: u64) Stats {
        return .{
            .total_urls = total_urls,
            .unique_domains = unique_domains,
            .start_time = std.time.nanoTimestamp(),
        };
    }

    pub fn recordSuccess(self: *Stats, bytes: u64, fetch_ms: u64) void {
        self.success.add(1);
        self.bytes_recv.add(bytes);
        self.fetch_ms_sum.add(fetch_ms);
    }

    pub fn recordFailure(self: *Stats) void {
        self.failed.add(1);
    }

    pub fn recordTimeout(self: *Stats) void {
        self.timeout.add(1);
    }

    pub fn recordDomainSkip(self: *Stats, count: u64) void {
        self.domain_skip.add(count);
    }

    pub fn done(self: *const Stats) u64 {
        return self.success.load() + self.failed.load() + self.timeout.load() +
            self.skipped.load() + self.domain_skip.load();
    }

    pub fn fetched(self: *const Stats) u64 {
        return self.success.load() + self.failed.load() + self.timeout.load();
    }

    pub fn elapsedMs(self: *const Stats) u64 {
        if (self.frozen) return self.frozen_elapsed_ms;
        const now = std.time.nanoTimestamp();
        const diff = now - self.start_time;
        return @intCast(@divFloor(diff, std.time.ns_per_ms));
    }

    pub fn speed(self: *const Stats) f64 {
        const elapsed_s = @as(f64, @floatFromInt(self.elapsedMs())) / 1000.0;
        if (elapsed_s <= 0) return 0;
        return @as(f64, @floatFromInt(self.fetched())) / elapsed_s;
    }

    pub fn avgFetchMs(self: *const Stats) f64 {
        const succ = self.success.load();
        if (succ == 0) return 0;
        return @as(f64, @floatFromInt(self.fetch_ms_sum.load())) / @as(f64, @floatFromInt(succ));
    }

    pub fn freeze(self: *Stats) void {
        if (self.frozen) return;
        self.frozen = true;
        self.frozen_elapsed_ms = self.elapsedMs();
    }

    pub fn updatePeakSpeed(self: *Stats, current_speed: f64) void {
        const speed_x10: u64 = @intFromFloat(current_speed * 10.0);
        const peak = self.peak_speed.load();
        if (speed_x10 > peak) {
            self.peak_speed.store(speed_x10);
        }
    }

    pub fn getPeakSpeed(self: *const Stats) f64 {
        return @as(f64, @floatFromInt(self.peak_speed.load())) / 10.0;
    }
};

test "Stats basic" {
    var s = Stats.init(1000, 50);
    s.recordSuccess(1024, 100);
    s.recordSuccess(2048, 200);
    s.recordFailure();
    s.recordTimeout();

    try std.testing.expectEqual(@as(u64, 2), s.success.load());
    try std.testing.expectEqual(@as(u64, 1), s.failed.load());
    try std.testing.expectEqual(@as(u64, 1), s.timeout.load());
    try std.testing.expectEqual(@as(u64, 4), s.done());
    try std.testing.expectEqual(@as(u64, 4), s.fetched());
    try std.testing.expectEqual(@as(u64, 3072), s.bytes_recv.load());
}
