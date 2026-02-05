import { describe, it, expect } from 'vitest';
import { LemmyEngine, LemmyCommunitiesEngine } from './lemmy';
import type { EngineParams } from './engine';

describe('LemmyEngine', () => {
  const engine = new LemmyEngine();

  const defaultParams: EngineParams = {
    page: 1,
    locale: 'en-US',
    safeSearch: 1,
    timeRange: '',
    engineData: {},
  };

  it('should have correct metadata', () => {
    expect(engine.name).toBe('lemmy');
    expect(engine.shortcut).toBe('lm');
    expect(engine.categories).toContain('social');
    expect(engine.supportsPaging).toBe(true);
    expect(engine.maxPage).toBe(10);
  });

  it('should build correct API URL', () => {
    const config = engine.buildRequest('test', defaultParams);
    expect(config.url).toContain('lemmy.ml/api/v3/search');
    expect(config.url).toContain('q=test');
    expect(config.url).toContain('type_=Posts');
    expect(config.method).toBe('GET');
  });

  it('should handle pagination', () => {
    const config = engine.buildRequest('test', {
      ...defaultParams,
      page: 3,
    });
    expect(config.url).toContain('page=3');
  });

  it('should apply time range sorting', () => {
    const config = engine.buildRequest('test', {
      ...defaultParams,
      timeRange: 'week',
    });
    expect(config.url).toContain('sort=TopWeek');
  });

  it('should parse post response', () => {
    const sampleResponse = JSON.stringify({
      type_: 'Posts',
      posts: [
        {
          post: {
            id: 12345,
            name: 'Test Post Title',
            body: 'This is the content of the test post',
            url: 'https://external-link.com/article',
            creator_id: 1,
            community_id: 10,
            nsfw: false,
            ap_id: 'https://lemmy.ml/post/12345',
            local: true,
            published: '2024-01-15T10:30:00.000Z',
            thumbnail_url: 'https://lemmy.ml/pictrs/thumbnail.jpg',
          },
          creator: {
            id: 1,
            name: 'testuser',
            display_name: 'Test User',
            avatar: 'https://lemmy.ml/avatar.png',
            actor_id: 'https://lemmy.ml/u/testuser',
            local: true,
          },
          community: {
            id: 10,
            name: 'testcommunity',
            title: 'Test Community',
            icon: 'https://lemmy.ml/community-icon.png',
            actor_id: 'https://lemmy.ml/c/testcommunity',
            local: true,
            nsfw: false,
            subscribers: 5000,
            posts: 100,
            comments: 500,
          },
          counts: {
            id: 1,
            post_id: 12345,
            comments: 42,
            score: 150,
            upvotes: 175,
            downvotes: 25,
            published: '2024-01-15T10:30:00.000Z',
          },
        },
      ],
    });

    const results = engine.parseResponse(sampleResponse, defaultParams);

    expect(results.results.length).toBe(1);

    const first = results.results[0];
    expect(first.url).toBe('https://external-link.com/article');
    expect(first.title).toBe('Test Post Title');
    expect(first.content).toContain('150 points');
    expect(first.content).toContain('42 comments');
    expect(first.source).toContain('!testcommunity@');
    expect(first.channel).toBe('Test User');
    expect(first.category).toBe('social');
    expect(first.publishedAt).toBe('2024-01-15T10:30:00.000Z');
    expect(first.thumbnailUrl).toBeTruthy();
    expect(first.metadata?.postId).toBe(12345);
    expect(first.metadata?.upvotes).toBe(175);
    expect(first.metadata?.downvotes).toBe(25);
  });

  it('should filter NSFW content when safe search is enabled', () => {
    const sampleResponse = JSON.stringify({
      type_: 'Posts',
      posts: [
        {
          post: {
            id: 1,
            name: 'Safe Post',
            nsfw: false,
            ap_id: 'https://lemmy.ml/post/1',
            local: true,
            published: '2024-01-15T10:30:00.000Z',
            creator_id: 1,
            community_id: 1,
          },
          creator: {
            id: 1,
            name: 'user1',
            actor_id: 'https://lemmy.ml/u/user1',
            local: true,
          },
          community: {
            id: 1,
            name: 'safe',
            title: 'Safe Community',
            actor_id: 'https://lemmy.ml/c/safe',
            local: true,
            nsfw: false,
            subscribers: 100,
            posts: 10,
            comments: 50,
          },
          counts: {
            id: 1,
            post_id: 1,
            comments: 0,
            score: 5,
            upvotes: 5,
            downvotes: 0,
            published: '2024-01-15T10:30:00.000Z',
          },
        },
        {
          post: {
            id: 2,
            name: 'NSFW Post',
            nsfw: true,
            ap_id: 'https://lemmy.ml/post/2',
            local: true,
            published: '2024-01-15T10:31:00.000Z',
            creator_id: 2,
            community_id: 2,
          },
          creator: {
            id: 2,
            name: 'user2',
            actor_id: 'https://lemmy.ml/u/user2',
            local: true,
          },
          community: {
            id: 2,
            name: 'nsfw',
            title: 'NSFW Community',
            actor_id: 'https://lemmy.ml/c/nsfw',
            local: true,
            nsfw: true,
            subscribers: 100,
            posts: 10,
            comments: 50,
          },
          counts: {
            id: 2,
            post_id: 2,
            comments: 0,
            score: 5,
            upvotes: 5,
            downvotes: 0,
            published: '2024-01-15T10:31:00.000Z',
          },
        },
      ],
    });

    const results = engine.parseResponse(sampleResponse, { ...defaultParams, safeSearch: 1 });
    expect(results.results.length).toBe(1);
    expect(results.results[0].title).toBe('Safe Post');

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
    const customEngine = new LemmyEngine({ instance: 'lemmy.world' });
    const config = customEngine.buildRequest('test', defaultParams);
    expect(config.url).toContain('lemmy.world/api/v3/search');
  });

  it('should search and return social results', async () => {
    const results = await fetchAndParse(engine, 'linux');

    expect(results.results.length).toBeGreaterThan(0);
    const first = results.results[0];
    expect(first.url).toBeTruthy();
    expect(first.title).toBeTruthy();
    expect(first.category).toBe('social');
    expect(first.source).toMatch(/^!/);
  }, 30000);
});

describe('LemmyCommunitiesEngine', () => {
  const engine = new LemmyCommunitiesEngine();

  const defaultParams: EngineParams = {
    page: 1,
    locale: 'en-US',
    safeSearch: 1,
    timeRange: '',
    engineData: {},
  };

  it('should have correct metadata', () => {
    expect(engine.name).toBe('lemmy communities');
    expect(engine.shortcut).toBe('lmc');
    expect(engine.categories).toContain('social');
  });

  it('should search for communities', () => {
    const config = engine.buildRequest('test', defaultParams);
    expect(config.url).toContain('type_=Communities');
  });

  it('should parse community results', () => {
    const sampleResponse = JSON.stringify({
      type_: 'Communities',
      communities: [
        {
          community: {
            id: 10,
            name: 'programming',
            title: 'Programming',
            description: 'A community for discussing programming topics',
            icon: 'https://lemmy.ml/community-icon.png',
            actor_id: 'https://lemmy.ml/c/programming',
            local: true,
            nsfw: false,
            subscribers: 50000,
            posts: 1000,
            comments: 10000,
          },
          counts: {
            id: 1,
            community_id: 10,
            subscribers: 50000,
            posts: 1000,
            comments: 10000,
            published: '2023-01-01T00:00:00.000Z',
            users_active_day: 100,
            users_active_week: 500,
            users_active_month: 2000,
            users_active_half_year: 5000,
          },
        },
      ],
    });

    const results = engine.parseResponse(sampleResponse, defaultParams);

    expect(results.results.length).toBe(1);

    const first = results.results[0];
    expect(first.url).toBe('https://lemmy.ml/c/programming');
    expect(first.title).toBe('Programming');
    expect(first.content).toContain('!programming');
    expect(first.content).toContain('50000 subscribers');
    expect(first.content).toContain('1000 posts');
    expect(first.thumbnailUrl).toBeTruthy();
    expect(first.metadata?.communityId).toBe(10);
    expect(first.metadata?.subscribers).toBe(50000);
  });
});

async function fetchAndParse(engine: LemmyEngine, query: string) {
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
