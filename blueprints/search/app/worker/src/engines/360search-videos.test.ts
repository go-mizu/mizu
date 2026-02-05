import { describe, it, expect } from 'vitest';
import { Search360VideosEngine } from './360search-videos';

describe('Search360VideosEngine', () => {
  const engine = new Search360VideosEngine();

  describe('metadata', () => {
    it('should have correct name and shortcut', () => {
      expect(engine.name).toBe('360search');
      expect(engine.shortcut).toBe('360v');
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
      expect(config.url).toContain('https://tv.360kan.com/v1/video/list');
      expect(config.url).toContain('q=test+query');
      expect(config.method).toBe('GET');
      expect(config.headers['Accept']).toBe('application/json');
    });

    it('should set start=0 for page 1', () => {
      const config = engine.buildRequest('test', defaultParams);
      expect(config.url).toContain('start=0');
      expect(config.url).toContain('count=10');
    });

    it('should calculate correct offset for page 2', () => {
      const config = engine.buildRequest('test', { ...defaultParams, page: 2 });
      expect(config.url).toContain('start=10');
    });

    it('should calculate correct offset for page 3', () => {
      const config = engine.buildRequest('test', { ...defaultParams, page: 3 });
      expect(config.url).toContain('start=20');
    });

    it('should calculate correct offset for page 5', () => {
      const config = engine.buildRequest('test', { ...defaultParams, page: 5 });
      expect(config.url).toContain('start=40');
    });

    it('should include Chinese language header', () => {
      const config = engine.buildRequest('test', defaultParams);
      expect(config.headers['Accept-Language']).toContain('zh-CN');
    });

    it('should properly encode Chinese queries', () => {
      const config = engine.buildRequest('电影', defaultParams);
      expect(config.url).toContain('q=%E7%94%B5%E5%BD%B1');
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
          list: [
            {
              title: '测试视频标题',
              description: '这是一个测试视频描述',
              play_url: 'https://www.360kan.com/video/12345',
              cover: 'https://p1.360kan.com/cover/12345.jpg',
              stream_url: 'https://stream.360kan.com/video/12345.m3u8',
              publish_time: 1704067200, // 2024-01-01 00:00:00 UTC
            },
            {
              title: '第二个视频',
              description: '另一个描述',
              play_url: 'https://www.360kan.com/video/67890',
              cover: 'https://p1.360kan.com/cover/67890.jpg',
              stream_url: 'https://stream.360kan.com/video/67890.m3u8',
              publish_time: 1706745600,
            },
          ],
        },
      });

      const results = engine.parseResponse(mockResponse, defaultParams);

      expect(results.results.length).toBe(2);

      const first = results.results[0];
      expect(first.url).toBe('https://www.360kan.com/video/12345');
      expect(first.title).toBe('测试视频标题');
      expect(first.content).toBe('这是一个测试视频描述');
      expect(first.thumbnailUrl).toBe('https://p1.360kan.com/cover/12345.jpg');
      expect(first.embedUrl).toBe(
        'https://stream.360kan.com/video/12345.m3u8'
      );
      expect(first.publishedAt).toBe('2024-01-01T00:00:00.000Z');
      expect(first.engine).toBe('360search');
      expect(first.category).toBe('videos');
      expect(first.template).toBe('videos');

      const second = results.results[1];
      expect(second.url).toBe('https://www.360kan.com/video/67890');
      expect(second.title).toBe('第二个视频');
    });

    it('should decode HTML entities in title and description', () => {
      const mockResponse = JSON.stringify({
        data: {
          list: [
            {
              title: 'Video &amp; Music &lt;2024&gt;',
              description: 'Description with &quot;quotes&quot; and &#39;apostrophes&#39;',
              play_url: 'https://www.360kan.com/video/1',
              cover: 'https://cover.jpg',
            },
          ],
        },
      });

      const results = engine.parseResponse(mockResponse, defaultParams);

      expect(results.results.length).toBe(1);
      expect(results.results[0].title).toBe('Video & Music <2024>');
      expect(results.results[0].content).toBe(
        'Description with "quotes" and \'apostrophes\''
      );
    });

    it('should handle missing optional fields gracefully', () => {
      const mockResponse = JSON.stringify({
        data: {
          list: [
            {
              play_url: 'https://www.360kan.com/video/minimal',
            },
          ],
        },
      });

      const results = engine.parseResponse(mockResponse, defaultParams);

      expect(results.results.length).toBe(1);
      const result = results.results[0];
      expect(result.url).toBe('https://www.360kan.com/video/minimal');
      expect(result.title).toBe('');
      expect(result.content).toBe('');
      expect(result.thumbnailUrl).toBe('');
      expect(result.embedUrl).toBe('');
      expect(result.publishedAt).toBe('');
    });

    it('should skip videos without play_url', () => {
      const mockResponse = JSON.stringify({
        data: {
          list: [
            {
              title: 'No URL Video',
              description: 'This should be skipped',
            },
            {
              title: 'Valid Video',
              play_url: 'https://www.360kan.com/video/valid',
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

    it('should return empty results for missing list field', () => {
      const mockResponse = JSON.stringify({
        data: {
          total: 0,
        },
      });

      const results = engine.parseResponse(mockResponse, defaultParams);
      expect(results.results).toEqual([]);
    });

    it('should return empty results for null list', () => {
      const mockResponse = JSON.stringify({
        data: {
          list: null,
        },
      });

      const results = engine.parseResponse(mockResponse, defaultParams);
      expect(results.results).toEqual([]);
    });

    it('should convert unix timestamp to ISO date correctly', () => {
      const testCases = [
        { timestamp: 1704067200, expected: '2024-01-01T00:00:00.000Z' },
        { timestamp: 1609459200, expected: '2021-01-01T00:00:00.000Z' },
        { timestamp: 0, expected: '1970-01-01T00:00:00.000Z' },
      ];

      for (const { timestamp, expected } of testCases) {
        const mockResponse = JSON.stringify({
          data: {
            list: [
              {
                play_url: 'https://test.com/video',
                publish_time: timestamp,
              },
            ],
          },
        });

        const results = engine.parseResponse(mockResponse, defaultParams);
        expect(results.results[0].publishedAt).toBe(expected);
      }
    });

    it('should handle non-numeric publish_time gracefully', () => {
      const mockResponse = JSON.stringify({
        data: {
          list: [
            {
              play_url: 'https://test.com/video',
              publish_time: 'invalid',
            },
          ],
        },
      });

      const results = engine.parseResponse(mockResponse, defaultParams);
      expect(results.results[0].publishedAt).toBe('');
    });

    it('should set correct score from engine weight', () => {
      const mockResponse = JSON.stringify({
        data: {
          list: [
            {
              play_url: 'https://test.com/video',
              title: 'Test',
            },
          ],
        },
      });

      const results = engine.parseResponse(mockResponse, defaultParams);
      expect(results.results[0].score).toBe(0.6);
    });
  });

  describe('live API test', () => {
    it('should search and handle response (may be empty for non-Chinese queries)', async () => {
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
          expect(first.engine).toBe('360search');
          expect(first.category).toBe('videos');
        }
      } else {
        // API not accessible, that's OK for a Chinese service
        expect(res.status).toBeGreaterThan(0);
      }
    }, 30000);
  });
});
