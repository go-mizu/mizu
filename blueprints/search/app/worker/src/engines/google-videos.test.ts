import { describe, it, expect } from 'vitest';
import { GoogleVideosEngine } from './google-videos';

describe('GoogleVideosEngine', () => {
  const engine = new GoogleVideosEngine();

  describe('metadata', () => {
    it('should have correct name and shortcut', () => {
      expect(engine.name).toBe('google_videos');
      expect(engine.shortcut).toBe('gov');
    });

    it('should have correct categories', () => {
      expect(engine.categories).toContain('videos');
      expect(engine.categories.length).toBe(1);
    });

    it('should have correct paging settings', () => {
      expect(engine.supportsPaging).toBe(true);
      expect(engine.maxPage).toBe(50);
    });

    it('should have correct timeout and weight', () => {
      expect(engine.timeout).toBe(10000);
      expect(engine.weight).toBe(1.0);
    });

    it('should be enabled by default', () => {
      expect(engine.disabled).toBe(false);
    });
  });

  describe('buildRequest', () => {
    it('should build correct base URL with tbm=vid', () => {
      const config = engine.buildRequest('test query', {
        page: 1,
        locale: 'en',
        safeSearch: 1,
        timeRange: '',
        engineData: {},
      });
      expect(config.url).toContain('https://www.google.com/search');
      expect(config.url).toContain('q=test+query');
      expect(config.url).toContain('tbm=vid');
      expect(config.method).toBe('GET');
    });

    it('should include language code from locale', () => {
      const config = engine.buildRequest('test', {
        page: 1,
        locale: 'fr-FR',
        safeSearch: 1,
        timeRange: '',
        engineData: {},
      });
      expect(config.url).toContain('hl=fr');
    });

    it('should handle simple locale codes', () => {
      const config = engine.buildRequest('test', {
        page: 1,
        locale: 'de',
        safeSearch: 1,
        timeRange: '',
        engineData: {},
      });
      expect(config.url).toContain('hl=de');
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

    it('should include CONSENT cookie', () => {
      const config = engine.buildRequest('test', {
        page: 1,
        locale: 'en',
        safeSearch: 1,
        timeRange: '',
        engineData: {},
      });
      expect(config.cookies).toContain('CONSENT=YES+');
    });

    describe('pagination', () => {
      it('should not include start parameter for page 1', () => {
        const config = engine.buildRequest('test', {
          page: 1,
          locale: 'en',
          safeSearch: 1,
          timeRange: '',
          engineData: {},
        });
        expect(config.url).not.toContain('start=');
      });

      it('should include correct start parameter for page 2', () => {
        const config = engine.buildRequest('test', {
          page: 2,
          locale: 'en',
          safeSearch: 1,
          timeRange: '',
          engineData: {},
        });
        expect(config.url).toContain('start=10');
      });

      it('should include correct start parameter for page 5', () => {
        const config = engine.buildRequest('test', {
          page: 5,
          locale: 'en',
          safeSearch: 1,
          timeRange: '',
          engineData: {},
        });
        expect(config.url).toContain('start=40');
      });
    });

    describe('safe search', () => {
      it('should set safe=off when safeSearch is 0', () => {
        const config = engine.buildRequest('test', {
          page: 1,
          locale: 'en',
          safeSearch: 0,
          timeRange: '',
          engineData: {},
        });
        expect(config.url).toContain('safe=off');
      });

      it('should not set safe parameter when safeSearch is 1 (moderate)', () => {
        const config = engine.buildRequest('test', {
          page: 1,
          locale: 'en',
          safeSearch: 1,
          timeRange: '',
          engineData: {},
        });
        expect(config.url).not.toContain('safe=');
      });

      it('should set safe=active when safeSearch is 2 (strict)', () => {
        const config = engine.buildRequest('test', {
          page: 1,
          locale: 'en',
          safeSearch: 2,
          timeRange: '',
          engineData: {},
        });
        expect(config.url).toContain('safe=active');
      });
    });

    describe('tbs filters', () => {
      describe('time range', () => {
        it('should add qdr:d for day filter', () => {
          const config = engine.buildRequest('test', {
            page: 1,
            locale: 'en',
            safeSearch: 1,
            timeRange: 'day',
            engineData: {},
          });
          expect(config.url).toContain('tbs=qdr%3Ad');
        });

        it('should add qdr:w for week filter', () => {
          const config = engine.buildRequest('test', {
            page: 1,
            locale: 'en',
            safeSearch: 1,
            timeRange: 'week',
            engineData: {},
          });
          expect(config.url).toContain('tbs=qdr%3Aw');
        });

        it('should add qdr:m for month filter', () => {
          const config = engine.buildRequest('test', {
            page: 1,
            locale: 'en',
            safeSearch: 1,
            timeRange: 'month',
            engineData: {},
          });
          expect(config.url).toContain('tbs=qdr%3Am');
        });

        it('should add qdr:y for year filter', () => {
          const config = engine.buildRequest('test', {
            page: 1,
            locale: 'en',
            safeSearch: 1,
            timeRange: 'year',
            engineData: {},
          });
          expect(config.url).toContain('tbs=qdr%3Ay');
        });

        it('should not add tbs when timeRange is empty', () => {
          const config = engine.buildRequest('test', {
            page: 1,
            locale: 'en',
            safeSearch: 1,
            timeRange: '',
            engineData: {},
          });
          expect(config.url).not.toContain('tbs=');
        });
      });

      describe('duration filter', () => {
        it('should add dur:s for short duration (< 4 min)', () => {
          const config = engine.buildRequest('test', {
            page: 1,
            locale: 'en',
            safeSearch: 1,
            timeRange: '',
            engineData: {},
            videoFilters: { duration: 'short' },
          });
          expect(config.url).toContain('tbs=dur%3As');
        });

        it('should add dur:m for medium duration (4-20 min)', () => {
          const config = engine.buildRequest('test', {
            page: 1,
            locale: 'en',
            safeSearch: 1,
            timeRange: '',
            engineData: {},
            videoFilters: { duration: 'medium' },
          });
          expect(config.url).toContain('tbs=dur%3Am');
        });

        it('should add dur:l for long duration (> 20 min)', () => {
          const config = engine.buildRequest('test', {
            page: 1,
            locale: 'en',
            safeSearch: 1,
            timeRange: '',
            engineData: {},
            videoFilters: { duration: 'long' },
          });
          expect(config.url).toContain('tbs=dur%3Al');
        });

        it('should not add duration filter when duration is any', () => {
          const config = engine.buildRequest('test', {
            page: 1,
            locale: 'en',
            safeSearch: 1,
            timeRange: '',
            engineData: {},
            videoFilters: { duration: 'any' },
          });
          expect(config.url).not.toContain('dur%3A');
        });
      });

      describe('quality filter', () => {
        it('should add hq:h for HD quality', () => {
          const config = engine.buildRequest('test', {
            page: 1,
            locale: 'en',
            safeSearch: 1,
            timeRange: '',
            engineData: {},
            videoFilters: { quality: 'hd' },
          });
          expect(config.url).toContain('tbs=hq%3Ah');
        });

        it('should add hq:h for 4K quality', () => {
          const config = engine.buildRequest('test', {
            page: 1,
            locale: 'en',
            safeSearch: 1,
            timeRange: '',
            engineData: {},
            videoFilters: { quality: '4k' },
          });
          expect(config.url).toContain('tbs=hq%3Ah');
        });

        it('should not add quality filter when quality is any', () => {
          const config = engine.buildRequest('test', {
            page: 1,
            locale: 'en',
            safeSearch: 1,
            timeRange: '',
            engineData: {},
            videoFilters: { quality: 'any' },
          });
          expect(config.url).not.toContain('hq%3A');
        });
      });

      describe('closed captions filter', () => {
        it('should add cc:1 when cc is true', () => {
          const config = engine.buildRequest('test', {
            page: 1,
            locale: 'en',
            safeSearch: 1,
            timeRange: '',
            engineData: {},
            videoFilters: { cc: true },
          });
          expect(config.url).toContain('tbs=cc%3A1');
        });

        it('should not add cc filter when cc is false', () => {
          const config = engine.buildRequest('test', {
            page: 1,
            locale: 'en',
            safeSearch: 1,
            timeRange: '',
            engineData: {},
            videoFilters: { cc: false },
          });
          expect(config.url).not.toContain('cc%3A');
        });
      });

      describe('combined filters', () => {
        it('should combine time range and duration filters', () => {
          const config = engine.buildRequest('test', {
            page: 1,
            locale: 'en',
            safeSearch: 1,
            timeRange: 'week',
            engineData: {},
            videoFilters: { duration: 'medium' },
          });
          expect(config.url).toContain('tbs=');
          expect(config.url).toContain('qdr%3Aw');
          expect(config.url).toContain('dur%3Am');
        });

        it('should combine multiple video filters', () => {
          const config = engine.buildRequest('test', {
            page: 1,
            locale: 'en',
            safeSearch: 1,
            timeRange: '',
            engineData: {},
            videoFilters: { duration: 'long', quality: 'hd', cc: true },
          });
          expect(config.url).toContain('tbs=');
          expect(config.url).toContain('dur%3Al');
          expect(config.url).toContain('hq%3Ah');
          expect(config.url).toContain('cc%3A1');
        });

        it('should combine all filter types', () => {
          const config = engine.buildRequest('test', {
            page: 1,
            locale: 'en',
            safeSearch: 1,
            timeRange: 'month',
            engineData: {},
            videoFilters: { duration: 'short', quality: 'hd', cc: true },
          });
          expect(config.url).toContain('tbs=');
          // All filters should be present
          const tbsMatch = config.url.match(/tbs=([^&]+)/);
          expect(tbsMatch).toBeTruthy();
          const tbs = decodeURIComponent(tbsMatch![1]);
          expect(tbs).toContain('qdr:m');
          expect(tbs).toContain('dur:s');
          expect(tbs).toContain('hq:h');
          expect(tbs).toContain('cc:1');
        });
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

    it('should return empty results for CAPTCHA page', () => {
      const body = '<html><body>sorry.google.com redirect</body></html>';
      const results = engine.parseResponse(body, defaultParams);
      expect(results.results).toEqual([]);
    });

    it('should return empty results for /sorry/ page', () => {
      const body = '<html><head><meta http-equiv="refresh" content="0;url=/sorry/"></head></html>';
      const results = engine.parseResponse(body, defaultParams);
      expect(results.results).toEqual([]);
    });

    describe('JSON-LD parsing', () => {
      it('should parse VideoObject from JSON-LD', () => {
        const body = `
          <html>
          <head>
            <script type="application/ld+json">
            {
              "@type": "VideoObject",
              "name": "Test Video Title",
              "description": "Test video description",
              "thumbnailUrl": "https://example.com/thumb.jpg",
              "uploadDate": "2024-01-15",
              "duration": "PT5M30S",
              "contentUrl": "https://www.youtube.com/watch?v=abc123XYZ",
              "embedUrl": "https://www.youtube.com/embed/abc123XYZ",
              "author": {
                "@type": "Person",
                "name": "Test Channel"
              },
              "interactionStatistic": {
                "@type": "WatchAction",
                "interactionCount": 12500
              }
            }
            </script>
          </head>
          <body></body>
          </html>
        `;
        const results = engine.parseResponse(body, defaultParams);

        expect(results.results.length).toBe(1);
        const result = results.results[0];
        expect(result.title).toBe('Test Video Title');
        expect(result.content).toBe('Test video description');
        expect(result.thumbnailUrl).toBe('https://example.com/thumb.jpg');
        expect(result.url).toBe('https://www.youtube.com/watch?v=abc123XYZ');
        expect(result.duration).toBe('5:30');
        expect(result.channel).toBe('Test Channel');
        expect(result.views).toBe(12500);
        expect(result.publishedAt).toBe('2024-01-15');
        expect(result.engine).toBe('google_videos');
        expect(result.category).toBe('videos');
        expect(result.template).toBe('videos');
      });

      it('should parse ISO 8601 duration with hours', () => {
        const body = `
          <html>
          <head>
            <script type="application/ld+json">
            {
              "@type": "VideoObject",
              "name": "Long Video",
              "duration": "PT1H23M45S",
              "contentUrl": "https://example.com/video"
            }
            </script>
          </head>
          <body></body>
          </html>
        `;
        const results = engine.parseResponse(body, defaultParams);
        expect(results.results[0].duration).toBe('1:23:45');
      });

      it('should handle thumbnailUrl as array', () => {
        const body = `
          <html>
          <head>
            <script type="application/ld+json">
            {
              "@type": "VideoObject",
              "name": "Video with array thumbnails",
              "thumbnailUrl": ["https://example.com/thumb1.jpg", "https://example.com/thumb2.jpg"],
              "contentUrl": "https://example.com/video"
            }
            </script>
          </head>
          <body></body>
          </html>
        `;
        const results = engine.parseResponse(body, defaultParams);
        expect(results.results[0].thumbnailUrl).toBe('https://example.com/thumb1.jpg');
      });

      it('should parse VideoObject from @graph', () => {
        const body = `
          <html>
          <head>
            <script type="application/ld+json">
            {
              "@context": "https://schema.org",
              "@graph": [
                {
                  "@type": "WebPage",
                  "name": "Page Title"
                },
                {
                  "@type": "VideoObject",
                  "name": "Video in Graph",
                  "contentUrl": "https://example.com/video"
                }
              ]
            }
            </script>
          </head>
          <body></body>
          </html>
        `;
        const results = engine.parseResponse(body, defaultParams);
        expect(results.results.length).toBe(1);
        expect(results.results[0].title).toBe('Video in Graph');
      });

      it('should fallback to YouTube thumbnail when thumbnailUrl is missing', () => {
        const body = `
          <html>
          <head>
            <script type="application/ld+json">
            {
              "@type": "VideoObject",
              "name": "YouTube Video",
              "contentUrl": "https://www.youtube.com/watch?v=dQw4w9WgXcQ"
            }
            </script>
          </head>
          <body></body>
          </html>
        `;
        const results = engine.parseResponse(body, defaultParams);
        expect(results.results[0].thumbnailUrl).toBe('https://img.youtube.com/vi/dQw4w9WgXcQ/hqdefault.jpg');
      });

      it('should generate embed URL for YouTube videos', () => {
        const body = `
          <html>
          <head>
            <script type="application/ld+json">
            {
              "@type": "VideoObject",
              "name": "YouTube Video",
              "contentUrl": "https://www.youtube.com/watch?v=dQw4w9WgXcQ"
            }
            </script>
          </head>
          <body></body>
          </html>
        `;
        const results = engine.parseResponse(body, defaultParams);
        expect(results.results[0].embedUrl).toBe('https://www.youtube-nocookie.com/embed/dQw4w9WgXcQ');
      });
    });

    describe('HTML parsing fallback', () => {
      it('should parse div.g video results', () => {
        const body = `
          <html>
          <body>
            <div class="g">
              <a href="https://www.youtube.com/watch?v=test123abcd">
                <h3>Test Video Title</h3>
              </a>
              <div class="VwiC3b">This is the video description snippet</div>
            </div>
          </body>
          </html>
        `;
        const results = engine.parseResponse(body, defaultParams);

        expect(results.results.length).toBe(1);
        const result = results.results[0];
        expect(result.url).toBe('https://www.youtube.com/watch?v=test123abcd');
        expect(result.title).toBe('Test Video Title');
        expect(result.content).toBe('This is the video description snippet');
        expect(result.thumbnailUrl).toBe('https://img.youtube.com/vi/test123abcd/hqdefault.jpg');
        expect(result.embedUrl).toBe('https://www.youtube-nocookie.com/embed/test123abcd');
      });

      it('should unwrap Google redirect URLs', () => {
        const body = `
          <html>
          <body>
            <div class="g">
              <a href="/url?q=https://www.youtube.com/watch?v=abc123XYZ&sa=U">
                <h3>Redirected Video</h3>
              </a>
            </div>
          </body>
          </html>
        `;
        const results = engine.parseResponse(body, defaultParams);

        expect(results.results.length).toBe(1);
        expect(results.results[0].url).toBe('https://www.youtube.com/watch?v=abc123XYZ');
      });

      it('should extract duration from text', () => {
        const body = `
          <html>
          <body>
            <div class="g">
              <a href="https://example.com/video">
                <h3>Video with Duration</h3>
              </a>
              <span>Duration: 12:34</span>
            </div>
          </body>
          </html>
        `;
        const results = engine.parseResponse(body, defaultParams);
        expect(results.results[0].duration).toBe('12:34');
      });

      it('should filter out Google internal URLs', () => {
        const body = `
          <html>
          <body>
            <div class="g">
              <a href="https://www.google.com/search?q=related">
                <h3>Google Search Link</h3>
              </a>
            </div>
            <div class="g">
              <a href="https://www.youtube.com/watch?v=valid123">
                <h3>Valid Video</h3>
              </a>
            </div>
          </body>
          </html>
        `;
        const results = engine.parseResponse(body, defaultParams);

        expect(results.results.length).toBe(1);
        expect(results.results[0].url).toContain('youtube.com');
      });

      it('should handle youtu.be short URLs', () => {
        const body = `
          <html>
          <body>
            <div class="g">
              <a href="https://youtu.be/shortURL123">
                <h3>Short URL Video</h3>
              </a>
            </div>
          </body>
          </html>
        `;
        const results = engine.parseResponse(body, defaultParams);
        expect(results.results[0].thumbnailUrl).toBe('https://img.youtube.com/vi/shortURL123/hqdefault.jpg');
        expect(results.results[0].embedUrl).toBe('https://www.youtube-nocookie.com/embed/shortURL123');
      });

      it('should handle YouTube embed URLs', () => {
        const body = `
          <html>
          <body>
            <div class="g">
              <a href="https://www.youtube.com/embed/embedID1234">
                <h3>Embed URL Video</h3>
              </a>
            </div>
          </body>
          </html>
        `;
        const results = engine.parseResponse(body, defaultParams);
        expect(results.results[0].thumbnailUrl).toBe('https://img.youtube.com/vi/embedID1234/hqdefault.jpg');
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

      // Note: Google may return no results due to rate limiting or CAPTCHA
      // So we just verify the response was parsed without errors
      expect(results).toHaveProperty('results');
      expect(results).toHaveProperty('suggestions');
      expect(Array.isArray(results.results)).toBe(true);

      // If we got results, verify their structure
      if (results.results.length > 0) {
        const first = results.results[0];
        expect(first.url).toBeTruthy();
        expect(first.title).toBeTruthy();
        expect(first.engine).toBe('google_videos');
        expect(first.category).toBe('videos');
      }
    }, 30000);
  });
});
