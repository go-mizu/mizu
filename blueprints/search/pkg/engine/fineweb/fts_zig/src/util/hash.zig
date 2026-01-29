//! Fast hashing using wyhash algorithm
//! Optimized for short strings (typical tokens)

const std = @import("std");

/// wyhash constants
const wyp0: u64 = 0xa0761d6478bd642f;
const wyp1: u64 = 0xe7037ed1a0b428db;
const wyp2: u64 = 0x8ebc6af09c88c6e3;
const wyp3: u64 = 0x589965cc75374cc3;

/// Read 8 bytes as u64, handling unaligned access
inline fn read64(ptr: [*]const u8) u64 {
    return std.mem.readInt(u64, ptr[0..8], .little);
}

/// Read 4 bytes as u64
inline fn read32(ptr: [*]const u8) u64 {
    return @as(u64, std.mem.readInt(u32, ptr[0..4], .little));
}

/// Multiply and xor high/low
inline fn wymix(a: u64, b: u64) u64 {
    const r = @as(u128, a) *% @as(u128, b);
    return @truncate(r ^ (r >> 64));
}

/// Fast wyhash for arbitrary length data
pub fn hash(data: []const u8) u64 {
    return hashWithSeed(data, 0);
}

/// wyhash with seed
pub fn hashWithSeed(data: []const u8, seed: u64) u64 {
    const len = data.len;
    const ptr = data.ptr;

    var a: u64 = undefined;
    var b: u64 = undefined;

    if (len <= 16) {
        if (len >= 4) {
            a = (read32(ptr) << 32) | read32(ptr + ((len >> 3) << 2));
            b = (read32(ptr + len - 4) << 32) | read32(ptr + len - 4 - ((len >> 3) << 2));
        } else if (len > 0) {
            a = @as(u64, ptr[0]) << 16 | @as(u64, ptr[len >> 1]) << 8 | @as(u64, ptr[len - 1]);
            b = 0;
        } else {
            a = 0;
            b = 0;
        }
    } else {
        var i: usize = len;
        var p = ptr;
        var se = seed;

        if (i > 48) {
            var see1 = se;
            var see2 = se;

            while (i > 48) {
                se = wymix(read64(p) ^ wyp1, read64(p + 8) ^ se);
                see1 = wymix(read64(p + 16) ^ wyp2, read64(p + 24) ^ see1);
                see2 = wymix(read64(p + 32) ^ wyp3, read64(p + 40) ^ see2);
                p += 48;
                i -= 48;
            }
            se ^= see1 ^ see2;
        }

        while (i > 16) {
            se = wymix(read64(p) ^ wyp1, read64(p + 8) ^ se);
            p += 16;
            i -= 16;
        }

        a = read64(p + i - 16);
        b = read64(p + i - 8);
    }

    return wymix(wyp1 ^ @as(u64, len), wymix(a ^ wyp1, b ^ seed));
}

/// Hash a token (optimized for short strings < 32 bytes)
pub inline fn hashToken(token: []const u8) u64 {
    return hash(token);
}

/// Combine two hashes (for multi-word phrases)
pub inline fn combineHashes(h1: u64, h2: u64) u64 {
    return wymix(h1, h2);
}

test "hash basic" {
    const h1 = hash("hello");
    const h2 = hash("world");
    const h3 = hash("hello");

    try std.testing.expect(h1 != h2);
    try std.testing.expectEqual(h1, h3);
}

test "hash empty" {
    const h1 = hash("");
    const h2 = hash("");
    // Empty string should hash consistently
    try std.testing.expectEqual(h1, h2);
}

test "hash vietnamese" {
    const h1 = hash("Việt Nam");
    const h2 = hash("việt nam");
    try std.testing.expect(h1 != h2); // Case sensitive
}
