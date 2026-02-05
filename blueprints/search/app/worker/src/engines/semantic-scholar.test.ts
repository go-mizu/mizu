import { describe, it, expect } from 'vitest';
import { SemanticScholarEngine } from './semantic-scholar';
import type { EngineParams } from './engine';

describe('SemanticScholarEngine', () => {
  const engine = new SemanticScholarEngine();

  const defaultParams: EngineParams = {
    page: 1,
    locale: 'en-US',
    safeSearch: 1,
    timeRange: '',
    engineData: {},
  };

  it('should have correct metadata', () => {
    expect(engine.name).toBe('semantic scholar');
    expect(engine.shortcut).toBe('s2');
    expect(engine.categories).toContain('science');
    expect(engine.supportsPaging).toBe(true);
    expect(engine.maxPage).toBe(10);
  });

  it('should build correct API URL', () => {
    const config = engine.buildRequest('deep learning', defaultParams);
    expect(config.url).toContain('api.semanticscholar.org/graph/v1/paper/search');
    expect(config.url).toContain('query=deep+learning');
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
    const currentYear = new Date().getFullYear();
    expect(config.url).toContain(`year=${currentYear - 1}-${currentYear}`);
  });

  it('should include proper headers', () => {
    const config = engine.buildRequest('test', defaultParams);
    expect(config.headers['User-Agent']).toBeTruthy();
    expect(config.headers['Accept']).toContain('application/json');
  });

  it('should parse API response', () => {
    const sampleResponse = JSON.stringify({
      total: 1000,
      offset: 0,
      next: 10,
      data: [
        {
          paperId: 'abc123def456',
          externalIds: {
            DOI: '10.1038/nature12373',
            ArXiv: '1301.3781',
            PubMed: '23868264',
          },
          url: 'https://www.semanticscholar.org/paper/abc123def456',
          title: 'Efficient Estimation of Word Representations in Vector Space',
          abstract: 'We propose two novel model architectures for computing continuous vector representations of words from very large data sets.',
          venue: 'International Conference on Learning Representations',
          year: 2013,
          citationCount: 35000,
          influentialCitationCount: 5000,
          isOpenAccess: true,
          openAccessPdf: {
            url: 'https://arxiv.org/pdf/1301.3781.pdf',
            status: 'green',
          },
          fieldsOfStudy: ['Computer Science', 'Mathematics'],
          authors: [
            { authorId: '1', name: 'Tomas Mikolov' },
            { authorId: '2', name: 'Kai Chen' },
            { authorId: '3', name: 'Greg Corrado' },
            { authorId: '4', name: 'Jeffrey Dean' },
          ],
          publicationDate: '2013-01-16',
          journal: {
            name: 'ICLR',
          },
        },
        {
          paperId: 'xyz789ghi012',
          title: 'Attention Is All You Need',
          abstract: 'The dominant sequence transduction models are based on complex recurrent or convolutional neural networks.',
          year: 2017,
          citationCount: 80000,
          influentialCitationCount: 12000,
          isOpenAccess: false,
          authors: [
            { authorId: '5', name: 'Ashish Vaswani' },
          ],
          publicationVenue: {
            name: 'Neural Information Processing Systems',
          },
        },
      ],
    });

    const results = engine.parseResponse(sampleResponse, defaultParams);

    expect(results.results.length).toBe(2);

    const first = results.results[0];
    expect(first.title).toBe('Efficient Estimation of Word Representations in Vector Space');
    expect(first.url).toBe('https://www.semanticscholar.org/paper/abc123def456');
    expect(first.authors).toEqual(['Tomas Mikolov', 'Kai Chen', 'Greg Corrado', 'Jeffrey Dean']);
    expect(first.doi).toBe('10.1038/nature12373');
    expect(first.journal).toBe('ICLR');
    expect(first.publishedAt).toBe('2013-01-16T00:00:00.000Z');
    expect(first.content).toContain('We propose two novel model architectures');
    expect(first.content).toContain('35000 citations');
    expect(first.content).toContain('5000 influential');
    expect(first.content).toContain('Open Access');
    expect(first.content).toContain('[PDF:');
    expect(first.engine).toBe('semantic scholar');
    expect(first.category).toBe('science');
    expect(first.template).toBe('paper');
    expect(first.metadata?.paperId).toBe('abc123def456');
    expect(first.metadata?.arxivId).toBe('1301.3781');
    expect(first.metadata?.pubmedId).toBe('23868264');

    const second = results.results[1];
    expect(second.title).toBe('Attention Is All You Need');
    expect(second.authors).toEqual(['Ashish Vaswani']);
    expect(second.journal).toBe('Neural Information Processing Systems');
    expect(second.publishedAt).toBe('2017');
  });

  it('should handle empty or malformed response', () => {
    const emptyResults = engine.parseResponse('{}', defaultParams);
    expect(emptyResults.results).toEqual([]);

    const malformedResults = engine.parseResponse('not json', defaultParams);
    expect(malformedResults.results).toEqual([]);

    const noDataResults = engine.parseResponse('{"total": 0}', defaultParams);
    expect(noDataResults.results).toEqual([]);
  });

  it('should search Semantic Scholar and return paper results', async () => {
    const results = await fetchAndParse(engine, 'transformer neural network');

    expect(results.results.length).toBeGreaterThan(0);
    const first = results.results[0];
    expect(first.url).toContain('semanticscholar.org');
    expect(first.title).toBeTruthy();
    expect(first.engine).toBe('semantic scholar');
    expect(first.category).toBe('science');
    expect(first.authors).toBeDefined();
    expect(Array.isArray(first.authors)).toBe(true);
  }, 30000);
});

async function fetchAndParse(engine: SemanticScholarEngine, query: string) {
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
