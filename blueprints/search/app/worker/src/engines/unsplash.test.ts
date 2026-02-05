import { describe, it, expect } from 'vitest';
import { UnsplashEngine } from './unsplash';
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

describe('UnsplashEngine', () => {
  const engine = new UnsplashEngine();

  it('should have correct metadata', () => {
    expect(engine.name).toBe('unsplash');
    expect(engine.shortcut).toBe('us');
    expect(engine.categories).toContain('images');
    expect(engine.supportsPaging).toBe(true);
    expect(engine.maxPage).toBe(50);
    expect(engine.timeout).toBe(8000);
    expect(engine.weight).toBe(0.9);
    expect(engine.disabled).toBe(false);
  });

  it('should build correct search URL', () => {
    const config = engine.buildRequest('nature mountains', createParams());
    expect(config.url).toContain('unsplash.com/napi/search/photos');
    expect(config.url).toContain('query=nature+mountains');
    expect(config.url).toContain('per_page=20');
    expect(config.method).toBe('GET');
    expect(config.headers['Accept']).toBe('application/json');
    expect(config.headers['Referer']).toBe('https://unsplash.com/');
  });

  it('should build URL with pagination', () => {
    const config = engine.buildRequest('test', createParams({ page: 5 }));
    expect(config.url).toContain('page=5');
  });

  it('should build URL with color filter', () => {
    const config = engine.buildRequest('flowers', createParams({
      imageFilters: { color: 'blue' },
    }));
    expect(config.url).toContain('color=blue');
  });

  it('should build URL with grayscale filter', () => {
    const config = engine.buildRequest('city', createParams({
      imageFilters: { color: 'gray' },
    }));
    expect(config.url).toContain('color=black_and_white');
  });

  it('should build URL with orientation filter (portrait)', () => {
    const config = engine.buildRequest('portrait', createParams({
      imageFilters: { aspect: 'tall' },
    }));
    expect(config.url).toContain('orientation=portrait');
  });

  it('should build URL with orientation filter (landscape)', () => {
    const config = engine.buildRequest('landscape', createParams({
      imageFilters: { aspect: 'wide' },
    }));
    expect(config.url).toContain('orientation=landscape');
  });

  it('should build URL with orientation filter (square)', () => {
    const config = engine.buildRequest('square', createParams({
      imageFilters: { aspect: 'square' },
    }));
    expect(config.url).toContain('orientation=squarish');
  });

  it('should build URL with latest sort for recent time range', () => {
    const config = engine.buildRequest('test', createParams({ timeRange: 'day' }));
    expect(config.url).toContain('order_by=latest');
  });

  it('should parse JSON response correctly', () => {
    const mockResponse = JSON.stringify({
      total: 1000,
      total_pages: 50,
      results: [
        {
          id: 'abc123',
          slug: 'beautiful-mountain-abc123',
          width: 4000,
          height: 3000,
          color: '#0066cc',
          description: 'Beautiful mountain landscape',
          alt_description: 'Snow-capped mountain at sunrise',
          urls: {
            raw: 'https://images.unsplash.com/photo-abc123?ixlib=rb-4.0.3',
            full: 'https://images.unsplash.com/photo-abc123?ixlib=rb-4.0.3&w=4000',
            regular: 'https://images.unsplash.com/photo-abc123?ixlib=rb-4.0.3&w=1080',
            small: 'https://images.unsplash.com/photo-abc123?ixlib=rb-4.0.3&w=400',
            thumb: 'https://images.unsplash.com/photo-abc123?ixlib=rb-4.0.3&w=200',
          },
          links: {
            self: 'https://api.unsplash.com/photos/abc123',
            html: 'https://unsplash.com/photos/beautiful-mountain-abc123',
            download: 'https://unsplash.com/photos/abc123/download',
          },
          user: {
            id: 'user123',
            username: 'johndoe',
            name: 'John Doe',
          },
        },
        {
          id: 'def456',
          width: 3000,
          height: 2000,
          alt_description: 'Ocean sunset',
          urls: {
            full: 'https://images.unsplash.com/photo-def456?w=3000',
            small: 'https://images.unsplash.com/photo-def456?w=400',
          },
          user: {
            name: 'Jane Smith',
          },
        },
      ],
    });

    const results = engine.parseResponse(mockResponse, createParams());

    expect(results.results.length).toBe(2);

    const first = results.results[0];
    expect(first.url).toBe('https://unsplash.com/photos/beautiful-mountain-abc123');
    expect(first.title).toBe('Snow-capped mountain at sunrise');
    expect(first.content).toBe('Beautiful mountain landscape');
    expect(first.imageUrl).toBe('https://images.unsplash.com/photo-abc123?ixlib=rb-4.0.3&w=4000');
    expect(first.thumbnailUrl).toBe('https://images.unsplash.com/photo-abc123?ixlib=rb-4.0.3&w=400');
    expect(first.resolution).toBe('4000x3000');
    expect(first.source).toBe('John Doe');
    expect(first.engine).toBe('unsplash');
    expect(first.category).toBe('images');

    const second = results.results[1];
    expect(second.title).toBe('Ocean sunset');
    expect(second.source).toBe('Jane Smith');
    expect(second.resolution).toBe('3000x2000');
  });

  it('should handle empty response', () => {
    const mockResponse = JSON.stringify({
      total: 0,
      total_pages: 0,
      results: [],
    });

    const results = engine.parseResponse(mockResponse, createParams());
    expect(results.results).toEqual([]);
  });

  it('should apply client-side size filtering', () => {
    const mockResponse = JSON.stringify({
      total: 2,
      total_pages: 1,
      results: [
        {
          id: 'small1',
          width: 800,
          height: 600,
          alt_description: 'Small image',
          urls: { full: 'https://unsplash.com/small.jpg', small: 'https://unsplash.com/small_s.jpg' },
          user: { name: 'User1' },
        },
        {
          id: 'large1',
          width: 4000,
          height: 3000,
          alt_description: 'Large image',
          urls: { full: 'https://unsplash.com/large.jpg', small: 'https://unsplash.com/large_s.jpg' },
          user: { name: 'User2' },
        },
      ],
    });

    const params = createParams({
      imageFilters: { size: 'large' },
    });

    const results = engine.parseResponse(mockResponse, params);
    expect(results.results.length).toBe(1);
    expect(results.results[0].title).toBe('Large image');
  });

  it('should apply min dimension filtering', () => {
    const mockResponse = JSON.stringify({
      total: 2,
      total_pages: 1,
      results: [
        {
          id: 'small1',
          width: 500,
          height: 400,
          alt_description: 'Small image',
          urls: { full: 'https://unsplash.com/small.jpg', small: 'https://unsplash.com/small_s.jpg' },
          user: { name: 'User1' },
        },
        {
          id: 'large1',
          width: 2000,
          height: 1500,
          alt_description: 'Large image',
          urls: { full: 'https://unsplash.com/large.jpg', small: 'https://unsplash.com/large_s.jpg' },
          user: { name: 'User2' },
        },
      ],
    });

    const params = createParams({
      imageFilters: { minWidth: 1000, minHeight: 800 },
    });

    const results = engine.parseResponse(mockResponse, params);
    expect(results.results.length).toBe(1);
    expect(results.results[0].title).toBe('Large image');
  });

  it('should return empty results for malformed JSON', () => {
    const results = engine.parseResponse('not valid json', createParams());
    expect(results.results).toEqual([]);
  });

  it('should use fallback title when alt_description is missing', () => {
    const mockResponse = JSON.stringify({
      total: 1,
      total_pages: 1,
      results: [
        {
          id: 'photo1',
          width: 2000,
          height: 1500,
          urls: { full: 'https://unsplash.com/photo1.jpg', small: 'https://unsplash.com/photo1_s.jpg' },
          user: { name: 'Photographer Name' },
        },
      ],
    });

    const results = engine.parseResponse(mockResponse, createParams());
    expect(results.results[0].title).toBe('Photo by Photographer Name');
  });

  // Live search test - skipped by default
  it.skip('should search and return image results (live)', async () => {
    const config = engine.buildRequest('nature', createParams());
    const res = await fetch(config.url, { headers: config.headers });
    const body = await res.text();
    const results = engine.parseResponse(body, createParams());

    expect(results.results.length).toBeGreaterThan(0);
    const first = results.results[0];
    expect(first.url).toContain('unsplash.com');
    expect(first.title).toBeTruthy();
    expect(first.imageUrl).toContain('images.unsplash.com');
    expect(first.category).toBe('images');
  }, 15000);
});
