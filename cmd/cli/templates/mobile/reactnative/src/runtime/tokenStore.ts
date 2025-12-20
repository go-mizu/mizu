import * as SecureStore from 'expo-secure-store';

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
