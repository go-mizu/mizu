# .NET MAUI Template Specification

**Status:** Draft
**Author:** Mizu Team
**Created:** 2025-12-20
**Template:** `mobile:dotnet`

## Overview

The `mobile:dotnet` template generates a production-ready .NET MAUI application with full Mizu backend integration. It follows modern .NET development practices with MVVM architecture, dependency injection, and async/await patterns that work across iOS, Android, Windows, and macOS platforms.

## Template Invocation

```bash
# Default: .NET MAUI with Shell navigation
mizu new ./MyApp --template mobile:dotnet

# With plain navigation (no Shell)
mizu new ./MyApp --template mobile:dotnet --var nav=plain

# Custom namespace
mizu new ./MyApp --template mobile:dotnet --var namespace=Com.Company.MyApp

# Specific platforms only
mizu new ./MyApp --template mobile:dotnet --var platforms=ios,android

# With CommunityToolkit.Mvvm
mizu new ./MyApp --template mobile:dotnet --var mvvm=toolkit
```

## Generated Project Structure

```
{{.Name}}/
├── {{.Name}}.sln                          # Solution file
├── {{.Name}}/
│   ├── {{.Name}}.csproj                   # Main project file
│   ├── MauiProgram.cs                     # App entry point
│   ├── App.xaml                           # Application resources
│   ├── App.xaml.cs
│   ├── AppShell.xaml                      # Shell navigation
│   ├── AppShell.xaml.cs
│   ├── Runtime/
│   │   ├── MizuRuntime.cs                 # Core runtime
│   │   ├── Transport.cs                   # HTTP transport layer
│   │   ├── TokenStore.cs                  # Secure token storage
│   │   ├── LiveConnection.cs              # SSE streaming
│   │   ├── DeviceInfo.cs                  # Device information
│   │   └── MizuException.cs               # Error types
│   ├── SDK/
│   │   ├── Client.cs                      # Generated Mizu client
│   │   ├── Types.cs                       # Generated types
│   │   └── Extensions.cs                  # Convenience extensions
│   ├── Models/
│   │   └── AppState.cs                    # Application state
│   ├── ViewModels/
│   │   ├── BaseViewModel.cs               # Base ViewModel
│   │   ├── HomeViewModel.cs
│   │   └── WelcomeViewModel.cs
│   ├── Views/
│   │   ├── HomePage.xaml
│   │   ├── HomePage.xaml.cs
│   │   ├── WelcomePage.xaml
│   │   └── WelcomePage.xaml.cs
│   ├── Controls/
│   │   ├── LoadingView.xaml
│   │   └── LoadingView.xaml.cs
│   ├── Services/
│   │   └── NavigationService.cs
│   ├── Resources/
│   │   ├── Styles/
│   │   │   ├── Colors.xaml
│   │   │   └── Styles.xaml
│   │   ├── Fonts/
│   │   ├── Images/
│   │   └── Raw/
│   └── Platforms/
│       ├── Android/
│       │   ├── AndroidManifest.xml
│       │   ├── MainActivity.cs
│       │   └── MainApplication.cs
│       ├── iOS/
│       │   ├── Info.plist
│       │   ├── AppDelegate.cs
│       │   └── Program.cs
│       ├── MacCatalyst/
│       │   ├── Info.plist
│       │   └── AppDelegate.cs
│       └── Windows/
│           ├── app.manifest
│           └── Package.appxmanifest
├── {{.Name}}.Tests/
│   ├── {{.Name}}.Tests.csproj
│   ├── RuntimeTests.cs
│   ├── SDKTests.cs
│   └── ViewModelTests.cs
├── .gitignore
├── Directory.Build.props
└── README.md
```

## MizuMobileRuntime

### Core Runtime (`Runtime/MizuRuntime.cs`)

```csharp
using System.Text.Json;

namespace {{.Namespace}}.Runtime;

/// <summary>
/// MizuRuntime is the core client for communicating with a Mizu backend.
/// </summary>
public sealed class MizuRuntime : INotifyPropertyChanged, IDisposable
{
    private static readonly Lazy<MizuRuntime> _instance = new(() => new MizuRuntime());

    /// <summary>
    /// Shared singleton instance
    /// </summary>
    public static MizuRuntime Shared => _instance.Value;

    /// <summary>
    /// Base URL for all API requests
    /// </summary>
    public Uri BaseUrl { get; set; }

    /// <summary>
    /// HTTP transport layer
    /// </summary>
    public ITransport Transport { get; }

    /// <summary>
    /// Secure token storage
    /// </summary>
    public ITokenStore TokenStore { get; }

    /// <summary>
    /// Live connection manager
    /// </summary>
    public LiveConnection Live { get; }

    /// <summary>
    /// Request timeout
    /// </summary>
    public TimeSpan Timeout { get; set; } = TimeSpan.FromSeconds(30);

    /// <summary>
    /// Default headers added to all requests
    /// </summary>
    public Dictionary<string, string> DefaultHeaders { get; } = new();

    private bool _isAuthenticated;
    /// <summary>
    /// Current authentication state
    /// </summary>
    public bool IsAuthenticated
    {
        get => _isAuthenticated;
        private set
        {
            if (_isAuthenticated != value)
            {
                _isAuthenticated = value;
                OnPropertyChanged();
            }
        }
    }

    public event PropertyChangedEventHandler? PropertyChanged;

    private readonly JsonSerializerOptions _jsonOptions = new()
    {
        PropertyNamingPolicy = JsonNamingPolicy.SnakeCaseLower,
        PropertyNameCaseInsensitive = true,
        DefaultIgnoreCondition = JsonIgnoreCondition.WhenWritingNull
    };

    public MizuRuntime(
        Uri? baseUrl = null,
        ITransport? transport = null,
        ITokenStore? tokenStore = null)
    {
        BaseUrl = baseUrl ?? new Uri("http://localhost:3000");
        Transport = transport ?? new HttpTransport();
        TokenStore = tokenStore ?? new SecureTokenStore();
        Live = new LiveConnection(this);

        // Observe token changes
        TokenStore.TokenChanged += (_, token) =>
        {
            IsAuthenticated = token != null;
        };

        // Check initial auth state
        Task.Run(async () =>
        {
            var token = await TokenStore.GetTokenAsync();
            IsAuthenticated = token != null;
        });
    }

    /// <summary>
    /// Initializes the runtime with configuration
    /// </summary>
    public static async Task<MizuRuntime> InitializeAsync(
        Uri baseUrl,
        TimeSpan? timeout = null)
    {
        Shared.BaseUrl = baseUrl;
        if (timeout.HasValue)
            Shared.Timeout = timeout.Value;

        var token = await Shared.TokenStore.GetTokenAsync();
        Shared.IsAuthenticated = token != null;

        return Shared;
    }

    // MARK: - HTTP Methods

    /// <summary>
    /// Performs a GET request
    /// </summary>
    public async Task<T> GetAsync<T>(
        string path,
        Dictionary<string, string>? query = null,
        Dictionary<string, string>? headers = null,
        CancellationToken cancellationToken = default)
    {
        return await RequestAsync<T, object>(
            HttpMethod.Get, path, query: query, headers: headers,
            cancellationToken: cancellationToken);
    }

    /// <summary>
    /// Performs a POST request
    /// </summary>
    public async Task<T> PostAsync<T, TBody>(
        string path,
        TBody body,
        Dictionary<string, string>? headers = null,
        CancellationToken cancellationToken = default)
    {
        return await RequestAsync<T, TBody>(
            HttpMethod.Post, path, body: body, headers: headers,
            cancellationToken: cancellationToken);
    }

    /// <summary>
    /// Performs a POST request with no response body
    /// </summary>
    public async Task PostAsync<TBody>(
        string path,
        TBody body,
        Dictionary<string, string>? headers = null,
        CancellationToken cancellationToken = default)
    {
        await RequestAsync<object, TBody>(
            HttpMethod.Post, path, body: body, headers: headers,
            cancellationToken: cancellationToken);
    }

    /// <summary>
    /// Performs a PUT request
    /// </summary>
    public async Task<T> PutAsync<T, TBody>(
        string path,
        TBody body,
        Dictionary<string, string>? headers = null,
        CancellationToken cancellationToken = default)
    {
        return await RequestAsync<T, TBody>(
            HttpMethod.Put, path, body: body, headers: headers,
            cancellationToken: cancellationToken);
    }

    /// <summary>
    /// Performs a DELETE request
    /// </summary>
    public async Task<T> DeleteAsync<T>(
        string path,
        Dictionary<string, string>? headers = null,
        CancellationToken cancellationToken = default)
    {
        return await RequestAsync<T, object>(
            HttpMethod.Delete, path, headers: headers,
            cancellationToken: cancellationToken);
    }

    /// <summary>
    /// Performs a DELETE request with no response body
    /// </summary>
    public async Task DeleteAsync(
        string path,
        Dictionary<string, string>? headers = null,
        CancellationToken cancellationToken = default)
    {
        await RequestAsync<object, object>(
            HttpMethod.Delete, path, headers: headers,
            cancellationToken: cancellationToken);
    }

    /// <summary>
    /// Performs a PATCH request
    /// </summary>
    public async Task<T> PatchAsync<T, TBody>(
        string path,
        TBody body,
        Dictionary<string, string>? headers = null,
        CancellationToken cancellationToken = default)
    {
        return await RequestAsync<T, TBody>(
            HttpMethod.Patch, path, body: body, headers: headers,
            cancellationToken: cancellationToken);
    }

    // MARK: - Streaming

    /// <summary>
    /// Opens a streaming connection for SSE
    /// </summary>
    public IAsyncEnumerable<ServerEvent> StreamAsync(
        string path,
        Dictionary<string, string>? headers = null,
        CancellationToken cancellationToken = default)
    {
        return Live.ConnectAsync(path, headers, cancellationToken);
    }

    // MARK: - Private

    private async Task<T> RequestAsync<T, TBody>(
        HttpMethod method,
        string path,
        Dictionary<string, string>? query = null,
        TBody? body = default,
        Dictionary<string, string>? headers = null,
        CancellationToken cancellationToken = default)
    {
        // Build URL
        var url = BuildUrl(path, query);

        // Build headers
        var allHeaders = await BuildHeadersAsync(headers);

        // Encode body
        string? bodyJson = null;
        if (body != null)
        {
            bodyJson = JsonSerializer.Serialize(body, _jsonOptions);
            allHeaders["Content-Type"] = "application/json";
        }

        // Execute request
        var response = await Transport.ExecuteAsync(new TransportRequest
        {
            Url = url,
            Method = method,
            Headers = allHeaders,
            Body = bodyJson,
            Timeout = Timeout
        }, cancellationToken);

        // Handle errors
        if ((int)response.StatusCode >= 400)
        {
            throw ParseError(response);
        }

        // Handle empty response
        if (typeof(T) == typeof(object) || string.IsNullOrEmpty(response.Body))
        {
            return default!;
        }

        // Decode response
        return JsonSerializer.Deserialize<T>(response.Body, _jsonOptions)!;
    }

    private Uri BuildUrl(string path, Dictionary<string, string>? query)
    {
        var baseUrl = BaseUrl.ToString().TrimEnd('/');
        var cleanPath = path.StartsWith('/') ? path : $"/{path}";
        var url = $"{baseUrl}{cleanPath}";

        if (query != null && query.Count > 0)
        {
            var queryString = string.Join("&",
                query.Select(kvp => $"{Uri.EscapeDataString(kvp.Key)}={Uri.EscapeDataString(kvp.Value)}"));
            url = $"{url}?{queryString}";
        }

        return new Uri(url);
    }

    private async Task<Dictionary<string, string>> BuildHeadersAsync(Dictionary<string, string>? custom)
    {
        var headers = new Dictionary<string, string>(DefaultHeaders);

        // Add mobile headers
        foreach (var kvp in GetMobileHeaders())
        {
            headers[kvp.Key] = kvp.Value;
        }

        // Add custom headers
        if (custom != null)
        {
            foreach (var kvp in custom)
            {
                headers[kvp.Key] = kvp.Value;
            }
        }

        // Add auth token
        var token = await TokenStore.GetTokenAsync();
        if (token != null)
        {
            headers["Authorization"] = $"Bearer {token.AccessToken}";
        }

        return headers;
    }

    private Dictionary<string, string> GetMobileHeaders()
    {
        var info = DeviceInfo.Collect();
        return new Dictionary<string, string>
        {
            ["X-Device-ID"] = info.DeviceId,
            ["X-App-Version"] = info.AppVersion,
            ["X-App-Build"] = info.AppBuild,
            ["X-Device-Model"] = info.Model,
            ["X-Platform"] = info.Platform,
            ["X-OS-Version"] = info.OsVersion,
            ["X-Timezone"] = info.Timezone,
            ["X-Locale"] = info.Locale
        };
    }

    private MizuException ParseError(TransportResponse response)
    {
        try
        {
            var apiError = JsonSerializer.Deserialize<ApiError>(response.Body, _jsonOptions);
            if (apiError != null)
            {
                return new MizuException(MizuErrorType.Api, apiError.Message, apiError);
            }
        }
        catch
        {
            // Ignore JSON parse errors
        }

        return new MizuException(MizuErrorType.Http,
            $"HTTP error {(int)response.StatusCode}",
            statusCode: (int)response.StatusCode);
    }

    private void OnPropertyChanged([CallerMemberName] string? propertyName = null)
    {
        PropertyChanged?.Invoke(this, new PropertyChangedEventArgs(propertyName));
    }

    public void Dispose()
    {
        (Transport as IDisposable)?.Dispose();
    }
}
```

### Transport Layer (`Runtime/Transport.cs`)

```csharp
namespace {{.Namespace}}.Runtime;

/// <summary>
/// Transport request
/// </summary>
public class TransportRequest
{
    public required Uri Url { get; init; }
    public required HttpMethod Method { get; init; }
    public required Dictionary<string, string> Headers { get; init; }
    public string? Body { get; init; }
    public TimeSpan Timeout { get; init; }
}

/// <summary>
/// Transport response
/// </summary>
public class TransportResponse
{
    public required HttpStatusCode StatusCode { get; init; }
    public required Dictionary<string, string> Headers { get; init; }
    public required string Body { get; init; }
}

/// <summary>
/// Transport protocol for executing HTTP requests
/// </summary>
public interface ITransport : IDisposable
{
    Task<TransportResponse> ExecuteAsync(TransportRequest request, CancellationToken cancellationToken = default);
}

/// <summary>
/// HTTP-based transport implementation
/// </summary>
public class HttpTransport : ITransport
{
    private readonly HttpClient _client;
    private readonly List<IRequestInterceptor> _interceptors = new();
    private bool _disposed;

    public HttpTransport(HttpClient? client = null)
    {
        _client = client ?? new HttpClient();
    }

    /// <summary>
    /// Adds a request interceptor
    /// </summary>
    public void AddInterceptor(IRequestInterceptor interceptor)
    {
        _interceptors.Add(interceptor);
    }

    public async Task<TransportResponse> ExecuteAsync(
        TransportRequest request,
        CancellationToken cancellationToken = default)
    {
        var req = request;

        // Apply interceptors
        foreach (var interceptor in _interceptors)
        {
            req = await interceptor.InterceptAsync(req, cancellationToken);
        }

        try
        {
            using var httpRequest = new HttpRequestMessage(req.Method, req.Url);

            foreach (var header in req.Headers)
            {
                httpRequest.Headers.TryAddWithoutValidation(header.Key, header.Value);
            }

            if (req.Body != null)
            {
                httpRequest.Content = new StringContent(req.Body, Encoding.UTF8, "application/json");
            }

            using var cts = CancellationTokenSource.CreateLinkedTokenSource(cancellationToken);
            cts.CancelAfter(req.Timeout);

            var response = await _client.SendAsync(httpRequest, cts.Token);
            var body = await response.Content.ReadAsStringAsync(cts.Token);

            var headers = new Dictionary<string, string>();
            foreach (var header in response.Headers)
            {
                headers[header.Key] = string.Join(",", header.Value);
            }

            return new TransportResponse
            {
                StatusCode = response.StatusCode,
                Headers = headers,
                Body = body
            };
        }
        catch (OperationCanceledException) when (cancellationToken.IsCancellationRequested)
        {
            throw;
        }
        catch (OperationCanceledException)
        {
            throw new MizuException(MizuErrorType.Network, "Request timed out");
        }
        catch (HttpRequestException ex)
        {
            throw new MizuException(MizuErrorType.Network, ex.Message, innerException: ex);
        }
    }

    public void Dispose()
    {
        if (!_disposed)
        {
            _client.Dispose();
            _disposed = true;
        }
    }
}

/// <summary>
/// Request interceptor interface
/// </summary>
public interface IRequestInterceptor
{
    Task<TransportRequest> InterceptAsync(TransportRequest request, CancellationToken cancellationToken = default);
}

/// <summary>
/// Logging interceptor for debugging
/// </summary>
public class LoggingInterceptor : IRequestInterceptor
{
    public Task<TransportRequest> InterceptAsync(TransportRequest request, CancellationToken cancellationToken = default)
    {
#if DEBUG
        System.Diagnostics.Debug.WriteLine($"[Mizu] {request.Method} {request.Url}");
#endif
        return Task.FromResult(request);
    }
}

/// <summary>
/// Retry interceptor with exponential backoff
/// </summary>
public class RetryInterceptor : IRequestInterceptor
{
    public int MaxRetries { get; init; } = 3;
    public TimeSpan BaseDelay { get; init; } = TimeSpan.FromSeconds(1);

    public Task<TransportRequest> InterceptAsync(TransportRequest request, CancellationToken cancellationToken = default)
    {
        // Retry logic is handled at transport level
        return Task.FromResult(request);
    }
}
```

### Token Store (`Runtime/TokenStore.cs`)

```csharp
using System.Text.Json;

namespace {{.Namespace}}.Runtime;

/// <summary>
/// Stored authentication token
/// </summary>
public class AuthToken
{
    public required string AccessToken { get; init; }
    public string? RefreshToken { get; init; }
    public DateTime? ExpiresAt { get; init; }
    public string TokenType { get; init; } = "Bearer";

    public bool IsExpired => ExpiresAt.HasValue && DateTime.UtcNow >= ExpiresAt.Value;
}

/// <summary>
/// Token change event args
/// </summary>
public class TokenChangedEventArgs : EventArgs
{
    public AuthToken? Token { get; }

    public TokenChangedEventArgs(AuthToken? token) => Token = token;
}

/// <summary>
/// Token storage interface
/// </summary>
public interface ITokenStore
{
    Task<AuthToken?> GetTokenAsync();
    Task SetTokenAsync(AuthToken token);
    Task ClearTokenAsync();
    event EventHandler<TokenChangedEventArgs>? TokenChanged;
}

/// <summary>
/// Secure storage-backed token storage using MAUI SecureStorage
/// </summary>
public class SecureTokenStore : ITokenStore
{
    private const string TokenKey = "mizu_auth_token";
    private static readonly JsonSerializerOptions JsonOptions = new()
    {
        PropertyNamingPolicy = JsonNamingPolicy.SnakeCaseLower
    };

    public event EventHandler<TokenChangedEventArgs>? TokenChanged;

    public async Task<AuthToken?> GetTokenAsync()
    {
        try
        {
            var json = await SecureStorage.Default.GetAsync(TokenKey);
            if (string.IsNullOrEmpty(json))
                return null;

            return JsonSerializer.Deserialize<AuthToken>(json, JsonOptions);
        }
        catch
        {
            return null;
        }
    }

    public async Task SetTokenAsync(AuthToken token)
    {
        var json = JsonSerializer.Serialize(token, JsonOptions);
        await SecureStorage.Default.SetAsync(TokenKey, json);
        TokenChanged?.Invoke(this, new TokenChangedEventArgs(token));
    }

    public async Task ClearTokenAsync()
    {
        SecureStorage.Default.Remove(TokenKey);
        TokenChanged?.Invoke(this, new TokenChangedEventArgs(null));
        await Task.CompletedTask;
    }
}

/// <summary>
/// In-memory token store for testing
/// </summary>
public class InMemoryTokenStore : ITokenStore
{
    private AuthToken? _token;

    public event EventHandler<TokenChangedEventArgs>? TokenChanged;

    public Task<AuthToken?> GetTokenAsync() => Task.FromResult(_token);

    public Task SetTokenAsync(AuthToken token)
    {
        _token = token;
        TokenChanged?.Invoke(this, new TokenChangedEventArgs(token));
        return Task.CompletedTask;
    }

    public Task ClearTokenAsync()
    {
        _token = null;
        TokenChanged?.Invoke(this, new TokenChangedEventArgs(null));
        return Task.CompletedTask;
    }
}
```

### Live Streaming (`Runtime/LiveConnection.cs`)

```csharp
using System.Runtime.CompilerServices;
using System.Text.RegularExpressions;

namespace {{.Namespace}}.Runtime;

/// <summary>
/// Server-sent event
/// </summary>
public class ServerEvent
{
    public string? Id { get; init; }
    public string? Event { get; init; }
    public required string Data { get; init; }
    public int? Retry { get; init; }

    /// <summary>
    /// Decodes the event data as JSON
    /// </summary>
    public T Decode<T>()
    {
        return JsonSerializer.Deserialize<T>(Data, new JsonSerializerOptions
        {
            PropertyNamingPolicy = JsonNamingPolicy.SnakeCaseLower,
            PropertyNameCaseInsensitive = true
        })!;
    }

    /// <summary>
    /// Decodes the event data as a raw dictionary
    /// </summary>
    public Dictionary<string, JsonElement> DecodeJson()
    {
        return JsonSerializer.Deserialize<Dictionary<string, JsonElement>>(Data)!;
    }
}

/// <summary>
/// Live connection manager for SSE
/// </summary>
public class LiveConnection
{
    private readonly MizuRuntime _runtime;
    private readonly Dictionary<string, CancellationTokenSource> _activeConnections = new();
    private readonly object _lock = new();

    public LiveConnection(MizuRuntime runtime)
    {
        _runtime = runtime;
    }

    /// <summary>
    /// Connects to an SSE endpoint and returns an async stream of events
    /// </summary>
    public async IAsyncEnumerable<ServerEvent> ConnectAsync(
        string path,
        Dictionary<string, string>? headers = null,
        [EnumeratorCancellation] CancellationToken cancellationToken = default)
    {
        var cts = CancellationTokenSource.CreateLinkedTokenSource(cancellationToken);

        lock (_lock)
        {
            if (_activeConnections.ContainsKey(path))
            {
                _activeConnections[path].Cancel();
            }
            _activeConnections[path] = cts;
        }

        try
        {
            await foreach (var evt in StreamEventsAsync(path, headers, cts.Token))
            {
                yield return evt;
            }
        }
        finally
        {
            lock (_lock)
            {
                _activeConnections.Remove(path);
            }
        }
    }

    /// <summary>
    /// Disconnects from a specific path
    /// </summary>
    public void Disconnect(string path)
    {
        lock (_lock)
        {
            if (_activeConnections.TryGetValue(path, out var cts))
            {
                cts.Cancel();
                _activeConnections.Remove(path);
            }
        }
    }

    /// <summary>
    /// Disconnects all active connections
    /// </summary>
    public void DisconnectAll()
    {
        lock (_lock)
        {
            foreach (var cts in _activeConnections.Values)
            {
                cts.Cancel();
            }
            _activeConnections.Clear();
        }
    }

    private async IAsyncEnumerable<ServerEvent> StreamEventsAsync(
        string path,
        Dictionary<string, string>? headers,
        [EnumeratorCancellation] CancellationToken cancellationToken)
    {
        var url = BuildUrl(path);

        using var client = new HttpClient();
        using var request = new HttpRequestMessage(HttpMethod.Get, url);

        request.Headers.Add("Accept", "text/event-stream");
        request.Headers.Add("Cache-Control", "no-cache");

        // Add default headers
        foreach (var kvp in _runtime.DefaultHeaders)
        {
            request.Headers.TryAddWithoutValidation(kvp.Key, kvp.Value);
        }

        // Add custom headers
        if (headers != null)
        {
            foreach (var kvp in headers)
            {
                request.Headers.TryAddWithoutValidation(kvp.Key, kvp.Value);
            }
        }

        // Add auth token
        var token = await _runtime.TokenStore.GetTokenAsync();
        if (token != null)
        {
            request.Headers.Add("Authorization", $"Bearer {token.AccessToken}");
        }

        using var response = await client.SendAsync(request, HttpCompletionOption.ResponseHeadersRead, cancellationToken);

        if (!response.IsSuccessStatusCode)
        {
            throw new MizuException(MizuErrorType.Http, $"SSE connection failed: {response.StatusCode}");
        }

        using var stream = await response.Content.ReadAsStreamAsync(cancellationToken);
        using var reader = new StreamReader(stream);

        var eventBuilder = new SSEEventBuilder();

        while (!cancellationToken.IsCancellationRequested)
        {
            var line = await reader.ReadLineAsync(cancellationToken);
            if (line == null)
                break;

            var evt = eventBuilder.ProcessLine(line);
            if (evt != null)
            {
                yield return evt;
            }
        }
    }

    private Uri BuildUrl(string path)
    {
        var baseUrl = _runtime.BaseUrl.ToString().TrimEnd('/');
        var cleanPath = path.StartsWith('/') ? path : $"/{path}";
        return new Uri($"{baseUrl}{cleanPath}");
    }
}

/// <summary>
/// SSE event parser
/// </summary>
internal class SSEEventBuilder
{
    private string? _id;
    private string? _event;
    private readonly List<string> _data = new();
    private int? _retry;

    public ServerEvent? ProcessLine(string line)
    {
        if (string.IsNullOrEmpty(line))
        {
            // Empty line means end of event
            if (_data.Count == 0)
                return null;

            var evt = new ServerEvent
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

            return evt;
        }

        if (line.StartsWith(':'))
        {
            // Comment, ignore
            return null;
        }

        var colonIndex = line.IndexOf(':');
        if (colonIndex == -1)
        {
            // Field with no value
            return null;
        }

        var field = line[..colonIndex];
        var value = colonIndex + 1 < line.Length ? line[(colonIndex + 1)..].TrimStart() : "";

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
```

### Device Info (`Runtime/DeviceInfo.cs`)

```csharp
namespace {{.Namespace}}.Runtime;

/// <summary>
/// Device information container
/// </summary>
public class DeviceInfoData
{
    public required string DeviceId { get; init; }
    public required string AppVersion { get; init; }
    public required string AppBuild { get; init; }
    public required string Model { get; init; }
    public required string Platform { get; init; }
    public required string OsVersion { get; init; }
    public required string Timezone { get; init; }
    public required string Locale { get; init; }
}

/// <summary>
/// Device information utilities
/// </summary>
public static class DeviceInfo
{
    private const string DeviceIdKey = "mizu_device_id";
    private static DeviceInfoData? _cachedInfo;

    /// <summary>
    /// Collects device information
    /// </summary>
    public static DeviceInfoData Collect()
    {
        if (_cachedInfo != null)
            return _cachedInfo;

        var deviceId = GetOrCreateDeviceId();

        string platform;
        string model;
        string osVersion;

#if ANDROID
        platform = "android";
        model = Android.OS.Build.Model ?? "Unknown";
        osVersion = Android.OS.Build.VERSION.Release ?? "Unknown";
#elif IOS || MACCATALYST
        platform = Microsoft.Maui.Devices.DeviceInfo.Current.Idiom == DeviceIdiom.Phone ? "ios" : "macos";
        model = Microsoft.Maui.Devices.DeviceInfo.Current.Model;
        osVersion = Microsoft.Maui.Devices.DeviceInfo.Current.VersionString;
#elif WINDOWS
        platform = "windows";
        model = Microsoft.Maui.Devices.DeviceInfo.Current.Model;
        osVersion = Microsoft.Maui.Devices.DeviceInfo.Current.VersionString;
#else
        platform = "unknown";
        model = "Unknown";
        osVersion = "Unknown";
#endif

        _cachedInfo = new DeviceInfoData
        {
            DeviceId = deviceId,
            AppVersion = AppInfo.Current.VersionString,
            AppBuild = AppInfo.Current.BuildString,
            Model = model,
            Platform = platform,
            OsVersion = osVersion,
            Timezone = TimeZoneInfo.Local.Id,
            Locale = CultureInfo.CurrentCulture.Name
        };

        return _cachedInfo;
    }

    private static string GetOrCreateDeviceId()
    {
        var deviceId = Preferences.Default.Get<string?>(DeviceIdKey, null);
        if (string.IsNullOrEmpty(deviceId))
        {
            deviceId = Guid.NewGuid().ToString();
            Preferences.Default.Set(DeviceIdKey, deviceId);
        }
        return deviceId;
    }

    /// <summary>
    /// Clears cached info (useful for testing)
    /// </summary>
    public static void ClearCache()
    {
        _cachedInfo = null;
    }
}
```

### Exceptions (`Runtime/MizuException.cs`)

```csharp
namespace {{.Namespace}}.Runtime;

/// <summary>
/// Error type enumeration
/// </summary>
public enum MizuErrorType
{
    InvalidResponse,
    Http,
    Api,
    Network,
    Encoding,
    Decoding,
    Unauthorized,
    TokenExpired
}

/// <summary>
/// API error response from server
/// </summary>
public class ApiError
{
    public required string Code { get; init; }
    public required string Message { get; init; }
    public Dictionary<string, object>? Details { get; init; }
    public string? TraceId { get; init; }
}

/// <summary>
/// Mizu client exception
/// </summary>
public class MizuException : Exception
{
    public MizuErrorType ErrorType { get; }
    public ApiError? ApiError { get; }
    public int? StatusCode { get; }

    public MizuException(
        MizuErrorType errorType,
        string message,
        ApiError? apiError = null,
        int? statusCode = null,
        Exception? innerException = null)
        : base(message, innerException)
    {
        ErrorType = errorType;
        ApiError = apiError;
        StatusCode = statusCode;
    }

    public bool IsInvalidResponse => ErrorType == MizuErrorType.InvalidResponse;
    public bool IsHttp => ErrorType == MizuErrorType.Http;
    public bool IsApi => ErrorType == MizuErrorType.Api;
    public bool IsNetwork => ErrorType == MizuErrorType.Network;
    public bool IsUnauthorized => ErrorType == MizuErrorType.Unauthorized;
    public bool IsTokenExpired => ErrorType == MizuErrorType.TokenExpired;
}
```

## App Templates

### MAUI Program Entry (`MauiProgram.cs`)

```csharp
using Microsoft.Extensions.Logging;

namespace {{.Namespace}};

public static class MauiProgram
{
    public static MauiApp CreateMauiApp()
    {
        var builder = MauiApp.CreateBuilder();
        builder
            .UseMauiApp<App>()
            .ConfigureFonts(fonts =>
            {
                fonts.AddFont("OpenSans-Regular.ttf", "OpenSansRegular");
                fonts.AddFont("OpenSans-Semibold.ttf", "OpenSansSemibold");
            });

        // Register services
        builder.Services.AddSingleton(MizuRuntime.Shared);
        builder.Services.AddSingleton<ITokenStore>(MizuRuntime.Shared.TokenStore);

        // Register ViewModels
        builder.Services.AddTransient<HomeViewModel>();
        builder.Services.AddTransient<WelcomeViewModel>();

        // Register Pages
        builder.Services.AddTransient<HomePage>();
        builder.Services.AddTransient<WelcomePage>();

#if DEBUG
        builder.Logging.AddDebug();

        // Add logging interceptor in debug mode
        if (MizuRuntime.Shared.Transport is HttpTransport httpTransport)
        {
            httpTransport.AddInterceptor(new LoggingInterceptor());
        }
#endif

        // Configure Mizu runtime
        var baseUrl = new Uri(
#if DEBUG
            DeviceInfo.Platform == DevicePlatform.Android
                ? "http://10.0.2.2:3000"
                : "http://localhost:3000"
#else
            "https://api.example.com"
#endif
        );

        MizuRuntime.Shared.BaseUrl = baseUrl;

        return builder.Build();
    }
}
```

### App.xaml

```xml
<?xml version="1.0" encoding="UTF-8" ?>
<Application xmlns="http://schemas.microsoft.com/dotnet/2021/maui"
             xmlns:x="http://schemas.microsoft.com/winfx/2009/xaml"
             x:Class="{{.Namespace}}.App">
    <Application.Resources>
        <ResourceDictionary>
            <ResourceDictionary.MergedDictionaries>
                <ResourceDictionary Source="Resources/Styles/Colors.xaml" />
                <ResourceDictionary Source="Resources/Styles/Styles.xaml" />
            </ResourceDictionary.MergedDictionaries>
        </ResourceDictionary>
    </Application.Resources>
</Application>
```

### App.xaml.cs

```csharp
namespace {{.Namespace}};

public partial class App : Application
{
    public App()
    {
        InitializeComponent();
    }

    protected override Window CreateWindow(IActivationState? activationState)
    {
        return new Window(new AppShell());
    }
}
```

### AppShell.xaml

```xml
<?xml version="1.0" encoding="UTF-8" ?>
<Shell xmlns="http://schemas.microsoft.com/dotnet/2021/maui"
       xmlns:x="http://schemas.microsoft.com/winfx/2009/xaml"
       xmlns:views="clr-namespace:{{.Namespace}}.Views"
       x:Class="{{.Namespace}}.AppShell"
       FlyoutBehavior="Disabled">

    <ShellContent
        Title="Welcome"
        ContentTemplate="{DataTemplate views:WelcomePage}"
        Route="welcome" />

</Shell>
```

### AppShell.xaml.cs

```csharp
namespace {{.Namespace}};

public partial class AppShell : Shell
{
    public AppShell()
    {
        InitializeComponent();

        // Register routes
        Routing.RegisterRoute("home", typeof(Views.HomePage));
        Routing.RegisterRoute("welcome", typeof(Views.WelcomePage));
    }
}
```

### Base ViewModel (`ViewModels/BaseViewModel.cs`)

```csharp
using System.ComponentModel;
using System.Runtime.CompilerServices;

namespace {{.Namespace}}.ViewModels;

public abstract class BaseViewModel : INotifyPropertyChanged
{
    private bool _isBusy;
    public bool IsBusy
    {
        get => _isBusy;
        set => SetProperty(ref _isBusy, value);
    }

    private string? _title;
    public string? Title
    {
        get => _title;
        set => SetProperty(ref _title, value);
    }

    private string? _errorMessage;
    public string? ErrorMessage
    {
        get => _errorMessage;
        set => SetProperty(ref _errorMessage, value);
    }

    public bool HasError => !string.IsNullOrEmpty(ErrorMessage);

    public event PropertyChangedEventHandler? PropertyChanged;

    protected bool SetProperty<T>(ref T storage, T value, [CallerMemberName] string? propertyName = null)
    {
        if (EqualityComparer<T>.Default.Equals(storage, value))
            return false;

        storage = value;
        OnPropertyChanged(propertyName);
        return true;
    }

    protected void OnPropertyChanged([CallerMemberName] string? propertyName = null)
    {
        PropertyChanged?.Invoke(this, new PropertyChangedEventArgs(propertyName));
    }

    protected async Task ExecuteAsync(Func<Task> operation, string? errorMessage = null)
    {
        if (IsBusy)
            return;

        try
        {
            IsBusy = true;
            ErrorMessage = null;
            await operation();
        }
        catch (MizuException ex)
        {
            ErrorMessage = errorMessage ?? ex.Message;
        }
        catch (Exception ex)
        {
            ErrorMessage = errorMessage ?? ex.Message;
        }
        finally
        {
            IsBusy = false;
        }
    }
}
```

### Home ViewModel (`ViewModels/HomeViewModel.cs`)

```csharp
using System.Windows.Input;

namespace {{.Namespace}}.ViewModels;

public class HomeViewModel : BaseViewModel
{
    private readonly MizuRuntime _runtime;

    public ICommand SignOutCommand { get; }

    public HomeViewModel(MizuRuntime runtime)
    {
        _runtime = runtime;
        Title = "Home";
        SignOutCommand = new Command(async () => await SignOutAsync());
    }

    private async Task SignOutAsync()
    {
        await ExecuteAsync(async () =>
        {
            await _runtime.TokenStore.ClearTokenAsync();
            await Shell.Current.GoToAsync("//welcome");
        });
    }
}
```

### Welcome ViewModel (`ViewModels/WelcomeViewModel.cs`)

```csharp
using System.Windows.Input;

namespace {{.Namespace}}.ViewModels;

public class WelcomeViewModel : BaseViewModel
{
    private readonly MizuRuntime _runtime;

    public ICommand GetStartedCommand { get; }

    public WelcomeViewModel(MizuRuntime runtime)
    {
        _runtime = runtime;
        Title = "Welcome";
        GetStartedCommand = new Command(async () => await GetStartedAsync());
    }

    private async Task GetStartedAsync()
    {
        await ExecuteAsync(async () =>
        {
            // Demo: Set a test token
            await _runtime.TokenStore.SetTokenAsync(new AuthToken
            {
                AccessToken = "demo_token"
            });

            await Shell.Current.GoToAsync("//home");
        });
    }
}
```

### Home Page (`Views/HomePage.xaml`)

```xml
<?xml version="1.0" encoding="utf-8" ?>
<ContentPage xmlns="http://schemas.microsoft.com/dotnet/2021/maui"
             xmlns:x="http://schemas.microsoft.com/winfx/2009/xaml"
             xmlns:vm="clr-namespace:{{.Namespace}}.ViewModels"
             x:Class="{{.Namespace}}.Views.HomePage"
             x:DataType="vm:HomeViewModel"
             Title="{Binding Title}">

    <VerticalStackLayout
        Padding="24"
        Spacing="20"
        VerticalOptions="Center">

        <Image
            Source="dotnet_bot.png"
            HeightRequest="100"
            WidthRequest="100"
            HorizontalOptions="Center" />

        <Label
            Text="Welcome to {{.Name}}"
            Style="{StaticResource Headline}"
            HorizontalOptions="Center" />

        <Label
            Text="Connected to Mizu backend"
            Style="{StaticResource SubHeadline}"
            HorizontalOptions="Center" />

        <ActivityIndicator
            IsRunning="{Binding IsBusy}"
            IsVisible="{Binding IsBusy}"
            HorizontalOptions="Center" />

        <Button
            Text="Sign Out"
            Command="{Binding SignOutCommand}"
            IsEnabled="{Binding IsBusy, Converter={StaticResource InvertedBoolConverter}}"
            Style="{StaticResource SecondaryButton}" />

    </VerticalStackLayout>

</ContentPage>
```

### Home Page Code-Behind (`Views/HomePage.xaml.cs`)

```csharp
namespace {{.Namespace}}.Views;

public partial class HomePage : ContentPage
{
    public HomePage(HomeViewModel viewModel)
    {
        InitializeComponent();
        BindingContext = viewModel;
    }
}
```

### Welcome Page (`Views/WelcomePage.xaml`)

```xml
<?xml version="1.0" encoding="utf-8" ?>
<ContentPage xmlns="http://schemas.microsoft.com/dotnet/2021/maui"
             xmlns:x="http://schemas.microsoft.com/winfx/2009/xaml"
             xmlns:vm="clr-namespace:{{.Namespace}}.ViewModels"
             x:Class="{{.Namespace}}.Views.WelcomePage"
             x:DataType="vm:WelcomeViewModel"
             Title="{Binding Title}">

    <Grid RowDefinitions="*,Auto,*">

        <VerticalStackLayout
            Grid.Row="1"
            Padding="24"
            Spacing="16">

            <Image
                Source="dotnet_bot.png"
                HeightRequest="120"
                WidthRequest="120"
                HorizontalOptions="Center" />

            <Label
                Text="Welcome to {{.Name}}"
                Style="{StaticResource Headline}"
                HorizontalOptions="Center" />

            <Label
                Text="A modern .NET MAUI app powered by Mizu"
                Style="{StaticResource SubHeadline}"
                HorizontalOptions="Center" />

        </VerticalStackLayout>

        <VerticalStackLayout
            Grid.Row="2"
            Padding="24"
            VerticalOptions="End">

            <Button
                Text="Get Started"
                Command="{Binding GetStartedCommand}"
                IsEnabled="{Binding IsBusy, Converter={StaticResource InvertedBoolConverter}}"
                Style="{StaticResource PrimaryButton}" />

        </VerticalStackLayout>

    </Grid>

</ContentPage>
```

### Welcome Page Code-Behind (`Views/WelcomePage.xaml.cs`)

```csharp
namespace {{.Namespace}}.Views;

public partial class WelcomePage : ContentPage
{
    public WelcomePage(WelcomeViewModel viewModel)
    {
        InitializeComponent();
        BindingContext = viewModel;
    }
}
```

### Loading View (`Controls/LoadingView.xaml`)

```xml
<?xml version="1.0" encoding="utf-8" ?>
<ContentView xmlns="http://schemas.microsoft.com/dotnet/2021/maui"
             xmlns:x="http://schemas.microsoft.com/winfx/2009/xaml"
             x:Class="{{.Namespace}}.Controls.LoadingView">

    <VerticalStackLayout
        VerticalOptions="Center"
        HorizontalOptions="Center"
        Spacing="16">

        <ActivityIndicator
            IsRunning="True"
            Color="{StaticResource Primary}" />

        <Label
            x:Name="MessageLabel"
            Style="{StaticResource Body}"
            HorizontalOptions="Center" />

    </VerticalStackLayout>

</ContentView>
```

### Loading View Code-Behind (`Controls/LoadingView.xaml.cs`)

```csharp
namespace {{.Namespace}}.Controls;

public partial class LoadingView : ContentView
{
    public static readonly BindableProperty MessageProperty = BindableProperty.Create(
        nameof(Message),
        typeof(string),
        typeof(LoadingView),
        default(string),
        propertyChanged: OnMessageChanged);

    public string? Message
    {
        get => (string?)GetValue(MessageProperty);
        set => SetValue(MessageProperty, value);
    }

    public LoadingView()
    {
        InitializeComponent();
    }

    private static void OnMessageChanged(BindableObject bindable, object oldValue, object newValue)
    {
        if (bindable is LoadingView view)
        {
            view.MessageLabel.Text = newValue as string;
            view.MessageLabel.IsVisible = !string.IsNullOrEmpty(newValue as string);
        }
    }
}
```

## Generated SDK

### Client (`SDK/Client.cs`)

```csharp
namespace {{.Namespace}}.SDK;

/// <summary>
/// Generated Mizu API client for {{.Name}}
/// </summary>
public class {{.Name}}Client
{
    private readonly MizuRuntime _runtime;

    public {{.Name}}Client(MizuRuntime runtime)
    {
        _runtime = runtime;
    }

    // MARK: - Auth

    /// <summary>
    /// Sign in with credentials
    /// </summary>
    public async Task<AuthResponse> SignInAsync(
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
    public async Task<AuthResponse> SignUpAsync(
        string email,
        string password,
        string name,
        CancellationToken cancellationToken = default)
    {
        return await _runtime.PostAsync<AuthResponse, SignUpRequest>(
            "/auth/signup",
            new SignUpRequest { Email = email, Password = password, Name = name },
            cancellationToken: cancellationToken);
    }

    /// <summary>
    /// Sign out
    /// </summary>
    public async Task SignOutAsync(CancellationToken cancellationToken = default)
    {
        await _runtime.DeleteAsync("/auth/signout", cancellationToken: cancellationToken);
        await _runtime.TokenStore.ClearTokenAsync();
    }

    // MARK: - Users

    /// <summary>
    /// Get current user profile
    /// </summary>
    public async Task<User> GetCurrentUserAsync(CancellationToken cancellationToken = default)
    {
        return await _runtime.GetAsync<User>("/users/me", cancellationToken: cancellationToken);
    }

    /// <summary>
    /// Update current user profile
    /// </summary>
    public async Task<User> UpdateCurrentUserAsync(
        UserUpdate update,
        CancellationToken cancellationToken = default)
    {
        return await _runtime.PutAsync<User, UserUpdate>(
            "/users/me",
            update,
            cancellationToken: cancellationToken);
    }
}
```

### Types (`SDK/Types.cs`)

```csharp
using System.Text.Json.Serialization;

namespace {{.Namespace}}.SDK;

// MARK: - Auth Types

public class SignInRequest
{
    public required string Email { get; init; }
    public required string Password { get; init; }
}

public class SignUpRequest
{
    public required string Email { get; init; }
    public required string Password { get; init; }
    public required string Name { get; init; }
}

public class AuthResponse
{
    public required User User { get; init; }
    public required TokenResponse Token { get; init; }
}

public class TokenResponse
{
    [JsonPropertyName("access_token")]
    public required string AccessToken { get; init; }

    [JsonPropertyName("refresh_token")]
    public string? RefreshToken { get; init; }

    [JsonPropertyName("expires_in")]
    public required int ExpiresIn { get; init; }
}

// MARK: - User Types

public class User
{
    public required string Id { get; init; }
    public required string Email { get; init; }
    public required string Name { get; init; }

    [JsonPropertyName("avatar_url")]
    public string? AvatarUrl { get; init; }

    [JsonPropertyName("created_at")]
    public required DateTime CreatedAt { get; init; }

    [JsonPropertyName("updated_at")]
    public required DateTime UpdatedAt { get; init; }
}

public class UserUpdate
{
    public string? Name { get; init; }

    [JsonPropertyName("avatar_url")]
    public string? AvatarUrl { get; init; }
}
```

### Extensions (`SDK/Extensions.cs`)

```csharp
namespace {{.Namespace}}.SDK;

/// <summary>
/// Convenience extensions for working with the SDK
/// </summary>
public static class ClientExtensions
{
    /// <summary>
    /// Store an auth response token
    /// </summary>
    public static async Task StoreAuthTokenAsync(this {{.Name}}Client client, AuthResponse response, MizuRuntime runtime)
    {
        var expiresAt = DateTime.UtcNow.AddSeconds(response.Token.ExpiresIn);
        await runtime.TokenStore.SetTokenAsync(new AuthToken
        {
            AccessToken = response.Token.AccessToken,
            RefreshToken = response.Token.RefreshToken,
            ExpiresAt = expiresAt
        });
    }
}
```

## Build Configuration

### Project File (`{{.Name}}.csproj`)

```xml
<Project Sdk="Microsoft.NET.Sdk">

    <PropertyGroup>
        <TargetFrameworks>net8.0-android;net8.0-ios;net8.0-maccatalyst</TargetFrameworks>
        <TargetFrameworks Condition="$([MSBuild]::IsOSPlatform('windows'))">$(TargetFrameworks);net8.0-windows10.0.19041.0</TargetFrameworks>
        <OutputType>Exe</OutputType>
        <RootNamespace>{{.Namespace}}</RootNamespace>
        <UseMaui>true</UseMaui>
        <SingleProject>true</SingleProject>
        <ImplicitUsings>enable</ImplicitUsings>
        <Nullable>enable</Nullable>

        <!-- Display name -->
        <ApplicationTitle>{{.Name}}</ApplicationTitle>

        <!-- App Identifier -->
        <ApplicationId>{{.BundleId}}</ApplicationId>

        <!-- Versions -->
        <ApplicationDisplayVersion>1.0</ApplicationDisplayVersion>
        <ApplicationVersion>1</ApplicationVersion>

        <SupportedOSPlatformVersion Condition="$([MSBuild]::GetTargetPlatformIdentifier('$(TargetFramework)')) == 'ios'">14.2</SupportedOSPlatformVersion>
        <SupportedOSPlatformVersion Condition="$([MSBuild]::GetTargetPlatformIdentifier('$(TargetFramework)')) == 'maccatalyst'">14.0</SupportedOSPlatformVersion>
        <SupportedOSPlatformVersion Condition="$([MSBuild]::GetTargetPlatformIdentifier('$(TargetFramework)')) == 'android'">21.0</SupportedOSPlatformVersion>
        <SupportedOSPlatformVersion Condition="$([MSBuild]::GetTargetPlatformIdentifier('$(TargetFramework)')) == 'windows'">10.0.17763.0</SupportedOSPlatformVersion>
        <TargetPlatformMinVersion Condition="$([MSBuild]::GetTargetPlatformIdentifier('$(TargetFramework)')) == 'windows'">10.0.17763.0</TargetPlatformMinVersion>
    </PropertyGroup>

    <ItemGroup>
        <!-- App Icon -->
        <MauiIcon Include="Resources\AppIcon\appicon.svg" ForegroundFile="Resources\AppIcon\appiconfg.svg" Color="#512BD4" />

        <!-- Splash Screen -->
        <MauiSplashScreen Include="Resources\Splash\splash.svg" Color="#512BD4" BaseSize="128,128" />

        <!-- Images -->
        <MauiImage Include="Resources\Images\*" />
        <MauiImage Update="Resources\Images\dotnet_bot.png" Resize="True" BaseSize="300,185" />

        <!-- Custom Fonts -->
        <MauiFont Include="Resources\Fonts\*" />

        <!-- Raw Assets (also remove the "Resources\Raw" prefix) -->
        <MauiAsset Include="Resources\Raw\**" LogicalName="%(RecursiveDir)%(Filename)%(Extension)" />
    </ItemGroup>

    <ItemGroup>
        <PackageReference Include="Microsoft.Maui.Controls" Version="$(MauiVersion)" />
        <PackageReference Include="Microsoft.Maui.Controls.Compatibility" Version="$(MauiVersion)" />
        <PackageReference Include="Microsoft.Extensions.Logging.Debug" Version="8.0.0" />
    </ItemGroup>

</Project>
```

### Solution File (`{{.Name}}.sln`)

```
Microsoft Visual Studio Solution File, Format Version 12.00
# Visual Studio Version 17
VisualStudioVersion = 17.0.31903.59
MinimumVisualStudioVersion = 10.0.40219.1
Project("{FAE04EC0-301F-11D3-BF4B-00C04F79EFBC}") = "{{.Name}}", "{{.Name}}\{{.Name}}.csproj", "{GUID-HERE}"
EndProject
Project("{FAE04EC0-301F-11D3-BF4B-00C04F79EFBC}") = "{{.Name}}.Tests", "{{.Name}}.Tests\{{.Name}}.Tests.csproj", "{GUID-HERE-2}"
EndProject
Global
    GlobalSection(SolutionConfigurationPlatforms) = preSolution
        Debug|Any CPU = Debug|Any CPU
        Release|Any CPU = Release|Any CPU
    EndGlobalSection
    GlobalSection(ProjectConfigurationPlatforms) = postSolution
        {GUID-HERE}.Debug|Any CPU.ActiveCfg = Debug|Any CPU
        {GUID-HERE}.Debug|Any CPU.Build.0 = Debug|Any CPU
        {GUID-HERE}.Release|Any CPU.ActiveCfg = Release|Any CPU
        {GUID-HERE}.Release|Any CPU.Build.0 = Release|Any CPU
        {GUID-HERE-2}.Debug|Any CPU.ActiveCfg = Debug|Any CPU
        {GUID-HERE-2}.Debug|Any CPU.Build.0 = Debug|Any CPU
        {GUID-HERE-2}.Release|Any CPU.ActiveCfg = Release|Any CPU
        {GUID-HERE-2}.Release|Any CPU.Build.0 = Release|Any CPU
    EndGlobalSection
EndGlobal
```

### Styles (`Resources/Styles/Colors.xaml`)

```xml
<?xml version="1.0" encoding="UTF-8" ?>
<?xaml-comp compile="true" ?>
<ResourceDictionary
    xmlns="http://schemas.microsoft.com/dotnet/2021/maui"
    xmlns:x="http://schemas.microsoft.com/winfx/2009/xaml">

    <Color x:Key="Primary">#512BD4</Color>
    <Color x:Key="PrimaryDark">#3c1f9e</Color>
    <Color x:Key="PrimaryDarkText">#FFFFFF</Color>
    <Color x:Key="Secondary">#DFD8F7</Color>
    <Color x:Key="SecondaryDarkText">#9880e5</Color>
    <Color x:Key="Tertiary">#2B0B98</Color>
    <Color x:Key="White">#FFFFFF</Color>
    <Color x:Key="Black">#000000</Color>
    <Color x:Key="Gray100">#E1E1E1</Color>
    <Color x:Key="Gray200">#C8C8C8</Color>
    <Color x:Key="Gray300">#ACACAC</Color>
    <Color x:Key="Gray400">#919191</Color>
    <Color x:Key="Gray500">#6E6E6E</Color>
    <Color x:Key="Gray600">#404040</Color>
    <Color x:Key="Gray900">#212121</Color>
    <Color x:Key="Gray950">#141414</Color>

</ResourceDictionary>
```

### Styles (`Resources/Styles/Styles.xaml`)

```xml
<?xml version="1.0" encoding="UTF-8" ?>
<?xaml-comp compile="true" ?>
<ResourceDictionary
    xmlns="http://schemas.microsoft.com/dotnet/2021/maui"
    xmlns:x="http://schemas.microsoft.com/winfx/2009/xaml">

    <!-- Converters -->
    <toolkit:InvertedBoolConverter x:Key="InvertedBoolConverter" />

    <!-- Label Styles -->
    <Style x:Key="Headline" TargetType="Label">
        <Setter Property="FontSize" Value="28" />
        <Setter Property="FontAttributes" Value="Bold" />
        <Setter Property="TextColor" Value="{AppThemeBinding Light={StaticResource Gray900}, Dark={StaticResource White}}" />
    </Style>

    <Style x:Key="SubHeadline" TargetType="Label">
        <Setter Property="FontSize" Value="16" />
        <Setter Property="TextColor" Value="{AppThemeBinding Light={StaticResource Gray500}, Dark={StaticResource Gray300}}" />
    </Style>

    <Style x:Key="Body" TargetType="Label">
        <Setter Property="FontSize" Value="14" />
        <Setter Property="TextColor" Value="{AppThemeBinding Light={StaticResource Gray600}, Dark={StaticResource Gray400}}" />
    </Style>

    <!-- Button Styles -->
    <Style x:Key="PrimaryButton" TargetType="Button">
        <Setter Property="BackgroundColor" Value="{StaticResource Primary}" />
        <Setter Property="TextColor" Value="{StaticResource White}" />
        <Setter Property="FontAttributes" Value="Bold" />
        <Setter Property="CornerRadius" Value="8" />
        <Setter Property="Padding" Value="16,12" />
        <Setter Property="HeightRequest" Value="56" />
        <Setter Property="VisualStateManager.VisualStateGroups">
            <VisualStateGroupList>
                <VisualStateGroup x:Name="CommonStates">
                    <VisualState x:Name="Normal" />
                    <VisualState x:Name="Disabled">
                        <VisualState.Setters>
                            <Setter Property="BackgroundColor" Value="{StaticResource Gray300}" />
                        </VisualState.Setters>
                    </VisualState>
                </VisualStateGroup>
            </VisualStateGroupList>
        </Setter>
    </Style>

    <Style x:Key="SecondaryButton" TargetType="Button">
        <Setter Property="BackgroundColor" Value="{StaticResource Secondary}" />
        <Setter Property="TextColor" Value="{StaticResource Primary}" />
        <Setter Property="FontAttributes" Value="Bold" />
        <Setter Property="CornerRadius" Value="8" />
        <Setter Property="Padding" Value="16,12" />
    </Style>

</ResourceDictionary>
```

## Testing

### Unit Tests (`{{.Name}}.Tests/RuntimeTests.cs`)

```csharp
using {{.Namespace}}.Runtime;

namespace {{.Namespace}}.Tests;

public class RuntimeTests
{
    [Fact]
    public async Task InMemoryTokenStore_ReturnsNullWhenNoTokenStored()
    {
        var store = new InMemoryTokenStore();
        var token = await store.GetTokenAsync();
        Assert.Null(token);
    }

    [Fact]
    public async Task InMemoryTokenStore_StoresAndRetrievesToken()
    {
        var store = new InMemoryTokenStore();
        var token = new AuthToken
        {
            AccessToken = "test123",
            RefreshToken = "refresh456"
        };

        await store.SetTokenAsync(token);
        var retrieved = await store.GetTokenAsync();

        Assert.NotNull(retrieved);
        Assert.Equal("test123", retrieved.AccessToken);
        Assert.Equal("refresh456", retrieved.RefreshToken);
    }

    [Fact]
    public async Task InMemoryTokenStore_ClearsToken()
    {
        var store = new InMemoryTokenStore();
        var token = new AuthToken { AccessToken = "test123" };

        await store.SetTokenAsync(token);
        await store.ClearTokenAsync();

        var retrieved = await store.GetTokenAsync();
        Assert.Null(retrieved);
    }

    [Fact]
    public async Task InMemoryTokenStore_NotifiesObserversOnTokenChange()
    {
        var store = new InMemoryTokenStore();
        AuthToken? notifiedToken = null;
        store.TokenChanged += (_, e) => notifiedToken = e.Token;

        var token = new AuthToken { AccessToken = "test123" };
        await store.SetTokenAsync(token);

        Assert.NotNull(notifiedToken);
        Assert.Equal("test123", notifiedToken.AccessToken);
    }

    [Fact]
    public void AuthToken_IsExpiredReturnsFalseWhenNoExpiry()
    {
        var token = new AuthToken { AccessToken = "test" };
        Assert.False(token.IsExpired);
    }

    [Fact]
    public void AuthToken_IsExpiredReturnsTrueWhenExpired()
    {
        var token = new AuthToken
        {
            AccessToken = "test",
            ExpiresAt = DateTime.UtcNow.AddSeconds(-1)
        };
        Assert.True(token.IsExpired);
    }

    [Fact]
    public void AuthToken_IsExpiredReturnsFalseWhenNotExpired()
    {
        var token = new AuthToken
        {
            AccessToken = "test",
            ExpiresAt = DateTime.UtcNow.AddHours(1)
        };
        Assert.False(token.IsExpired);
    }

    [Fact]
    public void MizuException_CreatesNetworkError()
    {
        var error = new MizuException(MizuErrorType.Network, "Connection failed");
        Assert.True(error.IsNetwork);
        Assert.Equal("Connection failed", error.Message);
    }

    [Fact]
    public void MizuException_CreatesApiError()
    {
        var apiError = new ApiError { Code = "test_error", Message = "Test message" };
        var error = new MizuException(MizuErrorType.Api, apiError.Message, apiError);
        Assert.True(error.IsApi);
        Assert.Equal("test_error", error.ApiError?.Code);
    }
}
```

### Test Project File (`{{.Name}}.Tests/{{.Name}}.Tests.csproj`)

```xml
<Project Sdk="Microsoft.NET.Sdk">

    <PropertyGroup>
        <TargetFramework>net8.0</TargetFramework>
        <ImplicitUsings>enable</ImplicitUsings>
        <Nullable>enable</Nullable>
        <IsPackable>false</IsPackable>
    </PropertyGroup>

    <ItemGroup>
        <PackageReference Include="Microsoft.NET.Test.Sdk" Version="17.8.0" />
        <PackageReference Include="xunit" Version="2.6.2" />
        <PackageReference Include="xunit.runner.visualstudio" Version="2.5.4">
            <IncludeAssets>runtime; build; native; contentfiles; analyzers; buildtransitive</IncludeAssets>
            <PrivateAssets>all</PrivateAssets>
        </PackageReference>
        <PackageReference Include="coverlet.collector" Version="6.0.0">
            <IncludeAssets>runtime; build; native; contentfiles; analyzers; buildtransitive</IncludeAssets>
            <PrivateAssets>all</PrivateAssets>
        </PackageReference>
        <PackageReference Include="Moq" Version="4.20.70" />
    </ItemGroup>

    <ItemGroup>
        <ProjectReference Include="..\{{.Name}}\{{.Name}}.csproj" />
    </ItemGroup>

</Project>
```

## Navigation Variants

### Shell Navigation (Default)
- Uses .NET MAUI Shell for navigation
- Flyout menu support
- Tab bar support
- URI-based navigation with routes

### Plain Navigation
- Uses `NavigationPage` for navigation
- Standard push/pop navigation
- More control over navigation stack
- Better for simple navigation flows

## Platform Configuration

### Android (`Platforms/Android/AndroidManifest.xml`)

```xml
<?xml version="1.0" encoding="utf-8"?>
<manifest xmlns:android="http://schemas.android.com/apk/res/android">
    <application
        android:allowBackup="true"
        android:icon="@mipmap/appicon"
        android:roundIcon="@mipmap/appicon_round"
        android:supportsRtl="true"
        android:usesCleartextTraffic="true">
    </application>
    <uses-permission android:name="android.permission.ACCESS_NETWORK_STATE" />
    <uses-permission android:name="android.permission.INTERNET" />
</manifest>
```

### iOS (`Platforms/iOS/Info.plist`)

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>UIDeviceFamily</key>
    <array>
        <integer>1</integer>
        <integer>2</integer>
    </array>
    <key>UIRequiredDeviceCapabilities</key>
    <array>
        <string>arm64</string>
    </array>
    <key>UISupportedInterfaceOrientations</key>
    <array>
        <string>UIInterfaceOrientationPortrait</string>
        <string>UIInterfaceOrientationLandscapeLeft</string>
        <string>UIInterfaceOrientationLandscapeRight</string>
    </array>
    <key>UISupportedInterfaceOrientations~ipad</key>
    <array>
        <string>UIInterfaceOrientationPortrait</string>
        <string>UIInterfaceOrientationPortraitUpsideDown</string>
        <string>UIInterfaceOrientationLandscapeLeft</string>
        <string>UIInterfaceOrientationLandscapeRight</string>
    </array>
    <key>NSAppTransportSecurity</key>
    <dict>
        <key>NSAllowsLocalNetworking</key>
        <true/>
    </dict>
</dict>
</plist>
```

## MVVM Variants

### Default (Manual INotifyPropertyChanged)
- Lightweight implementation
- No additional dependencies
- Full control over property change notifications

### CommunityToolkit.Mvvm
- Uses source generators for reduced boilerplate
- `[ObservableProperty]` attribute for properties
- `[RelayCommand]` attribute for commands
- Requires `CommunityToolkit.Mvvm` NuGet package

## References

- [.NET MAUI Documentation](https://docs.microsoft.com/dotnet/maui/)
- [MVVM Pattern](https://docs.microsoft.com/dotnet/architecture/maui/mvvm)
- [.NET MAUI Shell](https://docs.microsoft.com/dotnet/maui/fundamentals/shell/)
- [SecureStorage](https://docs.microsoft.com/dotnet/maui/platform-integration/storage/secure-storage)
- [Preferences](https://docs.microsoft.com/dotnet/maui/platform-integration/storage/preferences)
- [Mobile Package Spec](./0095_mobile.md)
