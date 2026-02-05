import { describe, it, expect } from 'vitest';
import { BilibiliEngine } from './bilibili';

describe('BilibiliEngine', () => {
  const engine = new BilibiliEngine();

  describe('metadata', () => {
    it('should have correct name and shortcut', () => {
      expect(engine.name).toBe('bilibili');
      expect(engine.shortcut).toBe('bili');
    });

    it('should have correct categories', () => {
      expect(engine.categories).toContain('videos');
      expect(engine.categories.length).toBe(1);
    });

    it('should have correct paging settings', () => {
      expect(engine.supportsPaging).toBe(true);
      expect(engine.maxPage).toBe(10);
    });

    it('should have correct timeout and weight', () => {
      expect(engine.timeout).toBe(10000);
      expect(engine.weight).toBe(0.65);
    });

    it('should be enabled by default', () => {
      expect(engine.disabled).toBe(false);
    });
  });

  describe('buildRequest', () => {
    it('should build correct base URL', () => {
      const config = engine.buildRequest('test query', {
        page: 1,
        locale: 'en',
        safeSearch: 1,
        timeRange: '',
        engineData: {},
      });

      expect(config.url).toContain('https://api.bilibili.com/x/web-interface/search/type');
      expect(config.url).toContain('keyword=test+query');
      expect(config.url).toContain('search_type=video');
      expect(config.url).toContain('page=1');
      expect(config.url).toContain('pagesize=20');
      expect(config.method).toBe('GET');
      expect(config.headers['Accept']).toBe('application/json');
      expect(config.headers['Referer']).toBe('https://search.bilibili.com/');
    });

    it('should handle pagination correctly', () => {
      const page1 = engine.buildRequest('test', {
        page: 1,
        locale: 'en',
        safeSearch: 1,
        timeRange: '',
        engineData: {},
      });
      expect(page1.url).toContain('page=1');

      const page5 = engine.buildRequest('test', {
        page: 5,
        locale: 'en',
        safeSearch: 1,
        timeRange: '',
        engineData: {},
      });
      expect(page5.url).toContain('page=5');
    });

    it('should handle Chinese keywords', () => {
      const config = engine.buildRequest('javascript', {
        page: 1,
        locale: 'zh',
        safeSearch: 1,
        timeRange: '',
        engineData: {},
      });

      expect(config.url).toContain('keyword=javascript');
      expect(config.headers['Accept-Language']).toContain('zh');
    });
  });

  describe('parseResponse', () => {
    const defaultParams = {
      page: 1,
      locale: 'en',
      safeSearch: 1 as const,
      timeRange: '' as const,
      engineData: {},
    };

    it('should parse valid Bilibili API response', () => {
      const mockResponse = JSON.stringify({
        code: 0,
        message: '0',
        data: {
          result: [
            {
              aid: 123456789,
              bvid: 'BV1xx411c7mD',
              title: '<em class="keyword">JavaScript</em> Tutorial',
              description: 'Learn JavaScript programming',
              pic: '//i0.hdslb.com/bfs/archive/abc123.jpg',
              play: 50000,
              video_review: 1200,
              favorites: 3500,
              tag: 'programming,javascript,tutorial',
              duration: '15:30',
              author: 'TechChannel',
              mid: 987654,
              pubdate: 1704067200,
            },
            {
              aid: 987654321,
              bvid: 'BV1yy411c8nP',
              title: 'Advanced <em class="keyword">JavaScript</em>',
              description: 'Deep dive into JS',
              pic: '//i1.hdslb.com/bfs/archive/def456.jpg',
              play: 25000,
              video_review: 800,
              favorites: 2000,
              duration: '1:05:30',
              author: 'ProDev',
              mid: 123456,
              pubdate: 1706745600,
            },
          ],
          numResults: 100,
          numPages: 5,
          page: 1,
          pagesize: 20,
        },
      });

      const results = engine.parseResponse(mockResponse, defaultParams);

      expect(results.results.length).toBe(2);

      const first = results.results[0];
      expect(first.url).toBe('https://www.bilibili.com/video/BV1xx411c7mD');
      expect(first.title).toBe('JavaScript Tutorial');
      expect(first.content).toBe('Learn JavaScript programming');
      expect(first.thumbnailUrl).toBe('https://i0.hdslb.com/bfs/archive/abc123.jpg');
      expect(first.duration).toBe('15:30');
      expect(first.channel).toBe('TechChannel');
      expect(first.views).toBe(50000);
      expect(first.publishedAt).toBe('2024-01-01T00:00:00.000Z');
      expect(first.embedUrl).toBe('https://player.bilibili.com/player.html?bvid=BV1xx411c7mD');
      expect(first.engine).toBe('bilibili');
      expect(first.category).toBe('videos');
      expect(first.template).toBe('videos');
      expect(first.metadata).toEqual({
        bvid: 'BV1xx411c7mD',
        aid: 123456789,
        favorites: 3500,
        danmaku: 1200,
        tags: 'programming,javascript,tutorial',
      });

      const second = results.results[1];
      expect(second.title).toBe('Advanced JavaScript');
      expect(second.duration).toBe('1:05:30');
      expect(second.views).toBe(25000);
    });

    it('should fallback to aid when bvid is missing', () => {
      const mockResponse = JSON.stringify({
        code: 0,
        data: {
          result: [
            {
              aid: 123456789,
              title: 'Old Video',
              duration: '5:00',
              author: 'OldChannel',
            },
          ],
        },
      });

      const results = engine.parseResponse(mockResponse, defaultParams);

      expect(results.results.length).toBe(1);
      expect(results.results[0].url).toBe('https://www.bilibili.com/video/av123456789');
      expect(results.results[0].embedUrl).toBe('https://player.bilibili.com/player.html?aid=123456789');
    });

    it('should use arcurl when provided', () => {
      const mockResponse = JSON.stringify({
        code: 0,
        data: {
          result: [
            {
              bvid: 'BV1test',
              title: 'Test Video',
              arcurl: 'https://www.bilibili.com/video/BV1test?custom=param',
            },
          ],
        },
      });

      const results = engine.parseResponse(mockResponse, defaultParams);

      expect(results.results.length).toBe(1);
      expect(results.results[0].url).toBe('https://www.bilibili.com/video/BV1test?custom=param');
    });

    it('should clean HTML tags from title and description', () => {
      const mockResponse = JSON.stringify({
        code: 0,
        data: {
          result: [
            {
              bvid: 'BV1html',
              title: '<em class="keyword">Test</em> &amp; <em class="keyword">HTML</em>',
              description: '&lt;script&gt;alert(&quot;xss&quot;)&lt;/script&gt;',
            },
          ],
        },
      });

      const results = engine.parseResponse(mockResponse, defaultParams);

      expect(results.results.length).toBe(1);
      expect(results.results[0].title).toBe('Test & HTML');
      expect(results.results[0].content).toBe('alert("xss")');
    });

    it('should handle missing optional fields gracefully', () => {
      const mockResponse = JSON.stringify({
        code: 0,
        data: {
          result: [
            {
              bvid: 'BV1minimal',
            },
          ],
        },
      });

      const results = engine.parseResponse(mockResponse, defaultParams);

      expect(results.results.length).toBe(1);

      const first = results.results[0];
      expect(first.url).toBe('https://www.bilibili.com/video/BV1minimal');
      expect(first.title).toBe('');
      expect(first.content).toBe('');
      expect(first.thumbnailUrl).toBe('');
      expect(first.duration).toBe('');
      expect(first.channel).toBe('');
      expect(first.views).toBe(0);
      expect(first.publishedAt).toBe('');
    });

    it('should skip videos without bvid and aid', () => {
      const mockResponse = JSON.stringify({
        code: 0,
        data: {
          result: [
            {
              title: 'No ID Video',
            },
            {
              bvid: 'BV1valid',
              title: 'Valid Video',
            },
          ],
        },
      });

      const results = engine.parseResponse(mockResponse, defaultParams);

      expect(results.results.length).toBe(1);
      expect(results.results[0].title).toBe('Valid Video');
    });

    it('should return empty results for error response', () => {
      const mockResponse = JSON.stringify({
        code: -400,
        message: 'Request error',
        data: null,
      });

      const results = engine.parseResponse(mockResponse, defaultParams);
      expect(results.results).toEqual([]);
    });

    it('should return empty results for invalid JSON', () => {
      const results = engine.parseResponse('not valid json', defaultParams);
      expect(results.results).toEqual([]);
    });

    it('should return empty results when data.result is missing', () => {
      const mockResponse = JSON.stringify({
        code: 0,
        data: {},
      });

      const results = engine.parseResponse(mockResponse, defaultParams);
      expect(results.results).toEqual([]);
    });

    it('should return empty results when data.result is not an array', () => {
      const mockResponse = JSON.stringify({
        code: 0,
        data: {
          result: 'not an array',
        },
      });

      const results = engine.parseResponse(mockResponse, defaultParams);
      expect(results.results).toEqual([]);
    });

    it('should normalize duration correctly', () => {
      const mockResponse = JSON.stringify({
        code: 0,
        data: {
          result: [
            { bvid: 'v1', duration: '5:05' },
            { bvid: 'v2', duration: '0:45' },
            { bvid: 'v3', duration: '1:00:00' },
            { bvid: 'v4', duration: '2:05:30' },
          ],
        },
      });

      const results = engine.parseResponse(mockResponse, defaultParams);

      expect(results.results[0].duration).toBe('5:05');
      expect(results.results[1].duration).toBe('0:45');
      expect(results.results[2].duration).toBe('1:00:00');
      expect(results.results[3].duration).toBe('2:05:30');
    });

    it('should convert protocol-relative URLs to HTTPS', () => {
      const mockResponse = JSON.stringify({
        code: 0,
        data: {
          result: [
            {
              bvid: 'BV1test',
              pic: '//i0.hdslb.com/bfs/archive/test.jpg',
            },
          ],
        },
      });

      const results = engine.parseResponse(mockResponse, defaultParams);

      expect(results.results.length).toBe(1);
      expect(results.results[0].thumbnailUrl).toBe('https://i0.hdslb.com/bfs/archive/test.jpg');
    });

    it('should use senddate when pubdate is missing', () => {
      const mockResponse = JSON.stringify({
        code: 0,
        data: {
          result: [
            {
              bvid: 'BV1date',
              senddate: 1704067200,
            },
          ],
        },
      });

      const results = engine.parseResponse(mockResponse, defaultParams);

      expect(results.results.length).toBe(1);
      expect(results.results[0].publishedAt).toBe('2024-01-01T00:00:00.000Z');
    });
  });

  describe('live API test', () => {
    // Skipped by default - enable for manual testing
    // Note: Bilibili API may have rate limiting or geo-restrictions
    it.skip('should search and return video results', async () => {
      const params = {
        page: 1,
        locale: 'zh',
        safeSearch: 1 as const,
        timeRange: '' as const,
        engineData: {},
      };
      const config = engine.buildRequest('javascript', params);
      const res = await fetch(config.url, {
        headers: config.headers,
      });

      expect(res.ok).toBe(true);

      const body = await res.text();
      const results = engine.parseResponse(body, params);

      expect(results.results.length).toBeGreaterThan(0);

      const first = results.results[0];
      expect(first.url).toContain('bilibili.com/video/');
      expect(first.title).toBeTruthy();
      expect(first.engine).toBe('bilibili');
      expect(first.category).toBe('videos');
    }, 30000);
  });
});
