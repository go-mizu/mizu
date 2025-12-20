# Kotlin Multiplatform Mobile Template Specification

**Status:** Draft
**Author:** Mizu Team
**Created:** 2025-12-20
**Template:** `mobile:kmm`

## Overview

The `mobile:kmm` template generates a production-ready Kotlin Multiplatform Mobile application with full Mizu backend integration. It follows modern KMM development practices with shared business logic, Ktor for networking, Kotlin Serialization for JSON, and platform-native UI (Jetpack Compose for Android, SwiftUI for iOS).

## Template Invocation

```bash
# Default: KMM with Compose + SwiftUI
mizu new ./MyApp --template mobile:kmm

# Custom package name
mizu new ./MyApp --template mobile:kmm --var package=com.company.myapp

# With SQLDelight for local storage
mizu new ./MyApp --template mobile:kmm --var storage=sqldelight

# Android only (no iOS)
mizu new ./MyApp --template mobile:kmm --var platforms=android
```

## Generated Project Structure

```
{{.Name}}/
├── shared/
│   ├── build.gradle.kts
│   └── src/
│       ├── commonMain/
│       │   └── kotlin/{{.Package}}/
│       │       ├── runtime/
│       │       │   ├── MizuRuntime.kt           # Core runtime
│       │       │   ├── Transport.kt              # HTTP transport layer
│       │       │   ├── TokenStore.kt             # Token storage interface
│       │       │   ├── LiveConnection.kt         # SSE streaming
│       │       │   ├── DeviceInfo.kt             # Device info interface
│       │       │   └── MizuError.kt              # Error types
│       │       ├── sdk/
│       │       │   ├── Client.kt                 # Generated Mizu client
│       │       │   ├── Types.kt                  # Generated types
│       │       │   └── Extensions.kt             # Convenience extensions
│       │       └── model/
│       │           └── AppState.kt               # Shared app state
│       ├── commonTest/
│       │   └── kotlin/{{.Package}}/
│       │       ├── RuntimeTest.kt
│       │       └── ClientTest.kt
│       ├── androidMain/
│       │   └── kotlin/{{.Package}}/
│       │       ├── runtime/
│       │       │   ├── AndroidTokenStore.kt      # Android secure storage
│       │       │   └── AndroidDeviceInfo.kt      # Android device info
│       │       └── Platform.android.kt
│       └── iosMain/
│           └── kotlin/{{.Package}}/
│               ├── runtime/
│               │   ├── IOSTokenStore.kt          # iOS Keychain storage
│               │   └── IOSDeviceInfo.kt          # iOS device info
│               └── Platform.ios.kt
├── androidApp/
│   ├── build.gradle.kts
│   └── src/main/
│       ├── kotlin/{{.Package}}/android/
│       │   ├── MainActivity.kt
│       │   ├── {{.Name}}App.kt
│       │   ├── navigation/
│       │   │   └── NavGraph.kt
│       │   ├── screens/
│       │   │   ├── HomeScreen.kt
│       │   │   └── WelcomeScreen.kt
│       │   ├── components/
│       │   │   ├── LoadingView.kt
│       │   │   └── ErrorView.kt
│       │   └── theme/
│       │       ├── Color.kt
│       │       ├── Theme.kt
│       │       └── Type.kt
│       ├── res/
│       │   ├── values/
│       │   │   ├── strings.xml
│       │   │   ├── colors.xml
│       │   │   └── themes.xml
│       │   └── drawable/
│       └── AndroidManifest.xml
├── iosApp/
│   ├── iosApp/
│   │   ├── {{.Name}}App.swift
│   │   ├── ContentView.swift
│   │   ├── Screens/
│   │   │   ├── HomeScreen.swift
│   │   │   └── WelcomeScreen.swift
│   │   ├── Components/
│   │   │   ├── LoadingView.swift
│   │   │   └── ErrorView.swift
│   │   ├── ViewModels/
│   │   │   └── AppViewModel.swift
│   │   ├── Assets.xcassets/
│   │   └── Info.plist
│   └── iosApp.xcodeproj/
├── build.gradle.kts
├── settings.gradle.kts
├── gradle.properties
├── gradle/
│   └── wrapper/
│       ├── gradle-wrapper.jar
│       └── gradle-wrapper.properties
├── gradlew
├── gradlew.bat
├── .gitignore
└── README.md
```

## MizuMobileRuntime

### Core Runtime (`shared/src/commonMain/kotlin/.../runtime/MizuRuntime.kt`)

```kotlin
package {{.Package}}.runtime

import io.ktor.client.*
import io.ktor.client.plugins.*
import io.ktor.client.plugins.contentnegotiation.*
import io.ktor.client.request.*
import io.ktor.client.statement.*
import io.ktor.http.*
import io.ktor.serialization.kotlinx.json.*
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.serialization.json.Json

/**
 * MizuRuntime is the core client for communicating with a Mizu backend.
 */
class MizuRuntime private constructor(
    private val config: Config
) {
    /** Runtime configuration */
    data class Config(
        val baseUrl: String = "http://localhost:3000",
        val timeout: Long = 30_000L,
        val tokenStore: TokenStore? = null,
        val deviceInfo: DeviceInfo? = null
    )

    /** Base URL for all API requests */
    var baseUrl: String = config.baseUrl
        private set

    /** Request timeout in milliseconds */
    var timeout: Long = config.timeout
        private set

    /** Secure token storage */
    val tokenStore: TokenStore = config.tokenStore ?: InMemoryTokenStore()

    /** Device info provider */
    val deviceInfo: DeviceInfo? = config.deviceInfo

    /** Live connection manager */
    val live: LiveConnection by lazy { LiveConnection(this) }

    /** Default headers added to all requests */
    val defaultHeaders: MutableMap<String, String> = mutableMapOf()

    /** JSON serializer */
    internal val json = Json {
        ignoreUnknownKeys = true
        isLenient = true
        encodeDefaults = true
        prettyPrint = false
    }

    /** HTTP client */
    internal val client: HttpClient by lazy {
        HttpClient {
            install(ContentNegotiation) {
                json(json)
            }
            install(HttpTimeout) {
                requestTimeoutMillis = timeout
                connectTimeoutMillis = timeout
                socketTimeoutMillis = timeout
            }
        }
    }

    private val _isAuthenticated = MutableStateFlow(false)
    /** Current authentication state */
    val isAuthenticated: StateFlow<Boolean> = _isAuthenticated.asStateFlow()

    init {
        // Observe token changes
        tokenStore.addObserver { token ->
            _isAuthenticated.value = token != null
        }
    }

    companion object {
        @Volatile
        private var instance: MizuRuntime? = null

        /** Shared singleton instance */
        val shared: MizuRuntime
            get() = instance ?: synchronized(this) {
                instance ?: MizuRuntime(Config()).also { instance = it }
            }

        /** Initializes the runtime with configuration */
        fun initialize(config: Config): MizuRuntime {
            return synchronized(this) {
                MizuRuntime(config).also { instance = it }
            }
        }

        /** Initializes with simple parameters */
        fun initialize(
            baseUrl: String,
            timeout: Long = 30_000L,
            tokenStore: TokenStore? = null,
            deviceInfo: DeviceInfo? = null
        ): MizuRuntime {
            return initialize(Config(baseUrl, timeout, tokenStore, deviceInfo))
        }
    }

    // MARK: - HTTP Methods

    /** Performs a GET request */
    suspend inline fun <reified T> get(
        path: String,
        query: Map<String, String>? = null,
        headers: Map<String, String>? = null
    ): T {
        return request(HttpMethod.Get, path, query = query, headers = headers)
    }

    /** Performs a POST request */
    suspend inline fun <reified T, reified B> post(
        path: String,
        body: B? = null,
        headers: Map<String, String>? = null
    ): T {
        return request(HttpMethod.Post, path, body = body, headers = headers)
    }

    /** Performs a PUT request */
    suspend inline fun <reified T, reified B> put(
        path: String,
        body: B? = null,
        headers: Map<String, String>? = null
    ): T {
        return request(HttpMethod.Put, path, body = body, headers = headers)
    }

    /** Performs a DELETE request */
    suspend inline fun <reified T> delete(
        path: String,
        headers: Map<String, String>? = null
    ): T {
        return request(HttpMethod.Delete, path, headers = headers)
    }

    /** Performs a PATCH request */
    suspend inline fun <reified T, reified B> patch(
        path: String,
        body: B? = null,
        headers: Map<String, String>? = null
    ): T {
        return request(HttpMethod.Patch, path, body = body, headers = headers)
    }

    // MARK: - Private

    @PublishedApi
    internal suspend inline fun <reified T, reified B> request(
        method: HttpMethod,
        path: String,
        query: Map<String, String>? = null,
        body: B? = null,
        headers: Map<String, String>? = null
    ): T {
        try {
            val response = client.request {
                this.method = method
                url {
                    takeFrom(buildUrl(path, query))
                }

                // Add default headers
                defaultHeaders.forEach { (key, value) ->
                    header(key, value)
                }

                // Add mobile headers
                deviceInfo?.let { info ->
                    header("X-Device-ID", info.deviceId)
                    header("X-App-Version", info.appVersion)
                    header("X-App-Build", info.appBuild)
                    header("X-Device-Model", info.model)
                    header("X-Platform", info.platform)
                    header("X-OS-Version", info.osVersion)
                    header("X-Timezone", info.timezone)
                    header("X-Locale", info.locale)
                }

                // Add custom headers
                headers?.forEach { (key, value) ->
                    header(key, value)
                }

                // Add auth token
                tokenStore.getToken()?.let { token ->
                    header("Authorization", "Bearer ${token.accessToken}")
                }

                // Add body
                if (body != null) {
                    contentType(ContentType.Application.Json)
                    setBody(body)
                }
            }

            // Handle errors
            if (response.status.value >= 400) {
                throw parseError(response)
            }

            // Handle empty response
            if (T::class == Unit::class) {
                @Suppress("UNCHECKED_CAST")
                return Unit as T
            }

            return json.decodeFromString(response.bodyAsText())
        } catch (e: MizuError) {
            throw e
        } catch (e: Exception) {
            throw MizuError.Network(e.message ?: "Network error", e)
        }
    }

    @PublishedApi
    internal fun buildUrl(path: String, query: Map<String, String>?): String {
        val base = baseUrl.trimEnd('/')
        val cleanPath = if (path.startsWith('/')) path else "/$path"
        val url = StringBuilder("$base$cleanPath")

        if (!query.isNullOrEmpty()) {
            url.append("?")
            url.append(query.entries.joinToString("&") { (key, value) ->
                "${key.encodeURLParameter()}=${value.encodeURLParameter()}"
            })
        }

        return url.toString()
    }

    @PublishedApi
    internal suspend fun parseError(response: HttpResponse): MizuError {
        return try {
            val body = response.bodyAsText()
            val apiError = json.decodeFromString<ApiError>(body)
            MizuError.Api(apiError)
        } catch (_: Exception) {
            MizuError.Http(response.status.value, response.bodyAsText())
        }
    }

    /** Closes the HTTP client */
    fun close() {
        client.close()
    }
}

private fun String.encodeURLParameter(): String {
    return this.encodeURLPath()
}
```

### Transport Layer (`shared/src/commonMain/kotlin/.../runtime/Transport.kt`)

```kotlin
package {{.Package}}.runtime

import io.ktor.client.*
import io.ktor.client.request.*
import io.ktor.client.statement.*
import io.ktor.http.*

/**
 * Transport request
 */
data class TransportRequest(
    val url: String,
    val method: HttpMethod,
    val headers: Map<String, String>,
    val body: String? = null,
    val timeout: Long = 30_000L
)

/**
 * Transport response
 */
data class TransportResponse(
    val statusCode: Int,
    val headers: Map<String, String>,
    val body: String
)

/**
 * Transport protocol for executing HTTP requests
 */
interface Transport {
    suspend fun execute(request: TransportRequest): TransportResponse
}

/**
 * Request interceptor
 */
interface RequestInterceptor {
    suspend fun intercept(request: TransportRequest): TransportRequest
}

/**
 * HTTP-based transport implementation
 */
class HttpTransport(
    private val client: HttpClient = HttpClient()
) : Transport {
    private val interceptors = mutableListOf<RequestInterceptor>()

    /** Adds a request interceptor */
    fun addInterceptor(interceptor: RequestInterceptor) {
        interceptors.add(interceptor)
    }

    override suspend fun execute(request: TransportRequest): TransportResponse {
        var req = request

        // Apply interceptors
        for (interceptor in interceptors) {
            req = interceptor.intercept(req)
        }

        try {
            val response = client.request(req.url) {
                method = req.method
                req.headers.forEach { (key, value) ->
                    header(key, value)
                }
                if (req.body != null) {
                    setBody(req.body)
                }
            }

            val headers = response.headers.entries().associate { it.key to it.value.joinToString(",") }

            return TransportResponse(
                statusCode = response.status.value,
                headers = headers,
                body = response.bodyAsText()
            )
        } catch (e: Exception) {
            throw MizuError.Network(e.message ?: "Network error", e)
        }
    }
}

/**
 * Logging interceptor for debugging
 */
class LoggingInterceptor : RequestInterceptor {
    override suspend fun intercept(request: TransportRequest): TransportRequest {
        println("[Mizu] ${request.method.value} ${request.url}")
        return request
    }
}
```

### Token Store (`shared/src/commonMain/kotlin/.../runtime/TokenStore.kt`)

```kotlin
package {{.Package}}.runtime

import kotlinx.serialization.Serializable

/**
 * Stored authentication token
 */
@Serializable
data class AuthToken(
    val accessToken: String,
    val refreshToken: String? = null,
    val expiresAt: Long? = null, // Unix timestamp in milliseconds
    val tokenType: String = "Bearer"
) {
    val isExpired: Boolean
        get() = expiresAt?.let { System.currentTimeMillis() >= it } ?: false
}

/**
 * Creates an auth token
 */
fun createAuthToken(
    accessToken: String,
    refreshToken: String? = null,
    expiresInSeconds: Long? = null,
    tokenType: String = "Bearer"
): AuthToken {
    val expiresAt = expiresInSeconds?.let { System.currentTimeMillis() + (it * 1000) }
    return AuthToken(accessToken, refreshToken, expiresAt, tokenType)
}

typealias TokenObserver = (AuthToken?) -> Unit

/**
 * Token storage interface
 */
interface TokenStore {
    /** Gets the current token */
    fun getToken(): AuthToken?

    /** Sets the token */
    suspend fun setToken(token: AuthToken)

    /** Clears the token */
    suspend fun clearToken()

    /** Adds an observer for token changes */
    fun addObserver(observer: TokenObserver)

    /** Removes an observer */
    fun removeObserver(observer: TokenObserver)
}

/**
 * In-memory token store for testing
 */
class InMemoryTokenStore : TokenStore {
    private var token: AuthToken? = null
    private val observers = mutableListOf<TokenObserver>()

    override fun getToken(): AuthToken? = token

    override suspend fun setToken(token: AuthToken) {
        this.token = token
        notifyObservers(token)
    }

    override suspend fun clearToken() {
        token = null
        notifyObservers(null)
    }

    override fun addObserver(observer: TokenObserver) {
        observers.add(observer)
    }

    override fun removeObserver(observer: TokenObserver) {
        observers.remove(observer)
    }

    private fun notifyObservers(token: AuthToken?) {
        observers.forEach { it(token) }
    }
}

// Expect/actual for platform-specific time
expect fun System.currentTimeMillis(): Long
```

### Live Streaming (`shared/src/commonMain/kotlin/.../runtime/LiveConnection.kt`)

```kotlin
package {{.Package}}.runtime

import io.ktor.client.plugins.sse.*
import io.ktor.client.request.*
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.flow
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.Json

/**
 * Server-sent event
 */
@Serializable
data class ServerEvent(
    val id: String? = null,
    val event: String? = null,
    val data: String,
    val retry: Int? = null
)

/**
 * Decodes event data as JSON
 */
inline fun <reified T> ServerEvent.decode(json: Json = Json): T {
    return json.decodeFromString(data)
}

/**
 * Live connection manager for SSE
 */
class LiveConnection(
    private val runtime: MizuRuntime
) {
    private val activeConnections = mutableMapOf<String, kotlinx.coroutines.Job>()

    /**
     * Connects to an SSE endpoint and returns a flow of events
     */
    fun connect(
        path: String,
        headers: Map<String, String>? = null
    ): Flow<ServerEvent> = flow {
        val url = runtime.buildUrl(path, null)

        try {
            runtime.client.sse(url) {
                // Add default headers
                runtime.defaultHeaders.forEach { (key, value) ->
                    header(key, value)
                }

                // Add custom headers
                headers?.forEach { (key, value) ->
                    header(key, value)
                }

                // Add auth token
                runtime.tokenStore.getToken()?.let { token ->
                    header("Authorization", "Bearer ${token.accessToken}")
                }
            }.incoming.collect { event ->
                event.data?.let { data ->
                    emit(
                        ServerEvent(
                            id = event.id,
                            event = event.event,
                            data = data,
                            retry = event.retry?.toInt()
                        )
                    )
                }
            }
        } catch (e: CancellationException) {
            throw e
        } catch (e: Exception) {
            throw MizuError.Network("SSE connection failed: ${e.message}", e)
        }
    }

    /** Disconnects from a specific path */
    fun disconnect(path: String) {
        activeConnections[path]?.cancel()
        activeConnections.remove(path)
    }

    /** Disconnects all active connections */
    fun disconnectAll() {
        activeConnections.values.forEach { it.cancel() }
        activeConnections.clear()
    }
}
```

### Device Info (`shared/src/commonMain/kotlin/.../runtime/DeviceInfo.kt`)

```kotlin
package {{.Package}}.runtime

/**
 * Device information interface
 */
interface DeviceInfo {
    val deviceId: String
    val appVersion: String
    val appBuild: String
    val model: String
    val platform: String
    val osVersion: String
    val timezone: String
    val locale: String
}

/**
 * Expect declaration for platform-specific device info
 */
expect class PlatformDeviceInfo() : DeviceInfo
```

### Errors (`shared/src/commonMain/kotlin/.../runtime/MizuError.kt`)

```kotlin
package {{.Package}}.runtime

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

/**
 * API error response from server
 */
@Serializable
data class ApiError(
    val code: String,
    val message: String,
    val details: Map<String, String>? = null,
    @SerialName("trace_id") val traceId: String? = null
)

/**
 * Mizu client error
 */
sealed class MizuError(
    override val message: String,
    override val cause: Throwable? = null
) : Exception(message, cause) {

    /** Invalid server response */
    class InvalidResponse(
        message: String = "Invalid server response"
    ) : MizuError(message)

    /** HTTP error */
    class Http(
        val statusCode: Int,
        val body: String
    ) : MizuError("HTTP error $statusCode")

    /** API error from server */
    class Api(
        val error: ApiError
    ) : MizuError(error.message)

    /** Network error */
    class Network(
        message: String,
        cause: Throwable? = null
    ) : MizuError(message, cause)

    /** Encoding error */
    class Encoding(
        message: String,
        cause: Throwable? = null
    ) : MizuError(message, cause)

    /** Decoding error */
    class Decoding(
        message: String,
        cause: Throwable? = null
    ) : MizuError(message, cause)

    /** Unauthorized */
    class Unauthorized(
        message: String = "Unauthorized"
    ) : MizuError(message)

    /** Token expired */
    class TokenExpired(
        message: String = "Token expired"
    ) : MizuError(message)
}

val MizuError.isInvalidResponse: Boolean get() = this is MizuError.InvalidResponse
val MizuError.isHttp: Boolean get() = this is MizuError.Http
val MizuError.isApi: Boolean get() = this is MizuError.Api
val MizuError.isNetwork: Boolean get() = this is MizuError.Network
val MizuError.isUnauthorized: Boolean get() = this is MizuError.Unauthorized
val MizuError.isTokenExpired: Boolean get() = this is MizuError.TokenExpired
```

## Platform-Specific Implementations

### Android Token Store (`shared/src/androidMain/kotlin/.../runtime/AndroidTokenStore.kt`)

```kotlin
package {{.Package}}.runtime

import android.content.Context
import android.content.SharedPreferences
import androidx.security.crypto.EncryptedSharedPreferences
import androidx.security.crypto.MasterKey
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json

/**
 * Android secure storage-backed token storage
 */
class AndroidTokenStore(context: Context) : TokenStore {
    private val masterKey = MasterKey.Builder(context)
        .setKeyScheme(MasterKey.KeyScheme.AES256_GCM)
        .build()

    private val prefs: SharedPreferences = EncryptedSharedPreferences.create(
        context,
        "mizu_secure_prefs",
        masterKey,
        EncryptedSharedPreferences.PrefKeyEncryptionScheme.AES256_SIV,
        EncryptedSharedPreferences.PrefValueEncryptionScheme.AES256_GCM
    )

    private val observers = mutableListOf<TokenObserver>()
    private val json = Json { ignoreUnknownKeys = true }

    override fun getToken(): AuthToken? {
        val tokenJson = prefs.getString(TOKEN_KEY, null) ?: return null
        return try {
            json.decodeFromString(tokenJson)
        } catch (_: Exception) {
            null
        }
    }

    override suspend fun setToken(token: AuthToken) {
        withContext(Dispatchers.IO) {
            prefs.edit()
                .putString(TOKEN_KEY, json.encodeToString(token))
                .apply()
        }
        notifyObservers(token)
    }

    override suspend fun clearToken() {
        withContext(Dispatchers.IO) {
            prefs.edit()
                .remove(TOKEN_KEY)
                .apply()
        }
        notifyObservers(null)
    }

    override fun addObserver(observer: TokenObserver) {
        observers.add(observer)
    }

    override fun removeObserver(observer: TokenObserver) {
        observers.remove(observer)
    }

    private fun notifyObservers(token: AuthToken?) {
        observers.forEach { it(token) }
    }

    companion object {
        private const val TOKEN_KEY = "mizu_auth_token"
    }
}

actual fun System.currentTimeMillis(): Long = java.lang.System.currentTimeMillis()
```

### Android Device Info (`shared/src/androidMain/kotlin/.../runtime/AndroidDeviceInfo.kt`)

```kotlin
package {{.Package}}.runtime

import android.annotation.SuppressLint
import android.content.Context
import android.os.Build
import android.provider.Settings
import java.util.*

/**
 * Android device info implementation
 */
class AndroidDeviceInfo(private val context: Context) : DeviceInfo {
    @SuppressLint("HardwareIds")
    override val deviceId: String by lazy {
        Settings.Secure.getString(context.contentResolver, Settings.Secure.ANDROID_ID)
    }

    override val appVersion: String by lazy {
        try {
            context.packageManager.getPackageInfo(context.packageName, 0).versionName ?: "1.0.0"
        } catch (_: Exception) {
            "1.0.0"
        }
    }

    override val appBuild: String by lazy {
        try {
            val pInfo = context.packageManager.getPackageInfo(context.packageName, 0)
            if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.P) {
                pInfo.longVersionCode.toString()
            } else {
                @Suppress("DEPRECATION")
                pInfo.versionCode.toString()
            }
        } catch (_: Exception) {
            "1"
        }
    }

    override val model: String = Build.MODEL

    override val platform: String = "android"

    override val osVersion: String = Build.VERSION.RELEASE

    override val timezone: String = TimeZone.getDefault().id

    override val locale: String = Locale.getDefault().toLanguageTag()
}

actual class PlatformDeviceInfo : DeviceInfo {
    override val deviceId: String = ""
    override val appVersion: String = "1.0.0"
    override val appBuild: String = "1"
    override val model: String = Build.MODEL
    override val platform: String = "android"
    override val osVersion: String = Build.VERSION.RELEASE
    override val timezone: String = TimeZone.getDefault().id
    override val locale: String = Locale.getDefault().toLanguageTag()
}
```

### iOS Token Store (`shared/src/iosMain/kotlin/.../runtime/IOSTokenStore.kt`)

```kotlin
package {{.Package}}.runtime

import kotlinx.cinterop.*
import platform.Foundation.*
import platform.Security.*
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json

/**
 * iOS Keychain-backed token storage
 */
class IOSTokenStore : TokenStore {
    private val observers = mutableListOf<TokenObserver>()
    private val json = Json { ignoreUnknownKeys = true }

    override fun getToken(): AuthToken? {
        val data = keychainGet(TOKEN_KEY) ?: return null
        return try {
            json.decodeFromString(data)
        } catch (_: Exception) {
            null
        }
    }

    override suspend fun setToken(token: AuthToken) {
        val tokenJson = json.encodeToString(token)
        keychainSet(TOKEN_KEY, tokenJson)
        notifyObservers(token)
    }

    override suspend fun clearToken() {
        keychainDelete(TOKEN_KEY)
        notifyObservers(null)
    }

    override fun addObserver(observer: TokenObserver) {
        observers.add(observer)
    }

    override fun removeObserver(observer: TokenObserver) {
        observers.remove(observer)
    }

    private fun notifyObservers(token: AuthToken?) {
        observers.forEach { it(token) }
    }

    @OptIn(ExperimentalForeignApi::class)
    private fun keychainGet(key: String): String? = memScoped {
        val query = mapOf(
            kSecClass to kSecClassGenericPassword,
            kSecAttrAccount to key.toNSString(),
            kSecReturnData to true,
            kSecMatchLimit to kSecMatchLimitOne
        ).toNSDictionary()

        val result = alloc<ObjCObjectVar<Any?>>()
        val status = SecItemCopyMatching(query, result.ptr)

        if (status == errSecSuccess) {
            val data = result.value as? NSData
            data?.let { NSString.create(data = it, encoding = NSUTF8StringEncoding) as? String }
        } else {
            null
        }
    }

    @OptIn(ExperimentalForeignApi::class)
    private fun keychainSet(key: String, value: String) = memScoped {
        keychainDelete(key)

        val valueData = (value as NSString).dataUsingEncoding(NSUTF8StringEncoding)!!
        val query = mapOf(
            kSecClass to kSecClassGenericPassword,
            kSecAttrAccount to key.toNSString(),
            kSecValueData to valueData
        ).toNSDictionary()

        SecItemAdd(query, null)
    }

    @OptIn(ExperimentalForeignApi::class)
    private fun keychainDelete(key: String) = memScoped {
        val query = mapOf(
            kSecClass to kSecClassGenericPassword,
            kSecAttrAccount to key.toNSString()
        ).toNSDictionary()

        SecItemDelete(query)
    }

    private fun String.toNSString(): NSString = this as NSString

    private fun Map<*, *>.toNSDictionary(): NSDictionary {
        return NSDictionary.dictionaryWithObjects(
            values.toList(),
            keys.toList()
        )
    }

    companion object {
        private const val TOKEN_KEY = "mizu_auth_token"
    }
}

actual fun System.currentTimeMillis(): Long =
    (NSDate().timeIntervalSince1970 * 1000).toLong()
```

### iOS Device Info (`shared/src/iosMain/kotlin/.../runtime/IOSDeviceInfo.kt`)

```kotlin
package {{.Package}}.runtime

import platform.Foundation.*
import platform.UIKit.*

/**
 * iOS device info implementation
 */
class IOSDeviceInfo : DeviceInfo {
    override val deviceId: String by lazy {
        UIDevice.currentDevice.identifierForVendor?.UUIDString ?: ""
    }

    override val appVersion: String by lazy {
        NSBundle.mainBundle.objectForInfoDictionaryKey("CFBundleShortVersionString") as? String ?: "1.0.0"
    }

    override val appBuild: String by lazy {
        NSBundle.mainBundle.objectForInfoDictionaryKey("CFBundleVersion") as? String ?: "1"
    }

    override val model: String = UIDevice.currentDevice.model

    override val platform: String = "ios"

    override val osVersion: String = UIDevice.currentDevice.systemVersion

    override val timezone: String = NSTimeZone.localTimeZone.name

    override val locale: String = NSLocale.currentLocale.localeIdentifier
}

actual class PlatformDeviceInfo : DeviceInfo {
    private val iosInfo = IOSDeviceInfo()
    override val deviceId: String get() = iosInfo.deviceId
    override val appVersion: String get() = iosInfo.appVersion
    override val appBuild: String get() = iosInfo.appBuild
    override val model: String get() = iosInfo.model
    override val platform: String get() = iosInfo.platform
    override val osVersion: String get() = iosInfo.osVersion
    override val timezone: String get() = iosInfo.timezone
    override val locale: String get() = iosInfo.locale
}
```

## Generated SDK

### Client (`shared/src/commonMain/kotlin/.../sdk/Client.kt`)

```kotlin
package {{.Package}}.sdk

import {{.Package}}.runtime.*

/**
 * Generated Mizu API client for {{.Name}}
 */
class {{.Name}}Client(
    private val runtime: MizuRuntime = MizuRuntime.shared
) {
    // MARK: - Auth

    /** Sign in with credentials */
    suspend fun signIn(email: String, password: String): AuthResponse {
        return runtime.post("/auth/signin", SignInRequest(email, password))
    }

    /** Sign up with credentials */
    suspend fun signUp(email: String, password: String, name: String): AuthResponse {
        return runtime.post("/auth/signup", SignUpRequest(email, password, name))
    }

    /** Sign out */
    suspend fun signOut() {
        runtime.delete<Unit>("/auth/signout")
        runtime.tokenStore.clearToken()
    }

    // MARK: - Users

    /** Get current user profile */
    suspend fun getCurrentUser(): User {
        return runtime.get("/users/me")
    }

    /** Update current user profile */
    suspend fun updateCurrentUser(update: UserUpdate): User {
        return runtime.put("/users/me", update)
    }

    // MARK: - Token Storage

    /** Store an auth response token */
    suspend fun storeAuthToken(response: AuthResponse) {
        val token = createAuthToken(
            accessToken = response.token.accessToken,
            refreshToken = response.token.refreshToken,
            expiresInSeconds = response.token.expiresIn.toLong()
        )
        runtime.tokenStore.setToken(token)
    }
}
```

### Types (`shared/src/commonMain/kotlin/.../sdk/Types.kt`)

```kotlin
package {{.Package}}.sdk

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

// MARK: - Auth Types

@Serializable
data class SignInRequest(
    val email: String,
    val password: String
)

@Serializable
data class SignUpRequest(
    val email: String,
    val password: String,
    val name: String
)

@Serializable
data class AuthResponse(
    val user: User,
    val token: TokenResponse
)

@Serializable
data class TokenResponse(
    @SerialName("access_token")
    val accessToken: String,
    @SerialName("refresh_token")
    val refreshToken: String? = null,
    @SerialName("expires_in")
    val expiresIn: Int
)

// MARK: - User Types

@Serializable
data class User(
    val id: String,
    val email: String,
    val name: String,
    @SerialName("avatar_url")
    val avatarUrl: String? = null,
    @SerialName("created_at")
    val createdAt: String,
    @SerialName("updated_at")
    val updatedAt: String
)

@Serializable
data class UserUpdate(
    val name: String? = null,
    @SerialName("avatar_url")
    val avatarUrl: String? = null
)
```

### Extensions (`shared/src/commonMain/kotlin/.../sdk/Extensions.kt`)

```kotlin
package {{.Package}}.sdk

import {{.Package}}.runtime.*

/**
 * Store an auth response token
 */
suspend fun MizuRuntime.storeAuthToken(response: AuthResponse) {
    val token = createAuthToken(
        accessToken = response.token.accessToken,
        refreshToken = response.token.refreshToken,
        expiresInSeconds = response.token.expiresIn.toLong()
    )
    tokenStore.setToken(token)
}
```

## Android App

### MainActivity (`androidApp/src/main/kotlin/.../android/MainActivity.kt`)

```kotlin
package {{.Package}}.android

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge

class MainActivity : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        enableEdgeToEdge()
        setContent {
            {{.Name}}App()
        }
    }
}
```

### App (`androidApp/src/main/kotlin/.../android/{{.Name}}App.kt`)

```kotlin
package {{.Package}}.android

import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import {{.Package}}.android.navigation.NavGraph
import {{.Package}}.android.theme.{{.Name}}Theme
import {{.Package}}.runtime.AndroidDeviceInfo
import {{.Package}}.runtime.AndroidTokenStore
import {{.Package}}.runtime.MizuRuntime

@Composable
fun {{.Name}}App() {
    val context = LocalContext.current

    // Initialize runtime
    LaunchedEffect(Unit) {
        MizuRuntime.initialize(
            baseUrl = if (BuildConfig.DEBUG) "http://10.0.2.2:3000" else "https://api.example.com",
            tokenStore = AndroidTokenStore(context),
            deviceInfo = AndroidDeviceInfo(context)
        )
    }

    {{.Name}}Theme {
        Surface(
            modifier = Modifier.fillMaxSize(),
            color = MaterialTheme.colorScheme.background
        ) {
            NavGraph()
        }
    }
}
```

### Navigation (`androidApp/src/main/kotlin/.../android/navigation/NavGraph.kt`)

```kotlin
package {{.Package}}.android.navigation

import androidx.compose.runtime.*
import androidx.navigation.compose.NavHost
import androidx.navigation.compose.composable
import androidx.navigation.compose.rememberNavController
import {{.Package}}.android.screens.HomeScreen
import {{.Package}}.android.screens.WelcomeScreen
import {{.Package}}.runtime.MizuRuntime

sealed class Screen(val route: String) {
    object Welcome : Screen("welcome")
    object Home : Screen("home")
}

@Composable
fun NavGraph() {
    val navController = rememberNavController()
    val isAuthenticated by MizuRuntime.shared.isAuthenticated.collectAsState()

    val startDestination = if (isAuthenticated) Screen.Home.route else Screen.Welcome.route

    NavHost(
        navController = navController,
        startDestination = startDestination
    ) {
        composable(Screen.Welcome.route) {
            WelcomeScreen(
                onNavigateToHome = {
                    navController.navigate(Screen.Home.route) {
                        popUpTo(Screen.Welcome.route) { inclusive = true }
                    }
                }
            )
        }
        composable(Screen.Home.route) {
            HomeScreen(
                onNavigateToWelcome = {
                    navController.navigate(Screen.Welcome.route) {
                        popUpTo(Screen.Home.route) { inclusive = true }
                    }
                }
            )
        }
    }
}
```

### Home Screen (`androidApp/src/main/kotlin/.../android/screens/HomeScreen.kt`)

```kotlin
package {{.Package}}.android.screens

import androidx.compose.foundation.layout.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import kotlinx.coroutines.launch
import {{.Package}}.runtime.MizuRuntime

@Composable
fun HomeScreen(
    onNavigateToWelcome: () -> Unit
) {
    val scope = rememberCoroutineScope()
    var isLoading by remember { mutableStateOf(false) }

    Column(
        modifier = Modifier
            .fillMaxSize()
            .padding(24.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.Center
    ) {
        Surface(
            shape = MaterialTheme.shapes.large,
            color = MaterialTheme.colorScheme.primaryContainer,
            modifier = Modifier.size(64.dp)
        ) {
            Box(contentAlignment = Alignment.Center) {
                Text("✓", style = MaterialTheme.typography.headlineLarge)
            }
        }

        Spacer(modifier = Modifier.height(16.dp))

        Text(
            text = "Welcome to {{.Name}}",
            style = MaterialTheme.typography.headlineMedium,
            fontWeight = FontWeight.Bold
        )

        Spacer(modifier = Modifier.height(8.dp))

        Text(
            text = "Connected to Mizu backend",
            style = MaterialTheme.typography.bodyLarge,
            color = MaterialTheme.colorScheme.onSurfaceVariant
        )

        Spacer(modifier = Modifier.height(32.dp))

        if (isLoading) {
            CircularProgressIndicator()
        } else {
            Button(
                onClick = {
                    scope.launch {
                        isLoading = true
                        try {
                            MizuRuntime.shared.tokenStore.clearToken()
                            onNavigateToWelcome()
                        } finally {
                            isLoading = false
                        }
                    }
                },
                colors = ButtonDefaults.buttonColors(
                    containerColor = MaterialTheme.colorScheme.secondaryContainer,
                    contentColor = MaterialTheme.colorScheme.onSecondaryContainer
                )
            ) {
                Text("Sign Out")
            }
        }
    }
}
```

### Welcome Screen (`androidApp/src/main/kotlin/.../android/screens/WelcomeScreen.kt`)

```kotlin
package {{.Package}}.android.screens

import androidx.compose.foundation.layout.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import kotlinx.coroutines.launch
import {{.Package}}.runtime.MizuRuntime
import {{.Package}}.runtime.createAuthToken

@Composable
fun WelcomeScreen(
    onNavigateToHome: () -> Unit
) {
    val scope = rememberCoroutineScope()
    var isLoading by remember { mutableStateOf(false) }

    Column(
        modifier = Modifier
            .fillMaxSize()
            .padding(24.dp)
    ) {
        Spacer(modifier = Modifier.weight(1f))

        Column(
            horizontalAlignment = Alignment.CenterHorizontally,
            modifier = Modifier.fillMaxWidth()
        ) {
            Surface(
                shape = MaterialTheme.shapes.large,
                color = MaterialTheme.colorScheme.primaryContainer,
                modifier = Modifier.size(80.dp)
            ) {
                Box(contentAlignment = Alignment.Center) {
                    Text("K", style = MaterialTheme.typography.displayMedium)
                }
            }

            Spacer(modifier = Modifier.height(24.dp))

            Text(
                text = "Welcome to {{.Name}}",
                style = MaterialTheme.typography.headlineLarge,
                fontWeight = FontWeight.Bold
            )

            Spacer(modifier = Modifier.height(16.dp))

            Text(
                text = "A modern KMM app powered by Mizu",
                style = MaterialTheme.typography.bodyLarge,
                color = MaterialTheme.colorScheme.onSurfaceVariant
            )
        }

        Spacer(modifier = Modifier.weight(1f))

        if (isLoading) {
            CircularProgressIndicator(
                modifier = Modifier.align(Alignment.CenterHorizontally)
            )
        } else {
            Button(
                onClick = {
                    scope.launch {
                        isLoading = true
                        try {
                            // Demo: Set a test token
                            MizuRuntime.shared.tokenStore.setToken(
                                createAuthToken(accessToken = "demo_token")
                            )
                            onNavigateToHome()
                        } finally {
                            isLoading = false
                        }
                    }
                },
                modifier = Modifier
                    .fillMaxWidth()
                    .height(56.dp)
            ) {
                Text("Get Started")
            }
        }

        Spacer(modifier = Modifier.height(24.dp))
    }
}
```

### Theme (`androidApp/src/main/kotlin/.../android/theme/Theme.kt`)

```kotlin
package {{.Package}}.android.theme

import android.app.Activity
import android.os.Build
import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.material3.*
import androidx.compose.runtime.Composable
import androidx.compose.runtime.SideEffect
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.toArgb
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.platform.LocalView
import androidx.core.view.WindowCompat

private val LightColorScheme = lightColorScheme(
    primary = Color(0xFF6366F1),
    onPrimary = Color.White,
    primaryContainer = Color(0xFFEEF2FF),
    onPrimaryContainer = Color(0xFF3730A3),
    secondary = Color(0xFF818CF8),
    onSecondary = Color.White,
    secondaryContainer = Color(0xFFE0E7FF),
    onSecondaryContainer = Color(0xFF3730A3),
    background = Color.White,
    onBackground = Color(0xFF1F2937),
    surface = Color.White,
    onSurface = Color(0xFF1F2937),
    surfaceVariant = Color(0xFFF3F4F6),
    onSurfaceVariant = Color(0xFF6B7280)
)

private val DarkColorScheme = darkColorScheme(
    primary = Color(0xFF818CF8),
    onPrimary = Color.Black,
    primaryContainer = Color(0xFF3730A3),
    onPrimaryContainer = Color(0xFFEEF2FF),
    secondary = Color(0xFFA5B4FC),
    onSecondary = Color.Black,
    secondaryContainer = Color(0xFF4338CA),
    onSecondaryContainer = Color(0xFFE0E7FF),
    background = Color(0xFF1F2937),
    onBackground = Color(0xFFF9FAFB),
    surface = Color(0xFF1F2937),
    onSurface = Color(0xFFF9FAFB),
    surfaceVariant = Color(0xFF374151),
    onSurfaceVariant = Color(0xFF9CA3AF)
)

@Composable
fun {{.Name}}Theme(
    darkTheme: Boolean = isSystemInDarkTheme(),
    dynamicColor: Boolean = true,
    content: @Composable () -> Unit
) {
    val colorScheme = when {
        dynamicColor && Build.VERSION.SDK_INT >= Build.VERSION_CODES.S -> {
            val context = LocalContext.current
            if (darkTheme) dynamicDarkColorScheme(context) else dynamicLightColorScheme(context)
        }
        darkTheme -> DarkColorScheme
        else -> LightColorScheme
    }

    val view = LocalView.current
    if (!view.isInEditMode) {
        SideEffect {
            val window = (view.context as Activity).window
            window.statusBarColor = colorScheme.primary.toArgb()
            WindowCompat.getInsetsController(window, view).isAppearanceLightStatusBars = !darkTheme
        }
    }

    MaterialTheme(
        colorScheme = colorScheme,
        typography = Typography,
        content = content
    )
}
```

## iOS App

### App Entry (`iosApp/iosApp/{{.Name}}App.swift`)

```swift
import SwiftUI
import shared

@main
struct {{.Name}}App: App {
    init() {
        // Initialize Mizu runtime
        #if DEBUG
        MizuRuntime.companion.initialize(
            baseUrl: "http://localhost:3000",
            tokenStore: IOSTokenStore(),
            deviceInfo: IOSDeviceInfo()
        )
        #else
        MizuRuntime.companion.initialize(
            baseUrl: "https://api.example.com",
            tokenStore: IOSTokenStore(),
            deviceInfo: IOSDeviceInfo()
        )
        #endif
    }

    var body: some Scene {
        WindowGroup {
            ContentView()
        }
    }
}
```

### Content View (`iosApp/iosApp/ContentView.swift`)

```swift
import SwiftUI
import shared

struct ContentView: View {
    @StateObject private var viewModel = AppViewModel()

    var body: some View {
        Group {
            if viewModel.isAuthenticated {
                HomeScreen(viewModel: viewModel)
            } else {
                WelcomeScreen(viewModel: viewModel)
            }
        }
        .animation(.easeInOut, value: viewModel.isAuthenticated)
    }
}
```

### Home Screen (`iosApp/iosApp/Screens/HomeScreen.swift`)

```swift
import SwiftUI
import shared

struct HomeScreen: View {
    @ObservedObject var viewModel: AppViewModel
    @State private var isLoading = false

    var body: some View {
        VStack(spacing: 20) {
            Spacer()

            Circle()
                .fill(Color.indigo.opacity(0.1))
                .frame(width: 64, height: 64)
                .overlay(
                    Text("✓")
                        .font(.largeTitle)
                        .foregroundColor(.indigo)
                )

            Text("Welcome to {{.Name}}")
                .font(.title)
                .fontWeight(.bold)

            Text("Connected to Mizu backend")
                .foregroundColor(.secondary)

            Spacer()

            if isLoading {
                ProgressView()
            } else {
                Button("Sign Out") {
                    Task {
                        isLoading = true
                        defer { isLoading = false }
                        await viewModel.signOut()
                    }
                }
                .buttonStyle(.bordered)
            }
        }
        .padding()
    }
}
```

### Welcome Screen (`iosApp/iosApp/Screens/WelcomeScreen.swift`)

```swift
import SwiftUI
import shared

struct WelcomeScreen: View {
    @ObservedObject var viewModel: AppViewModel
    @State private var isLoading = false

    var body: some View {
        VStack {
            Spacer()

            VStack(spacing: 24) {
                Circle()
                    .fill(Color.indigo.opacity(0.1))
                    .frame(width: 80, height: 80)
                    .overlay(
                        Text("K")
                            .font(.system(size: 40, weight: .bold))
                            .foregroundColor(.indigo)
                    )

                Text("Welcome to {{.Name}}")
                    .font(.largeTitle)
                    .fontWeight(.bold)

                Text("A modern KMM app powered by Mizu")
                    .foregroundColor(.secondary)
            }

            Spacer()

            if isLoading {
                ProgressView()
            } else {
                Button {
                    Task {
                        isLoading = true
                        defer { isLoading = false }
                        await viewModel.getStarted()
                    }
                } label: {
                    Text("Get Started")
                        .frame(maxWidth: .infinity)
                        .padding()
                }
                .buttonStyle(.borderedProminent)
            }
        }
        .padding()
    }
}
```

### App ViewModel (`iosApp/iosApp/ViewModels/AppViewModel.swift`)

```swift
import Foundation
import Combine
import shared

@MainActor
class AppViewModel: ObservableObject {
    @Published var isAuthenticated = false

    private var cancellables = Set<AnyCancellable>()

    init() {
        // Observe auth state from Kotlin
        observeAuthState()
    }

    private func observeAuthState() {
        // Check initial state
        let token = MizuRuntime.shared.tokenStore.getToken()
        isAuthenticated = token != nil

        // Observe changes
        MizuRuntime.shared.tokenStore.addObserver { [weak self] token in
            DispatchQueue.main.async {
                self?.isAuthenticated = token != nil
            }
        }
    }

    func getStarted() async {
        do {
            // Demo: Set a test token
            try await MizuRuntime.shared.tokenStore.setToken(
                token: AuthToken(
                    accessToken: "demo_token",
                    refreshToken: nil,
                    expiresAt: nil,
                    tokenType: "Bearer"
                )
            )
        } catch {
            print("Error: \(error)")
        }
    }

    func signOut() async {
        do {
            try await MizuRuntime.shared.tokenStore.clearToken()
        } catch {
            print("Error: \(error)")
        }
    }
}
```

## Build Configuration

### Root `build.gradle.kts`

```kotlin
plugins {
    alias(libs.plugins.androidApplication) apply false
    alias(libs.plugins.androidLibrary) apply false
    alias(libs.plugins.kotlinAndroid) apply false
    alias(libs.plugins.kotlinMultiplatform) apply false
    alias(libs.plugins.kotlinSerialization) apply false
    alias(libs.plugins.compose.compiler) apply false
}
```

### `settings.gradle.kts`

```kotlin
pluginManagement {
    repositories {
        google()
        gradlePluginPortal()
        mavenCentral()
    }
}

dependencyResolutionManagement {
    repositories {
        google()
        mavenCentral()
    }
}

rootProject.name = "{{.Name}}"
include(":androidApp")
include(":shared")
```

### `gradle/libs.versions.toml`

```toml
[versions]
agp = "8.5.2"
kotlin = "2.0.21"
compose = "1.7.3"
compose-material3 = "1.3.1"
androidx-activityCompose = "1.9.3"
ktor = "2.3.12"
kotlinx-coroutines = "1.9.0"
kotlinx-serialization = "1.7.3"
security-crypto = "1.1.0-alpha06"
navigation-compose = "2.8.3"

[libraries]
kotlin-test = { module = "org.jetbrains.kotlin:kotlin-test", version.ref = "kotlin" }
androidx-activity-compose = { module = "androidx.activity:activity-compose", version.ref = "androidx-activityCompose" }
compose-ui = { module = "androidx.compose.ui:ui", version.ref = "compose" }
compose-ui-tooling = { module = "androidx.compose.ui:ui-tooling", version.ref = "compose" }
compose-ui-tooling-preview = { module = "androidx.compose.ui:ui-tooling-preview", version.ref = "compose" }
compose-foundation = { module = "androidx.compose.foundation:foundation", version.ref = "compose" }
compose-material3 = { module = "androidx.compose.material3:material3", version.ref = "compose-material3" }
navigation-compose = { module = "androidx.navigation:navigation-compose", version.ref = "navigation-compose" }
ktor-client-core = { module = "io.ktor:ktor-client-core", version.ref = "ktor" }
ktor-client-okhttp = { module = "io.ktor:ktor-client-okhttp", version.ref = "ktor" }
ktor-client-darwin = { module = "io.ktor:ktor-client-darwin", version.ref = "ktor" }
ktor-client-content-negotiation = { module = "io.ktor:ktor-client-content-negotiation", version.ref = "ktor" }
ktor-serialization-kotlinx-json = { module = "io.ktor:ktor-serialization-kotlinx-json", version.ref = "ktor" }
ktor-client-logging = { module = "io.ktor:ktor-client-logging", version.ref = "ktor" }
kotlinx-coroutines-core = { module = "org.jetbrains.kotlinx:kotlinx-coroutines-core", version.ref = "kotlinx-coroutines" }
kotlinx-coroutines-android = { module = "org.jetbrains.kotlinx:kotlinx-coroutines-android", version.ref = "kotlinx-coroutines" }
kotlinx-serialization-json = { module = "org.jetbrains.kotlinx:kotlinx-serialization-json", version.ref = "kotlinx-serialization" }
security-crypto = { module = "androidx.security:security-crypto", version.ref = "security-crypto" }

[plugins]
androidApplication = { id = "com.android.application", version.ref = "agp" }
androidLibrary = { id = "com.android.library", version.ref = "agp" }
kotlinAndroid = { id = "org.jetbrains.kotlin.android", version.ref = "kotlin" }
kotlinMultiplatform = { id = "org.jetbrains.kotlin.multiplatform", version.ref = "kotlin" }
kotlinSerialization = { id = "org.jetbrains.kotlin.plugin.serialization", version.ref = "kotlin" }
compose-compiler = { id = "org.jetbrains.kotlin.plugin.compose", version.ref = "kotlin" }
```

### `gradle.properties`

```properties
org.gradle.jvmargs=-Xmx2048M -Dfile.encoding=UTF-8 -Dkotlin.daemon.jvm.options\="-Xmx2048M"
kotlin.code.style=official
android.nonTransitiveRClass=true
android.useAndroidX=true
kotlin.mpp.androidSourceSetLayoutVersion=2
kotlin.mpp.enableCInteropCommonization=true
```

### Shared Module `build.gradle.kts`

```kotlin
plugins {
    alias(libs.plugins.kotlinMultiplatform)
    alias(libs.plugins.androidLibrary)
    alias(libs.plugins.kotlinSerialization)
}

kotlin {
    androidTarget {
        compilations.all {
            kotlinOptions {
                jvmTarget = "1.8"
            }
        }
    }

    listOf(
        iosX64(),
        iosArm64(),
        iosSimulatorArm64()
    ).forEach { iosTarget ->
        iosTarget.binaries.framework {
            baseName = "shared"
            isStatic = true
        }
    }

    sourceSets {
        commonMain.dependencies {
            implementation(libs.ktor.client.core)
            implementation(libs.ktor.client.content.negotiation)
            implementation(libs.ktor.serialization.kotlinx.json)
            implementation(libs.ktor.client.logging)
            implementation(libs.kotlinx.coroutines.core)
            implementation(libs.kotlinx.serialization.json)
        }
        commonTest.dependencies {
            implementation(libs.kotlin.test)
        }
        androidMain.dependencies {
            implementation(libs.ktor.client.okhttp)
            implementation(libs.kotlinx.coroutines.android)
            implementation(libs.security.crypto)
        }
        iosMain.dependencies {
            implementation(libs.ktor.client.darwin)
        }
    }
}

android {
    namespace = "{{.Package}}.shared"
    compileSdk = 34
    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_1_8
        targetCompatibility = JavaVersion.VERSION_1_8
    }
    defaultConfig {
        minSdk = 24
    }
}
```

### Android App `build.gradle.kts`

```kotlin
plugins {
    alias(libs.plugins.androidApplication)
    alias(libs.plugins.kotlinAndroid)
    alias(libs.plugins.compose.compiler)
}

android {
    namespace = "{{.Package}}.android"
    compileSdk = 34
    defaultConfig {
        applicationId = "{{.Package}}"
        minSdk = 24
        targetSdk = 34
        versionCode = 1
        versionName = "1.0"
    }
    buildFeatures {
        compose = true
        buildConfig = true
    }
    packaging {
        resources {
            excludes += "/META-INF/{AL2.0,LGPL2.1}"
        }
    }
    buildTypes {
        getByName("release") {
            isMinifyEnabled = false
        }
    }
    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_1_8
        targetCompatibility = JavaVersion.VERSION_1_8
    }
    kotlinOptions {
        jvmTarget = "1.8"
    }
}

dependencies {
    implementation(projects.shared)
    implementation(libs.compose.ui)
    implementation(libs.compose.ui.tooling.preview)
    implementation(libs.compose.material3)
    implementation(libs.compose.foundation)
    implementation(libs.androidx.activity.compose)
    implementation(libs.navigation.compose)
    implementation(libs.kotlinx.coroutines.android)
    debugImplementation(libs.compose.ui.tooling)
}
```

## Testing

### Common Tests (`shared/src/commonTest/kotlin/.../RuntimeTest.kt`)

```kotlin
package {{.Package}}

import {{.Package}}.runtime.*
import kotlin.test.*

class RuntimeTest {
    @Test
    fun inMemoryTokenStore_returnsNullWhenNoTokenStored() {
        val store = InMemoryTokenStore()
        assertNull(store.getToken())
    }

    @Test
    fun inMemoryTokenStore_storesAndRetrievesToken() = runTest {
        val store = InMemoryTokenStore()
        val token = AuthToken(
            accessToken = "test123",
            refreshToken = "refresh456"
        )

        store.setToken(token)
        val retrieved = store.getToken()

        assertNotNull(retrieved)
        assertEquals("test123", retrieved.accessToken)
        assertEquals("refresh456", retrieved.refreshToken)
    }

    @Test
    fun inMemoryTokenStore_clearsToken() = runTest {
        val store = InMemoryTokenStore()
        val token = AuthToken(accessToken = "test123")

        store.setToken(token)
        store.clearToken()

        assertNull(store.getToken())
    }

    @Test
    fun inMemoryTokenStore_notifiesObserversOnTokenChange() = runTest {
        val store = InMemoryTokenStore()
        var notifiedToken: AuthToken? = null
        store.addObserver { token -> notifiedToken = token }

        val token = AuthToken(accessToken = "test123")
        store.setToken(token)

        assertNotNull(notifiedToken)
        assertEquals("test123", notifiedToken?.accessToken)
    }

    @Test
    fun authToken_isExpiredReturnsFalseWhenNoExpiry() {
        val token = AuthToken(accessToken = "test")
        assertFalse(token.isExpired)
    }

    @Test
    fun mizuError_createsNetworkError() {
        val error = MizuError.Network("Connection failed")
        assertTrue(error.isNetwork)
        assertEquals("Connection failed", error.message)
    }

    @Test
    fun mizuError_createsApiError() {
        val apiError = ApiError(code = "test_error", message = "Test message")
        val error = MizuError.Api(apiError)
        assertTrue(error.isApi)
        assertEquals("test_error", (error as MizuError.Api).error.code)
    }
}

// Test helper
expect fun runTest(block: suspend () -> Unit)
```

## References

- [Kotlin Multiplatform Documentation](https://kotlinlang.org/docs/multiplatform.html)
- [Ktor Client](https://ktor.io/docs/getting-started-ktor-client.html)
- [Kotlin Serialization](https://github.com/Kotlin/kotlinx.serialization)
- [Jetpack Compose](https://developer.android.com/jetpack/compose)
- [SwiftUI](https://developer.apple.com/documentation/swiftui/)
- [Mobile Package Spec](./0095_mobile.md)
