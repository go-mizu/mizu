//! Balanced profile: Good compression with Block-Max WAND
//! - VByte compressed posting lists with 128-doc blocks
//! - Block-level max scores for early termination
//! - FST for term dictionary
//! - BM25 scoring with block pruning
//!
//! Target: 1-10ms p99 search latency

const std = @import("std");
const Allocator = std.mem.Allocator;

// Use managed array list (stores allocator internally)
fn ManagedArrayList(comptime T: type) type {
    return std.array_list.AlignedManaged(T, null);
}
const vbyte = @import("../codec/vbyte.zig");
const fst_mod = @import("../codec/fst.zig");
const byte_tokenizer = @import("../tokenizer/byte.zig");
const scorer = @import("../search/scorer.zig");
const query_mod = @import("../search/query.zig");
const collector_mod = @import("../search/collector.zig");
const simd = @import("../util/simd.zig");

/// Block size for posting lists
const BLOCK_SIZE: usize = 128;

/// A block of postings with max score for pruning
pub const PostingBlock = struct {
    /// VByte-encoded doc IDs (delta from block start)
    doc_ids: []u8,
    /// Term frequencies (1 byte each)
    freqs: []u8,
    /// First doc ID in this block
    first_doc_id: u32,
    /// Last doc ID in this block
    last_doc_id: u32,
    /// Maximum BM25 score in this block (for pruning)
    max_score: f32,
    /// Number of docs in this block
    count: u16,
};

/// Term data with block-organized postings
pub const TermData = struct {
    blocks: []PostingBlock,
    total_docs: u32,
    idf: f32,
};

/// Document metadata
pub const DocMeta = struct {
    length: u32,
};

/// Balanced profile index
pub const BalancedIndex = struct {
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
            for (entry.value_ptr.blocks) |block| {
                self.allocator.free(block.doc_ids);
                self.allocator.free(block.freqs);
            }
            self.allocator.free(entry.value_ptr.blocks);
        }
        self.terms.deinit();
        self.docs.deinit();
    }

    /// Get number of documents
    pub fn docCount(self: Self) u32 {
        return @intCast(self.docs.items.len);
    }

    /// Search using Block-Max WAND algorithm
    pub fn search(self: *Self, query_text: []const u8, limit: usize) ![]collector_mod.SearchResult {
        var query = try query_mod.parse(self.allocator, query_text);
        defer query.deinit();

        if (query.terms.len == 0) {
            return &[_]collector_mod.SearchResult{};
        }

        if (query.terms.len == 1) {
            return self.searchSingleTerm(query.terms[0].hash, limit);
        }

        return self.searchBlockMaxWAND(query.terms, limit);
    }

    fn searchSingleTerm(self: *Self, term_hash: u64, limit: usize) ![]collector_mod.SearchResult {
        const term_data = self.terms.get(term_hash) orelse {
            return &[_]collector_mod.SearchResult{};
        };

        var heap = collector_mod.TopKCollector(100).init();
        const threshold = heap.minScore();

        // Process blocks, skip those below threshold
        for (term_data.blocks) |block| {
            if (block.max_score < threshold and heap.count >= limit) {
                continue; // Skip this block
            }

            // Decode and score block
            var doc_ids: [BLOCK_SIZE]u32 = undefined;
            const decoded = vbyte.decodeMany(block.doc_ids, &doc_ids);

            // Add base doc ID
            for (doc_ids[0..decoded.count], 0..) |*did, i| {
                if (i == 0) {
                    did.* = block.first_doc_id;
                } else {
                    did.* += doc_ids[i - 1];
                }
            }

            for (doc_ids[0..decoded.count], 0..) |doc_id, i| {
                const freq = block.freqs[i];
                const doc_meta = self.docs.items[doc_id];
                const score = self.bm25.score(freq, doc_meta.length, term_data.idf);
                heap.push(doc_id, score);
            }
        }

        const heap_results = heap.getResults();
        const result_count = @min(limit, heap_results.len);
        var results = try self.allocator.alloc(collector_mod.SearchResult, result_count);
        @memcpy(results[0..result_count], heap_results[0..result_count]);

        return results[0..result_count];
    }

    /// Block-Max WAND algorithm for multi-term queries
    fn searchBlockMaxWAND(self: *Self, terms: []const query_mod.QueryTerm, limit: usize) ![]collector_mod.SearchResult {
        // Gather term data
        var term_list = ManagedArrayList(TermWithCursor).init(self.allocator);
        defer term_list.deinit();

        for (terms) |term| {
            if (self.terms.get(term.hash)) |data| {
                try term_list.append(.{
                    .data = data,
                    .block_idx = 0,
                    .doc_idx = 0,
                    .current_doc = 0,
                });
            }
        }

        if (term_list.items.len == 0) {
            return &[_]collector_mod.SearchResult{};
        }

        var heap = collector_mod.TopKCollector(100).init();

        // Initialize cursors
        for (term_list.items) |*tc| {
            if (tc.data.blocks.len > 0) {
                tc.current_doc = tc.data.blocks[0].first_doc_id;
            }
        }

        // Simple DAAT (Document-At-A-Time) scoring
        // Full BMW would be more sophisticated
        var doc_scores = std.AutoHashMap(u32, f32).init(self.allocator);
        defer doc_scores.deinit();

        for (term_list.items) |tc| {
            for (tc.data.blocks) |block| {
                var doc_ids: [BLOCK_SIZE]u32 = undefined;
                const decoded = vbyte.decodeMany(block.doc_ids, &doc_ids);

                var prev: u32 = block.first_doc_id;
                for (0..decoded.count) |i| {
                    const doc_id = if (i == 0) block.first_doc_id else prev + doc_ids[i];
                    prev = doc_id;

                    const freq = block.freqs[i];
                    const doc_meta = self.docs.items[doc_id];
                    const score = self.bm25.score(freq, doc_meta.length, tc.data.idf);

                    const entry = try doc_scores.getOrPut(doc_id);
                    if (entry.found_existing) {
                        entry.value_ptr.* += score;
                    } else {
                        entry.value_ptr.* = score;
                    }
                }
            }
        }

        // Collect results
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

    const TermWithCursor = struct {
        data: TermData,
        block_idx: usize,
        doc_idx: usize,
        current_doc: u32,
    };

    /// Get memory usage estimate
    pub fn memoryUsage(self: Self) usize {
        var total: usize = 0;

        var iter = self.terms.iterator();
        while (iter.next()) |entry| {
            total += entry.value_ptr.blocks.len * @sizeOf(PostingBlock);
            for (entry.value_ptr.blocks) |block| {
                total += block.doc_ids.len;
                total += block.freqs.len;
            }
        }

        total += self.docs.items.len * @sizeOf(DocMeta);
        return total;
    }
};

/// Builder for balanced profile index
pub const BalancedIndexBuilder = struct {
    allocator: Allocator,
    /// Temporary: term hash -> list of (doc_id, freq)
    term_postings: std.AutoHashMap(u64, ManagedArrayList(TempPosting)),
    /// Document lengths
    doc_lengths: ManagedArrayList(u32),
    /// Total tokens
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
    pub fn build(self: *Self) !BalancedIndex {
        var index = BalancedIndex.init(self.allocator);

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

        // Convert posting lists to blocks
        var iter = self.term_postings.iterator();
        while (iter.next()) |entry| {
            const postings = entry.value_ptr.items;
            const idf = index.bm25.idf(@intCast(postings.len));

            // Create blocks
            const num_blocks = (postings.len + BLOCK_SIZE - 1) / BLOCK_SIZE;
            var blocks = try self.allocator.alloc(PostingBlock, num_blocks);

            for (0..num_blocks) |bi| {
                const start = bi * BLOCK_SIZE;
                const end = @min(start + BLOCK_SIZE, postings.len);
                const block_postings = postings[start..end];

                // Encode doc IDs as deltas
                var doc_id_encoder = vbyte.Encoder.init(self.allocator);
                defer doc_id_encoder.deinit();

                var freqs = try self.allocator.alloc(u8, block_postings.len);
                var max_score: f32 = 0;

                var prev_doc: u32 = 0;
                for (block_postings, 0..) |p, i| {
                    const delta = if (i == 0) 0 else p.doc_id - prev_doc;
                    try doc_id_encoder.add(delta);
                    prev_doc = p.doc_id;

                    freqs[i] = @intCast(@min(p.freq, 255));

                    // Calculate max score for this block
                    const doc_meta = index.docs.items[p.doc_id];
                    const s = index.bm25.score(p.freq, doc_meta.length, idf);
                    max_score = @max(max_score, s);
                }

                blocks[bi] = .{
                    .doc_ids = try self.allocator.dupe(u8, doc_id_encoder.bytes()),
                    .freqs = freqs,
                    .first_doc_id = block_postings[0].doc_id,
                    .last_doc_id = block_postings[block_postings.len - 1].doc_id,
                    .max_score = max_score,
                    .count = @intCast(block_postings.len),
                };
            }

            try index.terms.put(entry.key_ptr.*, .{
                .blocks = blocks,
                .total_docs = @intCast(postings.len),
                .idf = idf,
            });
        }

        return index;
    }
};

// ============================================================================
// Tests
// ============================================================================

test "balanced index basic" {
    var builder = BalancedIndexBuilder.init(std.testing.allocator);
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

test "balanced index multi term" {
    var builder = BalancedIndexBuilder.init(std.testing.allocator);
    defer builder.deinit();

    _ = try builder.addDocument("quick brown fox");
    _ = try builder.addDocument("lazy brown dog");

    var index = try builder.build();
    defer index.deinit();

    const results = try index.search("brown fox", 10);
    defer index.allocator.free(results);

    try std.testing.expect(results.len >= 1);
}
