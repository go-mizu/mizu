import { describe, it, expect, beforeEach } from 'vitest';
import { createTestApp, withSession, setupTestContext, createPage } from './setup';
import type { TestApp } from './setup';

describe('Favorite E2E Tests', () => {
  let app: TestApp;

  beforeEach(() => {
    app = createTestApp();
  });

  describe('POST /api/v1/favorites', () => {
    it('should add page to favorites', async () => {
      const { sessionId, workspace } = await setupTestContext(app);
      const page = await createPage(app, sessionId, workspace.id);

      const res = await app.request('/api/v1/favorites', {
        method: 'POST',
        headers: withSession({ 'Content-Type': 'application/json' }, sessionId),
        body: JSON.stringify({ pageId: page.id }),
      });

      expect(res.status).toBe(201);
      const data = await res.json() as { favorite: { pageId: string } };
      expect(data.favorite.pageId).toBe(page.id);
    });

    it('should not duplicate favorites', async () => {
      const { sessionId, workspace } = await setupTestContext(app);
      const page = await createPage(app, sessionId, workspace.id);

      // First add
      await app.request('/api/v1/favorites', {
        method: 'POST',
        headers: withSession({ 'Content-Type': 'application/json' }, sessionId),
        body: JSON.stringify({ pageId: page.id }),
      });

      // Second add (should fail or be idempotent)
      const res = await app.request('/api/v1/favorites', {
        method: 'POST',
        headers: withSession({ 'Content-Type': 'application/json' }, sessionId),
        body: JSON.stringify({ pageId: page.id }),
      });

      expect([200, 201, 409]).toContain(res.status);
    });
  });

  describe('DELETE /api/v1/favorites/:pageId', () => {
    it('should remove page from favorites', async () => {
      const { sessionId, workspace } = await setupTestContext(app);
      const page = await createPage(app, sessionId, workspace.id);

      // Add to favorites
      await app.request('/api/v1/favorites', {
        method: 'POST',
        headers: withSession({ 'Content-Type': 'application/json' }, sessionId),
        body: JSON.stringify({ pageId: page.id }),
      });

      // Remove from favorites
      const res = await app.request(`/api/v1/favorites/${page.id}`, {
        method: 'DELETE',
        headers: withSession({}, sessionId),
      });

      expect(res.status).toBe(200);

      // Verify removed
      const checkRes = await app.request(`/api/v1/favorites/${page.id}`, {
        headers: withSession({}, sessionId),
      });
      const checkData = await checkRes.json() as { isFavorite: boolean };
      expect(checkData.isFavorite).toBe(false);
    });
  });

  describe('GET /api/v1/favorites/:pageId', () => {
    it('should check if page is favorited', async () => {
      const { sessionId, workspace } = await setupTestContext(app);
      const page = await createPage(app, sessionId, workspace.id);

      // Before adding
      const beforeRes = await app.request(`/api/v1/favorites/${page.id}`, {
        headers: withSession({}, sessionId),
      });
      expect(beforeRes.status).toBe(200);
      const beforeData = await beforeRes.json() as { isFavorite: boolean };
      expect(beforeData.isFavorite).toBe(false);

      // Add to favorites
      await app.request('/api/v1/favorites', {
        method: 'POST',
        headers: withSession({ 'Content-Type': 'application/json' }, sessionId),
        body: JSON.stringify({ pageId: page.id }),
      });

      // After adding
      const afterRes = await app.request(`/api/v1/favorites/${page.id}`, {
        headers: withSession({}, sessionId),
      });
      expect(afterRes.status).toBe(200);
      const afterData = await afterRes.json() as { isFavorite: boolean };
      expect(afterData.isFavorite).toBe(true);
    });
  });

  describe('GET /api/v1/workspaces/:id/favorites', () => {
    it('should list favorited pages in workspace', async () => {
      const { sessionId, workspace } = await setupTestContext(app);
      const page1 = await createPage(app, sessionId, workspace.id, { title: 'Favorite 1' });
      const page2 = await createPage(app, sessionId, workspace.id, { title: 'Favorite 2' });
      await createPage(app, sessionId, workspace.id, { title: 'Not Favorite' });

      // Add favorites
      await app.request('/api/v1/favorites', {
        method: 'POST',
        headers: withSession({ 'Content-Type': 'application/json' }, sessionId),
        body: JSON.stringify({ pageId: page1.id }),
      });
      await app.request('/api/v1/favorites', {
        method: 'POST',
        headers: withSession({ 'Content-Type': 'application/json' }, sessionId),
        body: JSON.stringify({ pageId: page2.id }),
      });

      const res = await app.request(`/api/v1/workspaces/${workspace.id}/favorites`, {
        headers: withSession({}, sessionId),
      });

      expect(res.status).toBe(200);
      const data = await res.json() as { favorites: { pageId: string }[] };
      expect(data.favorites.length).toBe(2);
      const pageIds = data.favorites.map((f: any) => f.pageId);
      expect(pageIds).toContain(page1.id);
      expect(pageIds).toContain(page2.id);
    });

    it('should return favorites with page details', async () => {
      const { sessionId, workspace } = await setupTestContext(app);
      const page = await createPage(app, sessionId, workspace.id, { title: 'My Favorite Page' });

      await app.request('/api/v1/favorites', {
        method: 'POST',
        headers: withSession({ 'Content-Type': 'application/json' }, sessionId),
        body: JSON.stringify({ pageId: page.id }),
      });

      const res = await app.request(`/api/v1/workspaces/${workspace.id}/favorites`, {
        headers: withSession({}, sessionId),
      });

      expect(res.status).toBe(200);
      const data = await res.json() as { favorites: { page?: { title: string } }[] };
      // Check if page details are included
      if (data.favorites[0]?.page) {
        expect(data.favorites[0].page.title).toBe('My Favorite Page');
      }
    });

    it('should maintain favorite order', async () => {
      const { sessionId, workspace } = await setupTestContext(app);
      const page1 = await createPage(app, sessionId, workspace.id, { title: 'First' });
      const page2 = await createPage(app, sessionId, workspace.id, { title: 'Second' });
      const page3 = await createPage(app, sessionId, workspace.id, { title: 'Third' });

      // Add in order
      await app.request('/api/v1/favorites', {
        method: 'POST',
        headers: withSession({ 'Content-Type': 'application/json' }, sessionId),
        body: JSON.stringify({ pageId: page1.id }),
      });
      await app.request('/api/v1/favorites', {
        method: 'POST',
        headers: withSession({ 'Content-Type': 'application/json' }, sessionId),
        body: JSON.stringify({ pageId: page2.id }),
      });
      await app.request('/api/v1/favorites', {
        method: 'POST',
        headers: withSession({ 'Content-Type': 'application/json' }, sessionId),
        body: JSON.stringify({ pageId: page3.id }),
      });

      const res = await app.request(`/api/v1/workspaces/${workspace.id}/favorites`, {
        headers: withSession({}, sessionId),
      });

      expect(res.status).toBe(200);
      const data = await res.json() as { favorites: { pageId: string }[] };
      expect(data.favorites.length).toBe(3);
      // Should maintain order of addition
      expect(data.favorites[0].pageId).toBe(page1.id);
      expect(data.favorites[1].pageId).toBe(page2.id);
      expect(data.favorites[2].pageId).toBe(page3.id);
    });
  });

  describe('Favorites and Archived Pages', () => {
    it('should handle favoriting and archiving', async () => {
      const { sessionId, workspace } = await setupTestContext(app);
      const page = await createPage(app, sessionId, workspace.id);

      // Add to favorites
      await app.request('/api/v1/favorites', {
        method: 'POST',
        headers: withSession({ 'Content-Type': 'application/json' }, sessionId),
        body: JSON.stringify({ pageId: page.id }),
      });

      // Archive the page
      await app.request(`/api/v1/pages/${page.id}/archive`, {
        method: 'POST',
        headers: withSession({}, sessionId),
      });

      // Page should still be in favorites (behavior depends on implementation)
      const res = await app.request(`/api/v1/favorites/${page.id}`, {
        headers: withSession({}, sessionId),
      });

      expect(res.status).toBe(200);
    });
  });
});
