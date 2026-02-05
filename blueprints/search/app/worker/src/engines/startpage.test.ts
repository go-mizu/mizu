import { describe, it, expect } from 'vitest';
import { StartpageEngine } from './startpage';

describe('StartpageEngine', () => {
  const engine = new StartpageEngine();

  describe('metadata', () => {
    it('should have correct name and shortcut', () => {
      expect(engine.name).toBe('startpage');
      expect(engine.shortcut).toBe('sp');
    });

    it('should have correct categories', () => {
      expect(engine.categories).toContain('general');
      expect(engine.categories.length).toBe(1);
    });

    it('should have correct paging settings', () => {
      expect(engine.supportsPaging).toBe(true);
      expect(engine.maxPage).toBe(20);
    });

    it('should have correct timeout and weight', () => {
      expect(engine.timeout).toBe(12000);
      expect(engine.weight).toBe(0.95);
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
      expect(config.url).toContain('https://www.startpage.com/sp/search');
      expect(config.url).toContain('query=test+query');
      expect(config.method).toBe('GET');
    });

    it('should set category to web', () => {
      const config = engine.buildRequest('test', {
        page: 1,
        locale: 'en-US',
        safeSearch: 1,
        timeRange: '',
        engineData: {},
      });
      expect(config.url).toContain('cat=web');
    });

    it('should handle pagination correctly', () => {
      const page1 = engine.buildRequest('test', {
        page: 1,
        locale: 'en-US',
        safeSearch: 1,
        timeRange: '',
        engineData: {},
      });
      expect(page1.url).not.toContain('page=');

      const page2 = engine.buildRequest('test', {
        page: 2,
        locale: 'en-US',
        safeSearch: 1,
        timeRange: '',
        engineData: {},
      });
      expect(page2.url).toContain('page=2');

      const page5 = engine.buildRequest('test', {
        page: 5,
        locale: 'en-US',
        safeSearch: 1,
        timeRange: '',
        engineData: {},
      });
      expect(page5.url).toContain('page=5');
    });

    describe('time range filter', () => {
      it('should add with_date filter for day', () => {
        const config = engine.buildRequest('test', {
          page: 1,
          locale: 'en-US',
          safeSearch: 1,
          timeRange: 'day',
          engineData: {},
        });
        expect(config.url).toContain('with_date=d');
      });

      it('should add with_date filter for week', () => {
        const config = engine.buildRequest('test', {
          page: 1,
          locale: 'en-US',
          safeSearch: 1,
          timeRange: 'week',
          engineData: {},
        });
        expect(config.url).toContain('with_date=w');
      });

      it('should add with_date filter for month', () => {
        const config = engine.buildRequest('test', {
          page: 1,
          locale: 'en-US',
          safeSearch: 1,
          timeRange: 'month',
          engineData: {},
        });
        expect(config.url).toContain('with_date=m');
      });

      it('should add with_date filter for year', () => {
        const config = engine.buildRequest('test', {
          page: 1,
          locale: 'en-US',
          safeSearch: 1,
          timeRange: 'year',
          engineData: {},
        });
        expect(config.url).toContain('with_date=y');
      });

      it('should not add with_date filter when empty', () => {
        const config = engine.buildRequest('test', {
          page: 1,
          locale: 'en-US',
          safeSearch: 1,
          timeRange: '',
          engineData: {},
        });
        expect(config.url).not.toContain('with_date=');
      });
    });

    describe('locale/region', () => {
      it('should set language from locale', () => {
        const config = engine.buildRequest('test', {
          page: 1,
          locale: 'en-US',
          safeSearch: 1,
          timeRange: '',
          engineData: {},
        });
        expect(config.url).toContain('language=en');
      });

      it('should set UI language from locale', () => {
        const config = engine.buildRequest('test', {
          page: 1,
          locale: 'de-DE',
          safeSearch: 1,
          timeRange: '',
          engineData: {},
        });
        expect(config.url).toContain('lui=de');
      });

      it('should set locale code for different regions', () => {
        const deConfig = engine.buildRequest('test', {
          page: 1,
          locale: 'de-DE',
          safeSearch: 1,
          timeRange: '',
          engineData: {},
        });
        expect(deConfig.url).toContain('sc=de-DE');

        const frConfig = engine.buildRequest('test', {
          page: 1,
          locale: 'fr-FR',
          safeSearch: 1,
          timeRange: '',
          engineData: {},
        });
        expect(frConfig.url).toContain('sc=fr-FR');
      });
    });

    describe('safe search cookies', () => {
      it('should set strict preferences cookie when safeSearch is 2', () => {
        const config = engine.buildRequest('test', {
          page: 1,
          locale: 'en-US',
          safeSearch: 2,
          timeRange: '',
          engineData: {},
        });
        expect(config.cookies.length).toBeGreaterThan(0);
        expect(config.cookies.some(c => c.includes('preferences'))).toBe(true);
      });

      it('should set off preferences cookie when safeSearch is 0', () => {
        const config = engine.buildRequest('test', {
          page: 1,
          locale: 'en-US',
          safeSearch: 0,
          timeRange: '',
          engineData: {},
        });
        expect(config.cookies.length).toBeGreaterThan(0);
      });

      it('should have no preferences cookie for moderate safeSearch', () => {
        const config = engine.buildRequest('test', {
          page: 1,
          locale: 'en-US',
          safeSearch: 1,
          timeRange: '',
          engineData: {},
        });
        expect(config.cookies).toEqual([]);
      });
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

    it('should parse Startpage w-gl result HTML', () => {
      const mockHtml = `
        <div class="w-gl__result">
          <a class="w-gl__result-url" href="https://example.com/page">example.com</a>
          <h3 class="w-gl__result-title">Example Page Title</h3>
          <p class="w-gl__description">This is the description of the search result.</p>
        </div>
        <div class="w-gl__result">
          <a class="w-gl__result-url" href="https://another.com">another.com</a>
          <h3 class="w-gl__result-title">Another Result</h3>
          <p class="w-gl__description">Another description here.</p>
        </div>
      `;

      const results = engine.parseResponse(mockHtml, defaultParams);

      expect(results.results.length).toBe(2);

      const first = results.results[0];
      expect(first.url).toBe('https://example.com/page');
      expect(first.title).toBe('Example Page Title');
      expect(first.content).toContain('description');
      expect(first.engine).toBe('startpage');
      expect(first.category).toBe('general');

      const second = results.results[1];
      expect(second.url).toBe('https://another.com');
      expect(second.title).toBe('Another Result');
    });

    it('should handle results with nested h3 in link', () => {
      const mockHtml = `
        <div class="w-gl__result">
          <a href="https://example.com">
            <h3>Title Inside Link</h3>
          </a>
          <p class="w-gl__description">Description text.</p>
        </div>
      `;

      const results = engine.parseResponse(mockHtml, defaultParams);

      expect(results.results.length).toBe(1);
      expect(results.results[0].url).toBe('https://example.com');
      expect(results.results[0].title).toBe('Title Inside Link');
    });

    it('should unwrap Startpage proxy URLs', () => {
      const mockHtml = `
        <div class="w-gl__result">
          <a class="w-gl__result-url" href="https://www.startpage.com/do/proxy?u=https%3A%2F%2Fexample.com%2Fpage">example.com</a>
          <h3 class="w-gl__result-title">Proxied Result</h3>
        </div>
      `;

      const results = engine.parseResponse(mockHtml, defaultParams);

      expect(results.results.length).toBe(1);
      expect(results.results[0].url).toBe('https://example.com/page');
    });

    it('should return empty results for CAPTCHA page', () => {
      const captchaHtml = `
        <html>
          <body>
            <div class="g-recaptcha">Please verify you are human</div>
          </body>
        </html>
      `;

      const results = engine.parseResponse(captchaHtml, defaultParams);
      expect(results.results).toEqual([]);
    });

    it('should return empty results for no results page', () => {
      const noResultsHtml = `
        <html>
          <body>
            <div class="error-message">No results found</div>
          </body>
        </html>
      `;

      const results = engine.parseResponse(noResultsHtml, defaultParams);
      expect(results.results).toEqual([]);
    });

    it('should skip Startpage internal links', () => {
      const mockHtml = `
        <div class="w-gl__result">
          <a class="w-gl__result-url" href="https://www.startpage.com/about">
            <h3 class="w-gl__result-title">About Startpage</h3>
          </a>
        </div>
        <div class="w-gl__result">
          <a class="w-gl__result-url" href="https://example.com">
            <h3 class="w-gl__result-title">External Site</h3>
          </a>
        </div>
      `;

      const results = engine.parseResponse(mockHtml, defaultParams);

      expect(results.results.length).toBe(1);
      expect(results.results[0].url).toBe('https://example.com');
    });

    it('should parse related searches', () => {
      const mockHtml = `
        <div class="w-gl__result">
          <a class="w-gl__result-url" href="https://example.com">
            <h3 class="w-gl__result-title">Result</h3>
          </a>
        </div>
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

    it('should return empty results for empty HTML', () => {
      const results = engine.parseResponse('', defaultParams);
      expect(results.results).toEqual([]);
    });

    it('should use fallback result class', () => {
      const mockHtml = `
        <div class="result">
          <a href="https://example.com">
            <h3>Fallback Result</h3>
          </a>
          <p class="desc">Description here.</p>
        </div>
      `;

      const results = engine.parseResponse(mockHtml, defaultParams);

      expect(results.results.length).toBe(1);
      expect(results.results[0].title).toBe('Fallback Result');
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
      // Startpage may have rate limiting
      // expect(results.results.length).toBeGreaterThan(0);

      if (results.results.length > 0) {
        const first = results.results[0];
        expect(first.url).toBeTruthy();
        expect(first.url).toContain('http');
        expect(first.title).toBeTruthy();
        expect(first.engine).toBe('startpage');
        expect(first.category).toBe('general');
      }
    }, 30000);
  });
});
