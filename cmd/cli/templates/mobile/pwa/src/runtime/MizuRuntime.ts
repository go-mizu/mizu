import { Transport, HttpTransport, TransportResponse } from './transport';
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
  private _isOnline: boolean = typeof navigator !== 'undefined' ? navigator.onLine : true;
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
    if (typeof window === 'undefined') return;

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
        await this._request({
          method: req.method,
          path: req.path,
          body: req.body,
          headers: req.headers,
        });
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
