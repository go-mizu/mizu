/**
 * Storage abstraction layer — three backends: memory, KV, D1.
 *
 * Two levels:
 * 1. StorageBackend — generic key-value interface (memory, KV, D1-kv)
 * 2. Domain stores — typed interfaces (AccountStore, ThreadStore, etc.)
 *
 * Domain stores have D1-optimized implementations (direct SQL) and
 * generic fallbacks that wrap StorageBackend (for memory/KV).
 */

import { CACHE_TTL, MAX_THREADS } from './config'
import type {
  Account, AccountSummary, SessionState, RegistrationLog, OGData,
  Thread, ThreadMessage, ThreadSummary, ThreadIndex,
} from './types'

// ============================================================
// StorageBackend — generic key-value (kept for backward compat)
// ============================================================

export interface StorageBackend {
  readonly name: string
  get<T>(key: string): Promise<T | null>
  set<T>(key: string, value: T, ttlSeconds?: number): Promise<void>
  delete(key: string): Promise<void>
  list(prefix: string): Promise<string[]>
}

// --- Memory Storage ---

const memStore = new Map<string, { value: string; expiresAt: number | null }>()

export class MemoryStorage implements StorageBackend {
  readonly name = 'memory'

  async get<T>(key: string): Promise<T | null> {
    const entry = memStore.get(key)
    if (!entry) return null
    if (entry.expiresAt && Date.now() > entry.expiresAt) {
      memStore.delete(key)
      return null
    }
    return JSON.parse(entry.value) as T
  }

  async set<T>(key: string, value: T, ttlSeconds?: number): Promise<void> {
    const expiresAt = ttlSeconds && ttlSeconds > 0 ? Date.now() + ttlSeconds * 1000 : null
    memStore.set(key, { value: JSON.stringify(value), expiresAt })
  }

  async delete(key: string): Promise<void> {
    memStore.delete(key)
  }

  async list(prefix: string): Promise<string[]> {
    const keys: string[] = []
    const now = Date.now()
    for (const [key, entry] of memStore) {
      if (key.startsWith(prefix)) {
        if (entry.expiresAt && now > entry.expiresAt) {
          memStore.delete(key)
        } else {
          keys.push(key)
        }
      }
    }
    return keys
  }
}

// --- KV Storage ---

export class KVStorage implements StorageBackend {
  readonly name = 'kv'
  private kv: KVNamespace

  constructor(kv: KVNamespace) {
    this.kv = kv
  }

  async get<T>(key: string): Promise<T | null> {
    const val = await this.kv.get(key, 'text')
    if (!val) return null
    return JSON.parse(val) as T
  }

  async set<T>(key: string, value: T, ttlSeconds?: number): Promise<void> {
    try {
      const opts: KVNamespacePutOptions = {}
      if (ttlSeconds && ttlSeconds > 0) opts.expirationTtl = ttlSeconds
      await this.kv.put(key, JSON.stringify(value), opts)
    } catch (e) {
      console.error(`[KV] set "${key}" failed:`, e instanceof Error ? e.message : String(e))
    }
  }

  async delete(key: string): Promise<void> {
    await this.kv.delete(key)
  }

  async list(prefix: string): Promise<string[]> {
    const result = await this.kv.list({ prefix })
    return result.keys.map(k => k.name)
  }
}

// --- D1 Storage (legacy kv table) ---

export class D1Storage implements StorageBackend {
  readonly name = 'd1'
  private db: D1Database

  constructor(db: D1Database) {
    this.db = db
  }

  async get<T>(key: string): Promise<T | null> {
    const row = await this.db
      .prepare('SELECT value, expires_at FROM kv WHERE key = ?')
      .bind(key)
      .first<{ value: string; expires_at: number | null }>()
    if (!row) return null
    if (row.expires_at && Date.now() > row.expires_at) {
      await this.db.prepare('DELETE FROM kv WHERE key = ?').bind(key).run()
      return null
    }
    return JSON.parse(row.value) as T
  }

  async set<T>(key: string, value: T, ttlSeconds?: number): Promise<void> {
    const expiresAt = ttlSeconds && ttlSeconds > 0 ? Date.now() + ttlSeconds * 1000 : null
    await this.db
      .prepare('INSERT OR REPLACE INTO kv (key, value, expires_at) VALUES (?, ?, ?)')
      .bind(key, JSON.stringify(value), expiresAt)
      .run()
  }

  async delete(key: string): Promise<void> {
    await this.db.prepare('DELETE FROM kv WHERE key = ?').bind(key).run()
  }

  async list(prefix: string): Promise<string[]> {
    const now = Date.now()
    const rows = await this.db
      .prepare('SELECT key FROM kv WHERE key LIKE ? AND (expires_at IS NULL OR expires_at > ?)')
      .bind(prefix + '%', now)
      .all<{ key: string }>()
    return rows.results.map(r => r.key)
  }
}

// --- Factory (legacy) ---

const memoryStorage = new MemoryStorage()

export function getStorage(env: { KV?: KVNamespace; DB?: D1Database }, override?: string): StorageBackend {
  if (override === 'memory') return memoryStorage
  if (override === 'kv' && env.KV) return new KVStorage(env.KV)
  if (override === 'd1' && env.DB) return new D1Storage(env.DB)
  if (env.DB) return new D1Storage(env.DB)
  if (env.KV) return new KVStorage(env.KV)
  return memoryStorage
}

// ============================================================
// Domain Store Interfaces
// ============================================================

export interface AccountStore {
  readonly name: string
  addAccount(account: Account): Promise<void>
  getAccount(id: string): Promise<Account | null>
  deleteAccount(id: string): Promise<boolean>
  deleteAll(): Promise<number>
  listAccounts(): Promise<AccountSummary[]>
  getFailedAccounts(): Promise<Account[]>
  countActive(): Promise<number>
  recordUsage(id: string): Promise<void>
  markFailed(id: string, reason: string): Promise<void>
  disable(id: string, reason: string): Promise<void>
  restore(id: string, session: SessionState, proQueries: number): Promise<void>
  getAndIncrementRobin(): Promise<number>
  resetRobin(): Promise<void>
  appendLog(entry: RegistrationLog): Promise<void>
  getLogs(limit?: number): Promise<RegistrationLog[]>
  tryLock(ttlSeconds: number): Promise<boolean>
  unlock(): Promise<void>
}

export interface ThreadStore {
  readonly name: string
  createThread(thread: Thread): Promise<void>
  getThread(id: string): Promise<Thread | null>
  addFollowUp(threadId: string, userMsg: ThreadMessage, assistantMsg: ThreadMessage): Promise<Thread | null>
  deleteThread(id: string): Promise<boolean>
  listThreads(limit?: number): Promise<ThreadSummary[]>
}

export interface SessionStore {
  readonly name: string
  getPool(): Promise<SessionState[]>
  addToPool(session: SessionState): Promise<void>
  savePool(sessions: SessionState[]): Promise<void>
  getLegacy(): Promise<SessionState | null>
  saveLegacy(session: SessionState): Promise<void>
}

export interface OGStore {
  readonly name: string
  get(url: string): Promise<OGData | null>
  set(url: string, data: OGData, ttlSeconds?: number): Promise<void>
}

// ============================================================
// Generic Store Implementations (wrap StorageBackend)
// ============================================================

const MAX_LOG_ENTRIES = 100

export class GenericAccountStore implements AccountStore {
  readonly name: string
  constructor(private store: StorageBackend) { this.name = store.name }

  async addAccount(account: Account): Promise<void> {
    await this.store.set(`account:${account.id}`, account, CACHE_TTL.account)
    const index = await this.getIndex()
    index.push({ id: account.id, email: account.email, proQueries: account.proQueries, status: account.status, createdAt: account.createdAt })
    await this.store.set('accounts:index', index)
  }

  async getAccount(id: string): Promise<Account | null> {
    return this.store.get<Account>(`account:${id}`)
  }

  async deleteAccount(id: string): Promise<boolean> {
    const account = await this.getAccount(id)
    if (!account) return false
    await this.store.delete(`account:${id}`)
    const index = await this.getIndex()
    await this.store.set('accounts:index', index.filter(a => a.id !== id))
    return true
  }

  async deleteAll(): Promise<number> {
    const index = await this.getIndex()
    for (const a of index) await this.store.delete(`account:${a.id}`)
    await this.store.set('accounts:index', [])
    await this.store.delete('accounts:robin')
    return index.length
  }

  async listAccounts(): Promise<AccountSummary[]> {
    return this.getIndex()
  }

  async getFailedAccounts(): Promise<Account[]> {
    const index = await this.getIndex()
    const accounts: Account[] = []
    for (const f of index.filter(a => a.status === 'failed')) {
      const acc = await this.getAccount(f.id)
      if (acc) accounts.push(acc)
    }
    return accounts
  }

  async countActive(): Promise<number> {
    const index = await this.getIndex()
    return index.filter(a => a.status === 'active').length
  }

  async recordUsage(id: string): Promise<void> {
    const account = await this.getAccount(id)
    if (!account) return
    account.proQueries = Math.max(0, account.proQueries - 1)
    account.lastUsedAt = new Date().toISOString()
    account.totalQueriesUsed = (account.totalQueriesUsed || 0) + 1
    if (account.proQueries <= 0) {
      account.status = 'exhausted'
      account.disabledAt = new Date().toISOString()
      account.disableReason = 'Pro queries exhausted'
    }
    await this.store.set(`account:${id}`, account, CACHE_TTL.account)
    await this.updateIndex(id, { proQueries: account.proQueries, status: account.status })
  }

  async markFailed(id: string, _reason: string): Promise<void> {
    const account = await this.getAccount(id)
    if (!account) return
    account.status = 'failed'
    await this.store.set(`account:${id}`, account, CACHE_TTL.account)
    await this.updateIndex(id, { status: 'failed' })
  }

  async disable(id: string, reason: string): Promise<void> {
    const account = await this.getAccount(id)
    if (!account) return
    account.status = 'disabled'
    account.disabledAt = new Date().toISOString()
    account.disableReason = reason
    await this.store.set(`account:${id}`, account, CACHE_TTL.account)
    await this.updateIndex(id, { status: 'disabled' })
  }

  async restore(id: string, session: SessionState, proQueries: number): Promise<void> {
    const account = await this.getAccount(id)
    if (!account) return
    account.session = session
    account.proQueries = proQueries
    account.status = 'active'
    account.lastUsedAt = new Date().toISOString()
    delete account.disabledAt
    delete account.disableReason
    await this.store.set(`account:${id}`, account, CACHE_TTL.account)
    await this.updateIndex(id, { proQueries, status: 'active' })
  }

  async getAndIncrementRobin(): Promise<number> {
    const robin = (await this.store.get<number>('accounts:robin')) || 0
    await this.store.set('accounts:robin', robin + 1)
    return robin
  }

  async resetRobin(): Promise<void> {
    await this.store.delete('accounts:robin')
  }

  async appendLog(entry: RegistrationLog): Promise<void> {
    const logs = (await this.store.get<RegistrationLog[]>('accounts:log')) || []
    logs.push(entry)
    await this.store.set('accounts:log', logs.slice(-MAX_LOG_ENTRIES))
  }

  async getLogs(limit: number = MAX_LOG_ENTRIES): Promise<RegistrationLog[]> {
    const logs = (await this.store.get<RegistrationLog[]>('accounts:log')) || []
    return logs.slice(-limit)
  }

  async tryLock(ttlSeconds: number): Promise<boolean> {
    const existing = await this.store.get<string>('accounts:lock')
    if (existing) {
      const lockTime = new Date(existing).getTime()
      if (Date.now() - lockTime < ttlSeconds * 1000) return false
    }
    await this.store.set('accounts:lock', new Date().toISOString(), ttlSeconds)
    return true
  }

  async unlock(): Promise<void> {
    await this.store.delete('accounts:lock')
  }

  private async getIndex(): Promise<AccountSummary[]> {
    return (await this.store.get<AccountSummary[]>('accounts:index')) || []
  }

  private async updateIndex(id: string, updates: Partial<AccountSummary>): Promise<void> {
    const index = await this.getIndex()
    const entry = index.find(a => a.id === id)
    if (entry) {
      Object.assign(entry, updates)
      await this.store.set('accounts:index', index)
    }
  }
}

export class GenericThreadStore implements ThreadStore {
  readonly name: string
  constructor(private store: StorageBackend) { this.name = store.name }

  async createThread(thread: Thread): Promise<void> {
    await this.store.set(`thread:${thread.id}`, thread, CACHE_TTL.thread)
    const index = await this.getIndex()
    index.threads.unshift({
      id: thread.id, title: thread.title, mode: thread.mode, model: thread.model,
      messageCount: thread.messages.length, createdAt: thread.createdAt, updatedAt: thread.updatedAt,
    })
    if (index.threads.length > MAX_THREADS) index.threads = index.threads.slice(0, MAX_THREADS)
    await this.store.set('threads:recent', index, CACHE_TTL.threadIndex)
  }

  async getThread(id: string): Promise<Thread | null> {
    return this.store.get<Thread>(`thread:${id}`)
  }

  async addFollowUp(threadId: string, userMsg: ThreadMessage, assistantMsg: ThreadMessage): Promise<Thread | null> {
    const thread = await this.getThread(threadId)
    if (!thread) return null
    thread.messages.push(userMsg, assistantMsg)
    thread.updatedAt = userMsg.createdAt
    await this.store.set(`thread:${threadId}`, thread, CACHE_TTL.thread)
    const index = await this.getIndex()
    index.threads = index.threads.filter(t => t.id !== threadId)
    index.threads.unshift({
      id: thread.id, title: thread.title, mode: thread.mode, model: thread.model,
      messageCount: thread.messages.length, createdAt: thread.createdAt, updatedAt: thread.updatedAt,
    })
    await this.store.set('threads:recent', index, CACHE_TTL.threadIndex)
    return thread
  }

  async deleteThread(id: string): Promise<boolean> {
    const thread = await this.getThread(id)
    if (!thread) return false
    await this.store.delete(`thread:${id}`)
    const index = await this.getIndex()
    index.threads = index.threads.filter(t => t.id !== id)
    await this.store.set('threads:recent', index, CACHE_TTL.threadIndex)
    return true
  }

  async listThreads(limit?: number): Promise<ThreadSummary[]> {
    const index = await this.getIndex()
    return limit ? index.threads.slice(0, limit) : index.threads
  }

  private async getIndex(): Promise<ThreadIndex> {
    return (await this.store.get<ThreadIndex>('threads:recent')) || { threads: [] }
  }
}

export class GenericSessionStore implements SessionStore {
  readonly name: string
  constructor(private store: StorageBackend) { this.name = store.name }

  async getPool(): Promise<SessionState[]> {
    return (await this.store.get<SessionState[]>('sessions:pool')) || []
  }

  async addToPool(session: SessionState): Promise<void> {
    const pool = await this.getPool()
    pool.push(session)
    await this.store.set('sessions:pool', pool.slice(-5), CACHE_TTL.session)
  }

  async savePool(sessions: SessionState[]): Promise<void> {
    await this.store.set('sessions:pool', sessions.slice(-5), CACHE_TTL.session)
  }

  async getLegacy(): Promise<SessionState | null> {
    return this.store.get<SessionState>('session:anon')
  }

  async saveLegacy(session: SessionState): Promise<void> {
    await this.store.set('session:anon', session, CACHE_TTL.session)
  }
}

export class GenericOGStore implements OGStore {
  readonly name: string
  constructor(private store: StorageBackend) { this.name = store.name }

  async get(url: string): Promise<OGData | null> {
    return this.store.get<OGData>(`og:${url}`)
  }

  async set(url: string, data: OGData, ttlSeconds?: number): Promise<void> {
    await this.store.set(`og:${url}`, data, ttlSeconds)
  }
}

// ============================================================
// Domain Store Factories
// ============================================================

import { D1AccountStore, D1ThreadStore, D1SessionStore, D1OGStore } from './d1-stores'

export function getAccountStore(env: { DB?: D1Database; KV?: KVNamespace }, override?: string): AccountStore {
  if (override === 'memory') return new GenericAccountStore(memoryStorage)
  if (override === 'kv' && env.KV) return new GenericAccountStore(new KVStorage(env.KV))
  if (env.DB) return new D1AccountStore(env.DB)
  if (env.KV) return new GenericAccountStore(new KVStorage(env.KV))
  return new GenericAccountStore(memoryStorage)
}

export function getThreadStore(env: { DB?: D1Database; KV?: KVNamespace }, override?: string): ThreadStore {
  if (override === 'memory') return new GenericThreadStore(memoryStorage)
  if (override === 'kv' && env.KV) return new GenericThreadStore(new KVStorage(env.KV))
  if (env.DB) return new D1ThreadStore(env.DB)
  if (env.KV) return new GenericThreadStore(new KVStorage(env.KV))
  return new GenericThreadStore(memoryStorage)
}

export function getSessionStore(env: { DB?: D1Database; KV?: KVNamespace }, override?: string): SessionStore {
  if (override === 'memory') return new GenericSessionStore(memoryStorage)
  if (override === 'kv' && env.KV) return new GenericSessionStore(new KVStorage(env.KV))
  if (env.DB) return new D1SessionStore(env.DB)
  if (env.KV) return new GenericSessionStore(new KVStorage(env.KV))
  return new GenericSessionStore(memoryStorage)
}

export function getOGStore(env: { DB?: D1Database; KV?: KVNamespace }, override?: string): OGStore {
  if (override === 'memory') return new GenericOGStore(memoryStorage)
  if (override === 'kv' && env.KV) return new GenericOGStore(new KVStorage(env.KV))
  if (env.DB) return new D1OGStore(env.DB)
  if (env.KV) return new GenericOGStore(new KVStorage(env.KV))
  return new GenericOGStore(memoryStorage)
}
