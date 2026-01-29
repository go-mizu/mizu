//! High-Throughput Indexing Benchmark
//! Target: 1M docs/sec
//! Focus: Measure pure indexing throughput without I/O overhead

const std = @import("std");
const time = std.time;
const builtin = @import("builtin");

// Import tokenizer and hash utilities
const byte_tokenizer = @import("fts_zig").tokenizer.byte;
const hash_util = @import("fts_zig").util.hash;
const simd = @import("fts_zig").util.simd;

const Allocator = std.mem.Allocator;

/// Benchmark configuration
const BenchConfig = struct {
    num_docs: u32 = 100_000,
    num_workers: u32 = 0, // 0 = auto-detect
    warmup_docs: u32 = 10_000,
    iterations: u32 = 3,
};

/// Result from a single benchmark run
const BenchResult = struct {
    docs_per_sec: f64,
    mb_per_sec: f64,
    total_tokens: u64,
    elapsed_ns: u64,
};

/// Vietnamese sample documents (realistic content)
const vietnamese_templates = [_][]const u8{
    "Việt Nam là một quốc gia nằm ở Đông Nam Á với diện tích 331.212 km vuông và dân số hơn 100 triệu người",
    "Thành phố Hồ Chí Minh là trung tâm kinh tế lớn nhất cả nước với GDP chiếm hơn 20% tổng sản phẩm quốc nội",
    "Hà Nội là thủ đô nghìn năm văn hiến với nhiều di tích lịch sử và danh lam thắng cảnh nổi tiếng",
    "Công nghệ thông tin đang phát triển mạnh mẽ tại Việt Nam với nhiều startup công nghệ thành công",
    "Kinh tế Việt Nam tăng trưởng ổn định trong những năm gần đây với tốc độ GDP trung bình 6-7%",
    "Giáo dục là quốc sách hàng đầu của Việt Nam nhằm phát triển nguồn nhân lực chất lượng cao",
    "Trí tuệ nhân tạo và machine learning đang được ứng dụng rộng rãi trong nhiều lĩnh vực",
    "Du lịch Việt Nam thu hút hàng triệu khách quốc tế mỗi năm đến tham quan Hạ Long và Phú Quốc",
    "Văn hóa Việt Nam đa dạng và phong phú với 54 dân tộc anh em cùng chung sống hòa thuận",
    "Ẩm thực Việt Nam nổi tiếng thế giới với phở bò và bánh mì đã trở thành biểu tượng ẩm thực",
    "Nông nghiệp Việt Nam xuất khẩu gạo đứng thứ hai thế giới sau Thái Lan và Ấn Độ",
    "Công nghiệp sản xuất điện tử và dệt may đóng góp lớn vào kim ngạch xuất khẩu quốc gia",
};

/// Generate sample documents with realistic Vietnamese content
fn generateDocs(allocator: Allocator, count: u32) ![][]const u8 {
    var docs = try allocator.alloc([]const u8, count);
    var prng = std.Random.DefaultPrng.init(12345);
    const random = prng.random();

    for (0..count) |i| {
        // Combine 2-4 templates for variety (avg ~400 bytes per doc)
        const num_parts = 2 + random.uintLessThan(usize, 3);
        var total_len: usize = 0;

        var parts: [4][]const u8 = undefined;
        for (0..num_parts) |j| {
            parts[j] = vietnamese_templates[random.uintLessThan(usize, vietnamese_templates.len)];
            total_len += parts[j].len;
        }
        total_len += num_parts - 1; // spaces

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
        docs[i] = doc;
    }

    return docs;
}

fn freeDocs(allocator: Allocator, docs: [][]const u8) void {
    for (docs) |doc| {
        allocator.free(doc);
    }
    allocator.free(docs);
}

/// Phase 1: Pure tokenization benchmark (no indexing)
fn benchTokenizeOnly(docs: [][]const u8) BenchResult {
    const tokenizer = byte_tokenizer.ByteTokenizer.init(.{ .lowercase = true });
    var token_buf: [4096]byte_tokenizer.Token = undefined;
    var total_tokens: u64 = 0;
    var total_bytes: u64 = 0;

    const start = time.nanoTimestamp();

    for (docs) |doc| {
        const result = tokenizer.tokenize(doc, &token_buf);
        total_tokens += result.tokens.len;
        total_bytes += doc.len;
    }

    const end = time.nanoTimestamp();
    const elapsed_ns: u64 = @intCast(end - start);
    const elapsed_sec = @as(f64, @floatFromInt(elapsed_ns)) / @as(f64, time.ns_per_s);

    return .{
        .docs_per_sec = @as(f64, @floatFromInt(docs.len)) / elapsed_sec,
        .mb_per_sec = @as(f64, @floatFromInt(total_bytes)) / (1024 * 1024) / elapsed_sec,
        .total_tokens = total_tokens,
        .elapsed_ns = elapsed_ns,
    };
}

/// Phase 2: Tokenize + Hash (like Go's FNV hash during scan)
fn benchTokenizeAndHash(docs: [][]const u8) BenchResult {
    var total_tokens: u64 = 0;
    var total_bytes: u64 = 0;

    const start = time.nanoTimestamp();

    for (docs) |doc| {
        // Inline tokenization with hash computation (like Go's approach)
        var i: usize = 0;
        while (i < doc.len) {
            // Skip delimiters
            while (i < doc.len and isDelimiter(doc[i])) : (i += 1) {}
            if (i >= doc.len) break;

            // Scan token and compute hash inline
            const token_start = i;
            var h: u64 = 0xcbf29ce484222325; // FNV offset
            while (i < doc.len and !isDelimiter(doc[i])) {
                const c = toLower(doc[i]);
                h ^= c;
                h *%= 0x100000001b3; // FNV prime
                i += 1;
            }

            if (i > token_start) {
                total_tokens += 1;
                // Prevent dead code elimination
                std.mem.doNotOptimizeAway(h);
            }
        }
        total_bytes += doc.len;
    }

    const end = time.nanoTimestamp();
    const elapsed_ns: u64 = @intCast(end - start);
    const elapsed_sec = @as(f64, @floatFromInt(elapsed_ns)) / @as(f64, time.ns_per_s);

    return .{
        .docs_per_sec = @as(f64, @floatFromInt(docs.len)) / elapsed_sec,
        .mb_per_sec = @as(f64, @floatFromInt(total_bytes)) / (1024 * 1024) / elapsed_sec,
        .total_tokens = total_tokens,
        .elapsed_ns = elapsed_ns,
    };
}

/// Phase 3: SIMD-accelerated tokenization
fn benchTokenizeSIMD(docs: [][]const u8) BenchResult {
    var total_tokens: u64 = 0;
    var total_bytes: u64 = 0;

    const start = time.nanoTimestamp();

    for (docs) |doc| {
        var i: usize = 0;
        var in_token = false;
        var token_start: usize = 0;

        // SIMD path for 32-byte chunks
        while (i + 32 <= doc.len) {
            const chunk: *const [32]u8 = @ptrCast(doc.ptr + i);
            const delim_mask = simd.findDelimiters32(chunk);

            if (delim_mask == 0) {
                if (!in_token) {
                    token_start = i;
                    in_token = true;
                }
                i += 32;
                continue;
            }

            // Process delimiters
            if (!in_token and (delim_mask & 1) == 0) {
                token_start = i;
                in_token = true;
            }

            var mask = delim_mask;
            while (mask != 0) {
                const pos: usize = @ctz(mask);
                const abs_pos = i + pos;

                if (in_token and abs_pos > token_start) {
                    total_tokens += 1;
                    in_token = false;
                }

                const next = pos + 1;
                if (next < 32 and i + next < doc.len and !isDelimiter(doc[i + next])) {
                    token_start = i + next;
                    in_token = true;
                }

                mask &= mask - 1;
            }
            i += 32;
        }

        // Scalar remainder
        while (i < doc.len) {
            if (isDelimiter(doc[i])) {
                if (in_token) {
                    total_tokens += 1;
                    in_token = false;
                }
            } else if (!in_token) {
                token_start = i;
                in_token = true;
            }
            i += 1;
        }

        if (in_token) {
            total_tokens += 1;
        }

        total_bytes += doc.len;
    }

    const end = time.nanoTimestamp();
    const elapsed_ns: u64 = @intCast(end - start);
    const elapsed_sec = @as(f64, @floatFromInt(elapsed_ns)) / @as(f64, time.ns_per_s);

    return .{
        .docs_per_sec = @as(f64, @floatFromInt(docs.len)) / elapsed_sec,
        .mb_per_sec = @as(f64, @floatFromInt(total_bytes)) / (1024 * 1024) / elapsed_sec,
        .total_tokens = total_tokens,
        .elapsed_ns = elapsed_ns,
    };
}

/// Fixed-size hash table for term frequency counting (like Go's FixedHashTable)
const FixedHashTable = struct {
    keys: [1024]u64,
    values: [1024]u16,
    used: [1024]bool,
    count: u32,

    const Self = @This();
    const CAPACITY = 1024;
    const MASK = CAPACITY - 1;

    fn init() Self {
        return .{
            .keys = [_]u64{0} ** CAPACITY,
            .values = [_]u16{0} ** CAPACITY,
            .used = [_]bool{false} ** CAPACITY,
            .count = 0,
        };
    }

    fn clear(self: *Self) void {
        for (0..CAPACITY) |i| {
            if (self.used[i]) {
                self.used[i] = false;
                self.keys[i] = 0;
                self.values[i] = 0;
            }
        }
        self.count = 0;
    }

    fn increment(self: *Self, key: u64) void {
        var idx = key & MASK;
        var probes: u32 = 0;

        while (probes < CAPACITY) {
            if (!self.used[idx]) {
                self.used[idx] = true;
                self.keys[idx] = key;
                self.values[idx] = 1;
                self.count += 1;
                return;
            }
            if (self.keys[idx] == key) {
                self.values[idx] +|= 1;
                return;
            }
            idx = (idx + 1) & MASK;
            probes += 1;
        }
    }
};

/// Phase 4: Full indexing with hash table (like Go's FixedTokenize)
fn benchFullIndex(docs: [][]const u8) BenchResult {
    var total_tokens: u64 = 0;
    var total_bytes: u64 = 0;
    var table = FixedHashTable.init();

    const start = time.nanoTimestamp();

    for (docs) |doc| {
        table.clear();

        var i: usize = 0;
        while (i < doc.len) {
            while (i < doc.len and isDelimiter(doc[i])) : (i += 1) {}
            if (i >= doc.len) break;

            const token_start = i;
            var h: u64 = 0xcbf29ce484222325;
            while (i < doc.len and !isDelimiter(doc[i])) {
                const c = toLower(doc[i]);
                h ^= c;
                h *%= 0x100000001b3;
                i += 1;
            }

            if (i > token_start) {
                table.increment(h);
                total_tokens += 1;
            }
        }
        total_bytes += doc.len;
    }

    const end = time.nanoTimestamp();
    const elapsed_ns: u64 = @intCast(end - start);
    const elapsed_sec = @as(f64, @floatFromInt(elapsed_ns)) / @as(f64, time.ns_per_s);

    return .{
        .docs_per_sec = @as(f64, @floatFromInt(docs.len)) / elapsed_sec,
        .mb_per_sec = @as(f64, @floatFromInt(total_bytes)) / (1024 * 1024) / elapsed_sec,
        .total_tokens = total_tokens,
        .elapsed_ns = elapsed_ns,
    };
}

inline fn isDelimiter(c: u8) bool {
    return c <= ' ' or c == '.' or c == ',' or c == ';' or c == ':' or
        c == '!' or c == '?' or c == '(' or c == ')' or c == '[' or c == ']' or
        c == '"' or c == '\'' or c == '-' or c == '\n' or c == '\r' or c == '\t';
}

inline fn toLower(c: u8) u64 {
    if (c >= 'A' and c <= 'Z') {
        return c + 32;
    }
    return c;
}

fn printResult(name: []const u8, result: BenchResult, target: f64) void {
    const gap = target / result.docs_per_sec;
    const status = if (result.docs_per_sec >= target) "✓" else " ";
    std.debug.print("{s} {s: <25} : {d:>10.0} docs/sec | {d:>6.1} MB/s | gap: {d:>4.1}x\n", .{
        status,
        name,
        result.docs_per_sec,
        result.mb_per_sec,
        gap,
    });
}

/// Phase 5: Multi-threaded full indexing
fn benchMultiThreaded(allocator: Allocator, docs: [][]const u8, num_threads: u32) BenchResult {
    const Thread = std.Thread;

    const WorkerContext = struct {
        docs: [][]const u8,
        start_idx: usize,
        end_idx: usize,
        tokens: u64,
        bytes: u64,
    };

    var contexts = allocator.alloc(WorkerContext, num_threads) catch return .{
        .docs_per_sec = 0,
        .mb_per_sec = 0,
        .total_tokens = 0,
        .elapsed_ns = 0,
    };
    defer allocator.free(contexts);

    const docs_per_worker = docs.len / num_threads;

    for (0..num_threads) |i| {
        contexts[i] = .{
            .docs = docs,
            .start_idx = i * docs_per_worker,
            .end_idx = if (i == num_threads - 1) docs.len else (i + 1) * docs_per_worker,
            .tokens = 0,
            .bytes = 0,
        };
    }

    const worker_fn = struct {
        fn run(ctx: *WorkerContext) void {
            var table = FixedHashTable.init();

            for (ctx.start_idx..ctx.end_idx) |doc_idx| {
                const doc = ctx.docs[doc_idx];
                table.clear();

                var i: usize = 0;
                while (i < doc.len) {
                    while (i < doc.len and isDelimiter(doc[i])) : (i += 1) {}
                    if (i >= doc.len) break;

                    const token_start = i;
                    var h: u64 = 0xcbf29ce484222325;
                    while (i < doc.len and !isDelimiter(doc[i])) {
                        const c = toLower(doc[i]);
                        h ^= c;
                        h *%= 0x100000001b3;
                        i += 1;
                    }

                    if (i > token_start) {
                        table.increment(h);
                        ctx.tokens += 1;
                    }
                }
                ctx.bytes += doc.len;
            }
        }
    }.run;

    const start = time.nanoTimestamp();

    var threads = allocator.alloc(Thread, num_threads) catch return .{
        .docs_per_sec = 0,
        .mb_per_sec = 0,
        .total_tokens = 0,
        .elapsed_ns = 0,
    };
    defer allocator.free(threads);

    for (0..num_threads) |i| {
        threads[i] = Thread.spawn(.{}, worker_fn, .{&contexts[i]}) catch continue;
    }

    for (threads) |t| {
        t.join();
    }

    const end = time.nanoTimestamp();
    const elapsed_ns: u64 = @intCast(end - start);
    const elapsed_sec = @as(f64, @floatFromInt(elapsed_ns)) / @as(f64, time.ns_per_s);

    var total_tokens: u64 = 0;
    var total_bytes: u64 = 0;
    for (contexts) |ctx| {
        total_tokens += ctx.tokens;
        total_bytes += ctx.bytes;
    }

    return .{
        .docs_per_sec = @as(f64, @floatFromInt(docs.len)) / elapsed_sec,
        .mb_per_sec = @as(f64, @floatFromInt(total_bytes)) / (1024 * 1024) / elapsed_sec,
        .total_tokens = total_tokens,
        .elapsed_ns = elapsed_ns,
    };
}

pub fn main() !void {
    var gpa = std.heap.GeneralPurposeAllocator(.{}){};
    defer _ = gpa.deinit();
    const allocator = gpa.allocator();

    const args = try std.process.argsAlloc(allocator);
    defer std.process.argsFree(allocator, args);

    var config = BenchConfig{};

    // Parse args
    var i: usize = 1;
    while (i < args.len) : (i += 1) {
        if (std.mem.eql(u8, args[i], "--docs") and i + 1 < args.len) {
            config.num_docs = std.fmt.parseInt(u32, args[i + 1], 10) catch 100_000;
            i += 1;
        }
    }

    std.debug.print("\n", .{});
    std.debug.print("╔══════════════════════════════════════════════════════════════════╗\n", .{});
    std.debug.print("║          fts_zig High-Throughput Indexing Benchmark              ║\n", .{});
    std.debug.print("║                    Target: 1,000,000 docs/sec                    ║\n", .{});
    std.debug.print("╚══════════════════════════════════════════════════════════════════╝\n", .{});
    std.debug.print("\n", .{});

    // System info
    std.debug.print("System: {s} ({s})\n", .{
        @tagName(builtin.cpu.arch),
        @tagName(builtin.os.tag),
    });
    std.debug.print("Documents: {d}\n", .{config.num_docs});
    std.debug.print("\n", .{});

    // Generate test data
    std.debug.print("Generating {d} documents...\n", .{config.num_docs});
    const docs = try generateDocs(allocator, config.num_docs);
    defer freeDocs(allocator, docs);

    var total_bytes: u64 = 0;
    for (docs) |doc| {
        total_bytes += doc.len;
    }
    const avg_doc_size = total_bytes / docs.len;
    std.debug.print("Total size: {d:.2} MB (avg {d} bytes/doc)\n", .{
        @as(f64, @floatFromInt(total_bytes)) / (1024 * 1024),
        avg_doc_size,
    });
    std.debug.print("\n", .{});

    const TARGET: f64 = 1_000_000.0;

    // Warmup
    std.debug.print("Warming up...\n", .{});
    _ = benchTokenizeOnly(docs[0..@min(10000, docs.len)]);

    std.debug.print("\n", .{});
    std.debug.print("═══════════════════════════════════════════════════════════════════\n", .{});
    std.debug.print("Phase Breakdown (single-threaded baseline)\n", .{});
    std.debug.print("═══════════════════════════════════════════════════════════════════\n", .{});

    // Phase 1: Pure tokenization (ByteTokenizer)
    const r1 = benchTokenizeOnly(docs);
    printResult("1. ByteTokenizer", r1, TARGET);

    // Phase 2: Inline tokenize + FNV hash
    const r2 = benchTokenizeAndHash(docs);
    printResult("2. Inline Tokenize+Hash", r2, TARGET);

    // Phase 3: SIMD tokenization
    const r3 = benchTokenizeSIMD(docs);
    printResult("3. SIMD Tokenization", r3, TARGET);

    // Phase 4: Full index with hash table
    const r4 = benchFullIndex(docs);
    printResult("4. Full Index (FixedHash)", r4, TARGET);

    std.debug.print("\n", .{});
    std.debug.print("═══════════════════════════════════════════════════════════════════\n", .{});
    std.debug.print("Multi-Threaded Full Indexing\n", .{});
    std.debug.print("═══════════════════════════════════════════════════════════════════\n", .{});

    // Test with different thread counts
    const thread_counts = [_]u32{ 2, 4, 8 };
    for (thread_counts) |threads| {
        const r_mt = benchMultiThreaded(allocator, docs, threads);
        var name_buf: [32]u8 = undefined;
        const name = std.fmt.bufPrint(&name_buf, "{d}-thread Full Index", .{threads}) catch "Multi-thread";
        printResult(name, r_mt, TARGET);
    }

    std.debug.print("\n", .{});
    std.debug.print("═══════════════════════════════════════════════════════════════════\n", .{});
    std.debug.print("Analysis\n", .{});
    std.debug.print("═══════════════════════════════════════════════════════════════════\n", .{});

    const best = @max(@max(r1.docs_per_sec, r2.docs_per_sec), @max(r3.docs_per_sec, r4.docs_per_sec));
    std.debug.print("Best single-threaded: {d:.0} docs/sec\n", .{best});
    std.debug.print("Gap to 1M target: {d:.1}x\n", .{TARGET / best});
    std.debug.print("\n", .{});

    // Run 8-thread as final result
    const r_final = benchMultiThreaded(allocator, docs, 8);
    std.debug.print("Best 8-thread: {d:.0} docs/sec\n", .{r_final.docs_per_sec});
    std.debug.print("Gap to 1M target: {d:.1}x\n", .{TARGET / r_final.docs_per_sec});
    std.debug.print("\n", .{});
}

test "benchmark basic" {
    var gpa = std.heap.GeneralPurposeAllocator(.{}){};
    defer _ = gpa.deinit();

    const docs = try generateDocs(gpa.allocator(), 1000);
    defer freeDocs(gpa.allocator(), docs);

    const r1 = benchTokenizeOnly(docs);
    try std.testing.expect(r1.docs_per_sec > 0);

    const r2 = benchTokenizeAndHash(docs);
    try std.testing.expect(r2.docs_per_sec > 0);
}
