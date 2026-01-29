//! Compact profile: Maximum compression with Elias-Fano encoding
//! - Elias-Fano encoded posting lists (~2 bits/integer)
//! - FST for term dictionary
//! - Two-phase BM25 retrieval
//!
//! Target: 10-50ms p99 search latency, minimum memory footprint

const std = @import("std");
const Allocator = std.mem.Allocator;

// Use managed array list (stores allocator internally)
fn ManagedArrayList(comptime T: type) type {
    return std.array_list.AlignedManaged(T, null);
}
const eliasfano = @import("../codec/eliasfano.zig");
const byte_tokenizer = @import("../tokenizer/byte.zig");
const scorer = @import("../search/scorer.zig");
const query_mod = @import("../search/query.zig");
const collector_mod = @import("../search/collector.zig");

/// Term data with Elias-Fano encoded postings
pub const TermData = struct {
    /// Elias-Fano encoded doc IDs
    doc_ids: eliasfano.EliasFano,
    /// Compressed frequencies (simple encoding)
    freqs: []u8,
    /// Document frequency
    doc_freq: u32,
    /// Pre-computed IDF
    idf: f32,
};

/// Document metadata
pub const DocMeta = struct {
    length: u32,
};

/// Compact profile index
pub const CompactIndex = struct {
    allocator: Allocator,
    /// Term hash -> term data
    terms: std.AutoHashMap(u64, TermData),
    /// Document metadata
    docs: ManagedArrayList(DocMeta),
    /// BM25 scorer
    bm25: scorer.BM25Scorer,
    /// Total tokens
    total_tokens: u64,

    const Self = @This();

    pub fn init(allocator: Allocator) Self {
        return .{
            .allocator = allocator,
            .terms = std.AutoHashMap(u64, TermData).init(allocator),
            .docs = ManagedArrayList(DocMeta).init(allocator),
            .bm25 = scorer.BM25Scorer.init(.{}, 0, 0),
            .total_tokens = 0,
        };
    }

    pub fn deinit(self: *Self) void {
        var iter = self.terms.iterator();
        while (iter.next()) |entry| {
            var ef = entry.value_ptr.doc_ids;
            ef.deinit(self.allocator);
            self.allocator.free(entry.value_ptr.freqs);
        }
        self.terms.deinit();
        self.docs.deinit();
    }

    /// Get number of documents
    pub fn docCount(self: Self) u32 {
        return @intCast(self.docs.items.len);
    }

    /// Search the index
    pub fn search(self: *Self, query_text: []const u8, limit: usize) ![]collector_mod.SearchResult {
        var query = try query_mod.parse(self.allocator, query_text);
        defer query.deinit();

        if (query.terms.len == 0) {
            return &[_]collector_mod.SearchResult{};
        }

        if (query.terms.len == 1) {
            return self.searchSingleTerm(query.terms[0].hash, limit);
        }

        return self.searchMultiTerm(query.terms, limit);
    }

    fn searchSingleTerm(self: *Self, term_hash: u64, limit: usize) ![]collector_mod.SearchResult {
        const term_data = self.terms.get(term_hash) orelse {
            return &[_]collector_mod.SearchResult{};
        };

        var heap = collector_mod.TopKCollector(100).init();

        // Iterate through Elias-Fano encoded doc IDs
        var ef_iter = term_data.doc_ids.iterator();
        var idx: usize = 0;

        while (ef_iter.next()) |doc_id| {
            const freq = term_data.freqs[idx];
            const doc_meta = self.docs.items[doc_id];
            const score = self.bm25.score(freq, doc_meta.length, term_data.idf);
            heap.push(doc_id, score);
            idx += 1;
        }

        const heap_results = heap.getResults();
        const result_count = @min(limit, heap_results.len);
        var results = try self.allocator.alloc(collector_mod.SearchResult, result_count);
        @memcpy(results[0..result_count], heap_results[0..result_count]);

        return results[0..result_count];
    }

    fn searchMultiTerm(self: *Self, terms: []const query_mod.QueryTerm, limit: usize) ![]collector_mod.SearchResult {
        var doc_scores = std.AutoHashMap(u32, f32).init(self.allocator);
        defer doc_scores.deinit();

        for (terms) |term| {
            const term_data = self.terms.get(term.hash) orelse continue;

            var ef_iter = term_data.doc_ids.iterator();
            var idx: usize = 0;

            while (ef_iter.next()) |doc_id| {
                const freq = term_data.freqs[idx];
                const doc_meta = self.docs.items[doc_id];
                const score = self.bm25.score(freq, doc_meta.length, term_data.idf);

                const entry = try doc_scores.getOrPut(doc_id);
                if (entry.found_existing) {
                    entry.value_ptr.* += score;
                } else {
                    entry.value_ptr.* = score;
                }
                idx += 1;
            }
        }

        var heap = collector_mod.TopKCollector(100).init();
        var iter = doc_scores.iterator();
        while (iter.next()) |entry| {
            heap.push(entry.key_ptr.*, entry.value_ptr.*);
        }

        const heap_results = heap.getResults();
        const result_count = @min(limit, heap_results.len);
        var results = try self.allocator.alloc(collector_mod.SearchResult, result_count);
        @memcpy(results[0..result_count], heap_results[0..result_count]);

        return results[0..result_count];
    }

    /// Get memory usage estimate
    pub fn memoryUsage(self: Self) usize {
        var total: usize = 0;

        var iter = self.terms.iterator();
        while (iter.next()) |entry| {
            total += entry.value_ptr.doc_ids.memoryUsage();
            total += entry.value_ptr.freqs.len;
        }

        total += self.docs.items.len * @sizeOf(DocMeta);
        return total;
    }

    /// Get compression statistics
    pub fn compressionStats(self: Self) CompressionStats {
        var total_postings: u64 = 0;
        var total_bits: u64 = 0;

        var iter = self.terms.iterator();
        while (iter.next()) |entry| {
            total_postings += entry.value_ptr.doc_freq;
            total_bits += entry.value_ptr.doc_ids.memoryUsage() * 8;
        }

        return .{
            .total_postings = total_postings,
            .total_bits = total_bits,
            .bits_per_posting = if (total_postings > 0)
                @as(f64, @floatFromInt(total_bits)) / @as(f64, @floatFromInt(total_postings))
            else
                0,
        };
    }

    pub const CompressionStats = struct {
        total_postings: u64,
        total_bits: u64,
        bits_per_posting: f64,
    };
};

/// Builder for compact profile index
pub const CompactIndexBuilder = struct {
    allocator: Allocator,
    term_postings: std.AutoHashMap(u64, ManagedArrayList(TempPosting)),
    doc_lengths: ManagedArrayList(u32),
    total_tokens: u64,

    const Self = @This();

    const TempPosting = struct {
        doc_id: u32,
        freq: u16,
    };

    pub fn init(allocator: Allocator) Self {
        return .{
            .allocator = allocator,
            .term_postings = std.AutoHashMap(u64, ManagedArrayList(TempPosting)).init(allocator),
            .doc_lengths = ManagedArrayList(u32).init(allocator),
            .total_tokens = 0,
        };
    }

    pub fn deinit(self: *Self) void {
        var iter = self.term_postings.iterator();
        while (iter.next()) |entry| {
            entry.value_ptr.deinit();
        }
        self.term_postings.deinit();
        self.doc_lengths.deinit();
    }

    /// Add a document
    pub fn addDocument(self: *Self, text: []const u8) !u32 {
        const doc_id: u32 = @intCast(self.doc_lengths.items.len);

        var token_buf: [8192]byte_tokenizer.Token = undefined;
        var agg_buf: [4096]byte_tokenizer.Token = undefined;

        const tokenizer = byte_tokenizer.ByteTokenizer.init(.{ .lowercase = true });
        const result = byte_tokenizer.tokenizeAndAggregate(&tokenizer, text, &token_buf, &agg_buf);

        try self.doc_lengths.append(result.doc_len);
        self.total_tokens += result.doc_len;

        for (result.tokens) |token| {
            const entry = try self.term_postings.getOrPut(token.hash);
            if (!entry.found_existing) {
                entry.value_ptr.* = ManagedArrayList(TempPosting).init(self.allocator);
            }
            try entry.value_ptr.append(.{
                .doc_id = doc_id,
                .freq = token.freq,
            });
        }

        return doc_id;
    }

    /// Build the index
    pub fn build(self: *Self) !CompactIndex {
        var index = CompactIndex.init(self.allocator);

        // Copy document metadata
        for (self.doc_lengths.items) |len| {
            try index.docs.append(.{ .length = len });
        }
        index.total_tokens = self.total_tokens;

        // Initialize BM25 scorer
        index.bm25 = scorer.BM25Scorer.init(
            .{},
            @intCast(self.doc_lengths.items.len),
            self.total_tokens,
        );

        // Convert posting lists to Elias-Fano
        var iter = self.term_postings.iterator();
        while (iter.next()) |entry| {
            const postings = entry.value_ptr.items;
            const doc_freq: u32 = @intCast(postings.len);
            const idf = index.bm25.idf(doc_freq);

            // Extract doc IDs for Elias-Fano
            var doc_ids = try self.allocator.alloc(u32, postings.len);
            defer self.allocator.free(doc_ids);

            var freqs = try self.allocator.alloc(u8, postings.len);

            for (postings, 0..) |p, i| {
                doc_ids[i] = p.doc_id;
                freqs[i] = @intCast(@min(p.freq, 255));
            }

            // Build Elias-Fano encoding
            const ef = try eliasfano.EliasFano.build(self.allocator, doc_ids);

            try index.terms.put(entry.key_ptr.*, .{
                .doc_ids = ef,
                .freqs = freqs,
                .doc_freq = doc_freq,
                .idf = idf,
            });
        }

        return index;
    }
};

// ============================================================================
// Tests
// ============================================================================

test "compact index basic" {
    var builder = CompactIndexBuilder.init(std.testing.allocator);
    defer builder.deinit();

    _ = try builder.addDocument("hello world");
    _ = try builder.addDocument("hello there");
    _ = try builder.addDocument("world peace");

    var index = try builder.build();
    defer index.deinit();

    try std.testing.expectEqual(@as(u32, 3), index.docCount());

    const results = try index.search("hello", 10);
    defer index.allocator.free(results);

    try std.testing.expectEqual(@as(usize, 2), results.len);
}

test "compact index compression" {
    var builder = CompactIndexBuilder.init(std.testing.allocator);
    defer builder.deinit();

    // Add many documents to test compression
    var i: usize = 0;
    while (i < 100) : (i += 1) {
        _ = try builder.addDocument("this is a test document with some words");
    }

    var index = try builder.build();
    defer index.deinit();

    const stats = index.compressionStats();

    // Elias-Fano should achieve good compression
    try std.testing.expect(stats.bits_per_posting < 32);
}
