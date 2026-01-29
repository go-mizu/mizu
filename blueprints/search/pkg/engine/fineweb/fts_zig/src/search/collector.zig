//! Result collectors for search operations
//! Supports top-K collection with early termination

const std = @import("std");
const scorer = @import("scorer.zig");

/// Search result
pub const SearchResult = struct {
    doc_id: u32,
    score: f32,
};

/// Collector interface for gathering search results
pub const Collector = struct {
    ptr: *anyopaque,
    vtable: *const VTable,

    const VTable = struct {
        collect: *const fn (*anyopaque, u32, f32) void,
        threshold: *const fn (*anyopaque) f32,
        results: *const fn (*anyopaque) []SearchResult,
        reset: *const fn (*anyopaque) void,
    };

    pub fn collect(self: Collector, doc_id: u32, score: f32) void {
        self.vtable.collect(self.ptr, doc_id, score);
    }

    /// Get current score threshold (for early termination)
    pub fn threshold(self: Collector) f32 {
        return self.vtable.threshold(self.ptr);
    }

    /// Get collected results
    pub fn results(self: Collector) []SearchResult {
        return self.vtable.results(self.ptr);
    }

    pub fn reset(self: Collector) void {
        self.vtable.reset(self.ptr);
    }
};

/// Top-K collector using a min-heap
pub fn TopKCollector(comptime K: usize) type {
    return struct {
        heap: [K]SearchResult = undefined,
        count: usize = 0,
        sorted_results: [K]SearchResult = undefined,

        const Self = @This();

        pub fn init() Self {
            return .{};
        }

        pub fn collector(self: *Self) Collector {
            return .{
                .ptr = self,
                .vtable = &.{
                    .collect = collectFn,
                    .threshold = thresholdFn,
                    .results = resultsFn,
                    .reset = resetFn,
                },
            };
        }

        fn collectFn(ptr: *anyopaque, doc_id: u32, score: f32) void {
            const self: *Self = @ptrCast(@alignCast(ptr));
            self.push(doc_id, score);
        }

        fn thresholdFn(ptr: *anyopaque) f32 {
            const self: *Self = @ptrCast(@alignCast(ptr));
            return self.minScore();
        }

        fn resultsFn(ptr: *anyopaque) []SearchResult {
            const self: *Self = @ptrCast(@alignCast(ptr));
            return self.getResults();
        }

        fn resetFn(ptr: *anyopaque) void {
            const self: *Self = @ptrCast(@alignCast(ptr));
            self.count = 0;
        }

        pub fn push(self: *Self, doc_id: u32, score: f32) void {
            if (self.count < K) {
                self.heap[self.count] = .{ .doc_id = doc_id, .score = score };
                self.count += 1;
                self.bubbleUp(self.count - 1);
            } else if (score > self.heap[0].score) {
                self.heap[0] = .{ .doc_id = doc_id, .score = score };
                self.heapifyDown(0);
            }
        }

        pub fn minScore(self: Self) f32 {
            if (self.count < K) return 0;
            return self.heap[0].score;
        }

        pub fn getResults(self: *Self) []SearchResult {
            // Copy and sort by score descending
            @memcpy(self.sorted_results[0..self.count], self.heap[0..self.count]);

            std.mem.sort(SearchResult, self.sorted_results[0..self.count], {}, struct {
                fn cmp(_: void, a: SearchResult, b: SearchResult) bool {
                    return a.score > b.score;
                }
            }.cmp);

            return self.sorted_results[0..self.count];
        }

        fn bubbleUp(self: *Self, start: usize) void {
            var i = start;
            while (i > 0) {
                const parent = (i - 1) / 2;
                if (self.heap[i].score < self.heap[parent].score) {
                    std.mem.swap(SearchResult, &self.heap[i], &self.heap[parent]);
                    i = parent;
                } else break;
            }
        }

        fn heapifyDown(self: *Self, start: usize) void {
            var i = start;
            while (true) {
                var smallest = i;
                const left = 2 * i + 1;
                const right = 2 * i + 2;

                if (left < self.count and self.heap[left].score < self.heap[smallest].score) {
                    smallest = left;
                }
                if (right < self.count and self.heap[right].score < self.heap[smallest].score) {
                    smallest = right;
                }

                if (smallest == i) break;

                std.mem.swap(SearchResult, &self.heap[i], &self.heap[smallest]);
                i = smallest;
            }
        }
    };
}

/// Simple collector that stores all results (for small result sets)
pub const AllResultsCollector = struct {
    results_buf: std.ArrayList(SearchResult),
    max_results: usize,

    const Self = @This();

    pub fn init(allocator: std.mem.Allocator, max_results: usize) Self {
        return .{
            .results_buf = std.ArrayList(SearchResult).init(allocator),
            .max_results = max_results,
        };
    }

    pub fn deinit(self: *Self) void {
        self.results_buf.deinit();
    }

    pub fn collect(self: *Self, doc_id: u32, score: f32) void {
        if (self.results_buf.items.len < self.max_results) {
            self.results_buf.append(.{ .doc_id = doc_id, .score = score }) catch {};
        }
    }

    pub fn threshold(self: Self) f32 {
        _ = self;
        return 0; // No threshold
    }

    pub fn results(self: *Self) []SearchResult {
        // Sort by score descending
        std.mem.sort(SearchResult, self.results_buf.items, {}, struct {
            fn cmp(_: void, a: SearchResult, b: SearchResult) bool {
                return a.score > b.score;
            }
        }.cmp);
        return self.results_buf.items;
    }

    pub fn reset(self: *Self) void {
        self.results_buf.clearRetainingCapacity();
    }
};

// ============================================================================
// Tests
// ============================================================================

test "topk collector" {
    var collector_impl = TopKCollector(3).init();
    var coll = collector_impl.collector();

    coll.collect(1, 0.5);
    coll.collect(2, 0.9);
    coll.collect(3, 0.3);
    coll.collect(4, 0.8);
    coll.collect(5, 0.1);

    const results = coll.results();

    try std.testing.expectEqual(@as(usize, 3), results.len);
    try std.testing.expectEqual(@as(f32, 0.9), results[0].score);
    try std.testing.expectEqual(@as(f32, 0.8), results[1].score);
    try std.testing.expectEqual(@as(f32, 0.5), results[2].score);
}

test "topk threshold" {
    var collector_impl = TopKCollector(2).init();

    collector_impl.push(1, 0.5);
    try std.testing.expectEqual(@as(f32, 0), collector_impl.minScore());

    collector_impl.push(2, 0.3);
    try std.testing.expectEqual(@as(f32, 0.3), collector_impl.minScore());

    collector_impl.push(3, 0.8);
    try std.testing.expectEqual(@as(f32, 0.5), collector_impl.minScore());
}
