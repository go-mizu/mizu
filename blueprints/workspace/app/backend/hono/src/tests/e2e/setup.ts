import { createApp } from '../../app';
import type { StoreConfig } from '../../store/factory';
import type { Hono } from 'hono';
import type { Env, Variables } from '../../env';
import { tmpdir } from 'os';
import { join } from 'path';
import { randomUUID } from 'crypto';
import { clearRateLimitStore } from '../../middleware/ratelimit';

// Test driver type
export type TestDriver = 'sqlite' | 'postgres';

// Get test driver from environment variable
function getTestDriver(): TestDriver {
  const driver = process.env.TEST_DRIVER?.toLowerCase();
  if (driver === 'postgres') return 'postgres';
  return 'sqlite';
}

// Get current test driver (useful for test conditionals)
export function getCurrentTestDriver(): TestDriver {
  return getTestDriver();
}

// Check if PostgreSQL tests are available
export function isPostgresAvailable(): boolean {
  return !!process.env.POSTGRES_DSN;
}

export function createTestApp(driver?: TestDriver) {
  // Clear rate limit store to ensure test isolation
  clearRateLimitStore();

  const actualDriver = driver ?? getTestDriver();
  let storeConfig: StoreConfig;

  if (actualDriver === 'postgres') {
    const pgUrl = process.env.POSTGRES_DSN;
    if (!pgUrl) {
      throw new Error('POSTGRES_DSN environment variable required for postgres tests');
    }
    // Create unique schema for test isolation
    // Each test app gets its own schema that is dropped when the store closes
    const testSchema = `test_${randomUUID().replace(/-/g, '')}`;
    storeConfig = {
      driver: 'postgres',
      postgresUrl: pgUrl,
      postgresSchema: testSchema,
    };
  } else {
    // Use a unique temp file for each test app to ensure isolation
    const dbPath = join(tmpdir(), `test-db-${randomUUID()}.sqlite`);
    storeConfig = {
      driver: 'sqlite',
      sqlitePath: dbPath,
    };
  }

  return createApp({ storeConfig });
}

export type TestApp = ReturnType<typeof createTestApp>;

// Helper to extract session cookie from response
export function getSessionCookie(res: Response): string | null {
  const setCookie = res.headers.get('set-cookie');
  if (!setCookie) return null;
  const match = setCookie.match(/workspace_session=([^;]+)/);
  return match ? match[1] : null;
}

// Helper to make authenticated requests
export function withSession(headers: Record<string, string>, sessionId: string) {
  return {
    ...headers,
    Cookie: `workspace_session=${sessionId}`,
  };
}

// Test data generators
export function generateEmail() {
  return `test-${Date.now()}-${Math.random().toString(36).slice(2)}@example.com`;
}

export function generateSlug() {
  return `test-${Date.now()}-${Math.random().toString(36).slice(2)}`;
}

// Helper to register a user and get session
export async function registerUser(app: TestApp, overrides: { email?: string; name?: string; password?: string } = {}) {
  const email = overrides.email ?? generateEmail();
  const name = overrides.name ?? 'Test User';
  const password = overrides.password ?? 'password123';

  const res = await app.request('/api/v1/auth/register', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email, name, password }),
  });

  const data = await res.json() as { user: { id: string; email: string; name: string } };
  const sessionId = getSessionCookie(res);

  if (!sessionId) {
    throw new Error(`Registration failed: status=${res.status}, no session cookie`);
  }

  return { user: data.user, sessionId, email, password };
}

// Helper to create a workspace
export async function createWorkspace(app: TestApp, sessionId: string, overrides: { name?: string; slug?: string } = {}) {
  const name = overrides.name ?? 'Test Workspace';
  const slug = overrides.slug ?? generateSlug();

  const res = await app.request('/api/v1/workspaces', {
    method: 'POST',
    headers: withSession({ 'Content-Type': 'application/json' }, sessionId),
    body: JSON.stringify({ name, slug }),
  });

  const data = await res.json() as { workspace: { id: string; name: string; slug: string } };
  return data.workspace;
}

// Helper to create a page
export async function createPage(
  app: TestApp,
  sessionId: string,
  workspaceId: string,
  overrides: { title?: string; parentId?: string; parentType?: string } = {}
) {
  const res = await app.request('/api/v1/pages', {
    method: 'POST',
    headers: withSession({ 'Content-Type': 'application/json' }, sessionId),
    body: JSON.stringify({
      workspaceId,
      title: overrides.title ?? 'Test Page',
      parentId: overrides.parentId,
      parentType: overrides.parentType ?? 'workspace',
    }),
  });

  const data = await res.json() as { page: { id: string; title: string; workspaceId: string } };
  return data.page;
}

// Helper to create a block
export async function createBlock(
  app: TestApp,
  sessionId: string,
  pageId: string,
  overrides: { type?: string; content?: object; parentId?: string } = {}
) {
  const res = await app.request('/api/v1/blocks', {
    method: 'POST',
    headers: withSession({ 'Content-Type': 'application/json' }, sessionId),
    body: JSON.stringify({
      pageId,
      type: overrides.type ?? 'paragraph',
      content: overrides.content ?? { richText: [{ type: 'text', text: { content: 'Test content' } }] },
      parentId: overrides.parentId,
    }),
  });

  const data = await res.json() as { block: { id: string; type: string; pageId: string } };
  return data.block;
}

// Helper to create a database
export async function createDatabase(
  app: TestApp,
  sessionId: string,
  workspaceId: string,
  overrides: { title?: string; properties?: object[] } = {}
) {
  const res = await app.request('/api/v1/databases', {
    method: 'POST',
    headers: withSession({ 'Content-Type': 'application/json' }, sessionId),
    body: JSON.stringify({
      workspaceId,
      title: overrides.title ?? 'Test Database',
      properties: overrides.properties ?? [
        { id: 'title', name: 'Name', type: 'title' },
        { id: 'status', name: 'Status', type: 'select', config: { options: [{ id: 'todo', name: 'To Do', color: 'gray' }, { id: 'done', name: 'Done', color: 'green' }] } },
      ],
    }),
  });

  const data = await res.json() as { database: { id: string; title: string }; page: { id: string } };
  return data;
}

// Full test context with user, workspace, and session
export async function setupTestContext(app: TestApp) {
  const { user, sessionId } = await registerUser(app);
  const workspace = await createWorkspace(app, sessionId);
  return { user, sessionId, workspace, app };
}
