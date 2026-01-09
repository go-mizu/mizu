import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import type BetterSqlite3 from 'better-sqlite3';
import type { Database } from '../src/db/types.js';
import { createTestDb, createTestApp, registerAndLogin } from './setup.js';

describe('Workbooks API', () => {
  let db: Database;
  let rawDb: BetterSqlite3.Database;
  let app: ReturnType<typeof createTestApp>;
  let token: string;

  beforeEach(async () => {
    const result = createTestDb();
    db = result.db;
    rawDb = result.rawDb;
    app = createTestApp(db);

    const auth = await registerAndLogin(app);
    token = auth.token;
  });

  afterEach(() => {
    rawDb.close();
  });

  describe('POST /api/v1/workbooks', () => {
    it('should create a workbook with default sheet', async () => {
      const res = await app.request('/api/v1/workbooks', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`,
        },
        body: JSON.stringify({ name: 'My Workbook' }),
      });

      expect(res.status).toBe(201);
      const data = await res.json() as { workbook: { id: string; name: string } };
      expect(data.workbook.name).toBe('My Workbook');

      // Verify default sheet was created
      const sheetsRes = await app.request(`/api/v1/workbooks/${data.workbook.id}/sheets`, {
        headers: { 'Authorization': `Bearer ${token}` },
      });

      const sheetsData = await sheetsRes.json() as { sheets: Array<{ name: string }> };
      expect(sheetsData.sheets).toHaveLength(1);
      expect(sheetsData.sheets[0].name).toBe('Sheet1');
    });

    it('should require authentication', async () => {
      const res = await app.request('/api/v1/workbooks', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name: 'My Workbook' }),
      });

      expect(res.status).toBe(401);
    });
  });

  describe('GET /api/v1/workbooks', () => {
    it('should list user workbooks', async () => {
      // Create workbooks
      await app.request('/api/v1/workbooks', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`,
        },
        body: JSON.stringify({ name: 'Workbook 1' }),
      });

      await app.request('/api/v1/workbooks', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`,
        },
        body: JSON.stringify({ name: 'Workbook 2' }),
      });

      const res = await app.request('/api/v1/workbooks', {
        headers: { 'Authorization': `Bearer ${token}` },
      });

      expect(res.status).toBe(200);
      const data = await res.json() as { workbooks: Array<{ name: string }> };
      expect(data.workbooks).toHaveLength(2);
    });

    it('should not show other users workbooks', async () => {
      // Create workbook as first user
      await app.request('/api/v1/workbooks', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`,
        },
        body: JSON.stringify({ name: 'User 1 Workbook' }),
      });

      // Register second user
      const auth2 = await registerAndLogin(app, 'user2@example.com');

      // List workbooks as second user
      const res = await app.request('/api/v1/workbooks', {
        headers: { 'Authorization': `Bearer ${auth2.token}` },
      });

      const data = await res.json() as { workbooks: Array<{ name: string }> };
      expect(data.workbooks).toHaveLength(0);
    });
  });

  describe('GET /api/v1/workbooks/:id', () => {
    it('should get workbook by id', async () => {
      const createRes = await app.request('/api/v1/workbooks', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`,
        },
        body: JSON.stringify({ name: 'My Workbook' }),
      });

      const { workbook } = await createRes.json() as { workbook: { id: string } };

      const res = await app.request(`/api/v1/workbooks/${workbook.id}`, {
        headers: { 'Authorization': `Bearer ${token}` },
      });

      expect(res.status).toBe(200);
      const data = await res.json() as { workbook: { name: string } };
      expect(data.workbook.name).toBe('My Workbook');
    });

    it('should return 404 for unknown workbook', async () => {
      const res = await app.request('/api/v1/workbooks/unknown-id', {
        headers: { 'Authorization': `Bearer ${token}` },
      });

      expect(res.status).toBe(404);
    });

    it('should return 403 for other users workbook', async () => {
      // Create workbook as first user
      const createRes = await app.request('/api/v1/workbooks', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`,
        },
        body: JSON.stringify({ name: 'User 1 Workbook' }),
      });

      const { workbook } = await createRes.json() as { workbook: { id: string } };

      // Try to access as second user
      const auth2 = await registerAndLogin(app, 'user2@example.com');
      const res = await app.request(`/api/v1/workbooks/${workbook.id}`, {
        headers: { 'Authorization': `Bearer ${auth2.token}` },
      });

      expect(res.status).toBe(403);
    });
  });

  describe('PATCH /api/v1/workbooks/:id', () => {
    it('should update workbook name', async () => {
      const createRes = await app.request('/api/v1/workbooks', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`,
        },
        body: JSON.stringify({ name: 'Original Name' }),
      });

      const { workbook } = await createRes.json() as { workbook: { id: string } };

      const res = await app.request(`/api/v1/workbooks/${workbook.id}`, {
        method: 'PATCH',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`,
        },
        body: JSON.stringify({ name: 'Updated Name' }),
      });

      expect(res.status).toBe(200);
      const data = await res.json() as { workbook: { name: string } };
      expect(data.workbook.name).toBe('Updated Name');
    });
  });

  describe('DELETE /api/v1/workbooks/:id', () => {
    it('should delete workbook', async () => {
      const createRes = await app.request('/api/v1/workbooks', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`,
        },
        body: JSON.stringify({ name: 'To Delete' }),
      });

      const { workbook } = await createRes.json() as { workbook: { id: string } };

      const deleteRes = await app.request(`/api/v1/workbooks/${workbook.id}`, {
        method: 'DELETE',
        headers: { 'Authorization': `Bearer ${token}` },
      });

      expect(deleteRes.status).toBe(200);

      // Verify it's deleted
      const getRes = await app.request(`/api/v1/workbooks/${workbook.id}`, {
        headers: { 'Authorization': `Bearer ${token}` },
      });

      expect(getRes.status).toBe(404);
    });
  });
});
