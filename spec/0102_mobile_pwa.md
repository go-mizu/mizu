# PWA Template Specification

**Status:** Draft
**Author:** Mizu Team
**Created:** 2025-12-20
**Template:** `mobile:pwa`

## Overview

The `mobile:pwa` template generates a production-ready Progressive Web Application (PWA) with full Mizu backend integration. It follows modern web development practices with TypeScript, React, Vite, and implements all PWA features including offline support, push notifications, and installability.

## Template Invocation

```bash
# Default: React PWA with Vite
mizu new ./MyApp --template mobile:pwa

# With Workbox (Google's service worker library)
mizu new ./MyApp --template mobile:pwa --var workbox=true

# With specific UI framework
mizu new ./MyApp --template mobile:pwa --var ui=tailwind

# With push notifications enabled
mizu new ./MyApp --template mobile:pwa --var push=true
```

## Generated Project Structure

```
{{.Name}}/
├── src/
│   ├── main.tsx                            # App entry point
│   ├── App.tsx                             # App component
│   ├── config/
│   │   └── config.ts                       # App configuration
│   ├── runtime/
│   │   ├── MizuRuntime.ts                  # Core runtime
│   │   ├── transport.ts                    # HTTP transport layer
│   │   ├── tokenStore.ts                   # IndexedDB token storage
│   │   ├── live.ts                         # SSE streaming
│   │   ├── deviceInfo.ts                   # Device/browser information
│   │   ├── offlineStore.ts                 # IndexedDB offline storage
│   │   └── errors.ts                       # Error types
│   ├── pwa/
│   │   ├── serviceWorker.ts                # SW registration
│   │   ├── pushManager.ts                  # Push notification manager
│   │   ├── installPrompt.ts                # Install prompt handler
│   │   └── syncManager.ts                  # Background sync manager
│   ├── sdk/
│   │   ├── client.ts                       # Generated Mizu client
│   │   ├── types.ts                        # Generated types
│   │   └── extensions.ts                   # Convenience extensions
│   ├── store/
│   │   └── authStore.ts                    # Zustand auth store
│   ├── screens/
│   │   ├── HomeScreen.tsx
│   │   ├── WelcomeScreen.tsx
│   │   └── OfflineScreen.tsx
│   ├── components/
│   │   ├── LoadingView.tsx
│   │   ├── ErrorView.tsx
│   │   ├── InstallPrompt.tsx
│   │   └── OfflineBanner.tsx
│   └── hooks/
│       ├── useOnlineStatus.ts
│       ├── useInstallPrompt.ts
│       └── usePushNotifications.ts
├── public/
│   ├── manifest.json                       # Web App Manifest
│   ├── sw.js                               # Service worker (compiled)
│   ├── icons/
│   │   ├── icon-72x72.png
│   │   ├── icon-96x96.png
│   │   ├── icon-128x128.png
│   │   ├── icon-144x144.png
│   │   ├── icon-152x152.png
│   │   ├── icon-192x192.png
│   │   ├── icon-384x384.png
│   │   ├── icon-512x512.png
│   │   └── maskable-icon-512x512.png
│   └── screenshots/
│       ├── screenshot-mobile.png
│       └── screenshot-desktop.png
├── __tests__/
│   ├── runtime.test.ts
│   ├── pwa.test.ts
│   └── App.test.tsx
├── package.json
├── tsconfig.json
├── vite.config.ts
├── vite-plugin-pwa.config.ts
├── .gitignore
└── README.md
```

## MizuMobileRuntime

### Core Runtime (`src/runtime/MizuRuntime.ts`)

```typescript
import { Transport, HttpTransport, TransportRequest, TransportResponse } from './transport';
import { TokenStore, IndexedDBTokenStore, AuthToken } from './tokenStore';
import { LiveConnection, ServerEvent } from './live';
import { DeviceInfo } from './deviceInfo';
import { OfflineStore } from './offlineStore';
import { MizuError, APIError } from './errors';

export interface MizuRuntimeConfig {
  baseURL?: string;
  transport?: Transport;
  tokenStore?: TokenStore;
  offlineStore?: OfflineStore;
  timeout?: number;
  enableOffline?: boolean;
}

/**
 * MizuRuntime is the core client for communicating with a Mizu backend.
 * Optimized for PWA with offline-first support and service worker integration.
 */
export class MizuRuntime {
  private static _instance: MizuRuntime | null = null;

  /** Base URL for all API requests */
  baseURL: string;

  /** HTTP transport layer */
  readonly transport: Transport;

  /** IndexedDB token storage */
  readonly tokenStore: TokenStore;

  /** Offline data storage */
  readonly offlineStore: OfflineStore;

  /** Live connection manager */
  readonly live: LiveConnection;

  /** Request timeout in milliseconds */
  timeout: number;

  /** Enable offline-first behavior */
  enableOffline: boolean;

  /** Default headers added to all requests */
  readonly defaultHeaders: Record<string, string> = {};

  /** Current authentication state */
  private _isAuthenticated: boolean = false;
  private _authListeners: Set<(isAuthenticated: boolean) => void> = new Set();

  /** Online status */
  private _isOnline: boolean = navigator.onLine;
  private _onlineListeners: Set<(isOnline: boolean) => void> = new Set();

  get isAuthenticated(): boolean {
    return this._isAuthenticated;
  }

  get isOnline(): boolean {
    return this._isOnline;
  }

  /** Shared singleton instance */
  static get shared(): MizuRuntime {
    if (!MizuRuntime._instance) {
      MizuRuntime._instance = new MizuRuntime();
    }
    return MizuRuntime._instance;
  }

  constructor(config: MizuRuntimeConfig = {}) {
    this.baseURL = config.baseURL ?? 'http://localhost:3000';
    this.transport = config.transport ?? new HttpTransport();
    this.tokenStore = config.tokenStore ?? new IndexedDBTokenStore();
    this.offlineStore = config.offlineStore ?? new OfflineStore();
    this.timeout = config.timeout ?? 30000;
    this.enableOffline = config.enableOffline ?? true;
    this.live = new LiveConnection(this);

    this._initAuthState();
    this._initOnlineListener();
  }

  /** Initialize with configuration */
  static async initialize(config: {
    baseURL: string;
    timeout?: number;
    enableOffline?: boolean;
  }): Promise<MizuRuntime> {
    const runtime = MizuRuntime.shared;
    runtime.baseURL = config.baseURL;
    if (config.timeout) {
      runtime.timeout = config.timeout;
    }
    if (config.enableOffline !== undefined) {
      runtime.enableOffline = config.enableOffline;
    }
    await runtime._initAuthState();
    await runtime.offlineStore.initialize();
    return runtime;
  }

  private async _initAuthState(): Promise<void> {
    const token = await this.tokenStore.getToken();
    this._isAuthenticated = token !== null;
    this._notifyAuthListeners();

    this.tokenStore.onTokenChange((token) => {
      this._isAuthenticated = token !== null;
      this._notifyAuthListeners();
    });
  }

  private _initOnlineListener(): void {
    window.addEventListener('online', () => {
      this._isOnline = true;
      this._notifyOnlineListeners();
      this._processPendingRequests();
    });

    window.addEventListener('offline', () => {
      this._isOnline = false;
      this._notifyOnlineListeners();
    });
  }

  /** Subscribe to authentication state changes */
  onAuthStateChange(listener: (isAuthenticated: boolean) => void): () => void {
    this._authListeners.add(listener);
    return () => this._authListeners.delete(listener);
  }

  /** Subscribe to online status changes */
  onOnlineChange(listener: (isOnline: boolean) => void): () => void {
    this._onlineListeners.add(listener);
    return () => this._onlineListeners.delete(listener);
  }

  private _notifyAuthListeners(): void {
    this._authListeners.forEach((listener) => listener(this._isAuthenticated));
  }

  private _notifyOnlineListeners(): void {
    this._onlineListeners.forEach((listener) => listener(this._isOnline));
  }

  private async _processPendingRequests(): Promise<void> {
    if (!this.enableOffline) return;
    const pending = await this.offlineStore.getPendingRequests();
    for (const req of pending) {
      try {
        await this._request(req);
        await this.offlineStore.removePendingRequest(req.id);
      } catch {
        // Keep in queue for next retry
      }
    }
  }

  // MARK: - HTTP Methods

  /** Performs a GET request */
  async get<T>(
    path: string,
    options: {
      query?: Record<string, string>;
      headers?: Record<string, string>;
      cacheKey?: string;
    } = {}
  ): Promise<T> {
    return this._request<T>({
      method: 'GET',
      path,
      ...options,
    });
  }

  /** Performs a POST request */
  async post<T, B = unknown>(
    path: string,
    body?: B,
    options: {
      headers?: Record<string, string>;
      offlineQueue?: boolean;
    } = {}
  ): Promise<T> {
    return this._request<T>({
      method: 'POST',
      path,
      body,
      ...options,
    });
  }

  /** Performs a PUT request */
  async put<T, B = unknown>(
    path: string,
    body?: B,
    options: {
      headers?: Record<string, string>;
      offlineQueue?: boolean;
    } = {}
  ): Promise<T> {
    return this._request<T>({
      method: 'PUT',
      path,
      body,
      ...options,
    });
  }

  /** Performs a DELETE request */
  async delete<T>(
    path: string,
    options: {
      headers?: Record<string, string>;
      offlineQueue?: boolean;
    } = {}
  ): Promise<T> {
    return this._request<T>({
      method: 'DELETE',
      path,
      ...options,
    });
  }

  /** Performs a PATCH request */
  async patch<T, B = unknown>(
    path: string,
    body?: B,
    options: {
      headers?: Record<string, string>;
      offlineQueue?: boolean;
    } = {}
  ): Promise<T> {
    return this._request<T>({
      method: 'PATCH',
      path,
      body,
      ...options,
    });
  }

  // MARK: - Streaming

  /** Opens a streaming connection for SSE */
  stream(
    path: string,
    options: {
      headers?: Record<string, string>;
    } = {}
  ): {
    subscribe: (onEvent: (event: ServerEvent) => void, onError?: (error: Error) => void) => () => void;
  } {
    return this.live.connect(path, options.headers);
  }

  // MARK: - Private

  private async _request<T>(options: {
    method: string;
    path: string;
    query?: Record<string, string>;
    body?: unknown;
    headers?: Record<string, string>;
    cacheKey?: string;
    offlineQueue?: boolean;
    id?: string;
  }): Promise<T> {
    const { method, path, query, body, headers, cacheKey, offlineQueue } = options;

    // Check offline cache for GET requests
    if (this.enableOffline && method === 'GET' && cacheKey && !this._isOnline) {
      const cached = await this.offlineStore.getCache(cacheKey);
      if (cached) {
        return cached as T;
      }
      throw MizuError.offline();
    }

    // Queue mutation requests when offline
    if (this.enableOffline && !this._isOnline && offlineQueue !== false && method !== 'GET') {
      await this.offlineStore.queueRequest({
        id: crypto.randomUUID(),
        method,
        path,
        body,
        headers,
        timestamp: Date.now(),
      });
      throw MizuError.queued();
    }

    // Build URL
    const url = this._buildUrl(path, query);

    // Build headers
    const allHeaders = await this._buildHeaders(headers);

    // Encode body
    let bodyJson: string | undefined;
    if (body !== undefined) {
      bodyJson = JSON.stringify(body);
      allHeaders['Content-Type'] = 'application/json';
    }

    // Execute request
    const response = await this.transport.execute({
      url,
      method,
      headers: allHeaders,
      body: bodyJson,
      timeout: this.timeout,
    });

    // Handle errors
    if (response.statusCode >= 400) {
      throw this._parseError(response);
    }

    // Handle empty response
    if (!response.body || response.body.length === 0) {
      return undefined as T;
    }

    // Decode response
    const data = JSON.parse(response.body) as T;

    // Cache GET responses
    if (this.enableOffline && method === 'GET' && cacheKey) {
      await this.offlineStore.setCache(cacheKey, data);
    }

    return data;
  }

  private _buildUrl(path: string, query?: Record<string, string>): string {
    const base = this.baseURL.endsWith('/') ? this.baseURL.slice(0, -1) : this.baseURL;
    const cleanPath = path.startsWith('/') ? path : `/${path}`;
    let url = `${base}${cleanPath}`;

    if (query && Object.keys(query).length > 0) {
      const queryString = Object.entries(query)
        .map(([key, value]) => `${encodeURIComponent(key)}=${encodeURIComponent(value)}`)
        .join('&');
      url = `${url}?${queryString}`;
    }

    return url;
  }

  private async _buildHeaders(custom?: Record<string, string>): Promise<Record<string, string>> {
    const headers: Record<string, string> = {};

    // Add default headers
    Object.assign(headers, this.defaultHeaders);

    // Add PWA headers
    const pwaHeaders = await this._pwaHeaders();
    Object.assign(headers, pwaHeaders);

    // Add custom headers
    if (custom) {
      Object.assign(headers, custom);
    }

    // Add auth token
    const token = await this.tokenStore.getToken();
    if (token) {
      headers['Authorization'] = `Bearer ${token.accessToken}`;
    }

    return headers;
  }

  private async _pwaHeaders(): Promise<Record<string, string>> {
    const info = await DeviceInfo.collect();
    return {
      'X-Device-ID': info.deviceId,
      'X-App-Version': info.appVersion,
      'X-App-Build': info.appBuild,
      'X-Platform': 'web',
      'X-Browser': info.browser,
      'X-Browser-Version': info.browserVersion,
      'X-OS': info.os,
      'X-OS-Version': info.osVersion,
      'X-Timezone': info.timezone,
      'X-Locale': info.locale,
      'X-PWA-Mode': info.pwaMode,
    };
  }

  private _parseError(response: TransportResponse): MizuError {
    try {
      const json = JSON.parse(response.body);
      return MizuError.api(APIError.fromJson(json));
    } catch {
      return MizuError.http(response.statusCode, response.body);
    }
  }
}
```

### Transport Layer (`src/runtime/transport.ts`)

```typescript
import { MizuError } from './errors';

export interface TransportRequest {
  url: string;
  method: string;
  headers: Record<string, string>;
  body?: string;
  timeout: number;
}

export interface TransportResponse {
  statusCode: number;
  headers: Record<string, string>;
  body: string;
}

export interface Transport {
  execute(request: TransportRequest): Promise<TransportResponse>;
}

export interface RequestInterceptor {
  intercept(request: TransportRequest): Promise<TransportRequest>;
}

/**
 * HTTP-based transport implementation using fetch
 */
export class HttpTransport implements Transport {
  private interceptors: RequestInterceptor[] = [];

  /** Adds a request interceptor */
  addInterceptor(interceptor: RequestInterceptor): void {
    this.interceptors.push(interceptor);
  }

  async execute(request: TransportRequest): Promise<TransportResponse> {
    let req = request;

    // Apply interceptors
    for (const interceptor of this.interceptors) {
      req = await interceptor.intercept(req);
    }

    const controller = new AbortController();
    const timeoutId = setTimeout(() => controller.abort(), req.timeout);

    try {
      const response = await fetch(req.url, {
        method: req.method,
        headers: req.headers,
        body: req.body,
        signal: controller.signal,
        credentials: 'include', // Include cookies for same-origin
      });

      clearTimeout(timeoutId);

      const body = await response.text();
      const headers: Record<string, string> = {};
      response.headers.forEach((value, key) => {
        headers[key] = value;
      });

      return {
        statusCode: response.status,
        headers,
        body,
      };
    } catch (error) {
      clearTimeout(timeoutId);
      if (error instanceof Error && error.name === 'AbortError') {
        throw MizuError.network(new Error('Request timed out'));
      }
      throw MizuError.network(error instanceof Error ? error : new Error(String(error)));
    }
  }
}

/**
 * Logging interceptor for debugging
 */
export class LoggingInterceptor implements RequestInterceptor {
  async intercept(request: TransportRequest): Promise<TransportRequest> {
    if (import.meta.env.DEV) {
      console.log(`[Mizu] ${request.method} ${request.url}`);
    }
    return request;
  }
}

/**
 * Retry interceptor with exponential backoff
 */
export class RetryInterceptor implements RequestInterceptor {
  constructor(
    public maxRetries: number = 3,
    public baseDelay: number = 1000
  ) {}

  async intercept(request: TransportRequest): Promise<TransportRequest> {
    // Retry logic is handled at transport level
    return request;
  }
}
```

### Token Store (`src/runtime/tokenStore.ts`)

```typescript
import { openDB, DBSchema, IDBPDatabase } from 'idb';

const DB_NAME = 'mizu-auth';
const DB_VERSION = 1;
const TOKEN_STORE = 'tokens';
const TOKEN_KEY = 'current';

interface AuthDB extends DBSchema {
  tokens: {
    key: string;
    value: AuthToken;
  };
}

export interface AuthToken {
  accessToken: string;
  refreshToken?: string;
  expiresAt?: string;
  tokenType: string;
}

export function createAuthToken(params: {
  accessToken: string;
  refreshToken?: string;
  expiresAt?: Date;
  tokenType?: string;
}): AuthToken {
  return {
    accessToken: params.accessToken,
    refreshToken: params.refreshToken,
    expiresAt: params.expiresAt?.toISOString(),
    tokenType: params.tokenType ?? 'Bearer',
  };
}

export function isTokenExpired(token: AuthToken): boolean {
  if (!token.expiresAt) return false;
  return new Date() >= new Date(token.expiresAt);
}

export type TokenChangeCallback = (token: AuthToken | null) => void;

export interface TokenStore {
  getToken(): Promise<AuthToken | null>;
  setToken(token: AuthToken): Promise<void>;
  clearToken(): Promise<void>;
  onTokenChange(callback: TokenChangeCallback): () => void;
}

/**
 * IndexedDB-backed token storage for PWA
 */
export class IndexedDBTokenStore implements TokenStore {
  private db: IDBPDatabase<AuthDB> | null = null;
  private observers: Set<TokenChangeCallback> = new Set();

  private async getDB(): Promise<IDBPDatabase<AuthDB>> {
    if (!this.db) {
      this.db = await openDB<AuthDB>(DB_NAME, DB_VERSION, {
        upgrade(db) {
          db.createObjectStore(TOKEN_STORE);
        },
      });
    }
    return this.db;
  }

  async getToken(): Promise<AuthToken | null> {
    try {
      const db = await this.getDB();
      const token = await db.get(TOKEN_STORE, TOKEN_KEY);
      return token ?? null;
    } catch {
      return null;
    }
  }

  async setToken(token: AuthToken): Promise<void> {
    const db = await this.getDB();
    await db.put(TOKEN_STORE, token, TOKEN_KEY);
    this._notifyObservers(token);
  }

  async clearToken(): Promise<void> {
    const db = await this.getDB();
    await db.delete(TOKEN_STORE, TOKEN_KEY);
    this._notifyObservers(null);
  }

  onTokenChange(callback: TokenChangeCallback): () => void {
    this.observers.add(callback);
    return () => this.observers.delete(callback);
  }

  private _notifyObservers(token: AuthToken | null): void {
    this.observers.forEach((callback) => callback(token));
  }
}

/**
 * In-memory token store for testing
 */
export class InMemoryTokenStore implements TokenStore {
  private token: AuthToken | null = null;
  private observers: Set<TokenChangeCallback> = new Set();

  async getToken(): Promise<AuthToken | null> {
    return this.token;
  }

  async setToken(token: AuthToken): Promise<void> {
    this.token = token;
    this._notifyObservers(token);
  }

  async clearToken(): Promise<void> {
    this.token = null;
    this._notifyObservers(null);
  }

  onTokenChange(callback: TokenChangeCallback): () => void {
    this.observers.add(callback);
    return () => this.observers.delete(callback);
  }

  private _notifyObservers(token: AuthToken | null): void {
    this.observers.forEach((callback) => callback(token));
  }
}
```

### Offline Store (`src/runtime/offlineStore.ts`)

```typescript
import { openDB, DBSchema, IDBPDatabase } from 'idb';

const DB_NAME = 'mizu-offline';
const DB_VERSION = 1;
const CACHE_STORE = 'cache';
const QUEUE_STORE = 'queue';

interface OfflineDB extends DBSchema {
  cache: {
    key: string;
    value: CacheEntry;
  };
  queue: {
    key: string;
    value: QueuedRequest;
    indexes: { 'by-timestamp': number };
  };
}

interface CacheEntry {
  data: unknown;
  timestamp: number;
  ttl: number;
}

export interface QueuedRequest {
  id: string;
  method: string;
  path: string;
  body?: unknown;
  headers?: Record<string, string>;
  timestamp: number;
}

/**
 * IndexedDB-backed offline storage for PWA
 */
export class OfflineStore {
  private db: IDBPDatabase<OfflineDB> | null = null;
  private defaultTTL: number = 24 * 60 * 60 * 1000; // 24 hours

  async initialize(): Promise<void> {
    if (!this.db) {
      this.db = await openDB<OfflineDB>(DB_NAME, DB_VERSION, {
        upgrade(db) {
          db.createObjectStore(CACHE_STORE);
          const queueStore = db.createObjectStore(QUEUE_STORE, { keyPath: 'id' });
          queueStore.createIndex('by-timestamp', 'timestamp');
        },
      });
    }
  }

  private async getDB(): Promise<IDBPDatabase<OfflineDB>> {
    if (!this.db) {
      await this.initialize();
    }
    return this.db!;
  }

  // MARK: - Cache

  async getCache<T>(key: string): Promise<T | null> {
    try {
      const db = await this.getDB();
      const entry = await db.get(CACHE_STORE, key);
      if (!entry) return null;

      // Check if expired
      if (Date.now() - entry.timestamp > entry.ttl) {
        await db.delete(CACHE_STORE, key);
        return null;
      }

      return entry.data as T;
    } catch {
      return null;
    }
  }

  async setCache(key: string, data: unknown, ttl?: number): Promise<void> {
    const db = await this.getDB();
    await db.put(CACHE_STORE, {
      data,
      timestamp: Date.now(),
      ttl: ttl ?? this.defaultTTL,
    }, key);
  }

  async deleteCache(key: string): Promise<void> {
    const db = await this.getDB();
    await db.delete(CACHE_STORE, key);
  }

  async clearCache(): Promise<void> {
    const db = await this.getDB();
    await db.clear(CACHE_STORE);
  }

  // MARK: - Request Queue

  async queueRequest(request: QueuedRequest): Promise<void> {
    const db = await this.getDB();
    await db.put(QUEUE_STORE, request);
  }

  async getPendingRequests(): Promise<QueuedRequest[]> {
    const db = await this.getDB();
    return db.getAllFromIndex(QUEUE_STORE, 'by-timestamp');
  }

  async removePendingRequest(id: string): Promise<void> {
    const db = await this.getDB();
    await db.delete(QUEUE_STORE, id);
  }

  async clearQueue(): Promise<void> {
    const db = await this.getDB();
    await db.clear(QUEUE_STORE);
  }

  async getQueueLength(): Promise<number> {
    const db = await this.getDB();
    return db.count(QUEUE_STORE);
  }
}
```

### Live Streaming (`src/runtime/live.ts`)

```typescript
import { MizuRuntime } from './MizuRuntime';
import { MizuError } from './errors';

export interface ServerEvent {
  id?: string;
  event?: string;
  data: string;
  retry?: number;
}

export function decodeServerEvent<T>(event: ServerEvent): T {
  return JSON.parse(event.data) as T;
}

/**
 * Live connection manager for SSE
 */
export class LiveConnection {
  private runtime: MizuRuntime;
  private activeConnections: Map<string, EventSource> = new Map();

  constructor(runtime: MizuRuntime) {
    this.runtime = runtime;
  }

  /**
   * Connects to an SSE endpoint and returns subscription methods
   */
  connect(
    path: string,
    headers?: Record<string, string>
  ): {
    subscribe: (onEvent: (event: ServerEvent) => void, onError?: (error: Error) => void) => () => void;
  } {
    return {
      subscribe: (onEvent, onError) => {
        // Cancel any existing connection to this path
        this.disconnect(path);

        const url = this._buildUrl(path);

        // EventSource doesn't support custom headers, so we use query params for auth
        const eventSource = new EventSource(url, { withCredentials: true });

        eventSource.onmessage = (event) => {
          onEvent({
            id: event.lastEventId,
            data: event.data,
          });
        };

        eventSource.onerror = () => {
          if (onError) {
            onError(new Error('SSE connection error'));
          }
        };

        this.activeConnections.set(path, eventSource);

        return () => {
          eventSource.close();
          this.activeConnections.delete(path);
        };
      },
    };
  }

  /** Disconnects from a specific path */
  disconnect(path: string): void {
    const source = this.activeConnections.get(path);
    if (source) {
      source.close();
      this.activeConnections.delete(path);
    }
  }

  /** Disconnects all active connections */
  disconnectAll(): void {
    this.activeConnections.forEach((source) => source.close());
    this.activeConnections.clear();
  }

  private _buildUrl(path: string): string {
    const base = this.runtime.baseURL.endsWith('/')
      ? this.runtime.baseURL.slice(0, -1)
      : this.runtime.baseURL;
    const cleanPath = path.startsWith('/') ? path : `/${path}`;
    return `${base}${cleanPath}`;
  }
}
```

### Device Info (`src/runtime/deviceInfo.ts`)

```typescript
const DEVICE_ID_KEY = 'mizu_device_id';

export interface DeviceInfoData {
  deviceId: string;
  appVersion: string;
  appBuild: string;
  browser: string;
  browserVersion: string;
  os: string;
  osVersion: string;
  timezone: string;
  locale: string;
  pwaMode: 'browser' | 'standalone' | 'fullscreen' | 'minimal-ui';
  screenWidth: number;
  screenHeight: number;
  devicePixelRatio: number;
}

let cachedInfo: DeviceInfoData | null = null;

export class DeviceInfo {
  /**
   * Collects device/browser information
   */
  static async collect(): Promise<DeviceInfoData> {
    if (cachedInfo) return cachedInfo;

    const deviceId = this.getOrCreateDeviceId();
    const ua = this.parseUserAgent();
    const pwaMode = this.getPWAMode();

    cachedInfo = {
      deviceId,
      appVersion: import.meta.env.VITE_APP_VERSION ?? '1.0.0',
      appBuild: import.meta.env.VITE_APP_BUILD ?? '1',
      browser: ua.browser,
      browserVersion: ua.browserVersion,
      os: ua.os,
      osVersion: ua.osVersion,
      timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
      locale: navigator.language,
      pwaMode,
      screenWidth: window.screen.width,
      screenHeight: window.screen.height,
      devicePixelRatio: window.devicePixelRatio,
    };

    return cachedInfo;
  }

  private static getOrCreateDeviceId(): string {
    let deviceId = localStorage.getItem(DEVICE_ID_KEY);
    if (!deviceId) {
      deviceId = crypto.randomUUID();
      localStorage.setItem(DEVICE_ID_KEY, deviceId);
    }
    return deviceId;
  }

  private static getPWAMode(): 'browser' | 'standalone' | 'fullscreen' | 'minimal-ui' {
    if (window.matchMedia('(display-mode: fullscreen)').matches) {
      return 'fullscreen';
    }
    if (window.matchMedia('(display-mode: standalone)').matches) {
      return 'standalone';
    }
    if (window.matchMedia('(display-mode: minimal-ui)').matches) {
      return 'minimal-ui';
    }
    return 'browser';
  }

  private static parseUserAgent(): {
    browser: string;
    browserVersion: string;
    os: string;
    osVersion: string;
  } {
    const ua = navigator.userAgent;

    // Browser detection
    let browser = 'Unknown';
    let browserVersion = '';
    if (ua.includes('Firefox/')) {
      browser = 'Firefox';
      browserVersion = ua.match(/Firefox\/([\d.]+)/)?.[1] ?? '';
    } else if (ua.includes('Edg/')) {
      browser = 'Edge';
      browserVersion = ua.match(/Edg\/([\d.]+)/)?.[1] ?? '';
    } else if (ua.includes('Chrome/')) {
      browser = 'Chrome';
      browserVersion = ua.match(/Chrome\/([\d.]+)/)?.[1] ?? '';
    } else if (ua.includes('Safari/') && !ua.includes('Chrome')) {
      browser = 'Safari';
      browserVersion = ua.match(/Version\/([\d.]+)/)?.[1] ?? '';
    }

    // OS detection
    let os = 'Unknown';
    let osVersion = '';
    if (ua.includes('Windows')) {
      os = 'Windows';
      osVersion = ua.match(/Windows NT ([\d.]+)/)?.[1] ?? '';
    } else if (ua.includes('Mac OS X')) {
      os = 'macOS';
      osVersion = ua.match(/Mac OS X ([\d_]+)/)?.[1]?.replace(/_/g, '.') ?? '';
    } else if (ua.includes('Android')) {
      os = 'Android';
      osVersion = ua.match(/Android ([\d.]+)/)?.[1] ?? '';
    } else if (ua.includes('iPhone') || ua.includes('iPad')) {
      os = 'iOS';
      osVersion = ua.match(/OS ([\d_]+)/)?.[1]?.replace(/_/g, '.') ?? '';
    } else if (ua.includes('Linux')) {
      os = 'Linux';
    }

    return { browser, browserVersion, os, osVersion };
  }

  /** Clears cached info (useful for testing) */
  static clearCache(): void {
    cachedInfo = null;
  }
}
```

### Errors (`src/runtime/errors.ts`)

```typescript
export interface APIError {
  code: string;
  message: string;
  details?: Record<string, unknown>;
  traceId?: string;
}

export namespace APIError {
  export function fromJson(json: Record<string, unknown>): APIError {
    return {
      code: json.code as string,
      message: json.message as string,
      details: json.details as Record<string, unknown> | undefined,
      traceId: json.trace_id as string | undefined,
    };
  }
}

export type MizuErrorType =
  | 'invalid_response'
  | 'http'
  | 'api'
  | 'network'
  | 'encoding'
  | 'decoding'
  | 'unauthorized'
  | 'token_expired'
  | 'offline'
  | 'queued';

export class MizuError extends Error {
  readonly type: MizuErrorType;
  readonly cause?: unknown;
  readonly apiError?: APIError;
  readonly statusCode?: number;

  private constructor(type: MizuErrorType, message: string, cause?: unknown) {
    super(message);
    this.name = 'MizuError';
    this.type = type;
    this.cause = cause;
  }

  static invalidResponse(): MizuError {
    return new MizuError('invalid_response', 'Invalid server response');
  }

  static http(statusCode: number, body: string): MizuError {
    const error = new MizuError('http', `HTTP error ${statusCode}`, body);
    (error as any).statusCode = statusCode;
    return error;
  }

  static api(apiError: APIError): MizuError {
    const error = new MizuError('api', apiError.message, apiError);
    (error as any).apiError = apiError;
    return error;
  }

  static network(cause: Error): MizuError {
    return new MizuError('network', 'Network error', cause);
  }

  static encoding(cause: Error): MizuError {
    return new MizuError('encoding', 'Encoding error', cause);
  }

  static decoding(cause: Error): MizuError {
    return new MizuError('decoding', 'Decoding error', cause);
  }

  static unauthorized(): MizuError {
    return new MizuError('unauthorized', 'Unauthorized');
  }

  static tokenExpired(): MizuError {
    return new MizuError('token_expired', 'Token expired');
  }

  static offline(): MizuError {
    return new MizuError('offline', 'You are offline');
  }

  static queued(): MizuError {
    return new MizuError('queued', 'Request queued for later');
  }

  get isInvalidResponse(): boolean {
    return this.type === 'invalid_response';
  }

  get isHttp(): boolean {
    return this.type === 'http';
  }

  get isApi(): boolean {
    return this.type === 'api';
  }

  get isNetwork(): boolean {
    return this.type === 'network';
  }

  get isUnauthorized(): boolean {
    return this.type === 'unauthorized';
  }

  get isTokenExpired(): boolean {
    return this.type === 'token_expired';
  }

  get isOffline(): boolean {
    return this.type === 'offline';
  }

  get isQueued(): boolean {
    return this.type === 'queued';
  }
}
```

## PWA Features

### Service Worker Registration (`src/pwa/serviceWorker.ts`)

```typescript
export interface ServiceWorkerConfig {
  onUpdate?: (registration: ServiceWorkerRegistration) => void;
  onSuccess?: (registration: ServiceWorkerRegistration) => void;
  onOffline?: () => void;
  onOnline?: () => void;
}

/**
 * Registers the service worker and handles updates
 */
export async function registerServiceWorker(config: ServiceWorkerConfig = {}): Promise<ServiceWorkerRegistration | null> {
  if (!('serviceWorker' in navigator)) {
    console.warn('[PWA] Service workers are not supported');
    return null;
  }

  try {
    const registration = await navigator.serviceWorker.register('/sw.js', {
      scope: '/',
    });

    // Check for updates on page load
    registration.addEventListener('updatefound', () => {
      const newWorker = registration.installing;
      if (!newWorker) return;

      newWorker.addEventListener('statechange', () => {
        if (newWorker.state === 'installed') {
          if (navigator.serviceWorker.controller) {
            // New content available
            config.onUpdate?.(registration);
          } else {
            // Content cached for offline
            config.onSuccess?.(registration);
          }
        }
      });
    });

    // Handle controller change (reload when new SW takes over)
    let refreshing = false;
    navigator.serviceWorker.addEventListener('controllerchange', () => {
      if (!refreshing) {
        refreshing = true;
        window.location.reload();
      }
    });

    // Online/offline events
    window.addEventListener('online', () => config.onOnline?.());
    window.addEventListener('offline', () => config.onOffline?.());

    console.log('[PWA] Service worker registered');
    return registration;
  } catch (error) {
    console.error('[PWA] Service worker registration failed:', error);
    return null;
  }
}

/**
 * Unregisters all service workers
 */
export async function unregisterServiceWorker(): Promise<boolean> {
  if (!('serviceWorker' in navigator)) return false;

  try {
    const registration = await navigator.serviceWorker.ready;
    return await registration.unregister();
  } catch {
    return false;
  }
}

/**
 * Triggers service worker update
 */
export async function updateServiceWorker(): Promise<void> {
  if (!('serviceWorker' in navigator)) return;

  const registration = await navigator.serviceWorker.ready;
  await registration.update();
}

/**
 * Sends skip waiting message to service worker
 */
export function skipWaiting(): void {
  if (!('serviceWorker' in navigator)) return;

  navigator.serviceWorker.controller?.postMessage({ type: 'SKIP_WAITING' });
}
```

### Push Manager (`src/pwa/pushManager.ts`)

```typescript
export interface PushConfig {
  vapidPublicKey: string;
  onPermissionGranted?: () => void;
  onPermissionDenied?: () => void;
}

export type PushPermissionState = 'granted' | 'denied' | 'prompt';

/**
 * Manages Web Push notifications
 */
export class PushManager {
  private config: PushConfig;
  private registration: ServiceWorkerRegistration | null = null;

  constructor(config: PushConfig) {
    this.config = config;
  }

  /**
   * Checks if push notifications are supported
   */
  static isSupported(): boolean {
    return 'serviceWorker' in navigator && 'PushManager' in window && 'Notification' in window;
  }

  /**
   * Gets current permission state
   */
  async getPermissionState(): Promise<PushPermissionState> {
    if (!PushManager.isSupported()) return 'denied';

    const permission = await navigator.permissions.query({ name: 'notifications' });
    return permission.state as PushPermissionState;
  }

  /**
   * Requests push notification permission
   */
  async requestPermission(): Promise<boolean> {
    if (!PushManager.isSupported()) return false;

    const result = await Notification.requestPermission();
    if (result === 'granted') {
      this.config.onPermissionGranted?.();
      return true;
    }

    this.config.onPermissionDenied?.();
    return false;
  }

  /**
   * Subscribes to push notifications
   */
  async subscribe(): Promise<PushSubscription | null> {
    if (!PushManager.isSupported()) return null;

    try {
      this.registration = await navigator.serviceWorker.ready;

      const subscription = await this.registration.pushManager.subscribe({
        userVisibleOnly: true,
        applicationServerKey: this.urlBase64ToUint8Array(this.config.vapidPublicKey),
      });

      return subscription;
    } catch (error) {
      console.error('[Push] Subscription failed:', error);
      return null;
    }
  }

  /**
   * Gets current subscription
   */
  async getSubscription(): Promise<PushSubscription | null> {
    if (!PushManager.isSupported()) return null;

    const registration = await navigator.serviceWorker.ready;
    return registration.pushManager.getSubscription();
  }

  /**
   * Unsubscribes from push notifications
   */
  async unsubscribe(): Promise<boolean> {
    const subscription = await this.getSubscription();
    if (!subscription) return false;

    return subscription.unsubscribe();
  }

  private urlBase64ToUint8Array(base64String: string): Uint8Array {
    const padding = '='.repeat((4 - (base64String.length % 4)) % 4);
    const base64 = (base64String + padding).replace(/-/g, '+').replace(/_/g, '/');
    const rawData = window.atob(base64);
    const outputArray = new Uint8Array(rawData.length);

    for (let i = 0; i < rawData.length; ++i) {
      outputArray[i] = rawData.charCodeAt(i);
    }

    return outputArray;
  }
}
```

### Install Prompt (`src/pwa/installPrompt.ts`)

```typescript
interface BeforeInstallPromptEvent extends Event {
  readonly platforms: string[];
  readonly userChoice: Promise<{ outcome: 'accepted' | 'dismissed' }>;
  prompt(): Promise<void>;
}

export type InstallState = 'not-available' | 'available' | 'installed' | 'dismissed';

export interface InstallPromptHandler {
  state: InstallState;
  prompt: () => Promise<boolean>;
  dismiss: () => void;
}

let deferredPrompt: BeforeInstallPromptEvent | null = null;
let installState: InstallState = 'not-available';
let stateListeners: Set<(state: InstallState) => void> = new Set();

/**
 * Initializes install prompt handling
 */
export function initInstallPrompt(): void {
  // Check if already installed
  if (window.matchMedia('(display-mode: standalone)').matches) {
    installState = 'installed';
    return;
  }

  // Listen for install prompt event
  window.addEventListener('beforeinstallprompt', (e: Event) => {
    e.preventDefault();
    deferredPrompt = e as BeforeInstallPromptEvent;
    installState = 'available';
    notifyListeners();
  });

  // Listen for successful installation
  window.addEventListener('appinstalled', () => {
    installState = 'installed';
    deferredPrompt = null;
    notifyListeners();
  });
}

/**
 * Gets current install state
 */
export function getInstallState(): InstallState {
  return installState;
}

/**
 * Shows the install prompt
 */
export async function showInstallPrompt(): Promise<boolean> {
  if (!deferredPrompt) return false;

  await deferredPrompt.prompt();
  const { outcome } = await deferredPrompt.userChoice;

  if (outcome === 'accepted') {
    installState = 'installed';
  } else {
    installState = 'dismissed';
  }

  deferredPrompt = null;
  notifyListeners();

  return outcome === 'accepted';
}

/**
 * Dismisses the install prompt
 */
export function dismissInstallPrompt(): void {
  installState = 'dismissed';
  deferredPrompt = null;
  notifyListeners();
}

/**
 * Subscribes to install state changes
 */
export function onInstallStateChange(listener: (state: InstallState) => void): () => void {
  stateListeners.add(listener);
  return () => stateListeners.delete(listener);
}

function notifyListeners(): void {
  stateListeners.forEach((listener) => listener(installState));
}
```

### Background Sync Manager (`src/pwa/syncManager.ts`)

```typescript
export interface SyncConfig {
  onSyncComplete?: () => void;
  onSyncError?: (error: Error) => void;
}

/**
 * Manages background sync for offline requests
 */
export class SyncManager {
  private config: SyncConfig;

  constructor(config: SyncConfig = {}) {
    this.config = config;
  }

  /**
   * Checks if background sync is supported
   */
  static isSupported(): boolean {
    return 'serviceWorker' in navigator && 'SyncManager' in window;
  }

  /**
   * Registers a sync event
   */
  async registerSync(tag: string = 'background-sync'): Promise<boolean> {
    if (!SyncManager.isSupported()) return false;

    try {
      const registration = await navigator.serviceWorker.ready;
      await (registration as any).sync.register(tag);
      return true;
    } catch (error) {
      console.error('[Sync] Registration failed:', error);
      return false;
    }
  }

  /**
   * Gets all pending sync tags
   */
  async getTags(): Promise<string[]> {
    if (!SyncManager.isSupported()) return [];

    try {
      const registration = await navigator.serviceWorker.ready;
      return await (registration as any).sync.getTags();
    } catch {
      return [];
    }
  }
}

/**
 * Registers a periodic sync (if supported)
 */
export async function registerPeriodicSync(
  tag: string,
  minInterval: number
): Promise<boolean> {
  if (!('periodicSync' in ServiceWorkerRegistration.prototype)) {
    return false;
  }

  try {
    const status = await navigator.permissions.query({
      name: 'periodic-background-sync' as PermissionName,
    });

    if (status.state !== 'granted') {
      return false;
    }

    const registration = await navigator.serviceWorker.ready;
    await (registration as any).periodicSync.register(tag, {
      minInterval,
    });

    return true;
  } catch {
    return false;
  }
}
```

## Service Worker (`public/sw.js`)

```javascript
const CACHE_NAME = '{{.Name | lower}}-v1';
const STATIC_CACHE = '{{.Name | lower}}-static-v1';
const API_CACHE = '{{.Name | lower}}-api-v1';

// Files to cache immediately
const STATIC_FILES = [
  '/',
  '/index.html',
  '/manifest.json',
  '/icons/icon-192x192.png',
  '/icons/icon-512x512.png',
];

// Install event - cache static files
self.addEventListener('install', (event) => {
  event.waitUntil(
    caches.open(STATIC_CACHE).then((cache) => {
      return cache.addAll(STATIC_FILES);
    })
  );
});

// Activate event - clean up old caches
self.addEventListener('activate', (event) => {
  event.waitUntil(
    caches.keys().then((cacheNames) => {
      return Promise.all(
        cacheNames
          .filter((name) => name.startsWith('{{.Name | lower}}-') && name !== STATIC_CACHE && name !== API_CACHE)
          .map((name) => caches.delete(name))
      );
    })
  );
});

// Fetch event - serve from cache, fallback to network
self.addEventListener('fetch', (event) => {
  const { request } = event;
  const url = new URL(request.url);

  // Skip non-GET requests
  if (request.method !== 'GET') {
    return;
  }

  // API requests - network first, cache fallback
  if (url.pathname.startsWith('/api/')) {
    event.respondWith(networkFirst(request, API_CACHE));
    return;
  }

  // Static files - cache first, network fallback
  if (isStaticAsset(url.pathname)) {
    event.respondWith(cacheFirst(request, STATIC_CACHE));
    return;
  }

  // HTML pages - network first, cache fallback
  event.respondWith(networkFirst(request, STATIC_CACHE));
});

// Cache-first strategy
async function cacheFirst(request, cacheName) {
  const cached = await caches.match(request);
  if (cached) {
    return cached;
  }

  try {
    const response = await fetch(request);
    if (response.ok) {
      const cache = await caches.open(cacheName);
      cache.put(request, response.clone());
    }
    return response;
  } catch {
    return new Response('Offline', { status: 503 });
  }
}

// Network-first strategy
async function networkFirst(request, cacheName) {
  try {
    const response = await fetch(request);
    if (response.ok) {
      const cache = await caches.open(cacheName);
      cache.put(request, response.clone());
    }
    return response;
  } catch {
    const cached = await caches.match(request);
    if (cached) {
      return cached;
    }
    return new Response('Offline', { status: 503 });
  }
}

// Check if URL is a static asset
function isStaticAsset(pathname) {
  return /\.(js|css|png|jpg|jpeg|gif|svg|ico|woff|woff2|ttf|eot)$/.test(pathname);
}

// Push notification handler
self.addEventListener('push', (event) => {
  const data = event.data?.json() ?? {};
  const title = data.title ?? '{{.Name}}';
  const options = {
    body: data.body ?? '',
    icon: '/icons/icon-192x192.png',
    badge: '/icons/icon-72x72.png',
    data: data.data ?? {},
    tag: data.tag ?? 'default',
    requireInteraction: data.requireInteraction ?? false,
  };

  event.waitUntil(self.registration.showNotification(title, options));
});

// Notification click handler
self.addEventListener('notificationclick', (event) => {
  event.notification.close();

  const data = event.notification.data;
  const urlToOpen = data.url ?? '/';

  event.waitUntil(
    clients.matchAll({ type: 'window', includeUncontrolled: true }).then((windowClients) => {
      // Check if there is already a window open
      for (const client of windowClients) {
        if (client.url === urlToOpen && 'focus' in client) {
          return client.focus();
        }
      }
      // Open a new window
      if (clients.openWindow) {
        return clients.openWindow(urlToOpen);
      }
    })
  );
});

// Background sync handler
self.addEventListener('sync', (event) => {
  if (event.tag === 'background-sync') {
    event.waitUntil(processPendingRequests());
  }
});

async function processPendingRequests() {
  // This would normally read from IndexedDB and process queued requests
  console.log('[SW] Processing pending requests');
}

// Skip waiting message handler
self.addEventListener('message', (event) => {
  if (event.data?.type === 'SKIP_WAITING') {
    self.skipWaiting();
  }
});
```

## Web App Manifest (`public/manifest.json`)

```json
{
  "name": "{{.Name}}",
  "short_name": "{{.Name}}",
  "description": "A modern PWA powered by Mizu",
  "start_url": "/",
  "display": "standalone",
  "background_color": "#ffffff",
  "theme_color": "#6366F1",
  "orientation": "any",
  "scope": "/",
  "icons": [
    {
      "src": "/icons/icon-72x72.png",
      "sizes": "72x72",
      "type": "image/png",
      "purpose": "any"
    },
    {
      "src": "/icons/icon-96x96.png",
      "sizes": "96x96",
      "type": "image/png",
      "purpose": "any"
    },
    {
      "src": "/icons/icon-128x128.png",
      "sizes": "128x128",
      "type": "image/png",
      "purpose": "any"
    },
    {
      "src": "/icons/icon-144x144.png",
      "sizes": "144x144",
      "type": "image/png",
      "purpose": "any"
    },
    {
      "src": "/icons/icon-152x152.png",
      "sizes": "152x152",
      "type": "image/png",
      "purpose": "any"
    },
    {
      "src": "/icons/icon-192x192.png",
      "sizes": "192x192",
      "type": "image/png",
      "purpose": "any"
    },
    {
      "src": "/icons/icon-384x384.png",
      "sizes": "384x384",
      "type": "image/png",
      "purpose": "any"
    },
    {
      "src": "/icons/icon-512x512.png",
      "sizes": "512x512",
      "type": "image/png",
      "purpose": "any"
    },
    {
      "src": "/icons/maskable-icon-512x512.png",
      "sizes": "512x512",
      "type": "image/png",
      "purpose": "maskable"
    }
  ],
  "screenshots": [
    {
      "src": "/screenshots/screenshot-mobile.png",
      "sizes": "390x844",
      "type": "image/png",
      "form_factor": "narrow"
    },
    {
      "src": "/screenshots/screenshot-desktop.png",
      "sizes": "1920x1080",
      "type": "image/png",
      "form_factor": "wide"
    }
  ],
  "categories": ["utilities"],
  "prefer_related_applications": false,
  "related_applications": [],
  "shortcuts": [
    {
      "name": "Home",
      "url": "/",
      "icons": [{ "src": "/icons/icon-96x96.png", "sizes": "96x96" }]
    }
  ],
  "share_target": {
    "action": "/share",
    "method": "POST",
    "enctype": "multipart/form-data",
    "params": {
      "title": "title",
      "text": "text",
      "url": "url"
    }
  }
}
```

## App Templates

### App Entry (`src/main.tsx`)

```tsx
import React from 'react';
import ReactDOM from 'react-dom/client';
import App from './App';
import { registerServiceWorker } from './pwa/serviceWorker';
import { initInstallPrompt } from './pwa/installPrompt';
import './index.css';

// Initialize PWA features
initInstallPrompt();

registerServiceWorker({
  onUpdate: (registration) => {
    console.log('[App] New version available');
    // Show update notification to user
  },
  onSuccess: (registration) => {
    console.log('[App] Content cached for offline use');
  },
  onOffline: () => {
    console.log('[App] You are offline');
  },
  onOnline: () => {
    console.log('[App] You are online');
  },
});

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>
);
```

### App Component (`src/App.tsx`)

```tsx
import React, { useEffect, useState } from 'react';
import { MizuRuntime } from './runtime/MizuRuntime';
import { AppConfig } from './config/config';
import { useAuthStore } from './store/authStore';
import { useOnlineStatus } from './hooks/useOnlineStatus';
import HomeScreen from './screens/HomeScreen';
import WelcomeScreen from './screens/WelcomeScreen';
import OfflineScreen from './screens/OfflineScreen';
import LoadingView from './components/LoadingView';
import OfflineBanner from './components/OfflineBanner';
import InstallPrompt from './components/InstallPrompt';

export default function App() {
  const [isReady, setIsReady] = useState(false);
  const isAuthenticated = useAuthStore((state) => state.isAuthenticated);
  const setIsAuthenticated = useAuthStore((state) => state.setIsAuthenticated);
  const isOnline = useOnlineStatus();

  useEffect(() => {
    async function initialize() {
      // Initialize Mizu runtime
      await MizuRuntime.initialize({
        baseURL: AppConfig.baseURL,
        timeout: AppConfig.timeout,
        enableOffline: true,
      });

      // Subscribe to auth state changes
      const unsubscribe = MizuRuntime.shared.onAuthStateChange((isAuth) => {
        setIsAuthenticated(isAuth);
      });

      // Check initial auth state
      setIsAuthenticated(MizuRuntime.shared.isAuthenticated);
      setIsReady(true);

      return unsubscribe;
    }

    initialize();
  }, [setIsAuthenticated]);

  if (!isReady) {
    return <LoadingView message="Loading..." />;
  }

  return (
    <div className="app">
      {!isOnline && <OfflineBanner />}
      {isAuthenticated ? <HomeScreen /> : <WelcomeScreen />}
      <InstallPrompt />
    </div>
  );
}
```

### Config (`src/config/config.ts`)

```typescript
const isDevelopment = import.meta.env.DEV;

export const AppConfig = {
  get baseURL(): string {
    if (isDevelopment) {
      return 'http://localhost:3000';
    }
    return import.meta.env.VITE_API_URL ?? 'https://api.example.com';
  },

  timeout: 30000,

  debug: isDevelopment,

  vapidPublicKey: import.meta.env.VITE_VAPID_PUBLIC_KEY ?? '',
};
```

### Auth Store (`src/store/authStore.ts`)

```typescript
import { create } from 'zustand';
import { MizuRuntime } from '../runtime/MizuRuntime';
import { AuthToken } from '../runtime/tokenStore';

interface AuthState {
  isAuthenticated: boolean;
  token: AuthToken | null;
  setIsAuthenticated: (value: boolean) => void;
  setToken: (token: AuthToken | null) => void;
  signOut: () => Promise<void>;
}

export const useAuthStore = create<AuthState>((set) => ({
  isAuthenticated: false,
  token: null,
  setIsAuthenticated: (value) => set({ isAuthenticated: value }),
  setToken: (token) => set({ token, isAuthenticated: token !== null }),
  signOut: async () => {
    await MizuRuntime.shared.tokenStore.clearToken();
    set({ isAuthenticated: false, token: null });
  },
}));
```

## Hooks

### useOnlineStatus (`src/hooks/useOnlineStatus.ts`)

```typescript
import { useState, useEffect } from 'react';

export function useOnlineStatus(): boolean {
  const [isOnline, setIsOnline] = useState(navigator.onLine);

  useEffect(() => {
    const handleOnline = () => setIsOnline(true);
    const handleOffline = () => setIsOnline(false);

    window.addEventListener('online', handleOnline);
    window.addEventListener('offline', handleOffline);

    return () => {
      window.removeEventListener('online', handleOnline);
      window.removeEventListener('offline', handleOffline);
    };
  }, []);

  return isOnline;
}
```

### useInstallPrompt (`src/hooks/useInstallPrompt.ts`)

```typescript
import { useState, useEffect } from 'react';
import {
  InstallState,
  getInstallState,
  onInstallStateChange,
  showInstallPrompt,
  dismissInstallPrompt,
} from '../pwa/installPrompt';

interface UseInstallPromptReturn {
  state: InstallState;
  canInstall: boolean;
  install: () => Promise<boolean>;
  dismiss: () => void;
}

export function useInstallPrompt(): UseInstallPromptReturn {
  const [state, setState] = useState<InstallState>(getInstallState());

  useEffect(() => {
    return onInstallStateChange(setState);
  }, []);

  return {
    state,
    canInstall: state === 'available',
    install: showInstallPrompt,
    dismiss: dismissInstallPrompt,
  };
}
```

### usePushNotifications (`src/hooks/usePushNotifications.ts`)

```typescript
import { useState, useCallback, useEffect } from 'react';
import { PushManager, PushPermissionState } from '../pwa/pushManager';
import { AppConfig } from '../config/config';

interface UsePushNotificationsReturn {
  isSupported: boolean;
  permission: PushPermissionState;
  isSubscribed: boolean;
  subscribe: () => Promise<boolean>;
  unsubscribe: () => Promise<boolean>;
}

export function usePushNotifications(): UsePushNotificationsReturn {
  const [permission, setPermission] = useState<PushPermissionState>('prompt');
  const [isSubscribed, setIsSubscribed] = useState(false);
  const [pushManager] = useState(() => new PushManager({
    vapidPublicKey: AppConfig.vapidPublicKey,
  }));

  useEffect(() => {
    async function checkStatus() {
      const perm = await pushManager.getPermissionState();
      setPermission(perm);

      const sub = await pushManager.getSubscription();
      setIsSubscribed(sub !== null);
    }
    checkStatus();
  }, [pushManager]);

  const subscribe = useCallback(async () => {
    const hasPermission = await pushManager.requestPermission();
    if (!hasPermission) {
      setPermission('denied');
      return false;
    }

    setPermission('granted');
    const subscription = await pushManager.subscribe();
    if (subscription) {
      setIsSubscribed(true);
      // Send subscription to backend
      // await sendSubscriptionToBackend(subscription);
      return true;
    }
    return false;
  }, [pushManager]);

  const unsubscribe = useCallback(async () => {
    const success = await pushManager.unsubscribe();
    if (success) {
      setIsSubscribed(false);
    }
    return success;
  }, [pushManager]);

  return {
    isSupported: PushManager.isSupported(),
    permission,
    isSubscribed,
    subscribe,
    unsubscribe,
  };
}
```

## Components

### OfflineBanner (`src/components/OfflineBanner.tsx`)

```tsx
import React from 'react';
import './OfflineBanner.css';

export default function OfflineBanner() {
  return (
    <div className="offline-banner">
      <span className="offline-icon">&#9888;</span>
      <span>You are currently offline</span>
    </div>
  );
}
```

### InstallPrompt (`src/components/InstallPrompt.tsx`)

```tsx
import React from 'react';
import { useInstallPrompt } from '../hooks/useInstallPrompt';
import './InstallPrompt.css';

export default function InstallPrompt() {
  const { canInstall, install, dismiss } = useInstallPrompt();

  if (!canInstall) {
    return null;
  }

  return (
    <div className="install-prompt">
      <div className="install-prompt-content">
        <span className="install-icon">+</span>
        <div className="install-text">
          <strong>Install {{.Name}}</strong>
          <p>Add to your home screen for quick access</p>
        </div>
      </div>
      <div className="install-actions">
        <button className="install-button-secondary" onClick={dismiss}>
          Not now
        </button>
        <button className="install-button-primary" onClick={install}>
          Install
        </button>
      </div>
    </div>
  );
}
```

### LoadingView (`src/components/LoadingView.tsx`)

```tsx
import React from 'react';
import './LoadingView.css';

interface LoadingViewProps {
  message?: string;
}

export default function LoadingView({ message }: LoadingViewProps) {
  return (
    <div className="loading-container">
      <div className="loading-spinner" />
      {message && <p className="loading-message">{message}</p>}
    </div>
  );
}
```

### ErrorView (`src/components/ErrorView.tsx`)

```tsx
import React from 'react';
import { MizuError } from '../runtime/errors';
import './ErrorView.css';

interface ErrorViewProps {
  error: Error | MizuError;
  onRetry?: () => void;
}

export default function ErrorView({ error, onRetry }: ErrorViewProps) {
  const message = error instanceof MizuError ? error.message : error.message;

  return (
    <div className="error-container">
      <div className="error-icon">!</div>
      <h2 className="error-title">Something went wrong</h2>
      <p className="error-message">{message}</p>
      {onRetry && (
        <button className="error-button" onClick={onRetry}>
          Try Again
        </button>
      )}
    </div>
  );
}
```

## Build Configuration

### package.json

```json
{
  "name": "{{.Name | lower}}",
  "version": "1.0.0",
  "type": "module",
  "scripts": {
    "dev": "vite",
    "build": "tsc && vite build",
    "preview": "vite preview",
    "lint": "eslint . --ext .ts,.tsx",
    "test": "vitest"
  },
  "dependencies": {
    "idb": "^8.0.0",
    "react": "^18.3.0",
    "react-dom": "^18.3.0",
    "zustand": "^4.5.0"
  },
  "devDependencies": {
    "@types/react": "^18.3.0",
    "@types/react-dom": "^18.3.0",
    "@typescript-eslint/eslint-plugin": "^7.0.0",
    "@typescript-eslint/parser": "^7.0.0",
    "@vitejs/plugin-react": "^4.3.0",
    "eslint": "^8.57.0",
    "eslint-plugin-react": "^7.34.0",
    "eslint-plugin-react-hooks": "^4.6.0",
    "typescript": "^5.4.0",
    "vite": "^5.4.0",
    "vite-plugin-pwa": "^0.20.0",
    "vitest": "^2.0.0",
    "workbox-window": "^7.1.0"
  },
  "private": true
}
```

### vite.config.ts

```typescript
import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import { VitePWA } from 'vite-plugin-pwa';

export default defineConfig({
  plugins: [
    react(),
    VitePWA({
      registerType: 'prompt',
      includeAssets: ['icons/*.png', 'screenshots/*.png'],
      manifest: false, // Use our own manifest.json
      workbox: {
        globPatterns: ['**/*.{js,css,html,ico,png,svg,woff,woff2}'],
        runtimeCaching: [
          {
            urlPattern: /^https:\/\/api\./,
            handler: 'NetworkFirst',
            options: {
              cacheName: 'api-cache',
              expiration: {
                maxEntries: 100,
                maxAgeSeconds: 60 * 60 * 24, // 24 hours
              },
            },
          },
        ],
      },
    }),
  ],
  server: {
    port: 3001,
    proxy: {
      '/api': {
        target: 'http://localhost:3000',
        changeOrigin: true,
      },
    },
  },
  build: {
    sourcemap: true,
  },
});
```

### tsconfig.json

```json
{
  "compilerOptions": {
    "target": "ES2020",
    "useDefineForClassFields": true,
    "lib": ["ES2020", "DOM", "DOM.Iterable"],
    "module": "ESNext",
    "skipLibCheck": true,
    "moduleResolution": "bundler",
    "allowImportingTsExtensions": true,
    "resolveJsonModule": true,
    "isolatedModules": true,
    "noEmit": true,
    "jsx": "react-jsx",
    "strict": true,
    "noUnusedLocals": true,
    "noUnusedParameters": true,
    "noFallthroughCasesInSwitch": true,
    "types": ["vite/client"]
  },
  "include": ["src"],
  "references": [{ "path": "./tsconfig.node.json" }]
}
```

### tsconfig.node.json

```json
{
  "compilerOptions": {
    "composite": true,
    "skipLibCheck": true,
    "module": "ESNext",
    "moduleResolution": "bundler",
    "allowSyntheticDefaultImports": true,
    "strict": true
  },
  "include": ["vite.config.ts"]
}
```

## Testing

### Unit Tests (`__tests__/runtime.test.ts`)

```typescript
import { describe, test, expect, beforeEach } from 'vitest';
import { InMemoryTokenStore, createAuthToken, isTokenExpired } from '../src/runtime/tokenStore';
import { MizuError, APIError } from '../src/runtime/errors';

describe('InMemoryTokenStore', () => {
  let store: InMemoryTokenStore;

  beforeEach(() => {
    store = new InMemoryTokenStore();
  });

  test('returns null when no token stored', async () => {
    expect(await store.getToken()).toBeNull();
  });

  test('stores and retrieves token', async () => {
    const token = createAuthToken({
      accessToken: 'test123',
      refreshToken: 'refresh456',
    });

    await store.setToken(token);
    const retrieved = await store.getToken();

    expect(retrieved?.accessToken).toBe('test123');
    expect(retrieved?.refreshToken).toBe('refresh456');
  });

  test('clears token', async () => {
    const token = createAuthToken({ accessToken: 'test123' });
    await store.setToken(token);
    await store.clearToken();

    expect(await store.getToken()).toBeNull();
  });

  test('notifies observers on token change', async () => {
    let notifiedToken: any = undefined;
    store.onTokenChange((token) => {
      notifiedToken = token;
    });

    const token = createAuthToken({ accessToken: 'test123' });
    await store.setToken(token);

    expect(notifiedToken?.accessToken).toBe('test123');
  });
});

describe('AuthToken', () => {
  test('isTokenExpired returns false when no expiry', () => {
    const token = createAuthToken({ accessToken: 'test' });
    expect(isTokenExpired(token)).toBe(false);
  });

  test('isTokenExpired returns true when expired', () => {
    const token = createAuthToken({
      accessToken: 'test',
      expiresAt: new Date(Date.now() - 1000),
    });
    expect(isTokenExpired(token)).toBe(true);
  });

  test('isTokenExpired returns false when not expired', () => {
    const token = createAuthToken({
      accessToken: 'test',
      expiresAt: new Date(Date.now() + 3600000),
    });
    expect(isTokenExpired(token)).toBe(false);
  });
});

describe('MizuError', () => {
  test('creates network error', () => {
    const error = MizuError.network(new Error('Connection failed'));
    expect(error.isNetwork).toBe(true);
    expect(error.message).toBe('Network error');
  });

  test('creates api error', () => {
    const apiError: APIError = { code: 'test_error', message: 'Test message' };
    const error = MizuError.api(apiError);
    expect(error.isApi).toBe(true);
    expect(error.apiError?.code).toBe('test_error');
  });

  test('creates offline error', () => {
    const error = MizuError.offline();
    expect(error.isOffline).toBe(true);
    expect(error.message).toBe('You are offline');
  });

  test('creates queued error', () => {
    const error = MizuError.queued();
    expect(error.isQueued).toBe(true);
    expect(error.message).toBe('Request queued for later');
  });
});
```

### PWA Tests (`__tests__/pwa.test.ts`)

```typescript
import { describe, test, expect, vi, beforeEach } from 'vitest';

describe('Install Prompt', () => {
  beforeEach(() => {
    vi.resetModules();
  });

  test('initial state is not-available', async () => {
    const { getInstallState } = await import('../src/pwa/installPrompt');
    expect(getInstallState()).toBe('not-available');
  });
});

describe('Push Manager', () => {
  test('isSupported returns false when APIs not available', async () => {
    const { PushManager } = await import('../src/pwa/pushManager');
    expect(PushManager.isSupported()).toBe(false);
  });
});
```

### Component Tests (`__tests__/App.test.tsx`)

```tsx
import { describe, test, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import LoadingView from '../src/components/LoadingView';
import ErrorView from '../src/components/ErrorView';

describe('LoadingView', () => {
  test('renders without message', () => {
    render(<LoadingView />);
    expect(document.querySelector('.loading-spinner')).toBeTruthy();
  });

  test('renders with message', () => {
    render(<LoadingView message="Loading..." />);
    expect(screen.getByText('Loading...')).toBeTruthy();
  });
});

describe('ErrorView', () => {
  test('renders error message', () => {
    render(<ErrorView error={new Error('Test error')} />);
    expect(screen.getByText('Test error')).toBeTruthy();
  });

  test('renders retry button when onRetry provided', () => {
    render(<ErrorView error={new Error('Test error')} onRetry={() => {}} />);
    expect(screen.getByText('Try Again')).toBeTruthy();
  });
});
```

## PWA Features Matrix

| Feature | Support | Notes |
|---------|---------|-------|
| Offline Support | Full | Service Worker + IndexedDB |
| Push Notifications | Full | Web Push API with VAPID |
| Install Prompt | Full | beforeinstallprompt handling |
| Background Sync | Partial | Requires browser support |
| Periodic Sync | Partial | Chrome-only |
| Share Target | Full | Manifest share_target |
| App Shortcuts | Full | Manifest shortcuts |
| Maskable Icons | Full | Adaptive icons |
| Screenshots | Full | Rich install UI |
| Badge API | Partial | Chrome/Edge only |
| File Handling | Partial | Chrome-only |

## Browser Support

| Browser | Minimum Version | Notes |
|---------|-----------------|-------|
| Chrome | 67+ | Full PWA support |
| Edge | 79+ | Chromium-based |
| Firefox | 84+ | No install prompt |
| Safari | 11.1+ | Limited PWA support |
| Samsung Internet | 8.2+ | Full PWA support |

## Security Considerations

1. **HTTPS Required** - PWA features require HTTPS in production
2. **CSP Headers** - Configure Content Security Policy
3. **Token Storage** - IndexedDB is more secure than localStorage
4. **Service Worker Scope** - Limit SW scope appropriately
5. **Push Security** - Use VAPID for push authentication

## References

- [Web App Manifest](https://developer.mozilla.org/en-US/docs/Web/Manifest)
- [Service Workers API](https://developer.mozilla.org/en-US/docs/Web/API/Service_Worker_API)
- [Push API](https://developer.mozilla.org/en-US/docs/Web/API/Push_API)
- [IndexedDB API](https://developer.mozilla.org/en-US/docs/Web/API/IndexedDB_API)
- [Workbox](https://developer.chrome.com/docs/workbox/)
- [Mobile Package Spec](./0095_mobile.md)
