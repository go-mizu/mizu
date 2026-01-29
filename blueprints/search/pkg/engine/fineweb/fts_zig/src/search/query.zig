//! Query parser for search queries
//! Supports: terms, phrases ("quoted"), AND/OR operators

const std = @import("std");
const hash = @import("../util/hash.zig");
const byte_tokenizer = @import("../tokenizer/byte.zig");

/// Query term with hash
pub const QueryTerm = struct {
    hash: u64,
    text: []const u8,
    required: bool, // Must match (AND semantics)
};

/// Parsed query
pub const Query = struct {
    terms: []QueryTerm,
    is_phrase: bool,
    allocator: std.mem.Allocator,

    const Self = @This();

    pub fn deinit(self: *Self) void {
        self.allocator.free(self.terms);
    }
};

/// Parse a search query
pub fn parse(allocator: std.mem.Allocator, query_text: []const u8) !Query {
    if (query_text.len == 0) {
        return Query{
            .terms = &[_]QueryTerm{},
            .is_phrase = false,
            .allocator = allocator,
        };
    }

    // Check for phrase query (quoted)
    const is_phrase = query_text.len >= 2 and
        query_text[0] == '"' and
        query_text[query_text.len - 1] == '"';

    const text = if (is_phrase)
        query_text[1 .. query_text.len - 1]
    else
        query_text;

    // Tokenize query
    var token_buf: [256]byte_tokenizer.Token = undefined;
    const tokenizer = byte_tokenizer.ByteTokenizer.init(.{ .lowercase = true });
    const batch = tokenizer.tokenize(text, &token_buf);

    if (batch.tokens.len == 0) {
        return Query{
            .terms = &[_]QueryTerm{},
            .is_phrase = is_phrase,
            .allocator = allocator,
        };
    }

    // Convert to QueryTerms
    var terms = try allocator.alloc(QueryTerm, batch.tokens.len);

    for (batch.tokens, 0..) |tok, i| {
        terms[i] = .{
            .hash = tok.hash,
            .text = text[tok.start..][0..tok.len],
            .required = true, // Default to AND semantics
        };
    }

    return Query{
        .terms = terms,
        .is_phrase = is_phrase,
        .allocator = allocator,
    };
}

/// Simple query builder for programmatic construction
pub const QueryBuilder = struct {
    terms: std.ArrayList(QueryTerm),
    is_phrase: bool,

    const Self = @This();

    pub fn init(allocator: std.mem.Allocator) Self {
        return .{
            .terms = std.ArrayList(QueryTerm).init(allocator),
            .is_phrase = false,
        };
    }

    pub fn deinit(self: *Self) void {
        self.terms.deinit();
    }

    pub fn addTerm(self: *Self, text: []const u8, required: bool) !void {
        try self.terms.append(.{
            .hash = hash.hash(text),
            .text = text,
            .required = required,
        });
    }

    pub fn addTermHash(self: *Self, term_hash: u64, required: bool) !void {
        try self.terms.append(.{
            .hash = term_hash,
            .text = "",
            .required = required,
        });
    }

    pub fn setPhrase(self: *Self, is_phrase: bool) void {
        self.is_phrase = is_phrase;
    }

    pub fn build(self: *Self) Query {
        const terms = self.terms.toOwnedSlice() catch &[_]QueryTerm{};
        return Query{
            .terms = terms,
            .is_phrase = self.is_phrase,
            .allocator = self.terms.allocator,
        };
    }
};

// ============================================================================
// Tests
// ============================================================================

test "query parse basic" {
    var query = try parse(std.testing.allocator, "hello world");
    defer query.deinit();

    try std.testing.expectEqual(@as(usize, 2), query.terms.len);
    try std.testing.expect(!query.is_phrase);
}

test "query parse phrase" {
    var query = try parse(std.testing.allocator, "\"hello world\"");
    defer query.deinit();

    try std.testing.expectEqual(@as(usize, 2), query.terms.len);
    try std.testing.expect(query.is_phrase);
}

test "query parse empty" {
    var query = try parse(std.testing.allocator, "");
    defer query.deinit();

    try std.testing.expectEqual(@as(usize, 0), query.terms.len);
}

test "query builder" {
    var builder = QueryBuilder.init(std.testing.allocator);
    defer builder.deinit();

    try builder.addTerm("hello", true);
    try builder.addTerm("world", false);

    var query = builder.build();
    defer query.deinit();

    try std.testing.expectEqual(@as(usize, 2), query.terms.len);
    try std.testing.expect(query.terms[0].required);
    try std.testing.expect(!query.terms[1].required);
}
