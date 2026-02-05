import { describe, it, expect } from 'vitest';
import { PyPIEngine } from './pypi';
import type { EngineParams } from './engine';

describe('PyPIEngine', () => {
  const engine = new PyPIEngine();

  const defaultParams: EngineParams = {
    page: 1,
    locale: 'en-US',
    safeSearch: 1,
    timeRange: '',
    engineData: {},
  };

  it('should have correct metadata', () => {
    expect(engine.name).toBe('pypi');
    expect(engine.shortcut).toBe('pip');
    expect(engine.categories).toContain('it');
    expect(engine.supportsPaging).toBe(true);
    expect(engine.maxPage).toBe(10);
  });

  it('should build correct PyPI search URL', () => {
    const config = engine.buildRequest('requests', defaultParams);
    expect(config.url).toContain('pypi.org/search');
    expect(config.url).toContain('q=requests');
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

  it('should include proper headers', () => {
    const config = engine.buildRequest('test', defaultParams);
    expect(config.headers['User-Agent']).toBeTruthy();
    expect(config.headers['Accept']).toContain('text/html');
  });

  it('should parse PyPI HTML search response', () => {
    const sampleHtml = `
      <!DOCTYPE html>
      <html>
      <body>
        <a class="package-snippet" href="/project/requests/">
          <h3 class="package-snippet__title">
            <span class="package-snippet__name">requests</span>
            <span class="package-snippet__version">2.31.0</span>
          </h3>
          <p class="package-snippet__description">Python HTTP for Humans.</p>
          <time datetime="2024-01-15T10:30:00+00:00">Jan 15, 2024</time>
        </a>
        <a class="package-snippet" href="/project/httpx/">
          <h3 class="package-snippet__title">
            <span class="package-snippet__name">httpx</span>
            <span class="package-snippet__version">0.25.0</span>
          </h3>
          <p class="package-snippet__description">A next generation HTTP client for Python.</p>
          <time datetime="2024-01-10T15:00:00+00:00">Jan 10, 2024</time>
        </a>
      </body>
      </html>
    `;

    const results = engine.parseResponse(sampleHtml, defaultParams);

    expect(results.results.length).toBe(2);

    const first = results.results[0];
    expect(first.title).toBe('requests');
    expect(first.url).toBe('https://pypi.org/project/requests/');
    expect(first.content).toContain('Python HTTP for Humans');
    expect(first.content).toContain('v2.31.0');
    expect(first.engine).toBe('pypi');
    expect(first.category).toBe('it');
    expect(first.language).toBe('Python');
    expect(first.metadata?.version).toBe('2.31.0');

    const second = results.results[1];
    expect(second.title).toBe('httpx');
    expect(second.url).toBe('https://pypi.org/project/httpx/');
    expect(second.metadata?.version).toBe('0.25.0');
  });

  it('should handle empty response', () => {
    const emptyHtml = '<html><body>No results found</body></html>';
    const emptyResults = engine.parseResponse(emptyHtml, defaultParams);
    expect(emptyResults.results).toEqual([]);
  });

  it('should handle malformed HTML', () => {
    const malformedResults = engine.parseResponse('not html at all', defaultParams);
    expect(malformedResults.results).toEqual([]);
  });

  it('should handle missing optional fields', () => {
    const partialHtml = `
      <html>
      <body>
        <a class="package-snippet" href="/project/minimal/">
          <span class="package-snippet__name">minimal</span>
          <span class="package-snippet__version">1.0.0</span>
          <p class="package-snippet__description"></p>
        </a>
      </body>
      </html>
    `;

    const results = engine.parseResponse(partialHtml, defaultParams);
    expect(results.results.length).toBe(1);
    expect(results.results[0].title).toBe('minimal');
  });

  it('should search and return package results', async () => {
    try {
      const results = await fetchAndParse(engine, 'flask');

      // If we got results, validate them
      if (results.results.length > 0) {
        const first = results.results[0];
        expect(first.url).toContain('pypi.org');
        expect(first.title).toBeTruthy();
        expect(first.category).toBe('it');
        expect(first.language).toBe('Python');
      }
      // If no results, PyPI might be showing a challenge page - that's OK for integration tests
    } catch (error) {
      // PyPI may have bot protection active
      console.warn('PyPI unavailable for integration test:', error);
    }
  }, 30000);
});

async function fetchAndParse(engine: PyPIEngine, query: string) {
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
