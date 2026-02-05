import { describe, it, expect } from 'vitest';
import { YouTubeEngine } from './youtube';

describe('YouTubeEngine', () => {
  const engine = new YouTubeEngine();

  it('should have correct metadata', () => {
    expect(engine.name).toBe('youtube');
    expect(engine.categories).toContain('videos');
  });

  it('should build correct search URL', () => {
    const config = engine.buildRequest('typescript tutorial', {
      page: 1,
      locale: 'en',
      safeSearch: 1,
      timeRange: '',
      engineData: {},
    });
    expect(config.url).toContain('search_query=typescript');
    expect(config.method).toBe('GET');
  });

  it('should build URL with time range filter', () => {
    const config = engine.buildRequest('test', {
      page: 1,
      locale: 'en',
      safeSearch: 1,
      timeRange: 'week',
      engineData: {},
    });
    expect(config.url).toContain('sp=');
  });

  it('should search and return video results', async () => {
    const results = await fetchAndParse(engine, 'javascript tutorial');

    expect(results.results.length).toBeGreaterThan(0);
    const first = results.results[0];
    expect(first.url).toContain('youtube.com/watch');
    expect(first.title).toBeTruthy();
    expect(first.thumbnailUrl).toBeTruthy();
  }, 30000);
});

async function fetchAndParse(engine: YouTubeEngine, query: string) {
  const params = { page: 1, locale: 'en', safeSearch: 1 as const, timeRange: '' as const, engineData: {} };
  const config = engine.buildRequest(query, params);
  const res = await fetch(config.url, {
    headers: config.headers,
  });
  const body = await res.text();
  return engine.parseResponse(body, params);
}
