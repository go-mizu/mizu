import { Hono } from 'hono';
import type { Env, Variables } from '../env';
import { optionalAuthMiddleware, authMiddleware } from '../middleware/auth';
import { createServices } from '../services';

export const uiRoutes = new Hono<{ Bindings: Env; Variables: Variables }>();

// HTML template helper
function html(body: string, title = 'Workspace'): string {
  return `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>${title}</title>
  <link rel="stylesheet" href="/static/dist/style.css">
</head>
<body>
  <div id="app">${body}</div>
  <script type="module" src="/static/dist/main.js"></script>
</body>
</html>`;
}

// Redirect root to login or app
uiRoutes.get('/', optionalAuthMiddleware, async (c) => {
  const user = c.get('user');
  if (user) {
    return c.redirect('/app');
  }
  return c.redirect('/login');
});

// Login page
uiRoutes.get('/login', (c) => {
  return c.html(html('<div id="login-page"></div>', 'Login'));
});

// Register page
uiRoutes.get('/register', (c) => {
  return c.html(html('<div id="register-page"></div>', 'Register'));
});

// App redirect (to first workspace)
uiRoutes.get('/app', authMiddleware, async (c) => {
  const userId = c.get('userId')!;
  const store = c.get('store');
  const services = createServices(store);

  const workspaces = await services.workspaces.listByUser(userId);
  if (workspaces.length === 0) {
    // No workspaces, redirect to create one
    return c.html(html('<div id="onboarding-page"></div>', 'Welcome'));
  }

  return c.redirect(`/w/${workspaces[0].slug}`);
});

// Workspace page
uiRoutes.get('/w/:workspace', authMiddleware, async (c) => {
  const slug = c.req.param('workspace');
  const userId = c.get('userId')!;
  const user = c.get('user')!;
  const store = c.get('store');
  const services = createServices(store);

  const workspace = await services.workspaces.getBySlug(slug, userId);
  if (!workspace) {
    return c.html(html('<div>Workspace not found</div>', 'Not Found'), 404);
  }

  const pages = await services.pages.listByWorkspace(workspace.id, { parentId: null });

  const data = {
    user: { id: user.id, name: user.name, email: user.email },
    workspace,
    pages,
  };

  return c.html(html(`
    <div id="workspace-page" data-props='${JSON.stringify(data)}'></div>
  `, workspace.name));
});

// Page view
uiRoutes.get('/w/:workspace/p/:pageId', authMiddleware, async (c) => {
  const slug = c.req.param('workspace');
  const pageId = c.req.param('pageId');
  const userId = c.get('userId')!;
  const user = c.get('user')!;
  const store = c.get('store');
  const services = createServices(store);

  const workspace = await services.workspaces.getBySlug(slug, userId);
  if (!workspace) {
    return c.html(html('<div>Workspace not found</div>', 'Not Found'), 404);
  }

  const page = await services.pages.getWithHierarchy(pageId);
  if (!page) {
    return c.html(html('<div>Page not found</div>', 'Not Found'), 404);
  }

  const blocks = await services.blocks.getBlockTree(pageId);

  const data = {
    user: { id: user.id, name: user.name, email: user.email },
    workspace,
    page,
    blocks,
  };

  return c.html(html(`
    <div id="page-view" data-props='${JSON.stringify(data)}'></div>
  `, page.title || 'Untitled'));
});

// Database view
uiRoutes.get('/w/:workspace/d/:databaseId', authMiddleware, async (c) => {
  const slug = c.req.param('workspace');
  const databaseId = c.req.param('databaseId');
  const userId = c.get('userId')!;
  const user = c.get('user')!;
  const store = c.get('store');
  const services = createServices(store);

  const workspace = await services.workspaces.getBySlug(slug, userId);
  if (!workspace) {
    return c.html(html('<div>Workspace not found</div>', 'Not Found'), 404);
  }

  const database = await services.databases.getById(databaseId);
  if (!database) {
    return c.html(html('<div>Database not found</div>', 'Not Found'), 404);
  }

  const views = await services.views.listByDatabase(databaseId);
  const rows = await services.rows.listByDatabase(databaseId, { limit: 50 });

  const data = {
    user: { id: user.id, name: user.name, email: user.email },
    workspace,
    database,
    views,
    rows: rows.items,
  };

  return c.html(html(`
    <div id="database-view" data-props='${JSON.stringify(data)}'></div>
  `, database.title || 'Untitled Database'));
});

// Search page
uiRoutes.get('/w/:workspace/search', authMiddleware, async (c) => {
  const slug = c.req.param('workspace');
  const userId = c.get('userId')!;
  const user = c.get('user')!;
  const store = c.get('store');
  const services = createServices(store);

  const workspace = await services.workspaces.getBySlug(slug, userId);
  if (!workspace) {
    return c.html(html('<div>Workspace not found</div>', 'Not Found'), 404);
  }

  const data = {
    user: { id: user.id, name: user.name, email: user.email },
    workspace,
  };

  return c.html(html(`
    <div id="search-page" data-props='${JSON.stringify(data)}'></div>
  `, 'Search'));
});

// Settings page
uiRoutes.get('/w/:workspace/settings', authMiddleware, async (c) => {
  const slug = c.req.param('workspace');
  const userId = c.get('userId')!;
  const user = c.get('user')!;
  const store = c.get('store');
  const services = createServices(store);

  const workspace = await services.workspaces.getBySlug(slug, userId);
  if (!workspace) {
    return c.html(html('<div>Workspace not found</div>', 'Not Found'), 404);
  }

  const members = await services.workspaces.getMembers(workspace.id, userId);

  const data = {
    user: { id: user.id, name: user.name, email: user.email },
    workspace,
    members,
  };

  return c.html(html(`
    <div id="settings-page" data-props='${JSON.stringify(data)}'></div>
  `, 'Settings'));
});
