import SwiftUI

/// A reusable loading indicator view
struct LoadingView: View {
    var message: String?

    var body: some View {
        VStack(spacing: 16) {
            ProgressView()
                .scaleEffect(1.5)

            if let message = message {
                Text(message)
                    .foregroundColor(.secondary)
                    .font(.subheadline)
            }
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
        .background(Color(.systemBackground))
    }
}

/// A loading button that shows a spinner when loading
struct LoadingButton: View {
    let title: String
    let isLoading: Bool
    let action: () -> Void

    var body: some View {
        Button(action: action) {
            if isLoading {
                ProgressView()
                    .tint(.white)
            } else {
                Text(title)
            }
        }
        .disabled(isLoading)
    }
}

/// An overlay modifier for loading states
struct LoadingOverlay: ViewModifier {
    let isLoading: Bool
    var message: String?

    func body(content: Content) -> some View {
        ZStack {
            content
                .disabled(isLoading)
                .blur(radius: isLoading ? 2 : 0)

            if isLoading {
                LoadingView(message: message)
                    .background(.ultraThinMaterial)
            }
        }
    }
}

extension View {
    /// Adds a loading overlay to the view
    func loading(_ isLoading: Bool, message: String? = nil) -> some View {
        modifier(LoadingOverlay(isLoading: isLoading, message: message))
    }
}

#Preview {
    VStack {
        LoadingView(message: "Loading...")
    }
}
