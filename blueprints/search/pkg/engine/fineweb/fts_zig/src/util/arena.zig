//! Arena allocator optimized for batch document processing
//! Allows fast allocation with single bulk deallocation

const std = @import("std");
const Allocator = std.mem.Allocator;

fn ManagedArrayList(comptime T: type) type {
    return std.array_list.AlignedManaged(T, null);
}

/// Block size for arena chunks (2MB for good TLB behavior)
const BLOCK_SIZE: usize = 2 * 1024 * 1024;

/// Arena allocator that allocates from large blocks
pub const Arena = struct {
    blocks: ManagedArrayList([]u8),
    current: []u8,
    offset: usize,
    backing_allocator: Allocator,
    total_allocated: usize,

    const Self = @This();

    pub fn init(backing_allocator: Allocator) Self {
        return .{
            .blocks = ManagedArrayList([]u8).init(backing_allocator),
            .current = &[_]u8{},
            .offset = 0,
            .backing_allocator = backing_allocator,
            .total_allocated = 0,
        };
    }

    pub fn deinit(self: *Self) void {
        for (self.blocks.items) |block| {
            self.backing_allocator.free(block);
        }
        self.blocks.deinit();
        self.current = &[_]u8{};
        self.offset = 0;
        self.total_allocated = 0;
    }

    /// Reset arena for reuse (keeps allocated blocks)
    pub fn reset(self: *Self) void {
        self.offset = 0;
        if (self.blocks.items.len > 0) {
            self.current = self.blocks.items[0];
        }
        // Could optionally free excess blocks here
    }

    /// Allocate memory from the arena
    pub fn alloc(self: *Self, comptime T: type, n: usize) ![]T {
        const byte_count = @sizeOf(T) * n;
        const alignment = @alignOf(T);

        // Align offset
        const aligned_offset = std.mem.alignForward(usize, self.offset, alignment);
        const end_offset = aligned_offset + byte_count;

        if (end_offset > self.current.len) {
            try self.allocateNewBlock(byte_count);
            return self.alloc(T, n);
        }

        const ptr: [*]T = @ptrCast(@alignCast(self.current.ptr + aligned_offset));
        self.offset = end_offset;
        self.total_allocated += byte_count;

        return ptr[0..n];
    }

    /// Allocate a single item
    pub fn create(self: *Self, comptime T: type) !*T {
        const slice = try self.alloc(T, 1);
        return &slice[0];
    }

    /// Duplicate a slice
    pub fn dupe(self: *Self, comptime T: type, slice: []const T) ![]T {
        const new_slice = try self.alloc(T, slice.len);
        @memcpy(new_slice, slice);
        return new_slice;
    }

    /// Duplicate a string
    pub fn dupeString(self: *Self, str: []const u8) ![]u8 {
        return self.dupe(u8, str);
    }

    fn allocateNewBlock(self: *Self, min_size: usize) !void {
        const size = @max(BLOCK_SIZE, min_size);
        const block = try self.backing_allocator.alloc(u8, size);
        try self.blocks.append(block);
        self.current = block;
        self.offset = 0;
    }

    /// Get allocator interface
    pub fn allocator(self: *Self) Allocator {
        return .{
            .ptr = self,
            .vtable = &.{
                .alloc = allocFn,
                .resize = resizeFn,
                .free = freeFn,
            },
        };
    }

    fn allocFn(ctx: *anyopaque, len: usize, ptr_align: u8, _: usize) ?[*]u8 {
        const self: *Self = @ptrCast(@alignCast(ctx));
        const alignment = @as(usize, 1) << @intCast(ptr_align);
        const aligned_offset = std.mem.alignForward(usize, self.offset, alignment);
        const end_offset = aligned_offset + len;

        if (end_offset > self.current.len) {
            self.allocateNewBlock(len) catch return null;
            return allocFn(ctx, len, ptr_align, 0);
        }

        const ptr = self.current.ptr + aligned_offset;
        self.offset = end_offset;
        self.total_allocated += len;
        return ptr;
    }

    fn resizeFn(_: *anyopaque, _: [*]u8, _: usize, _: usize, _: u8, _: usize) bool {
        // Arena doesn't support resize
        return false;
    }

    fn freeFn(_: *anyopaque, _: [*]u8, _: usize, _: u8, _: usize) void {
        // Arena doesn't free individual allocations
    }
};

/// Thread-local arena for zero-contention allocation
pub const ThreadLocalArena = struct {
    arena: Arena,

    const Self = @This();

    threadlocal var instance: ?*Self = null;

    pub fn get() *Self {
        if (instance) |i| return i;
        const self = std.heap.page_allocator.create(Self) catch unreachable;
        self.* = .{ .arena = Arena.init(std.heap.page_allocator) };
        instance = self;
        return self;
    }

    pub fn alloc(comptime T: type, n: usize) ![]T {
        return get().arena.alloc(T, n);
    }

    pub fn reset() void {
        if (instance) |i| {
            i.arena.reset();
        }
    }
};

test "arena basic" {
    var arena = Arena.init(std.testing.allocator);
    defer arena.deinit();

    const nums = try arena.alloc(u32, 100);
    try std.testing.expectEqual(@as(usize, 100), nums.len);

    for (nums, 0..) |*n, i| {
        n.* = @intCast(i);
    }

    const str = try arena.dupeString("hello world");
    try std.testing.expectEqualStrings("hello world", str);
}

test "arena reset" {
    var arena = Arena.init(std.testing.allocator);
    defer arena.deinit();

    _ = try arena.alloc(u8, 1000);
    const before = arena.total_allocated;

    arena.reset();

    _ = try arena.alloc(u8, 1000);
    // After reset, we reuse the same block
    try std.testing.expect(arena.blocks.items.len == 1);
    _ = before;
}
