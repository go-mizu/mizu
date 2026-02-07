const std = @import("std");
const types = @import("types.zig");

/// Load seed URLs from a TSV file (url\tdomain per line).
/// Returns owned slices - caller must free with the same allocator.
pub fn loadFromTsv(allocator: std.mem.Allocator, path: []const u8) !SeedData {
    const file = try std.fs.cwd().openFile(path, .{});
    defer file.close();

    const stat = try file.stat();
    const content = try allocator.alloc(u8, stat.size);
    const bytes_read = try file.readAll(content);
    const data = content[0..bytes_read];

    return parseTsvContent(allocator, data);
}

/// Load seeds by running duckdb to extract from a parquet file.
pub fn loadFromParquet(allocator: std.mem.Allocator, parquet_path: []const u8, status_filter: u16, limit: u64) !SeedData {
    // Build the DuckDB query
    var query_buf: [2048]u8 = undefined;
    const query = if (limit > 0)
        std.fmt.bufPrint(&query_buf, "SELECT url, COALESCE(url_host_registered_domain,'') FROM read_parquet('{s}') WHERE fetch_status = {d} AND warc_filename IS NOT NULL LIMIT {d}", .{ parquet_path, status_filter, limit }) catch return error.QueryTooLong
    else
        std.fmt.bufPrint(&query_buf, "SELECT url, COALESCE(url_host_registered_domain,'') FROM read_parquet('{s}') WHERE fetch_status = {d} AND warc_filename IS NOT NULL", .{ parquet_path, status_filter }) catch return error.QueryTooLong;

    const result = try std.process.Child.run(.{
        .allocator = allocator,
        .argv = &.{ "duckdb", "-csv", "-separator", "\t", "-noheader", ":memory:", query },
        .max_output_bytes = 512 * 1024 * 1024, // 512MB max
    });
    defer allocator.free(result.stderr);

    if (result.term.Exited != 0) {
        std.debug.print("duckdb error: {s}\n", .{result.stderr});
        allocator.free(result.stdout);
        return error.DuckDBFailed;
    }

    return parseTsvContent(allocator, result.stdout);
}

pub const SeedData = struct {
    seeds: []types.SeedUrl,
    domains: []types.DomainInfo,
    domain_map: std.StringHashMap(u32),
    // Owned data that must be freed
    raw_data: []const u8,
    allocator: std.mem.Allocator,

    pub fn deinit(self: *SeedData) void {
        self.allocator.free(self.seeds);
        self.allocator.free(self.domains);
        self.domain_map.deinit();
        self.allocator.free(self.raw_data);
    }
};

fn parseTsvContent(allocator: std.mem.Allocator, data: []const u8) !SeedData {
    // Count lines for pre-allocation
    var line_count: usize = 0;
    for (data) |c| {
        if (c == '\n') line_count += 1;
    }
    if (data.len > 0 and data[data.len - 1] != '\n') line_count += 1;

    var seeds = try allocator.alloc(types.SeedUrl, line_count);
    var domain_map = std.StringHashMap(u32).init(allocator);
    var domain_list = std.ArrayList(types.DomainInfo){};

    var seed_count: usize = 0;
    var line_iter = std.mem.splitScalar(u8, data, '\n');

    while (line_iter.next()) |line| {
        if (line.len == 0) continue;

        // Split on tab
        if (std.mem.indexOfScalar(u8, line, '\t')) |tab_pos| {
            const url = line[0..tab_pos];
            const domain = line[tab_pos + 1 ..];

            // Get or create domain ID
            const domain_id = blk: {
                if (domain_map.get(domain)) |id| {
                    break :blk id;
                }
                const id: u32 = @intCast(domain_list.items.len);
                try domain_map.put(domain, id);
                var info = types.DomainInfo.init(domain);
                info.url_start = @intCast(seed_count);
                try domain_list.append(allocator, info);
                break :blk id;
            };

            // Update domain URL count
            domain_list.items[domain_id].url_count += 1;

            seeds[seed_count] = .{
                .url = url,
                .domain = domain,
                .domain_id = domain_id,
            };
            seed_count += 1;
        }
    }

    seeds = try allocator.realloc(seeds, seed_count);
    const domains = try domain_list.toOwnedSlice(allocator);

    return .{
        .seeds = seeds,
        .domains = domains,
        .domain_map = domain_map,
        .raw_data = data,
        .allocator = allocator,
    };
}

/// Load already-crawled URLs from result TSV files in a directory.
pub fn loadAlreadyCrawled(allocator: std.mem.Allocator, result_dir: []const u8) !std.StringHashMap(void) {
    var done = std.StringHashMap(void).init(allocator);

    var dir = std.fs.cwd().openDir(result_dir, .{ .iterate = true }) catch return done;
    defer dir.close();

    var iter = dir.iterate();
    while (try iter.next()) |entry| {
        if (!std.mem.startsWith(u8, entry.name, "results_") or !std.mem.endsWith(u8, entry.name, ".tsv")) continue;

        var path_buf: [1024]u8 = undefined;
        const path = std.fmt.bufPrint(&path_buf, "{s}/{s}", .{ result_dir, entry.name }) catch continue;

        const file = dir.openFile(entry.name, .{}) catch continue;
        defer file.close();

        const stat = file.stat() catch continue;
        const content = allocator.alloc(u8, stat.size) catch continue;
        defer allocator.free(content);
        const bytes_read = file.readAll(content) catch continue;
        const file_data = content[0..bytes_read];

        var line_iter = std.mem.splitScalar(u8, file_data, '\n');
        // Skip header
        _ = line_iter.next();

        while (line_iter.next()) |line| {
            if (line.len == 0) continue;
            if (std.mem.indexOfScalar(u8, line, '\t')) |tab_pos| {
                const url = line[0..tab_pos];
                const url_copy = allocator.dupe(u8, url) catch continue;
                done.put(url_copy, {}) catch continue;
            }
        }
        _ = path;
    }

    return done;
}

test "parseTsvContent" {
    const allocator = std.testing.allocator;
    const data = try allocator.dupe(u8, "https://example.com/page1\texample.com\nhttps://test.org/page2\ttest.org\n");
    var seed_data = try parseTsvContent(allocator, data);
    defer seed_data.deinit();

    try std.testing.expectEqual(@as(usize, 2), seed_data.seeds.len);
    try std.testing.expectEqual(@as(usize, 2), seed_data.domains.len);
    try std.testing.expectEqualStrings("https://example.com/page1", seed_data.seeds[0].url);
    try std.testing.expectEqualStrings("example.com", seed_data.seeds[0].domain);
}
