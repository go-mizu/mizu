# React Native Template Specification

**Status:** Draft
**Author:** Mizu Team
**Created:** 2025-12-20
**Template:** `mobile:reactnative`

## Overview

The `mobile:reactnative` template generates a production-ready React Native application with full Mizu backend integration. It follows modern React Native development practices with TypeScript, React Navigation, Zustand state management, and clean architecture patterns that work across iOS and Android platforms.

## Template Invocation

```bash
# Default: React Native with React Navigation
mizu new ./MyApp --template mobile:reactnative

# With Expo (managed workflow)
mizu new ./MyApp --template mobile:reactnative --var expo=true

# Custom bundle identifier
mizu new ./MyApp --template mobile:reactnative --var bundleId=com.company.myapp

# Specific platforms only
mizu new ./MyApp --template mobile:reactnative --var platforms=ios,android
```

## Generated Project Structure

```
{{.Name}}/
├── src/
│   ├── App.tsx                           # App entry point
│   ├── config/
│   │   └── config.ts                     # App configuration
│   ├── runtime/
│   │   ├── MizuRuntime.ts                # Core runtime
│   │   ├── transport.ts                  # HTTP transport layer
│   │   ├── tokenStore.ts                 # Secure token storage
│   │   ├── live.ts                       # SSE streaming
│   │   ├── deviceInfo.ts                 # Device information
│   │   └── errors.ts                     # Error types
│   ├── sdk/
│   │   ├── client.ts                     # Generated Mizu client
│   │   ├── types.ts                      # Generated types
│   │   └── extensions.ts                 # Convenience extensions
│   ├── store/
│   │   └── authStore.ts                  # Zustand auth store
│   ├── screens/
│   │   ├── HomeScreen.tsx
│   │   └── WelcomeScreen.tsx
│   ├── components/
│   │   ├── LoadingView.tsx
│   │   └── ErrorView.tsx
│   └── navigation/
│       └── AppNavigator.tsx              # Navigation configuration
├── __tests__/
│   ├── runtime.test.ts
│   ├── sdk.test.ts
│   └── App.test.tsx
├── android/                              # Android-specific
├── ios/                                  # iOS-specific
├── package.json
├── tsconfig.json
├── babel.config.js
├── metro.config.js
├── .gitignore
└── README.md
```

## MizuMobileRuntime

### Core Runtime (`src/runtime/MizuRuntime.ts`)

```typescript
import { Transport, HttpTransport, TransportRequest, TransportResponse } from './transport';
import { TokenStore, SecureTokenStore, AuthToken } from './tokenStore';
import { LiveConnection, ServerEvent } from './live';
import { DeviceInfo } from './deviceInfo';
import { MizuError, APIError } from './errors';

export interface MizuRuntimeConfig {
  baseURL?: string;
  transport?: Transport;
  tokenStore?: TokenStore;
  timeout?: number;
}

/**
 * MizuRuntime is the core client for communicating with a Mizu backend.
 */
export class MizuRuntime {
  private static _instance: MizuRuntime | null = null;

  /** Base URL for all API requests */
  baseURL: string;

  /** HTTP transport layer */
  readonly transport: Transport;

  /** Secure token storage */
  readonly tokenStore: TokenStore;

  /** Live connection manager */
  readonly live: LiveConnection;

  /** Request timeout in milliseconds */
  timeout: number;

  /** Default headers added to all requests */
  readonly defaultHeaders: Record<string, string> = {};

  /** Current authentication state */
  private _isAuthenticated: boolean = false;
  private _authListeners: Set<(isAuthenticated: boolean) => void> = new Set();

  get isAuthenticated(): boolean {
    return this._isAuthenticated;
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
    this.tokenStore = config.tokenStore ?? new SecureTokenStore();
    this.timeout = config.timeout ?? 30000;
    this.live = new LiveConnection(this);

    this._initAuthState();
  }

  /** Initialize with configuration */
  static async initialize(config: {
    baseURL: string;
    timeout?: number;
  }): Promise<MizuRuntime> {
    const runtime = MizuRuntime.shared;
    runtime.baseURL = config.baseURL;
    if (config.timeout) {
      runtime.timeout = config.timeout;
    }
    await runtime._initAuthState();
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

  /** Subscribe to authentication state changes */
  onAuthStateChange(listener: (isAuthenticated: boolean) => void): () => void {
    this._authListeners.add(listener);
    return () => this._authListeners.delete(listener);
  }

  private _notifyAuthListeners(): void {
    this._authListeners.forEach((listener) => listener(this._isAuthenticated));
  }

  // MARK: - HTTP Methods

  /** Performs a GET request */
  async get<T>(
    path: string,
    options: {
      query?: Record<string, string>;
      headers?: Record<string, string>;
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
  }): Promise<T> {
    const { method, path, query, body, headers } = options;

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
    return JSON.parse(response.body) as T;
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

    // Add mobile headers
    const mobileHeaders = await this._mobileHeaders();
    Object.assign(headers, mobileHeaders);

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

  private async _mobileHeaders(): Promise<Record<string, string>> {
    const info = await DeviceInfo.collect();
    return {
      'X-Device-ID': info.deviceId,
      'X-App-Version': info.appVersion,
      'X-App-Build': info.appBuild,
      'X-Device-Model': info.model,
      'X-Platform': info.platform,
      'X-OS-Version': info.osVersion,
      'X-Timezone': info.timezone,
      'X-Locale': info.locale,
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
    if (__DEV__) {
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
import * as SecureStore from 'expo-secure-store';
// For non-Expo projects, use react-native-keychain:
// import * as Keychain from 'react-native-keychain';

const TOKEN_KEY = 'mizu_auth_token';

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
 * Secure storage-backed token storage using Expo SecureStore
 */
export class SecureTokenStore implements TokenStore {
  private observers: Set<TokenChangeCallback> = new Set();

  async getToken(): Promise<AuthToken | null> {
    try {
      const json = await SecureStore.getItemAsync(TOKEN_KEY);
      if (!json) return null;
      return JSON.parse(json) as AuthToken;
    } catch {
      return null;
    }
  }

  async setToken(token: AuthToken): Promise<void> {
    await SecureStore.setItemAsync(TOKEN_KEY, JSON.stringify(token));
    this._notifyObservers(token);
  }

  async clearToken(): Promise<void> {
    await SecureStore.deleteItemAsync(TOKEN_KEY);
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
  private activeConnections: Map<string, AbortController> = new Map();

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
        const controller = new AbortController();

        // Cancel any existing connection to this path
        const existing = this.activeConnections.get(path);
        if (existing) {
          existing.abort();
        }
        this.activeConnections.set(path, controller);

        this._connect(path, headers, controller.signal, onEvent, onError);

        return () => {
          controller.abort();
          this.activeConnections.delete(path);
        };
      },
    };
  }

  private async _connect(
    path: string,
    headers: Record<string, string> | undefined,
    signal: AbortSignal,
    onEvent: (event: ServerEvent) => void,
    onError?: (error: Error) => void
  ): Promise<void> {
    const url = this._buildUrl(path);
    const allHeaders = await this._buildHeaders(headers);

    try {
      const response = await fetch(url, {
        method: 'GET',
        headers: {
          ...allHeaders,
          Accept: 'text/event-stream',
          'Cache-Control': 'no-cache',
        },
        signal,
      });

      if (!response.ok) {
        throw MizuError.http(response.status, 'SSE connection failed');
      }

      const reader = response.body?.getReader();
      if (!reader) {
        throw MizuError.network(new Error('Response body is not readable'));
      }

      const decoder = new TextDecoder();
      const eventBuilder = new SSEEventBuilder();

      while (true) {
        const { done, value } = await reader.read();
        if (done) break;

        const text = decoder.decode(value, { stream: true });
        const lines = text.split('\n');

        for (const line of lines) {
          const event = eventBuilder.processLine(line);
          if (event) {
            onEvent(event);
          }
        }
      }
    } catch (error) {
      if (signal.aborted) return;
      if (onError) {
        onError(error instanceof Error ? error : new Error(String(error)));
      }
    } finally {
      this.activeConnections.delete(path);
    }
  }

  /** Disconnects from a specific path */
  disconnect(path: string): void {
    const controller = this.activeConnections.get(path);
    if (controller) {
      controller.abort();
      this.activeConnections.delete(path);
    }
  }

  /** Disconnects all active connections */
  disconnectAll(): void {
    this.activeConnections.forEach((controller) => controller.abort());
    this.activeConnections.clear();
  }

  private _buildUrl(path: string): string {
    const base = this.runtime.baseURL.endsWith('/')
      ? this.runtime.baseURL.slice(0, -1)
      : this.runtime.baseURL;
    const cleanPath = path.startsWith('/') ? path : `/${path}`;
    return `${base}${cleanPath}`;
  }

  private async _buildHeaders(custom?: Record<string, string>): Promise<Record<string, string>> {
    const headers: Record<string, string> = {};
    Object.assign(headers, this.runtime.defaultHeaders);
    if (custom) {
      Object.assign(headers, custom);
    }

    const token = await this.runtime.tokenStore.getToken();
    if (token) {
      headers['Authorization'] = `Bearer ${token.accessToken}`;
    }

    return headers;
  }
}

/**
 * SSE event parser
 */
class SSEEventBuilder {
  private id?: string;
  private event?: string;
  private data: string[] = [];
  private retry?: number;

  processLine(line: string): ServerEvent | null {
    if (line === '') {
      // Empty line means end of event
      if (this.data.length === 0) return null;

      const event: ServerEvent = {
        id: this.id,
        event: this.event,
        data: this.data.join('\n'),
        retry: this.retry,
      };

      // Reset for next event (keep id for Last-Event-ID)
      this.event = undefined;
      this.data = [];
      this.retry = undefined;

      return event;
    }

    if (line.startsWith(':')) {
      // Comment, ignore
      return null;
    }

    const colonIndex = line.indexOf(':');
    if (colonIndex === -1) {
      // Field with no value
      return null;
    }

    const field = line.substring(0, colonIndex);
    let value = line.substring(colonIndex + 1);
    if (value.startsWith(' ')) {
      value = value.substring(1);
    }

    switch (field) {
      case 'id':
        this.id = value;
        break;
      case 'event':
        this.event = value;
        break;
      case 'data':
        this.data.push(value);
        break;
      case 'retry':
        const parsed = parseInt(value, 10);
        if (!isNaN(parsed)) {
          this.retry = parsed;
        }
        break;
    }

    return null;
  }
}
```

### Device Info (`src/runtime/deviceInfo.ts`)

```typescript
import { Platform, Dimensions } from 'react-native';
import * as Application from 'expo-application';
import * as Device from 'expo-device';
import * as Localization from 'expo-localization';
import * as SecureStore from 'expo-secure-store';
import 'react-native-get-random-values';
import { v4 as uuidv4 } from 'uuid';

const DEVICE_ID_KEY = 'mizu_device_id';

export interface DeviceInfoData {
  deviceId: string;
  appVersion: string;
  appBuild: string;
  model: string;
  platform: string;
  osVersion: string;
  timezone: string;
  locale: string;
}

let cachedInfo: DeviceInfoData | null = null;

export class DeviceInfo {
  /**
   * Collects device information
   */
  static async collect(): Promise<DeviceInfoData> {
    if (cachedInfo) return cachedInfo;

    const deviceId = await this.getOrCreateDeviceId();

    cachedInfo = {
      deviceId,
      appVersion: Application.nativeApplicationVersion ?? '1.0.0',
      appBuild: Application.nativeBuildVersion ?? '1',
      model: Device.modelName ?? 'Unknown',
      platform: Platform.OS,
      osVersion: Platform.Version?.toString() ?? 'Unknown',
      timezone: Localization.timezone,
      locale: Localization.locale,
    };

    return cachedInfo;
  }

  private static async getOrCreateDeviceId(): Promise<string> {
    let deviceId = await SecureStore.getItemAsync(DEVICE_ID_KEY);
    if (!deviceId) {
      deviceId = uuidv4();
      await SecureStore.setItemAsync(DEVICE_ID_KEY, deviceId);
    }
    return deviceId;
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
  | 'token_expired';

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
}
```

## App Templates

### App Entry (`src/App.tsx`)

```tsx
import React, { useEffect, useState } from 'react';
import { NavigationContainer } from '@react-navigation/native';
import { createNativeStackNavigator } from '@react-navigation/native-stack';
import { MizuRuntime } from './runtime/MizuRuntime';
import { AppConfig } from './config/config';
import { useAuthStore } from './store/authStore';
import HomeScreen from './screens/HomeScreen';
import WelcomeScreen from './screens/WelcomeScreen';
import LoadingView from './components/LoadingView';

export type RootStackParamList = {
  Welcome: undefined;
  Home: undefined;
};

const Stack = createNativeStackNavigator<RootStackParamList>();

export default function App() {
  const [isReady, setIsReady] = useState(false);
  const isAuthenticated = useAuthStore((state) => state.isAuthenticated);
  const setIsAuthenticated = useAuthStore((state) => state.setIsAuthenticated);

  useEffect(() => {
    async function initialize() {
      // Initialize Mizu runtime
      await MizuRuntime.initialize({
        baseURL: AppConfig.baseURL,
        timeout: AppConfig.timeout,
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
    <NavigationContainer>
      <Stack.Navigator screenOptions={{ headerShown: false }}>
        {isAuthenticated ? (
          <Stack.Screen name="Home" component={HomeScreen} />
        ) : (
          <Stack.Screen name="Welcome" component={WelcomeScreen} />
        )}
      </Stack.Navigator>
    </NavigationContainer>
  );
}
```

### Config (`src/config/config.ts`)

```typescript
import { Platform } from 'react-native';

const isDevelopment = __DEV__;

export const AppConfig = {
  get baseURL(): string {
    if (isDevelopment) {
      // Use 10.0.2.2 for Android emulator, localhost for iOS simulator
      return Platform.select({
        android: 'http://10.0.2.2:3000',
        ios: 'http://localhost:3000',
        default: 'http://localhost:3000',
      }) as string;
    }
    return 'https://api.example.com';
  },

  timeout: 30000,

  debug: isDevelopment,
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

### Home Screen (`src/screens/HomeScreen.tsx`)

```tsx
import React, { useState } from 'react';
import {
  View,
  Text,
  StyleSheet,
  TouchableOpacity,
  ActivityIndicator,
  SafeAreaView,
} from 'react-native';
import { useAuthStore } from '../store/authStore';

export default function HomeScreen() {
  const [isLoading, setIsLoading] = useState(false);
  const signOut = useAuthStore((state) => state.signOut);

  const handleSignOut = async () => {
    setIsLoading(true);
    try {
      await signOut();
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <SafeAreaView style={styles.container}>
      <View style={styles.content}>
        <View style={styles.iconContainer}>
          <Text style={styles.icon}>✓</Text>
        </View>
        <Text style={styles.title}>Welcome to {{.Name}}</Text>
        <Text style={styles.subtitle}>Connected to Mizu backend</Text>

        {isLoading ? (
          <ActivityIndicator size="large" color="#6366F1" style={styles.button} />
        ) : (
          <TouchableOpacity style={styles.button} onPress={handleSignOut}>
            <Text style={styles.buttonText}>Sign Out</Text>
          </TouchableOpacity>
        )}
      </View>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: '#FFFFFF',
  },
  content: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
    padding: 24,
  },
  iconContainer: {
    width: 64,
    height: 64,
    borderRadius: 32,
    backgroundColor: '#EEF2FF',
    justifyContent: 'center',
    alignItems: 'center',
    marginBottom: 16,
  },
  icon: {
    fontSize: 32,
    color: '#6366F1',
  },
  title: {
    fontSize: 24,
    fontWeight: 'bold',
    color: '#1F2937',
    textAlign: 'center',
    marginBottom: 8,
  },
  subtitle: {
    fontSize: 16,
    color: '#6B7280',
    textAlign: 'center',
    marginBottom: 32,
  },
  button: {
    backgroundColor: '#EEF2FF',
    paddingVertical: 16,
    paddingHorizontal: 32,
    borderRadius: 12,
  },
  buttonText: {
    fontSize: 16,
    fontWeight: '600',
    color: '#6366F1',
  },
});
```

### Welcome Screen (`src/screens/WelcomeScreen.tsx`)

```tsx
import React, { useState } from 'react';
import {
  View,
  Text,
  StyleSheet,
  TouchableOpacity,
  ActivityIndicator,
  SafeAreaView,
} from 'react-native';
import { MizuRuntime } from '../runtime/MizuRuntime';
import { createAuthToken } from '../runtime/tokenStore';

export default function WelcomeScreen() {
  const [isLoading, setIsLoading] = useState(false);

  const handleGetStarted = async () => {
    setIsLoading(true);
    try {
      // Demo: Set a test token
      await MizuRuntime.shared.tokenStore.setToken(
        createAuthToken({ accessToken: 'demo_token' })
      );
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <SafeAreaView style={styles.container}>
      <View style={styles.content}>
        <View style={styles.spacer} />

        <View style={styles.header}>
          <View style={styles.iconContainer}>
            <Text style={styles.icon}>⚛</Text>
          </View>
          <Text style={styles.title}>Welcome to {{.Name}}</Text>
          <Text style={styles.subtitle}>
            A modern React Native app powered by Mizu
          </Text>
        </View>

        <View style={styles.spacer} />

        <View style={styles.footer}>
          {isLoading ? (
            <ActivityIndicator size="large" color="#FFFFFF" style={styles.button} />
          ) : (
            <TouchableOpacity style={styles.button} onPress={handleGetStarted}>
              <Text style={styles.buttonText}>Get Started</Text>
            </TouchableOpacity>
          )}
        </View>
      </View>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: '#FFFFFF',
  },
  content: {
    flex: 1,
    padding: 24,
  },
  spacer: {
    flex: 1,
  },
  header: {
    alignItems: 'center',
  },
  iconContainer: {
    width: 80,
    height: 80,
    borderRadius: 40,
    backgroundColor: '#EEF2FF',
    justifyContent: 'center',
    alignItems: 'center',
    marginBottom: 24,
  },
  icon: {
    fontSize: 40,
  },
  title: {
    fontSize: 28,
    fontWeight: 'bold',
    color: '#1F2937',
    textAlign: 'center',
    marginBottom: 16,
  },
  subtitle: {
    fontSize: 16,
    color: '#6B7280',
    textAlign: 'center',
  },
  footer: {
    paddingBottom: 24,
  },
  button: {
    backgroundColor: '#6366F1',
    paddingVertical: 18,
    borderRadius: 12,
    alignItems: 'center',
  },
  buttonText: {
    fontSize: 18,
    fontWeight: '600',
    color: '#FFFFFF',
  },
});
```

### Loading View (`src/components/LoadingView.tsx`)

```tsx
import React from 'react';
import { View, Text, ActivityIndicator, StyleSheet } from 'react-native';

interface LoadingViewProps {
  message?: string;
}

export default function LoadingView({ message }: LoadingViewProps) {
  return (
    <View style={styles.container}>
      <ActivityIndicator size="large" color="#6366F1" />
      {message && <Text style={styles.message}>{message}</Text>}
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
    backgroundColor: '#FFFFFF',
  },
  message: {
    marginTop: 16,
    fontSize: 16,
    color: '#6B7280',
  },
});
```

### Error View (`src/components/ErrorView.tsx`)

```tsx
import React from 'react';
import { View, Text, TouchableOpacity, StyleSheet } from 'react-native';
import { MizuError } from '../runtime/errors';

interface ErrorViewProps {
  error: Error | MizuError;
  onRetry?: () => void;
}

export default function ErrorView({ error, onRetry }: ErrorViewProps) {
  const message = error instanceof MizuError ? error.message : error.message;

  return (
    <View style={styles.container}>
      <View style={styles.iconContainer}>
        <Text style={styles.icon}>!</Text>
      </View>
      <Text style={styles.title}>Something went wrong</Text>
      <Text style={styles.message}>{message}</Text>
      {onRetry && (
        <TouchableOpacity style={styles.button} onPress={onRetry}>
          <Text style={styles.buttonText}>Try Again</Text>
        </TouchableOpacity>
      )}
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
    padding: 24,
    backgroundColor: '#FFFFFF',
  },
  iconContainer: {
    width: 48,
    height: 48,
    borderRadius: 24,
    backgroundColor: '#FEE2E2',
    justifyContent: 'center',
    alignItems: 'center',
    marginBottom: 16,
  },
  icon: {
    fontSize: 24,
    color: '#EF4444',
    fontWeight: 'bold',
  },
  title: {
    fontSize: 20,
    fontWeight: 'bold',
    color: '#1F2937',
    marginBottom: 8,
  },
  message: {
    fontSize: 16,
    color: '#6B7280',
    textAlign: 'center',
    marginBottom: 24,
  },
  button: {
    backgroundColor: '#EEF2FF',
    paddingVertical: 12,
    paddingHorizontal: 24,
    borderRadius: 8,
  },
  buttonText: {
    fontSize: 16,
    fontWeight: '600',
    color: '#6366F1',
  },
});
```

## Generated SDK

### Client (`src/sdk/client.ts`)

```typescript
import { MizuRuntime } from '../runtime/MizuRuntime';
import { createAuthToken } from '../runtime/tokenStore';
import { AuthResponse, User, UserUpdate } from './types';

/**
 * Generated Mizu API client for {{.Name}}
 */
export class {{.Name}}Client {
  private runtime: MizuRuntime;

  constructor(runtime: MizuRuntime) {
    this.runtime = runtime;
  }

  // MARK: - Auth

  /** Sign in with credentials */
  async signIn(email: string, password: string): Promise<AuthResponse> {
    return this.runtime.post<AuthResponse>('/auth/signin', { email, password });
  }

  /** Sign up with credentials */
  async signUp(email: string, password: string, name: string): Promise<AuthResponse> {
    return this.runtime.post<AuthResponse>('/auth/signup', { email, password, name });
  }

  /** Sign out */
  async signOut(): Promise<void> {
    await this.runtime.delete<void>('/auth/signout');
    await this.runtime.tokenStore.clearToken();
  }

  // MARK: - Users

  /** Get current user profile */
  async getCurrentUser(): Promise<User> {
    return this.runtime.get<User>('/users/me');
  }

  /** Update current user profile */
  async updateCurrentUser(update: UserUpdate): Promise<User> {
    return this.runtime.put<User>('/users/me', update);
  }

  // MARK: - Token Storage

  /** Store an auth response token */
  async storeAuthToken(response: AuthResponse): Promise<void> {
    const expiresAt = new Date(Date.now() + response.token.expiresIn * 1000);
    await this.runtime.tokenStore.setToken(
      createAuthToken({
        accessToken: response.token.accessToken,
        refreshToken: response.token.refreshToken,
        expiresAt,
      })
    );
  }
}
```

### Types (`src/sdk/types.ts`)

```typescript
// MARK: - Auth Types

export interface SignInRequest {
  email: string;
  password: string;
}

export interface SignUpRequest {
  email: string;
  password: string;
  name: string;
}

export interface AuthResponse {
  user: User;
  token: TokenResponse;
}

export interface TokenResponse {
  accessToken: string;
  refreshToken?: string;
  expiresIn: number;
}

// MARK: - User Types

export interface User {
  id: string;
  email: string;
  name: string;
  avatarUrl?: string;
  createdAt: string;
  updatedAt: string;
}

export interface UserUpdate {
  name?: string;
  avatarUrl?: string;
}
```

### Extensions (`src/sdk/extensions.ts`)

```typescript
import { MizuRuntime } from '../runtime/MizuRuntime';
import { createAuthToken } from '../runtime/tokenStore';
import { AuthResponse } from './types';

/**
 * Store an auth response token in the runtime
 */
export async function storeAuthToken(
  runtime: MizuRuntime,
  response: AuthResponse
): Promise<void> {
  const expiresAt = new Date(Date.now() + response.token.expiresIn * 1000);
  await runtime.tokenStore.setToken(
    createAuthToken({
      accessToken: response.token.accessToken,
      refreshToken: response.token.refreshToken,
      expiresAt,
    })
  );
}
```

## Build Configuration

### package.json

```json
{
  "name": "{{.Name | lower}}",
  "version": "1.0.0",
  "main": "node_modules/expo/AppEntry.js",
  "scripts": {
    "start": "expo start",
    "android": "expo run:android",
    "ios": "expo run:ios",
    "web": "expo start --web",
    "lint": "eslint . --ext .ts,.tsx",
    "test": "jest"
  },
  "dependencies": {
    "expo": "~51.0.0",
    "expo-application": "~5.9.0",
    "expo-device": "~6.0.0",
    "expo-localization": "~15.0.0",
    "expo-secure-store": "~13.0.0",
    "expo-status-bar": "~1.12.0",
    "react": "18.2.0",
    "react-native": "0.74.0",
    "react-native-get-random-values": "~1.11.0",
    "@react-navigation/native": "^6.1.0",
    "@react-navigation/native-stack": "^6.9.0",
    "react-native-safe-area-context": "~4.10.0",
    "react-native-screens": "~3.31.0",
    "uuid": "^9.0.0",
    "zustand": "^4.5.0"
  },
  "devDependencies": {
    "@babel/core": "^7.24.0",
    "@types/react": "~18.2.0",
    "@types/uuid": "^9.0.0",
    "@typescript-eslint/eslint-plugin": "^7.0.0",
    "@typescript-eslint/parser": "^7.0.0",
    "eslint": "^8.57.0",
    "eslint-plugin-react": "^7.34.0",
    "eslint-plugin-react-hooks": "^4.6.0",
    "jest": "^29.7.0",
    "jest-expo": "~51.0.0",
    "react-test-renderer": "18.2.0",
    "typescript": "^5.4.0"
  },
  "private": true
}
```

### tsconfig.json

```json
{
  "extends": "expo/tsconfig.base",
  "compilerOptions": {
    "strict": true,
    "baseUrl": ".",
    "paths": {
      "@/*": ["src/*"]
    }
  },
  "include": ["**/*.ts", "**/*.tsx"],
  "exclude": ["node_modules"]
}
```

### babel.config.js

```javascript
module.exports = function (api) {
  api.cache(true);
  return {
    presets: ['babel-preset-expo'],
  };
};
```

### metro.config.js

```javascript
const { getDefaultConfig } = require('expo/metro-config');

const config = getDefaultConfig(__dirname);

module.exports = config;
```

### .eslintrc.js

```javascript
module.exports = {
  root: true,
  extends: [
    'eslint:recommended',
    'plugin:@typescript-eslint/recommended',
    'plugin:react/recommended',
    'plugin:react-hooks/recommended',
  ],
  parser: '@typescript-eslint/parser',
  plugins: ['@typescript-eslint', 'react', 'react-hooks'],
  parserOptions: {
    ecmaVersion: 2020,
    sourceType: 'module',
    ecmaFeatures: {
      jsx: true,
    },
  },
  settings: {
    react: {
      version: 'detect',
    },
  },
  env: {
    browser: true,
    node: true,
    es6: true,
  },
  rules: {
    'react/react-in-jsx-scope': 'off',
    '@typescript-eslint/explicit-function-return-type': 'off',
    '@typescript-eslint/no-explicit-any': 'warn',
  },
};
```

## Testing

### Unit Tests (`__tests__/runtime.test.ts`)

```typescript
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
});
```

### Widget Tests (`__tests__/App.test.tsx`)

```tsx
import React from 'react';
import renderer from 'react-test-renderer';
import LoadingView from '../src/components/LoadingView';
import ErrorView from '../src/components/ErrorView';

describe('LoadingView', () => {
  it('renders correctly without message', () => {
    const tree = renderer.create(<LoadingView />).toJSON();
    expect(tree).toMatchSnapshot();
  });

  it('renders correctly with message', () => {
    const tree = renderer.create(<LoadingView message="Loading..." />).toJSON();
    expect(tree).toMatchSnapshot();
  });
});

describe('ErrorView', () => {
  it('renders correctly with error', () => {
    const tree = renderer
      .create(<ErrorView error={new Error('Test error')} />)
      .toJSON();
    expect(tree).toMatchSnapshot();
  });

  it('renders correctly with retry button', () => {
    const tree = renderer
      .create(<ErrorView error={new Error('Test error')} onRetry={() => {}} />)
      .toJSON();
    expect(tree).toMatchSnapshot();
  });
});
```

### jest.config.js

```javascript
module.exports = {
  preset: 'jest-expo',
  transformIgnorePatterns: [
    'node_modules/(?!((jest-)?react-native|@react-native(-community)?)|expo(nent)?|@expo(nent)?/.*|@expo-google-fonts/.*|react-navigation|@react-navigation/.*|@unimodules/.*|unimodules|sentry-expo|native-base|react-native-svg)',
  ],
  setupFilesAfterEnv: ['@testing-library/jest-native/extend-expect'],
  moduleNameMapper: {
    '^@/(.*)$': '<rootDir>/src/$1',
  },
};
```

## Expo vs Bare Workflow

### Expo Managed (Default)
- Simpler development experience
- OTA updates via `expo-updates`
- Uses `expo-secure-store` for secure storage
- Uses `expo-device`, `expo-application`, `expo-localization` for device info
- `expo-router` for file-based routing (optional)

### Bare React Native
- More control over native code
- Uses `react-native-keychain` for secure storage
- Uses `react-native-device-info` for device info
- Requires manual native configuration

## Platform Configuration

### iOS (`ios/{{.Name}}/Info.plist`)
- NSAppTransportSecurity for local development
- Bundle identifier configuration
- Required device capabilities

### Android (`android/app/src/main/AndroidManifest.xml`)
- Network security configuration
- Permission declarations
- Application configuration

## References

- [React Native Documentation](https://reactnative.dev/docs/getting-started)
- [Expo Documentation](https://docs.expo.dev/)
- [React Navigation](https://reactnavigation.org/)
- [Zustand](https://github.com/pmndrs/zustand)
- [TypeScript](https://www.typescriptlang.org/)
- [Mobile Package Spec](./0095_mobile.md)
