import * as SQLite from 'expo-sqlite'
import * as SecureStore from 'expo-secure-store'

let db: SQLite.SQLiteDatabase | null = null

async function getDB(): Promise<SQLite.SQLiteDatabase> {
  if (db) return db
  db = await SQLite.openDatabaseAsync('xcache.db')
  await db.execAsync(`
    CREATE TABLE IF NOT EXISTS cache (
      key TEXT PRIMARY KEY,
      data TEXT NOT NULL,
      created_at INTEGER NOT NULL,
      expires_at INTEGER NOT NULL
    );
    CREATE INDEX IF NOT EXISTS idx_cache_expires ON cache(expires_at);
    CREATE TABLE IF NOT EXISTS search_history (
      query TEXT PRIMARY KEY,
      last_used INTEGER NOT NULL
    );
  `)
  return db
}

export async function cacheGet<T>(key: string): Promise<T | null> {
  const d = await getDB()
  const now = Math.floor(Date.now() / 1000)
  const row = await d.getFirstAsync<{ data: string }>(
    'SELECT data FROM cache WHERE key = ? AND expires_at > ?',
    [key, now]
  )
  if (!row) return null
  return JSON.parse(row.data) as T
}

export async function cacheSet<T>(key: string, value: T, ttlSeconds: number): Promise<void> {
  const d = await getDB()
  const now = Math.floor(Date.now() / 1000)
  await d.runAsync(
    'INSERT OR REPLACE INTO cache (key, data, created_at, expires_at) VALUES (?, ?, ?, ?)',
    [key, JSON.stringify(value), now, now + ttlSeconds]
  )
}

// Get stale data even if expired (for stale-while-revalidate)
export async function cacheGetStale<T>(key: string): Promise<T | null> {
  const d = await getDB()
  const row = await d.getFirstAsync<{ data: string }>(
    'SELECT data FROM cache WHERE key = ?',
    [key]
  )
  if (!row) return null
  return JSON.parse(row.data) as T
}

export async function clearCache(): Promise<void> {
  const d = await getDB()
  await d.execAsync('DELETE FROM cache')
}

// Search history
export async function addSearchHistory(query: string): Promise<void> {
  const d = await getDB()
  const now = Math.floor(Date.now() / 1000)
  await d.runAsync(
    'INSERT OR REPLACE INTO search_history (query, last_used) VALUES (?, ?)',
    [query, now]
  )
}

export async function getSearchHistory(): Promise<string[]> {
  const d = await getDB()
  const rows = await d.getAllAsync<{ query: string }>(
    'SELECT query FROM search_history ORDER BY last_used DESC LIMIT 20'
  )
  return rows.map(r => r.query)
}

export async function clearSearchHistory(): Promise<void> {
  const d = await getDB()
  await d.execAsync('DELETE FROM search_history')
}

// Cache statistics
export interface CacheStats {
  entryCount: number
  approximateSizeKB: number
}

export async function getCacheStats(): Promise<CacheStats> {
  const d = await getDB()
  const countRow = await d.getFirstAsync<{ count: number }>(
    'SELECT COUNT(*) as count FROM cache'
  )
  const sizeRow = await d.getFirstAsync<{ total: number }>(
    'SELECT COALESCE(SUM(LENGTH(data)), 0) as total FROM cache'
  )
  return {
    entryCount: countRow?.count ?? 0,
    approximateSizeKB: Math.round((sizeRow?.total ?? 0) / 1024),
  }
}

// Prune expired entries
export async function pruneExpiredCache(): Promise<number> {
  const d = await getDB()
  const now = Math.floor(Date.now() / 1000)
  const result = await d.runAsync('DELETE FROM cache WHERE expires_at <= ?', [now])
  return result.changes
}

// Credentials
export interface Credentials {
  authToken: string
  ct0: string
  bearerToken: string
}

export async function getCredentials(): Promise<Credentials> {
  const authToken = await SecureStore.getItemAsync('x_auth_token') || ''
  const ct0 = await SecureStore.getItemAsync('x_ct0') || ''
  const bearerToken = await SecureStore.getItemAsync('x_bearer_token') || ''
  return { authToken, ct0, bearerToken }
}

export async function setCredentials(creds: Credentials): Promise<void> {
  await SecureStore.setItemAsync('x_auth_token', creds.authToken)
  await SecureStore.setItemAsync('x_ct0', creds.ct0)
  await SecureStore.setItemAsync('x_bearer_token', creds.bearerToken)
}

export async function hasCredentials(): Promise<boolean> {
  const creds = await getCredentials()
  return !!(creds.authToken && creds.ct0 && creds.bearerToken)
}
