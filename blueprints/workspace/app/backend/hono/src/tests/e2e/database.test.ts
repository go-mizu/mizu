import { describe, it, expect, beforeEach } from 'vitest';
import { createTestApp, withSession, setupTestContext, createDatabase } from './setup';
import type { TestApp } from './setup';

describe('Database E2E Tests', () => {
  let app: TestApp;

  beforeEach(() => {
    app = createTestApp();
  });

  describe('POST /api/v1/databases', () => {
    it('should create a database', async () => {
      const { sessionId, workspace } = await setupTestContext(app);

      const res = await app.request('/api/v1/databases', {
        method: 'POST',
        headers: withSession({ 'Content-Type': 'application/json' }, sessionId),
        body: JSON.stringify({
          workspaceId: workspace.id,
          title: 'Tasks Database',
          properties: [
            { id: 'title', name: 'Name', type: 'title' },
            { id: 'status', name: 'Status', type: 'select' },
          ],
        }),
      });

      expect(res.status).toBe(201);
      const data = await res.json() as { database: { title: string; properties: any[] }; page: { id: string } };
      expect(data.database.title).toBe('Tasks Database');
      expect(data.database.properties.length).toBe(2);
      expect(data.page).toBeDefined(); // Database creates a hosting page
    });

    it('should create database with select property and options', async () => {
      const { sessionId, workspace } = await setupTestContext(app);

      const res = await app.request('/api/v1/databases', {
        method: 'POST',
        headers: withSession({ 'Content-Type': 'application/json' }, sessionId),
        body: JSON.stringify({
          workspaceId: workspace.id,
          title: 'Project Tracker',
          properties: [
            { id: 'title', name: 'Project', type: 'title' },
            {
              id: 'priority',
              name: 'Priority',
              type: 'select',
              config: {
                options: [
                  { id: 'high', name: 'High', color: 'red' },
                  { id: 'medium', name: 'Medium', color: 'yellow' },
                  { id: 'low', name: 'Low', color: 'green' },
                ],
              },
            },
          ],
        }),
      });

      expect(res.status).toBe(201);
      const data = await res.json() as { database: { properties: { id: string; config?: { options: any[] } }[] } };
      const priorityProp = data.database.properties.find(p => p.id === 'priority');
      expect(priorityProp?.config?.options.length).toBe(3);
    });
  });

  describe('GET /api/v1/databases/:id', () => {
    it('should get database schema', async () => {
      const { sessionId, workspace } = await setupTestContext(app);
      const { database } = await createDatabase(app, sessionId, workspace.id);

      const res = await app.request(`/api/v1/databases/${database.id}`, {
        headers: withSession({}, sessionId),
      });

      expect(res.status).toBe(200);
      const data = await res.json() as { database: { id: string; title: string; properties: any[] } };
      expect(data.database.id).toBe(database.id);
      expect(data.database.properties).toBeDefined();
    });

    it('should return 404 for non-existent database', async () => {
      const { sessionId } = await setupTestContext(app);

      const res = await app.request('/api/v1/databases/non-existent-id', {
        headers: withSession({}, sessionId),
      });

      expect(res.status).toBe(404);
    });
  });

  describe('PATCH /api/v1/databases/:id', () => {
    it('should update database title', async () => {
      const { sessionId, workspace } = await setupTestContext(app);
      const { database } = await createDatabase(app, sessionId, workspace.id);

      const res = await app.request(`/api/v1/databases/${database.id}`, {
        method: 'PATCH',
        headers: withSession({ 'Content-Type': 'application/json' }, sessionId),
        body: JSON.stringify({ title: 'Updated Database' }),
      });

      expect(res.status).toBe(200);
      const data = await res.json() as { database: { title: string } };
      expect(data.database.title).toBe('Updated Database');
    });

    it('should update database icon', async () => {
      const { sessionId, workspace } = await setupTestContext(app);
      const { database } = await createDatabase(app, sessionId, workspace.id);

      const res = await app.request(`/api/v1/databases/${database.id}`, {
        method: 'PATCH',
        headers: withSession({ 'Content-Type': 'application/json' }, sessionId),
        body: JSON.stringify({ icon: '📊' }),
      });

      expect(res.status).toBe(200);
      const data = await res.json() as { database: { icon: string } };
      expect(data.database.icon).toBe('📊');
    });
  });

  describe('DELETE /api/v1/databases/:id', () => {
    it('should delete database', async () => {
      const { sessionId, workspace } = await setupTestContext(app);
      const { database } = await createDatabase(app, sessionId, workspace.id);

      const res = await app.request(`/api/v1/databases/${database.id}`, {
        method: 'DELETE',
        headers: withSession({}, sessionId),
      });

      expect(res.status).toBe(200);

      // Verify deletion
      const getRes = await app.request(`/api/v1/databases/${database.id}`, {
        headers: withSession({}, sessionId),
      });
      expect(getRes.status).toBe(404);
    });
  });

  describe('Database Properties', () => {
    describe('POST /api/v1/databases/:id/properties', () => {
      it('should add text property', async () => {
        const { sessionId, workspace } = await setupTestContext(app);
        const { database } = await createDatabase(app, sessionId, workspace.id);

        const res = await app.request(`/api/v1/databases/${database.id}/properties`, {
          method: 'POST',
          headers: withSession({ 'Content-Type': 'application/json' }, sessionId),
          body: JSON.stringify({
            name: 'Description',
            type: 'rich_text',
          }),
        });

        expect(res.status).toBe(201);
        const data = await res.json() as { property: { name: string; type: string } };
        expect(data.property.name).toBe('Description');
        expect(data.property.type).toBe('rich_text');
      });

      it('should add number property', async () => {
        const { sessionId, workspace } = await setupTestContext(app);
        const { database } = await createDatabase(app, sessionId, workspace.id);

        const res = await app.request(`/api/v1/databases/${database.id}/properties`, {
          method: 'POST',
          headers: withSession({ 'Content-Type': 'application/json' }, sessionId),
          body: JSON.stringify({
            name: 'Price',
            type: 'number',
            config: { format: 'dollar' },
          }),
        });

        expect(res.status).toBe(201);
        const data = await res.json() as { property: { type: string; config: { format: string } } };
        expect(data.property.type).toBe('number');
        expect(data.property.config.format).toBe('dollar');
      });

      it('should add date property', async () => {
        const { sessionId, workspace } = await setupTestContext(app);
        const { database } = await createDatabase(app, sessionId, workspace.id);

        const res = await app.request(`/api/v1/databases/${database.id}/properties`, {
          method: 'POST',
          headers: withSession({ 'Content-Type': 'application/json' }, sessionId),
          body: JSON.stringify({
            name: 'Due Date',
            type: 'date',
          }),
        });

        expect(res.status).toBe(201);
        const data = await res.json() as { property: { type: string } };
        expect(data.property.type).toBe('date');
      });

      it('should add multi-select property', async () => {
        const { sessionId, workspace } = await setupTestContext(app);
        const { database } = await createDatabase(app, sessionId, workspace.id);

        const res = await app.request(`/api/v1/databases/${database.id}/properties`, {
          method: 'POST',
          headers: withSession({ 'Content-Type': 'application/json' }, sessionId),
          body: JSON.stringify({
            name: 'Tags',
            type: 'multi_select',
            config: {
              options: [
                { id: 'bug', name: 'Bug', color: 'red' },
                { id: 'feature', name: 'Feature', color: 'blue' },
              ],
            },
          }),
        });

        expect(res.status).toBe(201);
        const data = await res.json() as { property: { type: string } };
        expect(data.property.type).toBe('multi_select');
      });
    });

    describe('PATCH /api/v1/databases/:id/properties/:propId', () => {
      it('should update property name', async () => {
        const { sessionId, workspace } = await setupTestContext(app);
        const { database } = await createDatabase(app, sessionId, workspace.id);
        const propId = database.id ? 'status' : 'status'; // Property from initial creation

        // First add a property
        const addRes = await app.request(`/api/v1/databases/${database.id}/properties`, {
          method: 'POST',
          headers: withSession({ 'Content-Type': 'application/json' }, sessionId),
          body: JSON.stringify({
            name: 'Old Name',
            type: 'rich_text',
          }),
        });
        const addedProp = (await addRes.json() as { property: { id: string } }).property;

        const res = await app.request(`/api/v1/databases/${database.id}/properties/${addedProp.id}`, {
          method: 'PATCH',
          headers: withSession({ 'Content-Type': 'application/json' }, sessionId),
          body: JSON.stringify({ name: 'New Name' }),
        });

        expect(res.status).toBe(200);
        const data = await res.json() as { property: { name: string } };
        expect(data.property.name).toBe('New Name');
      });
    });

    describe('DELETE /api/v1/databases/:id/properties/:propId', () => {
      it('should delete property', async () => {
        const { sessionId, workspace } = await setupTestContext(app);
        const { database } = await createDatabase(app, sessionId, workspace.id);

        // First add a property
        const addRes = await app.request(`/api/v1/databases/${database.id}/properties`, {
          method: 'POST',
          headers: withSession({ 'Content-Type': 'application/json' }, sessionId),
          body: JSON.stringify({
            name: 'To Delete',
            type: 'rich_text',
          }),
        });
        const addedProp = (await addRes.json() as { property: { id: string } }).property;

        const res = await app.request(`/api/v1/databases/${database.id}/properties/${addedProp.id}`, {
          method: 'DELETE',
          headers: withSession({}, sessionId),
        });

        expect(res.status).toBe(200);
      });
    });
  });

  describe('Database Rows', () => {
    describe('POST /api/v1/databases/:id/rows', () => {
      it('should create a row', async () => {
        const { sessionId, workspace } = await setupTestContext(app);
        const { database } = await createDatabase(app, sessionId, workspace.id);

        const res = await app.request(`/api/v1/databases/${database.id}/rows`, {
          method: 'POST',
          headers: withSession({ 'Content-Type': 'application/json' }, sessionId),
          body: JSON.stringify({
            properties: {
              title: 'Task 1',
              status: 'todo',
            },
          }),
        });

        expect(res.status).toBe(201);
        const data = await res.json() as { row: { properties: { title: string } } };
        expect(data.row.properties.title).toBe('Task 1');
      });
    });

    describe('GET /api/v1/databases/:id/rows', () => {
      it('should list rows with pagination', async () => {
        const { sessionId, workspace } = await setupTestContext(app);
        const { database } = await createDatabase(app, sessionId, workspace.id);

        // Create multiple rows
        for (let i = 0; i < 5; i++) {
          await app.request(`/api/v1/databases/${database.id}/rows`, {
            method: 'POST',
            headers: withSession({ 'Content-Type': 'application/json' }, sessionId),
            body: JSON.stringify({
              properties: { title: `Task ${i + 1}` },
            }),
          });
        }

        const res = await app.request(`/api/v1/databases/${database.id}/rows?limit=3`, {
          headers: withSession({}, sessionId),
        });

        expect(res.status).toBe(200);
        const data = await res.json() as { items: any[]; hasMore: boolean };
        expect(data.items.length).toBe(3);
        expect(data.hasMore).toBe(true);
      });
    });
  });
});
