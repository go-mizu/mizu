import { describe, it, expect } from 'vitest';
import { VimeoEngine } from './vimeo';

describe('VimeoEngine', () => {
  const engine = new VimeoEngine();

  it('should have correct metadata', () => {
    expect(engine.name).toBe('vimeo');
    expect(engine.shortcut).toBe('vm');
    expect(engine.categories).toContain('videos');
    expect(engine.supportsPaging).toBe(true);
    expect(engine.maxPage).toBe(10);
    expect(engine.timeout).toBe(8000);
    expect(engine.weight).toBe(0.9);
    expect(engine.disabled).toBe(false);
  });

  it('should build correct search URL', () => {
    const config = engine.buildRequest('javascript tutorial', {
      page: 1,
      locale: 'en',
      safeSearch: 1,
      timeRange: '',
      engineData: {},
    });
    expect(config.url).toBe('https://vimeo.com/search?q=javascript+tutorial');
    expect(config.method).toBe('GET');
    expect(config.headers['Accept']).toContain('text/html');
  });

  it('should build URL with pagination', () => {
    const config = engine.buildRequest('test', {
      page: 3,
      locale: 'en',
      safeSearch: 1,
      timeRange: '',
      engineData: {},
    });
    expect(config.url).toBe('https://vimeo.com/search?q=test&page=3');
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

  it('should parse embedded JSON data', () => {
    // Simulate Vimeo's data-search-data attribute structure
    const mockHtml = `
      <html>
        <body>
          <div data-search-data="${escapeAttr(
            JSON.stringify({
              filtered: {
                data: [
                  {
                    clip_id: 123456789,
                    name: 'JavaScript Tutorial for Beginners',
                    description: 'Learn JavaScript from scratch',
                    link: 'https://vimeo.com/123456789',
                    duration: 3661,
                    pictures: {
                      sizes: [
                        { link: 'https://i.vimeocdn.com/video/123_100.jpg', width: 100 },
                        { link: 'https://i.vimeocdn.com/video/123_640.jpg', width: 640 },
                      ],
                    },
                    user: { name: 'CodeChannel' },
                    stats: { plays: 50000 },
                  },
                  {
                    clip_id: 987654321,
                    name: 'Advanced JS Patterns',
                    description: 'Deep dive into JavaScript patterns',
                    link: 'https://vimeo.com/987654321',
                    duration: 1800,
                    pictures: {
                      sizes: [
                        { link: 'https://i.vimeocdn.com/video/987_640.jpg', width: 640 },
                      ],
                    },
                    user: { name: 'DevTalks' },
                    stats: { plays: 25000 },
                  },
                ],
              },
            })
          )}">
          </div>
        </body>
      </html>
    `;

    const params = {
      page: 1,
      locale: 'en',
      safeSearch: 1 as const,
      timeRange: '' as const,
      engineData: {},
    };

    const results = engine.parseResponse(mockHtml, params);

    expect(results.results.length).toBe(2);

    const first = results.results[0];
    expect(first.url).toBe('https://vimeo.com/123456789');
    expect(first.title).toBe('JavaScript Tutorial for Beginners');
    expect(first.content).toBe('Learn JavaScript from scratch');
    expect(first.embedUrl).toBe('https://player.vimeo.com/video/123456789');
    expect(first.thumbnailUrl).toBe('https://i.vimeocdn.com/video/123_640.jpg');
    expect(first.channel).toBe('CodeChannel');
    expect(first.views).toBe(50000);
    expect(first.duration).toBe('1:01:01');
    expect(first.engine).toBe('vimeo');
    expect(first.category).toBe('videos');

    const second = results.results[1];
    expect(second.url).toBe('https://vimeo.com/987654321');
    expect(second.title).toBe('Advanced JS Patterns');
    expect(second.duration).toBe('30:00');
  });

  it('should parse HTML video links as fallback', () => {
    const mockHtml = `
      <html>
        <body>
          <a href="https://vimeo.com/111222333">Sample Video Title</a>
          <a href="https://vimeo.com/444555666">Another Video</a>
        </body>
      </html>
    `;

    const params = {
      page: 1,
      locale: 'en',
      safeSearch: 1 as const,
      timeRange: '' as const,
      engineData: {},
    };

    const results = engine.parseResponse(mockHtml, params);

    expect(results.results.length).toBe(2);

    const first = results.results[0];
    expect(first.url).toBe('https://vimeo.com/111222333');
    expect(first.title).toBe('Sample Video Title');
    expect(first.embedUrl).toBe('https://player.vimeo.com/video/111222333');
  });

  it('should parse JSON-LD VideoObject data', () => {
    const mockHtml = `
      <html>
        <head>
          <script type="application/ld+json">
            {
              "@type": "VideoObject",
              "url": "https://vimeo.com/555666777",
              "name": "LD JSON Video",
              "description": "Video from JSON-LD",
              "thumbnailUrl": "https://i.vimeocdn.com/video/555_640.jpg",
              "duration": "PT5M30S",
              "author": { "name": "LD Author" }
            }
          </script>
        </head>
        <body></body>
      </html>
    `;

    const params = {
      page: 1,
      locale: 'en',
      safeSearch: 1 as const,
      timeRange: '' as const,
      engineData: {},
    };

    const results = engine.parseResponse(mockHtml, params);

    expect(results.results.length).toBe(1);

    const first = results.results[0];
    expect(first.url).toBe('https://vimeo.com/555666777');
    expect(first.title).toBe('LD JSON Video');
    expect(first.content).toBe('Video from JSON-LD');
    expect(first.duration).toBe('5:30');
    expect(first.channel).toBe('LD Author');
    expect(first.embedUrl).toBe('https://player.vimeo.com/video/555666777');
  });

  it('should return empty results for Cloudflare-protected pages', () => {
    const cloudflareHtml = `
      <!DOCTYPE html>
      <html lang="en-US">
        <head><title>Just a moment...</title></head>
        <body>
          <noscript>
            <div class="h2">
              <span id="challenge-error-text">Enable JavaScript and cookies to continue</span>
            </div>
          </noscript>
        </body>
      </html>
    `;

    const params = {
      page: 1,
      locale: 'en',
      safeSearch: 1 as const,
      timeRange: '' as const,
      engineData: {},
    };

    const results = engine.parseResponse(cloudflareHtml, params);

    // Should gracefully return empty results when blocked
    expect(results.results).toEqual([]);
  });

  // Live search test - may fail due to Cloudflare protection
  it.skip('should search and return video results (live)', async () => {
    const results = await fetchAndParse(engine, 'javascript');

    expect(results.results.length).toBeGreaterThan(0);
    const first = results.results[0];
    expect(first.url).toContain('vimeo.com');
    expect(first.title).toBeTruthy();
    expect(first.embedUrl).toContain('player.vimeo.com/video/');
  }, 30000);
});

function escapeAttr(str: string): string {
  return str
    .replace(/&/g, '&amp;')
    .replace(/"/g, '&quot;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/'/g, '&#39;');
}

async function fetchAndParse(engine: VimeoEngine, query: string) {
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
