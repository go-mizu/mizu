# RFC 0065: Swift SDK Generator

## Summary

Add Swift SDK code generation to the Mizu contract system, enabling production-ready, type-safe Swift clients for iOS, macOS, watchOS, tvOS, and visionOS applications.

## Motivation

Swift is the primary language for Apple platform development, with a growing presence in server-side applications. A native Swift SDK provides:

1. **Native async/await support**: Leverages Swift's structured concurrency model
2. **Type safety**: Compile-time guarantees through Swift's strong type system
3. **Codable integration**: Native JSON encoding/decoding without third-party dependencies
4. **Platform integration**: Works seamlessly across all Apple platforms
5. **SwiftUI compatibility**: Easy integration with reactive UI frameworks

## Design Goals

### Developer Experience (DX)

- **Idiomatic Swift**: Follow Swift API design guidelines and naming conventions
- **Minimal dependencies**: Use only Foundation/URLSession (standard library)
- **Full async/await**: Modern concurrency without callback hell
- **Type-safe builders**: Optional builder pattern for complex requests
- **Comprehensive documentation**: DocC-compatible documentation comments

### Production Readiness

- **Retry logic**: Configurable retry with exponential backoff
- **Timeout handling**: Per-request and global timeout configuration
- **Cancellation**: Full support for Task cancellation and cooperative cancellation
- **Error handling**: Typed errors with status codes and response bodies
- **Thread safety**: Actor-based design for safe concurrent access
- **Logging**: Configurable logging hooks for debugging

## Architecture

### Package Structure

```
{package}/
├── Package.swift                 # Swift Package Manager manifest
└── Sources/
    └── {Package}/
        ├── Client.swift          # Main client class
        ├── Types.swift           # Generated model types
        ├── Resources.swift       # Resource namespaces
        ├── Streaming.swift       # SSE streaming support
        └── Errors.swift          # Error types
```

### Core Components

#### 1. Client (`Client.swift`)

The main entry point for API interactions:

```swift
/// Configuration options for the SDK client.
public struct ClientOptions: Sendable {
    public var apiKey: String?
    public var baseURL: URL
    public var timeout: TimeInterval
    public var maxRetries: Int
    public var defaultHeaders: [String: String]

    public init(
        apiKey: String? = nil,
        baseURL: URL? = nil,
        timeout: TimeInterval = 60,
        maxRetries: Int = 2,
        defaultHeaders: [String: String] = [:]
    )
}

/// The main SDK client providing access to all API resources.
public final class {ServiceName}: Sendable {
    /// Access to {resource} operations.
    public let {resource}: {Resource}Resource

    /// Creates a new client with the specified options.
    public init(options: ClientOptions = .init())

    /// Creates a new client with a modified configuration.
    public func with(options: ClientOptions) -> {ServiceName}
}
```

#### 2. Types (`Types.swift`)

All model types conform to `Codable` and `Sendable`:

```swift
/// Request/response models
public struct {TypeName}: Codable, Hashable, Sendable {
    /// Description from contract
    public let fieldName: FieldType

    public init(fieldName: FieldType)
}

/// Enum types with raw values
public enum {EnumName}: String, Codable, Hashable, Sendable, CaseIterable {
    case value1 = "value1"
    case value2 = "value2"
}

/// Union types using enum with associated values
public enum {UnionName}: Codable, Hashable, Sendable {
    case variant1(Variant1Type)
    case variant2(Variant2Type)

    /// The discriminator tag value
    public var tag: String { ... }
}
```

#### 3. Resources (`Resources.swift`)

Resource classes provide namespaced method access:

```swift
/// Operations for {resource}
public struct {Resource}Resource: Sendable {
    private let client: HTTPClient

    /// Description from contract
    public func methodName(request: RequestType) async throws -> ResponseType

    /// Streaming method
    public func streamMethod(request: RequestType) -> AsyncThrowingStream<ItemType, Error>
}
```

#### 4. Streaming (`Streaming.swift`)

SSE streaming support via `AsyncThrowingStream`:

```swift
/// An async sequence of server-sent events.
public struct EventStream<T: Decodable>: AsyncSequence {
    public typealias Element = T

    /// Collects all events into an array.
    public func collect() async throws -> [T]

    /// Cancels the stream.
    public func cancel()
}
```

#### 5. Errors (`Errors.swift`)

Typed error hierarchy:

```swift
/// Base error type for all SDK errors.
public enum SDKError: Error, Sendable {
    /// Network connection failed.
    case connectionError(underlying: Error)

    /// Server returned an error status code.
    case apiError(status: Int, message: String, body: Data?)

    /// Request timed out.
    case timeout

    /// Request was cancelled.
    case cancelled

    /// Failed to encode request body.
    case encodingError(underlying: Error)

    /// Failed to decode response body.
    case decodingError(underlying: Error)
}

/// Detailed API error with typed body.
public struct APIError: Error, Sendable {
    public let statusCode: Int
    public let message: String
    public let body: Any?

    /// Attempts to decode the error body as the specified type.
    public func decoded<T: Decodable>(as type: T.Type) throws -> T
}
```

## Type Mapping

### Primitive Types

| Contract Type     | Swift Type      |
|-------------------|-----------------|
| `string`          | `String`        |
| `bool`, `boolean` | `Bool`          |
| `int`             | `Int`           |
| `int8`            | `Int8`          |
| `int16`           | `Int16`         |
| `int32`           | `Int32`         |
| `int64`           | `Int64`         |
| `uint`            | `UInt`          |
| `uint8`           | `UInt8`         |
| `uint16`          | `UInt16`        |
| `uint32`          | `UInt32`        |
| `uint64`          | `UInt64`        |
| `float32`         | `Float`         |
| `float64`         | `Double`        |
| `time.Time`       | `Date`          |
| `json.RawMessage` | `AnyCodable`    |
| `any`             | `AnyCodable`    |

### Collection Types

| Contract Type      | Swift Type           |
|--------------------|----------------------|
| `[]T`              | `[SwiftType]`        |
| `map[string]T`     | `[String: SwiftType]`|

### Optional/Nullable

| Contract      | Swift Type     |
|---------------|----------------|
| `optional: T` | `T?`           |
| `nullable: T` | `T?`           |

### Struct Fields

Fields with `optional: true` or `nullable: true` become Swift optionals with custom `CodingKeys`:

```swift
public struct Request: Codable, Sendable {
    public let required: String
    public let optional: String?

    enum CodingKeys: String, CodingKey {
        case required
        case optional
    }

    public init(required: String, optional: String? = nil) {
        self.required = required
        self.optional = optional
    }
}
```

### Enum/Const Values

Fields with `enum` constraint generate String enums:

```swift
public enum Role: String, Codable, Hashable, Sendable, CaseIterable {
    case user = "user"
    case assistant = "assistant"
    case system = "system"
}
```

Fields with `const` constraint use a fixed value in initializers.

### Discriminated Unions

Union types use Swift enums with associated values:

```swift
public enum ContentBlock: Codable, Hashable, Sendable {
    case text(TextBlock)
    case image(ImageBlock)
    case toolUse(ToolUseBlock)

    private enum CodingKeys: String, CodingKey {
        case type  // discriminator tag
    }

    public init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        let type = try container.decode(String.self, forKey: .type)

        switch type {
        case "text":
            self = .text(try TextBlock(from: decoder))
        case "image":
            self = .image(try ImageBlock(from: decoder))
        case "tool_use":
            self = .toolUse(try ToolUseBlock(from: decoder))
        default:
            throw DecodingError.dataCorruptedError(...)
        }
    }

    public func encode(to encoder: Encoder) throws {
        switch self {
        case .text(let block):
            try block.encode(to: encoder)
        case .image(let block):
            try block.encode(to: encoder)
        case .toolUse(let block):
            try block.encode(to: encoder)
        }
    }
}
```

## HTTP Client Implementation

### Request Flow

```swift
internal actor HTTPClient {
    private let session: URLSession
    private let options: ClientOptions
    private let encoder: JSONEncoder
    private let decoder: JSONDecoder

    func request<T: Decodable>(
        method: String,
        path: String,
        body: (any Encodable)? = nil
    ) async throws -> T {
        let url = options.baseURL.appendingPathComponent(path)
        var request = URLRequest(url: url)
        request.httpMethod = method
        request.timeoutInterval = options.timeout

        // Apply headers
        applyDefaultHeaders(&request)
        applyAuth(&request)

        // Encode body
        if let body = body {
            request.httpBody = try encoder.encode(body)
            request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        }

        // Retry loop
        var lastError: Error?
        for attempt in 0...options.maxRetries {
            do {
                let (data, response) = try await session.data(for: request)
                return try handleResponse(data: data, response: response)
            } catch let error as SDKError {
                throw error  // Don't retry API errors
            } catch {
                lastError = error
                if attempt < options.maxRetries {
                    try await Task.sleep(nanoseconds: backoff(attempt: attempt))
                }
            }
        }
        throw SDKError.connectionError(underlying: lastError!)
    }
}
```

### Authentication

```swift
private func applyAuth(_ request: inout URLRequest) {
    guard let apiKey = options.apiKey else { return }

    switch options.authMode {
    case .bearer:
        request.setValue("Bearer \(apiKey)", forHTTPHeaderField: "Authorization")
    case .basic:
        request.setValue("Basic \(apiKey)", forHTTPHeaderField: "Authorization")
    case .none:
        break
    }
}
```

### SSE Streaming

```swift
func stream<T: Decodable>(
    method: String,
    path: String,
    body: (any Encodable)? = nil,
    as type: T.Type
) -> AsyncThrowingStream<T, Error> {
    AsyncThrowingStream { continuation in
        let task = Task {
            do {
                let url = options.baseURL.appendingPathComponent(path)
                var request = URLRequest(url: url)
                request.httpMethod = method
                request.setValue("text/event-stream", forHTTPHeaderField: "Accept")

                applyDefaultHeaders(&request)
                applyAuth(&request)

                if let body = body {
                    request.httpBody = try encoder.encode(body)
                    request.setValue("application/json", forHTTPHeaderField: "Content-Type")
                }

                let (bytes, response) = try await session.bytes(for: request)

                guard let httpResponse = response as? HTTPURLResponse else {
                    throw SDKError.connectionError(underlying: URLError(.badServerResponse))
                }

                if httpResponse.statusCode >= 400 {
                    var body = Data()
                    for try await byte in bytes {
                        body.append(byte)
                    }
                    throw try parseErrorResponse(statusCode: httpResponse.statusCode, data: body)
                }

                var buffer = ""
                for try await line in bytes.lines {
                    try Task.checkCancellation()

                    if line.isEmpty {
                        // Empty line = end of event
                        if !buffer.isEmpty {
                            let data = buffer.trimmingCharacters(in: .whitespaces)
                            buffer = ""

                            if data == "[DONE]" {
                                break
                            }

                            let item = try decoder.decode(T.self, from: Data(data.utf8))
                            continuation.yield(item)
                        }
                        continue
                    }

                    if line.hasPrefix("data:") {
                        var data = String(line.dropFirst(5))
                        if data.hasPrefix(" ") {
                            data = String(data.dropFirst())
                        }
                        if !buffer.isEmpty {
                            buffer += "\n"
                        }
                        buffer += data
                    }
                }

                continuation.finish()
            } catch {
                continuation.finish(throwing: error)
            }
        }

        continuation.onTermination = { @Sendable _ in
            task.cancel()
        }
    }
}
```

## Configuration

### Default Values

From contract `Defaults`:

```swift
public extension ClientOptions {
    static var `default`: ClientOptions {
        ClientOptions(
            baseURL: URL(string: "{defaults.baseURL}")!,
            timeout: 60,
            maxRetries: 2,
            defaultHeaders: [
                // From defaults.headers
            ]
        )
    }
}
```

### Environment Variables

The SDK does NOT automatically read environment variables for API keys. This is intentional for security and explicitness. Users should explicitly pass credentials:

```swift
let client = ServiceName(options: .init(
    apiKey: ProcessInfo.processInfo.environment["SERVICE_API_KEY"]
))
```

## Naming Conventions

### Swift Naming

| Contract       | Swift                    |
|----------------|--------------------------|
| `user-id`      | `userId`                 |
| `user_name`    | `userName`               |
| `UserData`     | `UserData`               |
| `create`       | `create`                 |
| `get-user`     | `getUser`                |

Functions:
- `toSwiftName(s)`: Converts to lowerCamelCase (for properties/methods)
- `toSwiftTypeName(s)`: Converts to UpperCamelCase (for types)
- `sanitizeIdent(s)`: Removes invalid characters

Special handling:
- Acronyms preserved in caps: `URL`, `ID`, `HTTP`, `API`, `SSE`, `JSON`
- Reserved words escaped: `` `class` ``, `` `protocol` ``, etc.

## Code Generation

### Generator Structure

```go
package sdkswift

type Config struct {
    // Package is the Swift module name.
    Package string

    // Version is the package version for Package.swift.
    Version string

    // Platforms specifies minimum platform versions.
    Platforms Platforms
}

type Platforms struct {
    iOS     string // e.g., "15.0"
    macOS   string // e.g., "12.0"
    watchOS string // e.g., "8.0"
    tvOS    string // e.g., "15.0"
}

func Generate(svc *contract.Service, cfg *Config) ([]*sdk.File, error)
```

### Template Files

```
templates/
├── Package.swift.tmpl      # SPM manifest
├── Client.swift.tmpl       # Main client
├── Types.swift.tmpl        # Model types
├── Resources.swift.tmpl    # Resource classes
├── Streaming.swift.tmpl    # SSE support
└── Errors.swift.tmpl       # Error types
```

### Generated Files

| File                | Purpose                           |
|---------------------|-----------------------------------|
| `Package.swift`     | Swift Package Manager manifest    |
| `Client.swift`      | Main client class                 |
| `Types.swift`       | All model type definitions        |
| `Resources.swift`   | Resource namespaces and methods   |
| `Streaming.swift`   | AsyncSequence SSE implementation  |
| `Errors.swift`      | Error type definitions            |

## Usage Examples

### Basic Usage

```swift
import ServiceSDK

// Create client
let client = ServiceName(options: .init(apiKey: "your-api-key"))

// Make a request
let response = try await client.completions.create(request: .init(
    model: "model-name",
    messages: [.init(role: .user, content: "Hello")]
))

print(response.content)
```

### Streaming

```swift
let stream = client.completions.createStream(request: .init(
    model: "model-name",
    messages: [.init(role: .user, content: "Hello")]
))

for try await event in stream {
    print(event.delta.text, terminator: "")
}
```

### Error Handling

```swift
do {
    let response = try await client.completions.create(request: request)
} catch let error as APIError {
    print("API Error \(error.statusCode): \(error.message)")
    if let body = try? error.decoded(as: ErrorResponse.self) {
        print("Error code: \(body.code)")
    }
} catch SDKError.timeout {
    print("Request timed out")
} catch SDKError.cancelled {
    print("Request was cancelled")
} catch {
    print("Unexpected error: \(error)")
}
```

### Task Cancellation

```swift
let task = Task {
    for try await event in client.completions.createStream(request: request) {
        print(event)
    }
}

// Cancel after 5 seconds
try await Task.sleep(nanoseconds: 5_000_000_000)
task.cancel()
```

### Custom Configuration

```swift
let client = ServiceName(options: .init(
    apiKey: "your-api-key",
    baseURL: URL(string: "https://custom.api.com")!,
    timeout: 120,
    maxRetries: 3,
    defaultHeaders: [
        "X-Custom-Header": "value"
    ]
))
```

## Testing

### Generator Tests

```go
func TestGenerate_NilService(t *testing.T)
func TestGenerate_ValidSwift_Compiles(t *testing.T)
func TestGenerate_ProducesExpectedFiles(t *testing.T)
func TestGenerate_TypeMapping(t *testing.T)
func TestGenerate_StreamingMethods(t *testing.T)
```

### Generated SDK Tests

The generated SDK includes a test file structure for users to add integration tests:

```
Tests/
└── {Package}Tests/
    └── {Package}Tests.swift
```

## Platform Support

### Minimum Versions

| Platform  | Minimum Version | Rationale                    |
|-----------|-----------------|------------------------------|
| iOS       | 15.0            | async/await, URLSession.bytes|
| macOS     | 12.0            | async/await, URLSession.bytes|
| watchOS   | 8.0             | async/await support          |
| tvOS      | 15.0            | async/await support          |
| visionOS  | 1.0             | All features available       |

### Dependencies

**None** - Uses only Foundation framework.

## Migration Path

For projects migrating from other HTTP clients:

1. Replace existing client initialization with SDK client
2. Update method calls to use generated resource methods
3. Replace custom model types with generated types
4. Update error handling to use SDK error types

## Future Enhancements

1. **Combine support**: Publishers for reactive programming
2. **Request interceptors**: Middleware for custom request/response handling
3. **Caching**: Built-in response caching
4. **Metrics**: Request timing and success rate tracking
5. **Offline support**: Queue requests when offline

## References

- [Swift API Design Guidelines](https://swift.org/documentation/api-design-guidelines/)
- [Swift Concurrency](https://docs.swift.org/swift-book/LanguageGuide/Concurrency.html)
- [URLSession](https://developer.apple.com/documentation/foundation/urlsession)
- [Codable](https://developer.apple.com/documentation/swift/codable)
- [Swift Package Manager](https://swift.org/package-manager/)
