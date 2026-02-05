/**
 * Integration tests for the health check endpoint.
 */
import { describe, it, expect, beforeEach, vi } from 'vitest';
import { createTestApp, parseJsonResponse } from '../../fixtures';

describe('GET /health', () => {
  let testApp: ReturnType<typeof createTestApp>;

  beforeEach(() => {
    testApp = createTestApp();
    vi.clearAllMocks();
  });

  it('returns 200 OK for health check', async () => {
    const res = await testApp.request('/health');

    expect(res.status).toBe(200);
    const body = await parseJsonResponse<{ status: string }>(res);
    expect(body.status).toBe('ok');
  });

  it('returns correct content type', async () => {
    const res = await testApp.request('/health');

    expect(res.headers.get('content-type')).toContain('application/json');
  });
});
