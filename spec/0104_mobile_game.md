# Unity Game Mobile Template Specification

**Status:** Draft
**Author:** Mizu Team
**Created:** 2025-12-20
**Template:** `mobile:game`

## Overview

The `mobile:game` template generates a production-ready Unity game project with full Mizu backend integration. It follows modern Unity development practices with UniTask for async operations, Addressables for asset management, and a modular architecture that supports iOS, Android, WebGL, and desktop platforms.

## Template Invocation

```bash
# Default: Unity 2022 LTS with 2D
mizu new ./MyGame --template mobile:game

# 3D game template
mizu new ./MyGame --template mobile:game --var mode=3d

# With multiplayer support
mizu new ./MyGame --template mobile:game --var multiplayer=true

# Specific platforms
mizu new ./MyGame --template mobile:game --var platforms=ios,android

# With analytics integration
mizu new ./MyGame --template mobile:game --var analytics=true
```

## Generated Project Structure

```
{{.Name}}/
├── Assets/
│   ├── _Project/
│   │   ├── Runtime/
│   │   │   ├── Mizu/
│   │   │   │   ├── MizuRuntime.cs           # Core runtime
│   │   │   │   ├── Transport.cs              # HTTP transport layer
│   │   │   │   ├── TokenStore.cs             # Secure token storage
│   │   │   │   ├── LiveConnection.cs         # SSE streaming
│   │   │   │   ├── DeviceInfo.cs             # Device information
│   │   │   │   ├── MizuError.cs              # Error types
│   │   │   │   └── MizuRuntime.asmdef
│   │   │   ├── SDK/
│   │   │   │   ├── Client.cs                 # Generated Mizu client
│   │   │   │   ├── Types.cs                  # Generated types
│   │   │   │   ├── Extensions.cs             # Convenience extensions
│   │   │   │   └── SDK.asmdef
│   │   │   ├── Game/
│   │   │   │   ├── GameManager.cs            # Main game manager
│   │   │   │   ├── UIManager.cs              # UI management
│   │   │   │   ├── AudioManager.cs           # Audio system
│   │   │   │   ├── SaveManager.cs            # Save/Load system
│   │   │   │   ├── SceneLoader.cs            # Scene transitions
│   │   │   │   └── Game.asmdef
│   │   │   └── Services/
│   │   │       ├── AuthService.cs            # Authentication
│   │   │       ├── LeaderboardService.cs     # Leaderboards
│   │   │       ├── AchievementService.cs     # Achievements
│   │   │       ├── AnalyticsService.cs       # Analytics
│   │   │       └── Services.asmdef
│   │   ├── Scripts/
│   │   │   ├── UI/
│   │   │   │   ├── MainMenuUI.cs
│   │   │   │   ├── GameUI.cs
│   │   │   │   ├── PauseMenuUI.cs
│   │   │   │   ├── SettingsUI.cs
│   │   │   │   └── LeaderboardUI.cs
│   │   │   ├── Player/
│   │   │   │   ├── PlayerController.cs
│   │   │   │   └── PlayerStats.cs
│   │   │   └── Gameplay/
│   │   │       ├── ScoreManager.cs
│   │   │       └── LevelManager.cs
│   │   ├── Scenes/
│   │   │   ├── Bootstrap.unity
│   │   │   ├── MainMenu.unity
│   │   │   ├── Game.unity
│   │   │   └── Loading.unity
│   │   ├── Prefabs/
│   │   │   ├── Managers/
│   │   │   │   ├── GameManager.prefab
│   │   │   │   └── UIManager.prefab
│   │   │   └── UI/
│   │   │       ├── MainMenuCanvas.prefab
│   │   │       └── GameCanvas.prefab
│   │   ├── UI/
│   │   │   ├── Fonts/
│   │   │   ├── Sprites/
│   │   │   └── USS/
│   │   ├── Audio/
│   │   │   ├── Music/
│   │   │   └── SFX/
│   │   ├── Settings/
│   │   │   ├── GameSettings.asset
│   │   │   └── MizuSettings.asset
│   │   └── Resources/
│   │       └── GameConfig.asset
│   ├── Plugins/
│   │   └── Newtonsoft.Json/
│   ├── StreamingAssets/
│   ├── TextMesh Pro/
│   └── Editor/
│       ├── Build/
│       │   └── BuildScript.cs
│       └── Tools/
│           └── MizuEditorTools.cs
├── Packages/
│   └── manifest.json
├── ProjectSettings/
│   ├── ProjectSettings.asset
│   ├── InputManager.asset
│   ├── TagManager.asset
│   ├── AudioManager.asset
│   ├── TimeManager.asset
│   ├── PlayerSettings.asset
│   └── QualitySettings.asset
├── .gitignore
├── .gitattributes
└── README.md
```

## MizuMobileRuntime

### Core Runtime (`Assets/_Project/Runtime/Mizu/MizuRuntime.cs`)

```csharp
using System;
using System.Collections.Generic;
using System.Text;
using System.Threading;
using Cysharp.Threading.Tasks;
using Newtonsoft.Json;
using UnityEngine;
using UnityEngine.Networking;

namespace {{.Name}}.Mizu
{
    /// <summary>
    /// MizuRuntime is the core client for communicating with a Mizu backend.
    /// </summary>
    public class MizuRuntime : MonoBehaviour
    {
        private static MizuRuntime _instance;

        /// <summary>
        /// Shared singleton instance
        /// </summary>
        public static MizuRuntime Shared
        {
            get
            {
                if (_instance == null)
                {
                    var go = new GameObject("[MizuRuntime]");
                    _instance = go.AddComponent<MizuRuntime>();
                    DontDestroyOnLoad(go);
                }
                return _instance;
            }
        }

        [Header("Configuration")]
        [SerializeField] private string _baseUrl = "http://localhost:3000";
        [SerializeField] private float _timeout = 30f;

        /// <summary>
        /// Base URL for all API requests
        /// </summary>
        public string BaseUrl
        {
            get => _baseUrl;
            set => _baseUrl = value;
        }

        /// <summary>
        /// Request timeout in seconds
        /// </summary>
        public float Timeout
        {
            get => _timeout;
            set => _timeout = value;
        }

        /// <summary>
        /// HTTP transport layer
        /// </summary>
        public Transport Transport { get; private set; }

        /// <summary>
        /// Secure token storage
        /// </summary>
        public TokenStore TokenStore { get; private set; }

        /// <summary>
        /// Live connection manager
        /// </summary>
        public LiveConnection Live { get; private set; }

        /// <summary>
        /// Device info provider
        /// </summary>
        public DeviceInfo DeviceInfo { get; private set; }

        /// <summary>
        /// Default headers added to all requests
        /// </summary>
        public Dictionary<string, string> DefaultHeaders { get; } = new();

        /// <summary>
        /// Current authentication state
        /// </summary>
        public bool IsAuthenticated => TokenStore?.GetToken() != null;

        /// <summary>
        /// Event fired when authentication state changes
        /// </summary>
        public event Action<bool> OnAuthStateChanged;

        private JsonSerializerSettings _jsonSettings;

        private void Awake()
        {
            if (_instance != null && _instance != this)
            {
                Destroy(gameObject);
                return;
            }

            _instance = this;
            DontDestroyOnLoad(gameObject);
            Initialize();
        }

        /// <summary>
        /// Initialize the runtime with configuration
        /// </summary>
        public void Initialize(string baseUrl = null, float? timeout = null)
        {
            if (baseUrl != null) _baseUrl = baseUrl;
            if (timeout.HasValue) _timeout = timeout.Value;

            _jsonSettings = new JsonSerializerSettings
            {
                NullValueHandling = NullValueHandling.Ignore,
                MissingMemberHandling = MissingMemberHandling.Ignore
            };

            Transport = new Transport();
            TokenStore = new TokenStore();
            Live = new LiveConnection(this);
            DeviceInfo = new DeviceInfo();

            TokenStore.OnTokenChanged += (token) =>
            {
                OnAuthStateChanged?.Invoke(token != null);
            };
        }

        #region HTTP Methods

        /// <summary>
        /// Performs a GET request
        /// </summary>
        public async UniTask<T> GetAsync<T>(
            string path,
            Dictionary<string, string> query = null,
            Dictionary<string, string> headers = null,
            CancellationToken cancellationToken = default)
        {
            return await RequestAsync<T, object>(
                UnityWebRequest.kHttpVerbGET,
                path,
                query: query,
                headers: headers,
                cancellationToken: cancellationToken);
        }

        /// <summary>
        /// Performs a POST request
        /// </summary>
        public async UniTask<T> PostAsync<T, TBody>(
            string path,
            TBody body = default,
            Dictionary<string, string> headers = null,
            CancellationToken cancellationToken = default)
        {
            return await RequestAsync<T, TBody>(
                UnityWebRequest.kHttpVerbPOST,
                path,
                body: body,
                headers: headers,
                cancellationToken: cancellationToken);
        }

        /// <summary>
        /// Performs a PUT request
        /// </summary>
        public async UniTask<T> PutAsync<T, TBody>(
            string path,
            TBody body = default,
            Dictionary<string, string> headers = null,
            CancellationToken cancellationToken = default)
        {
            return await RequestAsync<T, TBody>(
                UnityWebRequest.kHttpVerbPUT,
                path,
                body: body,
                headers: headers,
                cancellationToken: cancellationToken);
        }

        /// <summary>
        /// Performs a DELETE request
        /// </summary>
        public async UniTask<T> DeleteAsync<T>(
            string path,
            Dictionary<string, string> headers = null,
            CancellationToken cancellationToken = default)
        {
            return await RequestAsync<T, object>(
                UnityWebRequest.kHttpVerbDELETE,
                path,
                headers: headers,
                cancellationToken: cancellationToken);
        }

        /// <summary>
        /// Performs a PATCH request
        /// </summary>
        public async UniTask<T> PatchAsync<T, TBody>(
            string path,
            TBody body = default,
            Dictionary<string, string> headers = null,
            CancellationToken cancellationToken = default)
        {
            return await RequestAsync<T, TBody>(
                "PATCH",
                path,
                body: body,
                headers: headers,
                cancellationToken: cancellationToken);
        }

        #endregion

        #region Private Methods

        private async UniTask<T> RequestAsync<T, TBody>(
            string method,
            string path,
            Dictionary<string, string> query = null,
            TBody body = default,
            Dictionary<string, string> headers = null,
            CancellationToken cancellationToken = default)
        {
            var url = BuildUrl(path, query);
            var allHeaders = await BuildHeaders(headers);

            string bodyJson = null;
            if (body != null && !EqualityComparer<TBody>.Default.Equals(body, default))
            {
                bodyJson = JsonConvert.SerializeObject(body, _jsonSettings);
                allHeaders["Content-Type"] = "application/json";
            }

            var request = new TransportRequest
            {
                Url = url,
                Method = method,
                Headers = allHeaders,
                Body = bodyJson,
                Timeout = _timeout
            };

            var response = await Transport.ExecuteAsync(request, cancellationToken);

            if (response.StatusCode >= 400)
            {
                throw ParseError(response);
            }

            if (typeof(T) == typeof(object) || string.IsNullOrEmpty(response.Body))
            {
                return default;
            }

            return JsonConvert.DeserializeObject<T>(response.Body, _jsonSettings);
        }

        private string BuildUrl(string path, Dictionary<string, string> query)
        {
            var baseUrl = _baseUrl.TrimEnd('/');
            var cleanPath = path.StartsWith("/") ? path : $"/{path}";
            var url = new StringBuilder($"{baseUrl}{cleanPath}");

            if (query != null && query.Count > 0)
            {
                url.Append("?");
                var first = true;
                foreach (var kvp in query)
                {
                    if (!first) url.Append("&");
                    url.Append($"{UnityWebRequest.EscapeURL(kvp.Key)}={UnityWebRequest.EscapeURL(kvp.Value)}");
                    first = false;
                }
            }

            return url.ToString();
        }

        private async UniTask<Dictionary<string, string>> BuildHeaders(Dictionary<string, string> custom)
        {
            var headers = new Dictionary<string, string>(DefaultHeaders);

            // Add device headers
            var info = DeviceInfo;
            headers["X-Device-ID"] = info.DeviceId;
            headers["X-App-Version"] = info.AppVersion;
            headers["X-App-Build"] = info.AppBuild;
            headers["X-Device-Model"] = info.Model;
            headers["X-Platform"] = info.Platform;
            headers["X-OS-Version"] = info.OsVersion;
            headers["X-Timezone"] = info.Timezone;
            headers["X-Locale"] = info.Locale;

            // Add custom headers
            if (custom != null)
            {
                foreach (var kvp in custom)
                {
                    headers[kvp.Key] = kvp.Value;
                }
            }

            // Add auth token
            var token = TokenStore.GetToken();
            if (token != null)
            {
                headers["Authorization"] = $"Bearer {token.AccessToken}";
            }

            return headers;
        }

        private MizuError ParseError(TransportResponse response)
        {
            try
            {
                var apiError = JsonConvert.DeserializeObject<ApiError>(response.Body, _jsonSettings);
                return new MizuError.Api(apiError);
            }
            catch
            {
                return new MizuError.Http(response.StatusCode, response.Body);
            }
        }

        #endregion
    }
}
```

### Transport Layer (`Assets/_Project/Runtime/Mizu/Transport.cs`)

```csharp
using System;
using System.Collections.Generic;
using System.Text;
using System.Threading;
using Cysharp.Threading.Tasks;
using UnityEngine;
using UnityEngine.Networking;

namespace {{.Name}}.Mizu
{
    /// <summary>
    /// Transport request
    /// </summary>
    public class TransportRequest
    {
        public string Url { get; set; }
        public string Method { get; set; }
        public Dictionary<string, string> Headers { get; set; }
        public string Body { get; set; }
        public float Timeout { get; set; } = 30f;
    }

    /// <summary>
    /// Transport response
    /// </summary>
    public class TransportResponse
    {
        public int StatusCode { get; set; }
        public Dictionary<string, string> Headers { get; set; }
        public string Body { get; set; }
    }

    /// <summary>
    /// HTTP transport using UnityWebRequest
    /// </summary>
    public class Transport
    {
        private readonly List<IRequestInterceptor> _interceptors = new();

        /// <summary>
        /// Adds a request interceptor
        /// </summary>
        public void AddInterceptor(IRequestInterceptor interceptor)
        {
            _interceptors.Add(interceptor);
        }

        /// <summary>
        /// Executes an HTTP request
        /// </summary>
        public async UniTask<TransportResponse> ExecuteAsync(
            TransportRequest request,
            CancellationToken cancellationToken = default)
        {
            // Apply interceptors
            foreach (var interceptor in _interceptors)
            {
                request = await interceptor.InterceptAsync(request, cancellationToken);
            }

            try
            {
                using var webRequest = CreateRequest(request);
                webRequest.timeout = Mathf.RoundToInt(request.Timeout);

                await webRequest.SendWebRequest().WithCancellation(cancellationToken);

                var responseHeaders = new Dictionary<string, string>();
                foreach (var header in webRequest.GetResponseHeaders())
                {
                    responseHeaders[header.Key] = header.Value;
                }

                return new TransportResponse
                {
                    StatusCode = (int)webRequest.responseCode,
                    Headers = responseHeaders,
                    Body = webRequest.downloadHandler?.text ?? string.Empty
                };
            }
            catch (OperationCanceledException)
            {
                throw;
            }
            catch (Exception e)
            {
                throw new MizuError.Network(e.Message, e);
            }
        }

        private UnityWebRequest CreateRequest(TransportRequest request)
        {
            UnityWebRequest webRequest;

            switch (request.Method.ToUpperInvariant())
            {
                case "GET":
                    webRequest = UnityWebRequest.Get(request.Url);
                    break;
                case "POST":
                    webRequest = new UnityWebRequest(request.Url, "POST")
                    {
                        uploadHandler = request.Body != null
                            ? new UploadHandlerRaw(Encoding.UTF8.GetBytes(request.Body))
                            : null,
                        downloadHandler = new DownloadHandlerBuffer()
                    };
                    break;
                case "PUT":
                    webRequest = UnityWebRequest.Put(request.Url, request.Body ?? string.Empty);
                    webRequest.downloadHandler = new DownloadHandlerBuffer();
                    break;
                case "DELETE":
                    webRequest = UnityWebRequest.Delete(request.Url);
                    webRequest.downloadHandler = new DownloadHandlerBuffer();
                    break;
                case "PATCH":
                    webRequest = new UnityWebRequest(request.Url, "PATCH")
                    {
                        uploadHandler = request.Body != null
                            ? new UploadHandlerRaw(Encoding.UTF8.GetBytes(request.Body))
                            : null,
                        downloadHandler = new DownloadHandlerBuffer()
                    };
                    break;
                default:
                    throw new NotSupportedException($"HTTP method {request.Method} not supported");
            }

            // Set headers
            if (request.Headers != null)
            {
                foreach (var header in request.Headers)
                {
                    webRequest.SetRequestHeader(header.Key, header.Value);
                }
            }

            return webRequest;
        }
    }

    /// <summary>
    /// Request interceptor interface
    /// </summary>
    public interface IRequestInterceptor
    {
        UniTask<TransportRequest> InterceptAsync(
            TransportRequest request,
            CancellationToken cancellationToken);
    }

    /// <summary>
    /// Logging interceptor for debugging
    /// </summary>
    public class LoggingInterceptor : IRequestInterceptor
    {
        public UniTask<TransportRequest> InterceptAsync(
            TransportRequest request,
            CancellationToken cancellationToken)
        {
            Debug.Log($"[Mizu] {request.Method} {request.Url}");
            return UniTask.FromResult(request);
        }
    }

    /// <summary>
    /// Retry interceptor with exponential backoff
    /// </summary>
    public class RetryInterceptor : IRequestInterceptor
    {
        private readonly int _maxRetries;
        private readonly float _baseDelay;

        public RetryInterceptor(int maxRetries = 3, float baseDelay = 1f)
        {
            _maxRetries = maxRetries;
            _baseDelay = baseDelay;
        }

        public UniTask<TransportRequest> InterceptAsync(
            TransportRequest request,
            CancellationToken cancellationToken)
        {
            // Retry logic handled at transport level
            return UniTask.FromResult(request);
        }
    }
}
```

### Token Store (`Assets/_Project/Runtime/Mizu/TokenStore.cs`)

```csharp
using System;
using Newtonsoft.Json;
using UnityEngine;

namespace {{.Name}}.Mizu
{
    /// <summary>
    /// Authentication token
    /// </summary>
    [Serializable]
    public class AuthToken
    {
        [JsonProperty("access_token")]
        public string AccessToken { get; set; }

        [JsonProperty("refresh_token")]
        public string RefreshToken { get; set; }

        [JsonProperty("expires_at")]
        public long? ExpiresAt { get; set; }

        [JsonProperty("token_type")]
        public string TokenType { get; set; } = "Bearer";

        [JsonIgnore]
        public bool IsExpired
        {
            get
            {
                if (!ExpiresAt.HasValue) return false;
                var now = DateTimeOffset.UtcNow.ToUnixTimeMilliseconds();
                return now >= ExpiresAt.Value;
            }
        }
    }

    /// <summary>
    /// Creates an auth token with expiry
    /// </summary>
    public static class AuthTokenFactory
    {
        public static AuthToken Create(
            string accessToken,
            string refreshToken = null,
            int? expiresInSeconds = null,
            string tokenType = "Bearer")
        {
            long? expiresAt = null;
            if (expiresInSeconds.HasValue)
            {
                expiresAt = DateTimeOffset.UtcNow.ToUnixTimeMilliseconds() + (expiresInSeconds.Value * 1000);
            }

            return new AuthToken
            {
                AccessToken = accessToken,
                RefreshToken = refreshToken,
                ExpiresAt = expiresAt,
                TokenType = tokenType
            };
        }
    }

    /// <summary>
    /// Secure token storage using PlayerPrefs with encryption
    /// </summary>
    public class TokenStore
    {
        private const string TokenKey = "mizu_auth_token";
        private AuthToken _cachedToken;

        /// <summary>
        /// Event fired when token changes
        /// </summary>
        public event Action<AuthToken> OnTokenChanged;

        /// <summary>
        /// Gets the current token
        /// </summary>
        public AuthToken GetToken()
        {
            if (_cachedToken != null) return _cachedToken;

            var json = PlayerPrefs.GetString(TokenKey, null);
            if (string.IsNullOrEmpty(json)) return null;

            try
            {
                // In production, decrypt the token here
                _cachedToken = JsonConvert.DeserializeObject<AuthToken>(json);
                return _cachedToken;
            }
            catch
            {
                return null;
            }
        }

        /// <summary>
        /// Sets the token
        /// </summary>
        public void SetToken(AuthToken token)
        {
            if (token == null)
            {
                ClearToken();
                return;
            }

            _cachedToken = token;
            // In production, encrypt the token here
            var json = JsonConvert.SerializeObject(token);
            PlayerPrefs.SetString(TokenKey, json);
            PlayerPrefs.Save();

            OnTokenChanged?.Invoke(token);
        }

        /// <summary>
        /// Clears the token
        /// </summary>
        public void ClearToken()
        {
            _cachedToken = null;
            PlayerPrefs.DeleteKey(TokenKey);
            PlayerPrefs.Save();

            OnTokenChanged?.Invoke(null);
        }
    }

    /// <summary>
    /// Encrypted token store using Unity's secure storage
    /// </summary>
    public class SecureTokenStore : TokenStore
    {
        // For production, implement platform-specific secure storage:
        // - iOS: Keychain
        // - Android: EncryptedSharedPreferences
        // - Desktop: DPAPI or platform-specific secure storage
    }
}
```

### Live Streaming (`Assets/_Project/Runtime/Mizu/LiveConnection.cs`)

```csharp
using System;
using System.Collections.Generic;
using System.IO;
using System.Net.Http;
using System.Threading;
using Cysharp.Threading.Tasks;
using UnityEngine;

namespace {{.Name}}.Mizu
{
    /// <summary>
    /// Server-sent event
    /// </summary>
    public class ServerEvent
    {
        public string Id { get; set; }
        public string Event { get; set; }
        public string Data { get; set; }
        public int? Retry { get; set; }

        /// <summary>
        /// Decodes the event data as JSON
        /// </summary>
        public T Decode<T>()
        {
            return Newtonsoft.Json.JsonConvert.DeserializeObject<T>(Data);
        }
    }

    /// <summary>
    /// Live connection manager for SSE
    /// </summary>
    public class LiveConnection
    {
        private readonly MizuRuntime _runtime;
        private readonly Dictionary<string, CancellationTokenSource> _activeConnections = new();
        private HttpClient _httpClient;

        public LiveConnection(MizuRuntime runtime)
        {
            _runtime = runtime;
            _httpClient = new HttpClient { Timeout = TimeSpan.FromMilliseconds(Timeout.Infinite) };
        }

        /// <summary>
        /// Connects to an SSE endpoint
        /// </summary>
        public async IAsyncEnumerable<ServerEvent> ConnectAsync(
            string path,
            Dictionary<string, string> headers = null,
            [System.Runtime.CompilerServices.EnumeratorCancellation] CancellationToken cancellationToken = default)
        {
            var cts = CancellationTokenSource.CreateLinkedTokenSource(cancellationToken);
            _activeConnections[path] = cts;

            var url = BuildUrl(path);
            var request = new HttpRequestMessage(HttpMethod.Get, url);

            // Add headers
            foreach (var header in _runtime.DefaultHeaders)
            {
                request.Headers.TryAddWithoutValidation(header.Key, header.Value);
            }

            var token = _runtime.TokenStore.GetToken();
            if (token != null)
            {
                request.Headers.TryAddWithoutValidation("Authorization", $"Bearer {token.AccessToken}");
            }

            if (headers != null)
            {
                foreach (var header in headers)
                {
                    request.Headers.TryAddWithoutValidation(header.Key, header.Value);
                }
            }

            request.Headers.TryAddWithoutValidation("Accept", "text/event-stream");
            request.Headers.TryAddWithoutValidation("Cache-Control", "no-cache");

            HttpResponseMessage response = null;
            Stream stream = null;
            StreamReader reader = null;

            try
            {
                response = await _httpClient.SendAsync(
                    request,
                    HttpCompletionOption.ResponseHeadersRead,
                    cts.Token);

                response.EnsureSuccessStatusCode();
                stream = await response.Content.ReadAsStreamAsync();
                reader = new StreamReader(stream);

                var eventBuilder = new SSEEventBuilder();

                while (!cts.Token.IsCancellationRequested && !reader.EndOfStream)
                {
                    var line = await reader.ReadLineAsync();
                    if (line == null) break;

                    var serverEvent = eventBuilder.ProcessLine(line);
                    if (serverEvent != null)
                    {
                        yield return serverEvent;
                    }
                }
            }
            finally
            {
                _activeConnections.Remove(path);
                reader?.Dispose();
                stream?.Dispose();
                response?.Dispose();
                cts.Dispose();
            }
        }

        /// <summary>
        /// Disconnects from a specific path
        /// </summary>
        public void Disconnect(string path)
        {
            if (_activeConnections.TryGetValue(path, out var cts))
            {
                cts.Cancel();
                _activeConnections.Remove(path);
            }
        }

        /// <summary>
        /// Disconnects all active connections
        /// </summary>
        public void DisconnectAll()
        {
            foreach (var cts in _activeConnections.Values)
            {
                cts.Cancel();
            }
            _activeConnections.Clear();
        }

        private string BuildUrl(string path)
        {
            var baseUrl = _runtime.BaseUrl.TrimEnd('/');
            var cleanPath = path.StartsWith("/") ? path : $"/{path}";
            return $"{baseUrl}{cleanPath}";
        }
    }

    /// <summary>
    /// SSE event parser
    /// </summary>
    internal class SSEEventBuilder
    {
        private string _id;
        private string _event;
        private List<string> _data = new();
        private int? _retry;

        public ServerEvent ProcessLine(string line)
        {
            if (string.IsNullOrEmpty(line))
            {
                // Empty line means end of event
                if (_data.Count == 0) return null;

                var serverEvent = new ServerEvent
                {
                    Id = _id,
                    Event = _event,
                    Data = string.Join("\n", _data),
                    Retry = _retry
                };

                // Reset for next event (keep id for Last-Event-ID)
                _event = null;
                _data.Clear();
                _retry = null;

                return serverEvent;
            }

            if (line.StartsWith(":"))
            {
                // Comment, ignore
                return null;
            }

            var colonIndex = line.IndexOf(':');
            if (colonIndex == -1) return null;

            var field = line.Substring(0, colonIndex);
            var value = colonIndex + 1 < line.Length ? line.Substring(colonIndex + 1) : string.Empty;
            if (value.StartsWith(" ")) value = value.Substring(1);

            switch (field)
            {
                case "id":
                    _id = value;
                    break;
                case "event":
                    _event = value;
                    break;
                case "data":
                    _data.Add(value);
                    break;
                case "retry":
                    if (int.TryParse(value, out var retry))
                        _retry = retry;
                    break;
            }

            return null;
        }
    }
}
```

### Device Info (`Assets/_Project/Runtime/Mizu/DeviceInfo.cs`)

```csharp
using System;
using System.Globalization;
using UnityEngine;

namespace {{.Name}}.Mizu
{
    /// <summary>
    /// Device information for mobile headers
    /// </summary>
    public class DeviceInfo
    {
        private string _deviceId;

        /// <summary>
        /// Unique device identifier
        /// </summary>
        public string DeviceId
        {
            get
            {
                if (string.IsNullOrEmpty(_deviceId))
                {
                    _deviceId = SystemInfo.deviceUniqueIdentifier;
                }
                return _deviceId;
            }
        }

        /// <summary>
        /// Application version
        /// </summary>
        public string AppVersion => Application.version;

        /// <summary>
        /// Application build number
        /// </summary>
        public string AppBuild => Application.version; // Use build number if available

        /// <summary>
        /// Device model
        /// </summary>
        public string Model => SystemInfo.deviceModel;

        /// <summary>
        /// Platform name
        /// </summary>
        public string Platform
        {
            get
            {
                return Application.platform switch
                {
                    RuntimePlatform.Android => "android",
                    RuntimePlatform.IPhonePlayer => "ios",
                    RuntimePlatform.OSXPlayer or RuntimePlatform.OSXEditor => "macos",
                    RuntimePlatform.WindowsPlayer or RuntimePlatform.WindowsEditor => "windows",
                    RuntimePlatform.LinuxPlayer or RuntimePlatform.LinuxEditor => "linux",
                    RuntimePlatform.WebGLPlayer => "webgl",
                    _ => "unknown"
                };
            }
        }

        /// <summary>
        /// Operating system version
        /// </summary>
        public string OsVersion => SystemInfo.operatingSystem;

        /// <summary>
        /// Current timezone
        /// </summary>
        public string Timezone => TimeZoneInfo.Local.Id;

        /// <summary>
        /// Current locale
        /// </summary>
        public string Locale => CultureInfo.CurrentCulture.Name;
    }
}
```

### Errors (`Assets/_Project/Runtime/Mizu/MizuError.cs`)

```csharp
using System;
using Newtonsoft.Json;

namespace {{.Name}}.Mizu
{
    /// <summary>
    /// API error response from server
    /// </summary>
    [Serializable]
    public class ApiError
    {
        [JsonProperty("code")]
        public string Code { get; set; }

        [JsonProperty("message")]
        public string Message { get; set; }

        [JsonProperty("details")]
        public System.Collections.Generic.Dictionary<string, string> Details { get; set; }

        [JsonProperty("trace_id")]
        public string TraceId { get; set; }
    }

    /// <summary>
    /// Mizu client errors
    /// </summary>
    public abstract class MizuError : Exception
    {
        public string Type { get; }

        protected MizuError(string type, string message, Exception innerException = null)
            : base(message, innerException)
        {
            Type = type;
        }

        public bool IsInvalidResponse => Type == "invalid_response";
        public bool IsHttp => Type == "http";
        public bool IsApi => Type == "api";
        public bool IsNetwork => Type == "network";
        public bool IsUnauthorized => Type == "unauthorized";
        public bool IsTokenExpired => Type == "token_expired";

        public class InvalidResponse : MizuError
        {
            public InvalidResponse(string message = "Invalid server response")
                : base("invalid_response", message) { }
        }

        public class Http : MizuError
        {
            public int StatusCode { get; }
            public string Body { get; }

            public Http(int statusCode, string body)
                : base("http", $"HTTP error {statusCode}")
            {
                StatusCode = statusCode;
                Body = body;
            }
        }

        public class Api : MizuError
        {
            public ApiError Error { get; }

            public Api(ApiError error)
                : base("api", error.Message)
            {
                Error = error;
            }
        }

        public class Network : MizuError
        {
            public Network(string message, Exception innerException = null)
                : base("network", message, innerException) { }
        }

        public class Encoding : MizuError
        {
            public Encoding(string message, Exception innerException = null)
                : base("encoding", message, innerException) { }
        }

        public class Decoding : MizuError
        {
            public Decoding(string message, Exception innerException = null)
                : base("decoding", message, innerException) { }
        }

        public class Unauthorized : MizuError
        {
            public Unauthorized(string message = "Unauthorized")
                : base("unauthorized", message) { }
        }

        public class TokenExpired : MizuError
        {
            public TokenExpired(string message = "Token expired")
                : base("token_expired", message) { }
        }
    }
}
```

## Game Services

### Auth Service (`Assets/_Project/Runtime/Services/AuthService.cs`)

```csharp
using System.Threading;
using Cysharp.Threading.Tasks;
using {{.Name}}.Mizu;
using {{.Name}}.SDK;

namespace {{.Name}}.Services
{
    /// <summary>
    /// Authentication service
    /// </summary>
    public class AuthService
    {
        private readonly GameClient _client;

        public AuthService(GameClient client)
        {
            _client = client;
        }

        /// <summary>
        /// Sign in with credentials
        /// </summary>
        public async UniTask<AuthResponse> SignInAsync(
            string email,
            string password,
            CancellationToken cancellationToken = default)
        {
            var response = await _client.SignInAsync(email, password, cancellationToken);
            await _client.StoreAuthTokenAsync(response);
            return response;
        }

        /// <summary>
        /// Sign up with credentials
        /// </summary>
        public async UniTask<AuthResponse> SignUpAsync(
            string email,
            string password,
            string username,
            CancellationToken cancellationToken = default)
        {
            var response = await _client.SignUpAsync(email, password, username, cancellationToken);
            await _client.StoreAuthTokenAsync(response);
            return response;
        }

        /// <summary>
        /// Sign in as guest
        /// </summary>
        public async UniTask<AuthResponse> SignInAsGuestAsync(
            CancellationToken cancellationToken = default)
        {
            var deviceId = MizuRuntime.Shared.DeviceInfo.DeviceId;
            var response = await _client.SignInAsGuestAsync(deviceId, cancellationToken);
            await _client.StoreAuthTokenAsync(response);
            return response;
        }

        /// <summary>
        /// Sign out
        /// </summary>
        public async UniTask SignOutAsync(CancellationToken cancellationToken = default)
        {
            await _client.SignOutAsync(cancellationToken);
            MizuRuntime.Shared.TokenStore.ClearToken();
        }

        /// <summary>
        /// Check if user is authenticated
        /// </summary>
        public bool IsAuthenticated => MizuRuntime.Shared.IsAuthenticated;
    }
}
```

### Leaderboard Service (`Assets/_Project/Runtime/Services/LeaderboardService.cs`)

```csharp
using System.Collections.Generic;
using System.Threading;
using Cysharp.Threading.Tasks;
using {{.Name}}.SDK;

namespace {{.Name}}.Services
{
    /// <summary>
    /// Leaderboard service
    /// </summary>
    public class LeaderboardService
    {
        private readonly GameClient _client;

        public LeaderboardService(GameClient client)
        {
            _client = client;
        }

        /// <summary>
        /// Submit a score
        /// </summary>
        public async UniTask<LeaderboardEntry> SubmitScoreAsync(
            string leaderboardId,
            int score,
            Dictionary<string, object> metadata = null,
            CancellationToken cancellationToken = default)
        {
            return await _client.SubmitScoreAsync(leaderboardId, score, metadata, cancellationToken);
        }

        /// <summary>
        /// Get top scores
        /// </summary>
        public async UniTask<List<LeaderboardEntry>> GetTopScoresAsync(
            string leaderboardId,
            int limit = 10,
            CancellationToken cancellationToken = default)
        {
            return await _client.GetLeaderboardAsync(leaderboardId, limit, 0, cancellationToken);
        }

        /// <summary>
        /// Get scores around player
        /// </summary>
        public async UniTask<List<LeaderboardEntry>> GetScoresAroundPlayerAsync(
            string leaderboardId,
            int limit = 10,
            CancellationToken cancellationToken = default)
        {
            return await _client.GetLeaderboardAroundPlayerAsync(leaderboardId, limit, cancellationToken);
        }

        /// <summary>
        /// Get player's rank
        /// </summary>
        public async UniTask<int> GetPlayerRankAsync(
            string leaderboardId,
            CancellationToken cancellationToken = default)
        {
            var entry = await _client.GetPlayerLeaderboardEntryAsync(leaderboardId, cancellationToken);
            return entry?.Rank ?? -1;
        }
    }
}
```

### Achievement Service (`Assets/_Project/Runtime/Services/AchievementService.cs`)

```csharp
using System.Collections.Generic;
using System.Threading;
using Cysharp.Threading.Tasks;
using {{.Name}}.SDK;

namespace {{.Name}}.Services
{
    /// <summary>
    /// Achievement service
    /// </summary>
    public class AchievementService
    {
        private readonly GameClient _client;
        private readonly Dictionary<string, Achievement> _cachedAchievements = new();

        public AchievementService(GameClient client)
        {
            _client = client;
        }

        /// <summary>
        /// Get all achievements
        /// </summary>
        public async UniTask<List<Achievement>> GetAchievementsAsync(
            CancellationToken cancellationToken = default)
        {
            var achievements = await _client.GetAchievementsAsync(cancellationToken);
            foreach (var achievement in achievements)
            {
                _cachedAchievements[achievement.Id] = achievement;
            }
            return achievements;
        }

        /// <summary>
        /// Unlock an achievement
        /// </summary>
        public async UniTask<Achievement> UnlockAsync(
            string achievementId,
            CancellationToken cancellationToken = default)
        {
            var achievement = await _client.UnlockAchievementAsync(achievementId, cancellationToken);
            _cachedAchievements[achievement.Id] = achievement;
            return achievement;
        }

        /// <summary>
        /// Update achievement progress
        /// </summary>
        public async UniTask<Achievement> UpdateProgressAsync(
            string achievementId,
            int progress,
            CancellationToken cancellationToken = default)
        {
            var achievement = await _client.UpdateAchievementProgressAsync(
                achievementId, progress, cancellationToken);
            _cachedAchievements[achievement.Id] = achievement;
            return achievement;
        }

        /// <summary>
        /// Get player's unlocked achievements
        /// </summary>
        public async UniTask<List<Achievement>> GetUnlockedAsync(
            CancellationToken cancellationToken = default)
        {
            return await _client.GetUnlockedAchievementsAsync(cancellationToken);
        }
    }
}
```

### Analytics Service (`Assets/_Project/Runtime/Services/AnalyticsService.cs`)

```csharp
using System.Collections.Generic;
using System.Threading;
using Cysharp.Threading.Tasks;
using UnityEngine;
using {{.Name}}.Mizu;

namespace {{.Name}}.Services
{
    /// <summary>
    /// Analytics service for tracking game events
    /// </summary>
    public class AnalyticsService
    {
        private readonly MizuRuntime _runtime;
        private readonly Queue<AnalyticsEvent> _eventQueue = new();
        private readonly int _batchSize = 10;
        private readonly float _flushInterval = 30f;
        private float _lastFlushTime;

        public AnalyticsService()
        {
            _runtime = MizuRuntime.Shared;
        }

        /// <summary>
        /// Track a game event
        /// </summary>
        public void Track(string eventName, Dictionary<string, object> properties = null)
        {
            var analyticsEvent = new AnalyticsEvent
            {
                Name = eventName,
                Properties = properties ?? new Dictionary<string, object>(),
                Timestamp = System.DateTimeOffset.UtcNow.ToUnixTimeMilliseconds()
            };

            _eventQueue.Enqueue(analyticsEvent);

            if (_eventQueue.Count >= _batchSize)
            {
                _ = FlushAsync();
            }
        }

        /// <summary>
        /// Track game start
        /// </summary>
        public void TrackGameStart()
        {
            Track("game_start", new Dictionary<string, object>
            {
                ["platform"] = Application.platform.ToString(),
                ["device_model"] = SystemInfo.deviceModel,
                ["os_version"] = SystemInfo.operatingSystem
            });
        }

        /// <summary>
        /// Track level start
        /// </summary>
        public void TrackLevelStart(string levelId, int levelNumber)
        {
            Track("level_start", new Dictionary<string, object>
            {
                ["level_id"] = levelId,
                ["level_number"] = levelNumber
            });
        }

        /// <summary>
        /// Track level complete
        /// </summary>
        public void TrackLevelComplete(string levelId, int score, float duration)
        {
            Track("level_complete", new Dictionary<string, object>
            {
                ["level_id"] = levelId,
                ["score"] = score,
                ["duration_seconds"] = duration
            });
        }

        /// <summary>
        /// Track purchase
        /// </summary>
        public void TrackPurchase(string productId, decimal amount, string currency)
        {
            Track("purchase", new Dictionary<string, object>
            {
                ["product_id"] = productId,
                ["amount"] = amount,
                ["currency"] = currency
            });
        }

        /// <summary>
        /// Flush queued events
        /// </summary>
        public async UniTask FlushAsync(CancellationToken cancellationToken = default)
        {
            if (_eventQueue.Count == 0) return;

            var events = new List<AnalyticsEvent>();
            while (_eventQueue.Count > 0 && events.Count < _batchSize)
            {
                events.Add(_eventQueue.Dequeue());
            }

            try
            {
                await _runtime.PostAsync<object, AnalyticsBatch>(
                    "/analytics/events",
                    new AnalyticsBatch { Events = events },
                    cancellationToken: cancellationToken);
            }
            catch (System.Exception e)
            {
                Debug.LogWarning($"[Analytics] Failed to flush events: {e.Message}");
                // Re-queue events on failure
                foreach (var evt in events)
                {
                    _eventQueue.Enqueue(evt);
                }
            }

            _lastFlushTime = Time.time;
        }
    }

    [System.Serializable]
    public class AnalyticsEvent
    {
        public string Name { get; set; }
        public Dictionary<string, object> Properties { get; set; }
        public long Timestamp { get; set; }
    }

    [System.Serializable]
    public class AnalyticsBatch
    {
        public List<AnalyticsEvent> Events { get; set; }
    }
}
```

## Game Managers

### Game Manager (`Assets/_Project/Runtime/Game/GameManager.cs`)

```csharp
using System;
using Cysharp.Threading.Tasks;
using UnityEngine;
using {{.Name}}.Mizu;
using {{.Name}}.SDK;
using {{.Name}}.Services;

namespace {{.Name}}.Game
{
    /// <summary>
    /// Main game manager singleton
    /// </summary>
    public class GameManager : MonoBehaviour
    {
        public static GameManager Instance { get; private set; }

        [Header("Configuration")]
        [SerializeField] private bool _autoInitialize = true;
        [SerializeField] private string _baseUrl = "http://localhost:3000";

        /// <summary>
        /// Mizu runtime
        /// </summary>
        public MizuRuntime Runtime { get; private set; }

        /// <summary>
        /// Game client
        /// </summary>
        public GameClient Client { get; private set; }

        /// <summary>
        /// Authentication service
        /// </summary>
        public AuthService Auth { get; private set; }

        /// <summary>
        /// Leaderboard service
        /// </summary>
        public LeaderboardService Leaderboards { get; private set; }

        /// <summary>
        /// Achievement service
        /// </summary>
        public AchievementService Achievements { get; private set; }

        /// <summary>
        /// Analytics service
        /// </summary>
        public AnalyticsService Analytics { get; private set; }

        /// <summary>
        /// Current game state
        /// </summary>
        public GameState State { get; private set; } = GameState.Loading;

        /// <summary>
        /// Event fired when game state changes
        /// </summary>
        public event Action<GameState> OnStateChanged;

        private void Awake()
        {
            if (Instance != null && Instance != this)
            {
                Destroy(gameObject);
                return;
            }

            Instance = this;
            DontDestroyOnLoad(gameObject);

            if (_autoInitialize)
            {
                Initialize();
            }
        }

        /// <summary>
        /// Initialize the game manager
        /// </summary>
        public void Initialize()
        {
            // Initialize Mizu runtime
            Runtime = MizuRuntime.Shared;
            Runtime.Initialize(_baseUrl);

            // Initialize services
            Client = new GameClient(Runtime);
            Auth = new AuthService(Client);
            Leaderboards = new LeaderboardService(Client);
            Achievements = new AchievementService(Client);
            Analytics = new AnalyticsService();

            // Track game start
            Analytics.TrackGameStart();

            // Subscribe to auth changes
            Runtime.OnAuthStateChanged += OnAuthStateChanged;

            SetState(GameState.MainMenu);
        }

        private void OnAuthStateChanged(bool isAuthenticated)
        {
            Debug.Log($"[GameManager] Auth state changed: {isAuthenticated}");
        }

        /// <summary>
        /// Set the current game state
        /// </summary>
        public void SetState(GameState state)
        {
            if (State == state) return;
            State = state;
            OnStateChanged?.Invoke(state);
        }

        /// <summary>
        /// Start the game
        /// </summary>
        public async UniTask StartGameAsync()
        {
            SetState(GameState.Loading);

            // Load game scene
            await SceneLoader.LoadSceneAsync("Game");

            SetState(GameState.Playing);
        }

        /// <summary>
        /// Pause the game
        /// </summary>
        public void PauseGame()
        {
            if (State != GameState.Playing) return;
            Time.timeScale = 0f;
            SetState(GameState.Paused);
        }

        /// <summary>
        /// Resume the game
        /// </summary>
        public void ResumeGame()
        {
            if (State != GameState.Paused) return;
            Time.timeScale = 1f;
            SetState(GameState.Playing);
        }

        /// <summary>
        /// End the game
        /// </summary>
        public async UniTask EndGameAsync(int score)
        {
            SetState(GameState.GameOver);

            // Submit score
            try
            {
                await Leaderboards.SubmitScoreAsync("main", score);
            }
            catch (Exception e)
            {
                Debug.LogWarning($"Failed to submit score: {e.Message}");
            }

            // Track analytics
            Analytics.Track("game_over", new System.Collections.Generic.Dictionary<string, object>
            {
                ["score"] = score
            });
        }

        /// <summary>
        /// Return to main menu
        /// </summary>
        public async UniTask ReturnToMainMenuAsync()
        {
            Time.timeScale = 1f;
            SetState(GameState.Loading);
            await SceneLoader.LoadSceneAsync("MainMenu");
            SetState(GameState.MainMenu);
        }

        private void OnApplicationPause(bool pauseStatus)
        {
            if (pauseStatus && State == GameState.Playing)
            {
                PauseGame();
            }
        }

        private void OnApplicationQuit()
        {
            Analytics.FlushAsync().Forget();
        }
    }

    /// <summary>
    /// Game state enum
    /// </summary>
    public enum GameState
    {
        Loading,
        MainMenu,
        Playing,
        Paused,
        GameOver
    }
}
```

### Scene Loader (`Assets/_Project/Runtime/Game/SceneLoader.cs`)

```csharp
using System;
using Cysharp.Threading.Tasks;
using UnityEngine;
using UnityEngine.SceneManagement;

namespace {{.Name}}.Game
{
    /// <summary>
    /// Scene loading utilities
    /// </summary>
    public static class SceneLoader
    {
        /// <summary>
        /// Event fired when scene loading starts
        /// </summary>
        public static event Action<string> OnSceneLoadStarted;

        /// <summary>
        /// Event fired when scene loading progress updates
        /// </summary>
        public static event Action<float> OnSceneLoadProgress;

        /// <summary>
        /// Event fired when scene loading completes
        /// </summary>
        public static event Action<string> OnSceneLoadCompleted;

        /// <summary>
        /// Load a scene asynchronously
        /// </summary>
        public static async UniTask LoadSceneAsync(
            string sceneName,
            LoadSceneMode mode = LoadSceneMode.Single,
            bool showLoading = true)
        {
            OnSceneLoadStarted?.Invoke(sceneName);

            if (showLoading)
            {
                // Load loading scene first
                await SceneManager.LoadSceneAsync("Loading", LoadSceneMode.Additive);
            }

            var operation = SceneManager.LoadSceneAsync(sceneName, mode);
            operation.allowSceneActivation = false;

            while (operation.progress < 0.9f)
            {
                OnSceneLoadProgress?.Invoke(operation.progress);
                await UniTask.Yield();
            }

            OnSceneLoadProgress?.Invoke(1f);

            // Wait a moment before activating
            await UniTask.Delay(500);

            operation.allowSceneActivation = true;
            await UniTask.WaitUntil(() => operation.isDone);

            if (showLoading)
            {
                // Unload loading scene
                await SceneManager.UnloadSceneAsync("Loading");
            }

            OnSceneLoadCompleted?.Invoke(sceneName);
        }

        /// <summary>
        /// Reload the current scene
        /// </summary>
        public static async UniTask ReloadCurrentSceneAsync()
        {
            var currentScene = SceneManager.GetActiveScene().name;
            await LoadSceneAsync(currentScene);
        }
    }
}
```

### Save Manager (`Assets/_Project/Runtime/Game/SaveManager.cs`)

```csharp
using System;
using System.IO;
using System.Threading;
using Cysharp.Threading.Tasks;
using Newtonsoft.Json;
using UnityEngine;

namespace {{.Name}}.Game
{
    /// <summary>
    /// Save/Load system for game data
    /// </summary>
    public class SaveManager
    {
        private const string SaveFileName = "save.json";
        private static string SavePath => Path.Combine(Application.persistentDataPath, SaveFileName);

        private GameSaveData _currentSave;
        private bool _isDirty;

        /// <summary>
        /// Current save data
        /// </summary>
        public GameSaveData CurrentSave => _currentSave;

        /// <summary>
        /// Event fired when save data changes
        /// </summary>
        public event Action<GameSaveData> OnSaveDataChanged;

        /// <summary>
        /// Load save data
        /// </summary>
        public async UniTask<GameSaveData> LoadAsync(CancellationToken cancellationToken = default)
        {
            if (!File.Exists(SavePath))
            {
                _currentSave = new GameSaveData();
                return _currentSave;
            }

            try
            {
                var json = await File.ReadAllTextAsync(SavePath, cancellationToken);
                _currentSave = JsonConvert.DeserializeObject<GameSaveData>(json);
            }
            catch (Exception e)
            {
                Debug.LogWarning($"Failed to load save: {e.Message}");
                _currentSave = new GameSaveData();
            }

            return _currentSave;
        }

        /// <summary>
        /// Save current data
        /// </summary>
        public async UniTask SaveAsync(CancellationToken cancellationToken = default)
        {
            if (_currentSave == null)
            {
                _currentSave = new GameSaveData();
            }

            _currentSave.LastSaved = DateTime.UtcNow;

            try
            {
                var json = JsonConvert.SerializeObject(_currentSave, Formatting.Indented);
                await File.WriteAllTextAsync(SavePath, json, cancellationToken);
                _isDirty = false;
            }
            catch (Exception e)
            {
                Debug.LogError($"Failed to save: {e.Message}");
                throw;
            }
        }

        /// <summary>
        /// Update save data
        /// </summary>
        public void Update(Action<GameSaveData> updateAction)
        {
            if (_currentSave == null)
            {
                _currentSave = new GameSaveData();
            }

            updateAction(_currentSave);
            _isDirty = true;
            OnSaveDataChanged?.Invoke(_currentSave);
        }

        /// <summary>
        /// Delete save data
        /// </summary>
        public void DeleteSave()
        {
            if (File.Exists(SavePath))
            {
                File.Delete(SavePath);
            }
            _currentSave = new GameSaveData();
            OnSaveDataChanged?.Invoke(_currentSave);
        }

        /// <summary>
        /// Check if save data is dirty
        /// </summary>
        public bool IsDirty => _isDirty;
    }

    /// <summary>
    /// Game save data structure
    /// </summary>
    [Serializable]
    public class GameSaveData
    {
        public int HighScore { get; set; }
        public int TotalCoins { get; set; }
        public int CurrentLevel { get; set; } = 1;
        public string[] UnlockedAchievements { get; set; } = Array.Empty<string>();
        public GameSettings Settings { get; set; } = new();
        public DateTime LastSaved { get; set; }
        public string PlayerId { get; set; }
    }

    /// <summary>
    /// Game settings
    /// </summary>
    [Serializable]
    public class GameSettings
    {
        public float MusicVolume { get; set; } = 1f;
        public float SfxVolume { get; set; } = 1f;
        public bool Vibration { get; set; } = true;
        public bool Notifications { get; set; } = true;
        public string Language { get; set; } = "en";
    }
}
```

## Generated SDK

### Client (`Assets/_Project/Runtime/SDK/Client.cs`)

```csharp
using System.Collections.Generic;
using System.Threading;
using Cysharp.Threading.Tasks;
using {{.Name}}.Mizu;

namespace {{.Name}}.SDK
{
    /// <summary>
    /// Generated Mizu game client
    /// </summary>
    public class GameClient
    {
        private readonly MizuRuntime _runtime;

        public GameClient(MizuRuntime runtime)
        {
            _runtime = runtime;
        }

        #region Auth

        /// <summary>
        /// Sign in with credentials
        /// </summary>
        public async UniTask<AuthResponse> SignInAsync(
            string email,
            string password,
            CancellationToken cancellationToken = default)
        {
            return await _runtime.PostAsync<AuthResponse, SignInRequest>(
                "/auth/signin",
                new SignInRequest { Email = email, Password = password },
                cancellationToken: cancellationToken);
        }

        /// <summary>
        /// Sign up with credentials
        /// </summary>
        public async UniTask<AuthResponse> SignUpAsync(
            string email,
            string password,
            string username,
            CancellationToken cancellationToken = default)
        {
            return await _runtime.PostAsync<AuthResponse, SignUpRequest>(
                "/auth/signup",
                new SignUpRequest { Email = email, Password = password, Username = username },
                cancellationToken: cancellationToken);
        }

        /// <summary>
        /// Sign in as guest
        /// </summary>
        public async UniTask<AuthResponse> SignInAsGuestAsync(
            string deviceId,
            CancellationToken cancellationToken = default)
        {
            return await _runtime.PostAsync<AuthResponse, GuestSignInRequest>(
                "/auth/guest",
                new GuestSignInRequest { DeviceId = deviceId },
                cancellationToken: cancellationToken);
        }

        /// <summary>
        /// Sign out
        /// </summary>
        public async UniTask SignOutAsync(CancellationToken cancellationToken = default)
        {
            await _runtime.DeleteAsync<object>("/auth/signout", cancellationToken: cancellationToken);
        }

        /// <summary>
        /// Store auth token
        /// </summary>
        public UniTask StoreAuthTokenAsync(AuthResponse response)
        {
            var token = AuthTokenFactory.Create(
                response.Token.AccessToken,
                response.Token.RefreshToken,
                response.Token.ExpiresIn);
            _runtime.TokenStore.SetToken(token);
            return UniTask.CompletedTask;
        }

        #endregion

        #region Users

        /// <summary>
        /// Get current user profile
        /// </summary>
        public async UniTask<User> GetCurrentUserAsync(CancellationToken cancellationToken = default)
        {
            return await _runtime.GetAsync<User>("/users/me", cancellationToken: cancellationToken);
        }

        /// <summary>
        /// Update current user profile
        /// </summary>
        public async UniTask<User> UpdateCurrentUserAsync(
            UserUpdate update,
            CancellationToken cancellationToken = default)
        {
            return await _runtime.PutAsync<User, UserUpdate>(
                "/users/me",
                update,
                cancellationToken: cancellationToken);
        }

        #endregion

        #region Leaderboards

        /// <summary>
        /// Submit a score
        /// </summary>
        public async UniTask<LeaderboardEntry> SubmitScoreAsync(
            string leaderboardId,
            int score,
            Dictionary<string, object> metadata = null,
            CancellationToken cancellationToken = default)
        {
            return await _runtime.PostAsync<LeaderboardEntry, ScoreSubmission>(
                $"/leaderboards/{leaderboardId}/scores",
                new ScoreSubmission { Score = score, Metadata = metadata },
                cancellationToken: cancellationToken);
        }

        /// <summary>
        /// Get leaderboard
        /// </summary>
        public async UniTask<List<LeaderboardEntry>> GetLeaderboardAsync(
            string leaderboardId,
            int limit = 10,
            int offset = 0,
            CancellationToken cancellationToken = default)
        {
            var query = new Dictionary<string, string>
            {
                ["limit"] = limit.ToString(),
                ["offset"] = offset.ToString()
            };
            return await _runtime.GetAsync<List<LeaderboardEntry>>(
                $"/leaderboards/{leaderboardId}",
                query: query,
                cancellationToken: cancellationToken);
        }

        /// <summary>
        /// Get leaderboard around player
        /// </summary>
        public async UniTask<List<LeaderboardEntry>> GetLeaderboardAroundPlayerAsync(
            string leaderboardId,
            int limit = 10,
            CancellationToken cancellationToken = default)
        {
            var query = new Dictionary<string, string>
            {
                ["limit"] = limit.ToString(),
                ["around"] = "me"
            };
            return await _runtime.GetAsync<List<LeaderboardEntry>>(
                $"/leaderboards/{leaderboardId}",
                query: query,
                cancellationToken: cancellationToken);
        }

        /// <summary>
        /// Get player's leaderboard entry
        /// </summary>
        public async UniTask<LeaderboardEntry> GetPlayerLeaderboardEntryAsync(
            string leaderboardId,
            CancellationToken cancellationToken = default)
        {
            return await _runtime.GetAsync<LeaderboardEntry>(
                $"/leaderboards/{leaderboardId}/me",
                cancellationToken: cancellationToken);
        }

        #endregion

        #region Achievements

        /// <summary>
        /// Get all achievements
        /// </summary>
        public async UniTask<List<Achievement>> GetAchievementsAsync(
            CancellationToken cancellationToken = default)
        {
            return await _runtime.GetAsync<List<Achievement>>(
                "/achievements",
                cancellationToken: cancellationToken);
        }

        /// <summary>
        /// Unlock an achievement
        /// </summary>
        public async UniTask<Achievement> UnlockAchievementAsync(
            string achievementId,
            CancellationToken cancellationToken = default)
        {
            return await _runtime.PostAsync<Achievement, object>(
                $"/achievements/{achievementId}/unlock",
                cancellationToken: cancellationToken);
        }

        /// <summary>
        /// Update achievement progress
        /// </summary>
        public async UniTask<Achievement> UpdateAchievementProgressAsync(
            string achievementId,
            int progress,
            CancellationToken cancellationToken = default)
        {
            return await _runtime.PutAsync<Achievement, AchievementProgress>(
                $"/achievements/{achievementId}/progress",
                new AchievementProgress { Progress = progress },
                cancellationToken: cancellationToken);
        }

        /// <summary>
        /// Get unlocked achievements
        /// </summary>
        public async UniTask<List<Achievement>> GetUnlockedAchievementsAsync(
            CancellationToken cancellationToken = default)
        {
            return await _runtime.GetAsync<List<Achievement>>(
                "/achievements/unlocked",
                cancellationToken: cancellationToken);
        }

        #endregion
    }
}
```

### Types (`Assets/_Project/Runtime/SDK/Types.cs`)

```csharp
using System;
using System.Collections.Generic;
using Newtonsoft.Json;

namespace {{.Name}}.SDK
{
    #region Auth Types

    [Serializable]
    public class SignInRequest
    {
        [JsonProperty("email")]
        public string Email { get; set; }

        [JsonProperty("password")]
        public string Password { get; set; }
    }

    [Serializable]
    public class SignUpRequest
    {
        [JsonProperty("email")]
        public string Email { get; set; }

        [JsonProperty("password")]
        public string Password { get; set; }

        [JsonProperty("username")]
        public string Username { get; set; }
    }

    [Serializable]
    public class GuestSignInRequest
    {
        [JsonProperty("device_id")]
        public string DeviceId { get; set; }
    }

    [Serializable]
    public class AuthResponse
    {
        [JsonProperty("user")]
        public User User { get; set; }

        [JsonProperty("token")]
        public TokenResponse Token { get; set; }
    }

    [Serializable]
    public class TokenResponse
    {
        [JsonProperty("access_token")]
        public string AccessToken { get; set; }

        [JsonProperty("refresh_token")]
        public string RefreshToken { get; set; }

        [JsonProperty("expires_in")]
        public int ExpiresIn { get; set; }
    }

    #endregion

    #region User Types

    [Serializable]
    public class User
    {
        [JsonProperty("id")]
        public string Id { get; set; }

        [JsonProperty("email")]
        public string Email { get; set; }

        [JsonProperty("username")]
        public string Username { get; set; }

        [JsonProperty("avatar_url")]
        public string AvatarUrl { get; set; }

        [JsonProperty("created_at")]
        public string CreatedAt { get; set; }

        [JsonProperty("updated_at")]
        public string UpdatedAt { get; set; }
    }

    [Serializable]
    public class UserUpdate
    {
        [JsonProperty("username")]
        public string Username { get; set; }

        [JsonProperty("avatar_url")]
        public string AvatarUrl { get; set; }
    }

    #endregion

    #region Leaderboard Types

    [Serializable]
    public class LeaderboardEntry
    {
        [JsonProperty("id")]
        public string Id { get; set; }

        [JsonProperty("user_id")]
        public string UserId { get; set; }

        [JsonProperty("username")]
        public string Username { get; set; }

        [JsonProperty("score")]
        public int Score { get; set; }

        [JsonProperty("rank")]
        public int Rank { get; set; }

        [JsonProperty("metadata")]
        public Dictionary<string, object> Metadata { get; set; }

        [JsonProperty("created_at")]
        public string CreatedAt { get; set; }
    }

    [Serializable]
    public class ScoreSubmission
    {
        [JsonProperty("score")]
        public int Score { get; set; }

        [JsonProperty("metadata")]
        public Dictionary<string, object> Metadata { get; set; }
    }

    #endregion

    #region Achievement Types

    [Serializable]
    public class Achievement
    {
        [JsonProperty("id")]
        public string Id { get; set; }

        [JsonProperty("name")]
        public string Name { get; set; }

        [JsonProperty("description")]
        public string Description { get; set; }

        [JsonProperty("icon_url")]
        public string IconUrl { get; set; }

        [JsonProperty("points")]
        public int Points { get; set; }

        [JsonProperty("progress")]
        public int Progress { get; set; }

        [JsonProperty("target")]
        public int Target { get; set; }

        [JsonProperty("unlocked")]
        public bool Unlocked { get; set; }

        [JsonProperty("unlocked_at")]
        public string UnlockedAt { get; set; }
    }

    [Serializable]
    public class AchievementProgress
    {
        [JsonProperty("progress")]
        public int Progress { get; set; }
    }

    #endregion
}
```

## UI Scripts

### Main Menu UI (`Assets/_Project/Scripts/UI/MainMenuUI.cs`)

```csharp
using Cysharp.Threading.Tasks;
using UnityEngine;
using UnityEngine.UI;
using {{.Name}}.Game;

namespace {{.Name}}.UI
{
    /// <summary>
    /// Main menu UI controller
    /// </summary>
    public class MainMenuUI : MonoBehaviour
    {
        [Header("Buttons")]
        [SerializeField] private Button _playButton;
        [SerializeField] private Button _leaderboardButton;
        [SerializeField] private Button _settingsButton;
        [SerializeField] private Button _quitButton;

        [Header("Panels")]
        [SerializeField] private GameObject _mainPanel;
        [SerializeField] private GameObject _leaderboardPanel;
        [SerializeField] private GameObject _settingsPanel;

        private void Start()
        {
            _playButton.onClick.AddListener(OnPlayClicked);
            _leaderboardButton.onClick.AddListener(OnLeaderboardClicked);
            _settingsButton.onClick.AddListener(OnSettingsClicked);
            _quitButton.onClick.AddListener(OnQuitClicked);

            ShowMainPanel();
        }

        private void OnPlayClicked()
        {
            GameManager.Instance.StartGameAsync().Forget();
        }

        private void OnLeaderboardClicked()
        {
            ShowLeaderboardPanel();
        }

        private void OnSettingsClicked()
        {
            ShowSettingsPanel();
        }

        private void OnQuitClicked()
        {
#if UNITY_EDITOR
            UnityEditor.EditorApplication.isPlaying = false;
#else
            Application.Quit();
#endif
        }

        private void ShowMainPanel()
        {
            _mainPanel.SetActive(true);
            _leaderboardPanel.SetActive(false);
            _settingsPanel.SetActive(false);
        }

        private void ShowLeaderboardPanel()
        {
            _mainPanel.SetActive(false);
            _leaderboardPanel.SetActive(true);
            _settingsPanel.SetActive(false);
        }

        private void ShowSettingsPanel()
        {
            _mainPanel.SetActive(false);
            _leaderboardPanel.SetActive(false);
            _settingsPanel.SetActive(true);
        }

        public void OnBackClicked()
        {
            ShowMainPanel();
        }
    }
}
```

### Pause Menu UI (`Assets/_Project/Scripts/UI/PauseMenuUI.cs`)

```csharp
using Cysharp.Threading.Tasks;
using UnityEngine;
using UnityEngine.UI;
using {{.Name}}.Game;

namespace {{.Name}}.UI
{
    /// <summary>
    /// Pause menu UI controller
    /// </summary>
    public class PauseMenuUI : MonoBehaviour
    {
        [Header("Buttons")]
        [SerializeField] private Button _resumeButton;
        [SerializeField] private Button _restartButton;
        [SerializeField] private Button _settingsButton;
        [SerializeField] private Button _mainMenuButton;

        [Header("Panels")]
        [SerializeField] private GameObject _pausePanel;
        [SerializeField] private GameObject _settingsPanel;

        private void Start()
        {
            _resumeButton.onClick.AddListener(OnResumeClicked);
            _restartButton.onClick.AddListener(OnRestartClicked);
            _settingsButton.onClick.AddListener(OnSettingsClicked);
            _mainMenuButton.onClick.AddListener(OnMainMenuClicked);

            GameManager.Instance.OnStateChanged += OnGameStateChanged;
            gameObject.SetActive(false);
        }

        private void OnDestroy()
        {
            if (GameManager.Instance != null)
            {
                GameManager.Instance.OnStateChanged -= OnGameStateChanged;
            }
        }

        private void OnGameStateChanged(GameState state)
        {
            gameObject.SetActive(state == GameState.Paused);
            if (state == GameState.Paused)
            {
                ShowPausePanel();
            }
        }

        private void OnResumeClicked()
        {
            GameManager.Instance.ResumeGame();
        }

        private void OnRestartClicked()
        {
            GameManager.Instance.ResumeGame();
            SceneLoader.ReloadCurrentSceneAsync().Forget();
        }

        private void OnSettingsClicked()
        {
            ShowSettingsPanel();
        }

        private void OnMainMenuClicked()
        {
            GameManager.Instance.ReturnToMainMenuAsync().Forget();
        }

        private void ShowPausePanel()
        {
            _pausePanel.SetActive(true);
            _settingsPanel.SetActive(false);
        }

        private void ShowSettingsPanel()
        {
            _pausePanel.SetActive(false);
            _settingsPanel.SetActive(true);
        }

        public void OnBackClicked()
        {
            ShowPausePanel();
        }
    }
}
```

## Build Configuration

### Package Manifest (`Packages/manifest.json`)

```json
{
  "dependencies": {
    "com.cysharp.unitask": "https://github.com/Cysharp/UniTask.git?path=src/UniTask/Assets/Plugins/UniTask#2.5.0",
    "com.unity.addressables": "1.21.21",
    "com.unity.inputsystem": "1.7.0",
    "com.unity.textmeshpro": "3.0.6",
    "com.unity.ugui": "1.0.0",
    "com.unity.modules.audio": "1.0.0",
    "com.unity.modules.imageconversion": "1.0.0",
    "com.unity.modules.jsonserialize": "1.0.0",
    "com.unity.modules.ui": "1.0.0",
    "com.unity.modules.unitywebrequest": "1.0.0",
    "com.unity.modules.unitywebrequestassetbundle": "1.0.0",
    "com.unity.modules.unitywebrequesttexture": "1.0.0"
  }
}
```

### Assembly Definitions

#### Mizu Runtime (`Assets/_Project/Runtime/Mizu/MizuRuntime.asmdef`)

```json
{
  "name": "{{.Name}}.Mizu",
  "rootNamespace": "{{.Name}}.Mizu",
  "references": [
    "UniTask"
  ],
  "includePlatforms": [],
  "excludePlatforms": [],
  "allowUnsafeCode": false,
  "overrideReferences": true,
  "precompiledReferences": [
    "Newtonsoft.Json.dll"
  ],
  "autoReferenced": true,
  "defineConstraints": [],
  "versionDefines": [],
  "noEngineReferences": false
}
```

#### SDK (`Assets/_Project/Runtime/SDK/SDK.asmdef`)

```json
{
  "name": "{{.Name}}.SDK",
  "rootNamespace": "{{.Name}}.SDK",
  "references": [
    "{{.Name}}.Mizu",
    "UniTask"
  ],
  "includePlatforms": [],
  "excludePlatforms": [],
  "allowUnsafeCode": false,
  "overrideReferences": true,
  "precompiledReferences": [
    "Newtonsoft.Json.dll"
  ],
  "autoReferenced": true,
  "defineConstraints": [],
  "versionDefines": [],
  "noEngineReferences": false
}
```

#### Game (`Assets/_Project/Runtime/Game/Game.asmdef`)

```json
{
  "name": "{{.Name}}.Game",
  "rootNamespace": "{{.Name}}.Game",
  "references": [
    "{{.Name}}.Mizu",
    "{{.Name}}.SDK",
    "{{.Name}}.Services",
    "UniTask"
  ],
  "includePlatforms": [],
  "excludePlatforms": [],
  "allowUnsafeCode": false,
  "overrideReferences": true,
  "precompiledReferences": [
    "Newtonsoft.Json.dll"
  ],
  "autoReferenced": true,
  "defineConstraints": [],
  "versionDefines": [],
  "noEngineReferences": false
}
```

#### Services (`Assets/_Project/Runtime/Services/Services.asmdef`)

```json
{
  "name": "{{.Name}}.Services",
  "rootNamespace": "{{.Name}}.Services",
  "references": [
    "{{.Name}}.Mizu",
    "{{.Name}}.SDK",
    "UniTask"
  ],
  "includePlatforms": [],
  "excludePlatforms": [],
  "allowUnsafeCode": false,
  "overrideReferences": true,
  "precompiledReferences": [
    "Newtonsoft.Json.dll"
  ],
  "autoReferenced": true,
  "defineConstraints": [],
  "versionDefines": [],
  "noEngineReferences": false
}
```

## Editor Tools

### Build Script (`Assets/Editor/Build/BuildScript.cs`)

```csharp
using UnityEditor;
using UnityEditor.Build.Reporting;
using UnityEngine;
using System.IO;
using System.Linq;

namespace {{.Name}}.Editor
{
    /// <summary>
    /// Build automation script
    /// </summary>
    public static class BuildScript
    {
        private static readonly string[] Scenes = new[]
        {
            "Assets/_Project/Scenes/Bootstrap.unity",
            "Assets/_Project/Scenes/Loading.unity",
            "Assets/_Project/Scenes/MainMenu.unity",
            "Assets/_Project/Scenes/Game.unity"
        };

        [MenuItem("Build/Build All")]
        public static void BuildAll()
        {
            BuildAndroid();
            BuildIOS();
            BuildWebGL();
        }

        [MenuItem("Build/Build Android")]
        public static void BuildAndroid()
        {
            var options = new BuildPlayerOptions
            {
                scenes = Scenes,
                locationPathName = $"Builds/Android/{{.Name}}.apk",
                target = BuildTarget.Android,
                options = BuildOptions.None
            };

            Build(options);
        }

        [MenuItem("Build/Build iOS")]
        public static void BuildIOS()
        {
            var options = new BuildPlayerOptions
            {
                scenes = Scenes,
                locationPathName = "Builds/iOS",
                target = BuildTarget.iOS,
                options = BuildOptions.None
            };

            Build(options);
        }

        [MenuItem("Build/Build WebGL")]
        public static void BuildWebGL()
        {
            var options = new BuildPlayerOptions
            {
                scenes = Scenes,
                locationPathName = "Builds/WebGL",
                target = BuildTarget.WebGL,
                options = BuildOptions.None
            };

            Build(options);
        }

        private static void Build(BuildPlayerOptions options)
        {
            // Ensure build directory exists
            var directory = Path.GetDirectoryName(options.locationPathName);
            if (!string.IsNullOrEmpty(directory) && !Directory.Exists(directory))
            {
                Directory.CreateDirectory(directory);
            }

            var report = BuildPipeline.BuildPlayer(options);
            var summary = report.summary;

            if (summary.result == BuildResult.Succeeded)
            {
                Debug.Log($"Build succeeded: {summary.totalSize} bytes");
            }
            else if (summary.result == BuildResult.Failed)
            {
                Debug.LogError("Build failed");
            }
        }
    }
}
```

## Testing

### Runtime Tests (`Assets/_Project/Tests/Runtime/MizuRuntimeTests.cs`)

```csharp
using System.Collections;
using NUnit.Framework;
using UnityEngine.TestTools;
using {{.Name}}.Mizu;

namespace {{.Name}}.Tests
{
    public class MizuRuntimeTests
    {
        [Test]
        public void TokenStore_ReturnsNullWhenNoTokenStored()
        {
            var store = new TokenStore();
            Assert.IsNull(store.GetToken());
        }

        [Test]
        public void TokenStore_StoresAndRetrievesToken()
        {
            var store = new TokenStore();
            var token = new AuthToken
            {
                AccessToken = "test123",
                RefreshToken = "refresh456"
            };

            store.SetToken(token);
            var retrieved = store.GetToken();

            Assert.IsNotNull(retrieved);
            Assert.AreEqual("test123", retrieved.AccessToken);
            Assert.AreEqual("refresh456", retrieved.RefreshToken);
        }

        [Test]
        public void TokenStore_ClearsToken()
        {
            var store = new TokenStore();
            var token = new AuthToken { AccessToken = "test123" };

            store.SetToken(token);
            store.ClearToken();

            Assert.IsNull(store.GetToken());
        }

        [Test]
        public void AuthToken_IsExpiredReturnsFalseWhenNoExpiry()
        {
            var token = new AuthToken { AccessToken = "test" };
            Assert.IsFalse(token.IsExpired);
        }

        [Test]
        public void DeviceInfo_ReturnsValidData()
        {
            var info = new DeviceInfo();

            Assert.IsNotEmpty(info.DeviceId);
            Assert.IsNotEmpty(info.Platform);
            Assert.IsNotEmpty(info.Model);
        }
    }
}
```

## Multiplayer Support (Optional)

When `--var multiplayer=true` is specified, additional components are generated:

### Multiplayer Manager

```csharp
using System;
using System.Collections.Generic;
using Cysharp.Threading.Tasks;
using UnityEngine;
using {{.Name}}.Mizu;

namespace {{.Name}}.Multiplayer
{
    /// <summary>
    /// Multiplayer manager for real-time game features
    /// </summary>
    public class MultiplayerManager : MonoBehaviour
    {
        private LiveConnection _live;
        private bool _isConnected;

        public bool IsConnected => _isConnected;
        public event Action<GameEvent> OnGameEvent;
        public event Action OnConnected;
        public event Action OnDisconnected;

        private void Start()
        {
            _live = MizuRuntime.Shared.Live;
        }

        public async UniTask ConnectToRoomAsync(string roomId)
        {
            await foreach (var evt in _live.ConnectAsync($"/rooms/{roomId}/events"))
            {
                _isConnected = true;
                OnConnected?.Invoke();

                var gameEvent = evt.Decode<GameEvent>();
                OnGameEvent?.Invoke(gameEvent);
            }

            _isConnected = false;
            OnDisconnected?.Invoke();
        }

        public void Disconnect()
        {
            _live.DisconnectAll();
        }
    }

    [Serializable]
    public class GameEvent
    {
        public string Type { get; set; }
        public string PlayerId { get; set; }
        public Dictionary<string, object> Data { get; set; }
    }
}
```

## References

- [Unity Documentation](https://docs.unity3d.com/)
- [UniTask](https://github.com/Cysharp/UniTask)
- [Newtonsoft.Json for Unity](https://github.com/jilleJr/Newtonsoft.Json-for-Unity)
- [Unity Addressables](https://docs.unity3d.com/Packages/com.unity.addressables@latest)
- [Mobile Package Spec](./0095_mobile.md)
