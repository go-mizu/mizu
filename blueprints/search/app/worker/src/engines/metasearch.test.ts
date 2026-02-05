import { describe, it, expect, vi, beforeEach } from 'vitest';
import { MetaSearch, createDefaultMetaSearch } from './metasearch';
import type { OnlineEngine, EngineParams, RequestConfig, EngineResults } from './engine';

// Mock fetch globally
const mockFetch = vi.fn();
global.fetch = mockFetch;

// Create a mock engine for testing
function createMockEngine(
  name: string,
  categories: ('general' | 'images' | 'videos' | 'news')[],
  results: { url: string; title: string; content: string; score: number }[]
): OnlineEngine {
  return {
    name,
    shortcut: name.substring(0, 2),
    categories,
    supportsPaging: true,
    maxPage: 10,
    timeout: 5000,
    weight: 1,
    disabled: false,
    buildRequest: (_query: string, _params: EngineParams): RequestConfig => ({
      url: `https://${name}.example.com/search`,
      method: 'GET',
      headers: {},
      cookies: [],
    }),
    parseResponse: (_body: string, _resp: Response): EngineResults => ({
      results: results.map((r) => ({
        ...r,
        engine: name,
        category: categories[0],
      })),
      suggestions: [],
      corrections: [],
      engineData: {},
    }),
  };
}

describe('MetaSearch', () => {
  let metasearch: MetaSearch;

  beforeEach(() => {
    vi.clearAllMocks();
    metasearch = new MetaSearch();
    mockFetch.mockResolvedValue({
      ok: true,
      text: () => Promise.resolve(''),
    });
  });

  describe('register', () => {
    it('registers an engine', () => {
      const engine = createMockEngine('test', ['general'], []);
      metasearch.register(engine);
      expect(metasearch.get('test')).toBe(engine);
    });
  });

  describe('getByCategory', () => {
    it('returns engines for a category', () => {
      const engine1 = createMockEngine('engine1', ['general'], []);
      const engine2 = createMockEngine('engine2', ['images'], []);
      const engine3 = createMockEngine('engine3', ['general', 'images'], []);

      metasearch.register(engine1);
      metasearch.register(engine2);
      metasearch.register(engine3);

      const generalEngines = metasearch.getByCategory('general');
      expect(generalEngines).toHaveLength(2);
      expect(generalEngines.map((e) => e.name)).toContain('engine1');
      expect(generalEngines.map((e) => e.name)).toContain('engine3');
    });

    it('excludes disabled engines', () => {
      const engine1 = createMockEngine('engine1', ['general'], []);
      const engine2 = createMockEngine('engine2', ['general'], []);
      engine2.disabled = true;

      metasearch.register(engine1);
      metasearch.register(engine2);

      const engines = metasearch.getByCategory('general');
      expect(engines).toHaveLength(1);
      expect(engines[0].name).toBe('engine1');
    });
  });

  describe('listEngines', () => {
    it('returns all registered engine names', () => {
      metasearch.register(createMockEngine('a', ['general'], []));
      metasearch.register(createMockEngine('b', ['images'], []));

      const names = metasearch.listEngines();
      expect(names).toContain('a');
      expect(names).toContain('b');
    });
  });

  describe('search', () => {
    it('returns empty results when no engines for category', async () => {
      const result = await metasearch.search('test', 'general', {
        page: 1,
        locale: 'en',
        safeSearch: 1,
        timeRange: '',
        engineData: {},
      });

      expect(result.results).toHaveLength(0);
      expect(result.totalEngines).toBe(0);
    });

    it('aggregates results from multiple engines', async () => {
      const engine1 = createMockEngine('engine1', ['general'], [
        { url: 'https://a.com', title: 'A', content: 'Content A', score: 1 },
      ]);
      const engine2 = createMockEngine('engine2', ['general'], [
        { url: 'https://b.com', title: 'B', content: 'Content B', score: 2 },
      ]);

      metasearch.register(engine1);
      metasearch.register(engine2);

      const result = await metasearch.search('test', 'general', {
        page: 1,
        locale: 'en',
        safeSearch: 1,
        timeRange: '',
        engineData: {},
      });

      expect(result.results).toHaveLength(2);
      expect(result.totalEngines).toBe(2);
      expect(result.successfulEngines).toBe(2);
    });

    it('deduplicates results by URL', async () => {
      const engine1 = createMockEngine('engine1', ['general'], [
        { url: 'https://example.com/page', title: 'Title 1', content: 'Short', score: 1 },
      ]);
      const engine2 = createMockEngine('engine2', ['general'], [
        {
          url: 'https://www.example.com/page/',
          title: 'Title 2',
          content: 'Longer content here',
          score: 2,
        },
      ]);

      metasearch.register(engine1);
      metasearch.register(engine2);

      const result = await metasearch.search('test', 'general', {
        page: 1,
        locale: 'en',
        safeSearch: 1,
        timeRange: '',
        engineData: {},
      });

      // Should deduplicate to 1 result with merged score (1 + 2 = 3)
      expect(result.results).toHaveLength(1);
      expect(result.results[0].score).toBe(3);
      // Should keep longer content
      expect(result.results[0].content).toBe('Longer content here');
    });

    it('sorts results by score descending', async () => {
      const engine = createMockEngine('test', ['general'], [
        { url: 'https://low.com', title: 'Low', content: '', score: 1 },
        { url: 'https://high.com', title: 'High', content: '', score: 10 },
        { url: 'https://mid.com', title: 'Mid', content: '', score: 5 },
      ]);

      metasearch.register(engine);

      const result = await metasearch.search('test', 'general', {
        page: 1,
        locale: 'en',
        safeSearch: 1,
        timeRange: '',
        engineData: {},
      });

      expect(result.results[0].title).toBe('High');
      expect(result.results[1].title).toBe('Mid');
      expect(result.results[2].title).toBe('Low');
    });

    it('handles engine failures gracefully', async () => {
      const goodEngine = createMockEngine('good', ['general'], [
        { url: 'https://good.com', title: 'Good', content: '', score: 1 },
      ]);

      const badEngine: OnlineEngine = {
        ...createMockEngine('bad', ['general'], []),
        parseResponse: () => {
          throw new Error('Parse error');
        },
      };

      metasearch.register(goodEngine);
      metasearch.register(badEngine);

      const result = await metasearch.search('test', 'general', {
        page: 1,
        locale: 'en',
        safeSearch: 1,
        timeRange: '',
        engineData: {},
      });

      expect(result.results).toHaveLength(1);
      expect(result.successfulEngines).toBe(1);
      expect(result.failedEngines).toContain('bad');
    });
  });
});

describe('createDefaultMetaSearch', () => {
  it('creates a MetaSearch with all default engines', () => {
    const ms = createDefaultMetaSearch();

    const engines = ms.listEngines();

    // Should have all the built-in engines
    expect(engines).toContain('google');
    expect(engines).toContain('bing');
    expect(engines).toContain('brave');
    expect(engines).toContain('wikipedia');
    expect(engines).toContain('youtube');
    expect(engines).toContain('reddit');
    expect(engines).toContain('arxiv');
    expect(engines).toContain('github');
  });

  it('has engines for all main categories', () => {
    const ms = createDefaultMetaSearch();

    expect(ms.getByCategory('general').length).toBeGreaterThan(0);
    expect(ms.getByCategory('images').length).toBeGreaterThan(0);
    expect(ms.getByCategory('videos').length).toBeGreaterThan(0);
    expect(ms.getByCategory('news').length).toBeGreaterThan(0);
  });
});
