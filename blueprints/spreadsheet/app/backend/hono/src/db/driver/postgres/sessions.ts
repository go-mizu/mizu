/**
 * PostgreSQL sessions store
 */

import type postgres from 'postgres';
import type { Session, CreateSessionInput } from '../../types.js';

export class PostgresSessionsStore {
  constructor(private sql: postgres.Sql) {}

  async createSession(input: CreateSessionInput): Promise<Session> {
    const [session] = await this.sql<[Session]>`
      INSERT INTO sessions (id, user_id, token, expires_at, created_at)
      VALUES (${input.id}, ${input.user_id}, ${input.token}, ${input.expires_at}, NOW())
      RETURNING *
    `;
    return session;
  }

  async getSessionByToken(token: string): Promise<Session | null> {
    const [session] = await this.sql<[Session | undefined]>`
      SELECT * FROM sessions WHERE token = ${token} AND expires_at > NOW()
    `;
    return session ?? null;
  }

  async deleteSession(token: string): Promise<void> {
    await this.sql`DELETE FROM sessions WHERE token = ${token}`;
  }

  async deleteExpiredSessions(): Promise<void> {
    await this.sql`DELETE FROM sessions WHERE expires_at <= NOW()`;
  }
}
