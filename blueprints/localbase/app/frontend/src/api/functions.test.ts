import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { functionsApi } from './functions';

// Mock the api client
const mockApi = {
  get: vi.fn(),
  post: vi.fn(),
  put: vi.fn(),
  delete: vi.fn(),
};

vi.mock('./client', () => ({
  api: {
    get: (...args: any[]) => mockApi.get(...args),
    post: (...args: any[]) => mockApi.post(...args),
    put: (...args: any[]) => mockApi.put(...args),
    delete: (...args: any[]) => mockApi.delete(...args),
  },
}));

// Mock fetch for direct fetch calls
const mockFetch = vi.fn();
global.fetch = mockFetch;

describe('functionsApi', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  describe('Function CRUD operations', () => {
    describe('listFunctions', () => {
      it('fetches all functions', async () => {
        const mockFunctions = [
          { id: 'func-1', name: 'test-function', slug: 'test-function', status: 'active' },
          { id: 'func-2', name: 'another-function', slug: 'another-function', status: 'inactive' },
        ];
        mockApi.get.mockResolvedValue(mockFunctions);

        const result = await functionsApi.listFunctions();

        expect(mockApi.get).toHaveBeenCalledWith('/api/functions');
        expect(result).toEqual(mockFunctions);
      });
    });

    describe('getFunction', () => {
      it('fetches a single function by ID', async () => {
        const mockFunction = { id: 'func-1', name: 'test-function', slug: 'test-function' };
        mockApi.get.mockResolvedValue(mockFunction);

        const result = await functionsApi.getFunction('func-1');

        expect(mockApi.get).toHaveBeenCalledWith('/api/functions/func-1');
        expect(result).toEqual(mockFunction);
      });
    });

    describe('createFunction', () => {
      it('creates a new function', async () => {
        const newFunction = {
          id: 'func-new',
          name: 'new-function',
          slug: 'new-function',
          status: 'active',
        };
        mockApi.post.mockResolvedValue(newFunction);

        const result = await functionsApi.createFunction({
          name: 'new-function',
          verify_jwt: true,
        });

        expect(mockApi.post).toHaveBeenCalledWith('/api/functions', {
          name: 'new-function',
          verify_jwt: true,
        });
        expect(result).toEqual(newFunction);
      });

      it('creates function with template', async () => {
        const newFunction = { id: 'func-new', name: 'from-template' };
        mockApi.post.mockResolvedValue(newFunction);

        await functionsApi.createFunction({
          name: 'from-template',
          template_id: 'hello-world',
        });

        expect(mockApi.post).toHaveBeenCalledWith('/api/functions', {
          name: 'from-template',
          template_id: 'hello-world',
        });
      });

      it('creates function with custom slug', async () => {
        const newFunction = { id: 'func-new', name: 'my-function', slug: 'custom-slug' };
        mockApi.post.mockResolvedValue(newFunction);

        await functionsApi.createFunction({
          name: 'my-function',
          slug: 'custom-slug',
        });

        expect(mockApi.post).toHaveBeenCalledWith('/api/functions', {
          name: 'my-function',
          slug: 'custom-slug',
        });
      });
    });

    describe('updateFunction', () => {
      it('updates function properties', async () => {
        const updatedFunction = { id: 'func-1', name: 'updated-name', verify_jwt: false };
        mockApi.put.mockResolvedValue(updatedFunction);

        const result = await functionsApi.updateFunction('func-1', {
          name: 'updated-name',
          verify_jwt: false,
        });

        expect(mockApi.put).toHaveBeenCalledWith('/api/functions/func-1', {
          name: 'updated-name',
          verify_jwt: false,
        });
        expect(result).toEqual(updatedFunction);
      });

      it('updates function status', async () => {
        mockApi.put.mockResolvedValue({ id: 'func-1', status: 'inactive' });

        await functionsApi.updateFunction('func-1', { status: 'inactive' });

        expect(mockApi.put).toHaveBeenCalledWith('/api/functions/func-1', {
          status: 'inactive',
        });
      });
    });

    describe('deleteFunction', () => {
      it('deletes a function', async () => {
        mockApi.delete.mockResolvedValue(undefined);

        await functionsApi.deleteFunction('func-1');

        expect(mockApi.delete).toHaveBeenCalledWith('/api/functions/func-1');
      });
    });
  });

  describe('Source code operations', () => {
    describe('getSource', () => {
      it('fetches function source code', async () => {
        const mockSource = {
          source_code: 'console.log("hello")',
          is_draft: false,
        };
        mockApi.get.mockResolvedValue(mockSource);

        const result = await functionsApi.getSource('func-1');

        expect(mockApi.get).toHaveBeenCalledWith('/api/functions/func-1/source');
        expect(result).toEqual(mockSource);
      });

      it('returns draft state when present', async () => {
        const mockSource = {
          source_code: 'modified code',
          is_draft: true,
        };
        mockApi.get.mockResolvedValue(mockSource);

        const result = await functionsApi.getSource('func-1');

        expect(result.is_draft).toBe(true);
      });
    });

    describe('updateSource', () => {
      it('saves source code as draft', async () => {
        const saveResponse = { saved: true, is_draft: true };
        mockApi.put.mockResolvedValue(saveResponse);

        const result = await functionsApi.updateSource('func-1', {
          source_code: 'new source code',
        });

        expect(mockApi.put).toHaveBeenCalledWith('/api/functions/func-1/source', {
          source_code: 'new source code',
        });
        expect(result).toEqual(saveResponse);
      });

      it('saves source code with import map', async () => {
        mockApi.put.mockResolvedValue({ saved: true, is_draft: true });

        await functionsApi.updateSource('func-1', {
          source_code: 'import code',
          import_map: '{"imports": {}}',
        });

        expect(mockApi.put).toHaveBeenCalledWith('/api/functions/func-1/source', {
          source_code: 'import code',
          import_map: '{"imports": {}}',
        });
      });
    });
  });

  describe('Deployment operations', () => {
    describe('deployFunction', () => {
      it('deploys function with source code', async () => {
        const deployment = {
          id: 'dep-1',
          version: 2,
          status: 'deployed',
          deployed_at: '2025-01-15T10:00:00Z',
        };
        mockApi.post.mockResolvedValue(deployment);

        const result = await functionsApi.deployFunction('func-1', {
          source_code: 'production code',
        });

        expect(mockApi.post).toHaveBeenCalledWith('/api/functions/func-1/deploy', {
          source_code: 'production code',
        });
        expect(result).toEqual(deployment);
      });

      it('deploys function with import map', async () => {
        mockApi.post.mockResolvedValue({ id: 'dep-1', version: 2 });

        await functionsApi.deployFunction('func-1', {
          source_code: 'code with imports',
          import_map: '{"imports": {"lodash": "https://esm.sh/lodash"}}',
        });

        expect(mockApi.post).toHaveBeenCalledWith('/api/functions/func-1/deploy', {
          source_code: 'code with imports',
          import_map: '{"imports": {"lodash": "https://esm.sh/lodash"}}',
        });
      });
    });

    describe('listDeployments', () => {
      it('lists all deployments for a function', async () => {
        const mockDeployments = [
          { id: 'dep-2', version: 2, status: 'deployed' },
          { id: 'dep-1', version: 1, status: 'deployed' },
        ];
        mockApi.get.mockResolvedValue(mockDeployments);

        const result = await functionsApi.listDeployments('func-1');

        expect(mockApi.get).toHaveBeenCalledWith('/api/functions/func-1/deployments');
        expect(result).toEqual(mockDeployments);
      });
    });

    describe('downloadFunction', () => {
      it('downloads function source code', async () => {
        const sourceCode = 'function code content';
        mockFetch.mockResolvedValue({
          text: () => Promise.resolve(sourceCode),
        });

        const result = await functionsApi.downloadFunction('func-1');

        expect(mockFetch).toHaveBeenCalledWith('/api/functions/func-1/download', {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
          },
        });
        expect(result).toBe(sourceCode);
      });
    });
  });

  describe('Testing operations', () => {
    describe('testFunction', () => {
      it('tests function with POST request', async () => {
        const testResponse = {
          status: 200,
          body: { message: 'Hello World!' },
          headers: { 'content-type': 'application/json' },
          duration_ms: 45,
        };
        mockApi.post.mockResolvedValue(testResponse);

        const result = await functionsApi.testFunction('func-1', {
          method: 'POST',
          path: '/',
          headers: { 'Content-Type': 'application/json' },
          body: { name: 'World' },
        });

        expect(mockApi.post).toHaveBeenCalledWith('/api/functions/func-1/test', {
          method: 'POST',
          path: '/',
          headers: { 'Content-Type': 'application/json' },
          body: { name: 'World' },
        });
        expect(result).toEqual(testResponse);
      });

      it('tests function with GET request', async () => {
        mockApi.post.mockResolvedValue({ status: 200 });

        await functionsApi.testFunction('func-1', {
          method: 'GET',
          path: '/users',
        });

        expect(mockApi.post).toHaveBeenCalledWith('/api/functions/func-1/test', {
          method: 'GET',
          path: '/users',
        });
      });

      it('tests function with query params in path', async () => {
        mockApi.post.mockResolvedValue({ status: 200 });

        await functionsApi.testFunction('func-1', {
          method: 'GET',
          path: '/search?q=hello',
        });

        expect(mockApi.post).toHaveBeenCalledWith('/api/functions/func-1/test', {
          method: 'GET',
          path: '/search?q=hello',
        });
      });
    });
  });

  describe('Logs and metrics', () => {
    describe('getLogs', () => {
      it('fetches logs with default options', async () => {
        const mockLogs = {
          logs: [
            { id: 'log-1', level: 'info', message: 'Test log' },
          ],
        };
        mockApi.get.mockResolvedValue(mockLogs);

        const result = await functionsApi.getLogs('func-1');

        expect(mockApi.get).toHaveBeenCalledWith('/api/functions/func-1/logs');
        expect(result).toEqual(mockLogs);
      });

      it('fetches logs with limit', async () => {
        mockApi.get.mockResolvedValue({ logs: [] });

        await functionsApi.getLogs('func-1', { limit: 50 });

        expect(mockApi.get).toHaveBeenCalledWith('/api/functions/func-1/logs?limit=50');
      });

      it('fetches logs with level filter', async () => {
        mockApi.get.mockResolvedValue({ logs: [] });

        await functionsApi.getLogs('func-1', { level: 'error' });

        expect(mockApi.get).toHaveBeenCalledWith('/api/functions/func-1/logs?level=error');
      });

      it('fetches logs with since filter', async () => {
        mockApi.get.mockResolvedValue({ logs: [] });

        await functionsApi.getLogs('func-1', { since: '2025-01-15T00:00:00Z' });

        expect(mockApi.get).toHaveBeenCalledWith('/api/functions/func-1/logs?since=2025-01-15T00%3A00%3A00Z');
      });

      it('fetches logs with multiple options', async () => {
        mockApi.get.mockResolvedValue({ logs: [] });

        await functionsApi.getLogs('func-1', { limit: 100, level: 'info' });

        expect(mockApi.get).toHaveBeenCalledWith('/api/functions/func-1/logs?limit=100&level=info');
      });
    });

    describe('getMetrics', () => {
      it('fetches metrics with default period', async () => {
        const mockMetrics = {
          function_id: 'func-1',
          invocations: { total: 100, success: 95, error: 5 },
          latency: { avg: 50, p50: 45, p95: 100, p99: 200 },
        };
        mockApi.get.mockResolvedValue(mockMetrics);

        const result = await functionsApi.getMetrics('func-1');

        expect(mockApi.get).toHaveBeenCalledWith('/api/functions/func-1/metrics');
        expect(result).toEqual(mockMetrics);
      });

      it('fetches metrics with custom period', async () => {
        mockApi.get.mockResolvedValue({});

        await functionsApi.getMetrics('func-1', '7d');

        expect(mockApi.get).toHaveBeenCalledWith('/api/functions/func-1/metrics?period=7d');
      });

      it('supports all period options', async () => {
        mockApi.get.mockResolvedValue({});

        await functionsApi.getMetrics('func-1', '1h');
        expect(mockApi.get).toHaveBeenCalledWith('/api/functions/func-1/metrics?period=1h');

        await functionsApi.getMetrics('func-1', '24h');
        expect(mockApi.get).toHaveBeenCalledWith('/api/functions/func-1/metrics?period=24h');

        await functionsApi.getMetrics('func-1', '30d');
        expect(mockApi.get).toHaveBeenCalledWith('/api/functions/func-1/metrics?period=30d');
      });
    });
  });

  describe('Secret operations', () => {
    describe('listSecrets', () => {
      it('fetches all secrets', async () => {
        const mockSecrets = [
          { id: 'sec-1', name: 'API_KEY', created_at: '2025-01-10T10:00:00Z' },
          { id: 'sec-2', name: 'DB_URL', created_at: '2025-01-10T10:00:00Z' },
        ];
        mockApi.get.mockResolvedValue(mockSecrets);

        const result = await functionsApi.listSecrets();

        expect(mockApi.get).toHaveBeenCalledWith('/api/functions/secrets');
        expect(result).toEqual(mockSecrets);
      });
    });

    describe('createSecret', () => {
      it('creates a new secret', async () => {
        const newSecret = { id: 'sec-new', name: 'NEW_SECRET' };
        mockApi.post.mockResolvedValue(newSecret);

        const result = await functionsApi.createSecret({
          name: 'NEW_SECRET',
          value: 'secret-value-123',
        });

        expect(mockApi.post).toHaveBeenCalledWith('/api/functions/secrets', {
          name: 'NEW_SECRET',
          value: 'secret-value-123',
        });
        expect(result).toEqual(newSecret);
      });
    });

    describe('bulkUpdateSecrets', () => {
      it('bulk creates/updates secrets', async () => {
        const response = { created: 2, updated: 1, total: 3 };
        mockApi.put.mockResolvedValue(response);

        const result = await functionsApi.bulkUpdateSecrets({
          secrets: [
            { name: 'KEY1', value: 'value1' },
            { name: 'KEY2', value: 'value2' },
            { name: 'KEY3', value: 'value3' },
          ],
        });

        expect(mockApi.put).toHaveBeenCalledWith('/api/functions/secrets/bulk', {
          secrets: [
            { name: 'KEY1', value: 'value1' },
            { name: 'KEY2', value: 'value2' },
            { name: 'KEY3', value: 'value3' },
          ],
        });
        expect(result).toEqual(response);
      });
    });

    describe('deleteSecret', () => {
      it('deletes a secret by name', async () => {
        mockApi.delete.mockResolvedValue(undefined);

        await functionsApi.deleteSecret('API_KEY');

        expect(mockApi.delete).toHaveBeenCalledWith('/api/functions/secrets/API_KEY');
      });
    });
  });

  describe('Template operations', () => {
    describe('listTemplates', () => {
      it('fetches all templates', async () => {
        const mockTemplates = {
          templates: [
            { id: 'hello-world', name: 'Hello World', description: 'Basic handler' },
            { id: 'stripe', name: 'Stripe Webhook', description: 'Handle payments' },
          ],
        };
        mockApi.get.mockResolvedValue(mockTemplates);

        const result = await functionsApi.listTemplates();

        expect(mockApi.get).toHaveBeenCalledWith('/api/functions/templates');
        expect(result).toEqual(mockTemplates);
      });
    });

    describe('getTemplate', () => {
      it('fetches a specific template with source code', async () => {
        const template = {
          id: 'hello-world',
          source_code: 'export default function() { return "Hello" }',
        };
        mockApi.get.mockResolvedValue(template);

        const result = await functionsApi.getTemplate('hello-world');

        expect(mockApi.get).toHaveBeenCalledWith('/api/functions/templates/hello-world');
        expect(result).toEqual(template);
      });

      it('fetches template with import map', async () => {
        const template = {
          id: 'with-deps',
          source_code: 'import { x } from "lib"',
          import_map: '{"imports": {"lib": "https://esm.sh/lib"}}',
        };
        mockApi.get.mockResolvedValue(template);

        const result = await functionsApi.getTemplate('with-deps');

        expect(result.import_map).toBeDefined();
      });
    });
  });

  describe('Function invocation', () => {
    describe('invokeFunction', () => {
      it('invokes function with default POST method', async () => {
        const responseData = { result: 'success' };
        mockFetch.mockResolvedValue({
          ok: true,
          json: () => Promise.resolve(responseData),
        });

        const result = await functionsApi.invokeFunction('my-function');

        expect(mockFetch).toHaveBeenCalledWith('/functions/v1/my-function', {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
          },
          body: undefined,
        });
        expect(result).toEqual(responseData);
      });

      it('invokes function with custom method', async () => {
        mockFetch.mockResolvedValue({
          ok: true,
          json: () => Promise.resolve({}),
        });

        await functionsApi.invokeFunction('my-function', { method: 'GET' });

        expect(mockFetch).toHaveBeenCalledWith('/functions/v1/my-function', {
          method: 'GET',
          headers: {
            'Content-Type': 'application/json',
          },
          body: undefined,
        });
      });

      it('invokes function with body', async () => {
        mockFetch.mockResolvedValue({
          ok: true,
          json: () => Promise.resolve({}),
        });

        await functionsApi.invokeFunction('my-function', {
          body: { name: 'World' },
        });

        expect(mockFetch).toHaveBeenCalledWith('/functions/v1/my-function', {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
          },
          body: '{"name":"World"}',
        });
      });

      it('invokes function with custom headers', async () => {
        mockFetch.mockResolvedValue({
          ok: true,
          json: () => Promise.resolve({}),
        });

        await functionsApi.invokeFunction('my-function', {
          headers: {
            'Authorization': 'Bearer token',
            'X-Custom': 'value',
          },
        });

        expect(mockFetch).toHaveBeenCalledWith('/functions/v1/my-function', {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            'Authorization': 'Bearer token',
            'X-Custom': 'value',
          },
          body: undefined,
        });
      });

      it('throws error on failed invocation', async () => {
        mockFetch.mockResolvedValue({
          ok: false,
          json: () => Promise.resolve({ message: 'Function error' }),
        });

        await expect(functionsApi.invokeFunction('my-function')).rejects.toThrow('Function error');
      });

      it('handles invocation error without message', async () => {
        mockFetch.mockResolvedValue({
          ok: false,
          json: () => Promise.reject(new Error('Parse error')),
        });

        await expect(functionsApi.invokeFunction('my-function')).rejects.toThrow('Function invocation failed');
      });
    });
  });
});

describe('functionsApi edge cases', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('handles empty function list', async () => {
    mockApi.get.mockResolvedValue([]);

    const result = await functionsApi.listFunctions();

    expect(result).toEqual([]);
  });

  it('handles empty secrets list', async () => {
    mockApi.get.mockResolvedValue([]);

    const result = await functionsApi.listSecrets();

    expect(result).toEqual([]);
  });

  it('handles empty templates list', async () => {
    mockApi.get.mockResolvedValue({ templates: [] });

    const result = await functionsApi.listTemplates();

    expect(result.templates).toEqual([]);
  });

  it('handles empty deployments list', async () => {
    mockApi.get.mockResolvedValue([]);

    const result = await functionsApi.listDeployments('func-1');

    expect(result).toEqual([]);
  });

  it('handles empty logs response', async () => {
    mockApi.get.mockResolvedValue({ logs: [] });

    const result = await functionsApi.getLogs('func-1');

    expect(result.logs).toEqual([]);
  });
});
