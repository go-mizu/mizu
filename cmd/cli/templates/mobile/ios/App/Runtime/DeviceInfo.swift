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
        guard let data = newID.data(using: .utf8) else {
            return newID
        }

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

    /// Human-readable device model name
    public static var modelName: String {
        let identifier = model

        // iPhone models
        let modelMappings: [String: String] = [
            "iPhone14,2": "iPhone 13 Pro",
            "iPhone14,3": "iPhone 13 Pro Max",
            "iPhone14,4": "iPhone 13 mini",
            "iPhone14,5": "iPhone 13",
            "iPhone14,6": "iPhone SE (3rd generation)",
            "iPhone14,7": "iPhone 14",
            "iPhone14,8": "iPhone 14 Plus",
            "iPhone15,2": "iPhone 14 Pro",
            "iPhone15,3": "iPhone 14 Pro Max",
            "iPhone15,4": "iPhone 15",
            "iPhone15,5": "iPhone 15 Plus",
            "iPhone16,1": "iPhone 15 Pro",
            "iPhone16,2": "iPhone 15 Pro Max",
            // Simulators
            "i386": "Simulator",
            "x86_64": "Simulator",
            "arm64": "Simulator"
        ]

        return modelMappings[identifier] ?? identifier
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

    /// Current device orientation
    public static var orientation: UIDeviceOrientation {
        UIDevice.current.orientation
    }

    /// Device battery level (0.0 - 1.0)
    public static var batteryLevel: Float {
        UIDevice.current.isBatteryMonitoringEnabled = true
        return UIDevice.current.batteryLevel
    }

    /// Device battery state
    public static var batteryState: UIDevice.BatteryState {
        UIDevice.current.isBatteryMonitoringEnabled = true
        return UIDevice.current.batteryState
    }

    /// Available disk space in bytes
    public static var availableDiskSpace: Int64? {
        let fileURL = URL(fileURLWithPath: NSHomeDirectory())
        do {
            let values = try fileURL.resourceValues(forKeys: [.volumeAvailableCapacityForImportantUsageKey])
            return values.volumeAvailableCapacityForImportantUsage
        } catch {
            return nil
        }
    }

    /// Total disk space in bytes
    public static var totalDiskSpace: Int64? {
        let fileURL = URL(fileURLWithPath: NSHomeDirectory())
        do {
            let values = try fileURL.resourceValues(forKeys: [.volumeTotalCapacityKey])
            return values.volumeTotalCapacity.map { Int64($0) }
        } catch {
            return nil
        }
    }
}

// MARK: - Bundle Extensions

extension Bundle {
    /// App version string (e.g., "1.0.0")
    var appVersion: String {
        infoDictionary?["CFBundleShortVersionString"] as? String ?? "0.0.0"
    }

    /// Build number string (e.g., "1")
    var buildNumber: String {
        infoDictionary?["CFBundleVersion"] as? String ?? "0"
    }

    /// Full version string (e.g., "1.0.0 (1)")
    var fullVersion: String {
        "\(appVersion) (\(buildNumber))"
    }

    /// Display name
    var displayName: String {
        infoDictionary?["CFBundleDisplayName"] as? String ??
        infoDictionary?["CFBundleName"] as? String ?? ""
    }

    /// Bundle identifier
    var appBundleIdentifier: String {
        bundleIdentifier ?? "unknown"
    }
}
