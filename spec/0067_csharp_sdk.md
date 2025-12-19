# RFC 0067: C# SDK Generator

## Summary

Add C# SDK code generation to the Mizu contract system, enabling production-ready, type-safe C# clients for .NET applications including ASP.NET, MAUI, Blazor, and console applications.

## Motivation

C# is a primary language for enterprise development and cross-platform applications via .NET. A native C# SDK provides:

1. **async/await**: First-class async support with Task-based async pattern (TAP)
2. **IAsyncEnumerable**: Native streaming support for SSE with `await foreach`
3. **Nullable reference types**: Compile-time null safety with `?` annotations
4. **Records**: Immutable data models with value semantics and `with` expressions
5. **Pattern matching**: Exhaustive switch expressions for discriminated unions
6. **System.Text.Json**: High-performance, source-generated JSON serialization
7. **Cross-platform**: Single codebase for Windows, Linux, macOS, iOS, Android, WebAssembly

## Design Goals

### Developer Experience (DX)

- **Idiomatic C#**: Follow .NET naming conventions and design guidelines
- **Task-based async**: All network calls return `Task<T>` with cancellation support
- **IAsyncEnumerable streaming**: Server-sent events via `await foreach` pattern
- **Nullable reference types**: Full NRT support with `#nullable enable`
- **XML documentation**: IntelliSense-friendly `///` documentation comments
- **Minimal dependencies**: Only `System.Net.Http` and `System.Text.Json` (both built-in)
- **Source generators ready**: Compatible with System.Text.Json source generation

### Production Readiness

- **Retry logic**: Configurable retry with exponential backoff and jitter
- **Timeout handling**: Per-request and global timeout via CancellationToken
- **Cancellation**: Full CancellationToken support throughout
- **Error handling**: Exception hierarchy with typed API errors
- **Thread safety**: HttpClient best practices for concurrent access
- **Logging**: ILogger integration for structured logging
- **Resilience**: Compatible with Polly for advanced resilience patterns

## Architecture

### Package Structure

```
{PackageName}/
├── {PackageName}.csproj          # Project file
└── src/
    ├── {ServiceName}Client.cs    # Main client class
    ├── Models/
    │   └── Types.cs              # Generated model types (records)
    ├── Resources/
    │   └── {Resource}Resource.cs # Resource namespace classes
    ├── Streaming.cs              # SSE streaming support
    └── Exceptions.cs             # Exception types
```

### Core Components

#### 1. Client (`{ServiceName}Client.cs`)

The main entry point for API interactions:

```csharp
using System.Net.Http.Headers;

namespace {Namespace};

/// <summary>
/// Configuration options for the SDK client.
/// </summary>
public sealed record ClientOptions
{
    /// <summary>
    /// API key for authentication.
    /// </summary>
    public string? ApiKey { get; init; }

    /// <summary>
    /// Base URL for API requests.
    /// </summary>
    public string BaseUrl { get; init; } = "{default_base_url}";

    /// <summary>
    /// Request timeout. Default: 60 seconds.
    /// </summary>
    public TimeSpan Timeout { get; init; } = TimeSpan.FromSeconds(60);

    /// <summary>
    /// Maximum number of retry attempts for failed requests. Default: 2.
    /// </summary>
    public int MaxRetries { get; init; } = 2;

    /// <summary>
    /// Default headers to include in all requests.
    /// </summary>
    public IReadOnlyDictionary<string, string> DefaultHeaders { get; init; } =
        new Dictionary<string, string>();

    /// <summary>
    /// Authentication mode for API requests.
    /// </summary>
    public AuthMode AuthMode { get; init; } = AuthMode.Bearer;

    /// <summary>
    /// Optional HttpClient to use instead of creating a new one.
    /// </summary>
    public HttpClient? HttpClient { get; init; }
}

/// <summary>
/// Authentication mode for API requests.
/// </summary>
public enum AuthMode
{
    /// <summary>Bearer token authentication.</summary>
    Bearer,
    /// <summary>Basic authentication.</summary>
    Basic,
    /// <summary>No authentication.</summary>
    None
}

/// <summary>
/// The main SDK client providing access to all API resources.
/// </summary>
public sealed class {ServiceName}Client : IDisposable
{
    private readonly HttpClient _httpClient;
    private readonly bool _ownsHttpClient;
    private readonly ClientOptions _options;
    private bool _disposed;

    /// <summary>
    /// Access to {resource} operations.
    /// </summary>
    public {Resource}Resource {Resource} { get; }

    /// <summary>
    /// Creates a new client with the specified options.
    /// </summary>
    public {ServiceName}Client(ClientOptions? options = null)
    {
        _options = options ?? new ClientOptions();

        if (_options.HttpClient is not null)
        {
            _httpClient = _options.HttpClient;
            _ownsHttpClient = false;
        }
        else
        {
            _httpClient = new HttpClient { Timeout = _options.Timeout };
            _ownsHttpClient = true;
        }

        {Resource} = new {Resource}Resource(_httpClient, _options);
    }

    /// <summary>
    /// Creates a new client with modified configuration.
    /// </summary>
    public {ServiceName}Client WithOptions(Func<ClientOptions, ClientOptions> configure)
    {
        return new {ServiceName}Client(configure(_options));
    }

    public void Dispose()
    {
        if (_disposed) return;
        _disposed = true;

        if (_ownsHttpClient)
        {
            _httpClient.Dispose();
        }
    }
}
```

#### 2. Types (`Models/Types.cs`)

All model types use C# records with JSON serialization:

```csharp
using System.Text.Json.Serialization;

namespace {Namespace}.Models;

/// <summary>
/// Request/response model description.
/// </summary>
public sealed record {TypeName}
{
    /// <summary>
    /// Field description from contract.
    /// </summary>
    [JsonPropertyName("field_name")]
    public required string FieldName { get; init; }

    /// <summary>
    /// Optional field description.
    /// </summary>
    [JsonPropertyName("optional_field")]
    [JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)]
    public string? OptionalField { get; init; }
}

/// <summary>
/// Enum types with JSON string serialization.
/// </summary>
[JsonConverter(typeof(JsonStringEnumConverter<{EnumName}>))]
public enum {EnumName}
{
    [JsonPropertyName("value1")]
    Value1,

    [JsonPropertyName("value2")]
    Value2
}

/// <summary>
/// Discriminated unions using polymorphic JSON.
/// </summary>
[JsonPolymorphic(TypeDiscriminatorPropertyName = "type")]
[JsonDerivedType(typeof({Variant1}), "{variant1_value}")]
[JsonDerivedType(typeof({Variant2}), "{variant2_value}")]
public abstract record {UnionName}
{
    /// <summary>
    /// The discriminator tag value.
    /// </summary>
    [JsonPropertyName("type")]
    public abstract string Type { get; }
}

public sealed record {Variant1} : {UnionName}
{
    public override string Type => "{variant1_value}";

    [JsonPropertyName("field")]
    public required string Field { get; init; }
}
```

#### 3. Resources (`Resources/{Resource}Resource.cs`)

Resource classes provide namespaced method access:

```csharp
using System.Net.Http.Json;
using System.Runtime.CompilerServices;

namespace {Namespace}.Resources;

/// <summary>
/// Operations for {resource}.
/// </summary>
public sealed class {Resource}Resource
{
    private readonly HttpClient _httpClient;
    private readonly ClientOptions _options;

    internal {Resource}Resource(HttpClient httpClient, ClientOptions options)
    {
        _httpClient = httpClient;
        _options = options;
    }

    /// <summary>
    /// Method description from contract.
    /// </summary>
    /// <param name="request">The request parameters.</param>
    /// <param name="cancellationToken">Cancellation token.</param>
    /// <returns>The response.</returns>
    /// <exception cref="ApiException">Thrown when the API returns an error.</exception>
    public async Task<ResponseType> MethodNameAsync(
        RequestType request,
        CancellationToken cancellationToken = default)
    {
        // Implementation
    }

    /// <summary>
    /// Streaming method returning an async enumerable.
    /// </summary>
    public async IAsyncEnumerable<ItemType> StreamMethodAsync(
        RequestType request,
        [EnumeratorCancellation] CancellationToken cancellationToken = default)
    {
        // Implementation with yield return
    }
}
```

#### 4. Streaming (`Streaming.cs`)

SSE streaming support via IAsyncEnumerable:

```csharp
using System.Runtime.CompilerServices;
using System.Text;
using System.Text.Json;

namespace {Namespace};

internal static class SseParser
{
    /// <summary>
    /// Parses Server-Sent Events from an HTTP response stream.
    /// </summary>
    public static async IAsyncEnumerable<T> ParseAsync<T>(
        Stream stream,
        JsonSerializerOptions? jsonOptions = null,
        [EnumeratorCancellation] CancellationToken cancellationToken = default)
    {
        using var reader = new StreamReader(stream, Encoding.UTF8);
        var dataBuffer = new StringBuilder();

        while (!reader.EndOfStream)
        {
            cancellationToken.ThrowIfCancellationRequested();

            var line = await reader.ReadLineAsync(cancellationToken);
            if (line is null) break;

            if (line.Length == 0)
            {
                // Empty line = end of event
                var data = dataBuffer.ToString().Trim();
                dataBuffer.Clear();

                if (data.Length > 0 && data != "[DONE]")
                {
                    var item = JsonSerializer.Deserialize<T>(data, jsonOptions);
                    if (item is not null)
                    {
                        yield return item;
                    }
                }
            }
            else if (line.StartsWith("data:", StringComparison.Ordinal))
            {
                var content = line.AsSpan(5).TrimStart();
                if (dataBuffer.Length > 0) dataBuffer.AppendLine();
                dataBuffer.Append(content);
            }
            // Ignore other SSE fields (event:, id:, retry:)
        }
    }
}
```

#### 5. Exceptions (`Exceptions.cs`)

Exception hierarchy for typed error handling:

```csharp
namespace {Namespace};

/// <summary>
/// Base exception type for all SDK errors.
/// </summary>
public abstract class SdkException : Exception
{
    protected SdkException(string message) : base(message) { }
    protected SdkException(string message, Exception innerException)
        : base(message, innerException) { }

    /// <summary>
    /// Returns true if this error is potentially retriable.
    /// </summary>
    public abstract bool IsRetriable { get; }
}

/// <summary>
/// Network connection failed.
/// </summary>
public sealed class ConnectionException : SdkException
{
    public ConnectionException(Exception innerException)
        : base("Connection failed", innerException) { }

    public override bool IsRetriable => true;
}

/// <summary>
/// Server returned an error status code.
/// </summary>
public sealed class ApiException : SdkException
{
    public int StatusCode { get; }
    public string? ResponseBody { get; }

    public ApiException(int statusCode, string message, string? responseBody = null)
        : base(message)
    {
        StatusCode = statusCode;
        ResponseBody = responseBody;
    }

    public override bool IsRetriable => StatusCode >= 500 || StatusCode == 429;

    /// <summary>
    /// Returns true if this is a client error (4xx).
    /// </summary>
    public bool IsClientError => StatusCode >= 400 && StatusCode < 500;

    /// <summary>
    /// Returns true if this is a server error (5xx).
    /// </summary>
    public bool IsServerError => StatusCode >= 500;
}

/// <summary>
/// Request timed out or was cancelled.
/// </summary>
public sealed class TimeoutException : SdkException
{
    public TimeoutException() : base("Request timed out") { }
    public TimeoutException(Exception innerException)
        : base("Request timed out", innerException) { }

    public override bool IsRetriable => true;
}

/// <summary>
/// Request was cancelled via CancellationToken.
/// </summary>
public sealed class CancelledException : SdkException
{
    public CancelledException() : base("Request was cancelled") { }

    public override bool IsRetriable => false;
}

/// <summary>
/// Failed to serialize request body.
/// </summary>
public sealed class SerializationException : SdkException
{
    public SerializationException(Exception innerException)
        : base("Failed to serialize request", innerException) { }

    public override bool IsRetriable => false;
}

/// <summary>
/// Failed to deserialize response body.
/// </summary>
public sealed class DeserializationException : SdkException
{
    public DeserializationException(Exception innerException)
        : base("Failed to deserialize response", innerException) { }

    public override bool IsRetriable => false;
}
```

## Type Mapping

### Primitive Types

| Contract Type     | C# Type          |
|-------------------|------------------|
| `string`          | `string`         |
| `bool`, `boolean` | `bool`           |
| `int`             | `int`            |
| `int8`            | `sbyte`          |
| `int16`           | `short`          |
| `int32`           | `int`            |
| `int64`           | `long`           |
| `uint`            | `uint`           |
| `uint8`           | `byte`           |
| `uint16`          | `ushort`         |
| `uint32`          | `uint`           |
| `uint64`          | `ulong`          |
| `float32`         | `float`          |
| `float64`         | `double`         |
| `time.Time`       | `DateTimeOffset` |
| `json.RawMessage` | `JsonElement`    |
| `any`             | `JsonElement`    |

### Collection Types

| Contract Type      | C# Type                          |
|--------------------|----------------------------------|
| `[]T`              | `IReadOnlyList<CSharpType>`      |
| `map[string]T`     | `IReadOnlyDictionary<string, T>` |

### Optional/Nullable

| Contract      | C# Type             |
|---------------|---------------------|
| `optional: T` | `T?` (nullable)     |
| `nullable: T` | `T?` (nullable)     |

### Struct Fields

Fields with `optional: true` or `nullable: true` become nullable:

```csharp
public sealed record Request
{
    [JsonPropertyName("required")]
    public required string Required { get; init; }

    [JsonPropertyName("optional_field")]
    [JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)]
    public string? OptionalField { get; init; }
}
```

### Enum/Const Values

Fields with `enum` constraint generate strongly-typed enums:

```csharp
[JsonConverter(typeof(RoleJsonConverter))]
public enum Role
{
    User,
    Assistant,
    System
}

internal sealed class RoleJsonConverter : JsonConverter<Role>
{
    public override Role Read(ref Utf8JsonReader reader, Type typeToConvert, JsonSerializerOptions options)
    {
        var value = reader.GetString();
        return value switch
        {
            "user" => Role.User,
            "assistant" => Role.Assistant,
            "system" => Role.System,
            _ => throw new JsonException($"Unknown Role: {value}")
        };
    }

    public override void Write(Utf8JsonWriter writer, Role value, JsonSerializerOptions options)
    {
        var str = value switch
        {
            Role.User => "user",
            Role.Assistant => "assistant",
            Role.System => "system",
            _ => throw new JsonException($"Unknown Role: {value}")
        };
        writer.WriteStringValue(str);
    }
}
```

### Discriminated Unions

Union types use System.Text.Json polymorphic serialization:

```csharp
[JsonPolymorphic(TypeDiscriminatorPropertyName = "type")]
[JsonDerivedType(typeof(TextBlock), "text")]
[JsonDerivedType(typeof(ImageBlock), "image")]
[JsonDerivedType(typeof(ToolUseBlock), "tool_use")]
public abstract record ContentBlock
{
    [JsonPropertyName("type")]
    public abstract string Type { get; }
}

public sealed record TextBlock : ContentBlock
{
    public override string Type => "text";

    [JsonPropertyName("text")]
    public required string Text { get; init; }
}

public sealed record ImageBlock : ContentBlock
{
    public override string Type => "image";

    [JsonPropertyName("url")]
    public required string Url { get; init; }
}

public sealed record ToolUseBlock : ContentBlock
{
    public override string Type => "tool_use";

    [JsonPropertyName("id")]
    public required string Id { get; init; }

    [JsonPropertyName("name")]
    public required string Name { get; init; }

    [JsonPropertyName("input")]
    public required JsonElement Input { get; init; }
}
```

Pattern matching on unions:

```csharp
var message = contentBlock switch
{
    TextBlock text => $"Text: {text.Text}",
    ImageBlock image => $"Image: {image.Url}",
    ToolUseBlock tool => $"Tool: {tool.Name}",
    _ => throw new InvalidOperationException("Unknown content block type")
};
```

## HTTP Client Implementation

### Request Flow

```csharp
internal sealed class HttpClientWrapper
{
    private readonly HttpClient _httpClient;
    private readonly ClientOptions _options;
    private readonly JsonSerializerOptions _jsonOptions;
    private static readonly Random _jitter = new();

    public HttpClientWrapper(HttpClient httpClient, ClientOptions options)
    {
        _httpClient = httpClient;
        _options = options;
        _jsonOptions = new JsonSerializerOptions
        {
            PropertyNamingPolicy = JsonNamingPolicy.SnakeCaseLower,
            DefaultIgnoreCondition = JsonIgnoreCondition.WhenWritingNull
        };
    }

    public async Task<T> RequestAsync<T>(
        HttpMethod method,
        string path,
        object? body = null,
        CancellationToken cancellationToken = default)
    {
        Exception? lastException = null;

        for (var attempt = 0; attempt <= _options.MaxRetries; attempt++)
        {
            try
            {
                using var request = new HttpRequestMessage(method, BuildUri(path));
                ApplyHeaders(request);

                if (body is not null)
                {
                    request.Content = JsonContent.Create(body, options: _jsonOptions);
                }

                using var response = await _httpClient
                    .SendAsync(request, cancellationToken)
                    .ConfigureAwait(false);

                if (!response.IsSuccessStatusCode)
                {
                    var responseBody = await response.Content
                        .ReadAsStringAsync(cancellationToken)
                        .ConfigureAwait(false);

                    throw new ApiException(
                        (int)response.StatusCode,
                        $"API error: {response.StatusCode}",
                        responseBody);
                }

                var result = await response.Content
                    .ReadFromJsonAsync<T>(_jsonOptions, cancellationToken)
                    .ConfigureAwait(false);

                return result ?? throw new DeserializationException(
                    new InvalidOperationException("Response was null"));
            }
            catch (OperationCanceledException) when (cancellationToken.IsCancellationRequested)
            {
                throw new CancelledException();
            }
            catch (TaskCanceledException ex) when (!cancellationToken.IsCancellationRequested)
            {
                lastException = new TimeoutException(ex);
                if (attempt < _options.MaxRetries)
                {
                    await Task.Delay(GetBackoff(attempt), cancellationToken).ConfigureAwait(false);
                }
            }
            catch (HttpRequestException ex)
            {
                lastException = new ConnectionException(ex);
                if (attempt < _options.MaxRetries)
                {
                    await Task.Delay(GetBackoff(attempt), cancellationToken).ConfigureAwait(false);
                }
            }
            catch (ApiException ex) when (ex.IsRetriable && attempt < _options.MaxRetries)
            {
                lastException = ex;
                await Task.Delay(GetBackoff(attempt), cancellationToken).ConfigureAwait(false);
            }
        }

        throw lastException ?? new ConnectionException(new Exception("Unknown error"));
    }

    private Uri BuildUri(string path) => new($"{_options.BaseUrl.TrimEnd('/')}{path}");

    private TimeSpan GetBackoff(int attempt)
    {
        var baseDelay = TimeSpan.FromMilliseconds(500);
        var exponential = baseDelay * (1 << attempt);
        var jitter = TimeSpan.FromMilliseconds(_jitter.Next(0, 100));
        return exponential + jitter;
    }
}
```

### Authentication

```csharp
private void ApplyHeaders(HttpRequestMessage request)
{
    // Apply default headers
    foreach (var header in _options.DefaultHeaders)
    {
        request.Headers.TryAddWithoutValidation(header.Key, header.Value);
    }

    // Apply authentication
    if (_options.ApiKey is not null)
    {
        switch (_options.AuthMode)
        {
            case AuthMode.Bearer:
                request.Headers.Authorization =
                    new AuthenticationHeaderValue("Bearer", _options.ApiKey);
                break;
            case AuthMode.Basic:
                request.Headers.Authorization =
                    new AuthenticationHeaderValue("Basic", _options.ApiKey);
                break;
        }
    }
}
```

### SSE Streaming

```csharp
public async IAsyncEnumerable<T> StreamAsync<T>(
    HttpMethod method,
    string path,
    object? body = null,
    [EnumeratorCancellation] CancellationToken cancellationToken = default)
{
    using var request = new HttpRequestMessage(method, BuildUri(path));
    ApplyHeaders(request);
    request.Headers.Accept.Add(new MediaTypeWithQualityHeaderValue("text/event-stream"));

    if (body is not null)
    {
        request.Content = JsonContent.Create(body, options: _jsonOptions);
    }

    using var response = await _httpClient
        .SendAsync(request, HttpCompletionOption.ResponseHeadersRead, cancellationToken)
        .ConfigureAwait(false);

    if (!response.IsSuccessStatusCode)
    {
        var responseBody = await response.Content
            .ReadAsStringAsync(cancellationToken)
            .ConfigureAwait(false);

        throw new ApiException(
            (int)response.StatusCode,
            $"API error: {response.StatusCode}",
            responseBody);
    }

    await using var stream = await response.Content
        .ReadAsStreamAsync(cancellationToken)
        .ConfigureAwait(false);

    await foreach (var item in SseParser.ParseAsync<T>(stream, _jsonOptions, cancellationToken))
    {
        yield return item;
    }
}
```

## Configuration

### Default Values

From contract `Defaults`:

```csharp
public sealed record ClientOptions
{
    public string BaseUrl { get; init; } = "{defaults.baseURL}";
    public TimeSpan Timeout { get; init; } = TimeSpan.FromSeconds(60);
    public int MaxRetries { get; init; } = 2;
    public IReadOnlyDictionary<string, string> DefaultHeaders { get; init; } =
        new Dictionary<string, string>
        {
            // From defaults.headers
        };
}
```

### Environment Variables

The SDK does NOT automatically read environment variables. Users should handle this explicitly:

```csharp
var client = new ServiceNameClient(new ClientOptions
{
    ApiKey = Environment.GetEnvironmentVariable("SERVICE_API_KEY")
});
```

## Naming Conventions

### C# Naming

| Contract       | C#                       |
|----------------|--------------------------|
| `user-id`      | `UserId`                 |
| `user_name`    | `UserName`               |
| `user`         | `User`                   |
| `create`       | `CreateAsync`            |
| `get-user`     | `GetUserAsync`           |

Functions:
- `ToCSharpName(s)`: Converts to PascalCase (for properties/types)
- `ToCSharpMethodName(s)`: Converts to PascalCase + "Async" suffix
- `SanitizeIdent(s)`: Removes invalid characters

Special handling:
- Reserved words: Prefixed with `@` (`@class`, `@event`, etc.)
- Async methods: Always suffixed with `Async`
- C# keywords: `abstract`, `as`, `base`, `bool`, `break`, `byte`, `case`, `catch`, `char`, `checked`, `class`, `const`, `continue`, `decimal`, `default`, `delegate`, `do`, `double`, `else`, `enum`, `event`, `explicit`, `extern`, `false`, `finally`, `fixed`, `float`, `for`, `foreach`, `goto`, `if`, `implicit`, `in`, `int`, `interface`, `internal`, `is`, `lock`, `long`, `namespace`, `new`, `null`, `object`, `operator`, `out`, `override`, `params`, `private`, `protected`, `public`, `readonly`, `ref`, `return`, `sbyte`, `sealed`, `short`, `sizeof`, `stackalloc`, `static`, `string`, `struct`, `switch`, `this`, `throw`, `true`, `try`, `typeof`, `uint`, `ulong`, `unchecked`, `unsafe`, `ushort`, `using`, `virtual`, `void`, `volatile`, `while`

## Code Generation

### Generator Structure

```go
package sdkcsharp

type Config struct {
    // Namespace is the C# namespace for generated code.
    Namespace string

    // PackageName is the NuGet package name.
    PackageName string

    // Version is the package version.
    Version string

    // TargetFramework is the target .NET version.
    // Default: "net8.0".
    TargetFramework string

    // Nullable enables nullable reference types.
    // Default: true.
    Nullable bool
}

func Generate(svc *contract.Service, cfg *Config) ([]*sdk.File, error)
```

### Template Files

```
templates/
├── project.csproj.tmpl       # Project file
├── Client.cs.tmpl            # Main client
├── Types.cs.tmpl             # Model types (records)
├── Resources.cs.tmpl         # Resource classes
├── Streaming.cs.tmpl         # SSE support
└── Exceptions.cs.tmpl        # Exception types
```

### Generated Files

| File                              | Purpose                           |
|-----------------------------------|-----------------------------------|
| `{Package}.csproj`                | .NET project file                 |
| `src/{Service}Client.cs`          | Main client class                 |
| `src/Models/Types.cs`             | All model type definitions        |
| `src/Resources/{Resource}.cs`     | Resource namespaces and methods   |
| `src/Streaming.cs`                | IAsyncEnumerable SSE parsing      |
| `src/Exceptions.cs`               | Exception type definitions        |

## Usage Examples

### Basic Usage

```csharp
using ServiceName.Sdk;
using ServiceName.Sdk.Models;

// Create client
var client = new ServiceNameClient(new ClientOptions
{
    ApiKey = "your-api-key"
});

// Make a request
var response = await client.Completions.CreateAsync(new CreateRequest
{
    Model = "model-name",
    Messages = new[]
    {
        new Message { Role = Role.User, Content = "Hello" }
    }
});

Console.WriteLine(response.Content);
```

### Streaming

```csharp
await foreach (var chunk in client.Completions.CreateStreamAsync(new CreateRequest
{
    Model = "model-name",
    Messages = new[] { new Message { Role = Role.User, Content = "Hello" } }
}))
{
    Console.Write(chunk.Delta?.Text);
}
```

### Error Handling

```csharp
try
{
    var response = await client.Completions.CreateAsync(request);
}
catch (ApiException ex) when (ex.StatusCode == 429)
{
    Console.WriteLine($"Rate limited, retry after backoff");
}
catch (ApiException ex) when (ex.IsClientError)
{
    Console.WriteLine($"Client error {ex.StatusCode}: {ex.Message}");
}
catch (ApiException ex) when (ex.IsServerError)
{
    Console.WriteLine($"Server error {ex.StatusCode}: {ex.Message}");
}
catch (TimeoutException)
{
    Console.WriteLine("Request timed out");
}
catch (CancelledException)
{
    Console.WriteLine("Request was cancelled");
}
catch (SdkException ex)
{
    Console.WriteLine($"SDK error: {ex.Message}");
}
```

### Pattern Matching on Unions

```csharp
// Exhaustive pattern matching with switch expression
var message = contentBlock switch
{
    TextBlock text => $"Text: {text.Text}",
    ImageBlock image => $"Image: {image.Url}",
    ToolUseBlock { Name: var name, Input: var input } => $"Tool: {name}",
    _ => throw new InvalidOperationException("Unknown content block type")
};

// Pattern matching with type patterns
if (contentBlock is TextBlock { Text: var text })
{
    Console.WriteLine($"Got text: {text}");
}
```

### Custom Configuration

```csharp
var client = new ServiceNameClient(new ClientOptions
{
    ApiKey = "your-api-key",
    BaseUrl = "https://custom.api.com",
    Timeout = TimeSpan.FromMinutes(2),
    MaxRetries = 3,
    DefaultHeaders = new Dictionary<string, string>
    {
        ["X-Custom-Header"] = "value"
    }
});
```

### Dependency Injection (ASP.NET Core)

```csharp
// In Program.cs or Startup.cs
builder.Services.AddSingleton<ServiceNameClient>(sp =>
{
    var config = sp.GetRequiredService<IConfiguration>();
    return new ServiceNameClient(new ClientOptions
    {
        ApiKey = config["ServiceName:ApiKey"]
    });
});

// In a controller or service
public class MyService
{
    private readonly ServiceNameClient _client;

    public MyService(ServiceNameClient client)
    {
        _client = client;
    }

    public async Task<string> ProcessAsync()
    {
        var response = await _client.Completions.CreateAsync(request);
        return response.Content;
    }
}
```

### HttpClient Factory Integration

```csharp
// In Program.cs
builder.Services.AddHttpClient<ServiceNameClient>(client =>
{
    client.Timeout = TimeSpan.FromMinutes(2);
    client.DefaultRequestHeaders.Add("X-Custom", "value");
});

// Then use via DI
public class MyService
{
    private readonly ServiceNameClient _client;

    public MyService(ServiceNameClient client) => _client = client;
}
```

### Cancellation

```csharp
using var cts = new CancellationTokenSource(TimeSpan.FromSeconds(30));

try
{
    var response = await client.Completions.CreateAsync(request, cts.Token);
}
catch (CancelledException)
{
    Console.WriteLine("Request was cancelled");
}
```

### MAUI/Blazor Integration

```csharp
// In MauiProgram.cs
builder.Services.AddSingleton(new ServiceNameClient(new ClientOptions
{
    ApiKey = SecureStorage.GetAsync("api_key").Result
}));

// In a ViewModel
public partial class ChatViewModel : ObservableObject
{
    private readonly ServiceNameClient _client;

    [ObservableProperty]
    private string _response = "";

    public ChatViewModel(ServiceNameClient client)
    {
        _client = client;
    }

    [RelayCommand]
    private async Task SendMessageAsync(string message)
    {
        await foreach (var chunk in _client.Completions.CreateStreamAsync(
            new CreateRequest
            {
                Model = "model-name",
                Messages = new[] { new Message { Role = Role.User, Content = message } }
            }))
        {
            Response += chunk.Delta?.Text ?? "";
        }
    }
}
```

## Testing

### Generator Tests

```go
func TestGenerate_NilService(t *testing.T)
func TestGenerate_ValidCSharp_Compiles(t *testing.T)
func TestGenerate_ProducesExpectedFiles(t *testing.T)
func TestGenerate_TypeMapping(t *testing.T)
func TestGenerate_StreamingMethods(t *testing.T)
func TestGenerate_PolymorphicTypes(t *testing.T)
```

### Generated SDK Tests

The generated SDK is compatible with standard .NET testing frameworks:

```csharp
public class ClientTests
{
    [Fact]
    public async Task CreateAsync_ReturnsResponse()
    {
        // Use MockHttpMessageHandler or similar
        var handler = new MockHttpMessageHandler();
        handler.When("*").Respond("application/json", "{}");

        var httpClient = new HttpClient(handler);
        var client = new ServiceNameClient(new ClientOptions
        {
            HttpClient = httpClient,
            ApiKey = "test-key"
        });

        var response = await client.Completions.CreateAsync(request);
        Assert.NotNull(response);
    }
}
```

## Platform Support

### Dependencies

**Core Dependencies (built-in):**
- `System.Net.Http` - HTTP client
- `System.Text.Json` - JSON serialization

**No external NuGet packages required.**

### Target Frameworks

| Framework     | Minimum Version | Rationale                               |
|---------------|-----------------|----------------------------------------|
| .NET          | 8.0             | LTS, records, required members, NRT    |
| .NET Standard | 2.1             | For library compatibility (optional)    |

### Project File

```xml
<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
    <Nullable>enable</Nullable>
    <ImplicitUsings>enable</ImplicitUsings>
    <LangVersion>latest</LangVersion>

    <PackageId>{PackageName}</PackageId>
    <Version>{Version}</Version>
    <Description>{Description}</Description>
    <Authors>Generated</Authors>
  </PropertyGroup>
</Project>
```

## Migration Path

For projects migrating from other HTTP clients:

1. Replace existing HttpClient calls with SDK client methods
2. Update request/response types to use generated records
3. Replace manual JSON handling with typed models
4. Update error handling to use exception hierarchy
5. Replace callback patterns with async/await

## Future Enhancements

1. **Source generators**: System.Text.Json source generation for AOT
2. **Polly integration**: Built-in resilience policies
3. **Metrics**: OpenTelemetry integration for observability
4. **Caching**: Response caching with configurable strategies
5. **gRPC support**: Bi-directional streaming for gRPC contracts
6. **Mock client**: Built-in mock client for testing

## References

- [C# Coding Conventions](https://learn.microsoft.com/en-us/dotnet/csharp/fundamentals/coding-style/coding-conventions)
- [.NET API Design Guidelines](https://learn.microsoft.com/en-us/dotnet/standard/design-guidelines/)
- [System.Text.Json Overview](https://learn.microsoft.com/en-us/dotnet/standard/serialization/system-text-json/overview)
- [IAsyncEnumerable](https://learn.microsoft.com/en-us/dotnet/csharp/whats-new/csharp-8#asynchronous-streams)
- [Nullable Reference Types](https://learn.microsoft.com/en-us/dotnet/csharp/nullable-references)
- [Records](https://learn.microsoft.com/en-us/dotnet/csharp/language-reference/builtin-types/record)
- [Pattern Matching](https://learn.microsoft.com/en-us/dotnet/csharp/fundamentals/functional/pattern-matching)
