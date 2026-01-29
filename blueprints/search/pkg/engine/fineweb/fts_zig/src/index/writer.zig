//! Streaming index writer with background flushing
//! Accumulates documents in memory and flushes to segments

const std = @import("std");
const Allocator = std.mem.Allocator;
const Thread = std.Thread;
const Mutex = std.Thread.Mutex;
const segment = @import("segment.zig");
const byte_tokenizer = @import("../tokenizer/byte.zig");
const arena_mod = @import("../util/arena.zig");

fn ManagedArrayList(comptime T: type) type {
    return std.array_list.AlignedManaged(T, null);
}

/// Configuration for index writer
pub const WriterConfig = struct {
    /// Max documents before flushing to segment
    flush_threshold: u32 = 64 * 1024,
    /// Base path for segments
    base_path: []const u8,
    /// Profile to use
    profile: segment.Profile = .balanced,
    /// Enable background flushing
    background_flush: bool = true,
};

/// Temporary posting for a term
const TempPosting = struct {
    doc_id: u32,
    freq: u16,
};

/// Index writer with streaming support
pub const IndexWriter = struct {
    allocator: Allocator,
    config: WriterConfig,
    /// Current buffer: term hash -> postings
    term_postings: std.AutoHashMap(u64, ManagedArrayList(TempPosting)),
    /// Document lengths
    doc_lengths: ManagedArrayList(u32),
    /// Total tokens in buffer
    total_tokens: u64,
    /// Segment manager
    segment_manager: segment.SegmentManager,
    /// Global document ID counter
    global_doc_id: u32,
    /// Mutex for thread safety
    mutex: Mutex,

    const Self = @This();

    pub fn init(allocator: Allocator, config: WriterConfig) Self {
        return .{
            .allocator = allocator,
            .config = config,
            .term_postings = std.AutoHashMap(u64, ManagedArrayList(TempPosting)).init(allocator),
            .doc_lengths = ManagedArrayList(u32).init(allocator),
            .total_tokens = 0,
            .segment_manager = segment.SegmentManager.init(allocator, config.base_path),
            .global_doc_id = 0,
            .mutex = .{},
        };
    }

    pub fn deinit(self: *Self) void {
        var iter = self.term_postings.iterator();
        while (iter.next()) |entry| {
            entry.value_ptr.deinit();
        }
        self.term_postings.deinit();
        self.doc_lengths.deinit();
        self.segment_manager.deinit();
    }

    /// Add a document to the index
    pub fn addDocument(self: *Self, text: []const u8) !u32 {
        self.mutex.lock();
        defer self.mutex.unlock();

        const doc_id = self.global_doc_id;
        self.global_doc_id += 1;

        // Tokenize
        var token_buf: [8192]byte_tokenizer.Token = undefined;
        var agg_buf: [4096]byte_tokenizer.Token = undefined;

        const tokenizer = byte_tokenizer.ByteTokenizer.init(.{ .lowercase = true });
        const result = byte_tokenizer.tokenizeAndAggregate(&tokenizer, text, &token_buf, &agg_buf);

        // Store
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

        // Check flush threshold
        if (self.doc_lengths.items.len >= self.config.flush_threshold) {
            try self.flushInternal();
        }

        return doc_id;
    }

    /// Add multiple documents (batch API)
    pub fn addDocuments(self: *Self, texts: []const []const u8) !void {
        for (texts) |text| {
            _ = try self.addDocument(text);
        }
    }

    /// Flush current buffer to a segment
    pub fn flush(self: *Self) !void {
        self.mutex.lock();
        defer self.mutex.unlock();
        try self.flushInternal();
    }

    fn flushInternal(self: *Self) !void {
        if (self.doc_lengths.items.len == 0) return;

        // Create segment ID
        const seg_id = self.segment_manager.nextSegmentId(0);
        var path_buf: [256]u8 = undefined;
        var name_buf: [64]u8 = undefined;
        const seg_name = seg_id.filename(&name_buf);

        const full_path = std.fmt.bufPrint(&path_buf, "{s}/{s}", .{ self.config.base_path, seg_name }) catch return;

        // Ensure directory exists
        std.fs.cwd().makePath(self.config.base_path) catch {};

        // Write segment
        var writer = try segment.SegmentWriter.create(
            self.allocator,
            full_path,
            self.config.profile,
            64 * 1024 * 1024, // 64MB initial size
        );

        // Sort terms by hash for binary search
        var term_hashes = ManagedArrayList(u64).init(self.allocator);
        defer term_hashes.deinit();

        var iter = self.term_postings.iterator();
        while (iter.next()) |entry| {
            try term_hashes.append(entry.key_ptr.*);
        }

        std.mem.sort(u64, term_hashes.items, {}, std.sort.asc(u64));

        // Write terms
        for (term_hashes.items) |hash| {
            const postings = self.term_postings.get(hash).?;
            try writer.writeTerm(hash, 0, @intCast(postings.items.len));
        }
        writer.markTermsEnd();

        // Write postings (simplified - just offsets for now)
        writer.markPostingsEnd();

        // Write doc metadata
        for (self.doc_lengths.items) |len| {
            try writer.writeDocMeta(len);
        }

        writer.close();

        // Clear buffer
        var clear_iter = self.term_postings.iterator();
        while (clear_iter.next()) |entry| {
            entry.value_ptr.clearRetainingCapacity();
        }
        self.doc_lengths.clearRetainingCapacity();
        self.total_tokens = 0;
    }

    /// Get statistics
    pub fn stats(self: Self) Stats {
        return .{
            .buffered_docs = @intCast(self.doc_lengths.items.len),
            .total_docs = self.global_doc_id,
            .segment_count = @intCast(self.segment_manager.segments.items.len),
            .buffered_tokens = self.total_tokens,
        };
    }

    pub const Stats = struct {
        buffered_docs: u32,
        total_docs: u32,
        segment_count: u32,
        buffered_tokens: u64,
    };
};

// ============================================================================
// Tests
// ============================================================================

test "writer basic" {
    const path = "/tmp/fts_zig_writer_test";

    // Cleanup
    std.fs.cwd().deleteTree(path) catch {};

    var writer = IndexWriter.init(std.testing.allocator, .{
        .base_path = path,
        .flush_threshold = 10,
        .profile = .speed,
    });
    defer writer.deinit();

    // Add documents
    _ = try writer.addDocument("hello world");
    _ = try writer.addDocument("hello there");

    const s = writer.stats();
    try std.testing.expectEqual(@as(u32, 2), s.buffered_docs);
    try std.testing.expectEqual(@as(u32, 2), s.total_docs);

    // Flush
    try writer.flush();

    const s2 = writer.stats();
    try std.testing.expectEqual(@as(u32, 0), s2.buffered_docs);

    // Cleanup
    std.fs.cwd().deleteTree(path) catch {};
}
