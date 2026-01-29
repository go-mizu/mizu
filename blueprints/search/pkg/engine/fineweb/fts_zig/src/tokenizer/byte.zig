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

            // Process each byte position with a delimiter
            var mask = delim_mask;
            var local_offset: usize = 0;

            while (mask != 0) {
                const pos = @ctz(mask);
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

                // Clear this bit and continue
                mask &= mask - 1;
                local_offset = pos + 1;
            }

            // Check if we're starting a new token after last delimiter
            if (local_offset < 32) {
                const next_pos = i + local_offset;
                if (!in_token and next_pos < text.len and !isDelimiter(text[next_pos])) {
                    token_start = next_pos;
                    in_token = true;
                }
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

    /// Hash a token with optional lowercase conversion
    inline fn hashToken(self: Self, text: []const u8) u64 {
        if (self.config.lowercase) {
            return hashLowercase(text);
        }
        return hash.hash(text);
    }

    /// Hash with inline ASCII lowercase conversion
    fn hashLowercase(text: []const u8) u64 {
        // For short tokens, lowercase inline
        if (text.len <= 32) {
            var buf: [32]u8 = undefined;
            for (text, 0..) |c, j| {
                buf[j] = if (c >= 'A' and c <= 'Z') c + 32 else c;
            }
            return hash.hash(buf[0..text.len]);
        }

        // For longer tokens, hash directly (rare case)
        return hash.hash(text);
    }
};

/// Check if a byte is a delimiter
inline fn isDelimiter(c: u8) bool {
    return switch (c) {
        ' ', '\t', '\n', '\r', '.', ',', ';', ':', '?', '!', '(', ')', '[', ']', '{', '}', '"', '\'', '<', '>', '/', '\\', '|', '@', '#', '$', '%', '^', '&', '*', '-', '+', '=', '~', '`' => true,
        else => false,
    };
}

/// Aggregate tokens by hash, counting frequencies
/// Returns unique tokens with updated freq fields
pub fn aggregateTokens(tokens: []Token, out: []Token) []Token {
    if (tokens.len == 0) return out[0..0];

    // Sort by hash
    std.mem.sort(Token, tokens, {}, struct {
        fn lessThan(_: void, a: Token, b: Token) bool {
            return a.hash < b.hash;
        }
    }.lessThan);

    // Aggregate
    var out_idx: usize = 0;
    var current_hash = tokens[0].hash;
    var current_freq: u16 = 0;
    var first_token = tokens[0];

    for (tokens) |t| {
        if (t.hash == current_hash) {
            current_freq += 1;
        } else {
            if (out_idx < out.len) {
                out[out_idx] = first_token;
                out[out_idx].freq = current_freq;
                out_idx += 1;
            }
            current_hash = t.hash;
            current_freq = 1;
            first_token = t;
        }
    }

    // Final token
    if (out_idx < out.len) {
        out[out_idx] = first_token;
        out[out_idx].freq = current_freq;
        out_idx += 1;
    }

    return out[0..out_idx];
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
