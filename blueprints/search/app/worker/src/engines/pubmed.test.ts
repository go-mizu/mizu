import { describe, it, expect } from 'vitest';
import { PubMedEngine, executePubMedSearch } from './pubmed';
import type { EngineParams } from './engine';

describe('PubMedEngine', () => {
  const engine = new PubMedEngine();

  const defaultParams: EngineParams = {
    page: 1,
    locale: 'en-US',
    safeSearch: 1,
    timeRange: '',
    engineData: {},
  };

  it('should have correct metadata', () => {
    expect(engine.name).toBe('pubmed');
    expect(engine.shortcut).toBe('pm');
    expect(engine.categories).toContain('science');
    expect(engine.supportsPaging).toBe(true);
    expect(engine.maxPage).toBe(10);
  });

  it('should build correct E-utilities search URL', () => {
    const config = engine.buildRequest('cancer treatment', defaultParams);
    expect(config.url).toContain('eutils.ncbi.nlm.nih.gov/entrez/eutils/esearch.fcgi');
    expect(config.url).toContain('db=pubmed');
    expect(config.url).toContain('term=cancer+treatment');
    expect(config.url).toContain('retmode=json');
    expect(config.method).toBe('GET');
  });

  it('should handle pagination', () => {
    const config = engine.buildRequest('test', {
      ...defaultParams,
      page: 3,
    });
    expect(config.url).toContain('retstart=20'); // (3-1) * 10 = 20
  });

  it('should apply time range filter', () => {
    const config = engine.buildRequest('test', {
      ...defaultParams,
      timeRange: 'week',
    });
    expect(config.url).toContain('datetype=pdat');
    expect(config.url).toContain('reldate=7');
  });

  it('should include proper headers', () => {
    const config = engine.buildRequest('test', defaultParams);
    expect(config.headers['User-Agent']).toBeTruthy();
    expect(config.headers['Accept']).toContain('application/json');
  });

  it('should parse E-search response and extract IDs', () => {
    const sampleResponse = JSON.stringify({
      header: {
        type: 'esearch',
        version: '0.3',
      },
      esearchresult: {
        count: '1000',
        retmax: '10',
        retstart: '0',
        idlist: ['39587654', '39587321', '39586999'],
        querytranslation: 'cancer[All Fields] AND treatment[All Fields]',
      },
    });

    const results = engine.parseResponse(sampleResponse, defaultParams);

    expect(results.engineData['pubmed_ids']).toBe('39587654,39587321,39586999');
    expect(results.engineData['pubmed_count']).toBe('1000');
  });

  it('should build correct summary request', () => {
    const ids = ['39587654', '39587321'];
    const config = engine.buildSummaryRequest(ids);

    expect(config.url).toContain('eutils.ncbi.nlm.nih.gov/entrez/eutils/esummary.fcgi');
    expect(config.url).toContain('db=pubmed');
    expect(config.url).toContain('id=39587654%2C39587321');
    expect(config.url).toContain('retmode=json');
  });

  it('should parse summary response', () => {
    const sampleResponse = JSON.stringify({
      result: {
        uids: ['39587654', '39587321'],
        '39587654': {
          uid: '39587654',
          pubdate: '2024 Jan 15',
          epubdate: '2024 Jan 10',
          source: 'Nature',
          authors: [
            { name: 'Smith J', authtype: 'Author' },
            { name: 'Johnson M', authtype: 'Author' },
          ],
          title: 'A Novel Approach to Cancer Treatment Using Immunotherapy',
          volume: '615',
          issue: '7951',
          pages: '123-130',
          fulljournalname: 'Nature',
          sortpubdate: '2024/01/15 00:00',
          articleids: [
            { idtype: 'pubmed', value: '39587654' },
            { idtype: 'doi', value: '10.1038/s41586-024-07123-4' },
          ],
          pmcrefcount: 25,
        },
        '39587321': {
          uid: '39587321',
          pubdate: '2024 Jan 12',
          source: 'Science',
          authors: [
            { name: 'Williams A', authtype: 'Author' },
          ],
          title: 'CRISPR Gene Editing for Cancer Therapy',
          fulljournalname: 'Science',
          sortpubdate: '2024/01/12 00:00',
          elocationid: 'doi: 10.1126/science.abc1234',
        },
      },
    });

    const results = engine.parseSummaryResponse(sampleResponse, defaultParams);

    expect(results.results.length).toBe(2);

    const first = results.results[0];
    expect(first.title).toBe('A Novel Approach to Cancer Treatment Using Immunotherapy');
    expect(first.url).toBe('https://pubmed.ncbi.nlm.nih.gov/39587654/');
    expect(first.authors).toEqual(['Smith J', 'Johnson M']);
    expect(first.doi).toBe('10.1038/s41586-024-07123-4');
    expect(first.journal).toBe('Nature');
    expect(first.content).toContain('Nature');
    expect(first.content).toContain('615(7951)');
    expect(first.content).toContain('25 citations');
    expect(first.engine).toBe('pubmed');
    expect(first.category).toBe('science');
    expect(first.template).toBe('paper');
    expect(first.metadata?.pmid).toBe('39587654');

    const second = results.results[1];
    expect(second.title).toBe('CRISPR Gene Editing for Cancer Therapy');
    expect(second.authors).toEqual(['Williams A']);
    expect(second.doi).toBe('10.1126/science.abc1234');
  });

  it('should handle empty or malformed response', () => {
    const emptyResults = engine.parseResponse('{}', defaultParams);
    expect(emptyResults.results).toEqual([]);
    expect(emptyResults.engineData['pubmed_ids']).toBeUndefined();

    const malformedResults = engine.parseResponse('not json', defaultParams);
    expect(malformedResults.results).toEqual([]);
  });

  it('should search PubMed and return article results', async () => {
    const results = await executePubMedSearch('machine learning', defaultParams);

    expect(results.results.length).toBeGreaterThan(0);
    const first = results.results[0];
    expect(first.url).toContain('pubmed.ncbi.nlm.nih.gov');
    expect(first.title).toBeTruthy();
    expect(first.engine).toBe('pubmed');
    expect(first.category).toBe('science');
    expect(first.authors).toBeDefined();
    expect(Array.isArray(first.authors)).toBe(true);
  }, 30000);
});
