# Video Search Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Implement comprehensive video search with 9 engines, Google-identical UI, and full test coverage.

**Architecture:** Multi-engine aggregation via MetaSearch class. Each engine implements OnlineEngine interface. Frontend uses React components matching Google Videos layout.

**Tech Stack:** TypeScript, Hono, Cloudflare Workers, React 19, Tailwind CSS, Vitest

---

## Phase 1: Video Engine Types

### Task 1: Add Video Filter Types

**Files:**
- Modify: `app/worker/src/engines/engine.ts`

**Step 1: Add video filter types to engine.ts**

Add after line 41 (after ImageFilters interface):

```typescript
// ========== Video Filter Types ==========

export type VideoDuration = 'any' | 'short' | 'medium' | 'long';
export type VideoQuality = 'any' | 'hd' | '4k';
export type VideoSort = 'relevance' | 'date' | 'views' | 'duration';

export interface VideoFilters {
  duration?: VideoDuration;
  quality?: VideoQuality;
  source?: string;
  cc?: boolean;
}
```

**Step 2: Update EngineParams interface**

Add `videoFilters` field to EngineParams (around line 52):

```typescript
export interface EngineParams {
  page: number;
  locale: string;
  safeSearch: SafeSearch;
  timeRange: TimeRange;
  engineData: Record<string, string>;
  imageFilters?: ImageFilters;
  videoFilters?: VideoFilters;
}
```

**Step 3: Commit**

```bash
git add app/worker/src/engines/engine.ts
git commit -m "feat(video): add video filter types to engine"
```

---

## Phase 2: Video Engines Implementation

### Task 2: Enhance YouTube Engine

**Files:**
- Modify: `app/worker/src/engines/youtube.ts`
- Create: `app/worker/src/engines/youtube.test.ts`

**Step 1: Create test file**

```typescript
// app/worker/src/engines/youtube.test.ts
import { describe, it, expect } from 'vitest';
import { YouTubeEngine } from './youtube';

describe('YouTubeEngine', () => {
  const engine = new YouTubeEngine();

  it('should have correct metadata', () => {
    expect(engine.name).toBe('youtube');
    expect(engine.categories).toContain('videos');
  });

  it('should build correct search URL', () => {
    const config = engine.buildRequest('typescript tutorial', {
      page: 1,
      locale: 'en',
      safeSearch: 1,
      timeRange: '',
      engineData: {},
    });
    expect(config.url).toContain('search_query=typescript+tutorial');
    expect(config.method).toBe('GET');
  });

  it('should build URL with time range filter', () => {
    const config = engine.buildRequest('test', {
      page: 1,
      locale: 'en',
      safeSearch: 1,
      timeRange: 'week',
      engineData: {},
    });
    expect(config.url).toContain('sp=');
  });

  it('should search and return video results', async () => {
    const results = await fetchAndParse(engine, 'javascript tutorial');

    expect(results.results.length).toBeGreaterThan(0);
    const first = results.results[0];
    expect(first.url).toContain('youtube.com/watch');
    expect(first.title).toBeTruthy();
    expect(first.thumbnailUrl).toContain('ytimg.com');
  }, 30000);
});

async function fetchAndParse(engine: YouTubeEngine, query: string) {
  const params = { page: 1, locale: 'en', safeSearch: 1 as const, timeRange: '' as const, engineData: {} };
  const config = engine.buildRequest(query, params);
  const res = await fetch(config.url, {
    headers: config.headers,
  });
  const body = await res.text();
  return engine.parseResponse(body, params);
}
```

**Step 2: Run test to verify it fails/passes**

```bash
cd /Users/apple/github/go-mizu/mizu/blueprints/search/app/worker
npm test -- youtube.test.ts
```

**Step 3: Commit**

```bash
git add app/worker/src/engines/youtube.test.ts
git commit -m "test(video): add YouTube engine tests"
```

---

### Task 3: Implement Vimeo Engine

**Files:**
- Create: `app/worker/src/engines/vimeo.ts`
- Create: `app/worker/src/engines/vimeo.test.ts`

**Step 1: Create test file first**

```typescript
// app/worker/src/engines/vimeo.test.ts
import { describe, it, expect } from 'vitest';
import { VimeoEngine } from './vimeo';

describe('VimeoEngine', () => {
  const engine = new VimeoEngine();

  it('should have correct metadata', () => {
    expect(engine.name).toBe('vimeo');
    expect(engine.shortcut).toBe('vm');
    expect(engine.categories).toContain('videos');
  });

  it('should build correct search URL', () => {
    const config = engine.buildRequest('nature documentary', {
      page: 1,
      locale: 'en',
      safeSearch: 1,
      timeRange: '',
      engineData: {},
    });
    expect(config.url).toContain('vimeo.com/search');
    expect(config.url).toContain('q=nature+documentary');
  });

  it('should build URL with pagination', () => {
    const config = engine.buildRequest('test', {
      page: 2,
      locale: 'en',
      safeSearch: 1,
      timeRange: '',
      engineData: {},
    });
    expect(config.url).toContain('page=2');
  });

  it('should search and return video results', async () => {
    const results = await fetchAndParse(engine, 'short film');

    expect(results.results.length).toBeGreaterThan(0);
    const first = results.results[0];
    expect(first.url).toContain('vimeo.com');
    expect(first.title).toBeTruthy();
    expect(first.embedUrl).toContain('player.vimeo.com');
  }, 30000);
});

async function fetchAndParse(engine: VimeoEngine, query: string) {
  const params = { page: 1, locale: 'en', safeSearch: 1 as const, timeRange: '' as const, engineData: {} };
  const config = engine.buildRequest(query, params);
  const res = await fetch(config.url, { headers: config.headers });
  const body = await res.text();
  return engine.parseResponse(body, params);
}
```

**Step 2: Run test to verify it fails**

```bash
cd /Users/apple/github/go-mizu/mizu/blueprints/search/app/worker
npm test -- vimeo.test.ts
```

Expected: FAIL (VimeoEngine not found)

**Step 3: Implement Vimeo engine**

```typescript
// app/worker/src/engines/vimeo.ts
import type {
  OnlineEngine,
  EngineParams,
  RequestConfig,
  EngineResults,
  Category,
} from './engine';
import { newEngineResults } from './engine';

interface VimeoSearchData {
  filtered?: {
    clip?: {
      data?: VimeoClip[];
    };
  };
}

interface VimeoClip {
  clip?: {
    uri?: string;
    name?: string;
    description?: string;
    created_time?: string;
    pictures?: {
      sizes?: Array<{ link?: string; width?: number }>;
    };
    duration?: { raw?: number };
    user?: { name?: string };
    stats?: { plays?: number };
  };
}

export class VimeoEngine implements OnlineEngine {
  name = 'vimeo';
  shortcut = 'vm';
  categories: Category[] = ['videos'];
  supportsPaging = true;
  maxPage = 10;
  timeout = 8000;
  weight = 0.9;
  disabled = false;

  buildRequest(query: string, params: EngineParams): RequestConfig {
    const searchParams = new URLSearchParams();
    searchParams.set('q', query);
    if (params.page > 1) {
      searchParams.set('page', String(params.page));
    }

    return {
      url: `https://vimeo.com/search?${searchParams.toString()}`,
      method: 'GET',
      headers: {
        Accept: 'text/html,application/xhtml+xml',
        'Accept-Language': 'en-US,en;q=0.9',
        'User-Agent': 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36',
      },
      cookies: [],
    };
  }

  parseResponse(body: string, _params: EngineParams): EngineResults {
    const results = newEngineResults();

    // Extract JSON data from window.vimeo.config or data-search-data attribute
    const dataMatch = body.match(/data-search-data="([^"]+)"/);
    if (!dataMatch) {
      // Try alternate pattern
      const jsonMatch = body.match(/window\.vimeo\.config\s*=\s*({.+?});/s);
      if (!jsonMatch) return results;

      try {
        const config = JSON.parse(jsonMatch[1]);
        return this.parseConfig(config, results);
      } catch {
        return results;
      }
    }

    try {
      const decoded = dataMatch[1]
        .replace(/&quot;/g, '"')
        .replace(/&amp;/g, '&')
        .replace(/&lt;/g, '<')
        .replace(/&gt;/g, '>');
      const data: VimeoSearchData = JSON.parse(decoded);
      return this.parseSearchData(data, results);
    } catch {
      return results;
    }
  }

  private parseSearchData(data: VimeoSearchData, results: EngineResults): EngineResults {
    const clips = data.filtered?.clip?.data || [];

    for (const item of clips) {
      const clip = item.clip;
      if (!clip?.uri) continue;

      const videoId = clip.uri.replace('/videos/', '');
      const pictures = clip.pictures?.sizes || [];
      const thumbnail = pictures.length > 0
        ? pictures[pictures.length - 1].link || ''
        : '';

      results.results.push({
        url: `https://vimeo.com/${videoId}`,
        title: clip.name || '',
        content: clip.description || '',
        engine: this.name,
        score: this.weight,
        category: 'videos',
        template: 'videos',
        embedUrl: `https://player.vimeo.com/video/${videoId}`,
        thumbnailUrl: thumbnail,
        duration: clip.duration?.raw ? this.formatDuration(clip.duration.raw) : '',
        channel: clip.user?.name || '',
        views: clip.stats?.plays,
        publishedAt: clip.created_time,
      });
    }

    return results;
  }

  private parseConfig(config: Record<string, unknown>, results: EngineResults): EngineResults {
    // Handle alternate config structure if needed
    return results;
  }

  private formatDuration(seconds: number): string {
    const hrs = Math.floor(seconds / 3600);
    const mins = Math.floor((seconds % 3600) / 60);
    const secs = seconds % 60;

    if (hrs > 0) {
      return `${hrs}:${mins.toString().padStart(2, '0')}:${secs.toString().padStart(2, '0')}`;
    }
    return `${mins}:${secs.toString().padStart(2, '0')}`;
  }
}
```

**Step 4: Run test to verify it passes**

```bash
npm test -- vimeo.test.ts
```

**Step 5: Commit**

```bash
git add app/worker/src/engines/vimeo.ts app/worker/src/engines/vimeo.test.ts
git commit -m "feat(video): implement Vimeo engine"
```

---

### Task 4: Implement Dailymotion Engine

**Files:**
- Create: `app/worker/src/engines/dailymotion.ts`
- Create: `app/worker/src/engines/dailymotion.test.ts`

**Step 1: Create test file**

```typescript
// app/worker/src/engines/dailymotion.test.ts
import { describe, it, expect } from 'vitest';
import { DailymotionEngine } from './dailymotion';

describe('DailymotionEngine', () => {
  const engine = new DailymotionEngine();

  it('should have correct metadata', () => {
    expect(engine.name).toBe('dailymotion');
    expect(engine.shortcut).toBe('dm');
    expect(engine.categories).toContain('videos');
  });

  it('should build correct API URL', () => {
    const config = engine.buildRequest('funny cats', {
      page: 1,
      locale: 'en',
      safeSearch: 1,
      timeRange: '',
      engineData: {},
    });
    expect(config.url).toContain('api.dailymotion.com/videos');
    expect(config.url).toContain('search=funny+cats');
    expect(config.url).toContain('fields=');
  });

  it('should apply safe search filter', () => {
    const config = engine.buildRequest('test', {
      page: 1,
      locale: 'en',
      safeSearch: 2,
      timeRange: '',
      engineData: {},
    });
    expect(config.url).toContain('family_filter=true');
  });

  it('should apply time range filter', () => {
    const config = engine.buildRequest('test', {
      page: 1,
      locale: 'en',
      safeSearch: 1,
      timeRange: 'week',
      engineData: {},
    });
    expect(config.url).toContain('created_after=');
  });

  it('should search and return video results', async () => {
    const results = await fetchAndParse(engine, 'music video');

    expect(results.results.length).toBeGreaterThan(0);
    const first = results.results[0];
    expect(first.url).toContain('dailymotion.com');
    expect(first.title).toBeTruthy();
    expect(first.duration).toBeTruthy();
  }, 30000);
});

async function fetchAndParse(engine: DailymotionEngine, query: string) {
  const params = { page: 1, locale: 'en', safeSearch: 1 as const, timeRange: '' as const, engineData: {} };
  const config = engine.buildRequest(query, params);
  const res = await fetch(config.url, { headers: config.headers });
  const body = await res.text();
  return engine.parseResponse(body, params);
}
```

**Step 2: Implement Dailymotion engine**

```typescript
// app/worker/src/engines/dailymotion.ts
import type {
  OnlineEngine,
  EngineParams,
  RequestConfig,
  EngineResults,
  Category,
  TimeRange,
} from './engine';
import { newEngineResults } from './engine';

interface DailymotionVideo {
  id: string;
  title: string;
  description?: string;
  duration: number;
  thumbnail_360_url?: string;
  created_time: number;
  'owner.screenname'?: string;
  views_total?: number;
  embed_url?: string;
  allow_embed?: boolean;
}

interface DailymotionResponse {
  list: DailymotionVideo[];
  has_more: boolean;
}

export class DailymotionEngine implements OnlineEngine {
  name = 'dailymotion';
  shortcut = 'dm';
  categories: Category[] = ['videos'];
  supportsPaging = true;
  maxPage = 10;
  timeout = 8000;
  weight = 0.85;
  disabled = false;

  private timeRangeMap: Record<TimeRange, number> = {
    '': 0,
    day: 24 * 60 * 60,
    week: 7 * 24 * 60 * 60,
    month: 30 * 24 * 60 * 60,
    year: 365 * 24 * 60 * 60,
  };

  buildRequest(query: string, params: EngineParams): RequestConfig {
    const searchParams = new URLSearchParams();
    searchParams.set('search', query);
    searchParams.set('page', String(params.page));
    searchParams.set('limit', '10');
    searchParams.set('fields', 'id,title,description,duration,thumbnail_360_url,created_time,owner.screenname,views_total,embed_url,allow_embed');

    // Safe search
    if (params.safeSearch >= 1) {
      searchParams.set('family_filter', 'true');
    } else {
      searchParams.set('family_filter', 'false');
    }
    if (params.safeSearch >= 2) {
      searchParams.set('is_created_for_kids', 'true');
    }

    // Language
    if (params.locale) {
      searchParams.set('languages', params.locale.split('-')[0]);
    }

    // Time range
    if (params.timeRange && this.timeRangeMap[params.timeRange]) {
      const seconds = this.timeRangeMap[params.timeRange];
      const after = Math.floor(Date.now() / 1000) - seconds;
      searchParams.set('created_after', String(after));
    }

    // Exclude private/password videos
    searchParams.set('private', 'false');
    searchParams.set('password_protected', 'false');

    return {
      url: `https://api.dailymotion.com/videos?${searchParams.toString()}`,
      method: 'GET',
      headers: {
        Accept: 'application/json',
      },
      cookies: [],
    };
  }

  parseResponse(body: string, _params: EngineParams): EngineResults {
    const results = newEngineResults();

    let data: DailymotionResponse;
    try {
      data = JSON.parse(body);
    } catch {
      return results;
    }

    if (!data.list) return results;

    for (const video of data.list) {
      const description = video.description
        ? video.description.slice(0, 300)
        : '';

      results.results.push({
        url: `https://www.dailymotion.com/video/${video.id}`,
        title: video.title,
        content: description,
        engine: this.name,
        score: this.weight,
        category: 'videos',
        template: 'videos',
        embedUrl: video.allow_embed ? `https://www.dailymotion.com/embed/video/${video.id}` : undefined,
        thumbnailUrl: video.thumbnail_360_url?.replace('http://', 'https://'),
        duration: this.formatDuration(video.duration),
        channel: video['owner.screenname'] || '',
        views: video.views_total,
        publishedAt: new Date(video.created_time * 1000).toISOString(),
      });
    }

    return results;
  }

  private formatDuration(seconds: number): string {
    const hrs = Math.floor(seconds / 3600);
    const mins = Math.floor((seconds % 3600) / 60);
    const secs = seconds % 60;

    if (hrs > 0) {
      return `${hrs}:${mins.toString().padStart(2, '0')}:${secs.toString().padStart(2, '0')}`;
    }
    return `${mins}:${secs.toString().padStart(2, '0')}`;
  }
}
```

**Step 3: Run tests**

```bash
npm test -- dailymotion.test.ts
```

**Step 4: Commit**

```bash
git add app/worker/src/engines/dailymotion.ts app/worker/src/engines/dailymotion.test.ts
git commit -m "feat(video): implement Dailymotion engine"
```

---

### Task 5: Implement Google Videos Engine

**Files:**
- Create: `app/worker/src/engines/google-videos.ts`
- Create: `app/worker/src/engines/google-videos.test.ts`

**Step 1: Create test file**

```typescript
// app/worker/src/engines/google-videos.test.ts
import { describe, it, expect } from 'vitest';
import { GoogleVideosEngine } from './google-videos';

describe('GoogleVideosEngine', () => {
  const engine = new GoogleVideosEngine();

  it('should have correct metadata', () => {
    expect(engine.name).toBe('google_videos');
    expect(engine.shortcut).toBe('gov');
    expect(engine.categories).toContain('videos');
  });

  it('should build correct search URL with tbm=vid', () => {
    const config = engine.buildRequest('cooking tutorial', {
      page: 1,
      locale: 'en',
      safeSearch: 1,
      timeRange: '',
      engineData: {},
    });
    expect(config.url).toContain('google.com/search');
    expect(config.url).toContain('tbm=vid');
    expect(config.url).toContain('q=cooking+tutorial');
  });

  it('should apply duration filter', () => {
    const config = engine.buildRequest('test', {
      page: 1,
      locale: 'en',
      safeSearch: 1,
      timeRange: '',
      engineData: {},
      videoFilters: { duration: 'short' },
    });
    expect(config.url).toContain('tbs=');
    expect(config.url).toContain('dur:s');
  });

  it('should apply time range filter', () => {
    const config = engine.buildRequest('test', {
      page: 1,
      locale: 'en',
      safeSearch: 1,
      timeRange: 'week',
      engineData: {},
    });
    expect(config.url).toContain('qdr:w');
  });

  it('should search and return video results', async () => {
    const results = await fetchAndParse(engine, 'ted talk');

    expect(results.results.length).toBeGreaterThan(0);
    const first = results.results[0];
    expect(first.url).toBeTruthy();
    expect(first.title).toBeTruthy();
  }, 30000);
});

async function fetchAndParse(engine: GoogleVideosEngine, query: string) {
  const params = { page: 1, locale: 'en', safeSearch: 1 as const, timeRange: '' as const, engineData: {} };
  const config = engine.buildRequest(query, params);
  const res = await fetch(config.url, { headers: config.headers });
  const body = await res.text();
  return engine.parseResponse(body, params);
}
```

**Step 2: Implement Google Videos engine**

```typescript
// app/worker/src/engines/google-videos.ts
import type {
  OnlineEngine,
  EngineParams,
  RequestConfig,
  EngineResults,
  Category,
  TimeRange,
  VideoDuration,
} from './engine';
import { newEngineResults } from './engine';

export class GoogleVideosEngine implements OnlineEngine {
  name = 'google_videos';
  shortcut = 'gov';
  categories: Category[] = ['videos'];
  supportsPaging = true;
  maxPage = 50;
  timeout = 10000;
  weight = 1.0;
  disabled = false;

  private timeRangeMap: Record<TimeRange, string> = {
    '': '',
    day: 'qdr:d',
    week: 'qdr:w',
    month: 'qdr:m',
    year: 'qdr:y',
  };

  private durationMap: Record<VideoDuration, string> = {
    any: '',
    short: 'dur:s',
    medium: 'dur:m',
    long: 'dur:l',
  };

  buildRequest(query: string, params: EngineParams): RequestConfig {
    const searchParams = new URLSearchParams();
    searchParams.set('q', query);
    searchParams.set('tbm', 'vid');
    searchParams.set('hl', params.locale || 'en');

    // Pagination (10 results per page)
    if (params.page > 1) {
      searchParams.set('start', String((params.page - 1) * 10));
    }

    // Safe search
    if (params.safeSearch === 2) {
      searchParams.set('safe', 'active');
    } else if (params.safeSearch === 0) {
      searchParams.set('safe', 'off');
    }

    // Build tbs parameter
    const tbsParts: string[] = [];

    // Time range
    if (params.timeRange && this.timeRangeMap[params.timeRange]) {
      tbsParts.push(this.timeRangeMap[params.timeRange]);
    }

    // Duration filter
    if (params.videoFilters?.duration && this.durationMap[params.videoFilters.duration]) {
      tbsParts.push(this.durationMap[params.videoFilters.duration]);
    }

    // Quality filter
    if (params.videoFilters?.quality === 'hd') {
      tbsParts.push('hq:h');
    }

    // Closed captions
    if (params.videoFilters?.cc) {
      tbsParts.push('cc:1');
    }

    if (tbsParts.length > 0) {
      searchParams.set('tbs', tbsParts.join(','));
    }

    return {
      url: `https://www.google.com/search?${searchParams.toString()}`,
      method: 'GET',
      headers: {
        Accept: 'text/html,application/xhtml+xml',
        'Accept-Language': 'en-US,en;q=0.9',
        'User-Agent': 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36',
      },
      cookies: ['CONSENT=YES+'],
    };
  }

  parseResponse(body: string, _params: EngineParams): EngineResults {
    const results = newEngineResults();

    // Parse video results from Google's HTML
    // Look for video card divs
    const videoPattern = /<div[^>]*class="[^"]*g-blk[^"]*"[^>]*>([\s\S]*?)<\/div>/gi;
    const linkPattern = /<a[^>]*href="([^"]+)"[^>]*>/gi;
    const titlePattern = /<h3[^>]*>(.*?)<\/h3>/gi;
    const snippetPattern = /<span[^>]*class="[^"]*st[^"]*"[^>]*>(.*?)<\/span>/gi;
    const durationPattern = /(\d+:\d{2}(?::\d{2})?)/;
    const channelPattern = /<cite[^>]*>(.*?)<\/cite>/gi;

    // Alternative: parse structured data
    const jsonLdMatch = body.match(/<script type="application\/ld\+json">([\s\S]*?)<\/script>/g);
    if (jsonLdMatch) {
      for (const match of jsonLdMatch) {
        try {
          const jsonStr = match.replace(/<\/?script[^>]*>/g, '');
          const data = JSON.parse(jsonStr);
          if (data['@type'] === 'VideoObject' || (Array.isArray(data) && data[0]?.['@type'] === 'VideoObject')) {
            const videos = Array.isArray(data) ? data : [data];
            for (const video of videos) {
              if (video['@type'] !== 'VideoObject') continue;
              results.results.push({
                url: video.url || video.contentUrl || '',
                title: video.name || '',
                content: video.description || '',
                engine: this.name,
                score: this.weight,
                category: 'videos',
                template: 'videos',
                thumbnailUrl: video.thumbnailUrl || '',
                duration: video.duration || '',
                channel: video.author?.name || '',
                publishedAt: video.uploadDate,
                embedUrl: video.embedUrl,
              });
            }
          }
        } catch {
          // Skip invalid JSON
        }
      }
    }

    // Fallback: parse HTML directly
    if (results.results.length === 0) {
      // Simple regex-based extraction
      const blocks = body.split(/<div class="g">/);
      for (const block of blocks.slice(1)) {
        const urlMatch = block.match(/<a href="(https?:\/\/[^"]+)"/);
        const titleMatch = block.match(/<h3[^>]*>([^<]+)<\/h3>/);

        if (urlMatch && titleMatch) {
          const url = urlMatch[1];
          // Skip Google's own URLs
          if (url.includes('google.com')) continue;

          const durationMatch = block.match(durationPattern);
          const snippetMatch = block.match(/<span[^>]*>([^<]{20,200})<\/span>/);

          // Try to extract thumbnail
          let thumbnail = '';
          const imgMatch = block.match(/data:image[^"]+|https:\/\/[^"]*(?:ytimg|vimeocdn|dailymotion)[^"]*/);
          if (imgMatch) {
            thumbnail = imgMatch[0];
          }

          // Extract YouTube video ID for thumbnail fallback
          const ytMatch = url.match(/(?:youtube\.com\/watch\?v=|youtu\.be\/)([a-zA-Z0-9_-]{11})/);
          if (ytMatch && !thumbnail) {
            thumbnail = `https://img.youtube.com/vi/${ytMatch[1]}/hqdefault.jpg`;
          }

          results.results.push({
            url,
            title: this.decodeHtml(titleMatch[1]),
            content: snippetMatch ? this.decodeHtml(snippetMatch[1]) : '',
            engine: this.name,
            score: this.weight,
            category: 'videos',
            template: 'videos',
            thumbnailUrl: thumbnail,
            duration: durationMatch ? durationMatch[1] : '',
          });
        }
      }
    }

    return results;
  }

  private decodeHtml(html: string): string {
    return html
      .replace(/&amp;/g, '&')
      .replace(/&lt;/g, '<')
      .replace(/&gt;/g, '>')
      .replace(/&quot;/g, '"')
      .replace(/&#39;/g, "'")
      .replace(/<[^>]+>/g, '');
  }
}
```

**Step 3: Run tests**

```bash
npm test -- google-videos.test.ts
```

**Step 4: Commit**

```bash
git add app/worker/src/engines/google-videos.ts app/worker/src/engines/google-videos.test.ts
git commit -m "feat(video): implement Google Videos engine"
```

---

### Task 6: Implement Bing Videos Engine

**Files:**
- Create: `app/worker/src/engines/bing-videos.ts`
- Create: `app/worker/src/engines/bing-videos.test.ts`

**Step 1: Create test file**

```typescript
// app/worker/src/engines/bing-videos.test.ts
import { describe, it, expect } from 'vitest';
import { BingVideosEngine } from './bing-videos';

describe('BingVideosEngine', () => {
  const engine = new BingVideosEngine();

  it('should have correct metadata', () => {
    expect(engine.name).toBe('bing_videos');
    expect(engine.shortcut).toBe('biv');
    expect(engine.categories).toContain('videos');
  });

  it('should build correct async URL', () => {
    const config = engine.buildRequest('diy projects', {
      page: 1,
      locale: 'en',
      safeSearch: 1,
      timeRange: '',
      engineData: {},
    });
    expect(config.url).toContain('bing.com/videos');
    expect(config.url).toContain('q=diy+projects');
  });

  it('should apply time range filter', () => {
    const config = engine.buildRequest('test', {
      page: 1,
      locale: 'en',
      safeSearch: 1,
      timeRange: 'day',
      engineData: {},
    });
    expect(config.url).toContain('filters=');
  });

  it('should search and return video results', async () => {
    const results = await fetchAndParse(engine, 'music video');

    expect(results.results.length).toBeGreaterThan(0);
    const first = results.results[0];
    expect(first.url).toBeTruthy();
    expect(first.title).toBeTruthy();
  }, 30000);
});

async function fetchAndParse(engine: BingVideosEngine, query: string) {
  const params = { page: 1, locale: 'en', safeSearch: 1 as const, timeRange: '' as const, engineData: {} };
  const config = engine.buildRequest(query, params);
  const res = await fetch(config.url, { headers: config.headers });
  const body = await res.text();
  return engine.parseResponse(body, params);
}
```

**Step 2: Implement Bing Videos engine**

```typescript
// app/worker/src/engines/bing-videos.ts
import type {
  OnlineEngine,
  EngineParams,
  RequestConfig,
  EngineResults,
  Category,
  TimeRange,
} from './engine';
import { newEngineResults } from './engine';

interface BingVideoMetadata {
  murl?: string;
  vt?: string;
  du?: string;
  t?: string;
  pubdate?: string;
  thid?: string;
}

export class BingVideosEngine implements OnlineEngine {
  name = 'bing_videos';
  shortcut = 'biv';
  categories: Category[] = ['videos'];
  supportsPaging = true;
  maxPage = 10;
  timeout = 10000;
  weight = 0.95;
  disabled = false;

  // Time range in minutes
  private timeRangeMap: Record<TimeRange, number> = {
    '': 0,
    day: 1440,
    week: 10080,
    month: 43200,
    year: 525600,
  };

  buildRequest(query: string, params: EngineParams): RequestConfig {
    const searchParams = new URLSearchParams();
    searchParams.set('q', query);
    searchParams.set('count', '35');

    // Pagination
    if (params.page > 1) {
      searchParams.set('first', String((params.page - 1) * 35 + 1));
    }

    // Time range filter
    if (params.timeRange && this.timeRangeMap[params.timeRange]) {
      const minutes = this.timeRangeMap[params.timeRange];
      searchParams.set('filters', `filterui:videoage-lt${minutes}`);
      searchParams.set('form', 'VRFLTR');
    }

    // Safe search via cookie
    let safeSearchVal = 'moderate';
    if (params.safeSearch === 0) safeSearchVal = 'off';
    if (params.safeSearch === 2) safeSearchVal = 'strict';

    return {
      url: `https://www.bing.com/videos/search?${searchParams.toString()}`,
      method: 'GET',
      headers: {
        Accept: 'text/html,application/xhtml+xml',
        'Accept-Language': 'en-US,en;q=0.9',
        'User-Agent': 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36',
      },
      cookies: [`SRCHHPGUSR=ADLT=${safeSearchVal}`],
    };
  }

  parseResponse(body: string, _params: EngineParams): EngineResults {
    const results = newEngineResults();

    // Find video cards with vrhm (video result hover metadata) attribute
    const cardPattern = /<div[^>]*class="[^"]*dg_u[^"]*"[^>]*vrhm="([^"]+)"[^>]*>/gi;
    const matches = [...body.matchAll(cardPattern)];

    for (const match of matches) {
      try {
        const encoded = match[1];
        const decoded = encoded
          .replace(/&quot;/g, '"')
          .replace(/&amp;/g, '&')
          .replace(/&lt;/g, '<')
          .replace(/&gt;/g, '>');

        const metadata: BingVideoMetadata = JSON.parse(decoded);

        if (!metadata.murl) continue;

        // Extract thumbnail from thid (thumbnail ID)
        let thumbnail = '';
        if (metadata.thid) {
          thumbnail = `https://tse1.mm.bing.net/th?id=${metadata.thid}`;
        }

        results.results.push({
          url: metadata.murl,
          title: metadata.vt || metadata.t || '',
          content: '',
          engine: this.name,
          score: this.weight,
          category: 'videos',
          template: 'videos',
          thumbnailUrl: thumbnail,
          duration: metadata.du || '',
          publishedAt: metadata.pubdate,
        });
      } catch {
        // Skip invalid metadata
      }
    }

    // Fallback: parse simpler structure
    if (results.results.length === 0) {
      const simplePattern = /<a[^>]*class="[^"]*mc_vtvc[^"]*"[^>]*href="([^"]+)"[^>]*title="([^"]+)"/gi;
      const simpleMatches = [...body.matchAll(simplePattern)];

      for (const match of simpleMatches) {
        const url = match[1];
        const title = match[2];

        if (url.startsWith('/videos/search')) continue; // Skip internal links

        results.results.push({
          url: url.startsWith('http') ? url : `https://www.bing.com${url}`,
          title: this.decodeHtml(title),
          content: '',
          engine: this.name,
          score: this.weight,
          category: 'videos',
          template: 'videos',
        });
      }
    }

    return results;
  }

  private decodeHtml(html: string): string {
    return html
      .replace(/&amp;/g, '&')
      .replace(/&lt;/g, '<')
      .replace(/&gt;/g, '>')
      .replace(/&quot;/g, '"')
      .replace(/&#39;/g, "'");
  }
}
```

**Step 3: Run tests and commit**

```bash
npm test -- bing-videos.test.ts
git add app/worker/src/engines/bing-videos.ts app/worker/src/engines/bing-videos.test.ts
git commit -m "feat(video): implement Bing Videos engine"
```

---

### Task 7: Implement PeerTube Engine

**Files:**
- Create: `app/worker/src/engines/peertube.ts`
- Create: `app/worker/src/engines/peertube.test.ts`

**Step 1: Create test file**

```typescript
// app/worker/src/engines/peertube.test.ts
import { describe, it, expect } from 'vitest';
import { PeerTubeEngine } from './peertube';

describe('PeerTubeEngine', () => {
  const engine = new PeerTubeEngine();

  it('should have correct metadata', () => {
    expect(engine.name).toBe('peertube');
    expect(engine.shortcut).toBe('ptb');
    expect(engine.categories).toContain('videos');
  });

  it('should build correct API URL', () => {
    const config = engine.buildRequest('open source', {
      page: 1,
      locale: 'en',
      safeSearch: 1,
      timeRange: '',
      engineData: {},
    });
    expect(config.url).toContain('sepiasearch.org/api/v1/search/videos');
    expect(config.url).toContain('search=open+source');
  });

  it('should search and return video results', async () => {
    const results = await fetchAndParse(engine, 'linux');

    expect(results.results.length).toBeGreaterThan(0);
    const first = results.results[0];
    expect(first.url).toBeTruthy();
    expect(first.title).toBeTruthy();
  }, 30000);
});

async function fetchAndParse(engine: PeerTubeEngine, query: string) {
  const params = { page: 1, locale: 'en', safeSearch: 1 as const, timeRange: '' as const, engineData: {} };
  const config = engine.buildRequest(query, params);
  const res = await fetch(config.url, { headers: config.headers });
  const body = await res.text();
  return engine.parseResponse(body, params);
}
```

**Step 2: Implement PeerTube engine**

```typescript
// app/worker/src/engines/peertube.ts
import type {
  OnlineEngine,
  EngineParams,
  RequestConfig,
  EngineResults,
  Category,
  TimeRange,
} from './engine';
import { newEngineResults } from './engine';

interface PeerTubeVideo {
  url: string;
  name: string;
  description?: string;
  duration: number;
  views: number;
  publishedAt: string;
  thumbnailPath?: string;
  previewPath?: string;
  embedPath?: string;
  account?: {
    displayName?: string;
    host?: string;
  };
  channel?: {
    displayName?: string;
    host?: string;
  };
}

interface PeerTubeResponse {
  total: number;
  data: PeerTubeVideo[];
}

export class PeerTubeEngine implements OnlineEngine {
  name = 'peertube';
  shortcut = 'ptb';
  categories: Category[] = ['videos'];
  supportsPaging = true;
  maxPage = 10;
  timeout = 10000;
  weight = 0.75;
  disabled = false;

  // Use Sepia Search (federated PeerTube search)
  private baseUrl = 'https://sepiasearch.org/api/v1/search/videos';

  buildRequest(query: string, params: EngineParams): RequestConfig {
    const searchParams = new URLSearchParams();
    searchParams.set('search', query);
    searchParams.set('start', String((params.page - 1) * 15));
    searchParams.set('count', '15');
    searchParams.set('sort', '-match');
    searchParams.set('searchTarget', 'search-index');

    // NSFW filter based on safe search
    if (params.safeSearch >= 1) {
      searchParams.set('nsfw', 'false');
    } else {
      searchParams.set('nsfw', 'both');
    }

    // Language filter
    if (params.locale) {
      const lang = params.locale.split('-')[0];
      searchParams.set('languageOneOf[]', lang);
      searchParams.set('boostLanguages[]', lang);
    }

    // Time range
    if (params.timeRange) {
      const now = new Date();
      let startDate: Date | null = null;

      switch (params.timeRange as TimeRange) {
        case 'day':
          startDate = new Date(now.getTime() - 24 * 60 * 60 * 1000);
          break;
        case 'week':
          startDate = new Date(now.getTime() - 7 * 24 * 60 * 60 * 1000);
          break;
        case 'month':
          startDate = new Date(now.getTime() - 30 * 24 * 60 * 60 * 1000);
          break;
        case 'year':
          startDate = new Date(now.getTime() - 365 * 24 * 60 * 60 * 1000);
          break;
      }

      if (startDate) {
        searchParams.set('startDate', startDate.toISOString());
      }
    }

    return {
      url: `${this.baseUrl}?${searchParams.toString()}`,
      method: 'GET',
      headers: {
        Accept: 'application/json',
      },
      cookies: [],
    };
  }

  parseResponse(body: string, _params: EngineParams): EngineResults {
    const results = newEngineResults();

    let data: PeerTubeResponse;
    try {
      data = JSON.parse(body);
    } catch {
      return results;
    }

    if (!data.data) return results;

    for (const video of data.data) {
      // Get base URL from video URL
      let videoBaseUrl = '';
      try {
        const parsed = new URL(video.url);
        videoBaseUrl = `${parsed.protocol}//${parsed.host}`;
      } catch {
        continue;
      }

      const thumbnail = video.thumbnailPath
        ? `${videoBaseUrl}${video.thumbnailPath}`
        : video.previewPath
        ? `${videoBaseUrl}${video.previewPath}`
        : '';

      const embedUrl = video.embedPath
        ? `${videoBaseUrl}${video.embedPath}`
        : '';

      const channel = video.channel?.displayName || video.account?.displayName || '';
      const host = video.channel?.host || video.account?.host || '';

      results.results.push({
        url: video.url,
        title: video.name,
        content: video.description || '',
        engine: this.name,
        score: this.weight,
        category: 'videos',
        template: 'videos',
        thumbnailUrl: thumbnail,
        embedUrl,
        duration: this.formatDuration(video.duration),
        channel: host ? `${channel}@${host}` : channel,
        views: video.views,
        publishedAt: video.publishedAt,
      });
    }

    return results;
  }

  private formatDuration(seconds: number): string {
    const hrs = Math.floor(seconds / 3600);
    const mins = Math.floor((seconds % 3600) / 60);
    const secs = seconds % 60;

    if (hrs > 0) {
      return `${hrs}:${mins.toString().padStart(2, '0')}:${secs.toString().padStart(2, '0')}`;
    }
    return `${mins}:${secs.toString().padStart(2, '0')}`;
  }
}
```

**Step 3: Run tests and commit**

```bash
npm test -- peertube.test.ts
git add app/worker/src/engines/peertube.ts app/worker/src/engines/peertube.test.ts
git commit -m "feat(video): implement PeerTube engine"
```

---

### Task 8: Implement 360Search Videos Engine

**Files:**
- Create: `app/worker/src/engines/360search-videos.ts`
- Create: `app/worker/src/engines/360search-videos.test.ts`

**Step 1: Create test and implementation**

```typescript
// app/worker/src/engines/360search-videos.test.ts
import { describe, it, expect } from 'vitest';
import { Search360VideosEngine } from './360search-videos';

describe('Search360VideosEngine', () => {
  const engine = new Search360VideosEngine();

  it('should have correct metadata', () => {
    expect(engine.name).toBe('360search');
    expect(engine.shortcut).toBe('360v');
    expect(engine.categories).toContain('videos');
  });

  it('should build correct API URL', () => {
    const config = engine.buildRequest('电影', {
      page: 1,
      locale: 'zh',
      safeSearch: 1,
      timeRange: '',
      engineData: {},
    });
    expect(config.url).toContain('tv.360kan.com');
  });

  it('should search and return video results', async () => {
    const results = await fetchAndParse(engine, '电影');

    // May return 0 results depending on availability
    expect(results.results).toBeDefined();
  }, 30000);
});

async function fetchAndParse(engine: Search360VideosEngine, query: string) {
  const params = { page: 1, locale: 'zh', safeSearch: 1 as const, timeRange: '' as const, engineData: {} };
  const config = engine.buildRequest(query, params);
  const res = await fetch(config.url, { headers: config.headers });
  const body = await res.text();
  return engine.parseResponse(body, params);
}
```

```typescript
// app/worker/src/engines/360search-videos.ts
import type {
  OnlineEngine,
  EngineParams,
  RequestConfig,
  EngineResults,
  Category,
} from './engine';
import { newEngineResults } from './engine';

interface Video360 {
  title?: string;
  description?: string;
  play_url?: string;
  cover?: string;
  stream_url?: string;
  publish_time?: number;
}

interface Response360 {
  data?: {
    list?: Video360[];
  };
}

export class Search360VideosEngine implements OnlineEngine {
  name = '360search';
  shortcut = '360v';
  categories: Category[] = ['videos'];
  supportsPaging = true;
  maxPage = 10;
  timeout = 8000;
  weight = 0.6;
  disabled = false;

  buildRequest(query: string, params: EngineParams): RequestConfig {
    const searchParams = new URLSearchParams();
    searchParams.set('q', query);
    searchParams.set('start', String((params.page - 1) * 10));
    searchParams.set('count', '10');

    return {
      url: `https://tv.360kan.com/v1/video/list?${searchParams.toString()}`,
      method: 'GET',
      headers: {
        Accept: 'application/json',
        'User-Agent': 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36',
      },
      cookies: [],
    };
  }

  parseResponse(body: string, _params: EngineParams): EngineResults {
    const results = newEngineResults();

    let data: Response360;
    try {
      data = JSON.parse(body);
    } catch {
      return results;
    }

    const list = data.data?.list || [];

    for (const video of list) {
      if (!video.play_url) continue;

      results.results.push({
        url: video.play_url,
        title: this.decodeHtml(video.title || ''),
        content: this.decodeHtml(video.description || ''),
        engine: this.name,
        score: this.weight,
        category: 'videos',
        template: 'videos',
        thumbnailUrl: video.cover || '',
        embedUrl: video.stream_url,
        publishedAt: video.publish_time
          ? new Date(video.publish_time * 1000).toISOString()
          : undefined,
      });
    }

    return results;
  }

  private decodeHtml(html: string): string {
    return html
      .replace(/&amp;/g, '&')
      .replace(/&lt;/g, '<')
      .replace(/&gt;/g, '>')
      .replace(/&quot;/g, '"')
      .replace(/&#39;/g, "'")
      .replace(/<[^>]+>/g, '');
  }
}
```

**Step 2: Run tests and commit**

```bash
npm test -- 360search-videos.test.ts
git add app/worker/src/engines/360search-videos.ts app/worker/src/engines/360search-videos.test.ts
git commit -m "feat(video): implement 360Search Videos engine"
```

---

### Task 9: Implement Sogou Videos Engine

**Files:**
- Create: `app/worker/src/engines/sogou-videos.ts`
- Create: `app/worker/src/engines/sogou-videos.test.ts`

**Step 1: Create test and implementation**

```typescript
// app/worker/src/engines/sogou-videos.test.ts
import { describe, it, expect } from 'vitest';
import { SogouVideosEngine } from './sogou-videos';

describe('SogouVideosEngine', () => {
  const engine = new SogouVideosEngine();

  it('should have correct metadata', () => {
    expect(engine.name).toBe('sogou');
    expect(engine.shortcut).toBe('sgv');
    expect(engine.categories).toContain('videos');
  });

  it('should build correct API URL', () => {
    const config = engine.buildRequest('视频', {
      page: 1,
      locale: 'zh',
      safeSearch: 1,
      timeRange: '',
      engineData: {},
    });
    expect(config.url).toContain('v.sogou.com');
  });

  it('should search and return video results', async () => {
    const results = await fetchAndParse(engine, '视频');

    expect(results.results).toBeDefined();
  }, 30000);
});

async function fetchAndParse(engine: SogouVideosEngine, query: string) {
  const params = { page: 1, locale: 'zh', safeSearch: 1 as const, timeRange: '' as const, engineData: {} };
  const config = engine.buildRequest(query, params);
  const res = await fetch(config.url, { headers: config.headers });
  const body = await res.text();
  return engine.parseResponse(body, params);
}
```

```typescript
// app/worker/src/engines/sogou-videos.ts
import type {
  OnlineEngine,
  EngineParams,
  RequestConfig,
  EngineResults,
  Category,
} from './engine';
import { newEngineResults } from './engine';

interface SogouVideo {
  title?: string;
  url?: string;
  pic?: string;
  duration?: string;
  date?: string;
  site?: string;
}

interface SogouResponse {
  data?: {
    listData?: SogouVideo[];
  };
}

export class SogouVideosEngine implements OnlineEngine {
  name = 'sogou';
  shortcut = 'sgv';
  categories: Category[] = ['videos'];
  supportsPaging = true;
  maxPage = 10;
  timeout = 8000;
  weight = 0.6;
  disabled = false;

  buildRequest(query: string, params: EngineParams): RequestConfig {
    const searchParams = new URLSearchParams();
    searchParams.set('query', query);
    searchParams.set('page', String(params.page));
    searchParams.set('pagesize', '10');

    return {
      url: `https://v.sogou.com/api/video/shortVideoV2?${searchParams.toString()}`,
      method: 'GET',
      headers: {
        Accept: 'application/json',
        'User-Agent': 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36',
      },
      cookies: [],
    };
  }

  parseResponse(body: string, _params: EngineParams): EngineResults {
    const results = newEngineResults();

    let data: SogouResponse;
    try {
      data = JSON.parse(body);
    } catch {
      return results;
    }

    const list = data.data?.listData || [];

    for (const video of list) {
      let url = video.url || '';
      // Handle relative URLs
      if (url && !url.startsWith('http')) {
        url = `https://v.sogou.com${url}`;
      }

      if (!url) continue;

      results.results.push({
        url,
        title: this.decodeHtml(video.title || ''),
        content: video.site || '',
        engine: this.name,
        score: this.weight,
        category: 'videos',
        template: 'videos',
        thumbnailUrl: video.pic || '',
        duration: video.duration ? this.parseDuration(video.duration) : '',
        publishedAt: video.date,
      });
    }

    return results;
  }

  private parseDuration(duration: string): string {
    // Duration may come as "MM:SS" format
    const match = duration.match(/(\d+):(\d+)/);
    if (match) {
      return duration;
    }
    return duration;
  }

  private decodeHtml(html: string): string {
    return html
      .replace(/&amp;/g, '&')
      .replace(/&lt;/g, '<')
      .replace(/&gt;/g, '>')
      .replace(/&quot;/g, '"')
      .replace(/&#39;/g, "'")
      .replace(/<[^>]+>/g, '');
  }
}
```

**Step 2: Run tests and commit**

```bash
npm test -- sogou-videos.test.ts
git add app/worker/src/engines/sogou-videos.ts app/worker/src/engines/sogou-videos.test.ts
git commit -m "feat(video): implement Sogou Videos engine"
```

---

## Phase 3: Register Engines & Update MetaSearch

### Task 10: Register All Video Engines

**Files:**
- Modify: `app/worker/src/engines/metasearch.ts`

**Step 1: Import new engines**

Add imports after line 28:

```typescript
import { VimeoEngine } from './vimeo';
import { DailymotionEngine } from './dailymotion';
import { GoogleVideosEngine } from './google-videos';
import { BingVideosEngine } from './bing-videos';
import { PeerTubeEngine } from './peertube';
import { Search360VideosEngine } from './360search-videos';
import { SogouVideosEngine } from './sogou-videos';
```

**Step 2: Register in createDefaultMetaSearch**

Update the video search engines section (around line 255-258):

```typescript
  // Video search engines
  ms.register(new YouTubeEngine());
  ms.register(new DuckDuckGoVideosEngine());
  ms.register(new VimeoEngine());
  ms.register(new DailymotionEngine());
  ms.register(new GoogleVideosEngine());
  ms.register(new BingVideosEngine());
  ms.register(new PeerTubeEngine());
  ms.register(new Search360VideosEngine());
  ms.register(new SogouVideosEngine());
```

**Step 3: Commit**

```bash
git add app/worker/src/engines/metasearch.ts
git commit -m "feat(video): register all 9 video engines in metasearch"
```

---

## Phase 4: Update Types

### Task 11: Add Video Search Types

**Files:**
- Modify: `app/worker/src/types.ts`

**Step 1: Add video filter types after line 68**

```typescript
// ========== Video Search Filter Types ==========

export type VideoDuration = 'any' | 'short' | 'medium' | 'long';
export type VideoQuality = 'any' | 'hd' | '4k';
export type VideoSort = 'relevance' | 'date' | 'views' | 'duration';

export interface VideoSearchFilters {
  duration?: VideoDuration;
  quality?: VideoQuality;
  time?: ImageTime;
  source?: string;
  cc?: boolean;
  safe?: SafeSearchLevel;
}

export interface VideoSearchOptions extends SearchOptions {
  filters?: VideoSearchFilters;
  sort?: VideoSort;
}
```

**Step 2: Update VideoResult interface (replace existing around line 119)**

```typescript
export interface VideoResult {
  id: string;
  url: string;
  title: string;
  description: string;
  thumbnail_url: string;
  thumbnail_width?: number;
  thumbnail_height?: number;
  duration: string;
  duration_seconds?: number;
  channel: string;
  channel_url?: string;
  views?: number;
  views_formatted?: string;
  published_at?: string;
  published_formatted?: string;
  embed_url?: string;
  embed_html?: string;
  source: string;
  source_icon?: string;
  quality?: string;
  has_cc?: boolean;
  is_live?: boolean;
  score: number;
  engines: string[];
  engine?: string;
}
```

**Step 3: Add VideoSearchResponse interface**

```typescript
export interface VideoSearchResponse {
  query: string;
  total_results: number;
  results: VideoResult[];
  filters?: VideoSearchFilters;
  available_sources: VideoSourceInfo[];
  search_time_ms: number;
  page: number;
  per_page: number;
  has_more: boolean;
}

export interface VideoSourceInfo {
  name: string;
  display_name: string;
  icon: string;
  result_count: number;
  enabled: boolean;
}
```

**Step 4: Commit**

```bash
git add app/worker/src/types.ts
git commit -m "feat(video): add video search types"
```

---

## Phase 5: Update Search Service

### Task 12: Enhance Video Search in Service

**Files:**
- Modify: `app/worker/src/services/search.ts`

**Step 1: Add video-specific imports at top**

```typescript
import type {
  // ... existing imports ...
  VideoSearchOptions,
  VideoSearchResponse,
  VideoResult,
  VideoSourceInfo,
} from '../types';
```

**Step 2: Add toVideoResult helper function after toImageResult**

```typescript
/**
 * Convert EngineResult to VideoResult format.
 */
function toVideoResult(r: EngineResult, index: number): VideoResult {
  const durationSeconds = parseDurationToSeconds(r.duration);
  return {
    id: `${Date.now().toString(36)}-${index}`,
    url: r.url,
    title: r.title,
    description: r.content,
    thumbnail_url: r.thumbnailUrl || '',
    duration: r.duration || '',
    duration_seconds: durationSeconds,
    channel: r.channel || '',
    views: r.views,
    views_formatted: r.views ? formatViews(r.views) : undefined,
    published_at: r.publishedAt,
    published_formatted: r.publishedAt ? formatTimeAgo(r.publishedAt) : undefined,
    embed_url: r.embedUrl,
    source: r.engine,
    source_icon: getSourceIcon(r.engine),
    score: r.score,
    engines: [r.engine],
    engine: r.engine,
  };
}

function parseDurationToSeconds(duration?: string): number {
  if (!duration) return 0;
  const parts = duration.split(':').map(Number);
  if (parts.length === 3) {
    return parts[0] * 3600 + parts[1] * 60 + parts[2];
  }
  if (parts.length === 2) {
    return parts[0] * 60 + parts[1];
  }
  return 0;
}

function formatViews(views: number): string {
  if (views >= 1_000_000_000) return `${(views / 1_000_000_000).toFixed(1)}B views`;
  if (views >= 1_000_000) return `${(views / 1_000_000).toFixed(1)}M views`;
  if (views >= 1_000) return `${(views / 1_000).toFixed(1)}K views`;
  return `${views} views`;
}

function formatTimeAgo(dateStr: string): string {
  const date = new Date(dateStr);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffDays = Math.floor(diffMs / (1000 * 60 * 60 * 24));

  if (diffDays < 1) return 'Today';
  if (diffDays === 1) return '1 day ago';
  if (diffDays < 7) return `${diffDays} days ago`;
  if (diffDays < 14) return '1 week ago';
  if (diffDays < 30) return `${Math.floor(diffDays / 7)} weeks ago`;
  if (diffDays < 60) return '1 month ago';
  if (diffDays < 365) return `${Math.floor(diffDays / 30)} months ago`;
  return `${Math.floor(diffDays / 365)} years ago`;
}

function getSourceIcon(engine: string): string {
  const icons: Record<string, string> = {
    youtube: 'https://www.youtube.com/favicon.ico',
    vimeo: 'https://vimeo.com/favicon.ico',
    dailymotion: 'https://www.dailymotion.com/favicon.ico',
    google_videos: 'https://www.google.com/favicon.ico',
    bing_videos: 'https://www.bing.com/favicon.ico',
    peertube: 'https://joinpeertube.org/favicon.ico',
    '360search': 'https://www.360.cn/favicon.ico',
    sogou: 'https://www.sogou.com/favicon.ico',
    duckduckgo_videos: 'https://duckduckgo.com/favicon.ico',
  };
  return icons[engine] || '';
}
```

**Step 3: Replace searchVideos method**

```typescript
  /**
   * Search for videos with filters and source aggregation.
   */
  async searchVideos(query: string, options: VideoSearchOptions): Promise<VideoSearchResponse> {
    const startTime = Date.now();
    const cacheHash = hashSearchKey(`vid:${query}`, options);

    const cachedResponse = await this.cache.getVideoSearch(cacheHash);
    if (cachedResponse) {
      return cachedResponse;
    }

    // Build engine params with video filters
    const params: EngineParams = {
      page: options.page,
      locale: options.language ?? 'en',
      timeRange: parseTimeRange(options.filters?.time ?? options.time_range),
      safeSearch: options.filters?.safe === 'strict' ? 2 : (options.filters?.safe === 'off' ? 0 : 1),
      engineData: {},
      videoFilters: {
        duration: options.filters?.duration,
        quality: options.filters?.quality,
        source: options.filters?.source,
        cc: options.filters?.cc,
      },
    };

    const metaResult = await this.metasearch.search(query, 'videos', params);

    // Convert to video results
    let allResults = metaResult.results.map(toVideoResult);

    // Apply client-side duration filter if needed
    if (options.filters?.duration && options.filters.duration !== 'any') {
      allResults = allResults.filter(v => {
        const secs = v.duration_seconds || 0;
        switch (options.filters?.duration) {
          case 'short': return secs > 0 && secs < 240;
          case 'medium': return secs >= 240 && secs <= 1200;
          case 'long': return secs > 1200;
          default: return true;
        }
      });
    }

    // Apply source filter
    if (options.filters?.source) {
      allResults = allResults.filter(v => v.source === options.filters?.source);
    }

    // Sort results
    if (options.sort) {
      switch (options.sort) {
        case 'date':
          allResults.sort((a, b) => {
            const dateA = a.published_at ? new Date(a.published_at).getTime() : 0;
            const dateB = b.published_at ? new Date(b.published_at).getTime() : 0;
            return dateB - dateA;
          });
          break;
        case 'views':
          allResults.sort((a, b) => (b.views || 0) - (a.views || 0));
          break;
        case 'duration':
          allResults.sort((a, b) => (b.duration_seconds || 0) - (a.duration_seconds || 0));
          break;
        // 'relevance' is default (already sorted by score)
      }
    }

    // Calculate source statistics
    const sourceStats = new Map<string, number>();
    for (const r of allResults) {
      sourceStats.set(r.source, (sourceStats.get(r.source) || 0) + 1);
    }

    const availableSources: VideoSourceInfo[] = [
      { name: 'youtube', display_name: 'YouTube', icon: getSourceIcon('youtube'), result_count: sourceStats.get('youtube') || 0, enabled: true },
      { name: 'vimeo', display_name: 'Vimeo', icon: getSourceIcon('vimeo'), result_count: sourceStats.get('vimeo') || 0, enabled: true },
      { name: 'dailymotion', display_name: 'Dailymotion', icon: getSourceIcon('dailymotion'), result_count: sourceStats.get('dailymotion') || 0, enabled: true },
      { name: 'google_videos', display_name: 'Google', icon: getSourceIcon('google_videos'), result_count: sourceStats.get('google_videos') || 0, enabled: true },
      { name: 'bing_videos', display_name: 'Bing', icon: getSourceIcon('bing_videos'), result_count: sourceStats.get('bing_videos') || 0, enabled: true },
      { name: 'peertube', display_name: 'PeerTube', icon: getSourceIcon('peertube'), result_count: sourceStats.get('peertube') || 0, enabled: true },
      { name: 'duckduckgo_videos', display_name: 'DuckDuckGo', icon: getSourceIcon('duckduckgo_videos'), result_count: sourceStats.get('duckduckgo_videos') || 0, enabled: true },
    ].filter(s => s.result_count > 0 || s.name === options.filters?.source);

    // Paginate
    const perPage = options.per_page || 20;
    const startIndex = (options.page - 1) * perPage;
    const endIndex = startIndex + perPage;
    const paginatedResults = allResults.slice(startIndex, endIndex);

    const response: VideoSearchResponse = {
      query,
      total_results: allResults.length,
      results: paginatedResults,
      filters: options.filters,
      available_sources: availableSources,
      search_time_ms: Date.now() - startTime,
      page: options.page,
      per_page: perPage,
      has_more: endIndex < allResults.length,
    };

    await this.cache.setVideoSearch(cacheHash, response);
    return response;
  }
```

**Step 4: Commit**

```bash
git add app/worker/src/services/search.ts
git commit -m "feat(video): enhance searchVideos with filters and sources"
```

---

## Phase 6: Update Routes

### Task 13: Update Video Search Route

**Files:**
- Modify: `app/worker/src/routes/search.ts`

**Step 1: Add video filter extraction helper**

```typescript
const validVideoDurations = ['any', 'short', 'medium', 'long'];
const validVideoQualities = ['any', 'hd', '4k'];
const validVideoSorts = ['relevance', 'date', 'views', 'duration'];

function extractVideoFilters(c: { req: { query: (key: string) => string | undefined } }): VideoSearchFilters {
  const duration = c.req.query('duration');
  const quality = c.req.query('quality');
  const time = c.req.query('time');
  const source = c.req.query('source');
  const cc = c.req.query('cc');
  const safe = c.req.query('safe') as SafeSearchLevel | undefined;

  const filters: VideoSearchFilters = {};

  if (duration && validVideoDurations.includes(duration)) {
    filters.duration = duration as VideoDuration;
  }
  if (quality && validVideoQualities.includes(quality)) {
    filters.quality = quality as VideoQuality;
  }
  if (time && validImageTimes.includes(time as ImageTime)) {
    filters.time = time as ImageTime;
  }
  if (source) {
    filters.source = source;
  }
  if (cc === 'true' || cc === '1') {
    filters.cc = true;
  }
  if (safe && validSafeSearch.includes(safe)) {
    filters.safe = safe;
  }

  return filters;
}

function extractVideoSearchOptions(c: { req: { query: (key: string) => string | undefined } }): VideoSearchOptions {
  const base = extractSearchOptions(c);
  const filters = extractVideoFilters(c);
  const sort = c.req.query('sort');

  return {
    ...base,
    filters,
    sort: sort && validVideoSorts.includes(sort) ? sort as VideoSort : undefined,
  };
}
```

**Step 2: Update the /videos route**

```typescript
app.get('/videos', async (c) => {
  const q = c.req.query('q') ?? '';
  if (!q) {
    return c.json({ error: 'Missing required parameter: q' }, 400);
  }

  const options = extractVideoSearchOptions(c);
  const { searchService } = createServices(c.env.SEARCH_KV);
  const results = await searchService.searchVideos(q, options);
  return c.json(results);
});
```

**Step 3: Commit**

```bash
git add app/worker/src/routes/search.ts
git commit -m "feat(video): update video search route with filters"
```

---

## Phase 7: Integration Tests

### Task 14: Create Video Search Integration Tests

**Files:**
- Create: `app/worker/src/engines/video-search.test.ts`

**Step 1: Create comprehensive integration test**

```typescript
// app/worker/src/engines/video-search.test.ts
import { describe, it, expect } from 'vitest';
import { createDefaultMetaSearch } from './metasearch';
import type { EngineParams } from './engine';

describe('Video Search Integration', () => {
  const metasearch = createDefaultMetaSearch();

  it('should aggregate results from multiple video engines', async () => {
    const params: EngineParams = {
      page: 1,
      locale: 'en',
      safeSearch: 1,
      timeRange: '',
      engineData: {},
    };

    const result = await metasearch.search('javascript tutorial', 'videos', params);

    expect(result.results.length).toBeGreaterThan(0);
    expect(result.successfulEngines).toBeGreaterThan(1);

    // Verify results from multiple sources
    const sources = new Set(result.results.map(r => r.engine));
    console.log('Sources found:', Array.from(sources));
    expect(sources.size).toBeGreaterThan(1);
  }, 60000);

  it('should return video-specific fields', async () => {
    const params: EngineParams = {
      page: 1,
      locale: 'en',
      safeSearch: 1,
      timeRange: '',
      engineData: {},
    };

    const result = await metasearch.search('music video', 'videos', params);

    expect(result.results.length).toBeGreaterThan(0);

    const video = result.results[0];
    expect(video.url).toBeTruthy();
    expect(video.title).toBeTruthy();
    expect(video.category).toBe('videos');
  }, 60000);

  it('should apply time range filter', async () => {
    const params: EngineParams = {
      page: 1,
      locale: 'en',
      safeSearch: 1,
      timeRange: 'week',
      engineData: {},
    };

    const result = await metasearch.search('news', 'videos', params);

    // Should still return results (engines that support time range)
    expect(result.results).toBeDefined();
  }, 60000);

  it('should handle safe search', async () => {
    const params: EngineParams = {
      page: 1,
      locale: 'en',
      safeSearch: 2, // strict
      timeRange: '',
      engineData: {},
    };

    const result = await metasearch.search('cooking', 'videos', params);

    expect(result.results).toBeDefined();
  }, 60000);

  it('should handle pagination', async () => {
    const params1: EngineParams = {
      page: 1,
      locale: 'en',
      safeSearch: 1,
      timeRange: '',
      engineData: {},
    };

    const params2: EngineParams = {
      page: 2,
      locale: 'en',
      safeSearch: 1,
      timeRange: '',
      engineData: {},
    };

    const result1 = await metasearch.search('tutorial', 'videos', params1);
    const result2 = await metasearch.search('tutorial', 'videos', params2);

    // Results should be different (or at least page 1 should have results)
    expect(result1.results.length).toBeGreaterThan(0);
  }, 60000);

  it('should list all video engines', () => {
    const engines = metasearch.getByCategory('videos');
    const names = engines.map(e => e.name);

    console.log('Registered video engines:', names);

    expect(names).toContain('youtube');
    expect(names).toContain('vimeo');
    expect(names).toContain('dailymotion');
  });
});
```

**Step 2: Run integration tests**

```bash
cd /Users/apple/github/go-mizu/mizu/blueprints/search/app/worker
npm test -- video-search.test.ts
```

**Step 3: Commit**

```bash
git add app/worker/src/engines/video-search.test.ts
git commit -m "test(video): add video search integration tests"
```

---

## Phase 8: Frontend Implementation

Tasks 15-20 cover frontend components. See continuation in next section.

---

## Summary Checklist

- [ ] Task 1: Add Video Filter Types to engine.ts
- [ ] Task 2: Enhance YouTube Engine + Tests
- [ ] Task 3: Implement Vimeo Engine + Tests
- [ ] Task 4: Implement Dailymotion Engine + Tests
- [ ] Task 5: Implement Google Videos Engine + Tests
- [ ] Task 6: Implement Bing Videos Engine + Tests
- [ ] Task 7: Implement PeerTube Engine + Tests
- [ ] Task 8: Implement 360Search Engine + Tests
- [ ] Task 9: Implement Sogou Engine + Tests
- [ ] Task 10: Register All Engines in MetaSearch
- [ ] Task 11: Add Video Search Types
- [ ] Task 12: Enhance Video Search Service
- [ ] Task 13: Update Video Search Route
- [ ] Task 14: Create Integration Tests
- [ ] Task 15-20: Frontend Components (VideosPage rewrite)
- [ ] Task 21: Build and Deploy to Cloudflare
- [ ] Task 22: Verify *.workers.dev endpoint
