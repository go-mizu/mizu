import { describe, it, expect } from 'vitest';
import {
  StackOverflowEngine,
  ServerFaultEngine,
  SuperUserEngine,
  AskUbuntuEngine,
} from './stackoverflow';
import type { EngineParams } from './engine';

describe('StackOverflowEngine', () => {
  const engine = new StackOverflowEngine();

  const defaultParams: EngineParams = {
    page: 1,
    locale: 'en-US',
    safeSearch: 1,
    timeRange: '',
    engineData: {},
  };

  it('should have correct metadata', () => {
    expect(engine.name).toBe('stack overflow');
    expect(engine.shortcut).toBe('so');
    expect(engine.categories).toContain('it');
    expect(engine.supportsPaging).toBe(true);
    expect(engine.maxPage).toBe(10);
  });

  it('should build correct Stack Exchange API URL', () => {
    const config = engine.buildRequest('typescript generics', defaultParams);
    expect(config.url).toContain('api.stackexchange.com/2.3/search/advanced');
    expect(config.url).toContain('intitle=typescript');
    expect(config.url).toContain('site=stackoverflow');
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

  it('should apply time range filter', () => {
    const config = engine.buildRequest('test', {
      ...defaultParams,
      timeRange: 'week',
    });
    expect(config.url).toContain('fromdate=');
  });

  it('should include proper headers', () => {
    const config = engine.buildRequest('test', defaultParams);
    expect(config.headers['User-Agent']).toBeTruthy();
    expect(config.headers['Accept']).toContain('application/json');
  });

  it('should parse Stack Exchange API response', () => {
    const sampleResponse = JSON.stringify({
      items: [
        {
          question_id: 12345678,
          title: 'How to use TypeScript generics with React?',
          link: 'https://stackoverflow.com/questions/12345678/typescript-generics',
          body_markdown: 'I am trying to use TypeScript generics with React components. Here is my code...',
          tags: ['typescript', 'react', 'generics'],
          score: 150,
          answer_count: 5,
          view_count: 25000,
          is_answered: true,
          accepted_answer_id: 12345679,
          creation_date: 1705315800,
          last_activity_date: 1705402200,
          owner: {
            display_name: 'developer123',
            link: 'https://stackoverflow.com/users/123/developer123',
            reputation: 5000,
            profile_image: 'https://i.stack.imgur.com/avatar.jpg',
          },
        },
        {
          question_id: 87654321,
          title: 'TypeScript type inference not working',
          link: 'https://stackoverflow.com/questions/87654321/type-inference',
          tags: ['typescript', 'types'],
          score: 25,
          answer_count: 2,
          view_count: 3000,
          is_answered: false,
          creation_date: 1705245600,
          last_activity_date: 1705245600,
          owner: {
            display_name: 'newbie_dev',
          },
        },
      ],
      has_more: true,
      quota_max: 300,
      quota_remaining: 298,
    });

    const results = engine.parseResponse(sampleResponse, defaultParams);

    expect(results.results.length).toBe(2);

    const first = results.results[0];
    expect(first.title).toBe('How to use TypeScript generics with React?');
    expect(first.url).toBe('https://stackoverflow.com/questions/12345678/typescript-generics');
    expect(first.content).toContain('150 votes');
    expect(first.content).toContain('5 answers');
    expect(first.content).toContain('answered');
    expect(first.engine).toBe('stack overflow');
    expect(first.category).toBe('it');
    expect(first.topics).toContain('typescript');
    expect(first.topics).toContain('react');
    expect(first.metadata?.votes).toBe(150);
    expect(first.metadata?.answers).toBe(5);
    expect(first.metadata?.views).toBe(25000);
    expect(first.metadata?.isAnswered).toBe(true);
    expect(first.metadata?.hasAcceptedAnswer).toBe(true);
    expect(first.metadata?.author).toBe('developer123');

    const second = results.results[1];
    expect(second.title).toBe('TypeScript type inference not working');
    expect(second.metadata?.isAnswered).toBe(false);
    expect(second.metadata?.hasAcceptedAnswer).toBe(false);
  });

  it('should handle empty response', () => {
    const emptyResults = engine.parseResponse('{"items":[]}', defaultParams);
    expect(emptyResults.results).toEqual([]);
  });

  it('should handle malformed response', () => {
    const malformedResults = engine.parseResponse('not json', defaultParams);
    expect(malformedResults.results).toEqual([]);
  });

  it('should decode HTML entities in titles', () => {
    const response = JSON.stringify({
      items: [
        {
          question_id: 1,
          title: 'How to use &lt;T&gt; in TypeScript?',
          link: 'https://stackoverflow.com/questions/1/test',
          score: 10,
          answer_count: 1,
          view_count: 100,
          is_answered: true,
          creation_date: 1705315800,
          last_activity_date: 1705315800,
        },
      ],
    });

    const results = engine.parseResponse(response, defaultParams);
    expect(results.results[0].title).toBe('How to use <T> in TypeScript?');
  });

  it('should search and return Q&A results', async () => {
    const results = await fetchAndParse(engine, 'javascript async await');

    expect(results.results.length).toBeGreaterThan(0);
    const first = results.results[0];
    expect(first.url).toContain('stackoverflow.com');
    expect(first.title).toBeTruthy();
    expect(first.category).toBe('it');
    expect(first.metadata?.votes).toBeDefined();
  }, 30000);
});

describe('ServerFaultEngine', () => {
  const engine = new ServerFaultEngine();

  it('should have correct metadata', () => {
    expect(engine.name).toBe('server fault');
    expect(engine.shortcut).toBe('sf');
  });

  it('should use serverfault site', () => {
    const config = engine.buildRequest('nginx', {
      page: 1,
      locale: 'en-US',
      safeSearch: 1,
      timeRange: '',
      engineData: {},
    });
    expect(config.url).toContain('site=serverfault');
  });
});

describe('SuperUserEngine', () => {
  const engine = new SuperUserEngine();

  it('should have correct metadata', () => {
    expect(engine.name).toBe('super user');
    expect(engine.shortcut).toBe('su');
  });

  it('should use superuser site', () => {
    const config = engine.buildRequest('windows', {
      page: 1,
      locale: 'en-US',
      safeSearch: 1,
      timeRange: '',
      engineData: {},
    });
    expect(config.url).toContain('site=superuser');
  });
});

describe('AskUbuntuEngine', () => {
  const engine = new AskUbuntuEngine();

  it('should have correct metadata', () => {
    expect(engine.name).toBe('ask ubuntu');
    expect(engine.shortcut).toBe('au');
  });

  it('should use askubuntu site', () => {
    const config = engine.buildRequest('apt install', {
      page: 1,
      locale: 'en-US',
      safeSearch: 1,
      timeRange: '',
      engineData: {},
    });
    expect(config.url).toContain('site=askubuntu');
  });
});

async function fetchAndParse(engine: StackOverflowEngine, query: string) {
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
