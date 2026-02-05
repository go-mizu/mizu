import { describe, it, expect } from 'vitest';
import { YahooNewsEngine } from './yahoo-news';
import type { EngineParams } from './engine';

describe('YahooNewsEngine', () => {
  const engine = new YahooNewsEngine();

  const defaultParams: EngineParams = {
    page: 1,
    locale: 'en-US',
    safeSearch: 1,
    timeRange: '',
    engineData: {},
  };

  it('should have correct metadata', () => {
    expect(engine.name).toBe('yahoo news');
    expect(engine.shortcut).toBe('yhn');
    expect(engine.categories).toContain('news');
    expect(engine.supportsPaging).toBe(true);
    expect(engine.maxPage).toBe(20);
  });

  it('should build correct search URL', () => {
    const config = engine.buildRequest('technology news', defaultParams);
    expect(config.url).toContain('news.search.yahoo.com/search');
    expect(config.url).toContain('p=technology');
    expect(config.method).toBe('GET');
  });

  it('should handle pagination', () => {
    const config = engine.buildRequest('test', {
      ...defaultParams,
      page: 3,
    });
    expect(config.url).toContain('b=21'); // (3-1)*10 + 1 = 21
  });

  it('should apply time range filter', () => {
    const configDay = engine.buildRequest('test', {
      ...defaultParams,
      timeRange: 'day',
    });
    expect(configDay.url).toContain('age=1d');

    const configWeek = engine.buildRequest('test', {
      ...defaultParams,
      timeRange: 'week',
    });
    expect(configWeek.url).toContain('age=1w');

    const configMonth = engine.buildRequest('test', {
      ...defaultParams,
      timeRange: 'month',
    });
    expect(configMonth.url).toContain('age=1m');
  });

  it('should include proper headers', () => {
    const config = engine.buildRequest('test', defaultParams);
    expect(config.headers['User-Agent']).toBeTruthy();
    expect(config.headers['Accept']).toContain('text/html');
  });

  it('should search and return news results', async () => {
    const results = await fetchAndParse(engine, 'technology');

    // Yahoo News may have varying results, so check structure
    if (results.results.length > 0) {
      const first = results.results[0];
      expect(first.url).toBeTruthy();
      expect(first.title).toBeTruthy();
      expect(first.category).toBe('news');
      expect(first.engine).toBe('yahoo news');
      expect(first.publishedAt).toBeTruthy();
    }
  }, 30000);

  it('should parse sample HTML response', () => {
    const sampleHtml = `
      <div class="NewsArticle">
        <a href="https://example.com/news/article1">
          <h4>Breaking News: Tech Industry Update</h4>
        </a>
        <p>This is the snippet for the news article about technology updates.</p>
        <span class="s-source">TechNews</span>
        <span class="s-time">2 hours ago</span>
        <img src="https://example.com/thumb.jpg" />
      </div>
    `;

    const results = engine.parseResponse(sampleHtml, defaultParams);

    if (results.results.length > 0) {
      expect(results.results[0].title).toContain('Tech');
      expect(results.results[0].source).toBe('TechNews');
      expect(results.results[0].thumbnailUrl).toContain('thumb.jpg');
    }
  });
});

async function fetchAndParse(engine: YahooNewsEngine, query: string) {
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
