//! Speed profile: Maximum search speed with no compression
//! - Raw u32 arrays for posting lists
//! - Hash map for term index
//! - Pre-computed BM25 components
//! - Fully memory-resident
//!
//! Target: <1ms p99 search latency

const std = @import("std");
const Allocator = std.mem.Allocator;

// Use managed array list (stores allocator internally)
fn ManagedArrayList(comptime T: type) type {
    return std.array_list.AlignedManaged(T, null);
}
const simd = @import("../util/simd.zig");
const hash_util = @import("../util/hash.zig");
const arena_mod = @import("../util/arena.zig");
const byte_tokenizer = @import("../tokenizer/byte.zig");
const scorer = @import("../search/scorer.zig");
const query_mod = @import("../search/query.zig");
const collector_mod = @import("../search/collector.zig");

/// Posting list entry (uncompressed for speed)
pub const Posting = struct {
    doc_id: u32,
    freq: u16,
    _padding: u16 = 0,
};

/// Term data in the index
pub const TermData = struct {
    postings: []Posting,
    doc_freq: u32,
    idf: f32, // Pre-computed IDF
};

/// Document metadata
pub const DocMeta = struct {
    length: u32, // Number of tokens
    // Could add more fields: URL hash, timestamp, etc.
};

/// Speed profile index
pub const SpeedIndex = struct {
    allocator: Allocator,
    /// Term hash -> posting list
    terms: std.AutoHashMap(u64, TermData),
    /// Document metadata
    docs: ManagedArrayList(DocMeta),
    /// BM25 scorer
    bm25: scorer.BM25Scorer,
    /// Total tokens across all docs
    total_tokens: u64,
    /// Index is finalized (no more additions)
    finalized: bool,

    const Self = @This();

    pub fn init(allocator: Allocator) Self {
        return .{
            .allocator = allocator,
            .terms = std.AutoHashMap(u64, TermData).init(allocator),
            .docs = ManagedArrayList(DocMeta).init(allocator),
            .bm25 = scorer.BM25Scorer.init(.{}, 0, 0),
            .total_tokens = 0,
            .finalized = false,
        };
    }

    pub fn deinit(self: *Self) void {
        var iter = self.terms.iterator();
        while (iter.next()) |entry| {
            self.allocator.free(entry.value_ptr.postings);
        }
        self.terms.deinit();
        self.docs.deinit();
    }

    /// Get number of documents
    pub fn docCount(self: Self) u32 {
        return @intCast(self.docs.items.len);
    }

    /// Get number of unique terms
    pub fn termCount(self: Self) u32 {
        return @intCast(self.terms.count());
    }

    /// Get memory usage estimate
    pub fn memoryUsage(self: Self) usize {
        var total: usize = 0;

        // Term map overhead
        total += self.terms.capacity() * (@sizeOf(u64) + @sizeOf(TermData));

        // Posting lists
        var iter = self.terms.iterator();
        while (iter.next()) |entry| {
            total += entry.value_ptr.postings.len * @sizeOf(Posting);
        }

        // Doc metadata
        total += self.docs.items.len * @sizeOf(DocMeta);

        return total;
    }

    /// Search the index
    pub fn search(self: *Self, query_text: []const u8, limit: usize) ![]collector_mod.SearchResult {
        var query = try query_mod.parse(self.allocator, query_text);
        defer query.deinit();

        if (query.terms.len == 0) {
            return &[_]collector_mod.SearchResult{};
        }

        // Single term query (fast path)
        if (query.terms.len == 1) {
            return self.searchSingleTerm(query.terms[0].hash, limit);
        }

        // Multi-term query
        return self.searchMultiTerm(query.terms, limit);
    }

    fn searchSingleTerm(self: *Self, term_hash: u64, limit: usize) ![]collector_mod.SearchResult {
        const term_data = self.terms.get(term_hash) orelse {
            return &[_]collector_mod.SearchResult{};
        };

        // Allocate results
        const result_count = @min(limit, term_data.postings.len);
        var results = try self.allocator.alloc(collector_mod.SearchResult, result_count);

        // Score and collect (could use SIMD here for larger lists)
        var heap = collector_mod.TopKCollector(100).init();

        for (term_data.postings) |posting| {
            const doc_meta = self.docs.items[posting.doc_id];
            const score = self.bm25.score(posting.freq, doc_meta.length, term_data.idf);
            heap.push(posting.doc_id, score);
        }

        const heap_results = heap.getResults();
        const copy_count = @min(result_count, heap_results.len);
        @memcpy(results[0..copy_count], heap_results[0..copy_count]);

        return results[0..copy_count];
    }

    fn searchMultiTerm(self: *Self, terms: []const query_mod.QueryTerm, limit: usize) ![]collector_mod.SearchResult {
        // Accumulate scores per document
        var doc_scores = std.AutoHashMap(u32, f32).init(self.allocator);
        defer doc_scores.deinit();

        for (terms) |term| {
            const term_data = self.terms.get(term.hash) orelse continue;

            for (term_data.postings) |posting| {
                const doc_meta = self.docs.items[posting.doc_id];
                const score = self.bm25.score(posting.freq, doc_meta.length, term_data.idf);

                const entry = try doc_scores.getOrPut(posting.doc_id);
                if (entry.found_existing) {
                    entry.value_ptr.* += score;
                } else {
                    entry.value_ptr.* = score;
                }
            }
        }

        // Collect top-K
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
};

/// Builder for speed profile index
pub const SpeedIndexBuilder = struct {
    allocator: Allocator,
    /// Temporary storage: term hash -> list of (doc_id, freq)
    term_postings: std.AutoHashMap(u64, ManagedArrayList(Posting)),
    /// Document lengths
    doc_lengths: ManagedArrayList(u32),
    /// Total tokens
    total_tokens: u64,

    const Self = @This();

    pub fn init(allocator: Allocator) Self {
        return .{
            .allocator = allocator,
            .term_postings = std.AutoHashMap(u64, ManagedArrayList(Posting)).init(allocator),
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

    /// Add a document to the index
    pub fn addDocument(self: *Self, text: []const u8) !u32 {
        const doc_id: u32 = @intCast(self.doc_lengths.items.len);

        // Tokenize
        var token_buf: [8192]byte_tokenizer.Token = undefined;
        var agg_buf: [4096]byte_tokenizer.Token = undefined;

        const tokenizer = byte_tokenizer.ByteTokenizer.init(.{ .lowercase = true });
        const result = byte_tokenizer.tokenizeAndAggregate(&tokenizer, text, &token_buf, &agg_buf);

        // Store document length
        try self.doc_lengths.append(result.doc_len);
        self.total_tokens += result.doc_len;

        // Add to posting lists
        for (result.tokens) |token| {
            const entry = try self.term_postings.getOrPut(token.hash);
            if (!entry.found_existing) {
                entry.value_ptr.* = ManagedArrayList(Posting).init(self.allocator);
            }
            try entry.value_ptr.append(.{
                .doc_id = doc_id,
                .freq = token.freq,
            });
        }

        return doc_id;
    }

    /// Build the final index
    pub fn build(self: *Self) !SpeedIndex {
        var index = SpeedIndex.init(self.allocator);

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

        // Convert posting lists
        var iter = self.term_postings.iterator();
        while (iter.next()) |entry| {
            const postings = try self.allocator.dupe(Posting, entry.value_ptr.items);
            const doc_freq: u32 = @intCast(postings.len);

            try index.terms.put(entry.key_ptr.*, .{
                .postings = postings,
                .doc_freq = doc_freq,
                .idf = index.bm25.idf(doc_freq),
            });
        }

        index.finalized = true;
        return index;
    }
};

// ============================================================================
// Tests
// ============================================================================

test "speed index basic" {
    var builder = SpeedIndexBuilder.init(std.testing.allocator);
    defer builder.deinit();

    _ = try builder.addDocument("hello world");
    _ = try builder.addDocument("hello there");
    _ = try builder.addDocument("world peace");

    var index = try builder.build();
    defer index.deinit();

    try std.testing.expectEqual(@as(u32, 3), index.docCount());
    try std.testing.expect(index.termCount() >= 4); // hello, world, there, peace

    // Search
    const results = try index.search("hello", 10);
    defer index.allocator.free(results);

    try std.testing.expectEqual(@as(usize, 2), results.len);
}

test "speed index multi term" {
    var builder = SpeedIndexBuilder.init(std.testing.allocator);
    defer builder.deinit();

    _ = try builder.addDocument("the quick brown fox");
    _ = try builder.addDocument("the lazy brown dog");
    _ = try builder.addDocument("quick dog runs");

    var index = try builder.build();
    defer index.deinit();

    const results = try index.search("brown dog", 10);
    defer index.allocator.free(results);

    // Doc 1 (lazy brown dog) should rank highest as it has both terms
    try std.testing.expect(results.len >= 1);
}

test "speed index empty query" {
    var builder = SpeedIndexBuilder.init(std.testing.allocator);
    defer builder.deinit();

    _ = try builder.addDocument("test document");

    var index = try builder.build();
    defer index.deinit();

    const results = try index.search("", 10);

    try std.testing.expectEqual(@as(usize, 0), results.len);
}
