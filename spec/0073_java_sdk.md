# RFC 0073: Java SDK Generator

## Summary

Add Java SDK code generation to the Mizu contract system, enabling production-ready, type-safe Java clients for JVM applications, Android, and enterprise systems.

## Motivation

Java remains one of the most widely used languages in enterprise software development. A native Java SDK provides:

1. **Enterprise readiness**: Full compatibility with existing Java/Spring ecosystems
2. **Type safety**: Compile-time guarantees through Java's strong type system
3. **Modern async**: `CompletableFuture<T>` for non-blocking operations
4. **Reactive streaming**: `Iterator<T>` and `Stream<T>` for SSE consumption
5. **Builder pattern**: Fluent API for constructing complex requests
6. **Records support**: Immutable data classes (Java 17+) with fallback to POJOs
7. **Android compatibility**: Works on Android API 26+ with desugaring

## Design Goals

### Developer Experience (DX)

- **Idiomatic Java**: Follow Java naming conventions and design patterns
- **Fluent builders**: Builder pattern for all configuration and request types
- **CompletableFuture**: All async operations return CompletableFuture for composition
- **Optional<T>**: Proper nullable handling with Optional where appropriate
- **Comprehensive Javadoc**: Generated documentation for all public APIs
- **Minimal dependencies**: Jackson for JSON, java.net.http.HttpClient (Java 11+)

### Production Readiness

- **Retry logic**: Configurable retry with exponential backoff and jitter
- **Timeout handling**: Connection, read, and request-level timeouts
- **Thread safety**: Immutable configuration, thread-safe client
- **Error handling**: Rich exception hierarchy with typed error responses
- **Connection pooling**: Efficient HTTP connection management
- **Logging**: SLF4J integration for debugging
- **Metrics ready**: Hooks for observability integration

## Architecture

### Package Structure

```
{groupId}.{artifactId}/
├── pom.xml                              # Maven build configuration
└── src/main/java/{package}/
    ├── {ServiceName}Client.java         # Main client class
    ├── ClientOptions.java               # Configuration options
    ├── model/                           # Generated model types
    │   └── *.java
    ├── resource/                        # Resource classes
    │   └── *Resource.java
    ├── exception/                       # Exception hierarchy
    │   ├── SDKException.java
    │   ├── ApiException.java
    │   ├── ConnectionException.java
    │   └── *.java
    └── internal/                        # Internal implementation
        ├── HttpClientWrapper.java
        └── SSEReader.java
```

### Core Components

#### 1. Client (`{ServiceName}Client.java`)

The main entry point for API interactions:

```java
/**
 * Configuration options for the SDK client.
 */
public final class ClientOptions {
    private final String apiKey;
    private final String baseUrl;
    private final Duration timeout;
    private final Duration connectTimeout;
    private final int maxRetries;
    private final Map<String, String> defaultHeaders;
    private final AuthMode authMode;
    private final HttpClient httpClient;

    private ClientOptions(Builder builder) { /* ... */ }

    public static Builder builder() {
        return new Builder();
    }

    public static final class Builder {
        private String apiKey;
        private String baseUrl = "{default_base_url}";
        private Duration timeout = Duration.ofSeconds(60);
        private Duration connectTimeout = Duration.ofSeconds(10);
        private int maxRetries = 2;
        private Map<String, String> defaultHeaders = new HashMap<>();
        private AuthMode authMode = AuthMode.BEARER;
        private HttpClient httpClient;

        public Builder apiKey(String apiKey) { /* ... */ }
        public Builder baseUrl(String baseUrl) { /* ... */ }
        public Builder timeout(Duration timeout) { /* ... */ }
        public Builder connectTimeout(Duration connectTimeout) { /* ... */ }
        public Builder maxRetries(int maxRetries) { /* ... */ }
        public Builder addHeader(String name, String value) { /* ... */ }
        public Builder defaultHeaders(Map<String, String> headers) { /* ... */ }
        public Builder authMode(AuthMode authMode) { /* ... */ }
        public Builder httpClient(HttpClient httpClient) { /* ... */ }
        public ClientOptions build() { /* ... */ }
    }

    // Getters...
}

/**
 * Authentication mode for API requests.
 */
public enum AuthMode {
    BEARER,
    BASIC,
    NONE
}

/**
 * The main SDK client providing access to all API resources.
 *
 * <p>This client is thread-safe and can be shared across threads.
 * It is recommended to create a single instance and reuse it.
 *
 * <p>Example usage:
 * <pre>{@code
 * var client = {ServiceName}Client.builder()
 *     .apiKey("your-api-key")
 *     .build();
 *
 * var response = client.{resource}().{method}(request).join();
 * }</pre>
 */
public final class {ServiceName}Client implements AutoCloseable {
    private final ClientOptions options;
    private final HttpClientWrapper httpClient;

    private {ServiceName}Client(ClientOptions options) { /* ... */ }

    public static Builder builder() {
        return new Builder();
    }

    /** Access to {resource} operations. */
    public {Resource}Resource {resource}() { /* ... */ }

    @Override
    public void close() { /* ... */ }

    public static final class Builder {
        private String apiKey;
        // Other options with defaults...

        public Builder apiKey(String apiKey) { /* ... */ }
        // Other builder methods...

        public {ServiceName}Client build() {
            return new {ServiceName}Client(
                ClientOptions.builder()
                    .apiKey(apiKey)
                    // ... other options
                    .build()
            );
        }
    }
}
```

#### 2. Model Types (`model/*.java`)

All model types use immutable classes with builders:

```java
/**
 * Request/response model.
 *
 * @param fieldName Description from contract
 * @param optionalField Optional field
 */
public record {TypeName}(
    @JsonProperty("field_name") String fieldName,
    @JsonProperty("optional_field") @Nullable String optionalField
) {
    public static Builder builder() {
        return new Builder();
    }

    public static final class Builder {
        private String fieldName;
        private String optionalField;

        public Builder fieldName(String fieldName) {
            this.fieldName = fieldName;
            return this;
        }

        public Builder optionalField(String optionalField) {
            this.optionalField = optionalField;
            return this;
        }

        public {TypeName} build() {
            return new {TypeName}(fieldName, optionalField);
        }
    }
}

// For Java 11 compatibility, use POJO with builder:
public final class {TypeName} {
    private final String fieldName;
    private final String optionalField;

    private {TypeName}(Builder builder) {
        this.fieldName = builder.fieldName;
        this.optionalField = builder.optionalField;
    }

    @JsonProperty("field_name")
    public String getFieldName() { return fieldName; }

    @JsonProperty("optional_field")
    @Nullable
    public String getOptionalField() { return optionalField; }

    public static Builder builder() { return new Builder(); }

    public Builder toBuilder() {
        return new Builder()
            .fieldName(this.fieldName)
            .optionalField(this.optionalField);
    }

    public static final class Builder { /* ... */ }

    @Override
    public boolean equals(Object o) { /* ... */ }

    @Override
    public int hashCode() { /* ... */ }

    @Override
    public String toString() { /* ... */ }
}
```

**Enum types with JSON serialization:**

```java
/**
 * Valid values for role field.
 */
public enum Role {
    @JsonProperty("user") USER,
    @JsonProperty("assistant") ASSISTANT,
    @JsonProperty("system") SYSTEM;
}
```

**Discriminated unions using sealed interfaces (Java 17+) or abstract class:**

```java
/**
 * Discriminated union (tag: "type").
 */
@JsonTypeInfo(use = JsonTypeInfo.Id.NAME, property = "type")
@JsonSubTypes({
    @JsonSubTypes.Type(value = ContentBlock.Text.class, name = "text"),
    @JsonSubTypes.Type(value = ContentBlock.Image.class, name = "image"),
    @JsonSubTypes.Type(value = ContentBlock.ToolUse.class, name = "tool_use")
})
public sealed interface ContentBlock permits ContentBlock.Text, ContentBlock.Image, ContentBlock.ToolUse {

    /** Returns the discriminator value. */
    String type();

    /** Returns true if this is a Text variant. */
    default boolean isText() { return this instanceof Text; }

    /** Returns the Text value if present, empty otherwise. */
    default Optional<Text> asText() {
        return this instanceof Text t ? Optional.of(t) : Optional.empty();
    }

    record Text(
        @JsonProperty("type") String type,
        @JsonProperty("text") String text
    ) implements ContentBlock {
        public Text { type = "text"; }
        public Text(String text) { this("text", text); }
    }

    record Image(
        @JsonProperty("type") String type,
        @JsonProperty("url") String url
    ) implements ContentBlock {
        public Image { type = "image"; }
        public Image(String url) { this("image", url); }
    }

    record ToolUse(
        @JsonProperty("type") String type,
        @JsonProperty("id") String id,
        @JsonProperty("name") String name,
        @JsonProperty("input") JsonNode input
    ) implements ContentBlock {
        public ToolUse { type = "tool_use"; }
    }
}

// For Java 11 compatibility, use abstract class:
@JsonTypeInfo(use = JsonTypeInfo.Id.NAME, property = "type")
@JsonSubTypes({
    @JsonSubTypes.Type(value = ContentBlock.Text.class, name = "text"),
    @JsonSubTypes.Type(value = ContentBlock.Image.class, name = "image")
})
public abstract class ContentBlock {
    public abstract String getType();

    public boolean isText() { return this instanceof Text; }
    public Optional<Text> asText() {
        return this instanceof Text ? Optional.of((Text) this) : Optional.empty();
    }

    public static final class Text extends ContentBlock { /* ... */ }
    public static final class Image extends ContentBlock { /* ... */ }
}
```

#### 3. Resources (`resource/*Resource.java`)

Resource classes provide namespaced method access:

```java
/**
 * Operations for {resource}.
 */
public final class {Resource}Resource {
    private final HttpClientWrapper client;

    {Resource}Resource(HttpClientWrapper client) {
        this.client = client;
    }

    /**
     * Description from contract.
     *
     * @param request the request parameters
     * @return a CompletableFuture containing the response
     * @throws ApiException if the API returns an error
     * @throws ConnectionException if a network error occurs
     */
    public CompletableFuture<ResponseType> methodName(RequestType request) {
        return client.requestAsync(
            "POST",
            "/path",
            request,
            ResponseType.class
        );
    }

    /**
     * Synchronous version of {@link #methodName(RequestType)}.
     */
    public ResponseType methodNameSync(RequestType request) {
        return methodName(request).join();
    }

    /**
     * Streaming method returning an Iterator.
     *
     * @param request the request parameters
     * @return an Iterator over streamed items
     */
    public Iterator<ItemType> streamMethod(RequestType request) {
        return client.stream(
            "POST",
            "/path",
            request,
            ItemType.class
        );
    }

    /**
     * Streaming method returning a Stream (closes when terminal operation completes).
     *
     * @param request the request parameters
     * @return a Stream over streamed items
     */
    public Stream<ItemType> streamMethodStream(RequestType request) {
        Iterator<ItemType> iterator = streamMethod(request);
        return StreamSupport.stream(
            Spliterators.spliteratorUnknownSize(iterator, Spliterator.ORDERED),
            false
        ).onClose(() -> {
            if (iterator instanceof AutoCloseable) {
                try { ((AutoCloseable) iterator).close(); } catch (Exception ignored) {}
            }
        });
    }
}
```

#### 4. Streaming (`internal/SSEReader.java`)

SSE streaming support via Iterator:

```java
/**
 * Reads Server-Sent Events from an HTTP response.
 *
 * <p>This class implements both Iterator and AutoCloseable. The underlying
 * connection is closed when the iterator is exhausted or when close() is called.
 */
public final class SSEReader<T> implements Iterator<T>, AutoCloseable {
    private final BufferedReader reader;
    private final ObjectMapper mapper;
    private final Class<T> itemType;
    private T next;
    private boolean done;

    public SSEReader(InputStream inputStream, ObjectMapper mapper, Class<T> itemType) {
        this.reader = new BufferedReader(new InputStreamReader(inputStream, StandardCharsets.UTF_8));
        this.mapper = mapper;
        this.itemType = itemType;
    }

    @Override
    public boolean hasNext() {
        if (done) return false;
        if (next != null) return true;

        try {
            next = readNext();
            return next != null;
        } catch (IOException e) {
            throw new UncheckedIOException(e);
        }
    }

    @Override
    public T next() {
        if (!hasNext()) {
            throw new NoSuchElementException();
        }
        T result = next;
        next = null;
        return result;
    }

    private T readNext() throws IOException {
        StringBuilder data = new StringBuilder();
        String line;

        while ((line = reader.readLine()) != null) {
            if (line.isEmpty()) {
                // End of event
                String content = data.toString().trim();
                data.setLength(0);

                if (content.isEmpty() || content.equals("[DONE]")) {
                    continue;
                }

                return mapper.readValue(content, itemType);
            } else if (line.startsWith("data:")) {
                String content = line.substring(5).stripLeading();
                if (data.length() > 0) data.append("\n");
                data.append(content);
            }
            // Ignore other SSE fields (event:, id:, retry:)
        }

        done = true;
        return null;
    }

    @Override
    public void close() {
        done = true;
        try {
            reader.close();
        } catch (IOException ignored) {}
    }
}
```

#### 5. Exceptions (`exception/*.java`)

Exception hierarchy for typed error handling:

```java
/**
 * Base exception for all SDK errors.
 */
public class SDKException extends RuntimeException {
    public SDKException(String message) {
        super(message);
    }

    public SDKException(String message, Throwable cause) {
        super(message, cause);
    }
}

/**
 * Thrown when the API returns an error response (4xx/5xx).
 */
public class ApiException extends SDKException {
    private final int statusCode;
    private final String body;
    private final Map<String, List<String>> headers;

    public ApiException(int statusCode, String message, String body, Map<String, List<String>> headers) {
        super(message);
        this.statusCode = statusCode;
        this.body = body;
        this.headers = headers;
    }

    public int getStatusCode() { return statusCode; }
    public String getBody() { return body; }
    public Map<String, List<String>> getHeaders() { return headers; }

    public boolean isClientError() { return statusCode >= 400 && statusCode < 500; }
    public boolean isServerError() { return statusCode >= 500; }
    public boolean isRateLimitError() { return statusCode == 429; }
    public boolean isAuthenticationError() { return statusCode == 401; }
    public boolean isNotFoundError() { return statusCode == 404; }

    /**
     * Attempts to decode the error body as a typed error response.
     */
    public <T> Optional<T> decodeAs(Class<T> type, ObjectMapper mapper) {
        if (body == null || body.isEmpty()) {
            return Optional.empty();
        }
        try {
            return Optional.of(mapper.readValue(body, type));
        } catch (Exception e) {
            return Optional.empty();
        }
    }
}

/**
 * Thrown when a network connection error occurs.
 */
public class ConnectionException extends SDKException {
    public ConnectionException(String message, Throwable cause) {
        super(message, cause);
    }
}

/**
 * Thrown when a request times out.
 */
public class TimeoutException extends SDKException {
    public TimeoutException(String message) {
        super(message);
    }

    public TimeoutException(String message, Throwable cause) {
        super(message, cause);
    }
}

/**
 * Thrown when request encoding fails.
 */
public class EncodingException extends SDKException {
    public EncodingException(String message, Throwable cause) {
        super(message, cause);
    }
}

/**
 * Thrown when response decoding fails.
 */
public class DecodingException extends SDKException {
    public DecodingException(String message, Throwable cause) {
        super(message, cause);
    }
}
```

## Type Mapping

### Primitive Types

| Contract Type     | Java Type         |
|-------------------|-------------------|
| `string`          | `String`          |
| `bool`, `boolean` | `boolean`/`Boolean` |
| `int`             | `int`/`Integer`   |
| `int8`            | `byte`/`Byte`     |
| `int16`           | `short`/`Short`   |
| `int32`           | `int`/`Integer`   |
| `int64`           | `long`/`Long`     |
| `uint`            | `int`/`Integer`   |
| `uint8`           | `short`/`Short`   |
| `uint16`          | `int`/`Integer`   |
| `uint32`          | `long`/`Long`     |
| `uint64`          | `long`/`Long`     |
| `float32`         | `float`/`Float`   |
| `float64`         | `double`/`Double` |
| `time.Time`       | `Instant`         |
| `json.RawMessage` | `JsonNode`        |
| `any`             | `JsonNode`        |

### Collection Types

| Contract Type      | Java Type             |
|--------------------|-----------------------|
| `[]T`              | `List<JavaType>`      |
| `map[string]T`     | `Map<String, JavaType>` |

### Optional/Nullable

| Contract      | Java Type              |
|---------------|------------------------|
| `optional: T` | `@Nullable T` (null default) |
| `nullable: T` | `@Nullable T`          |

### Struct Fields

Fields with `optional: true` or `nullable: true` become nullable with Jackson annotations:

```java
public final class Request {
    private final String required;
    @Nullable
    private final String optionalField;

    // Constructor, getters, builder...
}
```

### Enum/Const Values

Fields with `enum` constraint generate Java enums:

```java
public enum Role {
    @JsonProperty("user") USER,
    @JsonProperty("assistant") ASSISTANT,
    @JsonProperty("system") SYSTEM;
}
```

### Discriminated Unions

Union types use Jackson's polymorphic type handling:

```java
@JsonTypeInfo(use = JsonTypeInfo.Id.NAME, property = "type")
@JsonSubTypes({
    @JsonSubTypes.Type(value = ContentBlock.Text.class, name = "text"),
    @JsonSubTypes.Type(value = ContentBlock.Image.class, name = "image")
})
public abstract class ContentBlock {
    // ...
}
```

## HTTP Client Implementation

### Request Flow

```java
final class HttpClientWrapper {
    private final ClientOptions options;
    private final HttpClient client;
    private final ObjectMapper mapper;

    HttpClientWrapper(ClientOptions options) {
        this.options = options;
        this.mapper = createObjectMapper();
        this.client = options.getHttpClient() != null
            ? options.getHttpClient()
            : HttpClient.newBuilder()
                .connectTimeout(options.getConnectTimeout())
                .build();
    }

    private ObjectMapper createObjectMapper() {
        return new ObjectMapper()
            .registerModule(new JavaTimeModule())
            .setSerializationInclusion(JsonInclude.Include.NON_NULL)
            .configure(DeserializationFeature.FAIL_ON_UNKNOWN_PROPERTIES, false);
    }

    <T> CompletableFuture<T> requestAsync(
            String method,
            String path,
            Object body,
            Class<T> responseType) {
        return CompletableFuture.supplyAsync(() -> {
            Exception lastException = null;

            for (int attempt = 0; attempt <= options.getMaxRetries(); attempt++) {
                try {
                    HttpRequest request = buildRequest(method, path, body);
                    HttpResponse<String> response = client.send(
                        request,
                        HttpResponse.BodyHandlers.ofString()
                    );

                    if (response.statusCode() >= 400) {
                        throw parseError(response);
                    }

                    if (responseType == Void.class) {
                        return null;
                    }

                    return mapper.readValue(response.body(), responseType);
                } catch (ApiException e) {
                    throw e;
                } catch (Exception e) {
                    lastException = e;
                    if (attempt < options.getMaxRetries()) {
                        sleep(backoff(attempt));
                    }
                }
            }

            throw new ConnectionException("Request failed after retries", lastException);
        });
    }

    private HttpRequest buildRequest(String method, String path, Object body) {
        HttpRequest.Builder builder = HttpRequest.newBuilder()
            .uri(URI.create(options.getBaseUrl() + path))
            .timeout(options.getTimeout());

        // Apply default headers
        options.getDefaultHeaders().forEach(builder::header);

        // Apply authentication
        if (options.getApiKey() != null) {
            switch (options.getAuthMode()) {
                case BEARER -> builder.header("Authorization", "Bearer " + options.getApiKey());
                case BASIC -> builder.header("Authorization", "Basic " + options.getApiKey());
                case NONE -> {}
            }
        }

        // Set body
        if (body != null) {
            try {
                String json = mapper.writeValueAsString(body);
                builder.header("Content-Type", "application/json");
                builder.method(method, HttpRequest.BodyPublishers.ofString(json));
            } catch (Exception e) {
                throw new EncodingException("Failed to encode request body", e);
            }
        } else {
            builder.method(method, HttpRequest.BodyPublishers.noBody());
        }

        return builder.build();
    }

    private long backoff(int attempt) {
        long baseDelay = 500; // 0.5 seconds
        long delay = baseDelay * (1L << attempt);
        // Add jitter (0-25%)
        delay += (long) (delay * 0.25 * Math.random());
        return Math.min(delay, 30000); // Cap at 30 seconds
    }
}
```

### SSE Streaming

```java
<T> Iterator<T> stream(String method, String path, Object body, Class<T> itemType) {
    HttpRequest request = buildRequest(method, path, body);
    request = request.newBuilder()
        .header("Accept", "text/event-stream")
        .build();

    try {
        HttpResponse<InputStream> response = client.send(
            request,
            HttpResponse.BodyHandlers.ofInputStream()
        );

        if (response.statusCode() >= 400) {
            String errorBody = new String(response.body().readAllBytes(), StandardCharsets.UTF_8);
            throw parseError(response.statusCode(), errorBody, response.headers().map());
        }

        return new SSEReader<>(response.body(), mapper, itemType);
    } catch (IOException | InterruptedException e) {
        throw new ConnectionException("Stream connection failed", e);
    }
}
```

## Configuration

### Default Values

From contract `Client`:

```java
public static ClientOptions defaultOptions() {
    return ClientOptions.builder()
        .baseUrl("{client.baseURL}")
        .timeout(Duration.ofSeconds(60))
        .connectTimeout(Duration.ofSeconds(10))
        .maxRetries(2)
        .defaultHeaders(Map.of(
            // From client.headers
        ))
        .authMode(AuthMode.BEARER)
        .build();
}
```

### Environment Variables

The SDK does NOT automatically read environment variables for API keys. This is intentional for security and explicitness:

```java
var client = ServiceNameClient.builder()
    .apiKey(System.getenv("SERVICE_API_KEY"))
    .build();
```

## Naming Conventions

### Java Naming

| Contract       | Java                    |
|----------------|-------------------------|
| `user-id`      | `userId`                |
| `user_name`    | `userName`              |
| `UserData`     | `UserData`              |
| `create`       | `create`                |
| `get-user`     | `getUser`               |
| `GET`          | `get`                   |

Functions:
- `toJavaName(s)`: Converts to lowerCamelCase (for methods/fields)
- `toJavaTypeName(s)`: Converts to UpperCamelCase (for types)
- `toJavaConstant(s)`: Converts to SCREAMING_SNAKE_CASE (for enum values)

Special handling:
- Reserved words: Prefixed with underscore (`_class`, `_interface`)
- Acronyms: Properly cased (`userId`, not `userID`)

## Code Generation

### Generator Structure

```go
package sdkjava

type Config struct {
    // Package is the Java package name.
    // Default: "com.example.{servicename}".
    Package string

    // GroupId is the Maven group ID.
    // Default: "com.example".
    GroupId string

    // ArtifactId is the Maven artifact ID.
    // Default: "{servicename}-sdk".
    ArtifactId string

    // Version is the package version.
    // Default: "0.0.0".
    Version string

    // JavaVersion is the target Java version.
    // Default: 11.
    JavaVersion int

    // UseRecords uses Java records instead of classes (requires Java 17+).
    // Default: false.
    UseRecords bool
}

func Generate(svc *contract.Service, cfg *Config) ([]*sdk.File, error)
```

### Template Files

```
templates/
├── pom.xml.tmpl              # Maven build configuration
├── Client.java.tmpl          # Main client class
├── ClientOptions.java.tmpl   # Configuration options
├── Types.java.tmpl           # Model types
├── Enums.java.tmpl           # Enum types
├── Resources.java.tmpl       # Resource classes
├── HttpClient.java.tmpl      # HTTP client wrapper
├── SSEReader.java.tmpl       # SSE streaming support
└── Exceptions.java.tmpl      # Exception types
```

### Generated Files

| File                       | Purpose                           |
|----------------------------|-----------------------------------|
| `pom.xml`                  | Maven build configuration         |
| `{Service}Client.java`     | Main client class                 |
| `ClientOptions.java`       | Configuration options             |
| `model/*.java`             | Model type definitions            |
| `resource/*Resource.java`  | Resource classes with methods     |
| `exception/*.java`         | Exception type definitions        |
| `internal/*.java`          | Internal implementation details   |

## Usage Examples

### Basic Usage

```java
import com.example.servicesdk.*;
import com.example.servicesdk.model.*;

// Create client
var client = ServiceNameClient.builder()
    .apiKey("your-api-key")
    .build();

// Make a request (async)
client.completions().create(
    CreateRequest.builder()
        .model("model-name")
        .messages(List.of(
            Message.builder()
                .role(Role.USER)
                .content("Hello")
                .build()
        ))
        .build()
).thenAccept(response -> {
    System.out.println(response.getContent());
});

// Make a request (sync)
var response = client.completions().createSync(
    CreateRequest.builder()
        .model("model-name")
        .messages(List.of(
            Message.builder()
                .role(Role.USER)
                .content("Hello")
                .build()
        ))
        .build()
);
```

### Streaming

```java
// Using Iterator
var iterator = client.completions().createStream(
    CreateRequest.builder()
        .model("model-name")
        .messages(List.of(
            Message.builder()
                .role(Role.USER)
                .content("Hello")
                .build()
        ))
        .build()
);

while (iterator.hasNext()) {
    var event = iterator.next();
    System.out.print(event.getDelta().getText());
}

// Using Stream API
client.completions().createStreamStream(request)
    .forEach(event -> System.out.print(event.getDelta().getText()));

// With try-with-resources for cleanup
try (var stream = client.completions().createStreamStream(request)) {
    stream.forEach(event -> System.out.print(event.getDelta().getText()));
}
```

### Error Handling

```java
try {
    var response = client.completions().createSync(request);
} catch (ApiException e) {
    if (e.isRateLimitError()) {
        System.out.println("Rate limited, retry after: " +
            e.getHeaders().get("Retry-After"));
    } else if (e.isAuthenticationError()) {
        System.out.println("Invalid API key");
    } else {
        System.out.println("API Error " + e.getStatusCode() + ": " + e.getMessage());
    }
} catch (TimeoutException e) {
    System.out.println("Request timed out");
} catch (ConnectionException e) {
    System.out.println("Network error: " + e.getMessage());
} catch (SDKException e) {
    System.out.println("SDK error: " + e.getMessage());
}
```

### Async Composition

```java
client.completions().create(request1)
    .thenCompose(response -> {
        // Chain another request based on first response
        return client.completions().create(
            CreateRequest.builder()
                .model("model-name")
                .messages(List.of(
                    Message.builder()
                        .role(Role.USER)
                        .content("Follow up: " + response.getContent())
                        .build()
                ))
                .build()
        );
    })
    .thenAccept(response -> {
        System.out.println(response.getContent());
    })
    .exceptionally(e -> {
        System.err.println("Error: " + e.getMessage());
        return null;
    });
```

### Custom Configuration

```java
var client = ServiceNameClient.builder()
    .apiKey("your-api-key")
    .baseUrl("https://custom.api.com")
    .timeout(Duration.ofSeconds(120))
    .connectTimeout(Duration.ofSeconds(30))
    .maxRetries(3)
    .addHeader("X-Custom-Header", "value")
    .build();
```

### Custom HTTP Client

```java
var httpClient = HttpClient.newBuilder()
    .connectTimeout(Duration.ofSeconds(10))
    .proxy(ProxySelector.of(new InetSocketAddress("proxy.example.com", 8080)))
    .authenticator(new Authenticator() {
        @Override
        protected PasswordAuthentication getPasswordAuthentication() {
            return new PasswordAuthentication("user", "password".toCharArray());
        }
    })
    .build();

var client = ServiceNameClient.builder()
    .apiKey("your-api-key")
    .httpClient(httpClient)
    .build();
```

## Testing

### Generator Tests

```go
func TestGenerate_NilService(t *testing.T)
func TestGenerate_ValidJava_Compiles(t *testing.T)
func TestGenerate_ProducesExpectedFiles(t *testing.T)
func TestGenerate_TypeMapping(t *testing.T)
func TestGenerate_StreamingMethods(t *testing.T)
func TestGenerate_Enums(t *testing.T)
func TestGenerate_Unions(t *testing.T)
```

### Generated SDK Tests

The generated SDK includes a test directory structure:

```
src/test/java/{package}/
├── {Service}ClientTest.java
└── model/
    └── *Test.java
```

## Platform Support

### Dependencies

**Core Dependencies (pom.xml):**
```xml
<dependencies>
    <!-- JSON processing -->
    <dependency>
        <groupId>com.fasterxml.jackson.core</groupId>
        <artifactId>jackson-databind</artifactId>
        <version>2.17.0</version>
    </dependency>
    <dependency>
        <groupId>com.fasterxml.jackson.datatype</groupId>
        <artifactId>jackson-datatype-jsr310</artifactId>
        <version>2.17.0</version>
    </dependency>

    <!-- Annotations -->
    <dependency>
        <groupId>org.jetbrains</groupId>
        <artifactId>annotations</artifactId>
        <version>24.0.0</version>
        <scope>provided</scope>
    </dependency>
</dependencies>
```

### Minimum Versions

| Platform    | Minimum Version | Rationale                           |
|-------------|-----------------|-------------------------------------|
| Java        | 11              | HttpClient, var, modern features    |
| Android SDK | 26 (8.0)        | Java 8 APIs, with desugaring: 21    |

### Java 17+ Features

When `UseRecords: true` is configured:
- Uses `record` instead of class for model types
- Uses `sealed interface` for unions
- Uses pattern matching in type checks

## Migration Path

For projects migrating from other HTTP clients:

1. Replace existing client initialization with SDK client builder
2. Update method calls to use generated resource methods
3. Replace custom model types with generated types
4. Update error handling to use exception hierarchy
5. Replace callback patterns with CompletableFuture

## Future Enhancements

1. **Virtual threads**: Java 21+ virtual thread support for blocking operations
2. **GraalVM native image**: Configuration for native compilation
3. **Reactive Streams**: Publisher/Subscriber support for streaming
4. **Request interceptors**: Middleware for custom request/response handling
5. **Response caching**: Built-in response caching
6. **Metrics**: Micrometer integration for observability
7. **Retry policies**: Pluggable retry strategies
8. **Circuit breaker**: Resilience4j integration

## References

- [Java Naming Conventions](https://www.oracle.com/java/technologies/javase/codeconventions-namingconventions.html)
- [Jackson JSON](https://github.com/FasterXML/jackson)
- [Java HttpClient](https://docs.oracle.com/en/java/javase/11/docs/api/java.net.http/java/net/http/HttpClient.html)
- [CompletableFuture](https://docs.oracle.com/en/java/javase/11/docs/api/java.base/java/util/concurrent/CompletableFuture.html)
- [Server-Sent Events](https://html.spec.whatwg.org/multipage/server-sent-events.html)
