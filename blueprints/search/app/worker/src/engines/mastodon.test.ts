import { describe, it, expect } from 'vitest';
import { MastodonEngine, MastodonAccountsEngine } from './mastodon';
import type { EngineParams } from './engine';

describe('MastodonEngine', () => {
  const engine = new MastodonEngine();

  const defaultParams: EngineParams = {
    page: 1,
    locale: 'en-US',
    safeSearch: 1,
    timeRange: '',
    engineData: {},
  };

  it('should have correct metadata', () => {
    expect(engine.name).toBe('mastodon');
    expect(engine.shortcut).toBe('mst');
    expect(engine.categories).toContain('social');
    expect(engine.supportsPaging).toBe(true);
    expect(engine.maxPage).toBe(10);
  });

  it('should build correct API URL', () => {
    const config = engine.buildRequest('test', defaultParams);
    expect(config.url).toContain('mastodon.social/api/v2/search');
    expect(config.url).toContain('q=test');
    expect(config.url).toContain('type=statuses');
    expect(config.method).toBe('GET');
  });

  it('should handle pagination', () => {
    const config = engine.buildRequest('test', {
      ...defaultParams,
      page: 3,
    });
    expect(config.url).toContain('offset=40'); // (3-1) * 20
  });

  it('should parse status response', () => {
    const sampleResponse = JSON.stringify({
      statuses: [
        {
          id: '123456',
          created_at: '2024-01-15T10:30:00.000Z',
          url: 'https://mastodon.social/@user/123456',
          uri: 'https://mastodon.social/users/user/statuses/123456',
          content: '<p>This is a test toot about TypeScript</p>',
          account: {
            id: '789',
            username: 'user',
            acct: 'user',
            display_name: 'Test User',
            url: 'https://mastodon.social/@user',
            avatar: 'https://mastodon.social/avatars/user.png',
            avatar_static: 'https://mastodon.social/avatars/user.png',
          },
          reblogs_count: 5,
          favourites_count: 10,
          replies_count: 3,
          media_attachments: [],
          sensitive: false,
        },
      ],
      accounts: [],
      hashtags: [
        { name: 'typescript', url: 'https://mastodon.social/tags/typescript' },
      ],
    });

    const results = engine.parseResponse(sampleResponse, defaultParams);

    expect(results.results.length).toBe(1);
    expect(results.suggestions).toContain('#typescript');

    const first = results.results[0];
    expect(first.url).toBe('https://mastodon.social/@user/123456');
    expect(first.title).toContain('Test User:');
    expect(first.content).toContain('test toot');
    expect(first.source).toBe('@user');
    expect(first.category).toBe('social');
    expect(first.publishedAt).toBe('2024-01-15T10:30:00.000Z');
    expect(first.metadata?.statusId).toBe('123456');
    expect(first.metadata?.reblogsCount).toBe(5);
    expect(first.metadata?.favouritesCount).toBe(10);
  });

  it('should filter sensitive content when safe search is enabled', () => {
    const sampleResponse = JSON.stringify({
      statuses: [
        {
          id: '1',
          created_at: '2024-01-15T10:30:00.000Z',
          url: 'https://mastodon.social/@user/1',
          content: '<p>Safe content</p>',
          account: {
            id: '1',
            username: 'user1',
            acct: 'user1',
            display_name: 'User 1',
            url: 'https://mastodon.social/@user1',
            avatar: '',
            avatar_static: '',
          },
          reblogs_count: 0,
          favourites_count: 0,
          sensitive: false,
        },
        {
          id: '2',
          created_at: '2024-01-15T10:31:00.000Z',
          url: 'https://mastodon.social/@user/2',
          content: '<p>Sensitive content</p>',
          account: {
            id: '2',
            username: 'user2',
            acct: 'user2',
            display_name: 'User 2',
            url: 'https://mastodon.social/@user2',
            avatar: '',
            avatar_static: '',
          },
          reblogs_count: 0,
          favourites_count: 0,
          sensitive: true,
        },
      ],
    });

    const results = engine.parseResponse(sampleResponse, { ...defaultParams, safeSearch: 1 });
    expect(results.results.length).toBe(1);
    expect(results.results[0].url).toContain('/1');

    const resultsNoSafe = engine.parseResponse(sampleResponse, { ...defaultParams, safeSearch: 0 });
    expect(resultsNoSafe.results.length).toBe(2);
  });

  it('should handle empty or malformed response', () => {
    const emptyResults = engine.parseResponse('{}', defaultParams);
    expect(emptyResults.results).toEqual([]);

    const malformedResults = engine.parseResponse('not json', defaultParams);
    expect(malformedResults.results).toEqual([]);
  });

  it('should support custom instance', () => {
    const customEngine = new MastodonEngine({ instance: 'hachyderm.io' });
    const config = customEngine.buildRequest('test', defaultParams);
    expect(config.url).toContain('hachyderm.io/api/v2/search');
  });

  it('should search and return social results', async () => {
    const results = await fetchAndParse(engine, 'javascript');

    // API may return empty results if instance doesn't allow unauthenticated search
    expect(results.results).toBeDefined();
    expect(Array.isArray(results.results)).toBe(true);
    if (results.results.length > 0) {
      const first = results.results[0];
      expect(first.url).toBeTruthy();
      expect(first.title).toBeTruthy();
      expect(first.category).toBe('social');
      expect(first.source).toMatch(/^@/);
    }
  }, 30000);
});

describe('MastodonAccountsEngine', () => {
  const engine = new MastodonAccountsEngine();

  const defaultParams: EngineParams = {
    page: 1,
    locale: 'en-US',
    safeSearch: 1,
    timeRange: '',
    engineData: {},
  };

  it('should have correct metadata', () => {
    expect(engine.name).toBe('mastodon accounts');
    expect(engine.shortcut).toBe('msta');
    expect(engine.categories).toContain('social');
  });

  it('should search for accounts', () => {
    const config = engine.buildRequest('test', defaultParams);
    expect(config.url).toContain('type=accounts');
  });

  it('should parse account results', () => {
    const sampleResponse = JSON.stringify({
      accounts: [
        {
          id: '123',
          username: 'testuser',
          acct: 'testuser@mastodon.social',
          display_name: 'Test User',
          url: 'https://mastodon.social/@testuser',
          avatar: 'https://mastodon.social/avatars/testuser.png',
          avatar_static: 'https://mastodon.social/avatars/testuser.png',
          followers_count: 1000,
          following_count: 500,
          statuses_count: 250,
        },
      ],
    });

    const results = engine.parseResponse(sampleResponse, defaultParams);

    expect(results.results.length).toBe(1);

    const first = results.results[0];
    expect(first.url).toBe('https://mastodon.social/@testuser');
    expect(first.title).toBe('Test User');
    expect(first.content).toContain('1000 followers');
    expect(first.content).toContain('250 posts');
    expect(first.thumbnailUrl).toBeTruthy();
    expect(first.metadata?.accountId).toBe('123');
    expect(first.metadata?.followersCount).toBe(1000);
  });
});

async function fetchAndParse(engine: MastodonEngine, query: string) {
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
