import { serve } from '@hono/node-server';
import { serveStatic } from '@hono/node-server/serve-static';
import { Hono } from 'hono';
import { logger } from 'hono/logger';
import { secureHeaders } from 'hono/secure-headers';
import type { Env, Variables } from './env';
import { corsMiddleware } from './middleware/cors';
import { errorHandler } from './middleware/error';
import { createStore } from './store/factory';
import { createRoutes } from './routes';

async function main() {
  const app = new Hono<{ Bindings: Env; Variables: Variables }>();

  // Global middleware
  app.use('*', logger());
  app.use('*', secureHeaders());
  app.use('*', corsMiddleware);

  // Error handler
  app.onError(errorHandler);

  // Initialize SQLite store
  const dbPath = process.env.DATABASE_PATH ?? './workspace.db';
  const store = await createStore({
    driver: 'sqlite',
    sqlitePath: dbPath,
  });

  // Store middleware
  app.use('*', async (c, next) => {
    c.set('store', store);
    // Mock env for development
    c.env = {
      ENVIRONMENT: process.env.NODE_ENV ?? 'development',
      DEV_MODE: process.env.DEV_MODE ?? 'true',
    } as Env;
    await next();
  });

  // Static files
  app.use('/static/*', serveStatic({ root: './' }));

  // API and UI routes
  app.route('/', createRoutes());

  // 404 handler
  app.notFound((c) => {
    return c.json({ error: 'Not found' }, 404);
  });

  // Seed dev data if needed
  if (process.env.DEV_MODE === 'true' || process.env.NODE_ENV === 'development') {
    await seedDevData(store);
  }

  const port = parseInt(process.env.PORT ?? '3000', 10);

  console.log(`Starting server on http://localhost:${port}`);

  serve({
    fetch: app.fetch,
    port,
  });
}

async function seedDevData(store: Awaited<ReturnType<typeof createStore>>) {
  const { hashPassword } = await import('./utils/password');
  const { generateId } = await import('./utils/id');

  // Check if dev user exists
  const existingUser = await store.users.getById('dev-user-001');
  if (existingUser) {
    console.log('Dev data already exists');
    return;
  }

  console.log('Seeding dev data...');

  // Create dev user
  const user = await store.users.create({
    id: 'dev-user-001',
    email: 'dev@example.com',
    name: 'Developer',
    passwordHash: await hashPassword('dev123'),
    settings: {
      theme: 'system',
      timezone: 'UTC',
      dateFormat: 'MM/DD/YYYY',
      startOfWeek: 0,
      emailDigest: true,
      desktopNotify: true,
    },
  });

  // Create dev workspace
  const workspace = await store.workspaces.create({
    id: 'dev-workspace-001',
    name: 'Development',
    slug: 'dev',
    plan: 'free',
    settings: {
      allowPublicPages: false,
      allowGuestInvites: false,
      defaultPermission: 'read',
      allowedDomains: [],
      exportEnabled: true,
    },
    ownerId: user.id,
  });

  // Add user as member
  await store.members.create({
    id: generateId(),
    workspaceId: workspace.id,
    userId: user.id,
    role: 'owner',
  });

  // Create sample page
  const page = await store.pages.create({
    id: generateId(),
    workspaceId: workspace.id,
    parentType: 'workspace',
    title: 'Welcome',
    icon: '👋',
    properties: {},
    isTemplate: false,
    isArchived: false,
    createdBy: user.id,
  });

  // Create sample block
  await store.blocks.create({
    id: generateId(),
    pageId: page.id,
    type: 'paragraph',
    content: {
      richText: [
        {
          type: 'text',
          text: { content: 'Welcome to your workspace!' },
        },
      ],
    },
    position: 0,
  });

  // Create sample database
  const dbPage = await store.pages.create({
    id: generateId(),
    workspaceId: workspace.id,
    parentType: 'workspace',
    title: 'Tasks',
    icon: '📋',
    properties: {},
    isTemplate: false,
    isArchived: false,
    createdBy: user.id,
  });

  const database = await store.databases.create({
    id: generateId(),
    workspaceId: workspace.id,
    pageId: dbPage.id,
    title: 'Tasks',
    icon: '📋',
    isInline: false,
    properties: [
      { id: 'title', name: 'Name', type: 'title' },
      {
        id: 'status',
        name: 'Status',
        type: 'select',
        config: {
          options: [
            { id: 'todo', name: 'To Do', color: 'gray' },
            { id: 'in-progress', name: 'In Progress', color: 'blue' },
            { id: 'done', name: 'Done', color: 'green' },
          ],
        },
      },
      { id: 'priority', name: 'Priority', type: 'select', config: {
        options: [
          { id: 'low', name: 'Low', color: 'gray' },
          { id: 'medium', name: 'Medium', color: 'yellow' },
          { id: 'high', name: 'High', color: 'red' },
        ],
      }},
    ],
  });

  // Create default view
  await store.views.create({
    id: generateId(),
    databaseId: database.id,
    name: 'All Tasks',
    type: 'table',
    position: 0,
  });

  // Create sample rows
  const tasks = [
    { title: 'Set up project', status: 'done', priority: 'high' },
    { title: 'Write documentation', status: 'in-progress', priority: 'medium' },
    { title: 'Add tests', status: 'todo', priority: 'high' },
  ];

  for (let i = 0; i < tasks.length; i++) {
    await store.pages.create({
      id: generateId(),
      workspaceId: workspace.id,
      parentId: dbPage.id,
      parentType: 'database',
      databaseId: database.id,
      rowPosition: i,
      title: tasks[i].title,
      properties: {
        status: tasks[i].status,
        priority: tasks[i].priority,
      },
      isTemplate: false,
      isArchived: false,
      createdBy: user.id,
    });
  }

  console.log('Dev data seeded successfully');
  console.log('Login with: dev@example.com / dev123');
}

main().catch(console.error);
