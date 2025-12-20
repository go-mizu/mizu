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
            throw MizuError.decoding(NSError(domain: "LiveConnection", code: 1, userInfo: [
                NSLocalizedDescriptionKey: "Failed to convert event data to UTF-8"
            ]))
        }
        let decoder = JSONDecoder()
        decoder.dateDecodingStrategy = .iso8601
        return try decoder.decode(type, from: data)
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

    /// Returns the number of active connections
    public var activeConnectionCount: Int {
        lock.withLock { activeTasks.count }
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

        // Add mobile headers
        request.setValue(DeviceInfo.deviceID, forHTTPHeaderField: "X-Device-ID")
        request.setValue(Bundle.main.appVersion, forHTTPHeaderField: "X-App-Version")
        request.setValue("ios", forHTTPHeaderField: "X-Platform")

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
                let statusCode = (response as? HTTPURLResponse)?.statusCode ?? 0
                continuation.finish(throwing: MizuError.http(statusCode: statusCode, data: Data()))
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
            } else {
                continuation.finish()
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
                event: self.event,
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

        let colonIndex = line.firstIndex(of: ":")
        let field: String
        let value: String

        if let idx = colonIndex {
            field = String(line[..<idx])
            let valueStart = line.index(after: idx)
            value = String(line[valueStart...]).trimmingCharacters(in: .whitespaces)
        } else {
            field = line
            value = ""
        }

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

/// Live connection state
public enum LiveConnectionState: Sendable {
    case disconnected
    case connecting
    case connected
    case reconnecting(attempt: Int)
    case error(Error)
}
