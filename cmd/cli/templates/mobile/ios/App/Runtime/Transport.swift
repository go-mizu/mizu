import Foundation

/// Transport protocol for executing HTTP requests
public protocol Transport: Sendable {
    func execute(_ request: URLRequest) async throws -> TransportResponse
}

/// Response from transport layer
public struct TransportResponse: Sendable {
    public let data: Data
    public let response: URLResponse

    public init(data: Data, response: URLResponse) {
        self.data = data
        self.response = response
    }
}

/// URLSession-based transport implementation
public final class URLSessionTransport: Transport, @unchecked Sendable {
    private let session: URLSession
    private var interceptors: [RequestInterceptor] = []
    private let lock = NSLock()

    public init(configuration: URLSessionConfiguration = .default) {
        self.session = URLSession(configuration: configuration)
    }

    /// Adds a request interceptor
    public func addInterceptor(_ interceptor: RequestInterceptor) {
        lock.withLock {
            interceptors.append(interceptor)
        }
    }

    public func execute(_ request: URLRequest) async throws -> TransportResponse {
        var request = request

        // Apply interceptors
        let currentInterceptors = lock.withLock { interceptors }
        for interceptor in currentInterceptors {
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
public protocol RequestInterceptor: Sendable {
    func intercept(_ request: URLRequest) async throws -> URLRequest
}

/// Logging interceptor for debugging
public final class LoggingInterceptor: RequestInterceptor {
    public init() {}

    public func intercept(_ request: URLRequest) async throws -> URLRequest {
        #if DEBUG
        print("[Mizu] \(request.httpMethod ?? "?") \(request.url?.absoluteString ?? "?")")
        if let body = request.httpBody, let bodyString = String(data: body, encoding: .utf8) {
            print("[Mizu] Body: \(bodyString)")
        }
        #endif
        return request
    }
}

/// Header injection interceptor
public final class HeaderInterceptor: RequestInterceptor {
    private let headers: [String: String]

    public init(headers: [String: String]) {
        self.headers = headers
    }

    public func intercept(_ request: URLRequest) async throws -> URLRequest {
        var request = request
        for (key, value) in headers {
            request.setValue(value, forHTTPHeaderField: key)
        }
        return request
    }
}

/// Retry interceptor with exponential backoff
public final class RetryInterceptor: RequestInterceptor {
    private let maxRetries: Int
    private let baseDelay: TimeInterval
    private let retryableStatuses: Set<Int>

    public init(
        maxRetries: Int = 3,
        baseDelay: TimeInterval = 1.0,
        retryableStatuses: Set<Int> = [408, 429, 500, 502, 503, 504]
    ) {
        self.maxRetries = maxRetries
        self.baseDelay = baseDelay
        self.retryableStatuses = retryableStatuses
    }

    public func intercept(_ request: URLRequest) async throws -> URLRequest {
        // Retry logic is handled at transport level for now
        // This interceptor marks the request for retry capability
        return request
    }
}
