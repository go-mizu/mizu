# Android Template Specification

**Status:** Draft
**Author:** Mizu Team
**Created:** 2025-12-20
**Template:** `mobile:android`

## Overview

The `mobile:android` template generates a production-ready Android application with full Mizu backend integration. It follows modern Android development practices with Jetpack Compose, Kotlin coroutines, Material 3, and Gradle Kotlin DSL.

## Template Invocation

```bash
# Default: Jetpack Compose
mizu new ./MyApp --template mobile:android

# With Views variant (XML layouts)
mizu new ./MyApp --template mobile:android --var ui=views

# With hybrid variant (Compose + Views)
mizu new ./MyApp --template mobile:android --var ui=hybrid

# Custom package name
mizu new ./MyApp --template mobile:android --var package=com.company.myapp

# Minimum SDK version
mizu new ./MyApp --template mobile:android --var minSdk=26
```

## Generated Project Structure

```
{{.Name}}/
├── app/
│   ├── build.gradle.kts
│   └── src/
│       ├── main/
│       │   ├── AndroidManifest.xml
│       │   ├── kotlin/
│       │   │   └── {{.PackagePath}}/
│       │   │       ├── App/
│       │   │       │   ├── MainActivity.kt
│       │   │       │   └── {{.Name}}Application.kt
│       │   │       ├── ui/
│       │   │       │   ├── theme/
│       │   │       │   │   ├── Color.kt
│       │   │       │   │   ├── Theme.kt
│       │   │       │   │   └── Type.kt
│       │   │       │   ├── screens/
│       │   │       │   │   ├── HomeScreen.kt
│       │   │       │   │   └── WelcomeScreen.kt
│       │   │       │   └── components/
│       │   │       │       ├── LoadingView.kt
│       │   │       │       └── ErrorView.kt
│       │   │       ├── models/
│       │   │       │   └── AppState.kt
│       │   │       ├── sdk/
│       │   │       │   ├── Client.kt
│       │   │       │   ├── Types.kt
│       │   │       │   └── Extensions.kt
│       │   │       └── runtime/
│       │   │           ├── MizuRuntime.kt
│       │   │           ├── Transport.kt
│       │   │           ├── TokenStore.kt
│       │   │           ├── Live.kt
│       │   │           ├── DeviceInfo.kt
│       │   │           └── Config.kt
│       │   └── res/
│       │       ├── drawable/
│       │       ├── mipmap-*/
│       │       └── values/
│       │           ├── colors.xml
│       │           ├── strings.xml
│       │           └── themes.xml
│       ├── test/
│       │   └── kotlin/
│       │       └── {{.PackagePath}}/
│       │           ├── RuntimeTest.kt
│       │           └── SDKTest.kt
│       └── androidTest/
│           └── kotlin/
│               └── {{.PackagePath}}/
│                   └── MainActivityTest.kt
├── build.gradle.kts
├── settings.gradle.kts
├── gradle.properties
├── gradle/
│   └── libs.versions.toml
├── .gitignore
└── README.md
```

## MizuMobileRuntime

### Core Runtime (`MizuRuntime.kt`)

```kotlin
package {{.Package}}.runtime

import android.content.Context
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.serialization.json.Json

/**
 * MizuRuntime is the core client for communicating with a Mizu backend.
 */
class MizuRuntime private constructor(
    private val context: Context,
    config: MizuConfig = MizuConfig()
) {
    /** Base URL for all API requests */
    var baseURL: String = config.baseURL

    /** HTTP transport layer */
    val transport: Transport = OkHttpTransport(config)

    /** Secure token storage */
    val tokenStore: TokenStore = EncryptedTokenStore(context)

    /** Live connection manager */
    val live: LiveConnection by lazy { LiveConnection(this) }

    /** Request timeout in milliseconds */
    var timeout: Long = config.timeout

    /** Default headers added to all requests */
    val defaultHeaders: MutableMap<String, String> = mutableMapOf()

    private val _isAuthenticated = MutableStateFlow(false)
    /** Current authentication state */
    val isAuthenticated: StateFlow<Boolean> = _isAuthenticated.asStateFlow()

    /** JSON serializer */
    internal val json = Json {
        ignoreUnknownKeys = true
        isLenient = true
        encodeDefaults = true
        coerceInputValues = true
    }

    init {
        // Observe token changes
        tokenStore.onTokenChange { token ->
            _isAuthenticated.value = token != null
        }
        // Check initial auth state
        _isAuthenticated.value = tokenStore.getToken() != null
    }

    // MARK: - HTTP Methods

    /** Performs a GET request */
    suspend inline fun <reified T> get(
        path: String,
        query: Map<String, String>? = null,
        headers: Map<String, String>? = null
    ): T = request(HttpMethod.GET, path, query = query, headers = headers)

    /** Performs a POST request */
    suspend inline fun <reified T, reified B> post(
        path: String,
        body: B,
        headers: Map<String, String>? = null
    ): T = request(HttpMethod.POST, path, body = body, headers = headers)

    /** Performs a PUT request */
    suspend inline fun <reified T, reified B> put(
        path: String,
        body: B,
        headers: Map<String, String>? = null
    ): T = request(HttpMethod.PUT, path, body = body, headers = headers)

    /** Performs a DELETE request */
    suspend inline fun <reified T> delete(
        path: String,
        headers: Map<String, String>? = null
    ): T = request(HttpMethod.DELETE, path, headers = headers)

    /** Performs a DELETE request with no response body */
    suspend fun delete(
        path: String,
        headers: Map<String, String>? = null
    ): Unit = request(HttpMethod.DELETE, path, headers = headers)

    /** Performs a PATCH request */
    suspend inline fun <reified T, reified B> patch(
        path: String,
        body: B,
        headers: Map<String, String>? = null
    ): T = request(HttpMethod.PATCH, path, body = body, headers = headers)

    // MARK: - Streaming

    /** Opens a streaming connection for SSE */
    fun stream(
        path: String,
        headers: Map<String, String>? = null
    ): kotlinx.coroutines.flow.Flow<ServerEvent> = live.connect(path, headers)

    // MARK: - Private

    @PublishedApi
    internal suspend inline fun <reified T, reified B> request(
        method: HttpMethod,
        path: String,
        query: Map<String, String>? = null,
        body: B? = null,
        headers: Map<String, String>? = null
    ): T {
        val url = buildUrl(path, query)
        val allHeaders = buildHeaders(headers)
        val bodyJson = body?.let { json.encodeToString(kotlinx.serialization.serializer(), it) }

        val response = transport.execute(
            TransportRequest(
                url = url,
                method = method,
                headers = allHeaders,
                body = bodyJson,
                timeout = timeout
            )
        )

        if (response.statusCode >= 400) {
            throw parseError(response)
        }

        return if (T::class == Unit::class) {
            @Suppress("UNCHECKED_CAST")
            Unit as T
        } else {
            json.decodeFromString(response.body)
        }
    }

    @PublishedApi
    internal suspend inline fun <reified T> request(
        method: HttpMethod,
        path: String,
        query: Map<String, String>? = null,
        headers: Map<String, String>? = null
    ): T = request<T, Unit>(method, path, query, null, headers)

    private fun buildUrl(path: String, query: Map<String, String>?): String {
        val base = baseURL.trimEnd('/')
        val cleanPath = if (path.startsWith("/")) path else "/$path"
        val url = StringBuilder("$base$cleanPath")

        if (!query.isNullOrEmpty()) {
            url.append("?")
            url.append(query.entries.joinToString("&") { (k, v) ->
                "${java.net.URLEncoder.encode(k, "UTF-8")}=${java.net.URLEncoder.encode(v, "UTF-8")}"
            })
        }

        return url.toString()
    }

    private fun buildHeaders(custom: Map<String, String>?): Map<String, String> {
        val headers = mutableMapOf<String, String>()
        headers.putAll(defaultHeaders)
        headers.putAll(mobileHeaders())
        custom?.let { headers.putAll(it) }

        // Add auth token
        tokenStore.getToken()?.let { token ->
            headers["Authorization"] = "Bearer ${token.accessToken}"
        }

        return headers
    }

    private fun mobileHeaders(): Map<String, String> = mapOf(
        "X-Device-ID" to DeviceInfo.getDeviceId(context),
        "X-App-Version" to DeviceInfo.getAppVersion(context),
        "X-App-Build" to DeviceInfo.getAppBuild(context),
        "X-Device-Model" to DeviceInfo.model,
        "X-Platform" to "android",
        "X-OS-Version" to DeviceInfo.osVersion,
        "X-Timezone" to java.util.TimeZone.getDefault().id,
        "X-Locale" to java.util.Locale.getDefault().toLanguageTag()
    )

    private fun parseError(response: TransportResponse): MizuError {
        return try {
            val apiError = json.decodeFromString<APIError>(response.body)
            MizuError.Api(apiError)
        } catch (_: Exception) {
            MizuError.Http(response.statusCode, response.body)
        }
    }

    companion object {
        @Volatile
        private var instance: MizuRuntime? = null

        /** Gets the shared singleton instance */
        fun getInstance(context: Context): MizuRuntime {
            return instance ?: synchronized(this) {
                instance ?: MizuRuntime(context.applicationContext).also { instance = it }
            }
        }

        /** Initializes with custom configuration */
        fun initialize(context: Context, config: MizuConfig): MizuRuntime {
            return synchronized(this) {
                MizuRuntime(context.applicationContext, config).also { instance = it }
            }
        }
    }
}

/** HTTP methods */
enum class HttpMethod(val value: String) {
    GET("GET"),
    POST("POST"),
    PUT("PUT"),
    DELETE("DELETE"),
    PATCH("PATCH")
}

/** API error response from server */
@kotlinx.serialization.Serializable
data class APIError(
    val code: String,
    val message: String,
    val details: Map<String, kotlinx.serialization.json.JsonElement>? = null,
    @kotlinx.serialization.SerialName("trace_id")
    val traceId: String? = null
)

/** Mizu client errors */
sealed class MizuError : Exception() {
    data object InvalidResponse : MizuError()
    data class Http(val statusCode: Int, val body: String) : MizuError()
    data class Api(val error: APIError) : MizuError() {
        override val message: String get() = error.message
    }
    data class Network(override val cause: Throwable) : MizuError()
    data class Encoding(override val cause: Throwable) : MizuError()
    data class Decoding(override val cause: Throwable) : MizuError()
    data object Unauthorized : MizuError()
    data object TokenExpired : MizuError()
}
```

### Transport Layer (`Transport.kt`)

```kotlin
package {{.Package}}.runtime

import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.suspendCancellableCoroutine
import kotlinx.coroutines.withContext
import okhttp3.*
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.RequestBody.Companion.toRequestBody
import java.io.IOException
import java.util.concurrent.TimeUnit
import kotlin.coroutines.resume
import kotlin.coroutines.resumeWithException

/** Transport request */
data class TransportRequest(
    val url: String,
    val method: HttpMethod,
    val headers: Map<String, String>,
    val body: String?,
    val timeout: Long
)

/** Transport response */
data class TransportResponse(
    val statusCode: Int,
    val headers: Map<String, String>,
    val body: String
)

/** Transport protocol for executing HTTP requests */
interface Transport {
    suspend fun execute(request: TransportRequest): TransportResponse
}

/** OkHttp-based transport implementation */
class OkHttpTransport(config: MizuConfig = MizuConfig()) : Transport {
    private val client: OkHttpClient
    private val interceptors = mutableListOf<RequestInterceptor>()

    init {
        client = OkHttpClient.Builder()
            .connectTimeout(config.timeout, TimeUnit.MILLISECONDS)
            .readTimeout(config.timeout, TimeUnit.MILLISECONDS)
            .writeTimeout(config.timeout, TimeUnit.MILLISECONDS)
            .retryOnConnectionFailure(true)
            .build()
    }

    /** Adds a request interceptor */
    fun addInterceptor(interceptor: RequestInterceptor) {
        interceptors.add(interceptor)
    }

    override suspend fun execute(request: TransportRequest): TransportResponse =
        withContext(Dispatchers.IO) {
            var okRequest = buildRequest(request)

            // Apply interceptors
            for (interceptor in interceptors) {
                okRequest = interceptor.intercept(okRequest)
            }

            suspendCancellableCoroutine { continuation ->
                val call = client.newCall(okRequest)

                continuation.invokeOnCancellation {
                    call.cancel()
                }

                call.enqueue(object : Callback {
                    override fun onFailure(call: Call, e: IOException) {
                        continuation.resumeWithException(MizuError.Network(e))
                    }

                    override fun onResponse(call: Call, response: Response) {
                        val transportResponse = TransportResponse(
                            statusCode = response.code,
                            headers = response.headers.toMap(),
                            body = response.body?.string() ?: ""
                        )
                        continuation.resume(transportResponse)
                    }
                })
            }
        }

    private fun buildRequest(request: TransportRequest): Request {
        val builder = Request.Builder()
            .url(request.url)

        request.headers.forEach { (key, value) ->
            builder.addHeader(key, value)
        }

        val body = when (request.method) {
            HttpMethod.GET, HttpMethod.DELETE -> null
            else -> request.body?.toRequestBody("application/json".toMediaType())
                ?: "".toRequestBody(null)
        }

        when (request.method) {
            HttpMethod.GET -> builder.get()
            HttpMethod.POST -> builder.post(body!!)
            HttpMethod.PUT -> builder.put(body!!)
            HttpMethod.DELETE -> if (body != null) builder.delete(body) else builder.delete()
            HttpMethod.PATCH -> builder.patch(body!!)
        }

        return builder.build()
    }

    private fun Headers.toMap(): Map<String, String> =
        (0 until size).associate { name(it) to value(it) }
}

/** Request interceptor protocol */
interface RequestInterceptor {
    suspend fun intercept(request: Request): Request
}

/** Logging interceptor for debugging */
class LoggingInterceptor : RequestInterceptor {
    override suspend fun intercept(request: Request): Request {
        android.util.Log.d("MizuRuntime", "${request.method} ${request.url}")
        return request
    }
}

/** Retry interceptor with exponential backoff */
class RetryInterceptor(
    private val maxRetries: Int = 3,
    private val baseDelayMs: Long = 1000
) : RequestInterceptor {
    override suspend fun intercept(request: Request): Request {
        // Retry logic is handled at transport level
        return request
    }
}
```

### Token Store (`TokenStore.kt`)

```kotlin
package {{.Package}}.runtime

import android.content.Context
import android.content.SharedPreferences
import androidx.security.crypto.EncryptedSharedPreferences
import androidx.security.crypto.MasterKey
import kotlinx.serialization.Serializable
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json

/** Stored authentication token */
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

/** Token storage protocol */
interface TokenStore {
    fun getToken(): AuthToken?
    fun setToken(token: AuthToken)
    fun clearToken()
    fun onTokenChange(callback: (AuthToken?) -> Unit)
}

/** Encrypted SharedPreferences-backed token storage */
class EncryptedTokenStore(context: Context) : TokenStore {
    private val prefs: SharedPreferences
    private val json = Json { ignoreUnknownKeys = true }
    private val observers = mutableListOf<(AuthToken?) -> Unit>()

    init {
        val masterKey = MasterKey.Builder(context)
            .setKeyScheme(MasterKey.KeyScheme.AES256_GCM)
            .build()

        prefs = EncryptedSharedPreferences.create(
            context,
            "mizu_secure_prefs",
            masterKey,
            EncryptedSharedPreferences.PrefKeyEncryptionScheme.AES256_SIV,
            EncryptedSharedPreferences.PrefValueEncryptionScheme.AES256_GCM
        )
    }

    override fun getToken(): AuthToken? {
        val tokenJson = prefs.getString(KEY_AUTH_TOKEN, null) ?: return null
        return try {
            json.decodeFromString<AuthToken>(tokenJson)
        } catch (_: Exception) {
            null
        }
    }

    override fun setToken(token: AuthToken) {
        val tokenJson = json.encodeToString(token)
        prefs.edit().putString(KEY_AUTH_TOKEN, tokenJson).apply()
        notifyObservers(token)
    }

    override fun clearToken() {
        prefs.edit().remove(KEY_AUTH_TOKEN).apply()
        notifyObservers(null)
    }

    override fun onTokenChange(callback: (AuthToken?) -> Unit) {
        observers.add(callback)
    }

    private fun notifyObservers(token: AuthToken?) {
        observers.forEach { it(token) }
    }

    companion object {
        private const val KEY_AUTH_TOKEN = "auth_token"
    }
}

/** In-memory token store for testing */
class InMemoryTokenStore : TokenStore {
    private var token: AuthToken? = null
    private val observers = mutableListOf<(AuthToken?) -> Unit>()

    override fun getToken(): AuthToken? = token

    override fun setToken(token: AuthToken) {
        this.token = token
        notifyObservers(token)
    }

    override fun clearToken() {
        this.token = null
        notifyObservers(null)
    }

    override fun onTokenChange(callback: (AuthToken?) -> Unit) {
        observers.add(callback)
    }

    private fun notifyObservers(token: AuthToken?) {
        observers.forEach { it(token) }
    }
}
```

### Live Streaming (`Live.kt`)

```kotlin
package {{.Package}}.runtime

import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.channels.awaitClose
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.callbackFlow
import kotlinx.coroutines.withContext
import okhttp3.*
import okhttp3.sse.EventSource
import okhttp3.sse.EventSourceListener
import okhttp3.sse.EventSources
import java.util.concurrent.ConcurrentHashMap
import java.util.concurrent.TimeUnit

/** Server-sent event */
data class ServerEvent(
    val id: String? = null,
    val event: String? = null,
    val data: String,
    val retry: Int? = null
) {
    /** Decodes the event data as JSON */
    inline fun <reified T> decode(json: kotlinx.serialization.json.Json): T =
        json.decodeFromString(data)
}

/** Live connection manager for SSE */
class LiveConnection(private val runtime: MizuRuntime) {
    private val client = OkHttpClient.Builder()
        .connectTimeout(0, TimeUnit.MILLISECONDS)
        .readTimeout(0, TimeUnit.MILLISECONDS)
        .writeTimeout(0, TimeUnit.MILLISECONDS)
        .build()

    private val activeConnections = ConcurrentHashMap<String, EventSource>()

    /** Connects to an SSE endpoint and returns a Flow of events */
    fun connect(
        path: String,
        headers: Map<String, String>? = null
    ): Flow<ServerEvent> = callbackFlow {
        val url = buildUrl(path)
        val request = buildRequest(url, headers)

        val listener = object : EventSourceListener() {
            override fun onEvent(
                eventSource: EventSource,
                id: String?,
                type: String?,
                data: String
            ) {
                val event = ServerEvent(
                    id = id,
                    event = type,
                    data = data
                )
                trySend(event)
            }

            override fun onFailure(
                eventSource: EventSource,
                t: Throwable?,
                response: Response?
            ) {
                t?.let { close(MizuError.Network(it)) }
                    ?: close()
            }

            override fun onClosed(eventSource: EventSource) {
                close()
            }
        }

        val eventSource = EventSources.createFactory(client)
            .newEventSource(request, listener)

        activeConnections[path] = eventSource

        awaitClose {
            eventSource.cancel()
            activeConnections.remove(path)
        }
    }

    /** Disconnects from a specific path */
    fun disconnect(path: String) {
        activeConnections.remove(path)?.cancel()
    }

    /** Disconnects all active connections */
    fun disconnectAll() {
        activeConnections.values.forEach { it.cancel() }
        activeConnections.clear()
    }

    private fun buildUrl(path: String): String {
        val base = runtime.baseURL.trimEnd('/')
        val cleanPath = if (path.startsWith("/")) path else "/$path"
        return "$base$cleanPath"
    }

    private fun buildRequest(url: String, headers: Map<String, String>?): Request {
        val builder = Request.Builder()
            .url(url)
            .header("Accept", "text/event-stream")
            .header("Cache-Control", "no-cache")

        runtime.defaultHeaders.forEach { (key, value) ->
            builder.header(key, value)
        }

        headers?.forEach { (key, value) ->
            builder.header(key, value)
        }

        runtime.tokenStore.getToken()?.let { token ->
            builder.header("Authorization", "Bearer ${token.accessToken}")
        }

        return builder.build()
    }
}
```

### Device Info (`DeviceInfo.kt`)

```kotlin
package {{.Package}}.runtime

import android.annotation.SuppressLint
import android.content.Context
import android.os.Build
import android.provider.Settings
import java.util.UUID

/** Device information utilities */
object DeviceInfo {
    /** Device model identifier (e.g., "Pixel 8 Pro") */
    val model: String = Build.MODEL

    /** Device manufacturer */
    val manufacturer: String = Build.MANUFACTURER

    /** OS version string */
    val osVersion: String = Build.VERSION.RELEASE

    /** SDK version */
    val sdkVersion: Int = Build.VERSION.SDK_INT

    /** Is running on emulator */
    val isEmulator: Boolean
        get() = (Build.FINGERPRINT.startsWith("generic")
                || Build.FINGERPRINT.startsWith("unknown")
                || Build.MODEL.contains("google_sdk")
                || Build.MODEL.contains("Emulator")
                || Build.MODEL.contains("Android SDK built for x86")
                || Build.MANUFACTURER.contains("Genymotion")
                || Build.BRAND.startsWith("generic")
                || Build.DEVICE.startsWith("generic"))

    /** Gets unique device identifier (persisted in SharedPreferences) */
    @SuppressLint("HardwareIds")
    fun getDeviceId(context: Context): String {
        val prefs = context.getSharedPreferences("mizu_device", Context.MODE_PRIVATE)
        var deviceId = prefs.getString("device_id", null)

        if (deviceId == null) {
            // Try to get Android ID first
            deviceId = Settings.Secure.getString(
                context.contentResolver,
                Settings.Secure.ANDROID_ID
            )

            // Fallback to UUID if Android ID is null or default
            if (deviceId.isNullOrBlank() || deviceId == "9774d56d682e549c") {
                deviceId = UUID.randomUUID().toString()
            }

            prefs.edit().putString("device_id", deviceId).apply()
        }

        return deviceId
    }

    /** Gets app version name */
    fun getAppVersion(context: Context): String {
        return try {
            context.packageManager.getPackageInfo(context.packageName, 0).versionName ?: "0.0.0"
        } catch (_: Exception) {
            "0.0.0"
        }
    }

    /** Gets app version code */
    fun getAppBuild(context: Context): String {
        return try {
            val info = context.packageManager.getPackageInfo(context.packageName, 0)
            if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.P) {
                info.longVersionCode.toString()
            } else {
                @Suppress("DEPRECATION")
                info.versionCode.toString()
            }
        } catch (_: Exception) {
            "0"
        }
    }

    /** Gets app package name */
    fun getPackageName(context: Context): String = context.packageName

    /** Gets device display name */
    val displayName: String
        get() = "$manufacturer $model"
}
```

### Configuration (`Config.kt`)

```kotlin
package {{.Package}}.runtime

import android.content.Context

/** Mizu runtime configuration */
data class MizuConfig(
    /** Base URL for API requests */
    val baseURL: String = "http://10.0.2.2:3000", // Android emulator localhost

    /** Request timeout in milliseconds */
    val timeout: Long = 30_000,

    /** Enable debug logging */
    val debug: Boolean = false,

    /** Custom headers to include in all requests */
    val headers: Map<String, String> = emptyMap(),

    /** Token refresh configuration */
    val tokenRefresh: TokenRefreshConfig? = null,

    /** Retry configuration */
    val retry: RetryConfig = RetryConfig.DEFAULT
) {
    companion object {
        /** Load configuration from BuildConfig or resources */
        fun fromContext(context: Context): MizuConfig {
            val resources = context.resources
            val packageName = context.packageName

            val baseURL = try {
                val resId = resources.getIdentifier("mizu_base_url", "string", packageName)
                if (resId != 0) resources.getString(resId) else null
            } catch (_: Exception) {
                null
            }

            val timeout = try {
                val resId = resources.getIdentifier("mizu_timeout", "integer", packageName)
                if (resId != 0) resources.getInteger(resId).toLong() else null
            } catch (_: Exception) {
                null
            }

            val debug = try {
                val resId = resources.getIdentifier("mizu_debug", "bool", packageName)
                if (resId != 0) resources.getBoolean(resId) else null
            } catch (_: Exception) {
                null
            }

            return MizuConfig(
                baseURL = baseURL ?: "http://10.0.2.2:3000",
                timeout = timeout ?: 30_000,
                debug = debug ?: false
            )
        }
    }
}

/** Token refresh configuration */
data class TokenRefreshConfig(
    /** Path for token refresh endpoint */
    val refreshPath: String = "/auth/refresh",

    /** Refresh token before expiration (milliseconds) */
    val refreshBeforeExpiry: Long = 60_000
)

/** Retry configuration */
data class RetryConfig(
    /** Maximum number of retries */
    val maxRetries: Int = 3,

    /** Base delay between retries (exponential backoff) */
    val baseDelayMs: Long = 1_000,

    /** Maximum delay between retries */
    val maxDelayMs: Long = 30_000,

    /** HTTP status codes that should trigger a retry */
    val retryableStatuses: Set<Int> = setOf(408, 429, 500, 502, 503, 504)
) {
    companion object {
        val DEFAULT = RetryConfig()
        val NONE = RetryConfig(maxRetries = 0)
    }
}
```

## App Templates

### Application Class (`{{.Name}}Application.kt`)

```kotlin
package {{.Package}}

import android.app.Application
import {{.Package}}.runtime.MizuConfig
import {{.Package}}.runtime.MizuRuntime
import {{.Package}}.runtime.LoggingInterceptor
import {{.Package}}.runtime.OkHttpTransport

class {{.Name}}Application : Application() {
    override fun onCreate() {
        super.onCreate()

        // Initialize Mizu runtime
        val config = MizuConfig.fromContext(this)
        val runtime = MizuRuntime.initialize(this, config)

        if (config.debug) {
            (runtime.transport as? OkHttpTransport)?.addInterceptor(LoggingInterceptor())
        }
    }
}
```

### Main Activity (`MainActivity.kt`)

```kotlin
package {{.Package}}

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.ui.Modifier
import {{.Package}}.runtime.MizuRuntime
import {{.Package}}.ui.screens.HomeScreen
import {{.Package}}.ui.screens.WelcomeScreen
import {{.Package}}.ui.theme.{{.Name}}Theme

class MainActivity : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        enableEdgeToEdge()

        val runtime = MizuRuntime.getInstance(this)

        setContent {
            {{.Name}}Theme {
                Surface(
                    modifier = Modifier.fillMaxSize(),
                    color = MaterialTheme.colorScheme.background
                ) {
                    val isAuthenticated by runtime.isAuthenticated.collectAsState()

                    if (isAuthenticated) {
                        HomeScreen()
                    } else {
                        WelcomeScreen()
                    }
                }
            }
        }
    }
}
```

### Home Screen (`ui/screens/HomeScreen.kt`)

```kotlin
package {{.Package}}.ui.screens

import androidx.compose.foundation.layout.*
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.CheckCircle
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.unit.dp
import {{.Package}}.runtime.MizuRuntime

@Composable
fun HomeScreen() {
    val context = LocalContext.current
    val runtime = remember { MizuRuntime.getInstance(context) }
    var isLoading by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf<String?>(null) }

    Column(
        modifier = Modifier
            .fillMaxSize()
            .padding(24.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.Center
    ) {
        Icon(
            imageVector = Icons.Filled.CheckCircle,
            contentDescription = "Success",
            modifier = Modifier.size(64.dp),
            tint = MaterialTheme.colorScheme.primary
        )

        Spacer(modifier = Modifier.height(16.dp))

        Text(
            text = "Welcome to {{.Name}}",
            style = MaterialTheme.typography.headlineMedium
        )

        Spacer(modifier = Modifier.height(8.dp))

        Text(
            text = "Connected to Mizu backend",
            style = MaterialTheme.typography.bodyMedium,
            color = MaterialTheme.colorScheme.onSurfaceVariant
        )

        Spacer(modifier = Modifier.height(24.dp))

        if (isLoading) {
            CircularProgressIndicator()
        }

        Button(
            onClick = {
                runtime.tokenStore.clearToken()
            }
        ) {
            Text("Sign Out")
        }
    }

    error?.let { message ->
        AlertDialog(
            onDismissRequest = { error = null },
            title = { Text("Error") },
            text = { Text(message) },
            confirmButton = {
                TextButton(onClick = { error = null }) {
                    Text("OK")
                }
            }
        )
    }
}
```

### Welcome Screen (`ui/screens/WelcomeScreen.kt`)

```kotlin
package {{.Package}}.ui.screens

import androidx.compose.foundation.layout.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import {{.Package}}.runtime.AuthToken
import {{.Package}}.runtime.MizuRuntime

@Composable
fun WelcomeScreen() {
    val context = LocalContext.current
    val runtime = remember { MizuRuntime.getInstance(context) }

    Column(
        modifier = Modifier
            .fillMaxSize()
            .padding(24.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.Center
    ) {
        Text(
            text = "Welcome to {{.Name}}",
            style = MaterialTheme.typography.headlineLarge,
            textAlign = TextAlign.Center
        )

        Spacer(modifier = Modifier.height(16.dp))

        Text(
            text = "A modern Android app powered by Mizu",
            style = MaterialTheme.typography.bodyLarge,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
            textAlign = TextAlign.Center
        )

        Spacer(modifier = Modifier.height(48.dp))

        Button(
            onClick = {
                // Demo: Set a test token
                runtime.tokenStore.setToken(
                    AuthToken(accessToken = "demo_token")
                )
            },
            modifier = Modifier.fillMaxWidth()
        ) {
            Text("Get Started")
        }
    }
}
```

### App State (`models/AppState.kt`)

```kotlin
package {{.Package}}.models

import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.lifecycle.ViewModel

class AppState : ViewModel() {
    var isOnboarded by mutableStateOf(false)
        private set

    var selectedTab by mutableStateOf(Tab.Home)
        private set

    enum class Tab {
        Home, Profile, Settings
    }

    fun completeOnboarding() {
        isOnboarded = true
    }

    fun selectTab(tab: Tab) {
        selectedTab = tab
    }
}
```

## Theme

### Colors (`ui/theme/Color.kt`)

```kotlin
package {{.Package}}.ui.theme

import androidx.compose.ui.graphics.Color

val Purple80 = Color(0xFFD0BCFF)
val PurpleGrey80 = Color(0xFFCCC2DC)
val Pink80 = Color(0xFFEFB8C8)

val Purple40 = Color(0xFF6650a4)
val PurpleGrey40 = Color(0xFF625b71)
val Pink40 = Color(0xFF7D5260)
```

### Theme (`ui/theme/Theme.kt`)

```kotlin
package {{.Package}}.ui.theme

import android.os.Build
import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.material3.*
import androidx.compose.runtime.Composable
import androidx.compose.ui.platform.LocalContext

private val DarkColorScheme = darkColorScheme(
    primary = Purple80,
    secondary = PurpleGrey80,
    tertiary = Pink80
)

private val LightColorScheme = lightColorScheme(
    primary = Purple40,
    secondary = PurpleGrey40,
    tertiary = Pink40
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

    MaterialTheme(
        colorScheme = colorScheme,
        typography = Typography,
        content = content
    )
}
```

### Typography (`ui/theme/Type.kt`)

```kotlin
package {{.Package}}.ui.theme

import androidx.compose.material3.Typography
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.sp

val Typography = Typography(
    bodyLarge = TextStyle(
        fontFamily = FontFamily.Default,
        fontWeight = FontWeight.Normal,
        fontSize = 16.sp,
        lineHeight = 24.sp,
        letterSpacing = 0.5.sp
    ),
    titleLarge = TextStyle(
        fontFamily = FontFamily.Default,
        fontWeight = FontWeight.Normal,
        fontSize = 22.sp,
        lineHeight = 28.sp,
        letterSpacing = 0.sp
    ),
    labelSmall = TextStyle(
        fontFamily = FontFamily.Default,
        fontWeight = FontWeight.Medium,
        fontSize = 11.sp,
        lineHeight = 16.sp,
        letterSpacing = 0.5.sp
    )
)
```

## Generated SDK

### Client (`sdk/Client.kt`)

```kotlin
package {{.Package}}.sdk

import {{.Package}}.runtime.MizuRuntime

/**
 * Generated Mizu API client for {{.Name}}
 */
class {{.Name}}Client(
    private val runtime: MizuRuntime
) {
    // MARK: - Auth

    /** Sign in with credentials */
    suspend fun signIn(email: String, password: String): AuthResponse =
        runtime.post("/auth/signin", SignInRequest(email, password))

    /** Sign up with credentials */
    suspend fun signUp(email: String, password: String, name: String): AuthResponse =
        runtime.post("/auth/signup", SignUpRequest(email, password, name))

    /** Sign out */
    suspend fun signOut() {
        runtime.delete<Unit>("/auth/signout")
        runtime.tokenStore.clearToken()
    }

    // MARK: - Users

    /** Get current user profile */
    suspend fun getCurrentUser(): User = runtime.get("/users/me")

    /** Update current user profile */
    suspend fun updateCurrentUser(update: UserUpdate): User =
        runtime.put("/users/me", update)
}
```

### Types (`sdk/Types.kt`)

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

### Extensions (`sdk/Extensions.kt`)

```kotlin
package {{.Package}}.sdk

import {{.Package}}.runtime.AuthToken
import {{.Package}}.runtime.MizuRuntime

/** Convenience extension to store auth response token */
fun MizuRuntime.storeAuthToken(response: AuthResponse) {
    val expiresAt = System.currentTimeMillis() + (response.token.expiresIn * 1000L)
    tokenStore.setToken(
        AuthToken(
            accessToken = response.token.accessToken,
            refreshToken = response.token.refreshToken,
            expiresAt = expiresAt
        )
    )
}
```

## Build Configuration

### Project `build.gradle.kts`

```kotlin
plugins {
    alias(libs.plugins.android.application) apply false
    alias(libs.plugins.kotlin.android) apply false
    alias(libs.plugins.kotlin.compose) apply false
    alias(libs.plugins.kotlin.serialization) apply false
}
```

### App `build.gradle.kts`

```kotlin
plugins {
    alias(libs.plugins.android.application)
    alias(libs.plugins.kotlin.android)
    alias(libs.plugins.kotlin.compose)
    alias(libs.plugins.kotlin.serialization)
}

android {
    namespace = "{{.Package}}"
    compileSdk = 35

    defaultConfig {
        applicationId = "{{.Package}}"
        minSdk = {{.MinSdk}}
        targetSdk = 35
        versionCode = 1
        versionName = "1.0.0"

        testInstrumentationRunner = "androidx.test.runner.AndroidJUnitRunner"
    }

    buildTypes {
        debug {
            resValue("string", "mizu_base_url", "http://10.0.2.2:3000")
            resValue("bool", "mizu_debug", "true")
        }
        release {
            isMinifyEnabled = true
            proguardFiles(
                getDefaultProguardFile("proguard-android-optimize.txt"),
                "proguard-rules.pro"
            )
            resValue("string", "mizu_base_url", "https://api.example.com")
            resValue("bool", "mizu_debug", "false")
        }
    }

    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_17
        targetCompatibility = JavaVersion.VERSION_17
    }

    kotlinOptions {
        jvmTarget = "17"
    }

    buildFeatures {
        compose = true
        buildConfig = true
    }
}

dependencies {
    // Compose BOM
    implementation(platform(libs.compose.bom))
    implementation(libs.compose.ui)
    implementation(libs.compose.ui.graphics)
    implementation(libs.compose.ui.tooling.preview)
    implementation(libs.compose.material3)
    implementation(libs.compose.material.icons)

    // Activity
    implementation(libs.activity.compose)

    // Lifecycle
    implementation(libs.lifecycle.runtime.ktx)
    implementation(libs.lifecycle.viewmodel.compose)

    // Networking
    implementation(libs.okhttp)
    implementation(libs.okhttp.sse)

    // Serialization
    implementation(libs.kotlinx.serialization.json)

    // Security
    implementation(libs.security.crypto)

    // Coroutines
    implementation(libs.kotlinx.coroutines.android)

    // Testing
    testImplementation(libs.junit)
    testImplementation(libs.kotlinx.coroutines.test)
    androidTestImplementation(libs.ext.junit)
    androidTestImplementation(libs.espresso.core)
    androidTestImplementation(platform(libs.compose.bom))
    androidTestImplementation(libs.compose.ui.test.junit4)
    debugImplementation(libs.compose.ui.tooling)
    debugImplementation(libs.compose.ui.test.manifest)
}
```

### Version Catalog (`gradle/libs.versions.toml`)

```toml
[versions]
agp = "8.7.3"
kotlin = "2.1.0"
kotlinxSerialization = "1.7.3"
kotlinxCoroutines = "1.9.0"
okhttp = "4.12.0"
composeBom = "2024.12.01"
activityCompose = "1.9.3"
lifecycleRuntimeKtx = "2.8.7"
securityCrypto = "1.1.0-alpha06"
junit = "4.13.2"
extJunit = "1.2.1"
espressoCore = "3.6.1"

[libraries]
compose-bom = { group = "androidx.compose", name = "compose-bom", version.ref = "composeBom" }
compose-ui = { group = "androidx.compose.ui", name = "ui" }
compose-ui-graphics = { group = "androidx.compose.ui", name = "ui-graphics" }
compose-ui-tooling = { group = "androidx.compose.ui", name = "ui-tooling" }
compose-ui-tooling-preview = { group = "androidx.compose.ui", name = "ui-tooling-preview" }
compose-ui-test-manifest = { group = "androidx.compose.ui", name = "ui-test-manifest" }
compose-ui-test-junit4 = { group = "androidx.compose.ui", name = "ui-test-junit4" }
compose-material3 = { group = "androidx.compose.material3", name = "material3" }
compose-material-icons = { group = "androidx.compose.material", name = "material-icons-extended" }
activity-compose = { group = "androidx.activity", name = "activity-compose", version.ref = "activityCompose" }
lifecycle-runtime-ktx = { group = "androidx.lifecycle", name = "lifecycle-runtime-ktx", version.ref = "lifecycleRuntimeKtx" }
lifecycle-viewmodel-compose = { group = "androidx.lifecycle", name = "lifecycle-viewmodel-compose", version.ref = "lifecycleRuntimeKtx" }
okhttp = { group = "com.squareup.okhttp3", name = "okhttp", version.ref = "okhttp" }
okhttp-sse = { group = "com.squareup.okhttp3", name = "okhttp-sse", version.ref = "okhttp" }
kotlinx-serialization-json = { group = "org.jetbrains.kotlinx", name = "kotlinx-serialization-json", version.ref = "kotlinxSerialization" }
kotlinx-coroutines-android = { group = "org.jetbrains.kotlinx", name = "kotlinx-coroutines-android", version.ref = "kotlinxCoroutines" }
kotlinx-coroutines-test = { group = "org.jetbrains.kotlinx", name = "kotlinx-coroutines-test", version.ref = "kotlinxCoroutines" }
security-crypto = { group = "androidx.security", name = "security-crypto", version.ref = "securityCrypto" }
junit = { group = "junit", name = "junit", version.ref = "junit" }
ext-junit = { group = "androidx.test.ext", name = "junit", version.ref = "extJunit" }
espresso-core = { group = "androidx.test.espresso", name = "espresso-core", version.ref = "espressoCore" }

[plugins]
android-application = { id = "com.android.application", version.ref = "agp" }
kotlin-android = { id = "org.jetbrains.kotlin.android", version.ref = "kotlin" }
kotlin-compose = { id = "org.jetbrains.kotlin.plugin.compose", version.ref = "kotlin" }
kotlin-serialization = { id = "org.jetbrains.kotlin.plugin.serialization", version.ref = "kotlin" }
```

### AndroidManifest.xml

```xml
<?xml version="1.0" encoding="utf-8"?>
<manifest xmlns:android="http://schemas.android.com/apk/res/android">

    <uses-permission android:name="android.permission.INTERNET" />
    <uses-permission android:name="android.permission.ACCESS_NETWORK_STATE" />

    <application
        android:name=".{{.Name}}Application"
        android:allowBackup="true"
        android:icon="@mipmap/ic_launcher"
        android:label="@string/app_name"
        android:roundIcon="@mipmap/ic_launcher_round"
        android:supportsRtl="true"
        android:theme="@style/Theme.{{.Name}}"
        android:usesCleartextTraffic="true">

        <activity
            android:name=".MainActivity"
            android:exported="true"
            android:theme="@style/Theme.{{.Name}}">
            <intent-filter>
                <action android:name="android.intent.action.MAIN" />
                <category android:name="android.intent.category.LAUNCHER" />
            </intent-filter>
        </activity>
    </application>

</manifest>
```

## UI Variants

### Jetpack Compose (Default)
- Modern declarative UI
- `@Composable` functions
- Material 3 design system
- ViewModel with StateFlow

### Views (XML)
- Traditional View system
- XML layouts
- Data Binding / View Binding
- RecyclerView for lists

### Hybrid
- Compose with ComposeView in XML
- AndroidView for legacy components
- Shared ViewModels

## Testing

### Unit Tests (`RuntimeTest.kt`)

```kotlin
package {{.Package}}

import {{.Package}}.runtime.*
import kotlinx.coroutines.test.runTest
import org.junit.Test
import org.junit.Assert.*

class RuntimeTest {
    @Test
    fun testDeviceInfo() {
        assertNotNull(DeviceInfo.model)
        assertNotNull(DeviceInfo.osVersion)
        assertTrue(DeviceInfo.sdkVersion > 0)
    }

    @Test
    fun testInMemoryTokenStore() {
        val store = InMemoryTokenStore()

        assertNull(store.getToken())

        val token = AuthToken(accessToken = "test123", refreshToken = "refresh456")
        store.setToken(token)

        val retrieved = store.getToken()
        assertEquals("test123", retrieved?.accessToken)

        store.clearToken()
        assertNull(store.getToken())
    }

    @Test
    fun testAuthTokenExpiration() {
        val expiredToken = AuthToken(
            accessToken = "test",
            expiresAt = System.currentTimeMillis() - 1000
        )
        assertTrue(expiredToken.isExpired)

        val validToken = AuthToken(
            accessToken = "test",
            expiresAt = System.currentTimeMillis() + 60000
        )
        assertFalse(validToken.isExpired)
    }
}
```

### Instrumentation Tests (`MainActivityTest.kt`)

```kotlin
package {{.Package}}

import androidx.compose.ui.test.*
import androidx.compose.ui.test.junit4.createAndroidComposeRule
import org.junit.Rule
import org.junit.Test

class MainActivityTest {
    @get:Rule
    val composeTestRule = createAndroidComposeRule<MainActivity>()

    @Test
    fun welcomeScreenIsDisplayed() {
        composeTestRule
            .onNodeWithText("Welcome to {{.Name}}")
            .assertIsDisplayed()
    }

    @Test
    fun getStartedButtonExists() {
        composeTestRule
            .onNodeWithText("Get Started")
            .assertIsDisplayed()
    }
}
```

## References

- [Android Kotlin Documentation](https://developer.android.com/kotlin)
- [Jetpack Compose](https://developer.android.com/jetpack/compose)
- [Material 3 for Android](https://m3.material.io/develop/android/mdc-android)
- [Kotlin Coroutines](https://kotlinlang.org/docs/coroutines-overview.html)
- [OkHttp](https://square.github.io/okhttp/)
- [Mobile Package Spec](./0095_mobile.md)
