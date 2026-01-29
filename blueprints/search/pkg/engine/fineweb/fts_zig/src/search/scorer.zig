//! BM25 scoring implementation with SIMD optimization
//! Standard Okapi BM25 with configurable parameters

const std = @import("std");
const simd = @import("../util/simd.zig");

/// BM25 parameters
pub const BM25Params = struct {
    k1: f32 = 1.2,
    b: f32 = 0.75,
};

/// BM25 scorer with precomputed statistics
pub const BM25Scorer = struct {
    params: BM25Params,
    avg_dl: f32,
    total_docs: u32,

    const Self = @This();

    pub fn init(params: BM25Params, total_docs: u32, total_terms: u64) Self {
        const avg_dl = if (total_docs > 0)
            @as(f32, @floatFromInt(total_terms)) / @as(f32, @floatFromInt(total_docs))
        else
            1.0;

        return .{
            .params = params,
            .avg_dl = avg_dl,
            .total_docs = total_docs,
        };
    }

    /// Calculate IDF for a term
    pub inline fn idf(self: Self, doc_freq: u32) f32 {
        const n = @as(f32, @floatFromInt(self.total_docs));
        const df = @as(f32, @floatFromInt(doc_freq));

        // IDF = log((N - df + 0.5) / (df + 0.5) + 1)
        return @log((n - df + 0.5) / (df + 0.5) + 1.0);
    }

    /// Score a single document
    pub inline fn score(self: Self, tf: u32, doc_len: u32, term_idf: f32) f32 {
        const tf_f = @as(f32, @floatFromInt(tf));
        const dl_f = @as(f32, @floatFromInt(doc_len));

        // norm = 1 - b + b * (dl / avg_dl)
        const norm = 1.0 - self.params.b + self.params.b * (dl_f / self.avg_dl);

        // score = idf * tf / (tf + k1 * norm)
        return term_idf * tf_f / (tf_f + self.params.k1 * norm);
    }

    /// Score 8 documents simultaneously using SIMD
    pub inline fn scoreSIMD(self: Self, tf: simd.Vec8f32, doc_len: simd.Vec8f32, term_idf: f32) simd.Vec8f32 {
        const k1_vec: simd.Vec8f32 = @splat(self.params.k1);
        const b_vec: simd.Vec8f32 = @splat(self.params.b);
        const one_minus_b: simd.Vec8f32 = @splat(1.0 - self.params.b);
        const avg_dl_vec: simd.Vec8f32 = @splat(self.avg_dl);
        const idf_vec: simd.Vec8f32 = @splat(term_idf);

        // norm = 1 - b + b * (dl / avg_dl)
        const norm = one_minus_b + b_vec * (doc_len / avg_dl_vec);

        // score = idf * tf / (tf + k1 * norm)
        const denom = tf + k1_vec * norm;
        return idf_vec * tf / denom;
    }
};

/// Score accumulator for multi-term queries
pub const ScoreAccumulator = struct {
    scores: []f32,
    doc_ids: []u32,
    count: usize,
    capacity: usize,

    const Self = @This();

    pub fn init(allocator: std.mem.Allocator, capacity: usize) !Self {
        return .{
            .scores = try allocator.alloc(f32, capacity),
            .doc_ids = try allocator.alloc(u32, capacity),
            .count = 0,
            .capacity = capacity,
        };
    }

    pub fn deinit(self: *Self, allocator: std.mem.Allocator) void {
        allocator.free(self.scores);
        allocator.free(self.doc_ids);
    }

    pub fn reset(self: *Self) void {
        self.count = 0;
    }

    /// Add or update a document's score
    pub fn addScore(self: *Self, doc_id: u32, score_delta: f32) void {
        // Linear search (for small result sets)
        // For larger sets, use a hash map
        for (self.doc_ids[0..self.count], 0..) |id, i| {
            if (id == doc_id) {
                self.scores[i] += score_delta;
                return;
            }
        }

        // New document
        if (self.count < self.capacity) {
            self.doc_ids[self.count] = doc_id;
            self.scores[self.count] = score_delta;
            self.count += 1;
        }
    }

    /// Get top-k results (sorted by score descending)
    pub fn topK(self: *Self, k: usize) []const Result {
        const n = @min(k, self.count);
        if (n == 0) return &[_]Result{};

        // Sort by score descending
        const indices = self.doc_ids[0..self.count];
        const scores = self.scores[0..self.count];

        // Simple selection for small k
        var i: usize = 0;
        while (i < n) : (i += 1) {
            var max_idx = i;
            var j = i + 1;
            while (j < self.count) : (j += 1) {
                if (scores[j] > scores[max_idx]) {
                    max_idx = j;
                }
            }

            // Swap
            if (max_idx != i) {
                std.mem.swap(f32, &scores[i], &scores[max_idx]);
                std.mem.swap(u32, &indices[i], &indices[max_idx]);
            }
        }

        return @as([*]const Result, @ptrCast(indices.ptr))[0..n];
    }

    pub const Result = struct {
        doc_id: u32,
        score: f32,
    };
};

/// Top-K heap for efficient result collection
pub fn TopKHeap(comptime K: usize) type {
    return struct {
        items: [K]Entry = undefined,
        count: usize = 0,

        const Self = @This();

        pub const Entry = struct {
            doc_id: u32,
            score: f32,

            fn lessThan(_: void, a: Entry, b: Entry) std.math.Order {
                // Min-heap: smallest score at root
                return std.math.order(a.score, b.score);
            }
        };

        /// Try to add an entry, returns true if added
        pub fn push(self: *Self, doc_id: u32, score: f32) bool {
            if (self.count < K) {
                // Heap not full, just add
                self.items[self.count] = .{ .doc_id = doc_id, .score = score };
                self.count += 1;

                // Bubble up
                var i = self.count - 1;
                while (i > 0) {
                    const parent = (i - 1) / 2;
                    if (self.items[i].score < self.items[parent].score) {
                        std.mem.swap(Entry, &self.items[i], &self.items[parent]);
                        i = parent;
                    } else break;
                }
                return true;
            }

            // Heap full, check if better than min
            if (score <= self.items[0].score) {
                return false;
            }

            // Replace root and heapify down
            self.items[0] = .{ .doc_id = doc_id, .score = score };
            self.heapifyDown(0);
            return true;
        }

        /// Get minimum score in heap (threshold for pruning)
        pub fn minScore(self: Self) f32 {
            if (self.count == 0) return 0;
            return self.items[0].score;
        }

        /// Get results sorted by score descending
        pub fn results(self: *Self) []Entry {
            // Sort in place (destroys heap property)
            std.mem.sort(Entry, self.items[0..self.count], {}, struct {
                fn cmp(_: void, a: Entry, b: Entry) bool {
                    return a.score > b.score;
                }
            }.cmp);
            return self.items[0..self.count];
        }

        fn heapifyDown(self: *Self, start: usize) void {
            var i = start;
            while (true) {
                var smallest = i;
                const left = 2 * i + 1;
                const right = 2 * i + 2;

                if (left < self.count and self.items[left].score < self.items[smallest].score) {
                    smallest = left;
                }
                if (right < self.count and self.items[right].score < self.items[smallest].score) {
                    smallest = right;
                }

                if (smallest == i) break;

                std.mem.swap(Entry, &self.items[i], &self.items[smallest]);
                i = smallest;
            }
        }
    };
}

// ============================================================================
// Tests
// ============================================================================

test "bm25 scorer basic" {
    const scorer = BM25Scorer.init(.{}, 1000, 100000);

    const idf_common = scorer.idf(500); // Common term
    const idf_rare = scorer.idf(10); // Rare term

    // Rare terms should have higher IDF
    try std.testing.expect(idf_rare > idf_common);

    // Score calculation
    const score1 = scorer.score(1, 100, idf_rare);
    const score2 = scorer.score(5, 100, idf_rare);

    // Higher TF should give higher score
    try std.testing.expect(score2 > score1);
}

test "topk heap" {
    var heap = TopKHeap(3){};

    _ = heap.push(1, 0.5);
    _ = heap.push(2, 0.8);
    _ = heap.push(3, 0.3);
    _ = heap.push(4, 0.9); // Should replace doc 3
    _ = heap.push(5, 0.1); // Should be rejected

    try std.testing.expectEqual(@as(usize, 3), heap.count);

    const results = heap.results();
    try std.testing.expectEqual(@as(f32, 0.9), results[0].score);
    try std.testing.expectEqual(@as(f32, 0.8), results[1].score);
    try std.testing.expectEqual(@as(f32, 0.5), results[2].score);
}
