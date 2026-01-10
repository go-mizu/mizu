/**
 * SQLite sessions store
 */

import type { SqliteExecutor } from './executor.js';
import type { Session, CreateSessionInput } from '../../types.js';

export class SqliteSessionsStore {
  constructor(private executor: SqliteExecutor) {}

  async createSession(input: CreateSessionInput): Promise<Session> {
    const now = new Date().toISOString();
    await this.executor.run(
      `INSERT INTO sessions (id, user_id, token, expires_at, created_at)
       VALUES (?, ?, ?, ?, ?)`,
      [input.id, input.user_id, input.token, input.expires_at, now]
    );
    const session = await this.executor.get<Session>(
      `SELECT * FROM sessions WHERE id = ?`,
      [input.id]
    );
    if (!session) throw new Error('Failed to create session');
    return session;
  }

  async getSessionByToken(token: string): Promise<Session | null> {
    return this.executor.get<Session>(
      `SELECT * FROM sessions WHERE token = ? AND expires_at > datetime('now')`,
      [token]
    );
  }

  async deleteSession(token: string): Promise<void> {
    await this.executor.run(`DELETE FROM sessions WHERE token = ?`, [token]);
  }

  async deleteExpiredSessions(): Promise<void> {
    await this.executor.run(`DELETE FROM sessions WHERE expires_at <= datetime('now')`);
  }
}
