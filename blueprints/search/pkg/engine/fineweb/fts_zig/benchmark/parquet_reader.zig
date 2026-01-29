//! Parquet file reader for FineWeb-2 Vietnamese dataset
//! Minimal implementation for reading text column from parquet files
//!
//! Note: This is a simplified reader. For production, use a full parquet library.
//! The FineWeb-2 dataset uses Snappy compression which requires external library.

const std = @import("std");
const Allocator = std.mem.Allocator;

// Use managed array list (stores allocator internally)
fn ManagedArrayList(comptime T: type) type {
    return std.array_list.AlignedManaged(T, null);
}

/// Parquet magic bytes
const PARQUET_MAGIC = "PAR1";

/// Parquet file reader
pub const ParquetReader = struct {
    allocator: Allocator,
    file: std.fs.File,
    file_size: u64,
    footer_size: u32,

    const Self = @This();

    pub fn open(allocator: Allocator, path: []const u8) !Self {
        const file = try std.fs.cwd().openFile(path, .{});
        errdefer file.close();

        const stat = try file.stat();
        const file_size = stat.size;

        // Read footer length (last 8 bytes: 4 bytes length + 4 bytes magic)
        try file.seekTo(file_size - 8);
        var footer_buf: [8]u8 = undefined;
        _ = try file.readAll(&footer_buf);

        // Verify magic
        if (!std.mem.eql(u8, footer_buf[4..8], PARQUET_MAGIC)) {
            return error.InvalidParquetFile;
        }

        const footer_size = std.mem.readInt(u32, footer_buf[0..4], .little);

        return Self{
            .allocator = allocator,
            .file = file,
            .file_size = file_size,
            .footer_size = footer_size,
        };
    }

    pub fn close(self: *Self) void {
        self.file.close();
    }

    /// Read documents from the parquet file
    /// Note: This is a placeholder - actual implementation would need
    /// to parse Thrift-encoded metadata and decompress data pages
    pub fn readDocuments(self: *Self, max_docs: u32) ![][]u8 {
        _ = self;
        _ = max_docs;

        // For now, return empty - actual parquet parsing would go here
        // Real implementation would:
        // 1. Read and parse Thrift-encoded FileMetaData from footer
        // 2. Find the "text" column
        // 3. Read row groups and data pages
        // 4. Decompress (Snappy) and decode (PLAIN or DELTA) values

        return &[_][]u8{};
    }
};

/// Read text documents from a directory of parquet files
pub fn readFineWebDataset(allocator: Allocator, dir_path: []const u8, max_docs: u32) ![][]u8 {
    var docs = ManagedArrayList([]u8).init(allocator);
    errdefer {
        for (docs.items) |d| allocator.free(d);
        docs.deinit();
    }

    // List parquet files
    var dir = try std.fs.cwd().openDir(dir_path, .{ .iterate = true });
    defer dir.close();

    var iter = dir.iterate();
    while (try iter.next()) |entry| {
        if (docs.items.len >= max_docs) break;

        if (entry.kind != .file) continue;
        if (!std.mem.endsWith(u8, entry.name, ".parquet")) continue;

        // Build full path
        var path_buf: [std.fs.max_path_bytes]u8 = undefined;
        const full_path = try std.fmt.bufPrint(&path_buf, "{s}/{s}", .{ dir_path, entry.name });

        // Try to read file
        var reader = ParquetReader.open(allocator, full_path) catch continue;
        defer reader.close();

        const file_docs = try reader.readDocuments(max_docs - @as(u32, @intCast(docs.items.len)));
        for (file_docs) |d| {
            try docs.append(d);
        }
    }

    return docs.toOwnedSlice();
}

/// Alternative: Read raw text files (one document per line)
pub fn readTextFile(allocator: Allocator, path: []const u8, max_docs: u32) ![][]u8 {
    const file = try std.fs.cwd().openFile(path, .{});
    defer file.close();

    var docs = ManagedArrayList([]u8).init(allocator);
    errdefer {
        for (docs.items) |d| allocator.free(d);
        docs.deinit();
    }

    var buf_reader = std.io.bufferedReader(file.reader());
    var reader = buf_reader.reader();

    var line_buf: [1024 * 1024]u8 = undefined; // 1MB max line

    while (docs.items.len < max_docs) {
        const line = reader.readUntilDelimiter(&line_buf, '\n') catch |err| {
            if (err == error.EndOfStream) break;
            return err;
        };

        if (line.len > 0) {
            const doc = try allocator.dupe(u8, line);
            try docs.append(doc);
        }
    }

    return docs.toOwnedSlice();
}

/// Load Vietnamese text samples for testing
pub fn loadVietnameseSamples(allocator: Allocator, count: u32) ![][]u8 {
    const samples = [_][]const u8{
        "Việt Nam là một quốc gia nằm ở phía đông bán đảo Đông Dương thuộc khu vực Đông Nam Á",
        "Thành phố Hồ Chí Minh là thành phố lớn nhất Việt Nam về dân số và quy mô đô thị hóa",
        "Hà Nội là thủ đô, đồng thời là thành phố đứng thứ hai về dân số của Việt Nam",
        "Đà Nẵng là thành phố trực thuộc trung ương lớn nhất miền Trung Việt Nam",
        "Công nghệ thông tin Việt Nam đang phát triển mạnh mẽ với nhiều công ty khởi nghiệp",
        "Kinh tế Việt Nam duy trì tốc độ tăng trưởng ổn định trong những năm gần đây",
        "Giáo dục Việt Nam đang trong quá trình đổi mới toàn diện và sâu rộng",
        "Du lịch Việt Nam thu hút hàng triệu khách quốc tế mỗi năm",
        "Ẩm thực Việt Nam nổi tiếng với phở, bánh mì và nhiều món ăn đặc sản khác",
        "Văn hóa Việt Nam đa dạng với 54 dân tộc anh em cùng sinh sống",
        "Trí tuệ nhân tạo đang được ứng dụng rộng rãi trong nhiều lĩnh vực tại Việt Nam",
        "Internet và công nghệ số đang thay đổi cuộc sống của người dân Việt Nam",
        "Blockchain và tiền điện tử đang thu hút sự quan tâm của nhiều nhà đầu tư",
        "Thương mại điện tử Việt Nam phát triển nhanh chóng trong những năm qua",
        "Nông nghiệp Việt Nam đóng vai trò quan trọng trong nền kinh tế quốc dân",
    };

    var docs = try allocator.alloc([]u8, count);
    var prng = std.Random.DefaultPrng.init(12345);
    const random = prng.random();

    for (0..count) |i| {
        // Combine 2-4 samples
        const num_parts = 2 + random.uintLessThan(usize, 3);
        var total_len: usize = 0;
        var parts: [4][]const u8 = undefined;

        for (0..num_parts) |j| {
            parts[j] = samples[random.uintLessThan(usize, samples.len)];
            total_len += parts[j].len + 1;
        }

        var doc = try allocator.alloc(u8, total_len);
        var offset: usize = 0;

        for (0..num_parts) |j| {
            @memcpy(doc[offset..][0..parts[j].len], parts[j]);
            offset += parts[j].len;
            if (j < num_parts - 1) {
                doc[offset] = ' ';
                offset += 1;
            }
        }

        docs[i] = doc[0..offset];
    }

    return docs;
}

test "load vietnamese samples" {
    const docs = try loadVietnameseSamples(std.testing.allocator, 100);
    defer {
        for (docs) |d| std.testing.allocator.free(d);
        std.testing.allocator.free(docs);
    }

    try std.testing.expectEqual(@as(usize, 100), docs.len);
    for (docs) |d| {
        try std.testing.expect(d.len > 0);
    }
}
