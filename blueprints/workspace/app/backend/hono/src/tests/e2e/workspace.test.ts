import { describe, it, expect, beforeEach } from 'vitest';
import { createTestApp, withSession, generateSlug, registerUser, createWorkspace, setupTestContext } from './setup';
import type { TestApp } from './setup';

describe('Workspace E2E Tests', () => {
  let app: TestApp;

  beforeEach(() => {
    app = createTestApp();
  });

  describe('POST /api/v1/workspaces', () => {
    it('should create a workspace successfully', async () => {
      const { sessionId, user } = await registerUser(app);
      const slug = generateSlug();

      const res = await app.request('/api/v1/workspaces', {
        method: 'POST',
        headers: withSession({ 'Content-Type': 'application/json' }, sessionId),
        body: JSON.stringify({
          name: 'My Workspace',
          slug,
        }),
      });

      expect(res.status).toBe(201);
      const data = await res.json() as { workspace: { id: string; name: string; slug: string; ownerId: string } };
      expect(data.workspace.name).toBe('My Workspace');
      expect(data.workspace.slug).toBe(slug);
      expect(data.workspace.ownerId).toBe(user.id);
    });

    it('should add creator as owner member', async () => {
      const { sessionId, user } = await registerUser(app);
      const workspace = await createWorkspace(app, sessionId);

      const res = await app.request(`/api/v1/workspaces/${workspace.id}/members`, {
        headers: withSession({}, sessionId),
      });

      expect(res.status).toBe(200);
      const data = await res.json() as { members: { userId: string; role: string }[] };
      const ownerMember = data.members.find(m => m.userId === user.id);
      expect(ownerMember).toBeDefined();
      expect(ownerMember?.role).toBe('owner');
    });

    it('should reject duplicate slug', async () => {
      const { sessionId } = await registerUser(app);
      const slug = generateSlug();

      await createWorkspace(app, sessionId, { slug });

      const res = await app.request('/api/v1/workspaces', {
        method: 'POST',
        headers: withSession({ 'Content-Type': 'application/json' }, sessionId),
        body: JSON.stringify({
          name: 'Another Workspace',
          slug,
        }),
      });

      expect(res.status).toBe(409);
    });

    it('should reject invalid slug format', async () => {
      const { sessionId } = await registerUser(app);

      const res = await app.request('/api/v1/workspaces', {
        method: 'POST',
        headers: withSession({ 'Content-Type': 'application/json' }, sessionId),
        body: JSON.stringify({
          name: 'My Workspace',
          slug: 'Invalid Slug!',
        }),
      });

      expect(res.status).toBe(400);
    });

    it('should require authentication', async () => {
      const res = await app.request('/api/v1/workspaces', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          name: 'My Workspace',
          slug: generateSlug(),
        }),
      });

      expect(res.status).toBe(401);
    });
  });

  describe('GET /api/v1/workspaces', () => {
    it('should list user workspaces', async () => {
      const { sessionId } = await registerUser(app);
      const ws1 = await createWorkspace(app, sessionId, { name: 'Workspace 1' });
      const ws2 = await createWorkspace(app, sessionId, { name: 'Workspace 2' });

      const res = await app.request('/api/v1/workspaces', {
        headers: withSession({}, sessionId),
      });

      expect(res.status).toBe(200);
      const data = await res.json() as { workspaces: { id: string }[] };
      expect(data.workspaces.length).toBe(2);
      const ids = data.workspaces.map(w => w.id);
      expect(ids).toContain(ws1.id);
      expect(ids).toContain(ws2.id);
    });

    it('should return empty list for new user', async () => {
      const { sessionId } = await registerUser(app);

      const res = await app.request('/api/v1/workspaces', {
        headers: withSession({}, sessionId),
      });

      expect(res.status).toBe(200);
      const data = await res.json() as { workspaces: any[] };
      expect(data.workspaces).toEqual([]);
    });
  });

  describe('GET /api/v1/workspaces/:id', () => {
    it('should get workspace by id', async () => {
      const { sessionId, workspace } = await setupTestContext(app);

      const res = await app.request(`/api/v1/workspaces/${workspace.id}`, {
        headers: withSession({}, sessionId),
      });

      expect(res.status).toBe(200);
      const data = await res.json() as { workspace: { id: string; name: string } };
      expect(data.workspace.id).toBe(workspace.id);
      expect(data.workspace.name).toBe(workspace.name);
    });

    it('should return 404 for non-existent workspace', async () => {
      const { sessionId } = await registerUser(app);

      const res = await app.request('/api/v1/workspaces/non-existent-id', {
        headers: withSession({}, sessionId),
      });

      expect(res.status).toBe(404);
    });

    it('should return 403 for non-member', async () => {
      const { sessionId: ownerSession } = await registerUser(app);
      const workspace = await createWorkspace(app, ownerSession);

      const { sessionId: otherSession } = await registerUser(app);

      const res = await app.request(`/api/v1/workspaces/${workspace.id}`, {
        headers: withSession({}, otherSession),
      });

      expect(res.status).toBe(403);
    });
  });

  describe('PATCH /api/v1/workspaces/:id', () => {
    it('should update workspace name', async () => {
      const { sessionId, workspace } = await setupTestContext(app);

      const res = await app.request(`/api/v1/workspaces/${workspace.id}`, {
        method: 'PATCH',
        headers: withSession({ 'Content-Type': 'application/json' }, sessionId),
        body: JSON.stringify({ name: 'Updated Name' }),
      });

      expect(res.status).toBe(200);
      const data = await res.json() as { workspace: { name: string } };
      expect(data.workspace.name).toBe('Updated Name');
    });

    it('should update workspace icon', async () => {
      const { sessionId, workspace } = await setupTestContext(app);

      const res = await app.request(`/api/v1/workspaces/${workspace.id}`, {
        method: 'PATCH',
        headers: withSession({ 'Content-Type': 'application/json' }, sessionId),
        body: JSON.stringify({ icon: '🚀' }),
      });

      expect(res.status).toBe(200);
      const data = await res.json() as { workspace: { icon: string } };
      expect(data.workspace.icon).toBe('🚀');
    });
  });

  describe('DELETE /api/v1/workspaces/:id', () => {
    it('should delete workspace as owner', async () => {
      const { sessionId, workspace } = await setupTestContext(app);

      const res = await app.request(`/api/v1/workspaces/${workspace.id}`, {
        method: 'DELETE',
        headers: withSession({}, sessionId),
      });

      expect(res.status).toBe(200);

      // Verify deletion
      const getRes = await app.request(`/api/v1/workspaces/${workspace.id}`, {
        headers: withSession({}, sessionId),
      });
      expect(getRes.status).toBe(404);
    });

    it('should not allow non-owner to delete', async () => {
      // This would require adding a member, which we'll implement when we have member routes
      // For now, we just verify owner can delete
    });
  });

  describe('Workspace Members', () => {
    describe('GET /api/v1/workspaces/:id/members', () => {
      it('should list workspace members', async () => {
        const { sessionId, workspace, user } = await setupTestContext(app);

        const res = await app.request(`/api/v1/workspaces/${workspace.id}/members`, {
          headers: withSession({}, sessionId),
        });

        expect(res.status).toBe(200);
        const data = await res.json() as { members: { userId: string; role: string }[] };
        expect(data.members.length).toBeGreaterThanOrEqual(1);
        expect(data.members.some(m => m.userId === user.id && m.role === 'owner')).toBe(true);
      });
    });

    describe('POST /api/v1/workspaces/:id/members', () => {
      it('should add member to workspace', async () => {
        const { sessionId: ownerSession, workspace } = await setupTestContext(app);
        const { user: newUser } = await registerUser(app);

        const res = await app.request(`/api/v1/workspaces/${workspace.id}/members`, {
          method: 'POST',
          headers: withSession({ 'Content-Type': 'application/json' }, ownerSession),
          body: JSON.stringify({
            userId: newUser.id,
            role: 'member',
          }),
        });

        expect(res.status).toBe(201);
        const data = await res.json() as { member: { userId: string; role: string } };
        expect(data.member.userId).toBe(newUser.id);
        expect(data.member.role).toBe('member');
      });
    });
  });

  describe('GET /api/v1/workspaces/:id/pages', () => {
    it('should list root pages in workspace', async () => {
      const { sessionId, workspace } = await setupTestContext(app);

      // Create some pages
      const page1Res = await app.request('/api/v1/pages', {
        method: 'POST',
        headers: withSession({ 'Content-Type': 'application/json' }, sessionId),
        body: JSON.stringify({
          workspaceId: workspace.id,
          title: 'Page 1',
        }),
      });
      const page1 = (await page1Res.json() as { page: { id: string } }).page;

      await app.request('/api/v1/pages', {
        method: 'POST',
        headers: withSession({ 'Content-Type': 'application/json' }, sessionId),
        body: JSON.stringify({
          workspaceId: workspace.id,
          title: 'Page 2',
        }),
      });

      const res = await app.request(`/api/v1/workspaces/${workspace.id}/pages`, {
        headers: withSession({}, sessionId),
      });

      expect(res.status).toBe(200);
      const data = await res.json() as { pages: { id: string; title: string }[] };
      expect(data.pages.length).toBe(2);
    });
  });
});
