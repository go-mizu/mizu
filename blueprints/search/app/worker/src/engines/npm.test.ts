import { describe, it, expect } from 'vitest';
import { NpmEngine, formatDownloads } from './npm';
import type { EngineParams } from './engine';

describe('NpmEngine', () => {
  const engine = new NpmEngine();

  const defaultParams: EngineParams = {
    page: 1,
    locale: 'en-US',
    safeSearch: 1,
    timeRange: '',
    engineData: {},
  };

  it('should have correct metadata', () => {
    expect(engine.name).toBe('npm');
    expect(engine.shortcut).toBe('npm');
    expect(engine.categories).toContain('it');
    expect(engine.supportsPaging).toBe(true);
    expect(engine.maxPage).toBe(10);
  });

  it('should build correct NPM Registry API URL', () => {
    const config = engine.buildRequest('react', defaultParams);
    expect(config.url).toContain('registry.npmjs.org/-/v1/search');
    expect(config.url).toContain('text=react');
    expect(config.url).toContain('size=20');
    expect(config.url).toContain('from=0');
    expect(config.method).toBe('GET');
  });

  it('should handle pagination', () => {
    const config = engine.buildRequest('test', {
      ...defaultParams,
      page: 3,
    });
    expect(config.url).toContain('from=40'); // (3-1) * 20
  });

  it('should include proper headers', () => {
    const config = engine.buildRequest('test', defaultParams);
    expect(config.headers['User-Agent']).toBeTruthy();
    expect(config.headers['Accept']).toContain('application/json');
  });

  it('should parse NPM Registry API response', () => {
    const sampleResponse = JSON.stringify({
      objects: [
        {
          package: {
            name: 'react',
            scope: 'unscoped',
            version: '18.2.0',
            description: 'React is a JavaScript library for building user interfaces.',
            keywords: ['react', 'frontend', 'ui', 'javascript'],
            date: '2024-01-15T10:30:00.000Z',
            links: {
              npm: 'https://www.npmjs.com/package/react',
              homepage: 'https://reactjs.org/',
              repository: 'https://github.com/facebook/react',
              bugs: 'https://github.com/facebook/react/issues',
            },
            author: {
              name: 'React Team',
            },
            publisher: {
              username: 'fb',
              email: 'fb@fb.com',
            },
            maintainers: [
              { username: 'fb' },
              { username: 'react-team' },
            ],
          },
          score: {
            final: 0.95,
            detail: {
              quality: 0.98,
              popularity: 0.99,
              maintenance: 0.88,
            },
          },
          searchScore: 100000,
        },
        {
          package: {
            name: 'react-dom',
            version: '18.2.0',
            description: 'React package for working with the DOM.',
            keywords: ['react', 'dom'],
            date: '2024-01-15T10:30:00.000Z',
            links: {
              npm: 'https://www.npmjs.com/package/react-dom',
            },
          },
          score: {
            final: 0.92,
            detail: {
              quality: 0.95,
              popularity: 0.97,
              maintenance: 0.85,
            },
          },
          flags: {
            deprecated: 'Use react-dom/client instead',
          },
        },
      ],
      total: 5000,
    });

    const results = engine.parseResponse(sampleResponse, defaultParams);

    expect(results.results.length).toBe(2);

    const first = results.results[0];
    expect(first.title).toBe('react');
    expect(first.url).toBe('https://www.npmjs.com/package/react');
    expect(first.content).toContain('JavaScript library for building user interfaces');
    expect(first.content).toContain('v18.2.0');
    expect(first.content).toContain('99% popularity');
    expect(first.engine).toBe('npm');
    expect(first.category).toBe('it');
    expect(first.topics).toContain('react');
    expect(first.topics).toContain('frontend');
    expect(first.metadata?.version).toBe('18.2.0');
    expect(first.metadata?.homepage).toBe('https://reactjs.org/');
    expect(first.metadata?.repository).toBe('https://github.com/facebook/react');
    expect(first.metadata?.author).toBe('React Team');

    const second = results.results[1];
    expect(second.title).toBe('react-dom');
    expect(second.content).toContain('DEPRECATED');
    expect(second.metadata?.deprecated).toBe('Use react-dom/client instead');
  });

  it('should handle empty response', () => {
    const emptyResults = engine.parseResponse('{"objects":[]}', defaultParams);
    expect(emptyResults.results).toEqual([]);
  });

  it('should handle malformed response', () => {
    const malformedResults = engine.parseResponse('not json', defaultParams);
    expect(malformedResults.results).toEqual([]);
  });

  it('should generate npm URL when not provided', () => {
    const response = JSON.stringify({
      objects: [
        {
          package: {
            name: 'custom-package',
            version: '1.0.0',
            links: {},
          },
        },
      ],
    });

    const results = engine.parseResponse(response, defaultParams);
    expect(results.results[0].url).toBe('https://www.npmjs.com/package/custom-package');
  });

  it('should search and return package results', async () => {
    const results = await fetchAndParse(engine, 'express');

    expect(results.results.length).toBeGreaterThan(0);
    const first = results.results[0];
    expect(first.url).toContain('npmjs.com');
    expect(first.title).toBeTruthy();
    expect(first.category).toBe('it');
    expect(first.metadata?.version).toBeTruthy();
  }, 30000);
});

describe('formatDownloads', () => {
  it('should format small numbers as-is', () => {
    expect(formatDownloads(0)).toBe('0');
    expect(formatDownloads(100)).toBe('100');
    expect(formatDownloads(999)).toBe('999');
  });

  it('should format thousands with k suffix', () => {
    expect(formatDownloads(1000)).toBe('1.0k');
    expect(formatDownloads(1500)).toBe('1.5k');
    expect(formatDownloads(999999)).toBe('1000.0k');
  });

  it('should format millions with M suffix', () => {
    expect(formatDownloads(1000000)).toBe('1.0M');
    expect(formatDownloads(2500000)).toBe('2.5M');
  });

  it('should format billions with B suffix', () => {
    expect(formatDownloads(1000000000)).toBe('1.0B');
    expect(formatDownloads(5000000000)).toBe('5.0B');
  });
});

async function fetchAndParse(engine: NpmEngine, query: string) {
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
