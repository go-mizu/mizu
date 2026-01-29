//! IPC Server for fts_zig
//! Communicates via Unix socket for CGO-free Go integration

const std = @import("std");
const fts = @import("fts_zig");

const Allocator = std.mem.Allocator;

// Message types (must match Go side)
const MSG_ADD_DOC: u8 = 1;
const MSG_BUILD: u8 = 2;
const MSG_SEARCH: u8 = 3;
const MSG_STATS: u8 = 4;
const MSG_CLOSE: u8 = 5;
const MSG_RESPONSE: u8 = 128;

pub fn main() !void {
    var gpa = std.heap.GeneralPurposeAllocator(.{}){};
    defer _ = gpa.deinit();
    const allocator = gpa.allocator();

    const args = try std.process.argsAlloc(allocator);
    defer std.process.argsFree(allocator, args);

    const socket_path = if (args.len > 1) args[1] else "/tmp/fts_zig.sock";
    const profile_str = if (args.len > 2) args[2] else "balanced";

    const profile: fts.index.segment.Profile = if (std.mem.eql(u8, profile_str, "speed"))
        .speed
    else if (std.mem.eql(u8, profile_str, "compact"))
        .compact
    else
        .balanced;

    std.debug.print("fts_zig IPC server starting...\n", .{});
    std.debug.print("  Socket: {s}\n", .{socket_path});
    std.debug.print("  Profile: {s}\n", .{profile_str});

    // Remove existing socket
    std.fs.cwd().deleteFile(socket_path) catch {};

    // Create server
    var server = Server.init(allocator, socket_path, profile);
    defer server.deinit();

    try server.run();
}

const Server = struct {
    allocator: Allocator,
    socket_path: []const u8,
    profile: fts.index.segment.Profile,
    listener: ?std.net.Server,

    // Index state
    speed_builder: ?*fts.profile.speed.SpeedIndexBuilder,
    speed_index: ?*fts.profile.speed.SpeedIndex,
    balanced_builder: ?*fts.profile.balanced.BalancedIndexBuilder,
    balanced_index: ?*fts.profile.balanced.BalancedIndex,

    const Self = @This();

    fn init(allocator: Allocator, socket_path: []const u8, profile: fts.index.segment.Profile) Self {
        return .{
            .allocator = allocator,
            .socket_path = socket_path,
            .profile = profile,
            .listener = null,
            .speed_builder = null,
            .speed_index = null,
            .balanced_builder = null,
            .balanced_index = null,
        };
    }

    fn deinit(self: *Self) void {
        if (self.listener) |*l| {
            l.deinit();
        }
        if (self.speed_builder) |b| {
            b.deinit();
            self.allocator.destroy(b);
        }
        if (self.speed_index) |i| {
            i.deinit();
            self.allocator.destroy(i);
        }
        if (self.balanced_builder) |b| {
            b.deinit();
            self.allocator.destroy(b);
        }
        if (self.balanced_index) |i| {
            i.deinit();
            self.allocator.destroy(i);
        }
    }

    fn run(self: *Self) !void {
        // Create Unix socket
        const addr = std.net.Address.initUnix(self.socket_path) catch return error.InvalidAddress;
        self.listener = try addr.listen(.{});

        std.debug.print("Listening on {s}\n", .{self.socket_path});

        // Create builder
        switch (self.profile) {
            .speed => {
                const builder = try self.allocator.create(fts.profile.speed.SpeedIndexBuilder);
                builder.* = fts.profile.speed.SpeedIndexBuilder.init(self.allocator);
                self.speed_builder = builder;
            },
            .balanced => {
                const builder = try self.allocator.create(fts.profile.balanced.BalancedIndexBuilder);
                builder.* = fts.profile.balanced.BalancedIndexBuilder.init(self.allocator);
                self.balanced_builder = builder;
            },
            else => {},
        }

        // Accept connections
        while (true) {
            const conn = self.listener.?.accept() catch |err| {
                std.debug.print("Accept error: {}\n", .{err});
                continue;
            };

            std.debug.print("Client connected\n", .{});
            self.handleClient(conn.stream) catch |err| {
                std.debug.print("Client error: {}\n", .{err});
            };
            conn.stream.close();
            std.debug.print("Client disconnected\n", .{});
        }
    }

    fn handleClient(self: *Self, stream: std.net.Stream) !void {
        while (true) {
            // Read message header
            var header: [5]u8 = undefined;
            const bytes_read = try stream.read(&header);
            if (bytes_read < 5) break;

            const msg_type = header[0];
            const payload_len = std.mem.readInt(u32, header[1..5], .little);

            // Read payload
            var payload: []u8 = &[_]u8{};
            if (payload_len > 0) {
                payload = try self.allocator.alloc(u8, payload_len);
                defer self.allocator.free(payload);
                _ = try stream.readAll(payload);
            }

            // Handle message
            const response = switch (msg_type) {
                MSG_ADD_DOC => self.handleAddDoc(payload),
                MSG_BUILD => self.handleBuild(),
                MSG_SEARCH => self.handleSearch(payload),
                MSG_STATS => self.handleStats(),
                MSG_CLOSE => return,
                else => &[_]u8{},
            };

            // Send response
            try self.sendResponse(stream, response);
        }
    }

    fn handleAddDoc(self: *Self, payload: []const u8) []const u8 {
        switch (self.profile) {
            .speed => {
                if (self.speed_builder) |b| {
                    _ = b.addDocument(payload) catch return "error";
                }
            },
            .balanced => {
                if (self.balanced_builder) |b| {
                    _ = b.addDocument(payload) catch return "error";
                }
            },
            else => {},
        }
        return "ok";
    }

    fn handleBuild(self: *Self) []const u8 {
        switch (self.profile) {
            .speed => {
                if (self.speed_builder) |b| {
                    const idx = self.allocator.create(fts.profile.speed.SpeedIndex) catch return "error";
                    idx.* = b.build() catch return "error";
                    self.speed_index = idx;
                    b.deinit();
                    self.allocator.destroy(b);
                    self.speed_builder = null;
                }
            },
            .balanced => {
                if (self.balanced_builder) |b| {
                    const idx = self.allocator.create(fts.profile.balanced.BalancedIndex) catch return "error";
                    idx.* = b.build() catch return "error";
                    self.balanced_index = idx;
                    b.deinit();
                    self.allocator.destroy(b);
                    self.balanced_builder = null;
                }
            },
            else => {},
        }
        return "ok";
    }

    fn handleSearch(self: *Self, payload: []const u8) []const u8 {
        if (payload.len < 4) return "";

        const limit = std.mem.readInt(u32, payload[0..4], .little);
        const query = payload[4..];

        _ = limit;
        _ = query;

        // TODO: Implement actual search and serialize results
        return "";
    }

    fn handleStats(self: *Self) []const u8 {
        _ = self;
        // TODO: Return serialized stats
        return "";
    }

    fn sendResponse(self: *Self, stream: std.net.Stream, data: []const u8) !void {
        _ = self;
        var header: [5]u8 = undefined;
        header[0] = MSG_RESPONSE;
        std.mem.writeInt(u32, header[1..5], @intCast(data.len), .little);

        _ = try stream.write(&header);
        if (data.len > 0) {
            _ = try stream.write(data);
        }
    }
};
