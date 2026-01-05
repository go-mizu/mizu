import { Hono } from 'hono';
import { zValidator } from '@hono/zod-validator';
import type { Env, Variables } from '../env';
import { CreateWorkspaceSchema, UpdateWorkspaceSchema, AddMemberSchema } from '../models/workspace';
import { authMiddleware } from '../middleware/auth';
import { createServices } from '../services';

export const workspaceRoutes = new Hono<{ Bindings: Env; Variables: Variables }>();

// All workspace routes require auth
workspaceRoutes.use('/*', authMiddleware);

// List workspaces
workspaceRoutes.get('/', async (c) => {
  const userId = c.get('userId')!;
  const store = c.get('store');
  const services = createServices(store);

  const workspaces = await services.workspaces.listByUser(userId);
  return c.json({ workspaces });
});

// Create workspace
workspaceRoutes.post(
  '/',
  zValidator('json', CreateWorkspaceSchema),
  async (c) => {
    const input = c.req.valid('json');
    const userId = c.get('userId')!;
    const store = c.get('store');
    const services = createServices(store);

    const workspace = await services.workspaces.create(input, userId);
    return c.json({ workspace }, 201);
  }
);

// Get workspace
workspaceRoutes.get('/:id', async (c) => {
  const id = c.req.param('id');
  const userId = c.get('userId')!;
  const store = c.get('store');
  const services = createServices(store);

  const workspace = await services.workspaces.getById(id, userId);
  if (!workspace) {
    return c.json({ error: 'Workspace not found' }, 404);
  }

  return c.json({ workspace });
});

// Update workspace
workspaceRoutes.patch(
  '/:id',
  zValidator('json', UpdateWorkspaceSchema),
  async (c) => {
    const id = c.req.param('id');
    const input = c.req.valid('json');
    const userId = c.get('userId')!;
    const store = c.get('store');
    const services = createServices(store);

    const workspace = await services.workspaces.update(id, input, userId);
    return c.json({ workspace });
  }
);

// Delete workspace
workspaceRoutes.delete('/:id', async (c) => {
  const id = c.req.param('id');
  const userId = c.get('userId')!;
  const store = c.get('store');
  const services = createServices(store);

  await services.workspaces.delete(id, userId);
  return c.json({ success: true });
});

// List members
workspaceRoutes.get('/:id/members', async (c) => {
  const id = c.req.param('id');
  const userId = c.get('userId')!;
  const store = c.get('store');
  const services = createServices(store);

  const members = await services.workspaces.getMembers(id, userId);
  return c.json({ members });
});

// Add member
workspaceRoutes.post(
  '/:id/members',
  zValidator('json', AddMemberSchema),
  async (c) => {
    const id = c.req.param('id');
    const input = c.req.valid('json');
    const userId = c.get('userId')!;
    const store = c.get('store');
    const services = createServices(store);

    const member = await services.workspaces.addMember(id, input, userId);
    return c.json({ member }, 201);
  }
);

// List pages in workspace
workspaceRoutes.get('/:id/pages', async (c) => {
  const workspaceId = c.req.param('id');
  const userId = c.get('userId')!;
  const store = c.get('store');
  const services = createServices(store);

  // Check membership
  const member = await services.workspaces.checkMembership(workspaceId, userId);
  if (!member) {
    return c.json({ error: 'Permission denied' }, 403);
  }

  const parentId = c.req.query('parent_id');
  const pages = await services.pages.listByWorkspace(workspaceId, {
    parentId: parentId === 'null' ? null : parentId,
  });

  return c.json({ pages });
});

// List favorites in workspace
workspaceRoutes.get('/:id/favorites', async (c) => {
  const workspaceId = c.req.param('id');
  const userId = c.get('userId')!;
  const store = c.get('store');
  const services = createServices(store);

  const favorites = await services.favorites.listPagesWithFavorites(userId, workspaceId);
  return c.json({ favorites });
});
