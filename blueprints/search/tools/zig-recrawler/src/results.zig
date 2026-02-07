const std = @import("std");
const types = @import("types.zig");

/// Sharded TSV result writer. Each shard has its own file and write buffer.
/// Thread-safe: each shard has a mutex for exclusive write access.
pub const ResultWriter = struct {
    shards: []Shard,
    dir_path: []const u8,
    allocator: std.mem.Allocator,
    total_written: std.atomic.Value(u64),

    const BUFFER_SIZE = 64 * 1024; // 64KB write buffer per shard

    const Shard = struct {
        file: std.fs.File,
        mutex: std.Thread.Mutex,
        buf: [BUFFER_SIZE]u8,
        buf_len: usize,
        written: u64,
    };

    pub fn init(allocator: std.mem.Allocator, dir_path: []const u8, shard_count: u32) !ResultWriter {
        // Create output directory
        std.fs.cwd().makePath(dir_path) catch {};

        const shards = try allocator.alloc(Shard, shard_count);

        for (shards, 0..) |*shard, i| {
            var name_buf: [256]u8 = undefined;
            const name = std.fmt.bufPrint(&name_buf, "{s}/results_{d:0>3}.tsv", .{ dir_path, i }) catch return error.PathTooLong;

            const file = try std.fs.cwd().createFile(name, .{});

            // Write TSV header
            file.writeAll("url\tstatus_code\tcontent_type\tcontent_length\tdomain\tredirect_url\tfetch_time_ms\terror\n") catch {};

            shard.* = .{
                .file = file,
                .mutex = .{},
                .buf = undefined,
                .buf_len = 0,
                .written = 0,
            };
        }

        return .{
            .shards = shards,
            .dir_path = dir_path,
            .allocator = allocator,
            .total_written = std.atomic.Value(u64).init(0),
        };
    }

    pub fn addResult(self: *ResultWriter, result: *const types.FetchResult) void {
        const shard_idx = types.fnv1a(result.url) % @as(u32, @intCast(self.shards.len));
        const shard = &self.shards[shard_idx];

        shard.mutex.lock();
        defer shard.mutex.unlock();

        // Format result as TSV line into shard buffer
        const needed = formatResult(result, shard.buf[shard.buf_len..]) catch {
            // Buffer too small, flush first
            self.flushShard(shard);
            const retried = formatResult(result, shard.buf[shard.buf_len..]) catch return;
            shard.buf_len += retried;
            _ = self.total_written.fetchAdd(1, .monotonic);
            return;
        };
        shard.buf_len += needed;

        // Flush if buffer is getting full
        if (shard.buf_len > BUFFER_SIZE - 1024) {
            self.flushShard(shard);
        }

        _ = self.total_written.fetchAdd(1, .monotonic);
    }

    fn formatResult(result: *const types.FetchResult, buf: []u8) !usize {
        var stream = std.io.fixedBufferStream(buf);
        const writer = stream.writer();

        // url
        try tsvEscape(writer, result.url);
        try writer.writeByte('\t');

        // status_code
        try writer.print("{d}", .{result.status_code});
        try writer.writeByte('\t');

        // content_type
        try tsvEscape(writer, result.contentTypeSlice());
        try writer.writeByte('\t');

        // content_length
        try writer.print("{d}", .{result.content_length});
        try writer.writeByte('\t');

        // domain
        try tsvEscape(writer, result.domain);
        try writer.writeByte('\t');

        // redirect_url
        try tsvEscape(writer, result.redirectSlice());
        try writer.writeByte('\t');

        // fetch_time_ms
        try writer.print("{d}", .{result.fetch_time_ms});
        try writer.writeByte('\t');

        // error
        try tsvEscape(writer, result.errorSlice());
        try writer.writeByte('\n');

        return stream.pos;
    }

    fn tsvEscape(writer: anytype, s: []const u8) !void {
        for (s) |c| {
            switch (c) {
                '\t' => try writer.writeAll("\\t"),
                '\n' => try writer.writeAll("\\n"),
                '\r' => try writer.writeAll("\\r"),
                else => try writer.writeByte(c),
            }
        }
    }

    fn flushShard(self: *ResultWriter, shard: *Shard) void {
        _ = self;
        if (shard.buf_len == 0) return;
        shard.file.writeAll(shard.buf[0..shard.buf_len]) catch {};
        shard.written += shard.buf_len;
        shard.buf_len = 0;
    }

    pub fn flush(self: *ResultWriter) void {
        for (self.shards) |*shard| {
            shard.mutex.lock();
            self.flushShard(shard);
            shard.mutex.unlock();
        }
    }

    pub fn totalWritten(self: *const ResultWriter) u64 {
        return self.total_written.load(.monotonic);
    }

    pub fn close(self: *ResultWriter) void {
        self.flush();
        for (self.shards) |*shard| {
            shard.file.close();
        }
        self.allocator.free(self.shards);
    }
};

test "ResultWriter format" {
    var result = types.FetchResult{
        .url = "https://example.com",
        .domain = "example.com",
        .status_code = 200,
        .content_type = undefined,
        .content_type_len = 0,
        .content_length = 1024,
        .fetch_time_ms = 150,
        .error_msg = undefined,
        .error_len = 0,
        .redirect_url = undefined,
        .redirect_len = 0,
    };
    @memcpy(result.content_type[0..9], "text/html");
    result.content_type_len = 9;

    var buf: [1024]u8 = undefined;
    const len = try ResultWriter.formatResult(&result, &buf);
    const line = buf[0..len];
    try std.testing.expect(std.mem.startsWith(u8, line, "https://example.com\t200\ttext/html\t1024\texample.com\t\t150\t\n"));
}
