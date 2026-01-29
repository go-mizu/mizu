//! Index manager coordinating segments, writer, and merger

const std = @import("std");
const Allocator = std.mem.Allocator;
const segment = @import("segment.zig");
const writer = @import("writer.zig");
const merger = @import("merger.zig");
const speed = @import("../profile/speed.zig");
const balanced = @import("../profile/balanced.zig");
const compact = @import("../profile/compact.zig");
const collector_mod = @import("../search/collector.zig");

/// Index configuration
pub const IndexConfig = struct {
    base_path: []const u8,
    profile: segment.Profile = .balanced,
    flush_threshold: u32 = 64 * 1024,
    enable_background_merge: bool = true,
};

/// Unified index manager
pub const IndexManager = struct {
    allocator: Allocator,
    config: IndexConfig,
    /// Index writer
    idx_writer: writer.IndexWriter,
    /// Background merger
    bg_merger: merger.Merger,
    /// In-memory index for recently added docs (for speed profile)
    memory_index: ?*speed.SpeedIndex,

    const Self = @This();

    pub fn init(allocator: Allocator, config: IndexConfig) Self {
        return .{
            .allocator = allocator,
            .config = config,
            .idx_writer = writer.IndexWriter.init(allocator, .{
                .base_path = config.base_path,
                .flush_threshold = config.flush_threshold,
                .profile = config.profile,
            }),
            .bg_merger = merger.Merger.init(allocator, config.base_path, .{}),
            .memory_index = null,
        };
    }

    pub fn deinit(self: *Self) void {
        self.idx_writer.deinit();
        self.bg_merger.deinit();
        if (self.memory_index) |idx| {
            idx.deinit();
            self.allocator.destroy(idx);
        }
    }

    /// Start background services
    pub fn start(self: *Self) !void {
        if (self.config.enable_background_merge) {
            try self.bg_merger.start();
        }
    }

    /// Stop background services
    pub fn stop(self: *Self) void {
        self.bg_merger.stop();
    }

    /// Add a document
    pub fn addDocument(self: *Self, text: []const u8) !u32 {
        return self.idx_writer.addDocument(text);
    }

    /// Add multiple documents
    pub fn addDocuments(self: *Self, texts: []const []const u8) !void {
        try self.idx_writer.addDocuments(texts);
    }

    /// Flush to segment
    pub fn flush(self: *Self) !void {
        try self.idx_writer.flush();
    }

    /// Search (currently only searches in-memory buffer)
    pub fn search(self: *Self, query: []const u8, limit: usize) ![]collector_mod.SearchResult {
        _ = self;
        _ = query;
        _ = limit;
        // TODO: Combine results from memory buffer and segments
        return &[_]collector_mod.SearchResult{};
    }

    /// Get statistics
    pub fn stats(self: Self) Stats {
        const writer_stats = self.idx_writer.stats();
        return .{
            .total_docs = writer_stats.total_docs,
            .buffered_docs = writer_stats.buffered_docs,
            .segment_count = writer_stats.segment_count,
            .total_tokens = writer_stats.buffered_tokens,
        };
    }

    pub const Stats = struct {
        total_docs: u32,
        buffered_docs: u32,
        segment_count: u32,
        total_tokens: u64,
    };
};

// ============================================================================
// Tests
// ============================================================================

test "manager basic" {
    const path = "/tmp/fts_zig_manager_test";
    std.fs.cwd().deleteTree(path) catch {};

    var manager = IndexManager.init(std.testing.allocator, .{
        .base_path = path,
        .flush_threshold = 100,
    });
    defer manager.deinit();

    _ = try manager.addDocument("hello world");
    _ = try manager.addDocument("test document");

    const s = manager.stats();
    try std.testing.expectEqual(@as(u32, 2), s.total_docs);

    std.fs.cwd().deleteTree(path) catch {};
}
