import { describe, it, expect } from 'vitest';
import { OpenStreetMapEngine, OpenStreetMapReverseEngine } from './openstreetmap';
import type { EngineParams } from './engine';

describe('OpenStreetMapEngine', () => {
  const engine = new OpenStreetMapEngine();

  const defaultParams: EngineParams = {
    page: 1,
    locale: 'en-US',
    safeSearch: 1,
    timeRange: '',
    engineData: {},
  };

  it('should have correct metadata', () => {
    expect(engine.name).toBe('openstreetmap');
    expect(engine.shortcut).toBe('osm');
    expect(engine.categories).toContain('general');
    expect(engine.supportsPaging).toBe(false);
    expect(engine.maxPage).toBe(1);
  });

  it('should build correct Nominatim URL', () => {
    const config = engine.buildRequest('New York', defaultParams);
    expect(config.url).toContain('nominatim.openstreetmap.org/search');
    expect(config.url).toContain('q=New+York');
    expect(config.url).toContain('format=jsonv2');
    expect(config.url).toContain('addressdetails=1');
    expect(config.url).toContain('extratags=1');
    expect(config.method).toBe('GET');
  });

  it('should include language preference', () => {
    const config = engine.buildRequest('Berlin', {
      ...defaultParams,
      locale: 'de-DE',
    });
    expect(config.url).toContain('accept-language=de');
  });

  it('should parse place response', () => {
    const sampleResponse = JSON.stringify([
      {
        place_id: 123456,
        licence: 'OpenStreetMap contributors',
        osm_type: 'relation',
        osm_id: 175905,
        lat: '40.7127281',
        lon: '-74.0060152',
        class: 'place',
        type: 'city',
        place_rank: 16,
        importance: 0.9876,
        addresstype: 'city',
        name: 'New York',
        display_name: 'New York, United States',
        boundingbox: ['40.4960439', '40.9152414', '-74.2557249', '-73.7000090'],
        address: {
          city: 'New York',
          state: 'New York',
          country: 'United States',
          country_code: 'us',
        },
        extratags: {
          website: 'https://www.nyc.gov/',
          wikipedia: 'en:New York City',
          wikidata: 'Q60',
          population: '8336817',
        },
      },
    ]);

    const results = engine.parseResponse(sampleResponse, defaultParams);

    expect(results.results.length).toBe(1);

    const first = results.results[0];
    expect(first.url).toContain('openstreetmap.org/relation/175905');
    expect(first.title).toBe('New York');
    expect(first.content).toContain('City');
    expect(first.content).toContain('40.71273');
    expect(first.content).toContain('-74.00602');
    expect(first.content).toContain('New York, United States');
    expect(first.content).toContain('Pop: 8.3M');
    expect(first.source).toBe('OpenStreetMap');
    expect(first.category).toBe('general');
    expect(first.thumbnailUrl).toContain('tile.openstreetmap.org');
    expect(first.metadata?.placeId).toBe(123456);
    expect(first.metadata?.osmType).toBe('relation');
    expect(first.metadata?.osmId).toBe(175905);
    expect(first.metadata?.latitude).toBeCloseTo(40.7127281);
    expect(first.metadata?.longitude).toBeCloseTo(-74.0060152);
    expect(first.metadata?.importance).toBeCloseTo(0.9876);
    expect(first.metadata?.population).toBe('8336817');
    expect(first.metadata?.wikipedia).toBe('en:New York City');
  });

  it('should handle various place types', () => {
    const sampleResponse = JSON.stringify([
      {
        place_id: 1,
        osm_type: 'node',
        osm_id: 1,
        lat: '48.8584',
        lon: '2.2945',
        class: 'tourism',
        type: 'attraction',
        importance: 0.95,
        name: 'Eiffel Tower',
        display_name: 'Eiffel Tower, Paris, France',
        address: { city: 'Paris', country: 'France' },
      },
      {
        place_id: 2,
        osm_type: 'way',
        osm_id: 2,
        lat: '51.5074',
        lon: '-0.1278',
        class: 'highway',
        type: 'primary',
        importance: 0.8,
        name: 'Oxford Street',
        display_name: 'Oxford Street, London, UK',
        address: { city: 'London', country: 'United Kingdom' },
      },
    ]);

    const results = engine.parseResponse(sampleResponse, defaultParams);

    expect(results.results.length).toBe(2);
    expect(results.results[0].title).toBe('Eiffel Tower');
    expect(results.results[0].content).toContain('Tourism');
    expect(results.results[1].title).toBe('Oxford Street');
    expect(results.results[1].content).toContain('Road/Highway');
  });

  it('should generate map tile thumbnail', () => {
    const sampleResponse = JSON.stringify([
      {
        place_id: 1,
        osm_type: 'node',
        osm_id: 1,
        lat: '51.5074',
        lon: '-0.1278',
        class: 'place',
        type: 'city',
        importance: 0.9,
        name: 'London',
        display_name: 'London, UK',
      },
    ]);

    const results = engine.parseResponse(sampleResponse, defaultParams);

    expect(results.results.length).toBe(1);
    expect(results.results[0].thumbnailUrl).toMatch(/tile\.openstreetmap\.org\/\d+\/\d+\/\d+\.png/);
  });

  it('should handle empty response', () => {
    const emptyResults = engine.parseResponse('[]', defaultParams);
    expect(emptyResults.results).toEqual([]);

    const malformedResults = engine.parseResponse('not json', defaultParams);
    expect(malformedResults.results).toEqual([]);
  });

  it('should support custom limit', () => {
    const customEngine = new OpenStreetMapEngine({ limit: 10 });
    const config = customEngine.buildRequest('test', defaultParams);
    expect(config.url).toContain('limit=10');
  });

  it('should search and return location results', async () => {
    const results = await fetchAndParse(engine, 'Paris');

    expect(results.results.length).toBeGreaterThan(0);
    const first = results.results[0];
    expect(first.url).toBeTruthy();
    expect(first.title).toBeTruthy();
    expect(first.source).toBe('OpenStreetMap');
    expect(first.metadata?.latitude).toBeDefined();
    expect(first.metadata?.longitude).toBeDefined();
  }, 30000);
});

describe('OpenStreetMapReverseEngine', () => {
  const engine = new OpenStreetMapReverseEngine();

  const defaultParams: EngineParams = {
    page: 1,
    locale: 'en-US',
    safeSearch: 1,
    timeRange: '',
    engineData: {},
  };

  it('should have correct metadata', () => {
    expect(engine.name).toBe('openstreetmap reverse');
    expect(engine.shortcut).toBe('osmr');
  });

  it('should build correct reverse geocoding URL', () => {
    const config = engine.buildRequest('40.7128,-74.0060', defaultParams);
    expect(config.url).toContain('nominatim.openstreetmap.org/reverse');
    expect(config.url).toContain('lat=40.7128');
    expect(config.url).toContain('lon=-74.0060');
  });

  it('should handle space-separated coordinates', () => {
    const config = engine.buildRequest('40.7128 -74.0060', defaultParams);
    expect(config.url).toContain('lat=40.7128');
    expect(config.url).toContain('lon=-74.0060');
  });

  it('should handle invalid coordinates', () => {
    const config = engine.buildRequest('invalid', defaultParams);
    expect(config.url).toBe('');
  });

  it('should parse reverse geocoding response', () => {
    const sampleResponse = JSON.stringify({
      place_id: 123,
      osm_type: 'relation',
      osm_id: 175905,
      lat: '40.7127281',
      lon: '-74.0060152',
      display_name: 'New York City Hall, Broadway, Civic Center, Manhattan, New York County, New York, 10007, United States',
      name: 'New York City Hall',
      address: {
        building: 'New York City Hall',
        road: 'Broadway',
        city: 'New York',
        state: 'New York',
        country: 'United States',
      },
    });

    const results = engine.parseResponse(sampleResponse, defaultParams);

    expect(results.results.length).toBe(1);
    expect(results.results[0].title).toBe('New York City Hall');
    expect(results.results[0].content).toContain('Broadway');
    expect(results.results[0].metadata?.latitude).toBeCloseTo(40.7127281);
    expect(results.results[0].metadata?.longitude).toBeCloseTo(-74.0060152);
  });
});

async function fetchAndParse(engine: OpenStreetMapEngine, query: string) {
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
