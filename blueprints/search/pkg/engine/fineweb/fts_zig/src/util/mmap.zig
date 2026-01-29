//! Memory-mapped file I/O for zero-copy segment access

const std = @import("std");
const builtin = @import("builtin");
const posix = std.posix;

/// Page size - use the minimum page size that posix.mmap expects
const PAGE_SIZE: usize = std.heap.page_size_min;

/// Memory-mapped file for reading
pub const MappedFile = struct {
    data: []align(PAGE_SIZE) u8,
    fd: posix.fd_t,
    len: usize,

    const Self = @This();

    /// Open and map a file for reading
    pub fn open(path: []const u8) !Self {
        const fd = try posix.open(path, .{ .ACCMODE = .RDONLY }, 0);
        errdefer posix.close(fd);

        const stat = try posix.fstat(fd);
        const len: usize = @intCast(stat.size);

        if (len == 0) {
            return Self{
                .data = &[_]u8{},
                .fd = fd,
                .len = 0,
            };
        }

        const data = try posix.mmap(
            null,
            len,
            posix.PROT.READ,
            .{ .TYPE = .SHARED },
            fd,
            0,
        );

        return Self{
            .data = @alignCast(data),
            .fd = fd,
            .len = len,
        };
    }

    /// Close and unmap the file
    pub fn close(self: *Self) void {
        if (self.len > 0) {
            posix.munmap(self.data);
        }
        posix.close(self.fd);
        self.* = undefined;
    }

    /// Get a slice of the mapped data
    pub fn slice(self: Self, offset: usize, length: usize) []const u8 {
        if (offset + length > self.len) {
            return &[_]u8{};
        }
        return self.data[offset..][0..length];
    }

    /// Read a value at an offset
    pub fn readAt(self: Self, comptime T: type, offset: usize) ?T {
        if (offset + @sizeOf(T) > self.len) return null;
        return std.mem.bytesAsValue(T, self.data[offset..][0..@sizeOf(T)]).*;
    }

    /// Advise the kernel about access pattern
    pub fn advise(self: Self, advice: Advice) void {
        if (self.len == 0) return;

        const adv: u32 = switch (advice) {
            .sequential => posix.MADV.SEQUENTIAL,
            .random => posix.MADV.RANDOM,
            .willneed => posix.MADV.WILLNEED,
            .dontneed => posix.MADV.DONTNEED,
        };

        posix.madvise(self.data, adv) catch {};
    }

    pub const Advice = enum {
        sequential,
        random,
        willneed,
        dontneed,
    };
};

/// Memory-mapped file for writing
pub const MappedFileWriter = struct {
    data: []align(PAGE_SIZE) u8,
    fd: posix.fd_t,
    capacity: usize,
    len: usize,

    const Self = @This();

    /// Create a new mapped file for writing
    pub fn create(path: []const u8, initial_size: usize) !Self {
        const fd = try posix.open(
            path,
            .{
                .ACCMODE = .RDWR,
                .CREAT = true,
                .TRUNC = true,
            },
            0o644,
        );
        errdefer posix.close(fd);

        // Extend file to initial size
        try posix.ftruncate(fd, @intCast(initial_size));

        const data = try posix.mmap(
            null,
            initial_size,
            posix.PROT.READ | posix.PROT.WRITE,
            .{ .TYPE = .SHARED },
            fd,
            0,
        );

        return Self{
            .data = @alignCast(data),
            .fd = fd,
            .capacity = initial_size,
            .len = 0,
        };
    }

    /// Close the file (truncates to actual written size)
    pub fn close(self: *Self) void {
        // Unmap (implicitly syncs)
        posix.munmap(self.data);

        // Truncate to actual size
        posix.ftruncate(self.fd, @intCast(self.len)) catch {};

        posix.close(self.fd);
        self.* = undefined;
    }

    /// Write data, growing if necessary
    pub fn write(self: *Self, data: []const u8) !void {
        if (self.len + data.len > self.capacity) {
            try self.grow(self.len + data.len);
        }

        @memcpy(self.data[self.len..][0..data.len], data);
        self.len += data.len;
    }

    /// Write a value
    pub fn writeValue(self: *Self, comptime T: type, value: T) !void {
        const bytes = std.mem.asBytes(&value);
        try self.write(bytes);
    }

    /// Get current position
    pub fn position(self: Self) usize {
        return self.len;
    }

    /// Reserve space and return a slice to write into
    pub fn reserve(self: *Self, size: usize) ![]u8 {
        if (self.len + size > self.capacity) {
            try self.grow(self.len + size);
        }

        const start = self.len;
        self.len += size;
        return self.data[start..][0..size];
    }

    fn grow(self: *Self, min_capacity: usize) !void {
        const new_capacity = @max(self.capacity * 2, min_capacity);

        // Unmap old region
        posix.munmap(self.data);

        // Extend file
        try posix.ftruncate(self.fd, @intCast(new_capacity));

        // Remap with new size
        const data = try posix.mmap(
            null,
            new_capacity,
            posix.PROT.READ | posix.PROT.WRITE,
            .{ .TYPE = .SHARED },
            self.fd,
            0,
        );

        self.data = @alignCast(data);
        self.capacity = new_capacity;
    }
};

/// Anonymous mmap for temporary buffers
pub fn allocAnonymous(size: usize) ![]align(PAGE_SIZE) u8 {
    return try posix.mmap(
        null,
        size,
        posix.PROT.READ | posix.PROT.WRITE,
        .{ .TYPE = .PRIVATE, .ANONYMOUS = true },
        -1,
        0,
    );
}

pub fn freeAnonymous(data: []align(PAGE_SIZE) u8) void {
    posix.munmap(data);
}

test "mmap write and read" {
    const path = "/tmp/fts_zig_mmap_test.bin";

    // Write
    {
        var writer = try MappedFileWriter.create(path, 4096);
        defer writer.close();

        try writer.writeValue(u32, 42);
        try writer.writeValue(u64, 0xDEADBEEF);
        try writer.write("hello world");
    }

    // Read
    {
        var reader = try MappedFile.open(path);
        defer reader.close();

        const v1 = reader.readAt(u32, 0).?;
        try std.testing.expectEqual(@as(u32, 42), v1);

        const v2 = reader.readAt(u64, 4).?;
        try std.testing.expectEqual(@as(u64, 0xDEADBEEF), v2);

        const str = reader.slice(12, 11);
        try std.testing.expectEqualStrings("hello world", str);
    }

    // Cleanup
    std.fs.cwd().deleteFile(path) catch {};
}
