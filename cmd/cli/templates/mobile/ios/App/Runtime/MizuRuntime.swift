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

    /// Performs a PATCH request
    public func patch<T: Decodable, B: Encodable>(
        _ path: String,
        body: B,
        headers: [String: String]? = nil
    ) async throws -> T {
        try await request(method: .patch, path: path, body: body, headers: headers)
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
            // swiftlint:disable:next force_cast
            return EmptyResponse() as! T
        }

        let decoder = JSONDecoder()
        decoder.dateDecodingStrategy = .iso8601
        return try decoder.decode(T.self, from: response.data)
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
