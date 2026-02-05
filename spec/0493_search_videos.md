# Spec 0493: Video Search for Cloudflare Worker

## Overview

Implement a comprehensive video search feature for the Search blueprint's Cloudflare Worker deployment. The implementation will aggregate results from 9 video engines (matching SearXNG's coverage) with a pixel-perfect Google Videos UI clone.

## Goals

1. **100% Feature Parity with Google Videos** - All filters, sorting, and UI elements
2. **9 Video Engine Aggregation** - YouTube, Vimeo, Dailymotion, Google Videos, Bing Videos, PeerTube, 360Search, Sogou
3. **Full Preview Mode** - Embedded players with auto-play thumbnail previews on hover
4. **Production-Ready** - Deployable to Cloudflare Workers with *.worker.dev working
5. **Comprehensive Testing** - Unit + live integration tests for all engines

## Video Engines

### Engine Matrix

| Engine | Platform | Method | Paging | Time Range | SafeSearch | Language | Priority |
|--------|----------|--------|--------|------------|------------|----------|----------|
| youtube | YouTube | HTML Scrape | Yes | Yes | No | No | 1 |
| vimeo | Vimeo | HTML Scrape | Yes | No | No | No | 2 |
| dailymotion | Dailymotion | REST API | Yes | Yes | Yes | Yes | 3 |
| google_videos | Google | HTML Scrape | Yes | Yes | Yes | Yes | 4 |
| bing_videos | Bing | HTML Scrape | Yes | Yes | Yes | No | 5 |
| peertube | PeerTube/Federated | REST API | Yes | Yes | Yes | Yes | 6 |
| 360search | 360kan (Chinese) | JSON API | Yes | No | No | No | 7 |
| sogou | Sogou (Chinese) | JSON API | Yes | No | No | No | 8 |

### Engine Specifications

#### 1. YouTube Engine (`youtube.ts`)

```typescript
interface YouTubeEngine {
  name: 'youtube';
  baseUrl: 'https://www.youtube.com/results';
  method: 'scrape';

  // Request
  params: {
    search_query: string;
    sp?: string; // Encoded filter (time range)
  };
  cookies: ['CONSENT=YES+'];

  // Time range encoding
  timeRangeMap: {
    day: 'EgIIAg%3D%3D',    // Ag
    week: 'EgIIAw%3D%3D',   // Aw
    month: 'EgIIBA%3D%3D',  // BA
    year: 'EgIIBQ%3D%3D',   // BQ
  };

  // Parse ytInitialData JSON from HTML
  parseSelector: 'var ytInitialData = ({.+?});</script>';
}
```

#### 2. Vimeo Engine (`vimeo.ts`)

```typescript
interface VimeoEngine {
  name: 'vimeo';
  baseUrl: 'https://vimeo.com/search';
  method: 'scrape';

  params: {
    q: string;
    page?: number;
  };

  // Extract JSON from window.vimeo.config
  parseSelector: 'window.vimeo.config';
}
```

#### 3. Dailymotion Engine (`dailymotion.ts`)

```typescript
interface DailymotionEngine {
  name: 'dailymotion';
  baseUrl: 'https://api.dailymotion.com/videos';
  method: 'api';

  params: {
    search: string;
    page: number;
    limit: 10;
    family_filter: boolean;
    is_created_for_kids: boolean;
    languages?: string;
    localization?: string;
    country?: string;
    fields: 'id,title,description,duration,thumbnail_360_url,created_time,owner.screenname,views_total,embed_url,allow_embed';
  };

  // Time range via created_after parameter
  timeRangeMap: {
    day: '-1 day',
    week: '-1 week',
    month: '-1 month',
    year: '-1 year',
  };
}
```

#### 4. Google Videos Engine (`google_videos.ts`)

```typescript
interface GoogleVideosEngine {
  name: 'google_videos';
  baseUrl: 'https://www.google.com/search';
  method: 'scrape';

  params: {
    q: string;
    tbm: 'vid';
    tbs?: string; // Time/duration/quality filters
    start?: number;
    hl?: string;
    gl?: string;
    safe?: 'active' | 'off';
  };

  // tbs parameter encoding
  filters: {
    duration: { short: 'dur:s', medium: 'dur:m', long: 'dur:l' };
    time: { hour: 'qdr:h', day: 'qdr:d', week: 'qdr:w', month: 'qdr:m', year: 'qdr:y' };
    quality: { hd: 'hq:h' };
    cc: { enabled: 'cc:1' };
  };
}
```

#### 5. Bing Videos Engine (`bing_videos.ts`)

```typescript
interface BingVideosEngine {
  name: 'bing_videos';
  baseUrl: 'https://www.bing.com/videos/asyncv2';
  method: 'scrape';

  params: {
    q: string;
    async: 'content';
    first: number;
    count: 35;
    form?: 'VRFLTR';
    qft?: string; // Time filter
  };

  // Time range in minutes
  timeRangeMap: {
    day: 'videoage-lt1440',
    week: 'videoage-lt10080',
    month: 'videoage-lt43200',
    year: 'videoage-lt525600',
  };
}
```

#### 6. PeerTube Engine (`peertube.ts`)

```typescript
interface PeerTubeEngine {
  name: 'peertube';
  baseUrl: 'https://sepiasearch.org/api/v1/search/videos'; // Sepia Search (federated)
  method: 'api';

  params: {
    search: string;
    start: number;
    count: 15;
    sort: '-match';
    nsfw: 'both' | 'false';
    languageOneOf?: string[];
    boostLanguages?: string[];
    startDate?: string; // ISO format for time range
  };
}
```

#### 7. 360Search Videos Engine (`360search_videos.ts`)

```typescript
interface 360SearchEngine {
  name: '360search';
  baseUrl: 'https://tv.360kan.com/v1/video/list';
  method: 'api';

  params: {
    q: string;
    start: number;
    count: 10;
  };
}
```

#### 8. Sogou Videos Engine (`sogou_videos.ts`)

```typescript
interface SogouEngine {
  name: 'sogou';
  baseUrl: 'https://v.sogou.com/api/video/shortVideoV2';
  method: 'api';

  params: {
    query: string;
    page: number;
    pagesize: 10;
  };
}
```

## API Specification

### Video Search Endpoint

```
GET /api/search/videos
```

#### Request Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `q` | string | required | Search query |
| `page` | number | 1 | Page number |
| `per_page` | number | 20 | Results per page (max 50) |
| `duration` | string | any | `short` (<4min), `medium` (4-20min), `long` (>20min) |
| `time` | string | any | `hour`, `day`, `week`, `month`, `year` |
| `quality` | string | any | `hd`, `4k` |
| `source` | string | all | Engine name to filter by |
| `safe` | string | moderate | `off`, `moderate`, `strict` |
| `sort` | string | relevance | `relevance`, `date`, `views`, `duration` |
| `lang` | string | en | Language code |
| `region` | string | | Region code |
| `cc` | boolean | false | Closed captions only |

#### Response Format

```typescript
interface VideoSearchResponse {
  query: string;
  total_results: number;
  results: VideoResult[];
  filters: AppliedFilters;
  available_sources: SourceInfo[];
  search_time_ms: number;
  page: number;
  per_page: number;
  has_more: boolean;
}

interface VideoResult {
  id: string;
  url: string;
  title: string;
  description: string;
  thumbnail_url: string;
  thumbnail_width?: number;
  thumbnail_height?: number;
  duration: string;           // "14:32" or "1:23:45"
  duration_seconds: number;
  channel: string;
  channel_url?: string;
  views?: number;
  views_formatted?: string;   // "1.2M views"
  published_at?: string;      // ISO date
  published_formatted?: string; // "2 days ago"
  embed_url?: string;
  embed_html?: string;
  source: string;             // Engine name
  source_icon?: string;       // Platform favicon
  quality?: string;           // "HD", "4K"
  has_cc?: boolean;
  is_live?: boolean;
  score: number;
  engines: string[];          // All engines that returned this result
}

interface AppliedFilters {
  duration?: string;
  time?: string;
  quality?: string;
  source?: string;
  safe?: string;
  sort?: string;
  cc?: boolean;
}

interface SourceInfo {
  name: string;
  display_name: string;
  icon: string;
  result_count: number;
  enabled: boolean;
}
```

## Frontend Specification

### Page Layout (Google Videos Clone)

```
┌─────────────────────────────────────────────────────────────────────────┐
│ [Logo]  [═══════════════ Search Box ═══════════════] [🔍] [Settings]   │
├─────────────────────────────────────────────────────────────────────────┤
│ [All] [Images] [Videos•] [News] [Maps] [More▼]     [Tools▼] [SafeSearch]│
├─────────────────────────────────────────────────────────────────────────┤
│ Duration: [Any▼]  Time: [Any▼]  Quality: [Any▼]  Source: [All▼]  [CC]  │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  ┌──────────────────┐  Title of Video                                  │
│  │                  │  source.com · Channel Name                        │
│  │   [▶ THUMBNAIL]  │  2 days ago · 1.2M views                         │
│  │      14:32       │  Description text that can span multiple lines   │
│  └──────────────────┘  and provides context about the video content... │
│                                                                         │
│  ┌──────────────────┐  Another Video Title Here                        │
│  │                  │  youtube.com · Creator Channel                    │
│  │   [▶ THUMBNAIL]  │  1 week ago · 500K views                         │
│  │       8:45       │  Another description for this video result...    │
│  └──────────────────┘                                                   │
│                                                                         │
│  ... more results ...                                                   │
│                                                                         │
│                        [Load More]                                      │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

### Component Specifications

#### VideoCard Component

```typescript
interface VideoCardProps {
  video: VideoResult;
  onPlay?: (video: VideoResult) => void;
  onHover?: (video: VideoResult) => void;
}
```

**Thumbnail Features:**
- 16:9 aspect ratio (320x180 or 480x270)
- Duration badge (bottom-right, black/80 bg, white text)
- Quality badge (top-right, "HD" or "4K" if applicable)
- Live badge (top-left, red "LIVE" if is_live)
- Play button overlay on hover (centered, semi-transparent)
- **Auto-preview on hover** (3-second delay, then animate/preview if supported)
- Source platform icon (bottom-left)

**Card Layout:**
- Thumbnail: 320px width, 180px height
- Title: 18px, #1a0dab, max 2 lines, truncate with ellipsis
- Meta line 1: Source domain · Channel name (14px, #70757a)
- Meta line 2: Time ago · View count (14px, #70757a)
- Description: 14px, #4d5156, max 2 lines

#### VideoFilters Component

```typescript
interface VideoFiltersProps {
  filters: AppliedFilters;
  onChange: (filters: AppliedFilters) => void;
  sources: SourceInfo[];
}
```

**Filter Dropdowns (Google-style chips):**
- Duration: Any duration, Short (<4 min), Medium (4-20 min), Long (>20 min)
- Time: Any time, Past hour, Past 24 hours, Past week, Past month, Past year
- Quality: Any quality, HD, 4K
- Source: All sources, YouTube, Vimeo, Dailymotion, etc.
- Closed captions toggle

#### VideoPlayer Modal

```typescript
interface VideoPlayerProps {
  video: VideoResult;
  isOpen: boolean;
  onClose: () => void;
}
```

**Features:**
- Modal overlay with embedded iframe player
- Support for YouTube, Vimeo, Dailymotion, PeerTube embeds
- Fallback to external link for unsupported platforms
- Keyboard navigation (Escape to close)

### Styling (Google Colors)

```css
:root {
  /* Text */
  --text-primary: #202124;
  --text-secondary: #4d5156;
  --text-tertiary: #70757a;
  --text-link: #1a0dab;
  --text-link-visited: #681da8;

  /* Backgrounds */
  --bg-primary: #ffffff;
  --bg-secondary: #f8f9fa;
  --bg-hover: #f1f3f4;
  --bg-active: #e8f0fe;

  /* Borders */
  --border-light: #dadce0;
  --border-focus: #1a73e8;

  /* Accents */
  --accent-blue: #1a73e8;
  --accent-red: #ea4335;
  --accent-green: #34a853;

  /* Video-specific */
  --duration-bg: rgba(0, 0, 0, 0.8);
  --live-badge: #cc0000;
  --hd-badge: #065fd4;
}
```

## File Structure

```
app/worker/src/
├── engines/
│   ├── engine.ts              # Base types and utilities
│   ├── metasearch.ts          # Aggregation logic
│   ├── youtube.ts             # YouTube (existing, enhance)
│   ├── vimeo.ts               # NEW
│   ├── dailymotion.ts         # NEW
│   ├── google_videos.ts       # NEW
│   ├── bing_videos.ts         # NEW
│   ├── peertube.ts            # NEW
│   ├── 360search_videos.ts    # NEW
│   └── sogou_videos.ts        # NEW
├── services/
│   └── search.ts              # Enhance searchVideos method
├── routes/
│   └── search.ts              # Enhance /videos endpoint
├── types.ts                   # Add VideoSearchResponse types
└── __tests__/
    ├── engines/
    │   ├── youtube.test.ts
    │   ├── vimeo.test.ts
    │   ├── dailymotion.test.ts
    │   ├── google_videos.test.ts
    │   ├── bing_videos.test.ts
    │   ├── peertube.test.ts
    │   ├── 360search_videos.test.ts
    │   └── sogou_videos.test.ts
    └── integration/
        └── video_search.test.ts

app/frontend/src/
├── pages/
│   └── VideosPage.tsx         # Complete rewrite
├── components/
│   ├── video/
│   │   ├── VideoCard.tsx      # NEW
│   │   ├── VideoFilters.tsx   # NEW
│   │   ├── VideoPlayer.tsx    # NEW
│   │   ├── VideoGrid.tsx      # NEW
│   │   └── VideoPreview.tsx   # NEW (hover preview)
│   └── ...
├── hooks/
│   └── useVideoSearch.ts      # NEW
├── api/
│   └── search.ts              # Enhance searchVideos
└── types/
    └── video.ts               # Video-specific types
```

## Testing Strategy

### Unit Tests (per engine)

Each engine must have tests that:
1. **Build correct request URL** with all parameter combinations
2. **Parse real response data** (captured samples, not mocks)
3. **Handle edge cases**: empty results, malformed data, rate limits
4. **Extract all fields correctly**: duration parsing, view count formatting, etc.

```typescript
// Example: youtube.test.ts
describe('YouTubeEngine', () => {
  it('should build correct search URL', () => {
    const engine = new YouTubeEngine();
    const config = engine.buildRequest('test query', { timeRange: 'week' });
    expect(config.url).toContain('search_query=test+query');
    expect(config.url).toContain('sp=EgIIAw');
  });

  it('should parse video results from real response', async () => {
    const engine = new YouTubeEngine();
    const results = await engine.search('typescript tutorial');

    expect(results.results.length).toBeGreaterThan(0);
    expect(results.results[0]).toMatchObject({
      url: expect.stringContaining('youtube.com/watch'),
      title: expect.any(String),
      duration: expect.stringMatching(/^\d+:\d{2}(:\d{2})?$/),
      channel: expect.any(String),
      thumbnail_url: expect.stringContaining('ytimg.com'),
    });
  });
});
```

### Integration Tests

Test the complete flow with live API calls:

```typescript
// video_search.test.ts
describe('Video Search Integration', () => {
  it('should aggregate results from multiple engines', async () => {
    const response = await searchVideos('javascript', { per_page: 20 });

    expect(response.results.length).toBeGreaterThan(0);
    expect(response.available_sources.length).toBeGreaterThan(1);

    // Verify results from multiple sources
    const sources = new Set(response.results.map(r => r.source));
    expect(sources.size).toBeGreaterThan(1);
  });

  it('should filter by duration', async () => {
    const response = await searchVideos('tutorial', { duration: 'short' });

    for (const video of response.results) {
      expect(video.duration_seconds).toBeLessThan(240); // 4 minutes
    }
  });

  it('should filter by time range', async () => {
    const response = await searchVideos('news', { time: 'day' });
    const oneDayAgo = Date.now() - 24 * 60 * 60 * 1000;

    for (const video of response.results) {
      if (video.published_at) {
        expect(new Date(video.published_at).getTime()).toBeGreaterThan(oneDayAgo);
      }
    }
  });

  it('should filter by source', async () => {
    const response = await searchVideos('music', { source: 'youtube' });

    for (const video of response.results) {
      expect(video.source).toBe('youtube');
    }
  });
});
```

### Test Configuration

```typescript
// vitest.config.ts
export default defineConfig({
  test: {
    testTimeout: 30000, // 30s for live API calls
    hookTimeout: 30000,
    include: ['src/**/*.test.ts'],
    env: {
      LIVE_TESTS: 'true', // Enable live API tests
    },
  },
});
```

## Deployment

### Wrangler Configuration

```toml
# wrangler.toml
name = "mizu-search"
main = "src/index.ts"
compatibility_date = "2024-01-01"

[vars]
ENVIRONMENT = "production"

[[kv_namespaces]]
binding = "SEARCH_KV"
id = "xxx"
preview_id = "yyy"

[site]
bucket = "./frontend/dist"
```

### Build & Deploy Commands

```bash
# Build frontend
cd app/frontend && npm run build

# Copy to worker
cp -r dist ../worker/frontend/

# Deploy worker
cd ../worker && npx wrangler deploy

# Verify deployment
curl https://mizu-search.YOUR_SUBDOMAIN.workers.dev/api/search/videos?q=test
```

### Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `SEARCH_KV` | Yes | KV namespace binding |
| `ENVIRONMENT` | No | `development` or `production` |
| `PEERTUBE_INSTANCE` | No | Custom PeerTube instance URL |

## Implementation Phases

### Phase 1: Engine Implementation (Core)
1. Enhance existing YouTube engine with all filters
2. Implement Vimeo engine
3. Implement Dailymotion engine
4. Implement Google Videos engine
5. Implement Bing Videos engine

### Phase 2: Engine Implementation (Extended)
6. Implement PeerTube engine
7. Implement 360Search engine
8. Implement Sogou engine
9. Update metasearch aggregation for video category

### Phase 3: API Enhancement
10. Enhance `/api/search/videos` with all filter parameters
11. Implement video-specific response format
12. Add source filtering and aggregation stats

### Phase 4: Frontend Rewrite
13. Create VideoCard component (Google-style)
14. Create VideoFilters component
15. Create VideoPlayer modal
16. Create VideoPreview (hover auto-play)
17. Rewrite VideosPage with new components

### Phase 5: Testing
18. Write unit tests for all engines (live API)
19. Write integration tests for video search
20. Manual testing of all filter combinations

### Phase 6: Deployment
21. Build and bundle frontend
22. Deploy to Cloudflare Workers
23. Verify *.workers.dev endpoint works
24. Performance testing and optimization

## Success Criteria

1. **All 9 engines return results** for common queries
2. **Filters work correctly** across all supported engines
3. **UI matches Google Videos** layout and styling
4. **Hover preview works** with smooth animation
5. **Embed players work** for YouTube, Vimeo, Dailymotion, PeerTube
6. **All tests pass** with live API calls
7. **Deployed and working** on *.workers.dev

## References

- [SearXNG Video Engines](https://github.com/searxng/searxng/tree/master/searx/engines)
- [Google Video Search Filters](https://serpapi.com/blog/filtering-google-images-and-google-videos-results/)
- [YouTube Search Filters](https://support.google.com/youtube/answer/111997)
- [Cloudflare Workers Docs](https://developers.cloudflare.com/workers/)
- [Hono Framework](https://hono.dev/)
