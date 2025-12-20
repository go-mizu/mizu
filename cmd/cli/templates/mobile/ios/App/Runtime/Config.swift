import Foundation

/// Mizu runtime configuration
public struct MizuConfig: Sendable {
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

    /// Default configuration
    public static let `default` = MizuConfig()

    /// Load configuration from environment
    public static func fromEnvironment() -> MizuConfig {
        var config = MizuConfig()

        if let urlString = ProcessInfo.processInfo.environment["MIZU_BASE_URL"],
           let url = URL(string: urlString) {
            config.baseURL = url
        }

        if let timeoutString = ProcessInfo.processInfo.environment["MIZU_TIMEOUT"],
           let timeout = TimeInterval(timeoutString) {
            config.timeout = timeout
        }

        if let debug = ProcessInfo.processInfo.environment["MIZU_DEBUG"] {
            config.debug = debug == "1" || debug.lowercased() == "true"
        }

        return config
    }

    /// Load configuration from Info.plist
    public static func fromInfoPlist() -> MizuConfig {
        var config = MizuConfig()

        if let urlString = Bundle.main.object(forInfoDictionaryKey: "MizuBaseURL") as? String,
           let url = URL(string: urlString) {
            config.baseURL = url
        }

        if let timeout = Bundle.main.object(forInfoDictionaryKey: "MizuTimeout") as? TimeInterval {
            config.timeout = timeout
        }

        if let debug = Bundle.main.object(forInfoDictionaryKey: "MizuDebug") as? Bool {
            config.debug = debug
        }

        return config
    }
}

/// Token refresh configuration
public struct TokenRefreshConfig: Sendable {
    /// Path for token refresh endpoint
    public var refreshPath: String

    /// Refresh token before expiration (seconds)
    public var refreshBeforeExpiry: TimeInterval

    /// Whether to automatically refresh tokens
    public var autoRefresh: Bool

    public init(
        refreshPath: String = "/auth/refresh",
        refreshBeforeExpiry: TimeInterval = 60,
        autoRefresh: Bool = true
    ) {
        self.refreshPath = refreshPath
        self.refreshBeforeExpiry = refreshBeforeExpiry
        self.autoRefresh = autoRefresh
    }
}

/// Retry configuration
public struct RetryConfig: Sendable {
    /// Maximum number of retries
    public var maxRetries: Int

    /// Base delay between retries (exponential backoff)
    public var baseDelay: TimeInterval

    /// Maximum delay between retries
    public var maxDelay: TimeInterval

    /// HTTP status codes that should trigger a retry
    public var retryableStatuses: Set<Int>

    public init(
        maxRetries: Int = 3,
        baseDelay: TimeInterval = 1.0,
        maxDelay: TimeInterval = 30.0,
        retryableStatuses: Set<Int> = [408, 429, 500, 502, 503, 504]
    ) {
        self.maxRetries = maxRetries
        self.baseDelay = baseDelay
        self.maxDelay = maxDelay
        self.retryableStatuses = retryableStatuses
    }

    /// Default retry configuration
    public static let `default` = RetryConfig()

    /// No retry configuration
    public static let none = RetryConfig(
        maxRetries: 0,
        baseDelay: 0,
        maxDelay: 0,
        retryableStatuses: []
    )

    /// Aggressive retry configuration
    public static let aggressive = RetryConfig(
        maxRetries: 5,
        baseDelay: 0.5,
        maxDelay: 60.0,
        retryableStatuses: [408, 425, 429, 500, 502, 503, 504]
    )
}

/// API version configuration
public struct APIVersionConfig: Sendable {
    /// API version to use
    public var version: String

    /// Header name for version
    public var headerName: String

    public init(version: String = "v1", headerName: String = "X-API-Version") {
        self.version = version
        self.headerName = headerName
    }
}
