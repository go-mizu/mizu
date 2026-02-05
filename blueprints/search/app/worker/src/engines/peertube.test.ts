import { describe, it, expect } from 'vitest';
import { PeerTubeEngine } from './peertube';

describe('PeerTubeEngine', () => {
  const engine = new PeerTubeEngine();

  describe('metadata', () => {
    it('should have correct metadata', () => {
      expect(engine.name).toBe('peertube');
      expect(engine.shortcut).toBe('ptb');
      expect(engine.categories).toContain('videos');
      expect(engine.supportsPaging).toBe(true);
      expect(engine.maxPage).toBe(10);
      expect(engine.timeout).toBe(10000);
      expect(engine.weight).toBe(0.75);
      expect(engine.disabled).toBe(false);
    });
  });

  describe('URL building', () => {
    it('should build correct search URL', () => {
      const config = engine.buildRequest('javascript tutorial', {
        page: 1,
        locale: 'en',
        safeSearch: 1,
        timeRange: '',
        engineData: {},
      });

      expect(config.url).toContain('https://sepiasearch.org/api/v1/search/videos');
      expect(config.url).toContain('search=javascript+tutorial');
      expect(config.url).toContain('start=0');
      expect(config.url).toContain('count=15');
      expect(config.url).toContain('sort=-match');
      expect(config.url).toContain('searchTarget=search-index');
      expect(config.method).toBe('GET');
      expect(config.headers['Accept']).toBe('application/json');
    });

    it('should build URL with pagination', () => {
      const config = engine.buildRequest('test', {
        page: 3,
        locale: 'en',
        safeSearch: 1,
        timeRange: '',
        engineData: {},
      });

      // Page 3 means start at offset 30 (15 * 2)
      expect(config.url).toContain('start=30');
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

    it('should set nsfw=both for safe search disabled', () => {
      const config = engine.buildRequest('test', {
        page: 1,
        locale: 'en',
        safeSearch: 0,
        timeRange: '',
        engineData: {},
      });

      expect(config.url).toContain('nsfw=both');
    });

    it('should include language parameters', () => {
      const config = engine.buildRequest('test', {
        page: 1,
        locale: 'fr-FR',
        safeSearch: 1,
        timeRange: '',
        engineData: {},
      });

      expect(config.url).toContain('languageOneOf%5B%5D=fr');
      expect(config.url).toContain('boostLanguages%5B%5D=fr');
    });

    it('should include startDate for day time range', () => {
      const config = engine.buildRequest('test', {
        page: 1,
        locale: 'en',
        safeSearch: 1,
        timeRange: 'day',
        engineData: {},
      });

      expect(config.url).toContain('startDate=');
      // Verify it's a valid ISO date
      const match = config.url.match(/startDate=([^&]+)/);
      expect(match).toBeTruthy();
      const dateStr = decodeURIComponent(match![1]);
      const date = new Date(dateStr);
      expect(date.getTime()).toBeLessThan(Date.now());
      expect(date.getTime()).toBeGreaterThan(Date.now() - 2 * 24 * 60 * 60 * 1000);
    });

    it('should include startDate for week time range', () => {
      const config = engine.buildRequest('test', {
        page: 1,
        locale: 'en',
        safeSearch: 1,
        timeRange: 'week',
        engineData: {},
      });

      expect(config.url).toContain('startDate=');
      const match = config.url.match(/startDate=([^&]+)/);
      const dateStr = decodeURIComponent(match![1]);
      const date = new Date(dateStr);
      expect(date.getTime()).toBeGreaterThan(Date.now() - 8 * 24 * 60 * 60 * 1000);
    });

    it('should include startDate for month time range', () => {
      const config = engine.buildRequest('test', {
        page: 1,
        locale: 'en',
        safeSearch: 1,
        timeRange: 'month',
        engineData: {},
      });

      expect(config.url).toContain('startDate=');
      const match = config.url.match(/startDate=([^&]+)/);
      const dateStr = decodeURIComponent(match![1]);
      const date = new Date(dateStr);
      expect(date.getTime()).toBeGreaterThan(Date.now() - 31 * 24 * 60 * 60 * 1000);
    });

    it('should include startDate for year time range', () => {
      const config = engine.buildRequest('test', {
        page: 1,
        locale: 'en',
        safeSearch: 1,
        timeRange: 'year',
        engineData: {},
      });

      expect(config.url).toContain('startDate=');
      const match = config.url.match(/startDate=([^&]+)/);
      const dateStr = decodeURIComponent(match![1]);
      const date = new Date(dateStr);
      expect(date.getTime()).toBeGreaterThan(Date.now() - 366 * 24 * 60 * 60 * 1000);
    });

    it('should not include startDate for empty time range', () => {
      const config = engine.buildRequest('test', {
        page: 1,
        locale: 'en',
        safeSearch: 1,
        timeRange: '',
        engineData: {},
      });

      expect(config.url).not.toContain('startDate=');
    });
  });

  describe('response parsing', () => {
    it('should parse PeerTube API response correctly', () => {
      const mockResponse = JSON.stringify({
        total: 100,
        data: [
          {
            url: 'https://framatube.org/videos/watch/abc123',
            name: 'JavaScript Tutorial',
            description: 'Learn JavaScript programming',
            duration: 3661,
            views: 5000,
            publishedAt: '2024-01-15T10:00:00Z',
            thumbnailPath: '/static/thumbnails/abc123.jpg',
            previewPath: '/static/previews/abc123.jpg',
            embedPath: '/videos/embed/abc123',
            account: { displayName: 'CodeMaster', host: 'framatube.org' },
            channel: { displayName: 'Programming Channel', host: 'framatube.org' },
          },
          {
            url: 'https://video.liberta.vip/videos/watch/def456',
            name: 'Python Basics',
            description: 'Python for beginners',
            duration: 1800,
            views: 2500,
            publishedAt: '2024-02-01T14:30:00Z',
            thumbnailPath: '/static/thumbnails/def456.jpg',
            embedPath: '/videos/embed/def456',
            account: { displayName: 'PyDev', host: 'video.liberta.vip' },
            channel: { displayName: 'Python Hub', host: 'video.liberta.vip' },
          },
        ],
      });

      const params = {
        page: 1,
        locale: 'en',
        safeSearch: 1 as const,
        timeRange: '' as const,
        engineData: {},
      };

      const results = engine.parseResponse(mockResponse, params);

      expect(results.results.length).toBe(2);

      const first = results.results[0];
      expect(first.url).toBe('https://framatube.org/videos/watch/abc123');
      expect(first.title).toBe('JavaScript Tutorial');
      expect(first.content).toBe('Learn JavaScript programming');
      expect(first.duration).toBe('1:01:01');
      expect(first.views).toBe(5000);
      expect(first.thumbnailUrl).toBe('https://framatube.org/static/thumbnails/abc123.jpg');
      expect(first.embedUrl).toBe('https://framatube.org/videos/embed/abc123');
      expect(first.channel).toBe('Programming Channel@framatube.org');
      expect(first.publishedAt).toBe('2024-01-15T10:00:00Z');
      expect(first.engine).toBe('peertube');
      expect(first.category).toBe('videos');
      expect(first.template).toBe('videos');

      const second = results.results[1];
      expect(second.url).toBe('https://video.liberta.vip/videos/watch/def456');
      expect(second.title).toBe('Python Basics');
      expect(second.duration).toBe('30:00');
      expect(second.channel).toBe('Python Hub@video.liberta.vip');
    });

    it('should fallback to account when channel is missing', () => {
      const mockResponse = JSON.stringify({
        total: 1,
        data: [
          {
            url: 'https://peertube.fr/videos/watch/xyz789',
            name: 'Test Video',
            duration: 300,
            views: 100,
            thumbnailPath: '/static/thumbnails/xyz789.jpg',
            embedPath: '/videos/embed/xyz789',
            account: { displayName: 'TestUser', host: 'peertube.fr' },
          },
        ],
      });

      const params = {
        page: 1,
        locale: 'en',
        safeSearch: 1 as const,
        timeRange: '' as const,
        engineData: {},
      };

      const results = engine.parseResponse(mockResponse, params);

      expect(results.results.length).toBe(1);
      expect(results.results[0].channel).toBe('TestUser@peertube.fr');
    });

    it('should fallback to previewPath when thumbnailPath is missing', () => {
      const mockResponse = JSON.stringify({
        total: 1,
        data: [
          {
            url: 'https://peertube.fr/videos/watch/xyz789',
            name: 'Test Video',
            previewPath: '/static/previews/xyz789.jpg',
            embedPath: '/videos/embed/xyz789',
          },
        ],
      });

      const params = {
        page: 1,
        locale: 'en',
        safeSearch: 1 as const,
        timeRange: '' as const,
        engineData: {},
      };

      const results = engine.parseResponse(mockResponse, params);

      expect(results.results.length).toBe(1);
      expect(results.results[0].thumbnailUrl).toBe('https://peertube.fr/static/previews/xyz789.jpg');
    });

    it('should handle missing optional fields gracefully', () => {
      const mockResponse = JSON.stringify({
        total: 1,
        data: [
          {
            url: 'https://peertube.fr/videos/watch/minimal',
            name: 'Minimal Video',
          },
        ],
      });

      const params = {
        page: 1,
        locale: 'en',
        safeSearch: 1 as const,
        timeRange: '' as const,
        engineData: {},
      };

      const results = engine.parseResponse(mockResponse, params);

      expect(results.results.length).toBe(1);
      const first = results.results[0];
      expect(first.url).toBe('https://peertube.fr/videos/watch/minimal');
      expect(first.title).toBe('Minimal Video');
      expect(first.content).toBe('');
      expect(first.duration).toBe('');
      expect(first.views).toBe(0);
      expect(first.thumbnailUrl).toBe('');
      expect(first.embedUrl).toBe('');
      expect(first.channel).toBe('');
    });

    it('should skip videos without url', () => {
      const mockResponse = JSON.stringify({
        total: 2,
        data: [
          { name: 'No URL Video' },
          { url: 'https://peertube.fr/videos/watch/valid', name: 'Valid Video' },
        ],
      });

      const params = {
        page: 1,
        locale: 'en',
        safeSearch: 1 as const,
        timeRange: '' as const,
        engineData: {},
      };

      const results = engine.parseResponse(mockResponse, params);

      expect(results.results.length).toBe(1);
      expect(results.results[0].title).toBe('Valid Video');
    });

    it('should skip videos without name', () => {
      const mockResponse = JSON.stringify({
        total: 2,
        data: [
          { url: 'https://peertube.fr/videos/watch/noname' },
          { url: 'https://peertube.fr/videos/watch/valid', name: 'Valid Video' },
        ],
      });

      const params = {
        page: 1,
        locale: 'en',
        safeSearch: 1 as const,
        timeRange: '' as const,
        engineData: {},
      };

      const results = engine.parseResponse(mockResponse, params);

      expect(results.results.length).toBe(1);
      expect(results.results[0].title).toBe('Valid Video');
    });

    it('should return empty results for invalid JSON', () => {
      const params = {
        page: 1,
        locale: 'en',
        safeSearch: 1 as const,
        timeRange: '' as const,
        engineData: {},
      };

      const results = engine.parseResponse('not valid json', params);

      expect(results.results).toEqual([]);
    });

    it('should return empty results when data is missing', () => {
      const mockResponse = JSON.stringify({ total: 0 });

      const params = {
        page: 1,
        locale: 'en',
        safeSearch: 1 as const,
        timeRange: '' as const,
        engineData: {},
      };

      const results = engine.parseResponse(mockResponse, params);

      expect(results.results).toEqual([]);
    });

    it('should return empty results when data is not an array', () => {
      const mockResponse = JSON.stringify({ total: 0, data: 'not an array' });

      const params = {
        page: 1,
        locale: 'en',
        safeSearch: 1 as const,
        timeRange: '' as const,
        engineData: {},
      };

      const results = engine.parseResponse(mockResponse, params);

      expect(results.results).toEqual([]);
    });

    it('should format duration correctly for various lengths', () => {
      const mockResponse = JSON.stringify({
        total: 4,
        data: [
          { url: 'https://pt.org/v/1', name: 'Short', duration: 45 },
          { url: 'https://pt.org/v/2', name: 'Medium', duration: 605 },
          { url: 'https://pt.org/v/3', name: 'Long', duration: 3661 },
          { url: 'https://pt.org/v/4', name: 'Very Long', duration: 7322 },
        ],
      });

      const params = {
        page: 1,
        locale: 'en',
        safeSearch: 1 as const,
        timeRange: '' as const,
        engineData: {},
      };

      const results = engine.parseResponse(mockResponse, params);

      expect(results.results[0].duration).toBe('0:45');
      expect(results.results[1].duration).toBe('10:05');
      expect(results.results[2].duration).toBe('1:01:01');
      expect(results.results[3].duration).toBe('2:02:02');
    });
  });

  describe('live API test', () => {
    it('should search and return video results from Sepia Search', async () => {
      const results = await fetchAndParse(engine, 'programming');

      expect(results.results.length).toBeGreaterThan(0);

      const first = results.results[0];
      expect(first.url).toBeTruthy();
      expect(first.url).toContain('/videos/watch/');
      expect(first.title).toBeTruthy();
      expect(first.engine).toBe('peertube');
      expect(first.category).toBe('videos');

      // Most PeerTube videos should have these
      if (first.thumbnailUrl) {
        expect(first.thumbnailUrl).toMatch(/^https?:\/\//);
      }
      if (first.embedUrl) {
        expect(first.embedUrl).toContain('/videos/embed/');
      }
    }, 30000);

    it('should respect time range filter', async () => {
      const params = {
        page: 1,
        locale: 'en',
        safeSearch: 1 as const,
        timeRange: 'week' as const,
        engineData: {},
      };
      const config = engine.buildRequest('linux', params);

      let body: string;
      try {
        const res = await fetch(config.url, {
          headers: config.headers,
        });
        body = await res.text();
      } catch (err) {
        // Network errors may occur in some environments - skip test gracefully
        console.warn('Network error in time range test, skipping:', err);
        return;
      }

      const results = engine.parseResponse(body, params);

      // Should return results (may be empty if nothing matches)
      expect(Array.isArray(results.results)).toBe(true);

      // If we have results with publishedAt, verify they are recent
      for (const result of results.results) {
        if (result.publishedAt) {
          const publishDate = new Date(result.publishedAt);
          const weekAgo = new Date(Date.now() - 8 * 24 * 60 * 60 * 1000);
          expect(publishDate.getTime()).toBeGreaterThan(weekAgo.getTime());
        }
      }
    }, 30000);
  });
});

async function fetchAndParse(engine: PeerTubeEngine, query: string) {
  const params = {
    page: 1,
    locale: 'en',
    safeSearch: 1 as const,
    timeRange: '' as const,
    engineData: {},
  };
  const config = engine.buildRequest(query, params);
  const res = await fetch(config.url, {
    headers: config.headers,
  });
  const body = await res.text();
  return engine.parseResponse(body, params);
}
