import { describe, it, expect } from 'vitest';
import { OdyseeEngine } from './odysee';

describe('OdyseeEngine', () => {
  const engine = new OdyseeEngine();

  describe('metadata', () => {
    it('should have correct name and shortcut', () => {
      expect(engine.name).toBe('odysee');
      expect(engine.shortcut).toBe('od');
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
      expect(engine.weight).toBe(0.75);
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

      expect(config.url).toContain('https://lighthouse.odysee.tv/search');
      expect(config.url).toContain('s=test+query');
      expect(config.url).toContain('size=20');
      expect(config.url).toContain('from=0');
      expect(config.url).toContain('mediaType=video');
      expect(config.url).toContain('free_only=true');
      expect(config.method).toBe('GET');
      expect(config.headers['Accept']).toBe('application/json');
    });

    it('should calculate correct offset for pagination', () => {
      const page1 = engine.buildRequest('test', {
        page: 1,
        locale: 'en',
        safeSearch: 1,
        timeRange: '',
        engineData: {},
      });
      expect(page1.url).toContain('from=0');

      const page2 = engine.buildRequest('test', {
        page: 2,
        locale: 'en',
        safeSearch: 1,
        timeRange: '',
        engineData: {},
      });
      expect(page2.url).toContain('from=20');

      const page5 = engine.buildRequest('test', {
        page: 5,
        locale: 'en',
        safeSearch: 1,
        timeRange: '',
        engineData: {},
      });
      expect(page5.url).toContain('from=80');
    });

    it('should set nsfw=false for safe search enabled', () => {
      const config = engine.buildRequest('test', {
        page: 1,
        locale: 'en',
        safeSearch: 1,
        timeRange: '',
        engineData: {},
      });

      expect(config.url).toContain('nsfw=false');
    });

    it('should set nsfw=false for strict safe search', () => {
      const config = engine.buildRequest('test', {
        page: 1,
        locale: 'en',
        safeSearch: 2,
        timeRange: '',
        engineData: {},
      });

      expect(config.url).toContain('nsfw=false');
    });

    it('should not set nsfw filter for safe search off', () => {
      const config = engine.buildRequest('test', {
        page: 1,
        locale: 'en',
        safeSearch: 0,
        timeRange: '',
        engineData: {},
      });

      expect(config.url).not.toContain('nsfw=');
    });

    it('should add release_time filter for day time range', () => {
      const config = engine.buildRequest('test', {
        page: 1,
        locale: 'en',
        safeSearch: 1,
        timeRange: 'day',
        engineData: {},
      });

      expect(config.url).toContain('release_time=');
      // Verify it contains a timestamp greater than constraint
      expect(config.url).toMatch(/release_time=%3E\d+/);
    });

    it('should add release_time filter for week time range', () => {
      const config = engine.buildRequest('test', {
        page: 1,
        locale: 'en',
        safeSearch: 1,
        timeRange: 'week',
        engineData: {},
      });

      expect(config.url).toContain('release_time=');
    });

    it('should add release_time filter for month time range', () => {
      const config = engine.buildRequest('test', {
        page: 1,
        locale: 'en',
        safeSearch: 1,
        timeRange: 'month',
        engineData: {},
      });

      expect(config.url).toContain('release_time=');
    });

    it('should add release_time filter for year time range', () => {
      const config = engine.buildRequest('test', {
        page: 1,
        locale: 'en',
        safeSearch: 1,
        timeRange: 'year',
        engineData: {},
      });

      expect(config.url).toContain('release_time=');
    });

    it('should not add release_time filter for empty time range', () => {
      const config = engine.buildRequest('test', {
        page: 1,
        locale: 'en',
        safeSearch: 1,
        timeRange: '',
        engineData: {},
      });

      expect(config.url).not.toContain('release_time=');
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

    it('should parse Lighthouse API response', () => {
      const mockResponse = JSON.stringify([
        {
          claimId: 'abc123def456',
          name: 'my-test-video',
          title: 'My Test Video Title',
          description: 'A description of my test video',
          thumbnail_url: 'https://thumbnails.odycdn.com/abc123.jpg',
          duration: 305,
          channel: '@TestChannel',
          release_time: 1704067200,
          effective_amount: 5000,
        },
        {
          claimId: 'xyz789ghi012',
          name: 'another-video',
          title: 'Another Great Video',
          description: 'Second video description',
          thumbnail_url: 'https://thumbnails.odycdn.com/xyz789.jpg',
          duration: 3661,
          channel: '@AnotherChannel',
          release_time: 1706745600,
          effective_amount: 10000,
        },
      ]);

      const results = engine.parseResponse(mockResponse, defaultParams);

      expect(results.results.length).toBe(2);

      const first = results.results[0];
      expect(first.url).toContain('odysee.com');
      expect(first.url).toContain('my-test-video');
      expect(first.title).toBe('My Test Video Title');
      expect(first.content).toBe('A description of my test video');
      expect(first.thumbnailUrl).toBe('https://thumbnails.odycdn.com/abc123.jpg');
      expect(first.duration).toBe('5:05');
      expect(first.channel).toBe('@TestChannel');
      expect(first.views).toBe(5000);
      expect(first.publishedAt).toBe('2024-01-01T00:00:00.000Z');
      expect(first.embedUrl).toContain('$/embed/my-test-video/abc123def456');
      expect(first.engine).toBe('odysee');
      expect(first.category).toBe('videos');
      expect(first.template).toBe('videos');

      const second = results.results[1];
      expect(second.title).toBe('Another Great Video');
      expect(second.duration).toBe('1:01:01');
      expect(second.views).toBe(10000);
    });

    it('should parse nested value structure', () => {
      const mockResponse = JSON.stringify([
        {
          claimId: 'nested123',
          name: 'nested-video',
          value: {
            title: 'Nested Title',
            description: 'Nested Description',
            thumbnail: {
              url: 'https://thumbnails.odycdn.com/nested.jpg',
            },
            video: {
              duration: 120,
            },
            release_time: 1704067200,
          },
          signing_channel: {
            name: '@NestedChannel',
          },
        },
      ]);

      const results = engine.parseResponse(mockResponse, defaultParams);

      expect(results.results.length).toBe(1);

      const first = results.results[0];
      expect(first.title).toBe('Nested Title');
      expect(first.content).toBe('Nested Description');
      expect(first.thumbnailUrl).toBe('https://thumbnails.odycdn.com/nested.jpg');
      expect(first.duration).toBe('2:00');
      expect(first.channel).toBe('NestedChannel');
    });

    it('should fallback to name for title when not provided', () => {
      const mockResponse = JSON.stringify([
        {
          claimId: 'notitle123',
          name: 'video-without-title',
        },
      ]);

      const results = engine.parseResponse(mockResponse, defaultParams);

      expect(results.results.length).toBe(1);
      expect(results.results[0].title).toBe('video without title');
    });

    it('should handle missing optional fields gracefully', () => {
      const mockResponse = JSON.stringify([
        {
          claimId: 'minimal123',
          name: 'minimal-video',
        },
      ]);

      const results = engine.parseResponse(mockResponse, defaultParams);

      expect(results.results.length).toBe(1);

      const first = results.results[0];
      expect(first.url).toContain('minimal-video:minimal123');
      expect(first.content).toBe('');
      expect(first.thumbnailUrl).toBe('');
      expect(first.duration).toBe('');
      expect(first.channel).toBe('');
      expect(first.views).toBe(0);
      expect(first.publishedAt).toBe('');
    });

    it('should skip claims without claimId', () => {
      const mockResponse = JSON.stringify([
        {
          name: 'no-claim-id',
          title: 'No Claim ID Video',
        },
        {
          claimId: 'valid123',
          name: 'valid-video',
          title: 'Valid Video',
        },
      ]);

      const results = engine.parseResponse(mockResponse, defaultParams);

      expect(results.results.length).toBe(1);
      expect(results.results[0].title).toBe('Valid Video');
    });

    it('should skip claims without name', () => {
      const mockResponse = JSON.stringify([
        {
          claimId: 'noname123',
          title: 'No Name Video',
        },
        {
          claimId: 'valid456',
          name: 'valid-video',
          title: 'Valid Video',
        },
      ]);

      const results = engine.parseResponse(mockResponse, defaultParams);

      expect(results.results.length).toBe(1);
      expect(results.results[0].title).toBe('Valid Video');
    });

    it('should return empty results for invalid JSON', () => {
      const results = engine.parseResponse('not valid json', defaultParams);
      expect(results.results).toEqual([]);
    });

    it('should return empty results when response is not an array', () => {
      const mockResponse = JSON.stringify({ error: 'not found' });
      const results = engine.parseResponse(mockResponse, defaultParams);
      expect(results.results).toEqual([]);
    });

    it('should format duration correctly for various lengths', () => {
      const mockResponse = JSON.stringify([
        { claimId: 'v1', name: 'v1', duration: 45 },
        { claimId: 'v2', name: 'v2', duration: 605 },
        { claimId: 'v3', name: 'v3', duration: 3661 },
        { claimId: 'v4', name: 'v4', duration: 7322 },
      ]);

      const results = engine.parseResponse(mockResponse, defaultParams);

      expect(results.results[0].duration).toBe('0:45');
      expect(results.results[1].duration).toBe('10:05');
      expect(results.results[2].duration).toBe('1:01:01');
      expect(results.results[3].duration).toBe('2:02:02');
    });

    it('should build correct URL with channel', () => {
      const mockResponse = JSON.stringify([
        {
          claimId: 'abc123',
          name: 'my-video',
          channel: '@MyChannel',
        },
      ]);

      const results = engine.parseResponse(mockResponse, defaultParams);

      expect(results.results.length).toBe(1);
      expect(results.results[0].url).toContain('@MyChannel');
      expect(results.results[0].url).toContain('my-video');
    });

    it('should build correct URL without channel', () => {
      const mockResponse = JSON.stringify([
        {
          claimId: 'abc123def456',
          name: 'orphan-video',
        },
      ]);

      const results = engine.parseResponse(mockResponse, defaultParams);

      expect(results.results.length).toBe(1);
      expect(results.results[0].url).toBe('https://odysee.com/orphan-video:abc123def456');
    });
  });

  describe('live API test', () => {
    it('should search and return video results', async () => {
      const params = {
        page: 1,
        locale: 'en',
        safeSearch: 1 as const,
        timeRange: '' as const,
        engineData: {},
      };
      const config = engine.buildRequest('programming', params);
      const res = await fetch(config.url, {
        headers: config.headers,
      });

      expect(res.ok).toBe(true);

      const body = await res.text();
      const results = engine.parseResponse(body, params);

      expect(results.results.length).toBeGreaterThan(0);

      const first = results.results[0];
      expect(first.url).toContain('odysee.com');
      expect(first.title).toBeTruthy();
      expect(first.engine).toBe('odysee');
      expect(first.category).toBe('videos');

      // Most Odysee videos should have these
      if (first.thumbnailUrl) {
        expect(first.thumbnailUrl).toMatch(/^https?:\/\//);
      }
      if (first.embedUrl) {
        expect(first.embedUrl).toContain('$/embed/');
      }
    }, 30000);

    it('should respect pagination', async () => {
      const params1 = {
        page: 1,
        locale: 'en',
        safeSearch: 1 as const,
        timeRange: '' as const,
        engineData: {},
      };
      const params2 = {
        page: 2,
        locale: 'en',
        safeSearch: 1 as const,
        timeRange: '' as const,
        engineData: {},
      };

      const config1 = engine.buildRequest('tutorial', params1);
      const config2 = engine.buildRequest('tutorial', params2);

      const [res1, res2] = await Promise.all([
        fetch(config1.url, { headers: config1.headers }),
        fetch(config2.url, { headers: config2.headers }),
      ]);

      const [body1, body2] = await Promise.all([res1.text(), res2.text()]);

      const results1 = engine.parseResponse(body1, params1);
      const results2 = engine.parseResponse(body2, params2);

      // Both pages should have results
      expect(results1.results.length).toBeGreaterThan(0);
      expect(results2.results.length).toBeGreaterThan(0);

      // Results should be different
      if (results1.results[0] && results2.results[0]) {
        expect(results1.results[0].url).not.toBe(results2.results[0].url);
      }
    }, 30000);
  });
});
