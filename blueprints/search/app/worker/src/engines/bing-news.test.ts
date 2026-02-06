import { describe, it, expect } from 'vitest';
import { BingNewsEngine } from './bing';
import type { EngineParams } from './engine';

describe('BingNewsEngine', () => {
  const engine = new BingNewsEngine();

  const defaultParams: EngineParams = {
    page: 1,
    locale: 'en-US',
    safeSearch: 1,
    timeRange: '',
    engineData: {},
  };

  describe('metadata', () => {
    it('should have correct name', () => {
      expect(engine.name).toBe('bing news');
    });

    it('should have correct shortcut', () => {
      expect(engine.shortcut).toBe('bn');
    });

    it('should include news category', () => {
      expect(engine.categories).toContain('news');
    });

    it('should support paging', () => {
      expect(engine.supportsPaging).toBe(true);
    });
  });

  describe('buildRequest', () => {
    it('should build a URL containing bing.com/news', () => {
      const config = engine.buildRequest('breaking news', defaultParams);
      expect(config.url).toContain('bing.com/news');
      expect(config.method).toBe('GET');
    });

    it('should set first param for pagination', () => {
      const config = engine.buildRequest('test', {
        ...defaultParams,
        page: 3,
      });
      // first = (3-1) * 10 + 1 = 21
      const url = new URL(config.url);
      expect(url.searchParams.get('first')).toBe('21');
    });

    it('should set first=1 on page 1', () => {
      const config = engine.buildRequest('test', defaultParams);
      const url = new URL(config.url);
      expect(url.searchParams.get('first')).toBe('1');
    });

    it('should include query parameter', () => {
      const config = engine.buildRequest('ai technology', defaultParams);
      const url = new URL(config.url);
      expect(url.searchParams.get('q')).toBe('ai technology');
    });

    it('should apply time range filter', () => {
      const config = engine.buildRequest('test', {
        ...defaultParams,
        timeRange: 'day',
      });
      expect(config.url).toContain('qft=');
    });
  });

  describe('parseResponse', () => {
    const sampleHtml = `
      <div class="newsitem">
        <a href="https://example.com/article1" class="title">Breaking: AI Advances</a>
        <div class="snippet">AI technology continues to advance at a rapid pace.</div>
        <div class="source" data-author="John Smith">
          <span>TechCrunch</span>
          <span class="time">5 hours ago</span>
        </div>
        <img src="https://th.bing.com/thumbnail1.jpg" />
      </div>
      <div class="newsitem">
        <a href="https://example.com/article2">Second Story</a>
        <div class="summary">Summary of second story.</div>
        <div class="source"><span>CNN</span></div>
        <img src="https://th.bing.com/thumbnail2.jpg" />
      </div>
    `;

    it('should return correct number of results', () => {
      const results = engine.parseResponse(sampleHtml, defaultParams);
      expect(results.results.length).toBe(2);
    });

    it('should extract URLs correctly', () => {
      const results = engine.parseResponse(sampleHtml, defaultParams);
      expect(results.results[0].url).toBe('https://example.com/article1');
      expect(results.results[1].url).toBe('https://example.com/article2');
    });

    it('should extract titles correctly', () => {
      const results = engine.parseResponse(sampleHtml, defaultParams);
      expect(results.results[0].title).toBe('Breaking: AI Advances');
      expect(results.results[1].title).toBe('Second Story');
    });

    it('should extract content from snippet or summary', () => {
      const results = engine.parseResponse(sampleHtml, defaultParams);
      expect(results.results[0].content).toContain('AI technology continues to advance');
      expect(results.results[1].content).toContain('Summary of second story');
    });

    it('should extract source names', () => {
      const results = engine.parseResponse(sampleHtml, defaultParams);
      expect(results.results[0].source).toBeTruthy();
      expect(results.results[1].source).toBeTruthy();
    });

    it('should extract publishedAt as ISO date string when relative time is found', () => {
      const results = engine.parseResponse(sampleHtml, defaultParams);
      const first = results.results[0];
      expect(first.publishedAt).toBeTruthy();
      // Should be a valid ISO date string
      const date = new Date(first.publishedAt!);
      expect(date.toISOString()).toBe(first.publishedAt);
    });

    it('should extract thumbnailUrl starting with http', () => {
      const results = engine.parseResponse(sampleHtml, defaultParams);
      expect(results.results[0].thumbnailUrl).toMatch(/^https?:\/\//);
      expect(results.results[0].thumbnailUrl).toContain('thumbnail1.jpg');
      expect(results.results[1].thumbnailUrl).toMatch(/^https?:\/\//);
      expect(results.results[1].thumbnailUrl).toContain('thumbnail2.jpg');
    });

    it('should extract author from data-author attribute when present', () => {
      const results = engine.parseResponse(sampleHtml, defaultParams);
      expect(results.results[0].author).toBe('John Smith');
    });

    it('should not have author when data-author is absent', () => {
      const results = engine.parseResponse(sampleHtml, defaultParams);
      expect(results.results[1].author).toBeUndefined();
    });

    it('should set category to news for all results', () => {
      const results = engine.parseResponse(sampleHtml, defaultParams);
      for (const result of results.results) {
        expect(result.category).toBe('news');
      }
    });

    it('should set engine name to bing news for all results', () => {
      const results = engine.parseResponse(sampleHtml, defaultParams);
      for (const result of results.results) {
        expect(result.engine).toBe('bing news');
      }
    });

    it('should handle news-card class elements', () => {
      const cardHtml = `
        <div class="news-card">
          <a href="https://example.com/card-article">Card News Title</a>
          <div class="snippet">Card news snippet content.</div>
          <div class="source"><span>BBC</span></div>
          <img src="https://th.bing.com/card-thumb.jpg" />
        </div>
      `;

      const results = engine.parseResponse(cardHtml, defaultParams);
      expect(results.results.length).toBe(1);
      expect(results.results[0].url).toBe('https://example.com/card-article');
      expect(results.results[0].title).toBe('Card News Title');
    });

    it('should return empty results for empty HTML', () => {
      const results = engine.parseResponse('', defaultParams);
      expect(results.results).toEqual([]);
    });

    it('should return empty results for HTML with no news items', () => {
      const results = engine.parseResponse('<div>No news here</div>', defaultParams);
      expect(results.results).toEqual([]);
    });
  });
});
