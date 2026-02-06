import { describe, it, expect, vi, beforeEach } from 'vitest';
import { JinaSearchEngine, JinaReaderEngine } from './jina';
import type { EngineParams } from './engine';

// ========== JinaSearchEngine Tests ==========

describe('JinaSearchEngine', () => {
  const engine = new JinaSearchEngine();

  const defaultParams: EngineParams = {
    page: 1,
    locale: 'en',
    safeSearch: 1,
    timeRange: '',
    engineData: {},
  };

  it('should have correct metadata', () => {
    expect(engine.name).toBe('jina');
    expect(engine.shortcut).toBe('ji');
    expect(engine.categories).toContain('general');
    expect(engine.weight).toBe(1.5);
    expect(engine.timeout).toBe(15_000);
    expect(engine.supportsPaging).toBe(false);
    expect(engine.maxPage).toBe(1);
  });

  it('should build correct search URL', () => {
    const config = engine.buildRequest('typescript tutorial', defaultParams);
    expect(config.url).toBe('https://s.jina.ai/typescript%20tutorial');
    expect(config.method).toBe('GET');
    expect(config.headers['Accept']).toBe('application/json');
  });

  it('should include auth header when API key is provided', () => {
    const params: EngineParams = {
      ...defaultParams,
      engineData: { jina_api_key: 'test-key-123' },
    };
    const config = engine.buildRequest('test', params);
    expect(config.headers['Authorization']).toBe('Bearer test-key-123');
  });

  it('should not include auth header when API key is empty', () => {
    const config = engine.buildRequest('test', defaultParams);
    expect(config.headers['Authorization']).toBeUndefined();
  });

  it('should URL-encode the query', () => {
    const config = engine.buildRequest('hello world & more', defaultParams);
    expect(config.url).toBe('https://s.jina.ai/hello%20world%20%26%20more');
  });

  it('should parse valid JSON response', () => {
    const body = JSON.stringify({
      code: 200,
      data: [
        {
          title: 'TypeScript Tutorial',
          url: 'https://example.com/ts',
          content: 'Learn TypeScript from scratch',
          description: 'A beginner guide',
        },
        {
          title: 'Advanced TypeScript',
          url: 'https://example.com/advanced',
          content: 'Advanced patterns and types',
          description: 'For experienced developers',
        },
      ],
    });

    const results = engine.parseResponse(body, defaultParams);
    expect(results.results).toHaveLength(2);
    expect(results.results[0].url).toBe('https://example.com/ts');
    expect(results.results[0].title).toBe('TypeScript Tutorial');
    expect(results.results[0].content).toBe('Learn TypeScript from scratch');
    expect(results.results[0].engine).toBe('jina');
    expect(results.results[0].category).toBe('general');
  });

  it('should use content over description', () => {
    const body = JSON.stringify({
      data: [
        {
          title: 'Test',
          url: 'https://example.com',
          content: 'Primary content',
          description: 'Fallback description',
        },
      ],
    });

    const results = engine.parseResponse(body, defaultParams);
    expect(results.results[0].content).toBe('Primary content');
  });

  it('should fall back to description when content is empty', () => {
    const body = JSON.stringify({
      data: [
        {
          title: 'Test',
          url: 'https://example.com',
          content: '',
          description: 'Fallback description',
        },
      ],
    });

    const results = engine.parseResponse(body, defaultParams);
    expect(results.results[0].content).toBe('Fallback description');
  });

  it('should skip results without url or title', () => {
    const body = JSON.stringify({
      data: [
        { title: 'No URL', url: '', content: 'test' },
        { title: '', url: 'https://example.com', content: 'test' },
        { title: 'Valid', url: 'https://example.com/valid', content: 'test' },
      ],
    });

    const results = engine.parseResponse(body, defaultParams);
    expect(results.results).toHaveLength(1);
    expect(results.results[0].title).toBe('Valid');
  });

  it('should truncate long content to 500 characters', () => {
    const longContent = 'A'.repeat(600);
    const body = JSON.stringify({
      data: [
        {
          title: 'Test',
          url: 'https://example.com',
          content: longContent,
          description: '',
        },
      ],
    });

    const results = engine.parseResponse(body, defaultParams);
    expect(results.results[0].content.length).toBeLessThanOrEqual(500);
    expect(results.results[0].content.endsWith('...')).toBe(true);
  });

  it('should return empty results for empty data array', () => {
    const body = JSON.stringify({ data: [] });
    const results = engine.parseResponse(body, defaultParams);
    expect(results.results).toHaveLength(0);
  });

  it('should return empty results when data is missing', () => {
    const body = JSON.stringify({ code: 200 });
    const results = engine.parseResponse(body, defaultParams);
    expect(results.results).toHaveLength(0);
  });

  it('should return empty results for malformed JSON', () => {
    const results = engine.parseResponse('not json at all', defaultParams);
    expect(results.results).toHaveLength(0);
  });

  it('should return empty results for non-array data', () => {
    const body = JSON.stringify({ data: 'not an array' });
    const results = engine.parseResponse(body, defaultParams);
    expect(results.results).toHaveLength(0);
  });

  // Live integration test (requires JINA_API_KEY)
  // @ts-expect-error - process.env available in vitest node environment
  const apiKey = globalThis.process?.env?.JINA_API_KEY as string | undefined;

  if (apiKey) {
    it('should return real search results (live)', async () => {
      const params: EngineParams = {
        ...defaultParams,
        engineData: { jina_api_key: apiKey },
      };
      const config = engine.buildRequest('typescript', params);
      const response = await fetch(config.url, {
        method: config.method,
        headers: config.headers,
      });
      const body = await response.text();
      const results = engine.parseResponse(body, params);

      expect(results.results.length).toBeGreaterThan(0);
      expect(results.results[0].url).toBeTruthy();
      expect(results.results[0].title).toBeTruthy();
    }, 30000);
  }
});

// ========== JinaReaderEngine Tests ==========

describe('JinaReaderEngine', () => {
  const reader = new JinaReaderEngine();

  // Mock fetch for unit tests
  const mockFetch = vi.fn();

  beforeEach(() => {
    vi.restoreAllMocks();
    mockFetch.mockReset();
    vi.stubGlobal('fetch', mockFetch);
  });

  it('should call correct URL with auth headers', async () => {
    mockFetch.mockResolvedValue({
      ok: true,
      text: () =>
        Promise.resolve(
          JSON.stringify({
            data: {
              title: 'Test Page',
              content: '# Hello',
              url: 'https://example.com',
              description: 'A test',
            },
          })
        ),
    });

    await reader.readPage('https://example.com/page', 'test-key');

    expect(mockFetch).toHaveBeenCalledTimes(1);
    const [url, options] = mockFetch.mock.calls[0];
    expect(url).toBe('https://r.jina.ai/https://example.com/page');
    expect(options.method).toBe('GET');
    expect(options.headers['Authorization']).toBe('Bearer test-key');
    expect(options.headers['Accept']).toBe('application/json');
    expect(options.headers['X-Return-Format']).toBe('markdown');
  });

  it('should return all fields from valid response', async () => {
    mockFetch.mockResolvedValue({
      ok: true,
      text: () =>
        Promise.resolve(
          JSON.stringify({
            data: {
              title: 'My Page Title',
              content: '# Content\n\nSome markdown text.',
              url: 'https://example.com/resolved',
              description: 'Page description',
              images: ['https://example.com/img1.jpg'],
            },
          })
        ),
    });

    const result = await reader.readPage('https://example.com', 'key');

    expect(result.title).toBe('My Page Title');
    expect(result.content).toBe('# Content\n\nSome markdown text.');
    expect(result.url).toBe('https://example.com/resolved');
    expect(result.description).toBe('Page description');
    expect(result.images).toEqual(['https://example.com/img1.jpg']);
  });

  it('should handle missing optional fields gracefully', async () => {
    mockFetch.mockResolvedValue({
      ok: true,
      text: () =>
        Promise.resolve(
          JSON.stringify({
            data: {
              title: 'Minimal',
            },
          })
        ),
    });

    const result = await reader.readPage('https://example.com', 'key');

    expect(result.title).toBe('Minimal');
    expect(result.content).toBe('');
    expect(result.description).toBe('');
    expect(result.images).toBeUndefined();
  });

  it('should use provided URL as fallback when response URL is missing', async () => {
    mockFetch.mockResolvedValue({
      ok: true,
      text: () =>
        Promise.resolve(
          JSON.stringify({
            data: {
              title: 'Test',
              content: 'content',
            },
          })
        ),
    });

    const result = await reader.readPage('https://fallback.com', 'key');
    expect(result.url).toBe('https://fallback.com');
  });

  it('should throw on HTTP error', async () => {
    mockFetch.mockResolvedValue({
      ok: false,
      status: 422,
      statusText: 'Unprocessable Entity',
    });

    await expect(
      reader.readPage('https://bad-url.com', 'key')
    ).rejects.toThrow('Jina Reader: HTTP 422 Unprocessable Entity');
  });

  it('should throw on empty data', async () => {
    mockFetch.mockResolvedValue({
      ok: true,
      text: () => Promise.resolve(JSON.stringify({ code: 200 })),
    });

    await expect(
      reader.readPage('https://example.com', 'key')
    ).rejects.toThrow('Jina Reader: No data in response');
  });

  it('should work without API key (no auth header)', async () => {
    mockFetch.mockResolvedValue({
      ok: true,
      text: () =>
        Promise.resolve(
          JSON.stringify({
            data: { title: 'Test', content: 'ok' },
          })
        ),
    });

    await reader.readPage('https://example.com', '');

    const [, options] = mockFetch.mock.calls[0];
    expect(options.headers['Authorization']).toBeUndefined();
  });

  // Live integration test (requires JINA_API_KEY)
  // @ts-expect-error - process.env available in vitest node environment
  const apiKey = globalThis.process?.env?.JINA_API_KEY as string | undefined;

  if (apiKey) {
    it('should read a real page (live)', async () => {
      vi.restoreAllMocks();
      // Restore real fetch for live test
      vi.unstubAllGlobals();

      const realReader = new JinaReaderEngine();
      const result = await realReader.readPage(
        'https://example.com',
        apiKey
      );

      expect(result.title).toBeTruthy();
      expect(result.content).toBeTruthy();
      expect(result.url).toContain('example.com');
    }, 30000);
  }
});
