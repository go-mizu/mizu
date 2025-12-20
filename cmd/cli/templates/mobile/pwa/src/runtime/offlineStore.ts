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
