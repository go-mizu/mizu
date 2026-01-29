//! IPC server implementation

const std = @import("std");
const protocol = @import("protocol.zig");

pub const ServerConfig = struct {
    socket_path: []const u8 = "/tmp/fts_zig.sock",
    max_connections: u32 = 10,
};

/// Generic message handler interface
pub const MessageHandler = struct {
    ptr: *anyopaque,
    vtable: *const VTable,

    const VTable = struct {
        handle_add_doc: *const fn (*anyopaque, []const u8) []const u8,
        handle_build: *const fn (*anyopaque) []const u8,
        handle_search: *const fn (*anyopaque, []const u8) []const u8,
        handle_stats: *const fn (*anyopaque) []const u8,
    };

    pub fn handleAddDoc(self: MessageHandler, payload: []const u8) []const u8 {
        return self.vtable.handle_add_doc(self.ptr, payload);
    }

    pub fn handleBuild(self: MessageHandler) []const u8 {
        return self.vtable.handle_build(self.ptr);
    }

    pub fn handleSearch(self: MessageHandler, payload: []const u8) []const u8 {
        return self.vtable.handle_search(self.ptr, payload);
    }

    pub fn handleStats(self: MessageHandler) []const u8 {
        return self.vtable.handle_stats(self.ptr);
    }
};
