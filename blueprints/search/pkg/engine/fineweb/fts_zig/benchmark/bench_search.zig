//! Search latency benchmark
//! Measures p50, p95, p99 latencies and QPS

const std = @import("std");
const fts = @import("fts_zig");
const time = std.time;

// Use managed array list (stores allocator internally)
fn ManagedArrayList(comptime T: type) type {
    return std.array_list.AlignedManaged(T, null);
}

/// Vietnamese test queries (same as Go benchmarks)
const queries = [_][]const u8{
    // Common terms
    "Việt Nam",
    "thành phố",
    "người",
    "năm",
    // Medium frequency
    "công nghệ",
    "kinh tế",
    "giáo dục",
    // Rare terms
    "blockchain",
    "cryptocurrency",
    "metaverse",
    // Multi-word
    "công nghệ thông tin",
    "trí tuệ nhân tạo",
    "Hồ Chí Minh",
    // Numbers/Mixed
    "2024",
    "COVID-19",
    "internet",
    "AI",
};

pub fn main() !void {
    var gpa = std.heap.GeneralPurposeAllocator(.{}){};
    defer _ = gpa.deinit();
    const allocator = gpa.allocator();

    const args = try std.process.argsAlloc(allocator);
    defer std.process.argsFree(allocator, args);

    var doc_count: u32 = 100_000;
    var iterations: u32 = 10_000;
    var warmup: u32 = 1_000;

    var i: usize = 1;
    while (i < args.len) : (i += 1) {
        if (std.mem.eql(u8, args[i], "--docs") and i + 1 < args.len) {
            doc_count = std.fmt.parseInt(u32, args[i + 1], 10) catch 100_000;
            i += 1;
        } else if (std.mem.eql(u8, args[i], "--iter") and i + 1 < args.len) {
            iterations = std.fmt.parseInt(u32, args[i + 1], 10) catch 10_000;
            i += 1;
        }
    }

    std.debug.print("\n=== Search Latency Benchmark ===\n\n", .{});
    std.debug.print("Configuration:\n", .{});
    std.debug.print("  Documents: {d}\n", .{doc_count});
    std.debug.print("  Iterations: {d}\n", .{iterations});
    std.debug.print("  Warmup: {d}\n", .{warmup});
    std.debug.print("  Queries: {d}\n", .{queries.len});
    std.debug.print("\n", .{});

    // Build index
    std.debug.print("Building index...\n", .{});
    var builder = fts.profile.speed.SpeedIndexBuilder.init(allocator);

    // Generate sample docs
    var prng = std.Random.DefaultPrng.init(12345);
    const random = prng.random();

    const templates = [_][]const u8{
        "Việt Nam là quốc gia Đông Nam Á",
        "Thành phố Hồ Chí Minh là trung tâm kinh tế",
        "Công nghệ thông tin phát triển mạnh",
        "Giáo dục là quốc sách hàng đầu",
        "Trí tuệ nhân tạo đang thay đổi thế giới",
        "Internet phổ biến khắp mọi nơi",
        "Blockchain và cryptocurrency đang phát triển",
        "Du lịch Việt Nam hấp dẫn du khách",
    };

    for (0..doc_count) |_| {
        const t1 = templates[random.uintLessThan(usize, templates.len)];
        const t2 = templates[random.uintLessThan(usize, templates.len)];

        var buf: [512]u8 = undefined;
        const doc = std.fmt.bufPrint(&buf, "{s} {s}", .{ t1, t2 }) catch continue;
        _ = try builder.addDocument(doc);
    }

    var index = try builder.build();
    defer index.deinit();
    builder.deinit();

    std.debug.print("Index built: {d} docs, {d} terms\n\n", .{ index.docCount(), index.termCount() });

    // Run benchmark
    try runSearchBenchmark(allocator, &index, warmup, iterations);
}

fn runSearchBenchmark(allocator: std.mem.Allocator, index: *fts.profile.speed.SpeedIndex, warmup: u32, iterations: u32) !void {
    // Warmup
    std.debug.print("Warming up...\n", .{});
    for (0..warmup) |_| {
        for (queries) |q| {
            const results = try index.search(q, 10);
            allocator.free(results);
        }
    }

    // Collect latencies per query
    var all_latencies = ManagedArrayList(u64).init(allocator);
    defer all_latencies.deinit();

    var query_latencies: [queries.len]ManagedArrayList(u64) = undefined;
    for (&query_latencies) |*ql| {
        ql.* = ManagedArrayList(u64).init(allocator);
    }
    defer for (&query_latencies) |*ql| {
        ql.deinit();
    };

    std.debug.print("Running benchmark...\n\n", .{});

    for (0..iterations) |_| {
        for (queries, 0..) |q, qi| {
            const start = time.nanoTimestamp();
            const results = try index.search(q, 10);
            const end = time.nanoTimestamp();
            allocator.free(results);

            const latency: u64 = @intCast(end - start);
            try all_latencies.append(latency);
            try query_latencies[qi].append(latency);
        }
    }

    // Calculate and print results
    std.debug.print("Overall Results:\n", .{});
    printLatencyStats(&all_latencies);

    std.debug.print("\nPer-Query Results:\n", .{});
    std.debug.print("| Query                      | p50 (µs) | p95 (µs) | p99 (µs) |\n", .{});
    std.debug.print("|----------------------------|----------|----------|----------|\n", .{});

    for (queries, 0..) |q, qi| {
        const stats = calcStats(&query_latencies[qi]);
        std.debug.print("| {s: <26} | {d: >8.1} | {d: >8.1} | {d: >8.1} |\n", .{
            q,
            stats.p50_us,
            stats.p95_us,
            stats.p99_us,
        });
    }
}

const Stats = struct {
    p50_us: f64,
    p95_us: f64,
    p99_us: f64,
    mean_us: f64,
    qps: f64,
};

fn calcStats(latencies: *ManagedArrayList(u64)) Stats {
    if (latencies.items.len == 0) {
        return .{ .p50_us = 0, .p95_us = 0, .p99_us = 0, .mean_us = 0, .qps = 0 };
    }

    std.mem.sort(u64, latencies.items, {}, std.sort.asc(u64));

    const p50_idx = latencies.items.len / 2;
    const p95_idx = latencies.items.len * 95 / 100;
    const p99_idx = latencies.items.len * 99 / 100;

    var total: u64 = 0;
    for (latencies.items) |l| {
        total += l;
    }
    const mean = total / latencies.items.len;

    return .{
        .p50_us = @as(f64, @floatFromInt(latencies.items[p50_idx])) / 1000,
        .p95_us = @as(f64, @floatFromInt(latencies.items[p95_idx])) / 1000,
        .p99_us = @as(f64, @floatFromInt(latencies.items[p99_idx])) / 1000,
        .mean_us = @as(f64, @floatFromInt(mean)) / 1000,
        .qps = @as(f64, time.ns_per_s) / @as(f64, @floatFromInt(mean)),
    };
}

fn printLatencyStats(latencies: *ManagedArrayList(u64)) void {
    const stats = calcStats(latencies);
    std.debug.print("  p50: {d:.1} µs\n", .{stats.p50_us});
    std.debug.print("  p95: {d:.1} µs\n", .{stats.p95_us});
    std.debug.print("  p99: {d:.1} µs\n", .{stats.p99_us});
    std.debug.print("  Mean: {d:.1} µs\n", .{stats.mean_us});
    std.debug.print("  QPS: {d:.0}\n", .{stats.qps});
}
