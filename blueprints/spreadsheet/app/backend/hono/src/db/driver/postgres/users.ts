/**
 * PostgreSQL users store
 */

import type postgres from 'postgres';
import type { User, CreateUserInput } from '../../types.js';

export class PostgresUsersStore {
  constructor(private sql: postgres.Sql) {}

  async createUser(input: CreateUserInput & { id: string; password_hash: string }): Promise<User> {
    const [user] = await this.sql<[User]>`
      INSERT INTO users (id, email, name, password_hash, created_at, updated_at)
      VALUES (${input.id}, ${input.email}, ${input.name}, ${input.password_hash}, NOW(), NOW())
      RETURNING *
    `;
    return user;
  }

  async getUserById(id: string): Promise<User | null> {
    const [user] = await this.sql<[User | undefined]>`
      SELECT * FROM users WHERE id = ${id}
    `;
    return user ?? null;
  }

  async getUserByEmail(email: string): Promise<User | null> {
    const [user] = await this.sql<[User | undefined]>`
      SELECT * FROM users WHERE email = ${email}
    `;
    return user ?? null;
  }
}
