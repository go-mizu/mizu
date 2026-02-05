import { describe, it, expect } from 'vitest';
import { MojeekEngine } from './mojeek';

describe('MojeekEngine', () => {
  const engine = new MojeekEngine();

  describe('metadata', () => {
    it('should have correct name and shortcut', () => {
      expect(engine.name).toBe('mojeek');
      expect(engine.shortcut).toBe('mj');
    });

    it('should have correct categories', () => {
      expect(engine.categories).toContain('general');
      expect(engine.categories.length).toBe(1);
    });

    it('should have correct paging settings', () => {
      expect(engine.supportsPaging).toBe(true);
      expect(engine.maxPage).toBe(100);
    });

    it('should have correct timeout and weight', () => {
      expect(engine.timeout).toBe(10000);
      expect(engine.weight).toBe(0.8);
    });

    it('should be enabled by default', () => {
      expect(engine.disabled).toBe(false);
    });
  });

  describe('buildRequest', () => {
    it('should build correct base URL', () => {
      const config = engine.buildRequest('test query', {
        page: 1,
        locale: 'en-US',
        safeSearch: 1,
        timeRange: '',
        engineData: {},
      });
      expect(config.url).toContain('https://www.mojeek.com/search');
      expect(config.url).toContain('q=test+query');
      expect(config.method).toBe('GET');
    });

    it('should handle pagination correctly', () => {
      const page1 = engine.buildRequest('test', {
        page: 1,
        locale: 'en-US',
        safeSearch: 1,
        timeRange: '',
        engineData: {},
      });
      expect(page1.url).not.toContain('s=');

      const page2 = engine.buildRequest('test', {
        page: 2,
        locale: 'en-US',
        safeSearch: 1,
        timeRange: '',
        engineData: {},
      });
      expect(page2.url).toContain('s=10');

      const page5 = engine.buildRequest('test', {
        page: 5,
        locale: 'en-US',
        safeSearch: 1,
        timeRange: '',
        engineData: {},
      });
      expect(page5.url).toContain('s=40');
    });

    describe('time range filter', () => {
      it('should add date filter for day', () => {
        const config = engine.buildRequest('test', {
          page: 1,
          locale: 'en-US',
          safeSearch: 1,
          timeRange: 'day',
          engineData: {},
        });
        expect(config.url).toContain('date=day');
      });

      it('should add date filter for week', () => {
        const config = engine.buildRequest('test', {
          page: 1,
          locale: 'en-US',
          safeSearch: 1,
          timeRange: 'week',
          engineData: {},
        });
        expect(config.url).toContain('date=week');
      });

      it('should add date filter for month', () => {
        const config = engine.buildRequest('test', {
          page: 1,
          locale: 'en-US',
          safeSearch: 1,
          timeRange: 'month',
          engineData: {},
        });
        expect(config.url).toContain('date=month');
      });

      it('should add date filter for year', () => {
        const config = engine.buildRequest('test', {
          page: 1,
          locale: 'en-US',
          safeSearch: 1,
          timeRange: 'year',
          engineData: {},
        });
        expect(config.url).toContain('date=year');
      });

      it('should not add date filter when empty', () => {
        const config = engine.buildRequest('test', {
          page: 1,
          locale: 'en-US',
          safeSearch: 1,
          timeRange: '',
          engineData: {},
        });
        expect(config.url).not.toContain('date=');
      });
    });

    describe('safe search', () => {
      it('should set safe search off when safeSearch is 0', () => {
        const config = engine.buildRequest('test', {
          page: 1,
          locale: 'en-US',
          safeSearch: 0,
          timeRange: '',
          engineData: {},
        });
        expect(config.url).toContain('safe=0');
      });

      it('should set safe search moderate when safeSearch is 1', () => {
        const config = engine.buildRequest('test', {
          page: 1,
          locale: 'en-US',
          safeSearch: 1,
          timeRange: '',
          engineData: {},
        });
        expect(config.url).toContain('safe=1');
      });

      it('should set safe search strict when safeSearch is 2', () => {
        const config = engine.buildRequest('test', {
          page: 1,
          locale: 'en-US',
          safeSearch: 2,
          timeRange: '',
          engineData: {},
        });
        expect(config.url).toContain('safe=2');
      });
    });

    describe('locale/region', () => {
      it('should set language bias from locale', () => {
        const config = engine.buildRequest('test', {
          page: 1,
          locale: 'en-US',
          safeSearch: 1,
          timeRange: '',
          engineData: {},
        });
        expect(config.url).toContain('lb=en');
      });

      it('should set region code from locale', () => {
        const config = engine.buildRequest('test', {
          page: 1,
          locale: 'de-DE',
          safeSearch: 1,
          timeRange: '',
          engineData: {},
        });
        expect(config.url).toContain('arc=DE');
        expect(config.url).toContain('lb=de');
      });

      it('should handle locale without region', () => {
        const config = engine.buildRequest('test', {
          page: 1,
          locale: 'fr',
          safeSearch: 1,
          timeRange: '',
          engineData: {},
        });
        expect(config.url).toContain('lb=fr');
      });
    });

    it('should include format parameter', () => {
      const config = engine.buildRequest('test', {
        page: 1,
        locale: 'en-US',
        safeSearch: 1,
        timeRange: '',
        engineData: {},
      });
      expect(config.url).toContain('fmt=html');
    });
  });

  describe('parseResponse', () => {
    const defaultParams = {
      page: 1,
      locale: 'en-US',
      safeSearch: 1 as const,
      timeRange: '' as const,
      engineData: {},
    };

    it('should parse Mojeek result HTML', () => {
      const mockHtml = `
        <ul class="results-standard">
          <li class="result">
            <a class="ob" href="https://example.com/page">
              <h2>Example Page Title</h2>
            </a>
            <p class="s">This is the description of the search result.</p>
          </li>
          <li class="result">
            <a class="ob" href="https://another.com">
              <h2>Another Result</h2>
            </a>
            <p class="s">Another description here.</p>
          </li>
        </ul>
      `;

      const results = engine.parseResponse(mockHtml, defaultParams);

      expect(results.results.length).toBe(2);

      const first = results.results[0];
      expect(first.url).toBe('https://example.com/page');
      expect(first.title).toBe('Example Page Title');
      expect(first.content).toContain('description');
      expect(first.engine).toBe('mojeek');
      expect(first.category).toBe('general');

      const second = results.results[1];
      expect(second.url).toBe('https://another.com');
      expect(second.title).toBe('Another Result');
    });

    it('should handle results without wrapper class', () => {
      const mockHtml = `
        <li class="result">
          <a href="https://example.com">
            <h2>Test Title</h2>
          </a>
          <p class="snippet">Some description text.</p>
        </li>
      `;

      const results = engine.parseResponse(mockHtml, defaultParams);

      expect(results.results.length).toBe(1);
      expect(results.results[0].url).toBe('https://example.com');
      expect(results.results[0].title).toBe('Test Title');
    });

    it('should return empty results for no results page', () => {
      const noResultsHtml = `
        <html>
          <body>
            <div>No results found for your query</div>
          </body>
        </html>
      `;

      const results = engine.parseResponse(noResultsHtml, defaultParams);
      expect(results.results).toEqual([]);
    });

    it('should skip Mojeek internal links', () => {
      const mockHtml = `
        <li class="result">
          <a class="ob" href="https://www.mojeek.com/search?q=related">
            <h2>Related Search</h2>
          </a>
        </li>
        <li class="result">
          <a class="ob" href="https://example.com">
            <h2>External Site</h2>
          </a>
        </li>
      `;

      const results = engine.parseResponse(mockHtml, defaultParams);

      expect(results.results.length).toBe(1);
      expect(results.results[0].url).toBe('https://example.com');
    });

    it('should parse related searches', () => {
      const mockHtml = `
        <li class="result">
          <a class="ob" href="https://example.com">
            <h2>Result</h2>
          </a>
        </li>
        <div class="related-searches">
          <a href="#">Related Search 1</a>
          <a href="#">Related Search 2</a>
        </div>
      `;

      const results = engine.parseResponse(mockHtml, defaultParams);

      expect(results.suggestions.length).toBe(2);
      expect(results.suggestions).toContain('Related Search 1');
      expect(results.suggestions).toContain('Related Search 2');
    });

    it('should parse "Also try" suggestions', () => {
      const mockHtml = `
        <li class="result">
          <a class="ob" href="https://example.com">
            <h2>Result</h2>
          </a>
        </li>
        <div>Also try: <a>suggestion one</a> <a>suggestion two</a></div>
      `;

      const results = engine.parseResponse(mockHtml, defaultParams);

      expect(results.suggestions).toContain('suggestion one');
      expect(results.suggestions).toContain('suggestion two');
    });

    it('should return empty results for empty HTML', () => {
      const results = engine.parseResponse('', defaultParams);
      expect(results.results).toEqual([]);
    });

    it('should handle description in different elements', () => {
      const mockHtml = `
        <li class="result">
          <a class="ob" href="https://example.com">
            <h2>Title</h2>
          </a>
          <span class="desc">Description from span element.</span>
        </li>
      `;

      const results = engine.parseResponse(mockHtml, defaultParams);

      expect(results.results.length).toBe(1);
      expect(results.results[0].content).toContain('Description');
    });
  });

  describe('live API test', () => {
    it('should search and return web results', async () => {
      const params = {
        page: 1,
        locale: 'en-US',
        safeSearch: 1 as const,
        timeRange: '' as const,
        engineData: {},
      };
      const config = engine.buildRequest('javascript tutorial', params);
      const res = await fetch(config.url, {
        headers: config.headers,
      });

      expect(res.ok).toBe(true);

      const body = await res.text();
      const results = engine.parseResponse(body, params);

      expect(results).toBeDefined();
      expect(results.results).toBeDefined();
      // Mojeek should return results since it's privacy-friendly
      expect(results.results.length).toBeGreaterThan(0);

      if (results.results.length > 0) {
        const first = results.results[0];
        expect(first.url).toBeTruthy();
        expect(first.url).toContain('http');
        expect(first.title).toBeTruthy();
        expect(first.engine).toBe('mojeek');
        expect(first.category).toBe('general');
      }
    }, 30000);
  });
});
