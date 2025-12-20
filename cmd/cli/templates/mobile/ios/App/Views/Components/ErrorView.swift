import SwiftUI

/// A reusable error view
struct ErrorView: View {
    let error: Error
    var retryAction: (() -> Void)?

    var body: some View {
        ContentUnavailableView {
            Label("Error", systemImage: "exclamationmark.triangle")
        } description: {
            Text(error.localizedDescription)
        } actions: {
            if let retryAction = retryAction {
                Button("Try Again", action: retryAction)
                    .buttonStyle(.bordered)
            }
        }
    }
}

/// An empty state view
struct EmptyStateView: View {
    let title: String
    let description: String?
    let icon: String
    var action: (() -> Void)?
    var actionTitle: String?

    var body: some View {
        ContentUnavailableView {
            Label(title, systemImage: icon)
        } description: {
            if let description = description {
                Text(description)
            }
        } actions: {
            if let action = action, let actionTitle = actionTitle {
                Button(actionTitle, action: action)
                    .buttonStyle(.bordered)
            }
        }
    }
}

#Preview("Error View") {
    ErrorView(
        error: MizuError.network(URLError(.notConnectedToInternet)),
        retryAction: {}
    )
}

#Preview("Empty State") {
    EmptyStateView(
        title: "No Items",
        description: "You don't have any items yet",
        icon: "tray",
        action: {},
        actionTitle: "Add Item"
    )
}
