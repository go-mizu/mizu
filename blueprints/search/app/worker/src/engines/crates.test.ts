import { describe, it, expect } from 'vitest';
import {
  CratesEngine,
  CratesPopularEngine,
  CratesTrendingEngine,
  CratesRecentEngine,
  CratesNewEngine,
} from './crates';
import type { EngineParams } from './engine';

describe('CratesEngine', () => {
  const engine = new CratesEngine();

  const defaultParams: EngineParams = {
    page: 1,
    locale: 'en-US',
    safeSearch: 1,
    timeRange: '',
    engineData: {},
  };

  it('should have correct metadata', () => {
    expect(engine.name).toBe('crates.io');
    expect(engine.shortcut).toBe('crate');
    expect(engine.categories).toContain('it');
    expect(engine.supportsPaging).toBe(true);
    expect(engine.maxPage).toBe(10);
  });

  it('should build correct crates.io API URL', () => {
    const config = engine.buildRequest('serde', defaultParams);
    expect(config.url).toContain('crates.io/api/v1/crates');
    expect(config.url).toContain('q=serde');
    expect(config.url).toContain('per_page=20');
    expect(config.url).toContain('page=1');
    expect(config.url).toContain('sort=relevance');
    expect(config.method).toBe('GET');
  });

  it('should handle pagination', () => {
    const config = engine.buildRequest('test', {
      ...defaultParams,
      page: 3,
    });
    expect(config.url).toContain('page=3');
  });

  it('should include proper headers', () => {
    const config = engine.buildRequest('test', defaultParams);
    expect(config.headers['User-Agent']).toBeTruthy();
    expect(config.headers['Accept']).toContain('application/json');
  });

  it('should parse crates.io API response', () => {
    const sampleResponse = JSON.stringify({
      crates: [
        {
          id: 'serde',
          name: 'serde',
          description: 'A generic serialization/deserialization framework',
          documentation: 'https://docs.rs/serde',
          homepage: 'https://serde.rs',
          repository: 'https://github.com/serde-rs/serde',
          max_version: '1.0.195',
          max_stable_version: '1.0.195',
          newest_version: '1.0.195',
          downloads: 250000000,
          recent_downloads: 15000000,
          created_at: '2015-03-01T00:00:00.000Z',
          updated_at: '2024-01-15T10:30:00.000Z',
          exact_match: true,
          keywords: ['serde', 'serialization', 'deserialization'],
          categories: ['encoding', 'no-std'],
        },
        {
          id: 'serde_json',
          name: 'serde_json',
          description: 'A JSON serialization file format',
          documentation: 'https://docs.rs/serde_json',
          repository: 'https://github.com/serde-rs/json',
          max_stable_version: '1.0.111',
          downloads: 200000000,
          recent_downloads: 12000000,
          updated_at: '2024-01-10T15:00:00.000Z',
          keywords: ['json', 'serde'],
        },
      ],
      meta: {
        total: 150,
        next_page: '/api/v1/crates?page=2',
      },
    });

    const results = engine.parseResponse(sampleResponse, defaultParams);

    expect(results.results.length).toBe(2);

    const first = results.results[0];
    expect(first.title).toBe('serde');
    expect(first.url).toBe('https://crates.io/crates/serde');
    expect(first.content).toContain('serialization/deserialization framework');
    expect(first.content).toContain('v1.0.195');
    expect(first.content).toContain('250.0M downloads');
    expect(first.content).toContain('15.0M recent');
    expect(first.engine).toBe('crates.io');
    expect(first.category).toBe('it');
    expect(first.language).toBe('Rust');
    expect(first.topics).toContain('serde');
    expect(first.topics).toContain('encoding');
    expect(first.metadata?.version).toBe('1.0.195');
    expect(first.metadata?.downloads).toBe(250000000);
    expect(first.metadata?.recentDownloads).toBe(15000000);
    expect(first.metadata?.documentation).toBe('https://docs.rs/serde');
    expect(first.metadata?.homepage).toBe('https://serde.rs');
    expect(first.metadata?.repository).toBe('https://github.com/serde-rs/serde');
    expect(first.metadata?.exactMatch).toBe(true);

    const second = results.results[1];
    expect(second.title).toBe('serde_json');
    expect(second.metadata?.version).toBe('1.0.111');
  });

  it('should handle empty response', () => {
    const emptyResults = engine.parseResponse('{"crates":[]}', defaultParams);
    expect(emptyResults.results).toEqual([]);
  });

  it('should handle malformed response', () => {
    const malformedResults = engine.parseResponse('not json', defaultParams);
    expect(malformedResults.results).toEqual([]);
  });

  it('should handle missing optional fields', () => {
    const response = JSON.stringify({
      crates: [
        {
          id: 'minimal',
          name: 'minimal',
          downloads: 100,
        },
      ],
    });

    const results = engine.parseResponse(response, defaultParams);
    expect(results.results.length).toBe(1);
    expect(results.results[0].title).toBe('minimal');
  });

  it('should search and return crate results', async () => {
    const results = await fetchAndParse(engine, 'tokio');

    expect(results.results.length).toBeGreaterThan(0);
    const first = results.results[0];
    expect(first.url).toContain('crates.io');
    expect(first.title).toBeTruthy();
    expect(first.category).toBe('it');
    expect(first.language).toBe('Rust');
    expect(first.metadata?.downloads).toBeDefined();
  }, 30000);
});

describe('CratesPopularEngine', () => {
  const engine = new CratesPopularEngine();

  it('should have correct name', () => {
    expect(engine.name).toBe('crates.io (popular)');
  });

  it('should sort by downloads', () => {
    const config = engine.buildRequest('test', {
      page: 1,
      locale: 'en-US',
      safeSearch: 1,
      timeRange: '',
      engineData: {},
    });
    expect(config.url).toContain('sort=downloads');
  });
});

describe('CratesTrendingEngine', () => {
  const engine = new CratesTrendingEngine();

  it('should have correct name', () => {
    expect(engine.name).toBe('crates.io (trending)');
  });

  it('should sort by recent downloads', () => {
    const config = engine.buildRequest('test', {
      page: 1,
      locale: 'en-US',
      safeSearch: 1,
      timeRange: '',
      engineData: {},
    });
    expect(config.url).toContain('sort=recent-downloads');
  });
});

describe('CratesRecentEngine', () => {
  const engine = new CratesRecentEngine();

  it('should have correct name', () => {
    expect(engine.name).toBe('crates.io (recent)');
  });

  it('should sort by recent updates', () => {
    const config = engine.buildRequest('test', {
      page: 1,
      locale: 'en-US',
      safeSearch: 1,
      timeRange: '',
      engineData: {},
    });
    expect(config.url).toContain('sort=recent-updates');
  });
});

describe('CratesNewEngine', () => {
  const engine = new CratesNewEngine();

  it('should have correct name', () => {
    expect(engine.name).toBe('crates.io (new)');
  });

  it('should sort by new', () => {
    const config = engine.buildRequest('test', {
      page: 1,
      locale: 'en-US',
      safeSearch: 1,
      timeRange: '',
      engineData: {},
    });
    expect(config.url).toContain('sort=new');
  });
});

async function fetchAndParse(engine: CratesEngine, query: string) {
  const params: EngineParams = {
    page: 1,
    locale: 'en-US',
    safeSearch: 1,
    timeRange: '',
    engineData: {},
  };
  const config = engine.buildRequest(query, params);
  const res = await fetch(config.url, {
    headers: config.headers,
  });
  const body = await res.text();
  return engine.parseResponse(body, params);
}
