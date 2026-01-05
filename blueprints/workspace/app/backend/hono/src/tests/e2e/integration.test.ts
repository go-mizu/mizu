import { describe, it, expect, beforeEach } from 'vitest';
import { createTestApp, withSession, registerUser, createWorkspace, createPage, createBlock, createDatabase, generateEmail, generateSlug } from './setup';
import type { TestApp } from './setup';

describe('Integration E2E Tests', () => {
  let app: TestApp;

  beforeEach(() => {
    app = createTestApp();
  });

  describe('Complete User Journey', () => {
    it('should complete new user onboarding flow', async () => {
      // 1. Register new user
      const { user, sessionId } = await registerUser(app, {
        email: generateEmail(),
        name: 'New User',
        password: 'securepassword123',
      });
      expect(user.id).toBeDefined();
      expect(sessionId).toBeDefined();

      // 2. Create first workspace
      const workspace = await createWorkspace(app, sessionId, {
        name: 'My First Workspace',
        slug: generateSlug(),
      });
      expect(workspace.id).toBeDefined();

      // 3. Create first page
      const page = await createPage(app, sessionId, workspace.id, {
        title: 'Welcome to my workspace',
      });
      expect(page.id).toBeDefined();

      // 4. Add blocks to page
      const block = await createBlock(app, sessionId, page.id, {
        type: 'paragraph',
        content: { richText: [{ type: 'text', text: { content: 'Hello, World!' } }] },
      });
      expect(block.id).toBeDefined();

      // 5. Verify content is saved
      const pageRes = await app.request(`/api/v1/pages/${page.id}/blocks`, {
        headers: withSession({}, sessionId),
      });
      expect(pageRes.status).toBe(200);
      const pageData = await pageRes.json() as { blocks: any[] };
      expect(pageData.blocks.length).toBe(1);
    });
  });

  describe('Project Management Setup', () => {
    it('should set up a complete project management system', async () => {
      const { sessionId, user } = await registerUser(app);
      const workspace = await createWorkspace(app, sessionId);

      // 1. Create database with properties
      const dbRes = await app.request('/api/v1/databases', {
        method: 'POST',
        headers: withSession({ 'Content-Type': 'application/json' }, sessionId),
        body: JSON.stringify({
          workspaceId: workspace.id,
          title: 'Project Tasks',
          properties: [
            { id: 'title', name: 'Task', type: 'title' },
            {
              id: 'status',
              name: 'Status',
              type: 'select',
              config: {
                options: [
                  { id: 'backlog', name: 'Backlog', color: 'gray' },
                  { id: 'todo', name: 'To Do', color: 'blue' },
                  { id: 'in_progress', name: 'In Progress', color: 'yellow' },
                  { id: 'done', name: 'Done', color: 'green' },
                ],
              },
            },
            { id: 'assignee', name: 'Assignee', type: 'person' },
            { id: 'due_date', name: 'Due Date', type: 'date' },
            {
              id: 'priority',
              name: 'Priority',
              type: 'select',
              config: {
                options: [
                  { id: 'low', name: 'Low', color: 'gray' },
                  { id: 'medium', name: 'Medium', color: 'yellow' },
                  { id: 'high', name: 'High', color: 'red' },
                ],
              },
            },
          ],
        }),
      });
      expect(dbRes.status).toBe(201);
      const { database } = await dbRes.json() as { database: { id: string } };

      // 2. Create table view
      const tableViewRes = await app.request(`/api/v1/databases/${database.id}/views`, {
        method: 'POST',
        headers: withSession({ 'Content-Type': 'application/json' }, sessionId),
        body: JSON.stringify({
          name: 'All Tasks',
          type: 'table',
        }),
      });
      expect(tableViewRes.status).toBe(201);

      // 3. Create board view grouped by status
      const boardViewRes = await app.request(`/api/v1/databases/${database.id}/views`, {
        method: 'POST',
        headers: withSession({ 'Content-Type': 'application/json' }, sessionId),
        body: JSON.stringify({
          name: 'Kanban Board',
          type: 'board',
          groupBy: 'status',
        }),
      });
      expect(boardViewRes.status).toBe(201);

      // 4. Add rows with properties
      for (let i = 0; i < 5; i++) {
        const rowRes = await app.request(`/api/v1/databases/${database.id}/rows`, {
          method: 'POST',
          headers: withSession({ 'Content-Type': 'application/json' }, sessionId),
          body: JSON.stringify({
            properties: {
              title: `Task ${i + 1}`,
              status: ['backlog', 'todo', 'in_progress', 'done', 'todo'][i],
              priority: ['low', 'medium', 'high', 'medium', 'low'][i],
            },
          }),
        });
        expect(rowRes.status).toBe(201);
      }

      // 5. Query and verify
      const rowsRes = await app.request(`/api/v1/databases/${database.id}/rows`, {
        headers: withSession({}, sessionId),
      });
      expect(rowsRes.status).toBe(200);
      const rowsData = await rowsRes.json() as { items: any[] };
      expect(rowsData.items.length).toBe(5);

      // 6. Update a row status
      const firstRow = rowsData.items[0];
      const updateRes = await app.request(`/api/v1/rows/${firstRow.id}`, {
        method: 'PATCH',
        headers: withSession({ 'Content-Type': 'application/json' }, sessionId),
        body: JSON.stringify({
          properties: { status: 'done' },
        }),
      });
      expect(updateRes.status).toBe(200);
    });
  });

  describe('Team Collaboration Workflow', () => {
    it('should support team collaboration', async () => {
      // 1. User A creates workspace
      const { user: userA, sessionId: sessionA } = await registerUser(app, { name: 'User A' });
      const workspace = await createWorkspace(app, sessionA, { name: 'Team Workspace' });

      // 2. User A invites User B
      const { user: userB, sessionId: sessionB } = await registerUser(app, { name: 'User B' });
      const inviteRes = await app.request(`/api/v1/workspaces/${workspace.id}/members`, {
        method: 'POST',
        headers: withSession({ 'Content-Type': 'application/json' }, sessionA),
        body: JSON.stringify({
          userId: userB.id,
          role: 'member',
        }),
      });
      expect(inviteRes.status).toBe(201);

      // 3. User B creates a page
      const pageRes = await app.request('/api/v1/pages', {
        method: 'POST',
        headers: withSession({ 'Content-Type': 'application/json' }, sessionB),
        body: JSON.stringify({
          workspaceId: workspace.id,
          title: 'User B\'s Page',
        }),
      });
      expect(pageRes.status).toBe(201);
      const { page } = await pageRes.json() as { page: { id: string } };

      // 4. User A comments on the page
      const commentRes = await app.request('/api/v1/comments', {
        method: 'POST',
        headers: withSession({ 'Content-Type': 'application/json' }, sessionA),
        body: JSON.stringify({
          workspaceId: workspace.id,
          targetType: 'page',
          targetId: page.id,
          content: [{ type: 'text', text: { content: 'Great work!' } }],
        }),
      });
      expect(commentRes.status).toBe(201);
      const { comment: parentComment } = await commentRes.json() as { comment: { id: string } };

      // 5. User B replies to the comment
      const replyRes = await app.request('/api/v1/comments', {
        method: 'POST',
        headers: withSession({ 'Content-Type': 'application/json' }, sessionB),
        body: JSON.stringify({
          workspaceId: workspace.id,
          targetType: 'page',
          targetId: page.id,
          parentId: parentComment.id,
          content: [{ type: 'text', text: { content: 'Thanks!' } }],
        }),
      });
      expect(replyRes.status).toBe(201);

      // 6. User A resolves the comment thread
      const resolveRes = await app.request(`/api/v1/comments/${parentComment.id}/resolve`, {
        method: 'POST',
        headers: withSession({}, sessionA),
      });
      expect(resolveRes.status).toBe(200);
      const resolveData = await resolveRes.json() as { comment: { isResolved: boolean } };
      expect(resolveData.comment.isResolved).toBe(true);
    });
  });

  describe('Public Page Sharing Workflow', () => {
    it('should support public page sharing', async () => {
      const { sessionId, workspace } = await registerUser(app).then(async ({ sessionId, user }) => {
        const ws = await createWorkspace(app, sessionId);
        return { sessionId, workspace: ws, user };
      });

      // 1. Create page with content
      const page = await createPage(app, sessionId, workspace.id, { title: 'Public Page' });
      await createBlock(app, sessionId, page.id, {
        type: 'paragraph',
        content: { richText: [{ type: 'text', text: { content: 'This is public content.' } }] },
      });

      // 2. Create public share link
      const shareRes = await app.request(`/api/v1/pages/${page.id}/shares`, {
        method: 'POST',
        headers: withSession({ 'Content-Type': 'application/json' }, sessionId),
        body: JSON.stringify({
          type: 'link',
          permission: 'read',
        }),
      });
      expect(shareRes.status).toBe(201);
      const { share } = await shareRes.json() as { share: { token: string } };

      // 3. Access page via share link (no auth)
      const validateRes = await app.request(`/api/v1/shares/validate/${share.token}`);
      expect(validateRes.status).toBe(200);
      const validateData = await validateRes.json() as { page: { title: string }; permission: string };
      expect(validateData.page.title).toBe('Public Page');
      expect(validateData.permission).toBe('read');

      // 4. Revoke share
      const revokeRes = await app.request(`/api/v1/shares/${share.id}`, {
        method: 'DELETE',
        headers: withSession({}, sessionId),
      });
      expect(revokeRes.status).toBe(200);

      // 5. Verify link no longer works
      const invalidRes = await app.request(`/api/v1/shares/validate/${share.token}`);
      expect(invalidRes.status).toBe(404);
    });
  });

  describe('Nested Page Structure', () => {
    it('should handle deep page nesting', async () => {
      const { sessionId } = await registerUser(app);
      const workspace = await createWorkspace(app, sessionId);

      // Create nested structure: Level 1 > Level 2 > Level 3 > Level 4
      const level1 = await createPage(app, sessionId, workspace.id, { title: 'Level 1' });
      const level2 = await createPage(app, sessionId, workspace.id, {
        title: 'Level 2',
        parentId: level1.id,
        parentType: 'page',
      });
      const level3 = await createPage(app, sessionId, workspace.id, {
        title: 'Level 3',
        parentId: level2.id,
        parentType: 'page',
      });
      const level4 = await createPage(app, sessionId, workspace.id, {
        title: 'Level 4',
        parentId: level3.id,
        parentType: 'page',
      });

      // Get hierarchy
      const hierRes = await app.request(`/api/v1/pages/${level4.id}/hierarchy`, {
        headers: withSession({}, sessionId),
      });
      expect(hierRes.status).toBe(200);
      const hierData = await hierRes.json() as { hierarchy: { title: string }[] };
      expect(hierData.hierarchy.length).toBe(4);
      expect(hierData.hierarchy.map((p: any) => p.title)).toEqual(['Level 1', 'Level 2', 'Level 3', 'Level 4']);
    });
  });

  describe('Block Editor Operations', () => {
    it('should handle complex block operations', async () => {
      const { sessionId } = await registerUser(app);
      const workspace = await createWorkspace(app, sessionId);
      const page = await createPage(app, sessionId, workspace.id);

      // 1. Create various block types
      const heading = await createBlock(app, sessionId, page.id, {
        type: 'heading_1',
        content: { richText: [{ type: 'text', text: { content: 'Document Title' } }] },
      });

      const paragraph = await createBlock(app, sessionId, page.id, {
        type: 'paragraph',
        content: { richText: [{ type: 'text', text: { content: 'Introduction text.' } }] },
      });

      const toggle = await createBlock(app, sessionId, page.id, {
        type: 'toggle',
        content: { richText: [{ type: 'text', text: { content: 'Click to expand' } }] },
      });

      // Nested block inside toggle
      await createBlock(app, sessionId, page.id, {
        type: 'paragraph',
        content: { richText: [{ type: 'text', text: { content: 'Hidden content' } }] },
        parentId: toggle.id,
      });

      const todo = await createBlock(app, sessionId, page.id, {
        type: 'to_do',
        content: { richText: [{ type: 'text', text: { content: 'Task item' } }], checked: false },
      });

      const code = await createBlock(app, sessionId, page.id, {
        type: 'code',
        content: { richText: [{ type: 'text', text: { content: 'console.log("Hello");' } }], language: 'javascript' },
      });

      // 2. Get all blocks (returns tree structure)
      const blocksRes = await app.request(`/api/v1/pages/${page.id}/blocks`, {
        headers: withSession({}, sessionId),
      });
      expect(blocksRes.status).toBe(200);
      const blocksData = await blocksRes.json() as { blocks: any[] };
      expect(blocksData.blocks.length).toBe(5); // 5 root blocks
      // Nested block is inside toggle.children
      const toggleBlock = blocksData.blocks.find((b: any) => b.type === 'toggle');
      expect(toggleBlock?.children?.length).toBe(1);

      // 3. Move paragraph before heading
      const moveRes = await app.request(`/api/v1/blocks/${paragraph.id}/move`, {
        method: 'POST',
        headers: withSession({ 'Content-Type': 'application/json' }, sessionId),
        body: JSON.stringify({
          position: 0.5, // Before heading
        }),
      });
      expect(moveRes.status).toBe(200);

      // 4. Update todo as checked
      const checkRes = await app.request(`/api/v1/blocks/${todo.id}`, {
        method: 'PATCH',
        headers: withSession({ 'Content-Type': 'application/json' }, sessionId),
        body: JSON.stringify({
          content: { richText: [{ type: 'text', text: { content: 'Task item' } }], checked: true },
        }),
      });
      expect(checkRes.status).toBe(200);
      const checkData = await checkRes.json() as { block: { content: { checked: boolean } } };
      expect(checkData.block.content.checked).toBe(true);

      // 5. Delete code block
      const deleteRes = await app.request(`/api/v1/blocks/${code.id}`, {
        method: 'DELETE',
        headers: withSession({}, sessionId),
      });
      expect(deleteRes.status).toBe(200);

      // Verify final state
      const finalRes = await app.request(`/api/v1/pages/${page.id}/blocks`, {
        headers: withSession({}, sessionId),
      });
      const finalData = await finalRes.json() as { blocks: any[] };
      expect(finalData.blocks.length).toBe(4); // Was 5 root blocks, one deleted
    });
  });
});
