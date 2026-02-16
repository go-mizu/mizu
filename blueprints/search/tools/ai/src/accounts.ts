/**
 * Account manager — business logic layer over AccountStore.
 *
 * Handles encryption, round-robin rotation, orchestration of store calls + logging.
 * The actual data access is delegated to AccountStore implementations
 * (D1AccountStore for SQL, GenericAccountStore for memory/KV).
 */

import type { AccountStore } from './storage'
import { encrypt, decrypt } from './crypto'
import type { Account, AccountSummary, SessionState, RegistrationLog } from './types'

const DEFAULT_PRO_QUERIES = 5
const REGISTRATION_LOCK_TTL = 60
const MIN_ACTIVE_ACCOUNTS = 1

function nanoid(len: number = 8): string {
  const chars = 'abcdefghijklmnopqrstuvwxyz0123456789'
  let id = ''
  const bytes = crypto.getRandomValues(new Uint8Array(len))
  for (let i = 0; i < len; i++) id += chars[bytes[i] % chars.length]
  return id
}

export class AccountManager {
  private store: AccountStore
  private secret: string

  constructor(store: AccountStore, secret: string = 'default-dev-secret') {
    this.store = store
    this.secret = secret
  }

  get backend(): string { return this.store.name }

  // --- Logging ---

  async log(entry: RegistrationLog): Promise<void> {
    await this.store.appendLog(entry)
  }

  async getLogs(): Promise<RegistrationLog[]> {
    return this.store.getLogs()
  }

  // --- Lock ---

  async tryLock(): Promise<boolean> {
    return this.store.tryLock(REGISTRATION_LOCK_TTL)
  }

  async unlock(): Promise<void> {
    await this.store.unlock()
  }

  // --- Account CRUD ---

  async addAccount(
    email: string,
    emailProvider: string,
    emailPassword: string,
    session: SessionState,
    proQueries: number = DEFAULT_PRO_QUERIES,
  ): Promise<string> {
    const id = nanoid()
    const now = new Date().toISOString()
    const emailPasswordEnc = emailPassword ? await encrypt(emailPassword, this.secret) : ''

    const account: Account = {
      id,
      email,
      emailProvider,
      emailPasswordEnc,
      session,
      proQueries,
      status: 'active',
      createdAt: now,
      lastUsedAt: now,
      totalQueriesUsed: 0,
    }

    await this.store.addAccount(account)
    return id
  }

  async nextAccount(): Promise<Account | null> {
    const all = await this.store.listAccounts()
    const active = all.filter(a => a.status === 'active')
    if (active.length === 0) return null

    const robin = await this.store.getAndIncrementRobin()
    const summary = active[robin % active.length]
    return this.store.getAccount(summary.id)
  }

  async recordUsage(accountId: string): Promise<void> {
    await this.store.recordUsage(accountId)
  }

  async markFailed(accountId: string, reason: string): Promise<void> {
    await this.store.markFailed(accountId, reason)
    await this.log({
      timestamp: new Date().toISOString(),
      event: 'error',
      message: `Account ${accountId} marked failed: ${reason}`,
      accountId,
      error: reason,
    })
  }

  async disable(accountId: string, reason: string): Promise<void> {
    await this.store.disable(accountId, reason)
    await this.log({
      timestamp: new Date().toISOString(),
      event: 'disabled',
      message: `Account ${accountId} disabled: ${reason}`,
      accountId,
    })
  }

  async restore(accountId: string, session: SessionState, proQueries: number = DEFAULT_PRO_QUERIES): Promise<void> {
    await this.store.restore(accountId, session, proQueries)
    const account = await this.store.getAccount(accountId)
    await this.log({
      timestamp: new Date().toISOString(),
      event: 'relogin',
      message: `Account ${accountId} restored via re-login (${proQueries} pro queries)`,
      accountId,
      email: account?.email,
    })
  }

  async decryptPassword(accountId: string): Promise<string | null> {
    const account = await this.store.getAccount(accountId)
    if (!account?.emailPasswordEnc) return null
    try {
      return await decrypt(account.emailPasswordEnc, this.secret)
    } catch {
      return null
    }
  }

  async getFailedAccounts(): Promise<Account[]> {
    return this.store.getFailedAccounts()
  }

  async needsRegistration(): Promise<boolean> {
    return (await this.store.countActive()) < MIN_ACTIVE_ACCOUNTS
  }

  async listAccounts(): Promise<{ accounts: AccountSummary[]; active: number; total: number }> {
    const accounts = await this.store.listAccounts()
    const active = accounts.filter(a => a.status === 'active').length
    return { accounts, active, total: accounts.length }
  }

  async getAccount(accountId: string): Promise<Account | null> {
    return this.store.getAccount(accountId)
  }

  async deleteAccount(accountId: string): Promise<boolean> {
    return this.store.deleteAccount(accountId)
  }

  async deleteAll(): Promise<number> {
    return this.store.deleteAll()
  }
}
