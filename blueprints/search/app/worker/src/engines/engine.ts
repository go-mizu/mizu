/**
 * Engine interfaces and base types for the search engine abstraction layer.
 * Ported from Go: pkg/engine/local/engines/engine.go
 */

// ========== Core Types ==========

export type Category =
  | 'general'
  | 'images'
  | 'videos'
  | 'news'
  | 'science'
  | 'it'
  | 'social';

export type TimeRange = '' | 'day' | 'week' | 'month' | 'year';

export type SafeSearch = 0 | 1 | 2; // 0=off, 1=moderate, 2=strict

// ========== Image Filter Types ==========

export type ImageSize = 'any' | 'large' | 'medium' | 'small' | 'icon';
export type ImageColor = 'any' | 'color' | 'gray' | 'transparent' | 'red' | 'orange' | 'yellow' | 'green' | 'teal' | 'blue' | 'purple' | 'pink' | 'white' | 'black' | 'brown';
export type ImageType = 'any' | 'face' | 'photo' | 'clipart' | 'lineart' | 'animated';
export type ImageAspect = 'any' | 'tall' | 'square' | 'wide' | 'panoramic';
export type ImageRights = 'any' | 'creative_commons' | 'commercial';
export type ImageFileType = 'any' | 'jpg' | 'png' | 'gif' | 'webp' | 'svg' | 'bmp' | 'ico';

export interface ImageFilters {
  size?: ImageSize;
  color?: ImageColor;
  type?: ImageType;
  aspect?: ImageAspect;
  rights?: ImageRights;
  filetype?: ImageFileType;
  minWidth?: number;
  minHeight?: number;
  maxWidth?: number;
  maxHeight?: number;
}

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

// ========== Engine Parameters ==========

export interface EngineParams {
  page: number;
  locale: string;
  safeSearch: SafeSearch;
  timeRange: TimeRange;
  engineData: Record<string, string>;
  imageFilters?: ImageFilters;
  videoFilters?: VideoFilters;
}

// ========== Engine Result ==========

export interface EngineResult {
  url: string;
  title: string;
  content: string;
  engine: string;
  score: number;
  category: Category;
  template?: string;

  // Image fields
  imageUrl?: string;
  thumbnailUrl?: string;
  resolution?: string;

  // Video fields
  embedUrl?: string;
  duration?: string;
  channel?: string;
  views?: number;

  // News/social fields
  source?: string;
  publishedAt?: string;

  // Science fields
  authors?: string[];
  doi?: string;
  journal?: string;

  // IT fields
  stars?: number;
  language?: string;
  topics?: string[];

  // Custom metadata (for engine-specific data)
  metadata?: Record<string, unknown>;
}

// ========== Engine Results Collection ==========

export interface EngineResults {
  results: EngineResult[];
  suggestions: string[];
  corrections: string[];
  engineData: Record<string, string>;
}

export function newEngineResults(): EngineResults {
  return {
    results: [],
    suggestions: [],
    corrections: [],
    engineData: {},
  };
}

// ========== Request Config ==========

export interface RequestConfig {
  url: string;
  method: string;
  headers: Record<string, string>;
  cookies: string[];
  body?: string;
}

// ========== Online Engine Interface ==========

export interface OnlineEngine {
  name: string;
  shortcut: string;
  categories: Category[];
  supportsPaging: boolean;
  maxPage: number;
  timeout: number;
  weight: number;
  disabled: boolean;

  buildRequest(query: string, params: EngineParams): RequestConfig;
  parseResponse(body: string, params: EngineParams): EngineResults;
}

// ========== Engine Execution Helper ==========

/**
 * Execute an engine's search request: build the request, fetch with timeout,
 * and parse the response. Returns the parsed EngineResults.
 */
export async function executeEngine(
  engine: OnlineEngine,
  query: string,
  params: EngineParams
): Promise<EngineResults> {
  const config = engine.buildRequest(query, params);

  const headers = new Headers(config.headers);
  if (config.cookies.length > 0) {
    headers.set('Cookie', config.cookies.join('; '));
  }

  const controller = new AbortController();
  const timeoutId = setTimeout(() => controller.abort(), engine.timeout);

  try {
    const response = await fetch(config.url, {
      method: config.method,
      headers,
      body: config.body || undefined,
      signal: controller.signal,
      redirect: 'follow',
    });

    if (!response.ok) {
      throw new Error(
        `${engine.name}: HTTP ${response.status} ${response.statusText}`
      );
    }

    const body = await response.text();
    const results = engine.parseResponse(body, params);

    // Tag all results with the engine name and default score from weight
    for (const r of results.results) {
      r.engine = engine.name;
      if (r.score === 0) {
        r.score = engine.weight;
      }
    }

    return results;
  } finally {
    clearTimeout(timeoutId);
  }
}
