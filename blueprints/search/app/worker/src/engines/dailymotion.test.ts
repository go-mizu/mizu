import { describe, it, expect } from 'vitest';
import { DailymotionEngine } from './dailymotion';

describe('DailymotionEngine', () => {
  const engine = new DailymotionEngine();

  describe('metadata', () => {
    it('should have correct name and shortcut', () => {
      expect(engine.name).toBe('dailymotion');
      expect(engine.shortcut).toBe('dm');
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
      expect(engine.weight).toBe(0.85);
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
      expect(config.url).toContain('https://api.dailymotion.com/videos');
      expect(config.url).toContain('search=test+query');
      expect(config.method).toBe('GET');
      expect(config.headers['Accept']).toBe('application/json');
    });

    it('should include correct fields parameter', () => {
      const config = engine.buildRequest('test', {
        page: 1,
        locale: 'en',
        safeSearch: 1,
        timeRange: '',
        engineData: {},
      });
      expect(config.url).toContain('fields=');
      expect(config.url).toContain('id');
      expect(config.url).toContain('title');
      expect(config.url).toContain('description');
      expect(config.url).toContain('duration');
      expect(config.url).toContain('thumbnail_360_url');
      expect(config.url).toContain('owner.screenname');
      expect(config.url).toContain('views_total');
      expect(config.url).toContain('embed_url');
      expect(config.url).toContain('allow_embed');
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
      expect(page1.url).toContain('limit=10');

      const page3 = engine.buildRequest('test', {
        page: 3,
        locale: 'en',
        safeSearch: 1,
        timeRange: '',
        engineData: {},
      });
      expect(page3.url).toContain('page=3');
    });

    describe('safe search', () => {
      it('should disable family filter when safeSearch is 0 (off)', () => {
        const config = engine.buildRequest('test', {
          page: 1,
          locale: 'en',
          safeSearch: 0,
          timeRange: '',
          engineData: {},
        });
        expect(config.url).toContain('family_filter=false');
        expect(config.url).not.toContain('is_created_for_kids');
      });

      it('should enable family filter when safeSearch is 1 (moderate)', () => {
        const config = engine.buildRequest('test', {
          page: 1,
          locale: 'en',
          safeSearch: 1,
          timeRange: '',
          engineData: {},
        });
        expect(config.url).toContain('family_filter=true');
        expect(config.url).not.toContain('is_created_for_kids');
      });

      it('should enable strict mode when safeSearch is 2', () => {
        const config = engine.buildRequest('test', {
          page: 1,
          locale: 'en',
          safeSearch: 2,
          timeRange: '',
          engineData: {},
        });
        expect(config.url).toContain('family_filter=true');
        expect(config.url).toContain('is_created_for_kids=true');
      });
    });

    describe('time range filter', () => {
      it('should add created_after for day filter', () => {
        const now = Math.floor(Date.now() / 1000);
        const config = engine.buildRequest('test', {
          page: 1,
          locale: 'en',
          safeSearch: 1,
          timeRange: 'day',
          engineData: {},
        });
        expect(config.url).toContain('created_after=');
        // Verify timestamp is approximately 24 hours ago (within 10 seconds tolerance)
        const match = config.url.match(/created_after=(\d+)/);
        expect(match).toBeTruthy();
        const timestamp = parseInt(match![1], 10);
        expect(timestamp).toBeGreaterThan(now - 24 * 60 * 60 - 10);
        expect(timestamp).toBeLessThanOrEqual(now - 24 * 60 * 60 + 10);
      });

      it('should add created_after for week filter', () => {
        const now = Math.floor(Date.now() / 1000);
        const config = engine.buildRequest('test', {
          page: 1,
          locale: 'en',
          safeSearch: 1,
          timeRange: 'week',
          engineData: {},
        });
        const match = config.url.match(/created_after=(\d+)/);
        expect(match).toBeTruthy();
        const timestamp = parseInt(match![1], 10);
        expect(timestamp).toBeGreaterThan(now - 7 * 24 * 60 * 60 - 10);
        expect(timestamp).toBeLessThanOrEqual(now - 7 * 24 * 60 * 60 + 10);
      });

      it('should add created_after for month filter', () => {
        const now = Math.floor(Date.now() / 1000);
        const config = engine.buildRequest('test', {
          page: 1,
          locale: 'en',
          safeSearch: 1,
          timeRange: 'month',
          engineData: {},
        });
        const match = config.url.match(/created_after=(\d+)/);
        expect(match).toBeTruthy();
        const timestamp = parseInt(match![1], 10);
        expect(timestamp).toBeGreaterThan(now - 30 * 24 * 60 * 60 - 10);
        expect(timestamp).toBeLessThanOrEqual(now - 30 * 24 * 60 * 60 + 10);
      });

      it('should add created_after for year filter', () => {
        const now = Math.floor(Date.now() / 1000);
        const config = engine.buildRequest('test', {
          page: 1,
          locale: 'en',
          safeSearch: 1,
          timeRange: 'year',
          engineData: {},
        });
        const match = config.url.match(/created_after=(\d+)/);
        expect(match).toBeTruthy();
        const timestamp = parseInt(match![1], 10);
        expect(timestamp).toBeGreaterThan(now - 365 * 24 * 60 * 60 - 10);
        expect(timestamp).toBeLessThanOrEqual(now - 365 * 24 * 60 * 60 + 10);
      });

      it('should not add created_after when timeRange is empty', () => {
        const config = engine.buildRequest('test', {
          page: 1,
          locale: 'en',
          safeSearch: 1,
          timeRange: '',
          engineData: {},
        });
        expect(config.url).not.toContain('created_after');
      });
    });

    describe('locale/language filter', () => {
      it('should extract language from locale', () => {
        const config = engine.buildRequest('test', {
          page: 1,
          locale: 'en-US',
          safeSearch: 1,
          timeRange: '',
          engineData: {},
        });
        expect(config.url).toContain('languages=en');
      });

      it('should handle simple locale codes', () => {
        const config = engine.buildRequest('test', {
          page: 1,
          locale: 'fr',
          safeSearch: 1,
          timeRange: '',
          engineData: {},
        });
        expect(config.url).toContain('languages=fr');
      });
    });

    it('should exclude private and password-protected videos', () => {
      const config = engine.buildRequest('test', {
        page: 1,
        locale: 'en',
        safeSearch: 1,
        timeRange: '',
        engineData: {},
      });
      expect(config.url).toContain('private=false');
      expect(config.url).toContain('password_protected=false');
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

    it('should parse valid API response', () => {
      const mockResponse = JSON.stringify({
        page: 1,
        limit: 10,
        explicit: false,
        total: 100,
        has_more: true,
        list: [
          {
            id: 'x8abcde',
            title: 'Test Video Title',
            description: 'Test video description here',
            duration: 185,
            thumbnail_360_url: 'https://s2.dmcdn.net/v/abcde/x360',
            created_time: 1704067200, // 2024-01-01 00:00:00 UTC
            'owner.screenname': 'TestChannel',
            views_total: 12500,
            embed_url: 'https://www.dailymotion.com/embed/video/x8abcde',
            allow_embed: true,
          },
          {
            id: 'x8fghij',
            title: 'Another Video',
            description: 'Second video description',
            duration: 3661,
            thumbnail_360_url: 'https://s2.dmcdn.net/v/fghij/x360',
            created_time: 1706745600,
            'owner.screenname': 'AnotherChannel',
            views_total: 50000,
            allow_embed: true,
          },
        ],
      });

      const results = engine.parseResponse(mockResponse, defaultParams);

      expect(results.results.length).toBe(2);

      const first = results.results[0];
      expect(first.url).toBe('https://www.dailymotion.com/video/x8abcde');
      expect(first.title).toBe('Test Video Title');
      expect(first.content).toBe('Test video description here');
      expect(first.duration).toBe('3:05');
      expect(first.thumbnailUrl).toBe('https://s2.dmcdn.net/v/abcde/x360');
      expect(first.channel).toBe('TestChannel');
      expect(first.views).toBe(12500);
      expect(first.embedUrl).toBe(
        'https://www.dailymotion.com/embed/video/x8abcde'
      );
      expect(first.publishedAt).toBe('2024-01-01T00:00:00.000Z');
      expect(first.engine).toBe('dailymotion');
      expect(first.category).toBe('videos');
      expect(first.template).toBe('videos');

      const second = results.results[1];
      expect(second.url).toBe('https://www.dailymotion.com/video/x8fghij');
      expect(second.title).toBe('Another Video');
      expect(second.duration).toBe('1:01:01');
      expect(second.views).toBe(50000);
    });

    it('should handle videos with embedding disabled', () => {
      const mockResponse = JSON.stringify({
        list: [
          {
            id: 'x8noemb',
            title: 'No Embed Video',
            description: 'Cannot be embedded',
            duration: 60,
            allow_embed: false,
          },
        ],
      });

      const results = engine.parseResponse(mockResponse, defaultParams);

      expect(results.results.length).toBe(1);
      expect(results.results[0].embedUrl).toBe('');
    });

    it('should handle missing optional fields gracefully', () => {
      const mockResponse = JSON.stringify({
        list: [
          {
            id: 'x8minimal',
          },
        ],
      });

      const results = engine.parseResponse(mockResponse, defaultParams);

      expect(results.results.length).toBe(1);
      const result = results.results[0];
      expect(result.url).toBe('https://www.dailymotion.com/video/x8minimal');
      expect(result.title).toBe('');
      expect(result.content).toBe('');
      expect(result.duration).toBe('');
      expect(result.channel).toBe('');
      expect(result.views).toBe(0);
      expect(result.thumbnailUrl).toBe('');
      expect(result.publishedAt).toBe('');
    });

    it('should skip videos without id', () => {
      const mockResponse = JSON.stringify({
        list: [
          {
            title: 'Video without ID',
            description: 'This should be skipped',
          },
          {
            id: 'x8valid',
            title: 'Valid Video',
          },
        ],
      });

      const results = engine.parseResponse(mockResponse, defaultParams);

      expect(results.results.length).toBe(1);
      expect(results.results[0].title).toBe('Valid Video');
    });

    it('should return empty results for invalid JSON', () => {
      const results = engine.parseResponse('not valid json', defaultParams);
      expect(results.results).toEqual([]);
    });

    it('should return empty results for missing list', () => {
      const mockResponse = JSON.stringify({
        page: 1,
        total: 0,
      });

      const results = engine.parseResponse(mockResponse, defaultParams);
      expect(results.results).toEqual([]);
    });

    it('should return empty results for null list', () => {
      const mockResponse = JSON.stringify({
        list: null,
      });

      const results = engine.parseResponse(mockResponse, defaultParams);
      expect(results.results).toEqual([]);
    });

    it('should format duration correctly', () => {
      const mockResponse = JSON.stringify({
        list: [
          { id: 'v1', duration: 5 }, // 5 seconds
          { id: 'v2', duration: 65 }, // 1 min 5 sec
          { id: 'v3', duration: 3600 }, // 1 hour
          { id: 'v4', duration: 3661 }, // 1 hour 1 min 1 sec
          { id: 'v5', duration: 36610 }, // 10 hours 10 min 10 sec
        ],
      });

      const results = engine.parseResponse(mockResponse, defaultParams);

      expect(results.results[0].duration).toBe('0:05');
      expect(results.results[1].duration).toBe('1:05');
      expect(results.results[2].duration).toBe('1:00:00');
      expect(results.results[3].duration).toBe('1:01:01');
      expect(results.results[4].duration).toBe('10:10:10');
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
      const config = engine.buildRequest('javascript tutorial', params);
      const res = await fetch(config.url, {
        headers: config.headers,
      });

      expect(res.ok).toBe(true);

      const body = await res.text();
      const results = engine.parseResponse(body, params);

      expect(results.results.length).toBeGreaterThan(0);
      const first = results.results[0];
      expect(first.url).toContain('dailymotion.com/video/');
      expect(first.title).toBeTruthy();
      expect(first.engine).toBe('dailymotion');
      expect(first.category).toBe('videos');
    }, 30000);
  });
});
