import { describe, it, expect } from 'vitest';
import { YandexEngine } from './yandex';

describe('YandexEngine', () => {
  const engine = new YandexEngine();

  describe('metadata', () => {
    it('should have correct name and shortcut', () => {
      expect(engine.name).toBe('yandex');
      expect(engine.shortcut).toBe('ya');
    });

    it('should have correct categories', () => {
      expect(engine.categories).toContain('general');
      expect(engine.categories.length).toBe(1);
    });

    it('should have correct paging settings', () => {
      expect(engine.supportsPaging).toBe(true);
      expect(engine.maxPage).toBe(50);
    });

    it('should have correct timeout and weight', () => {
      expect(engine.timeout).toBe(10000);
      expect(engine.weight).toBe(0.85);
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
      expect(config.url).toContain('https://yandex.com/search/');
      expect(config.url).toContain('text=test+query');
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
      expect(page1.url).not.toContain('p=');

      const page2 = engine.buildRequest('test', {
        page: 2,
        locale: 'en-US',
        safeSearch: 1,
        timeRange: '',
        engineData: {},
      });
      expect(page2.url).toContain('p=1');

      const page5 = engine.buildRequest('test', {
        page: 5,
        locale: 'en-US',
        safeSearch: 1,
        timeRange: '',
        engineData: {},
      });
      expect(page5.url).toContain('p=4');
    });

    describe('time range filter', () => {
      it('should add time filter for day', () => {
        const config = engine.buildRequest('test', {
          page: 1,
          locale: 'en-US',
          safeSearch: 1,
          timeRange: 'day',
          engineData: {},
        });
        expect(config.url).toContain('within=77');
      });

      it('should add time filter for week', () => {
        const config = engine.buildRequest('test', {
          page: 1,
          locale: 'en-US',
          safeSearch: 1,
          timeRange: 'week',
          engineData: {},
        });
        expect(config.url).toContain('within=1');
      });

      it('should add time filter for month', () => {
        const config = engine.buildRequest('test', {
          page: 1,
          locale: 'en-US',
          safeSearch: 1,
          timeRange: 'month',
          engineData: {},
        });
        expect(config.url).toContain('within=2');
      });

      it('should add time filter for year', () => {
        const config = engine.buildRequest('test', {
          page: 1,
          locale: 'en-US',
          safeSearch: 1,
          timeRange: 'year',
          engineData: {},
        });
        expect(config.url).toContain('within=3');
      });

      it('should not add time filter when empty', () => {
        const config = engine.buildRequest('test', {
          page: 1,
          locale: 'en-US',
          safeSearch: 1,
          timeRange: '',
          engineData: {},
        });
        expect(config.url).not.toContain('within=');
      });
    });

    describe('safe search', () => {
      it('should set family filter off when safeSearch is 0', () => {
        const config = engine.buildRequest('test', {
          page: 1,
          locale: 'en-US',
          safeSearch: 0,
          timeRange: '',
          engineData: {},
        });
        expect(config.url).toContain('family=0');
      });

      it('should set family filter moderate when safeSearch is 1', () => {
        const config = engine.buildRequest('test', {
          page: 1,
          locale: 'en-US',
          safeSearch: 1,
          timeRange: '',
          engineData: {},
        });
        expect(config.url).toContain('family=1');
      });

      it('should set family filter strict when safeSearch is 2', () => {
        const config = engine.buildRequest('test', {
          page: 1,
          locale: 'en-US',
          safeSearch: 2,
          timeRange: '',
          engineData: {},
        });
        expect(config.url).toContain('family=2');
      });
    });

    describe('locale/region', () => {
      it('should set region code for US locale', () => {
        const config = engine.buildRequest('test', {
          page: 1,
          locale: 'en-US',
          safeSearch: 1,
          timeRange: '',
          engineData: {},
        });
        expect(config.url).toContain('lr=84');
      });

      it('should set region code for UK locale', () => {
        const config = engine.buildRequest('test', {
          page: 1,
          locale: 'en-GB',
          safeSearch: 1,
          timeRange: '',
          engineData: {},
        });
        expect(config.url).toContain('lr=102');
      });

      it('should set region code for Russian locale', () => {
        const config = engine.buildRequest('test', {
          page: 1,
          locale: 'ru-RU',
          safeSearch: 1,
          timeRange: '',
          engineData: {},
        });
        expect(config.url).toContain('lr=225');
        expect(config.url).not.toContain('lang=');
      });

      it('should set language for non-Russian locales', () => {
        const config = engine.buildRequest('test', {
          page: 1,
          locale: 'de-DE',
          safeSearch: 1,
          timeRange: '',
          engineData: {},
        });
        expect(config.url).toContain('lang=de');
      });
    });

    it('should request 10 results per page', () => {
      const config = engine.buildRequest('test', {
        page: 1,
        locale: 'en-US',
        safeSearch: 1,
        timeRange: '',
        engineData: {},
      });
      expect(config.url).toContain('numdoc=10');
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

    it('should parse Yandex serp-item results', () => {
      const mockHtml = `
        <li class="serp-item" data-cid="0">
          <h2 class="OrganicTitle">
            <a class="Link" href="https://example.com/page">
              Example Page Title
            </a>
          </h2>
          <div class="OrganicText">This is the description of the result.</div>
        </li>
        <li class="serp-item" data-cid="1">
          <h2 class="OrganicTitle">
            <a class="Link" href="https://another.com">
              Another Result Title
            </a>
          </h2>
          <span class="extended-text">Another description here.</span>
        </li>
      `;

      const results = engine.parseResponse(mockHtml, defaultParams);

      expect(results.results.length).toBe(2);

      const first = results.results[0];
      expect(first.url).toBe('https://example.com/page');
      expect(first.title).toBe('Example Page Title');
      expect(first.content).toContain('description');
      expect(first.engine).toBe('yandex');
      expect(first.category).toBe('general');

      const second = results.results[1];
      expect(second.url).toBe('https://another.com');
      expect(second.title).toBe('Another Result Title');
    });

    it('should parse organic class results as fallback', () => {
      const mockHtml = `
        <div class="organic">
          <h2>
            <a href="https://example.com">Test Title</a>
          </h2>
          <div class="TextContainer">Some description text.</div>
        </div>
      `;

      const results = engine.parseResponse(mockHtml, defaultParams);

      expect(results.results.length).toBe(1);
      expect(results.results[0].url).toBe('https://example.com');
      expect(results.results[0].title).toBe('Test Title');
    });

    it('should return empty results for CAPTCHA page', () => {
      const captchaHtml = `
        <html>
          <body>
            <div class="showcaptcha">Please complete the captcha</div>
          </body>
        </html>
      `;

      const results = engine.parseResponse(captchaHtml, defaultParams);
      expect(results.results).toEqual([]);
    });

    it('should skip Yandex internal links', () => {
      const mockHtml = `
        <li class="serp-item">
          <h2 class="OrganicTitle">
            <a href="https://yandex.ru/turbo/example.com">Yandex Turbo</a>
          </h2>
        </li>
        <li class="serp-item">
          <h2 class="OrganicTitle">
            <a href="https://yabs.yandex.ru/count/xxx">Ad Link</a>
          </h2>
        </li>
        <li class="serp-item">
          <h2 class="OrganicTitle">
            <a href="https://example.com">External Site</a>
          </h2>
        </li>
      `;

      const results = engine.parseResponse(mockHtml, defaultParams);

      expect(results.results.length).toBe(1);
      expect(results.results[0].url).toBe('https://example.com');
    });

    it('should parse related searches/suggestions', () => {
      const mockHtml = `
        <li class="serp-item">
          <h2 class="OrganicTitle">
            <a href="https://example.com">Result</a>
          </h2>
        </li>
        <div class="misspell">Did you mean: alternative query</div>
        <li class="suggest2-item">Suggested Search 1</li>
        <li class="suggest2-item">Suggested Search 2</li>
      `;

      const results = engine.parseResponse(mockHtml, defaultParams);

      expect(results.suggestions.length).toBeGreaterThanOrEqual(1);
    });

    it('should handle Path element for URL extraction', () => {
      const mockHtml = `
        <li class="serp-item">
          <div class="Path">
            <a href="https://example.com/deep/path">example.com</a>
          </div>
          <h2>Page Title From H2</h2>
        </li>
      `;

      const results = engine.parseResponse(mockHtml, defaultParams);

      expect(results.results.length).toBe(1);
      expect(results.results[0].url).toBe('https://example.com/deep/path');
    });

    it('should return empty results for empty HTML', () => {
      const results = engine.parseResponse('', defaultParams);
      expect(results.results).toEqual([]);
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

      // Yandex may trigger CAPTCHA, so we allow empty results
      expect(results).toBeDefined();
      expect(results.results).toBeDefined();
      // Note: Live test may return 0 results due to CAPTCHA
      // expect(results.results.length).toBeGreaterThan(0);
    }, 30000);
  });
});
