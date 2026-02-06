import { describe, it, expect } from 'vitest';
import { DuckDuckGoNewsEngine } from './duckduckgo-news';
import type { EngineParams } from './engine';

describe('DuckDuckGoNewsEngine', () => {
  const engine = new DuckDuckGoNewsEngine();

  const defaultParams: EngineParams = {
    page: 1,
    locale: 'us-en',
    safeSearch: 1,
    timeRange: '' as const,
    engineData: {},
  };

  describe('metadata', () => {
    it('should have correct name', () => {
      expect(engine.name).toBe('duckduckgo news');
    });

    it('should have correct shortcut', () => {
      expect(engine.shortcut).toBe('ddgn');
    });

    it('should include news category', () => {
      expect(engine.categories).toContain('news');
    });

    it('should support paging', () => {
      expect(engine.supportsPaging).toBe(true);
    });
  });

  describe('buildRequest', () => {
    it('should build a URL starting with the DDG news endpoint', () => {
      const config = engine.buildRequest('ai news', defaultParams);
      expect(config.url).toMatch(/^https:\/\/duckduckgo\.com\/news\.js/);
      expect(config.method).toBe('GET');
    });

    it('should include the query parameter', () => {
      const config = engine.buildRequest('ai news', defaultParams);
      expect(config.url).toContain('q=ai+news');
    });

    it('should include locale parameter', () => {
      const config = engine.buildRequest('test', defaultParams);
      expect(config.url).toContain('l=us-en');
    });

    it('should include JSON output parameter', () => {
      const config = engine.buildRequest('test', defaultParams);
      expect(config.url).toContain('o=json');
    });

    it('should handle pagination with offset parameter', () => {
      const config = engine.buildRequest('test', {
        ...defaultParams,
        page: 3,
      });
      // offset = (3-1) * 30 = 60
      expect(config.url).toContain('s=60');
    });

    it('should not include offset for page 1', () => {
      const config = engine.buildRequest('test', defaultParams);
      expect(config.url).not.toContain('s=');
    });

    it('should set safe search parameter', () => {
      const configOff = engine.buildRequest('test', {
        ...defaultParams,
        safeSearch: 0,
      });
      expect(configOff.url).toContain('kp=-2');

      const configStrict = engine.buildRequest('test', {
        ...defaultParams,
        safeSearch: 2,
      });
      expect(configStrict.url).toContain('kp=1');

      const configModerate = engine.buildRequest('test', {
        ...defaultParams,
        safeSearch: 1,
      });
      expect(configModerate.url).toContain('kp=-1');
    });

    it('should apply time range filter', () => {
      const configDay = engine.buildRequest('test', {
        ...defaultParams,
        timeRange: 'day',
      });
      expect(configDay.url).toContain('df=d');

      const configWeek = engine.buildRequest('test', {
        ...defaultParams,
        timeRange: 'week',
      });
      expect(configWeek.url).toContain('df=w');

      const configMonth = engine.buildRequest('test', {
        ...defaultParams,
        timeRange: 'month',
      });
      expect(configMonth.url).toContain('df=m');
    });

    it('should include proper headers', () => {
      const config = engine.buildRequest('test', defaultParams);
      expect(config.headers['User-Agent']).toBeTruthy();
      expect(config.headers['Accept']).toContain('application/json');
    });
  });

  describe('parseResponse', () => {
    it('should correctly parse a results array', () => {
      const body = JSON.stringify({
        results: [
          {
            date: 1706918400,
            excerpt: 'Article excerpt here...',
            image: 'https://example.com/image.jpg',
            relative_time: '5 hours ago',
            source: 'TechCrunch',
            title: 'AI News: Major Breakthrough',
            url: 'https://example.com/article',
          },
          {
            date: 1706832000,
            excerpt: 'Second article excerpt.',
            image: 'https://example.com/image2.jpg',
            relative_time: '1 day ago',
            source: 'BBC',
            title: 'Global Update',
            url: 'https://example.com/article2',
          },
        ],
      });

      const results = engine.parseResponse(body, defaultParams);

      expect(results.results.length).toBe(2);

      const first = results.results[0];
      expect(first.url).toBe('https://example.com/article');
      expect(first.title).toBe('AI News: Major Breakthrough');
      expect(first.engine).toBe('duckduckgo news');

      const second = results.results[1];
      expect(second.url).toBe('https://example.com/article2');
      expect(second.title).toBe('Global Update');
    });

    it('should convert Unix timestamp (date * 1000) to ISO string', () => {
      const timestamp = 1706918400; // 2024-02-03T00:00:00.000Z
      const body = JSON.stringify({
        results: [
          {
            date: timestamp,
            excerpt: 'Test',
            source: 'Test Source',
            title: 'Test Title',
            url: 'https://example.com/test',
          },
        ],
      });

      const results = engine.parseResponse(body, defaultParams);

      expect(results.results.length).toBe(1);
      const expectedISO = new Date(timestamp * 1000).toISOString();
      expect(results.results[0].publishedAt).toBe(expectedISO);
    });

    it('should extract source from the response', () => {
      const body = JSON.stringify({
        results: [
          {
            date: 1706918400,
            excerpt: 'Some content',
            source: 'TechCrunch',
            title: 'Article Title',
            url: 'https://example.com/article',
          },
        ],
      });

      const results = engine.parseResponse(body, defaultParams);
      expect(results.results[0].source).toBe('TechCrunch');
    });

    it('should extract excerpt as content', () => {
      const body = JSON.stringify({
        results: [
          {
            date: 1706918400,
            excerpt: 'This is the article excerpt content.',
            source: 'Source',
            title: 'Title',
            url: 'https://example.com/article',
          },
        ],
      });

      const results = engine.parseResponse(body, defaultParams);
      expect(results.results[0].content).toBe(
        'This is the article excerpt content.'
      );
    });

    it('should extract image as thumbnailUrl', () => {
      const body = JSON.stringify({
        results: [
          {
            date: 1706918400,
            excerpt: 'Test',
            image: 'https://example.com/thumb.jpg',
            source: 'Source',
            title: 'Title',
            url: 'https://example.com/article',
          },
        ],
      });

      const results = engine.parseResponse(body, defaultParams);
      expect(results.results[0].thumbnailUrl).toBe(
        'https://example.com/thumb.jpg'
      );
    });

    it('should set category to news and template to news', () => {
      const body = JSON.stringify({
        results: [
          {
            date: 1706918400,
            excerpt: 'Test',
            source: 'Source',
            title: 'Title',
            url: 'https://example.com/article',
          },
        ],
      });

      const results = engine.parseResponse(body, defaultParams);
      expect(results.results[0].category).toBe('news');
      expect(results.results[0].template).toBe('news');
    });

    it('should handle missing optional fields gracefully', () => {
      const body = JSON.stringify({
        results: [
          {
            date: 0,
            excerpt: '',
            source: '',
            title: 'Minimal Article',
            url: 'https://example.com/minimal',
          },
        ],
      });

      const results = engine.parseResponse(body, defaultParams);

      expect(results.results.length).toBe(1);
      expect(results.results[0].title).toBe('Minimal Article');
      expect(results.results[0].url).toBe('https://example.com/minimal');
      expect(results.results[0].content).toBe('');
      expect(results.results[0].source).toBe('');
      // image is undefined, so thumbnailUrl should be undefined
      expect(results.results[0].thumbnailUrl).toBeUndefined();
    });

    it('should skip results missing url or title', () => {
      const body = JSON.stringify({
        results: [
          {
            date: 1706918400,
            excerpt: 'No url',
            source: 'Source',
            title: 'Has Title',
            url: '',
          },
          {
            date: 1706918400,
            excerpt: 'No title',
            source: 'Source',
            title: '',
            url: 'https://example.com/article',
          },
          {
            date: 1706918400,
            excerpt: 'Valid',
            source: 'Source',
            title: 'Valid Title',
            url: 'https://example.com/valid',
          },
        ],
      });

      const results = engine.parseResponse(body, defaultParams);
      expect(results.results.length).toBe(1);
      expect(results.results[0].title).toBe('Valid Title');
    });

    it('should handle empty response with no results array', () => {
      const emptyResults = engine.parseResponse('{}', defaultParams);
      expect(emptyResults.results).toEqual([]);
    });

    it('should handle empty results array', () => {
      const body = JSON.stringify({ results: [] });
      const results = engine.parseResponse(body, defaultParams);
      expect(results.results).toEqual([]);
    });

    it('should handle malformed JSON gracefully', () => {
      const results = engine.parseResponse('not valid json{{{', defaultParams);
      expect(results.results).toEqual([]);
    });

    it('should handle completely empty body', () => {
      const results = engine.parseResponse('', defaultParams);
      expect(results.results).toEqual([]);
    });
  });
});
