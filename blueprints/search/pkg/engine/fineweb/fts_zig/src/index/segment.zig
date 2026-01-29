//! Segment abstraction for streaming indexing
//! Segments are immutable once written, enabling lock-free reads

const std = @import("std");
const Allocator = std.mem.Allocator;
const mmap = @import("../util/mmap.zig");

fn ManagedArrayList(comptime T: type) type {
    return std.array_list.AlignedManaged(T, null);
}

/// Segment header (stored at start of segment file)
pub const SegmentHeader = extern struct {
    magic: [4]u8 = .{ 'F', 'T', 'S', 'Z' },
    version: u32 = 1,
    profile: u8, // 0=speed, 1=balanced, 2=compact
    _reserved: [3]u8 = .{ 0, 0, 0 },
    doc_count: u32,
    term_count: u32,
    total_tokens: u64,
    terms_offset: u64, // Offset to term dictionary
    postings_offset: u64, // Offset to posting lists
    docs_offset: u64, // Offset to document metadata
    index_size: u64, // Total segment size
};

/// Profile types
pub const Profile = enum(u8) {
    speed = 0,
    balanced = 1,
    compact = 2,
};

/// Segment ID
pub const SegmentId = struct {
    generation: u32,
    level: u8,
    sequence: u32,

    pub fn toString(self: SegmentId, buf: []u8) []u8 {
        return std.fmt.bufPrint(buf, "seg_{d}_{d}_{d}", .{ self.generation, self.level, self.sequence }) catch buf[0..0];
    }

    pub fn filename(self: SegmentId, buf: []u8) []u8 {
        return std.fmt.bufPrint(buf, "seg_{d}_{d}_{d}.fts", .{ self.generation, self.level, self.sequence }) catch buf[0..0];
    }
};

/// Segment writer for creating new segments
pub const SegmentWriter = struct {
    allocator: Allocator,
    writer: mmap.MappedFileWriter,
    header: SegmentHeader,
    terms_written: u32,
    docs_written: u32,

    const Self = @This();

    pub fn create(allocator: Allocator, path: []const u8, profile: Profile, initial_size: usize) !Self {
        var writer = try mmap.MappedFileWriter.create(path, initial_size);

        var header = SegmentHeader{
            .profile = @intFromEnum(profile),
            .doc_count = 0,
            .term_count = 0,
            .total_tokens = 0,
            .terms_offset = @sizeOf(SegmentHeader),
            .postings_offset = 0,
            .docs_offset = 0,
            .index_size = 0,
        };

        // Reserve space for header (will write final values at close)
        try writer.write(std.mem.asBytes(&header));

        return Self{
            .allocator = allocator,
            .writer = writer,
            .header = header,
            .terms_written = 0,
            .docs_written = 0,
        };
    }

    /// Write a term entry
    pub fn writeTerm(self: *Self, term_hash: u64, posting_offset: u64, doc_freq: u32) !void {
        const entry = TermEntry{
            .hash = term_hash,
            .posting_offset = posting_offset,
            .doc_freq = doc_freq,
        };
        try self.writer.write(std.mem.asBytes(&entry));
        self.terms_written += 1;
    }

    /// Write posting data
    pub fn writePostings(self: *Self, data: []const u8) !u64 {
        const offset = self.writer.position();
        try self.writer.write(data);
        return offset;
    }

    /// Write document metadata
    pub fn writeDocMeta(self: *Self, length: u32) !void {
        const meta = DocMetaEntry{ .length = length };
        try self.writer.write(std.mem.asBytes(&meta));
        self.docs_written += 1;
    }

    /// Mark positions of sections
    pub fn markTermsEnd(self: *Self) void {
        self.header.postings_offset = self.writer.position();
    }

    pub fn markPostingsEnd(self: *Self) void {
        self.header.docs_offset = self.writer.position();
    }

    /// Finalize and close the segment
    pub fn close(self: *Self) void {
        // Update header
        self.header.term_count = self.terms_written;
        self.header.doc_count = self.docs_written;
        self.header.index_size = self.writer.position();

        // Write header at start
        @memcpy(self.writer.data[0..@sizeOf(SegmentHeader)], std.mem.asBytes(&self.header));

        self.writer.close();
    }

    const TermEntry = extern struct {
        hash: u64,
        posting_offset: u64,
        doc_freq: u32,
        _padding: u32 = 0,
    };

    const DocMetaEntry = extern struct {
        length: u32,
    };
};

/// Segment reader for querying segments
pub const SegmentReader = struct {
    allocator: Allocator,
    mapped: mmap.MappedFile,
    header: SegmentHeader,

    const Self = @This();

    pub fn open(allocator: Allocator, path: []const u8) !Self {
        var mapped = try mmap.MappedFile.open(path);
        errdefer mapped.close();

        if (mapped.len < @sizeOf(SegmentHeader)) {
            return error.InvalidSegment;
        }

        const header = mapped.readAt(SegmentHeader, 0) orelse return error.InvalidSegment;

        // Verify magic
        if (!std.mem.eql(u8, &header.magic, "FTSZ")) {
            return error.InvalidMagic;
        }

        return Self{
            .allocator = allocator,
            .mapped = mapped,
            .header = header,
        };
    }

    pub fn close(self: *Self) void {
        self.mapped.close();
    }

    /// Get document count
    pub fn docCount(self: Self) u32 {
        return self.header.doc_count;
    }

    /// Get term count
    pub fn termCount(self: Self) u32 {
        return self.header.term_count;
    }

    /// Get profile
    pub fn profile(self: Self) Profile {
        return @enumFromInt(self.header.profile);
    }

    /// Lookup a term by hash
    pub fn lookupTerm(self: Self, term_hash: u64) ?TermInfo {
        const terms_start = self.header.terms_offset;
        const terms_end = self.header.postings_offset;
        const entry_size = @sizeOf(SegmentWriter.TermEntry);
        const term_count = (terms_end - terms_start) / entry_size;

        // Binary search
        var low: usize = 0;
        var high: usize = term_count;

        while (low < high) {
            const mid = low + (high - low) / 2;
            const offset = terms_start + mid * entry_size;
            const entry = self.mapped.readAt(SegmentWriter.TermEntry, offset) orelse return null;

            if (entry.hash == term_hash) {
                return TermInfo{
                    .posting_offset = entry.posting_offset,
                    .doc_freq = entry.doc_freq,
                };
            } else if (entry.hash < term_hash) {
                low = mid + 1;
            } else {
                high = mid;
            }
        }

        return null;
    }

    /// Get posting data slice
    pub fn getPostings(self: Self, offset: u64, length: usize) []const u8 {
        return self.mapped.slice(@intCast(offset), length);
    }

    /// Get document metadata
    pub fn getDocMeta(self: Self, doc_id: u32) ?DocMeta {
        const offset = self.header.docs_offset + doc_id * @sizeOf(SegmentWriter.DocMetaEntry);
        const entry = self.mapped.readAt(SegmentWriter.DocMetaEntry, offset) orelse return null;
        return DocMeta{ .length = entry.length };
    }

    pub const TermInfo = struct {
        posting_offset: u64,
        doc_freq: u32,
    };

    pub const DocMeta = struct {
        length: u32,
    };
};

/// Segment manager for handling multiple segments
pub const SegmentManager = struct {
    allocator: Allocator,
    base_path: []const u8,
    segments: ManagedArrayList(*SegmentReader),
    next_generation: u32,
    next_sequence: [4]u32, // Per-level sequence numbers

    const Self = @This();

    pub fn init(allocator: Allocator, base_path: []const u8) Self {
        return .{
            .allocator = allocator,
            .base_path = base_path,
            .segments = ManagedArrayList(*SegmentReader).init(allocator),
            .next_generation = 0,
            .next_sequence = .{ 0, 0, 0, 0 },
        };
    }

    pub fn deinit(self: *Self) void {
        for (self.segments.items) |seg| {
            seg.close();
            self.allocator.destroy(seg);
        }
        self.segments.deinit();
    }

    /// Generate next segment ID for a level
    pub fn nextSegmentId(self: *Self, level: u8) SegmentId {
        const seq = self.next_sequence[level];
        self.next_sequence[level] += 1;

        return .{
            .generation = self.next_generation,
            .level = level,
            .sequence = seq,
        };
    }

    /// Add a segment
    pub fn addSegment(self: *Self, segment: *SegmentReader) !void {
        try self.segments.append(segment);
    }

    /// Get total document count across all segments
    pub fn totalDocs(self: Self) u32 {
        var total: u32 = 0;
        for (self.segments.items) |seg| {
            total += seg.docCount();
        }
        return total;
    }
};

// ============================================================================
// Tests
// ============================================================================

test "segment write read" {
    const path = "/tmp/fts_zig_segment_test.fts";

    // Write segment
    {
        var writer = try SegmentWriter.create(std.testing.allocator, path, .speed, 4096);

        // Write some terms
        try writer.writeTerm(100, 0, 5);
        try writer.writeTerm(200, 100, 3);
        writer.markTermsEnd();

        // Write postings (dummy data)
        _ = try writer.writePostings(&[_]u8{ 1, 2, 3, 4 });
        writer.markPostingsEnd();

        // Write doc metadata
        try writer.writeDocMeta(50);
        try writer.writeDocMeta(75);

        writer.close();
    }

    // Read segment
    {
        var reader = try SegmentReader.open(std.testing.allocator, path);
        defer reader.close();

        try std.testing.expectEqual(@as(u32, 2), reader.termCount());
        try std.testing.expectEqual(@as(u32, 2), reader.docCount());
        try std.testing.expectEqual(Profile.speed, reader.profile());

        // Lookup term
        const term = reader.lookupTerm(100);
        try std.testing.expect(term != null);
        try std.testing.expectEqual(@as(u32, 5), term.?.doc_freq);

        // Get doc meta
        const doc = reader.getDocMeta(0);
        try std.testing.expect(doc != null);
        try std.testing.expectEqual(@as(u32, 50), doc.?.length);
    }

    // Cleanup
    std.fs.cwd().deleteFile(path) catch {};
}
