import { describe, it, expect } from 'vitest';
import { RumbleEngine } from './rumble';

describe('RumbleEngine', () => {
  const engine = new RumbleEngine();

  describe('metadata', () => {
    it('should have correct name and shortcut', () => {
      expect(engine.name).toBe('rumble');
      expect(engine.shortcut).toBe('rb');
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
      expect(engine.weight).toBe(0.7);
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

      expect(config.url).toContain('https://rumble.com/search/video');
      expect(config.url).toContain('q=test+query');
      expect(config.method).toBe('GET');
      expect(config.headers['Accept']).toContain('text/html');
    });

    it('should not add page parameter for page 1', () => {
      const config = engine.buildRequest('test', {
        page: 1,
        locale: 'en',
        safeSearch: 1,
        timeRange: '',
        engineData: {},
      });

      expect(config.url).not.toContain('page=');
    });

    it('should add page parameter for page 2+', () => {
      const config = engine.buildRequest('test', {
        page: 3,
        locale: 'en',
        safeSearch: 1,
        timeRange: '',
        engineData: {},
      });

      expect(config.url).toContain('page=3');
    });

    it('should add date filter for day time range', () => {
      const config = engine.buildRequest('test', {
        page: 1,
        locale: 'en',
        safeSearch: 1,
        timeRange: 'day',
        engineData: {},
      });

      expect(config.url).toContain('date=today');
    });

    it('should add date filter for week time range', () => {
      const config = engine.buildRequest('test', {
        page: 1,
        locale: 'en',
        safeSearch: 1,
        timeRange: 'week',
        engineData: {},
      });

      expect(config.url).toContain('date=this-week');
    });

    it('should add date filter for month time range', () => {
      const config = engine.buildRequest('test', {
        page: 1,
        locale: 'en',
        safeSearch: 1,
        timeRange: 'month',
        engineData: {},
      });

      expect(config.url).toContain('date=this-month');
    });

    it('should add date filter for year time range', () => {
      const config = engine.buildRequest('test', {
        page: 1,
        locale: 'en',
        safeSearch: 1,
        timeRange: 'year',
        engineData: {},
      });

      expect(config.url).toContain('date=this-year');
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

    it('should parse JSON-LD VideoObject data', () => {
      const mockHtml = `
        <html>
          <head>
            <script type="application/ld+json">
              {
                "@type": "VideoObject",
                "url": "https://rumble.com/v1abc23-test-video.html",
                "name": "Test Video Title",
                "description": "This is a test video description",
                "thumbnailUrl": "https://sp.rmbl.ws/thumb/v1abc23.jpg",
                "duration": "PT5M30S",
                "uploadDate": "2024-01-15",
                "author": { "name": "TestChannel" },
                "interactionCount": 12500
              }
            </script>
          </head>
          <body></body>
        </html>
      `;

      const results = engine.parseResponse(mockHtml, defaultParams);

      expect(results.results.length).toBe(1);

      const first = results.results[0];
      expect(first.url).toBe('https://rumble.com/v1abc23-test-video.html');
      expect(first.title).toBe('Test Video Title');
      expect(first.content).toBe('This is a test video description');
      expect(first.thumbnailUrl).toBe('https://sp.rmbl.ws/thumb/v1abc23.jpg');
      expect(first.duration).toBe('5:30');
      expect(first.channel).toBe('TestChannel');
      expect(first.views).toBe(12500);
      expect(first.publishedAt).toBe('2024-01-15');
      expect(first.embedUrl).toBe('https://rumble.com/embed/v1abc23/');
      expect(first.engine).toBe('rumble');
      expect(first.category).toBe('videos');
      expect(first.template).toBe('videos');
    });

    it('should parse JSON-LD ItemList data', () => {
      const mockHtml = `
        <html>
          <head>
            <script type="application/ld+json">
              {
                "@type": "ItemList",
                "itemListElement": [
                  {
                    "item": {
                      "@type": "VideoObject",
                      "url": "https://rumble.com/v1video1-first.html",
                      "name": "First Video",
                      "duration": "PT10M0S"
                    }
                  },
                  {
                    "item": {
                      "@type": "VideoObject",
                      "url": "https://rumble.com/v2video2-second.html",
                      "name": "Second Video",
                      "duration": "PT1H5M30S"
                    }
                  }
                ]
              }
            </script>
          </head>
          <body></body>
        </html>
      `;

      const results = engine.parseResponse(mockHtml, defaultParams);

      expect(results.results.length).toBe(2);
      expect(results.results[0].title).toBe('First Video');
      expect(results.results[0].duration).toBe('10:00');
      expect(results.results[1].title).toBe('Second Video');
      expect(results.results[1].duration).toBe('1:05:30');
    });

    it('should parse video links from HTML as fallback', () => {
      // Rumble URLs: https://rumble.com/v1abc23-title.html
      // The regex captures the last segment before .html after the last dash
      const mockHtml = `
        <html>
          <body>
            <a href="https://rumble.com/v1test1-myvideo123456.html">My Video Title</a>
            <a href="https://rumble.com/v2test2-anothervideo789.html">Another Video</a>
          </body>
        </html>
      `;

      const results = engine.parseResponse(mockHtml, defaultParams);

      expect(results.results.length).toBe(2);
      expect(results.results[0].url).toBe('https://rumble.com/v1test1-myvideo123456.html');
      expect(results.results[0].title).toBe('My Video Title');
      // Note: The regex extracts the ID from the end of the URL
      expect(results.results[0].embedUrl).toContain('https://rumble.com/embed/');
      expect(results.results[1].url).toBe('https://rumble.com/v2test2-anothervideo789.html');
      expect(results.results[1].title).toBe('Another Video');
    });

    it('should handle missing optional fields gracefully', () => {
      const mockHtml = `
        <html>
          <head>
            <script type="application/ld+json">
              {
                "@type": "VideoObject",
                "url": "https://rumble.com/v1min23-minimal.html",
                "name": "Minimal Video"
              }
            </script>
          </head>
          <body></body>
        </html>
      `;

      const results = engine.parseResponse(mockHtml, defaultParams);

      expect(results.results.length).toBe(1);
      const first = results.results[0];
      expect(first.url).toBe('https://rumble.com/v1min23-minimal.html');
      expect(first.title).toBe('Minimal Video');
      expect(first.content).toBe('');
      expect(first.duration).toBe('');
      expect(first.thumbnailUrl).toBe('');
      expect(first.channel).toBe('');
      expect(first.views).toBe(0);
    });

    it('should skip videos without URL', () => {
      const mockHtml = `
        <html>
          <head>
            <script type="application/ld+json">
              {
                "@type": "VideoObject",
                "name": "No URL Video"
              }
            </script>
          </head>
          <body></body>
        </html>
      `;

      const results = engine.parseResponse(mockHtml, defaultParams);

      expect(results.results.length).toBe(0);
    });

    it('should return empty results for invalid JSON', () => {
      const results = engine.parseResponse('not valid json at all', defaultParams);
      expect(results.results).toEqual([]);
    });

    it('should format duration correctly', () => {
      const mockHtml = `
        <html>
          <head>
            <script type="application/ld+json">
              [
                {
                  "@type": "VideoObject",
                  "url": "https://rumble.com/v1a-a.html",
                  "name": "V1",
                  "duration": "PT45S"
                },
                {
                  "@type": "VideoObject",
                  "url": "https://rumble.com/v1b-b.html",
                  "name": "V2",
                  "duration": "PT5M5S"
                },
                {
                  "@type": "VideoObject",
                  "url": "https://rumble.com/v1c-c.html",
                  "name": "V3",
                  "duration": "PT1H1M1S"
                },
                {
                  "@type": "VideoObject",
                  "url": "https://rumble.com/v1d-d.html",
                  "name": "V4",
                  "duration": "PT2H30M"
                }
              ]
            </script>
          </head>
          <body></body>
        </html>
      `;

      const results = engine.parseResponse(mockHtml, defaultParams);

      expect(results.results[0].duration).toBe('0:45');
      expect(results.results[1].duration).toBe('5:05');
      expect(results.results[2].duration).toBe('1:01:01');
      expect(results.results[3].duration).toBe('2:30:00');
    });

    it('should handle view counts in various formats', () => {
      const mockHtml = `
        <html>
          <head>
            <script type="application/ld+json">
              [
                {
                  "@type": "VideoObject",
                  "url": "https://rumble.com/v1a-a.html",
                  "name": "V1",
                  "interactionCount": 1000
                },
                {
                  "@type": "VideoObject",
                  "url": "https://rumble.com/v1b-b.html",
                  "name": "V2",
                  "interactionCount": "5,000"
                }
              ]
            </script>
          </head>
          <body></body>
        </html>
      `;

      const results = engine.parseResponse(mockHtml, defaultParams);

      expect(results.results[0].views).toBe(1000);
      expect(results.results[1].views).toBe(5000);
    });
  });

  describe('live API test', () => {
    // Skipped by default - enable for manual testing
    it.skip('should search and return video results', async () => {
      const params = {
        page: 1,
        locale: 'en',
        safeSearch: 1 as const,
        timeRange: '' as const,
        engineData: {},
      };
      const config = engine.buildRequest('news', params);
      const res = await fetch(config.url, {
        headers: config.headers,
      });

      const body = await res.text();
      const results = engine.parseResponse(body, params);

      expect(results.results.length).toBeGreaterThan(0);
      const first = results.results[0];
      expect(first.url).toContain('rumble.com');
      expect(first.title).toBeTruthy();
      expect(first.engine).toBe('rumble');
      expect(first.category).toBe('videos');
    }, 30000);
  });
});
