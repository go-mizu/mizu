/**
 * Mock KV namespace for testing.
 * Provides an in-memory implementation of the KVNamespace interface.
 */
import { vi } from 'vitest';

export interface MockKVData {
  [key: string]: string;
}

export function createMockKV(initialData: MockKVData = {}): KVNamespace {
  const store = new Map(Object.entries(initialData));

  return {
    get: vi.fn(async (key: string, options?: { type?: string }) => {
      const value = store.get(key);
      if (value === undefined) return null;

      if (options?.type === 'json') {
        try {
          return JSON.parse(value);
        } catch {
          return null;
        }
      }
      return value;
    }),

    put: vi.fn(async (key: string, value: string, _options?: KVNamespacePutOptions) => {
      store.set(key, value);
    }),

    delete: vi.fn(async (key: string) => {
      store.delete(key);
    }),

    list: vi.fn(async (options?: { prefix?: string; limit?: number; cursor?: string }) => {
      const prefix = options?.prefix ?? '';
      const limit = options?.limit ?? 1000;

      const keys = [...store.keys()]
        .filter((k) => k.startsWith(prefix))
        .slice(0, limit)
        .map((name) => ({ name }));

      return {
        keys,
        list_complete: keys.length < limit,
        cursor: '',
        cacheStatus: null,
      };
    }),

    getWithMetadata: vi.fn(async (key: string, _options?: { type?: string }) => {
      const value = store.get(key);
      return {
        value: value ?? null,
        metadata: null,
        cacheStatus: null,
      };
    }),
  } as unknown as KVNamespace;
}

/**
 * Create a mock KV with pre-populated cache data.
 */
export function createMockKVWithCache(cacheData: Record<string, unknown>): KVNamespace {
  const initialData: MockKVData = {};

  for (const [key, value] of Object.entries(cacheData)) {
    initialData[`cache:${key}`] = JSON.stringify(value);
  }

  return createMockKV(initialData);
}
