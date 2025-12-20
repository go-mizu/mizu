# Mobile Template Implementation Plan

**Status:** In Progress
**Author:** Mizu Team
**Created:** 2025-12-20
**Spec Reference:** [0096_mobile_template.md](./0096_mobile_template.md)

## Overview

This document details the implementation plan for the `mobile:*` template matrix defined in the mobile template spec. The implementation follows the same patterns as `frontend:*` templates.

## Architecture

### Template Structure

```
cmd/cli/templates/
├── mobile/
│   ├── template.json           # Parent manifest with subTemplates
│   ├── _common/                # Shared files across all mobile templates
│   │   └── (none initially - mobile templates are too different)
│   ├── ios/
│   │   ├── template.json
│   │   ├── {{.Name}}.xcodeproj/
│   │   ├── {{.Name}}/
│   │   │   ├── App/
│   │   │   ├── SDK/            # Generated Mizu SDK
│   │   │   ├── Runtime/        # MizuMobileRuntime
│   │   │   └── Resources/
│   │   └── Package.swift
│   ├── android/
│   │   ├── template.json
│   │   ├── app/
│   │   ├── sdk/               # Generated Kotlin SDK
│   │   └── build.gradle.kts
│   ├── flutter/
│   │   ├── template.json
│   │   ├── lib/
│   │   │   ├── sdk/           # Generated Dart SDK
│   │   │   └── runtime/
│   │   └── pubspec.yaml
│   ├── reactnative/
│   │   ├── template.json
│   │   ├── src/
│   │   │   ├── sdk/           # Generated TS SDK
│   │   │   └── runtime/
│   │   └── package.json
│   └── ... (other templates)
```

### CLI Integration

Templates are invoked via:
```bash
mizu new ./myapp --template mobile:ios
mizu new ./myapp --template mobile:android
mizu new ./myapp --template mobile:flutter
mizu new ./myapp --template mobile:reactnative
```

With variants:
```bash
mizu new ./myapp --template mobile:ios --var ui=swiftui
mizu new ./myapp --template mobile:ios --var ui=uikit
mizu new ./myapp --template mobile:android --var ui=compose
mizu new ./myapp --template mobile:android --var ui=views
```

## Implementation Phases

### Phase 1: iOS Template (Priority)

**Files to Create:**

| Path | Purpose |
|------|---------|
| `mobile/template.json` | Parent manifest with subTemplates |
| `mobile/ios/template.json` | iOS template manifest |
| `mobile/ios/{{.Name}}.xcodeproj/project.pbxproj.tmpl` | Xcode project |
| `mobile/ios/{{.Name}}/App/{{.Name}}App.swift.tmpl` | SwiftUI app entry |
| `mobile/ios/{{.Name}}/App/ContentView.swift.tmpl` | Main content view |
| `mobile/ios/{{.Name}}/Runtime/MizuRuntime.swift` | Core runtime |
| `mobile/ios/{{.Name}}/Runtime/Transport.swift` | URLSession transport |
| `mobile/ios/{{.Name}}/Runtime/TokenStore.swift` | Keychain storage |
| `mobile/ios/{{.Name}}/Runtime/Live.swift` | SSE/streaming support |
| `mobile/ios/{{.Name}}/SDK/Client.swift.tmpl` | Generated API client |
| `mobile/ios/{{.Name}}/SDK/Types.swift.tmpl` | Generated types |
| `mobile/ios/Package.swift.tmpl` | Swift Package Manager |
| `mobile/ios/{{.Name}}/Resources/Assets.xcassets/` | Asset catalog |
| `mobile/ios/{{.Name}}/Resources/Info.plist.tmpl` | App configuration |

**MizuMobileRuntime Components:**

1. **Transport Layer** (`Transport.swift`)
   - URLSession-based HTTP client
   - Async/await support
   - Request/response interceptors
   - Automatic retry logic
   - Timeout configuration

2. **Token Store** (`TokenStore.swift`)
   - Keychain-backed secure storage
   - Token refresh handling
   - Automatic auth header injection

3. **Live Streaming** (`Live.swift`)
   - SSE (Server-Sent Events) support
   - AsyncSequence-based API
   - Automatic reconnection
   - Backoff strategies

4. **Sync Support** (`Sync.swift`)
   - Delta sync with tokens
   - Conflict resolution
   - Offline queue

### Phase 2: Android Template

**Files to Create:**

| Path | Purpose |
|------|---------|
| `mobile/android/template.json` | Android template manifest |
| `mobile/android/app/build.gradle.kts.tmpl` | App module build |
| `mobile/android/app/src/main/kotlin/.../MainActivity.kt.tmpl` | Main activity |
| `mobile/android/app/src/main/kotlin/.../MizuApp.kt.tmpl` | Application class |
| `mobile/android/runtime/MizuRuntime.kt` | Core runtime |
| `mobile/android/runtime/Transport.kt` | OkHttp transport |
| `mobile/android/runtime/TokenStore.kt` | EncryptedSharedPrefs storage |
| `mobile/android/runtime/Live.kt` | Flow-based streaming |
| `mobile/android/sdk/Client.kt.tmpl` | Generated API client |
| `mobile/android/sdk/Types.kt.tmpl` | Generated types |
| `mobile/android/build.gradle.kts.tmpl` | Root build file |
| `mobile/android/settings.gradle.kts.tmpl` | Settings |

**mizu-mobile-runtime Components:**

1. **Transport Layer** (`Transport.kt`)
   - OkHttp-based HTTP client
   - Coroutine support
   - Interceptor chain
   - Certificate pinning

2. **Token Store** (`TokenStore.kt`)
   - EncryptedSharedPreferences
   - Biometric authentication support
   - Token refresh flow

3. **Live Streaming** (`Live.kt`)
   - Flow-based SSE support
   - StateFlow for state management
   - Automatic reconnection

### Phase 3: Flutter Template

**Files to Create:**

| Path | Purpose |
|------|---------|
| `mobile/flutter/template.json` | Flutter template manifest |
| `mobile/flutter/pubspec.yaml.tmpl` | Dependencies |
| `mobile/flutter/lib/main.dart.tmpl` | App entry |
| `mobile/flutter/lib/runtime/mizu_runtime.dart` | Core runtime |
| `mobile/flutter/lib/runtime/transport.dart` | HTTP transport |
| `mobile/flutter/lib/runtime/token_store.dart` | Secure storage |
| `mobile/flutter/lib/runtime/live.dart` | Stream support |
| `mobile/flutter/lib/sdk/client.dart.tmpl` | Generated API client |
| `mobile/flutter/lib/sdk/types.dart.tmpl` | Generated types |

### Phase 4: React Native Template

**Files to Create:**

| Path | Purpose |
|------|---------|
| `mobile/reactnative/template.json` | RN template manifest |
| `mobile/reactnative/package.json.tmpl` | Dependencies |
| `mobile/reactnative/src/runtime/MizuRuntime.ts` | Core runtime |
| `mobile/reactnative/src/runtime/transport.ts` | Fetch transport |
| `mobile/reactnative/src/runtime/tokenStore.ts` | SecureStore |
| `mobile/reactnative/src/runtime/live.ts` | EventSource |
| `mobile/reactnative/src/sdk/client.ts.tmpl` | Generated API client |
| `mobile/reactnative/src/sdk/types.ts.tmpl` | Generated types |
| `mobile/reactnative/App.tsx.tmpl` | App component |

## Shared Concepts

All mobile runtimes implement the same core concepts:

### 1. MizuClient Interface

```
interface MizuClient {
  // Configuration
  baseURL: string
  auth: TokenStore
  headers: Headers

  // HTTP methods
  get(path, options): Response
  post(path, body, options): Response
  put(path, body, options): Response
  delete(path, options): Response

  // Streaming
  stream(path, options): AsyncIterator<Event>

  // Lifecycle
  configure(options): void
  close(): void
}
```

### 2. TokenStore Interface

```
interface TokenStore {
  getToken(): Token?
  setToken(token): void
  clearToken(): void
  refresh(): Token?
  onTokenChange(callback): void
}
```

### 3. Transport Interface

```
interface Transport {
  execute(request): Response
  addInterceptor(interceptor): void
  setTimeout(timeout): void
}
```

### 4. Live/Streaming Interface

```
interface LiveConnection {
  connect(): void
  disconnect(): void
  onEvent(callback): void
  onError(callback): void
  onReconnect(callback): void
}
```

## Template Variables

All mobile templates support these variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `{{.Name}}` | Project name | Directory name |
| `{{.Module}}` | Go module path | `example.com/{{.Name}}` |
| `{{.Vars.ui}}` | UI framework variant | Platform default |
| `{{.Vars.bundleId}}` | Bundle/package ID | `com.example.{{.Name}}` |
| `{{.Vars.minSdk}}` | Minimum SDK version | Platform default |

## Testing Strategy

### Unit Tests
- Runtime components (Transport, TokenStore)
- SDK generation
- Template rendering

### Integration Tests
- Full template generation
- Build verification (Xcode, Gradle, etc.)
- API client functionality

### E2E Tests
- Generate template
- Build project
- Run basic API calls

## Success Criteria

1. **Consistent DX**: Same mental model across all platforms
2. **Build Success**: Generated projects compile without errors
3. **Minimal Dependencies**: Only essential platform dependencies
4. **Type Safety**: Full type safety in generated SDKs
5. **Documentation**: Inline docs and README for each template

## Implementation Order

1. **iOS Template** - SwiftUI-first, URLSession transport
2. **Android Template** - Compose-first, OkHttp transport
3. **Flutter Template** - Cross-platform, http package
4. **React Native Template** - TypeScript, fetch API
5. **PWA Template** - Vite, service workers
6. **KMM Template** - Shared Kotlin logic
7. **MAUI Template** - .NET MAUI
8. **Game Template** - Unity/Unreal

## Dependencies

- Go 1.22+ (for template CLI)
- Swift 5.9+ / Xcode 15+ (iOS)
- Kotlin 1.9+ / Gradle 8+ (Android)
- Flutter 3.16+ (Flutter)
- React Native 0.73+ (RN)

## References

- [Frontend Templates](../cmd/cli/templates/frontend/) - Pattern reference
- [Mobile Package Spec](./0095_mobile.md) - Backend integration
- [Mobile Template Matrix](./0096_mobile_template.md) - Template definitions
