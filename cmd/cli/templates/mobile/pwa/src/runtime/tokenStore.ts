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
