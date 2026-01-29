//! C ABI exports for FFI integration (CGO, etc.)
//! All functions use C calling convention and simple types

const std = @import("std");
const main = @import("main.zig");

/// Opaque handle to an index
pub const IndexHandle = *anyopaque;

/// Search result for FFI
pub const FFISearchResult = extern struct {
    doc_id: u32,
    score: f32,
};

/// Index statistics
pub const FFIStats = extern struct {
    doc_count: u32,
    term_count: u32,
    memory_bytes: u64,
};

/// Error codes
pub const FFIError = enum(i32) {
    ok = 0,
    invalid_handle = -1,
    allocation_failed = -2,
    io_error = -3,
    invalid_argument = -4,
    not_found = -5,
};

// Global allocator for FFI
var gpa = std.heap.GeneralPurposeAllocator(.{}){};
const allocator = gpa.allocator();

// ============================================================================
// Speed Profile FFI
// ============================================================================

/// Create a new speed index builder
export fn fts_speed_builder_create() ?IndexHandle {
    const builder = allocator.create(main.profile.speed.SpeedIndexBuilder) catch return null;
    builder.* = main.profile.speed.SpeedIndexBuilder.init(allocator);
    return @ptrCast(builder);
}

/// Add a document to the speed index builder
export fn fts_speed_builder_add(handle: IndexHandle, text: [*]const u8, text_len: usize) i32 {
    const builder: *main.profile.speed.SpeedIndexBuilder = @ptrCast(@alignCast(handle));
    _ = builder.addDocument(text[0..text_len]) catch return @intFromEnum(FFIError.allocation_failed);
    return @intFromEnum(FFIError.ok);
}

/// Build the speed index from builder
export fn fts_speed_builder_build(handle: IndexHandle) ?IndexHandle {
    const builder: *main.profile.speed.SpeedIndexBuilder = @ptrCast(@alignCast(handle));
    const idx = allocator.create(main.profile.speed.SpeedIndex) catch return null;
    idx.* = builder.build() catch return null;
    return @ptrCast(idx);
}

/// Destroy a speed index builder
export fn fts_speed_builder_destroy(handle: IndexHandle) void {
    const builder: *main.profile.speed.SpeedIndexBuilder = @ptrCast(@alignCast(handle));
    builder.deinit();
    allocator.destroy(builder);
}

/// Search the speed index
export fn fts_speed_search(
    handle: IndexHandle,
    query: [*]const u8,
    query_len: usize,
    results: [*]FFISearchResult,
    max_results: usize,
) i32 {
    const idx: *main.profile.speed.SpeedIndex = @ptrCast(@alignCast(handle));
    const search_results = idx.search(query[0..query_len], max_results) catch {
        return 0;
    };
    defer idx.allocator.free(search_results);

    const count = @min(search_results.len, max_results);
    for (search_results[0..count], 0..) |r, i| {
        results[i] = .{ .doc_id = r.doc_id, .score = r.score };
    }

    return @intCast(count);
}

/// Get speed index statistics
export fn fts_speed_stats(handle: IndexHandle, stats: *FFIStats) void {
    const idx: *main.profile.speed.SpeedIndex = @ptrCast(@alignCast(handle));
    stats.doc_count = idx.docCount();
    stats.term_count = idx.termCount();
    stats.memory_bytes = idx.memoryUsage();
}

/// Destroy a speed index
export fn fts_speed_destroy(handle: IndexHandle) void {
    const idx: *main.profile.speed.SpeedIndex = @ptrCast(@alignCast(handle));
    idx.deinit();
    allocator.destroy(idx);
}

// ============================================================================
// Balanced Profile FFI
// ============================================================================

/// Create a new balanced index builder
export fn fts_balanced_builder_create() ?IndexHandle {
    const builder = allocator.create(main.profile.balanced.BalancedIndexBuilder) catch return null;
    builder.* = main.profile.balanced.BalancedIndexBuilder.init(allocator);
    return @ptrCast(builder);
}

/// Add a document to the balanced index builder
export fn fts_balanced_builder_add(handle: IndexHandle, text: [*]const u8, text_len: usize) i32 {
    const builder: *main.profile.balanced.BalancedIndexBuilder = @ptrCast(@alignCast(handle));
    _ = builder.addDocument(text[0..text_len]) catch return @intFromEnum(FFIError.allocation_failed);
    return @intFromEnum(FFIError.ok);
}

/// Build the balanced index from builder
export fn fts_balanced_builder_build(handle: IndexHandle) ?IndexHandle {
    const builder: *main.profile.balanced.BalancedIndexBuilder = @ptrCast(@alignCast(handle));
    const idx = allocator.create(main.profile.balanced.BalancedIndex) catch return null;
    idx.* = builder.build() catch return null;
    return @ptrCast(idx);
}

/// Destroy a balanced index builder
export fn fts_balanced_builder_destroy(handle: IndexHandle) void {
    const builder: *main.profile.balanced.BalancedIndexBuilder = @ptrCast(@alignCast(handle));
    builder.deinit();
    allocator.destroy(builder);
}

/// Search the balanced index
export fn fts_balanced_search(
    handle: IndexHandle,
    query: [*]const u8,
    query_len: usize,
    results: [*]FFISearchResult,
    max_results: usize,
) i32 {
    const idx: *main.profile.balanced.BalancedIndex = @ptrCast(@alignCast(handle));
    const search_results = idx.search(query[0..query_len], max_results) catch {
        return 0;
    };
    defer idx.allocator.free(search_results);

    const count = @min(search_results.len, max_results);
    for (search_results[0..count], 0..) |r, i| {
        results[i] = .{ .doc_id = r.doc_id, .score = r.score };
    }

    return @intCast(count);
}

/// Destroy a balanced index
export fn fts_balanced_destroy(handle: IndexHandle) void {
    const idx: *main.profile.balanced.BalancedIndex = @ptrCast(@alignCast(handle));
    idx.deinit();
    allocator.destroy(idx);
}

// ============================================================================
// Compact Profile FFI
// ============================================================================

/// Create a new compact index builder
export fn fts_compact_builder_create() ?IndexHandle {
    const builder = allocator.create(main.profile.compact.CompactIndexBuilder) catch return null;
    builder.* = main.profile.compact.CompactIndexBuilder.init(allocator);
    return @ptrCast(builder);
}

/// Add a document to the compact index builder
export fn fts_compact_builder_add(handle: IndexHandle, text: [*]const u8, text_len: usize) i32 {
    const builder: *main.profile.compact.CompactIndexBuilder = @ptrCast(@alignCast(handle));
    _ = builder.addDocument(text[0..text_len]) catch return @intFromEnum(FFIError.allocation_failed);
    return @intFromEnum(FFIError.ok);
}

/// Build the compact index from builder
export fn fts_compact_builder_build(handle: IndexHandle) ?IndexHandle {
    const builder: *main.profile.compact.CompactIndexBuilder = @ptrCast(@alignCast(handle));
    const idx = allocator.create(main.profile.compact.CompactIndex) catch return null;
    idx.* = builder.build() catch return null;
    return @ptrCast(idx);
}

/// Destroy a compact index builder
export fn fts_compact_builder_destroy(handle: IndexHandle) void {
    const builder: *main.profile.compact.CompactIndexBuilder = @ptrCast(@alignCast(handle));
    builder.deinit();
    allocator.destroy(builder);
}

/// Search the compact index
export fn fts_compact_search(
    handle: IndexHandle,
    query: [*]const u8,
    query_len: usize,
    results: [*]FFISearchResult,
    max_results: usize,
) i32 {
    const idx: *main.profile.compact.CompactIndex = @ptrCast(@alignCast(handle));
    const search_results = idx.search(query[0..query_len], max_results) catch {
        return 0;
    };
    defer idx.allocator.free(search_results);

    const count = @min(search_results.len, max_results);
    for (search_results[0..count], 0..) |r, i| {
        results[i] = .{ .doc_id = r.doc_id, .score = r.score };
    }

    return @intCast(count);
}

/// Destroy a compact index
export fn fts_compact_destroy(handle: IndexHandle) void {
    const idx: *main.profile.compact.CompactIndex = @ptrCast(@alignCast(handle));
    idx.deinit();
    allocator.destroy(idx);
}

// ============================================================================
// Utility FFI
// ============================================================================

/// Get library version
export fn fts_version() [*:0]const u8 {
    return "0.1.0";
}

/// Hash a string (for debugging)
export fn fts_hash(text: [*]const u8, text_len: usize) u64 {
    return main.util.hash.hash(text[0..text_len]);
}
