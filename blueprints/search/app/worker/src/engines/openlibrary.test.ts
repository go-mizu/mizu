import { describe, it, expect } from 'vitest';
import { OpenLibraryEngine } from './openlibrary';
import type { EngineParams } from './engine';

describe('OpenLibraryEngine', () => {
  const engine = new OpenLibraryEngine();

  const defaultParams: EngineParams = {
    page: 1,
    locale: 'en-US',
    safeSearch: 1,
    timeRange: '',
    engineData: {},
  };

  it('should have correct metadata', () => {
    expect(engine.name).toBe('open library');
    expect(engine.shortcut).toBe('ol');
    expect(engine.categories).toContain('science');
    expect(engine.supportsPaging).toBe(true);
    expect(engine.maxPage).toBe(10);
  });

  it('should build correct API URL', () => {
    const config = engine.buildRequest('artificial intelligence', defaultParams);
    expect(config.url).toContain('openlibrary.org/search.json');
    expect(config.url).toContain('q=artificial+intelligence');
    expect(config.url).toContain('offset=0');
    expect(config.url).toContain('limit=10');
    expect(config.url).toContain('fields=');
    expect(config.method).toBe('GET');
  });

  it('should handle pagination', () => {
    const config = engine.buildRequest('test', {
      ...defaultParams,
      page: 3,
    });
    expect(config.url).toContain('offset=20'); // (3-1) * 10 = 20
  });

  it('should apply year filter for time range', () => {
    const config = engine.buildRequest('test', {
      ...defaultParams,
      timeRange: 'year',
    });
    expect(config.url).toContain('publish_year=');
  });

  it('should include proper headers', () => {
    const config = engine.buildRequest('test', defaultParams);
    expect(config.headers['User-Agent']).toBeTruthy();
    expect(config.headers['Accept']).toContain('application/json');
  });

  it('should parse API response', () => {
    const sampleResponse = JSON.stringify({
      numFound: 5000,
      start: 0,
      numFoundExact: true,
      docs: [
        {
          key: '/works/OL45804W',
          title: 'The Lord of the Rings',
          subtitle: 'The Fellowship of the Ring',
          author_name: ['J.R.R. Tolkien'],
          author_key: ['OL26320A'],
          first_publish_year: 1954,
          publish_year: [1954, 1966, 1973, 2001, 2020],
          publisher: ['Houghton Mifflin', 'Allen & Unwin', 'HarperCollins'],
          isbn: ['9780618640157', '0618640150', '9780261103573'],
          language: ['eng', 'spa', 'fra'],
          subject: ['Fiction', 'Fantasy', 'Middle Earth'],
          edition_count: 150,
          ebook_access: 'borrowable',
          has_fulltext: true,
          cover_i: 8739161,
          first_sentence: ['When Mr. Bilbo Baggins of Bag End announced that he would shortly be celebrating his eleventy-first birthday with a party of special magnificence, there was much talk and excitement in Hobbiton.'],
          ratings_average: 4.5,
          ratings_count: 12000,
          number_of_pages_median: 423,
        },
        {
          key: '/works/OL362427W',
          title: '1984',
          author_name: ['George Orwell'],
          first_publish_year: 1949,
          publisher: ['Secker and Warburg'],
          isbn: ['9780451524935'],
          edition_count: 200,
          cover_edition_key: 'OL1234567M',
          ebook_access: 'public',
        },
      ],
    });

    const results = engine.parseResponse(sampleResponse, defaultParams);

    expect(results.results.length).toBe(2);

    const first = results.results[0];
    expect(first.title).toBe('The Lord of the Rings: The Fellowship of the Ring');
    expect(first.url).toBe('https://openlibrary.org/works/OL45804W');
    expect(first.authors).toEqual(['J.R.R. Tolkien']);
    expect(first.publishedAt).toBe('1954');
    expect(first.thumbnailUrl).toBe('https://covers.openlibrary.org/b/id/8739161-M.jpg');
    expect(first.content).toContain('When Mr. Bilbo Baggins');
    expect(first.content).toContain('Publisher: Houghton Mifflin');
    expect(first.content).toContain('150 editions');
    expect(first.content).toContain('423 pages');
    expect(first.content).toContain('Available to read');
    expect(first.content).toContain('4.5/5 (12000 ratings)');
    expect(first.engine).toBe('open library');
    expect(first.category).toBe('science');
    expect(first.template).toBe('book');
    expect(first.metadata?.workKey).toBe('/works/OL45804W');
    expect(first.metadata?.isbn).toBe('9780618640157');
    expect(first.metadata?.editionCount).toBe(150);
    expect(first.metadata?.subjects).toContain('Fiction');

    const second = results.results[1];
    expect(second.title).toBe('1984');
    expect(second.authors).toEqual(['George Orwell']);
    expect(second.publishedAt).toBe('1949');
    expect(second.thumbnailUrl).toBe('https://covers.openlibrary.org/b/olid/OL1234567M-M.jpg');
    expect(second.content).toContain('Available to read');
  });

  it('should prefer ISBN-13 over ISBN-10', () => {
    const sampleResponse = JSON.stringify({
      docs: [
        {
          key: '/works/OL12345W',
          title: 'Test Book',
          isbn: ['0123456789', '9781234567890'],
        },
      ],
    });

    const results = engine.parseResponse(sampleResponse, defaultParams);
    expect(results.results[0].metadata?.isbn).toBe('9781234567890');
  });

  it('should handle books without cover', () => {
    const sampleResponse = JSON.stringify({
      docs: [
        {
          key: '/works/OL99999W',
          title: 'Book Without Cover',
          author_name: ['Unknown Author'],
        },
      ],
    });

    const results = engine.parseResponse(sampleResponse, defaultParams);
    expect(results.results[0].thumbnailUrl).toBeUndefined();
  });

  it('should handle empty or malformed response', () => {
    const emptyResults = engine.parseResponse('{}', defaultParams);
    expect(emptyResults.results).toEqual([]);

    const malformedResults = engine.parseResponse('not json', defaultParams);
    expect(malformedResults.results).toEqual([]);

    const noDocsResults = engine.parseResponse('{"numFound": 0}', defaultParams);
    expect(noDocsResults.results).toEqual([]);
  });

  it('should search Open Library and return book results', async () => {
    const results = await fetchAndParse(engine, 'python programming');

    expect(results.results.length).toBeGreaterThan(0);
    const first = results.results[0];
    expect(first.url).toContain('openlibrary.org');
    expect(first.title).toBeTruthy();
    expect(first.engine).toBe('open library');
    expect(first.category).toBe('science');
    expect(first.authors).toBeDefined();
    expect(Array.isArray(first.authors)).toBe(true);
  }, 30000);
});

async function fetchAndParse(engine: OpenLibraryEngine, query: string) {
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
