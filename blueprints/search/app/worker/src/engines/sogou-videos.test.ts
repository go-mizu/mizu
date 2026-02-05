import { describe, it, expect } from 'vitest';
import { SogouVideosEngine } from './sogou-videos';

describe('SogouVideosEngine', () => {
  const engine = new SogouVideosEngine();

  describe('metadata', () => {
    it('should have correct name and shortcut', () => {
      expect(engine.name).toBe('sogou');
      expect(engine.shortcut).toBe('sgv');
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
      expect(engine.timeout).toBe(8000);
      expect(engine.weight).toBe(0.6);
    });

    it('should be enabled by default', () => {
      expect(engine.disabled).toBe(false);
    });
  });

  describe('buildRequest', () => {
    const defaultParams = {
      page: 1,
      locale: 'zh-CN',
      safeSearch: 1 as const,
      timeRange: '' as const,
      engineData: {},
    };

    it('should build correct base URL', () => {
      const config = engine.buildRequest('test query', defaultParams);
      expect(config.url).toContain(
        'https://v.sogou.com/api/video/shortVideoV2'
      );
      expect(config.url).toContain('query=test+query');
      expect(config.method).toBe('GET');
      expect(config.headers['Accept']).toBe('application/json');
    });

    it('should set page=1 for page 1', () => {
      const config = engine.buildRequest('test', defaultParams);
      expect(config.url).toContain('page=1');
      expect(config.url).toContain('pagesize=10');
    });

    it('should set correct page for page 2', () => {
      const config = engine.buildRequest('test', { ...defaultParams, page: 2 });
      expect(config.url).toContain('page=2');
    });

    it('should set correct page for page 3', () => {
      const config = engine.buildRequest('test', { ...defaultParams, page: 3 });
      expect(config.url).toContain('page=3');
    });

    it('should set correct page for page 5', () => {
      const config = engine.buildRequest('test', { ...defaultParams, page: 5 });
      expect(config.url).toContain('page=5');
    });

    it('should include Chinese language header', () => {
      const config = engine.buildRequest('test', defaultParams);
      expect(config.headers['Accept-Language']).toContain('zh-CN');
    });

    it('should properly encode Chinese queries', () => {
      const config = engine.buildRequest('电影', defaultParams);
      expect(config.url).toContain('query=%E7%94%B5%E5%BD%B1');
    });
  });

  describe('parseResponse', () => {
    const defaultParams = {
      page: 1,
      locale: 'zh-CN',
      safeSearch: 1 as const,
      timeRange: '' as const,
      engineData: {},
    };

    it('should parse valid API response', () => {
      const mockResponse = JSON.stringify({
        data: {
          listData: [
            {
              title: '测试视频标题',
              url: 'https://v.sogou.com/video/12345',
              pic: 'https://img.sogou.com/cover/12345.jpg',
              duration: '05:30',
              date: '2024-01-01',
              site: '优酷',
            },
            {
              title: '第二个视频',
              url: 'https://v.sogou.com/video/67890',
              pic: 'https://img.sogou.com/cover/67890.jpg',
              duration: '10:15',
              date: '2024-02-15',
              site: '爱奇艺',
            },
          ],
        },
      });

      const results = engine.parseResponse(mockResponse, defaultParams);

      expect(results.results.length).toBe(2);

      const first = results.results[0];
      expect(first.url).toBe('https://v.sogou.com/video/12345');
      expect(first.title).toBe('测试视频标题');
      expect(first.content).toBe('优酷');
      expect(first.thumbnailUrl).toBe('https://img.sogou.com/cover/12345.jpg');
      expect(first.duration).toBe('05:30');
      expect(first.publishedAt).toBe('2024-01-01');
      expect(first.engine).toBe('sogou');
      expect(first.category).toBe('videos');
      expect(first.template).toBe('videos');

      const second = results.results[1];
      expect(second.url).toBe('https://v.sogou.com/video/67890');
      expect(second.title).toBe('第二个视频');
      expect(second.content).toBe('爱奇艺');
    });

    it('should handle relative URLs by prepending base URL', () => {
      const mockResponse = JSON.stringify({
        data: {
          listData: [
            {
              title: 'Relative URL Video',
              url: '/path/to/video',
              pic: 'https://img.sogou.com/cover.jpg',
              duration: '03:00',
              date: '2024-03-01',
              site: 'Bilibili',
            },
          ],
        },
      });

      const results = engine.parseResponse(mockResponse, defaultParams);

      expect(results.results.length).toBe(1);
      expect(results.results[0].url).toBe('https://v.sogou.com/path/to/video');
    });

    it('should handle relative URLs without leading slash', () => {
      const mockResponse = JSON.stringify({
        data: {
          listData: [
            {
              title: 'No Slash Video',
              url: 'video/12345',
              pic: 'https://img.sogou.com/cover.jpg',
              duration: '02:00',
              date: '2024-04-01',
              site: 'Tencent',
            },
          ],
        },
      });

      const results = engine.parseResponse(mockResponse, defaultParams);

      expect(results.results.length).toBe(1);
      expect(results.results[0].url).toBe('https://v.sogou.com/video/12345');
    });

    it('should not modify absolute URLs', () => {
      const mockResponse = JSON.stringify({
        data: {
          listData: [
            {
              title: 'Absolute URL Video',
              url: 'https://www.example.com/video/12345',
              pic: 'https://img.example.com/cover.jpg',
              duration: '04:00',
              date: '2024-05-01',
              site: 'Example',
            },
          ],
        },
      });

      const results = engine.parseResponse(mockResponse, defaultParams);

      expect(results.results.length).toBe(1);
      expect(results.results[0].url).toBe(
        'https://www.example.com/video/12345'
      );
    });

    it('should decode HTML entities in title', () => {
      const mockResponse = JSON.stringify({
        data: {
          listData: [
            {
              title: 'Video &amp; Music &lt;2024&gt;',
              url: 'https://v.sogou.com/video/1',
              pic: 'https://cover.jpg',
              duration: '01:00',
              date: '2024-01-01',
              site: 'Site',
            },
          ],
        },
      });

      const results = engine.parseResponse(mockResponse, defaultParams);

      expect(results.results.length).toBe(1);
      expect(results.results[0].title).toBe('Video & Music <2024>');
    });

    it('should decode numeric HTML entities', () => {
      const mockResponse = JSON.stringify({
        data: {
          listData: [
            {
              title: 'Quotes &quot;here&quot; and &#39;apostrophes&#39;',
              url: 'https://v.sogou.com/video/1',
              pic: 'https://cover.jpg',
            },
          ],
        },
      });

      const results = engine.parseResponse(mockResponse, defaultParams);

      expect(results.results[0].title).toBe(
        'Quotes "here" and \'apostrophes\''
      );
    });

    it('should handle missing optional fields gracefully', () => {
      const mockResponse = JSON.stringify({
        data: {
          listData: [
            {
              url: 'https://v.sogou.com/video/minimal',
            },
          ],
        },
      });

      const results = engine.parseResponse(mockResponse, defaultParams);

      expect(results.results.length).toBe(1);
      const result = results.results[0];
      expect(result.url).toBe('https://v.sogou.com/video/minimal');
      expect(result.title).toBe('');
      expect(result.content).toBe('');
      expect(result.thumbnailUrl).toBe('');
      expect(result.duration).toBe('');
      expect(result.publishedAt).toBe('');
    });

    it('should skip videos without url', () => {
      const mockResponse = JSON.stringify({
        data: {
          listData: [
            {
              title: 'No URL Video',
              pic: 'https://cover.jpg',
            },
            {
              title: 'Valid Video',
              url: 'https://v.sogou.com/video/valid',
            },
          ],
        },
      });

      const results = engine.parseResponse(mockResponse, defaultParams);

      expect(results.results.length).toBe(1);
      expect(results.results[0].title).toBe('Valid Video');
    });

    it('should return empty results for invalid JSON', () => {
      const results = engine.parseResponse('not valid json', defaultParams);
      expect(results.results).toEqual([]);
    });

    it('should return empty results for missing data field', () => {
      const mockResponse = JSON.stringify({
        status: 'ok',
      });

      const results = engine.parseResponse(mockResponse, defaultParams);
      expect(results.results).toEqual([]);
    });

    it('should return empty results for missing listData field', () => {
      const mockResponse = JSON.stringify({
        data: {
          total: 0,
        },
      });

      const results = engine.parseResponse(mockResponse, defaultParams);
      expect(results.results).toEqual([]);
    });

    it('should return empty results for null listData', () => {
      const mockResponse = JSON.stringify({
        data: {
          listData: null,
        },
      });

      const results = engine.parseResponse(mockResponse, defaultParams);
      expect(results.results).toEqual([]);
    });

    it('should set correct score from engine weight', () => {
      const mockResponse = JSON.stringify({
        data: {
          listData: [
            {
              url: 'https://v.sogou.com/video/test',
              title: 'Test',
            },
          ],
        },
      });

      const results = engine.parseResponse(mockResponse, defaultParams);
      expect(results.results[0].score).toBe(0.6);
    });

    it('should preserve duration format as-is', () => {
      const testCases = [
        { duration: '05:30', expected: '05:30' },
        { duration: '1:23:45', expected: '1:23:45' },
        { duration: '00:45', expected: '00:45' },
        { duration: '2:00:00', expected: '2:00:00' },
      ];

      for (const { duration, expected } of testCases) {
        const mockResponse = JSON.stringify({
          data: {
            listData: [
              {
                url: 'https://v.sogou.com/video/test',
                duration,
              },
            ],
          },
        });

        const results = engine.parseResponse(mockResponse, defaultParams);
        expect(results.results[0].duration).toBe(expected);
      }
    });
  });

  describe('live API test', () => {
    it('should search and handle response (may be empty or error for non-Chinese queries)', async () => {
      const params = {
        page: 1,
        locale: 'zh-CN',
        safeSearch: 1 as const,
        timeRange: '' as const,
        engineData: {},
      };

      // Use a Chinese query for better results
      const config = engine.buildRequest('电影', params);

      const res = await fetch(config.url, {
        headers: config.headers,
      });

      // The API may not be accessible or may return errors
      // from outside China, so we just verify we can make the request
      // and parse whatever response we get
      if (res.ok) {
        const body = await res.text();
        const results = engine.parseResponse(body, params);

        // Results array exists (may be empty)
        expect(Array.isArray(results.results)).toBe(true);

        // If we got results, verify structure
        if (results.results.length > 0) {
          const first = results.results[0];
          expect(first.url).toBeTruthy();
          expect(first.engine).toBe('sogou');
          expect(first.category).toBe('videos');
        }
      } else {
        // API not accessible, that's OK for a Chinese service
        expect(res.status).toBeGreaterThan(0);
      }
    }, 30000);
  });
});
