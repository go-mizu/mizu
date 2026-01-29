//! fts_zig: High-performance full-text search in Zig
//! Target: 1M docs/sec indexing with streaming/incremental support
//!
//! Profiles:
//! - speed: Raw arrays, no compression, <1ms p99 search
//! - balanced: Block-Max WAND + VByte, 1-10ms p99 search
//! - compact: Elias-Fano encoding, 10-50ms p99 search

const std = @import("std");

// Re-export core modules
pub const tokenizer = struct {
    pub const byte = @import("tokenizer/byte.zig");
    pub const unicode = @import("tokenizer/unicode.zig");
    pub const vietnamese = @import("tokenizer/vietnamese.zig");
};

pub const codec = struct {
    pub const vbyte = @import("codec/vbyte.zig");
    pub const eliasfano = @import("codec/eliasfano.zig");
    pub const fst = @import("codec/fst.zig");
};

pub const profile = struct {
    pub const speed = @import("profile/speed.zig");
    pub const balanced = @import("profile/balanced.zig");
    pub const compact = @import("profile/compact.zig");
};

pub const search = struct {
    pub const query = @import("search/query.zig");
    pub const scorer = @import("search/scorer.zig");
    pub const collector = @import("search/collector.zig");
};

pub const index = struct {
    pub const segment = @import("index/segment.zig");
    pub const writer = @import("index/writer.zig");
    pub const merger = @import("index/merger.zig");
    pub const manager = @import("index/manager.zig");
};

pub const util = struct {
    pub const hash = @import("util/hash.zig");
    pub const simd = @import("util/simd.zig");
    pub const arena = @import("util/arena.zig");
    pub const mmap = @import("util/mmap.zig");
};

// FFI exports
pub const ffi = @import("ffi.zig");

/// Version information
pub const version = struct {
    pub const major = 0;
    pub const minor = 1;
    pub const patch = 0;
    pub const string = "0.1.0";
};

/// Profile selection
pub const Profile = index.segment.Profile;

/// Create a new index with the specified profile
pub fn createIndex(allocator: std.mem.Allocator, base_path: []const u8, prof: Profile) !*index.manager.IndexManager {
    const manager = try allocator.create(index.manager.IndexManager);
    manager.* = index.manager.IndexManager.init(allocator, .{
        .base_path = base_path,
        .profile = prof,
    });
    return manager;
}

/// Create an in-memory speed index (for benchmarking)
pub fn createSpeedIndex(allocator: std.mem.Allocator) profile.speed.SpeedIndexBuilder {
    return profile.speed.SpeedIndexBuilder.init(allocator);
}

/// Create an in-memory balanced index
pub fn createBalancedIndex(allocator: std.mem.Allocator) profile.balanced.BalancedIndexBuilder {
    return profile.balanced.BalancedIndexBuilder.init(allocator);
}

/// Create an in-memory compact index
pub fn createCompactIndex(allocator: std.mem.Allocator) profile.compact.CompactIndexBuilder {
    return profile.compact.CompactIndexBuilder.init(allocator);
}

// ============================================================================
// Tests
// ============================================================================

test "main exports" {
    // Verify all modules compile
    _ = tokenizer.byte;
    _ = tokenizer.unicode;
    _ = tokenizer.vietnamese;
    _ = codec.vbyte;
    _ = codec.eliasfano;
    _ = codec.fst;
    _ = profile.speed;
    _ = profile.balanced;
    _ = profile.compact;
    _ = search.query;
    _ = search.scorer;
    _ = search.collector;
    _ = index.segment;
    _ = index.writer;
    _ = index.merger;
    _ = index.manager;
    _ = util.hash;
    _ = util.simd;
    _ = util.arena;
    _ = util.mmap;
}

test "version" {
    try std.testing.expectEqualStrings("0.1.0", version.string);
}

test "create speed index" {
    var builder = createSpeedIndex(std.testing.allocator);
    defer builder.deinit();

    _ = try builder.addDocument("hello world");
    _ = try builder.addDocument("test document");

    var idx = try builder.build();
    defer idx.deinit();

    try std.testing.expectEqual(@as(u32, 2), idx.docCount());
}
