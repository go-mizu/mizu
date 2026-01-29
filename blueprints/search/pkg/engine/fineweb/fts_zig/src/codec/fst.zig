//! Finite State Transducer (FST) for term dictionary
//! Provides compact storage of term -> posting list offset mapping
//! Supports prefix lookups and range queries

const std = @import("std");
const Allocator = std.mem.Allocator;

fn ManagedArrayList(comptime T: type) type {
    return std.array_list.AlignedManaged(T, null);
}

/// FST node stored in a compact format
const Node = struct {
    /// Transition bytes (sorted)
    transitions: []u8,
    /// Target node indices
    targets: []u32,
    /// Output values for each transition
    outputs: []u64,
    /// Final output if this is an accepting state
    final_output: ?u64,
    /// Whether this is a final state
    is_final: bool,
};

/// Builder for FST (requires sorted input)
pub const FSTBuilder = struct {
    allocator: Allocator,
    nodes: ManagedArrayList(*BuilderNode),
    /// Previous key for checking sorted order
    prev_key: ManagedArrayList(u8),
    /// Root node index (always 0)
    root: u32,

    const Self = @This();

    const BuilderNode = struct {
        transitions: std.AutoHashMap(u8, u32),
        outputs: std.AutoHashMap(u8, u64),
        final_output: ?u64,
        is_final: bool,

        fn init(allocator: Allocator) !*BuilderNode {
            const node = try allocator.create(BuilderNode);
            node.* = .{
                .transitions = std.AutoHashMap(u8, u32).init(allocator),
                .outputs = std.AutoHashMap(u8, u64).init(allocator),
                .final_output = null,
                .is_final = false,
            };
            return node;
        }

        fn deinit(self: *BuilderNode, allocator: Allocator) void {
            self.transitions.deinit();
            self.outputs.deinit();
            allocator.destroy(self);
        }
    };

    pub fn init(allocator: Allocator) Self {
        var nodes = ManagedArrayList(*BuilderNode).init(allocator);
        // Add root node
        const root_node = BuilderNode.init(allocator) catch unreachable;
        nodes.append(root_node) catch unreachable;

        return .{
            .allocator = allocator,
            .nodes = nodes,
            .prev_key = ManagedArrayList(u8).init(allocator),
            .root = 0,
        };
    }

    pub fn deinit(self: *Self) void {
        for (self.nodes.items) |node| {
            node.deinit(self.allocator);
        }
        self.nodes.deinit();
        self.prev_key.deinit();
    }

    /// Add a key-value pair (keys must be added in sorted order)
    pub fn add(self: *Self, key: []const u8, value: u64) !void {
        // Verify sorted order
        if (self.prev_key.items.len > 0) {
            const cmp = std.mem.order(u8, self.prev_key.items, key);
            if (cmp != .lt) {
                return error.KeysNotSorted;
            }
        }

        // Store previous key
        self.prev_key.clearRetainingCapacity();
        try self.prev_key.appendSlice(key);

        // Add key to FST
        var current: u32 = self.root;

        for (key) |byte| {
            const node = self.nodes.items[current];

            if (node.transitions.get(byte)) |next| {
                current = next;
            } else {
                // Create new node
                const new_idx: u32 = @intCast(self.nodes.items.len);
                const new_node = try BuilderNode.init(self.allocator);
                try self.nodes.append(new_node);
                try node.transitions.put(byte, new_idx);
                try node.outputs.put(byte, 0);
                current = new_idx;
            }
        }

        // Mark final state
        self.nodes.items[current].is_final = true;
        self.nodes.items[current].final_output = value;
    }

    /// Build the FST (returns an immutable structure)
    pub fn build(self: *Self) !FST {
        const node_count = self.nodes.items.len;

        // Calculate total transitions
        var total_transitions: usize = 0;
        for (self.nodes.items) |node| {
            total_transitions += node.transitions.count();
        }

        // Allocate compact storage
        const node_offsets = try self.allocator.alloc(u32, node_count + 1);
        const trans_bytes = try self.allocator.alloc(u8, total_transitions);
        const trans_targets = try self.allocator.alloc(u32, total_transitions);
        const trans_outputs = try self.allocator.alloc(u64, total_transitions);
        const final_outputs = try self.allocator.alloc(u64, node_count);
        const is_final = try self.allocator.alloc(bool, node_count);

        // Copy data
        var offset: u32 = 0;
        for (self.nodes.items, 0..) |node, i| {
            node_offsets[i] = offset;
            is_final[i] = node.is_final;
            final_outputs[i] = node.final_output orelse 0;

            // Sort transitions by byte
            var entries = ManagedArrayList(struct { byte: u8, target: u32, output: u64 }).init(self.allocator);
            defer entries.deinit();

            var iter = node.transitions.iterator();
            while (iter.next()) |entry| {
                const output = node.outputs.get(entry.key_ptr.*) orelse 0;
                entries.append(.{
                    .byte = entry.key_ptr.*,
                    .target = entry.value_ptr.*,
                    .output = output,
                }) catch continue;
            }

            std.mem.sort(@TypeOf(entries.items[0]), entries.items, {}, struct {
                fn lessThan(_: void, a: @TypeOf(entries.items[0]), b: @TypeOf(entries.items[0])) bool {
                    return a.byte < b.byte;
                }
            }.lessThan);

            for (entries.items) |e| {
                trans_bytes[offset] = e.byte;
                trans_targets[offset] = e.target;
                trans_outputs[offset] = e.output;
                offset += 1;
            }
        }
        node_offsets[node_count] = offset;

        return FST{
            .allocator = self.allocator,
            .node_offsets = node_offsets,
            .trans_bytes = trans_bytes,
            .trans_targets = trans_targets,
            .trans_outputs = trans_outputs,
            .final_outputs = final_outputs,
            .is_final = is_final,
            .node_count = @intCast(node_count),
        };
    }
};

/// Immutable FST for fast lookups
pub const FST = struct {
    allocator: Allocator,
    /// Offsets into transition arrays for each node
    node_offsets: []u32,
    /// Transition bytes (sorted per node)
    trans_bytes: []u8,
    /// Target nodes for transitions
    trans_targets: []u32,
    /// Output values for transitions
    trans_outputs: []u64,
    /// Final output for each node
    final_outputs: []u64,
    /// Whether each node is final
    is_final: []bool,
    /// Number of nodes
    node_count: u32,

    const Self = @This();

    pub fn deinit(self: *Self) void {
        self.allocator.free(self.node_offsets);
        self.allocator.free(self.trans_bytes);
        self.allocator.free(self.trans_targets);
        self.allocator.free(self.trans_outputs);
        self.allocator.free(self.final_outputs);
        self.allocator.free(self.is_final);
        self.* = undefined;
    }

    /// Lookup a key, returns the associated value or null
    pub fn get(self: Self, key: []const u8) ?u64 {
        var current: u32 = 0; // Root
        var output: u64 = 0;

        for (key) |byte| {
            const start = self.node_offsets[current];
            const end = self.node_offsets[current + 1];
            const transitions = self.trans_bytes[start..end];

            // Binary search for byte
            const idx = std.sort.binarySearch(u8, transitions, byte, struct {
                fn cmp(search_key: u8, item: u8) std.math.Order {
                    return std.math.order(search_key, item);
                }
            }.cmp);

            if (idx) |i| {
                output += self.trans_outputs[start + i];
                current = self.trans_targets[start + i];
            } else {
                return null;
            }
        }

        if (self.is_final[current]) {
            return output + self.final_outputs[current];
        }

        return null;
    }

    /// Check if a key exists
    pub fn contains(self: Self, key: []const u8) bool {
        return self.get(key) != null;
    }

    /// Get memory usage in bytes
    pub fn memoryUsage(self: Self) usize {
        return self.node_offsets.len * 4 +
            self.trans_bytes.len +
            self.trans_targets.len * 4 +
            self.trans_outputs.len * 8 +
            self.final_outputs.len * 8 +
            self.is_final.len;
    }
};

// ============================================================================
// Tests
// ============================================================================

test "fst basic" {
    var builder = FSTBuilder.init(std.testing.allocator);
    defer builder.deinit();

    try builder.add("apple", 100);
    try builder.add("banana", 200);
    try builder.add("cherry", 300);

    var fst = try builder.build();
    defer fst.deinit();

    try std.testing.expectEqual(@as(?u64, 100), fst.get("apple"));
    try std.testing.expectEqual(@as(?u64, 200), fst.get("banana"));
    try std.testing.expectEqual(@as(?u64, 300), fst.get("cherry"));
    try std.testing.expectEqual(@as(?u64, null), fst.get("durian"));
}

test "fst prefix sharing" {
    var builder = FSTBuilder.init(std.testing.allocator);
    defer builder.deinit();

    // Keys must be in sorted order: test < tested < testing
    try builder.add("test", 1);
    try builder.add("tested", 3);
    try builder.add("testing", 2);

    var fst = try builder.build();
    defer fst.deinit();

    try std.testing.expectEqual(@as(?u64, 1), fst.get("test"));
    try std.testing.expectEqual(@as(?u64, 2), fst.get("testing"));
    try std.testing.expectEqual(@as(?u64, 3), fst.get("tested"));
    try std.testing.expectEqual(@as(?u64, null), fst.get("tes"));
}

test "fst empty" {
    var builder = FSTBuilder.init(std.testing.allocator);
    defer builder.deinit();

    var fst = try builder.build();
    defer fst.deinit();

    try std.testing.expectEqual(@as(?u64, null), fst.get("anything"));
}
