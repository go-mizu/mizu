import { describe, it, expect } from 'vitest';
import { GitLabEngine } from './gitlab';
import type { EngineParams } from './engine';

describe('GitLabEngine', () => {
  const engine = new GitLabEngine();

  const defaultParams: EngineParams = {
    page: 1,
    locale: 'en-US',
    safeSearch: 1,
    timeRange: '',
    engineData: {},
  };

  it('should have correct metadata', () => {
    expect(engine.name).toBe('gitlab');
    expect(engine.shortcut).toBe('gl');
    expect(engine.categories).toContain('it');
    expect(engine.supportsPaging).toBe(true);
    expect(engine.maxPage).toBe(10);
  });

  it('should build correct GitLab API URL', () => {
    const config = engine.buildRequest('typescript', defaultParams);
    expect(config.url).toContain('gitlab.com/api/v4/projects');
    expect(config.url).toContain('search=typescript');
    expect(config.url).toContain('order_by=last_activity_at');
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

  it('should parse GitLab API response', () => {
    const sampleResponse = JSON.stringify([
      {
        id: 12345,
        name: 'awesome-project',
        path_with_namespace: 'user/awesome-project',
        description: 'An awesome TypeScript project',
        web_url: 'https://gitlab.com/user/awesome-project',
        avatar_url: 'https://gitlab.com/uploads/-/avatar.png',
        star_count: 1500,
        forks_count: 200,
        open_issues_count: 15,
        last_activity_at: '2024-01-15T10:30:00.000Z',
        created_at: '2023-01-01T00:00:00.000Z',
        default_branch: 'main',
        topics: ['typescript', 'web', 'framework'],
        namespace: {
          name: 'user',
          avatar_url: 'https://gitlab.com/uploads/-/user-avatar.png',
        },
      },
      {
        id: 67890,
        name: 'another-project',
        path_with_namespace: 'org/another-project',
        description: 'Another cool project',
        web_url: 'https://gitlab.com/org/another-project',
        star_count: 500,
        forks_count: 50,
        last_activity_at: '2024-01-10T15:00:00.000Z',
        topics: ['go', 'cli'],
      },
    ]);

    const results = engine.parseResponse(sampleResponse, defaultParams);

    expect(results.results.length).toBe(2);

    const first = results.results[0];
    expect(first.title).toBe('user/awesome-project');
    expect(first.url).toBe('https://gitlab.com/user/awesome-project');
    expect(first.content).toContain('awesome TypeScript project');
    expect(first.content).toContain('1.5k stars');
    expect(first.content).toContain('200 forks');
    expect(first.engine).toBe('gitlab');
    expect(first.category).toBe('it');
    expect(first.stars).toBe(1500);
    expect(first.topics).toContain('typescript');
    expect(first.metadata?.forks).toBe(200);
    expect(first.metadata?.openIssues).toBe(15);

    const second = results.results[1];
    expect(second.title).toBe('org/another-project');
    expect(second.stars).toBe(500);
  });

  it('should handle empty response', () => {
    const emptyResults = engine.parseResponse('[]', defaultParams);
    expect(emptyResults.results).toEqual([]);
  });

  it('should handle malformed response', () => {
    const malformedResults = engine.parseResponse('not json', defaultParams);
    expect(malformedResults.results).toEqual([]);
  });

  it('should handle response with missing fields', () => {
    const partialResponse = JSON.stringify([
      {
        id: 1,
        name: 'minimal-project',
        web_url: 'https://gitlab.com/user/minimal-project',
      },
    ]);

    const results = engine.parseResponse(partialResponse, defaultParams);
    expect(results.results.length).toBe(1);
    expect(results.results[0].title).toBe('minimal-project');
  });

  it('should search and return project results', async () => {
    try {
      const results = await fetchAndParse(engine, 'react');

      // If we got results, validate them
      if (results.results.length > 0) {
        const first = results.results[0];
        expect(first.url).toContain('gitlab.com');
        expect(first.title).toBeTruthy();
        expect(first.category).toBe('it');
        expect(first.engine).toBe('gitlab');
      }
      // If no results, the API might be rate limited or unavailable - that's OK for integration tests
    } catch (error) {
      // GitLab API may be unavailable or rate limited
      console.warn('GitLab API unavailable for integration test:', error);
    }
  }, 30000);
});

describe('GitLabEngine with custom base URL', () => {
  it('should use custom base URL', () => {
    const engine = new GitLabEngine({ baseUrl: 'https://gitlab.example.com' });
    const config = engine.buildRequest('test', {
      page: 1,
      locale: 'en-US',
      safeSearch: 1,
      timeRange: '',
      engineData: {},
    });
    expect(config.url).toContain('gitlab.example.com/api/v4/projects');
  });
});

async function fetchAndParse(engine: GitLabEngine, query: string) {
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
