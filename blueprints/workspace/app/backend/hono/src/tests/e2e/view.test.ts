import { describe, it, expect, beforeEach } from 'vitest';
import { createTestApp, withSession, setupTestContext, createDatabase } from './setup';
import type { TestApp } from './setup';

describe('View E2E Tests', () => {
  let app: TestApp;

  beforeEach(() => {
    app = createTestApp();
  });

  describe('POST /api/v1/databases/:id/views', () => {
    it('should create a table view', async () => {
      const { sessionId, workspace } = await setupTestContext(app);
      const { database } = await createDatabase(app, sessionId, workspace.id);

      const res = await app.request(`/api/v1/databases/${database.id}/views`, {
        method: 'POST',
        headers: withSession({ 'Content-Type': 'application/json' }, sessionId),
        body: JSON.stringify({
          name: 'All Tasks',
          type: 'table',
        }),
      });

      expect(res.status).toBe(201);
      const data = await res.json() as { view: { name: string; type: string } };
      expect(data.view.name).toBe('All Tasks');
      expect(data.view.type).toBe('table');
    });

    it('should create a board view with groupBy', async () => {
      const { sessionId, workspace } = await setupTestContext(app);
      const { database } = await createDatabase(app, sessionId, workspace.id);

      const res = await app.request(`/api/v1/databases/${database.id}/views`, {
        method: 'POST',
        headers: withSession({ 'Content-Type': 'application/json' }, sessionId),
        body: JSON.stringify({
          name: 'Kanban Board',
          type: 'board',
          groupBy: 'status',
        }),
      });

      expect(res.status).toBe(201);
      const data = await res.json() as { view: { type: string; groupBy: string } };
      expect(data.view.type).toBe('board');
      expect(data.view.groupBy).toBe('status');
    });

    it('should create a calendar view', async () => {
      const { sessionId, workspace } = await setupTestContext(app);

      // Create database with date property
      const dbRes = await app.request('/api/v1/databases', {
        method: 'POST',
        headers: withSession({ 'Content-Type': 'application/json' }, sessionId),
        body: JSON.stringify({
          workspaceId: workspace.id,
          title: 'Events',
          properties: [
            { id: 'title', name: 'Event', type: 'title' },
            { id: 'date', name: 'Date', type: 'date' },
          ],
        }),
      });
      const database = (await dbRes.json() as { database: { id: string } }).database;

      const res = await app.request(`/api/v1/databases/${database.id}/views`, {
        method: 'POST',
        headers: withSession({ 'Content-Type': 'application/json' }, sessionId),
        body: JSON.stringify({
          name: 'Calendar',
          type: 'calendar',
          calendarBy: 'date',
        }),
      });

      expect(res.status).toBe(201);
      const data = await res.json() as { view: { type: string; calendarBy: string } };
      expect(data.view.type).toBe('calendar');
      expect(data.view.calendarBy).toBe('date');
    });

    it('should create a gallery view', async () => {
      const { sessionId, workspace } = await setupTestContext(app);
      const { database } = await createDatabase(app, sessionId, workspace.id);

      const res = await app.request(`/api/v1/databases/${database.id}/views`, {
        method: 'POST',
        headers: withSession({ 'Content-Type': 'application/json' }, sessionId),
        body: JSON.stringify({
          name: 'Gallery',
          type: 'gallery',
        }),
      });

      expect(res.status).toBe(201);
      const data = await res.json() as { view: { type: string } };
      expect(data.view.type).toBe('gallery');
    });

    it('should create a list view', async () => {
      const { sessionId, workspace } = await setupTestContext(app);
      const { database } = await createDatabase(app, sessionId, workspace.id);

      const res = await app.request(`/api/v1/databases/${database.id}/views`, {
        method: 'POST',
        headers: withSession({ 'Content-Type': 'application/json' }, sessionId),
        body: JSON.stringify({
          name: 'List View',
          type: 'list',
        }),
      });

      expect(res.status).toBe(201);
      const data = await res.json() as { view: { type: string } };
      expect(data.view.type).toBe('list');
    });
  });

  describe('GET /api/v1/databases/:id/views', () => {
    it('should list views for database', async () => {
      const { sessionId, workspace } = await setupTestContext(app);
      const { database } = await createDatabase(app, sessionId, workspace.id);

      // Create multiple views
      await app.request(`/api/v1/databases/${database.id}/views`, {
        method: 'POST',
        headers: withSession({ 'Content-Type': 'application/json' }, sessionId),
        body: JSON.stringify({ name: 'Table', type: 'table' }),
      });
      await app.request(`/api/v1/databases/${database.id}/views`, {
        method: 'POST',
        headers: withSession({ 'Content-Type': 'application/json' }, sessionId),
        body: JSON.stringify({ name: 'Board', type: 'board' }),
      });

      const res = await app.request(`/api/v1/databases/${database.id}/views`, {
        headers: withSession({}, sessionId),
      });

      expect(res.status).toBe(200);
      const data = await res.json() as { views: any[] };
      expect(data.views.length).toBe(2);
    });
  });

  describe('GET /api/v1/views/:id', () => {
    it('should get view by id', async () => {
      const { sessionId, workspace } = await setupTestContext(app);
      const { database } = await createDatabase(app, sessionId, workspace.id);

      const createRes = await app.request(`/api/v1/databases/${database.id}/views`, {
        method: 'POST',
        headers: withSession({ 'Content-Type': 'application/json' }, sessionId),
        body: JSON.stringify({ name: 'My View', type: 'table' }),
      });
      const view = (await createRes.json() as { view: { id: string } }).view;

      const res = await app.request(`/api/v1/views/${view.id}`, {
        headers: withSession({}, sessionId),
      });

      expect(res.status).toBe(200);
      const data = await res.json() as { view: { id: string; name: string } };
      expect(data.view.id).toBe(view.id);
      expect(data.view.name).toBe('My View');
    });
  });

  describe('PATCH /api/v1/views/:id', () => {
    it('should update view name', async () => {
      const { sessionId, workspace } = await setupTestContext(app);
      const { database } = await createDatabase(app, sessionId, workspace.id);

      const createRes = await app.request(`/api/v1/databases/${database.id}/views`, {
        method: 'POST',
        headers: withSession({ 'Content-Type': 'application/json' }, sessionId),
        body: JSON.stringify({ name: 'Old Name', type: 'table' }),
      });
      const view = (await createRes.json() as { view: { id: string } }).view;

      const res = await app.request(`/api/v1/views/${view.id}`, {
        method: 'PATCH',
        headers: withSession({ 'Content-Type': 'application/json' }, sessionId),
        body: JSON.stringify({ name: 'New Name' }),
      });

      expect(res.status).toBe(200);
      const data = await res.json() as { view: { name: string } };
      expect(data.view.name).toBe('New Name');
    });

    it('should update view filters', async () => {
      const { sessionId, workspace } = await setupTestContext(app);
      const { database } = await createDatabase(app, sessionId, workspace.id);

      const createRes = await app.request(`/api/v1/databases/${database.id}/views`, {
        method: 'POST',
        headers: withSession({ 'Content-Type': 'application/json' }, sessionId),
        body: JSON.stringify({ name: 'Filtered View', type: 'table' }),
      });
      const view = (await createRes.json() as { view: { id: string } }).view;

      const res = await app.request(`/api/v1/views/${view.id}`, {
        method: 'PATCH',
        headers: withSession({ 'Content-Type': 'application/json' }, sessionId),
        body: JSON.stringify({
          filter: {
            type: 'and',
            conditions: [
              { propertyId: 'status', operator: 'equals', value: 'todo' },
            ],
          },
        }),
      });

      expect(res.status).toBe(200);
      const data = await res.json() as { view: { filter: { type: string } } };
      expect(data.view.filter).toBeDefined();
      expect(data.view.filter.type).toBe('and');
    });

    it('should update view sorts', async () => {
      const { sessionId, workspace } = await setupTestContext(app);
      const { database } = await createDatabase(app, sessionId, workspace.id);

      const createRes = await app.request(`/api/v1/databases/${database.id}/views`, {
        method: 'POST',
        headers: withSession({ 'Content-Type': 'application/json' }, sessionId),
        body: JSON.stringify({ name: 'Sorted View', type: 'table' }),
      });
      const view = (await createRes.json() as { view: { id: string } }).view;

      const res = await app.request(`/api/v1/views/${view.id}`, {
        method: 'PATCH',
        headers: withSession({ 'Content-Type': 'application/json' }, sessionId),
        body: JSON.stringify({
          sorts: [
            { propertyId: 'title', direction: 'ascending' },
          ],
        }),
      });

      expect(res.status).toBe(200);
      const data = await res.json() as { view: { sorts: { propertyId: string; direction: string }[] } };
      expect(data.view.sorts).toBeDefined();
      expect(data.view.sorts[0].propertyId).toBe('title');
    });
  });

  describe('DELETE /api/v1/views/:id', () => {
    it('should delete view', async () => {
      const { sessionId, workspace } = await setupTestContext(app);
      const { database } = await createDatabase(app, sessionId, workspace.id);

      const createRes = await app.request(`/api/v1/databases/${database.id}/views`, {
        method: 'POST',
        headers: withSession({ 'Content-Type': 'application/json' }, sessionId),
        body: JSON.stringify({ name: 'To Delete', type: 'table' }),
      });
      const view = (await createRes.json() as { view: { id: string } }).view;

      const res = await app.request(`/api/v1/views/${view.id}`, {
        method: 'DELETE',
        headers: withSession({}, sessionId),
      });

      expect(res.status).toBe(200);

      // Verify deletion
      const getRes = await app.request(`/api/v1/views/${view.id}`, {
        headers: withSession({}, sessionId),
      });
      expect(getRes.status).toBe(404);
    });
  });

  describe('POST /api/v1/views/:id/query', () => {
    it('should query view with filters', async () => {
      const { sessionId, workspace } = await setupTestContext(app);
      const { database } = await createDatabase(app, sessionId, workspace.id);

      // Create rows
      await app.request(`/api/v1/databases/${database.id}/rows`, {
        method: 'POST',
        headers: withSession({ 'Content-Type': 'application/json' }, sessionId),
        body: JSON.stringify({ properties: { title: 'Task 1', status: 'todo' } }),
      });
      await app.request(`/api/v1/databases/${database.id}/rows`, {
        method: 'POST',
        headers: withSession({ 'Content-Type': 'application/json' }, sessionId),
        body: JSON.stringify({ properties: { title: 'Task 2', status: 'done' } }),
      });

      // Create view
      const viewRes = await app.request(`/api/v1/databases/${database.id}/views`, {
        method: 'POST',
        headers: withSession({ 'Content-Type': 'application/json' }, sessionId),
        body: JSON.stringify({ name: 'Query View', type: 'table' }),
      });
      const view = (await viewRes.json() as { view: { id: string } }).view;

      // Query with filter
      const res = await app.request(`/api/v1/views/${view.id}/query`, {
        method: 'POST',
        headers: withSession({ 'Content-Type': 'application/json' }, sessionId),
        body: JSON.stringify({
          filter: {
            type: 'and',
            conditions: [
              { propertyId: 'status', operator: 'equals', value: 'todo' },
            ],
          },
        }),
      });

      expect(res.status).toBe(200);
      const data = await res.json() as { items: { properties: { status: string } }[] };
      expect(data.items.every((item: any) => item.properties.status === 'todo')).toBe(true);
    });

    it('should query view with sorting', async () => {
      const { sessionId, workspace } = await setupTestContext(app);
      const { database } = await createDatabase(app, sessionId, workspace.id);

      // Create rows
      await app.request(`/api/v1/databases/${database.id}/rows`, {
        method: 'POST',
        headers: withSession({ 'Content-Type': 'application/json' }, sessionId),
        body: JSON.stringify({ properties: { title: 'Zebra' } }),
      });
      await app.request(`/api/v1/databases/${database.id}/rows`, {
        method: 'POST',
        headers: withSession({ 'Content-Type': 'application/json' }, sessionId),
        body: JSON.stringify({ properties: { title: 'Apple' } }),
      });

      // Create view
      const viewRes = await app.request(`/api/v1/databases/${database.id}/views`, {
        method: 'POST',
        headers: withSession({ 'Content-Type': 'application/json' }, sessionId),
        body: JSON.stringify({ name: 'Sorted View', type: 'table' }),
      });
      const view = (await viewRes.json() as { view: { id: string } }).view;

      // Query with sort
      const res = await app.request(`/api/v1/views/${view.id}/query`, {
        method: 'POST',
        headers: withSession({ 'Content-Type': 'application/json' }, sessionId),
        body: JSON.stringify({
          sorts: [{ propertyId: 'title', direction: 'ascending' }],
        }),
      });

      expect(res.status).toBe(200);
      const data = await res.json() as { items: { properties: { title: string } }[] };
      // First item should be 'Apple' if sorted ascending
      if (data.items.length >= 2) {
        expect(data.items[0].properties.title).toBe('Apple');
      }
    });
  });
});
