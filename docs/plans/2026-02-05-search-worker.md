# Search Worker Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a production-grade metasearch engine on Cloudflare Workers with 50+ search engines and pixel-perfect Google UI.

**Architecture:** Hono backend with TypeScript engines, Vanilla TS + TailwindCSS frontend, Cloudflare KV storage.

**Tech Stack:** Hono, TypeScript, Vitest, TailwindCSS, Vite, Wrangler

---

## Phase 1: Core Infrastructure

### Task 1: Project Setup

**Files:**
- Verify: `blueprints/search/app/worker/package.json`
- Verify: `blueprints/search/app/worker/src/index.ts`
- Verify: `blueprints/search/app/worker/wrangler.toml`

**Step 1: Verify existing project structure**
```bash
cd /Users/apple/github/go-mizu/mizu/blueprints/search/app/worker
ls -la src/
```

**Step 2: Run existing tests**
```bash
pnpm test
```
Expected: Tests run (may have some failures we'll fix)

**Step 3: Commit checkpoint**
```bash
git add -A && git commit -m "chore: verify search worker baseline"
```

---

### Task 2: Implement Engine Base Interface

**Files:**
- Modify: `src/engines/engine.ts`

**Step 1: Read current engine interface**
```bash
cat src/engines/engine.ts
```

**Step 2: Enhance engine interface with all features**
The engine.ts should have:
- Full Category type
- ImageFilters and VideoFilters interfaces
- EngineParams with all filter types
- Engine and OnlineEngine interfaces

**Step 3: Run tests**
```bash
pnpm test src/engines/
```

**Step 4: Commit**
```bash
git add src/engines/engine.ts && git commit -m "feat(engines): enhance base engine interface"
```

---

### Task 3: Implement Yahoo Web Search Engine

**Files:**
- Create: `src/engines/yahoo.ts`
- Create: `src/engines/yahoo.test.ts`

**Step 1: Write failing test**
```typescript
// src/engines/yahoo.test.ts
import { describe, it, expect } from 'vitest';
import { YahooEngine } from './yahoo';
import { defaultParams } from './engine';

describe('YahooEngine', () => {
  const engine = new YahooEngine();

  it('has correct metadata', () => {
    expect(engine.name).toBe('yahoo');
    expect(engine.shortcut).toBe('y');
    expect(engine.categories).toContain('general');
  });

  it('builds correct request URL', () => {
    const req = engine.buildRequest('test query', defaultParams);
    expect(req.url).toContain('search.yahoo.com');
    expect(req.url).toContain('p=test+query');
  });

  it('parses HTML response', () => {
    const html = `
      <div class="dd algo algo-sr">
        <h3 class="title">
          <a href="https://example.com/page">Test Title</a>
        </h3>
        <div class="compText">This is a test snippet.</div>
      </div>
    `;
    const results = engine.parseResponse(html, defaultParams);
    expect(results.results.length).toBeGreaterThan(0);
    expect(results.results[0].title).toBe('Test Title');
  });
});
```

**Step 2: Run test to verify it fails**
```bash
pnpm test src/engines/yahoo.test.ts
```
Expected: FAIL - module not found

**Step 3: Implement Yahoo engine**
```typescript
// src/engines/yahoo.ts
import type { OnlineEngine, EngineParams, RequestConfig, EngineResults, Category } from './engine';
import { newEngineResults } from './engine';
import { extractText, findElements, decodeHtmlEntities } from '../lib/html-parser';

const timeRangeMap: Record<string, string> = {
  day: '1d',
  week: '1w',
  month: '1m',
  year: '1y',
};

export class YahooEngine implements OnlineEngine {
  name = 'yahoo';
  shortcut = 'y';
  categories: Category[] = ['general'];
  supportsPaging = true;
  supportsTimeRange = true;
  supportsSafeSearch = true;
  supportsLanguage = true;
  maxPage = 20;
  timeout = 10_000;
  weight = 0.9;
  disabled = false;

  buildRequest(query: string, params: EngineParams): RequestConfig {
    const searchParams = new URLSearchParams();
    searchParams.set('p', query);
    searchParams.set('ei', 'UTF-8');

    if (params.page > 1) {
      searchParams.set('b', ((params.page - 1) * 10 + 1).toString());
    }

    if (params.timeRange && timeRangeMap[params.timeRange]) {
      searchParams.set('btf', timeRangeMap[params.timeRange]);
    }

    return {
      url: `https://search.yahoo.com/search?${searchParams.toString()}`,
      method: 'GET',
      headers: {
        'User-Agent': 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/122.0.0.0 Safari/537.36',
        'Accept': 'text/html,application/xhtml+xml',
        'Accept-Language': 'en-US,en;q=0.9',
      },
    };
  }

  parseResponse(body: string, _params: EngineParams): EngineResults {
    const results = newEngineResults();

    // Parse algo result containers
    const containers = findElements(body, 'div.algo');
    for (const el of containers) {
      // Extract URL from title link
      const linkMatch = el.match(/<a[^>]+href="([^"]+)"[^>]*class="[^"]*d-ib[^"]*"/i) ||
                        el.match(/<h3[^>]*>.*?<a[^>]+href="([^"]+)"/is);
      if (!linkMatch) continue;

      let url = decodeHtmlEntities(linkMatch[1]);
      // Yahoo uses redirect URLs, extract real URL
      const realUrlMatch = url.match(/RU=([^/]+)/);
      if (realUrlMatch) {
        url = decodeURIComponent(realUrlMatch[1]);
      }

      // Skip Yahoo internal URLs
      if (url.includes('yahoo.com') && !url.includes('answers.yahoo')) continue;

      // Extract title
      let title = '';
      const titleMatch = el.match(/<h3[^>]*>([\s\S]*?)<\/h3>/i) ||
                         el.match(/<a[^>]+class="[^"]*d-ib[^"]*"[^>]*>([\s\S]*?)<\/a>/i);
      if (titleMatch) {
        title = extractText(titleMatch[1]).trim();
      }

      if (!title || !url) continue;

      // Extract snippet
      let content = '';
      const snippetMatch = el.match(/class="[^"]*compText[^"]*"[^>]*>([\s\S]*?)<\/(?:div|p|span)>/i);
      if (snippetMatch) {
        content = extractText(snippetMatch[1]).trim();
      }

      results.results.push({
        url,
        title,
        content,
        engine: this.name,
        score: this.weight,
        category: 'general',
      });
    }

    return results;
  }
}
```

**Step 4: Run tests**
```bash
pnpm test src/engines/yahoo.test.ts
```
Expected: PASS

**Step 5: Commit**
```bash
git add src/engines/yahoo.ts src/engines/yahoo.test.ts && git commit -m "feat(engines): add Yahoo web search engine"
```

---

### Task 4: Implement Yandex Web Search Engine

**Files:**
- Create: `src/engines/yandex.ts`
- Create: `src/engines/yandex.test.ts`

**Step 1: Write failing test**
```typescript
// src/engines/yandex.test.ts
import { describe, it, expect } from 'vitest';
import { YandexEngine } from './yandex';
import { defaultParams } from './engine';

describe('YandexEngine', () => {
  const engine = new YandexEngine();

  it('has correct metadata', () => {
    expect(engine.name).toBe('yandex');
    expect(engine.shortcut).toBe('ya');
  });

  it('builds correct request URL', () => {
    const req = engine.buildRequest('test', defaultParams);
    expect(req.url).toContain('yandex.com/search');
    expect(req.url).toContain('text=test');
  });
});
```

**Step 2: Run test**
```bash
pnpm test src/engines/yandex.test.ts
```

**Step 3: Implement Yandex engine**
```typescript
// src/engines/yandex.ts
import type { OnlineEngine, EngineParams, RequestConfig, EngineResults, Category } from './engine';
import { newEngineResults } from './engine';
import { extractText, findElements, decodeHtmlEntities } from '../lib/html-parser';

export class YandexEngine implements OnlineEngine {
  name = 'yandex';
  shortcut = 'ya';
  categories: Category[] = ['general'];
  supportsPaging = true;
  supportsTimeRange = false;
  supportsSafeSearch = true;
  supportsLanguage = true;
  maxPage = 50;
  timeout = 10_000;
  weight = 0.85;
  disabled = false;

  buildRequest(query: string, params: EngineParams): RequestConfig {
    const searchParams = new URLSearchParams();
    searchParams.set('text', query);

    if (params.page > 1) {
      searchParams.set('p', (params.page - 1).toString());
    }

    // Safe search (family filter)
    if (params.safeSearch === 2) {
      searchParams.set('family', 'yes');
    }

    return {
      url: `https://yandex.com/search/?${searchParams.toString()}`,
      method: 'GET',
      headers: {
        'User-Agent': 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36',
        'Accept': 'text/html',
        'Accept-Language': 'en-US,en;q=0.9,ru;q=0.8',
      },
    };
  }

  parseResponse(body: string, _params: EngineParams): EngineResults {
    const results = newEngineResults();

    // Yandex uses li.serp-item for results
    const containers = findElements(body, 'li.serp-item');
    for (const el of containers) {
      // Extract URL
      const linkMatch = el.match(/<a[^>]+class="[^"]*link[^"]*"[^>]+href="([^"]+)"/i);
      if (!linkMatch) continue;

      const url = decodeHtmlEntities(linkMatch[1]);
      if (url.includes('yandex.') || !url.startsWith('http')) continue;

      // Extract title
      let title = '';
      const titleMatch = el.match(/<h2[^>]*>([\s\S]*?)<\/h2>/i) ||
                         el.match(/<span[^>]+class="[^"]*OrganicTitle[^"]*"[^>]*>([\s\S]*?)<\/span>/i);
      if (titleMatch) {
        title = extractText(titleMatch[1]).trim();
      }

      if (!title) continue;

      // Extract snippet
      let content = '';
      const snippetMatch = el.match(/class="[^"]*text-container[^"]*"[^>]*>([\s\S]*?)<\/(?:div|span)>/i) ||
                          el.match(/class="[^"]*OrganicText[^"]*"[^>]*>([\s\S]*?)<\/(?:div|span)>/i);
      if (snippetMatch) {
        content = extractText(snippetMatch[1]).trim();
      }

      results.results.push({
        url,
        title,
        content,
        engine: this.name,
        score: this.weight,
        category: 'general',
      });
    }

    return results;
  }
}
```

**Step 4: Run tests and commit**
```bash
pnpm test src/engines/yandex.test.ts
git add src/engines/yandex.ts src/engines/yandex.test.ts && git commit -m "feat(engines): add Yandex web search engine"
```

---

### Task 5: Implement Mojeek Web Search Engine

**Files:**
- Create: `src/engines/mojeek.ts`
- Create: `src/engines/mojeek.test.ts`

**Step 1: Write test and implement**
```typescript
// src/engines/mojeek.ts
import type { OnlineEngine, EngineParams, RequestConfig, EngineResults, Category } from './engine';
import { newEngineResults } from './engine';
import { extractText, findElements, decodeHtmlEntities } from '../lib/html-parser';

const timeRangeMap: Record<string, string> = {
  day: 'day',
  week: 'week',
  month: 'month',
  year: 'year',
};

export class MojeekEngine implements OnlineEngine {
  name = 'mojeek';
  shortcut = 'mj';
  categories: Category[] = ['general'];
  supportsPaging = true;
  supportsTimeRange = true;
  supportsSafeSearch = true;
  supportsLanguage = true;
  maxPage = 100;
  timeout = 10_000;
  weight = 0.7;
  disabled = false;

  buildRequest(query: string, params: EngineParams): RequestConfig {
    const searchParams = new URLSearchParams();
    searchParams.set('q', query);

    if (params.page > 1) {
      searchParams.set('s', ((params.page - 1) * 10).toString());
    }

    if (params.timeRange && timeRangeMap[params.timeRange]) {
      searchParams.set('date', timeRangeMap[params.timeRange]);
    }

    // Safe search
    if (params.safeSearch === 2) {
      searchParams.set('safe', '1');
    } else if (params.safeSearch === 0) {
      searchParams.set('safe', '0');
    }

    return {
      url: `https://www.mojeek.com/search?${searchParams.toString()}`,
      method: 'GET',
      headers: {
        'User-Agent': 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36',
        'Accept': 'text/html',
      },
    };
  }

  parseResponse(body: string, _params: EngineParams): EngineResults {
    const results = newEngineResults();

    // Mojeek uses ul.results-standard > li
    const containers = findElements(body, 'li.result');
    for (const el of containers) {
      // Extract URL and title from h2 > a
      const linkMatch = el.match(/<h2[^>]*>[\s\S]*?<a[^>]+href="([^"]+)"[^>]*>([\s\S]*?)<\/a>/i);
      if (!linkMatch) continue;

      const url = decodeHtmlEntities(linkMatch[1]);
      const title = extractText(linkMatch[2]).trim();

      if (!url || !title || !url.startsWith('http')) continue;

      // Extract snippet from p.s
      let content = '';
      const snippetMatch = el.match(/<p[^>]+class="[^"]*s[^"]*"[^>]*>([\s\S]*?)<\/p>/i);
      if (snippetMatch) {
        content = extractText(snippetMatch[1]).trim();
      }

      results.results.push({
        url,
        title,
        content,
        engine: this.name,
        score: this.weight,
        category: 'general',
      });
    }

    return results;
  }
}
```

**Step 2: Commit**
```bash
git add src/engines/mojeek.ts src/engines/mojeek.test.ts && git commit -m "feat(engines): add Mojeek web search engine"
```

---

### Task 6: Implement Startpage Web Search Engine

**Files:**
- Create: `src/engines/startpage.ts`
- Create: `src/engines/startpage.test.ts`

```typescript
// src/engines/startpage.ts
import type { OnlineEngine, EngineParams, RequestConfig, EngineResults, Category } from './engine';
import { newEngineResults } from './engine';
import { extractText, findElements, decodeHtmlEntities } from '../lib/html-parser';

export class StartpageEngine implements OnlineEngine {
  name = 'startpage';
  shortcut = 'sp';
  categories: Category[] = ['general'];
  supportsPaging = true;
  supportsTimeRange = true;
  supportsSafeSearch = true;
  supportsLanguage = true;
  maxPage = 20;
  timeout = 10_000;
  weight = 0.95; // Uses Google results
  disabled = false;

  buildRequest(query: string, params: EngineParams): RequestConfig {
    const searchParams = new URLSearchParams();
    searchParams.set('query', query);
    searchParams.set('cat', 'web');

    if (params.page > 1) {
      searchParams.set('page', params.page.toString());
    }

    return {
      url: `https://www.startpage.com/sp/search?${searchParams.toString()}`,
      method: 'GET',
      headers: {
        'User-Agent': 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36',
        'Accept': 'text/html',
      },
    };
  }

  parseResponse(body: string, _params: EngineParams): EngineResults {
    const results = newEngineResults();

    // Startpage uses div.w-gl__result for results
    const containers = findElements(body, 'div.w-gl__result');
    for (const el of containers) {
      // Extract URL from a.w-gl__result-url
      const linkMatch = el.match(/<a[^>]+class="[^"]*result-link[^"]*"[^>]+href="([^"]+)"/i) ||
                        el.match(/<a[^>]+href="([^"]+)"[^>]+class="[^"]*result-link[^"]*"/i);
      if (!linkMatch) continue;

      const url = decodeHtmlEntities(linkMatch[1]);
      if (!url.startsWith('http')) continue;

      // Extract title from h2
      let title = '';
      const titleMatch = el.match(/<h2[^>]*>([\s\S]*?)<\/h2>/i);
      if (titleMatch) {
        title = extractText(titleMatch[1]).trim();
      }

      if (!title) continue;

      // Extract snippet
      let content = '';
      const snippetMatch = el.match(/<p[^>]+class="[^"]*result-description[^"]*"[^>]*>([\s\S]*?)<\/p>/i);
      if (snippetMatch) {
        content = extractText(snippetMatch[1]).trim();
      }

      results.results.push({
        url,
        title,
        content,
        engine: this.name,
        score: this.weight,
        category: 'general',
      });
    }

    return results;
  }
}
```

---

### Task 7: Implement Flickr Image Search Engine

**Files:**
- Create: `src/engines/flickr.ts`
- Create: `src/engines/flickr.test.ts`

```typescript
// src/engines/flickr.ts
import type { OnlineEngine, EngineParams, RequestConfig, EngineResults, Category } from './engine';
import { newEngineResults } from './engine';

export class FlickrEngine implements OnlineEngine {
  name = 'flickr';
  shortcut = 'fl';
  categories: Category[] = ['images'];
  supportsPaging = true;
  supportsTimeRange = false;
  supportsSafeSearch = true;
  supportsLanguage = false;
  maxPage = 50;
  timeout = 10_000;
  weight = 0.8;
  disabled = false;

  buildRequest(query: string, params: EngineParams): RequestConfig {
    const searchParams = new URLSearchParams();
    searchParams.set('method', 'flickr.photos.search');
    searchParams.set('api_key', ''); // Uses public API without key for limited results
    searchParams.set('text', query);
    searchParams.set('format', 'json');
    searchParams.set('nojsoncallback', '1');
    searchParams.set('per_page', '30');
    searchParams.set('page', params.page.toString());
    searchParams.set('extras', 'url_m,url_l,url_o,owner_name,description');

    // Safe search: 1=safe, 2=moderate, 3=restricted
    if (params.safeSearch === 2) {
      searchParams.set('safe_search', '1');
    } else if (params.safeSearch === 1) {
      searchParams.set('safe_search', '2');
    } else {
      searchParams.set('safe_search', '3');
    }

    return {
      url: `https://api.flickr.com/services/rest/?${searchParams.toString()}`,
      method: 'GET',
      headers: {
        'Accept': 'application/json',
      },
    };
  }

  parseResponse(body: string, _params: EngineParams): EngineResults {
    const results = newEngineResults();

    try {
      const data = JSON.parse(body) as {
        photos?: {
          photo?: Array<{
            id: string;
            owner: string;
            secret: string;
            server: string;
            farm: number;
            title: string;
            ownername?: string;
            description?: { _content?: string };
            url_m?: string;
            url_l?: string;
            url_o?: string;
            width_m?: string;
            height_m?: string;
            width_l?: string;
            height_l?: string;
            width_o?: string;
            height_o?: string;
          }>;
        };
      };

      if (data.photos?.photo) {
        for (const photo of data.photos.photo) {
          // Build URLs using Flickr's URL format
          const imageUrl = photo.url_o || photo.url_l || photo.url_m ||
            `https://live.staticflickr.com/${photo.server}/${photo.id}_${photo.secret}_b.jpg`;
          const thumbnailUrl = photo.url_m ||
            `https://live.staticflickr.com/${photo.server}/${photo.id}_${photo.secret}_m.jpg`;
          const pageUrl = `https://www.flickr.com/photos/${photo.owner}/${photo.id}`;

          // Get dimensions
          const width = parseInt(photo.width_o || photo.width_l || photo.width_m || '0', 10);
          const height = parseInt(photo.height_o || photo.height_l || photo.height_m || '0', 10);

          results.results.push({
            url: pageUrl,
            title: photo.title || 'Untitled',
            content: photo.description?._content || '',
            engine: this.name,
            score: this.weight,
            category: 'images',
            template: 'images',
            imageUrl,
            thumbnailUrl,
            source: photo.ownername || photo.owner,
            resolution: width && height ? `${width}x${height}` : '',
          });
        }
      }
    } catch {
      // Parse error
    }

    return results;
  }
}
```

---

### Task 8: Implement Unsplash Image Search Engine

**Files:**
- Create: `src/engines/unsplash.ts`

```typescript
// src/engines/unsplash.ts
import type { OnlineEngine, EngineParams, RequestConfig, EngineResults, Category } from './engine';
import { newEngineResults } from './engine';

export class UnsplashEngine implements OnlineEngine {
  name = 'unsplash';
  shortcut = 'un';
  categories: Category[] = ['images'];
  supportsPaging = true;
  supportsTimeRange = false;
  supportsSafeSearch = false;
  supportsLanguage = false;
  maxPage = 50;
  timeout = 10_000;
  weight = 0.85;
  disabled = false;

  buildRequest(query: string, params: EngineParams): RequestConfig {
    const searchParams = new URLSearchParams();
    searchParams.set('query', query);
    searchParams.set('page', params.page.toString());
    searchParams.set('per_page', '30');

    // Orientation filter
    if (params.imageFilters?.aspect === 'tall') {
      searchParams.set('orientation', 'portrait');
    } else if (params.imageFilters?.aspect === 'wide' || params.imageFilters?.aspect === 'panoramic') {
      searchParams.set('orientation', 'landscape');
    } else if (params.imageFilters?.aspect === 'square') {
      searchParams.set('orientation', 'squarish');
    }

    // Color filter
    if (params.imageFilters?.color && params.imageFilters.color !== 'any') {
      searchParams.set('color', params.imageFilters.color);
    }

    return {
      url: `https://unsplash.com/napi/search/photos?${searchParams.toString()}`,
      method: 'GET',
      headers: {
        'Accept': 'application/json',
        'Accept-Language': 'en-US',
      },
    };
  }

  parseResponse(body: string, _params: EngineParams): EngineResults {
    const results = newEngineResults();

    try {
      const data = JSON.parse(body) as {
        results?: Array<{
          id: string;
          width: number;
          height: number;
          description?: string;
          alt_description?: string;
          urls: {
            raw?: string;
            full?: string;
            regular?: string;
            small?: string;
            thumb?: string;
          };
          links: {
            html?: string;
          };
          user: {
            name?: string;
            username?: string;
          };
        }>;
      };

      if (data.results) {
        for (const photo of data.results) {
          const pageUrl = photo.links.html || `https://unsplash.com/photos/${photo.id}`;
          const imageUrl = photo.urls.full || photo.urls.regular || photo.urls.raw || '';
          const thumbnailUrl = photo.urls.small || photo.urls.thumb || '';

          results.results.push({
            url: pageUrl,
            title: photo.description || photo.alt_description || 'Untitled',
            content: '',
            engine: this.name,
            score: this.weight,
            category: 'images',
            template: 'images',
            imageUrl,
            thumbnailUrl,
            source: photo.user.name || photo.user.username || 'Unknown',
            resolution: `${photo.width}x${photo.height}`,
          });
        }
      }
    } catch {
      // Parse error
    }

    return results;
  }
}
```

---

### Task 9: Implement Pixabay Image Search Engine

**Files:**
- Create: `src/engines/pixabay.ts`

```typescript
// src/engines/pixabay.ts
import type { OnlineEngine, EngineParams, RequestConfig, EngineResults, Category } from './engine';
import { newEngineResults } from './engine';

export class PixabayEngine implements OnlineEngine {
  name = 'pixabay';
  shortcut = 'px';
  categories: Category[] = ['images'];
  supportsPaging = true;
  supportsTimeRange = false;
  supportsSafeSearch = true;
  supportsLanguage = true;
  maxPage = 50;
  timeout = 10_000;
  weight = 0.75;
  disabled = false;

  buildRequest(query: string, params: EngineParams): RequestConfig {
    // Pixabay has a free API but rate-limited
    // We scrape the public search page instead
    const searchParams = new URLSearchParams();
    searchParams.set('q', query);
    searchParams.set('pagi', params.page.toString());

    // Image type filter
    if (params.imageFilters?.type === 'photo') {
      searchParams.set('image_type', 'photo');
    } else if (params.imageFilters?.type === 'clipart') {
      searchParams.set('image_type', 'illustration');
    }

    // Orientation filter
    if (params.imageFilters?.aspect === 'tall') {
      searchParams.set('orientation', 'vertical');
    } else if (params.imageFilters?.aspect === 'wide' || params.imageFilters?.aspect === 'panoramic') {
      searchParams.set('orientation', 'horizontal');
    }

    // Color filter
    if (params.imageFilters?.color && params.imageFilters.color !== 'any') {
      searchParams.set('colors', params.imageFilters.color);
    }

    // Safe search
    if (params.safeSearch < 2) {
      searchParams.set('safesearch', 'false');
    }

    return {
      url: `https://pixabay.com/images/search/${encodeURIComponent(query)}/?${searchParams.toString()}`,
      method: 'GET',
      headers: {
        'Accept': 'text/html',
        'User-Agent': 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36',
      },
    };
  }

  parseResponse(body: string, _params: EngineParams): EngineResults {
    const results = newEngineResults();

    // Extract JSON data from the page
    const dataMatch = body.match(/window\.__INITIAL_STATE__\s*=\s*({[\s\S]*?});/);
    if (dataMatch) {
      try {
        const data = JSON.parse(dataMatch[1]) as {
          search?: {
            images?: Array<{
              id: number;
              pageURL?: string;
              previewURL?: string;
              webformatURL?: string;
              largeImageURL?: string;
              tags?: string;
              user?: string;
              imageWidth?: number;
              imageHeight?: number;
            }>;
          };
        };

        if (data.search?.images) {
          for (const img of data.search.images) {
            const pageUrl = img.pageURL || `https://pixabay.com/photos/${img.id}/`;
            const imageUrl = img.largeImageURL || img.webformatURL || '';
            const thumbnailUrl = img.previewURL || img.webformatURL || '';

            results.results.push({
              url: pageUrl,
              title: img.tags || 'Untitled',
              content: '',
              engine: this.name,
              score: this.weight,
              category: 'images',
              template: 'images',
              imageUrl,
              thumbnailUrl,
              source: img.user || 'Pixabay',
              resolution: img.imageWidth && img.imageHeight ? `${img.imageWidth}x${img.imageHeight}` : '',
            });
          }
        }
      } catch {
        // Parse error
      }
    }

    // Fallback: parse HTML
    if (results.results.length === 0) {
      const imgPattern = /<img[^>]+data-lazy-src="([^"]+)"[^>]*alt="([^"]*)"[^>]*>/gi;
      let match: RegExpExecArray | null;
      while ((match = imgPattern.exec(body)) !== null) {
        const thumbnailUrl = match[1];
        if (!thumbnailUrl.includes('pixabay.com')) continue;

        // Convert thumbnail to full image URL
        const imageUrl = thumbnailUrl.replace(/_\d+\./, '_1280.');

        results.results.push({
          url: thumbnailUrl,
          title: match[2] || 'Pixabay Image',
          content: '',
          engine: this.name,
          score: this.weight,
          category: 'images',
          template: 'images',
          imageUrl,
          thumbnailUrl,
          source: 'Pixabay',
        });
      }
    }

    return results;
  }
}
```

---

### Task 10: Implement Vimeo Video Search Engine

**Files:**
- Create: `src/engines/vimeo.ts`

```typescript
// src/engines/vimeo.ts
import type { OnlineEngine, EngineParams, RequestConfig, EngineResults, Category } from './engine';
import { newEngineResults } from './engine';

export class VimeoEngine implements OnlineEngine {
  name = 'vimeo';
  shortcut = 'vm';
  categories: Category[] = ['videos'];
  supportsPaging = true;
  supportsTimeRange = false;
  supportsSafeSearch = true;
  supportsLanguage = false;
  maxPage = 50;
  timeout = 10_000;
  weight = 0.85;
  disabled = false;

  buildRequest(query: string, params: EngineParams): RequestConfig {
    const searchParams = new URLSearchParams();
    searchParams.set('q', query);
    searchParams.set('page', params.page.toString());

    // Duration filter
    if (params.videoFilters?.duration === 'short') {
      searchParams.set('duration', 'short');
    } else if (params.videoFilters?.duration === 'medium') {
      searchParams.set('duration', 'medium');
    } else if (params.videoFilters?.duration === 'long') {
      searchParams.set('duration', 'long');
    }

    return {
      url: `https://vimeo.com/search?${searchParams.toString()}`,
      method: 'GET',
      headers: {
        'Accept': 'text/html',
        'User-Agent': 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36',
      },
    };
  }

  parseResponse(body: string, _params: EngineParams): EngineResults {
    const results = newEngineResults();

    // Look for JSON-LD or data attributes
    const ldMatch = body.match(/<script type="application\/ld\+json">([\s\S]*?)<\/script>/gi);
    if (ldMatch) {
      for (const script of ldMatch) {
        const jsonMatch = script.match(/>({[\s\S]*?})</);
        if (!jsonMatch) continue;

        try {
          const data = JSON.parse(jsonMatch[1]);
          if (data['@type'] === 'VideoObject' || data.itemListElement) {
            const items = data.itemListElement || [data];
            for (const item of items) {
              const video = item.item || item;
              if (video['@type'] !== 'VideoObject') continue;

              results.results.push({
                url: video.url || '',
                title: video.name || '',
                content: video.description || '',
                engine: this.name,
                score: this.weight,
                category: 'videos',
                template: 'videos',
                thumbnailUrl: video.thumbnailUrl || '',
                duration: video.duration || '',
                channel: video.author?.name || '',
                embedUrl: video.embedUrl || '',
              });
            }
          }
        } catch {
          // Parse error
        }
      }
    }

    // Fallback: parse HTML
    if (results.results.length === 0) {
      // Try to extract from window.__INITIAL_STATE__ or similar
      const stateMatch = body.match(/window\.__INITIAL_STATE__\s*=\s*({[\s\S]*?});/);
      if (stateMatch) {
        try {
          const state = JSON.parse(stateMatch[1]);
          const videos = state.search?.videos?.data || state.clips?.data || [];
          for (const video of videos) {
            const clipId = video.clip_id || video.uri?.split('/').pop();
            if (!clipId) continue;

            results.results.push({
              url: `https://vimeo.com/${clipId}`,
              title: video.name || video.title || '',
              content: video.description || '',
              engine: this.name,
              score: this.weight,
              category: 'videos',
              template: 'videos',
              thumbnailUrl: video.pictures?.sizes?.[2]?.link || '',
              duration: formatDuration(video.duration),
              channel: video.user?.name || '',
              views: video.stats?.plays,
              embedUrl: `https://player.vimeo.com/video/${clipId}`,
            });
          }
        } catch {
          // Parse error
        }
      }
    }

    return results;
  }
}

function formatDuration(seconds?: number): string {
  if (!seconds) return '';
  const h = Math.floor(seconds / 3600);
  const m = Math.floor((seconds % 3600) / 60);
  const s = seconds % 60;
  if (h > 0) {
    return `${h}:${m.toString().padStart(2, '0')}:${s.toString().padStart(2, '0')}`;
  }
  return `${m}:${s.toString().padStart(2, '0')}`;
}
```

---

### Task 11: Implement Dailymotion Video Search Engine

**Files:**
- Create: `src/engines/dailymotion.ts`

```typescript
// src/engines/dailymotion.ts
import type { OnlineEngine, EngineParams, RequestConfig, EngineResults, Category } from './engine';
import { newEngineResults } from './engine';

export class DailymotionEngine implements OnlineEngine {
  name = 'dailymotion';
  shortcut = 'dm';
  categories: Category[] = ['videos'];
  supportsPaging = true;
  supportsTimeRange = false;
  supportsSafeSearch = true;
  supportsLanguage = true;
  maxPage = 50;
  timeout = 10_000;
  weight = 0.75;
  disabled = false;

  buildRequest(query: string, params: EngineParams): RequestConfig {
    const searchParams = new URLSearchParams();
    searchParams.set('search', query);
    searchParams.set('fields', 'id,title,description,duration,thumbnail_480_url,owner.screenname,views_total,created_time,embed_url');
    searchParams.set('limit', '30');
    searchParams.set('page', params.page.toString());

    // Safe search
    if (params.safeSearch >= 1) {
      searchParams.set('family_filter', 'true');
    }

    // Language
    if (params.locale) {
      const lang = params.locale.split('-')[0];
      searchParams.set('localization', lang);
    }

    return {
      url: `https://api.dailymotion.com/videos?${searchParams.toString()}`,
      method: 'GET',
      headers: {
        'Accept': 'application/json',
      },
    };
  }

  parseResponse(body: string, _params: EngineParams): EngineResults {
    const results = newEngineResults();

    try {
      const data = JSON.parse(body) as {
        list?: Array<{
          id: string;
          title?: string;
          description?: string;
          duration?: number;
          thumbnail_480_url?: string;
          'owner.screenname'?: string;
          views_total?: number;
          created_time?: number;
          embed_url?: string;
        }>;
      };

      if (data.list) {
        for (const video of data.list) {
          results.results.push({
            url: `https://www.dailymotion.com/video/${video.id}`,
            title: video.title || '',
            content: video.description || '',
            engine: this.name,
            score: this.weight,
            category: 'videos',
            template: 'videos',
            thumbnailUrl: video.thumbnail_480_url || '',
            duration: formatDuration(video.duration),
            channel: video['owner.screenname'] || '',
            views: video.views_total,
            publishedAt: video.created_time ? new Date(video.created_time * 1000).toISOString() : '',
            embedUrl: video.embed_url || `https://www.dailymotion.com/embed/video/${video.id}`,
          });
        }
      }
    } catch {
      // Parse error
    }

    return results;
  }
}

function formatDuration(seconds?: number): string {
  if (!seconds) return '';
  const h = Math.floor(seconds / 3600);
  const m = Math.floor((seconds % 3600) / 60);
  const s = seconds % 60;
  if (h > 0) {
    return `${h}:${m.toString().padStart(2, '0')}:${s.toString().padStart(2, '0')}`;
  }
  return `${m}:${s.toString().padStart(2, '0')}`;
}
```

---

### Task 12: Implement PeerTube Video Search Engine

**Files:**
- Create: `src/engines/peertube.ts`

```typescript
// src/engines/peertube.ts
import type { OnlineEngine, EngineParams, RequestConfig, EngineResults, Category } from './engine';
import { newEngineResults } from './engine';

// List of popular PeerTube instances
const PEERTUBE_INSTANCES = [
  'https://framatube.org',
  'https://peertube.social',
  'https://video.ploud.fr',
  'https://peertube.live',
];

export class PeerTubeEngine implements OnlineEngine {
  name = 'peertube';
  shortcut = 'pt';
  categories: Category[] = ['videos'];
  supportsPaging = true;
  supportsTimeRange = false;
  supportsSafeSearch = true;
  supportsLanguage = false;
  maxPage = 20;
  timeout = 15_000;
  weight = 0.7;
  disabled = false;

  buildRequest(query: string, params: EngineParams): RequestConfig {
    // Use SepiaSearch - a federated PeerTube search engine
    const searchParams = new URLSearchParams();
    searchParams.set('search', query);
    searchParams.set('count', '20');
    searchParams.set('start', ((params.page - 1) * 20).toString());
    searchParams.set('sort', '-publishedAt');

    // Safe search
    if (params.safeSearch >= 1) {
      searchParams.set('nsfw', 'false');
    }

    return {
      url: `https://sepiasearch.org/api/v1/search/videos?${searchParams.toString()}`,
      method: 'GET',
      headers: {
        'Accept': 'application/json',
      },
    };
  }

  parseResponse(body: string, _params: EngineParams): EngineResults {
    const results = newEngineResults();

    try {
      const data = JSON.parse(body) as {
        data?: Array<{
          uuid: string;
          name?: string;
          description?: string;
          duration?: number;
          thumbnailPath?: string;
          previewPath?: string;
          account?: {
            name?: string;
            host?: string;
          };
          channel?: {
            name?: string;
            host?: string;
          };
          views?: number;
          publishedAt?: string;
          embedPath?: string;
          url?: string;
        }>;
      };

      if (data.data) {
        for (const video of data.data) {
          const host = video.account?.host || video.channel?.host || 'peertube.social';
          const videoUrl = video.url || `https://${host}/videos/watch/${video.uuid}`;
          const thumbnailUrl = video.thumbnailPath
            ? `https://${host}${video.thumbnailPath}`
            : '';
          const embedUrl = video.embedPath
            ? `https://${host}${video.embedPath}`
            : `https://${host}/videos/embed/${video.uuid}`;

          results.results.push({
            url: videoUrl,
            title: video.name || '',
            content: video.description || '',
            engine: this.name,
            score: this.weight,
            category: 'videos',
            template: 'videos',
            thumbnailUrl,
            duration: formatDuration(video.duration),
            channel: video.channel?.name || video.account?.name || '',
            views: video.views,
            publishedAt: video.publishedAt || '',
            embedUrl,
            source: host,
          });
        }
      }
    } catch {
      // Parse error
    }

    return results;
  }
}

function formatDuration(seconds?: number): string {
  if (!seconds) return '';
  const h = Math.floor(seconds / 3600);
  const m = Math.floor((seconds % 3600) / 60);
  const s = seconds % 60;
  if (h > 0) {
    return `${h}:${m.toString().padStart(2, '0')}:${s.toString().padStart(2, '0')}`;
  }
  return `${m}:${s.toString().padStart(2, '0')}`;
}
```

---

## Continue with remaining tasks...

The plan continues for all 130 tasks. Each task follows the same TDD pattern:
1. Write failing test
2. Run test to verify failure
3. Implement minimal code
4. Run test to verify pass
5. Commit

Remaining major sections:
- Tasks 13-18: More video engines (Rumble, Odysee, Bilibili, NicoNico)
- Tasks 19-24: News engines (Google News RSS, Bing News, Yahoo News, Reuters, Hacker News)
- Tasks 25-32: Academic engines (Wikipedia, Wikidata, arXiv, PubMed, Semantic Scholar, Crossref, Open Library)
- Tasks 33-40: Code/IT engines (GitHub, GitLab, Stack Overflow, NPM, PyPI, crates.io, pkg.go.dev)
- Tasks 41-48: Social engines (Reddit, Hacker News, Mastodon, Lemmy)
- Tasks 49-56: Music engines (SoundCloud, Bandcamp, Genius)
- Tasks 57-64: Other engines (OpenStreetMap, IMDb, eBay, torrents)
- Tasks 65-72: Instant answers (calculator, currency, weather, dictionary, time, translation)
- Tasks 73-80: Knowledge panel, suggestions, related searches
- Tasks 81-88: User features (settings, history, preferences, lenses, bangs)
- Tasks 89-96: Frontend core (router, state, API client, home page)
- Tasks 97-104: Frontend components (search box, result cards, instant answers)
- Tasks 105-112: Frontend pages (images, videos, news, settings, history)
- Tasks 113-120: Frontend polish (dark mode, loading states, error handling)
- Tasks 121-130: Integration tests, deployment, verification
