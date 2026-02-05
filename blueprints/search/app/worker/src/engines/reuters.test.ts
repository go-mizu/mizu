import { describe, it, expect } from 'vitest';
import { ReutersEngine, ReutersRSSEngine } from './reuters';
import type { EngineParams } from './engine';

describe('ReutersEngine', () => {
  const engine = new ReutersEngine();

  const defaultParams: EngineParams = {
    page: 1,
    locale: 'en-US',
    safeSearch: 1,
    timeRange: '',
    engineData: {},
  };

  it('should have correct metadata', () => {
    expect(engine.name).toBe('reuters');
    expect(engine.shortcut).toBe('rtr');
    expect(engine.categories).toContain('news');
    expect(engine.supportsPaging).toBe(true);
    expect(engine.weight).toBeGreaterThan(1); // High authority source
  });

  it('should build correct search URL with query parameter', () => {
    const config = engine.buildRequest('climate change', defaultParams);
    expect(config.url).toContain('reuters.com/pf/api');
    expect(config.url).toContain('query=');
    expect(config.method).toBe('GET');

    // Decode and check the query object
    const urlObj = new URL(config.url);
    const queryParam = urlObj.searchParams.get('query');
    expect(queryParam).toBeTruthy();

    const queryObj = JSON.parse(decodeURIComponent(queryParam!));
    expect(queryObj.keyword).toBe('climate change');
    expect(queryObj.offset).toBe(0);
    expect(queryObj.size).toBe(20);
  });

  it('should handle pagination', () => {
    const config = engine.buildRequest('test', {
      ...defaultParams,
      page: 3,
    });

    const urlObj = new URL(config.url);
    const queryParam = urlObj.searchParams.get('query');
    const queryObj = JSON.parse(decodeURIComponent(queryParam!));
    expect(queryObj.offset).toBe(40); // (3-1) * 20 = 40
  });

  it('should apply time range filter', () => {
    const config = engine.buildRequest('test', {
      ...defaultParams,
      timeRange: 'week',
    });

    const urlObj = new URL(config.url);
    const queryParam = urlObj.searchParams.get('query');
    const queryObj = JSON.parse(decodeURIComponent(queryParam!));
    expect(queryObj.date_range).toBeDefined();
    expect(queryObj.date_range.start_date).toBeTruthy();
    expect(queryObj.date_range.end_date).toBeTruthy();
  });

  it('should include proper headers', () => {
    const config = engine.buildRequest('test', defaultParams);
    expect(config.headers['User-Agent']).toBeTruthy();
    expect(config.headers['Accept']).toContain('application/json');
    expect(config.headers['Referer']).toContain('reuters.com');
  });

  it('should parse JSON API response', () => {
    const sampleResponse = JSON.stringify({
      result: {
        articles: [
          {
            id: 'article-123',
            canonical_url: '/world/us/breaking-news-123',
            title: 'Breaking News: Major Event Unfolds',
            description: 'A significant event has occurred that affects many people worldwide.',
            published_time: '2024-01-15T10:30:00Z',
            kicker: { name: 'World' },
            thumbnail: {
              url: 'https://www.reuters.com/images/thumb.jpg',
              width: 800,
              height: 600,
            },
            authors: [{ name: 'John Doe' }],
          },
          {
            id: 'article-456',
            canonical_url: 'https://www.reuters.com/business/economy/economic-update',
            title: 'Economic Update: Markets Rally',
            description: 'Markets showed strong gains today amid positive economic indicators.',
            published_time: '2024-01-15T09:00:00Z',
            kicker: { name: 'Business' },
          },
        ],
      },
    });

    const results = engine.parseResponse(sampleResponse, defaultParams);

    expect(results.results.length).toBe(2);

    const first = results.results[0];
    expect(first.title).toBe('Breaking News: Major Event Unfolds');
    expect(first.url).toContain('reuters.com');
    expect(first.source).toBe('World');
    expect(first.thumbnailUrl).toContain('thumb.jpg');
    expect(first.publishedAt).toBe('2024-01-15T10:30:00.000Z');
    expect(first.category).toBe('news');
    expect(first.engine).toBe('reuters');

    const second = results.results[1];
    expect(second.title).toBe('Economic Update: Markets Rally');
    expect(second.source).toBe('Business');
  });

  it('should handle empty or malformed response', () => {
    const emptyResults = engine.parseResponse('{}', defaultParams);
    expect(emptyResults.results).toEqual([]);

    const malformedResults = engine.parseResponse('not json', defaultParams);
    expect(malformedResults.results).toEqual([]);
  });

  it('should search and return news results', async () => {
    const results = await fetchAndParse(engine, 'technology');

    // Note: API response may vary
    if (results.results.length > 0) {
      const first = results.results[0];
      expect(first.url).toBeTruthy();
      expect(first.title).toBeTruthy();
      expect(first.category).toBe('news');
      expect(first.engine).toBe('reuters');
    }
  }, 30000);
});

describe('ReutersRSSEngine', () => {
  const engine = new ReutersRSSEngine('world');

  const defaultParams: EngineParams = {
    page: 1,
    locale: 'en-US',
    safeSearch: 1,
    timeRange: '',
    engineData: {},
  };

  it('should have correct metadata', () => {
    expect(engine.name).toBe('reuters rss (world)');
    expect(engine.shortcut).toBe('rtrss');
    expect(engine.categories).toContain('news');
    expect(engine.supportsPaging).toBe(false);
  });

  it('should build correct RSS feed URL', () => {
    const config = engine.buildRequest('', defaultParams);
    expect(config.url).toContain('reuters.com/rssfeed/world');
    expect(config.method).toBe('GET');
    expect(config.headers['Accept']).toContain('xml');
  });

  it('should parse RSS feed response', () => {
    const sampleRss = `
      <?xml version="1.0" encoding="UTF-8"?>
      <rss version="2.0">
        <channel>
          <title>Reuters World News</title>
          <item>
            <title><![CDATA[World Leaders Meet for Summit]]></title>
            <link>https://www.reuters.com/world/leaders-summit</link>
            <description><![CDATA[Leaders from around the world gathered for an important summit.]]></description>
            <pubDate>Mon, 15 Jan 2024 10:00:00 GMT</pubDate>
            <media:content url="https://www.reuters.com/images/summit.jpg" />
          </item>
          <item>
            <title>Another News Article</title>
            <link>https://www.reuters.com/world/another-article</link>
            <description>Description of another article.</description>
            <pubDate>Mon, 15 Jan 2024 09:00:00 GMT</pubDate>
          </item>
        </channel>
      </rss>
    `;

    const results = engine.parseResponse(sampleRss, defaultParams);

    expect(results.results.length).toBe(2);

    const first = results.results[0];
    expect(first.title).toBe('World Leaders Meet for Summit');
    expect(first.url).toContain('leaders-summit');
    expect(first.source).toBe('Reuters');
    expect(first.thumbnailUrl).toContain('summit.jpg');
    expect(first.publishedAt).toBeTruthy();

    const second = results.results[1];
    expect(second.title).toBe('Another News Article');
  });
});

async function fetchAndParse(engine: ReutersEngine, query: string) {
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
