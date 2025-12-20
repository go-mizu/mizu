# Flutter Template Specification

**Status:** Draft
**Author:** Mizu Team
**Created:** 2025-12-20
**Template:** `mobile:flutter`

## Overview

The `mobile:flutter` template generates a production-ready Flutter application with full Mizu backend integration. It follows modern Flutter development practices with Material 3, Riverpod state management, and clean architecture patterns that work across iOS, Android, web, and desktop platforms.

## Template Invocation

```bash
# Default: Material 3
mizu new ./MyApp --template mobile:flutter

# With Cupertino variant (iOS-style)
mizu new ./MyApp --template mobile:flutter --var ui=cupertino

# Custom organization
mizu new ./MyApp --template mobile:flutter --var org=com.company

# Specific platforms only
mizu new ./MyApp --template mobile:flutter --var platforms=ios,android
```

## Generated Project Structure

```
{{.Name}}/
├── lib/
│   ├── main.dart                      # App entry point
│   ├── app.dart                       # MaterialApp/CupertinoApp configuration
│   ├── config/
│   │   └── config.dart                # App configuration
│   ├── runtime/
│   │   ├── mizu_runtime.dart          # Core runtime
│   │   ├── transport.dart             # HTTP transport layer
│   │   ├── token_store.dart           # Secure token storage
│   │   ├── live.dart                  # SSE streaming
│   │   ├── device_info.dart           # Device information
│   │   └── exceptions.dart            # Error types
│   ├── sdk/
│   │   ├── client.dart                # Generated Mizu client
│   │   ├── types.dart                 # Generated types
│   │   └── extensions.dart            # Convenience extensions
│   ├── models/
│   │   └── app_state.dart             # Application state
│   ├── screens/
│   │   ├── home_screen.dart
│   │   └── welcome_screen.dart
│   ├── widgets/
│   │   ├── loading_view.dart
│   │   └── error_view.dart
│   └── providers/
│       └── providers.dart             # Riverpod providers
├── test/
│   ├── runtime_test.dart
│   ├── sdk_test.dart
│   └── widget_test.dart
├── integration_test/
│   └── app_test.dart
├── android/                           # Android-specific
├── ios/                               # iOS-specific
├── web/                               # Web-specific (optional)
├── pubspec.yaml
├── analysis_options.yaml
├── .gitignore
└── README.md
```

## MizuMobileRuntime

### Core Runtime (`lib/runtime/mizu_runtime.dart`)

```dart
import 'dart:async';
import 'dart:convert';
import 'package:flutter/foundation.dart';
import 'transport.dart';
import 'token_store.dart';
import 'live.dart';
import 'device_info.dart';
import 'exceptions.dart';

/// MizuRuntime is the core client for communicating with a Mizu backend.
class MizuRuntime extends ChangeNotifier {
  /// Shared singleton instance
  static final MizuRuntime shared = MizuRuntime._internal();

  /// Base URL for all API requests
  String baseURL;

  /// HTTP transport layer
  final Transport transport;

  /// Secure token storage
  final TokenStore tokenStore;

  /// Live connection manager
  late final LiveConnection live;

  /// Request timeout duration
  Duration timeout;

  /// Default headers added to all requests
  final Map<String, String> defaultHeaders = {};

  /// Current authentication state
  bool _isAuthenticated = false;
  bool get isAuthenticated => _isAuthenticated;

  /// Private constructor for singleton
  MizuRuntime._internal({
    this.baseURL = 'http://localhost:3000',
    Transport? transport,
    TokenStore? tokenStore,
    this.timeout = const Duration(seconds: 30),
  })  : transport = transport ?? HttpTransport(),
        tokenStore = tokenStore ?? SecureTokenStore() {
    live = LiveConnection(this);
    _initAuthState();
  }

  /// Factory constructor
  factory MizuRuntime({
    String baseURL = 'http://localhost:3000',
    Transport? transport,
    TokenStore? tokenStore,
    Duration timeout = const Duration(seconds: 30),
  }) {
    shared.baseURL = baseURL;
    shared.timeout = timeout;
    return shared;
  }

  /// Initialize with configuration
  static Future<MizuRuntime> initialize({
    required String baseURL,
    Duration timeout = const Duration(seconds: 30),
  }) async {
    shared.baseURL = baseURL;
    shared.timeout = timeout;
    await shared._initAuthState();
    return shared;
  }

  Future<void> _initAuthState() async {
    final token = await tokenStore.getToken();
    _isAuthenticated = token != null;
    notifyListeners();

    tokenStore.onTokenChange((token) {
      _isAuthenticated = token != null;
      notifyListeners();
    });
  }

  // MARK: - HTTP Methods

  /// Performs a GET request
  Future<T> get<T>(
    String path, {
    Map<String, String>? query,
    Map<String, String>? headers,
    T Function(Map<String, dynamic>)? fromJson,
  }) async {
    return _request<T, void>(
      method: 'GET',
      path: path,
      query: query,
      headers: headers,
      fromJson: fromJson,
    );
  }

  /// Performs a POST request
  Future<T> post<T, B>(
    String path, {
    B? body,
    Map<String, String>? headers,
    T Function(Map<String, dynamic>)? fromJson,
    Map<String, dynamic> Function(B)? toJson,
  }) async {
    return _request<T, B>(
      method: 'POST',
      path: path,
      body: body,
      headers: headers,
      fromJson: fromJson,
      toJson: toJson,
    );
  }

  /// Performs a PUT request
  Future<T> put<T, B>(
    String path, {
    B? body,
    Map<String, String>? headers,
    T Function(Map<String, dynamic>)? fromJson,
    Map<String, dynamic> Function(B)? toJson,
  }) async {
    return _request<T, B>(
      method: 'PUT',
      path: path,
      body: body,
      headers: headers,
      fromJson: fromJson,
      toJson: toJson,
    );
  }

  /// Performs a DELETE request
  Future<T> delete<T>(
    String path, {
    Map<String, String>? headers,
    T Function(Map<String, dynamic>)? fromJson,
  }) async {
    return _request<T, void>(
      method: 'DELETE',
      path: path,
      headers: headers,
      fromJson: fromJson,
    );
  }

  /// Performs a PATCH request
  Future<T> patch<T, B>(
    String path, {
    B? body,
    Map<String, String>? headers,
    T Function(Map<String, dynamic>)? fromJson,
    Map<String, dynamic> Function(B)? toJson,
  }) async {
    return _request<T, B>(
      method: 'PATCH',
      path: path,
      body: body,
      headers: headers,
      fromJson: fromJson,
      toJson: toJson,
    );
  }

  // MARK: - Streaming

  /// Opens a streaming connection for SSE
  Stream<ServerEvent> stream(
    String path, {
    Map<String, String>? headers,
  }) {
    return live.connect(path: path, headers: headers);
  }

  // MARK: - Private

  Future<T> _request<T, B>({
    required String method,
    required String path,
    Map<String, String>? query,
    B? body,
    Map<String, String>? headers,
    T Function(Map<String, dynamic>)? fromJson,
    Map<String, dynamic> Function(B)? toJson,
  }) async {
    // Build URL
    var url = _buildUrl(path, query);

    // Build headers
    final allHeaders = await _buildHeaders(headers);

    // Encode body
    String? bodyJson;
    if (body != null && toJson != null) {
      bodyJson = jsonEncode(toJson(body));
      allHeaders['Content-Type'] = 'application/json';
    } else if (body != null && body is Map) {
      bodyJson = jsonEncode(body);
      allHeaders['Content-Type'] = 'application/json';
    }

    // Execute request
    final response = await transport.execute(
      TransportRequest(
        url: url,
        method: method,
        headers: allHeaders,
        body: bodyJson,
        timeout: timeout,
      ),
    );

    // Handle errors
    if (response.statusCode >= 400) {
      throw _parseError(response);
    }

    // Handle empty response
    if (T == void || response.body.isEmpty) {
      return null as T;
    }

    // Decode response
    final json = jsonDecode(response.body);
    if (fromJson != null) {
      return fromJson(json as Map<String, dynamic>);
    }
    return json as T;
  }

  String _buildUrl(String path, Map<String, String>? query) {
    final base = baseURL.endsWith('/') ? baseURL.substring(0, baseURL.length - 1) : baseURL;
    final cleanPath = path.startsWith('/') ? path : '/$path';
    var url = '$base$cleanPath';

    if (query != null && query.isNotEmpty) {
      final queryString = query.entries
          .map((e) => '${Uri.encodeComponent(e.key)}=${Uri.encodeComponent(e.value)}')
          .join('&');
      url = '$url?$queryString';
    }

    return url;
  }

  Future<Map<String, String>> _buildHeaders(Map<String, String>? custom) async {
    final headers = <String, String>{};
    headers.addAll(defaultHeaders);
    headers.addAll(await _mobileHeaders());
    if (custom != null) {
      headers.addAll(custom);
    }

    // Add auth token
    final token = await tokenStore.getToken();
    if (token != null) {
      headers['Authorization'] = 'Bearer ${token.accessToken}';
    }

    return headers;
  }

  Future<Map<String, String>> _mobileHeaders() async {
    final info = await DeviceInfo.collect();
    return {
      'X-Device-ID': info.deviceId,
      'X-App-Version': info.appVersion,
      'X-App-Build': info.appBuild,
      'X-Device-Model': info.model,
      'X-Platform': info.platform,
      'X-OS-Version': info.osVersion,
      'X-Timezone': info.timezone,
      'X-Locale': info.locale,
    };
  }

  MizuError _parseError(TransportResponse response) {
    try {
      final json = jsonDecode(response.body) as Map<String, dynamic>;
      return MizuError.api(APIError.fromJson(json));
    } catch (_) {
      return MizuError.http(response.statusCode, response.body);
    }
  }
}
```

### Transport Layer (`lib/runtime/transport.dart`)

```dart
import 'dart:async';
import 'dart:io';
import 'package:http/http.dart' as http;
import 'exceptions.dart';

/// Transport request
class TransportRequest {
  final String url;
  final String method;
  final Map<String, String> headers;
  final String? body;
  final Duration timeout;

  const TransportRequest({
    required this.url,
    required this.method,
    required this.headers,
    this.body,
    required this.timeout,
  });
}

/// Transport response
class TransportResponse {
  final int statusCode;
  final Map<String, String> headers;
  final String body;

  const TransportResponse({
    required this.statusCode,
    required this.headers,
    required this.body,
  });
}

/// Transport protocol for executing HTTP requests
abstract class Transport {
  Future<TransportResponse> execute(TransportRequest request);
}

/// HTTP-based transport implementation
class HttpTransport implements Transport {
  final http.Client _client;
  final List<RequestInterceptor> _interceptors = [];

  HttpTransport({http.Client? client}) : _client = client ?? http.Client();

  /// Adds a request interceptor
  void addInterceptor(RequestInterceptor interceptor) {
    _interceptors.add(interceptor);
  }

  @override
  Future<TransportResponse> execute(TransportRequest request) async {
    var req = request;

    // Apply interceptors
    for (final interceptor in _interceptors) {
      req = await interceptor.intercept(req);
    }

    try {
      final uri = Uri.parse(req.url);
      http.Response response;

      switch (req.method.toUpperCase()) {
        case 'GET':
          response = await _client
              .get(uri, headers: req.headers)
              .timeout(req.timeout);
          break;
        case 'POST':
          response = await _client
              .post(uri, headers: req.headers, body: req.body)
              .timeout(req.timeout);
          break;
        case 'PUT':
          response = await _client
              .put(uri, headers: req.headers, body: req.body)
              .timeout(req.timeout);
          break;
        case 'DELETE':
          response = await _client
              .delete(uri, headers: req.headers, body: req.body)
              .timeout(req.timeout);
          break;
        case 'PATCH':
          response = await _client
              .patch(uri, headers: req.headers, body: req.body)
              .timeout(req.timeout);
          break;
        default:
          throw MizuError.invalidResponse();
      }

      return TransportResponse(
        statusCode: response.statusCode,
        headers: response.headers,
        body: response.body,
      );
    } on SocketException catch (e) {
      throw MizuError.network(e);
    } on TimeoutException catch (e) {
      throw MizuError.network(e);
    } catch (e) {
      throw MizuError.network(e);
    }
  }

  void dispose() {
    _client.close();
  }
}

/// Request interceptor protocol
abstract class RequestInterceptor {
  Future<TransportRequest> intercept(TransportRequest request);
}

/// Logging interceptor for debugging
class LoggingInterceptor implements RequestInterceptor {
  @override
  Future<TransportRequest> intercept(TransportRequest request) async {
    print('[Mizu] ${request.method} ${request.url}');
    return request;
  }
}

/// Retry interceptor with exponential backoff
class RetryInterceptor implements RequestInterceptor {
  final int maxRetries;
  final Duration baseDelay;

  const RetryInterceptor({
    this.maxRetries = 3,
    this.baseDelay = const Duration(seconds: 1),
  });

  @override
  Future<TransportRequest> intercept(TransportRequest request) async {
    // Retry logic is handled at transport level
    return request;
  }
}
```

### Token Store (`lib/runtime/token_store.dart`)

```dart
import 'dart:convert';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';

/// Stored authentication token
class AuthToken {
  final String accessToken;
  final String? refreshToken;
  final DateTime? expiresAt;
  final String tokenType;

  const AuthToken({
    required this.accessToken,
    this.refreshToken,
    this.expiresAt,
    this.tokenType = 'Bearer',
  });

  bool get isExpired {
    if (expiresAt == null) return false;
    return DateTime.now().isAfter(expiresAt!);
  }

  Map<String, dynamic> toJson() => {
        'access_token': accessToken,
        'refresh_token': refreshToken,
        'expires_at': expiresAt?.toIso8601String(),
        'token_type': tokenType,
      };

  factory AuthToken.fromJson(Map<String, dynamic> json) => AuthToken(
        accessToken: json['access_token'] as String,
        refreshToken: json['refresh_token'] as String?,
        expiresAt: json['expires_at'] != null
            ? DateTime.parse(json['expires_at'] as String)
            : null,
        tokenType: json['token_type'] as String? ?? 'Bearer',
      );
}

/// Token change callback type
typedef TokenChangeCallback = void Function(AuthToken? token);

/// Token storage protocol
abstract class TokenStore {
  Future<AuthToken?> getToken();
  Future<void> setToken(AuthToken token);
  Future<void> clearToken();
  void onTokenChange(TokenChangeCallback callback);
}

/// Secure storage-backed token storage
class SecureTokenStore implements TokenStore {
  static const _key = 'mizu_auth_token';
  final FlutterSecureStorage _storage;
  final List<TokenChangeCallback> _observers = [];

  SecureTokenStore({FlutterSecureStorage? storage})
      : _storage = storage ??
            const FlutterSecureStorage(
              aOptions: AndroidOptions(encryptedSharedPreferences: true),
              iOptions: IOSOptions(accessibility: KeychainAccessibility.first_unlock),
            );

  @override
  Future<AuthToken?> getToken() async {
    final json = await _storage.read(key: _key);
    if (json == null) return null;
    try {
      return AuthToken.fromJson(jsonDecode(json) as Map<String, dynamic>);
    } catch (_) {
      return null;
    }
  }

  @override
  Future<void> setToken(AuthToken token) async {
    await _storage.write(key: _key, value: jsonEncode(token.toJson()));
    _notifyObservers(token);
  }

  @override
  Future<void> clearToken() async {
    await _storage.delete(key: _key);
    _notifyObservers(null);
  }

  @override
  void onTokenChange(TokenChangeCallback callback) {
    _observers.add(callback);
  }

  void _notifyObservers(AuthToken? token) {
    for (final observer in _observers) {
      observer(token);
    }
  }
}

/// In-memory token store for testing
class InMemoryTokenStore implements TokenStore {
  AuthToken? _token;
  final List<TokenChangeCallback> _observers = [];

  @override
  Future<AuthToken?> getToken() async => _token;

  @override
  Future<void> setToken(AuthToken token) async {
    _token = token;
    _notifyObservers(token);
  }

  @override
  Future<void> clearToken() async {
    _token = null;
    _notifyObservers(null);
  }

  @override
  void onTokenChange(TokenChangeCallback callback) {
    _observers.add(callback);
  }

  void _notifyObservers(AuthToken? token) {
    for (final observer in _observers) {
      observer(token);
    }
  }
}
```

### Live Streaming (`lib/runtime/live.dart`)

```dart
import 'dart:async';
import 'dart:convert';
import 'package:http/http.dart' as http;
import 'mizu_runtime.dart';
import 'exceptions.dart';

/// Server-sent event
class ServerEvent {
  final String? id;
  final String? event;
  final String data;
  final int? retry;

  const ServerEvent({
    this.id,
    this.event,
    required this.data,
    this.retry,
  });

  /// Decodes the event data as JSON
  T decode<T>(T Function(Map<String, dynamic>) fromJson) {
    return fromJson(jsonDecode(data) as Map<String, dynamic>);
  }

  /// Decodes the event data as a raw map
  Map<String, dynamic> decodeJson() {
    return jsonDecode(data) as Map<String, dynamic>;
  }
}

/// Live connection manager for SSE
class LiveConnection {
  final MizuRuntime _runtime;
  final Map<String, StreamSubscription<String>> _activeConnections = {};

  LiveConnection(this._runtime);

  /// Connects to an SSE endpoint and returns a stream of events
  Stream<ServerEvent> connect({
    required String path,
    Map<String, String>? headers,
  }) {
    final controller = StreamController<ServerEvent>.broadcast();

    _connectInternal(path, headers, controller);

    return controller.stream;
  }

  Future<void> _connectInternal(
    String path,
    Map<String, String>? headers,
    StreamController<ServerEvent> controller,
  ) async {
    final url = _buildUrl(path);
    final allHeaders = await _buildHeaders(headers);

    final request = http.Request('GET', Uri.parse(url));
    request.headers.addAll(allHeaders);
    request.headers['Accept'] = 'text/event-stream';
    request.headers['Cache-Control'] = 'no-cache';

    try {
      final client = http.Client();
      final response = await client.send(request);

      if (response.statusCode != 200) {
        controller.addError(MizuError.http(response.statusCode, 'SSE connection failed'));
        return;
      }

      final eventBuilder = _SSEEventBuilder();
      final subscription = response.stream
          .transform(utf8.decoder)
          .transform(const LineSplitter())
          .listen(
        (line) {
          final event = eventBuilder.processLine(line);
          if (event != null) {
            controller.add(event);
          }
        },
        onError: (error) {
          controller.addError(MizuError.network(error));
        },
        onDone: () {
          controller.close();
          _activeConnections.remove(path);
        },
        cancelOnError: false,
      );

      _activeConnections[path] = subscription;
    } catch (e) {
      controller.addError(MizuError.network(e));
    }
  }

  /// Disconnects from a specific path
  void disconnect(String path) {
    _activeConnections[path]?.cancel();
    _activeConnections.remove(path);
  }

  /// Disconnects all active connections
  void disconnectAll() {
    for (final subscription in _activeConnections.values) {
      subscription.cancel();
    }
    _activeConnections.clear();
  }

  String _buildUrl(String path) {
    final base = _runtime.baseURL.endsWith('/')
        ? _runtime.baseURL.substring(0, _runtime.baseURL.length - 1)
        : _runtime.baseURL;
    final cleanPath = path.startsWith('/') ? path : '/$path';
    return '$base$cleanPath';
  }

  Future<Map<String, String>> _buildHeaders(Map<String, String>? custom) async {
    final headers = <String, String>{};
    headers.addAll(_runtime.defaultHeaders);
    if (custom != null) {
      headers.addAll(custom);
    }

    final token = await _runtime.tokenStore.getToken();
    if (token != null) {
      headers['Authorization'] = 'Bearer ${token.accessToken}';
    }

    return headers;
  }
}

/// SSE event parser
class _SSEEventBuilder {
  String? _id;
  String? _event;
  final List<String> _data = [];
  int? _retry;

  ServerEvent? processLine(String line) {
    if (line.isEmpty) {
      // Empty line means end of event
      if (_data.isEmpty) return null;

      final event = ServerEvent(
        id: _id,
        event: _event,
        data: _data.join('\n'),
        retry: _retry,
      );

      // Reset for next event (keep id for Last-Event-ID)
      _event = null;
      _data.clear();
      _retry = null;

      return event;
    }

    if (line.startsWith(':')) {
      // Comment, ignore
      return null;
    }

    final colonIndex = line.indexOf(':');
    if (colonIndex == -1) {
      // Field with no value
      return null;
    }

    final field = line.substring(0, colonIndex);
    var value = line.substring(colonIndex + 1);
    if (value.startsWith(' ')) {
      value = value.substring(1);
    }

    switch (field) {
      case 'id':
        _id = value;
        break;
      case 'event':
        _event = value;
        break;
      case 'data':
        _data.add(value);
        break;
      case 'retry':
        _retry = int.tryParse(value);
        break;
    }

    return null;
  }
}
```

### Device Info (`lib/runtime/device_info.dart`)

```dart
import 'dart:io';
import 'package:device_info_plus/device_info_plus.dart';
import 'package:package_info_plus/package_info_plus.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:uuid/uuid.dart';

/// Device information container
class DeviceInfoData {
  final String deviceId;
  final String appVersion;
  final String appBuild;
  final String model;
  final String platform;
  final String osVersion;
  final String timezone;
  final String locale;

  const DeviceInfoData({
    required this.deviceId,
    required this.appVersion,
    required this.appBuild,
    required this.model,
    required this.platform,
    required this.osVersion,
    required this.timezone,
    required this.locale,
  });
}

/// Device information utilities
class DeviceInfo {
  static const _deviceIdKey = 'mizu_device_id';
  static final _storage = const FlutterSecureStorage(
    aOptions: AndroidOptions(encryptedSharedPreferences: true),
  );
  static final _deviceInfoPlugin = DeviceInfoPlugin();
  static DeviceInfoData? _cachedInfo;

  /// Collects device information
  static Future<DeviceInfoData> collect() async {
    if (_cachedInfo != null) return _cachedInfo!;

    final deviceId = await _getOrCreateDeviceId();
    final packageInfo = await PackageInfo.fromPlatform();

    String model;
    String osVersion;
    String platform;

    if (Platform.isIOS) {
      final iosInfo = await _deviceInfoPlugin.iosInfo;
      model = iosInfo.utsname.machine;
      osVersion = iosInfo.systemVersion;
      platform = 'ios';
    } else if (Platform.isAndroid) {
      final androidInfo = await _deviceInfoPlugin.androidInfo;
      model = androidInfo.model;
      osVersion = androidInfo.version.release;
      platform = 'android';
    } else if (Platform.isMacOS) {
      final macInfo = await _deviceInfoPlugin.macOsInfo;
      model = macInfo.model;
      osVersion = '${macInfo.majorVersion}.${macInfo.minorVersion}.${macInfo.patchVersion}';
      platform = 'macos';
    } else if (Platform.isWindows) {
      final windowsInfo = await _deviceInfoPlugin.windowsInfo;
      model = windowsInfo.productName;
      osVersion = '${windowsInfo.majorVersion}.${windowsInfo.minorVersion}';
      platform = 'windows';
    } else if (Platform.isLinux) {
      final linuxInfo = await _deviceInfoPlugin.linuxInfo;
      model = linuxInfo.prettyName;
      osVersion = linuxInfo.versionId ?? 'unknown';
      platform = 'linux';
    } else {
      model = 'unknown';
      osVersion = 'unknown';
      platform = 'unknown';
    }

    _cachedInfo = DeviceInfoData(
      deviceId: deviceId,
      appVersion: packageInfo.version,
      appBuild: packageInfo.buildNumber,
      model: model,
      platform: platform,
      osVersion: osVersion,
      timezone: DateTime.now().timeZoneName,
      locale: Platform.localeName,
    );

    return _cachedInfo!;
  }

  static Future<String> _getOrCreateDeviceId() async {
    var deviceId = await _storage.read(key: _deviceIdKey);
    if (deviceId == null) {
      deviceId = const Uuid().v4();
      await _storage.write(key: _deviceIdKey, value: deviceId);
    }
    return deviceId;
  }

  /// Clears cached info (useful for testing)
  static void clearCache() {
    _cachedInfo = null;
  }
}
```

### Exceptions (`lib/runtime/exceptions.dart`)

```dart
/// API error response from server
class APIError {
  final String code;
  final String message;
  final Map<String, dynamic>? details;
  final String? traceId;

  const APIError({
    required this.code,
    required this.message,
    this.details,
    this.traceId,
  });

  factory APIError.fromJson(Map<String, dynamic> json) => APIError(
        code: json['code'] as String,
        message: json['message'] as String,
        details: json['details'] as Map<String, dynamic>?,
        traceId: json['trace_id'] as String?,
      );

  @override
  String toString() => 'APIError($code): $message';
}

/// Mizu client errors
class MizuError implements Exception {
  final String type;
  final String message;
  final Object? cause;

  const MizuError._(this.type, this.message, [this.cause]);

  factory MizuError.invalidResponse() =>
      const MizuError._('invalid_response', 'Invalid server response');

  factory MizuError.http(int statusCode, String body) =>
      MizuError._('http', 'HTTP error $statusCode', body);

  factory MizuError.api(APIError error) =>
      MizuError._('api', error.message, error);

  factory MizuError.network(Object error) =>
      MizuError._('network', 'Network error', error);

  factory MizuError.encoding(Object error) =>
      MizuError._('encoding', 'Encoding error', error);

  factory MizuError.decoding(Object error) =>
      MizuError._('decoding', 'Decoding error', error);

  factory MizuError.unauthorized() =>
      const MizuError._('unauthorized', 'Unauthorized');

  factory MizuError.tokenExpired() =>
      const MizuError._('token_expired', 'Token expired');

  bool get isInvalidResponse => type == 'invalid_response';
  bool get isHttp => type == 'http';
  bool get isApi => type == 'api';
  bool get isNetwork => type == 'network';
  bool get isUnauthorized => type == 'unauthorized';
  bool get isTokenExpired => type == 'token_expired';

  APIError? get apiError => cause is APIError ? cause as APIError : null;
  int? get statusCode => isHttp && cause is String ? null : null;

  @override
  String toString() => 'MizuError($type): $message';
}
```

## App Templates

### Main Entry (`lib/main.dart`)

```dart
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'app.dart';
import 'config/config.dart';
import 'runtime/mizu_runtime.dart';

Future<void> main() async {
  WidgetsFlutterBinding.ensureInitialized();

  // Initialize Mizu runtime
  await MizuRuntime.initialize(
    baseURL: AppConfig.baseURL,
    timeout: AppConfig.timeout,
  );

  runApp(
    const ProviderScope(
      child: {{.Name}}App(),
    ),
  );
}
```

### App Configuration (`lib/app.dart`)

```dart
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'providers/providers.dart';
import 'screens/home_screen.dart';
import 'screens/welcome_screen.dart';

class {{.Name}}App extends ConsumerWidget {
  const {{.Name}}App({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final isAuthenticated = ref.watch(isAuthenticatedProvider);

    return MaterialApp(
      title: '{{.Name}}',
      debugShowCheckedModeBanner: false,
      theme: ThemeData(
        colorScheme: ColorScheme.fromSeed(seedColor: Colors.deepPurple),
        useMaterial3: true,
      ),
      darkTheme: ThemeData(
        colorScheme: ColorScheme.fromSeed(
          seedColor: Colors.deepPurple,
          brightness: Brightness.dark,
        ),
        useMaterial3: true,
      ),
      themeMode: ThemeMode.system,
      home: isAuthenticated ? const HomeScreen() : const WelcomeScreen(),
    );
  }
}
```

### Config (`lib/config/config.dart`)

```dart
import 'package:flutter/foundation.dart';

/// App configuration
class AppConfig {
  /// Base URL for API requests
  static String get baseURL {
    if (kDebugMode) {
      // Use 10.0.2.2 for Android emulator, localhost for iOS simulator
      return const String.fromEnvironment(
        'MIZU_BASE_URL',
        defaultValue: 'http://10.0.2.2:3000',
      );
    }
    return const String.fromEnvironment(
      'MIZU_BASE_URL',
      defaultValue: 'https://api.example.com',
    );
  }

  /// Request timeout
  static Duration get timeout => const Duration(
        seconds: int.fromEnvironment('MIZU_TIMEOUT', defaultValue: 30),
      );

  /// Enable debug mode
  static bool get debug => kDebugMode;
}
```

### Providers (`lib/providers/providers.dart`)

```dart
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../runtime/mizu_runtime.dart';
import '../runtime/token_store.dart';
import '../sdk/client.dart';

/// Mizu runtime provider
final mizuRuntimeProvider = Provider<MizuRuntime>((ref) {
  return MizuRuntime.shared;
});

/// Authentication state provider
final isAuthenticatedProvider = StreamProvider<bool>((ref) {
  final runtime = ref.watch(mizuRuntimeProvider);
  return Stream.value(runtime.isAuthenticated).asBroadcastStream();
});

/// SDK client provider
final clientProvider = Provider<{{.Name}}Client>((ref) {
  final runtime = ref.watch(mizuRuntimeProvider);
  return {{.Name}}Client(runtime);
});

/// Auth token provider
final authTokenProvider = FutureProvider<AuthToken?>((ref) async {
  final runtime = ref.watch(mizuRuntimeProvider);
  return runtime.tokenStore.getToken();
});
```

### Home Screen (`lib/screens/home_screen.dart`)

```dart
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../providers/providers.dart';

class HomeScreen extends ConsumerStatefulWidget {
  const HomeScreen({super.key});

  @override
  ConsumerState<HomeScreen> createState() => _HomeScreenState();
}

class _HomeScreenState extends ConsumerState<HomeScreen> {
  bool _isLoading = false;

  Future<void> _signOut() async {
    setState(() => _isLoading = true);
    try {
      final runtime = ref.read(mizuRuntimeProvider);
      await runtime.tokenStore.clearToken();
    } finally {
      setState(() => _isLoading = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Home'),
        backgroundColor: Theme.of(context).colorScheme.inversePrimary,
      ),
      body: Center(
        child: Padding(
          padding: const EdgeInsets.all(24.0),
          child: Column(
            mainAxisAlignment: MainAxisAlignment.center,
            children: [
              Icon(
                Icons.check_circle,
                size: 64,
                color: Theme.of(context).colorScheme.primary,
              ),
              const SizedBox(height: 16),
              Text(
                'Welcome to {{.Name}}',
                style: Theme.of(context).textTheme.headlineMedium,
                textAlign: TextAlign.center,
              ),
              const SizedBox(height: 8),
              Text(
                'Connected to Mizu backend',
                style: Theme.of(context).textTheme.bodyLarge?.copyWith(
                      color: Theme.of(context).colorScheme.onSurfaceVariant,
                    ),
              ),
              const SizedBox(height: 32),
              if (_isLoading)
                const CircularProgressIndicator()
              else
                FilledButton.tonal(
                  onPressed: _signOut,
                  child: const Text('Sign Out'),
                ),
            ],
          ),
        ),
      ),
    );
  }
}
```

### Welcome Screen (`lib/screens/welcome_screen.dart`)

```dart
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../providers/providers.dart';
import '../runtime/token_store.dart';

class WelcomeScreen extends ConsumerWidget {
  const WelcomeScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return Scaffold(
      body: SafeArea(
        child: Padding(
          padding: const EdgeInsets.all(24.0),
          child: Column(
            mainAxisAlignment: MainAxisAlignment.center,
            children: [
              const Spacer(),
              Icon(
                Icons.flutter_dash,
                size: 80,
                color: Theme.of(context).colorScheme.primary,
              ),
              const SizedBox(height: 24),
              Text(
                'Welcome to {{.Name}}',
                style: Theme.of(context).textTheme.headlineLarge,
                textAlign: TextAlign.center,
              ),
              const SizedBox(height: 16),
              Text(
                'A modern Flutter app powered by Mizu',
                style: Theme.of(context).textTheme.bodyLarge?.copyWith(
                      color: Theme.of(context).colorScheme.onSurfaceVariant,
                    ),
                textAlign: TextAlign.center,
              ),
              const Spacer(),
              FilledButton(
                onPressed: () => _getStarted(context, ref),
                style: FilledButton.styleFrom(
                  minimumSize: const Size.fromHeight(56),
                ),
                child: const Text('Get Started'),
              ),
              const SizedBox(height: 48),
            ],
          ),
        ),
      ),
    );
  }

  Future<void> _getStarted(BuildContext context, WidgetRef ref) async {
    // Demo: Set a test token
    final runtime = ref.read(mizuRuntimeProvider);
    await runtime.tokenStore.setToken(
      const AuthToken(accessToken: 'demo_token'),
    );
  }
}
```

### Loading View (`lib/widgets/loading_view.dart`)

```dart
import 'package:flutter/material.dart';

class LoadingView extends StatelessWidget {
  final String? message;

  const LoadingView({super.key, this.message});

  @override
  Widget build(BuildContext context) {
    return Center(
      child: Column(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          const CircularProgressIndicator(),
          if (message != null) ...[
            const SizedBox(height: 16),
            Text(
              message!,
              style: Theme.of(context).textTheme.bodyMedium,
            ),
          ],
        ],
      ),
    );
  }
}
```

### Error View (`lib/widgets/error_view.dart`)

```dart
import 'package:flutter/material.dart';
import '../runtime/exceptions.dart';

class ErrorView extends StatelessWidget {
  final Object error;
  final VoidCallback? onRetry;

  const ErrorView({
    super.key,
    required this.error,
    this.onRetry,
  });

  @override
  Widget build(BuildContext context) {
    String message;
    if (error is MizuError) {
      message = (error as MizuError).message;
    } else {
      message = error.toString();
    }

    return Center(
      child: Padding(
        padding: const EdgeInsets.all(24.0),
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            Icon(
              Icons.error_outline,
              size: 48,
              color: Theme.of(context).colorScheme.error,
            ),
            const SizedBox(height: 16),
            Text(
              'Something went wrong',
              style: Theme.of(context).textTheme.titleLarge,
            ),
            const SizedBox(height: 8),
            Text(
              message,
              style: Theme.of(context).textTheme.bodyMedium?.copyWith(
                    color: Theme.of(context).colorScheme.onSurfaceVariant,
                  ),
              textAlign: TextAlign.center,
            ),
            if (onRetry != null) ...[
              const SizedBox(height: 24),
              FilledButton.tonal(
                onPressed: onRetry,
                child: const Text('Try Again'),
              ),
            ],
          ],
        ),
      ),
    );
  }
}
```

### App State (`lib/models/app_state.dart`)

```dart
import 'package:flutter/foundation.dart';

/// Application state
class AppState extends ChangeNotifier {
  bool _isOnboarded = false;
  bool get isOnboarded => _isOnboarded;

  AppTab _selectedTab = AppTab.home;
  AppTab get selectedTab => _selectedTab;

  void completeOnboarding() {
    _isOnboarded = true;
    notifyListeners();
  }

  void selectTab(AppTab tab) {
    _selectedTab = tab;
    notifyListeners();
  }
}

enum AppTab {
  home,
  profile,
  settings,
}
```

## Generated SDK

### Client (`lib/sdk/client.dart`)

```dart
import '../runtime/mizu_runtime.dart';
import '../runtime/token_store.dart';
import 'types.dart';

/// Generated Mizu API client for {{.Name}}
class {{.Name}}Client {
  final MizuRuntime _runtime;

  {{.Name}}Client(this._runtime);

  // MARK: - Auth

  /// Sign in with credentials
  Future<AuthResponse> signIn({
    required String email,
    required String password,
  }) async {
    final response = await _runtime.post<Map<String, dynamic>, Map<String, dynamic>>(
      '/auth/signin',
      body: {'email': email, 'password': password},
    );
    return AuthResponse.fromJson(response);
  }

  /// Sign up with credentials
  Future<AuthResponse> signUp({
    required String email,
    required String password,
    required String name,
  }) async {
    final response = await _runtime.post<Map<String, dynamic>, Map<String, dynamic>>(
      '/auth/signup',
      body: {'email': email, 'password': password, 'name': name},
    );
    return AuthResponse.fromJson(response);
  }

  /// Sign out
  Future<void> signOut() async {
    await _runtime.delete<void>('/auth/signout');
    await _runtime.tokenStore.clearToken();
  }

  // MARK: - Users

  /// Get current user profile
  Future<User> getCurrentUser() async {
    final response = await _runtime.get<Map<String, dynamic>>('/users/me');
    return User.fromJson(response);
  }

  /// Update current user profile
  Future<User> updateCurrentUser(UserUpdate update) async {
    final response = await _runtime.put<Map<String, dynamic>, Map<String, dynamic>>(
      '/users/me',
      body: update.toJson(),
    );
    return User.fromJson(response);
  }
}

/// Extension to store auth response token
extension AuthResponseExtension on {{.Name}}Client {
  Future<void> storeAuthToken(AuthResponse response) async {
    final expiresAt = DateTime.now().add(Duration(seconds: response.token.expiresIn));
    await _runtime.tokenStore.setToken(
      AuthToken(
        accessToken: response.token.accessToken,
        refreshToken: response.token.refreshToken,
        expiresAt: expiresAt,
      ),
    );
  }
}
```

### Types (`lib/sdk/types.dart`)

```dart
// MARK: - Auth Types

class SignInRequest {
  final String email;
  final String password;

  const SignInRequest({required this.email, required this.password});

  Map<String, dynamic> toJson() => {'email': email, 'password': password};
}

class SignUpRequest {
  final String email;
  final String password;
  final String name;

  const SignUpRequest({
    required this.email,
    required this.password,
    required this.name,
  });

  Map<String, dynamic> toJson() => {
        'email': email,
        'password': password,
        'name': name,
      };
}

class AuthResponse {
  final User user;
  final TokenResponse token;

  const AuthResponse({required this.user, required this.token});

  factory AuthResponse.fromJson(Map<String, dynamic> json) => AuthResponse(
        user: User.fromJson(json['user'] as Map<String, dynamic>),
        token: TokenResponse.fromJson(json['token'] as Map<String, dynamic>),
      );
}

class TokenResponse {
  final String accessToken;
  final String? refreshToken;
  final int expiresIn;

  const TokenResponse({
    required this.accessToken,
    this.refreshToken,
    required this.expiresIn,
  });

  factory TokenResponse.fromJson(Map<String, dynamic> json) => TokenResponse(
        accessToken: json['access_token'] as String,
        refreshToken: json['refresh_token'] as String?,
        expiresIn: json['expires_in'] as int,
      );
}

// MARK: - User Types

class User {
  final String id;
  final String email;
  final String name;
  final String? avatarUrl;
  final DateTime createdAt;
  final DateTime updatedAt;

  const User({
    required this.id,
    required this.email,
    required this.name,
    this.avatarUrl,
    required this.createdAt,
    required this.updatedAt,
  });

  factory User.fromJson(Map<String, dynamic> json) => User(
        id: json['id'] as String,
        email: json['email'] as String,
        name: json['name'] as String,
        avatarUrl: json['avatar_url'] as String?,
        createdAt: DateTime.parse(json['created_at'] as String),
        updatedAt: DateTime.parse(json['updated_at'] as String),
      );

  Map<String, dynamic> toJson() => {
        'id': id,
        'email': email,
        'name': name,
        'avatar_url': avatarUrl,
        'created_at': createdAt.toIso8601String(),
        'updated_at': updatedAt.toIso8601String(),
      };
}

class UserUpdate {
  final String? name;
  final String? avatarUrl;

  const UserUpdate({this.name, this.avatarUrl});

  Map<String, dynamic> toJson() {
    final map = <String, dynamic>{};
    if (name != null) map['name'] = name;
    if (avatarUrl != null) map['avatar_url'] = avatarUrl;
    return map;
  }
}
```

### Extensions (`lib/sdk/extensions.dart`)

```dart
import '../runtime/mizu_runtime.dart';
import '../runtime/token_store.dart';
import 'types.dart';

/// Convenience extensions for working with the SDK
extension MizuRuntimeExtensions on MizuRuntime {
  /// Store an auth response token
  Future<void> storeAuthToken(AuthResponse response) async {
    final expiresAt = DateTime.now().add(Duration(seconds: response.token.expiresIn));
    await tokenStore.setToken(
      AuthToken(
        accessToken: response.token.accessToken,
        refreshToken: response.token.refreshToken,
        expiresAt: expiresAt,
      ),
    );
  }
}
```

## Build Configuration

### pubspec.yaml

```yaml
name: {{.Name | lower}}
description: A Flutter app powered by Mizu
publish_to: 'none'
version: 1.0.0+1

environment:
  sdk: '>=3.2.0 <4.0.0'

dependencies:
  flutter:
    sdk: flutter
  flutter_riverpod: ^2.4.9
  http: ^1.1.2
  flutter_secure_storage: ^9.0.0
  device_info_plus: ^10.1.0
  package_info_plus: ^5.0.1
  uuid: ^4.2.2

dev_dependencies:
  flutter_test:
    sdk: flutter
  flutter_lints: ^3.0.1
  integration_test:
    sdk: flutter
  mocktail: ^1.0.1

flutter:
  uses-material-design: true
```

### analysis_options.yaml

```yaml
include: package:flutter_lints/flutter.yaml

linter:
  rules:
    - prefer_const_constructors
    - prefer_const_declarations
    - prefer_final_fields
    - prefer_final_locals
    - avoid_print
    - require_trailing_commas

analyzer:
  errors:
    invalid_annotation_target: ignore
  exclude:
    - '**/*.g.dart'
    - '**/*.freezed.dart'
```

## Testing

### Unit Tests (`test/runtime_test.dart`)

```dart
import 'package:flutter_test/flutter_test.dart';
import 'package:{{.Name | lower}}/runtime/token_store.dart';
import 'package:{{.Name | lower}}/runtime/exceptions.dart';

void main() {
  group('InMemoryTokenStore', () {
    late InMemoryTokenStore store;

    setUp(() {
      store = InMemoryTokenStore();
    });

    test('returns null when no token stored', () async {
      expect(await store.getToken(), isNull);
    });

    test('stores and retrieves token', () async {
      const token = AuthToken(
        accessToken: 'test123',
        refreshToken: 'refresh456',
      );

      await store.setToken(token);
      final retrieved = await store.getToken();

      expect(retrieved?.accessToken, equals('test123'));
      expect(retrieved?.refreshToken, equals('refresh456'));
    });

    test('clears token', () async {
      const token = AuthToken(accessToken: 'test123');
      await store.setToken(token);
      await store.clearToken();

      expect(await store.getToken(), isNull);
    });

    test('notifies observers on token change', () async {
      AuthToken? notifiedToken;
      store.onTokenChange((token) => notifiedToken = token);

      const token = AuthToken(accessToken: 'test123');
      await store.setToken(token);

      expect(notifiedToken?.accessToken, equals('test123'));
    });
  });

  group('AuthToken', () {
    test('isExpired returns false when no expiry', () {
      const token = AuthToken(accessToken: 'test');
      expect(token.isExpired, isFalse);
    });

    test('isExpired returns true when expired', () {
      final token = AuthToken(
        accessToken: 'test',
        expiresAt: DateTime.now().subtract(const Duration(seconds: 1)),
      );
      expect(token.isExpired, isTrue);
    });

    test('isExpired returns false when not expired', () {
      final token = AuthToken(
        accessToken: 'test',
        expiresAt: DateTime.now().add(const Duration(hours: 1)),
      );
      expect(token.isExpired, isFalse);
    });
  });

  group('MizuError', () {
    test('creates network error', () {
      final error = MizuError.network(Exception('test'));
      expect(error.isNetwork, isTrue);
      expect(error.message, equals('Network error'));
    });

    test('creates api error', () {
      const apiError = APIError(code: 'test_error', message: 'Test message');
      final error = MizuError.api(apiError);
      expect(error.isApi, isTrue);
      expect(error.apiError?.code, equals('test_error'));
    });
  });
}
```

### Widget Tests (`test/widget_test.dart`)

```dart
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:{{.Name | lower}}/screens/welcome_screen.dart';
import 'package:{{.Name | lower}}/widgets/loading_view.dart';
import 'package:{{.Name | lower}}/widgets/error_view.dart';

void main() {
  group('WelcomeScreen', () {
    testWidgets('displays welcome message', (tester) async {
      await tester.pumpWidget(
        const ProviderScope(
          child: MaterialApp(home: WelcomeScreen()),
        ),
      );

      expect(find.text('Welcome to {{.Name}}'), findsOneWidget);
      expect(find.text('Get Started'), findsOneWidget);
    });
  });

  group('LoadingView', () {
    testWidgets('displays progress indicator', (tester) async {
      await tester.pumpWidget(
        const MaterialApp(home: Scaffold(body: LoadingView())),
      );

      expect(find.byType(CircularProgressIndicator), findsOneWidget);
    });

    testWidgets('displays message when provided', (tester) async {
      await tester.pumpWidget(
        const MaterialApp(
          home: Scaffold(body: LoadingView(message: 'Loading...')),
        ),
      );

      expect(find.text('Loading...'), findsOneWidget);
    });
  });

  group('ErrorView', () {
    testWidgets('displays error message', (tester) async {
      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(body: ErrorView(error: Exception('Test error'))),
        ),
      );

      expect(find.text('Something went wrong'), findsOneWidget);
    });

    testWidgets('displays retry button when callback provided', (tester) async {
      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(
            body: ErrorView(
              error: Exception('Test'),
              onRetry: () {},
            ),
          ),
        ),
      );

      expect(find.text('Try Again'), findsOneWidget);
    });
  });
}
```

### Integration Tests (`integration_test/app_test.dart`)

```dart
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:integration_test/integration_test.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:{{.Name | lower}}/app.dart';

void main() {
  IntegrationTestWidgetsFlutterBinding.ensureInitialized();

  group('App Integration Tests', () {
    testWidgets('Welcome screen to Home screen flow', (tester) async {
      await tester.pumpWidget(
        const ProviderScope(
          child: {{.Name}}App(),
        ),
      );
      await tester.pumpAndSettle();

      // Verify welcome screen is shown
      expect(find.text('Welcome to {{.Name}}'), findsOneWidget);

      // Tap get started button
      await tester.tap(find.text('Get Started'));
      await tester.pumpAndSettle();

      // Verify home screen is shown
      expect(find.text('Connected to Mizu backend'), findsOneWidget);
    });
  });
}
```

## UI Variants

### Material 3 (Default)
- Modern Material Design 3
- Dynamic color theming
- Adaptive layouts
- Material widgets

### Cupertino
- iOS-style design
- CupertinoApp and CupertinoPageScaffold
- iOS navigation patterns
- Platform-specific widgets

## Platform Configuration

### iOS (`ios/Runner/Info.plist`)
- NSAppTransportSecurity for local development
- Bundle identifier configuration
- Required device capabilities

### Android (`android/app/build.gradle`)
- minSdk configuration
- proguard rules for release builds
- Network security configuration

## References

- [Flutter Documentation](https://docs.flutter.dev/)
- [Dart Language](https://dart.dev/)
- [Riverpod State Management](https://riverpod.dev/)
- [Material 3 Design](https://m3.material.io/)
- [flutter_secure_storage](https://pub.dev/packages/flutter_secure_storage)
- [device_info_plus](https://pub.dev/packages/device_info_plus)
- [Mobile Package Spec](./0095_mobile.md)
