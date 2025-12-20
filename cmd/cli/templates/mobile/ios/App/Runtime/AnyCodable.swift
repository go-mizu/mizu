import Foundation

/// Type-erased Codable for dynamic values
public struct AnyCodable: Codable, Sendable {
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
            throw DecodingError.dataCorruptedError(
                in: container,
                debugDescription: "Unable to decode value"
            )
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
            throw EncodingError.invalidValue(
                value,
                .init(codingPath: [], debugDescription: "Unable to encode value of type \(type(of: value))")
            )
        }
    }
}

// MARK: - Convenience Extensions

extension AnyCodable: ExpressibleByNilLiteral {
    public init(nilLiteral: ()) {
        self.value = NSNull()
    }
}

extension AnyCodable: ExpressibleByBooleanLiteral {
    public init(booleanLiteral value: Bool) {
        self.value = value
    }
}

extension AnyCodable: ExpressibleByIntegerLiteral {
    public init(integerLiteral value: Int) {
        self.value = value
    }
}

extension AnyCodable: ExpressibleByFloatLiteral {
    public init(floatLiteral value: Double) {
        self.value = value
    }
}

extension AnyCodable: ExpressibleByStringLiteral {
    public init(stringLiteral value: String) {
        self.value = value
    }
}

extension AnyCodable: ExpressibleByArrayLiteral {
    public init(arrayLiteral elements: Any...) {
        self.value = elements
    }
}

extension AnyCodable: ExpressibleByDictionaryLiteral {
    public init(dictionaryLiteral elements: (String, Any)...) {
        self.value = Dictionary(uniqueKeysWithValues: elements)
    }
}

// MARK: - Value Access

extension AnyCodable {
    /// Returns the value as a Bool if possible
    public var boolValue: Bool? {
        value as? Bool
    }

    /// Returns the value as an Int if possible
    public var intValue: Int? {
        value as? Int
    }

    /// Returns the value as a Double if possible
    public var doubleValue: Double? {
        value as? Double
    }

    /// Returns the value as a String if possible
    public var stringValue: String? {
        value as? String
    }

    /// Returns the value as an array if possible
    public var arrayValue: [Any]? {
        value as? [Any]
    }

    /// Returns the value as a dictionary if possible
    public var dictionaryValue: [String: Any]? {
        value as? [String: Any]
    }

    /// Returns true if the value is nil
    public var isNil: Bool {
        value is NSNull
    }
}

// MARK: - Equatable (limited)

extension AnyCodable: Equatable {
    public static func == (lhs: AnyCodable, rhs: AnyCodable) -> Bool {
        switch (lhs.value, rhs.value) {
        case is (NSNull, NSNull):
            return true
        case let (lhs as Bool, rhs as Bool):
            return lhs == rhs
        case let (lhs as Int, rhs as Int):
            return lhs == rhs
        case let (lhs as Double, rhs as Double):
            return lhs == rhs
        case let (lhs as String, rhs as String):
            return lhs == rhs
        default:
            return false
        }
    }
}

// MARK: - CustomStringConvertible

extension AnyCodable: CustomStringConvertible {
    public var description: String {
        switch value {
        case is NSNull:
            return "nil"
        case let bool as Bool:
            return bool.description
        case let int as Int:
            return int.description
        case let double as Double:
            return double.description
        case let string as String:
            return "\"\(string)\""
        case let array as [Any]:
            return "[\(array.count) items]"
        case let dict as [String: Any]:
            return "{\(dict.count) keys}"
        default:
            return String(describing: value)
        }
    }
}
