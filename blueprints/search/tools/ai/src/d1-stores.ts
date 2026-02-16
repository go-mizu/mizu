/**
 * D1-optimized store implementations.
 *
 * Each store uses direct SQL against specialized tables (migration 0002).
 * Key optimizations vs the generic kv-blob approach:
 * - recordUsage: 1 UPDATE vs 4 get/set ops
 * - createThread: 1 batch INSERT vs set blob + read-modify-write index
 * - addFollowUp: 1 batch vs full blob rewrite + index update
 * - deleteThread: DELETE CASCADE vs get + delete + read-modify-write index
 * - listThreads: SELECT ORDER BY vs read entire index blob
 */

import type { AccountStore, ThreadStore, SessionStore, OGStore } from './storage'
import type {
  Account, AccountSummary, SessionState, RegistrationLog, OGData,
  Thread, ThreadMessage, ThreadSummary,
} from './types'

// ============================================================
// D1AccountStore
// ============================================================

export class D1AccountStore implements AccountStore {
  readonly name = 'd1'
  constructor(private db: D1Database) {}

  async addAccount(account: Account): Promise<void> {
    await this.db.prepare(`
      INSERT INTO accounts (id, email, email_provider, email_password_enc, session_csrf, session_cookies, session_created_at, pro_queries, status, created_at, last_used_at, total_queries_used)
      VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
    `).bind(
      account.id, account.email, account.emailProvider, account.emailPasswordEnc,
      account.session.csrfToken, account.session.cookies, account.session.createdAt,
      account.proQueries, account.status, account.createdAt, account.lastUsedAt,
      account.totalQueriesUsed,
    ).run()
  }

  async getAccount(id: string): Promise<Account | null> {
    const row = await this.db.prepare('SELECT * FROM accounts WHERE id = ?').bind(id).first<Record<string, unknown>>()
    return row ? toAccount(row) : null
  }

  async deleteAccount(id: string): Promise<boolean> {
    const result = await this.db.prepare('DELETE FROM accounts WHERE id = ?').bind(id).run()
    return (result.meta?.changes ?? 0) > 0
  }

  async deleteAll(): Promise<number> {
    const count = await this.db.prepare('SELECT COUNT(*) as cnt FROM accounts').first<{ cnt: number }>()
    await this.db.batch([
      this.db.prepare('DELETE FROM accounts'),
      this.db.prepare('UPDATE account_robin SET value = 0 WHERE id = 1'),
    ])
    return count?.cnt ?? 0
  }

  async listAccounts(): Promise<AccountSummary[]> {
    const rows = await this.db.prepare(
      'SELECT id, email, pro_queries, status, created_at FROM accounts ORDER BY created_at DESC',
    ).all<Record<string, unknown>>()
    return rows.results.map(r => ({
      id: r.id as string,
      email: r.email as string,
      proQueries: r.pro_queries as number,
      status: r.status as string,
      createdAt: r.created_at as string,
    }))
  }

  async getFailedAccounts(): Promise<Account[]> {
    const rows = await this.db.prepare("SELECT * FROM accounts WHERE status = 'failed'").all<Record<string, unknown>>()
    return rows.results.map(toAccount)
  }

  async countActive(): Promise<number> {
    const row = await this.db.prepare("SELECT COUNT(*) as cnt FROM accounts WHERE status = 'active'").first<{ cnt: number }>()
    return row?.cnt ?? 0
  }

  /** Single UPDATE — replaces get+modify+set+updateIndex (4 ops → 1). */
  async recordUsage(id: string): Promise<void> {
    const now = new Date().toISOString()
    await this.db.prepare(`
      UPDATE accounts SET
        pro_queries = MAX(0, pro_queries - 1),
        last_used_at = ?,
        total_queries_used = total_queries_used + 1,
        status = CASE WHEN pro_queries <= 1 THEN 'exhausted' ELSE status END,
        disabled_at = CASE WHEN pro_queries <= 1 THEN ? ELSE disabled_at END,
        disable_reason = CASE WHEN pro_queries <= 1 THEN 'Pro queries exhausted' ELSE disable_reason END
      WHERE id = ?
    `).bind(now, now, id).run()
  }

  async markFailed(id: string, _reason: string): Promise<void> {
    await this.db.prepare("UPDATE accounts SET status = 'failed' WHERE id = ?").bind(id).run()
  }

  async disable(id: string, reason: string): Promise<void> {
    const now = new Date().toISOString()
    await this.db.prepare(
      "UPDATE accounts SET status = 'disabled', disabled_at = ?, disable_reason = ? WHERE id = ?",
    ).bind(now, reason, id).run()
  }

  async restore(id: string, session: SessionState, proQueries: number): Promise<void> {
    const now = new Date().toISOString()
    await this.db.prepare(`
      UPDATE accounts SET
        session_csrf = ?, session_cookies = ?, session_created_at = ?,
        pro_queries = ?, status = 'active', last_used_at = ?,
        disabled_at = NULL, disable_reason = NULL
      WHERE id = ?
    `).bind(session.csrfToken, session.cookies, session.createdAt, proQueries, now, id).run()
  }

  async getAndIncrementRobin(): Promise<number> {
    const row = await this.db.prepare('SELECT value FROM account_robin WHERE id = 1').first<{ value: number }>()
    const current = row?.value ?? 0
    await this.db.prepare('UPDATE account_robin SET value = ? WHERE id = 1').bind(current + 1).run()
    return current
  }

  async resetRobin(): Promise<void> {
    await this.db.prepare('UPDATE account_robin SET value = 0 WHERE id = 1').run()
  }

  /** Single INSERT — replaces read-modify-write of log array (2 ops → 1). */
  async appendLog(entry: RegistrationLog): Promise<void> {
    await this.db.prepare(`
      INSERT INTO account_logs (timestamp, event, message, provider, email, account_id, duration_ms, error)
      VALUES (?, ?, ?, ?, ?, ?, ?, ?)
    `).bind(
      entry.timestamp, entry.event, entry.message,
      entry.provider ?? null, entry.email ?? null, entry.accountId ?? null,
      entry.durationMs ?? null, entry.error ?? null,
    ).run()
  }

  async getLogs(limit: number = 100): Promise<RegistrationLog[]> {
    const rows = await this.db.prepare(
      'SELECT * FROM account_logs ORDER BY id DESC LIMIT ?',
    ).bind(limit).all<Record<string, unknown>>()
    return rows.results.reverse().map(r => ({
      timestamp: r.timestamp as string,
      event: r.event as RegistrationLog['event'],
      message: r.message as string,
      provider: (r.provider as string) || undefined,
      email: (r.email as string) || undefined,
      accountId: (r.account_id as string) || undefined,
      durationMs: (r.duration_ms as number) || undefined,
      error: (r.error as string) || undefined,
    }))
  }

  /** Atomic UPDATE WHERE — replaces get+check+set (2 ops → 1). */
  async tryLock(ttlSeconds: number): Promise<boolean> {
    const now = Date.now()
    const expiresAt = now + ttlSeconds * 1000
    const result = await this.db.prepare(
      'UPDATE account_lock SET locked_at = ?, expires_at = ? WHERE id = 1 AND (expires_at IS NULL OR expires_at < ?)',
    ).bind(new Date().toISOString(), expiresAt, now).run()
    return (result.meta?.changes ?? 0) > 0
  }

  async unlock(): Promise<void> {
    await this.db.prepare('UPDATE account_lock SET locked_at = NULL, expires_at = NULL WHERE id = 1').run()
  }
}

// ============================================================
// D1ThreadStore
// ============================================================

export class D1ThreadStore implements ThreadStore {
  readonly name = 'd1'
  constructor(private db: D1Database) {}

  /** Batch INSERT thread + messages — replaces set blob + read-modify-write index (3 ops → 1 batch). */
  async createThread(thread: Thread): Promise<void> {
    const stmts: D1PreparedStatement[] = [
      this.db.prepare(`
        INSERT INTO threads (id, title, mode, model, message_count, created_at, updated_at)
        VALUES (?, ?, ?, ?, ?, ?, ?)
      `).bind(thread.id, thread.title, thread.mode, thread.model, thread.messages.length, thread.createdAt, thread.updatedAt),
    ]
    for (let i = 0; i < thread.messages.length; i++) {
      stmts.push(messageInsert(this.db, thread.id, i, thread.messages[i]))
    }
    await this.db.batch(stmts)
  }

  async getThread(id: string): Promise<Thread | null> {
    const [threadResult, msgsResult] = await this.db.batch([
      this.db.prepare('SELECT * FROM threads WHERE id = ?').bind(id),
      this.db.prepare('SELECT * FROM thread_messages WHERE thread_id = ? ORDER BY seq').bind(id),
    ])

    const row = threadResult.results[0] as Record<string, unknown> | undefined
    if (!row) return null

    const messages: ThreadMessage[] = (msgsResult.results as Record<string, unknown>[]).map(toMessage)

    return {
      id: row.id as string,
      title: row.title as string,
      mode: row.mode as string,
      model: row.model as string,
      messages,
      createdAt: row.created_at as string,
      updatedAt: row.updated_at as string,
    }
  }

  /** Batch INSERT 2 messages + UPDATE count — replaces full blob rewrite + index update (4 ops → 1 batch). */
  async addFollowUp(threadId: string, userMsg: ThreadMessage, assistantMsg: ThreadMessage): Promise<Thread | null> {
    const row = await this.db.prepare('SELECT message_count FROM threads WHERE id = ?').bind(threadId).first<{ message_count: number }>()
    if (!row) return null

    const seq = row.message_count
    await this.db.batch([
      messageInsert(this.db, threadId, seq, userMsg),
      messageInsert(this.db, threadId, seq + 1, assistantMsg),
      this.db.prepare('UPDATE threads SET message_count = message_count + 2, updated_at = ? WHERE id = ?').bind(userMsg.createdAt, threadId),
    ])

    return this.getThread(threadId)
  }

  /** DELETE CASCADE — replaces get + delete blob + read-modify-write index (4 ops → 1). */
  async deleteThread(id: string): Promise<boolean> {
    // Delete messages first (in case CASCADE isn't enabled), then thread
    await this.db.batch([
      this.db.prepare('DELETE FROM thread_messages WHERE thread_id = ?').bind(id),
      this.db.prepare('DELETE FROM threads WHERE id = ?').bind(id),
    ])
    return true
  }

  /** SELECT ORDER BY — replaces reading entire index blob. */
  async listThreads(limit: number = 100): Promise<ThreadSummary[]> {
    const rows = await this.db.prepare(
      'SELECT id, title, mode, model, message_count, created_at, updated_at FROM threads ORDER BY updated_at DESC LIMIT ?',
    ).bind(limit).all<Record<string, unknown>>()
    return rows.results.map(r => ({
      id: r.id as string,
      title: r.title as string,
      mode: r.mode as string,
      model: r.model as string,
      messageCount: r.message_count as number,
      createdAt: r.created_at as string,
      updatedAt: r.updated_at as string,
    }))
  }
}

// ============================================================
// D1SessionStore
// ============================================================

export class D1SessionStore implements SessionStore {
  readonly name = 'd1'
  constructor(private db: D1Database) {}

  async getPool(): Promise<SessionState[]> {
    const rows = await this.db.prepare(
      'SELECT csrf_token, cookies, created_at FROM sessions WHERE is_legacy = 0 ORDER BY id DESC LIMIT 5',
    ).all<Record<string, unknown>>()
    return rows.results.map(toSession)
  }

  async addToPool(session: SessionState): Promise<void> {
    await this.db.prepare(
      'INSERT INTO sessions (csrf_token, cookies, created_at, is_legacy) VALUES (?, ?, ?, 0)',
    ).bind(session.csrfToken, session.cookies, session.createdAt).run()
  }

  async savePool(sessions: SessionState[]): Promise<void> {
    const stmts: D1PreparedStatement[] = [
      this.db.prepare('DELETE FROM sessions WHERE is_legacy = 0'),
    ]
    for (const s of sessions) {
      stmts.push(this.db.prepare(
        'INSERT INTO sessions (csrf_token, cookies, created_at, is_legacy) VALUES (?, ?, ?, 0)',
      ).bind(s.csrfToken, s.cookies, s.createdAt))
    }
    await this.db.batch(stmts)
  }

  async getLegacy(): Promise<SessionState | null> {
    const row = await this.db.prepare(
      'SELECT csrf_token, cookies, created_at FROM sessions WHERE is_legacy = 1 ORDER BY id DESC LIMIT 1',
    ).first<Record<string, unknown>>()
    return row ? toSession(row) : null
  }

  async saveLegacy(session: SessionState): Promise<void> {
    await this.db.batch([
      this.db.prepare('DELETE FROM sessions WHERE is_legacy = 1'),
      this.db.prepare(
        'INSERT INTO sessions (csrf_token, cookies, created_at, is_legacy) VALUES (?, ?, ?, 1)',
      ).bind(session.csrfToken, session.cookies, session.createdAt),
    ])
  }
}

// ============================================================
// D1OGStore
// ============================================================

export class D1OGStore implements OGStore {
  readonly name = 'd1'
  constructor(private db: D1Database) {}

  async get(url: string): Promise<OGData | null> {
    const row = await this.db.prepare(
      'SELECT title, description, image, site_name FROM og_cache WHERE url = ? AND (expires_at IS NULL OR expires_at > ?)',
    ).bind(url, Date.now()).first<Record<string, unknown>>()
    if (!row) return null
    return {
      title: row.title as string,
      description: row.description as string,
      image: row.image as string,
      siteName: row.site_name as string,
    }
  }

  async set(url: string, data: OGData, ttlSeconds?: number): Promise<void> {
    const expiresAt = ttlSeconds ? Date.now() + ttlSeconds * 1000 : null
    await this.db.prepare(
      'INSERT OR REPLACE INTO og_cache (url, title, description, image, site_name, expires_at) VALUES (?, ?, ?, ?, ?, ?)',
    ).bind(url, data.title, data.description, data.image, data.siteName, expiresAt).run()
  }
}

// ============================================================
// Helpers
// ============================================================

function toAccount(row: Record<string, unknown>): Account {
  return {
    id: row.id as string,
    email: row.email as string,
    emailProvider: row.email_provider as string,
    emailPasswordEnc: row.email_password_enc as string,
    session: {
      csrfToken: row.session_csrf as string,
      cookies: row.session_cookies as string,
      createdAt: row.session_created_at as string,
    },
    proQueries: row.pro_queries as number,
    status: row.status as Account['status'],
    createdAt: row.created_at as string,
    lastUsedAt: row.last_used_at as string,
    disabledAt: (row.disabled_at as string) || undefined,
    disableReason: (row.disable_reason as string) || undefined,
    totalQueriesUsed: row.total_queries_used as number,
  }
}

function toSession(row: Record<string, unknown>): SessionState {
  return {
    csrfToken: row.csrf_token as string,
    cookies: row.cookies as string,
    createdAt: row.created_at as string,
  }
}

function toMessage(m: Record<string, unknown>): ThreadMessage {
  return {
    role: m.role as 'user' | 'assistant',
    content: m.content as string,
    citations: jsonOrUndefined(m.citations),
    webResults: jsonOrUndefined(m.web_results),
    relatedQueries: jsonOrUndefined(m.related_queries),
    images: jsonOrUndefined(m.images),
    videos: jsonOrUndefined(m.videos),
    thinkingSteps: jsonOrUndefined(m.thinking_steps),
    backendUUID: (m.backend_uuid as string) || undefined,
    model: (m.model as string) || undefined,
    durationMs: (m.duration_ms as number) || undefined,
    createdAt: m.created_at as string,
  }
}

function jsonOrUndefined(val: unknown): any {
  if (val === null || val === undefined) return undefined
  if (typeof val === 'string') {
    try { return JSON.parse(val) } catch { return undefined }
  }
  return val
}

function messageInsert(db: D1Database, threadId: string, seq: number, msg: ThreadMessage): D1PreparedStatement {
  return db.prepare(`
    INSERT INTO thread_messages (thread_id, seq, role, content, citations, web_results, related_queries, images, videos, thinking_steps, backend_uuid, model, duration_ms, created_at)
    VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
  `).bind(
    threadId, seq, msg.role, msg.content,
    msg.citations ? JSON.stringify(msg.citations) : null,
    msg.webResults ? JSON.stringify(msg.webResults) : null,
    msg.relatedQueries ? JSON.stringify(msg.relatedQueries) : null,
    msg.images ? JSON.stringify(msg.images) : null,
    msg.videos ? JSON.stringify(msg.videos) : null,
    msg.thinkingSteps ? JSON.stringify(msg.thinkingSteps) : null,
    msg.backendUUID ?? null,
    msg.model ?? null,
    msg.durationMs ?? null,
    msg.createdAt,
  )
}
