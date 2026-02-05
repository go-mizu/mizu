import { describe, it, expect } from 'vitest';
import { PkgGoDevEngine } from './pkg-go-dev';
import type { EngineParams } from './engine';

describe('PkgGoDevEngine', () => {
  const engine = new PkgGoDevEngine();

  const defaultParams: EngineParams = {
    page: 1,
    locale: 'en-US',
    safeSearch: 1,
    timeRange: '',
    engineData: {},
  };

  it('should have correct metadata', () => {
    expect(engine.name).toBe('pkg.go.dev');
    expect(engine.shortcut).toBe('go');
    expect(engine.categories).toContain('it');
    expect(engine.supportsPaging).toBe(true);
    expect(engine.maxPage).toBe(10);
  });

  it('should build correct pkg.go.dev search URL', () => {
    const config = engine.buildRequest('gin', defaultParams);
    expect(config.url).toContain('pkg.go.dev/search');
    expect(config.url).toContain('q=gin');
    expect(config.url).toContain('m=package');
    expect(config.method).toBe('GET');
  });

  it('should handle pagination', () => {
    const config = engine.buildRequest('test', {
      ...defaultParams,
      page: 3,
    });
    expect(config.url).toContain('page=3');
  });

  it('should not include page param for first page', () => {
    const config = engine.buildRequest('test', defaultParams);
    expect(config.url).not.toContain('page=');
  });

  it('should include proper headers', () => {
    const config = engine.buildRequest('test', defaultParams);
    expect(config.headers['User-Agent']).toBeTruthy();
    expect(config.headers['Accept']).toContain('text/html');
  });

  it('should parse pkg.go.dev HTML search response', () => {
    const sampleHtml = `
      <!DOCTYPE html>
      <html>
      <body>
        <div class="SearchSnippet">
          <div class="SearchSnippet-header">
            <a href="/github.com/gin-gonic/gin" data-test-id="snippet-title">github.com/gin-gonic/gin</a>
            <span class="SearchSnippet-header-version">v1.9.1</span>
          </div>
          <p class="SearchSnippet-synopsis">Gin is a HTTP web framework written in Go.</p>
          <div class="SearchSnippet-infoLabel">
            <span>Imported by: 85000</span>
            <span class="go-textSubtle">MIT</span>
            <span>Published: Jan 15, 2024</span>
          </div>
        </div>
        <div class="SearchSnippet">
          <div class="SearchSnippet-header">
            <a href="/github.com/labstack/echo" data-test-id="snippet-title">github.com/labstack/echo</a>
            <span class="SearchSnippet-header-version">v4.11.4</span>
          </div>
          <p class="SearchSnippet-synopsis">High performance, minimalist Go web framework.</p>
          <div class="SearchSnippet-infoLabel">
            <span>Imported by: 25000</span>
            <span class="go-textSubtle">MIT</span>
          </div>
        </div>
        <div class="Pagination"></div>
      </body>
      </html>
    `;

    const results = engine.parseResponse(sampleHtml, defaultParams);

    expect(results.results.length).toBe(2);

    const first = results.results[0];
    expect(first.title).toBe('github.com/gin-gonic/gin');
    expect(first.url).toBe('https://pkg.go.dev/github.com/gin-gonic/gin');
    expect(first.content).toContain('Gin is a HTTP web framework');
    expect(first.content).toContain('v1.9.1');
    expect(first.content).toContain('85.0k imports');
    expect(first.content).toContain('MIT');
    expect(first.engine).toBe('pkg.go.dev');
    expect(first.category).toBe('it');
    expect(first.language).toBe('Go');
    expect(first.metadata?.version).toBe('v1.9.1');
    expect(first.metadata?.importedBy).toBe(85000);
    expect(first.metadata?.license).toBe('MIT');

    const second = results.results[1];
    expect(second.title).toBe('github.com/labstack/echo');
    expect(second.metadata?.version).toBe('v4.11.4');
    expect(second.metadata?.importedBy).toBe(25000);
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
        <div class="SearchSnippet">
          <a href="/github.com/user/minimal" data-test-id="snippet-title">github.com/user/minimal</a>
          <span class="SearchSnippet-header-version">v1.0.0</span>
        </div>
        <div class="Pagination"></div>
      </body>
      </html>
    `;

    const results = engine.parseResponse(partialHtml, defaultParams);
    expect(results.results.length).toBe(1);
    expect(results.results[0].title).toBe('github.com/user/minimal');
  });

  it('should fallback to link extraction when no snippets found', () => {
    const fallbackHtml = `
      <html>
      <body>
        <a href="/github.com/user/repo">github.com/user/repo</a>
        <p class="synopsis">A cool package</p>
        <a href="/golang.org/x/tools">golang.org/x/tools</a>
      </body>
      </html>
    `;

    const results = engine.parseResponse(fallbackHtml, defaultParams);
    expect(results.results.length).toBeGreaterThan(0);
  });

  it('should search and return Go package results', async () => {
    const results = await fetchAndParse(engine, 'http router');

    expect(results.results.length).toBeGreaterThan(0);
    const first = results.results[0];
    expect(first.url).toContain('pkg.go.dev');
    expect(first.title).toBeTruthy();
    expect(first.category).toBe('it');
    expect(first.language).toBe('Go');
  }, 30000);
});

async function fetchAndParse(engine: PkgGoDevEngine, query: string) {
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
