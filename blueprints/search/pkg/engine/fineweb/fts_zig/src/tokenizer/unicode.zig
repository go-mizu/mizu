//! Unicode-aware tokenizer with Vietnamese support
//! Features: UTF-8 validation, case folding, optional diacritic stripping
//! Target: ~500MB/sec (4x slower than byte-level, but linguistically correct)

const std = @import("std");
const hash = @import("../util/hash.zig");
const byte_tokenizer = @import("byte.zig");

pub const Token = byte_tokenizer.Token;
pub const TokenBatch = byte_tokenizer.TokenBatch;

/// Configuration for Unicode tokenizer
pub const Config = struct {
    /// Normalize Unicode (NFD/NFC)
    normalize: bool = true,
    /// Strip diacritics for accent-insensitive search
    strip_diacritics: bool = false,
    /// Case fold (lowercase)
    case_fold: bool = true,
    /// Minimum token length in bytes
    min_length: u8 = 1,
    /// Maximum token length in bytes
    max_length: u16 = 256,
};

/// Unicode tokenizer with Vietnamese support
pub const UnicodeTokenizer = struct {
    config: Config,

    const Self = @This();

    pub fn init(config: Config) Self {
        return .{ .config = config };
    }

    /// Tokenize UTF-8 text
    pub fn tokenize(self: Self, text: []const u8, out_tokens: []Token, work_buf: []u8) TokenBatch {
        if (text.len == 0) {
            return .{ .tokens = out_tokens[0..0], .doc_len = 0 };
        }

        var token_count: usize = 0;
        var i: usize = 0;

        while (i < text.len and token_count < out_tokens.len) {
            // Skip whitespace and delimiters
            while (i < text.len and isUtf8Delimiter(text, i)) {
                i += utf8ByteLen(text[i]);
            }

            if (i >= text.len) break;

            // Start of token
            const token_start = i;

            // Find end of token
            while (i < text.len and !isUtf8Delimiter(text, i)) {
                i += utf8ByteLen(text[i]);
            }

            const token_end = i;
            const raw_token = text[token_start..token_end];

            if (raw_token.len < self.config.min_length or raw_token.len > self.config.max_length) {
                continue;
            }

            // Process token (normalize, case fold, etc.)
            const processed = self.processToken(raw_token, work_buf);
            const token_hash = hash.hash(processed);

            out_tokens[token_count] = .{
                .hash = token_hash,
                .start = @intCast(token_start),
                .len = @intCast(raw_token.len),
                .freq = 1,
            };
            token_count += 1;
        }

        return .{
            .tokens = out_tokens[0..token_count],
            .doc_len = @intCast(token_count),
        };
    }

    /// Process a token: normalize, case fold, optionally strip diacritics
    fn processToken(self: Self, token: []const u8, buf: []u8) []const u8 {
        if (!self.config.case_fold and !self.config.strip_diacritics) {
            return token;
        }

        var out_len: usize = 0;
        var i: usize = 0;

        while (i < token.len and out_len < buf.len - 4) {
            const byte_len = utf8ByteLen(token[i]);
            if (i + byte_len > token.len) break;

            const slice = token[i..][0..byte_len];
            const codepoint = decodeUtf8(slice) orelse {
                i += 1;
                continue;
            };

            var processed_cp = codepoint;

            // Case folding
            if (self.config.case_fold) {
                processed_cp = caseFold(processed_cp);
            }

            // Strip diacritics (for Vietnamese)
            if (self.config.strip_diacritics) {
                processed_cp = stripDiacritics(processed_cp);
            }

            // Encode back to UTF-8
            const encoded_len = encodeUtf8(processed_cp, buf[out_len..]);
            out_len += encoded_len;

            i += byte_len;
        }

        return buf[0..out_len];
    }
};

/// Get the byte length of a UTF-8 character from its first byte
inline fn utf8ByteLen(first_byte: u8) usize {
    if (first_byte < 0x80) return 1;
    if (first_byte < 0xC0) return 1; // Invalid, treat as 1
    if (first_byte < 0xE0) return 2;
    if (first_byte < 0xF0) return 3;
    return 4;
}

/// Check if position in UTF-8 text is a delimiter
fn isUtf8Delimiter(text: []const u8, pos: usize) bool {
    if (pos >= text.len) return true;

    const first_byte = text[pos];

    // ASCII delimiters (fast path)
    if (first_byte < 0x80) {
        return switch (first_byte) {
            ' ', '\t', '\n', '\r', '.', ',', ';', ':', '?', '!', '(', ')', '[', ']', '{', '}', '"', '\'', '<', '>', '/', '\\', '|', '@', '#', '$', '%', '^', '&', '*', '-', '+', '=', '~', '`', 0 => true,
            else => false,
        };
    }

    // Check for Unicode whitespace/punctuation
    const byte_len = utf8ByteLen(first_byte);
    if (pos + byte_len > text.len) return true;

    const codepoint = decodeUtf8(text[pos..][0..byte_len]) orelse return true;

    return isUnicodeDelimiter(codepoint);
}

/// Check if a Unicode codepoint is a delimiter
fn isUnicodeDelimiter(cp: u21) bool {
    // Common Unicode whitespace and punctuation
    return switch (cp) {
        // Whitespace
        0x0020, // Space
        0x00A0, // No-break space
        0x1680, // Ogham space
        0x2000...0x200A, // Various spaces
        0x2028, // Line separator
        0x2029, // Paragraph separator
        0x202F, // Narrow no-break space
        0x205F, // Medium mathematical space
        0x3000, // Ideographic space
        // Common punctuation
        0x2010...0x2015, // Dashes
        0x2018...0x201F, // Quotation marks
        0x2026, // Ellipsis
        0x3001, // Ideographic comma
        0x3002, // Ideographic period
        0xFF01...0xFF0F, // Fullwidth punctuation
        => true,
        else => false,
    };
}

/// Decode UTF-8 bytes to codepoint
fn decodeUtf8(bytes: []const u8) ?u21 {
    if (bytes.len == 0) return null;

    const first = bytes[0];

    if (first < 0x80) {
        return first;
    } else if (first < 0xC0) {
        return null; // Invalid
    } else if (first < 0xE0) {
        if (bytes.len < 2) return null;
        return (@as(u21, first & 0x1F) << 6) | (bytes[1] & 0x3F);
    } else if (first < 0xF0) {
        if (bytes.len < 3) return null;
        return (@as(u21, first & 0x0F) << 12) |
            (@as(u21, bytes[1] & 0x3F) << 6) |
            (bytes[2] & 0x3F);
    } else {
        if (bytes.len < 4) return null;
        return (@as(u21, first & 0x07) << 18) |
            (@as(u21, bytes[1] & 0x3F) << 12) |
            (@as(u21, bytes[2] & 0x3F) << 6) |
            (bytes[3] & 0x3F);
    }
}

/// Encode codepoint to UTF-8
fn encodeUtf8(cp: u21, buf: []u8) usize {
    if (cp < 0x80) {
        buf[0] = @truncate(cp);
        return 1;
    } else if (cp < 0x800) {
        buf[0] = @truncate(0xC0 | (cp >> 6));
        buf[1] = @truncate(0x80 | (cp & 0x3F));
        return 2;
    } else if (cp < 0x10000) {
        buf[0] = @truncate(0xE0 | (cp >> 12));
        buf[1] = @truncate(0x80 | ((cp >> 6) & 0x3F));
        buf[2] = @truncate(0x80 | (cp & 0x3F));
        return 3;
    } else {
        buf[0] = @truncate(0xF0 | (cp >> 18));
        buf[1] = @truncate(0x80 | ((cp >> 12) & 0x3F));
        buf[2] = @truncate(0x80 | ((cp >> 6) & 0x3F));
        buf[3] = @truncate(0x80 | (cp & 0x3F));
        return 4;
    }
}

/// Simple case folding (ASCII + common Vietnamese)
fn caseFold(cp: u21) u21 {
    // ASCII uppercase
    if (cp >= 'A' and cp <= 'Z') {
        return cp + 32;
    }

    // Vietnamese uppercase vowels with diacritics
    return switch (cp) {
        // A variants
        0x00C0...0x00C5 => cp + 32, // À-Å -> à-å
        0x0102 => 0x0103, // Ă -> ă
        // E variants
        0x00C8...0x00CB => cp + 32, // È-Ë -> è-ë
        // I variants
        0x00CC...0x00CF => cp + 32, // Ì-Ï -> ì-ï
        // O variants
        0x00D2...0x00D6 => cp + 32, // Ò-Ö -> ò-ö
        0x01A0 => 0x01A1, // Ơ -> ơ
        // U variants
        0x00D9...0x00DC => cp + 32, // Ù-Ü -> ù-ü
        0x01AF => 0x01B0, // Ư -> ư
        // Y variants
        0x00DD => 0x00FD, // Ý -> ý
        0x1EF2 => 0x1EF3, // Ỳ -> ỳ
        // D variant
        0x0110 => 0x0111, // Đ -> đ
        else => cp,
    };
}

/// Strip diacritics from Vietnamese characters
fn stripDiacritics(cp: u21) u21 {
    return switch (cp) {
        // a variants
        0x00E0, 0x00E1, 0x00E2, 0x00E3, 0x00E4, 0x00E5, 0x0103, 0x1EA1, 0x1EA3, 0x1EA5, 0x1EA7, 0x1EA9, 0x1EAB, 0x1EAD, 0x1EAF, 0x1EB1, 0x1EB3, 0x1EB5, 0x1EB7 => 'a',
        // e variants
        0x00E8, 0x00E9, 0x00EA, 0x00EB, 0x1EB9, 0x1EBB, 0x1EBD, 0x1EBF, 0x1EC1, 0x1EC3, 0x1EC5, 0x1EC7 => 'e',
        // i variants
        0x00EC, 0x00ED, 0x00EE, 0x00EF, 0x1EC9, 0x1ECB => 'i',
        // o variants
        0x00F2, 0x00F3, 0x00F4, 0x00F5, 0x00F6, 0x01A1, 0x1ECD, 0x1ECF, 0x1ED1, 0x1ED3, 0x1ED5, 0x1ED7, 0x1ED9, 0x1EDB, 0x1EDD, 0x1EDF, 0x1EE1, 0x1EE3 => 'o',
        // u variants
        0x00F9, 0x00FA, 0x00FB, 0x00FC, 0x01B0, 0x1EE5, 0x1EE7, 0x1EE9, 0x1EEB, 0x1EED, 0x1EEF, 0x1EF1 => 'u',
        // y variants
        0x00FD, 0x00FF, 0x1EF3, 0x1EF5, 0x1EF7, 0x1EF9 => 'y',
        // d variant
        0x0111 => 'd',
        else => cp,
    };
}

// ============================================================================
// Tests
// ============================================================================

test "unicode tokenize basic" {
    const tokenizer = UnicodeTokenizer.init(.{});
    var tokens: [100]Token = undefined;
    var work_buf: [256]u8 = undefined;

    const result = tokenizer.tokenize("hello world", &tokens, &work_buf);
    try std.testing.expectEqual(@as(usize, 2), result.tokens.len);
}

test "unicode tokenize vietnamese" {
    const tokenizer = UnicodeTokenizer.init(.{ .case_fold = true });
    var tokens: [100]Token = undefined;
    var work_buf: [256]u8 = undefined;

    const result = tokenizer.tokenize("Việt Nam", &tokens, &work_buf);
    try std.testing.expectEqual(@as(usize, 2), result.tokens.len);
}

test "unicode case folding" {
    // Test that uppercase and lowercase produce same hash
    const tokenizer = UnicodeTokenizer.init(.{ .case_fold = true });
    var tokens1: [10]Token = undefined;
    var tokens2: [10]Token = undefined;
    var work_buf: [256]u8 = undefined;

    const r1 = tokenizer.tokenize("HELLO", &tokens1, &work_buf);
    const r2 = tokenizer.tokenize("hello", &tokens2, &work_buf);

    try std.testing.expectEqual(r1.tokens[0].hash, r2.tokens[0].hash);
}

test "unicode strip diacritics" {
    const tokenizer = UnicodeTokenizer.init(.{
        .case_fold = true,
        .strip_diacritics = true,
    });
    var tokens1: [10]Token = undefined;
    var tokens2: [10]Token = undefined;
    var work_buf: [256]u8 = undefined;

    const r1 = tokenizer.tokenize("việt", &tokens1, &work_buf);
    const r2 = tokenizer.tokenize("viet", &tokens2, &work_buf);

    try std.testing.expectEqual(r1.tokens[0].hash, r2.tokens[0].hash);
}

test "utf8 decode encode roundtrip" {
    const test_cases = [_][]const u8{
        "a",
        "à",
        "ă",
        "Việt",
        "日本語",
    };

    var buf: [4]u8 = undefined;

    for (test_cases) |s| {
        var i: usize = 0;
        while (i < s.len) {
            const byte_len = utf8ByteLen(s[i]);
            const cp = decodeUtf8(s[i..][0..byte_len]).?;
            const encoded_len = encodeUtf8(cp, &buf);
            try std.testing.expectEqualSlices(u8, s[i..][0..byte_len], buf[0..encoded_len]);
            i += byte_len;
        }
    }
}
