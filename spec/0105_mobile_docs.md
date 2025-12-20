# Mobile Documentation Tab - Specification

## Overview

This specification defines the comprehensive documentation structure for the "Mobile" tab in Mizu's documentation. The mobile documentation covers the `mobile` package and CLI templates for building mobile backends with Mizu.

## Goals

1. **Comprehensive Coverage**: Document all features of the `mobile` package
2. **Framework Guides**: Provide detailed guides for each supported platform (iOS, Android, Flutter, React Native, etc.)
3. **Template Documentation**: Explain all 8 mobile CLI templates
4. **Same Quality as Frontend**: Match the depth and style of `docs/frontend/*`
5. **Practical Examples**: Include real-world code examples for backend and client

## Mobile Package Features

The `mobile` package provides:

- **Device Detection**: Parse device info from headers and User-Agent
- **API Versioning**: Semantic version middleware with deprecation warnings
- **Offline Sync**: Delta synchronization with sync tokens
- **Push Notifications**: Cross-platform push token management (APNS, FCM, WNS)
- **Deep Links**: Universal Links (iOS) and App Links (Android)
- **App Store Integration**: Version checking, force updates, maintenance mode
- **Structured Errors**: Consistent mobile-friendly error responses
- **Pagination**: Page-based and cursor-based pagination helpers

## CLI Templates (8 total)

| Template | Framework | Description |
|----------|-----------|-------------|
| `mobile/ios` | Swift + SwiftUI | Native iOS app with Mizu SDK |
| `mobile/android` | Kotlin + Compose | Native Android app with Mizu SDK |
| `mobile/flutter` | Dart + Riverpod | Cross-platform Flutter app |
| `mobile/reactnative` | TypeScript + Expo | React Native with Zustand |
| `mobile/pwa` | React + Vite | Progressive Web App |
| `mobile/kmm` | Kotlin Multiplatform | Shared code for iOS/Android |
| `mobile/dotnet` | .NET MAUI | Cross-platform C# app |
| `mobile/game` | Unity + C# | Game with multiplayer support |

## Documentation Structure

### docs.json Entry

```json
{
  "tab": "Mobile",
  "groups": [
    {
      "group": "Getting Started",
      "pages": [
        "mobile/overview",
        "mobile/quick-start"
      ]
    },
    {
      "group": "Core Concepts",
      "pages": [
        "mobile/device",
        "mobile/versioning",
        "mobile/errors",
        "mobile/pagination"
      ]
    },
    {
      "group": "Backend Features",
      "pages": [
        "mobile/sync",
        "mobile/push",
        "mobile/deeplinks",
        "mobile/appstore"
      ]
    },
    {
      "group": "Native Platforms",
      "pages": [
        "mobile/ios",
        "mobile/android"
      ]
    },
    {
      "group": "Cross-Platform",
      "pages": [
        "mobile/flutter",
        "mobile/reactnative",
        "mobile/kmm",
        "mobile/dotnet"
      ]
    },
    {
      "group": "Web & Games",
      "pages": [
        "mobile/pwa",
        "mobile/game"
      ]
    },
    {
      "group": "Deployment",
      "pages": [
        "mobile/production",
        "mobile/security"
      ]
    },
    {
      "group": "Templates",
      "pages": [
        "mobile/templates",
        "mobile/adapters"
      ]
    },
    {
      "group": "Reference",
      "pages": [
        "mobile/api-reference",
        "mobile/headers",
        "mobile/troubleshooting"
      ]
    }
  ]
}
```

## Page Specifications

### Getting Started

#### mobile/overview.mdx
- What is Mizu Mobile?
- When to use it (vs traditional REST)
- Supported platforms
- Feature comparison table
- Architecture diagram
- Quick comparison (native vs cross-platform)
- What's Next? CardGroup

#### mobile/quick-start.mdx
- Prerequisites (Go, CLI)
- Create first mobile backend
- Add mobile middleware
- Example API endpoints
- Test with curl (mobile headers)
- Run with mobile client

### Core Concepts

#### mobile/device.mdx
- Device struct fields
- Header parsing
- User-Agent detection
- Platform detection
- DeviceFromCtx usage
- Validation options
- Code examples

#### mobile/versioning.mdx
- Version struct
- VersionMiddleware
- Version detection sources (header, query, path)
- Deprecation warnings
- Version-aware handlers
- Migration patterns

#### mobile/errors.mdx
- Error struct
- Error codes (ErrInvalidRequest, ErrUnauthorized, etc.)
- SendError helper
- WithDetails builder
- Error response format
- Localized errors

#### mobile/pagination.mdx
- PageRequest parsing
- Page-based pagination
- Cursor-based pagination
- Response formats
- Client implementation

### Backend Features

#### mobile/sync.mdx
- Sync tokens
- Delta synchronization
- SyncRequest parsing
- SyncResponse/SyncDelta types
- Conflict resolution
- Full sync vs incremental
- Example: offline-first todo app

#### mobile/push.mdx
- PushToken struct
- Provider detection (APNS, FCM, WNS)
- Token validation
- PushPayload builder
- ToAPNS/ToFCM conversion
- Registration endpoints
- Topic subscriptions

#### mobile/deeplinks.mdx
- Universal Links (apple-app-site-association)
- App Links (assetlinks.json)
- UniversalLinkMiddleware
- DeepLinkHandler
- Custom URL schemes
- Fallback to web
- Testing deep links

#### mobile/appstore.mdx
- AppInfo struct
- AppInfoProvider interface
- StaticAppInfo
- CheckUpdate
- Force update support
- Maintenance mode
- Store URLs

### Native Platforms

#### mobile/ios.mdx
- Why iOS with Mizu
- Quick start with template
- Project structure
- SwiftUI integration
- Mizu runtime
- API client generation
- Authentication patterns
- Deep dive: Transport layer

#### mobile/android.mdx
- Why Android with Mizu
- Quick start with template
- Project structure
- Jetpack Compose integration
- Mizu runtime (Kotlin)
- API client generation
- Authentication patterns
- Deep dive: OkHttp integration

### Cross-Platform

#### mobile/flutter.mdx
- Why Flutter with Mizu
- Quick start with template
- Project structure
- Riverpod state management
- Mizu runtime (Dart)
- API client generation
- Platform-specific code
- Multi-platform deployment

#### mobile/reactnative.mdx
- Why React Native with Mizu
- Quick start with template (Expo)
- Project structure
- Zustand state management
- TypeScript SDK
- Navigation integration
- Native module bridges

#### mobile/kmm.mdx
- What is Kotlin Multiplatform
- Quick start with template
- Shared code architecture
- Platform-specific UI
- Network client (Ktor)
- SQLDelight for storage
- Publishing shared library

#### mobile/dotnet.mdx
- Why .NET MAUI with Mizu
- Quick start with template
- Project structure
- MVVM pattern
- HttpClient integration
- Platform-specific features
- Shell navigation

### Web & Games

#### mobile/pwa.mdx
- What is a PWA?
- When to use PWA vs native
- Quick start with template
- Service worker setup
- Offline support
- Push notifications (web)
- Installation prompt
- App manifest

#### mobile/game.mdx
- Unity integration overview
- Quick start with template
- UniTask for async
- REST client implementation
- Multiplayer with WebSockets
- Analytics integration
- Cross-platform builds

### Deployment

#### mobile/production.mdx
- Environment configuration
- API versioning strategy
- Performance optimization
- Caching strategies
- Rate limiting
- Monitoring mobile metrics
- Error tracking

#### mobile/security.mdx
- Authentication patterns
- Token storage (Keychain, Keystore)
- Certificate pinning
- Request signing
- Data encryption
- Security headers
- Common vulnerabilities

### Templates

#### mobile/templates.mdx
- Available templates (8)
- Template comparison table
- Creating projects
- Customizing templates
- Template variables
- Common configurations

#### mobile/adapters.mdx
- Mobile adapters overview
- iOS adapter
- Android adapter
- Flutter adapter
- React Native adapter
- Capacitor adapter
- Creating custom adapters

### Reference

#### mobile/api-reference.mdx
- Full API documentation
- Types
- Functions
- Middleware
- Handlers
- Constants

#### mobile/headers.mdx
- Standard mobile headers
- Request headers (X-Device-ID, X-App-Version, etc.)
- Response headers (X-API-Version, X-Sync-Token, etc.)
- Header best practices

#### mobile/troubleshooting.mdx
- Common issues
- Debug logging
- Testing endpoints
- Device simulation
- Network inspection

## File Count

Total pages: **28 documentation files**

## Implementation Order

1. Getting Started (overview, quick-start)
2. Core Concepts (device, versioning, errors, pagination)
3. Backend Features (sync, push, deeplinks, appstore)
4. Native Platforms (ios, android)
5. Cross-Platform (flutter, reactnative, kmm, dotnet)
6. Web & Games (pwa, game)
7. Deployment (production, security)
8. Templates (templates, adapters)
9. Reference (api-reference, headers, troubleshooting)

## Style Guidelines

1. Follow frontmatter format:
   ```yaml
   ---
   title: "Page Title"
   description: "One-line description"
   icon: "icon-name"
   ---
   ```

2. Use consistent section headers matching frontend docs

3. Include practical code examples for both Go backend and client code

4. Use CardGroup for navigation at page bottom

5. Tables for comparisons and feature lists

6. Diagrams for architecture and flow

7. Warning/Note callouts for important information
