const std = @import("std");
const types = @import("types.zig");
const stats_mod = @import("stats.zig");
const dns_mod = @import("dns.zig");
const http_mod = @import("http.zig");
const seeds_mod = @import("seeds.zig");
const results_mod = @import("results.zig");
const display_mod = @import("display.zig");

const log = std.debug.print;

const Config = struct {
    // Input
    seeds_file: ?[]const u8 = null,
    parquet_file: ?[]const u8 = null,

    // Output
    output_dir: []const u8 = "results",
    dns_cache: ?[]const u8 = null,
    result_shards: u32 = 16,

    // Workers
    workers: u32 = 1024,
    dns_workers: u32 = 64,
    timeout_ms: u32 = 5000,
    dns_timeout_ms: u32 = 2000,

    // Throttling
    max_conns_per_domain: u8 = 8,
    fail_threshold: u16 = 2,

    // Behavior
    status_only: bool = true,
    head_only: bool = false,
    limit: u64 = 0, // 0 = no limit
    status_filter: u16 = 200,
    skip_done: bool = true, // skip already-crawled URLs
};

pub fn main() !void {
    var gpa = std.heap.GeneralPurposeAllocator(.{}){};
    defer _ = gpa.deinit();
    const allocator = gpa.allocator();

    var config = Config{};
    parseArgs(allocator, &config) catch |err| {
        log("Error parsing arguments: {s}\n", .{@errorName(err)});
        printUsage();
        std.process.exit(1);
    };

    if (config.seeds_file == null and config.parquet_file == null) {
        printUsage();
        std.process.exit(1);
    }

    run(allocator, &config) catch |err| {
        log("Fatal error: {s}\n", .{@errorName(err)});
        std.process.exit(1);
    };
}

fn run(allocator: std.mem.Allocator, config: *const Config) !void {
    // ── Phase 0: Load seeds ──
    log("\n  Loading seeds...\n", .{});

    var seed_data = if (config.parquet_file) |pf|
        try seeds_mod.loadFromParquet(allocator, pf, config.status_filter, config.limit)
    else if (config.seeds_file) |sf|
        try seeds_mod.loadFromTsv(allocator, sf)
    else
        return error.NoInput;

    defer seed_data.deinit();

    log("  Loaded {d} URLs from {d} domains\n\n", .{ seed_data.seeds.len, seed_data.domains.len });

    if (seed_data.seeds.len == 0) {
        log("  No seeds to process.\n", .{});
        return;
    }

    // ── Phase 0.5: Resume — skip already-crawled URLs ──
    var already_crawled: ?std.StringHashMap(void) = null;
    var active_seeds: []const types.SeedUrl = seed_data.seeds;
    var filtered_seeds_buf: ?[]types.SeedUrl = null;

    if (config.skip_done) {
        already_crawled = seeds_mod.loadAlreadyCrawled(allocator, config.output_dir) catch blk: {
            break :blk null;
        };
        if (already_crawled) |*ac| {
            if (ac.count() > 0) {
                log("  Resuming: {d} URLs already crawled\n", .{ac.count()});
                // Filter out already-crawled URLs
                var filtered = std.ArrayList(types.SeedUrl){};
                for (seed_data.seeds) |seed| {
                    if (!ac.contains(seed.url)) {
                        filtered.append(allocator, seed) catch continue;
                    }
                }
                filtered_seeds_buf = filtered.toOwnedSlice(allocator) catch null;
                if (filtered_seeds_buf) |fsb| {
                    active_seeds = fsb;
                    log("  Remaining: {d} URLs to crawl\n\n", .{active_seeds.len});
                }
            }
        }
    }
    defer {
        if (filtered_seeds_buf) |fsb| allocator.free(fsb);
        if (already_crawled) |*ac| {
            var key_iter = ac.keyIterator();
            while (key_iter.next()) |key| {
                allocator.free(key.*);
            }
            ac.deinit();
        }
    }

    if (active_seeds.len == 0) {
        log("  All URLs already crawled. Nothing to do.\n", .{});
        return;
    }

    // ── Initialize stats ──
    var stats = stats_mod.Stats.init(@intCast(active_seeds.len), @intCast(seed_data.domains.len));

    // ── Phase 1: DNS Resolution ──
    log("  Phase 1: DNS Resolution\n", .{});

    var dns_resolver = dns_mod.DnsResolver.init(allocator, config.dns_timeout_ms);
    defer dns_resolver.deinit();

    // Load DNS cache if available
    if (config.dns_cache) |cache_path| {
        const loaded = dns_mod.loadDnsCache(seed_data.domains, cache_path) catch 0;
        if (loaded > 0) {
            log("  Loaded DNS cache: {d} entries\n", .{loaded});
        }
    }

    dns_resolver.resolveBatch(seed_data.domains, config.dns_workers, &stats, null);

    const dns_live = dns_resolver.resolved.load(.acquire);
    const dns_dead = dns_resolver.dead.load(.acquire);
    const dns_tout = dns_resolver.timed_out.load(.acquire);
    log("  DNS complete in {d}ms: {d} live, {d} dead, {d} timeout\n\n", .{
        dns_resolver.duration_ms,
        dns_live,
        dns_dead,
        dns_tout,
    });

    // Save DNS cache
    if (config.dns_cache) |cache_path| {
        dns_mod.saveDnsCache(seed_data.domains, cache_path) catch {};
    }

    // Count live URLs (skip dead domains)
    var live_urls: u64 = 0;
    for (active_seeds) |seed| {
        if (seed.domain_id < seed_data.domains.len) {
            if (!seed_data.domains[seed.domain_id].isDead()) {
                live_urls += 1;
            }
        }
    }
    log("  Live URLs to fetch: {d}\n\n", .{live_urls});

    if (live_urls == 0) {
        log("  No live domains. Exiting.\n", .{});
        return;
    }

    // ── Phase 2: HTTP Fetch ──
    log("  Phase 2: HTTP Fetch\n", .{});

    var result_writer = try results_mod.ResultWriter.init(allocator, config.output_dir, config.result_shards);
    defer result_writer.close();

    var fetcher = http_mod.HttpFetcher.init(
        seed_data.domains,
        &stats,
        &result_writer,
        .{
            .workers = config.workers,
            .timeout_ms = config.timeout_ms,
            .max_conns_per_domain = config.max_conns_per_domain,
            .fail_threshold = config.fail_threshold,
            .status_only = config.status_only,
            .head_only = config.head_only,
        },
    );

    // Start display thread
    var display = display_mod.Display.init("zig-recrawler");
    var display_running = std.atomic.Value(bool).init(true);

    const DisplayCtx = struct {
        display: *display_mod.Display,
        stats: *stats_mod.Stats,
        running: *std.atomic.Value(bool),
    };
    var display_ctx = DisplayCtx{
        .display = &display,
        .stats = &stats,
        .running = &display_running,
    };

    const display_thread = std.Thread.spawn(.{ .stack_size = 256 * 1024 }, displayLoop, .{&display_ctx}) catch null;

    // Run HTTP fetching
    fetcher.fetchAll(allocator, active_seeds);

    // Stop display
    display_running.store(false, .release);
    if (display_thread) |dt| dt.join();

    // Flush results
    result_writer.flush();
    stats.freeze();

    // ── Final display ──
    display.render(&stats);

    // Summary
    const elapsed_s = @as(f64, @floatFromInt(stats.elapsedMs())) / 1000.0;
    const total_speed = if (elapsed_s > 0) @as(f64, @floatFromInt(stats.fetched())) / elapsed_s else 0;
    log("\n  Done. {d} results written to {s}/\n", .{ result_writer.totalWritten(), config.output_dir });
    log("  Total: {d} fetched in {d:.1}s ({d:.0} URLs/s)\n\n", .{
        stats.fetched(),
        elapsed_s,
        total_speed,
    });
}

fn displayLoop(ctx: anytype) void {
    while (ctx.running.load(.acquire)) {
        ctx.display.render(ctx.stats);
        std.Thread.sleep(500 * std.time.ns_per_ms);
    }
}

fn parseArgs(allocator: std.mem.Allocator, config: *Config) !void {
    _ = allocator;
    var args = std.process.args();
    _ = args.next(); // skip program name

    while (args.next()) |arg| {
        if (std.mem.eql(u8, arg, "--seeds") or std.mem.eql(u8, arg, "-s")) {
            config.seeds_file = args.next() orelse return error.MissingValue;
        } else if (std.mem.eql(u8, arg, "--parquet") or std.mem.eql(u8, arg, "-p")) {
            config.parquet_file = args.next() orelse return error.MissingValue;
        } else if (std.mem.eql(u8, arg, "--output") or std.mem.eql(u8, arg, "-o")) {
            config.output_dir = args.next() orelse return error.MissingValue;
        } else if (std.mem.eql(u8, arg, "--dns-cache")) {
            config.dns_cache = args.next() orelse return error.MissingValue;
        } else if (std.mem.eql(u8, arg, "--workers") or std.mem.eql(u8, arg, "-w")) {
            config.workers = try parseU32(args.next() orelse return error.MissingValue);
        } else if (std.mem.eql(u8, arg, "--dns-workers")) {
            config.dns_workers = try parseU32(args.next() orelse return error.MissingValue);
        } else if (std.mem.eql(u8, arg, "--timeout") or std.mem.eql(u8, arg, "-t")) {
            config.timeout_ms = try parseU32(args.next() orelse return error.MissingValue);
        } else if (std.mem.eql(u8, arg, "--dns-timeout")) {
            config.dns_timeout_ms = try parseU32(args.next() orelse return error.MissingValue);
        } else if (std.mem.eql(u8, arg, "--max-conns-per-domain")) {
            config.max_conns_per_domain = @intCast(try parseU32(args.next() orelse return error.MissingValue));
        } else if (std.mem.eql(u8, arg, "--fail-threshold")) {
            config.fail_threshold = @intCast(try parseU32(args.next() orelse return error.MissingValue));
        } else if (std.mem.eql(u8, arg, "--shards")) {
            config.result_shards = try parseU32(args.next() orelse return error.MissingValue);
        } else if (std.mem.eql(u8, arg, "--limit")) {
            config.limit = try parseU64(args.next() orelse return error.MissingValue);
        } else if (std.mem.eql(u8, arg, "--status-filter")) {
            config.status_filter = @intCast(try parseU32(args.next() orelse return error.MissingValue));
        } else if (std.mem.eql(u8, arg, "--status-only")) {
            config.status_only = true;
        } else if (std.mem.eql(u8, arg, "--full-body")) {
            config.status_only = false;
        } else if (std.mem.eql(u8, arg, "--head")) {
            config.head_only = true;
        } else if (std.mem.eql(u8, arg, "--no-resume")) {
            config.skip_done = false;
        } else if (std.mem.eql(u8, arg, "--help") or std.mem.eql(u8, arg, "-h")) {
            printUsage();
            std.process.exit(0);
        }
    }
}

fn parseU32(s: []const u8) !u32 {
    return std.fmt.parseInt(u32, s, 10) catch return error.InvalidNumber;
}

fn parseU64(s: []const u8) !u64 {
    return std.fmt.parseInt(u64, s, 10) catch return error.InvalidNumber;
}

fn printUsage() void {
    log(
        \\
        \\  zig-recrawler - High-throughput URL recrawler
        \\
        \\  USAGE:
        \\    zig-recrawler --seeds <file.tsv>  [options]
        \\    zig-recrawler --parquet <file.parquet> [options]
        \\
        \\  INPUT:
        \\    --seeds, -s <file>       TSV file with url\tdomain per line
        \\    --parquet, -p <file>     Parquet file (uses duckdb to extract)
        \\    --limit <n>              Limit number of URLs (parquet mode)
        \\    --status-filter <code>   Filter by fetch_status (default: 200)
        \\
        \\  OUTPUT:
        \\    --output, -o <dir>       Output directory (default: results)
        \\    --shards <n>             Number of result shards (default: 16)
        \\    --dns-cache <file>       DNS cache file (load/save)
        \\
        \\  WORKERS:
        \\    --workers, -w <n>        HTTP worker threads (default: 1024)
        \\    --dns-workers <n>        DNS worker threads (default: 64)
        \\    --timeout, -t <ms>       HTTP timeout in ms (default: 5000)
        \\    --dns-timeout <ms>       DNS timeout in ms (default: 2000)
        \\
        \\  THROTTLING:
        \\    --max-conns-per-domain <n>  Max concurrent connections per domain (default: 8)
        \\    --fail-threshold <n>        Failures before marking domain dead (default: 2)
        \\
        \\  BEHAVIOR:
        \\    --status-only            Only fetch headers (default)
        \\    --full-body              Fetch full response body
        \\    --head                   Use HEAD requests
        \\    --no-resume              Don't skip already-crawled URLs
        \\    --help, -h               Show this help
        \\
    , .{});
}

// Re-export all modules for testing
test {
    _ = @import("types.zig");
    _ = @import("stats.zig");
    _ = @import("seeds.zig");
    _ = @import("dns.zig");
    _ = @import("http.zig");
    _ = @import("results.zig");
    _ = @import("display.zig");
}
