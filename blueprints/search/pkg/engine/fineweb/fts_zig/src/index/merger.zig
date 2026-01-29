//! Background segment merger
//! Implements tiered merge policy for LSM-style indexing

const std = @import("std");
const Allocator = std.mem.Allocator;
const Thread = std.Thread;
const segment = @import("segment.zig");

fn ManagedArrayList(comptime T: type) type {
    return std.array_list.AlignedManaged(T, null);
}

/// Merge policy configuration
pub const MergePolicy = struct {
    /// Number of segments at each level before merging
    segments_per_level: u32 = 4,
    /// Maximum number of levels
    max_levels: u8 = 4,
    /// Minimum segment size for merging (bytes)
    min_segment_size: usize = 1024 * 1024, // 1MB
};

/// Merge task representing segments to merge
pub const MergeTask = struct {
    level: u8,
    segments: []segment.SegmentId,
    target_level: u8,
};

/// Background merger worker
pub const Merger = struct {
    allocator: Allocator,
    policy: MergePolicy,
    base_path: []const u8,
    running: std.atomic.Value(bool),
    thread: ?Thread,

    const Self = @This();

    pub fn init(allocator: Allocator, base_path: []const u8, policy: MergePolicy) Self {
        return .{
            .allocator = allocator,
            .policy = policy,
            .base_path = base_path,
            .running = std.atomic.Value(bool).init(false),
            .thread = null,
        };
    }

    pub fn deinit(self: *Self) void {
        self.stop();
    }

    /// Start background merger thread
    pub fn start(self: *Self) !void {
        if (self.running.load(.acquire)) return;

        self.running.store(true, .release);
        self.thread = try Thread.spawn(.{}, mergeLoop, .{self});
    }

    /// Stop background merger
    pub fn stop(self: *Self) void {
        self.running.store(false, .release);
        if (self.thread) |t| {
            t.join();
            self.thread = null;
        }
    }

    /// Main merge loop
    fn mergeLoop(self: *Self) void {
        while (self.running.load(.acquire)) {
            // Check for merge candidates
            const task = self.findMergeCandidate() catch null;

            if (task) |t| {
                self.executeMerge(t) catch {};
                self.allocator.free(t.segments);
            } else {
                // No work, sleep
                std.time.sleep(100 * std.time.ns_per_ms);
            }
        }
    }

    /// Find segments that should be merged
    fn findMergeCandidate(self: *Self) !?MergeTask {
        // Scan directory for segments at each level
        var dir = std.fs.cwd().openDir(self.base_path, .{ .iterate = true }) catch return null;
        defer dir.close();

        // Count segments per level
        var level_counts: [4]u32 = .{ 0, 0, 0, 0 };
        var level_segments: [4]ManagedArrayList(segment.SegmentId) = undefined;

        for (&level_segments) |*ls| {
            ls.* = ManagedArrayList(segment.SegmentId).init(self.allocator);
        }
        defer for (&level_segments) |*ls| {
            ls.deinit();
        };

        var iter = dir.iterate();
        while (try iter.next()) |entry| {
            if (entry.kind != .file) continue;
            if (!std.mem.endsWith(u8, entry.name, ".fts")) continue;

            // Parse segment name: seg_gen_level_seq.fts
            var parts = std.mem.split(u8, entry.name[0 .. entry.name.len - 4], "_");
            _ = parts.next(); // "seg"

            const gen_str = parts.next() orelse continue;
            const level_str = parts.next() orelse continue;
            const seq_str = parts.next() orelse continue;

            const gen = std.fmt.parseInt(u32, gen_str, 10) catch continue;
            const level = std.fmt.parseInt(u8, level_str, 10) catch continue;
            const seq = std.fmt.parseInt(u32, seq_str, 10) catch continue;

            if (level >= 4) continue;

            level_counts[level] += 1;
            try level_segments[level].append(.{
                .generation = gen,
                .level = level,
                .sequence = seq,
            });
        }

        // Find level with enough segments to merge
        for (0..self.policy.max_levels) |level| {
            if (level_counts[level] >= self.policy.segments_per_level) {
                const segs = level_segments[level].items;
                const task_segs = try self.allocator.alloc(segment.SegmentId, self.policy.segments_per_level);
                @memcpy(task_segs, segs[0..self.policy.segments_per_level]);

                return MergeTask{
                    .level = @intCast(level),
                    .segments = task_segs,
                    .target_level = @intCast(level + 1),
                };
            }
        }

        return null;
    }

    /// Execute a merge task
    fn executeMerge(self: *Self, task: MergeTask) !void {
        _ = self;
        _ = task;
        // TODO: Implement actual merge logic
        // 1. Open all source segments
        // 2. Create new target segment at target_level
        // 3. Merge posting lists
        // 4. Write merged segment
        // 5. Delete source segments
    }
};

/// Manual merge trigger (for testing/debugging)
pub fn forceMerge(allocator: Allocator, base_path: []const u8, target_level: u8) !void {
    _ = allocator;
    _ = base_path;
    _ = target_level;
    // TODO: Force merge all segments to target level
}

// ============================================================================
// Tests
// ============================================================================

test "merger init" {
    var merger = Merger.init(
        std.testing.allocator,
        "/tmp/fts_zig_merge_test",
        .{},
    );
    defer merger.deinit();

    // Just test init/deinit
    try std.testing.expect(!merger.running.load(.acquire));
}
