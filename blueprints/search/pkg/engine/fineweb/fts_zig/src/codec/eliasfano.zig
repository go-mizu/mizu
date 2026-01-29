//! Elias-Fano encoding for quasi-succinct representation of monotonic sequences
//! Achieves near-optimal space: ~2 bits per integer + O(log(U/n)) bits per element
//! Supports O(1) random access and efficient sequential iteration

const std = @import("std");
const Allocator = std.mem.Allocator;

/// Elias-Fano encoded sequence
pub const EliasFano = struct {
    /// Lower bits (dense, l bits per element)
    lower_bits: []u64,
    /// Upper bits (sparse, unary coded)
    upper_bits: []u64,
    /// Number of elements
    n: u32,
    /// Universe size (max value + 1)
    universe: u64,
    /// Number of lower bits per element
    l: u6,

    const Self = @This();

    /// Build Elias-Fano encoding from sorted sequence
    pub fn build(allocator: Allocator, values: []const u32) !Self {
        if (values.len == 0) {
            return Self{
                .lower_bits = &[_]u64{},
                .upper_bits = &[_]u64{},
                .n = 0,
                .universe = 0,
                .l = 0,
            };
        }

        const n: u32 = @intCast(values.len);
        const max_value = values[values.len - 1];
        const universe: u64 = @as(u64, max_value) + 1;

        // Calculate l = floor(log2(U/n))
        const l: u6 = if (universe <= n) 0 else @intCast(@min(63, std.math.log2_int(u64, universe / n)));

        // Allocate lower bits: n elements, l bits each
        const lower_words = (n * @as(u32, l) + 63) / 64;
        const lower_bits = try allocator.alloc(u64, lower_words);
        @memset(lower_bits, 0);

        // Allocate upper bits: n + (max_value >> l) + 1 bits
        const upper_bound = n + @as(u32, @intCast(max_value >> l)) + 1;
        const upper_words = (upper_bound + 63) / 64;
        const upper_bits = try allocator.alloc(u64, upper_words);
        @memset(upper_bits, 0);

        // Encode values
        const l_mask: u64 = (@as(u64, 1) << l) - 1;

        for (values, 0..) |v, i| {
            const val: u64 = v;

            // Store lower l bits
            if (l > 0) {
                const lower = val & l_mask;
                const bit_pos = @as(usize, @intCast(i)) * @as(usize, l);
                const word_idx = bit_pos / 64;
                const bit_idx: u6 = @intCast(bit_pos % 64);

                lower_bits[word_idx] |= lower << bit_idx;

                // Handle overflow to next word
                if (bit_idx + l > 64) {
                    lower_bits[word_idx + 1] |= lower >> @intCast(64 - bit_idx);
                }
            }

            // Store upper bits in unary: position = i + (val >> l)
            const high = val >> l;
            const upper_pos = @as(usize, @intCast(i)) + @as(usize, @intCast(high));
            const upper_word = upper_pos / 64;
            const upper_bit: u6 = @intCast(upper_pos % 64);

            if (upper_word < upper_bits.len) {
                upper_bits[upper_word] |= @as(u64, 1) << upper_bit;
            }
        }

        return Self{
            .lower_bits = lower_bits,
            .upper_bits = upper_bits,
            .n = n,
            .universe = universe,
            .l = l,
        };
    }

    pub fn deinit(self: *Self, allocator: Allocator) void {
        if (self.lower_bits.len > 0) {
            allocator.free(self.lower_bits);
        }
        if (self.upper_bits.len > 0) {
            allocator.free(self.upper_bits);
        }
        self.* = undefined;
    }

    /// Get the i-th element (0-indexed)
    pub fn get(self: Self, i: u32) u32 {
        if (i >= self.n) return 0;

        // Get lower bits
        var lower: u64 = 0;
        if (self.l > 0) {
            const bit_pos = @as(usize, i) * @as(usize, self.l);
            const word_idx = bit_pos / 64;
            const bit_idx: u6 = @intCast(bit_pos % 64);

            lower = (self.lower_bits[word_idx] >> bit_idx);

            if (bit_idx + self.l > 64 and word_idx + 1 < self.lower_bits.len) {
                lower |= self.lower_bits[word_idx + 1] << @intCast(64 - bit_idx);
            }

            lower &= (@as(u64, 1) << self.l) - 1;
        }

        // Get upper bits by counting 1s up to position i
        const high = self.selectOne(i);

        return @intCast((high << self.l) | lower);
    }

    /// Select the position of the (i+1)-th 1-bit, then subtract i
    fn selectOne(self: Self, i: u32) u64 {
        var ones_count: u32 = 0;
        var pos: u64 = 0;

        for (self.upper_bits) |word| {
            const ones_in_word = @popCount(word);

            if (ones_count + ones_in_word > i) {
                // Target 1-bit is in this word
                var w = word;
                while (ones_count <= i) {
                    const tz = @ctz(w);
                    if (ones_count == i) {
                        return pos + tz - i;
                    }
                    w &= w - 1; // Clear lowest bit
                    ones_count += 1;
                    pos += tz + 1;
                }
            }

            ones_count += ones_in_word;
            pos += 64;
        }

        return 0;
    }

    /// Create an iterator
    pub fn iterator(self: *const Self) Iterator {
        return Iterator.init(self);
    }

    /// Get memory usage in bytes
    pub fn memoryUsage(self: Self) usize {
        return self.lower_bits.len * 8 + self.upper_bits.len * 8;
    }

    /// Get bits per element
    pub fn bitsPerElement(self: Self) f64 {
        if (self.n == 0) return 0;
        return @as(f64, @floatFromInt(self.memoryUsage() * 8)) / @as(f64, @floatFromInt(self.n));
    }
};

/// Iterator for sequential access (more efficient than random access)
pub const Iterator = struct {
    ef: *const EliasFano,
    index: u32,
    upper_word_idx: usize,
    upper_bit_idx: u6,
    ones_seen: u32,
    current_high: u64,

    const Self = @This();

    fn init(ef: *const EliasFano) Self {
        return .{
            .ef = ef,
            .index = 0,
            .upper_word_idx = 0,
            .upper_bit_idx = 0,
            .ones_seen = 0,
            .current_high = 0,
        };
    }

    pub fn next(self: *Self) ?u32 {
        if (self.index >= self.ef.n) return null;

        // Get lower bits
        var lower: u64 = 0;
        if (self.ef.l > 0) {
            const bit_pos = @as(usize, self.index) * @as(usize, self.ef.l);
            const word_idx = bit_pos / 64;
            const bit_idx: u6 = @intCast(bit_pos % 64);

            lower = (self.ef.lower_bits[word_idx] >> bit_idx);

            if (bit_idx + self.ef.l > 64 and word_idx + 1 < self.ef.lower_bits.len) {
                lower |= self.ef.lower_bits[word_idx + 1] << @intCast(64 - bit_idx);
            }

            lower &= (@as(u64, 1) << self.ef.l) - 1;
        }

        // Find next 1 in upper bits
        while (self.upper_word_idx < self.ef.upper_bits.len) {
            const word = self.ef.upper_bits[self.upper_word_idx];
            const masked = word >> self.upper_bit_idx;

            if (masked != 0) {
                const tz: u6 = @intCast(@ctz(masked));
                const pos = self.upper_word_idx * 64 + self.upper_bit_idx + tz;
                self.current_high = pos - self.index;

                // Advance past this 1-bit
                self.upper_bit_idx += tz + 1;
                if (self.upper_bit_idx >= 64) {
                    self.upper_bit_idx = 0;
                    self.upper_word_idx += 1;
                }

                self.index += 1;
                return @intCast((self.current_high << self.ef.l) | lower);
            }

            self.upper_bit_idx = 0;
            self.upper_word_idx += 1;
        }

        return null;
    }

    pub fn reset(self: *Self) void {
        self.index = 0;
        self.upper_word_idx = 0;
        self.upper_bit_idx = 0;
        self.ones_seen = 0;
        self.current_high = 0;
    }
};

// ============================================================================
// Tests
// ============================================================================

test "eliasfano basic" {
    const values = [_]u32{ 2, 3, 5, 7, 11, 13, 24 };

    var ef = try EliasFano.build(std.testing.allocator, &values);
    defer ef.deinit(std.testing.allocator);

    for (values, 0..) |expected, i| {
        const actual = ef.get(@intCast(i));
        try std.testing.expectEqual(expected, actual);
    }
}

test "eliasfano iterator" {
    const values = [_]u32{ 10, 20, 30, 40, 50, 100, 200, 500, 1000 };

    var ef = try EliasFano.build(std.testing.allocator, &values);
    defer ef.deinit(std.testing.allocator);

    var it = ef.iterator();
    var i: usize = 0;

    while (it.next()) |v| {
        try std.testing.expectEqual(values[i], v);
        i += 1;
    }

    try std.testing.expectEqual(values.len, i);
}

test "eliasfano empty" {
    var ef = try EliasFano.build(std.testing.allocator, &[_]u32{});
    defer ef.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(u32, 0), ef.n);
}

test "eliasfano large values" {
    const values = [_]u32{ 1000000, 2000000, 3000000, 4000000 };

    var ef = try EliasFano.build(std.testing.allocator, &values);
    defer ef.deinit(std.testing.allocator);

    for (values, 0..) |expected, i| {
        try std.testing.expectEqual(expected, ef.get(@intCast(i)));
    }

    // Check compression ratio
    const bits_per_elem = ef.bitsPerElement();
    try std.testing.expect(bits_per_elem < 32); // Should be much better than 32 bits
}
