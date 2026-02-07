const std = @import("std");
const posix = std.posix;
const stats_mod = @import("stats.zig");

/// Terminal display renderer. Produces ANSI-formatted progress output
/// matching the Go recrawler's display format.
pub const Display = struct {
    label: []const u8,
    last_lines: u32,
    speed_history: [20]SpeedSample,
    speed_idx: u8,
    speed_count: u8,

    const SpeedSample = struct {
        time_ms: u64,
        fetched: u64,
        done: u64,
        bytes: u64,
    };

    pub fn init(label: []const u8) Display {
        return .{
            .label = label,
            .last_lines = 0,
            .speed_history = std.mem.zeroes([20]SpeedSample),
            .speed_idx = 0,
            .speed_count = 0,
        };
    }

    /// Render the current stats to stderr with ANSI cursor control.
    pub fn render(self: *Display, stats: *const stats_mod.Stats) void {
        const fd = posix.STDERR_FILENO;
        var buf: [4096]u8 = undefined;
        var pos: usize = 0;

        // Clear previous output
        if (self.last_lines > 0) {
            const clear = std.fmt.bufPrint(buf[pos..], "\x1b[{d}A\x1b[J", .{self.last_lines}) catch return;
            pos += clear.len;
        }

        // Record speed sample
        const sample = SpeedSample{
            .time_ms = stats.elapsedMs(),
            .fetched = stats.fetched(),
            .done = stats.done(),
            .bytes = stats.bytes_recv.load(),
        };
        self.speed_history[self.speed_idx] = sample;
        self.speed_idx = (self.speed_idx + 1) % 20;
        if (self.speed_count < 20) self.speed_count += 1;

        // Calculate rolling speed (10s window)
        const rolling_speed = self.rollingSpeed();
        const rolling_bw = self.rollingBandwidth();

        const done_val = stats.done();
        const total = stats.total_urls;
        const succ = stats.success.load();
        const fail = stats.failed.load();
        const tout = stats.timeout.load();
        const skip = stats.skipped.load();
        const dskip = stats.domain_skip.load();
        const elapsed_ms = stats.elapsedMs();
        const bytes_total = stats.bytes_recv.load();
        const pct = if (total > 0) @as(f64, @floatFromInt(done_val)) / @as(f64, @floatFromInt(total)) * 100.0 else 0.0;

        // ETA
        var eta_buf: [16]u8 = undefined;
        const eta = blk: {
            if (elapsed_ms > 2000 and done_val > 0) {
                const done_speed = self.rollingDoneSpeed();
                if (done_speed > 0) {
                    const remaining = total - done_val;
                    const eta_s: u64 = @intFromFloat(@as(f64, @floatFromInt(remaining)) / done_speed);
                    break :blk formatDuration(eta_s, &eta_buf);
                }
            }
            break :blk "---";
        };

        // Elapsed
        var elapsed_buf: [16]u8 = undefined;
        const elapsed_str = formatDuration(elapsed_ms / 1000, &elapsed_buf);

        // Progress bar
        const bar_width: u32 = 40;
        const filled: u32 = @min(@as(u32, @intFromFloat(pct / 100.0 * @as(f64, @floatFromInt(bar_width)))), bar_width);

        // Print header
        const h = std.fmt.bufPrint(buf[pos..], "  Recrawl: {s}  |  {s} URLs  |  {s} domains\n", .{
            self.label,
            fmtInt(total),
            fmtInt(stats.unique_domains),
        }) catch return;
        pos += h.len;

        // Progress bar
        const bar_prefix = "  ";
        @memcpy(buf[pos .. pos + bar_prefix.len], bar_prefix);
        pos += bar_prefix.len;
        var i: u32 = 0;
        while (i < bar_width) : (i += 1) {
            if (i < filled) {
                @memcpy(buf[pos .. pos + 3], "\xe2\x96\x88"); // █
                pos += 3;
            } else {
                @memcpy(buf[pos .. pos + 3], "\xe2\x96\x91"); // ░
                pos += 3;
            }
        }
        const bar_suffix = std.fmt.bufPrint(buf[pos..], "  {d:>5.1}%  {s}/{s}\n\n", .{ pct, fmtInt(done_val), fmtInt(total) }) catch return;
        pos += bar_suffix.len;

        // Speed line
        const speed_line = std.fmt.bufPrint(buf[pos..], "  Speed   {s}/s  |  Peak {s}/s  |  {s}/s  |  Total {s}\n", .{
            fmtInt(@as(u64, @intFromFloat(rolling_speed))),
            fmtInt(@as(u64, @intFromFloat(stats.getPeakSpeed()))),
            fmtBytes(@as(u64, @intFromFloat(rolling_bw))),
            fmtBytes(bytes_total),
        }) catch return;
        pos += speed_line.len;

        const eta_line = std.fmt.bufPrint(buf[pos..], "  ETA     {s}  |  Elapsed {s}  |  Avg {d}ms/req  |  Avg {s}/s\n\n", .{
            eta,
            elapsed_str,
            @as(u64, @intFromFloat(stats.avgFetchMs())),
            fmtBytes(if (elapsed_ms > 0) bytes_total * 1000 / elapsed_ms else 0),
        }) catch return;
        pos += eta_line.len;

        // Success/failure
        const status_line = std.fmt.bufPrint(buf[pos..], "  ok {s} ({d:.1}%)  fail {s} ({d:.1}%)  timeout {s} ({d:.1}%)\n", .{
            fmtInt(succ), safePct(succ, done_val),
            fmtInt(fail), safePct(fail, done_val),
            fmtInt(tout), safePct(tout, done_val),
        }) catch return;
        pos += status_line.len;
        const skip_line = std.fmt.bufPrint(buf[pos..], "  skip {s}  domain-dead {s} ({d:.1}%)\n\n", .{
            fmtInt(skip),
            fmtInt(dskip), safePct(dskip, done_val),
        }) catch return;
        pos += skip_line.len;

        // DNS
        const dns_live = stats.dns_live.load();
        const dns_dead = stats.dns_dead.load();
        const dns_tout = stats.dns_timeout.load();
        const dns_total = dns_live + dns_dead + dns_tout;
        var dns_lines: u32 = 0;
        if (dns_total > 0) {
            const dns_line = std.fmt.bufPrint(buf[pos..], "  DNS     {s}/{s}  |  {s} live  |  {s} dead  |  {s} timeout\n", .{
                fmtInt(dns_total), fmtInt(stats.unique_domains),
                fmtInt(dns_live), fmtInt(dns_dead), fmtInt(dns_tout),
            }) catch return;
            pos += dns_line.len;
            dns_lines = 1;
        }

        // Probe
        const probe_ok = stats.probe_reachable.load();
        const probe_fail = stats.probe_unreachable.load();
        const probe_total = probe_ok + probe_fail;
        var probe_lines: u32 = 0;
        if (probe_total > 0) {
            const probe_line = std.fmt.bufPrint(buf[pos..], "  Probe   {s}/{s}  |  {s} reachable  |  {s} unreachable\n", .{
                fmtInt(probe_total), fmtInt(dns_live),
                fmtInt(probe_ok), fmtInt(probe_fail),
            }) catch return;
            pos += probe_line.len;
            probe_lines = 1;
        }

        self.last_lines = 11 + dns_lines + probe_lines;

        // Update peak speed
        @constCast(stats).updatePeakSpeed(rolling_speed);

        // Write everything in one syscall
        _ = posix.write(fd, buf[0..pos]) catch {};
    }

    fn rollingSpeed(self: *const Display) f64 {
        if (self.speed_count < 2) return 0;
        const latest = self.speed_history[if (self.speed_idx == 0) self.speed_count - 1 else self.speed_idx - 1];

        var oldest_idx: u8 = 0;
        var found = false;
        for (0..self.speed_count) |j| {
            const idx = (self.speed_idx + 20 - self.speed_count + j) % 20;
            if (latest.time_ms > self.speed_history[idx].time_ms and
                latest.time_ms - self.speed_history[idx].time_ms <= 10000)
            {
                oldest_idx = @intCast(idx);
                found = true;
                break;
            }
        }
        if (!found) return 0;

        const oldest = self.speed_history[oldest_idx];
        const dt = @as(f64, @floatFromInt(latest.time_ms - oldest.time_ms)) / 1000.0;
        if (dt <= 0) return 0;
        return @as(f64, @floatFromInt(latest.fetched - oldest.fetched)) / dt;
    }

    fn rollingDoneSpeed(self: *const Display) f64 {
        if (self.speed_count < 2) return 0;
        const latest = self.speed_history[if (self.speed_idx == 0) self.speed_count - 1 else self.speed_idx - 1];
        var oldest_idx: u8 = 0;
        var found = false;
        for (0..self.speed_count) |j| {
            const idx = (self.speed_idx + 20 - self.speed_count + j) % 20;
            if (latest.time_ms > self.speed_history[idx].time_ms and
                latest.time_ms - self.speed_history[idx].time_ms <= 10000)
            {
                oldest_idx = @intCast(idx);
                found = true;
                break;
            }
        }
        if (!found) return 0;
        const oldest = self.speed_history[oldest_idx];
        const dt = @as(f64, @floatFromInt(latest.time_ms - oldest.time_ms)) / 1000.0;
        if (dt <= 0) return 0;
        return @as(f64, @floatFromInt(latest.done - oldest.done)) / dt;
    }

    fn rollingBandwidth(self: *const Display) f64 {
        if (self.speed_count < 2) return 0;
        const latest = self.speed_history[if (self.speed_idx == 0) self.speed_count - 1 else self.speed_idx - 1];
        var oldest_idx: u8 = 0;
        var found = false;
        for (0..self.speed_count) |j| {
            const idx = (self.speed_idx + 20 - self.speed_count + j) % 20;
            if (latest.time_ms > self.speed_history[idx].time_ms and
                latest.time_ms - self.speed_history[idx].time_ms <= 10000)
            {
                oldest_idx = @intCast(idx);
                found = true;
                break;
            }
        }
        if (!found) return 0;
        const oldest = self.speed_history[oldest_idx];
        const dt = @as(f64, @floatFromInt(latest.time_ms - oldest.time_ms)) / 1000.0;
        if (dt <= 0) return 0;
        return @as(f64, @floatFromInt(latest.bytes - oldest.bytes)) / dt;
    }
};

// ── Formatting helpers ──

const IntBuf = struct {
    buf: [32]u8 = undefined,
    len: u8 = 0,

    fn slice(self: *const IntBuf) []const u8 {
        return self.buf[0..self.len];
    }
};

fn fmtInt(n: u64) []const u8 {
    // Thread-local buffer for formatting
    const S = struct {
        threadlocal var bufs: [4]IntBuf = .{ .{}, .{}, .{}, .{} };
        threadlocal var idx: u2 = 0;
    };

    const buf_idx = S.idx;
    S.idx +%= 1;
    const buf = &S.bufs[buf_idx];

    if (n < 1000) {
        const result = std.fmt.bufPrint(&buf.buf, "{d}", .{n}) catch return "?";
        buf.len = @intCast(result.len);
        return buf.slice();
    }

    // Format with commas
    var temp: [20]u8 = undefined;
    const raw = std.fmt.bufPrint(&temp, "{d}", .{n}) catch return "?";

    var p: u8 = 0;
    for (raw, 0..) |c, i| {
        if (i > 0 and (raw.len - i) % 3 == 0) {
            buf.buf[p] = ',';
            p += 1;
        }
        buf.buf[p] = c;
        p += 1;
    }
    buf.len = p;
    return buf.slice();
}

fn fmtBytes(b: u64) []const u8 {
    const S = struct {
        threadlocal var bufs: [2][32]u8 = .{ undefined, undefined };
        threadlocal var idx: u1 = 0;
    };

    const buf_idx = S.idx;
    S.idx +%= 1;
    const buf = &S.bufs[buf_idx];

    const result = if (b < 1024)
        std.fmt.bufPrint(buf, "{d} B", .{b})
    else if (b < 1024 * 1024)
        std.fmt.bufPrint(buf, "{d:.1} KB", .{@as(f64, @floatFromInt(b)) / 1024.0})
    else if (b < 1024 * 1024 * 1024)
        std.fmt.bufPrint(buf, "{d:.1} MB", .{@as(f64, @floatFromInt(b)) / (1024.0 * 1024.0)})
    else
        std.fmt.bufPrint(buf, "{d:.2} GB", .{@as(f64, @floatFromInt(b)) / (1024.0 * 1024.0 * 1024.0)});

    if (result) |s| {
        return s;
    } else |_| {
        return "0 B";
    }
}

fn formatDuration(total_seconds: u64, buf: *[16]u8) []const u8 {
    const h = total_seconds / 3600;
    const m = (total_seconds % 3600) / 60;
    const s = total_seconds % 60;

    if (h > 0) {
        const result = std.fmt.bufPrint(buf, "{d:0>2}:{d:0>2}:{d:0>2}", .{ h, m, s }) catch return "??:??:??";
        return result;
    }
    const result = std.fmt.bufPrint(buf, "{d:0>2}:{d:0>2}", .{ m, s }) catch return "??:??";
    return result;
}

fn safePct(part: u64, total: u64) f64 {
    if (total == 0) return 0;
    return @as(f64, @floatFromInt(part)) / @as(f64, @floatFromInt(total)) * 100.0;
}
