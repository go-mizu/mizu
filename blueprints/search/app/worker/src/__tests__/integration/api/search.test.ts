/**
 * Integration tests for the search API endpoints.
 */
import { describe, it, expect, beforeEach, vi } from 'vitest';
import { createTestApp, parseJsonResponse } from '../../fixtures';
import type { SearchResponse } from '../../../types';

describe('GET /api/search', () => {
  let testApp: ReturnType<typeof createTestApp>;

  beforeEach(() => {
    testApp = createTestApp();
    vi.clearAllMocks();
  });

  it('returns 400 for missing query', async () => {
    const res = await testApp.request('/api/search');

    expect(res.status).toBe(400);
    const body = await parseJsonResponse<{ error: string }>(res);
    expect(body.error).toBe('Missing required parameter: q');
  });

  it('returns 400 for empty query', async () => {
    const res = await testApp.request('/api/search?q=');

    expect(res.status).toBe(400);
    const body = await parseJsonResponse<{ error: string }>(res);
    expect(body.error).toBe('Missing required parameter: q');
  });

  it('returns search results for valid query', async () => {
    const res = await testApp.request('/api/search?q=typescript');

    expect(res.status).toBe(200);
    const body = await parseJsonResponse<SearchResponse>(res);

    expect(body).toMatchObject({
      query: 'typescript',
      results: expect.any(Array),
      page: 1,
      per_page: 10,
    });
  });

  it('supports pagination', async () => {
    const res = await testApp.request('/api/search?q=test&page=2&per_page=20');

    expect(res.status).toBe(200);
    const body = await parseJsonResponse<SearchResponse>(res);

    expect(body.page).toBe(2);
    expect(body.per_page).toBe(20);
  });

  it('supports time range filter', async () => {
    const res = await testApp.request('/api/search?q=test&time=week');

    expect(res.status).toBe(200);
    const body = await parseJsonResponse<SearchResponse>(res);
    expect(body.query).toBe('test');
  });

  it('supports region filter', async () => {
    const res = await testApp.request('/api/search?q=test&region=us');

    expect(res.status).toBe(200);
    const body = await parseJsonResponse<SearchResponse>(res);
    expect(body.query).toBe('test');
  });

  it('supports language filter', async () => {
    const res = await testApp.request('/api/search?q=test&lang=en');

    expect(res.status).toBe(200);
    const body = await parseJsonResponse<SearchResponse>(res);
    expect(body.query).toBe('test');
  });

  it('supports safe search filter', async () => {
    const res = await testApp.request('/api/search?q=test&safe=strict');

    expect(res.status).toBe(200);
    const body = await parseJsonResponse<SearchResponse>(res);
    expect(body.query).toBe('test');
  });
});

describe('GET /api/search/images', () => {
  let testApp: ReturnType<typeof createTestApp>;

  beforeEach(() => {
    testApp = createTestApp();
    vi.clearAllMocks();
  });

  it('returns 400 for missing query', async () => {
    const res = await testApp.request('/api/search/images');

    expect(res.status).toBe(400);
    const body = await parseJsonResponse<{ error: string }>(res);
    expect(body.error).toBe('Missing required parameter: q');
  });

  it('returns image results for valid query', async () => {
    const res = await testApp.request('/api/search/images?q=cat');

    expect(res.status).toBe(200);
    const body = await parseJsonResponse<{ query: string; results: unknown[] }>(res);

    expect(body).toMatchObject({
      query: 'cat',
      results: expect.any(Array),
    });
  });

  it('supports image size filter', async () => {
    const res = await testApp.request('/api/search/images?q=cat&size=large');

    expect(res.status).toBe(200);
  });

  it('supports image color filter', async () => {
    const res = await testApp.request('/api/search/images?q=cat&color=red');

    expect(res.status).toBe(200);
  });

  it('supports image type filter', async () => {
    const res = await testApp.request('/api/search/images?q=cat&type=photo');

    expect(res.status).toBe(200);
  });

  it('supports image aspect filter', async () => {
    const res = await testApp.request('/api/search/images?q=cat&aspect=wide');

    expect(res.status).toBe(200);
  });
});

describe('GET /api/search/videos', () => {
  let testApp: ReturnType<typeof createTestApp>;

  beforeEach(() => {
    testApp = createTestApp();
    vi.clearAllMocks();
  });

  it('returns 400 for missing query', async () => {
    const res = await testApp.request('/api/search/videos');

    expect(res.status).toBe(400);
    const body = await parseJsonResponse<{ error: string }>(res);
    expect(body.error).toBe('Missing required parameter: q');
  });

  it('returns video results for valid query', async () => {
    const res = await testApp.request('/api/search/videos?q=tutorial');

    expect(res.status).toBe(200);
    const body = await parseJsonResponse<{ query: string; results: unknown[] }>(res);

    expect(body).toMatchObject({
      query: 'tutorial',
      results: expect.any(Array),
    });
  });

  it('supports video duration filter', async () => {
    const res = await testApp.request('/api/search/videos?q=tutorial&duration=short');

    expect(res.status).toBe(200);
  });
});

describe('GET /api/search/news', () => {
  let testApp: ReturnType<typeof createTestApp>;

  beforeEach(() => {
    testApp = createTestApp();
    vi.clearAllMocks();
  });

  it('returns 400 for missing query', async () => {
    const res = await testApp.request('/api/search/news');

    expect(res.status).toBe(400);
    const body = await parseJsonResponse<{ error: string }>(res);
    expect(body.error).toBe('Missing required parameter: q');
  });

  it('returns news results for valid query', async () => {
    const res = await testApp.request('/api/search/news?q=technology');

    expect(res.status).toBe(200);
    const body = await parseJsonResponse<{ query: string; results: unknown[] }>(res);

    expect(body).toMatchObject({
      query: 'technology',
      results: expect.any(Array),
    });
  });
});
