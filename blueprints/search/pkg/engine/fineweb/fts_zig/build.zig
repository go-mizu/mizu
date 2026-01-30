const std = @import("std");

pub fn build(b: *std.Build) void {
    const target = b.standardTargetOptions(.{});
    const optimize = b.standardOptimizeOption(.{});

    // Profile selection
    const profile = b.option([]const u8, "profile", "Search profile: speed, balanced, compact") orelse "balanced";

    // Build options
    const options = b.addOptions();
    options.addOption([]const u8, "profile", profile);

    // =========================================================================
    // Create main module
    // =========================================================================
    const main_mod = b.createModule(.{
        .root_source_file = b.path("src/main.zig"),
        .target = target,
        .optimize = optimize,
    });
    main_mod.addOptions("build_options", options);

    // =========================================================================
    // Shared Library (for CGO integration)
    // =========================================================================
    const lib = b.addLibrary(.{
        .name = "fts_zig",
        .root_module = main_mod,
        .linkage = .dynamic,
    });
    lib.linkLibC();
    b.installArtifact(lib);

    // =========================================================================
    // Static Library
    // =========================================================================
    const static_mod = b.createModule(.{
        .root_source_file = b.path("src/main.zig"),
        .target = target,
        .optimize = optimize,
    });
    static_mod.addOptions("build_options", options);

    const static_lib = b.addLibrary(.{
        .name = "fts_zig_static",
        .root_module = static_mod,
        .linkage = .static,
    });
    static_lib.linkLibC();
    b.installArtifact(static_lib);

    // =========================================================================
    // IPC Server Binary
    // =========================================================================
    const ipc_mod = b.createModule(.{
        .root_source_file = b.path("ipc/main_server.zig"),
        .target = target,
        .optimize = optimize,
    });
    ipc_mod.addImport("fts_zig", main_mod);
    ipc_mod.addOptions("build_options", options);

    const ipc_server = b.addExecutable(.{
        .name = "fts_zig_server",
        .root_module = ipc_mod,
    });
    ipc_server.linkLibC();
    b.installArtifact(ipc_server);

    // =========================================================================
    // Benchmark Executable
    // =========================================================================
    const bench_mod = b.createModule(.{
        .root_source_file = b.path("benchmark/bench_e2e.zig"),
        .target = target,
        .optimize = optimize,
    });
    bench_mod.addImport("fts_zig", main_mod);
    bench_mod.addOptions("build_options", options);

    const bench = b.addExecutable(.{
        .name = "fts_zig_bench",
        .root_module = bench_mod,
    });
    bench.linkLibC();
    b.installArtifact(bench);

    // =========================================================================
    // Unit Tests
    // =========================================================================
    const test_mod = b.createModule(.{
        .root_source_file = b.path("src/main.zig"),
        .target = target,
        .optimize = optimize,
    });
    test_mod.addOptions("build_options", options);

    const unit_tests = b.addTest(.{
        .root_module = test_mod,
    });
    unit_tests.linkLibC();

    const run_unit_tests = b.addRunArtifact(unit_tests);

    const test_step = b.step("test", "Run unit tests");
    test_step.dependOn(&run_unit_tests.step);

    // =========================================================================
    // Benchmark Step
    // =========================================================================
    const run_bench = b.addRunArtifact(bench);
    if (b.args) |args| {
        run_bench.addArgs(args);
    }

    const bench_step = b.step("bench", "Run benchmarks");
    bench_step.dependOn(&run_bench.step);

    // =========================================================================
    // Throughput Benchmark (1M docs/sec target)
    // =========================================================================
    const throughput_mod = b.createModule(.{
        .root_source_file = b.path("benchmark/bench_throughput.zig"),
        .target = target,
        .optimize = .ReleaseFast, // Always optimize for speed
    });
    throughput_mod.addImport("fts_zig", main_mod);
    const parquet_mod = b.createModule(.{
        .root_source_file = b.path("benchmark/parquet_reader.zig"),
        .target = target,
        .optimize = .ReleaseFast,
    });
    parquet_mod.linkSystemLibrary("zstd", .{});
    parquet_mod.addIncludePath(.{ .cwd_relative = "/opt/homebrew/include" });
    parquet_mod.addLibraryPath(.{ .cwd_relative = "/opt/homebrew/lib" });
    throughput_mod.addImport("parquet_reader", parquet_mod);
    throughput_mod.addOptions("build_options", options);

    const throughput_bench = b.addExecutable(.{
        .name = "fts_zig_throughput",
        .root_module = throughput_mod,
    });
    throughput_bench.linkLibC();
    b.installArtifact(throughput_bench);

    const run_throughput = b.addRunArtifact(throughput_bench);
    if (b.args) |args| {
        run_throughput.addArgs(args);
    }

    const throughput_step = b.step("throughput", "Run throughput benchmark (1M docs/sec target)");
    throughput_step.dependOn(&run_throughput.step);
}
