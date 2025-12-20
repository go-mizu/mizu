import SwiftUI
import Combine

/// Application state management
@MainActor
final class AppState: ObservableObject {
    /// Whether the user has completed onboarding
    @Published var isOnboarded: Bool

    /// Current selected tab
    @Published var selectedTab: Tab = .home

    /// Available tabs
    enum Tab: String, CaseIterable, Identifiable {
        case home
        case explore
        case profile
        case settings

        var id: String { rawValue }

        var title: String {
            switch self {
            case .home: return "Home"
            case .explore: return "Explore"
            case .profile: return "Profile"
            case .settings: return "Settings"
            }
        }

        var icon: String {
            switch self {
            case .home: return "house"
            case .explore: return "magnifyingglass"
            case .profile: return "person"
            case .settings: return "gearshape"
            }
        }

        var selectedIcon: String {
            switch self {
            case .home: return "house.fill"
            case .explore: return "magnifyingglass"
            case .profile: return "person.fill"
            case .settings: return "gearshape.fill"
            }
        }
    }

    init() {
        self.isOnboarded = UserDefaults.standard.bool(forKey: "isOnboarded")
    }

    /// Marks onboarding as complete
    func completeOnboarding() {
        isOnboarded = true
        UserDefaults.standard.set(true, forKey: "isOnboarded")
    }

    /// Resets onboarding state
    func resetOnboarding() {
        isOnboarded = false
        UserDefaults.standard.set(false, forKey: "isOnboarded")
    }
}
