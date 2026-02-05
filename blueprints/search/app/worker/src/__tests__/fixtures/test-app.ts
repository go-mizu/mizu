/**
 * Test application factory for integration tests.
 * Creates a fully configured app instance with mock dependencies.
 */
import app from '../../index';
import type { Env } from '../../types';
import { createMockKV, type MockKVData } from './mock-kv';

export interface TestAppOptions {
  /** Initial KV data */
  kvData?: MockKVData;
  /** Environment override */
  environment?: Env['ENVIRONMENT'];
}

export interface TestApp {
  /** The Hono app instance */
  app: typeof app;
  /** Make a request to the app */
  request: (path: string, init?: RequestInit) => Promise<Response>;
  /** The mock KV namespace */
  kv: KVNamespace;
  /** The mock static content KV */
  staticKv: KVNamespace;
  /** The test environment */
  env: Env;
}

/**
 * Create a test application with mock dependencies.
 *
 * @example
 * ```typescript
 * const testApp = createTestApp();
 * const response = await testApp.request('/api/search?q=test');
 * expect(response.status).toBe(200);
 * ```
 */
export function createTestApp(options: TestAppOptions = {}): TestApp {
  const { kvData = {}, environment = 'development' } = options;

  const kv = createMockKV(kvData);
  const staticKv = createMockKV();

  const env: Env = {
    SEARCH_KV: kv,
    __STATIC_CONTENT: staticKv,
    ENVIRONMENT: environment,
  };

  const request = async (path: string, init?: RequestInit): Promise<Response> => {
    const url = new URL(path, 'http://localhost');
    return app.request(url.toString(), init, env);
  };

  return {
    app,
    request,
    kv,
    staticKv,
    env,
  };
}

/**
 * Create a test request with common defaults.
 */
export function createTestRequest(
  path: string,
  options: {
    method?: string;
    body?: unknown;
    headers?: Record<string, string>;
  } = {}
): Request {
  const { method = 'GET', body, headers = {} } = options;

  const url = new URL(path, 'http://localhost');

  const init: RequestInit = {
    method,
    headers: {
      'Content-Type': 'application/json',
      ...headers,
    },
  };

  if (body) {
    init.body = JSON.stringify(body);
  }

  return new Request(url.toString(), init);
}

/**
 * Helper to parse JSON response body.
 */
export async function parseJsonResponse<T>(response: Response): Promise<T> {
  const text = await response.text();
  try {
    return JSON.parse(text) as T;
  } catch {
    throw new Error(`Failed to parse JSON response: ${text}`);
  }
}
