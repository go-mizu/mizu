import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import type BetterSqlite3 from 'better-sqlite3';
import type { Database } from '../src/db/types.js';
import { createTestDb, createTestApp } from './setup.js';

describe('Auth API', () => {
  let db: Database;
  let rawDb: BetterSqlite3.Database;
  let app: ReturnType<typeof createTestApp>;

  beforeEach(() => {
    const result = createTestDb();
    db = result.db;
    rawDb = result.rawDb;
    app = createTestApp(db);
  });

  afterEach(() => {
    rawDb.close();
  });

  describe('POST /api/v1/auth/register', () => {
    it('should register a new user', async () => {
      const res = await app.request('/api/v1/auth/register', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          email: 'test@example.com',
          name: 'Test User',
          password: 'password123',
        }),
      });

      expect(res.status).toBe(201);
      const data = await res.json();
      expect(data).toHaveProperty('token');
      expect(data).toHaveProperty('user');
      expect(data.user.email).toBe('test@example.com');
      expect(data.user.name).toBe('Test User');
      expect(data.user).not.toHaveProperty('password_hash');
    });

    it('should reject duplicate email', async () => {
      // Register first user
      await app.request('/api/v1/auth/register', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          email: 'test@example.com',
          name: 'Test User',
          password: 'password123',
        }),
      });

      // Try to register same email
      const res = await app.request('/api/v1/auth/register', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          email: 'test@example.com',
          name: 'Another User',
          password: 'password456',
        }),
      });

      expect(res.status).toBe(409);
    });

    it('should validate email format', async () => {
      const res = await app.request('/api/v1/auth/register', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          email: 'invalid-email',
          name: 'Test User',
          password: 'password123',
        }),
      });

      expect(res.status).toBe(400);
    });

    it('should validate password length', async () => {
      const res = await app.request('/api/v1/auth/register', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          email: 'test@example.com',
          name: 'Test User',
          password: '123',
        }),
      });

      expect(res.status).toBe(400);
    });
  });

  describe('POST /api/v1/auth/login', () => {
    beforeEach(async () => {
      // Register a user
      await app.request('/api/v1/auth/register', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          email: 'test@example.com',
          name: 'Test User',
          password: 'password123',
        }),
      });
    });

    it('should login with valid credentials', async () => {
      const res = await app.request('/api/v1/auth/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          email: 'test@example.com',
          password: 'password123',
        }),
      });

      expect(res.status).toBe(200);
      const data = await res.json();
      expect(data).toHaveProperty('token');
      expect(data).toHaveProperty('user');
    });

    it('should reject invalid password', async () => {
      const res = await app.request('/api/v1/auth/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          email: 'test@example.com',
          password: 'wrongpassword',
        }),
      });

      expect(res.status).toBe(401);
    });

    it('should reject unknown email', async () => {
      const res = await app.request('/api/v1/auth/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          email: 'unknown@example.com',
          password: 'password123',
        }),
      });

      expect(res.status).toBe(401);
    });
  });

  describe('GET /api/v1/auth/me', () => {
    it('should return current user with valid token', async () => {
      // Register and get token
      const registerRes = await app.request('/api/v1/auth/register', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          email: 'test@example.com',
          name: 'Test User',
          password: 'password123',
        }),
      });

      const { token } = await registerRes.json() as { token: string };

      // Get current user
      const res = await app.request('/api/v1/auth/me', {
        headers: { 'Authorization': `Bearer ${token}` },
      });

      expect(res.status).toBe(200);
      const data = await res.json();
      expect(data.user.email).toBe('test@example.com');
    });

    it('should reject without token', async () => {
      const res = await app.request('/api/v1/auth/me');
      expect(res.status).toBe(401);
    });

    it('should reject with invalid token', async () => {
      const res = await app.request('/api/v1/auth/me', {
        headers: { 'Authorization': 'Bearer invalid-token' },
      });
      expect(res.status).toBe(401);
    });
  });

  describe('POST /api/v1/auth/logout', () => {
    it('should logout user', async () => {
      // Register and get token
      const registerRes = await app.request('/api/v1/auth/register', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          email: 'test@example.com',
          name: 'Test User',
          password: 'password123',
        }),
      });

      const { token } = await registerRes.json() as { token: string };

      // Logout
      const logoutRes = await app.request('/api/v1/auth/logout', {
        method: 'POST',
        headers: { 'Authorization': `Bearer ${token}` },
      });

      expect(logoutRes.status).toBe(200);

      // Verify token no longer works
      const meRes = await app.request('/api/v1/auth/me', {
        headers: { 'Authorization': `Bearer ${token}` },
      });

      expect(meRes.status).toBe(401);
    });
  });
});
