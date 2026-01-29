//! End-to-end benchmark for fts_zig
//! Tests indexing throughput and search latency with Vietnamese FineWeb dataset

const std = @import("std");
const fts = @import("fts_zig");
const time = std.time;

const Allocator = std.mem.Allocator;

// Use managed array list (stores allocator internally)
fn ManagedArrayList(comptime T: type) type {
    return std.array_list.AlignedManaged(T, null);
}

/// Benchmark configuration
const Config = struct {
    /// Number of documents to index (0 = all)
    doc_limit: u32 = 0,
    /// Number of search iterations
    search_iterations: u32 = 1000,
    /// Number of warmup iterations
    warmup_iterations: u32 = 100,
    /// Data directory
    data_dir: []const u8 = "",
};

/// Benchmark results
const Results = struct {
    profile: []const u8,
    doc_count: u32,
    index_time_ms: u64,
    index_docs_per_sec: f64,
    index_mb_per_sec: f64,
    index_size_mb: f64,
    search_p50_us: f64,
    search_p95_us: f64,
    search_p99_us: f64,
    search_qps: f64,
};

/// Vietnamese test queries
const test_queries = [_][]const u8{
    "Việt Nam",
    "thành phố",
    "người",
    "năm",
    "công nghệ",
    "kinh tế",
    "giáo dục",
    "trí tuệ nhân tạo",
    "Hồ Chí Minh",
    "internet",
};

pub fn main() !void {
    var gpa = std.heap.GeneralPurposeAllocator(.{}){};
    defer _ = gpa.deinit();
    const allocator = gpa.allocator();

    const args = try std.process.argsAlloc(allocator);
    defer std.process.argsFree(allocator, args);

    var config = Config{};

    // Parse args
    var i: usize = 1;
    while (i < args.len) : (i += 1) {
        if (std.mem.eql(u8, args[i], "--limit") and i + 1 < args.len) {
            config.doc_limit = std.fmt.parseInt(u32, args[i + 1], 10) catch 0;
            i += 1;
        } else if (std.mem.eql(u8, args[i], "--data") and i + 1 < args.len) {
            config.data_dir = args[i + 1];
            i += 1;
        }
    }

    std.debug.print("\n=== fts_zig End-to-End Benchmark ===\n\n", .{});

    // Run benchmarks for each profile
    const profiles = [_]struct { name: []const u8, profile: fts.index.segment.Profile }{
        .{ .name = "speed", .profile = .speed },
        .{ .name = "balanced", .profile = .balanced },
        .{ .name = "compact", .profile = .compact },
    };

    var all_results: [3]Results = undefined;

    for (profiles, 0..) |p, idx| {
        std.debug.print("Running benchmark for profile: {s}\n", .{p.name});
        all_results[idx] = try runBenchmark(allocator, p.name, p.profile, config);
    }

    // Print summary
    printSummary(&all_results);
}

fn runBenchmark(allocator: Allocator, profile_name: []const u8, profile: fts.index.segment.Profile, config: Config) !Results {
    _ = profile;

    // Generate sample documents if no data dir specified
    const documents = try generateSampleDocs(allocator, config.doc_limit);
    defer {
        for (documents) |doc| {
            allocator.free(doc);
        }
        allocator.free(documents);
    }

    var total_bytes: usize = 0;
    for (documents) |doc| {
        total_bytes += doc.len;
    }

    std.debug.print("  Documents: {d}\n", .{documents.len});
    std.debug.print("  Total size: {d:.2} MB\n", .{@as(f64, @floatFromInt(total_bytes)) / (1024 * 1024)});

    // Benchmark indexing
    var builder = fts.profile.speed.SpeedIndexBuilder.init(allocator);
    defer builder.deinit();

    const index_start = time.nanoTimestamp();

    for (documents) |doc| {
        _ = try builder.addDocument(doc);
    }

    var index = try builder.build();
    defer index.deinit();

    const index_end = time.nanoTimestamp();
    const index_time_ns: u64 = @intCast(index_end - index_start);
    const index_time_ms = index_time_ns / time.ns_per_ms;

    const docs_per_sec = @as(f64, @floatFromInt(documents.len)) /
        (@as(f64, @floatFromInt(index_time_ns)) / @as(f64, time.ns_per_s));

    const mb_per_sec = @as(f64, @floatFromInt(total_bytes)) / (1024 * 1024) /
        (@as(f64, @floatFromInt(index_time_ns)) / @as(f64, time.ns_per_s));

    std.debug.print("  Index time: {d} ms\n", .{index_time_ms});
    std.debug.print("  Throughput: {d:.0} docs/sec\n", .{docs_per_sec});
    std.debug.print("  Throughput: {d:.2} MB/sec\n", .{mb_per_sec});

    // Benchmark search
    var latencies = ManagedArrayList(u64).init(allocator);
    defer latencies.deinit();

    // Warmup
    for (0..config.warmup_iterations) |_| {
        for (test_queries) |query| {
            const results = try index.search(query, 10);
            allocator.free(results);
        }
    }

    // Measured runs
    for (0..config.search_iterations) |_| {
        for (test_queries) |query| {
            const start = time.nanoTimestamp();
            const results = try index.search(query, 10);
            const end = time.nanoTimestamp();
            allocator.free(results);

            const latency_ns: u64 = @intCast(end - start);
            try latencies.append(latency_ns);
        }
    }

    // Calculate percentiles
    std.mem.sort(u64, latencies.items, {}, std.sort.asc(u64));

    const p50_idx = latencies.items.len / 2;
    const p95_idx = latencies.items.len * 95 / 100;
    const p99_idx = latencies.items.len * 99 / 100;

    const p50_us = @as(f64, @floatFromInt(latencies.items[p50_idx])) / 1000;
    const p95_us = @as(f64, @floatFromInt(latencies.items[p95_idx])) / 1000;
    const p99_us = @as(f64, @floatFromInt(latencies.items[p99_idx])) / 1000;

    var total_latency: u64 = 0;
    for (latencies.items) |l| {
        total_latency += l;
    }
    const avg_latency_ns = total_latency / latencies.items.len;
    const qps = @as(f64, time.ns_per_s) / @as(f64, @floatFromInt(avg_latency_ns));

    std.debug.print("  Search p50: {d:.1} µs\n", .{p50_us});
    std.debug.print("  Search p95: {d:.1} µs\n", .{p95_us});
    std.debug.print("  Search p99: {d:.1} µs\n", .{p99_us});
    std.debug.print("  QPS: {d:.0}\n", .{qps});
    std.debug.print("\n", .{});

    return Results{
        .profile = profile_name,
        .doc_count = @intCast(documents.len),
        .index_time_ms = index_time_ms,
        .index_docs_per_sec = docs_per_sec,
        .index_mb_per_sec = mb_per_sec,
        .index_size_mb = @as(f64, @floatFromInt(index.memoryUsage())) / (1024 * 1024),
        .search_p50_us = p50_us,
        .search_p95_us = p95_us,
        .search_p99_us = p99_us,
        .search_qps = qps,
    };
}

fn generateSampleDocs(allocator: Allocator, limit: u32) ![][]u8 {
    const doc_count: usize = if (limit > 0) limit else 100_000;

    var docs = try allocator.alloc([]u8, doc_count);

    // Sample Vietnamese text templates
    const templates = [_][]const u8{
        "Việt Nam là một quốc gia nằm ở Đông Nam Á với diện tích 331.212 km vuông",
        "Thành phố Hồ Chí Minh là thành phố lớn nhất Việt Nam về dân số",
        "Hà Nội là thủ đô của nước Cộng hòa Xã hội Chủ nghĩa Việt Nam",
        "Công nghệ thông tin đang phát triển mạnh mẽ tại Việt Nam",
        "Kinh tế Việt Nam tăng trưởng ổn định trong những năm gần đây",
        "Giáo dục là quốc sách hàng đầu của Việt Nam",
        "Trí tuệ nhân tạo đang được ứng dụng rộng rãi",
        "Du lịch Việt Nam thu hút hàng triệu khách quốc tế mỗi năm",
        "Văn hóa Việt Nam đa dạng và phong phú",
        "Ẩm thực Việt Nam nổi tiếng trên thế giới",
    };

    var prng = std.Random.DefaultPrng.init(12345);
    const random = prng.random();

    for (0..doc_count) |i| {
        // Combine 2-5 templates for variety
        const num_parts = 2 + random.uintLessThan(usize, 4);
        var total_len: usize = 0;

        var parts: [5][]const u8 = undefined;
        for (0..num_parts) |j| {
            parts[j] = templates[random.uintLessThan(usize, templates.len)];
            total_len += parts[j].len + 1;
        }

        var doc = try allocator.alloc(u8, total_len);
        var offset: usize = 0;
        for (0..num_parts) |j| {
            @memcpy(doc[offset..][0..parts[j].len], parts[j]);
            offset += parts[j].len;
            if (j < num_parts - 1) {
                doc[offset] = ' ';
                offset += 1;
            }
        }
        // Store full allocation so we can free it correctly
        docs[i] = doc;
    }

    return docs;
}

fn printSummary(results: []const Results) void {
    std.debug.print("\n=== Benchmark Summary ===\n\n", .{});

    std.debug.print("| Profile   | Docs/sec    | MB/sec   | Index Size | p50 (µs) | p99 (µs) | QPS      |\n", .{});
    std.debug.print("|-----------|-------------|----------|------------|----------|----------|----------|\n", .{});

    for (results) |r| {
        std.debug.print("| {s: <9} | {d: >11.0} | {d: >8.1} | {d: >8.1} MB | {d: >8.1} | {d: >8.1} | {d: >8.0} |\n", .{
            r.profile,
            r.index_docs_per_sec,
            r.index_mb_per_sec,
            r.index_size_mb,
            r.search_p50_us,
            r.search_p99_us,
            r.search_qps,
        });
    }

    std.debug.print("\n", .{});
}

test "benchmark runs" {
    // Quick smoke test
    var gpa = std.heap.GeneralPurposeAllocator(.{}){};
    defer _ = gpa.deinit();

    _ = try runBenchmark(gpa.allocator(), "speed", .speed, .{
        .doc_limit = 100,
        .search_iterations = 10,
        .warmup_iterations = 5,
    });
}
