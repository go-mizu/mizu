//! SIMD utilities for high-performance text processing
//! Uses Zig's @Vector for portable SIMD operations

const std = @import("std");
const builtin = @import("builtin");

/// Vector width based on architecture
pub const VECTOR_WIDTH = if (builtin.cpu.arch.isX86())
    if (std.Target.x86.featureSetHas(builtin.cpu.features, .avx2)) 32 else 16
else if (builtin.cpu.arch.isAARCH64())
    16
else
    16;

/// Primary vector types
pub const Vec32u8 = @Vector(32, u8);
pub const Vec16u8 = @Vector(16, u8);
pub const Vec8u32 = @Vector(8, u32);
pub const Vec4u64 = @Vector(4, u64);
pub const Vec8f32 = @Vector(8, f32);
pub const Vec4f64 = @Vector(4, f64);

/// Character class masks for tokenization
pub const CharClass = struct {
    /// Check if bytes are ASCII whitespace
    pub inline fn isWhitespace(v: Vec32u8) @Vector(32, bool) {
        const space = v == @as(Vec32u8, @splat(' '));
        const tab = v == @as(Vec32u8, @splat('\t'));
        const newline = v == @as(Vec32u8, @splat('\n'));
        const cr = v == @as(Vec32u8, @splat('\r'));
        return space or tab or newline or cr;
    }

    /// Check if bytes are ASCII punctuation (common delimiters)
    pub inline fn isPunctuation(v: Vec32u8) @Vector(32, bool) {
        const period = v == @as(Vec32u8, @splat('.'));
        const comma = v == @as(Vec32u8, @splat(','));
        const semicolon = v == @as(Vec32u8, @splat(';'));
        const colon = v == @as(Vec32u8, @splat(':'));
        const question = v == @as(Vec32u8, @splat('?'));
        const exclaim = v == @as(Vec32u8, @splat('!'));
        const lparen = v == @as(Vec32u8, @splat('('));
        const rparen = v == @as(Vec32u8, @splat(')'));
        return period or comma or semicolon or colon or question or exclaim or lparen or rparen;
    }

    /// Check if bytes are token delimiters (whitespace or punctuation)
    pub inline fn isDelimiter(v: Vec32u8) @Vector(32, bool) {
        return isWhitespace(v) or isPunctuation(v);
    }

    /// Convert bool vector to bitmask
    pub inline fn toBitmask(v: @Vector(32, bool)) u32 {
        return @bitCast(v);
    }
};

/// Find positions of delimiters in a 32-byte chunk
pub inline fn findDelimiters32(chunk: *const [32]u8) u32 {
    const v: Vec32u8 = chunk.*;
    const mask = CharClass.isDelimiter(v);
    return CharClass.toBitmask(mask);
}

/// Find positions of delimiters in a 16-byte chunk (for non-AVX2)
pub inline fn findDelimiters16(chunk: *const [16]u8) u16 {
    const v: Vec16u8 = chunk.*;

    const space = v == @as(Vec16u8, @splat(' '));
    const tab = v == @as(Vec16u8, @splat('\t'));
    const newline = v == @as(Vec16u8, @splat('\n'));

    const mask = space or tab or newline;
    return @bitCast(mask);
}

/// Count trailing zeros (position of first set bit)
pub inline fn ctz(x: anytype) @TypeOf(x) {
    return @ctz(x);
}

/// Count leading zeros
pub inline fn clz(x: anytype) @TypeOf(x) {
    return @clz(x);
}

/// Population count (number of set bits)
pub inline fn popCount(x: anytype) @TypeOf(x) {
    return @popCount(x);
}

/// SIMD min for u32 vectors
pub inline fn min8u32(a: Vec8u32, b: Vec8u32) Vec8u32 {
    return @min(a, b);
}

/// SIMD max for u32 vectors
pub inline fn max8u32(a: Vec8u32, b: Vec8u32) Vec8u32 {
    return @max(a, b);
}

/// Horizontal sum of f32 vector
pub inline fn horizontalSum8f32(v: Vec8f32) f32 {
    return @reduce(.Add, v);
}

/// Horizontal max of f32 vector
pub inline fn horizontalMax8f32(v: Vec8f32) f32 {
    return @reduce(.Max, v);
}

/// Load 8 u32 values, handling potential alignment
pub inline fn load8u32(ptr: [*]const u32) Vec8u32 {
    return ptr[0..8].*;
}

/// Store 8 u32 values
pub inline fn store8u32(ptr: [*]u32, v: Vec8u32) void {
    ptr[0..8].* = v;
}

/// Gather operation (load from non-contiguous addresses)
/// Note: True gather isn't available in Zig vectors, so we do scalar loads
pub inline fn gather8u32(base: [*]const u32, indices: Vec8u32) Vec8u32 {
    var result: [8]u32 = undefined;
    const idx_array: [8]u32 = indices;
    inline for (0..8) |i| {
        result[i] = base[idx_array[i]];
    }
    return result;
}

/// Prefetch memory for reading
pub inline fn prefetchRead(ptr: anytype) void {
    const addr: [*]const u8 = @ptrCast(ptr);
    @prefetch(addr, .{ .rw = .read, .locality = 3, .cache = .data });
}

/// Prefetch memory for writing
pub inline fn prefetchWrite(ptr: anytype) void {
    const addr: [*]u8 = @ptrCast(@constCast(ptr));
    @prefetch(addr, .{ .rw = .write, .locality = 3, .cache = .data });
}

/// BM25 scoring for 8 documents simultaneously
pub const BM25SIMD = struct {
    k1: f32,
    b: f32,
    avg_dl: f32,

    pub fn init(k1: f32, b: f32, avg_dl: f32) BM25SIMD {
        return .{ .k1 = k1, .b = b, .avg_dl = avg_dl };
    }

    /// Score 8 documents at once
    /// tf: term frequencies, dl: document lengths, idf: inverse document frequency
    pub inline fn score8(self: BM25SIMD, tf: Vec8f32, dl: Vec8f32, idf: f32) Vec8f32 {
        const k1_vec: Vec8f32 = @splat(self.k1);
        const b_vec: Vec8f32 = @splat(self.b);
        const one_minus_b: Vec8f32 = @splat(1.0 - self.b);
        const avg_dl_vec: Vec8f32 = @splat(self.avg_dl);
        const idf_vec: Vec8f32 = @splat(idf);

        // norm = 1 - b + b * (dl / avg_dl)
        const norm = one_minus_b + b_vec * (dl / avg_dl_vec);

        // score = idf * tf / (tf + k1 * norm)
        const denom = tf + k1_vec * norm;
        return idf_vec * tf / denom;
    }
};

test "simd delimiter detection" {
    const chunk: [32]u8 = "hello world, this is a test!   ".*;
    const mask = findDelimiters32(&chunk);

    // Positions 5 (space), 11 (,), 12 (space), 17 (space), 20 (space), 22 (space), 27 (!), 28-31 (spaces)
    try std.testing.expect(mask & (1 << 5) != 0); // space after "hello"
    try std.testing.expect(mask & (1 << 11) != 0); // comma
}

test "simd bm25 scoring" {
    const bm25 = BM25SIMD.init(1.2, 0.75, 100.0);

    const tf: Vec8f32 = .{ 1, 2, 3, 4, 5, 1, 2, 3 };
    const dl: Vec8f32 = .{ 100, 100, 100, 100, 100, 50, 150, 200 };
    const idf: f32 = 2.0;

    const scores = bm25.score8(tf, dl, idf);
    const scores_arr: [8]f32 = scores;

    // All scores should be positive
    for (scores_arr) |s| {
        try std.testing.expect(s > 0);
    }
}
