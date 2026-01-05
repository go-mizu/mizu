import { describe, it, expect, beforeEach } from 'vitest';
import { createTestApp, withSession, setupTestContext, createPage, createBlock } from './setup';
import type { TestApp } from './setup';

describe('Block E2E Tests', () => {
  let app: TestApp;

  beforeEach(() => {
    app = createTestApp();
  });

  describe('POST /api/v1/blocks', () => {
    it('should create a paragraph block', async () => {
      const { sessionId, workspace } = await setupTestContext(app);
      const page = await createPage(app, sessionId, workspace.id);

      const res = await app.request('/api/v1/blocks', {
        method: 'POST',
        headers: withSession({ 'Content-Type': 'application/json' }, sessionId),
        body: JSON.stringify({
          pageId: page.id,
          type: 'paragraph',
          content: { richText: [{ type: 'text', text: { content: 'Hello World' } }] },
        }),
      });

      expect(res.status).toBe(201);
      const data = await res.json() as { block: { type: string; pageId: string; content: object } };
      expect(data.block.type).toBe('paragraph');
      expect(data.block.pageId).toBe(page.id);
    });

    it('should create a heading block', async () => {
      const { sessionId, workspace } = await setupTestContext(app);
      const page = await createPage(app, sessionId, workspace.id);

      const res = await app.request('/api/v1/blocks', {
        method: 'POST',
        headers: withSession({ 'Content-Type': 'application/json' }, sessionId),
        body: JSON.stringify({
          pageId: page.id,
          type: 'heading_1',
          content: { richText: [{ type: 'text', text: { content: 'My Heading' } }] },
        }),
      });

      expect(res.status).toBe(201);
      const data = await res.json() as { block: { type: string } };
      expect(data.block.type).toBe('heading_1');
    });

    it('should create a to-do block with checked state', async () => {
      const { sessionId, workspace } = await setupTestContext(app);
      const page = await createPage(app, sessionId, workspace.id);

      const res = await app.request('/api/v1/blocks', {
        method: 'POST',
        headers: withSession({ 'Content-Type': 'application/json' }, sessionId),
        body: JSON.stringify({
          pageId: page.id,
          type: 'to_do',
          content: {
            richText: [{ type: 'text', text: { content: 'Task item' } }],
            checked: true,
          },
        }),
      });

      expect(res.status).toBe(201);
      const data = await res.json() as { block: { type: string; content: { checked: boolean } } };
      expect(data.block.type).toBe('to_do');
      expect(data.block.content.checked).toBe(true);
    });

    it('should create a code block with language', async () => {
      const { sessionId, workspace } = await setupTestContext(app);
      const page = await createPage(app, sessionId, workspace.id);

      const res = await app.request('/api/v1/blocks', {
        method: 'POST',
        headers: withSession({ 'Content-Type': 'application/json' }, sessionId),
        body: JSON.stringify({
          pageId: page.id,
          type: 'code',
          content: {
            richText: [{ type: 'text', text: { content: 'console.log("Hello")' } }],
            language: 'typescript',
          },
        }),
      });

      expect(res.status).toBe(201);
      const data = await res.json() as { block: { type: string; content: { language: string } } };
      expect(data.block.type).toBe('code');
      expect(data.block.content.language).toBe('typescript');
    });

    it('should create a callout block with icon', async () => {
      const { sessionId, workspace } = await setupTestContext(app);
      const page = await createPage(app, sessionId, workspace.id);

      const res = await app.request('/api/v1/blocks', {
        method: 'POST',
        headers: withSession({ 'Content-Type': 'application/json' }, sessionId),
        body: JSON.stringify({
          pageId: page.id,
          type: 'callout',
          content: {
            richText: [{ type: 'text', text: { content: 'Important note' } }],
            icon: '⚠️',
            color: 'yellow',
          },
        }),
      });

      expect(res.status).toBe(201);
      const data = await res.json() as { block: { type: string; content: { icon: string; color: string } } };
      expect(data.block.type).toBe('callout');
      expect(data.block.content.icon).toBe('⚠️');
      expect(data.block.content.color).toBe('yellow');
    });

    it('should create nested block', async () => {
      const { sessionId, workspace } = await setupTestContext(app);
      const page = await createPage(app, sessionId, workspace.id);
      const parentBlock = await createBlock(app, sessionId, page.id, { type: 'toggle' });

      const res = await app.request('/api/v1/blocks', {
        method: 'POST',
        headers: withSession({ 'Content-Type': 'application/json' }, sessionId),
        body: JSON.stringify({
          pageId: page.id,
          type: 'paragraph',
          content: { richText: [{ type: 'text', text: { content: 'Nested content' } }] },
          parentId: parentBlock.id,
        }),
      });

      expect(res.status).toBe(201);
      const data = await res.json() as { block: { parentId: string } };
      expect(data.block.parentId).toBe(parentBlock.id);
    });

    it('should auto-increment position', async () => {
      const { sessionId, workspace } = await setupTestContext(app);
      const page = await createPage(app, sessionId, workspace.id);

      const block1Res = await app.request('/api/v1/blocks', {
        method: 'POST',
        headers: withSession({ 'Content-Type': 'application/json' }, sessionId),
        body: JSON.stringify({
          pageId: page.id,
          type: 'paragraph',
          content: { richText: [] },
        }),
      });
      const block1 = (await block1Res.json() as { block: { position: number } }).block;

      const block2Res = await app.request('/api/v1/blocks', {
        method: 'POST',
        headers: withSession({ 'Content-Type': 'application/json' }, sessionId),
        body: JSON.stringify({
          pageId: page.id,
          type: 'paragraph',
          content: { richText: [] },
        }),
      });
      const block2 = (await block2Res.json() as { block: { position: number } }).block;

      expect(block2.position).toBeGreaterThan(block1.position);
    });
  });

  describe('GET /api/v1/blocks/:id', () => {
    it('should get block by id', async () => {
      const { sessionId, workspace } = await setupTestContext(app);
      const page = await createPage(app, sessionId, workspace.id);
      const block = await createBlock(app, sessionId, page.id);

      const res = await app.request(`/api/v1/blocks/${block.id}`, {
        headers: withSession({}, sessionId),
      });

      expect(res.status).toBe(200);
      const data = await res.json() as { block: { id: string } };
      expect(data.block.id).toBe(block.id);
    });

    it('should return 404 for non-existent block', async () => {
      const { sessionId } = await setupTestContext(app);

      const res = await app.request('/api/v1/blocks/non-existent-id', {
        headers: withSession({}, sessionId),
      });

      expect(res.status).toBe(404);
    });
  });

  describe('PATCH /api/v1/blocks/:id', () => {
    it('should update block content', async () => {
      const { sessionId, workspace } = await setupTestContext(app);
      const page = await createPage(app, sessionId, workspace.id);
      const block = await createBlock(app, sessionId, page.id);

      const res = await app.request(`/api/v1/blocks/${block.id}`, {
        method: 'PATCH',
        headers: withSession({ 'Content-Type': 'application/json' }, sessionId),
        body: JSON.stringify({
          content: { richText: [{ type: 'text', text: { content: 'Updated content' } }] },
        }),
      });

      expect(res.status).toBe(200);
      const data = await res.json() as { block: { content: { richText: { text: { content: string } }[] } } };
      expect(data.block.content.richText[0].text.content).toBe('Updated content');
    });

    it('should update block type', async () => {
      const { sessionId, workspace } = await setupTestContext(app);
      const page = await createPage(app, sessionId, workspace.id);
      const block = await createBlock(app, sessionId, page.id, { type: 'paragraph' });

      const res = await app.request(`/api/v1/blocks/${block.id}`, {
        method: 'PATCH',
        headers: withSession({ 'Content-Type': 'application/json' }, sessionId),
        body: JSON.stringify({
          type: 'heading_1',
        }),
      });

      expect(res.status).toBe(200);
      const data = await res.json() as { block: { type: string } };
      expect(data.block.type).toBe('heading_1');
    });
  });

  describe('DELETE /api/v1/blocks/:id', () => {
    it('should delete block', async () => {
      const { sessionId, workspace } = await setupTestContext(app);
      const page = await createPage(app, sessionId, workspace.id);
      const block = await createBlock(app, sessionId, page.id);

      const res = await app.request(`/api/v1/blocks/${block.id}`, {
        method: 'DELETE',
        headers: withSession({}, sessionId),
      });

      expect(res.status).toBe(200);

      // Verify deletion
      const getRes = await app.request(`/api/v1/blocks/${block.id}`, {
        headers: withSession({}, sessionId),
      });
      expect(getRes.status).toBe(404);
    });

    it('should cascade delete child blocks', async () => {
      const { sessionId, workspace } = await setupTestContext(app);
      const page = await createPage(app, sessionId, workspace.id);
      const parentBlock = await createBlock(app, sessionId, page.id, { type: 'toggle' });
      const childBlock = await createBlock(app, sessionId, page.id, { parentId: parentBlock.id });

      await app.request(`/api/v1/blocks/${parentBlock.id}`, {
        method: 'DELETE',
        headers: withSession({}, sessionId),
      });

      // Verify child is also deleted
      const childRes = await app.request(`/api/v1/blocks/${childBlock.id}`, {
        headers: withSession({}, sessionId),
      });
      expect(childRes.status).toBe(404);
    });
  });

  describe('POST /api/v1/blocks/:id/move', () => {
    it('should move block to new position', async () => {
      const { sessionId, workspace } = await setupTestContext(app);
      const page = await createPage(app, sessionId, workspace.id);
      const block1 = await createBlock(app, sessionId, page.id);
      const block2 = await createBlock(app, sessionId, page.id);
      const block3 = await createBlock(app, sessionId, page.id);

      // Move block3 after block1
      const res = await app.request(`/api/v1/blocks/${block3.id}/move`, {
        method: 'POST',
        headers: withSession({ 'Content-Type': 'application/json' }, sessionId),
        body: JSON.stringify({
          afterId: block1.id,
        }),
      });

      expect(res.status).toBe(200);
    });

    it('should move block to different parent', async () => {
      const { sessionId, workspace } = await setupTestContext(app);
      const page = await createPage(app, sessionId, workspace.id);
      const block1 = await createBlock(app, sessionId, page.id, { type: 'toggle' });
      const block2 = await createBlock(app, sessionId, page.id, { type: 'toggle' });
      const childBlock = await createBlock(app, sessionId, page.id, { parentId: block1.id });

      // Move childBlock under block2
      const res = await app.request(`/api/v1/blocks/${childBlock.id}/move`, {
        method: 'POST',
        headers: withSession({ 'Content-Type': 'application/json' }, sessionId),
        body: JSON.stringify({
          parentId: block2.id,
        }),
      });

      expect(res.status).toBe(200);
      const data = await res.json() as { block: { parentId: string } };
      expect(data.block.parentId).toBe(block2.id);
    });
  });

  describe('Block Tree Structure', () => {
    it('should return blocks in correct order', async () => {
      const { sessionId, workspace } = await setupTestContext(app);
      const page = await createPage(app, sessionId, workspace.id);

      await createBlock(app, sessionId, page.id, { content: { richText: [{ type: 'text', text: { content: 'First' } }] } });
      await createBlock(app, sessionId, page.id, { content: { richText: [{ type: 'text', text: { content: 'Second' } }] } });
      await createBlock(app, sessionId, page.id, { content: { richText: [{ type: 'text', text: { content: 'Third' } }] } });

      const res = await app.request(`/api/v1/pages/${page.id}/blocks`, {
        headers: withSession({}, sessionId),
      });

      expect(res.status).toBe(200);
      const data = await res.json() as { blocks: { content: { richText: { text: { content: string } }[] } }[] };
      expect(data.blocks[0].content.richText[0].text.content).toBe('First');
      expect(data.blocks[1].content.richText[0].text.content).toBe('Second');
      expect(data.blocks[2].content.richText[0].text.content).toBe('Third');
    });

    it('should build hierarchical tree structure', async () => {
      const { sessionId, workspace } = await setupTestContext(app);
      const page = await createPage(app, sessionId, workspace.id);

      const parentBlock = await createBlock(app, sessionId, page.id, { type: 'toggle' });
      await createBlock(app, sessionId, page.id, { parentId: parentBlock.id });
      await createBlock(app, sessionId, page.id, { parentId: parentBlock.id });

      const res = await app.request(`/api/v1/pages/${page.id}/blocks`, {
        headers: withSession({}, sessionId),
      });

      expect(res.status).toBe(200);
      const data = await res.json() as { blocks: { id: string; children?: { id: string }[] }[] };

      // Tree structure: root blocks are at top level, children are nested
      expect(data.blocks.length).toBe(1);
      expect(data.blocks[0].id).toBe(parentBlock.id);
      expect(data.blocks[0].children?.length).toBe(2);
    });
  });
});
