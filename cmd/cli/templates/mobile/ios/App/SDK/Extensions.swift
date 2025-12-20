import Foundation

// MARK: - Date Extensions

extension Date {
    /// Returns a relative time string (e.g., "2 hours ago")
    var relativeString: String {
        let formatter = RelativeDateTimeFormatter()
        formatter.unitsStyle = .full
        return formatter.localizedString(for: self, relativeTo: Date())
    }

    /// Returns a formatted date string
    func formatted(style: DateFormatter.Style = .medium) -> String {
        let formatter = DateFormatter()
        formatter.dateStyle = style
        formatter.timeStyle = .none
        return formatter.string(from: self)
    }

    /// Returns a formatted date and time string
    func formattedWithTime(dateStyle: DateFormatter.Style = .medium, timeStyle: DateFormatter.Style = .short) -> String {
        let formatter = DateFormatter()
        formatter.dateStyle = dateStyle
        formatter.timeStyle = timeStyle
        return formatter.string(from: self)
    }
}

// MARK: - String Extensions

extension String {
    /// Returns true if the string is a valid email format
    var isValidEmail: Bool {
        let emailRegex = #"^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$"#
        return range(of: emailRegex, options: .regularExpression) != nil
    }

    /// Returns the string with leading and trailing whitespace removed
    var trimmed: String {
        trimmingCharacters(in: .whitespacesAndNewlines)
    }

    /// Returns nil if the string is empty after trimming
    var nilIfEmpty: String? {
        let trimmed = self.trimmed
        return trimmed.isEmpty ? nil : trimmed
    }
}

// MARK: - URL Extensions

extension URL {
    /// Creates a URL by appending query parameters
    func appending(queryItems: [String: String]) -> URL {
        guard var components = URLComponents(url: self, resolvingAgainstBaseURL: true) else {
            return self
        }

        var existingItems = components.queryItems ?? []
        existingItems.append(contentsOf: queryItems.map { URLQueryItem(name: $0.key, value: $0.value) })
        components.queryItems = existingItems

        return components.url ?? self
    }
}

// MARK: - Data Extensions

extension Data {
    /// Returns a pretty-printed JSON string
    var prettyPrintedJSON: String? {
        guard let object = try? JSONSerialization.jsonObject(with: self),
              let data = try? JSONSerialization.data(withJSONObject: object, options: .prettyPrinted) else {
            return nil
        }
        return String(data: data, encoding: .utf8)
    }
}

// MARK: - Optional Extensions

extension Optional where Wrapped == String {
    /// Returns true if the optional is nil or empty
    var isNilOrEmpty: Bool {
        switch self {
        case .none:
            return true
        case .some(let value):
            return value.trimmed.isEmpty
        }
    }
}

// MARK: - Collection Extensions

extension Collection {
    /// Returns the element at the specified index if it exists, otherwise nil
    subscript(safe index: Index) -> Element? {
        indices.contains(index) ? self[index] : nil
    }
}

// MARK: - Result Extensions

extension Result where Failure == Error {
    /// Creates a Result from an async throwing expression
    static func from(_ body: () async throws -> Success) async -> Result<Success, Error> {
        do {
            return .success(try await body())
        } catch {
            return .failure(error)
        }
    }
}
