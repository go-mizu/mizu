# RFC 0065: Kotlin SDK Generator

## Summary

Add Kotlin SDK code generation to the Mizu contract system, enabling production-ready, type-safe Kotlin clients for Android, JVM, and Kotlin Multiplatform (KMP) applications.

## Motivation

Kotlin is the primary language for Android development and increasingly popular for server-side JVM applications. A native Kotlin SDK provides:

1. **Coroutine-based async**: Native suspend functions and Flow for streaming
2. **Type safety**: Compile-time guarantees through Kotlin's expressive type system
3. **Null safety**: Explicit nullability with `?` operator
4. **Data classes**: Concise, immutable data models with automatic `equals`, `hashCode`, `copy`
5. **Sealed classes**: Elegant discriminated union support
6. **Java interop**: Full compatibility with existing Java codebases

## Design Goals

### Developer Experience (DX)

- **Idiomatic Kotlin**: Follow Kotlin coding conventions and naming standards
- **Suspend functions**: All network calls are suspend functions for structured concurrency
- **Flow for streaming**: Server-sent events via `Flow<T>` for reactive consumption
- **DSL builders**: Optional builder DSL for complex request construction
- **Comprehensive documentation**: KDoc-compatible documentation comments
- **Minimal dependencies**: Only kotlinx.serialization and ktor-client (or OkHttp)

### Production Readiness

- **Retry logic**: Configurable retry with exponential backoff
- **Timeout handling**: Per-request and global timeout configuration
- **Cancellation**: Full support for coroutine cancellation and cooperative cancellation
- **Error handling**: Sealed class hierarchy for typed error handling
- **Thread safety**: Safe for concurrent access from multiple coroutines
- **Logging**: Configurable logging interceptor for debugging

## Architecture

### Package Structure

```
{package}/
├── build.gradle.kts              # Gradle build configuration
└── src/main/kotlin/{package}/
    ├── Client.kt                 # Main client class
    ├── Types.kt                  # Generated model types
    ├── Resources.kt              # Resource namespaces
    ├── Streaming.kt              # SSE streaming support
    └── Errors.kt                 # Error types
```

### Core Components

#### 1. Client (`Client.kt`)

The main entry point for API interactions:

```kotlin
/**
 * Configuration options for the SDK client.
 */
data class ClientOptions(
    val apiKey: String? = null,
    val baseUrl: String = "{default_base_url}",
    val timeout: Duration = 60.seconds,
    val maxRetries: Int = 2,
    val defaultHeaders: Map<String, String> = emptyMap(),
    val authMode: AuthMode = AuthMode.BEARER,
    val httpClient: HttpClient? = null
)

/**
 * Authentication mode for API requests.
 */
enum class AuthMode {
    BEARER,
    BASIC,
    NONE
}

/**
 * The main SDK client providing access to all API resources.
 */
class {ServiceName}(
    private val options: ClientOptions = ClientOptions()
) {
    /** Access to {resource} operations. */
    val {resource}: {Resource}Resource = {Resource}Resource(httpClient)

    /**
     * Creates a new client with modified configuration.
     */
    fun with(
        apiKey: String? = options.apiKey,
        baseUrl: String = options.baseUrl,
        timeout: Duration = options.timeout,
        maxRetries: Int = options.maxRetries,
        defaultHeaders: Map<String, String> = options.defaultHeaders
    ): {ServiceName}
}
```

#### 2. Types (`Types.kt`)

All model types use Kotlin data classes with kotlinx.serialization:

```kotlin
/**
 * Request/response models
 */
@Serializable
data class {TypeName}(
    /** Description from contract */
    val fieldName: FieldType,

    /** Optional field */
    val optionalField: String? = null
)

/**
 * Enum types with serialized names
 */
@Serializable
enum class {EnumName} {
    @SerialName("value1") VALUE_1,
    @SerialName("value2") VALUE_2
}

/**
 * Discriminated unions using sealed classes
 */
@Serializable
sealed class {UnionName} {
    /** The discriminator tag value */
    abstract val type: String

    @Serializable
    @SerialName("variant1")
    data class Variant1(
        override val type: String = "variant1",
        // variant fields
    ) : {UnionName}()

    @Serializable
    @SerialName("variant2")
    data class Variant2(
        override val type: String = "variant2",
        // variant fields
    ) : {UnionName}()
}
```

#### 3. Resources (`Resources.kt`)

Resource classes provide namespaced method access:

```kotlin
/**
 * Operations for {resource}
 */
class {Resource}Resource internal constructor(
    private val client: HttpClientWrapper
) {
    /**
     * Description from contract
     */
    suspend fun methodName(request: RequestType): ResponseType

    /**
     * Streaming method returning a Flow
     */
    fun streamMethod(request: RequestType): Flow<ItemType>
}
```

#### 4. Streaming (`Streaming.kt`)

SSE streaming support via Kotlin Flow:

```kotlin
/**
 * Parses Server-Sent Events from a byte stream into a Flow.
 */
internal fun <T> parseSSEStream(
    inputStream: InputStream,
    decoder: Json,
    typeInfo: KType
): Flow<T> = flow {
    val reader = BufferedReader(InputStreamReader(inputStream))
    var buffer = StringBuilder()

    reader.forEachLine { line ->
        if (line.isEmpty()) {
            // Empty line = end of event
            val data = buffer.toString().trim()
            buffer = StringBuilder()

            if (data.isNotEmpty() && data != "[DONE]") {
                emit(decoder.decodeFromString(typeInfo, data))
            }
        } else if (line.startsWith("data:")) {
            val content = line.removePrefix("data:").trimStart()
            if (buffer.isNotEmpty()) buffer.append("\n")
            buffer.append(content)
        }
    }
}.flowOn(Dispatchers.IO)
```

#### 5. Errors (`Errors.kt`)

Sealed class hierarchy for typed errors:

```kotlin
/**
 * Base error type for all SDK errors.
 */
sealed class SDKException : Exception() {
    /** Network connection failed. */
    data class ConnectionError(
        override val cause: Throwable
    ) : SDKException()

    /** Server returned an error status code. */
    data class ApiError(
        val statusCode: Int,
        override val message: String,
        val body: String? = null
    ) : SDKException()

    /** Request timed out. */
    object Timeout : SDKException()

    /** Request was cancelled. */
    object Cancelled : SDKException()

    /** Failed to encode request body. */
    data class EncodingError(
        override val cause: Throwable
    ) : SDKException()

    /** Failed to decode response body. */
    data class DecodingError(
        override val cause: Throwable
    ) : SDKException()
}
```

## Type Mapping

### Primitive Types

| Contract Type     | Kotlin Type    |
|-------------------|----------------|
| `string`          | `String`       |
| `bool`, `boolean` | `Boolean`      |
| `int`             | `Int`          |
| `int8`            | `Byte`         |
| `int16`           | `Short`        |
| `int32`           | `Int`          |
| `int64`           | `Long`         |
| `uint`            | `UInt`         |
| `uint8`           | `UByte`        |
| `uint16`          | `UShort`       |
| `uint32`          | `UInt`         |
| `uint64`          | `ULong`        |
| `float32`         | `Float`        |
| `float64`         | `Double`       |
| `time.Time`       | `Instant`      |
| `json.RawMessage` | `JsonElement`  |
| `any`             | `JsonElement`  |

### Collection Types

| Contract Type      | Kotlin Type              |
|--------------------|--------------------------|
| `[]T`              | `List<KotlinType>`       |
| `map[string]T`     | `Map<String, KotlinType>`|

### Optional/Nullable

| Contract      | Kotlin Type    |
|---------------|----------------|
| `optional: T` | `T? = null`    |
| `nullable: T` | `T?`           |

### Struct Fields

Fields with `optional: true` or `nullable: true` become nullable with default null:

```kotlin
@Serializable
data class Request(
    val required: String,
    @SerialName("optional_field")
    val optionalField: String? = null
)
```

### Enum/Const Values

Fields with `enum` constraint generate sealed value classes or enums:

```kotlin
@Serializable
enum class Role {
    @SerialName("user") USER,
    @SerialName("assistant") ASSISTANT,
    @SerialName("system") SYSTEM
}
```

Fields with `const` constraint use a fixed value in data class.

### Discriminated Unions

Union types use Kotlin sealed classes with @Serializable polymorphism:

```kotlin
@Serializable
sealed class ContentBlock {
    abstract val type: String

    @Serializable
    @SerialName("text")
    data class Text(
        override val type: String = "text",
        val text: String
    ) : ContentBlock()

    @Serializable
    @SerialName("image")
    data class Image(
        override val type: String = "image",
        val url: String
    ) : ContentBlock()

    @Serializable
    @SerialName("tool_use")
    data class ToolUse(
        override val type: String = "tool_use",
        val id: String,
        val name: String,
        val input: JsonElement
    ) : ContentBlock()
}
```

## HTTP Client Implementation

### Request Flow

```kotlin
internal class HttpClientWrapper(
    private val options: ClientOptions
) {
    private val client: HttpClient = options.httpClient ?: HttpClient(CIO) {
        install(ContentNegotiation) {
            json(Json {
                ignoreUnknownKeys = true
                isLenient = true
            })
        }
        install(HttpTimeout) {
            requestTimeoutMillis = options.timeout.inWholeMilliseconds
        }
    }

    private val json = Json {
        ignoreUnknownKeys = true
        isLenient = true
    }

    suspend inline fun <reified T> request(
        method: HttpMethod,
        path: String,
        body: Any? = null
    ): T {
        var lastException: Exception? = null

        repeat(options.maxRetries + 1) { attempt ->
            try {
                val response = client.request {
                    this.method = method
                    url(options.baseUrl + path)

                    applyHeaders()
                    applyAuth()

                    body?.let {
                        contentType(ContentType.Application.Json)
                        setBody(it)
                    }
                }

                if (response.status.value >= 400) {
                    throw parseErrorResponse(response)
                }

                return response.body()
            } catch (e: SDKException) {
                throw e // Don't retry SDK errors
            } catch (e: Exception) {
                lastException = e
                if (attempt < options.maxRetries) {
                    delay(backoff(attempt))
                }
            }
        }

        throw SDKException.ConnectionError(
            lastException ?: Exception("Unknown error")
        )
    }

    private fun backoff(attempt: Int): Long {
        val baseDelay = 500L // 0.5 seconds
        return baseDelay * (1 shl attempt) // Exponential backoff
    }
}
```

### Authentication

```kotlin
private fun HttpRequestBuilder.applyAuth() {
    options.apiKey?.let { key ->
        when (options.authMode) {
            AuthMode.BEARER -> header("Authorization", "Bearer $key")
            AuthMode.BASIC -> header("Authorization", "Basic $key")
            AuthMode.NONE -> { }
        }
    }
}
```

### SSE Streaming

```kotlin
fun <T> stream(
    method: HttpMethod,
    path: String,
    body: Any? = null,
    typeInfo: TypeInfo
): Flow<T> = flow {
    val response = client.request {
        this.method = method
        url(options.baseUrl + path)
        header("Accept", "text/event-stream")

        applyHeaders()
        applyAuth()

        body?.let {
            contentType(ContentType.Application.Json)
            setBody(it)
        }
    }

    if (response.status.value >= 400) {
        throw parseErrorResponse(response)
    }

    val channel: ByteReadChannel = response.body()
    var buffer = StringBuilder()

    while (!channel.isClosedForRead) {
        val line = channel.readUTF8Line() ?: break

        if (line.isEmpty()) {
            val data = buffer.toString().trim()
            buffer = StringBuilder()

            if (data.isNotEmpty() && data != "[DONE]") {
                @Suppress("UNCHECKED_CAST")
                emit(json.decodeFromString(typeInfo.type, data) as T)
            }
        } else if (line.startsWith("data:")) {
            val content = line.removePrefix("data:").trimStart()
            if (buffer.isNotEmpty()) buffer.append("\n")
            buffer.append(content)
        }
    }
}.flowOn(Dispatchers.IO)
```

## Configuration

### Default Values

From contract `Defaults`:

```kotlin
companion object {
    val DEFAULT = ClientOptions(
        baseUrl = "{defaults.baseURL}",
        timeout = 60.seconds,
        maxRetries = 2,
        defaultHeaders = mapOf(
            // From defaults.headers
        )
    )
}
```

### Environment Variables

The SDK does NOT automatically read environment variables for API keys. This is intentional for security and explicitness:

```kotlin
val client = ServiceName(
    options = ClientOptions(
        apiKey = System.getenv("SERVICE_API_KEY")
    )
)
```

## Naming Conventions

### Kotlin Naming

| Contract       | Kotlin                   |
|----------------|--------------------------|
| `user-id`      | `userId`                 |
| `user_name`    | `userName`               |
| `UserData`     | `UserData`               |
| `create`       | `create`                 |
| `get-user`     | `getUser`                |

Functions:
- `toKotlinName(s)`: Converts to lowerCamelCase (for properties/methods)
- `toKotlinTypeName(s)`: Converts to UpperCamelCase (for types)
- `sanitizeIdent(s)`: Removes invalid characters

Special handling:
- Reserved words: Escaped with backticks (`` `class` ``, `` `object` ``, etc.)
- Acronyms: Preserved in proper case (`userId` not `userID`)

## Code Generation

### Generator Structure

```go
package sdkkotlin

type Config struct {
    // Package is the Kotlin package name.
    Package string

    // Version is the package version for build.gradle.kts.
    Version string

    // GroupId is the Maven group ID.
    GroupId string

    // UseKtor uses Ktor HTTP client (default true).
    // If false, uses OkHttp.
    UseKtor bool
}

func Generate(svc *contract.Service, cfg *Config) ([]*sdk.File, error)
```

### Template Files

```
templates/
├── build.gradle.kts.tmpl    # Gradle build configuration
├── Client.kt.tmpl           # Main client
├── Types.kt.tmpl            # Model types
├── Resources.kt.tmpl        # Resource classes
├── Streaming.kt.tmpl        # SSE support
└── Errors.kt.tmpl           # Error types
```

### Generated Files

| File                 | Purpose                           |
|----------------------|-----------------------------------|
| `build.gradle.kts`   | Gradle build configuration        |
| `Client.kt`          | Main client class                 |
| `Types.kt`           | All model type definitions        |
| `Resources.kt`       | Resource namespaces and methods   |
| `Streaming.kt`       | Flow-based SSE implementation     |
| `Errors.kt`          | Error type definitions            |

## Usage Examples

### Basic Usage

```kotlin
import com.example.servicesdk.*

// Create client
val client = ServiceName(
    options = ClientOptions(apiKey = "your-api-key")
)

// Make a request
val response = client.completions.create(
    CreateRequest(
        model = "model-name",
        messages = listOf(
            Message(role = Role.USER, content = "Hello")
        )
    )
)

println(response.content)
```

### Streaming

```kotlin
client.completions.createStream(
    CreateRequest(
        model = "model-name",
        messages = listOf(Message(role = Role.USER, content = "Hello"))
    )
).collect { event ->
    print(event.delta.text)
}
```

### Error Handling

```kotlin
try {
    val response = client.completions.create(request)
} catch (e: SDKException.ApiError) {
    println("API Error ${e.statusCode}: ${e.message}")
} catch (e: SDKException.Timeout) {
    println("Request timed out")
} catch (e: SDKException.Cancelled) {
    println("Request was cancelled")
} catch (e: SDKException) {
    println("SDK error: $e")
}
```

### Coroutine Cancellation

```kotlin
val job = launch {
    client.completions.createStream(request).collect { event ->
        println(event)
    }
}

// Cancel after 5 seconds
delay(5000)
job.cancel()
```

### Custom Configuration

```kotlin
val client = ServiceName(
    options = ClientOptions(
        apiKey = "your-api-key",
        baseUrl = "https://custom.api.com",
        timeout = 120.seconds,
        maxRetries = 3,
        defaultHeaders = mapOf(
            "X-Custom-Header" to "value"
        )
    )
)
```

### DSL Builder Pattern

```kotlin
val response = client.completions.create {
    model = "model-name"
    messages {
        message(Role.USER, "Hello")
        message(Role.ASSISTANT, "Hi there!")
        message(Role.USER, "How are you?")
    }
    maxTokens = 1024
    temperature = 0.7
}
```

## Testing

### Generator Tests

```go
func TestGenerate_NilService(t *testing.T)
func TestGenerate_ValidKotlin_Compiles(t *testing.T)
func TestGenerate_ProducesExpectedFiles(t *testing.T)
func TestGenerate_TypeMapping(t *testing.T)
func TestGenerate_StreamingMethods(t *testing.T)
```

### Generated SDK Tests

The generated SDK includes a test structure for users to add integration tests:

```
src/test/kotlin/{package}/
└── {Package}Test.kt
```

## Platform Support

### Dependencies

**Core Dependencies:**
- `org.jetbrains.kotlinx:kotlinx-serialization-json:1.6.0`
- `org.jetbrains.kotlinx:kotlinx-coroutines-core:1.8.0`

**HTTP Client (one of):**
- `io.ktor:ktor-client-core:2.3.7` (default)
- `io.ktor:ktor-client-cio:2.3.7`
- `io.ktor:ktor-client-content-negotiation:2.3.7`
- `io.ktor:ktor-serialization-kotlinx-json:2.3.7`

OR

- `com.squareup.okhttp3:okhttp:4.12.0`

### Minimum Versions

| Platform    | Minimum Version | Rationale                        |
|-------------|-----------------|----------------------------------|
| Kotlin      | 1.9.0           | Stable serialization, coroutines |
| Java        | 11              | Modern JVM features              |
| Android SDK | 24 (7.0)        | Coroutines, Ktor support         |

## Migration Path

For projects migrating from other HTTP clients:

1. Replace existing client initialization with SDK client
2. Update method calls to use generated resource methods
3. Replace custom model types with generated data classes
4. Update error handling to use sealed class pattern
5. Replace callback patterns with suspend functions

## Future Enhancements

1. **Kotlin Multiplatform**: iOS, JS, and Native targets
2. **Compose Multiplatform**: State management integration
3. **Request interceptors**: Middleware for custom request/response handling
4. **Response caching**: Built-in response caching
5. **Metrics**: Request timing and success rate tracking
6. **Offline support**: Queue requests when offline

## References

- [Kotlin Coding Conventions](https://kotlinlang.org/docs/coding-conventions.html)
- [Kotlin Coroutines](https://kotlinlang.org/docs/coroutines-guide.html)
- [Kotlin Serialization](https://github.com/Kotlin/kotlinx.serialization)
- [Ktor Client](https://ktor.io/docs/client.html)
- [Kotlin Flow](https://kotlinlang.org/docs/flow.html)
