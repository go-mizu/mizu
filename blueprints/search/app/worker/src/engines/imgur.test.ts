import { describe, it, expect } from 'vitest';
import { ImgurEngine } from './imgur';
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

describe('ImgurEngine', () => {
  const engine = new ImgurEngine();

  it('should have correct metadata', () => {
    expect(engine.name).toBe('imgur');
    expect(engine.shortcut).toBe('im');
    expect(engine.categories).toContain('images');
    expect(engine.supportsPaging).toBe(true);
    expect(engine.maxPage).toBe(50);
    expect(engine.timeout).toBe(8000);
    expect(engine.weight).toBe(0.85);
    expect(engine.disabled).toBe(false);
  });

  it('should build correct search URL', () => {
    const config = engine.buildRequest('funny cats', createParams());
    expect(config.url).toContain('api.imgur.com/3/gallery/search');
    expect(config.url).toContain('q=funny');
    expect(config.method).toBe('GET');
    expect(config.headers['Accept']).toBe('application/json');
    expect(config.headers['Authorization']).toContain('Client-ID');
  });

  it('should build URL with pagination (0-based)', () => {
    const config = engine.buildRequest('test', createParams({ page: 3 }));
    // Page 3 should be index 2 (0-based)
    expect(config.url).toContain('/2?');
  });

  it('should build URL with time range filter', () => {
    const configDay = engine.buildRequest('test', createParams({ timeRange: 'day' }));
    expect(configDay.url).toContain('/time/day/');

    const configWeek = engine.buildRequest('test', createParams({ timeRange: 'week' }));
    expect(configWeek.url).toContain('/time/week/');

    const configMonth = engine.buildRequest('test', createParams({ timeRange: 'month' }));
    expect(configMonth.url).toContain('/time/month/');

    const configYear = engine.buildRequest('test', createParams({ timeRange: 'year' }));
    expect(configYear.url).toContain('/time/year/');
  });

  it('should build URL with all time when no time range', () => {
    const config = engine.buildRequest('test', createParams({ timeRange: '' }));
    expect(config.url).toContain('/time/all/');
  });

  it('should parse JSON API response correctly', () => {
    const mockResponse = JSON.stringify({
      data: [
        {
          id: 'abc123',
          title: 'Funny Cat Picture',
          description: 'A very funny cat doing cat things',
          datetime: 1704067200,
          type: 'image/jpeg',
          animated: false,
          width: 1920,
          height: 1080,
          views: 50000,
          link: 'https://i.imgur.com/abc123.jpg',
          account_url: 'CatLover',
          in_gallery: true,
          nsfw: false,
          score: 1234,
          points: 5678,
          ups: 6000,
          downs: 322,
        },
        {
          id: 'def456',
          title: 'Cool Album',
          description: 'An album of cool stuff',
          animated: false,
          cover: 'xyz789',
          cover_width: 2560,
          cover_height: 1440,
          images_count: 5,
          images: [
            {
              id: 'xyz789',
              width: 2560,
              height: 1440,
              link: 'https://i.imgur.com/xyz789.jpg',
            },
            {
              id: 'img002',
              width: 1920,
              height: 1080,
              link: 'https://i.imgur.com/img002.jpg',
            },
          ],
          account_url: 'CoolUser',
          in_gallery: true,
          nsfw: false,
        },
      ],
      success: true,
      status: 200,
    });

    const results = engine.parseResponse(mockResponse, createParams());

    expect(results.results.length).toBe(2);

    const first = results.results[0];
    expect(first.url).toBe('https://imgur.com/gallery/abc123');
    expect(first.title).toBe('Funny Cat Picture');
    expect(first.content).toBe('A very funny cat doing cat things');
    expect(first.imageUrl).toBe('https://i.imgur.com/abc123.jpg');
    expect(first.thumbnailUrl).toBe('https://i.imgur.com/abc123m.jpg');
    expect(first.resolution).toBe('1920x1080');
    expect(first.source).toBe('CatLover');
    expect(first.engine).toBe('imgur');
    expect(first.category).toBe('images');

    const second = results.results[1];
    expect(second.title).toBe('Cool Album');
    // Should use cover image from album
    expect(second.thumbnailUrl).toBe('https://i.imgur.com/xyz789m.jpg');
    expect(second.resolution).toBe('2560x1440');
    expect(second.source).toBe('CoolUser');
  });

  it('should filter out NSFW content when safe search is enabled', () => {
    const mockResponse = JSON.stringify({
      data: [
        {
          id: 'safe1',
          title: 'Safe Image',
          width: 800,
          height: 600,
          link: 'https://i.imgur.com/safe1.jpg',
          nsfw: false,
          in_gallery: true,
        },
        {
          id: 'nsfw1',
          title: 'NSFW Image',
          width: 800,
          height: 600,
          link: 'https://i.imgur.com/nsfw1.jpg',
          nsfw: true,
          in_gallery: true,
        },
      ],
      success: true,
      status: 200,
    });

    const params = createParams({ safeSearch: 1 });
    const results = engine.parseResponse(mockResponse, params);

    expect(results.results.length).toBe(1);
    expect(results.results[0].title).toBe('Safe Image');
  });

  it('should include NSFW content when safe search is off', () => {
    const mockResponse = JSON.stringify({
      data: [
        {
          id: 'safe1',
          title: 'Safe Image',
          width: 800,
          height: 600,
          link: 'https://i.imgur.com/safe1.jpg',
          nsfw: false,
          in_gallery: true,
        },
        {
          id: 'nsfw1',
          title: 'NSFW Image',
          width: 800,
          height: 600,
          link: 'https://i.imgur.com/nsfw1.jpg',
          nsfw: true,
          in_gallery: true,
        },
      ],
      success: true,
      status: 200,
    });

    const params = createParams({ safeSearch: 0 });
    const results = engine.parseResponse(mockResponse, params);

    expect(results.results.length).toBe(2);
  });

  it('should apply client-side size filtering', () => {
    const mockResponse = JSON.stringify({
      data: [
        {
          id: 'small1',
          title: 'Small Image',
          width: 500,
          height: 400,
          link: 'https://i.imgur.com/small1.jpg',
          in_gallery: true,
        },
        {
          id: 'large1',
          title: 'Large Image',
          width: 2560,
          height: 1440,
          link: 'https://i.imgur.com/large1.jpg',
          in_gallery: true,
        },
      ],
      success: true,
      status: 200,
    });

    const params = createParams({
      imageFilters: { size: 'large' },
    });

    const results = engine.parseResponse(mockResponse, params);
    expect(results.results.length).toBe(1);
    expect(results.results[0].title).toBe('Large Image');
  });

  it('should apply aspect ratio filtering', () => {
    const mockResponse = JSON.stringify({
      data: [
        {
          id: 'portrait1',
          title: 'Portrait Image',
          width: 600,
          height: 1000,
          link: 'https://i.imgur.com/portrait1.jpg',
          in_gallery: true,
        },
        {
          id: 'landscape1',
          title: 'Landscape Image',
          width: 1920,
          height: 1080,
          link: 'https://i.imgur.com/landscape1.jpg',
          in_gallery: true,
        },
      ],
      success: true,
      status: 200,
    });

    const params = createParams({
      imageFilters: { aspect: 'wide' },
    });

    const results = engine.parseResponse(mockResponse, params);
    expect(results.results.length).toBe(1);
    expect(results.results[0].title).toBe('Landscape Image');
  });

  it('should filter by animation type', () => {
    const mockResponse = JSON.stringify({
      data: [
        {
          id: 'static1',
          title: 'Static Image',
          animated: false,
          width: 800,
          height: 600,
          link: 'https://i.imgur.com/static1.jpg',
          in_gallery: true,
        },
        {
          id: 'animated1',
          title: 'Animated GIF',
          animated: true,
          width: 800,
          height: 600,
          link: 'https://i.imgur.com/animated1.gif',
          mp4: 'https://i.imgur.com/animated1.mp4',
          in_gallery: true,
        },
      ],
      success: true,
      status: 200,
    });

    // Filter for photos only (static)
    const photoParams = createParams({
      imageFilters: { type: 'photo' },
    });
    const photoResults = engine.parseResponse(mockResponse, photoParams);
    expect(photoResults.results.length).toBe(1);
    expect(photoResults.results[0].title).toBe('Static Image');

    // Filter for animated only
    const animatedParams = createParams({
      imageFilters: { type: 'animated' },
    });
    const animatedResults = engine.parseResponse(mockResponse, animatedParams);
    expect(animatedResults.results.length).toBe(1);
    expect(animatedResults.results[0].title).toBe('Animated GIF');
  });

  it('should handle API error response', () => {
    const mockResponse = JSON.stringify({
      data: null,
      success: false,
      status: 403,
    });

    const results = engine.parseResponse(mockResponse, createParams());
    expect(results.results).toEqual([]);
  });

  it('should handle empty response', () => {
    const mockResponse = JSON.stringify({
      data: [],
      success: true,
      status: 200,
    });

    const results = engine.parseResponse(mockResponse, createParams());
    expect(results.results).toEqual([]);
  });

  it('should handle malformed JSON gracefully', () => {
    const results = engine.parseResponse('not valid json', createParams());
    expect(Array.isArray(results.results)).toBe(true);
  });

  it('should use static thumbnail for animated content', () => {
    const mockResponse = JSON.stringify({
      data: [
        {
          id: 'gif123',
          title: 'Animated GIF',
          animated: true,
          width: 800,
          height: 600,
          link: 'https://i.imgur.com/gif123.gif',
          mp4: 'https://i.imgur.com/gif123.mp4',
          in_gallery: true,
        },
      ],
      success: true,
      status: 200,
    });

    const results = engine.parseResponse(mockResponse, createParams());

    expect(results.results.length).toBe(1);
    // Should use static thumbnail (h suffix) instead of gif/mp4
    expect(results.results[0].imageUrl).toBe('https://i.imgur.com/gif123h.jpg');
  });

  // Live search test - skipped by default
  it.skip('should search and return image results (live)', async () => {
    const config = engine.buildRequest('cats', createParams());
    const res = await fetch(config.url, { headers: config.headers });
    const body = await res.text();
    const results = engine.parseResponse(body, createParams());

    expect(results.results.length).toBeGreaterThan(0);
    const first = results.results[0];
    expect(first.url).toContain('imgur.com');
    expect(first.title).toBeTruthy();
    expect(first.imageUrl).toContain('i.imgur.com');
    expect(first.category).toBe('images');
  }, 15000);
});
