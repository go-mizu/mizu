import { describe, it, expect, beforeEach } from 'vitest';
import { createTestApp, withSession, setupTestContext, createPage } from './setup';
import type { TestApp } from './setup';

describe('Page E2E Tests', () => {
  let app: TestApp;

  beforeEach(() => {
    app = createTestApp();
  });

  describe('POST /api/v1/pages', () => {
    it('should create a root page', async () => {
      const { sessionId, workspace, user } = await setupTestContext(app);

      const res = await app.request('/api/v1/pages', {
        method: 'POST',
        headers: withSession({ 'Content-Type': 'application/json' }, sessionId),
        body: JSON.stringify({
          workspaceId: workspace.id,
          title: 'My First Page',
        }),
      });

      expect(res.status).toBe(201);
      const data = await res.json() as { page: { id: string; title: string; workspaceId: string; createdBy: string } };
      expect(data.page.title).toBe('My First Page');
      expect(data.page.workspaceId).toBe(workspace.id);
      expect(data.page.createdBy).toBe(user.id);
    });

    it('should create a nested page', async () => {
      const { sessionId, workspace } = await setupTestContext(app);
      const parentPage = await createPage(app, sessionId, workspace.id, { title: 'Parent Page' });

      const res = await app.request('/api/v1/pages', {
        method: 'POST',
        headers: withSession({ 'Content-Type': 'application/json' }, sessionId),
        body: JSON.stringify({
          workspaceId: workspace.id,
          title: 'Child Page',
          parentId: parentPage.id,
          parentType: 'page',
        }),
      });

      expect(res.status).toBe(201);
      const data = await res.json() as { page: { parentId: string; parentType: string } };
      expect(data.page.parentId).toBe(parentPage.id);
      expect(data.page.parentType).toBe('page');
    });

    it('should create page with icon', async () => {
      const { sessionId, workspace } = await setupTestContext(app);

      const res = await app.request('/api/v1/pages', {
        method: 'POST',
        headers: withSession({ 'Content-Type': 'application/json' }, sessionId),
        body: JSON.stringify({
          workspaceId: workspace.id,
          title: 'Page with Icon',
          icon: '📚',
        }),
      });

      expect(res.status).toBe(201);
      const data = await res.json() as { page: { icon: string } };
      expect(data.page.icon).toBe('📚');
    });
  });

  describe('GET /api/v1/pages/:id', () => {
    it('should get page by id', async () => {
      const { sessionId, workspace } = await setupTestContext(app);
      const page = await createPage(app, sessionId, workspace.id);

      const res = await app.request(`/api/v1/pages/${page.id}`, {
        headers: withSession({}, sessionId),
      });

      expect(res.status).toBe(200);
      const data = await res.json() as { page: { id: string; title: string } };
      expect(data.page.id).toBe(page.id);
      expect(data.page.title).toBe(page.title);
    });

    it('should return 404 for non-existent page', async () => {
      const { sessionId } = await setupTestContext(app);

      const res = await app.request('/api/v1/pages/non-existent-id', {
        headers: withSession({}, sessionId),
      });

      expect(res.status).toBe(404);
    });
  });

  describe('PATCH /api/v1/pages/:id', () => {
    it('should update page title', async () => {
      const { sessionId, workspace } = await setupTestContext(app);
      const page = await createPage(app, sessionId, workspace.id);

      const res = await app.request(`/api/v1/pages/${page.id}`, {
        method: 'PATCH',
        headers: withSession({ 'Content-Type': 'application/json' }, sessionId),
        body: JSON.stringify({ title: 'Updated Title' }),
      });

      expect(res.status).toBe(200);
      const data = await res.json() as { page: { title: string } };
      expect(data.page.title).toBe('Updated Title');
    });

    it('should update page icon', async () => {
      const { sessionId, workspace } = await setupTestContext(app);
      const page = await createPage(app, sessionId, workspace.id);

      const res = await app.request(`/api/v1/pages/${page.id}`, {
        method: 'PATCH',
        headers: withSession({ 'Content-Type': 'application/json' }, sessionId),
        body: JSON.stringify({ icon: '🎉' }),
      });

      expect(res.status).toBe(200);
      const data = await res.json() as { page: { icon: string } };
      expect(data.page.icon).toBe('🎉');
    });

    it('should update page cover', async () => {
      const { sessionId, workspace } = await setupTestContext(app);
      const page = await createPage(app, sessionId, workspace.id);

      const res = await app.request(`/api/v1/pages/${page.id}`, {
        method: 'PATCH',
        headers: withSession({ 'Content-Type': 'application/json' }, sessionId),
        body: JSON.stringify({
          cover: 'https://example.com/cover.jpg',
          coverY: 0.3,
        }),
      });

      expect(res.status).toBe(200);
      const data = await res.json() as { page: { cover: string; coverY: number } };
      expect(data.page.cover).toBe('https://example.com/cover.jpg');
      expect(data.page.coverY).toBe(0.3);
    });
  });

  describe('DELETE /api/v1/pages/:id', () => {
    it('should delete page', async () => {
      const { sessionId, workspace } = await setupTestContext(app);
      const page = await createPage(app, sessionId, workspace.id);

      const res = await app.request(`/api/v1/pages/${page.id}`, {
        method: 'DELETE',
        headers: withSession({}, sessionId),
      });

      expect(res.status).toBe(200);

      // Verify deletion
      const getRes = await app.request(`/api/v1/pages/${page.id}`, {
        headers: withSession({}, sessionId),
      });
      expect(getRes.status).toBe(404);
    });

    it('should cascade delete child pages', async () => {
      const { sessionId, workspace } = await setupTestContext(app);
      const parentPage = await createPage(app, sessionId, workspace.id, { title: 'Parent' });
      const childPage = await createPage(app, sessionId, workspace.id, {
        title: 'Child',
        parentId: parentPage.id,
        parentType: 'page'
      });

      await app.request(`/api/v1/pages/${parentPage.id}`, {
        method: 'DELETE',
        headers: withSession({}, sessionId),
      });

      // Verify child is also deleted
      const childRes = await app.request(`/api/v1/pages/${childPage.id}`, {
        headers: withSession({}, sessionId),
      });
      expect(childRes.status).toBe(404);
    });
  });

  describe('POST /api/v1/pages/:id/archive', () => {
    it('should archive page', async () => {
      const { sessionId, workspace } = await setupTestContext(app);
      const page = await createPage(app, sessionId, workspace.id);

      const res = await app.request(`/api/v1/pages/${page.id}/archive`, {
        method: 'POST',
        headers: withSession({}, sessionId),
      });

      expect(res.status).toBe(200);
      const data = await res.json() as { page: { isArchived: boolean } };
      expect(data.page.isArchived).toBe(true);
    });

    it('should exclude archived pages from listing', async () => {
      const { sessionId, workspace } = await setupTestContext(app);
      const page = await createPage(app, sessionId, workspace.id);

      await app.request(`/api/v1/pages/${page.id}/archive`, {
        method: 'POST',
        headers: withSession({}, sessionId),
      });

      const res = await app.request(`/api/v1/workspaces/${workspace.id}/pages`, {
        headers: withSession({}, sessionId),
      });

      const data = await res.json() as { pages: { id: string }[] };
      expect(data.pages.find(p => p.id === page.id)).toBeUndefined();
    });
  });

  describe('POST /api/v1/pages/:id/restore', () => {
    it('should restore archived page', async () => {
      const { sessionId, workspace } = await setupTestContext(app);
      const page = await createPage(app, sessionId, workspace.id);

      await app.request(`/api/v1/pages/${page.id}/archive`, {
        method: 'POST',
        headers: withSession({}, sessionId),
      });

      const res = await app.request(`/api/v1/pages/${page.id}/restore`, {
        method: 'POST',
        headers: withSession({}, sessionId),
      });

      expect(res.status).toBe(200);
      const data = await res.json() as { page: { isArchived: boolean } };
      expect(data.page.isArchived).toBe(false);
    });
  });

  describe('POST /api/v1/pages/:id/duplicate', () => {
    it('should duplicate page', async () => {
      const { sessionId, workspace } = await setupTestContext(app);
      const page = await createPage(app, sessionId, workspace.id, { title: 'Original Page' });

      const res = await app.request(`/api/v1/pages/${page.id}/duplicate`, {
        method: 'POST',
        headers: withSession({}, sessionId),
      });

      expect(res.status).toBe(201);
      const data = await res.json() as { page: { id: string; title: string } };
      expect(data.page.id).not.toBe(page.id);
      expect(data.page.title).toContain('Original Page');
    });
  });

  describe('GET /api/v1/pages/:id/hierarchy', () => {
    it('should return page hierarchy', async () => {
      const { sessionId, workspace } = await setupTestContext(app);
      const page1 = await createPage(app, sessionId, workspace.id, { title: 'Level 1' });
      const page2 = await createPage(app, sessionId, workspace.id, {
        title: 'Level 2',
        parentId: page1.id,
        parentType: 'page'
      });
      const page3 = await createPage(app, sessionId, workspace.id, {
        title: 'Level 3',
        parentId: page2.id,
        parentType: 'page'
      });

      const res = await app.request(`/api/v1/pages/${page3.id}/hierarchy`, {
        headers: withSession({}, sessionId),
      });

      expect(res.status).toBe(200);
      const data = await res.json() as { hierarchy: { id: string; title: string }[] };
      expect(data.hierarchy.length).toBe(3);
      expect(data.hierarchy[0].title).toBe('Level 1');
      expect(data.hierarchy[1].title).toBe('Level 2');
      expect(data.hierarchy[2].title).toBe('Level 3');
    });
  });

  describe('Block Operations', () => {
    describe('GET /api/v1/pages/:id/blocks', () => {
      it('should return page blocks', async () => {
        const { sessionId, workspace } = await setupTestContext(app);
        const page = await createPage(app, sessionId, workspace.id);

        // Create some blocks
        await app.request('/api/v1/blocks', {
          method: 'POST',
          headers: withSession({ 'Content-Type': 'application/json' }, sessionId),
          body: JSON.stringify({
            pageId: page.id,
            type: 'paragraph',
            content: { richText: [{ type: 'text', text: { content: 'Hello World' } }] },
          }),
        });

        const res = await app.request(`/api/v1/pages/${page.id}/blocks`, {
          headers: withSession({}, sessionId),
        });

        expect(res.status).toBe(200);
        const data = await res.json() as { blocks: { type: string }[] };
        expect(data.blocks.length).toBe(1);
        expect(data.blocks[0].type).toBe('paragraph');
      });
    });

    describe('PUT /api/v1/pages/:id/blocks', () => {
      it('should batch update blocks', async () => {
        const { sessionId, workspace } = await setupTestContext(app);
        const page = await createPage(app, sessionId, workspace.id);

        const blocks = [
          { id: 'block-1', type: 'heading_1', content: { richText: [{ type: 'text', text: { content: 'Title' } }] }, position: 1 },
          { id: 'block-2', type: 'paragraph', content: { richText: [{ type: 'text', text: { content: 'Content' } }] }, position: 2 },
        ];

        const res = await app.request(`/api/v1/pages/${page.id}/blocks`, {
          method: 'PUT',
          headers: withSession({ 'Content-Type': 'application/json' }, sessionId),
          body: JSON.stringify({ blocks }),
        });

        expect(res.status).toBe(200);
        const data = await res.json() as { blocks: { id: string; type: string }[] };
        expect(data.blocks.length).toBe(2);
      });
    });
  });
});
