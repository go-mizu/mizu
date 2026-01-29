//! Indexing throughput benchmark
//! Measures raw indexing speed without search

const std = @import("std");
const fts = @import("fts_zig");
const time = std.time;

pub fn main() !void {
    var gpa = std.heap.GeneralPurposeAllocator(.{}){};
    defer _ = gpa.deinit();
    const allocator = gpa.allocator();

    const args = try std.process.argsAlloc(allocator);
    defer std.process.argsFree(allocator, args);

    var doc_count: u32 = 1_000_000; // Default 1M docs
    var doc_size: u32 = 500; // Average doc size in bytes

    // Parse args
    var i: usize = 1;
    while (i < args.len) : (i += 1) {
        if (std.mem.eql(u8, args[i], "--count") and i + 1 < args.len) {
            doc_count = std.fmt.parseInt(u32, args[i + 1], 10) catch 1_000_000;
            i += 1;
        } else if (std.mem.eql(u8, args[i], "--size") and i + 1 < args.len) {
            doc_size = std.fmt.parseInt(u32, args[i + 1], 10) catch 500;
            i += 1;
        }
    }

    std.debug.print("\n=== Indexing Throughput Benchmark ===\n\n", .{});
    std.debug.print("Configuration:\n", .{});
    std.debug.print("  Documents: {d}\n", .{doc_count});
    std.debug.print("  Avg doc size: {d} bytes\n", .{doc_size});
    std.debug.print("\n", .{});

    // Generate documents
    std.debug.print("Generating documents...\n", .{});
    const docs = try generateDocs(allocator, doc_count, doc_size);
    defer {
        for (docs) |doc| {
            allocator.free(doc);
        }
        allocator.free(docs);
    }

    var total_bytes: u64 = 0;
    for (docs) |doc| {
        total_bytes += doc.len;
    }

    std.debug.print("Total data size: {d:.2} MB\n\n", .{@as(f64, @floatFromInt(total_bytes)) / (1024 * 1024)});

    // Benchmark each profile
    try benchmarkProfile(allocator, "speed", docs, total_bytes);
    try benchmarkProfile(allocator, "balanced", docs, total_bytes);
    try benchmarkProfile(allocator, "compact", docs, total_bytes);
}

fn benchmarkProfile(allocator: std.mem.Allocator, profile_name: []const u8, docs: []const []const u8, total_bytes: u64) !void {
    std.debug.print("Profile: {s}\n", .{profile_name});

    // Choose builder based on profile
    const is_speed = std.mem.eql(u8, profile_name, "speed");
    const is_balanced = std.mem.eql(u8, profile_name, "balanced");
    const is_compact = std.mem.eql(u8, profile_name, "compact");

    var speed_builder: ?fts.profile.speed.SpeedIndexBuilder = null;
    var balanced_builder: ?fts.profile.balanced.BalancedIndexBuilder = null;
    var compact_builder: ?fts.profile.compact.CompactIndexBuilder = null;

    if (is_speed) {
        speed_builder = fts.profile.speed.SpeedIndexBuilder.init(allocator);
    } else if (is_balanced) {
        balanced_builder = fts.profile.balanced.BalancedIndexBuilder.init(allocator);
    } else if (is_compact) {
        compact_builder = fts.profile.compact.CompactIndexBuilder.init(allocator);
    }

    defer {
        if (speed_builder) |*b| b.deinit();
        if (balanced_builder) |*b| b.deinit();
        if (compact_builder) |*b| b.deinit();
    }

    // Benchmark
    const start = time.nanoTimestamp();

    for (docs) |doc| {
        if (speed_builder) |*b| {
            _ = try b.addDocument(doc);
        } else if (balanced_builder) |*b| {
            _ = try b.addDocument(doc);
        } else if (compact_builder) |*b| {
            _ = try b.addDocument(doc);
        }
    }

    // Build index
    var speed_index: ?fts.profile.speed.SpeedIndex = null;
    var balanced_index: ?fts.profile.balanced.BalancedIndex = null;
    var compact_index: ?fts.profile.compact.CompactIndex = null;

    if (speed_builder) |*b| {
        speed_index = try b.build();
    } else if (balanced_builder) |*b| {
        balanced_index = try b.build();
    } else if (compact_builder) |*b| {
        compact_index = try b.build();
    }

    defer {
        if (speed_index) |*i| i.deinit();
        if (balanced_index) |*i| i.deinit();
        if (compact_index) |*i| i.deinit();
    }

    const end = time.nanoTimestamp();
    const elapsed_ns: u64 = @intCast(end - start);
    const elapsed_sec = @as(f64, @floatFromInt(elapsed_ns)) / @as(f64, time.ns_per_s);

    const docs_per_sec = @as(f64, @floatFromInt(docs.len)) / elapsed_sec;
    const mb_per_sec = @as(f64, @floatFromInt(total_bytes)) / (1024 * 1024) / elapsed_sec;

    // Get index size
    var index_size: usize = 0;
    if (speed_index) |i| {
        index_size = i.memoryUsage();
    } else if (balanced_index) |i| {
        index_size = i.memoryUsage();
    } else if (compact_index) |i| {
        index_size = i.memoryUsage();
    }

    std.debug.print("  Time: {d:.2} sec\n", .{elapsed_sec});
    std.debug.print("  Throughput: {d:.0} docs/sec\n", .{docs_per_sec});
    std.debug.print("  Throughput: {d:.2} MB/sec\n", .{mb_per_sec});
    std.debug.print("  Index size: {d:.2} MB\n", .{@as(f64, @floatFromInt(index_size)) / (1024 * 1024)});
    std.debug.print("\n", .{});
}

fn generateDocs(allocator: std.mem.Allocator, count: u32, avg_size: u32) ![][]u8 {
    var docs = try allocator.alloc([]u8, count);

    const words = [_][]const u8{
        "Việt", "Nam", "thành", "phố", "người", "công", "nghệ", "kinh",
        "tế", "giáo", "dục", "văn", "hóa", "du", "lịch", "ẩm", "thực",
        "internet", "trí", "tuệ", "nhân", "tạo", "blockchain", "data",
        "machine", "learning", "artificial", "intelligence", "cloud",
        "computing", "software", "development", "programming", "system",
    };

    var prng = std.Random.DefaultPrng.init(42);
    const random = prng.random();

    for (0..count) |i| {
        // Random size around avg_size
        const size = avg_size / 2 + random.uintLessThan(u32, avg_size);
        var doc = try allocator.alloc(u8, size);

        var offset: usize = 0;
        while (offset < size - 20) {
            const word = words[random.uintLessThan(usize, words.len)];
            const word_len = @min(word.len, size - offset - 1);
            @memcpy(doc[offset..][0..word_len], word[0..word_len]);
            offset += word_len;
            if (offset < size - 1) {
                doc[offset] = ' ';
                offset += 1;
            }
        }

        docs[i] = doc[0..offset];
    }

    return docs;
}
