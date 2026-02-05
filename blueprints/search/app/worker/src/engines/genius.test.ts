import { describe, it, expect } from 'vitest';
import { GeniusEngine, GeniusLyricsEngine } from './genius';
import type { EngineParams } from './engine';

describe('GeniusEngine', () => {
  const engine = new GeniusEngine();

  const defaultParams: EngineParams = {
    page: 1,
    locale: 'en-US',
    safeSearch: 1,
    timeRange: '',
    engineData: {},
  };

  it('should have correct metadata', () => {
    expect(engine.name).toBe('genius');
    expect(engine.shortcut).toBe('gn');
    expect(engine.categories).toContain('videos');
    expect(engine.supportsPaging).toBe(true);
    expect(engine.maxPage).toBe(5);
  });

  it('should build correct API URL', () => {
    const config = engine.buildRequest('bohemian rhapsody', defaultParams);
    expect(config.url).toContain('genius.com/api/search/multi');
    expect(config.url).toContain('q=bohemian+rhapsody');
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

  it('should parse song response', () => {
    const sampleResponse = JSON.stringify({
      response: {
        sections: [
          {
            type: 'song',
            hits: [
              {
                type: 'song',
                index: 'song',
                result: {
                  id: 12345,
                  title: 'Bohemian Rhapsody',
                  title_with_featured: 'Bohemian Rhapsody',
                  url: 'https://genius.com/Queen-bohemian-rhapsody-lyrics',
                  path: '/Queen-bohemian-rhapsody-lyrics',
                  full_title: 'Bohemian Rhapsody by Queen',
                  song_art_image_url: 'https://images.genius.com/artwork.jpg',
                  song_art_image_thumbnail_url: 'https://images.genius.com/artwork_thumb.jpg',
                  release_date_for_display: 'October 31, 1975',
                  release_date_components: {
                    year: 1975,
                    month: 10,
                    day: 31,
                  },
                  primary_artist: {
                    id: 100,
                    name: 'Queen',
                    url: 'https://genius.com/artists/Queen',
                    image_url: 'https://images.genius.com/queen.jpg',
                    is_verified: true,
                  },
                  stats: {
                    hot: true,
                    pageviews: 5000000,
                  },
                  annotation_count: 150,
                  pyongs_count: 500,
                  lyrics_state: 'complete',
                },
              },
            ],
          },
        ],
      },
    });

    const results = engine.parseResponse(sampleResponse, defaultParams);

    expect(results.results.length).toBe(1);

    const first = results.results[0];
    expect(first.url).toBe('https://genius.com/Queen-bohemian-rhapsody-lyrics');
    expect(first.title).toBe('Bohemian Rhapsody');
    expect(first.channel).toBe('Queen');
    expect(first.content).toContain('by Queen');
    expect(first.content).toContain('October 31, 1975');
    expect(first.content).toContain('views');
    expect(first.content).toContain('150 annotations');
    expect(first.content).toContain('Hot');
    expect(first.source).toBe('Genius');
    expect(first.thumbnailUrl).toBeTruthy();
    expect(first.publishedAt).toBe('1975-10-31T00:00:00.000Z');
    expect(first.metadata?.songId).toBe(12345);
    expect(first.metadata?.artistId).toBe(100);
    expect(first.metadata?.pageviews).toBe(5000000);
    expect(first.metadata?.annotationCount).toBe(150);
    expect(first.metadata?.isHot).toBe(true);
  });

  it('should parse multiple sections', () => {
    const sampleResponse = JSON.stringify({
      response: {
        sections: [
          {
            type: 'top_hit',
            hits: [
              {
                type: 'song',
                result: {
                  id: 1,
                  title: 'Top Hit Song',
                  url: 'https://genius.com/song1',
                  full_title: 'Top Hit Song by Artist',
                  primary_artist: { id: 1, name: 'Artist', url: '' },
                },
              },
            ],
          },
          {
            type: 'song',
            hits: [
              {
                type: 'song',
                result: {
                  id: 2,
                  title: 'Another Song',
                  url: 'https://genius.com/song2',
                  full_title: 'Another Song by Artist 2',
                  primary_artist: { id: 2, name: 'Artist 2', url: '' },
                },
              },
            ],
          },
        ],
      },
    });

    const results = engine.parseResponse(sampleResponse, defaultParams);

    expect(results.results.length).toBe(2);
    expect(results.results[0].title).toContain('Top Hit Song');
    expect(results.results[1].title).toContain('Another Song');
  });

  it('should format large numbers', () => {
    const sampleResponse = JSON.stringify({
      response: {
        sections: [
          {
            type: 'song',
            hits: [
              {
                type: 'song',
                result: {
                  id: 1,
                  title: 'Popular Song',
                  url: 'https://genius.com/song',
                  full_title: 'Popular Song',
                  primary_artist: { id: 1, name: 'Artist', url: '' },
                  stats: { pageviews: 1500000 },
                },
              },
            ],
          },
        ],
      },
    });

    const results = engine.parseResponse(sampleResponse, defaultParams);

    expect(results.results.length).toBe(1);
    expect(results.results[0].content).toContain('1.5M views');
  });

  it('should handle empty or malformed response', () => {
    const emptyResults = engine.parseResponse('{}', defaultParams);
    expect(emptyResults.results).toEqual([]);

    const malformedResults = engine.parseResponse('not json', defaultParams);
    expect(malformedResults.results).toEqual([]);

    const noSections = engine.parseResponse(JSON.stringify({ response: {} }), defaultParams);
    expect(noSections.results).toEqual([]);
  });

  it('should search and return music results', async () => {
    const results = await fetchAndParse(engine, 'beatles');

    // API may return empty results due to network/rate-limiting issues
    expect(results.results).toBeDefined();
    expect(Array.isArray(results.results)).toBe(true);
    if (results.results.length > 0) {
      const first = results.results[0];
      expect(first.url).toBeTruthy();
      expect(first.title).toBeTruthy();
      expect(first.source).toBe('Genius');
    }
  }, 30000);
});

describe('GeniusLyricsEngine', () => {
  const engine = new GeniusLyricsEngine();

  const defaultParams: EngineParams = {
    page: 1,
    locale: 'en-US',
    safeSearch: 1,
    timeRange: '',
    engineData: {},
  };

  it('should have correct metadata', () => {
    expect(engine.name).toBe('genius lyrics');
    expect(engine.shortcut).toBe('gnl');
    expect(engine.categories).toContain('videos');
  });

  it('should append lyrics to search query', () => {
    const config = engine.buildRequest('hey jude', defaultParams);
    expect(config.url).toContain('q=hey+jude+lyrics');
    expect(config.url).toContain('genius.com/api/search/song');
  });

  it('should only include complete lyrics', () => {
    const sampleResponse = JSON.stringify({
      response: {
        sections: [
          {
            hits: [
              {
                type: 'song',
                result: {
                  id: 1,
                  title: 'Complete Lyrics',
                  url: 'https://genius.com/song1',
                  full_title: 'Complete Lyrics by Artist',
                  primary_artist: { id: 1, name: 'Artist', url: '' },
                  lyrics_state: 'complete',
                },
              },
              {
                type: 'song',
                result: {
                  id: 2,
                  title: 'Incomplete Lyrics',
                  url: 'https://genius.com/song2',
                  full_title: 'Incomplete Lyrics by Artist',
                  primary_artist: { id: 2, name: 'Artist', url: '' },
                  lyrics_state: 'incomplete',
                },
              },
            ],
          },
        ],
      },
    });

    const results = engine.parseResponse(sampleResponse, defaultParams);

    expect(results.results.length).toBe(1);
    expect(results.results[0].title).toBe('Complete Lyrics by Artist');
    expect(results.results[0].content).toContain('Lyrics by');
  });
});

async function fetchAndParse(engine: GeniusEngine, query: string) {
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
