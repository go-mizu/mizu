import Foundation
import Security

/// Stored authentication token
public struct AuthToken: Codable, Sendable {
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
public protocol TokenStore: Sendable {
    func getToken() -> AuthToken?
    func setToken(_ token: AuthToken) throws
    func clearToken() throws
    func onTokenChange(_ callback: @escaping @Sendable (AuthToken?) -> Void)
}

/// Keychain-backed token storage
public final class KeychainTokenStore: TokenStore, @unchecked Sendable {
    private let service: String
    private let account: String
    private var observers: [@Sendable (AuthToken?) -> Void] = []
    private let lock = NSLock()

    public init(
        service: String = Bundle.main.bundleIdentifier ?? "com.mizu.app",
        account: String = "auth_token"
    ) {
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

    public func onTokenChange(_ callback: @escaping @Sendable (AuthToken?) -> Void) {
        lock.withLock {
            observers.append(callback)
        }
    }

    private func notifyObservers(_ token: AuthToken?) {
        let currentObservers = lock.withLock { observers }
        currentObservers.forEach { $0(token) }
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

/// In-memory token store for testing
public final class InMemoryTokenStore: TokenStore, @unchecked Sendable {
    private var token: AuthToken?
    private var observers: [@Sendable (AuthToken?) -> Void] = []
    private let lock = NSLock()

    public init() {}

    public func getToken() -> AuthToken? {
        lock.withLock { token }
    }

    public func setToken(_ token: AuthToken) throws {
        lock.withLock {
            self.token = token
        }
        notifyObservers(token)
    }

    public func clearToken() throws {
        lock.withLock {
            self.token = nil
        }
        notifyObservers(nil)
    }

    public func onTokenChange(_ callback: @escaping @Sendable (AuthToken?) -> Void) {
        lock.withLock {
            observers.append(callback)
        }
    }

    private func notifyObservers(_ token: AuthToken?) {
        let currentObservers = lock.withLock { observers }
        currentObservers.forEach { $0(token) }
    }
}
