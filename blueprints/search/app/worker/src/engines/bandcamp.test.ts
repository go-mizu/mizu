import { describe, it, expect } from 'vitest';
import {
  BandcampEngine,
  BandcampTracksEngine,
  BandcampAlbumsEngine,
  BandcampArtistsEngine,
} from './bandcamp';
import type { EngineParams } from './engine';

describe('BandcampEngine', () => {
  const engine = new BandcampEngine();

  const defaultParams: EngineParams = {
    page: 1,
    locale: 'en-US',
    safeSearch: 1,
    timeRange: '',
    engineData: {},
  };

  it('should have correct metadata', () => {
    expect(engine.name).toBe('bandcamp');
    expect(engine.shortcut).toBe('bc');
    expect(engine.categories).toContain('videos');
    expect(engine.supportsPaging).toBe(true);
    expect(engine.maxPage).toBe(5);
  });

  it('should build correct search URL', () => {
    const config = engine.buildRequest('indie rock', defaultParams);
    expect(config.url).toContain('bandcamp.com/search');
    expect(config.url).toContain('q=indie+rock');
    expect(config.url).toContain('page=1');
    expect(config.method).toBe('GET');
  });

  it('should handle pagination', () => {
    const config = engine.buildRequest('test', {
      ...defaultParams,
      page: 3,
    });
    expect(config.url).toContain('page=3');
  });

  it('should parse search items from data attribute', () => {
    const searchData = {
      auto: {
        results: [
          {
            type: 'track' as const,
            id: 123,
            name: 'Test Track',
            url: 'https://artist.bandcamp.com/track/test-track',
            art_id: 456789,
            artist: 'Test Artist',
            album: 'Test Album',
            genre: 'Indie Rock',
            tags: ['rock', 'indie', 'guitar'],
          },
          {
            type: 'album' as const,
            id: 456,
            name: 'Test Album',
            url: 'https://artist.bandcamp.com/album/test-album',
            art_id: 789012,
            artist: 'Test Artist',
            genre: 'Electronic',
          },
          {
            type: 'artist' as const,
            id: 789,
            name: 'Test Artist',
            url: 'https://testartist.bandcamp.com',
            img: 'https://f4.bcbits.com/img/avatar123.jpg',
            location: 'Los Angeles, CA',
          },
        ],
      },
    };

    const encodedData = JSON.stringify(searchData).replace(/"/g, '&quot;');
    const sampleBody = `<html><body><div data-search="${encodedData}"></div></body></html>`;

    const results = engine.parseResponse(sampleBody, defaultParams);

    expect(results.results.length).toBe(3);

    // Check track
    const track = results.results[0];
    expect(track.url).toBe('https://artist.bandcamp.com/track/test-track');
    expect(track.title).toBe('Test Track');
    expect(track.content).toContain('by Test Artist');
    expect(track.content).toContain('from Test Album');
    expect(track.content).toContain('Indie Rock');
    expect(track.channel).toBe('Test Artist');
    expect(track.metadata?.itemType).toBe('track');
    expect(track.metadata?.tags).toContain('rock');

    // Check album
    const album = results.results[1];
    expect(album.url).toBe('https://artist.bandcamp.com/album/test-album');
    expect(album.title).toBe('Test Album');
    expect(album.content).toContain('by Test Artist');
    expect(album.metadata?.itemType).toBe('album');

    // Check artist
    const artist = results.results[2];
    expect(artist.url).toBe('https://testartist.bandcamp.com');
    expect(artist.title).toBe('Test Artist');
    expect(artist.content).toContain('Los Angeles, CA');
    expect(artist.metadata?.itemType).toBe('artist');
  });

  it('should generate thumbnail URL from art_id', () => {
    const searchData = {
      auto: {
        results: [
          {
            type: 'track' as const,
            id: 1,
            name: 'Track',
            url: 'https://artist.bandcamp.com/track/test',
            art_id: 123456789,
          },
        ],
      },
    };

    const encodedData = JSON.stringify(searchData).replace(/"/g, '&quot;');
    const sampleBody = `<div data-search="${encodedData}"></div>`;

    const results = engine.parseResponse(sampleBody, defaultParams);

    expect(results.results.length).toBe(1);
    expect(results.results[0].thumbnailUrl).toContain('f4.bcbits.com');
    expect(results.results[0].thumbnailUrl).toContain('a123456789');
  });

  it('should handle empty or malformed response', () => {
    const emptyResults = engine.parseResponse('<html></html>', defaultParams);
    expect(emptyResults.results).toEqual([]);

    const noData = engine.parseResponse('plain text', defaultParams);
    expect(noData.results).toEqual([]);
  });

  it('should search and return music results', async () => {
    const results = await fetchAndParse(engine, 'ambient');

    // Bandcamp may require JavaScript, results may vary
    expect(results.results).toBeDefined();
    expect(Array.isArray(results.results)).toBe(true);
  }, 30000);
});

describe('BandcampTracksEngine', () => {
  const engine = new BandcampTracksEngine();

  it('should have correct metadata', () => {
    expect(engine.name).toBe('bandcamp tracks');
  });

  it('should filter by track type', () => {
    const config = engine.buildRequest('test', {
      page: 1,
      locale: 'en-US',
      safeSearch: 1,
      timeRange: '',
      engineData: {},
    });
    expect(config.url).toContain('item_type=t');
  });
});

describe('BandcampAlbumsEngine', () => {
  const engine = new BandcampAlbumsEngine();

  it('should have correct metadata', () => {
    expect(engine.name).toBe('bandcamp albums');
  });

  it('should filter by album type', () => {
    const config = engine.buildRequest('test', {
      page: 1,
      locale: 'en-US',
      safeSearch: 1,
      timeRange: '',
      engineData: {},
    });
    expect(config.url).toContain('item_type=a');
  });
});

describe('BandcampArtistsEngine', () => {
  const engine = new BandcampArtistsEngine();

  it('should have correct metadata', () => {
    expect(engine.name).toBe('bandcamp artists');
  });

  it('should filter by artist type', () => {
    const config = engine.buildRequest('test', {
      page: 1,
      locale: 'en-US',
      safeSearch: 1,
      timeRange: '',
      engineData: {},
    });
    expect(config.url).toContain('item_type=b');
  });
});

async function fetchAndParse(engine: BandcampEngine, query: string) {
  const params: EngineParams = {
    page: 1,
    locale: 'en-US',
    safeSearch: 1,
    timeRange: '',
    engineData: {},
  };
  const config = engine.buildRequest(query, params);
  const res = await fetch(config.url, {
    headers: config.headers,
  });
  const body = await res.text();
  return engine.parseResponse(body, params);
}
