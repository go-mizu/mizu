import { describe, it, expect } from 'vitest';
import { FlickrEngine } from './flickr';
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

describe('FlickrEngine', () => {
  const engine = new FlickrEngine();

  it('should have correct metadata', () => {
    expect(engine.name).toBe('flickr');
    expect(engine.shortcut).toBe('fl');
    expect(engine.categories).toContain('images');
    expect(engine.supportsPaging).toBe(true);
    expect(engine.maxPage).toBe(100);
    expect(engine.timeout).toBe(8000);
    expect(engine.weight).toBe(0.85);
    expect(engine.disabled).toBe(false);
  });

  it('should build correct search URL', () => {
    const config = engine.buildRequest('nature landscape', createParams());
    expect(config.url).toContain('flickr.com/services/rest');
    expect(config.url).toContain('method=flickr.photos.search');
    expect(config.url).toContain('text=nature+landscape');
    expect(config.url).toContain('format=json');
    expect(config.method).toBe('GET');
    expect(config.headers['Accept']).toBe('application/json');
  });

  it('should build URL with pagination', () => {
    const config = engine.buildRequest('test', createParams({ page: 3 }));
    expect(config.url).toContain('page=3');
  });

  it('should build URL with safe search', () => {
    // Safe search off
    const configOff = engine.buildRequest('test', createParams({ safeSearch: 0 }));
    expect(configOff.url).toContain('safe_search=3');

    // Safe search moderate
    const configMod = engine.buildRequest('test', createParams({ safeSearch: 1 }));
    expect(configMod.url).toContain('safe_search=2');

    // Safe search strict
    const configStrict = engine.buildRequest('test', createParams({ safeSearch: 2 }));
    expect(configStrict.url).toContain('safe_search=1');
  });

  it('should build URL with color filter', () => {
    const config = engine.buildRequest('flowers', createParams({
      imageFilters: { color: 'red' },
    }));
    expect(config.url).toContain('color_codes=0');
  });

  it('should build URL with license filter', () => {
    const config = engine.buildRequest('photos', createParams({
      imageFilters: { rights: 'creative_commons' },
    }));
    expect(config.url).toContain('license=1%2C2%2C3%2C4%2C5%2C6%2C7%2C8');
  });

  it('should build URL with time range', () => {
    const config = engine.buildRequest('test', createParams({ timeRange: 'week' }));
    expect(config.url).toContain('min_upload_date=');
  });

  it('should parse JSON response correctly', () => {
    const mockResponse = JSON.stringify({
      photos: {
        page: 1,
        pages: 100,
        perpage: 20,
        total: 2000,
        photo: [
          {
            id: '12345678901',
            owner: 'user123',
            secret: 'abc123',
            server: '65535',
            farm: 66,
            title: 'Beautiful Sunset',
            ispublic: 1,
            ownername: 'John Doe',
            description: { _content: 'A beautiful sunset over the ocean' },
            o_width: '4000',
            o_height: '3000',
            url_o: 'https://live.staticflickr.com/65535/12345678901_abc123_o.jpg',
            url_l: 'https://live.staticflickr.com/65535/12345678901_abc123_b.jpg',
            url_m: 'https://live.staticflickr.com/65535/12345678901_abc123_m.jpg',
            url_s: 'https://live.staticflickr.com/65535/12345678901_abc123_s.jpg',
          },
          {
            id: '98765432101',
            owner: 'user456',
            secret: 'def456',
            server: '65535',
            farm: 66,
            title: 'Mountain Peak',
            ispublic: 1,
            ownername: 'Jane Smith',
            o_width: '2000',
            o_height: '1500',
            url_l: 'https://live.staticflickr.com/65535/98765432101_def456_b.jpg',
          },
        ],
      },
      stat: 'ok',
    });

    const results = engine.parseResponse(mockResponse, createParams());

    expect(results.results.length).toBe(2);

    const first = results.results[0];
    expect(first.url).toBe('https://www.flickr.com/photos/user123/12345678901');
    expect(first.title).toBe('Beautiful Sunset');
    expect(first.content).toBe('A beautiful sunset over the ocean');
    expect(first.imageUrl).toBe('https://live.staticflickr.com/65535/12345678901_abc123_o.jpg');
    expect(first.thumbnailUrl).toBe('https://live.staticflickr.com/65535/12345678901_abc123_s.jpg');
    expect(first.resolution).toBe('4000x3000');
    expect(first.source).toBe('John Doe');
    expect(first.engine).toBe('flickr');
    expect(first.category).toBe('images');

    const second = results.results[1];
    expect(second.title).toBe('Mountain Peak');
    expect(second.source).toBe('Jane Smith');
  });

  it('should handle API error response', () => {
    const mockResponse = JSON.stringify({
      stat: 'fail',
      code: 100,
      message: 'Invalid API Key',
    });

    const results = engine.parseResponse(mockResponse, createParams());
    expect(results.results).toEqual([]);
  });

  it('should apply client-side size filtering', () => {
    const mockResponse = JSON.stringify({
      photos: {
        page: 1,
        pages: 1,
        perpage: 20,
        total: 2,
        photo: [
          {
            id: '111',
            owner: 'user1',
            secret: 'sec1',
            server: '65535',
            farm: 66,
            title: 'Small Image',
            o_width: '500',
            o_height: '400',
          },
          {
            id: '222',
            owner: 'user2',
            secret: 'sec2',
            server: '65535',
            farm: 66,
            title: 'Large Image',
            o_width: '3000',
            o_height: '2000',
          },
        ],
      },
      stat: 'ok',
    });

    const params = createParams({
      imageFilters: { minWidth: 1000 },
    });

    const results = engine.parseResponse(mockResponse, params);
    expect(results.results.length).toBe(1);
    expect(results.results[0].title).toBe('Large Image');
  });

  it('should return empty results for malformed JSON', () => {
    const results = engine.parseResponse('not valid json', createParams());
    expect(results.results).toEqual([]);
  });

  // Live search test - skipped by default
  it.skip('should search and return image results (live)', async () => {
    const config = engine.buildRequest('mountains', createParams());
    const res = await fetch(config.url, { headers: config.headers });
    const body = await res.text();
    const results = engine.parseResponse(body, createParams());

    expect(results.results.length).toBeGreaterThan(0);
    const first = results.results[0];
    expect(first.url).toContain('flickr.com/photos');
    expect(first.title).toBeTruthy();
    expect(first.imageUrl).toBeTruthy();
    expect(first.category).toBe('images');
  }, 15000);
});
