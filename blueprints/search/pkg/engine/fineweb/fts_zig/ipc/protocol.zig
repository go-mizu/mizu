//! Wire protocol for IPC communication
//! Simple binary protocol with msgpack-like encoding

const std = @import("std");

/// Message types
pub const MessageType = enum(u8) {
    add_document = 1,
    build = 2,
    search = 3,
    stats = 4,
    close = 5,
    response = 128,
    error_response = 129,
};

/// Message header (5 bytes)
pub const Header = extern struct {
    msg_type: u8,
    payload_len: u32,
};

/// Search request
pub const SearchRequest = struct {
    query: []const u8,
    limit: u32,

    pub fn encode(self: SearchRequest, allocator: std.mem.Allocator) ![]u8 {
        var buf = try allocator.alloc(u8, 4 + self.query.len);
        std.mem.writeInt(u32, buf[0..4], self.limit, .little);
        @memcpy(buf[4..], self.query);
        return buf;
    }

    pub fn decode(data: []const u8) ?SearchRequest {
        if (data.len < 4) return null;
        return .{
            .limit = std.mem.readInt(u32, data[0..4], .little),
            .query = data[4..],
        };
    }
};

/// Search result (8 bytes each)
pub const SearchResultWire = extern struct {
    doc_id: u32,
    score_bits: u32, // float32 as bits
};

/// Encode search results for wire
pub fn encodeSearchResults(results: []const SearchResultWire, allocator: std.mem.Allocator) ![]u8 {
    const header_size = 4; // count
    const result_size = @sizeOf(SearchResultWire);
    var buf = try allocator.alloc(u8, header_size + results.len * result_size);

    std.mem.writeInt(u32, buf[0..4], @intCast(results.len), .little);

    for (results, 0..) |r, i| {
        const offset = header_size + i * result_size;
        std.mem.writeInt(u32, buf[offset..][0..4], r.doc_id, .little);
        std.mem.writeInt(u32, buf[offset + 4 ..][0..4], r.score_bits, .little);
    }

    return buf;
}

/// Stats response
pub const StatsResponse = extern struct {
    doc_count: u32,
    term_count: u32,
    memory_bytes: u64,
};

pub fn encodeStats(stats: StatsResponse) [16]u8 {
    var buf: [16]u8 = undefined;
    std.mem.writeInt(u32, buf[0..4], stats.doc_count, .little);
    std.mem.writeInt(u32, buf[4..8], stats.term_count, .little);
    std.mem.writeInt(u64, buf[8..16], stats.memory_bytes, .little);
    return buf;
}
