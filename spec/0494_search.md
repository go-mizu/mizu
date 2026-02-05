# 0494: Mizu Search Worker - Complete Google-like Search Engine

## Summary

A production-grade metasearch engine deployed as a Cloudflare Worker with Hono. Implements 100% of Google Search features with a pixel-perfect UI, 50+ search engines (ported from SearXNG), and comprehensive testing. Deployable to `*.workers.dev`.

## Goals

1. **100% Google Feature Parity** - Search results, images, videos, news, instant answers, knowledge panels, related searches, "People also ask"
2. **50+ Search Engines** - All major engines from SearXNG ported to TypeScript
3. **Pixel-Perfect Google UI** - Identical visual appearance to Google Search
4. **Production Quality** - Full test coverage with real API validation
5. **Edge Deployment** - Optimized for Cloudflare Workers

## Architecture

```
app/worker/
├── src/
│   ├── index.ts                    # Hono app entry
│   ├── types.ts                    # All TypeScript types
│   ├── routes/
│   │   ├── search.ts               # GET /api/search[/images/videos/news/maps/books]
│   │   ├── suggest.ts              # GET /api/suggest, /api/suggest/trending
│   │   ├── instant.ts              # Instant answers (calculator, weather, etc.)
│   │   ├── knowledge.ts            # Knowledge panel
│   │   ├── preferences.ts          # User preferences (boost/block domains)
│   │   ├── lenses.ts               # Custom search filters
│   │   ├── history.ts              # Search history
│   │   ├── settings.ts             # User settings
│   │   ├── bangs.ts                # Bang shortcuts (!g, !w, etc.)
│   │   ├── widgets.ts              # Cheatsheets, related searches
│   │   ├── news.ts                 # News home/categories
│   │   └── health.ts               # Health check
│   ├── engines/
│   │   ├── engine.ts               # Base engine interface
│   │   ├── metasearch.ts           # Orchestrator
│   │   │
│   │   │ # ===== WEB SEARCH ENGINES =====
│   │   ├── google.ts               # Google Web + Images
│   │   ├── bing.ts                 # Bing Web + Images + News + Videos
│   │   ├── duckduckgo.ts           # DDG Images, Videos, News (JSON APIs)
│   │   ├── brave.ts                # Brave Search
│   │   ├── yahoo.ts                # Yahoo Search
│   │   ├── yandex.ts               # Yandex (RU)
│   │   ├── baidu.ts                # Baidu (CN)
│   │   ├── qwant.ts                # Qwant (EU, disabled - CAPTCHA)
│   │   ├── startpage.ts            # Startpage (Google proxy)
│   │   ├── mojeek.ts               # Mojeek (independent index)
│   │   ├── presearch.ts            # Presearch (decentralized)
│   │   │
│   │   │ # ===== VIDEO ENGINES =====
│   │   ├── youtube.ts              # YouTube (ytInitialData scraping)
│   │   ├── vimeo.ts                # Vimeo API
│   │   ├── dailymotion.ts          # Dailymotion API
│   │   ├── peertube.ts             # PeerTube (federated)
│   │   ├── rumble.ts               # Rumble
│   │   ├── odysee.ts               # Odysee/LBRY
│   │   ├── bilibili.ts             # Bilibili (CN)
│   │   ├── niconico.ts             # NicoNico (JP)
│   │   │
│   │   │ # ===== IMAGE ENGINES =====
│   │   ├── google-images.ts        # Google Images (JSON)
│   │   ├── bing-images.ts          # Bing Images
│   │   ├── flickr.ts               # Flickr API
│   │   ├── unsplash.ts             # Unsplash API
│   │   ├── pixabay.ts              # Pixabay API
│   │   ├── deviantart.ts           # DeviantArt
│   │   ├── imgur.ts                # Imgur
│   │   │
│   │   │ # ===== NEWS ENGINES =====
│   │   ├── google-news.ts          # Google News
│   │   ├── bing-news.ts            # Bing News
│   │   ├── duckduckgo-news.ts      # DDG News
│   │   ├── yahoo-news.ts           # Yahoo News
│   │   ├── reuters.ts              # Reuters
│   │   │
│   │   │ # ===== REFERENCE/ACADEMIC =====
│   │   ├── wikipedia.ts            # Wikipedia (26+ languages)
│   │   ├── wikidata.ts             # Wikidata entities
│   │   ├── arxiv.ts                # arXiv papers
│   │   ├── pubmed.ts               # PubMed medical
│   │   ├── semantic-scholar.ts     # Semantic Scholar
│   │   ├── crossref.ts             # Crossref DOI
│   │   ├── openlibrary.ts          # Open Library books
│   │   │
│   │   │ # ===== CODE/IT =====
│   │   ├── github.ts               # GitHub repos
│   │   ├── github-code.ts          # GitHub code search
│   │   ├── gitlab.ts               # GitLab
│   │   ├── stackoverflow.ts        # Stack Overflow
│   │   ├── npm.ts                  # NPM packages
│   │   ├── pypi.ts                 # PyPI packages
│   │   ├── crates.ts               # Rust crates.io
│   │   ├── pkg-go-dev.ts           # Go packages
│   │   │
│   │   │ # ===== SOCIAL =====
│   │   ├── reddit.ts               # Reddit
│   │   ├── hackernews.ts           # Hacker News
│   │   ├── mastodon.ts             # Mastodon (federated)
│   │   ├── lemmy.ts                # Lemmy (federated)
│   │   │
│   │   │ # ===== MUSIC =====
│   │   ├── soundcloud.ts           # SoundCloud
│   │   ├── bandcamp.ts             # Bandcamp
│   │   ├── genius.ts               # Genius lyrics
│   │   │
│   │   │ # ===== TORRENTS/FILES =====
│   │   ├── piratebay.ts            # The Pirate Bay
│   │   ├── 1337x.ts                # 1337x
│   │   ├── nyaa.ts                 # Nyaa (anime)
│   │   │
│   │   │ # ===== MAPS =====
│   │   ├── openstreetmap.ts        # OpenStreetMap
│   │   ├── photon.ts               # Photon geocoder
│   │   │
│   │   │ # ===== SHOPPING =====
│   │   ├── ebay.ts                 # eBay
│   │   ├── amazon.ts               # Amazon (disabled - CAPTCHA)
│   │   │
│   │   │ # ===== TRANSLATION =====
│   │   ├── lingva.ts               # Lingva Translate
│   │   │
│   │   │ # ===== MOVIES/TV =====
│   │   ├── imdb.ts                 # IMDb
│   │   ├── rottentomatoes.ts       # Rotten Tomatoes
│   │
│   ├── services/
│   │   ├── search.ts               # Search orchestration + caching
│   │   ├── instant.ts              # Calculator, currency, weather, etc.
│   │   ├── suggest.ts              # Autocomplete
│   │   ├── bang.ts                 # Bang parsing
│   │   └── knowledge.ts            # Knowledge panels
│   │
│   ├── store/
│   │   ├── kv.ts                   # Cloudflare KV adapter
│   │   ├── cache.ts                # Search result caching
│   │   └── news-store.ts           # News category caching
│   │
│   ├── middleware/
│   │   ├── cors.ts                 # CORS headers
│   │   ├── timing.ts               # Server-Timing
│   │   └── session.ts              # Session handling
│   │
│   └── lib/
│       ├── html-parser.ts          # HTML parsing utilities
│       ├── xml-parser.ts           # XML parsing (RSS/Atom)
│       └── utils.ts                # URL helpers, sanitization
│
├── frontend/                       # Vanilla TS + TailwindCSS
│   ├── src/
│   │   ├── app.ts                  # SPA router
│   │   ├── api.ts                  # API client
│   │   ├── pages/
│   │   │   ├── home.ts             # Landing page
│   │   │   ├── search.ts           # Web results
│   │   │   ├── images.ts           # Image grid
│   │   │   ├── videos.ts           # Video results
│   │   │   ├── news.ts             # News results
│   │   │   ├── news-home.ts        # News homepage
│   │   │   ├── maps.ts             # Map results
│   │   │   ├── settings.ts         # Settings
│   │   │   └── history.ts          # History
│   │   ├── components/
│   │   │   ├── search-box.ts       # Search input
│   │   │   ├── search-result.ts    # Result card
│   │   │   ├── instant-answer.ts   # Instant answers
│   │   │   ├── knowledge-panel.ts  # Knowledge panel
│   │   │   ├── people-also-ask.ts  # PAA accordion
│   │   │   ├── image-result.ts     # Image tile
│   │   │   ├── video-result.ts     # Video card
│   │   │   ├── news-result.ts      # News card
│   │   │   ├── pagination.ts       # Page nav
│   │   │   ├── tabs.ts             # Category tabs
│   │   │   └── filters.ts          # Time/region filters
│   │   ├── styles/
│   │   │   └── main.css            # Google-identical styles
│   │   └── lib/
│   │       ├── router.ts           # SPA router
│   │       └── state.ts            # State management
│   ├── index.html
│   ├── tailwind.config.ts
│   └── vite.config.ts
│
├── tests/                          # Full test suite
│   ├── engines/                    # Engine unit tests
│   ├── services/                   # Service tests
│   ├── routes/                     # API route tests
│   └── e2e/                        # End-to-end tests
│
├── wrangler.toml
├── package.json
├── tsconfig.json
└── vitest.config.ts
```

## Complete Engine List (50+ engines)

### Web Search (11 engines)
| Engine | Shortcut | Method | Status |
|--------|----------|--------|--------|
| Google | `g` | HTML (GSA UA + arc_id) | ✅ Enabled |
| Bing | `b` | HTML | ✅ Enabled |
| DuckDuckGo | `ddg` | HTML (VQD) | ⚠️ CAPTCHA issues |
| Brave | `br` | HTML | ✅ Enabled |
| Yahoo | `y` | HTML | ✅ Enabled |
| Yandex | `ya` | HTML | ✅ Enabled |
| Baidu | `bd` | JSON | ✅ Enabled |
| Qwant | `qw` | JSON | ❌ CAPTCHA |
| Startpage | `sp` | HTML | ✅ Enabled |
| Mojeek | `mj` | HTML | ✅ Enabled |
| Presearch | `ps` | JSON | ✅ Enabled |

### Image Search (9 engines)
| Engine | Shortcut | Method | Status |
|--------|----------|--------|--------|
| Google Images | `gi` | JSON | ✅ Enabled |
| Bing Images | `bi` | JSON | ✅ Enabled |
| DuckDuckGo Images | `ddi` | JSON (VQD) | ✅ Enabled |
| Flickr | `fl` | JSON API | ✅ Enabled |
| Unsplash | `un` | JSON API | ✅ Enabled |
| Pixabay | `px` | JSON API | ✅ Enabled |
| DeviantArt | `da` | JSON | ✅ Enabled |
| Imgur | `im` | JSON API | ✅ Enabled |
| Wikimedia Commons | `wc` | MediaWiki API | ✅ Enabled |

### Video Search (10 engines)
| Engine | Shortcut | Method | Status |
|--------|----------|--------|--------|
| YouTube | `yt` | HTML (ytInitialData) | ✅ Enabled |
| Bing Videos | `bv` | JSON | ✅ Enabled |
| DuckDuckGo Videos | `ddv` | JSON (VQD) | ✅ Enabled |
| Vimeo | `vm` | JSON API | ✅ Enabled |
| Dailymotion | `dm` | JSON API | ✅ Enabled |
| PeerTube | `pt` | JSON API | ✅ Enabled |
| Rumble | `rb` | HTML | ✅ Enabled |
| Odysee | `od` | JSON API | ✅ Enabled |
| Bilibili | `bl` | JSON | ✅ Enabled |
| NicoNico | `nn` | JSON | ✅ Enabled |

### News Search (6 engines)
| Engine | Shortcut | Method | Status |
|--------|----------|--------|--------|
| Google News | `gn` | RSS | ✅ Enabled |
| Bing News | `bn` | HTML | ✅ Enabled |
| DuckDuckGo News | `ddn` | JSON (VQD) | ✅ Enabled |
| Yahoo News | `yn` | HTML | ✅ Enabled |
| Reuters | `rt` | HTML | ✅ Enabled |
| Hacker News | `hn` | JSON API | ✅ Enabled |

### Academic/Reference (8 engines)
| Engine | Shortcut | Method | Status |
|--------|----------|--------|--------|
| Wikipedia | `w` | MediaWiki API | ✅ Enabled |
| Wikidata | `wd` | SPARQL | ✅ Enabled |
| arXiv | `arx` | XML (Atom) | ✅ Enabled |
| PubMed | `pm` | XML (E-utils) | ✅ Enabled |
| Semantic Scholar | `ss` | JSON API | ✅ Enabled |
| Crossref | `cr` | JSON API | ✅ Enabled |
| Open Library | `ol` | JSON API | ✅ Enabled |
| Wolfram Alpha | `wa` | JSON API | 🔑 Needs key |

### Code/IT (8 engines)
| Engine | Shortcut | Method | Status |
|--------|----------|--------|--------|
| GitHub | `gh` | REST API | ✅ Enabled |
| GitHub Code | `ghc` | REST API | 🔑 Needs auth |
| GitLab | `gl` | REST API | ✅ Enabled |
| Stack Overflow | `so` | JSON API | ✅ Enabled |
| NPM | `npm` | JSON API | ✅ Enabled |
| PyPI | `pypi` | JSON API | ✅ Enabled |
| crates.io | `crate` | JSON API | ✅ Enabled |
| pkg.go.dev | `go` | HTML | ✅ Enabled |

### Social (4 engines)
| Engine | Shortcut | Method | Status |
|--------|----------|--------|--------|
| Reddit | `re` | JSON API | ✅ Enabled |
| Hacker News | `hn` | JSON API | ✅ Enabled |
| Mastodon | `mst` | JSON API | ✅ Enabled |
| Lemmy | `lem` | JSON API | ✅ Enabled |

### Music (3 engines)
| Engine | Shortcut | Method | Status |
|--------|----------|--------|--------|
| SoundCloud | `sc` | JSON API | ✅ Enabled |
| Bandcamp | `bc` | HTML | ✅ Enabled |
| Genius | `gen` | JSON API | ✅ Enabled |

### Files/Torrents (3 engines)
| Engine | Shortcut | Method | Status |
|--------|----------|--------|--------|
| 1337x | `1337` | HTML | ✅ Enabled |
| The Pirate Bay | `tpb` | HTML | ✅ Enabled |
| Nyaa | `nyaa` | HTML | ✅ Enabled |

### Maps (2 engines)
| Engine | Shortcut | Method | Status |
|--------|----------|--------|--------|
| OpenStreetMap | `osm` | JSON API | ✅ Enabled |
| Photon | `pho` | JSON API | ✅ Enabled |

### Other (4 engines)
| Engine | Shortcut | Method | Status |
|--------|----------|--------|--------|
| IMDb | `imdb` | HTML | ✅ Enabled |
| Rotten Tomatoes | `rt` | HTML | ✅ Enabled |
| eBay | `ebay` | HTML | ✅ Enabled |
| Lingva Translate | `tr` | JSON API | ✅ Enabled |

## API Specification

### Search Endpoints

```
GET /api/search?q=<query>&category=general&page=1&per_page=10&time=&region=&lang=&safe=moderate
GET /api/search/images?q=<query>&page=1&size=&color=&type=&aspect=&time=&rights=
GET /api/search/videos?q=<query>&page=1&duration=&time=
GET /api/search/news?q=<query>&page=1&time=&source=
GET /api/search/maps?q=<query>&lat=&lon=&zoom=
GET /api/search/music?q=<query>&page=1
GET /api/search/files?q=<query>&page=1&type=
GET /api/search/science?q=<query>&page=1
GET /api/search/it?q=<query>&page=1&type=
GET /api/search/social?q=<query>&page=1&platform=
```

### Instant Answers

```
GET /api/instant/calculate?q=<expr>
GET /api/instant/convert?q=<unit conversion>
GET /api/instant/currency?q=<currency conversion>
GET /api/instant/weather?q=<location>
GET /api/instant/define?q=<word>
GET /api/instant/time?q=<location>
GET /api/instant/translate?q=<text>&from=&to=
GET /api/instant/ip?q=my ip
GET /api/instant/color?q=#hex
```

### Suggestions

```
GET /api/suggest?q=<prefix>
GET /api/suggest/trending
GET /api/suggest/related?q=<query>
```

### User Data

```
GET/PUT    /api/settings
GET/DELETE /api/history
GET/POST/DELETE /api/preferences
GET/POST/PUT/DELETE /api/lenses
GET/POST/DELETE /api/bangs
GET /api/bangs/parse?q=<query>
```

### Knowledge

```
GET /api/knowledge/:query
GET /api/people-also-ask?q=<query>
GET /api/related?q=<query>
```

### Widgets

```
GET /api/widgets
PUT /api/widgets
GET /api/cheatsheet/:language
GET /api/cheatsheets
```

## Type Definitions

```typescript
// ===== Search Results =====

interface SearchResult {
  id: string;
  url: string;
  title: string;
  snippet: string;
  domain: string;
  favicon?: string;
  thumbnail?: { url: string; width?: number; height?: number };
  published?: string;
  score: number;
  highlights?: string[];
  sitelinks?: { title: string; url: string }[];
  engines: string[];
}

interface ImageResult {
  id: string;
  url: string;
  thumbnail_url: string;
  title: string;
  source_url: string;
  source_domain: string;
  width: number;
  height: number;
  file_size?: number;
  format?: string;
  engines: string[];
}

interface VideoResult {
  id: string;
  url: string;
  thumbnail_url: string;
  title: string;
  description: string;
  duration_seconds?: number;
  duration_formatted?: string;
  channel: string;
  channel_url?: string;
  views?: number;
  published_at?: string;
  embed_url?: string;
  platform: string;
  engines: string[];
}

interface NewsResult {
  id: string;
  url: string;
  title: string;
  snippet: string;
  source: string;
  source_url?: string;
  image_url?: string;
  published_at: string;
  author?: string;
  engines: string[];
}

interface MapResult {
  id: string;
  name: string;
  address: string;
  lat: number;
  lon: number;
  type: string;
  osm_id?: string;
  osm_type?: string;
  boundingbox?: [number, number, number, number];
}

interface MusicResult {
  id: string;
  url: string;
  title: string;
  artist: string;
  album?: string;
  duration_seconds?: number;
  thumbnail_url?: string;
  stream_url?: string;
  platform: string;
  engines: string[];
}

interface FileResult {
  id: string;
  url: string;
  title: string;
  size_bytes?: number;
  seeders?: number;
  leechers?: number;
  magnet?: string;
  source: string;
  engines: string[];
}

interface ScienceResult {
  id: string;
  url: string;
  title: string;
  abstract: string;
  authors: string[];
  published_at?: string;
  journal?: string;
  doi?: string;
  citations?: number;
  pdf_url?: string;
  source: string;
  engines: string[];
}

// ===== Search Response =====

interface SearchResponse {
  query: string;
  corrected_query?: string;
  category: string;
  total_results: number;
  results: SearchResult[];
  images?: ImageResult[];
  videos?: VideoResult[];
  news?: NewsResult[];
  suggestions?: string[];
  related_searches?: string[];
  instant_answer?: InstantAnswer;
  knowledge_panel?: KnowledgePanel;
  people_also_ask?: PeopleAlsoAsk[];
  widgets?: Widget[];
  search_time_ms: number;
  page: number;
  per_page: number;
  has_more: boolean;
  engines_used: string[];
  // Bang handling
  redirect?: string;
  bang?: { name: string; trigger: string };
}

// ===== Instant Answers =====

interface InstantAnswer {
  type: 'calculator' | 'currency' | 'weather' | 'definition' | 'time' |
        'unit' | 'translate' | 'ip' | 'color' | 'stock' | 'sports';
  query: string;
  result: string;
  data?: Record<string, any>;
}

interface CalculatorAnswer extends InstantAnswer {
  type: 'calculator';
  data: {
    expression: string;
    result: number;
  };
}

interface WeatherAnswer extends InstantAnswer {
  type: 'weather';
  data: {
    location: string;
    temperature_c: number;
    temperature_f: number;
    condition: string;
    humidity: number;
    wind_kph: number;
    wind_direction: string;
    feels_like_c: number;
    feels_like_f: number;
    icon: string;
    forecast?: WeatherForecast[];
  };
}

interface WeatherForecast {
  date: string;
  high_c: number;
  low_c: number;
  condition: string;
  icon: string;
}

interface CurrencyAnswer extends InstantAnswer {
  type: 'currency';
  data: {
    from_amount: number;
    from_currency: string;
    to_amount: number;
    to_currency: string;
    rate: number;
    updated_at: string;
  };
}

interface DefinitionAnswer extends InstantAnswer {
  type: 'definition';
  data: {
    word: string;
    phonetic?: string;
    audio_url?: string;
    meanings: {
      part_of_speech: string;
      definitions: {
        definition: string;
        example?: string;
        synonyms?: string[];
        antonyms?: string[];
      }[];
    }[];
  };
}

interface TimeAnswer extends InstantAnswer {
  type: 'time';
  data: {
    location: string;
    timezone: string;
    time: string;
    date: string;
    utc_offset: string;
    is_dst: boolean;
  };
}

interface TranslateAnswer extends InstantAnswer {
  type: 'translate';
  data: {
    source_lang: string;
    target_lang: string;
    source_text: string;
    translated_text: string;
    detected_lang?: string;
  };
}

// ===== Knowledge Panel =====

interface KnowledgePanel {
  title: string;
  subtitle?: string;
  description: string;
  image?: string;
  facts: { label: string; value: string; url?: string }[];
  links: { title: string; url: string; icon?: string }[];
  source: string;
  source_url?: string;
  related?: { title: string; image?: string; url: string }[];
}

// ===== People Also Ask =====

interface PeopleAlsoAsk {
  question: string;
  answer: string;
  source_url?: string;
  source_title?: string;
}

// ===== User Data =====

interface SearchSettings {
  safe_search: 'off' | 'moderate' | 'strict';
  results_per_page: number;
  region: string;
  language: string;
  theme: 'light' | 'dark' | 'system';
  open_in_new_tab: boolean;
  show_thumbnails: boolean;
  default_category: string;
  engines_enabled: Record<string, boolean>;
  instant_answers_enabled: boolean;
}

interface UserPreference {
  id: string;
  domain: string;
  action: 'boost' | 'lower' | 'block';
  level: number;
  created_at: string;
}

interface SearchLens {
  id: string;
  name: string;
  description?: string;
  include_domains?: string[];
  exclude_domains?: string[];
  include_keywords?: string[];
  exclude_keywords?: string[];
  region?: string;
  file_type?: string;
  engines?: string[];
  is_public: boolean;
  created_at: string;
}

interface SearchHistory {
  id: string;
  query: string;
  category: string;
  results_count: number;
  clicked_url?: string;
  searched_at: string;
}

interface Bang {
  id: string;
  trigger: string;
  name: string;
  url_template: string;
  category: string;
  icon?: string;
  is_default: boolean;
}

// ===== Engine Interface =====

interface Engine {
  name: string;
  shortcut: string;
  categories: Category[];
  supportsPaging: boolean;
  supportsTimeRange: boolean;
  supportsSafeSearch: boolean;
  supportsLanguage: boolean;
  maxPage: number;
  timeout: number;
  weight: number;
  disabled: boolean;
}

interface OnlineEngine extends Engine {
  buildRequest(query: string, params: EngineParams): RequestConfig;
  parseResponse(body: string, params: EngineParams): EngineResults;
}

interface EngineParams {
  page: number;
  locale: string;
  safeSearch: 0 | 1 | 2;
  timeRange: '' | 'day' | 'week' | 'month' | 'year';
  engineData: Record<string, string>;
  // Image filters
  imageFilters?: ImageFilters;
  // Video filters
  videoFilters?: VideoFilters;
}

interface ImageFilters {
  size?: 'any' | 'large' | 'medium' | 'small' | 'icon';
  color?: 'any' | 'color' | 'gray' | 'transparent' | string;
  type?: 'any' | 'photo' | 'clipart' | 'lineart' | 'animated' | 'face';
  aspect?: 'any' | 'tall' | 'square' | 'wide' | 'panoramic';
  time?: 'any' | 'day' | 'week' | 'month' | 'year';
  rights?: 'any' | 'creative_commons' | 'commercial';
  filetype?: 'any' | 'jpg' | 'png' | 'gif' | 'svg' | 'webp';
  minWidth?: number;
  minHeight?: number;
  maxWidth?: number;
  maxHeight?: number;
}

interface VideoFilters {
  duration?: 'any' | 'short' | 'medium' | 'long';
  quality?: 'any' | 'hd' | '4k';
  time?: 'any' | 'day' | 'week' | 'month' | 'year';
}

type Category = 'general' | 'images' | 'videos' | 'news' | 'maps' |
               'music' | 'files' | 'science' | 'it' | 'social';
```

## Engine Implementation Patterns

### Standard Online Engine Pattern

```typescript
export class ExampleEngine implements OnlineEngine {
  name = 'example';
  shortcut = 'ex';
  categories: Category[] = ['general'];
  supportsPaging = true;
  supportsTimeRange = true;
  supportsSafeSearch = true;
  supportsLanguage = true;
  maxPage = 10;
  timeout = 5000;
  weight = 1.0;
  disabled = false;

  buildRequest(query: string, params: EngineParams): RequestConfig {
    const searchParams = new URLSearchParams();
    searchParams.set('q', query);
    searchParams.set('page', params.page.toString());

    if (params.timeRange) {
      searchParams.set('time', this.mapTimeRange(params.timeRange));
    }

    return {
      url: `https://example.com/search?${searchParams}`,
      method: 'GET',
      headers: {
        'User-Agent': 'Mozilla/5.0...',
        'Accept': 'text/html',
      },
    };
  }

  parseResponse(body: string, params: EngineParams): EngineResults {
    const results = newEngineResults();

    // Parse HTML/JSON and populate results
    const elements = findElements(body, 'div.result');
    for (const el of elements) {
      results.results.push({
        url: extractHref(el),
        title: extractText(el, 'h3'),
        content: extractText(el, '.snippet'),
        engine: this.name,
        score: this.weight,
        category: 'general',
      });
    }

    return results;
  }

  private mapTimeRange(range: string): string {
    return { day: 'd', week: 'w', month: 'm', year: 'y' }[range] || '';
  }
}
```

### MetaSearch Orchestrator

```typescript
async function executeSearch(
  query: string,
  category: Category,
  params: SearchParams
): Promise<SearchResponse> {
  // 1. Select engines for category
  const engines = selectEngines(category, params.engines);

  // 2. Pre-fetch tokens (e.g., DDG VQD)
  const engineData = await prefetchTokens(query, engines, params.locale);

  // 3. Execute all engines in parallel
  const engineParams: EngineParams = {
    page: params.page,
    locale: params.locale,
    safeSearch: params.safeSearch,
    timeRange: params.timeRange,
    engineData,
    imageFilters: params.imageFilters,
    videoFilters: params.videoFilters,
  };

  const promises = engines.map(engine =>
    executeEngine(engine, query, engineParams)
      .catch(err => ({ results: [], error: err }))
  );

  const responses = await Promise.allSettled(promises);

  // 4. Aggregate results
  const allResults: EngineResult[] = [];
  const enginesUsed: string[] = [];

  for (let i = 0; i < responses.length; i++) {
    const response = responses[i];
    if (response.status === 'fulfilled' && response.value.results.length > 0) {
      allResults.push(...response.value.results);
      enginesUsed.push(engines[i].name);
    }
  }

  // 5. Deduplicate by URL
  const deduplicated = deduplicateResults(allResults);

  // 6. Score and sort
  const scored = scoreResults(deduplicated, params.preferences);

  // 7. Paginate
  const paginated = scored.slice(0, params.perPage);

  return {
    query,
    category,
    results: paginated,
    engines_used: enginesUsed,
    search_time_ms: Date.now() - startTime,
    // ... other fields
  };
}
```

## Frontend - Google-Identical UI

### Color Palette (Exact Google Colors)

```css
:root {
  /* Primary colors */
  --google-blue: #4285f4;
  --google-red: #ea4335;
  --google-yellow: #fbbc05;
  --google-green: #34a853;

  /* Text colors */
  --text-primary: #202124;
  --text-secondary: #5f6368;
  --text-tertiary: #70757a;
  --text-light: #9aa0a6;

  /* Link colors */
  --link-blue: #1a0dab;
  --link-visited: #681da8;
  --link-url: #006621;

  /* UI colors */
  --border: #dadce0;
  --border-light: #ebebeb;
  --hover-bg: #f1f3f4;
  --surface: #f8f9fa;
  --surface-dark: #f1f3f4;
  --background: #ffffff;

  /* Shadows */
  --shadow-sm: 0 1px 2px 0 rgba(60, 64, 67, 0.3), 0 1px 3px 1px rgba(60, 64, 67, 0.15);
  --shadow-md: 0 1px 3px 0 rgba(60, 64, 67, 0.3), 0 4px 8px 3px rgba(60, 64, 67, 0.15);
  --shadow-lg: 0 4px 6px 0 rgba(60, 64, 67, 0.3), 0 8px 16px 3px rgba(60, 64, 67, 0.15);
}
```

### Key UI Components

**Search Box (Google-identical)**
- Rounded pill shape (24px border-radius)
- Shadow on hover/focus
- Microphone and camera icons
- Clear button (X) when text present
- Autocomplete dropdown with up to 10 suggestions
- Recent searches when empty
- Keyboard navigation (arrows, enter, escape)

**Search Result Card**
- Favicon (16x16) + breadcrumb URL in green
- Title as blue link (visited = purple)
- Snippet with bold keyword highlights
- Optional thumbnail on right
- Optional sitelinks (2-column layout)
- "More results from" link for same domain

**Instant Answer Box**
- Calculator: Large result display, expression shown
- Weather: Current conditions, 5-day forecast, location
- Dictionary: Word, phonetic, meanings, synonyms
- Currency: From/To amounts, exchange rate, graph
- Time: Clock display, timezone info

**Knowledge Panel (Right Sidebar)**
- Title + subtitle
- Main image
- Description paragraph
- Facts table (Wikipedia-style)
- External links section
- "People also search for" carousel

**Image Results**
- Masonry grid layout
- Hover to show title overlay
- Click for lightbox with full image
- Related images section

**Video Results**
- Thumbnail with duration badge
- Title, channel, views, date
- Description snippet
- Platform icon (YouTube, Vimeo, etc.)

**News Results**
- Article card with image
- Source name and time ago
- Headline and snippet
- Related articles accordion

## Instant Answer Implementation

### Calculator
- Use math.js for safe expression evaluation
- Support: +, -, *, /, ^, %, sqrt, sin, cos, tan, log, ln, pi, e, factorial
- Pattern detection: `<number> <op> <number>`, functions, constants

### Unit Converter
- Categories: length, weight, temperature, volume, area, speed, data, time, pressure, energy
- 100+ unit types
- Pattern: `<number> <unit> to <unit>`, `<number> <unit> in <unit>`

### Currency
- API: frankfurter.app (free, no key)
- 30+ currencies
- Cache rates for 1 hour
- Pattern: `<number> <currency> to <currency>`, `convert <n> <cur>`

### Weather
- API: wttr.in (free, no key)
- Current conditions + 3-day forecast
- Pattern: `weather <location>`, `weather in <city>`

### Dictionary
- API: api.dictionaryapi.dev
- Pattern: `define <word>`, `meaning of <word>`, `what is <word>`

### Time
- API: worldtimeapi.org
- Pattern: `time in <location>`, `what time in <city>`

### Translation
- API: lingva.ml (LibreTranslate)
- Pattern: `translate <text> to <language>`

## Testing Strategy

### Unit Tests (Vitest)

**Engine Tests** - Each engine has:
- Request building test (correct URL, headers, params)
- Response parsing test with saved fixtures
- Edge cases (empty results, CAPTCHA detection, malformed HTML)
- Real API test (skipped in CI, manual verification)

**Service Tests**:
- Calculator: all operators, edge cases
- Converter: all unit categories
- Currency: rate fetching, conversion
- Bang parser: trigger detection, URL building

### Integration Tests

- Full search flow: query → engines → aggregation → response
- Caching: verify KV reads/writes
- Settings: CRUD operations persist
- Bang redirects: parse and redirect correctly

### E2E Tests (Playwright)

- Home page renders
- Search executes and shows results
- Category tabs work
- Instant answers display
- Pagination works
- Settings persist across sessions

### Real API Verification

Each engine includes a `.test.ts` file with:
```typescript
describe('GoogleEngine', () => {
  it.skipIf(!process.env.RUN_REAL_TESTS)('returns real results', async () => {
    const engine = new GoogleEngine();
    const req = engine.buildRequest('hello world', defaultParams);
    const response = await fetch(req.url, { headers: req.headers });
    const body = await response.text();
    const results = engine.parseResponse(body, defaultParams);

    expect(results.results.length).toBeGreaterThan(0);
    expect(results.results[0].url).toMatch(/^https?:\/\//);
    expect(results.results[0].title).toBeTruthy();
  });
});
```

## Deployment

### Cloudflare Worker Configuration

```toml
# wrangler.toml
name = "mizu-search"
main = "src/index.ts"
compatibility_date = "2026-02-01"
compatibility_flags = ["nodejs_compat"]

[site]
bucket = "./static"

[[kv_namespaces]]
binding = "SEARCH_KV"
id = "generated-id"
preview_id = "generated-preview-id"

[vars]
ENVIRONMENT = "production"

# Optional API keys (set via wrangler secret)
# WEATHER_API_KEY = ""
# WOLFRAM_APP_ID = ""
```

### Deployment Commands

```bash
# Development
cd blueprints/search/app/worker
pnpm install
pnpm dev

# Build frontend
cd frontend && pnpm build

# Deploy
pnpm run deploy

# Verify
curl https://mizu-search.YOUR_SUBDOMAIN.workers.dev/health
```

## Implementation Plan

### Phase 1: Core Infrastructure (Tasks 1-10)
1. Set up project structure with Hono
2. Implement base engine interface
3. Create HTML/XML parser utilities
4. Implement KV storage adapter
5. Create metasearch orchestrator
6. Add CORS and timing middleware
7. Set up Vitest configuration
8. Create basic route handlers
9. Add health check endpoint
10. Configure wrangler.toml

### Phase 2: Web Search Engines (Tasks 11-20)
11. Implement Google engine
12. Implement Bing engine
13. Implement DuckDuckGo engine (VQD handling)
14. Implement Brave engine
15. Implement Yahoo engine
16. Implement Yandex engine
17. Implement Mojeek engine
18. Implement Startpage engine
19. Add web search tests
20. Verify all web engines work

### Phase 3: Image Search Engines (Tasks 21-28)
21. Implement Google Images engine
22. Implement Bing Images engine
23. Implement DuckDuckGo Images engine
24. Implement Flickr engine
25. Implement Unsplash engine
26. Implement Pixabay engine
27. Add image filter support
28. Verify all image engines work

### Phase 4: Video Search Engines (Tasks 29-38)
29. Implement YouTube engine
30. Implement Bing Videos engine
31. Implement DuckDuckGo Videos engine
32. Implement Vimeo engine
33. Implement Dailymotion engine
34. Implement PeerTube engine
35. Implement Rumble engine
36. Implement Odysee engine
37. Add video filter support
38. Verify all video engines work

### Phase 5: News Search Engines (Tasks 39-44)
39. Implement Google News RSS engine
40. Implement Bing News engine
41. Implement DuckDuckGo News engine
42. Implement Yahoo News engine
43. Add news category routing
44. Verify all news engines work

### Phase 6: Academic/Reference Engines (Tasks 45-52)
45. Implement Wikipedia engine
46. Implement Wikidata engine
47. Implement arXiv engine
48. Implement PubMed engine
49. Implement Semantic Scholar engine
50. Implement Crossref engine
51. Implement Open Library engine
52. Verify all academic engines work

### Phase 7: Code/IT Engines (Tasks 53-60)
53. Implement GitHub engine
54. Implement GitLab engine
55. Implement Stack Overflow engine
56. Implement NPM engine
57. Implement PyPI engine
58. Implement crates.io engine
59. Implement pkg.go.dev engine
60. Verify all IT engines work

### Phase 8: Social/Music/Other Engines (Tasks 61-72)
61. Implement Reddit engine
62. Implement Hacker News engine
63. Implement Mastodon engine
64. Implement Lemmy engine
65. Implement SoundCloud engine
66. Implement Bandcamp engine
67. Implement Genius engine
68. Implement 1337x engine
69. Implement OpenStreetMap engine
70. Implement IMDb engine
71. Implement eBay engine
72. Verify all other engines work

### Phase 9: Instant Answers (Tasks 73-82)
73. Implement calculator service
74. Implement unit converter
75. Implement currency converter
76. Implement weather service
77. Implement dictionary service
78. Implement time service
79. Implement translation service
80. Add instant answer detection
81. Add instant answer routing
82. Verify all instant answers work

### Phase 10: Knowledge & Suggestions (Tasks 83-88)
83. Implement knowledge panel service
84. Add Wikipedia knowledge extraction
85. Implement suggestion service
86. Add related searches
87. Implement "People also ask"
88. Verify knowledge features work

### Phase 11: User Features (Tasks 89-98)
89. Implement settings CRUD
90. Implement history CRUD
91. Implement preferences CRUD
92. Implement lenses CRUD
93. Implement bang parser
94. Add built-in bangs (100+)
95. Implement custom bangs
96. Add search caching
97. Add result enrichment
98. Verify all user features work

### Phase 12: Frontend - Core (Tasks 99-110)
99. Set up Vite + TailwindCSS
100. Create SPA router
101. Create state management
102. Create API client
103. Implement home page
104. Implement search box component
105. Implement autocomplete dropdown
106. Implement search results page
107. Implement result card component
108. Implement pagination component
109. Implement category tabs
110. Verify core frontend works

### Phase 13: Frontend - Features (Tasks 111-122)
111. Implement instant answer component
112. Implement knowledge panel component
113. Implement image results page
114. Implement image lightbox
115. Implement video results page
116. Implement news results page
117. Implement "People also ask" component
118. Implement filters component
119. Implement settings page
120. Implement history page
121. Add dark mode support
122. Verify all frontend features work

### Phase 14: Polish & Deploy (Tasks 123-130)
123. Add loading states
124. Add error handling
125. Add offline fallback
126. Optimize bundle size
127. Add SEO meta tags
128. Run full test suite
129. Deploy to Cloudflare
130. Verify *.workers.dev works

## Success Criteria

1. **50+ Search Engines** - All engines implemented and tested
2. **Real Results** - Each engine returns actual search results
3. **100% Feature Parity** - All Google Search features present
4. **Pixel-Perfect UI** - Visually identical to Google
5. **Full Test Coverage** - Unit, integration, and E2E tests
6. **Production Deployment** - Live on *.workers.dev
7. **Sub-200ms Latency** - Fast edge response times
8. **<100KB Bundle** - Minimal frontend size

## Non-Goals

- AI/LLM features (requires external APIs)
- User accounts/authentication (uses client-side storage)
- Server-side rendering (SPA only)
- Mobile apps (web-only)
- Ads or monetization
