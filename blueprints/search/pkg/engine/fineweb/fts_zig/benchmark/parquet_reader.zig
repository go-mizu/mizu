//! Parquet file reader for FineWeb-2 Vietnamese dataset
//! Reads the "text" column directly from parquet files using:
//!   - Thrift Compact Protocol for metadata parsing
//!   - ZSTD decompression (via std.compress.zstd)
//!   - PLAIN and RLE_DICTIONARY page decoding
//!
//! Returns texts as packed contiguous slices for cache-friendly benchmarking.

const std = @import("std");
const Allocator = std.mem.Allocator;

fn ManagedArrayList(comptime T: type) type {
    return std.array_list.AlignedManaged(T, null);
}

// ============================================================================
// Public API
// ============================================================================

pub const ReadResult = struct {
    docs: [][]const u8,
    buffer: []u8, // backing storage — all doc slices point into this
    total_bytes: u64,

    pub fn deinit(self: *ReadResult, allocator: Allocator) void {
        allocator.free(self.docs);
        std.heap.page_allocator.free(self.buffer);
    }
};

/// Read text column from parquet files in a directory (or a single file).
/// Returns packed contiguous slices suitable for benchmarking.
pub fn readTexts(allocator: Allocator, path: []const u8, max_docs: u32) !ReadResult {
    // Determine if path is a file or directory
    const stat = try std.fs.cwd().statFile(path);
    var files = ManagedArrayList([]const u8).init(allocator);
    defer {
        for (files.items) |f| allocator.free(f);
        files.deinit();
    }

    if (stat.kind == .directory) {
        var dir = try std.fs.cwd().openDir(path, .{ .iterate = true });
        defer dir.close();
        var iter = dir.iterate();
        while (try iter.next()) |entry| {
            if (entry.kind != .file) continue;
            if (!std.mem.endsWith(u8, entry.name, ".parquet")) continue;
            const full = try std.fmt.allocPrint(allocator, "{s}/{s}", .{ path, entry.name });
            try files.append(full);
        }
        // Sort for deterministic order
        std.mem.sort([]const u8, files.items, {}, struct {
            fn lessThan(_: void, a: []const u8, b: []const u8) bool {
                return std.mem.order(u8, a, b) == .lt;
            }
        }.lessThan);
    } else {
        try files.append(try allocator.dupe(u8, path));
    }

    if (files.items.len == 0) return error.NoParquetFiles;

    std.debug.print("Found {d} parquet file(s)\n", .{files.items.len});

    // Phase 1: Read all texts into a temporary list
    var text_list = ManagedArrayList([]const u8).init(allocator);
    defer text_list.deinit();
    // We'll collect owned text copies, then pack them
    var temp_texts = ManagedArrayList([]u8).init(allocator);
    defer {
        for (temp_texts.items) |t| allocator.free(t);
        temp_texts.deinit();
    }

    var total_text_bytes: u64 = 0;

    for (files.items) |file_path| {
        if (max_docs > 0 and temp_texts.items.len >= max_docs) break;

        const remaining = if (max_docs > 0) max_docs - @as(u32, @intCast(temp_texts.items.len)) else 0;
        const file_texts = readParquetFile(allocator, file_path, remaining) catch |err| {
            std.debug.print("Warning: skipping {s}: {s}\n", .{ file_path, @errorName(err) });
            continue;
        };
        defer allocator.free(file_texts.doc_slices);
        defer std.heap.page_allocator.free(file_texts.raw_buffer);

        for (file_texts.doc_slices) |text| {
            if (max_docs > 0 and temp_texts.items.len >= max_docs) break;
            const copy = try allocator.dupe(u8, text);
            try temp_texts.append(copy);
            total_text_bytes += text.len;
        }

        std.debug.print("  {s}: {d} docs, {d:.1} MB text\n", .{
            file_path,
            file_texts.doc_slices.len,
            @as(f64, @floatFromInt(total_text_bytes)) / (1024 * 1024),
        });
    }

    const doc_count = temp_texts.items.len;
    if (doc_count == 0) return error.NoDocuments;

    // Phase 2: Pack into contiguous buffer
    const buffer = try std.heap.page_allocator.alloc(u8, total_text_bytes);
    const docs = try allocator.alloc([]const u8, doc_count);

    var offset: usize = 0;
    for (temp_texts.items, 0..) |text, i| {
        @memcpy(buffer[offset..][0..text.len], text);
        docs[i] = buffer[offset..][0..text.len];
        offset += text.len;
    }

    std.debug.print("Loaded {d} documents, {d:.1} MB total (packed contiguous)\n", .{
        doc_count,
        @as(f64, @floatFromInt(total_text_bytes)) / (1024 * 1024),
    });

    return ReadResult{
        .docs = docs,
        .buffer = buffer,
        .total_bytes = total_text_bytes,
    };
}

// ============================================================================
// Internal: Single file reader
// ============================================================================

const FileTexts = struct {
    doc_slices: [][]const u8,
    raw_buffer: []u8, // backing buffer for doc_slices
};

fn readParquetFile(allocator: Allocator, path: []const u8, max_docs: u32) !FileTexts {
    const file = try std.fs.cwd().openFile(path, .{});
    defer file.close();

    const file_stat = try file.stat();
    const file_size: usize = @intCast(file_stat.size);
    if (file_size < 12) return error.FileTooSmall;

    // Read footer: last 8 bytes = [4 bytes footer_len LE] [4 bytes "PAR1"]
    try file.seekTo(file_size - 8);
    var tail: [8]u8 = undefined;
    _ = try file.readAll(&tail);

    if (!std.mem.eql(u8, tail[4..8], "PAR1")) return error.NotParquet;
    const footer_len: usize = std.mem.readInt(u32, tail[0..4], .little);
    if (footer_len + 8 > file_size) return error.InvalidFooter;

    // Read footer (Thrift-encoded FileMetaData)
    const footer_offset = file_size - 8 - footer_len;
    try file.seekTo(footer_offset);
    const footer_buf = try allocator.alloc(u8, footer_len);
    defer allocator.free(footer_buf);
    _ = try file.readAll(footer_buf);

    // Parse FileMetaData
    const metadata = try parseFileMetaData(allocator, footer_buf);
    defer {
        for (metadata.row_groups) |rg| allocator.free(rg.columns);
        allocator.free(metadata.row_groups);
        for (metadata.schema) |s| allocator.free(s.name);
        allocator.free(metadata.schema);
    }

    // Find text column index
    const text_col_idx = findColumnIndex(metadata.schema, "text") orelse return error.NoTextColumn;

    // Collect all texts from row groups
    var all_texts = ManagedArrayList([]u8).init(allocator);
    defer {
        // Only free if we error — on success, caller owns via raw_buffer
    }
    var total_bytes: usize = 0;

    for (metadata.row_groups) |rg| {
        if (max_docs > 0 and all_texts.items.len >= max_docs) break;
        if (text_col_idx >= rg.columns.len) continue;

        const col = rg.columns[text_col_idx];

        // Read column chunk data (dictionary page + data pages)
        const chunk_offset: usize = @intCast(col.dictionary_page_offset orelse col.data_page_offset);
        const chunk_end: usize = @intCast(col.data_page_offset + col.total_compressed_size);
        const chunk_size = chunk_end - chunk_offset;

        try file.seekTo(chunk_offset);
        const chunk_data = try allocator.alloc(u8, chunk_size);
        defer allocator.free(chunk_data);
        _ = try file.readAll(chunk_data);

        // Decode pages within this column chunk
        const remaining = if (max_docs > 0) max_docs - @as(u32, @intCast(all_texts.items.len)) else 0;
        const texts = try decodeColumnChunk(allocator, chunk_data, col, remaining);
        defer allocator.free(texts);

        for (texts) |text| {
            if (max_docs > 0 and all_texts.items.len >= max_docs) {
                allocator.free(text);
                continue;
            }
            try all_texts.append(text);
            total_bytes += text.len;
        }
    }

    // Pack into contiguous buffer
    const buffer = try std.heap.page_allocator.alloc(u8, total_bytes);
    const doc_slices = try allocator.alloc([]const u8, all_texts.items.len);

    var offset: usize = 0;
    for (all_texts.items, 0..) |text, i| {
        @memcpy(buffer[offset..][0..text.len], text);
        doc_slices[i] = buffer[offset..][0..text.len];
        offset += text.len;
        allocator.free(text);
    }

    return FileTexts{
        .doc_slices = doc_slices,
        .raw_buffer = buffer,
    };
}

// ============================================================================
// Thrift Compact Protocol decoder
// ============================================================================

const ThriftReader = struct {
    data: []const u8,
    pos: usize,
    last_field_id: i16,

    fn init(data: []const u8) ThriftReader {
        return .{ .data = data, .pos = 0, .last_field_id = 0 };
    }

    fn readByte(self: *ThriftReader) !u8 {
        if (self.pos >= self.data.len) return error.EndOfBuffer;
        const b = self.data[self.pos];
        self.pos += 1;
        return b;
    }

    fn readBytes(self: *ThriftReader, n: usize) ![]const u8 {
        if (self.pos + n > self.data.len) return error.EndOfBuffer;
        const result = self.data[self.pos .. self.pos + n];
        self.pos += n;
        return result;
    }

    fn readVarint(self: *ThriftReader) !u64 {
        var result: u64 = 0;
        var shift: u6 = 0;
        while (true) {
            const b = try self.readByte();
            result |= @as(u64, b & 0x7F) << shift;
            if (b & 0x80 == 0) break;
            if (shift >= 63) return error.VarintTooLong;
            shift +|= 7;
        }
        return result;
    }

    fn readI32(self: *ThriftReader) !i32 {
        const n: u32 = @truncate(try self.readVarint());
        return @bitCast((n >> 1) ^ (-%@as(u32, n & 1)));
    }

    fn readI64(self: *ThriftReader) !i64 {
        const n: u64 = try self.readVarint();
        return @bitCast((n >> 1) ^ (-%@as(u64, n & 1)));
    }

    fn readI16(self: *ThriftReader) !i16 {
        const n: u16 = @truncate(try self.readVarint());
        return @bitCast((n >> 1) ^ (-%@as(u16, n & 1)));
    }

    fn readBinary(self: *ThriftReader) ![]const u8 {
        const len: usize = @intCast(try self.readVarint());
        return try self.readBytes(len);
    }

    const FieldHeader = struct {
        field_id: i16,
        type_id: u4,
    };

    fn readFieldHeader(self: *ThriftReader) !?FieldHeader {
        const byte = try self.readByte();
        if (byte == 0) return null; // STOP

        const type_id: u4 = @truncate(byte & 0x0F);
        const delta: i16 = @intCast((byte >> 4) & 0x0F);

        if (delta != 0) {
            self.last_field_id += delta;
        } else {
            self.last_field_id = try self.readI16();
        }

        return FieldHeader{ .field_id = self.last_field_id, .type_id = type_id };
    }

    fn pushStruct(self: *ThriftReader) i16 {
        const saved = self.last_field_id;
        self.last_field_id = 0;
        return saved;
    }

    fn popStruct(self: *ThriftReader, saved: i16) void {
        self.last_field_id = saved;
    }

    // Thrift compact type IDs
    const T_BOOL_TRUE = 1;
    const T_BOOL_FALSE = 2;
    const T_BYTE = 3;
    const T_I16 = 4;
    const T_I32 = 5;
    const T_I64 = 6;
    const T_DOUBLE = 7;
    const T_BINARY = 8;
    const T_LIST = 9;
    const T_SET = 10;
    const T_MAP = 11;
    const T_STRUCT = 12;

    fn skipValue(self: *ThriftReader, type_id: u4) !void {
        switch (type_id) {
            T_BOOL_TRUE, T_BOOL_FALSE => {},
            T_BYTE => self.pos += 1,
            T_I16, T_I32, T_I64 => _ = try self.readVarint(),
            T_DOUBLE => self.pos += 8,
            T_BINARY => {
                const len: usize = @intCast(try self.readVarint());
                self.pos += len;
            },
            T_LIST, T_SET => {
                const header = try self.readByte();
                const elem_type: u4 = @truncate(header & 0x0F);
                var size: usize = @intCast((header >> 4) & 0x0F);
                if (size == 0x0F) size = @intCast(try self.readVarint());
                for (0..size) |_| try self.skipValue(elem_type);
            },
            T_MAP => {
                const size: usize = @intCast(try self.readVarint());
                if (size > 0) {
                    const types = try self.readByte();
                    const key_type: u4 = @truncate((types >> 4) & 0x0F);
                    const val_type: u4 = @truncate(types & 0x0F);
                    for (0..size) |_| {
                        try self.skipValue(key_type);
                        try self.skipValue(val_type);
                    }
                }
            },
            T_STRUCT => {
                const saved = self.pushStruct();
                while (try self.readFieldHeader()) |fh| {
                    try self.skipValue(fh.type_id);
                }
                self.popStruct(saved);
            },
            else => return error.UnknownThriftType,
        }
    }
};

// ============================================================================
// Parquet metadata structures
// ============================================================================

const SchemaElement = struct {
    name: []u8,
    type_value: ?i32 = null,
    num_children: i32 = 0,
};

const ColumnChunkMeta = struct {
    type_value: i32 = 0,
    codec: i32 = 0,
    num_values: i64 = 0,
    total_uncompressed_size: i64 = 0,
    total_compressed_size: i64 = 0,
    data_page_offset: i64 = 0,
    dictionary_page_offset: ?i64 = null,
};

const RowGroupMeta = struct {
    columns: []ColumnChunkMeta,
    total_byte_size: i64 = 0,
    num_rows: i64 = 0,
};

const FileMetaData = struct {
    version: i32 = 0,
    schema: []SchemaElement,
    num_rows: i64 = 0,
    row_groups: []RowGroupMeta,
};

// Parquet enums
const CODEC_UNCOMPRESSED: i32 = 0;
const CODEC_ZSTD: i32 = 6;
const PAGE_DATA: i32 = 0;
const PAGE_DICTIONARY: i32 = 2;
const PAGE_DATA_V2: i32 = 3;
const ENC_PLAIN: i32 = 0;
const ENC_RLE: i32 = 3;
const ENC_RLE_DICTIONARY: i32 = 8;

// ============================================================================
// Metadata parsing
// ============================================================================

fn parseFileMetaData(allocator: Allocator, data: []const u8) !FileMetaData {
    var reader = ThriftReader.init(data);
    var meta = FileMetaData{
        .schema = &.{},
        .row_groups = &.{},
    };

    while (try reader.readFieldHeader()) |fh| {
        switch (fh.field_id) {
            1 => meta.version = try reader.readI32(), // version
            2 => { // schema: list<SchemaElement>
                const list_header = try reader.readByte();
                var size: usize = @intCast((list_header >> 4) & 0x0F);
                if (size == 0x0F) size = @intCast(try reader.readVarint());

                meta.schema = try allocator.alloc(SchemaElement, size);
                for (0..size) |i| {
                    meta.schema[i] = try parseSchemaElement(allocator, &reader);
                }
            },
            3 => meta.num_rows = try reader.readI64(), // num_rows
            4 => { // row_groups: list<RowGroup>
                const list_header = try reader.readByte();
                var size: usize = @intCast((list_header >> 4) & 0x0F);
                if (size == 0x0F) size = @intCast(try reader.readVarint());

                meta.row_groups = try allocator.alloc(RowGroupMeta, size);
                for (0..size) |i| {
                    meta.row_groups[i] = try parseRowGroup(allocator, &reader);
                }
            },
            else => try reader.skipValue(fh.type_id),
        }
    }

    return meta;
}

fn parseSchemaElement(allocator: Allocator, reader: *ThriftReader) !SchemaElement {
    var elem = SchemaElement{ .name = &.{} };
    const saved = reader.pushStruct();
    defer reader.popStruct(saved);

    while (try reader.readFieldHeader()) |fh| {
        switch (fh.field_id) {
            1 => elem.type_value = try reader.readI32(),
            4 => {
                const name_bytes = try reader.readBinary();
                elem.name = try allocator.dupe(u8, name_bytes);
            },
            5 => elem.num_children = try reader.readI32(),
            else => try reader.skipValue(fh.type_id),
        }
    }
    return elem;
}

fn parseRowGroup(allocator: Allocator, reader: *ThriftReader) !RowGroupMeta {
    var rg = RowGroupMeta{ .columns = &.{} };
    const saved = reader.pushStruct();
    defer reader.popStruct(saved);

    while (try reader.readFieldHeader()) |fh| {
        switch (fh.field_id) {
            1 => { // columns: list<ColumnChunk>
                const list_header = try reader.readByte();
                var size: usize = @intCast((list_header >> 4) & 0x0F);
                if (size == 0x0F) size = @intCast(try reader.readVarint());

                rg.columns = try allocator.alloc(ColumnChunkMeta, size);
                for (0..size) |i| {
                    rg.columns[i] = try parseColumnChunk(reader);
                }
            },
            2 => rg.total_byte_size = try reader.readI64(),
            3 => rg.num_rows = try reader.readI64(),
            else => try reader.skipValue(fh.type_id),
        }
    }
    return rg;
}

fn parseColumnChunk(reader: *ThriftReader) !ColumnChunkMeta {
    var col = ColumnChunkMeta{};
    const saved = reader.pushStruct();
    defer reader.popStruct(saved);

    while (try reader.readFieldHeader()) |fh| {
        switch (fh.field_id) {
            3 => { // meta_data: ColumnMetaData (inline struct)
                col = try parseColumnMetaData(reader);
            },
            else => try reader.skipValue(fh.type_id),
        }
    }
    return col;
}

fn parseColumnMetaData(reader: *ThriftReader) !ColumnChunkMeta {
    var col = ColumnChunkMeta{};
    const saved = reader.pushStruct();
    defer reader.popStruct(saved);

    while (try reader.readFieldHeader()) |fh| {
        switch (fh.field_id) {
            1 => col.type_value = try reader.readI32(), // type
            2 => { // encodings: list<Encoding>
                const list_header = try reader.readByte();
                var size: usize = @intCast((list_header >> 4) & 0x0F);
                if (size == 0x0F) size = @intCast(try reader.readVarint());
                for (0..size) |_| _ = try reader.readI32();
            },
            3 => { // path_in_schema: list<string>
                const list_header = try reader.readByte();
                var size: usize = @intCast((list_header >> 4) & 0x0F);
                if (size == 0x0F) size = @intCast(try reader.readVarint());
                for (0..size) |_| _ = try reader.readBinary();
            },
            4 => col.codec = try reader.readI32(), // compression codec
            5 => col.num_values = try reader.readI64(),
            6 => col.total_uncompressed_size = try reader.readI64(),
            7 => col.total_compressed_size = try reader.readI64(),
            9 => col.data_page_offset = try reader.readI64(),
            11 => col.dictionary_page_offset = try reader.readI64(),
            else => try reader.skipValue(fh.type_id),
        }
    }
    return col;
}

fn findColumnIndex(schema: []const SchemaElement, name: []const u8) ?usize {
    // Schema[0] is the root message; leaf columns start at index 1
    // Column index for row group columns = leaf_index (0-based)
    var leaf_idx: usize = 0;
    for (schema[1..]) |elem| {
        if (elem.num_children == 0) { // leaf column
            if (std.mem.eql(u8, elem.name, name)) return leaf_idx;
            leaf_idx += 1;
        }
    }
    return null;
}

// ============================================================================
// Page header parsing
// ============================================================================

const PageHeader = struct {
    page_type: i32 = 0,
    uncompressed_size: i32 = 0,
    compressed_size: i32 = 0,
    // DataPageHeader fields
    dp_num_values: i32 = 0,
    dp_encoding: i32 = 0,
    dp_def_encoding: i32 = 0,
    dp_rep_encoding: i32 = 0,
    // DictionaryPageHeader fields
    dict_num_values: i32 = 0,
    dict_encoding: i32 = 0,
    // DataPageHeaderV2 fields
    dpv2_num_values: i32 = 0,
    dpv2_num_nulls: i32 = 0,
    dpv2_num_rows: i32 = 0,
    dpv2_encoding: i32 = 0,
    dpv2_def_levels_byte_length: i32 = 0,
    dpv2_rep_levels_byte_length: i32 = 0,
    dpv2_is_compressed: bool = true,
};

fn parsePageHeader(reader: *ThriftReader) !PageHeader {
    var ph = PageHeader{};
    const saved = reader.pushStruct();
    defer reader.popStruct(saved);

    while (try reader.readFieldHeader()) |fh| {
        switch (fh.field_id) {
            1 => ph.page_type = try reader.readI32(),
            2 => ph.uncompressed_size = try reader.readI32(),
            3 => ph.compressed_size = try reader.readI32(),
            5 => { // DataPageHeader
                const s2 = reader.pushStruct();
                defer reader.popStruct(s2);
                while (try reader.readFieldHeader()) |fh2| {
                    switch (fh2.field_id) {
                        1 => ph.dp_num_values = try reader.readI32(),
                        2 => ph.dp_encoding = try reader.readI32(),
                        3 => ph.dp_def_encoding = try reader.readI32(),
                        4 => ph.dp_rep_encoding = try reader.readI32(),
                        else => try reader.skipValue(fh2.type_id),
                    }
                }
            },
            7 => { // DictionaryPageHeader
                const s2 = reader.pushStruct();
                defer reader.popStruct(s2);
                while (try reader.readFieldHeader()) |fh2| {
                    switch (fh2.field_id) {
                        1 => ph.dict_num_values = try reader.readI32(),
                        2 => ph.dict_encoding = try reader.readI32(),
                        else => try reader.skipValue(fh2.type_id),
                    }
                }
            },
            8 => { // DataPageHeaderV2
                const s2 = reader.pushStruct();
                defer reader.popStruct(s2);
                while (try reader.readFieldHeader()) |fh2| {
                    switch (fh2.field_id) {
                        1 => ph.dpv2_num_values = try reader.readI32(),
                        2 => ph.dpv2_num_nulls = try reader.readI32(),
                        3 => ph.dpv2_num_rows = try reader.readI32(),
                        4 => ph.dpv2_encoding = try reader.readI32(),
                        5 => ph.dpv2_def_levels_byte_length = try reader.readI32(),
                        6 => ph.dpv2_rep_levels_byte_length = try reader.readI32(),
                        7 => {
                            // is_compressed: bool
                            ph.dpv2_is_compressed = (fh2.type_id == ThriftReader.T_BOOL_TRUE);
                        },
                        else => try reader.skipValue(fh2.type_id),
                    }
                }
            },
            else => try reader.skipValue(fh.type_id),
        }
    }
    return ph;
}

// ============================================================================
// ZSTD decompression (via C libzstd for reliable one-shot decompression)
// ============================================================================

const c = @cImport({
    @cInclude("zstd.h");
});

fn decompressZstd(allocator: Allocator, compressed: []const u8, uncompressed_size: usize) ![]u8 {
    const output = try allocator.alloc(u8, uncompressed_size);
    errdefer allocator.free(output);

    const result = c.ZSTD_decompress(output.ptr, output.len, compressed.ptr, compressed.len);
    if (c.ZSTD_isError(result) != 0) {
        return error.ZstdDecompressError;
    }
    if (result != uncompressed_size) {
        return error.IncompleteDecompression;
    }
    return output;
}

// ============================================================================
// Page decoding: PLAIN and RLE_DICTIONARY for BYTE_ARRAY
// ============================================================================

/// Decode PLAIN-encoded BYTE_ARRAY values
fn decodePlainByteArray(allocator: Allocator, data: []const u8, num_values: usize) ![][]u8 {
    var texts = try allocator.alloc([]u8, num_values);
    var pos: usize = 0;
    var count: usize = 0;

    while (count < num_values and pos + 4 <= data.len) {
        const len: usize = std.mem.readInt(u32, data[pos..][0..4], .little);
        pos += 4;
        if (pos + len > data.len) break;
        texts[count] = try allocator.dupe(u8, data[pos..][0..len]);
        pos += len;
        count += 1;
    }

    if (count < num_values) {
        // Shrink
        const result = try allocator.realloc(texts, count);
        return result;
    }
    return texts;
}

/// Decode RLE/Bit-Packed Hybrid encoded integers (for dictionary indices)
fn decodeRleBitPacked(data: []const u8, bit_width: u8, num_values: usize, out: []u32) !usize {
    if (bit_width == 0) {
        // All values are 0
        @memset(out[0..@min(num_values, out.len)], 0);
        return @min(num_values, out.len);
    }

    var pos: usize = 0;
    var count: usize = 0;

    while (count < num_values and pos < data.len) {
        // Read header (varint)
        var header: u32 = 0;
        var shift: u5 = 0;
        while (pos < data.len) {
            const b = data[pos];
            pos += 1;
            header |= @as(u32, b & 0x7F) << shift;
            if (b & 0x80 == 0) break;
            shift +|= 7;
        }

        if (header & 1 == 1) {
            // Bit-packed run: (header >> 1) groups of 8 values
            const num_groups = header >> 1;
            const total_values = num_groups * 8;
            const bytes_needed = (total_values * bit_width + 7) / 8;

            if (pos + bytes_needed > data.len) break;

            var bit_pos: usize = 0;
            const mask: u32 = (@as(u32, 1) << @intCast(bit_width)) - 1;

            for (0..total_values) |_| {
                if (count >= num_values) break;
                const byte_idx = bit_pos / 8;
                const bit_offset: u5 = @intCast(bit_pos % 8);

                // Read up to 4 bytes to handle cross-byte values
                var raw: u32 = 0;
                const remaining_bytes = @min(data.len - pos - byte_idx, 4);
                for (0..remaining_bytes) |bi| {
                    raw |= @as(u32, data[pos + byte_idx + bi]) << @intCast(bi * 8);
                }

                out[count] = (raw >> bit_offset) & mask;
                count += 1;
                bit_pos += bit_width;
            }
            pos += bytes_needed;
        } else {
            // RLE run: (header >> 1) repeats of value
            const run_len = header >> 1;
            const value_bytes = (bit_width + 7) / 8;

            if (pos + value_bytes > data.len) break;

            var value: u32 = 0;
            for (0..value_bytes) |bi| {
                value |= @as(u32, data[pos + bi]) << @intCast(bi * 8);
            }
            pos += value_bytes;

            for (0..run_len) |_| {
                if (count >= num_values) break;
                out[count] = value;
                count += 1;
            }
        }
    }
    return count;
}

/// Skip definition levels in a data page (RLE-encoded, max_def_level=1)
/// Returns the data after the def levels and the count of non-null values
fn skipDefLevels(data: []const u8, num_values: usize) !struct { remaining: []const u8, non_null_count: usize } {
    if (data.len < 4) return error.DefLevelsTooShort;

    // First 4 bytes: length of encoded def levels
    const def_len: usize = std.mem.readInt(u32, data[0..4], .little);
    if (4 + def_len > data.len) return error.DefLevelsOverflow;

    // Decode def levels to count non-nulls
    // For max_def_level=1, bit_width=1
    var indices_buf: [8192]u32 = undefined;
    const decoded = try decodeRleBitPacked(data[4 .. 4 + def_len], 1, num_values, &indices_buf);

    var non_null: usize = 0;
    for (indices_buf[0..decoded]) |v| {
        if (v != 0) non_null += 1;
    }

    return .{ .remaining = data[4 + def_len ..], .non_null_count = non_null };
}

// ============================================================================
// Column chunk decoder
// ============================================================================

fn decodeColumnChunk(allocator: Allocator, chunk_data: []const u8, col: ColumnChunkMeta, max_docs: u32) ![][]u8 {
    var all_texts = ManagedArrayList([]u8).init(allocator);
    errdefer {
        for (all_texts.items) |t| allocator.free(t);
        all_texts.deinit();
    }

    // Dictionary (populated if DICTIONARY_PAGE encountered)
    var dictionary: ?[][]u8 = null;
    var dict_count: usize = 0;
    defer {
        if (dictionary) |dict| {
            for (dict[0..dict_count]) |d| allocator.free(d);
            allocator.free(dict);
        }
    }

    var pos: usize = 0;
    var values_read: i64 = 0;

    while (pos < chunk_data.len and values_read < col.num_values) {
        if (max_docs > 0 and all_texts.items.len >= max_docs) break;

        // Parse page header
        var header_reader = ThriftReader.init(chunk_data[pos..]);
        const page_header = parsePageHeader(&header_reader) catch break;
        pos += header_reader.pos;

        const compressed_size: usize = @intCast(page_header.compressed_size);
        if (pos + compressed_size > chunk_data.len) break;

        const page_data = chunk_data[pos .. pos + compressed_size];
        pos += compressed_size;

        // Decompress if needed
        const uncompressed_size: usize = @intCast(page_header.uncompressed_size);
        const decoded_data = if (col.codec == CODEC_ZSTD and compressed_size != uncompressed_size)
            try decompressZstd(allocator, page_data, uncompressed_size)
        else
            try allocator.dupe(u8, page_data);
        defer allocator.free(decoded_data);

        if (page_header.page_type == PAGE_DICTIONARY) {
            // Dictionary page: PLAIN-encoded byte arrays
            const num_dict_values: usize = @intCast(page_header.dict_num_values);
            dictionary = try decodePlainByteArray(allocator, decoded_data, num_dict_values);
            dict_count = if (dictionary) |d| d.len else 0;
        } else if (page_header.page_type == PAGE_DATA) {
            const num_values: usize = @intCast(page_header.dp_num_values);

            if (page_header.dp_encoding == ENC_RLE_DICTIONARY) {
                // RLE_DICTIONARY: skip def levels, decode indices, look up dictionary
                const dict = dictionary orelse return error.NoDictionary;

                // Skip definition levels (RLE-encoded for nullable column)
                const after_def = try skipDefLevels(decoded_data, num_values);
                const values_data = after_def.remaining;

                if (values_data.len < 1) continue;
                const bit_width = values_data[0];
                const indices_data = values_data[1..];

                var indices_buf = try allocator.alloc(u32, after_def.non_null_count);
                defer allocator.free(indices_buf);

                const decoded_count = try decodeRleBitPacked(indices_data, bit_width, after_def.non_null_count, indices_buf);

                for (indices_buf[0..decoded_count]) |idx| {
                    if (max_docs > 0 and all_texts.items.len >= max_docs) break;
                    if (idx < dict_count) {
                        try all_texts.append(try allocator.dupe(u8, dict[idx]));
                    }
                }
                values_read += @intCast(num_values);
            } else if (page_header.dp_encoding == ENC_PLAIN) {
                // PLAIN: skip def levels, then plain byte arrays
                const after_def = try skipDefLevels(decoded_data, num_values);

                const texts = try decodePlainByteArray(allocator, after_def.remaining, after_def.non_null_count);
                defer allocator.free(texts);

                for (texts) |text| {
                    if (max_docs > 0 and all_texts.items.len >= max_docs) {
                        allocator.free(text);
                        continue;
                    }
                    try all_texts.append(text);
                }
                values_read += @intCast(num_values);
            } else {
                // Unknown encoding, skip
                values_read += @intCast(num_values);
            }
        } else if (page_header.page_type == PAGE_DATA_V2) {
            const num_values: usize = @intCast(page_header.dpv2_num_values);
            const num_nulls: usize = @intCast(page_header.dpv2_num_nulls);
            const non_null = num_values - num_nulls;
            const def_len: usize = @intCast(page_header.dpv2_def_levels_byte_length);
            const rep_len: usize = @intCast(page_header.dpv2_rep_levels_byte_length);

            // Skip rep + def level data
            const values_start = rep_len + def_len;
            if (values_start >= decoded_data.len) continue;
            const values_data = decoded_data[values_start..];

            if (page_header.dpv2_encoding == ENC_RLE_DICTIONARY) {
                const dict = dictionary orelse return error.NoDictionary;
                if (values_data.len < 1) continue;
                const bit_width = values_data[0];
                const indices_data = values_data[1..];

                var indices_buf = try allocator.alloc(u32, non_null);
                defer allocator.free(indices_buf);

                const decoded_count = try decodeRleBitPacked(indices_data, bit_width, non_null, indices_buf);
                for (indices_buf[0..decoded_count]) |idx| {
                    if (max_docs > 0 and all_texts.items.len >= max_docs) break;
                    if (idx < dict_count) {
                        try all_texts.append(try allocator.dupe(u8, dict[idx]));
                    }
                }
            } else if (page_header.dpv2_encoding == ENC_PLAIN) {
                const texts = try decodePlainByteArray(allocator, values_data, non_null);
                defer allocator.free(texts);
                for (texts) |text| {
                    if (max_docs > 0 and all_texts.items.len >= max_docs) {
                        allocator.free(text);
                        continue;
                    }
                    try all_texts.append(text);
                }
            }
            values_read += @intCast(num_values);
        }
    }

    return try all_texts.toOwnedSlice();
}

// ============================================================================
// Tests
// ============================================================================

test "thrift reader varint" {
    const data = [_]u8{ 0x96, 0x01 }; // 150 in varint
    var reader = ThriftReader.init(&data);
    const v = try reader.readVarint();
    try std.testing.expectEqual(@as(u64, 150), v);
}

test "thrift reader zigzag i32" {
    // zigzag(0)=0, zigzag(-1)=1, zigzag(1)=2, zigzag(-2)=3
    const data = [_]u8{0}; // zigzag 0 = 0
    var reader = ThriftReader.init(&data);
    const v = try reader.readI32();
    try std.testing.expectEqual(@as(i32, 0), v);
}

test "rle decode all ones" {
    // RLE run: header=(1000 << 1) | 0 = 2000, value=1
    // 2000 in varint: 0xD0 0x0F
    const data = [_]u8{ 0xD0, 0x0F, 0x01 };
    var out: [1000]u32 = undefined;
    const count = try decodeRleBitPacked(&data, 1, 1000, &out);
    try std.testing.expectEqual(@as(usize, 1000), count);
    for (out[0..count]) |v| {
        try std.testing.expectEqual(@as(u32, 1), v);
    }
}

test "plain byte array decode" {
    // Two strings: "hello" and "world"
    var buf: [18]u8 = undefined;
    std.mem.writeInt(u32, buf[0..4], 5, .little);
    @memcpy(buf[4..9], "hello");
    std.mem.writeInt(u32, buf[9..13], 5, .little);
    @memcpy(buf[13..18], "world");

    const texts = try decodePlainByteArray(std.testing.allocator, &buf, 2);
    defer {
        for (texts) |t| std.testing.allocator.free(t);
        std.testing.allocator.free(texts);
    }

    try std.testing.expectEqual(@as(usize, 2), texts.len);
    try std.testing.expectEqualStrings("hello", texts[0]);
    try std.testing.expectEqualStrings("world", texts[1]);
}
