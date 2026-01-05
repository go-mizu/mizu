import { describe, it, expect, beforeEach } from 'vitest';
import { createTestApp, getSessionCookie, withSession, generateEmail, registerUser } from './setup';
import type { TestApp } from './setup';

describe('Authentication E2E Tests', () => {
  let app: TestApp;

  beforeEach(() => {
    app = createTestApp();
  });

  describe('POST /api/v1/auth/register', () => {
    it('should register a new user successfully', async () => {
      const email = generateEmail();
      const res = await app.request('/api/v1/auth/register', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          email,
          name: 'Test User',
          password: 'password123',
        }),
      });

      expect(res.status).toBe(201);
      const data = await res.json() as { user: { id: string; email: string; name: string } };
      expect(data.user).toBeDefined();
      expect(data.user.email).toBe(email);
      expect(data.user.name).toBe('Test User');
      expect((data.user as any).passwordHash).toBeUndefined(); // Should not expose password hash
    });

    it('should set session cookie on registration', async () => {
      const res = await app.request('/api/v1/auth/register', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          email: generateEmail(),
          name: 'Test User',
          password: 'password123',
        }),
      });

      expect(res.status).toBe(201);
      const sessionId = getSessionCookie(res);
      expect(sessionId).toBeTruthy();
    });

    it('should reject registration with existing email', async () => {
      const email = generateEmail();

      // Register first user
      await app.request('/api/v1/auth/register', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          email,
          name: 'User 1',
          password: 'password123',
        }),
      });

      // Try to register with same email
      const res = await app.request('/api/v1/auth/register', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          email,
          name: 'User 2',
          password: 'password456',
        }),
      });

      expect(res.status).toBe(409);
    });

    it('should reject registration with invalid email format', async () => {
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

    it('should reject registration with short password', async () => {
      const res = await app.request('/api/v1/auth/register', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          email: generateEmail(),
          name: 'Test User',
          password: '123',
        }),
      });

      expect(res.status).toBe(400);
    });

    it('should reject registration with empty name', async () => {
      const res = await app.request('/api/v1/auth/register', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          email: generateEmail(),
          name: '',
          password: 'password123',
        }),
      });

      expect(res.status).toBe(400);
    });
  });

  describe('POST /api/v1/auth/login', () => {
    it('should login successfully with valid credentials', async () => {
      const email = generateEmail();
      const password = 'password123';

      // Register first
      await app.request('/api/v1/auth/register', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email, name: 'Test User', password }),
      });

      // Login
      const res = await app.request('/api/v1/auth/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email, password }),
      });

      expect(res.status).toBe(200);
      const data = await res.json() as { user: { email: string } };
      expect(data.user.email).toBe(email);
    });

    it('should set session cookie on login', async () => {
      const email = generateEmail();
      const password = 'password123';

      await app.request('/api/v1/auth/register', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email, name: 'Test User', password }),
      });

      const res = await app.request('/api/v1/auth/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email, password }),
      });

      expect(res.status).toBe(200);
      const sessionId = getSessionCookie(res);
      expect(sessionId).toBeTruthy();
    });

    it('should reject login with wrong password', async () => {
      const email = generateEmail();

      await app.request('/api/v1/auth/register', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email, name: 'Test User', password: 'password123' }),
      });

      const res = await app.request('/api/v1/auth/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email, password: 'wrongpassword' }),
      });

      // API returns 400 for invalid credentials (error message contains "Invalid")
      expect(res.status).toBe(400);
    });

    it('should reject login with non-existent email', async () => {
      const res = await app.request('/api/v1/auth/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          email: 'nonexistent@example.com',
          password: 'password123',
        }),
      });

      // API returns 400 for invalid credentials (error message contains "Invalid")
      expect(res.status).toBe(400);
    });
  });

  describe('GET /api/v1/auth/me', () => {
    it('should return current user with valid session', async () => {
      const { user, sessionId } = await registerUser(app);

      const res = await app.request('/api/v1/auth/me', {
        headers: withSession({}, sessionId),
      });

      expect(res.status).toBe(200);
      const data = await res.json() as { user: { id: string; email: string } };
      expect(data.user.id).toBe(user.id);
      expect(data.user.email).toBe(user.email);
    });

    it('should reject request without session', async () => {
      const res = await app.request('/api/v1/auth/me');

      expect(res.status).toBe(401);
    });

    it('should reject request with invalid session', async () => {
      const res = await app.request('/api/v1/auth/me', {
        headers: withSession({}, 'invalid-session-id'),
      });

      expect(res.status).toBe(401);
    });
  });

  describe('POST /api/v1/auth/logout', () => {
    it('should logout successfully', async () => {
      const { sessionId } = await registerUser(app);

      const res = await app.request('/api/v1/auth/logout', {
        method: 'POST',
        headers: withSession({}, sessionId),
      });

      expect(res.status).toBe(200);
      const data = await res.json() as { success: boolean };
      expect(data.success).toBe(true);
    });

    it('should clear session cookie on logout', async () => {
      const { sessionId } = await registerUser(app);

      const res = await app.request('/api/v1/auth/logout', {
        method: 'POST',
        headers: withSession({}, sessionId),
      });

      expect(res.status).toBe(200);
      const setCookie = res.headers.get('set-cookie');
      expect(setCookie).toContain('workspace_session=');
    });

    it('should invalidate session after logout', async () => {
      const { sessionId } = await registerUser(app);

      // Logout
      await app.request('/api/v1/auth/logout', {
        method: 'POST',
        headers: withSession({}, sessionId),
      });

      // Try to access with old session
      const res = await app.request('/api/v1/auth/me', {
        headers: withSession({}, sessionId),
      });

      expect(res.status).toBe(401);
    });
  });
});
