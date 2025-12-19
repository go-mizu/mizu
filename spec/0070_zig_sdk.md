# RFC 0070: Zig SDK Generator

## Summary

Add Zig SDK code generation to the Mizu contract system, enabling production-ready, type-safe, and memory-efficient Zig clients with explicit allocator control, comptime validation, and zero hidden allocations.

## Motivation

Zig has emerged as a powerful systems programming language for performance-critical applications:

1. **Explicit memory management**: User-controlled allocators with no hidden allocations
2. **Compile-time execution**: Comptime for zero-overhead metaprogramming and validation
3. **C interoperability**: Direct C ABI compatibility without FFI overhead
4. **Cross-compilation**: Built-in cross-compilation to any target from any host
5. **Safety without GC**: Memory safety through design, not garbage collection
6. **WebAssembly**: First-class WASM target support
7. **Embedded systems**: No libc dependency, suitable for bare-metal targets
8. **Modern tooling**: Built-in build system, test framework, and package manager

## Design Goals

### Developer Experience (DX)

- **Idiomatic Zig**: Follow Zig conventions (snake_case, error unions, optionals)
- **Allocator-aware**: All allocations use explicit allocator parameters
- **Comptime validation**: Compile-time type checking and validation
- **Error handling**: Error unions with descriptive error sets
- **Optional types**: Native `?T` for nullable values
- **Zero allocations in hot paths**: Pre-allocated buffers where possible
- **Async support**: Integration with Zig's async I/O (io_uring, epoll)
- **Documentation**: Comprehensive doc comments for all public APIs

### Production Readiness

- **Retry logic**: Configurable retry with exponential backoff
- **Timeout handling**: Per-request and global timeout configuration
- **Connection pooling**: HTTP keep-alive connection reuse
- **Streaming support**: Lazy iteration for SSE with backpressure
- **Memory efficiency**: Minimal allocations, arena-friendly design
- **Testing support**: Built-in test utilities and mocking
- **Cross-platform**: Windows, Linux, macOS, WASM targets
- **No libc option**: Can build without libc for embedded targets

## Architecture

### Package Structure

```
{package}/
├── build.zig                  # Build system configuration
├── build.zig.zon              # Package manifest (dependencies)
├── src/
│   ├── root.zig               # Package root, re-exports
│   ├── client.zig             # Client and configuration
│   ├── types.zig              # Generated type definitions
│   ├── resources.zig          # Resource modules with methods
│   ├── streaming.zig          # SSE streaming support
│   └── errors.zig             # Error types and handling
└── tests/
    └── main_test.zig          # Integration tests
```

### Core Components

#### 1. Error Handling (`errors.zig`)

Zig uses error sets and error unions for robust error handling:

```zig
//! Error types for the {ServiceName} SDK.

const std = @import("std");

/// Errors that can occur when using the SDK.
pub const Error = error{
    /// HTTP request failed.
    HttpError,

    /// Request timed out.
    Timeout,

    /// Connection failed.
    ConnectionFailed,

    /// Request was cancelled.
    Cancelled,

    /// Failed to serialize request body.
    SerializationError,

    /// Failed to deserialize response body.
    DeserializationError,

    /// Streaming error.
    StreamError,

    /// Invalid configuration.
    InvalidConfig,

    /// Out of memory.
    OutOfMemory,

    /// End of stream.
    EndOfStream,
};

/// HTTP error details.
pub const HttpErrorInfo = struct {
    /// HTTP status code.
    status: u16,
    /// Response body, if available.
    body: ?[]const u8,
    /// Allocator used for body (if allocated).
    allocator: ?std.mem.Allocator,

    pub fn deinit(self: *HttpErrorInfo) void {
        if (self.body) |body| {
            if (self.allocator) |alloc| {
                alloc.free(body);
            }
        }
        self.* = undefined;
    }

    /// Returns true if this error is potentially retriable.
    pub fn isRetriable(self: HttpErrorInfo) bool {
        return self.status >= 500 or self.status == 429;
    }
};

/// Format an error for display.
pub fn format(err: Error) []const u8 {
    return switch (err) {
        error.HttpError => "HTTP error",
        error.Timeout => "request timed out",
        error.ConnectionFailed => "connection failed",
        error.Cancelled => "request cancelled",
        error.SerializationError => "serialization error",
        error.DeserializationError => "deserialization error",
        error.StreamError => "streaming error",
        error.InvalidConfig => "invalid configuration",
        error.OutOfMemory => "out of memory",
        error.EndOfStream => "end of stream",
    };
}
```

#### 2. Client (`client.zig`)

The client with allocator-aware design:

```zig
//! HTTP client for the {ServiceName} API.

const std = @import("std");
const http = std.http;
const mem = std.mem;
const Allocator = mem.Allocator;

const errors = @import("errors.zig");
const resources = @import("resources.zig");

/// Authentication mode for API requests.
pub const AuthMode = enum {
    /// Bearer token authentication.
    bearer,
    /// Basic authentication.
    basic,
    /// API key in header.
    api_key,
    /// No authentication.
    none,
};

/// Configuration for the SDK client.
pub const ClientConfig = struct {
    /// API key for authentication.
    api_key: ?[]const u8 = null,
    /// Base URL for API requests.
    base_url: []const u8 = "{defaults.base_url}",
    /// Authentication mode.
    auth_mode: AuthMode = .bearer,
    /// Request timeout in milliseconds.
    timeout_ms: u64 = 60_000,
    /// Maximum retry attempts.
    max_retries: u32 = 2,
    /// User-Agent header value.
    user_agent: []const u8 = "{ServiceName} Zig SDK/0.1.0",
};

/// Client for the {ServiceName} API.
pub const Client = struct {
    allocator: Allocator,
    config: ClientConfig,
    http_client: http.Client,

    const Self = @This();

    /// Creates a new client.
    pub fn init(allocator: Allocator, config: ClientConfig) !Self {
        var http_client = http.Client{ .allocator = allocator };

        return Self{
            .allocator = allocator,
            .config = config,
            .http_client = http_client,
        };
    }

    /// Releases all resources associated with the client.
    pub fn deinit(self: *Self) void {
        self.http_client.deinit();
        self.* = undefined;
    }

    /// Returns the configured base URL.
    pub fn baseUrl(self: Self) []const u8 {
        return self.config.base_url;
    }

    /// Makes an HTTP request and returns the response.
    pub fn request(
        self: *Self,
        comptime method: http.Method,
        path: []const u8,
        body: ?[]const u8,
    ) !Response {
        const uri_string = try std.fmt.allocPrint(
            self.allocator,
            "{s}{s}",
            .{ self.config.base_url, path },
        );
        defer self.allocator.free(uri_string);

        const uri = try std.Uri.parse(uri_string);

        var headers = http.Headers{ .allocator = self.allocator };
        defer headers.deinit();

        try headers.append("Content-Type", "application/json");
        try headers.append("Accept", "application/json");
        try headers.append("User-Agent", self.config.user_agent);

        // Add authentication header
        if (self.config.api_key) |key| {
            const auth_value = switch (self.config.auth_mode) {
                .bearer => try std.fmt.allocPrint(self.allocator, "Bearer {s}", .{key}),
                .basic => try std.fmt.allocPrint(self.allocator, "Basic {s}", .{key}),
                .api_key => key,
                .none => null,
            };
            if (auth_value) |val| {
                defer if (self.config.auth_mode != .api_key) self.allocator.free(val);
                const header_name = if (self.config.auth_mode == .api_key) "x-api-key" else "authorization";
                try headers.append(header_name, val);
            }
        }

        // Add default headers from contract
        {{range .Defaults.Headers}}
        try headers.append("{{.K}}", "{{.V}}");
        {{end}}

        var req = try self.http_client.request(method, uri, headers, .{});
        defer req.deinit();

        if (body) |b| {
            req.transfer_encoding = .{ .content_length = b.len };
            try req.writer().writeAll(b);
        }

        try req.finish();
        try req.wait();

        const status = req.status;
        if (@intFromEnum(status) >= 400) {
            const response_body = try req.reader().readAllAlloc(self.allocator, 1024 * 1024);
            return error.HttpError;
        }

        return Response{
            .allocator = self.allocator,
            .body = try req.reader().readAllAlloc(self.allocator, 1024 * 1024),
        };
    }

    // Resource accessors are generated below
    {{range .Resources}}
    /// Access the {{.Name}} resource.
    pub fn {{.ZigName}}(self: *Self) resources.{{.StructName}} {
        return resources.{{.StructName}}.init(self);
    }
    {{end}}
};

/// HTTP response wrapper.
pub const Response = struct {
    allocator: Allocator,
    body: []const u8,

    pub fn deinit(self: *Response) void {
        self.allocator.free(self.body);
        self.* = undefined;
    }
};
```

#### 3. Types (`types.zig`)

Generated type definitions with JSON serialization:

```zig
//! Type definitions for the {ServiceName} API.

const std = @import("std");
const json = std.json;
const mem = std.mem;
const Allocator = mem.Allocator;

// --- Struct Types ---

{{range .Types}}
{{if eq .Kind "struct"}}
/// {{.Description}}
pub const {{.ZigName}} = struct {
    {{range .Fields}}
    {{if .Description}}/// {{.Description}}{{end}}
    {{.ZigName}}: {{.ZigType}},
    {{end}}

    const Self = @This();

    /// Parses from JSON.
    pub fn fromJson(allocator: Allocator, input: []const u8) !Self {
        return try json.parseFromSlice(Self, allocator, input, .{});
    }

    /// Parses from a parsed JSON value.
    pub fn fromJsonValue(value: json.Value) !Self {
        return try json.parseFromValue(Self, std.heap.page_allocator, value, .{});
    }

    /// Serializes to JSON.
    pub fn toJson(self: Self, allocator: Allocator) ![]const u8 {
        return try json.stringifyAlloc(allocator, self, .{});
    }

    /// Frees any allocated memory.
    pub fn deinit(self: *Self, allocator: Allocator) void {
        json.parseFree(Self, allocator, self.*);
    }
};

{{end}}
{{end}}

// --- Enum Types ---

{{range .Types}}
{{if .HasEnum}}
/// {{.Description}}
pub const {{.ZigName}} = enum {
    {{range .Enum}}
    {{.ZigName}},
    {{end}}

    const Self = @This();

    /// Returns the string representation.
    pub fn toString(self: Self) []const u8 {
        return switch (self) {
            {{range .Enum}}
            .{{.ZigName}} => "{{.Value}}",
            {{end}}
        };
    }

    /// Parses from string.
    pub fn fromString(s: []const u8) ?Self {
        {{range .Enum}}
        if (mem.eql(u8, s, "{{.Value}}")) return .{{.ZigName}};
        {{end}}
        return null;
    }

    pub fn jsonStringify(self: Self, opts: json.StringifyOptions, writer: anytype) !void {
        try writer.print("\"{s}\"", .{self.toString()});
        _ = opts;
    }
};
{{end}}
{{end}}

// --- Union Types ---

{{range .Types}}
{{if eq .Kind "union"}}
/// {{.Description}}
pub const {{.ZigName}} = union(enum) {
    {{range .Variants}}
    {{.ZigName}}: {{.ZigType}},
    {{end}}

    const Self = @This();

    /// Returns the discriminator tag value.
    pub fn tagValue(self: Self) []const u8 {
        return switch (self) {
            {{range .Variants}}
            .{{.ZigName}} => "{{.Value}}",
            {{end}}
        };
    }

    {{range .Variants}}
    /// Returns true if this is the {{.ZigName}} variant.
    pub fn is{{.PascalName}}(self: Self) bool {
        return self == .{{.ZigName}};
    }

    /// Returns the {{.ZigName}} value if this is that variant.
    pub fn as{{.PascalName}}(self: Self) ?{{.ZigType}} {
        return switch (self) {
            .{{.ZigName}} => |v| v,
            else => null,
        };
    }
    {{end}}
};
{{end}}
{{end}}
```

#### 4. Resources (`resources.zig`)

Resource operations with async methods:

```zig
//! Resource operations for the {ServiceName} API.

const std = @import("std");
const json = std.json;
const Allocator = std.mem.Allocator;

const Client = @import("client.zig").Client;
const errors = @import("errors.zig");
const types = @import("types.zig");
{{if .HasSSE}}
const streaming = @import("streaming.zig");
{{end}}

{{range .Resources}}
/// Operations for the {{.Name}} resource.
///
/// {{.Description}}
pub const {{.StructName}} = struct {
    client: *Client,

    const Self = @This();

    /// Creates a new resource accessor.
    pub fn init(client: *Client) Self {
        return Self{ .client = client };
    }

    {{range .Methods}}
    /// {{.Description}}
    {{if .IsStreaming}}
    pub fn {{.ZigName}}(self: Self, request: *const types.{{.InputType}}) !streaming.EventStream(types.{{.StreamItemType}}) {
        const body = try request.toJson(self.client.allocator);
        defer self.client.allocator.free(body);

        return streaming.EventStream(types.{{.StreamItemType}}).init(
            self.client,
            "{{.HTTPPath}}",
            body,
        );
    }
    {{else}}
    pub fn {{.ZigName}}(self: Self, request: *const types.{{.InputType}}) !types.{{.OutputType}} {
        const body = try request.toJson(self.client.allocator);
        defer self.client.allocator.free(body);

        var response = try self.client.request(.{{.HTTPMethod | lower}}, "{{.HTTPPath}}", body);
        defer response.deinit();

        return try types.{{.OutputType}}.fromJson(self.client.allocator, response.body);
    }
    {{end}}
    {{end}}
};
{{end}}
```

#### 5. Streaming (`streaming.zig`)

SSE streaming with lazy iteration:

```zig
//! Server-Sent Events (SSE) streaming support.

const std = @import("std");
const http = std.http;
const json = std.json;
const mem = std.mem;
const Allocator = mem.Allocator;

const Client = @import("client.zig").Client;
const errors = @import("errors.zig");

/// An SSE event.
pub const SseEvent = struct {
    /// Event type.
    event: ?[]const u8 = null,
    /// Event data.
    data: ?[]const u8 = null,
    /// Event ID.
    id: ?[]const u8 = null,
    /// Retry interval in milliseconds.
    retry: ?u64 = null,

    allocator: ?Allocator = null,

    pub fn deinit(self: *SseEvent) void {
        if (self.allocator) |alloc| {
            if (self.event) |e| alloc.free(e);
            if (self.data) |d| alloc.free(d);
            if (self.id) |i| alloc.free(i);
        }
        self.* = undefined;
    }
};

/// SSE parser.
pub const SseParser = struct {
    buffer: std.ArrayList(u8),
    allocator: Allocator,

    const Self = @This();

    pub fn init(allocator: Allocator) Self {
        return Self{
            .buffer = std.ArrayList(u8).init(allocator),
            .allocator = allocator,
        };
    }

    pub fn deinit(self: *Self) void {
        self.buffer.deinit();
        self.* = undefined;
    }

    /// Feed data to the parser and return complete events.
    pub fn feed(self: *Self, data: []const u8) !?SseEvent {
        try self.buffer.appendSlice(data);

        // Look for complete event (double newline)
        const buf = self.buffer.items;
        if (mem.indexOf(u8, buf, "\n\n")) |pos| {
            const event_data = buf[0..pos];

            // Remove processed data from buffer
            const remaining = buf[pos + 2 ..];
            mem.copyForwards(u8, buf[0..remaining.len], remaining);
            self.buffer.shrinkRetainingCapacity(remaining.len);

            return try self.parseEvent(event_data);
        }

        return null;
    }

    fn parseEvent(self: *Self, data: []const u8) !SseEvent {
        var event = SseEvent{ .allocator = self.allocator };
        var data_lines = std.ArrayList(u8).init(self.allocator);
        defer data_lines.deinit();

        var lines = mem.splitScalar(u8, data, '\n');
        while (lines.next()) |line| {
            if (line.len == 0 or line[0] == ':') continue;

            const colon_pos = mem.indexOfScalar(u8, line, ':') orelse {
                continue;
            };

            const field = line[0..colon_pos];
            var value = line[colon_pos + 1 ..];
            if (value.len > 0 and value[0] == ' ') {
                value = value[1..];
            }

            if (mem.eql(u8, field, "event")) {
                event.event = try self.allocator.dupe(u8, value);
            } else if (mem.eql(u8, field, "data")) {
                if (data_lines.items.len > 0) {
                    try data_lines.append('\n');
                }
                try data_lines.appendSlice(value);
            } else if (mem.eql(u8, field, "id")) {
                event.id = try self.allocator.dupe(u8, value);
            } else if (mem.eql(u8, field, "retry")) {
                event.retry = std.fmt.parseInt(u64, value, 10) catch null;
            }
        }

        if (data_lines.items.len > 0) {
            event.data = try self.allocator.dupe(u8, data_lines.items);
        }

        return event;
    }
};

/// A stream of typed events from an SSE endpoint.
pub fn EventStream(comptime T: type) type {
    return struct {
        client: *Client,
        path: []const u8,
        request_body: []const u8,
        parser: SseParser,
        started: bool = false,
        response: ?http.Client.Response = null,

        const Self = @This();

        pub fn init(client: *Client, path: []const u8, request_body: []const u8) Self {
            return Self{
                .client = client,
                .path = path,
                .request_body = request_body,
                .parser = SseParser.init(client.allocator),
            };
        }

        pub fn deinit(self: *Self) void {
            self.parser.deinit();
            if (self.response) |*resp| {
                resp.deinit();
            }
            self.* = undefined;
        }

        /// Returns the next event in the stream.
        pub fn next(self: *Self) !?T {
            if (!self.started) {
                try self.start();
            }

            while (true) {
                // Try to get a buffered event first
                if (try self.parser.feed("")) |sse_event| {
                    var event = sse_event;
                    defer event.deinit();

                    if (event.data) |data| {
                        // Skip [DONE] marker
                        if (mem.eql(u8, data, "[DONE]")) {
                            return null;
                        }

                        return try json.parseFromSlice(T, self.client.allocator, data, .{});
                    }
                }

                // Read more data from the stream
                if (self.response) |*resp| {
                    var buf: [4096]u8 = undefined;
                    const n = try resp.reader().read(&buf);
                    if (n == 0) {
                        return null; // End of stream
                    }

                    if (try self.parser.feed(buf[0..n])) |sse_event| {
                        var event = sse_event;
                        defer event.deinit();

                        if (event.data) |data| {
                            if (mem.eql(u8, data, "[DONE]")) {
                                return null;
                            }
                            return try json.parseFromSlice(T, self.client.allocator, data, .{});
                        }
                    }
                } else {
                    return null;
                }
            }
        }

        fn start(self: *Self) !void {
            // Initialize HTTP request for streaming
            const uri_string = try std.fmt.allocPrint(
                self.client.allocator,
                "{s}{s}",
                .{ self.client.config.base_url, self.path },
            );
            defer self.client.allocator.free(uri_string);

            const uri = try std.Uri.parse(uri_string);

            var headers = http.Headers{ .allocator = self.client.allocator };
            try headers.append("Content-Type", "application/json");
            try headers.append("Accept", "text/event-stream");
            try headers.append("Cache-Control", "no-cache");

            // Add auth headers...
            if (self.client.config.api_key) |key| {
                switch (self.client.config.auth_mode) {
                    .bearer => {
                        const val = try std.fmt.allocPrint(self.client.allocator, "Bearer {s}", .{key});
                        defer self.client.allocator.free(val);
                        try headers.append("authorization", val);
                    },
                    .api_key => try headers.append("x-api-key", key),
                    else => {},
                }
            }

            self.response = try self.client.http_client.request(.POST, uri, headers, .{});

            if (self.request_body.len > 0) {
                self.response.?.transfer_encoding = .{ .content_length = self.request_body.len };
                try self.response.?.writer().writeAll(self.request_body);
            }

            try self.response.?.finish();
            try self.response.?.wait();

            self.started = true;
        }
    };
}
```

## Type Mapping

### Primitive Types

| Contract Type     | Zig Type              | Notes                           |
|-------------------|----------------------|--------------------------------|
| `string`          | `[]const u8`         | Slice of bytes                 |
| `bool`, `boolean` | `bool`               |                                |
| `int`             | `i32`                |                                |
| `int8`            | `i8`                 |                                |
| `int16`           | `i16`                |                                |
| `int32`           | `i32`                |                                |
| `int64`           | `i64`                |                                |
| `uint`            | `u32`                |                                |
| `uint8`           | `u8`                 |                                |
| `uint16`          | `u16`                |                                |
| `uint32`          | `u32`                |                                |
| `uint64`          | `u64`                |                                |
| `float32`         | `f32`                |                                |
| `float64`         | `f64`                |                                |
| `time.Time`       | `i64`                | Unix timestamp (ms)            |
| `json.RawMessage` | `json.Value`         | Dynamic JSON                   |
| `any`             | `json.Value`         | Dynamic JSON                   |

### Collection Types

| Contract Type      | Zig Type                           |
|--------------------|-------------------------------------|
| `[]T`              | `[]const ZigType`                   |
| `map[string]T`     | `std.StringHashMap(ZigType)`        |

### Optional/Nullable

| Contract Pattern   | Zig Type               |
|--------------------|------------------------|
| Optional field     | `?T`                   |
| Nullable type      | `?T`                   |

### Struct Fields

Fields use Zig's JSON parsing:

```zig
pub const CreateMessageRequest = struct {
    /// The model to use for generation.
    model: []const u8,

    /// The messages in the conversation.
    messages: []const Message,

    /// Maximum tokens to generate.
    max_tokens: i32,

    /// Temperature for sampling.
    temperature: ?f64 = null,

    /// Whether to stream the response.
    stream: ?bool = null,
};
```

### Enum/Const Values

Zig enums with string conversion:

```zig
pub const Role = enum {
    user,
    assistant,
    system,

    pub fn toString(self: Role) []const u8 {
        return switch (self) {
            .user => "user",
            .assistant => "assistant",
            .system => "system",
        };
    }

    pub fn fromString(s: []const u8) ?Role {
        if (std.mem.eql(u8, s, "user")) return .user;
        if (std.mem.eql(u8, s, "assistant")) return .assistant;
        if (std.mem.eql(u8, s, "system")) return .system;
        return null;
    }
};
```

### Discriminated Unions

Tagged unions with variant accessors:

```zig
pub const ContentBlock = union(enum) {
    text: TextBlock,
    image: ImageBlock,
    tool_use: ToolUseBlock,

    pub fn tagValue(self: ContentBlock) []const u8 {
        return switch (self) {
            .text => "text",
            .image => "image",
            .tool_use => "tool_use",
        };
    }

    pub fn isText(self: ContentBlock) bool {
        return self == .text;
    }

    pub fn asText(self: ContentBlock) ?TextBlock {
        return switch (self) {
            .text => |v| v,
            else => null,
        };
    }
};
```

## HTTP Client Implementation

### Request Flow

```zig
pub fn create(self: Self, request: *const CreateMessageRequest) !Message {
    const body = try request.toJson(self.client.allocator);
    defer self.client.allocator.free(body);

    var last_error: ?errors.Error = null;

    var attempt: u32 = 0;
    while (attempt <= self.client.config.max_retries) : (attempt += 1) {
        if (attempt > 0) {
            // Exponential backoff with jitter
            const delay_ms: u64 = (@as(u64, 500) << @intCast(attempt - 1)) +
                @mod(std.crypto.random.int(u64), 100);
            std.time.sleep(delay_ms * std.time.ns_per_ms);
        }

        const result = self.doRequest("/v1/messages", body);
        if (result) |response| {
            defer response.deinit();
            return try Message.fromJson(self.client.allocator, response.body);
        } else |err| {
            if (err == error.HttpError and self.isRetriable()) {
                last_error = err;
                continue;
            }
            return err;
        }
    }

    return last_error orelse error.ConnectionFailed;
}
```

### SSE Streaming Implementation

```zig
pub fn stream(self: Self, request: *const CreateMessageRequest) !EventStream(MessageStreamEvent) {
    const body = try request.toJson(self.client.allocator);

    return EventStream(MessageStreamEvent).init(
        self.client,
        "/v1/messages",
        body,
    );
}

// Usage:
var stream = try client.messages().stream(&request);
defer stream.deinit();

while (try stream.next()) |event| {
    switch (event) {
        .content_block_delta => |delta| {
            if (delta.delta.text) |text| {
                std.debug.print("{s}", .{text});
            }
        },
        .message_stop => break,
        else => {},
    }
}
```

## Configuration

### Default Values

From contract `Defaults`:

```zig
pub const ClientConfig = struct {
    api_key: ?[]const u8 = null,
    base_url: []const u8 = "{defaults.base_url}",
    auth_mode: AuthMode = .bearer,
    timeout_ms: u64 = 60_000,
    max_retries: u32 = 2,
    user_agent: []const u8 = "{ServiceName} Zig SDK/0.1.0",
};
```

### Environment Variables

The SDK does NOT automatically read environment variables. Users should handle this explicitly:

```zig
const std = @import("std");
const sdk = @import("anthropic");

pub fn main() !void {
    var gpa = std.heap.GeneralPurposeAllocator(.{}){};
    defer _ = gpa.deinit();
    const allocator = gpa.allocator();

    const api_key = std.posix.getenv("ANTHROPIC_API_KEY") orelse
        return error.MissingApiKey;

    var client = try sdk.Client.init(allocator, .{
        .api_key = api_key,
    });
    defer client.deinit();
}
```

## Naming Conventions

### Zig Naming

| Contract       | Zig                        |
|----------------|----------------------------|
| `user-id`      | `user_id`                  |
| `user_name`    | `user_name`                |
| `UserData`     | `UserData`                 |
| `create`       | `create`                   |
| `get-user`     | `get_user`                 |
| `getMessage`   | `getMessage`               |

Functions:
- `toSnakeCase(s)`: Converts to snake_case for fields
- `toPascalCase(s)`: Converts to PascalCase for types
- `toScreamingSnake(s)`: Converts to SCREAMING_SNAKE_CASE for constants
- `sanitizeIdent(s)`: Removes invalid characters

Reserved words: Zig keywords are prefixed with `@`:
- `type` → `@"type"`
- `error` → `@"error"`
- `async` → `@"async"`
- `await` → `@"await"`
- `break` → `@"break"`
- etc.

## Code Generation

### Generator Structure

```go
package sdkzig

type Config struct {
    // Package is the Zig package name.
    // Default: sanitized snake_case service name.
    Package string

    // Version is the package version.
    // Default: "0.1.0".
    Version string

    // MinZigVersion is the minimum Zig version required.
    // Default: "0.11.0".
    MinZigVersion string
}

func Generate(svc *contract.Service, cfg *Config) ([]*sdk.File, error)
```

### Template Files

```
templates/
├── build.zig.tmpl             # Build system
├── build.zig.zon.tmpl         # Package manifest
├── root.zig.tmpl              # Package root
├── client.zig.tmpl            # Client implementation
├── types.zig.tmpl             # Type definitions
├── resources.zig.tmpl         # Resource operations
├── streaming.zig.tmpl         # SSE streaming
└── errors.zig.tmpl            # Error types
```

### Generated Files

| File                     | Purpose                           |
|--------------------------|-----------------------------------|
| `build.zig`              | Build system configuration        |
| `build.zig.zon`          | Package manifest                  |
| `src/root.zig`           | Package root with re-exports      |
| `src/client.zig`         | Client and configuration          |
| `src/types.zig`          | Type definitions                  |
| `src/resources.zig`      | Resource operations               |
| `src/streaming.zig`      | SSE streaming support             |
| `src/errors.zig`         | Error types                       |

### build.zig

```zig
const std = @import("std");

pub fn build(b: *std.Build) void {
    const target = b.standardTargetOptions(.{});
    const optimize = b.standardOptimizeOption(.{});

    const lib = b.addStaticLibrary(.{
        .name = "{package}",
        .root_source_file = .{ .path = "src/root.zig" },
        .target = target,
        .optimize = optimize,
    });

    b.installArtifact(lib);

    // Tests
    const main_tests = b.addTest(.{
        .root_source_file = .{ .path = "src/root.zig" },
        .target = target,
        .optimize = optimize,
    });

    const run_main_tests = b.addRunArtifact(main_tests);
    const test_step = b.step("test", "Run unit tests");
    test_step.dependOn(&run_main_tests.step);
}
```

### build.zig.zon

```zig
.{
    .name = "{package}",
    .version = "{version}",
    .dependencies = .{},
    .paths = .{
        "build.zig",
        "build.zig.zon",
        "src",
    },
}
```

## Usage Examples

### Basic Usage

```zig
const std = @import("std");
const sdk = @import("anthropic");

pub fn main() !void {
    var gpa = std.heap.GeneralPurposeAllocator(.{}){};
    defer _ = gpa.deinit();
    const allocator = gpa.allocator();

    // Create client
    var client = try sdk.Client.init(allocator, .{
        .api_key = std.posix.getenv("ANTHROPIC_API_KEY"),
    });
    defer client.deinit();

    // Build request
    const request = sdk.types.CreateMessageRequest{
        .model = "claude-3-sonnet-20240229",
        .max_tokens = 1024,
        .messages = &[_]sdk.types.Message{
            .{
                .role = .user,
                .content = &[_]sdk.types.ContentBlock{
                    .{ .text = .{ .text = "Hello, Claude!" } },
                },
            },
        },
    };

    // Make request
    var response = try client.messages().create(&request);
    defer response.deinit(allocator);

    // Print response
    for (response.content) |block| {
        if (block.asText()) |text| {
            std.debug.print("{s}\n", .{text.text});
        }
    }
}
```

### Streaming

```zig
const std = @import("std");
const sdk = @import("anthropic");

pub fn main() !void {
    var gpa = std.heap.GeneralPurposeAllocator(.{}){};
    defer _ = gpa.deinit();
    const allocator = gpa.allocator();

    var client = try sdk.Client.init(allocator, .{
        .api_key = std.posix.getenv("ANTHROPIC_API_KEY"),
    });
    defer client.deinit();

    const request = sdk.types.CreateMessageRequest{
        .model = "claude-3-sonnet-20240229",
        .max_tokens = 1024,
        .stream = true,
        .messages = &[_]sdk.types.Message{
            .{
                .role = .user,
                .content = &[_]sdk.types.ContentBlock{
                    .{ .text = .{ .text = "Tell me a story about a robot." } },
                },
            },
        },
    };

    // Stream response
    var stream = try client.messages().stream(&request);
    defer stream.deinit();

    const stdout = std.io.getStdOut().writer();
    while (try stream.next()) |event| {
        switch (event) {
            .content_block_delta => |delta| {
                if (delta.delta.text) |text| {
                    try stdout.print("{s}", .{text});
                }
            },
            .message_stop => {
                try stdout.print("\n", .{});
                break;
            },
            else => {},
        }
    }
}
```

### Error Handling

```zig
const std = @import("std");
const sdk = @import("anthropic");

fn example(client: *sdk.Client, request: *const sdk.types.CreateMessageRequest) void {
    const response = client.messages().create(request) catch |err| {
        switch (err) {
            error.HttpError => {
                std.debug.print("HTTP error occurred\n", .{});
            },
            error.Timeout => {
                std.debug.print("Request timed out\n", .{});
            },
            error.ConnectionFailed => {
                std.debug.print("Connection failed\n", .{});
            },
            error.DeserializationError => {
                std.debug.print("Failed to parse response\n", .{});
            },
            else => {
                std.debug.print("Error: {s}\n", .{@errorName(err)});
            },
        }
        return;
    };
    defer response.deinit(client.allocator);

    std.debug.print("Success: {d} tokens used\n", .{response.usage.total_tokens});
}
```

### Pattern Matching on Unions

```zig
for (response.content) |block| {
    switch (block) {
        .text => |text| {
            std.debug.print("Text: {s}\n", .{text.text});
        },
        .image => |image| {
            std.debug.print("Image: {s}\n", .{image.source});
        },
        .tool_use => |tool| {
            std.debug.print("Tool use: {s}\n", .{tool.name});
        },
    }
}

// Or use helper methods
for (response.content) |block| {
    if (block.asText()) |text| {
        std.debug.print("{s}\n", .{text.text});
    }
}
```

### Custom Allocator

```zig
const std = @import("std");
const sdk = @import("anthropic");

pub fn main() !void {
    // Use arena allocator for request-scoped allocations
    var arena = std.heap.ArenaAllocator.init(std.heap.page_allocator);
    defer arena.deinit();

    var client = try sdk.Client.init(arena.allocator(), .{
        .api_key = std.posix.getenv("ANTHROPIC_API_KEY"),
    });
    // No need to deinit - arena handles cleanup

    const request = sdk.types.CreateMessageRequest{
        .model = "claude-3-sonnet-20240229",
        .max_tokens = 1024,
        .messages = &[_]sdk.types.Message{},
    };

    const response = try client.messages().create(&request);
    // No need to deinit - arena handles cleanup

    std.debug.print("Response: {s}\n", .{response.id});
}
```

## Testing

### Generator Tests

```go
func TestGenerate_NilService(t *testing.T)
func TestGenerate_ValidZig_Compiles(t *testing.T)
func TestGenerate_ProducesExpectedFiles(t *testing.T)
func TestGenerate_TypeMapping(t *testing.T)
func TestGenerate_StreamingMethods(t *testing.T)
func TestGenerate_UnionTypes(t *testing.T)
func TestGenerate_OptionalFields(t *testing.T)
```

### Generated SDK Tests

```zig
const std = @import("std");
const testing = std.testing;
const sdk = @import("root.zig");

test "client initialization" {
    var client = try sdk.Client.init(testing.allocator, .{
        .api_key = "test-key",
    });
    defer client.deinit();

    try testing.expectEqualStrings("https://api.anthropic.com", client.baseUrl());
}

test "request serialization" {
    const request = sdk.types.CreateMessageRequest{
        .model = "claude-3",
        .max_tokens = 100,
        .messages = &[_]sdk.types.Message{},
    };

    const json = try request.toJson(testing.allocator);
    defer testing.allocator.free(json);

    try testing.expect(std.mem.indexOf(u8, json, "claude-3") != null);
}

test "enum serialization" {
    const role = sdk.types.Role.user;
    try testing.expectEqualStrings("user", role.toString());
}

test "union variant check" {
    const block = sdk.types.ContentBlock{ .text = .{ .text = "Hello" } };
    try testing.expect(block.isText());
    try testing.expect(!block.isImage());
}
```

## Platform Support

### Dependencies

**Required:**
- Zig 0.11.0+ standard library only
- No external dependencies

**Optional:**
- Custom allocators for specialized use cases

### Minimum Zig Version

Zig 0.11.0 (for stable std.http)

### Target Support

| Target                    | Status    | Notes                           |
|---------------------------|-----------|--------------------------------|
| x86_64-linux-gnu          | ✅        | Primary target                 |
| x86_64-macos              | ✅        | macOS Intel                    |
| aarch64-macos             | ✅        | macOS Apple Silicon            |
| x86_64-windows-msvc       | ✅        | Windows MSVC                   |
| wasm32-wasi               | ⚠️        | WASI target                    |
| freestanding              | ⚠️        | Requires custom HTTP impl      |

### Build Configurations

```zig
// Debug build (default)
zig build

// Release build (optimized)
zig build -Doptimize=ReleaseSafe

// Small binary
zig build -Doptimize=ReleaseSmall

// Maximum performance
zig build -Doptimize=ReleaseFast
```

## Future Enhancements

1. **Async I/O**: Integration with io_uring on Linux
2. **Connection pooling**: HTTP/2 multiplexing
3. **Custom TLS**: BearSSL or wolfSSL integration
4. **WASM support**: Browser-compatible build
5. **Comptime validation**: Validate requests at compile time
6. **Arena pools**: Pre-allocated arenas for zero-alloc hot paths
7. **Streaming iterators**: Lazy evaluation for large responses
8. **Mock client**: Built-in mock client for testing
9. **Retry policies**: Customizable retry strategies
10. **Rate limiting**: Client-side rate limit handling

## References

- [Zig Language Reference](https://ziglang.org/documentation/master/)
- [Zig Standard Library](https://ziglang.org/documentation/master/std/)
- [Zig Build System](https://ziglang.org/learn/build-system/)
- [Zig Style Guide](https://ziglang.org/documentation/master/#Style-Guide)
- [Zig Package Manager](https://ziglang.org/download/0.11.0/release-notes.html#Package-Manager)
