import { describe, it, expect } from 'vitest';
import { IMDbEngine, IMDbAdvancedEngine, IMDbTitleEngine, IMDbNameEngine } from './imdb';
import type { EngineParams } from './engine';

describe('IMDbEngine', () => {
  const engine = new IMDbEngine();

  const defaultParams: EngineParams = {
    page: 1,
    locale: 'en-US',
    safeSearch: 1,
    timeRange: '',
    engineData: {},
  };

  it('should have correct metadata', () => {
    expect(engine.name).toBe('imdb');
    expect(engine.shortcut).toBe('imdb');
    expect(engine.categories).toContain('general');
    expect(engine.categories).toContain('videos');
    expect(engine.supportsPaging).toBe(false);
    expect(engine.maxPage).toBe(1);
  });

  it('should build correct suggestion API URL', () => {
    const config = engine.buildRequest('The Matrix', defaultParams);
    expect(config.url).toContain('v3.sg.media-imdb.com/suggestion');
    expect(config.url).toContain('The%20Matrix');
    expect(config.method).toBe('GET');
  });

  it('should parse title suggestion response', () => {
    const sampleResponse = JSON.stringify({
      d: [
        {
          id: 'tt0133093',
          l: 'The Matrix',
          s: 'Keanu Reeves, Laurence Fishburne',
          y: 1999,
          q: 'feature',
          rank: 80,
          i: {
            imageUrl: 'https://m.media-amazon.com/images/matrix.jpg',
            width: 300,
            height: 400,
          },
        },
        {
          id: 'tt10838180',
          l: 'The Matrix Resurrections',
          s: 'Keanu Reeves, Carrie-Anne Moss',
          y: 2021,
          q: 'feature',
          rank: 1200,
          i: {
            imageUrl: 'https://m.media-amazon.com/images/matrix4.jpg',
            width: 300,
            height: 400,
          },
        },
      ],
      q: 'the matrix',
      v: 1,
    });

    const results = engine.parseResponse(sampleResponse, defaultParams);

    expect(results.results.length).toBe(2);

    const first = results.results[0];
    expect(first.url).toBe('https://www.imdb.com/title/tt0133093/');
    expect(first.title).toBe('The Matrix');
    expect(first.content).toContain('Movie');
    expect(first.content).toContain('1999');
    expect(first.content).toContain('Keanu Reeves');
    expect(first.content).toContain('Rank #80');
    expect(first.source).toBe('IMDb');
    expect(first.thumbnailUrl).toBeTruthy();
    expect(first.publishedAt).toBe('1999-01-01T00:00:00Z');
    expect(first.metadata?.imdbId).toBe('tt0133093');
    expect(first.metadata?.isTitle).toBe(true);
    expect(first.metadata?.type).toBe('feature');
    expect(first.metadata?.year).toBe(1999);
    expect(first.metadata?.rank).toBe(80);
  });

  it('should parse TV series with year range', () => {
    const sampleResponse = JSON.stringify({
      d: [
        {
          id: 'tt0944947',
          l: 'Game of Thrones',
          s: 'Emilia Clarke, Peter Dinklage',
          yr: '2011-2019',
          q: 'TV series',
          rank: 50,
          i: {
            imageUrl: 'https://m.media-amazon.com/images/got.jpg',
            width: 300,
            height: 400,
          },
        },
      ],
      q: 'game of thrones',
      v: 1,
    });

    const results = engine.parseResponse(sampleResponse, defaultParams);

    expect(results.results.length).toBe(1);

    const first = results.results[0];
    expect(first.content).toContain('TV Series');
    expect(first.content).toContain('2011-2019');
    expect(first.metadata?.yearRange).toBe('2011-2019');
  });

  it('should parse person results', () => {
    const sampleResponse = JSON.stringify({
      d: [
        {
          id: 'nm0000206',
          l: 'Keanu Reeves',
          s: 'Actor, The Matrix (1999)',
          rank: 100,
          i: {
            imageUrl: 'https://m.media-amazon.com/images/keanu.jpg',
            width: 200,
            height: 300,
          },
        },
      ],
      q: 'keanu reeves',
      v: 1,
    });

    const results = engine.parseResponse(sampleResponse, defaultParams);

    expect(results.results.length).toBe(1);

    const first = results.results[0];
    expect(first.url).toBe('https://www.imdb.com/name/nm0000206/');
    expect(first.title).toBe('Keanu Reeves');
    expect(first.content).toContain('Person');
    expect(first.content).toContain('Actor, The Matrix (1999)');
    expect(first.metadata?.imdbId).toBe('nm0000206');
    expect(first.metadata?.isTitle).toBe(false);
    expect(first.metadata?.isPerson).toBe(true);
  });

  it('should handle various content types', () => {
    const sampleResponse = JSON.stringify({
      d: [
        { id: 'tt1', l: 'Movie', q: 'feature', y: 2020 },
        { id: 'tt2', l: 'TV Show', q: 'TV series', yr: '2020-' },
        { id: 'tt3', l: 'TV Movie', q: 'TV movie', y: 2021 },
        { id: 'tt4', l: 'Short', q: 'short', y: 2022 },
        { id: 'tt5', l: 'Video Game', q: 'video game', y: 2023 },
      ],
      q: 'test',
      v: 1,
    });

    const results = engine.parseResponse(sampleResponse, defaultParams);

    expect(results.results.length).toBe(5);
    expect(results.results[0].content).toContain('Movie');
    expect(results.results[1].content).toContain('TV Series');
    expect(results.results[2].content).toContain('TV Movie');
    expect(results.results[3].content).toContain('Short Film');
    expect(results.results[4].content).toContain('Video Game');
  });

  it('should handle empty response', () => {
    const emptyResults = engine.parseResponse('{}', defaultParams);
    expect(emptyResults.results).toEqual([]);

    const malformedResults = engine.parseResponse('not json', defaultParams);
    expect(malformedResults.results).toEqual([]);

    const noData = engine.parseResponse(JSON.stringify({ q: 'test', v: 1 }), defaultParams);
    expect(noData.results).toEqual([]);
  });

  it('should resize image URLs', () => {
    const sampleResponse = JSON.stringify({
      d: [
        {
          id: 'tt1',
          l: 'Test',
          i: {
            imageUrl: 'https://m.media-amazon.com/images/test._V1_UX1000_.jpg',
            width: 1000,
            height: 1500,
          },
        },
      ],
      q: 'test',
      v: 1,
    });

    const results = engine.parseResponse(sampleResponse, defaultParams);

    expect(results.results.length).toBe(1);
    expect(results.results[0].thumbnailUrl).toContain('_V1_UX300.');
  });

  it('should search and return IMDb results', async () => {
    const results = await fetchAndParse(engine, 'inception');

    expect(results.results.length).toBeGreaterThan(0);
    const first = results.results[0];
    expect(first.url).toContain('imdb.com');
    expect(first.title).toBeTruthy();
    expect(first.source).toBe('IMDb');
    expect(first.metadata?.imdbId).toBeTruthy();
  }, 30000);
});

describe('IMDbAdvancedEngine', () => {
  const engine = new IMDbAdvancedEngine();

  const defaultParams: EngineParams = {
    page: 1,
    locale: 'en-US',
    safeSearch: 1,
    timeRange: '',
    engineData: {},
  };

  it('should have correct metadata', () => {
    expect(engine.name).toBe('imdb advanced');
    expect(engine.shortcut).toBe('imdba');
    expect(engine.supportsPaging).toBe(true);
    expect(engine.maxPage).toBe(5);
  });

  it('should build correct find URL', () => {
    const config = engine.buildRequest('test', defaultParams);
    expect(config.url).toContain('imdb.com/find');
    expect(config.url).toContain('q=test');
  });
});

describe('IMDbTitleEngine', () => {
  const engine = new IMDbTitleEngine();

  it('should have correct metadata', () => {
    expect(engine.name).toBe('imdb titles');
  });

  it('should filter by title type', () => {
    const config = engine.buildRequest('test', {
      page: 1,
      locale: 'en-US',
      safeSearch: 1,
      timeRange: '',
      engineData: {},
    });
    expect(config.url).toContain('s=tt');
  });
});

describe('IMDbNameEngine', () => {
  const engine = new IMDbNameEngine();

  it('should have correct metadata', () => {
    expect(engine.name).toBe('imdb names');
  });

  it('should filter by name type', () => {
    const config = engine.buildRequest('test', {
      page: 1,
      locale: 'en-US',
      safeSearch: 1,
      timeRange: '',
      engineData: {},
    });
    expect(config.url).toContain('s=nm');
  });
});

async function fetchAndParse(engine: IMDbEngine, query: string) {
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
