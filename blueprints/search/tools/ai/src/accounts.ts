/**
 * Account manager — KV-based account storage with round-robin rotation + debug logging.
 * Ported from pkg/dcrawler/perplexity/accounts.go
 *
 * Everything is internal — no public APIs. Registration happens in background via waitUntil().
 * Full debug logs stored in KV for inspection via wrangler kv:key get.
 *
 * Storage keys:
 *   account:{id}       → Account JSON (with session)
 *   accounts:index     → AccountSummary[]
 *   accounts:robin     → number (round-robin pointer)
 *   accounts:log       → RegistrationLog[] (last 50 entries)
 *   accounts:lock      → string (ISO timestamp, prevents concurrent registration)
 */

import { Cache } from './cache'
import { CACHE_TTL } from './config'
import type { Account, AccountSummary, SessionState } from './types'

const DEFAULT_PRO_QUERIES = 5
const MAX_LOG_ENTRIES = 50
const REGISTRATION_LOCK_TTL = 60 // seconds — prevent overlapping registrations
const MIN_ACTIVE_ACCOUNTS = 1 // trigger background registration when below this

export interface RegistrationLog {
  timestamp: string
  event: 'start' | 'email_created' | 'signin_sent' | 'email_received' | 'auth_complete' | 'account_saved' | 'error'
  message: string
  provider?: string
  email?: string
  accountId?: string
  durationMs?: number
  error?: string
}

function nanoid(len: number = 8): string {
  const chars = 'abcdefghijklmnopqrstuvwxyz0123456789'
  let id = ''
  const bytes = crypto.getRandomValues(new Uint8Array(len))
  for (let i = 0; i < len; i++) id += chars[bytes[i] % chars.length]
  return id
}

export class AccountManager {
  private cache: Cache

  constructor(kv: KVNamespace) {
    this.cache = new Cache(kv)
  }

  // --- Logging ---

  /** Append a log entry. Keeps last 50 entries. */
  async log(entry: RegistrationLog): Promise<void> {
    const logs = (await this.cache.get<RegistrationLog[]>('accounts:log')) || []
    logs.push(entry)
    // Trim to last MAX_LOG_ENTRIES
    const trimmed = logs.slice(-MAX_LOG_ENTRIES)
    await this.cache.set('accounts:log', trimmed)
  }

  /** Get all registration logs. */
  async getLogs(): Promise<RegistrationLog[]> {
    return (await this.cache.get<RegistrationLog[]>('accounts:log')) || []
  }

  // --- Registration lock ---

  /** Try to acquire registration lock. Returns true if acquired. */
  async tryLock(): Promise<boolean> {
    const existing = await this.cache.get<string>('accounts:lock')
    if (existing) {
      // Check if lock is stale (older than LOCK_TTL)
      const lockTime = new Date(existing).getTime()
      if (Date.now() - lockTime < REGISTRATION_LOCK_TTL * 1000) {
        return false // lock is still active
      }
    }
    await this.cache.set('accounts:lock', new Date().toISOString(), REGISTRATION_LOCK_TTL)
    return true
  }

  /** Release registration lock. */
  async unlock(): Promise<void> {
    await this.cache.delete('accounts:lock')
  }

  // --- Account CRUD ---

  /** Add a new account. Returns account ID. */
  async addAccount(email: string, session: SessionState, proQueries: number = DEFAULT_PRO_QUERIES): Promise<string> {
    const id = nanoid()
    const now = new Date().toISOString()

    const account: Account = {
      id,
      email,
      session,
      proQueries,
      status: 'active',
      createdAt: now,
      lastUsedAt: now,
    }

    await this.cache.set(`account:${id}`, account, CACHE_TTL.account)

    // Update index
    const index = await this.getIndex()
    index.push({
      id,
      email,
      proQueries,
      status: 'active',
      createdAt: now,
    })
    await this.cache.set('accounts:index', index)

    return id
  }

  /** Get the next active account via round-robin. Returns null if none available. */
  async nextAccount(): Promise<Account | null> {
    const index = await this.getIndex()
    const active = index.filter(a => a.status === 'active')
    if (active.length === 0) return null

    let robin = (await this.cache.get<number>('accounts:robin')) || 0
    robin = robin % active.length
    const summary = active[robin]
    await this.cache.set('accounts:robin', robin + 1)

    return this.cache.get<Account>(`account:${summary.id}`)
  }

  /** Record usage after a successful pro query. Decrements proQueries. */
  async recordUsage(accountId: string): Promise<void> {
    const account = await this.cache.get<Account>(`account:${accountId}`)
    if (!account) return

    account.proQueries = Math.max(0, account.proQueries - 1)
    account.lastUsedAt = new Date().toISOString()

    if (account.proQueries <= 0) {
      account.status = 'exhausted'
    }

    await this.cache.set(`account:${accountId}`, account, CACHE_TTL.account)
    await this.updateIndex(accountId, { proQueries: account.proQueries, status: account.status })
  }

  /** Mark account as failed. */
  async markFailed(accountId: string, reason: string): Promise<void> {
    const account = await this.cache.get<Account>(`account:${accountId}`)
    if (!account) return

    account.status = 'failed'
    await this.cache.set(`account:${accountId}`, account, CACHE_TTL.account)
    await this.updateIndex(accountId, { status: 'failed' })
    await this.log({
      timestamp: new Date().toISOString(),
      event: 'error',
      message: `Account ${accountId} marked failed: ${reason}`,
      accountId,
      error: reason,
    })
  }

  /** Check if we need more accounts. */
  async needsRegistration(): Promise<boolean> {
    const index = await this.getIndex()
    const active = index.filter(a => a.status === 'active')
    return active.length < MIN_ACTIVE_ACCOUNTS
  }

  /** List all accounts (summary, no session data). */
  async listAccounts(): Promise<{ accounts: AccountSummary[]; active: number; total: number }> {
    const index = await this.getIndex()
    const active = index.filter(a => a.status === 'active').length
    return { accounts: index, active, total: index.length }
  }

  /** Delete an account. */
  async deleteAccount(accountId: string): Promise<boolean> {
    const account = await this.cache.get<Account>(`account:${accountId}`)
    if (!account) return false

    await this.cache.delete(`account:${accountId}`)
    const index = await this.getIndex()
    const filtered = index.filter(a => a.id !== accountId)
    await this.cache.set('accounts:index', filtered)
    return true
  }

  private async getIndex(): Promise<AccountSummary[]> {
    return (await this.cache.get<AccountSummary[]>('accounts:index')) || []
  }

  private async updateIndex(accountId: string, updates: Partial<AccountSummary>): Promise<void> {
    const index = await this.getIndex()
    const entry = index.find(a => a.id === accountId)
    if (entry) {
      Object.assign(entry, updates)
      await this.cache.set('accounts:index', index)
    }
  }
}
