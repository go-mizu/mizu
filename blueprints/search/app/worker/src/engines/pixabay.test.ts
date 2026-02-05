import { describe, it, expect } from 'vitest';
import { PixabayEngine } from './pixabay';
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

describe('PixabayEngine', () => {
  const engine = new PixabayEngine();

  it('should have correct metadata', () => {
    expect(engine.name).toBe('pixabay');
    expect(engine.shortcut).toBe('px');
    expect(engine.categories).toContain('images');
    expect(engine.supportsPaging).toBe(true);
    expect(engine.maxPage).toBe(50);
    expect(engine.timeout).toBe(8000);
    expect(engine.weight).toBe(0.85);
    expect(engine.disabled).toBe(false);
  });

  it('should build correct search URL', () => {
    const config = engine.buildRequest('nature forest', createParams());
    expect(config.url).toContain('pixabay.com/images/search/');
    expect(config.url).toContain('nature%20forest');
    expect(config.method).toBe('GET');
    expect(config.headers['Accept']).toContain('text/html');
  });

  it('should build URL with pagination', () => {
    const config = engine.buildRequest('test', createParams({ page: 3 }));
    expect(config.url).toContain('pagi=3');
  });

  it('should build URL with safe search enabled', () => {
    const config = engine.buildRequest('test', createParams({ safeSearch: 1 }));
    expect(config.url).toContain('safesearch=true');
  });

  it('should build URL without safe search when disabled', () => {
    const config = engine.buildRequest('test', createParams({ safeSearch: 0 }));
    expect(config.url).not.toContain('safesearch=true');
  });

  it('should build URL with color filter', () => {
    const config = engine.buildRequest('flowers', createParams({
      imageFilters: { color: 'red' },
    }));
    expect(config.url).toContain('colors=red');
  });

  it('should build URL with transparent color filter', () => {
    const config = engine.buildRequest('icons', createParams({
      imageFilters: { color: 'transparent' },
    }));
    expect(config.url).toContain('colors=transparent');
  });

  it('should build URL with image type filter (photo)', () => {
    const config = engine.buildRequest('nature', createParams({
      imageFilters: { type: 'photo' },
    }));
    expect(config.url).toContain('image_type=photo');
  });

  it('should build URL with image type filter (vector)', () => {
    const config = engine.buildRequest('icons', createParams({
      imageFilters: { type: 'lineart' },
    }));
    expect(config.url).toContain('image_type=vector');
  });

  it('should build URL with orientation filter', () => {
    const config = engine.buildRequest('landscape', createParams({
      imageFilters: { aspect: 'wide' },
    }));
    expect(config.url).toContain('orientation=horizontal');
  });

  it('should build URL with min dimensions', () => {
    const config = engine.buildRequest('wallpaper', createParams({
      imageFilters: { minWidth: 1920, minHeight: 1080 },
    }));
    expect(config.url).toContain('min_width=1920');
    expect(config.url).toContain('min_height=1080');
  });

  it('should parse JSON-style embedded data', () => {
    // The JSON must be on one line for the regex to match
    const jsonData = JSON.stringify({
      images: [
        {
          id: 123456,
          pageURL: 'https://pixabay.com/photos/flower-rose-red-123456/',
          largeImageURL: 'https://pixabay.com/get/flower_1280.jpg',
          webformatURL: 'https://pixabay.com/get/flower_640.jpg',
          previewURL: 'https://pixabay.com/get/flower_150.jpg',
          tags: 'flower, rose, red',
          imageWidth: 4000,
          imageHeight: 3000,
          user: 'photographer123',
        },
      ],
    });
    const mockHtml = `
      <html>
        <head>
          <script type="application/json">${jsonData}</script>
        </head>
        <body></body>
      </html>
    `;

    const results = engine.parseResponse(mockHtml, createParams());

    expect(results.results.length).toBe(1);

    const first = results.results[0];
    expect(first.url).toBe('https://pixabay.com/photos/flower-rose-red-123456/');
    expect(first.title).toBe('flower, rose, red');
    expect(first.imageUrl).toBe('https://pixabay.com/get/flower_1280.jpg');
    expect(first.thumbnailUrl).toBe('https://pixabay.com/get/flower_150.jpg');
    expect(first.resolution).toBe('4000x3000');
    expect(first.source).toBe('photographer123');
    expect(first.engine).toBe('pixabay');
    expect(first.category).toBe('images');
  });

  it('should parse HTML links as fallback', () => {
    const mockHtml = `
      <html>
        <body>
          <div>
            <a href="/photos/sunset-ocean-beach-12345678/">
              <img src="https://cdn.pixabay.com/photo/sunset_150.jpg" alt="Sunset over the ocean" />
            </a>
            <a href="/photos/mountain-snow-peak-87654321/">
              <img src="https://cdn.pixabay.com/photo/mountain_150.jpg" alt="Snow capped mountain" />
            </a>
          </div>
        </body>
      </html>
    `;

    const results = engine.parseResponse(mockHtml, createParams());

    expect(results.results.length).toBe(2);

    const first = results.results[0];
    expect(first.url).toBe('https://pixabay.com/photos/sunset-ocean-beach-12345678/');
    expect(first.title).toBe('Sunset over the ocean');
    expect(first.source).toBe('Pixabay');
    expect(first.engine).toBe('pixabay');
  });

  it('should apply client-side size filtering', () => {
    const jsonData = JSON.stringify({
      images: [
        {
          id: 111,
          pageURL: 'https://pixabay.com/photos/small-111/',
          largeImageURL: 'https://pixabay.com/get/small.jpg',
          tags: 'small image',
          imageWidth: 500,
          imageHeight: 400,
          user: 'user1',
        },
        {
          id: 222,
          pageURL: 'https://pixabay.com/photos/large-222/',
          largeImageURL: 'https://pixabay.com/get/large.jpg',
          tags: 'large image',
          imageWidth: 3000,
          imageHeight: 2000,
          user: 'user2',
        },
      ],
    });
    const mockHtml = `<script type="application/json">${jsonData}</script>`;

    const params = createParams({
      imageFilters: { minWidth: 1000 },
    });

    const results = engine.parseResponse(mockHtml, params);
    expect(results.results.length).toBe(1);
    expect(results.results[0].title).toBe('large image');
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

  it('should handle malformed JSON gracefully', () => {
    const mockHtml = `
      <script type="application/json">
        { invalid json here
      </script>
    `;

    // Should not throw, just return empty results
    const results = engine.parseResponse(mockHtml, createParams());
    expect(Array.isArray(results.results)).toBe(true);
  });

  // Live search test - skipped by default
  it.skip('should search and return image results (live)', async () => {
    const config = engine.buildRequest('nature', createParams());
    const res = await fetch(config.url, { headers: config.headers });
    const body = await res.text();
    const results = engine.parseResponse(body, createParams());

    expect(results.results.length).toBeGreaterThan(0);
    const first = results.results[0];
    expect(first.url).toContain('pixabay.com');
    expect(first.title).toBeTruthy();
    expect(first.category).toBe('images');
  }, 15000);
});
