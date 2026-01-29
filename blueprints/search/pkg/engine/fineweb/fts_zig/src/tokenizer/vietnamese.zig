//! Vietnamese-specific text processing utilities
//! Handles proper accent preservation and syllable boundaries

const std = @import("std");

/// Vietnamese tone marks (combining diacritics)
pub const ToneMarks = struct {
    pub const GRAVE: u21 = 0x0300; // ̀  (huyền)
    pub const ACUTE: u21 = 0x0301; // ́  (sắc)
    pub const TILDE: u21 = 0x0303; // ̃  (ngã)
    pub const HOOK_ABOVE: u21 = 0x0309; // ̉  (hỏi)
    pub const DOT_BELOW: u21 = 0x0323; // ̣  (nặng)

    pub fn isToneMark(cp: u21) bool {
        return cp == GRAVE or cp == ACUTE or cp == TILDE or cp == HOOK_ABOVE or cp == DOT_BELOW;
    }
};

/// Vietnamese base vowels
pub const Vowels = struct {
    // Plain vowels
    pub const A: u21 = 'a';
    pub const E: u21 = 'e';
    pub const I: u21 = 'i';
    pub const O: u21 = 'o';
    pub const U: u21 = 'u';
    pub const Y: u21 = 'y';

    // Special Vietnamese vowels
    pub const A_BREVE: u21 = 0x0103; // ă
    pub const A_CIRCUMFLEX: u21 = 0x00E2; // â
    pub const E_CIRCUMFLEX: u21 = 0x00EA; // ê
    pub const O_CIRCUMFLEX: u21 = 0x00F4; // ô
    pub const O_HORN: u21 = 0x01A1; // ơ
    pub const U_HORN: u21 = 0x01B0; // ư

    pub fn isVietnameseVowel(cp: u21) bool {
        return switch (cp) {
            'a', 'e', 'i', 'o', 'u', 'y', 'A', 'E', 'I', 'O', 'U', 'Y', 0x0103, 0x0102, // ă Ă
            0x00E2, 0x00C2, // â Â
            0x00EA, 0x00CA, // ê Ê
            0x00F4, 0x00D4, // ô Ô
            0x01A1, 0x01A0, // ơ Ơ
            0x01B0, 0x01AF, // ư Ư
            => true,
            else => false,
        };
    }

    /// Normalize vowel to base form (remove circumflex/breve/horn but keep tone)
    pub fn toBaseVowel(cp: u21) u21 {
        return switch (cp) {
            0x0103, 0x0102 => 'a', // ă Ă
            0x00E2, 0x00C2 => 'a', // â Â -> a (for simplification)
            0x00EA, 0x00CA => 'e', // ê Ê
            0x00F4, 0x00D4 => 'o', // ô Ô
            0x01A1, 0x01A0 => 'o', // ơ Ơ
            0x01B0, 0x01AF => 'u', // ư Ư
            else => cp,
        };
    }
};

/// Vietnamese consonants (including đ)
pub const Consonants = struct {
    pub const D_STROKE: u21 = 0x0111; // đ
    pub const D_STROKE_UPPER: u21 = 0x0110; // Đ

    pub fn isVietnameseConsonant(cp: u21) bool {
        const lower = if (cp >= 'A' and cp <= 'Z') cp + 32 else cp;
        return switch (lower) {
            'b', 'c', 'd', 'g', 'h', 'k', 'l', 'm', 'n', 'p', 'q', 'r', 's', 't', 'v', 'x', 0x0111 => true,
            else => false,
        };
    }
};

/// Check if a codepoint is a Vietnamese character
pub fn isVietnameseChar(cp: u21) bool {
    // ASCII letters
    if ((cp >= 'a' and cp <= 'z') or (cp >= 'A' and cp <= 'Z')) {
        return true;
    }

    // Vietnamese-specific characters
    return switch (cp) {
        // Lowercase with diacritics (a-based)
        0x00E0...0x00E3, 0x00E5, 0x0103, 0x1EA1, 0x1EA3, 0x1EA5, 0x1EA7, 0x1EA9, 0x1EAB, 0x1EAD, 0x1EAF, 0x1EB1, 0x1EB3, 0x1EB5, 0x1EB7 => true,
        // e-based
        0x00E8...0x00EB, 0x1EB9, 0x1EBB, 0x1EBD, 0x1EBF, 0x1EC1, 0x1EC3, 0x1EC5, 0x1EC7 => true,
        // i-based
        0x00EC...0x00EF, 0x1EC9, 0x1ECB => true,
        // o-based
        0x00F2...0x00F5, 0x01A1, 0x1ECD, 0x1ECF, 0x1ED1, 0x1ED3, 0x1ED5, 0x1ED7, 0x1ED9, 0x1EDB, 0x1EDD, 0x1EDF, 0x1EE1, 0x1EE3 => true,
        // u-based
        0x00F9...0x00FC, 0x01B0, 0x1EE5, 0x1EE7, 0x1EE9, 0x1EEB, 0x1EED, 0x1EEF, 0x1EF1 => true,
        // y-based
        0x00FD, 0x1EF3, 0x1EF5, 0x1EF7, 0x1EF9 => true,
        // đ
        0x0111 => true,
        // Uppercase variants
        0x00C0...0x00C3, 0x00C5, 0x0102 => true,
        0x00C8...0x00CB => true,
        0x00CC...0x00CF => true,
        0x00D2...0x00D5, 0x01A0 => true,
        0x00D9...0x00DC, 0x01AF => true,
        0x00DD => true,
        0x0110 => true,
        else => false,
    };
}

/// Precomposed Vietnamese character table (for fast lookup)
/// Maps (base vowel, tone) -> precomposed character
pub const PrecomposedTable = struct {
    /// Get precomposed form if available
    pub fn get(base: u21, tone: u21) ?u21 {
        // This is a simplified version - full table would be larger
        const key = (@as(u32, base) << 16) | @as(u32, tone);

        return switch (key) {
            // a + tones
            ('a' << 16) | ToneMarks.GRAVE => 0x00E0, // à
            ('a' << 16) | ToneMarks.ACUTE => 0x00E1, // á
            ('a' << 16) | ToneMarks.TILDE => 0x00E3, // ã
            ('a' << 16) | ToneMarks.HOOK_ABOVE => 0x1EA3, // ả
            ('a' << 16) | ToneMarks.DOT_BELOW => 0x1EA1, // ạ
            // e + tones
            ('e' << 16) | ToneMarks.GRAVE => 0x00E8, // è
            ('e' << 16) | ToneMarks.ACUTE => 0x00E9, // é
            ('e' << 16) | ToneMarks.TILDE => 0x1EBD, // ẽ
            ('e' << 16) | ToneMarks.HOOK_ABOVE => 0x1EBB, // ẻ
            ('e' << 16) | ToneMarks.DOT_BELOW => 0x1EB9, // ẹ
            // Add more as needed...
            else => null,
        };
    }
};

/// Normalize Vietnamese text to NFC (precomposed) form
/// This ensures consistent hashing regardless of input encoding
pub fn normalizeNFC(input: []const u8, output: []u8) usize {
    // Simple implementation: just copy for now
    // Full NFC normalization would require Unicode tables
    const len = @min(input.len, output.len);
    @memcpy(output[0..len], input[0..len]);
    return len;
}

/// Common Vietnamese words (for stop word filtering if needed)
pub const StopWords = struct {
    const words = [_][]const u8{
        "và",   "của",  "là",   "có",   "được", "cho",  "không", "này",
        "với",  "các",  "để",   "trong", "từ",   "một",  "những", "khi",
        "như",  "đã",   "cũng", "hay",  "hoặc", "thì",  "mà",    "nếu",
        "vì",   "do",   "bởi",  "tại",  "về",   "trên", "dưới",  "theo",
    };

    pub fn isStopWord(word: []const u8) bool {
        for (words) |sw| {
            if (std.mem.eql(u8, word, sw)) return true;
        }
        return false;
    }
};

// ============================================================================
// Tests
// ============================================================================

test "vietnamese char detection" {
    try std.testing.expect(isVietnameseChar('a'));
    try std.testing.expect(isVietnameseChar(0x00E0)); // à
    try std.testing.expect(isVietnameseChar(0x0111)); // đ
    try std.testing.expect(!isVietnameseChar(0x4E2D)); // 中 (Chinese)
}

test "tone mark detection" {
    try std.testing.expect(ToneMarks.isToneMark(0x0301)); // acute
    try std.testing.expect(!ToneMarks.isToneMark('a'));
}

test "stop word detection" {
    try std.testing.expect(StopWords.isStopWord("và"));
    try std.testing.expect(StopWords.isStopWord("của"));
    try std.testing.expect(!StopWords.isStopWord("Việt"));
}
