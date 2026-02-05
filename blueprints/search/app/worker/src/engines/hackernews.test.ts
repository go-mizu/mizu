import { describe, it, expect } from 'vitest';
import {
  HackerNewsEngine,
  HackerNewsFrontPageEngine,
  HackerNewsShowHNEngine,
  HackerNewsAskHNEngine,
} from './hackernews';
import type { EngineParams } from './engine';

describe('HackerNewsEngine', () => {
  const engine = new HackerNewsEngine();

  const defaultParams: EngineParams = {
    page: 1,
    locale: 'en-US',
    safeSearch: 1,
    timeRange: '',
    engineData: {},
  };

  it('should have correct metadata', () => {
    expect(engine.name).toBe('hacker news');
    expect(engine.shortcut).toBe('hn');
    expect(engine.categories).toContain('news');
    expect(engine.categories).toContain('it');
    expect(engine.supportsPaging).toBe(true);
    expect(engine.maxPage).toBe(30);
  });

  it('should build correct Algolia search URL', () => {
    const config = engine.buildRequest('typescript', defaultParams);
    expect(config.url).toContain('hn.algolia.com/api/v1/search');
    expect(config.url).toContain('query=typescript');
    expect(config.url).toContain('tags=story');
    expect(config.url).toContain('page=0'); // 0-indexed
    expect(config.method).toBe('GET');
  });

  it('should handle pagination', () => {
    const config = engine.buildRequest('test', {
      ...defaultParams,
      page: 3,
    });
    expect(config.url).toContain('page=2'); // page 3 -> index 2
  });

  it('should apply time range filter', () => {
    const config = engine.buildRequest('test', {
      ...defaultParams,
      timeRange: 'week',
    });
    expect(config.url).toContain('numericFilters=');
    expect(config.url).toContain('created_at_i');
  });

  it('should include proper headers', () => {
    const config = engine.buildRequest('test', defaultParams);
    expect(config.headers['User-Agent']).toBeTruthy();
    expect(config.headers['Accept']).toContain('application/json');
  });

  it('should parse Algolia API response', () => {
    const sampleResponse = JSON.stringify({
      hits: [
        {
          objectID: '12345678',
          title: 'Show HN: A new TypeScript framework',
          url: 'https://github.com/example/framework',
          author: 'developer123',
          points: 150,
          num_comments: 42,
          created_at: '2024-01-15T10:30:00.000Z',
          created_at_i: 1705315800,
          _tags: ['story', 'show_hn', 'author_developer123'],
        },
        {
          objectID: '87654321',
          title: 'Ask HN: What tech are you learning in 2024?',
          author: 'curious_dev',
          points: 200,
          num_comments: 156,
          story_text: '<p>I\'m curious what technologies everyone is focusing on this year.</p>',
          created_at: '2024-01-14T15:00:00.000Z',
          created_at_i: 1705245600,
          _tags: ['story', 'ask_hn', 'author_curious_dev'],
        },
      ],
      nbHits: 1000,
      page: 0,
      nbPages: 50,
      hitsPerPage: 30,
    });

    const results = engine.parseResponse(sampleResponse, defaultParams);

    expect(results.results.length).toBe(2);

    const first = results.results[0];
    expect(first.title).toBe('Show HN: A new TypeScript framework');
    expect(first.url).toBe('https://github.com/example/framework');
    expect(first.source).toBe('github.com');
    expect(first.publishedAt).toBe('2024-01-15T10:30:00.000Z');
    expect(first.content).toContain('150 points');
    expect(first.content).toContain('42 comments');
    expect(first.engine).toBe('hacker news');
    expect(first.category).toBe('news');
    expect(first.metadata?.hnId).toBe('12345678');
    expect(first.metadata?.hnUrl).toContain('news.ycombinator.com');

    const second = results.results[1];
    expect(second.title).toBe('Ask HN: What tech are you learning in 2024?');
    expect(second.url).toContain('news.ycombinator.com'); // No external URL
    expect(second.source).toBe('Hacker News');
    expect(second.content).toContain('curious');
  });

  it('should handle empty or malformed response', () => {
    const emptyResults = engine.parseResponse('{}', defaultParams);
    expect(emptyResults.results).toEqual([]);

    const malformedResults = engine.parseResponse('not json', defaultParams);
    expect(malformedResults.results).toEqual([]);
  });

  it('should search and return news results', async () => {
    const results = await fetchAndParse(engine, 'react');

    expect(results.results.length).toBeGreaterThan(0);
    const first = results.results[0];
    expect(first.url).toBeTruthy();
    expect(first.title).toBeTruthy();
    expect(first.category).toBe('news');
    expect(first.publishedAt).toBeTruthy();
    expect(first.metadata?.hnId).toBeTruthy();
  }, 30000);
});

describe('HackerNewsEngine (sort by date)', () => {
  const engine = new HackerNewsEngine({ sortByDate: true });

  it('should have correct name', () => {
    expect(engine.name).toBe('hacker news (recent)');
  });

  it('should use search_by_date endpoint', () => {
    const config = engine.buildRequest('test', {
      page: 1,
      locale: 'en-US',
      safeSearch: 1,
      timeRange: '',
      engineData: {},
    });
    expect(config.url).toContain('search_by_date');
  });
});

describe('HackerNewsFrontPageEngine', () => {
  const engine = new HackerNewsFrontPageEngine();

  const defaultParams: EngineParams = {
    page: 1,
    locale: 'en-US',
    safeSearch: 1,
    timeRange: '',
    engineData: {},
  };

  it('should have correct metadata', () => {
    expect(engine.name).toBe('hacker news front page');
    expect(engine.shortcut).toBe('hnfp');
    expect(engine.supportsPaging).toBe(false);
  });

  it('should build correct front page URL', () => {
    const config = engine.buildRequest('', defaultParams);
    expect(config.url).toContain('hn.algolia.com/api/v1/search');
    expect(config.url).toContain('tags=front_page');
  });

  it('should fetch front page stories', async () => {
    const results = await fetchAndParseFrontPage(engine);

    expect(results.results.length).toBeGreaterThan(0);
    const first = results.results[0];
    expect(first.url).toBeTruthy();
    expect(first.title).toBeTruthy();
    expect(first.engine).toBe('hacker news front page');
  }, 30000);
});

describe('HackerNewsShowHNEngine', () => {
  const engine = new HackerNewsShowHNEngine();

  it('should have correct name', () => {
    expect(engine.name).toBe('hacker news (show hn)');
  });

  it('should filter by show_hn tag', () => {
    const config = engine.buildRequest('test', {
      page: 1,
      locale: 'en-US',
      safeSearch: 1,
      timeRange: '',
      engineData: {},
    });
    expect(config.url).toContain('tags=show_hn');
  });
});

describe('HackerNewsAskHNEngine', () => {
  const engine = new HackerNewsAskHNEngine();

  it('should have correct name', () => {
    expect(engine.name).toBe('hacker news (ask hn)');
  });

  it('should filter by ask_hn tag', () => {
    const config = engine.buildRequest('test', {
      page: 1,
      locale: 'en-US',
      safeSearch: 1,
      timeRange: '',
      engineData: {},
    });
    expect(config.url).toContain('tags=ask_hn');
  });
});

async function fetchAndParse(engine: HackerNewsEngine, query: string) {
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

async function fetchAndParseFrontPage(engine: HackerNewsFrontPageEngine) {
  const params: EngineParams = {
    page: 1,
    locale: 'en-US',
    safeSearch: 1,
    timeRange: '',
    engineData: {},
  };
  const config = engine.buildRequest('', params);
  const res = await fetch(config.url, {
    headers: config.headers,
  });
  const body = await res.text();
  return engine.parseResponse(body, params);
}
