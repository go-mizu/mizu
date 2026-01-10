/**
 * SQLite users store
 */

import type { SqliteExecutor } from './executor.js';
import type { User, CreateUserInput } from '../../types.js';

export class SqliteUsersStore {
  constructor(private executor: SqliteExecutor) {}

  async createUser(input: CreateUserInput & { id: string; password_hash: string }): Promise<User> {
    const now = new Date().toISOString();
    await this.executor.run(
      `INSERT INTO users (id, email, name, password_hash, created_at, updated_at)
       VALUES (?, ?, ?, ?, ?, ?)`,
      [input.id, input.email, input.name, input.password_hash, now, now]
    );
    const user = await this.getUserById(input.id);
    if (!user) throw new Error('Failed to create user');
    return user;
  }

  async getUserById(id: string): Promise<User | null> {
    return this.executor.get<User>(
      `SELECT * FROM users WHERE id = ?`,
      [id]
    );
  }

  async getUserByEmail(email: string): Promise<User | null> {
    return this.executor.get<User>(
      `SELECT * FROM users WHERE email = ?`,
      [email]
    );
  }
}
