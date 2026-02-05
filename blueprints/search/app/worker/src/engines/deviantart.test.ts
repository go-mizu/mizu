import { describe, it, expect } from 'vitest';
import { DeviantArtEngine } from './deviantart';
import type { EngineParams } from './engine';

function createParams(overrides?: Partial<EngineParams>): EngineParams {
  return {
    page: 1,
    locale: 'en-US',
    safeSearch: 1,
    timeRange: '',
    engineData: {},
    ...overrides,
  };
}

describe('DeviantArtEngine', () => {
  const engine = new DeviantArtEngine();

  it('should have correct metadata', () => {
    expect(engine.name).toBe('deviantart');
    expect(engine.shortcut).toBe('da');
    expect(engine.categories).toContain('images');
    expect(engine.supportsPaging).toBe(true);
    expect(engine.maxPage).toBe(50);
    expect(engine.timeout).toBe(10000);
    expect(engine.weight).toBe(0.8);
    expect(engine.disabled).toBe(false);
  });

  it('should build correct search URL', () => {
    const config = engine.buildRequest('digital art fantasy', createParams());
    expect(config.url).toContain('deviantart.com/search');
    expect(config.url).toContain('q=digital+art+fantasy');
    expect(config.method).toBe('GET');
    expect(config.headers['Accept']).toContain('text/html');
  });

  it('should build URL with pagination (offset)', () => {
    const config = engine.buildRequest('test', createParams({ page: 3 }));
    // Page 3 with 24 items per page = offset 48
    expect(config.url).toContain('offset=48');
  });

  it('should build URL with time range filter', () => {
    const configDay = engine.buildRequest('test', createParams({ timeRange: 'day' }));
    expect(configDay.url).toContain('order=popular-24hr');

    const configWeek = engine.buildRequest('test', createParams({ timeRange: 'week' }));
    expect(configWeek.url).toContain('order=popular-1week');

    const configMonth = engine.buildRequest('test', createParams({ timeRange: 'month' }));
    expect(configMonth.url).toContain('order=popular-1month');
  });

  it('should build URL with mature content filter', () => {
    const config = engine.buildRequest('test', createParams({ safeSearch: 1 }));
    expect(config.url).toContain('mature_content=false');
  });

  it('should not add mature filter when safe search is off', () => {
    const config = engine.buildRequest('test', createParams({ safeSearch: 0 }));
    expect(config.url).not.toContain('mature_content=false');
  });

  it('should set agegate cookie when safe search is enabled', () => {
    const config = engine.buildRequest('test', createParams({ safeSearch: 1 }));
    expect(config.cookies).toContain('agegate_state=1');
  });

  it('should parse Apollo state JSON', () => {
    const mockHtml = `
      <html>
        <script>
          window.__APOLLO_STATE__ = {
            "Deviation:12345": {
              "deviationId": 12345,
              "url": "https://www.deviantart.com/artist/art/Beautiful-Art-12345",
              "title": "Beautiful Digital Art",
              "media": {
                "baseUri": "https://images-wixmp-ed30a86b8c4ca887773594c2.wixmp.com/f/abc123/",
                "prettyName": "beautiful_art",
                "types": [
                  { "t": "fullview", "w": 1920, "h": 1080, "c": "v1/fill/w_1920,h_1080/beautiful_art.jpg" },
                  { "t": "150", "w": 150, "h": 84, "c": "v1/fill/w_150,h_84/beautiful_art.jpg" }
                ],
                "token": ["token123"]
              },
              "author": {
                "username": "DigitalArtist"
              },
              "extended": {
                "originalFile": {
                  "width": 1920,
                  "height": 1080
                }
              }
            },
            "Deviation:67890": {
              "deviationId": 67890,
              "url": "https://www.deviantart.com/other/art/Cool-Art-67890",
              "title": "Cool Art Piece",
              "media": {
                "baseUri": "https://images-wixmp-ed30a86b8c4ca887773594c2.wixmp.com/f/def456/",
                "types": [
                  { "t": "fullview", "w": 800, "h": 600, "c": "v1/fill/w_800,h_600/cool_art.jpg" }
                ],
                "token": ["token456"]
              },
              "author": {
                "username": "AnotherArtist"
              }
            }
          };
        </script>
      </html>
    `;

    const results = engine.parseResponse(mockHtml, createParams());

    expect(results.results.length).toBe(2);

    const first = results.results[0];
    expect(first.url).toBe('https://www.deviantart.com/artist/art/Beautiful-Art-12345');
    expect(first.title).toBe('Beautiful Digital Art');
    expect(first.imageUrl).toContain('w_1920,h_1080');
    expect(first.imageUrl).toContain('token123');
    expect(first.thumbnailUrl).toContain('w_150,h_84');
    expect(first.resolution).toBe('1920x1080');
    expect(first.source).toBe('DigitalArtist');
    expect(first.engine).toBe('deviantart');
    expect(first.category).toBe('images');

    const second = results.results[1];
    expect(second.title).toBe('Cool Art Piece');
    expect(second.source).toBe('AnotherArtist');
  });

  it('should parse HTML links as fallback', () => {
    const mockHtml = `
      <html>
        <body>
          <div>
            <a href="https://www.deviantart.com/artist1/art/Fantasy-Dragon-123456789">
              <img src="https://images-wixmp.wixmp.com/f/abc/v1/fill/w_300/dragon.jpg" />
            </a>
            <a href="https://www.deviantart.com/artist2/art/Space-Scene-987654321">
              <img src="https://images-wixmp.wixmp.com/f/def/v1/fill/w_300/space.jpg" />
            </a>
          </div>
        </body>
      </html>
    `;

    const results = engine.parseResponse(mockHtml, createParams());

    expect(results.results.length).toBe(2);

    const first = results.results[0];
    expect(first.url).toContain('deviantart.com/artist1/art/Fantasy-Dragon');
    expect(first.title).toBe('Fantasy Dragon');
    expect(first.source).toBe('artist1');
    expect(first.engine).toBe('deviantart');
  });

  it('should apply client-side size filtering', () => {
    const mockHtml = `
      <script>
        window.__APOLLO_STATE__ = {
          "Deviation:111": {
            "deviationId": 111,
            "url": "https://www.deviantart.com/a/art/Small-111",
            "title": "Small Image",
            "media": {
              "baseUri": "https://images.wixmp.com/f/small/",
              "types": [{ "t": "fullview", "w": 400, "h": 300, "c": "small.jpg" }]
            },
            "author": { "username": "user1" }
          },
          "Deviation:222": {
            "deviationId": 222,
            "url": "https://www.deviantart.com/a/art/Large-222",
            "title": "Large Image",
            "media": {
              "baseUri": "https://images.wixmp.com/f/large/",
              "types": [{ "t": "fullview", "w": 2000, "h": 1500, "c": "large.jpg" }]
            },
            "author": { "username": "user2" }
          }
        };
      </script>
    `;

    const params = createParams({
      imageFilters: { size: 'large' },
    });

    const results = engine.parseResponse(mockHtml, params);
    expect(results.results.length).toBe(1);
    expect(results.results[0].title).toBe('Large Image');
  });

  it('should apply aspect ratio filtering', () => {
    const mockHtml = `
      <script>
        window.__APOLLO_STATE__ = {
          "Deviation:111": {
            "deviationId": 111,
            "url": "https://www.deviantart.com/a/art/Portrait-111",
            "title": "Portrait Image",
            "media": {
              "baseUri": "https://images.wixmp.com/f/portrait/",
              "types": [{ "t": "fullview", "w": 600, "h": 1000, "c": "portrait.jpg" }]
            },
            "author": { "username": "user1" }
          },
          "Deviation:222": {
            "deviationId": 222,
            "url": "https://www.deviantart.com/a/art/Landscape-222",
            "title": "Landscape Image",
            "media": {
              "baseUri": "https://images.wixmp.com/f/landscape/",
              "types": [{ "t": "fullview", "w": 1920, "h": 1080, "c": "landscape.jpg" }]
            },
            "author": { "username": "user2" }
          }
        };
      </script>
    `;

    const params = createParams({
      imageFilters: { aspect: 'wide' },
    });

    const results = engine.parseResponse(mockHtml, params);
    expect(results.results.length).toBe(1);
    expect(results.results[0].title).toBe('Landscape Image');
  });

  it('should handle empty response', () => {
    const mockHtml = `
      <html>
        <body>
          <p>No results found</p>
        </body>
      </html>
    `;

    const results = engine.parseResponse(mockHtml, createParams());
    expect(results.results).toEqual([]);
  });

  it('should handle malformed state JSON gracefully', () => {
    const mockHtml = `
      <script>
        window.__APOLLO_STATE__ = { invalid json
      </script>
    `;

    // Should not throw
    const results = engine.parseResponse(mockHtml, createParams());
    expect(Array.isArray(results.results)).toBe(true);
  });

  // Live search test - skipped by default
  it.skip('should search and return image results (live)', async () => {
    const config = engine.buildRequest('digital art', createParams());
    const res = await fetch(config.url, { headers: config.headers });
    const body = await res.text();
    const results = engine.parseResponse(body, createParams());

    expect(results.results.length).toBeGreaterThan(0);
    const first = results.results[0];
    expect(first.url).toContain('deviantart.com');
    expect(first.title).toBeTruthy();
    expect(first.category).toBe('images');
  }, 15000);
});
