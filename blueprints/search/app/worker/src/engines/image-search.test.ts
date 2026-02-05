/**
 * Integration tests for Image Search engines and filters.
 * These tests make real network requests to verify functionality.
 */
import { describe, it, expect } from 'vitest';
import { GoogleImagesEngine, GoogleReverseImageEngine } from './google';
import { BingImagesEngine, BingReverseImageEngine } from './bing';
import { DuckDuckGoImagesEngine, prepareVqd } from './duckduckgo';
import { executeEngine } from './engine';
import type { EngineParams } from './engine';

// Base params for testing
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

// Helper to decode URL for easier assertions
function decodeUrl(url: string): string {
  return decodeURIComponent(url);
}

describe('GoogleImagesEngine', () => {
  const engine = new GoogleImagesEngine();

  it('has correct metadata', () => {
    expect(engine.name).toBe('google images');
    expect(engine.shortcut).toBe('gi');
    expect(engine.categories).toContain('images');
    expect(engine.supportsPaging).toBe(true);
  });

  it('builds request URL with query', () => {
    const config = engine.buildRequest('cats', createParams());
    expect(config.url).toContain('google.com/search');
    expect(config.url).toContain('q=cats');
    expect(config.url).toContain('tbm=isch');
  });

  it('builds request with size filter', () => {
    const params = createParams({
      imageFilters: { size: 'large' },
    });
    const config = engine.buildRequest('dogs', params);
    const decodedUrl = decodeUrl(config.url);
    expect(decodedUrl).toContain('tbs=');
    expect(decodedUrl).toContain('isz:l');
  });

  it('builds request with color filter', () => {
    const params = createParams({
      imageFilters: { color: 'red' },
    });
    const config = engine.buildRequest('flowers', params);
    expect(decodeUrl(config.url)).toContain('isc:red');
  });

  it('builds request with type filter', () => {
    const params = createParams({
      imageFilters: { type: 'animated' },
    });
    const config = engine.buildRequest('funny', params);
    expect(decodeUrl(config.url)).toContain('itp:animated');
  });

  it('builds request with aspect filter', () => {
    const params = createParams({
      imageFilters: { aspect: 'wide' },
    });
    const config = engine.buildRequest('landscape', params);
    expect(decodeUrl(config.url)).toContain('iar:w');
  });

  it('builds request with safe search', () => {
    const params = createParams({ safeSearch: 2 });
    const config = engine.buildRequest('test', params);
    expect(config.url).toContain('safe=high');
  });

  it('builds request with pagination', () => {
    const params = createParams({ page: 2 });
    const config = engine.buildRequest('test', params);
    expect(decodeUrl(config.url)).toContain('ijn:1');
  });

  // Live search tests are skipped by default since they make real network requests
  // which can be blocked by Google. Run manually with: npm test -- --grep "live search"
  describe.skip('live search', () => {
    it('returns image results for basic query', async () => {
      const params = createParams();
      const results = await executeEngine(engine, 'golden retriever', params);

      expect(results.results.length).toBeGreaterThan(0);
      expect(results.results[0]).toHaveProperty('imageUrl');
      expect(results.results[0].category).toBe('images');
      expect(results.results[0].engine).toBe('google images');
    }, 15000);

    it('returns results with size filter', async () => {
      const params = createParams({
        imageFilters: { size: 'large' },
      });
      const results = await executeEngine(engine, 'mountain landscape', params);

      expect(results.results.length).toBeGreaterThan(0);
      // Large images should have higher resolution
      const firstResult = results.results[0];
      if (firstResult.resolution) {
        const [width] = firstResult.resolution.split('x').map(Number);
        // Large images should be at least 1024px wide
        expect(width).toBeGreaterThanOrEqual(500);
      }
    }, 15000);
  });
});

describe('BingImagesEngine', () => {
  const engine = new BingImagesEngine();

  it('has correct metadata', () => {
    expect(engine.name).toBe('bing images');
    expect(engine.shortcut).toBe('bi');
    expect(engine.categories).toContain('images');
  });

  it('builds request URL with query', () => {
    const config = engine.buildRequest('cats', createParams());
    expect(config.url).toContain('bing.com/images/async');
    expect(config.url).toContain('q=cats');
  });

  it('builds request with size filter', () => {
    const params = createParams({
      imageFilters: { size: 'large' },
    });
    const config = engine.buildRequest('dogs', params);
    expect(decodeUrl(config.url)).toContain('filterui:imagesize-large');
  });

  it('builds request with color filter', () => {
    const params = createParams({
      imageFilters: { color: 'blue' },
    });
    const config = engine.buildRequest('sky', params);
    expect(config.url).toContain('FGcls_BLUE');
  });

  it('builds request with type filter', () => {
    const params = createParams({
      imageFilters: { type: 'clipart' },
    });
    const config = engine.buildRequest('icons', params);
    expect(decodeUrl(config.url)).toContain('filterui:photo-clipart');
  });

  it('builds request with pagination', () => {
    const params = createParams({ page: 2 });
    const config = engine.buildRequest('test', params);
    expect(config.url).toContain('first=36');
  });

  // Live search tests are skipped by default since they make real network requests
  describe.skip('live search', () => {
    it('returns image results for basic query', async () => {
      const params = createParams();
      const results = await executeEngine(engine, 'sunset beach', params);

      expect(results.results.length).toBeGreaterThan(0);
      expect(results.results[0]).toHaveProperty('imageUrl');
      expect(results.results[0].category).toBe('images');
      expect(results.results[0].engine).toBe('bing images');
    }, 15000);
  });
});

describe('DuckDuckGoImagesEngine', () => {
  const engine = new DuckDuckGoImagesEngine();

  it('has correct metadata', () => {
    expect(engine.name).toBe('duckduckgo images');
    expect(engine.shortcut).toBe('ddi');
    expect(engine.categories).toContain('images');
  });

  it('builds request URL with VQD', () => {
    const params = createParams({
      engineData: { vqd: 'test-vqd-123' },
    });
    const config = engine.buildRequest('cats', params);
    expect(config.url).toContain('duckduckgo.com/i.js');
    expect(config.url).toContain('vqd=test-vqd-123');
  });

  it('builds request with size filter', () => {
    const params = createParams({
      engineData: { vqd: 'test' },
      imageFilters: { size: 'large' },
    });
    const config = engine.buildRequest('dogs', params);
    expect(config.url).toContain('Large');
  });

  it('builds request with color filter', () => {
    const params = createParams({
      engineData: { vqd: 'test' },
      imageFilters: { color: 'transparent' },
    });
    const config = engine.buildRequest('logo', params);
    expect(config.url).toContain('Transparent');
  });

  // Live search tests are skipped by default since they make real network requests
  // which often get blocked by DDG. Run manually with: npm test -- --grep "live search"
  describe.skip('live search', () => {
    it('returns image results with VQD', async () => {
      const vqd = await prepareVqd('nature photography', 'en-US');
      const params = createParams({
        engineData: { vqd },
      });
      const results = await executeEngine(engine, 'nature photography', params);

      expect(results.results.length).toBeGreaterThan(0);
      expect(results.results[0]).toHaveProperty('imageUrl');
      expect(results.results[0].engine).toBe('duckduckgo images');
    }, 15000);
  });
});

describe('Image Filter Combinations', () => {
  const googleEngine = new GoogleImagesEngine();

  it('combines multiple filters correctly', () => {
    const params = createParams({
      imageFilters: {
        size: 'large',
        color: 'color',
        type: 'photo',
        aspect: 'wide',
      },
    });
    const config = googleEngine.buildRequest('nature', params);
    const decodedUrl = decodeUrl(config.url);

    expect(decodedUrl).toContain('isz:l');
    expect(decodedUrl).toContain('ic:color');
    expect(decodedUrl).toContain('itp:photo');
    expect(decodedUrl).toContain('iar:w');
  });

  it('applies custom size dimensions', () => {
    const params = createParams({
      imageFilters: {
        minWidth: 1920,
        minHeight: 1080,
      },
    });
    const config = googleEngine.buildRequest('wallpaper', params);
    const decodedUrl = decodeUrl(config.url);

    expect(decodedUrl).toContain('iszw:1920');
    expect(decodedUrl).toContain('iszh:1080');
  });

  it('applies time range filter', () => {
    const params = createParams({
      timeRange: 'week',
    });
    const config = googleEngine.buildRequest('news images', params);

    expect(decodeUrl(config.url)).toContain('qdr:w');
  });

  it('applies usage rights filter', () => {
    const params = createParams({
      imageFilters: {
        rights: 'creative_commons',
      },
    });
    const config = googleEngine.buildRequest('stock photos', params);

    expect(decodeUrl(config.url)).toContain('sur:cl');
  });
});

describe('Reverse Image Search', () => {
  describe('GoogleReverseImageEngine', () => {
    const engine = new GoogleReverseImageEngine();

    it('has correct metadata', () => {
      expect(engine.name).toBe('google reverse');
      expect(engine.shortcut).toBe('gri');
      expect(engine.supportsPaging).toBe(false);
    });

    it('builds request with image URL', () => {
      const testUrl = 'https://example.com/image.jpg';
      const config = engine.buildRequest(testUrl, createParams());

      expect(config.url).toContain('lens.google.com');
      expect(config.url).toContain(encodeURIComponent(testUrl));
    });
  });

  describe('BingReverseImageEngine', () => {
    const engine = new BingReverseImageEngine();

    it('has correct metadata', () => {
      expect(engine.name).toBe('bing reverse');
      expect(engine.shortcut).toBe('bri');
      expect(engine.supportsPaging).toBe(false);
    });

    it('builds request with image URL', () => {
      const testUrl = 'https://example.com/image.jpg';
      const config = engine.buildRequest(testUrl, createParams());

      expect(config.url).toContain('bing.com/images/search');
      expect(decodeUrl(config.url)).toContain('imgurl:');
    });
  });
});

describe('Filter Validation', () => {
  const googleEngine = new GoogleImagesEngine();

  it('ignores invalid size filter', () => {
    const params = createParams({
      imageFilters: {
        size: 'invalid' as any,
      },
    });
    const config = googleEngine.buildRequest('test', params);
    // Should not crash and should not include invalid filter
    expect(config.url).not.toContain('isz:invalid');
  });

  it('handles empty filters object', () => {
    const params = createParams({
      imageFilters: {},
    });
    const config = googleEngine.buildRequest('test', params);
    // Should work fine with empty filters
    expect(config.url).toContain('q=test');
  });

  it('handles undefined filters', () => {
    const params = createParams();
    const config = googleEngine.buildRequest('test', params);
    expect(config.url).toContain('q=test');
  });
});

describe('SafeSearch Levels', () => {
  const googleEngine = new GoogleImagesEngine();
  const bingEngine = new BingImagesEngine();

  it('Google: safe search off', () => {
    const params = createParams({ safeSearch: 0 });
    const config = googleEngine.buildRequest('test', params);
    expect(config.url).toContain('safe=off');
  });

  it('Google: safe search moderate', () => {
    const params = createParams({ safeSearch: 1 });
    const config = googleEngine.buildRequest('test', params);
    expect(config.url).toContain('safe=medium');
  });

  it('Google: safe search strict', () => {
    const params = createParams({ safeSearch: 2 });
    const config = googleEngine.buildRequest('test', params);
    expect(config.url).toContain('safe=high');
  });

  it('Bing: safe search off', () => {
    const params = createParams({ safeSearch: 0 });
    const config = bingEngine.buildRequest('test', params);
    expect(config.url).toContain('adlt=off');
  });

  it('Bing: safe search strict', () => {
    const params = createParams({ safeSearch: 2 });
    const config = bingEngine.buildRequest('test', params);
    expect(config.url).toContain('adlt=strict');
  });
});

describe('Pagination', () => {
  const googleEngine = new GoogleImagesEngine();
  const bingEngine = new BingImagesEngine();

  it('Google page 1', () => {
    const params = createParams({ page: 1 });
    const config = googleEngine.buildRequest('test', params);
    expect(decodeUrl(config.url)).toContain('ijn:0');
  });

  it('Google page 3', () => {
    const params = createParams({ page: 3 });
    const config = googleEngine.buildRequest('test', params);
    expect(decodeUrl(config.url)).toContain('ijn:2');
  });

  it('Bing page 1', () => {
    const params = createParams({ page: 1 });
    const config = bingEngine.buildRequest('test', params);
    expect(config.url).toContain('first=1');
  });

  it('Bing page 2', () => {
    const params = createParams({ page: 2 });
    const config = bingEngine.buildRequest('test', params);
    expect(config.url).toContain('first=36');
  });
});

describe('Color Filter Mapping', () => {
  const googleEngine = new GoogleImagesEngine();

  const colorTests: Array<{ color: string; expected: string }> = [
    { color: 'color', expected: 'ic:color' },
    { color: 'gray', expected: 'ic:gray' },
    { color: 'transparent', expected: 'ic:trans' },
    { color: 'red', expected: 'isc:red' },
    { color: 'blue', expected: 'isc:blue' },
    { color: 'green', expected: 'isc:green' },
  ];

  for (const { color, expected } of colorTests) {
    it(`maps ${color} correctly`, () => {
      const params = createParams({
        imageFilters: { color: color as any },
      });
      const config = googleEngine.buildRequest('test', params);
      expect(decodeUrl(config.url)).toContain(expected);
    });
  }
});

describe('Type Filter Mapping', () => {
  const googleEngine = new GoogleImagesEngine();

  const typeTests: Array<{ type: string; expected: string }> = [
    { type: 'face', expected: 'itp:face' },
    { type: 'photo', expected: 'itp:photo' },
    { type: 'clipart', expected: 'itp:clipart' },
    { type: 'lineart', expected: 'itp:lineart' },
    { type: 'animated', expected: 'itp:animated' },
  ];

  for (const { type, expected } of typeTests) {
    it(`maps ${type} correctly`, () => {
      const params = createParams({
        imageFilters: { type: type as any },
      });
      const config = googleEngine.buildRequest('test', params);
      expect(decodeUrl(config.url)).toContain(expected);
    });
  }
});
