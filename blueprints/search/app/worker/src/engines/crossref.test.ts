import { describe, it, expect } from 'vitest';
import { CrossrefEngine } from './crossref';
import type { EngineParams } from './engine';

describe('CrossrefEngine', () => {
  const engine = new CrossrefEngine();

  const defaultParams: EngineParams = {
    page: 1,
    locale: 'en-US',
    safeSearch: 1,
    timeRange: '',
    engineData: {},
  };

  it('should have correct metadata', () => {
    expect(engine.name).toBe('crossref');
    expect(engine.shortcut).toBe('cr');
    expect(engine.categories).toContain('science');
    expect(engine.supportsPaging).toBe(true);
    expect(engine.maxPage).toBe(10);
  });

  it('should build correct API URL', () => {
    const config = engine.buildRequest('climate change', defaultParams);
    expect(config.url).toContain('api.crossref.org/works');
    expect(config.url).toContain('query=climate+change');
    expect(config.url).toContain('offset=0');
    expect(config.url).toContain('rows=10');
    expect(config.url).toContain('sort=relevance');
    expect(config.method).toBe('GET');
  });

  it('should handle pagination', () => {
    const config = engine.buildRequest('test', {
      ...defaultParams,
      page: 3,
    });
    expect(config.url).toContain('offset=20'); // (3-1) * 10 = 20
  });

  it('should apply date filter for time range', () => {
    const config = engine.buildRequest('test', {
      ...defaultParams,
      timeRange: 'month',
    });
    expect(config.url).toContain('filter=from-pub-date');
  });

  it('should include polite pool headers', () => {
    const config = engine.buildRequest('test', defaultParams);
    expect(config.headers['User-Agent']).toContain('MizuSearch');
    expect(config.headers['Accept']).toContain('application/json');
  });

  it('should parse API response', () => {
    const sampleResponse = JSON.stringify({
      status: 'ok',
      'message-type': 'work-list',
      'message-version': '1.0.0',
      message: {
        'total-results': 10000,
        items: [
          {
            DOI: '10.1038/nature12373',
            URL: 'http://dx.doi.org/10.1038/nature12373',
            title: ['Quantum computing with superconducting circuits'],
            'container-title': ['Nature'],
            author: [
              { given: 'John', family: 'Martinis' },
              { given: 'Andrew', family: 'Cleland' },
            ],
            published: {
              'date-parts': [[2013, 9, 5]],
            },
            type: 'journal-article',
            publisher: 'Springer Nature',
            volume: '501',
            issue: '7466',
            page: '84-88',
            'is-referenced-by-count': 5000,
            'references-count': 50,
            subject: ['Multidisciplinary'],
            abstract: '<jats:p>This paper reviews recent advances in quantum computing using superconducting circuits.</jats:p>',
            license: [
              {
                URL: 'https://www.nature.com/nature/journal/v501/n7466/full/nature12373.html',
                'content-version': 'vor',
              },
            ],
            link: [
              {
                URL: 'https://www.nature.com/articles/nature12373.pdf',
                'content-type': 'application/pdf',
              },
            ],
            ISSN: ['0028-0836', '1476-4687'],
          },
          {
            DOI: '10.1126/science.1231930',
            title: ['CRISPR-Cas Systems for Editing Genomes'],
            author: [
              { name: 'Jennifer Doudna' },
            ],
            issued: {
              'date-parts': [[2013, 2, 15]],
            },
            type: 'journal-article',
            publisher: 'AAAS',
            'is-referenced-by-count': 8000,
          },
        ],
        'items-per-page': 10,
        query: {
          'start-index': 0,
          'search-terms': 'quantum computing',
        },
      },
    });

    const results = engine.parseResponse(sampleResponse, defaultParams);

    expect(results.results.length).toBe(2);

    const first = results.results[0];
    expect(first.title).toBe('Quantum computing with superconducting circuits');
    expect(first.url).toBe('http://dx.doi.org/10.1038/nature12373');
    expect(first.authors).toEqual(['John Martinis', 'Andrew Cleland']);
    expect(first.doi).toBe('10.1038/nature12373');
    expect(first.journal).toBe('Nature');
    expect(first.publishedAt).toBeTruthy();
    expect(first.content).toContain('This paper reviews recent advances');
    expect(first.content).toContain('journal article');
    expect(first.content).toContain('5000 citations');
    expect(first.content).toContain('Springer Nature');
    expect(first.content).toContain('[PDF:');
    expect(first.engine).toBe('crossref');
    expect(first.category).toBe('science');
    expect(first.template).toBe('paper');
    expect(first.metadata?.type).toBe('journal-article');
    expect(first.metadata?.volume).toBe('501');
    expect(first.metadata?.issue).toBe('7466');
    expect(first.metadata?.page).toBe('84-88');
    expect(first.metadata?.issn).toContain('0028-0836');

    const second = results.results[1];
    expect(second.title).toBe('CRISPR-Cas Systems for Editing Genomes');
    expect(second.authors).toEqual(['Jennifer Doudna']);
    expect(second.doi).toBe('10.1126/science.1231930');
  });

  it('should handle works without DOI URL', () => {
    const sampleResponse = JSON.stringify({
      status: 'ok',
      message: {
        items: [
          {
            DOI: '10.1234/test.5678',
            title: ['Test Paper Without URL'],
            author: [],
          },
        ],
      },
    });

    const results = engine.parseResponse(sampleResponse, defaultParams);

    expect(results.results.length).toBe(1);
    expect(results.results[0].url).toBe('https://doi.org/10.1234/test.5678');
  });

  it('should handle empty or malformed response', () => {
    const emptyResults = engine.parseResponse('{}', defaultParams);
    expect(emptyResults.results).toEqual([]);

    const malformedResults = engine.parseResponse('not json', defaultParams);
    expect(malformedResults.results).toEqual([]);

    const noItemsResults = engine.parseResponse('{"status": "ok", "message": {}}', defaultParams);
    expect(noItemsResults.results).toEqual([]);
  });

  it('should search Crossref and return publication results', async () => {
    const results = await fetchAndParse(engine, 'machine learning');

    expect(results.results.length).toBeGreaterThan(0);
    const first = results.results[0];
    expect(first.url).toBeTruthy();
    expect(first.title).toBeTruthy();
    expect(first.engine).toBe('crossref');
    expect(first.category).toBe('science');
    expect(first.doi).toBeTruthy();
  }, 30000);
});

async function fetchAndParse(engine: CrossrefEngine, query: string) {
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
