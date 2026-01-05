import { describe, it, expect, beforeEach } from 'vitest';
import { createTestApp, withSession, setupTestContext, createPage, createBlock, registerUser } from './setup';
import type { TestApp } from './setup';

describe('Comment E2E Tests', () => {
  let app: TestApp;

  beforeEach(() => {
    app = createTestApp();
  });

  describe('POST /api/v1/comments', () => {
    it('should create a page comment', async () => {
      const { sessionId, workspace, user } = await setupTestContext(app);
      const page = await createPage(app, sessionId, workspace.id);

      const res = await app.request('/api/v1/comments', {
        method: 'POST',
        headers: withSession({ 'Content-Type': 'application/json' }, sessionId),
        body: JSON.stringify({
          workspaceId: workspace.id,
          targetType: 'page',
          targetId: page.id,
          content: [{ type: 'text', text: { content: 'Great page!' } }],
        }),
      });

      expect(res.status).toBe(201);
      const data = await res.json() as { comment: { targetType: string; targetId: string; authorId: string } };
      expect(data.comment.targetType).toBe('page');
      expect(data.comment.targetId).toBe(page.id);
      expect(data.comment.authorId).toBe(user.id);
    });

    it('should create a block comment', async () => {
      const { sessionId, workspace } = await setupTestContext(app);
      const page = await createPage(app, sessionId, workspace.id);
      const block = await createBlock(app, sessionId, page.id);

      const res = await app.request('/api/v1/comments', {
        method: 'POST',
        headers: withSession({ 'Content-Type': 'application/json' }, sessionId),
        body: JSON.stringify({
          workspaceId: workspace.id,
          targetType: 'block',
          targetId: block.id,
          content: [{ type: 'text', text: { content: 'Comment on block' } }],
        }),
      });

      expect(res.status).toBe(201);
      const data = await res.json() as { comment: { targetType: string; targetId: string } };
      expect(data.comment.targetType).toBe('block');
      expect(data.comment.targetId).toBe(block.id);
    });

    it('should create a reply to comment', async () => {
      const { sessionId, workspace } = await setupTestContext(app);
      const page = await createPage(app, sessionId, workspace.id);

      // Create parent comment
      const parentRes = await app.request('/api/v1/comments', {
        method: 'POST',
        headers: withSession({ 'Content-Type': 'application/json' }, sessionId),
        body: JSON.stringify({
          workspaceId: workspace.id,
          targetType: 'page',
          targetId: page.id,
          content: [{ type: 'text', text: { content: 'Parent comment' } }],
        }),
      });
      const parentComment = (await parentRes.json() as { comment: { id: string } }).comment;

      // Create reply
      const res = await app.request('/api/v1/comments', {
        method: 'POST',
        headers: withSession({ 'Content-Type': 'application/json' }, sessionId),
        body: JSON.stringify({
          workspaceId: workspace.id,
          targetType: 'page',
          targetId: page.id,
          parentId: parentComment.id,
          content: [{ type: 'text', text: { content: 'Reply comment' } }],
        }),
      });

      expect(res.status).toBe(201);
      const data = await res.json() as { comment: { parentId: string } };
      expect(data.comment.parentId).toBe(parentComment.id);
    });
  });

  describe('GET /api/v1/pages/:id/comments', () => {
    it('should list page comments', async () => {
      const { sessionId, workspace } = await setupTestContext(app);
      const page = await createPage(app, sessionId, workspace.id);

      // Create comments
      await app.request('/api/v1/comments', {
        method: 'POST',
        headers: withSession({ 'Content-Type': 'application/json' }, sessionId),
        body: JSON.stringify({
          workspaceId: workspace.id,
          targetType: 'page',
          targetId: page.id,
          content: [{ type: 'text', text: { content: 'Comment 1' } }],
        }),
      });
      await app.request('/api/v1/comments', {
        method: 'POST',
        headers: withSession({ 'Content-Type': 'application/json' }, sessionId),
        body: JSON.stringify({
          workspaceId: workspace.id,
          targetType: 'page',
          targetId: page.id,
          content: [{ type: 'text', text: { content: 'Comment 2' } }],
        }),
      });

      const res = await app.request(`/api/v1/pages/${page.id}/comments`, {
        headers: withSession({}, sessionId),
      });

      expect(res.status).toBe(200);
      const data = await res.json() as { comments: any[] };
      expect(data.comments.length).toBe(2);
    });

    it('should return comments with author info', async () => {
      const { sessionId, workspace, user } = await setupTestContext(app);
      const page = await createPage(app, sessionId, workspace.id);

      await app.request('/api/v1/comments', {
        method: 'POST',
        headers: withSession({ 'Content-Type': 'application/json' }, sessionId),
        body: JSON.stringify({
          workspaceId: workspace.id,
          targetType: 'page',
          targetId: page.id,
          content: [{ type: 'text', text: { content: 'My comment' } }],
        }),
      });

      const res = await app.request(`/api/v1/pages/${page.id}/comments`, {
        headers: withSession({}, sessionId),
      });

      expect(res.status).toBe(200);
      const data = await res.json() as { comments: { authorId: string }[] };
      expect(data.comments[0].authorId).toBe(user.id);
    });
  });

  describe('PATCH /api/v1/comments/:id', () => {
    it('should update comment content', async () => {
      const { sessionId, workspace } = await setupTestContext(app);
      const page = await createPage(app, sessionId, workspace.id);

      const commentRes = await app.request('/api/v1/comments', {
        method: 'POST',
        headers: withSession({ 'Content-Type': 'application/json' }, sessionId),
        body: JSON.stringify({
          workspaceId: workspace.id,
          targetType: 'page',
          targetId: page.id,
          content: [{ type: 'text', text: { content: 'Original' } }],
        }),
      });
      const comment = (await commentRes.json() as { comment: { id: string } }).comment;

      const res = await app.request(`/api/v1/comments/${comment.id}`, {
        method: 'PATCH',
        headers: withSession({ 'Content-Type': 'application/json' }, sessionId),
        body: JSON.stringify({
          content: [{ type: 'text', text: { content: 'Updated' } }],
        }),
      });

      expect(res.status).toBe(200);
      const data = await res.json() as { comment: { content: { text: string }[] } };
      expect(data.comment.content[0].text).toBe('Updated');
    });

    it('should not allow non-author to update', async () => {
      const { sessionId: authorSession, workspace } = await setupTestContext(app);
      const page = await createPage(app, authorSession, workspace.id);

      const commentRes = await app.request('/api/v1/comments', {
        method: 'POST',
        headers: withSession({ 'Content-Type': 'application/json' }, authorSession),
        body: JSON.stringify({
          workspaceId: workspace.id,
          targetType: 'page',
          targetId: page.id,
          content: [{ type: 'text', text: { content: 'Original' } }],
        }),
      });
      const comment = (await commentRes.json() as { comment: { id: string } }).comment;

      // Add another user to workspace and try to update
      const { user: otherUser, sessionId: otherSession } = await registerUser(app);
      await app.request(`/api/v1/workspaces/${workspace.id}/members`, {
        method: 'POST',
        headers: withSession({ 'Content-Type': 'application/json' }, authorSession),
        body: JSON.stringify({ userId: otherUser.id, role: 'member' }),
      });

      const res = await app.request(`/api/v1/comments/${comment.id}`, {
        method: 'PATCH',
        headers: withSession({ 'Content-Type': 'application/json' }, otherSession),
        body: JSON.stringify({
          content: [{ type: 'text', text: { content: 'Hacked' } }],
        }),
      });

      expect(res.status).toBe(403);
    });
  });

  describe('DELETE /api/v1/comments/:id', () => {
    it('should delete comment by author', async () => {
      const { sessionId, workspace } = await setupTestContext(app);
      const page = await createPage(app, sessionId, workspace.id);

      const commentRes = await app.request('/api/v1/comments', {
        method: 'POST',
        headers: withSession({ 'Content-Type': 'application/json' }, sessionId),
        body: JSON.stringify({
          workspaceId: workspace.id,
          targetType: 'page',
          targetId: page.id,
          content: [{ type: 'text', text: { content: 'To delete' } }],
        }),
      });
      const comment = (await commentRes.json() as { comment: { id: string } }).comment;

      const res = await app.request(`/api/v1/comments/${comment.id}`, {
        method: 'DELETE',
        headers: withSession({}, sessionId),
      });

      expect(res.status).toBe(200);
    });

    it('should not allow non-author to delete', async () => {
      const { sessionId: authorSession, workspace } = await setupTestContext(app);
      const page = await createPage(app, authorSession, workspace.id);

      const commentRes = await app.request('/api/v1/comments', {
        method: 'POST',
        headers: withSession({ 'Content-Type': 'application/json' }, authorSession),
        body: JSON.stringify({
          workspaceId: workspace.id,
          targetType: 'page',
          targetId: page.id,
          content: [{ type: 'text', text: { content: 'Protected' } }],
        }),
      });
      const comment = (await commentRes.json() as { comment: { id: string } }).comment;

      const { user: otherUser, sessionId: otherSession } = await registerUser(app);
      await app.request(`/api/v1/workspaces/${workspace.id}/members`, {
        method: 'POST',
        headers: withSession({ 'Content-Type': 'application/json' }, authorSession),
        body: JSON.stringify({ userId: otherUser.id, role: 'member' }),
      });

      const res = await app.request(`/api/v1/comments/${comment.id}`, {
        method: 'DELETE',
        headers: withSession({}, otherSession),
      });

      expect(res.status).toBe(403);
    });
  });

  describe('POST /api/v1/comments/:id/resolve', () => {
    it('should resolve comment thread', async () => {
      const { sessionId, workspace } = await setupTestContext(app);
      const page = await createPage(app, sessionId, workspace.id);

      const commentRes = await app.request('/api/v1/comments', {
        method: 'POST',
        headers: withSession({ 'Content-Type': 'application/json' }, sessionId),
        body: JSON.stringify({
          workspaceId: workspace.id,
          targetType: 'page',
          targetId: page.id,
          content: [{ type: 'text', text: { content: 'Issue found' } }],
        }),
      });
      const comment = (await commentRes.json() as { comment: { id: string } }).comment;

      const res = await app.request(`/api/v1/comments/${comment.id}/resolve`, {
        method: 'POST',
        headers: withSession({}, sessionId),
      });

      expect(res.status).toBe(200);
      const data = await res.json() as { comment: { isResolved: boolean } };
      expect(data.comment.isResolved).toBe(true);
    });
  });
});
