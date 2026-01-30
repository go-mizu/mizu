//! Byte-level SIMD tokenizer for maximum throughput
//! Target: ~2GB/sec on modern CPUs
//!
//! Design:
//! - Process 32 bytes at a time using SIMD
//! - Hash tokens directly without allocation
//! - Language-agnostic (works for Vietnamese byte sequences)

const std = @import("std");
const simd = @import("../util/simd.zig");
const hash = @import("../util/hash.zig");

/// Token with hash and position information
pub const Token = struct {
    hash: u64,
    start: u32,
    len: u16,
    freq: u16, // Term frequency (for aggregation)
};

/// Batch of tokens from a document
pub const TokenBatch = struct {
    tokens: []Token,
    doc_len: u32, // Total token count (for BM25 normalization)
};

/// Configuration for tokenizer
pub const Config = struct {
    /// Minimum token length (skip shorter tokens)
    min_length: u8 = 1,
    /// Maximum token length (truncate longer tokens)
    max_length: u16 = 256,
    /// Convert to lowercase (ASCII only for speed)
    lowercase: bool = true,
};

/// SIMD-accelerated byte tokenizer
pub const ByteTokenizer = struct {
    config: Config,

    const Self = @This();

    pub fn init(config: Config) Self {
        return .{ .config = config };
    }

    /// Tokenize text into pre-allocated buffer
    /// Returns slice of tokens actually written
    pub fn tokenize(self: Self, text: []const u8, out_tokens: []Token) TokenBatch {
        if (text.len == 0) {
            return .{ .tokens = out_tokens[0..0], .doc_len = 0 };
        }

        var token_count: usize = 0;
        var token_start: usize = 0;
        var in_token = false;
        var i: usize = 0;

        // Process 32 bytes at a time with SIMD
        while (i + 32 <= text.len and token_count < out_tokens.len) {
            const chunk: *const [32]u8 = @ptrCast(text.ptr + i);
            const delim_mask = simd.findDelimiters32(chunk);

            if (delim_mask == 0) {
                // No delimiters in this chunk
                if (!in_token) {
                    token_start = i;
                    in_token = true;
                }
                i += 32;
                continue;
            }

            // If not in a token and first char is not a delimiter, start a token
            if (!in_token and (delim_mask & 1) == 0) {
                token_start = i;
                in_token = true;
            }

            // Process each byte position with a delimiter
            var mask = delim_mask;

            while (mask != 0) {
                const pos: usize = @ctz(mask);
                const abs_pos = i + pos;

                if (in_token and abs_pos > token_start) {
                    // End of token
                    const token_len = abs_pos - token_start;
                    if (token_len >= self.config.min_length and token_len <= self.config.max_length) {
                        const token_text = text[token_start..abs_pos];
                        out_tokens[token_count] = .{
                            .hash = self.hashToken(token_text),
                            .start = @intCast(token_start),
                            .len = @intCast(token_len),
                            .freq = 1,
                        };
                        token_count += 1;
                        if (token_count >= out_tokens.len) break;
                    }
                    in_token = false;
                }

                // Check if next position (after this delimiter) starts a new token
                const next_in_chunk = pos + 1;
                if (next_in_chunk < 32) {
                    const next_abs = i + next_in_chunk;
                    if (next_abs < text.len and !isDelimiter(text[next_abs])) {
                        token_start = next_abs;
                        in_token = true;
                    }
                }

                // Clear this bit and continue
                mask &= mask - 1;
            }

            i += 32;
        }

        // Handle remaining bytes (scalar fallback)
        while (i < text.len and token_count < out_tokens.len) {
            const c = text[i];

            if (isDelimiter(c)) {
                if (in_token and i > token_start) {
                    const token_len = i - token_start;
                    if (token_len >= self.config.min_length and token_len <= self.config.max_length) {
                        const token_text = text[token_start..i];
                        out_tokens[token_count] = .{
                            .hash = self.hashToken(token_text),
                            .start = @intCast(token_start),
                            .len = @intCast(token_len),
                            .freq = 1,
                        };
                        token_count += 1;
                    }
                    in_token = false;
                }
            } else if (!in_token) {
                token_start = i;
                in_token = true;
            }

            i += 1;
        }

        // Handle final token
        if (in_token and i > token_start and token_count < out_tokens.len) {
            const token_len = i - token_start;
            if (token_len >= self.config.min_length and token_len <= self.config.max_length) {
                const token_text = text[token_start..i];
                out_tokens[token_count] = .{
                    .hash = self.hashToken(token_text),
                    .start = @intCast(token_start),
                    .len = @intCast(token_len),
                    .freq = 1,
                };
                token_count += 1;
            }
        }

        return .{
            .tokens = out_tokens[0..token_count],
            .doc_len = @intCast(token_count),
        };
    }

    /// Hash a token with optional lowercase conversion.
    /// Uses LUT-based lowercase for short tokens to avoid branch mispredictions.
    inline fn hashToken(self: Self, text: []const u8) u64 {
        if (self.config.lowercase) {
            return hashLowercase(text);
        }
        return hash.hash(text);
    }

    /// Hash with inline ASCII lowercase conversion using comptime LUT.
    /// The to_lower_lut converts uppercase to lowercase in a single array lookup
    /// (no branches), then hashes the lowercased buffer with wyhash.
    fn hashLowercase(text: []const u8) u64 {
        if (text.len <= 64) {
            var buf: [64]u8 = undefined;
            for (text, 0..) |c, j| {
                buf[j] = to_lower_lut[c];
            }
            return hash.hash(buf[0..text.len]);
        }

        // For longer tokens (rare), use simple branch
        var buf2: [256]u8 = undefined;
        const len = @min(text.len, 256);
        for (text[0..len], 0..) |c, j| {
            buf2[j] = to_lower_lut[c];
        }
        return hash.hash(buf2[0..len]);
    }
};

// ============================================================================
// Comptime LUTs: Character classification + lowercase (single array lookup)
// ============================================================================

/// Delimiter detection LUT: 1 = delimiter, 0 = alphanumeric.
/// Single array lookup replaces 30+ branch comparisons.
const delimiter_lut = blk: {
    var table: [256]u8 = [_]u8{0} ** 256;
    // Control characters and space
    for (0..33) |c| table[c] = 1;
    // Punctuation and symbols
    const delims = ".,;:?!()[]{}\"'<>/\\|@#$%^&*-+=~`";
    for (delims) |c| table[c] = 1;
    break :blk table;
};

/// Combined lowercase + classification LUT.
/// 0 = delimiter, otherwise lowercase byte value.
/// Used for single-pass tokenize+hash without separate lowercase step.
const to_lower_lut = blk: {
    var table: [256]u8 = [_]u8{0} ** 256;
    for ('a'..'z' + 1) |c| table[c] = @intCast(c);
    for ('A'..'Z' + 1) |c| table[c] = @intCast(c | 0x20); // lowercase
    for ('0'..'9' + 1) |c| table[c] = @intCast(c);
    for (128..256) |c| table[c] = @intCast(c); // UTF-8 continuation bytes
    break :blk table;
};

/// Check if a byte is a delimiter (single array lookup)
inline fn isDelimiter(c: u8) bool {
    return delimiter_lut[c] != 0;
}

// ============================================================================
// Hash-table-based aggregation with usedSlots tracking — O(n) vs O(n log n)
// ============================================================================

const AGG_CAPACITY: usize = 2048; // Must be power of 2
const AGG_MASK: u64 = AGG_CAPACITY - 1;
const AGG_MAX_USED: usize = AGG_CAPACITY; // Can fill entire table in worst case

/// Aggregation hash table: open-addressing with linear probing.
/// Uses usedSlots tracking for O(n) clear instead of O(CAPACITY) memset.
const AggTable = struct {
    hashes: [AGG_CAPACITY]u64,
    tokens: [AGG_CAPACITY]Token, // Stores first occurrence + accumulated freq
    used_slots: [AGG_MAX_USED]u16,
    num_used: u16,

    const Self = @This();

    fn init() Self {
        return .{
            .hashes = [_]u64{0} ** AGG_CAPACITY,
            .tokens = undefined,
            .used_slots = undefined,
            .num_used = 0,
        };
    }

    /// Insert a token, incrementing frequency if already present.
    inline fn insert(self: *Self, token: Token) void {
        const key = if (token.hash == 0) 1 else token.hash;
        var idx: usize = @intCast(key & AGG_MASK);

        while (true) {
            if (self.hashes[idx] == 0) {
                // Empty slot: insert new
                self.hashes[idx] = key;
                self.tokens[idx] = token;
                if (self.num_used < AGG_MAX_USED) {
                    self.used_slots[self.num_used] = @intCast(idx);
                    self.num_used += 1;
                }
                return;
            }
            if (self.hashes[idx] == key) {
                // Existing: increment frequency
                self.tokens[idx].freq +|= 1;
                return;
            }
            idx = (idx + 1) & @as(usize, @intCast(AGG_MASK));
        }
    }

    /// Copy unique tokens to output buffer and clear the table.
    fn drainTo(self: *Self, out: []Token) []Token {
        const count = @min(self.num_used, out.len);
        for (self.used_slots[0..count], 0..) |slot_idx, i| {
            out[i] = self.tokens[slot_idx];
        }
        // Clear only used slots
        for (self.used_slots[0..self.num_used]) |slot_idx| {
            self.hashes[slot_idx] = 0;
        }
        self.num_used = 0;
        return out[0..count];
    }
};

/// Thread-local aggregation table (avoids repeated init overhead)
threadlocal var tl_agg_table: AggTable = AggTable.init();

/// Aggregate tokens by hash, counting frequencies — O(n) with hash table.
/// Returns unique tokens with updated freq fields.
pub fn aggregateTokens(tokens: []Token, out: []Token) []Token {
    if (tokens.len == 0) return out[0..0];

    const table = &tl_agg_table;

    for (tokens) |t| {
        table.insert(t);
    }

    return table.drainTo(out);
}

/// Tokenize and aggregate in one pass (convenience function)
pub fn tokenizeAndAggregate(
    tokenizer: *const ByteTokenizer,
    text: []const u8,
    token_buf: []Token,
    agg_buf: []Token,
) struct { tokens: []Token, doc_len: u32 } {
    const batch = tokenizer.tokenize(text, token_buf);
    const aggregated = aggregateTokens(batch.tokens, agg_buf);
    return .{ .tokens = aggregated, .doc_len = batch.doc_len };
}

// ============================================================================
// Tests
// ============================================================================

test "tokenize basic" {
    const tokenizer = ByteTokenizer.init(.{});
    var tokens: [100]Token = undefined;

    const result = tokenizer.tokenize("hello world", &tokens);

    try std.testing.expectEqual(@as(usize, 2), result.tokens.len);
    try std.testing.expectEqual(@as(u32, 2), result.doc_len);
}

test "tokenize vietnamese" {
    const tokenizer = ByteTokenizer.init(.{});
    var tokens: [100]Token = undefined;

    const text = "Thành phố Hồ Chí Minh";
    const result = tokenizer.tokenize(text, &tokens);

    try std.testing.expectEqual(@as(usize, 5), result.tokens.len);
}

test "tokenize with punctuation" {
    const tokenizer = ByteTokenizer.init(.{});
    var tokens: [100]Token = undefined;

    const result = tokenizer.tokenize("hello, world! how are you?", &tokens);

    try std.testing.expectEqual(@as(usize, 5), result.tokens.len);
}

test "tokenize empty" {
    const tokenizer = ByteTokenizer.init(.{});
    var tokens: [100]Token = undefined;

    const result = tokenizer.tokenize("", &tokens);

    try std.testing.expectEqual(@as(usize, 0), result.tokens.len);
}

test "tokenize long text simd" {
    const tokenizer = ByteTokenizer.init(.{});
    var tokens: [1000]Token = undefined;

    // Text longer than 32 bytes to trigger SIMD path
    const text = "the quick brown fox jumps over the lazy dog and continues running across the field";
    const result = tokenizer.tokenize(text, &tokens);

    try std.testing.expect(result.tokens.len > 10);
}

test "aggregate tokens" {
    var tokens = [_]Token{
        .{ .hash = 100, .start = 0, .len = 5, .freq = 1 },
        .{ .hash = 200, .start = 6, .len = 5, .freq = 1 },
        .{ .hash = 100, .start = 12, .len = 5, .freq = 1 },
        .{ .hash = 100, .start = 18, .len = 5, .freq = 1 },
    };

    var out: [10]Token = undefined;
    const aggregated = aggregateTokens(&tokens, &out);

    try std.testing.expectEqual(@as(usize, 2), aggregated.len);

    // Find the token with hash 100
    for (aggregated) |t| {
        if (t.hash == 100) {
            try std.testing.expectEqual(@as(u16, 3), t.freq);
        }
    }
}
