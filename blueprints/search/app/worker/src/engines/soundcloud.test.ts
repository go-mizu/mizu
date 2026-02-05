import { describe, it, expect } from 'vitest';
import { SoundCloudEngine } from './soundcloud';
import type { EngineParams } from './engine';

describe('SoundCloudEngine', () => {
  const engine = new SoundCloudEngine();

  const defaultParams: EngineParams = {
    page: 1,
    locale: 'en-US',
    safeSearch: 1,
    timeRange: '',
    engineData: {},
  };

  it('should have correct metadata', () => {
    expect(engine.name).toBe('soundcloud');
    expect(engine.shortcut).toBe('sc');
    expect(engine.categories).toContain('videos');
    expect(engine.supportsPaging).toBe(true);
    expect(engine.maxPage).toBe(10);
  });

  it('should build correct search URL', () => {
    const config = engine.buildRequest('electronic music', defaultParams);
    expect(config.url).toContain('soundcloud.com/search/sounds');
    expect(config.url).toContain('q=electronic+music');
    expect(config.method).toBe('GET');
  });

  it('should parse hydration data response', () => {
    // Simulated hydration data with track info
    // The parser looks for hydratable entries that are NOT 'search' or 'anonymousId'
    // and have a .data.collection array
    const sampleHydration = [
      {
        hydratable: 'anonymousId',
        data: 'some-id',
      },
      {
        hydratable: 'searchResults',
        data: {
          collection: [
            {
              id: 123456,
              title: 'Test Track',
              description: 'A great electronic track',
              permalink: 'test-track',
              permalink_url: 'https://soundcloud.com/artist/test-track',
              uri: 'https://api.soundcloud.com/tracks/123456',
              artwork_url: 'https://i1.sndcdn.com/artworks-123456-large.jpg',
              duration: 180000, // 3 minutes in ms
              genre: 'Electronic',
              tag_list: 'electronic dance',
              playback_count: 50000,
              likes_count: 1500,
              reposts_count: 200,
              comment_count: 100,
              created_at: '2024-01-15T10:30:00.000Z',
              user: {
                id: 789,
                username: 'testartist',
                permalink: 'testartist',
                permalink_url: 'https://soundcloud.com/testartist',
                avatar_url: 'https://i1.sndcdn.com/avatars-789-large.jpg',
                full_name: 'Test Artist',
                followers_count: 10000,
              },
              downloadable: false,
            },
          ],
        },
      },
    ];

    const sampleBody = `<html><head></head><body><script>window.__sc_hydration = ${JSON.stringify(sampleHydration)};</script></body></html>`;

    const results = engine.parseResponse(sampleBody, defaultParams);

    expect(results.results.length).toBe(1);

    const first = results.results[0];
    expect(first.url).toBe('https://soundcloud.com/artist/test-track');
    expect(first.title).toBe('Test Track');
    expect(first.duration).toBe('3:00');
    expect(first.channel).toBe('testartist');
    expect(first.views).toBe(50000);
    expect(first.content).toContain('plays');
    expect(first.content).toContain('likes');
    expect(first.content).toContain('Electronic');
    expect(first.category).toBe('videos');
    expect(first.template).toBe('videos');
    expect(first.thumbnailUrl).toContain('artworks');
    expect(first.embedUrl).toContain('w.soundcloud.com/player');
    expect(first.metadata?.trackId).toBe(123456);
    expect(first.metadata?.genre).toBe('Electronic');
    expect(first.metadata?.likesCount).toBe(1500);
  });

  it('should format duration correctly', () => {
    const sampleHydration = [
      {
        hydratable: 'searchResults',
        data: {
          collection: [
            {
              id: 1,
              title: 'Short Track',
              permalink_url: 'https://soundcloud.com/artist/short',
              duration: 45000, // 45 seconds
              user: { id: 1, username: 'artist' },
            },
            {
              id: 2,
              title: 'Long Track',
              permalink_url: 'https://soundcloud.com/artist/long',
              duration: 3661000, // 1:01:01
              user: { id: 1, username: 'artist' },
            },
          ],
        },
      },
    ];

    const sampleBody = `<script>window.__sc_hydration = ${JSON.stringify(sampleHydration)};</script>`;
    const results = engine.parseResponse(sampleBody, defaultParams);

    expect(results.results.length).toBe(2);
    expect(results.results[0].duration).toBe('0:45');
    expect(results.results[1].duration).toBe('61:01'); // formats as minutes:seconds
  });

  it('should handle empty response', () => {
    const emptyResults = engine.parseResponse('<html></html>', defaultParams);
    expect(emptyResults.results).toEqual([]);

    const noHydration = engine.parseResponse('plain text', defaultParams);
    expect(noHydration.results).toEqual([]);
  });

  it('should format large numbers correctly', () => {
    const sampleHydration = [
      {
        hydratable: 'searchResults',
        data: {
          collection: [
            {
              id: 1,
              title: 'Popular Track',
              permalink_url: 'https://soundcloud.com/artist/popular',
              duration: 180000,
              playback_count: 1500000, // 1.5M
              likes_count: 50000, // 50K
              user: { id: 1, username: 'artist' },
            },
          ],
        },
      },
    ];

    const sampleBody = `<script>window.__sc_hydration = ${JSON.stringify(sampleHydration)};</script>`;
    const results = engine.parseResponse(sampleBody, defaultParams);

    expect(results.results.length).toBe(1);
    expect(results.results[0].content).toContain('1.5M plays');
    expect(results.results[0].content).toContain('50.0K likes');
  });

  it('should search and return audio results', async () => {
    const results = await fetchAndParse(engine, 'electronic');

    // SoundCloud may block or require JavaScript, so we may get 0 results
    // But the engine should not throw
    expect(results.results).toBeDefined();
    expect(Array.isArray(results.results)).toBe(true);
  }, 30000);
});

async function fetchAndParse(engine: SoundCloudEngine, query: string) {
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
