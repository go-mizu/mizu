# RFC 0066: Dart SDK Generator

## Summary

Add Dart SDK code generation to the Mizu contract system, enabling production-ready, type-safe Dart clients for Flutter, web, and server-side applications.

## Motivation

Dart is the primary language for Flutter development and increasingly popular for full-stack applications. A native Dart SDK provides:

1. **Async/await**: Native Future-based async operations with structured concurrency
2. **Stream for SSE**: First-class Stream support for reactive event handling
3. **Sound null safety**: Compile-time null safety guarantees
4. **Immutable classes**: Records and final classes for immutable data models
5. **Sealed classes**: Dart 3 sealed classes for exhaustive pattern matching on unions
6. **Cross-platform**: Single codebase for mobile, web, and server

## Design Goals

### Developer Experience (DX)

- **Idiomatic Dart**: Follow Dart effective style guide and naming conventions
- **Future-based async**: All network calls return `Future<T>` for structured concurrency
- **Stream for streaming**: Server-sent events via `Stream<T>` for reactive consumption
- **Null safety**: Full sound null safety with `?` operator and required parameters
- **Documentation comments**: dartdoc-compatible `///` documentation comments
- **Minimal dependencies**: Only `http` package for networking, `meta` for annotations

### Production Readiness

- **Retry logic**: Configurable retry with exponential backoff
- **Timeout handling**: Per-request and global timeout configuration
- **Cancellation**: Support for request cancellation via `CancelToken`
- **Error handling**: Sealed exception hierarchy for typed error handling
- **Thread safety**: Safe for concurrent access from multiple isolates
- **Logging**: Configurable logging interceptor for debugging

## Architecture

### Package Structure

```
{package}/
├── pubspec.yaml              # Dart package manifest
└── lib/
    ├── {package}.dart        # Library export file
    └── src/
        ├── client.dart       # Main client class
        ├── types.dart        # Generated model types
        ├── resources.dart    # Resource namespaces
        ├── streaming.dart    # SSE streaming support
        └── errors.dart       # Error types
```

### Core Components

#### 1. Client (`client.dart`)

The main entry point for API interactions:

```dart
/// Configuration options for the SDK client.
class ClientOptions {
  /// API key for authentication.
  final String? apiKey;

  /// Base URL for API requests.
  final String baseUrl;

  /// Request timeout.
  final Duration timeout;

  /// Maximum number of retry attempts for failed requests.
  final int maxRetries;

  /// Default headers to include in all requests.
  final Map<String, String> defaultHeaders;

  /// Authentication mode.
  final AuthMode authMode;

  const ClientOptions({
    this.apiKey,
    this.baseUrl = '{default_base_url}',
    this.timeout = const Duration(seconds: 60),
    this.maxRetries = 2,
    this.defaultHeaders = const {},
    this.authMode = AuthMode.bearer,
  });

  /// Creates a copy with modified fields.
  ClientOptions copyWith({
    String? apiKey,
    String? baseUrl,
    Duration? timeout,
    int? maxRetries,
    Map<String, String>? defaultHeaders,
    AuthMode? authMode,
  });
}

/// Authentication mode for API requests.
enum AuthMode {
  /// Bearer token authentication.
  bearer,
  /// Basic authentication.
  basic,
  /// No authentication.
  none,
}

/// The main SDK client providing access to all API resources.
class {ServiceName} {
  final ClientOptions options;
  final HttpClientWrapper _client;

  /// Access to {resource} operations.
  late final {Resource}Resource {resource};

  {ServiceName}({ClientOptions? options})
      : options = options ?? const ClientOptions(),
        _client = HttpClientWrapper(options ?? const ClientOptions()) {
    {resource} = {Resource}Resource(_client);
  }

  /// Creates a new client with modified configuration.
  {ServiceName} copyWith({
    String? apiKey,
    String? baseUrl,
    Duration? timeout,
    int? maxRetries,
    Map<String, String>? defaultHeaders,
    AuthMode? authMode,
  });

  /// Closes the HTTP client and releases resources.
  void close();
}
```

#### 2. Types (`types.dart`)

All model types use Dart 3 features with JSON serialization:

```dart
/// Request/response models
class {TypeName} {
  /// Description from contract
  final String fieldName;

  /// Optional field
  final String? optionalField;

  const {TypeName}({
    required this.fieldName,
    this.optionalField,
  });

  factory {TypeName}.fromJson(Map<String, dynamic> json);
  Map<String, dynamic> toJson();

  @override
  bool operator ==(Object other);

  @override
  int get hashCode;

  @override
  String toString();

  /// Creates a copy with modified fields.
  {TypeName} copyWith({
    String? fieldName,
    String? optionalField,
  });
}

/// Enum types with serialized names
enum {EnumName} {
  value1('value1'),
  value2('value2');

  final String value;
  const {EnumName}(this.value);

  factory {EnumName}.fromJson(String json);
  String toJson() => value;
}

/// Discriminated unions using sealed classes
sealed class {UnionName} {
  /// The discriminator tag value.
  String get type;

  const {UnionName}();

  factory {UnionName}.fromJson(Map<String, dynamic> json);
  Map<String, dynamic> toJson();
}

final class {Variant1} extends {UnionName} {
  @override
  String get type => 'variant1';

  // variant fields
  final String field;

  const {Variant1}({required this.field});

  factory {Variant1}.fromJson(Map<String, dynamic> json);

  @override
  Map<String, dynamic> toJson();
}
```

#### 3. Resources (`resources.dart`)

Resource classes provide namespaced method access:

```dart
/// Operations for {resource}
class {Resource}Resource {
  final HttpClientWrapper _client;

  const {Resource}Resource(this._client);

  /// Description from contract
  Future<ResponseType> methodName(RequestType request);

  /// Streaming method returning a Stream
  Stream<ItemType> streamMethod(RequestType request);
}
```

#### 4. Streaming (`streaming.dart`)

SSE streaming support via Dart Stream:

```dart
/// Parses Server-Sent Events from a byte stream into a Stream.
Stream<T> parseSSEStream<T>(
  Stream<List<int>> byteStream,
  T Function(Map<String, dynamic>) fromJson,
) async* {
  final buffer = StringBuffer();
  String partial = '';

  await for (final chunk in byteStream.transform(utf8.decoder)) {
    partial += chunk;
    final lines = partial.split('\n');
    partial = lines.removeLast();

    for (final line in lines) {
      if (line.isEmpty) {
        // Empty line = end of event
        final data = buffer.toString().trim();
        buffer.clear();

        if (data.isNotEmpty && data != '[DONE]') {
          yield fromJson(jsonDecode(data));
        }
      } else if (line.startsWith('data:')) {
        final content = line.substring(5).trimLeft();
        if (buffer.isNotEmpty) buffer.writeln();
        buffer.write(content);
      }
    }
  }
}
```

#### 5. Errors (`errors.dart`)

Sealed class hierarchy for typed errors:

```dart
/// Base error type for all SDK errors.
sealed class SDKException implements Exception {
  const SDKException();

  /// Returns true if this error is potentially retriable.
  bool get isRetriable;

  /// Returns true if this is a client error (4xx).
  bool get isClientError;

  /// Returns true if this is a server error (5xx).
  bool get isServerError;
}

/// Network connection failed.
final class ConnectionException extends SDKException {
  final Object? cause;
  const ConnectionException([this.cause]);

  @override
  bool get isRetriable => true;

  @override
  bool get isClientError => false;

  @override
  bool get isServerError => false;
}

/// Server returned an error status code.
final class ApiException extends SDKException {
  final int statusCode;
  final String message;
  final String? body;

  const ApiException({
    required this.statusCode,
    required this.message,
    this.body,
  });

  @override
  bool get isRetriable => statusCode >= 500 || statusCode == 429;

  @override
  bool get isClientError => statusCode >= 400 && statusCode < 500;

  @override
  bool get isServerError => statusCode >= 500;
}

/// Request timed out.
final class TimeoutException extends SDKException {
  const TimeoutException();

  @override
  bool get isRetriable => true;

  @override
  bool get isClientError => false;

  @override
  bool get isServerError => false;
}

/// Request was cancelled.
final class CancelledException extends SDKException {
  const CancelledException();

  @override
  bool get isRetriable => false;

  @override
  bool get isClientError => false;

  @override
  bool get isServerError => false;
}

/// Failed to encode request body.
final class EncodingException extends SDKException {
  final Object? cause;
  const EncodingException([this.cause]);

  @override
  bool get isRetriable => false;

  @override
  bool get isClientError => false;

  @override
  bool get isServerError => false;
}

/// Failed to decode response body.
final class DecodingException extends SDKException {
  final Object? cause;
  const DecodingException([this.cause]);

  @override
  bool get isRetriable => false;

  @override
  bool get isClientError => false;

  @override
  bool get isServerError => false;
}
```

## Type Mapping

### Primitive Types

| Contract Type     | Dart Type      |
|-------------------|----------------|
| `string`          | `String`       |
| `bool`, `boolean` | `bool`         |
| `int`             | `int`          |
| `int8`            | `int`          |
| `int16`           | `int`          |
| `int32`           | `int`          |
| `int64`           | `int`          |
| `uint`            | `int`          |
| `uint8`           | `int`          |
| `uint16`          | `int`          |
| `uint32`          | `int`          |
| `uint64`          | `int`          |
| `float32`         | `double`       |
| `float64`         | `double`       |
| `time.Time`       | `DateTime`     |
| `json.RawMessage` | `Object`       |
| `any`             | `Object`       |

### Collection Types

| Contract Type      | Dart Type                   |
|--------------------|-----------------------------|
| `[]T`              | `List<DartType>`            |
| `map[string]T`     | `Map<String, DartType>`     |

### Optional/Nullable

| Contract      | Dart Type           |
|---------------|---------------------|
| `optional: T` | `T?` (nullable)     |
| `nullable: T` | `T?` (nullable)     |

### Struct Fields

Fields with `optional: true` or `nullable: true` become nullable:

```dart
class Request {
  final String required;
  final String? optionalField;

  const Request({
    required this.required,
    this.optionalField,
  });

  factory Request.fromJson(Map<String, dynamic> json) {
    return Request(
      required: json['required'] as String,
      optionalField: json['optional_field'] as String?,
    );
  }

  Map<String, dynamic> toJson() => {
    'required': required,
    if (optionalField != null) 'optional_field': optionalField,
  };
}
```

### Enum/Const Values

Fields with `enum` constraint generate enums:

```dart
enum Role {
  user('user'),
  assistant('assistant'),
  system('system');

  final String value;
  const Role(this.value);

  factory Role.fromJson(String json) {
    return Role.values.firstWhere(
      (e) => e.value == json,
      orElse: () => throw ArgumentError('Unknown Role: $json'),
    );
  }

  String toJson() => value;
}
```

Fields with `const` constraint use a fixed value.

### Discriminated Unions

Union types use Dart 3 sealed classes for exhaustive pattern matching:

```dart
sealed class ContentBlock {
  const ContentBlock();

  String get type;

  factory ContentBlock.fromJson(Map<String, dynamic> json) {
    final type = json['type'] as String;
    return switch (type) {
      'text' => TextBlock.fromJson(json),
      'image' => ImageBlock.fromJson(json),
      'tool_use' => ToolUseBlock.fromJson(json),
      _ => throw ArgumentError('Unknown ContentBlock type: $type'),
    };
  }

  Map<String, dynamic> toJson();
}

final class TextBlock extends ContentBlock {
  @override
  String get type => 'text';

  final String text;

  const TextBlock({required this.text});

  factory TextBlock.fromJson(Map<String, dynamic> json) {
    return TextBlock(text: json['text'] as String);
  }

  @override
  Map<String, dynamic> toJson() => {
    'type': type,
    'text': text,
  };
}

final class ImageBlock extends ContentBlock {
  @override
  String get type => 'image';

  final String url;

  const ImageBlock({required this.url});

  factory ImageBlock.fromJson(Map<String, dynamic> json) {
    return ImageBlock(url: json['url'] as String);
  }

  @override
  Map<String, dynamic> toJson() => {
    'type': type,
    'url': url,
  };
}

final class ToolUseBlock extends ContentBlock {
  @override
  String get type => 'tool_use';

  final String id;
  final String name;
  final Object input;

  const ToolUseBlock({
    required this.id,
    required this.name,
    required this.input,
  });

  factory ToolUseBlock.fromJson(Map<String, dynamic> json) {
    return ToolUseBlock(
      id: json['id'] as String,
      name: json['name'] as String,
      input: json['input'],
    );
  }

  @override
  Map<String, dynamic> toJson() => {
    'type': type,
    'id': id,
    'name': name,
    'input': input,
  };
}
```

## HTTP Client Implementation

### Request Flow

```dart
class HttpClientWrapper {
  final ClientOptions options;
  final http.Client _client;

  HttpClientWrapper(this.options) : _client = http.Client();

  Future<T> request<T>(
    String method,
    String path,
    T Function(Map<String, dynamic>) fromJson, {
    Object? body,
  }) async {
    Exception? lastException;

    for (var attempt = 0; attempt <= options.maxRetries; attempt++) {
      try {
        final uri = Uri.parse('${options.baseUrl}$path');

        final request = http.Request(method, uri);
        _applyHeaders(request);
        _applyAuth(request);

        if (body != null) {
          request.headers['Content-Type'] = 'application/json';
          request.body = jsonEncode(body);
        }

        final streamedResponse = await _client
            .send(request)
            .timeout(options.timeout);

        final response = await http.Response.fromStream(streamedResponse);

        if (response.statusCode >= 400) {
          throw _parseErrorResponse(response);
        }

        return fromJson(jsonDecode(response.body));
      } on SDKException {
        rethrow; // Don't retry SDK errors
      } catch (e) {
        lastException = e as Exception;
        if (attempt < options.maxRetries) {
          await Future.delayed(_backoff(attempt));
        }
      }
    }

    throw ConnectionException(lastException);
  }

  Duration _backoff(int attempt) {
    const baseDelay = Duration(milliseconds: 500);
    return baseDelay * (1 << attempt); // Exponential backoff
  }
}
```

### Authentication

```dart
void _applyAuth(http.Request request) {
  final apiKey = options.apiKey;
  if (apiKey != null) {
    switch (options.authMode) {
      case AuthMode.bearer:
        request.headers['Authorization'] = 'Bearer $apiKey';
      case AuthMode.basic:
        request.headers['Authorization'] = 'Basic $apiKey';
      case AuthMode.none:
        break;
    }
  }
}
```

### SSE Streaming

```dart
Stream<T> stream<T>(
  String method,
  String path,
  T Function(Map<String, dynamic>) fromJson, {
  Object? body,
}) async* {
  final uri = Uri.parse('${options.baseUrl}$path');

  final request = http.Request(method, uri);
  _applyHeaders(request);
  _applyAuth(request);
  request.headers['Accept'] = 'text/event-stream';

  if (body != null) {
    request.headers['Content-Type'] = 'application/json';
    request.body = jsonEncode(body);
  }

  final response = await _client.send(request).timeout(options.timeout);

  if (response.statusCode >= 400) {
    final body = await response.stream.bytesToString();
    throw _parseErrorFromBody(response.statusCode, body);
  }

  yield* parseSSEStream(response.stream, fromJson);
}
```

## Configuration

### Default Values

From contract `Defaults`:

```dart
class ClientOptions {
  static const defaultOptions = ClientOptions(
    baseUrl: '{defaults.baseURL}',
    timeout: Duration(seconds: 60),
    maxRetries: 2,
    defaultHeaders: {
      // From defaults.headers
    },
  );
}
```

### Environment Variables

The SDK does NOT automatically read environment variables for API keys. This is intentional for security and explicitness:

```dart
final client = ServiceName(
  options: ClientOptions(
    apiKey: Platform.environment['SERVICE_API_KEY'],
  ),
);
```

## Naming Conventions

### Dart Naming

| Contract       | Dart                     |
|----------------|--------------------------|
| `user-id`      | `userId`                 |
| `user_name`    | `userName`               |
| `UserData`     | `UserData`               |
| `create`       | `create`                 |
| `get-user`     | `getUser`                |

Functions:
- `toDartName(s)`: Converts to lowerCamelCase (for properties/methods)
- `toDartTypeName(s)`: Converts to UpperCamelCase (for types)
- `sanitizeIdent(s)`: Removes invalid characters

Special handling:
- Reserved words: Prefixed with `$` (`$class`, `$if`, etc.)
- Dart keywords: `abstract`, `as`, `assert`, `async`, `await`, `break`, `case`, `catch`, `class`, `const`, `continue`, `covariant`, `default`, `deferred`, `do`, `dynamic`, `else`, `enum`, `export`, `extends`, `extension`, `external`, `factory`, `false`, `final`, `finally`, `for`, `Function`, `get`, `hide`, `if`, `implements`, `import`, `in`, `interface`, `is`, `late`, `library`, `mixin`, `new`, `null`, `of`, `on`, `operator`, `part`, `required`, `rethrow`, `return`, `sealed`, `set`, `show`, `static`, `super`, `switch`, `sync`, `this`, `throw`, `true`, `try`, `type`, `typedef`, `var`, `void`, `when`, `while`, `with`, `yield`

## Code Generation

### Generator Structure

```go
package sdkdart

type Config struct {
    // Package is the Dart package name.
    Package string

    // Version is the package version for pubspec.yaml.
    Version string

    // Description is the package description.
    Description string

    // MinSDK is the minimum Dart SDK version.
    // Default: "3.0.0".
    MinSDK string
}

func Generate(svc *contract.Service, cfg *Config) ([]*sdk.File, error)
```

### Template Files

```
templates/
├── pubspec.yaml.tmpl        # Package manifest
├── lib.dart.tmpl            # Library export file
├── client.dart.tmpl         # Main client
├── types.dart.tmpl          # Model types
├── resources.dart.tmpl      # Resource classes
├── streaming.dart.tmpl      # SSE support
└── errors.dart.tmpl         # Error types
```

### Generated Files

| File                    | Purpose                           |
|-------------------------|-----------------------------------|
| `pubspec.yaml`          | Dart package manifest             |
| `lib/{package}.dart`    | Library export file               |
| `lib/src/client.dart`   | Main client class                 |
| `lib/src/types.dart`    | All model type definitions        |
| `lib/src/resources.dart`| Resource namespaces and methods   |
| `lib/src/streaming.dart`| Stream-based SSE implementation   |
| `lib/src/errors.dart`   | Error type definitions            |

## Usage Examples

### Basic Usage

```dart
import 'package:service_sdk/service_sdk.dart';

// Create client
final client = ServiceName(
  options: ClientOptions(apiKey: 'your-api-key'),
);

// Make a request
final response = await client.completions.create(
  CreateRequest(
    model: 'model-name',
    messages: [
      Message(role: Role.user, content: 'Hello'),
    ],
  ),
);

print(response.content);
```

### Streaming

```dart
await for (final event in client.completions.createStream(
  CreateRequest(
    model: 'model-name',
    messages: [Message(role: Role.user, content: 'Hello')],
  ),
)) {
  print(event.delta.text);
}
```

### Error Handling

```dart
try {
  final response = await client.completions.create(request);
} on ApiException catch (e) {
  print('API Error ${e.statusCode}: ${e.message}');
} on TimeoutException {
  print('Request timed out');
} on CancelledException {
  print('Request was cancelled');
} on SDKException catch (e) {
  print('SDK error: $e');
}
```

### Pattern Matching on Unions

```dart
// Exhaustive pattern matching with Dart 3 sealed classes
switch (contentBlock) {
  case TextBlock(:final text):
    print('Text: $text');
  case ImageBlock(:final url):
    print('Image: $url');
  case ToolUseBlock(:final name, :final input):
    print('Tool use: $name($input)');
}
```

### Custom Configuration

```dart
final client = ServiceName(
  options: ClientOptions(
    apiKey: 'your-api-key',
    baseUrl: 'https://custom.api.com',
    timeout: Duration(seconds: 120),
    maxRetries: 3,
    defaultHeaders: {
      'X-Custom-Header': 'value',
    },
  ),
);
```

### Flutter Integration

```dart
class ApiProvider extends InheritedWidget {
  final ServiceName client;

  const ApiProvider({
    required this.client,
    required Widget child,
  }) : super(child: child);

  static ServiceName of(BuildContext context) {
    return context.dependOnInheritedWidgetOfExactType<ApiProvider>()!.client;
  }
}

// Usage in widget
class MyWidget extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    final client = ApiProvider.of(context);
    // Use client...
  }
}
```

### Riverpod Integration

```dart
final clientProvider = Provider<ServiceName>((ref) {
  final client = ServiceName(
    options: ClientOptions(apiKey: 'your-api-key'),
  );
  ref.onDispose(() => client.close());
  return client;
});

final messagesProvider = FutureProvider.autoDispose
    .family<Message, CreateRequest>((ref, request) async {
  final client = ref.watch(clientProvider);
  return client.completions.create(request);
});
```

## Testing

### Generator Tests

```go
func TestGenerate_NilService(t *testing.T)
func TestGenerate_ValidDart_Compiles(t *testing.T)
func TestGenerate_ProducesExpectedFiles(t *testing.T)
func TestGenerate_TypeMapping(t *testing.T)
func TestGenerate_StreamingMethods(t *testing.T)
```

### Generated SDK Tests

The generated SDK includes a test structure for users to add integration tests:

```
test/
└── {package}_test.dart
```

## Platform Support

### Dependencies

**Core Dependencies:**
- `http: ^1.1.0` - HTTP client

**Dev Dependencies:**
- `test: ^1.24.0` - Testing framework
- `lints: ^3.0.0` - Dart linting rules

### Minimum Versions

| Platform    | Minimum Version | Rationale                           |
|-------------|-----------------|-------------------------------------|
| Dart SDK    | 3.0.0           | Sealed classes, patterns, records   |
| Flutter     | 3.10.0          | Dart 3 support                      |

## Migration Path

For projects migrating from other HTTP clients:

1. Replace existing client initialization with SDK client
2. Update method calls to use generated resource methods
3. Replace custom model types with generated classes
4. Update error handling to use sealed exception pattern
5. Replace callback patterns with Future/Stream

## Future Enhancements

1. **Isolate support**: Background processing for heavy requests
2. **Caching**: Built-in response caching with configurable strategies
3. **Offline mode**: Queue requests when offline with sync on reconnect
4. **Request interceptors**: Middleware for custom request/response handling
5. **Metrics**: Request timing and success rate tracking
6. **Mock client**: Built-in mock client for testing

## References

- [Effective Dart](https://dart.dev/effective-dart)
- [Dart Language Tour](https://dart.dev/language)
- [Dart Null Safety](https://dart.dev/null-safety)
- [Dart Patterns](https://dart.dev/language/patterns)
- [Sealed Classes](https://dart.dev/language/class-modifiers#sealed)
- [http Package](https://pub.dev/packages/http)
