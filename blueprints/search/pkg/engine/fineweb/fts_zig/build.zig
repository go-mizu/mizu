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
    // Shared Library (for CGO integration)
    // =========================================================================
    const lib = b.addSharedLibrary(.{
        .name = "fts_zig",
        .root_source_file = b.path("src/main.zig"),
        .target = target,
        .optimize = optimize,
    });
    lib.root_module.addOptions("build_options", options);

    // Link libc for mmap and other system calls
    lib.linkLibC();

    b.installArtifact(lib);

    // =========================================================================
    // Static Library
    // =========================================================================
    const static_lib = b.addStaticLibrary(.{
        .name = "fts_zig_static",
        .root_source_file = b.path("src/main.zig"),
        .target = target,
        .optimize = optimize,
    });
    static_lib.root_module.addOptions("build_options", options);
    static_lib.linkLibC();

    b.installArtifact(static_lib);

    // =========================================================================
    // IPC Server Binary
    // =========================================================================
    const ipc_server = b.addExecutable(.{
        .name = "fts_zig_server",
        .root_source_file = b.path("ipc/main_server.zig"),
        .target = target,
        .optimize = optimize,
    });
    ipc_server.root_module.addOptions("build_options", options);
    ipc_server.root_module.addImport("fts_zig", &lib.root_module);
    ipc_server.linkLibC();

    b.installArtifact(ipc_server);

    // =========================================================================
    // Benchmark Executable
    // =========================================================================
    const bench = b.addExecutable(.{
        .name = "fts_zig_bench",
        .root_source_file = b.path("benchmark/bench_e2e.zig"),
        .target = target,
        .optimize = optimize,
    });
    bench.root_module.addOptions("build_options", options);
    bench.root_module.addImport("fts_zig", &lib.root_module);
    bench.linkLibC();

    b.installArtifact(bench);

    // =========================================================================
    // Unit Tests
    // =========================================================================
    const unit_tests = b.addTest(.{
        .root_source_file = b.path("src/main.zig"),
        .target = target,
        .optimize = optimize,
    });
    unit_tests.root_module.addOptions("build_options", options);
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
    // Generate C Header
    // =========================================================================
    const header_step = b.step("header", "Generate C header for FFI");
    _ = header_step;
    // Header is manually maintained in fts_zig.h for stability
}
