import { describe, it, expect } from 'vitest';
import { BingVideosEngine } from './bing-videos';

describe('BingVideosEngine', () => {
  const engine = new BingVideosEngine();

  describe('metadata', () => {
    it('should have correct name and shortcut', () => {
      expect(engine.name).toBe('bing_videos');
      expect(engine.shortcut).toBe('biv');
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
      expect(engine.weight).toBe(0.95);
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
      expect(config.url).toContain('https://www.bing.com/videos/search');
      expect(config.url).toContain('q=test+query');
      expect(config.method).toBe('GET');
    });

    it('should include count parameter', () => {
      const config = engine.buildRequest('test', {
        page: 1,
        locale: 'en',
        safeSearch: 1,
        timeRange: '',
        engineData: {},
      });
      expect(config.url).toContain('count=35');
    });

    it('should include Chrome User-Agent header', () => {
      const config = engine.buildRequest('test', {
        page: 1,
        locale: 'en',
        safeSearch: 1,
        timeRange: '',
        engineData: {},
      });
      expect(config.headers['User-Agent']).toContain('Chrome');
    });

    it('should include Accept-Language header', () => {
      const config = engine.buildRequest('test', {
        page: 1,
        locale: 'en',
        safeSearch: 1,
        timeRange: '',
        engineData: {},
      });
      expect(config.headers['Accept-Language']).toBe('en-US,en;q=0.9');
    });

    describe('pagination', () => {
      it('should set first=1 for page 1', () => {
        const config = engine.buildRequest('test', {
          page: 1,
          locale: 'en',
          safeSearch: 1,
          timeRange: '',
          engineData: {},
        });
        expect(config.url).toContain('first=1');
      });

      it('should set first=36 for page 2 ((2-1)*35+1)', () => {
        const config = engine.buildRequest('test', {
          page: 2,
          locale: 'en',
          safeSearch: 1,
          timeRange: '',
          engineData: {},
        });
        expect(config.url).toContain('first=36');
      });

      it('should set first=71 for page 3 ((3-1)*35+1)', () => {
        const config = engine.buildRequest('test', {
          page: 3,
          locale: 'en',
          safeSearch: 1,
          timeRange: '',
          engineData: {},
        });
        expect(config.url).toContain('first=71');
      });

      it('should set first=316 for page 10 ((10-1)*35+1)', () => {
        const config = engine.buildRequest('test', {
          page: 10,
          locale: 'en',
          safeSearch: 1,
          timeRange: '',
          engineData: {},
        });
        expect(config.url).toContain('first=316');
      });
    });

    describe('time range filters', () => {
      it('should add videoage-lt1440 for day filter', () => {
        const config = engine.buildRequest('test', {
          page: 1,
          locale: 'en',
          safeSearch: 1,
          timeRange: 'day',
          engineData: {},
        });
        expect(config.url).toContain('filters=filterui%3Avideoage-lt1440');
        expect(config.url).toContain('form=VRFLTR');
      });

      it('should add videoage-lt10080 for week filter', () => {
        const config = engine.buildRequest('test', {
          page: 1,
          locale: 'en',
          safeSearch: 1,
          timeRange: 'week',
          engineData: {},
        });
        expect(config.url).toContain('filters=filterui%3Avideoage-lt10080');
        expect(config.url).toContain('form=VRFLTR');
      });

      it('should add videoage-lt43200 for month filter', () => {
        const config = engine.buildRequest('test', {
          page: 1,
          locale: 'en',
          safeSearch: 1,
          timeRange: 'month',
          engineData: {},
        });
        expect(config.url).toContain('filters=filterui%3Avideoage-lt43200');
        expect(config.url).toContain('form=VRFLTR');
      });

      it('should add videoage-lt525600 for year filter', () => {
        const config = engine.buildRequest('test', {
          page: 1,
          locale: 'en',
          safeSearch: 1,
          timeRange: 'year',
          engineData: {},
        });
        expect(config.url).toContain('filters=filterui%3Avideoage-lt525600');
        expect(config.url).toContain('form=VRFLTR');
      });

      it('should not add filters when timeRange is empty', () => {
        const config = engine.buildRequest('test', {
          page: 1,
          locale: 'en',
          safeSearch: 1,
          timeRange: '',
          engineData: {},
        });
        expect(config.url).not.toContain('filters=');
        expect(config.url).not.toContain('form=VRFLTR');
      });
    });

    describe('safe search via cookies', () => {
      it('should set ADLT=off cookie when safeSearch is 0', () => {
        const config = engine.buildRequest('test', {
          page: 1,
          locale: 'en',
          safeSearch: 0,
          timeRange: '',
          engineData: {},
        });
        expect(config.cookies).toContain('SRCHHPGUSR=ADLT=off');
      });

      it('should set ADLT=moderate cookie when safeSearch is 1', () => {
        const config = engine.buildRequest('test', {
          page: 1,
          locale: 'en',
          safeSearch: 1,
          timeRange: '',
          engineData: {},
        });
        expect(config.cookies).toContain('SRCHHPGUSR=ADLT=moderate');
      });

      it('should set ADLT=strict cookie when safeSearch is 2', () => {
        const config = engine.buildRequest('test', {
          page: 1,
          locale: 'en',
          safeSearch: 2,
          timeRange: '',
          engineData: {},
        });
        expect(config.cookies).toContain('SRCHHPGUSR=ADLT=strict');
      });
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

    it('should return empty results for sorry page', () => {
      const body = '<html><body>sorry.bing.com redirect</body></html>';
      const results = engine.parseResponse(body, defaultParams);
      expect(results.results).toEqual([]);
    });

    it('should return empty results for /sorry/ page', () => {
      const body = '<html><head><meta http-equiv="refresh" content="0;url=/sorry/"></head></html>';
      const results = engine.parseResponse(body, defaultParams);
      expect(results.results).toEqual([]);
    });

    describe('vrhm attribute parsing', () => {
      it('should parse video metadata from vrhm attribute', () => {
        const metadata = {
          murl: 'https://www.youtube.com/watch?v=abc123XYZ',
          vt: 'Test Video Title',
          du: '5:30',
          thid: 'OIP.testThumbId',
          desc: 'Video description here',
          ch: 'Test Channel',
          vw: '50K views',
        };
        const encodedJson = encodeURIComponent(JSON.stringify(metadata))
          .replace(/'/g, '&#39;')
          .replace(/"/g, '&quot;');

        const body = `
          <html>
          <body>
            <div vrhm="${encodedJson}">
              <a href="${metadata.murl}">Video Link</a>
            </div>
          </body>
          </html>
        `;

        const results = engine.parseResponse(body, defaultParams);

        expect(results.results.length).toBe(1);
        const result = results.results[0];
        expect(result.url).toBe('https://www.youtube.com/watch?v=abc123XYZ');
        expect(result.title).toBe('Test Video Title');
        expect(result.content).toBe('Video description here');
        expect(result.duration).toBe('5:30');
        expect(result.thumbnailUrl).toBe('https://tse1.mm.bing.net/th?id=OIP.testThumbId');
        expect(result.channel).toBe('Test Channel');
        expect(result.views).toBe(50000);
        expect(result.engine).toBe('bing_videos');
        expect(result.category).toBe('videos');
        expect(result.template).toBe('videos');
      });

      it('should generate YouTube embed URL for YouTube videos', () => {
        const metadata = {
          murl: 'https://www.youtube.com/watch?v=dQw4w9WgXcQ',
          vt: 'YouTube Video',
          thid: 'test123',
        };
        const encodedJson = encodeURIComponent(JSON.stringify(metadata));

        const body = `<div vrhm="${encodedJson}"></div>`;
        const results = engine.parseResponse(body, defaultParams);

        expect(results.results.length).toBe(1);
        expect(results.results[0].embedUrl).toBe('https://www.youtube-nocookie.com/embed/dQw4w9WgXcQ');
      });

      it('should handle youtu.be short URLs for embed', () => {
        const metadata = {
          murl: 'https://youtu.be/dQw4w9WgXcQ',
          vt: 'Short URL Video',
          thid: 'test123',
        };
        const encodedJson = encodeURIComponent(JSON.stringify(metadata));

        const body = `<div vrhm="${encodedJson}"></div>`;
        const results = engine.parseResponse(body, defaultParams);

        expect(results.results.length).toBe(1);
        expect(results.results[0].embedUrl).toBe('https://www.youtube-nocookie.com/embed/dQw4w9WgXcQ');
      });

      it('should parse multiple video results', () => {
        const metadata1 = {
          murl: 'https://example.com/video1',
          vt: 'Video One',
          thid: 'thumb1',
        };
        const metadata2 = {
          murl: 'https://example.com/video2',
          vt: 'Video Two',
          thid: 'thumb2',
        };
        const body = `
          <div vrhm="${encodeURIComponent(JSON.stringify(metadata1))}"></div>
          <div vrhm="${encodeURIComponent(JSON.stringify(metadata2))}"></div>
        `;

        const results = engine.parseResponse(body, defaultParams);

        expect(results.results.length).toBe(2);
        expect(results.results[0].title).toBe('Video One');
        expect(results.results[1].title).toBe('Video Two');
      });

      it('should skip results without murl', () => {
        const metadata = {
          vt: 'No URL Video',
          thid: 'thumb1',
        };
        const body = `<div vrhm="${encodeURIComponent(JSON.stringify(metadata))}"></div>`;

        const results = engine.parseResponse(body, defaultParams);
        expect(results.results.length).toBe(0);
      });

      it('should skip results without title', () => {
        const metadata = {
          murl: 'https://example.com/video',
          thid: 'thumb1',
        };
        const body = `<div vrhm="${encodeURIComponent(JSON.stringify(metadata))}"></div>`;

        const results = engine.parseResponse(body, defaultParams);
        expect(results.results.length).toBe(0);
      });
    });

    describe('duration parsing', () => {
      it('should parse MM:SS duration format', () => {
        const metadata = {
          murl: 'https://example.com/video',
          vt: 'Duration Test',
          du: '5:30',
          thid: 'thumb1',
        };
        const body = `<div vrhm="${encodeURIComponent(JSON.stringify(metadata))}"></div>`;

        const results = engine.parseResponse(body, defaultParams);
        expect(results.results[0].duration).toBe('5:30');
      });

      it('should parse H:MM:SS duration format', () => {
        const metadata = {
          murl: 'https://example.com/video',
          vt: 'Long Video',
          du: '1:23:45',
          thid: 'thumb1',
        };
        const body = `<div vrhm="${encodeURIComponent(JSON.stringify(metadata))}"></div>`;

        const results = engine.parseResponse(body, defaultParams);
        expect(results.results[0].duration).toBe('1:23:45');
      });

      it('should parse numeric seconds duration', () => {
        const metadata = {
          murl: 'https://example.com/video',
          vt: 'Numeric Duration',
          du: '330',  // 5 minutes 30 seconds
          thid: 'thumb1',
        };
        const body = `<div vrhm="${encodeURIComponent(JSON.stringify(metadata))}"></div>`;

        const results = engine.parseResponse(body, defaultParams);
        expect(results.results[0].duration).toBe('5:30');
      });

      it('should handle missing duration', () => {
        const metadata = {
          murl: 'https://example.com/video',
          vt: 'No Duration',
          thid: 'thumb1',
        };
        const body = `<div vrhm="${encodeURIComponent(JSON.stringify(metadata))}"></div>`;

        const results = engine.parseResponse(body, defaultParams);
        expect(results.results[0].duration).toBe('');
      });
    });

    describe('view count parsing', () => {
      it('should parse K suffix (thousands)', () => {
        const metadata = {
          murl: 'https://example.com/video',
          vt: 'Popular Video',
          vw: '50K views',
          thid: 'thumb1',
        };
        const body = `<div vrhm="${encodeURIComponent(JSON.stringify(metadata))}"></div>`;

        const results = engine.parseResponse(body, defaultParams);
        expect(results.results[0].views).toBe(50000);
      });

      it('should parse M suffix (millions)', () => {
        const metadata = {
          murl: 'https://example.com/video',
          vt: 'Viral Video',
          vw: '2.5M views',
          thid: 'thumb1',
        };
        const body = `<div vrhm="${encodeURIComponent(JSON.stringify(metadata))}"></div>`;

        const results = engine.parseResponse(body, defaultParams);
        expect(results.results[0].views).toBe(2500000);
      });

      it('should parse plain numbers', () => {
        const metadata = {
          murl: 'https://example.com/video',
          vt: 'Small Video',
          vw: '1234',
          thid: 'thumb1',
        };
        const body = `<div vrhm="${encodeURIComponent(JSON.stringify(metadata))}"></div>`;

        const results = engine.parseResponse(body, defaultParams);
        expect(results.results[0].views).toBe(1234);
      });

      it('should handle missing view count', () => {
        const metadata = {
          murl: 'https://example.com/video',
          vt: 'No Views Video',
          thid: 'thumb1',
        };
        const body = `<div vrhm="${encodeURIComponent(JSON.stringify(metadata))}"></div>`;

        const results = engine.parseResponse(body, defaultParams);
        expect(results.results[0].views).toBeUndefined();
      });
    });

    describe('thumbnail URL building', () => {
      it('should build correct thumbnail URL from thid', () => {
        const metadata = {
          murl: 'https://example.com/video',
          vt: 'Thumbnail Test',
          thid: 'OIP.abc123xyz',
        };
        const body = `<div vrhm="${encodeURIComponent(JSON.stringify(metadata))}"></div>`;

        const results = engine.parseResponse(body, defaultParams);
        expect(results.results[0].thumbnailUrl).toBe('https://tse1.mm.bing.net/th?id=OIP.abc123xyz');
      });

      it('should return empty thumbnail URL when thid is missing', () => {
        const metadata = {
          murl: 'https://example.com/video',
          vt: 'No Thumbnail',
        };
        const body = `<div vrhm="${encodeURIComponent(JSON.stringify(metadata))}"></div>`;

        const results = engine.parseResponse(body, defaultParams);
        expect(results.results[0].thumbnailUrl).toBe('');
      });
    });

    describe('murl fallback parsing', () => {
      it('should parse murl from JSON string patterns', () => {
        const body = `
          <html>
          <body>
            <script>
              var data = {"murl":"https://example.com/video1","vt":"Fallback Video"};
            </script>
          </body>
          </html>
        `;

        const results = engine.parseResponse(body, defaultParams);

        expect(results.results.length).toBeGreaterThanOrEqual(1);
        expect(results.results[0].url).toBe('https://example.com/video1');
      });

      it('should parse murl from HTML-encoded JSON', () => {
        const body = `
          <html>
          <body>
            <div data-info="murl&quot;:&quot;https://example.com/video2&quot;">
            </div>
          </body>
          </html>
        `;

        const results = engine.parseResponse(body, defaultParams);

        expect(results.results.length).toBeGreaterThanOrEqual(1);
        expect(results.results[0].url).toBe('https://example.com/video2');
      });
    });

    it('should return empty results for empty body', () => {
      const results = engine.parseResponse('', defaultParams);
      expect(results.results).toEqual([]);
    });

    it('should return empty suggestions and corrections', () => {
      const body = '<html><body></body></html>';
      const results = engine.parseResponse(body, defaultParams);
      expect(results.suggestions).toEqual([]);
      expect(results.corrections).toEqual([]);
    });

    it('should decode HTML entities in metadata', () => {
      const metadata = {
        murl: 'https://example.com/video',
        vt: 'Video &amp; More',
        desc: 'Description with &quot;quotes&quot;',
        ch: 'Channel &lt;Test&gt;',
        thid: 'thumb1',
      };
      const body = `<div vrhm="${encodeURIComponent(JSON.stringify(metadata))}"></div>`;

      const results = engine.parseResponse(body, defaultParams);

      expect(results.results[0].title).toBe('Video & More');
      expect(results.results[0].content).toBe('Description with "quotes"');
      expect(results.results[0].channel).toBe('Channel <Test>');
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

      const headers = new Headers(config.headers);
      if (config.cookies.length > 0) {
        headers.set('Cookie', config.cookies.join('; '));
      }

      const res = await fetch(config.url, {
        headers,
      });

      expect(res.ok).toBe(true);

      const body = await res.text();
      const results = engine.parseResponse(body, params);

      // Bing may return different formats, but we should parse without errors
      expect(results).toHaveProperty('results');
      expect(results).toHaveProperty('suggestions');
      expect(Array.isArray(results.results)).toBe(true);

      // If we got results, verify their structure
      if (results.results.length > 0) {
        const first = results.results[0];
        expect(first.url).toBeTruthy();
        expect(first.title).toBeTruthy();
        expect(first.engine).toBe('bing_videos');
        expect(first.category).toBe('videos');
      }
    }, 30000);

    it('should handle pagination', async () => {
      const params = {
        page: 2,
        locale: 'en',
        safeSearch: 1 as const,
        timeRange: '' as const,
        engineData: {},
      };
      const config = engine.buildRequest('python programming', params);

      const headers = new Headers(config.headers);
      if (config.cookies.length > 0) {
        headers.set('Cookie', config.cookies.join('; '));
      }

      const res = await fetch(config.url, {
        headers,
      });

      expect(res.ok).toBe(true);
      expect(config.url).toContain('first=36');
    }, 30000);

    it('should handle time range filters', async () => {
      const params = {
        page: 1,
        locale: 'en',
        safeSearch: 1 as const,
        timeRange: 'week' as const,
        engineData: {},
      };
      const config = engine.buildRequest('news today', params);

      const headers = new Headers(config.headers);
      if (config.cookies.length > 0) {
        headers.set('Cookie', config.cookies.join('; '));
      }

      const res = await fetch(config.url, {
        headers,
      });

      expect(res.ok).toBe(true);
      expect(config.url).toContain('filters=filterui%3Avideoage-lt10080');
    }, 30000);
  });
});
