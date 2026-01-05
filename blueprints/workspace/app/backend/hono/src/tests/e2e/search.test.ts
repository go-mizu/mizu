import { describe, it, expect, beforeEach } from 'vitest';
import { createTestApp, withSession, setupTestContext, createPage, createDatabase } from './setup';
import type { TestApp } from './setup';

describe('Search E2E Tests', () => {
  let app: TestApp;

  beforeEach(() => {
    app = createTestApp();
  });

  describe('GET /api/v1/workspaces/:id/search', () => {
    it('should search pages by title', async () => {
      const { sessionId, workspace } = await setupTestContext(app);

      await createPage(app, sessionId, workspace.id, { title: 'Meeting Notes' });
      await createPage(app, sessionId, workspace.id, { title: 'Project Plan' });
      await createPage(app, sessionId, workspace.id, { title: 'Meeting Agenda' });

      const res = await app.request(`/api/v1/workspaces/${workspace.id}/search?q=Meeting`, {
        headers: withSession({}, sessionId),
      });

      expect(res.status).toBe(200);
      const data = await res.json() as { results: { title: string }[] };
      expect(data.results.length).toBe(2);
      expect(data.results.every((r: any) => r.title.includes('Meeting'))).toBe(true);
    });

    it('should search databases by title', async () => {
      const { sessionId, workspace } = await setupTestContext(app);

      await createDatabase(app, sessionId, workspace.id, { title: 'Task Tracker' });
      await createDatabase(app, sessionId, workspace.id, { title: 'Bug Database' });
      await createPage(app, sessionId, workspace.id, { title: 'Task List' });

      const res = await app.request(`/api/v1/workspaces/${workspace.id}/search?q=Task`, {
        headers: withSession({}, sessionId),
      });

      expect(res.status).toBe(200);
      const data = await res.json() as { results: { title: string }[] };
      expect(data.results.length).toBeGreaterThanOrEqual(2);
    });

    it('should exclude archived pages from search', async () => {
      const { sessionId, workspace } = await setupTestContext(app);

      const page = await createPage(app, sessionId, workspace.id, { title: 'Active Page' });
      const archivedPage = await createPage(app, sessionId, workspace.id, { title: 'Archived Page' });

      // Archive the page
      await app.request(`/api/v1/pages/${archivedPage.id}/archive`, {
        method: 'POST',
        headers: withSession({}, sessionId),
      });

      const res = await app.request(`/api/v1/workspaces/${workspace.id}/search?q=Page`, {
        headers: withSession({}, sessionId),
      });

      expect(res.status).toBe(200);
      const data = await res.json() as { results: { id: string }[] };
      expect(data.results.find((r: any) => r.id === archivedPage.id)).toBeUndefined();
    });

    it('should return empty results for no matches', async () => {
      const { sessionId, workspace } = await setupTestContext(app);

      await createPage(app, sessionId, workspace.id, { title: 'Test Page' });

      const res = await app.request(`/api/v1/workspaces/${workspace.id}/search?q=NonexistentTerm`, {
        headers: withSession({}, sessionId),
      });

      expect(res.status).toBe(200);
      const data = await res.json() as { results: any[] };
      expect(data.results.length).toBe(0);
    });

    it('should be case insensitive', async () => {
      const { sessionId, workspace } = await setupTestContext(app);

      await createPage(app, sessionId, workspace.id, { title: 'Important Document' });

      const res = await app.request(`/api/v1/workspaces/${workspace.id}/search?q=important`, {
        headers: withSession({}, sessionId),
      });

      expect(res.status).toBe(200);
      const data = await res.json() as { results: { title: string }[] };
      expect(data.results.length).toBe(1);
    });
  });

  describe('GET /api/v1/workspaces/:id/quick-search', () => {
    it('should return limited results', async () => {
      const { sessionId, workspace } = await setupTestContext(app);

      // Create many pages
      for (let i = 0; i < 10; i++) {
        await createPage(app, sessionId, workspace.id, { title: `Document ${i}` });
      }

      const res = await app.request(`/api/v1/workspaces/${workspace.id}/quick-search?q=Document`, {
        headers: withSession({}, sessionId),
      });

      expect(res.status).toBe(200);
      const data = await res.json() as { results: any[] };
      expect(data.results.length).toBeLessThanOrEqual(5);
    });

    it('should prioritize exact matches', async () => {
      const { sessionId, workspace } = await setupTestContext(app);

      await createPage(app, sessionId, workspace.id, { title: 'Meeting' });
      await createPage(app, sessionId, workspace.id, { title: 'Meeting Notes Long Title' });
      await createPage(app, sessionId, workspace.id, { title: 'Unrelated Meeting Stuff' });

      const res = await app.request(`/api/v1/workspaces/${workspace.id}/quick-search?q=Meeting`, {
        headers: withSession({}, sessionId),
      });

      expect(res.status).toBe(200);
      const data = await res.json() as { results: { title: string }[] };
      // Exact match should come first
      if (data.results.length > 0) {
        expect(data.results[0].title).toBe('Meeting');
      }
    });
  });

  describe('GET /api/v1/workspaces/:id/recent', () => {
    it('should return recently updated pages', async () => {
      const { sessionId, workspace } = await setupTestContext(app);

      const page1 = await createPage(app, sessionId, workspace.id, { title: 'Old Page' });
      const page2 = await createPage(app, sessionId, workspace.id, { title: 'New Page' });

      // Update page2 to make it more recent
      await app.request(`/api/v1/pages/${page2.id}`, {
        method: 'PATCH',
        headers: withSession({ 'Content-Type': 'application/json' }, sessionId),
        body: JSON.stringify({ title: 'Updated New Page' }),
      });

      const res = await app.request(`/api/v1/workspaces/${workspace.id}/recent`, {
        headers: withSession({}, sessionId),
      });

      expect(res.status).toBe(200);
      const data = await res.json() as { items: { id: string }[] };
      expect(data.items.length).toBeGreaterThanOrEqual(2);
      // Most recently updated should be first
      expect(data.items[0].id).toBe(page2.id);
    });

    it('should respect limit parameter', async () => {
      const { sessionId, workspace } = await setupTestContext(app);

      for (let i = 0; i < 10; i++) {
        await createPage(app, sessionId, workspace.id, { title: `Page ${i}` });
      }

      const res = await app.request(`/api/v1/workspaces/${workspace.id}/recent?limit=5`, {
        headers: withSession({}, sessionId),
      });

      expect(res.status).toBe(200);
      const data = await res.json() as { items: any[] };
      expect(data.items.length).toBe(5);
    });

    it('should exclude database rows', async () => {
      const { sessionId, workspace } = await setupTestContext(app);

      await createPage(app, sessionId, workspace.id, { title: 'Regular Page' });
      const { database } = await createDatabase(app, sessionId, workspace.id);

      // Create a row in the database
      await app.request(`/api/v1/databases/${database.id}/rows`, {
        method: 'POST',
        headers: withSession({ 'Content-Type': 'application/json' }, sessionId),
        body: JSON.stringify({ properties: { title: 'Row Item' } }),
      });

      const res = await app.request(`/api/v1/workspaces/${workspace.id}/recent`, {
        headers: withSession({}, sessionId),
      });

      expect(res.status).toBe(200);
      const data = await res.json() as { items: { title: string }[] };
      // Should not include the row
      expect(data.items.find((i: any) => i.title === 'Row Item')).toBeUndefined();
    });
  });

  describe('Database Search', () => {
    it('should search database rows by title', async () => {
      const { sessionId, workspace } = await setupTestContext(app);
      const { database } = await createDatabase(app, sessionId, workspace.id);

      // Create rows
      await app.request(`/api/v1/databases/${database.id}/rows`, {
        method: 'POST',
        headers: withSession({ 'Content-Type': 'application/json' }, sessionId),
        body: JSON.stringify({ properties: { title: 'Fix login bug' } }),
      });
      await app.request(`/api/v1/databases/${database.id}/rows`, {
        method: 'POST',
        headers: withSession({ 'Content-Type': 'application/json' }, sessionId),
        body: JSON.stringify({ properties: { title: 'Add feature' } }),
      });
      await app.request(`/api/v1/databases/${database.id}/rows`, {
        method: 'POST',
        headers: withSession({ 'Content-Type': 'application/json' }, sessionId),
        body: JSON.stringify({ properties: { title: 'Fix styling bug' } }),
      });

      const res = await app.request(`/api/v1/workspaces/${workspace.id}/search?q=bug`, {
        headers: withSession({}, sessionId),
      });

      expect(res.status).toBe(200);
      const data = await res.json() as { results: { title: string }[] };
      const bugResults = data.results.filter((r: any) => r.title && r.title.includes('bug'));
      expect(bugResults.length).toBe(2);
    });
  });
});
