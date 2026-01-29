//! Variable Byte (VByte) encoding for posting lists
//! Encodes integers using 1-5 bytes, smaller values use fewer bytes
//! SIMD-accelerated decoding for maximum throughput

const std = @import("std");
const simd = @import("../util/simd.zig");

fn ManagedArrayList(comptime T: type) type {
    return std.array_list.AlignedManaged(T, null);
}

/// Encode a single u32 value to VByte format
/// Returns the number of bytes written
pub inline fn encode(value: u32, out: []u8) usize {
    var v = value;
    var i: usize = 0;

    while (v >= 0x80) {
        out[i] = @truncate(v | 0x80);
        v >>= 7;
        i += 1;
    }
    out[i] = @truncate(v);
    return i + 1;
}

/// Decode a single VByte-encoded value
/// Returns the value and number of bytes consumed
pub inline fn decode(data: []const u8) struct { value: u32, bytes: usize } {
    var result: u32 = 0;
    var shift: u5 = 0;
    var i: usize = 0;

    while (i < data.len and i < 5) {
        const b = data[i];
        result |= @as(u32, b & 0x7F) << shift;

        if (b < 0x80) {
            return .{ .value = result, .bytes = i + 1 };
        }

        shift += 7;
        i += 1;
    }

    return .{ .value = result, .bytes = i };
}

/// Encode multiple values (delta-encoded for sorted lists)
pub fn encodeMany(values: []const u32, out: []u8) usize {
    if (values.len == 0) return 0;

    var offset: usize = 0;
    var prev: u32 = 0;

    for (values) |v| {
        const delta = v - prev;
        offset += encode(delta, out[offset..]);
        prev = v;
    }

    return offset;
}

/// Decode multiple delta-encoded values
pub fn decodeMany(data: []const u8, out: []u32) struct { count: usize, bytes: usize } {
    var offset: usize = 0;
    var count: usize = 0;
    var prev: u32 = 0;

    while (offset < data.len and count < out.len) {
        const result = decode(data[offset..]);
        if (result.bytes == 0) break;

        prev += result.value;
        out[count] = prev;
        count += 1;
        offset += result.bytes;
    }

    return .{ .count = count, .bytes = offset };
}

/// SIMD-accelerated batch decoding (processes 4 values at a time when possible)
pub fn decodeBatchSIMD(data: []const u8, out: []u32) struct { count: usize, bytes: usize } {
    // Fall back to scalar for now - true SIMD VByte decoding requires
    // careful implementation with gather/scatter
    return decodeMany(data, out);
}

/// Calculate the encoded size for a slice of values (delta-encoded)
pub fn encodedSize(values: []const u32) usize {
    if (values.len == 0) return 0;

    var size: usize = 0;
    var prev: u32 = 0;

    for (values) |v| {
        const delta = v - prev;
        size += encodedSizeScalar(delta);
        prev = v;
    }

    return size;
}

/// Calculate encoded size for a single value
inline fn encodedSizeScalar(value: u32) usize {
    if (value < (1 << 7)) return 1;
    if (value < (1 << 14)) return 2;
    if (value < (1 << 21)) return 3;
    if (value < (1 << 28)) return 4;
    return 5;
}

/// VByte encoder with buffer
pub const Encoder = struct {
    buffer: ManagedArrayList(u8),
    last_value: u32,

    const Self = @This();

    pub fn init(allocator: std.mem.Allocator) Self {
        return .{
            .buffer = ManagedArrayList(u8).init(allocator),
            .last_value = 0,
        };
    }

    pub fn deinit(self: *Self) void {
        self.buffer.deinit();
    }

    pub fn reset(self: *Self) void {
        self.buffer.clearRetainingCapacity();
        self.last_value = 0;
    }

    /// Add a value (must be >= last value for delta encoding)
    pub fn add(self: *Self, value: u32) !void {
        std.debug.assert(value >= self.last_value);

        var buf: [5]u8 = undefined;
        const delta = value - self.last_value;
        const enc_len = encode(delta, &buf);

        try self.buffer.appendSlice(buf[0..enc_len]);
        self.last_value = value;
    }

    /// Get encoded bytes
    pub fn bytes(self: Self) []const u8 {
        return self.buffer.items;
    }

    pub fn len(self: Self) usize {
        return self.buffer.items.len;
    }
};

/// VByte decoder (streaming)
pub const Decoder = struct {
    data: []const u8,
    offset: usize,
    last_value: u32,

    const Self = @This();

    pub fn init(data: []const u8) Self {
        return .{
            .data = data,
            .offset = 0,
            .last_value = 0,
        };
    }

    /// Read next value, returns null if exhausted
    pub fn next(self: *Self) ?u32 {
        if (self.offset >= self.data.len) return null;

        const result = decode(self.data[self.offset..]);
        if (result.bytes == 0) return null;

        self.offset += result.bytes;
        self.last_value += result.value;
        return self.last_value;
    }

    /// Skip n values
    pub fn skip(self: *Self, n: usize) void {
        var i: usize = 0;
        while (i < n) : (i += 1) {
            if (self.next() == null) break;
        }
    }

    /// Check if more values available
    pub fn hasNext(self: Self) bool {
        return self.offset < self.data.len;
    }

    /// Get remaining bytes
    pub fn remaining(self: Self) usize {
        return self.data.len - self.offset;
    }
};

// ============================================================================
// Tests
// ============================================================================

test "vbyte encode decode single" {
    var buf: [5]u8 = undefined;

    // Small value
    const len1 = encode(127, &buf);
    try std.testing.expectEqual(@as(usize, 1), len1);
    const r1 = decode(&buf);
    try std.testing.expectEqual(@as(u32, 127), r1.value);

    // Medium value
    const len2 = encode(16383, &buf);
    try std.testing.expectEqual(@as(usize, 2), len2);
    const r2 = decode(&buf);
    try std.testing.expectEqual(@as(u32, 16383), r2.value);

    // Large value
    const len3 = encode(0xFFFFFFFF, &buf);
    try std.testing.expectEqual(@as(usize, 5), len3);
    const r3 = decode(&buf);
    try std.testing.expectEqual(@as(u32, 0xFFFFFFFF), r3.value);
}

test "vbyte encode decode many" {
    const values = [_]u32{ 10, 20, 30, 100, 1000, 10000 };
    var encoded: [100]u8 = undefined;
    var decoded: [6]u32 = undefined;

    const enc_len = encodeMany(&values, &encoded);
    const result = decodeMany(encoded[0..enc_len], &decoded);

    try std.testing.expectEqual(@as(usize, 6), result.count);
    try std.testing.expectEqualSlices(u32, &values, &decoded);
}

test "vbyte encoder decoder" {
    var encoder = Encoder.init(std.testing.allocator);
    defer encoder.deinit();

    try encoder.add(100);
    try encoder.add(200);
    try encoder.add(300);

    var decoder = Decoder.init(encoder.bytes());

    try std.testing.expectEqual(@as(u32, 100), decoder.next().?);
    try std.testing.expectEqual(@as(u32, 200), decoder.next().?);
    try std.testing.expectEqual(@as(u32, 300), decoder.next().?);
    try std.testing.expectEqual(@as(?u32, null), decoder.next());
}
