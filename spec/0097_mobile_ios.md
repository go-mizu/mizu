# iOS Template Specification

**Status:** Draft
**Author:** Mizu Team
**Created:** 2025-12-20
**Template:** `mobile:ios`

## Overview

The `mobile:ios` template generates a production-ready iOS application with full Mizu backend integration. It follows Apple's modern development practices with SwiftUI-first design, async/await concurrency, and Swift Package Manager.

## Template Invocation

```bash
# Default: SwiftUI
mizu new ./MyApp --template mobile:ios

# With UIKit variant
mizu new ./MyApp --template mobile:ios --var ui=uikit

# With hybrid variant (SwiftUI + UIKit)
mizu new ./MyApp --template mobile:ios --var ui=hybrid

# Custom bundle ID
mizu new ./MyApp --template mobile:ios --var bundleId=com.company.myapp

# Minimum iOS version
mizu new ./MyApp --template mobile:ios --var minIOS=15
```

## Generated Project Structure

```
{{.Name}}/
├── {{.Name}}.xcodeproj/
│   └── project.pbxproj
├── {{.Name}}/
│   ├── App/
│   │   ├── {{.Name}}App.swift          # @main entry point
│   │   ├── ContentView.swift           # Root view
│   │   ├── AppDelegate.swift           # (UIKit/hybrid only)
│   │   └── SceneDelegate.swift         # (UIKit/hybrid only)
│   ├── Views/
│   │   ├── HomeView.swift
│   │   └── Components/
│   │       └── LoadingView.swift
│   ├── Models/
│   │   └── AppState.swift
│   ├── SDK/
│   │   ├── Client.swift                # Generated Mizu client
│   │   ├── Types.swift                 # Generated types
│   │   └── Extensions.swift            # Convenience extensions
│   ├── Runtime/
│   │   ├── MizuRuntime.swift           # Core runtime
│   │   ├── Transport.swift             # URLSession transport
│   │   ├── TokenStore.swift            # Keychain storage
│   │   ├── Live.swift                  # SSE streaming
│   │   ├── Sync.swift                  # Offline sync
│   │   └── Config.swift                # Configuration
│   └── Resources/
│       ├── Assets.xcassets/
│       │   ├── AppIcon.appiconset/
│       │   ├── AccentColor.colorset/
│       │   └── Contents.json
│       ├── Info.plist
│       ├── LaunchScreen.storyboard
│       └── Localizable.strings
├── {{.Name}}Tests/
│   ├── {{.Name}}Tests.swift
│   ├── RuntimeTests.swift
│   └── SDKTests.swift
├── {{.Name}}UITests/
│   └── {{.Name}}UITests.swift
├── Package.swift                        # SPM manifest (optional)
├── .gitignore
└── README.md
```

## MizuMobileRuntime

### Core Runtime (`MizuRuntime.swift`)

```swift
import Foundation

/// MizuRuntime is the core client for communicating with a Mizu backend.
@MainActor
public final class MizuRuntime: ObservableObject {
    /// Shared singleton instance
    public static let shared = MizuRuntime()

    /// Base URL for all API requests
    public var baseURL: URL

    /// HTTP transport layer
    public let transport: Transport

    /// Secure token storage
    public let tokenStore: TokenStore

    /// Live connection manager
    public private(set) lazy var live: LiveConnection = {
        LiveConnection(runtime: self)
    }()

    /// Current authentication state
    @Published public private(set) var isAuthenticated: Bool = false

    /// Default headers added to all requests
    public var defaultHeaders: [String: String] = [:]

    /// Request timeout interval
    public var timeoutInterval: TimeInterval = 30

    /// Initializes the runtime with configuration
    public init(
        baseURL: URL = URL(string: "http://localhost:3000")!,
        transport: Transport? = nil,
        tokenStore: TokenStore? = nil
    ) {
        self.baseURL = baseURL
        self.transport = transport ?? URLSessionTransport()
        self.tokenStore = tokenStore ?? KeychainTokenStore()

        // Observe token changes
        self.tokenStore.onTokenChange { [weak self] token in
            Task { @MainActor in
                self?.isAuthenticated = token != nil
            }
        }

        // Check initial auth state
        self.isAuthenticated = self.tokenStore.getToken() != nil
    }

    // MARK: - HTTP Methods

    /// Performs a GET request
    public func get<T: Decodable>(
        _ path: String,
        query: [String: String]? = nil,
        headers: [String: String]? = nil
    ) async throws -> T {
        try await request(method: .get, path: path, query: query, headers: headers)
    }

    /// Performs a POST request
    public func post<T: Decodable, B: Encodable>(
        _ path: String,
        body: B,
        headers: [String: String]? = nil
    ) async throws -> T {
        try await request(method: .post, path: path, body: body, headers: headers)
    }

    /// Performs a PUT request
    public func put<T: Decodable, B: Encodable>(
        _ path: String,
        body: B,
        headers: [String: String]? = nil
    ) async throws -> T {
        try await request(method: .put, path: path, body: body, headers: headers)
    }

    /// Performs a DELETE request
    public func delete<T: Decodable>(
        _ path: String,
        headers: [String: String]? = nil
    ) async throws -> T {
        try await request(method: .delete, path: path, headers: headers)
    }

    /// Performs a DELETE request with no response body
    public func delete(
        _ path: String,
        headers: [String: String]? = nil
    ) async throws {
        let _: EmptyResponse = try await request(method: .delete, path: path, headers: headers)
    }

    // MARK: - Streaming

    /// Opens a streaming connection for SSE
    public func stream(
        _ path: String,
        headers: [String: String]? = nil
    ) -> AsyncThrowingStream<ServerEvent, Error> {
        live.connect(path: path, headers: headers)
    }

    // MARK: - Private

    private func request<T: Decodable, B: Encodable>(
        method: HTTPMethod,
        path: String,
        query: [String: String]? = nil,
        body: B? = nil,
        headers: [String: String]? = nil
    ) async throws -> T {
        var url = baseURL.appendingPathComponent(path)

        // Add query parameters
        if let query = query, !query.isEmpty {
            var components = URLComponents(url: url, resolvingAgainstBaseURL: true)!
            components.queryItems = query.map { URLQueryItem(name: $0.key, value: $0.value) }
            url = components.url!
        }

        var request = URLRequest(url: url)
        request.httpMethod = method.rawValue
        request.timeoutInterval = timeoutInterval

        // Merge headers
        var allHeaders = defaultHeaders
        allHeaders.merge(mobileHeaders()) { _, new in new }
        if let headers = headers {
            allHeaders.merge(headers) { _, new in new }
        }

        // Add auth token
        if let token = tokenStore.getToken() {
            allHeaders["Authorization"] = "Bearer \(token.accessToken)"
        }

        // Set headers
        for (key, value) in allHeaders {
            request.setValue(value, forHTTPHeaderField: key)
        }

        // Encode body
        if let body = body {
            request.httpBody = try JSONEncoder().encode(body)
            request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        }

        // Execute request
        let response = try await transport.execute(request)

        // Handle response
        guard let httpResponse = response.response as? HTTPURLResponse else {
            throw MizuError.invalidResponse
        }

        // Handle errors
        if httpResponse.statusCode >= 400 {
            throw try parseError(data: response.data, statusCode: httpResponse.statusCode)
        }

        // Decode response
        if T.self == EmptyResponse.self {
            return EmptyResponse() as! T
        }

        return try JSONDecoder().decode(T.self, from: response.data)
    }

    private func request<T: Decodable>(
        method: HTTPMethod,
        path: String,
        query: [String: String]? = nil,
        headers: [String: String]? = nil
    ) async throws -> T {
        try await request(method: method, path: path, query: query, body: Optional<EmptyBody>.none, headers: headers)
    }

    private func mobileHeaders() -> [String: String] {
        var headers: [String: String] = [:]

        // Device identification
        headers["X-Device-ID"] = DeviceInfo.deviceID
        headers["X-App-Version"] = Bundle.main.appVersion
        headers["X-App-Build"] = Bundle.main.buildNumber
        headers["X-Device-Model"] = DeviceInfo.model
        headers["X-Platform"] = "ios"
        headers["X-OS-Version"] = DeviceInfo.osVersion
        headers["X-Timezone"] = TimeZone.current.identifier
        headers["X-Locale"] = Locale.current.identifier

        return headers
    }

    private func parseError(data: Data, statusCode: Int) throws -> MizuError {
        if let apiError = try? JSONDecoder().decode(APIError.self, from: data) {
            return .api(apiError)
        }
        return .http(statusCode: statusCode, data: data)
    }
}

// MARK: - Supporting Types

public enum HTTPMethod: String {
    case get = "GET"
    case post = "POST"
    case put = "PUT"
    case delete = "DELETE"
    case patch = "PATCH"
}

public struct EmptyResponse: Decodable {}
public struct EmptyBody: Encodable {}

/// API error response from server
public struct APIError: Decodable, LocalizedError {
    public let code: String
    public let message: String
    public let details: [String: AnyCodable]?
    public let traceID: String?

    public var errorDescription: String? { message }

    enum CodingKeys: String, CodingKey {
        case code, message, details
        case traceID = "trace_id"
    }
}

/// Mizu client errors
public enum MizuError: LocalizedError {
    case invalidResponse
    case http(statusCode: Int, data: Data)
    case api(APIError)
    case network(Error)
    case encoding(Error)
    case decoding(Error)
    case unauthorized
    case tokenExpired

    public var errorDescription: String? {
        switch self {
        case .invalidResponse:
            return "Invalid server response"
        case .http(let code, _):
            return "HTTP error \(code)"
        case .api(let error):
            return error.message
        case .network(let error):
            return error.localizedDescription
        case .encoding(let error):
            return "Encoding error: \(error.localizedDescription)"
        case .decoding(let error):
            return "Decoding error: \(error.localizedDescription)"
        case .unauthorized:
            return "Unauthorized"
        case .tokenExpired:
            return "Token expired"
        }
    }
}
```

### Transport Layer (`Transport.swift`)

```swift
import Foundation

/// Transport protocol for executing HTTP requests
public protocol Transport {
    func execute(_ request: URLRequest) async throws -> TransportResponse
}

/// Response from transport layer
public struct TransportResponse {
    public let data: Data
    public let response: URLResponse
}

/// URLSession-based transport implementation
public final class URLSessionTransport: Transport {
    private let session: URLSession
    private var interceptors: [RequestInterceptor] = []

    public init(configuration: URLSessionConfiguration = .default) {
        self.session = URLSession(configuration: configuration)
    }

    /// Adds a request interceptor
    public func addInterceptor(_ interceptor: RequestInterceptor) {
        interceptors.append(interceptor)
    }

    public func execute(_ request: URLRequest) async throws -> TransportResponse {
        var request = request

        // Apply interceptors
        for interceptor in interceptors {
            request = try await interceptor.intercept(request)
        }

        do {
            let (data, response) = try await session.data(for: request)
            return TransportResponse(data: data, response: response)
        } catch {
            throw MizuError.network(error)
        }
    }
}

/// Request interceptor protocol
public protocol RequestInterceptor {
    func intercept(_ request: URLRequest) async throws -> URLRequest
}

/// Logging interceptor for debugging
public final class LoggingInterceptor: RequestInterceptor {
    public init() {}

    public func intercept(_ request: URLRequest) async throws -> URLRequest {
        #if DEBUG
        print("[Mizu] \(request.httpMethod ?? "?") \(request.url?.absoluteString ?? "?")")
        #endif
        return request
    }
}

/// Retry interceptor with exponential backoff
public final class RetryInterceptor: RequestInterceptor {
    private let maxRetries: Int
    private let baseDelay: TimeInterval

    public init(maxRetries: Int = 3, baseDelay: TimeInterval = 1.0) {
        self.maxRetries = maxRetries
        self.baseDelay = baseDelay
    }

    public func intercept(_ request: URLRequest) async throws -> URLRequest {
        // Retry logic is handled at transport level, this just marks the request
        return request
    }
}
```

### Token Store (`TokenStore.swift`)

```swift
import Foundation
import Security

/// Stored authentication token
public struct AuthToken: Codable {
    public let accessToken: String
    public let refreshToken: String?
    public let expiresAt: Date?
    public let tokenType: String

    public var isExpired: Bool {
        guard let expiresAt = expiresAt else { return false }
        return Date() >= expiresAt
    }

    public init(
        accessToken: String,
        refreshToken: String? = nil,
        expiresAt: Date? = nil,
        tokenType: String = "Bearer"
    ) {
        self.accessToken = accessToken
        self.refreshToken = refreshToken
        self.expiresAt = expiresAt
        self.tokenType = tokenType
    }
}

/// Token storage protocol
public protocol TokenStore {
    func getToken() -> AuthToken?
    func setToken(_ token: AuthToken) throws
    func clearToken() throws
    func onTokenChange(_ callback: @escaping (AuthToken?) -> Void)
}

/// Keychain-backed token storage
public final class KeychainTokenStore: TokenStore {
    private let service: String
    private let account: String
    private var observers: [(AuthToken?) -> Void] = []

    public init(service: String = Bundle.main.bundleIdentifier ?? "com.mizu.app",
                account: String = "auth_token") {
        self.service = service
        self.account = account
    }

    public func getToken() -> AuthToken? {
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
            kSecAttrAccount as String: account,
            kSecReturnData as String: true,
            kSecMatchLimit as String: kSecMatchLimitOne
        ]

        var result: AnyObject?
        let status = SecItemCopyMatching(query as CFDictionary, &result)

        guard status == errSecSuccess,
              let data = result as? Data else {
            return nil
        }

        return try? JSONDecoder().decode(AuthToken.self, from: data)
    }

    public func setToken(_ token: AuthToken) throws {
        let data = try JSONEncoder().encode(token)

        // Delete existing token first
        try? clearToken()

        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
            kSecAttrAccount as String: account,
            kSecValueData as String: data,
            kSecAttrAccessible as String: kSecAttrAccessibleWhenUnlockedThisDeviceOnly
        ]

        let status = SecItemAdd(query as CFDictionary, nil)
        guard status == errSecSuccess else {
            throw KeychainError.unableToSave(status)
        }

        notifyObservers(token)
    }

    public func clearToken() throws {
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
            kSecAttrAccount as String: account
        ]

        let status = SecItemDelete(query as CFDictionary)
        guard status == errSecSuccess || status == errSecItemNotFound else {
            throw KeychainError.unableToDelete(status)
        }

        notifyObservers(nil)
    }

    public func onTokenChange(_ callback: @escaping (AuthToken?) -> Void) {
        observers.append(callback)
    }

    private func notifyObservers(_ token: AuthToken?) {
        observers.forEach { $0(token) }
    }
}

/// Keychain errors
public enum KeychainError: LocalizedError {
    case unableToSave(OSStatus)
    case unableToDelete(OSStatus)

    public var errorDescription: String? {
        switch self {
        case .unableToSave(let status):
            return "Unable to save to keychain: \(status)"
        case .unableToDelete(let status):
            return "Unable to delete from keychain: \(status)"
        }
    }
}
```

### Live Streaming (`Live.swift`)

```swift
import Foundation

/// Server-sent event
public struct ServerEvent: Sendable {
    public let id: String?
    public let event: String?
    public let data: String
    public let retry: Int?

    public init(id: String? = nil, event: String? = nil, data: String, retry: Int? = nil) {
        self.id = id
        self.event = event
        self.data = data
        self.retry = retry
    }

    /// Decodes the event data as JSON
    public func decode<T: Decodable>(_ type: T.Type) throws -> T {
        guard let data = data.data(using: .utf8) else {
            throw MizuError.decoding(NSError(domain: "LiveConnection", code: 1))
        }
        return try JSONDecoder().decode(type, from: data)
    }
}

/// Live connection manager for SSE
public final class LiveConnection: @unchecked Sendable {
    private weak var runtime: MizuRuntime?
    private var activeTasks: [String: Task<Void, Never>] = [:]
    private let lock = NSLock()

    init(runtime: MizuRuntime) {
        self.runtime = runtime
    }

    /// Connects to an SSE endpoint and returns an async stream of events
    public func connect(
        path: String,
        headers: [String: String]? = nil
    ) -> AsyncThrowingStream<ServerEvent, Error> {
        AsyncThrowingStream { continuation in
            let task = Task {
                await self.streamEvents(path: path, headers: headers, continuation: continuation)
            }

            lock.withLock {
                activeTasks[path] = task
            }

            continuation.onTermination = { @Sendable _ in
                task.cancel()
                self.lock.withLock {
                    self.activeTasks.removeValue(forKey: path)
                }
            }
        }
    }

    /// Disconnects from a specific path
    public func disconnect(path: String) {
        lock.withLock {
            activeTasks[path]?.cancel()
            activeTasks.removeValue(forKey: path)
        }
    }

    /// Disconnects all active connections
    public func disconnectAll() {
        lock.withLock {
            activeTasks.values.forEach { $0.cancel() }
            activeTasks.removeAll()
        }
    }

    private func streamEvents(
        path: String,
        headers: [String: String]?,
        continuation: AsyncThrowingStream<ServerEvent, Error>.Continuation
    ) async {
        guard let runtime = runtime else {
            continuation.finish(throwing: MizuError.invalidResponse)
            return
        }

        let url = runtime.baseURL.appendingPathComponent(path)
        var request = URLRequest(url: url)
        request.setValue("text/event-stream", forHTTPHeaderField: "Accept")
        request.setValue("no-cache", forHTTPHeaderField: "Cache-Control")

        // Add default headers
        for (key, value) in runtime.defaultHeaders {
            request.setValue(value, forHTTPHeaderField: key)
        }

        // Add custom headers
        if let headers = headers {
            for (key, value) in headers {
                request.setValue(value, forHTTPHeaderField: key)
            }
        }

        // Add auth token
        if let token = runtime.tokenStore.getToken() {
            request.setValue("Bearer \(token.accessToken)", forHTTPHeaderField: "Authorization")
        }

        do {
            let (bytes, response) = try await URLSession.shared.bytes(for: request)

            guard let httpResponse = response as? HTTPURLResponse,
                  httpResponse.statusCode == 200 else {
                continuation.finish(throwing: MizuError.invalidResponse)
                return
            }

            var eventBuilder = SSEEventBuilder()

            for try await line in bytes.lines {
                if Task.isCancelled {
                    break
                }

                if let event = eventBuilder.processLine(line) {
                    continuation.yield(event)
                }
            }

            continuation.finish()
        } catch {
            if !Task.isCancelled {
                continuation.finish(throwing: MizuError.network(error))
            }
        }
    }
}

/// SSE event parser
private struct SSEEventBuilder {
    private var id: String?
    private var event: String?
    private var data: [String] = []
    private var retry: Int?

    mutating func processLine(_ line: String) -> ServerEvent? {
        if line.isEmpty {
            // Empty line means end of event
            guard !data.isEmpty else { return nil }

            let event = ServerEvent(
                id: id,
                event: event,
                data: data.joined(separator: "\n"),
                retry: retry
            )

            // Reset for next event (keep id for Last-Event-ID)
            self.event = nil
            self.data = []
            self.retry = nil

            return event
        }

        if line.hasPrefix(":") {
            // Comment, ignore
            return nil
        }

        let parts = line.split(separator: ":", maxSplits: 1)
        let field = String(parts[0])
        let value = parts.count > 1 ? String(parts[1]).trimmingCharacters(in: .whitespaces) : ""

        switch field {
        case "id":
            id = value
        case "event":
            event = value
        case "data":
            data.append(value)
        case "retry":
            retry = Int(value)
        default:
            break
        }

        return nil
    }
}
```

### Device Info (`DeviceInfo.swift`)

```swift
import Foundation
import UIKit

/// Device information utilities
public enum DeviceInfo {
    /// Unique device identifier (persisted in Keychain)
    public static var deviceID: String {
        let service = Bundle.main.bundleIdentifier ?? "com.mizu.app"
        let account = "device_id"

        // Try to get existing ID
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
            kSecAttrAccount as String: account,
            kSecReturnData as String: true
        ]

        var result: AnyObject?
        let status = SecItemCopyMatching(query as CFDictionary, &result)

        if status == errSecSuccess,
           let data = result as? Data,
           let id = String(data: data, encoding: .utf8) {
            return id
        }

        // Generate new ID
        let newID = UUID().uuidString
        let data = newID.data(using: .utf8)!

        let addQuery: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
            kSecAttrAccount as String: account,
            kSecValueData as String: data,
            kSecAttrAccessible as String: kSecAttrAccessibleWhenUnlockedThisDeviceOnly
        ]

        SecItemAdd(addQuery as CFDictionary, nil)
        return newID
    }

    /// Device model identifier (e.g., "iPhone15,2")
    public static var model: String {
        var systemInfo = utsname()
        uname(&systemInfo)
        return withUnsafePointer(to: &systemInfo.machine) {
            $0.withMemoryRebound(to: CChar.self, capacity: 1) {
                String(validatingUTF8: $0) ?? "Unknown"
            }
        }
    }

    /// OS version string
    public static var osVersion: String {
        UIDevice.current.systemVersion
    }

    /// Device name (user-assigned)
    public static var name: String {
        UIDevice.current.name
    }

    /// Is running on simulator
    public static var isSimulator: Bool {
        #if targetEnvironment(simulator)
        return true
        #else
        return false
        #endif
    }
}

// MARK: - Bundle Extensions

extension Bundle {
    var appVersion: String {
        infoDictionary?["CFBundleShortVersionString"] as? String ?? "0.0.0"
    }

    var buildNumber: String {
        infoDictionary?["CFBundleVersion"] as? String ?? "0"
    }

    var displayName: String {
        infoDictionary?["CFBundleDisplayName"] as? String ??
        infoDictionary?["CFBundleName"] as? String ?? ""
    }
}
```

### Configuration (`Config.swift`)

```swift
import Foundation

/// Mizu runtime configuration
public struct MizuConfig {
    /// Base URL for API requests
    public var baseURL: URL

    /// Request timeout interval
    public var timeout: TimeInterval

    /// Enable debug logging
    public var debug: Bool

    /// Custom headers to include in all requests
    public var headers: [String: String]

    /// Token refresh configuration
    public var tokenRefresh: TokenRefreshConfig?

    /// Retry configuration
    public var retry: RetryConfig

    public init(
        baseURL: URL = URL(string: "http://localhost:3000")!,
        timeout: TimeInterval = 30,
        debug: Bool = false,
        headers: [String: String] = [:],
        tokenRefresh: TokenRefreshConfig? = nil,
        retry: RetryConfig = .default
    ) {
        self.baseURL = baseURL
        self.timeout = timeout
        self.debug = debug
        self.headers = headers
        self.tokenRefresh = tokenRefresh
        self.retry = retry
    }

    /// Load configuration from environment
    public static func fromEnvironment() -> MizuConfig {
        var config = MizuConfig()

        if let urlString = ProcessInfo.processInfo.environment["MIZU_BASE_URL"],
           let url = URL(string: urlString) {
            config.baseURL = url
        }

        if let debug = ProcessInfo.processInfo.environment["MIZU_DEBUG"] {
            config.debug = debug == "1" || debug.lowercased() == "true"
        }

        return config
    }
}

/// Token refresh configuration
public struct TokenRefreshConfig {
    /// Path for token refresh endpoint
    public var refreshPath: String

    /// Refresh token before expiration (seconds)
    public var refreshBeforeExpiry: TimeInterval

    public init(refreshPath: String = "/auth/refresh", refreshBeforeExpiry: TimeInterval = 60) {
        self.refreshPath = refreshPath
        self.refreshBeforeExpiry = refreshBeforeExpiry
    }
}

/// Retry configuration
public struct RetryConfig {
    /// Maximum number of retries
    public var maxRetries: Int

    /// Base delay between retries (exponential backoff)
    public var baseDelay: TimeInterval

    /// Maximum delay between retries
    public var maxDelay: TimeInterval

    /// HTTP status codes that should trigger a retry
    public var retryableStatuses: Set<Int>

    public static let `default` = RetryConfig(
        maxRetries: 3,
        baseDelay: 1.0,
        maxDelay: 30.0,
        retryableStatuses: [408, 429, 500, 502, 503, 504]
    )

    public static let none = RetryConfig(
        maxRetries: 0,
        baseDelay: 0,
        maxDelay: 0,
        retryableStatuses: []
    )
}
```

## App Templates

### SwiftUI App Entry (`{{.Name}}App.swift`)

```swift
import SwiftUI

@main
struct {{.Name}}App: App {
    @StateObject private var appState = AppState()

    init() {
        // Configure Mizu runtime
        let config = MizuConfig.fromEnvironment()
        MizuRuntime.shared.baseURL = config.baseURL
        MizuRuntime.shared.timeoutInterval = config.timeout

        if config.debug {
            (MizuRuntime.shared.transport as? URLSessionTransport)?
                .addInterceptor(LoggingInterceptor())
        }
    }

    var body: some Scene {
        WindowGroup {
            ContentView()
                .environmentObject(appState)
                .environmentObject(MizuRuntime.shared)
        }
    }
}
```

### Content View (`ContentView.swift`)

```swift
import SwiftUI

struct ContentView: View {
    @EnvironmentObject var runtime: MizuRuntime
    @EnvironmentObject var appState: AppState

    var body: some View {
        NavigationStack {
            if runtime.isAuthenticated {
                HomeView()
            } else {
                WelcomeView()
            }
        }
    }
}

#Preview {
    ContentView()
        .environmentObject(AppState())
        .environmentObject(MizuRuntime.shared)
}
```

### Home View (`Views/HomeView.swift`)

```swift
import SwiftUI

struct HomeView: View {
    @EnvironmentObject var runtime: MizuRuntime
    @State private var isLoading = false
    @State private var error: Error?

    var body: some View {
        VStack(spacing: 20) {
            Image(systemName: "checkmark.circle.fill")
                .font(.system(size: 60))
                .foregroundColor(.green)

            Text("Welcome to {{.Name}}")
                .font(.title)
                .fontWeight(.bold)

            Text("Connected to Mizu backend")
                .foregroundColor(.secondary)

            if isLoading {
                ProgressView()
            }

            Button("Sign Out") {
                signOut()
            }
            .buttonStyle(.bordered)
        }
        .padding()
        .navigationTitle("Home")
        .alert("Error", isPresented: .constant(error != nil)) {
            Button("OK") { error = nil }
        } message: {
            if let error = error {
                Text(error.localizedDescription)
            }
        }
    }

    private func signOut() {
        do {
            try runtime.tokenStore.clearToken()
        } catch {
            self.error = error
        }
    }
}
```

### App State (`Models/AppState.swift`)

```swift
import SwiftUI
import Combine

@MainActor
final class AppState: ObservableObject {
    @Published var isOnboarded: Bool
    @Published var selectedTab: Tab = .home

    enum Tab: String, CaseIterable {
        case home
        case profile
        case settings
    }

    init() {
        self.isOnboarded = UserDefaults.standard.bool(forKey: "isOnboarded")
    }

    func completeOnboarding() {
        isOnboarded = true
        UserDefaults.standard.set(true, forKey: "isOnboarded")
    }
}
```

## Generated SDK

The SDK is generated based on the Mizu contract. Example structure:

### Client (`SDK/Client.swift`)

```swift
import Foundation

/// Generated Mizu API client for {{.Name}}
@MainActor
public final class {{.Name}}Client {
    private let runtime: MizuRuntime

    public init(runtime: MizuRuntime = .shared) {
        self.runtime = runtime
    }

    // MARK: - Auth

    /// Sign in with credentials
    public func signIn(email: String, password: String) async throws -> AuthResponse {
        try await runtime.post("/auth/signin", body: SignInRequest(email: email, password: password))
    }

    /// Sign up with credentials
    public func signUp(email: String, password: String, name: String) async throws -> AuthResponse {
        try await runtime.post("/auth/signup", body: SignUpRequest(email: email, password: password, name: name))
    }

    /// Sign out
    public func signOut() async throws {
        try await runtime.delete("/auth/signout")
        try runtime.tokenStore.clearToken()
    }

    // MARK: - Users

    /// Get current user profile
    public func getCurrentUser() async throws -> User {
        try await runtime.get("/users/me")
    }

    /// Update current user profile
    public func updateCurrentUser(_ update: UserUpdate) async throws -> User {
        try await runtime.put("/users/me", body: update)
    }
}
```

### Types (`SDK/Types.swift`)

```swift
import Foundation

// MARK: - Auth Types

public struct SignInRequest: Encodable {
    public let email: String
    public let password: String
}

public struct SignUpRequest: Encodable {
    public let email: String
    public let password: String
    public let name: String
}

public struct AuthResponse: Decodable {
    public let user: User
    public let token: TokenResponse
}

public struct TokenResponse: Decodable {
    public let accessToken: String
    public let refreshToken: String?
    public let expiresIn: Int

    enum CodingKeys: String, CodingKey {
        case accessToken = "access_token"
        case refreshToken = "refresh_token"
        case expiresIn = "expires_in"
    }
}

// MARK: - User Types

public struct User: Codable, Identifiable {
    public let id: String
    public let email: String
    public let name: String
    public let avatarURL: String?
    public let createdAt: Date
    public let updatedAt: Date

    enum CodingKeys: String, CodingKey {
        case id, email, name
        case avatarURL = "avatar_url"
        case createdAt = "created_at"
        case updatedAt = "updated_at"
    }
}

public struct UserUpdate: Encodable {
    public var name: String?
    public var avatarURL: String?

    enum CodingKeys: String, CodingKey {
        case name
        case avatarURL = "avatar_url"
    }
}

// MARK: - Utility Types

/// Type-erased Codable for dynamic values
public struct AnyCodable: Codable {
    public let value: Any

    public init(_ value: Any) {
        self.value = value
    }

    public init(from decoder: Decoder) throws {
        let container = try decoder.singleValueContainer()

        if container.decodeNil() {
            self.value = NSNull()
        } else if let bool = try? container.decode(Bool.self) {
            self.value = bool
        } else if let int = try? container.decode(Int.self) {
            self.value = int
        } else if let double = try? container.decode(Double.self) {
            self.value = double
        } else if let string = try? container.decode(String.self) {
            self.value = string
        } else if let array = try? container.decode([AnyCodable].self) {
            self.value = array.map { $0.value }
        } else if let dict = try? container.decode([String: AnyCodable].self) {
            self.value = dict.mapValues { $0.value }
        } else {
            throw DecodingError.dataCorruptedError(in: container, debugDescription: "Unable to decode value")
        }
    }

    public func encode(to encoder: Encoder) throws {
        var container = encoder.singleValueContainer()

        switch value {
        case is NSNull:
            try container.encodeNil()
        case let bool as Bool:
            try container.encode(bool)
        case let int as Int:
            try container.encode(int)
        case let double as Double:
            try container.encode(double)
        case let string as String:
            try container.encode(string)
        case let array as [Any]:
            try container.encode(array.map { AnyCodable($0) })
        case let dict as [String: Any]:
            try container.encode(dict.mapValues { AnyCodable($0) })
        default:
            throw EncodingError.invalidValue(value, .init(codingPath: [], debugDescription: "Unable to encode value"))
        }
    }
}
```

## Project Configuration

### Info.plist

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>CFBundleDevelopmentRegion</key>
    <string>$(DEVELOPMENT_LANGUAGE)</string>
    <key>CFBundleExecutable</key>
    <string>$(EXECUTABLE_NAME)</string>
    <key>CFBundleIdentifier</key>
    <string>$(PRODUCT_BUNDLE_IDENTIFIER)</string>
    <key>CFBundleInfoDictionaryVersion</key>
    <string>6.0</string>
    <key>CFBundleName</key>
    <string>$(PRODUCT_NAME)</string>
    <key>CFBundlePackageType</key>
    <string>$(PRODUCT_BUNDLE_PACKAGE_TYPE)</string>
    <key>CFBundleShortVersionString</key>
    <string>1.0</string>
    <key>CFBundleVersion</key>
    <string>1</string>
    <key>LSRequiresIPhoneOS</key>
    <true/>
    <key>UIApplicationSceneManifest</key>
    <dict>
        <key>UIApplicationSupportsMultipleScenes</key>
        <true/>
    </dict>
    <key>UILaunchScreen</key>
    <dict/>
    <key>UIRequiredDeviceCapabilities</key>
    <array>
        <string>armv7</string>
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

### Package.swift (Optional SPM)

```swift
// swift-tools-version:5.9
import PackageDescription

let package = Package(
    name: "{{.Name}}",
    platforms: [
        .iOS(.v16)
    ],
    products: [
        .library(name: "{{.Name}}SDK", targets: ["{{.Name}}SDK"]),
        .library(name: "MizuRuntime", targets: ["MizuRuntime"])
    ],
    dependencies: [],
    targets: [
        .target(
            name: "MizuRuntime",
            path: "{{.Name}}/Runtime"
        ),
        .target(
            name: "{{.Name}}SDK",
            dependencies: ["MizuRuntime"],
            path: "{{.Name}}/SDK"
        ),
        .testTarget(
            name: "{{.Name}}Tests",
            dependencies: ["{{.Name}}SDK", "MizuRuntime"],
            path: "{{.Name}}Tests"
        )
    ]
)
```

## UI Variants

### SwiftUI (Default)
- `@main` entry point with `App` protocol
- `@StateObject` for state management
- `@EnvironmentObject` for dependency injection
- Modern SwiftUI views and modifiers

### UIKit
- `UIApplicationDelegate` entry point
- `UINavigationController` for navigation
- `UITableView`/`UICollectionView` for lists
- Programmatic UI (no storyboards)

### Hybrid
- SwiftUI `App` with UIKit integration via `UIViewControllerRepresentable`
- Shared navigation coordination
- Mix of SwiftUI and UIKit views

## Testing

### Unit Tests

```swift
import XCTest
@testable import {{.Name}}

final class RuntimeTests: XCTestCase {
    var runtime: MizuRuntime!

    override func setUp() {
        runtime = MizuRuntime(baseURL: URL(string: "http://localhost:3000")!)
    }

    func testDeviceID() {
        let id1 = DeviceInfo.deviceID
        let id2 = DeviceInfo.deviceID
        XCTAssertEqual(id1, id2, "Device ID should be stable")
    }

    func testTokenStore() throws {
        let store = KeychainTokenStore(service: "test", account: "test_token")

        let token = AuthToken(accessToken: "test123", refreshToken: "refresh456")
        try store.setToken(token)

        let retrieved = store.getToken()
        XCTAssertEqual(retrieved?.accessToken, "test123")

        try store.clearToken()
        XCTAssertNil(store.getToken())
    }
}
```

### UI Tests

```swift
import XCTest

final class {{.Name}}UITests: XCTestCase {
    var app: XCUIApplication!

    override func setUp() {
        continueAfterFailure = false
        app = XCUIApplication()
        app.launch()
    }

    func testWelcomeScreen() {
        XCTAssertTrue(app.staticTexts["Welcome to {{.Name}}"].exists)
    }
}
```

## References

- [Apple Swift Documentation](https://swift.org/documentation/)
- [SwiftUI Tutorials](https://developer.apple.com/tutorials/swiftui)
- [URLSession Documentation](https://developer.apple.com/documentation/foundation/urlsession)
- [Keychain Services](https://developer.apple.com/documentation/security/keychain_services)
- [Mobile Package Spec](./0095_mobile.md)
