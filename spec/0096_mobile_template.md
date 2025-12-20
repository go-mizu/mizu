Below is a **complete, opinionated `mizu new -t mobile:*` matrix**, designed to stay **unified, minimal, and maintainable**, while covering real-world mobile stacks with strong DX.

The guiding rules applied:

* One template per platform family
* Language-first, not UI-first
* UI differences are variants, not forks
* Mizu runtime wiring is invariant
* Optional adapters, never forced frameworks

---

## `mizu new -t mobile:*` templates

### 1. `mobile:ios`

**Purpose**
Native Apple mobile apps using Swift.

**Default**

* Swift
* SwiftUI-first
* Swift Package Manager
* iOS 16+

**Variants**

* `--ui swiftui` (default)
* `--ui uikit`
* `--ui hybrid`

**Generated**

* `MizuMobileRuntime` (Swift)
* Generated Swift SDK
* URLSession transport
* Keychain-backed token store
* Async/await + AsyncSequence

**Who it’s for**

* Modern iOS apps
* Teams migrating from UIKit
* Apps that want offline-first + realtime

---

### 2. `mobile:android`

**Purpose**
Native Android apps using Kotlin.

**Default**

* Kotlin
* Jetpack Compose-first
* Gradle (Kotlin DSL)
* Android 8+ (API 26+)

**Variants**

* `--ui compose` (default)
* `--ui views`
* `--ui hybrid`

**Generated**

* `mizu-mobile-runtime` (Kotlin)
* Generated Kotlin SDK
* OkHttp transport
* Encrypted storage
* Coroutines + Flow

**Who it’s for**

* Modern Android apps
* Enterprise Android projects
* Apps with heavy realtime needs

---

### 3. `mobile:flutter`

**Purpose**
Cross-platform native apps from one Dart codebase.

**Default**

* Flutter
* Material + Cupertino baseline
* Dart async + Stream

**Variants**

* none initially (Flutter already abstracts UI)

**Generated**

* `mizu_mobile_runtime` (Dart)
* Generated Dart SDK
* HttpClient transport
* sqflite store
* Stream-based live

**Who it’s for**

* Teams shipping iOS + Android fast
* Startups and internal tools
* Apps needing offline sync

---

### 4. `mobile:reactnative`

**Purpose**
Cross-platform apps using JavaScript/TypeScript.

**Default**

* React Native
* TypeScript
* Metro bundler

**Variants**

* `--expo` (Expo managed)
* `--bare` (bare RN)

**Generated**

* `@go-mizu/mobile-runtime`
* Generated TS SDK
* fetch + WebSocket transport
* SecureStore / Keychain adapters

**Who it’s for**

* Web-heavy teams
* Rapid iteration products
* Shared frontend logic

---

### 5. `mobile:web` (PWA)

**Purpose**
Installable mobile web apps.

**Default**

* TypeScript
* Vite
* PWA config

**Variants**

* `--framework react`
* `--framework vue`
* `--framework svelte`

**Generated**

* Same TS runtime as RN
* IndexedDB storage
* Service Worker hooks

**Who it’s for**

* Lightweight mobile experiences
* Enterprise internal apps
* Kiosk or offline web apps

---

### 6. `mobile:kmm` (Kotlin Multiplatform Mobile)

**Purpose**
Shared Kotlin domain logic with native UIs.

**Default**

* Kotlin Multiplatform
* Shared module + platform UIs

**Variants**

* `--ui ios=swiftui android=compose` (default)
* `--ui ios=uikit android=views`

**Generated**

* Shared Mizu runtime in Kotlin
* Platform-specific transports
* Shared sync + live logic

**Who it’s for**

* Teams wanting maximum logic reuse
* Long-lived enterprise codebases

---

### 7. `mobile:dotnet`

**Purpose**
Enterprise-grade C# mobile apps.

**Default**

* .NET MAUI
* C# async/await

**Variants**

* `--legacy xamarin`

**Generated**

* C# runtime
* HttpClient transport
* IAsyncEnumerable for live streams

**Who it’s for**

* Enterprise teams
* Existing .NET organizations
* Regulated environments

---

### 8. `mobile:game`

**Purpose**
Games with backend connectivity.

**Default**

* Unity (C#)

**Variants**

* `--engine unity`
* `--engine unreal`

**Generated**

* Minimal runtime
* HTTP + WS only
* Auth + telemetry

**Who it’s for**

* Mobile games
* Realtime leaderboards
* Matchmaking backends

---

## CLI summary table

| Template             | Language | UI              | Scope          |
| -------------------- | -------- | --------------- | -------------- |
| `mobile:ios`         | Swift    | SwiftUI / UIKit | Native iOS     |
| `mobile:android`     | Kotlin   | Compose / Views | Native Android |
| `mobile:flutter`     | Dart     | Flutter         | Cross-platform |
| `mobile:reactnative` | TS       | React Native    | Cross-platform |
| `mobile:web`         | TS       | Web frameworks  | PWA            |
| `mobile:kmm`         | Kotlin   | Native UIs      | Shared logic   |
| `mobile:dotnet`      | C#       | MAUI/Xamarin    | Enterprise     |
| `mobile:game`        | C#/C++   | Unity/Unreal    | Games          |

---

## Golden rule for all mobile templates

Every `mobile:*` template must guarantee:

* Same mental model across platforms
* Same Mizu runtime concepts
* Same contract-driven client shape
* Same auth, sync, and live semantics
* UI differences do not affect backend integration

This is what makes **Mizu feel like a platform, not a pile of SDKs**.

If you want next steps, I can:

* design the exact template file plans for each `mobile:*`
* define the CLI flag schema
* generate the `mobile/*` template manifests
* or write the shared `MizuBootstrap` code for all platforms
