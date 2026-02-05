import { describe, it, expect, beforeEach } from 'vitest';
import { MetaSearch, createDefaultMetaSearch } from './metasearch';
import type { OnlineEngine, EngineParams, RequestConfig, EngineResults, Category } from './engine';

// Create a simple test engine that doesn't need network
function createTestEngine(
  name: string,
  categories: Category[],
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
    parseResponse: (_body: string, _params: EngineParams): EngineResults => ({
      results: results.map((r) => ({
        ...r,
        engine: name,
        category: categories[0],
      })),
      suggestions: [`suggestion for ${name}`],
      corrections: [],
      engineData: {},
    }),
  };
}

describe('MetaSearch', () => {
  let metasearch: MetaSearch;

  beforeEach(() => {
    metasearch = new MetaSearch();
  });

  describe('engine registration', () => {
    it('registers an engine', () => {
      const engine = createTestEngine('test', ['general'], []);
      metasearch.register(engine);
      expect(metasearch.get('test')).toBe(engine);
    });

    it('overwrites engine with same name', () => {
      const engine1 = createTestEngine('test', ['general'], []);
      const engine2 = createTestEngine('test', ['images'], []);

      metasearch.register(engine1);
      metasearch.register(engine2);

      expect(metasearch.get('test')?.categories).toContain('images');
    });

    it('returns undefined for unregistered engine', () => {
      expect(metasearch.get('nonexistent')).toBeUndefined();
    });
  });

  describe('getByCategory', () => {
    it('returns engines for a category', () => {
      const engine1 = createTestEngine('engine1', ['general'], []);
      const engine2 = createTestEngine('engine2', ['images'], []);
      const engine3 = createTestEngine('engine3', ['general', 'images'], []);

      metasearch.register(engine1);
      metasearch.register(engine2);
      metasearch.register(engine3);

      const generalEngines = metasearch.getByCategory('general');
      expect(generalEngines).toHaveLength(2);
      expect(generalEngines.map((e) => e.name)).toContain('engine1');
      expect(generalEngines.map((e) => e.name)).toContain('engine3');
    });

    it('excludes disabled engines', () => {
      const engine1 = createTestEngine('engine1', ['general'], []);
      const engine2 = createTestEngine('engine2', ['general'], []);
      engine2.disabled = true;

      metasearch.register(engine1);
      metasearch.register(engine2);

      const engines = metasearch.getByCategory('general');
      expect(engines).toHaveLength(1);
      expect(engines[0].name).toBe('engine1');
    });

    it('returns empty array for unknown category', () => {
      const engine = createTestEngine('test', ['general'], []);
      metasearch.register(engine);

      const engines = metasearch.getByCategory('science');
      expect(engines).toHaveLength(0);
    });
  });

  describe('listEngines', () => {
    it('returns all registered engine names', () => {
      metasearch.register(createTestEngine('alpha', ['general'], []));
      metasearch.register(createTestEngine('beta', ['images'], []));
      metasearch.register(createTestEngine('gamma', ['videos'], []));

      const names = metasearch.listEngines();
      expect(names).toContain('alpha');
      expect(names).toContain('beta');
      expect(names).toContain('gamma');
    });

    it('returns empty array when no engines', () => {
      const names = metasearch.listEngines();
      expect(names).toHaveLength(0);
    });
  });
});

describe('createDefaultMetaSearch', () => {
  let metasearch: MetaSearch;

  beforeEach(() => {
    metasearch = createDefaultMetaSearch();
  });

  it('creates MetaSearch with all default engines', () => {
    const engines = metasearch.listEngines();

    // General search
    expect(engines).toContain('google');
    expect(engines).toContain('bing');
    expect(engines).toContain('brave');
    expect(engines).toContain('wikipedia');

    // Images (names use spaces)
    expect(engines).toContain('google images');
    expect(engines).toContain('bing images');
    expect(engines).toContain('duckduckgo images');

    // Videos
    expect(engines).toContain('youtube');
    expect(engines).toContain('duckduckgo videos');

    // News
    expect(engines).toContain('bing news');
    expect(engines).toContain('duckduckgo news');

    // Specialized
    expect(engines).toContain('arxiv');
    expect(engines).toContain('github');
    expect(engines).toContain('reddit');
  });

  it('has engines for general category', () => {
    const engines = metasearch.getByCategory('general');
    expect(engines.length).toBeGreaterThanOrEqual(3);
    expect(engines.map((e) => e.name)).toContain('google');
    expect(engines.map((e) => e.name)).toContain('bing');
  });

  it('has engines for images category', () => {
    const engines = metasearch.getByCategory('images');
    expect(engines.length).toBeGreaterThanOrEqual(2);
  });

  it('has engines for videos category', () => {
    const engines = metasearch.getByCategory('videos');
    expect(engines.length).toBeGreaterThanOrEqual(1);
    expect(engines.map((e) => e.name)).toContain('youtube');
  });

  it('has engines for news category', () => {
    const engines = metasearch.getByCategory('news');
    expect(engines.length).toBeGreaterThanOrEqual(1);
  });

  it('has engines for science category', () => {
    const engines = metasearch.getByCategory('science');
    expect(engines.length).toBeGreaterThanOrEqual(1);
    expect(engines.map((e) => e.name)).toContain('arxiv');
  });

  it('has engines for IT category', () => {
    const engines = metasearch.getByCategory('it');
    expect(engines.length).toBeGreaterThanOrEqual(1);
    expect(engines.map((e) => e.name)).toContain('github');
  });

  it('has engines for social category', () => {
    const engines = metasearch.getByCategory('social');
    expect(engines.length).toBeGreaterThanOrEqual(1);
    expect(engines.map((e) => e.name)).toContain('reddit');
  });

  describe('engine properties', () => {
    it('google engine has correct properties', () => {
      const google = metasearch.get('google');
      expect(google).toBeDefined();
      expect(google?.shortcut).toBe('g');
      expect(google?.categories).toContain('general');
      expect(google?.supportsPaging).toBe(true);
      expect(google?.disabled).toBe(false);
    });

    it('youtube engine has correct properties', () => {
      const youtube = metasearch.get('youtube');
      expect(youtube).toBeDefined();
      expect(youtube?.shortcut).toBe('yt');
      expect(youtube?.categories).toContain('videos');
    });

    it('wikipedia engine has correct properties', () => {
      const wikipedia = metasearch.get('wikipedia');
      expect(wikipedia).toBeDefined();
      expect(wikipedia?.shortcut).toBe('w');
      expect(wikipedia?.categories).toContain('general');
    });
  });
});
