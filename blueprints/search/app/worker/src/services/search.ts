import type {
  SearchResponse,
  SearchResult,
  SearchOptions,
  SearchHistory,
  InstantAnswer,
  SearchLens,
  ImageSearchOptions,
  ImageSearchFilters,
  ImageSearchResponse,
  ImageResult,
  ReverseImageSearchResponse,
  VideoSearchOptions,
  VideoSearchResponse,
  VideoResult,
  VideoSourceInfo,
  VideoDuration,
} from '../types';
import type { CacheStore } from '../store/cache';
import type { KVStore } from '../store/kv';
import type { BangService } from './bang';
import type { InstantService } from './instant';
import type { KnowledgeService } from './knowledge';
import type { MetaSearch } from '../engines/metasearch';
import { getReverseImageEngines } from '../engines/metasearch';
import { executeEngine } from '../engines/engine';
import type { Category, EngineParams, EngineResult, TimeRange, ImageFilters } from '../engines/engine';

/**
 * Convert time range string to TimeRange type.
 * Note: 'hour' is not a standard TimeRange, so we map it to 'day' for engine compatibility.
 */
function parseTimeRange(tr: string | undefined): TimeRange {
  if (tr === 'hour') {
    // 'hour' is not a standard TimeRange, map to 'day' for engines that don't support it
    return 'day';
  }
  if (tr === 'day' || tr === 'week' || tr === 'month' || tr === 'year') {
    return tr;
  }
  return '';
}

/**
 * Convert ImageSearchFilters to engine ImageFilters.
 */
function toImageFilters(filters?: ImageSearchFilters): ImageFilters | undefined {
  if (!filters) return undefined;
  return {
    size: filters.size,
    color: filters.color,
    type: filters.type,
    aspect: filters.aspect,
    rights: filters.rights,
    filetype: filters.filetype,
    minWidth: filters.min_width,
    minHeight: filters.min_height,
    maxWidth: filters.max_width,
    maxHeight: filters.max_height,
  };
}

/**
 * Convert SearchOptions to EngineParams and Category for metasearch.
 */
function toEngineParams(options: SearchOptions, engineSecrets: Record<string, string> = {}): { category: Category; params: EngineParams } {
  let category: Category = 'general';
  if (options.file_type === 'image') {
    category = 'images';
  } else if (options.file_type === 'video') {
    category = 'videos';
  } else if (options.file_type === 'news') {
    category = 'news';
  }

  return {
    category,
    params: {
      page: options.page,
      locale: options.language ?? 'en',
      timeRange: parseTimeRange(options.time_range),
      safeSearch: options.safe_search === 'strict' ? 2 : (options.safe_search === 'off' ? 0 : 1),
      engineData: engineSecrets,
    },
  };
}

/**
 * Convert ImageSearchOptions to EngineParams for image search.
 */
function toImageEngineParams(options: ImageSearchOptions, engineSecrets: Record<string, string> = {}): EngineParams {
  const safeLevel = options.filters?.safe;
  let safeSearch: 0 | 1 | 2 = 1;
  if (safeLevel === 'strict') safeSearch = 2;
  else if (safeLevel === 'off') safeSearch = 0;

  return {
    page: options.page,
    locale: options.language ?? 'en',
    timeRange: parseTimeRange(options.filters?.time ?? options.time_range),
    safeSearch,
    engineData: engineSecrets,
    imageFilters: toImageFilters(options.filters),
  };
}

/**
 * Convert EngineResult to SearchResult format.
 */
function buildMetadata(r: EngineResult): Record<string, unknown> | undefined {
  const metadata: Record<string, unknown> = { ...(r.metadata ?? {}) };

  if (r.authors && r.authors.length > 0) {
    metadata.authors = r.authors.join(', ');
  }
  if (r.doi) {
    metadata.doi = r.doi;
  }
  if (r.journal) {
    metadata.journal = r.journal;
  }

  if (r.publishedAt) {
    const year = new Date(r.publishedAt).getFullYear();
    if (!isNaN(year) && !metadata.year) {
      metadata.year = year;
    }
  }

  if (r.category === 'science') {
    if (!metadata.source) metadata.source = r.engine;
    const citations =
      (metadata.citationCount as number | undefined) ??
      (metadata.citations as number | undefined) ??
      (metadata.pmcrefcount as number | undefined);
    if (citations !== undefined) metadata.citations = citations;

    const pdfUrl =
      (metadata.pdf_url as string | undefined) ??
      (metadata.openAccessPdfUrl as string | undefined);
    if (pdfUrl) metadata.pdf_url = pdfUrl;
  }

  if (r.category === 'it') {
    if (r.stars !== undefined) metadata.stars = r.stars;
    if (r.language) metadata.language = r.language;
    if (r.topics && r.topics.length > 0) metadata.topics = r.topics;
    if (!metadata.source) metadata.source = r.engine;
  }

  if (r.category === 'videos' && ['soundcloud', 'bandcamp', 'genius'].includes(r.engine)) {
    if (!metadata.source) metadata.source = r.engine;
    const artist =
      (metadata.artistName as string | undefined) ??
      (metadata.artist as string | undefined) ??
      (r.channel || undefined);
    if (artist) metadata.artist = artist;
    const album =
      (metadata.albumName as string | undefined) ??
      (metadata.album as string | undefined);
    if (album) metadata.album = album;
    if (r.duration) metadata.duration = r.duration;
  }

  if (r.category === 'social') {
    if (!metadata.source) metadata.source = r.engine;
    if (r.source && !metadata.subreddit && r.source.startsWith('r/')) {
      metadata.subreddit = r.source.replace(/^r\//, '');
    }
    if (r.source && !metadata.source_label) {
      metadata.source_label = r.source;
    }
    if (!metadata.author && r.channel) metadata.author = r.channel;
    if (!metadata.published && r.publishedAt) metadata.published = r.publishedAt;
  }

  if (r.engine === 'openstreetmap') {
    const latRaw = (metadata.lat as number | string | undefined) ?? (metadata.latitude as number | string | undefined);
    const lonRaw = (metadata.lon as number | string | undefined) ?? (metadata.longitude as number | string | undefined);
    const lat = typeof latRaw === 'string' ? parseFloat(latRaw) : latRaw;
    const lon = typeof lonRaw === 'string' ? parseFloat(lonRaw) : lonRaw;
    if (lat !== undefined && !isNaN(lat)) metadata.lat = lat;
    if (lon !== undefined && !isNaN(lon)) metadata.lon = lon;
    if (!metadata.type && metadata['type']) metadata.type = metadata['type'];
  }

  if (!metadata.pdf_url && r.content) {
    const match = r.content.match(/\[PDF:\s*(https?:\/\/[^\]]+)\]/i);
    if (match?.[1]) metadata.pdf_url = match[1];
  }

  return Object.keys(metadata).length > 0 ? metadata : undefined;
}

function toSearchResult(r: EngineResult, index: number): SearchResult {
  return {
    id: `${Date.now().toString(36)}-${index}`,
    url: r.url,
    title: r.title,
    snippet: r.content,
    content: r.content,
    domain: extractDomain(r.url),
    thumbnail: r.thumbnailUrl ? { url: r.thumbnailUrl } : undefined,
    published: r.publishedAt,
    score: r.score,
    crawled_at: new Date().toISOString(),
    engine: r.engine,
    engines: [r.engine],
    metadata: buildMetadata(r),
  };
}

/**
 * Convert EngineResult to ImageResult format.
 */
function toImageResult(r: EngineResult, index: number): ImageResult {
  const [width, height] = parseResolution(r.resolution);
  return {
    id: `${Date.now().toString(36)}-${index}`,
    url: r.imageUrl || r.url,
    thumbnail_url: r.thumbnailUrl || '',
    title: r.title,
    source_url: r.url,
    source_domain: extractDomain(r.url),
    width,
    height,
    file_size: 0,
    format: extractFormat(r.imageUrl || r.url),
    engine: r.engine,
    score: r.score,
  };
}

function parseResolution(resolution?: string): [number, number] {
  if (!resolution) return [0, 0];
  const match = resolution.match(/(\d+)x(\d+)/);
  if (match) {
    return [parseInt(match[1], 10), parseInt(match[2], 10)];
  }
  return [0, 0];
}

function extractFormat(url: string): string {
  const match = url.match(/\.(\w+)(?:\?|$)/i);
  if (match) {
    const ext = match[1].toLowerCase();
    if (['jpg', 'jpeg', 'png', 'gif', 'webp', 'svg', 'bmp', 'ico'].includes(ext)) {
      return ext === 'jpeg' ? 'jpg' : ext;
    }
  }
  return 'unknown';
}

function extractDomain(url: string): string {
  try {
    return new URL(url).hostname;
  } catch {
    return '';
  }
}

function normalizeHost(input: string): string {
  const trimmed = input.trim();
  if (!trimmed) return '';
  const withoutProtocol = trimmed.replace(/^https?:\/\//i, '');
  return withoutProtocol.split('/')[0].replace(/^www\./i, '').toLowerCase();
}

function matchesDomain(url: string, domain: string): boolean {
  const host = normalizeHost(url);
  const target = normalizeHost(domain);
  if (!host || !target) return false;
  return host === target || host.endsWith(`.${target}`);
}

function matchesKeywords(result: SearchResult, keywords: string[], mode: 'include' | 'exclude'): boolean {
  if (keywords.length === 0) return mode === 'include';
  const haystack = `${result.title} ${result.snippet} ${result.content ?? ''}`.toLowerCase();
  const matched = keywords.some((keyword) => haystack.includes(keyword.toLowerCase()));
  return mode === 'include' ? matched : !matched;
}

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
  if (parts.length === 3) return parts[0] * 3600 + parts[1] * 60 + parts[2];
  if (parts.length === 2) return parts[0] * 60 + parts[1];
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
  if (diffDays < 30) return `${Math.floor(diffDays / 7)} weeks ago`;
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

// Instant answer detection patterns
const CALC_PATTERN = /^\d+[\s]*[+\-*/^%][\s]*\d+/;
const FUNC_PATTERN = /^(sqrt|sin|cos|tan|log|ln|abs|ceil|floor|round)\s*\(/i;
const UNIT_PATTERN = /^(\d+\.?\d*)\s*(mm|cm|m|km|in|ft|yd|mi|mg|g|kg|lb|oz|ton|c|f|k|ml|l|gal|qt|pt|cup|tbsp|tsp|fl_oz|mm2|cm2|m2|km2|in2|ft2|acre|hectare|m\/s|km\/h|mph|knots|b|kb|mb|gb|tb|pb|ms|s|min|hr|day|week|month|year)\s+(to|in)\s+(mm|cm|m|km|in|ft|yd|mi|mg|g|kg|lb|oz|ton|c|f|k|ml|l|gal|qt|pt|cup|tbsp|tsp|fl_oz|mm2|cm2|m2|km2|in2|ft2|acre|hectare|m\/s|km\/h|mph|knots|b|kb|mb|gb|tb|pb|ms|s|min|hr|day|week|month|year)$/i;
const CURRENCY_PATTERN = /^(\d+\.?\d*)\s*(usd|eur|gbp|jpy|cad|aud|chf|cny|inr|krw|brl|mxn|sgd|hkd|nzd|sek|nok|dkk|pln|zar|try|thb|idr|php|czk|ils|clp|myr|twd|ars|cop|sar|aed|egp|vnd|bgn|hrk|huf|isk|ron|rub)\s+(to|in)\s+(usd|eur|gbp|jpy|cad|aud|chf|cny|inr|krw|brl|mxn|sgd|hkd|nzd|sek|nok|dkk|pln|zar|try|thb|idr|php|czk|ils|clp|myr|twd|ars|cop|sar|aed|egp|vnd|bgn|hrk|huf|isk|ron|rub)$/i;
const WEATHER_PATTERN = /^weather\s+(in\s+)?(.+)/i;
const DEFINE_PATTERN = /^(?:define|meaning\s+of)\s+(.+)/i;
const TIME_PATTERN = /^(?:time\s+in|what\s+time.*in)\s+(.+)/i;

function generateId(): string {
  const timestamp = Date.now().toString(36);
  const random = Math.random().toString(36).substring(2, 8);
  return `${timestamp}-${random}`;
}

function hashSearchKey(query: string, options: SearchOptions | ImageSearchOptions): string {
  const imageFilters = (options as ImageSearchOptions).filters;
  const filterStr = imageFilters
    ? `|${imageFilters.size ?? ''}|${imageFilters.color ?? ''}|${imageFilters.type ?? ''}|${imageFilters.aspect ?? ''}|${imageFilters.time ?? ''}|${imageFilters.rights ?? ''}|${imageFilters.filetype ?? ''}|${imageFilters.safe ?? ''}`
    : '';
  const key = `${query}|${options.page}|${options.per_page}|${options.time_range ?? ''}|${options.region ?? ''}|${options.language ?? ''}|${options.safe_search ?? ''}|${options.site ?? ''}|${options.lens ?? ''}${filterStr}`;
  let hash = 0;
  for (let i = 0; i < key.length; i++) {
    const char = key.charCodeAt(i);
    hash = ((hash << 5) - hash + char) | 0;
  }
  return Math.abs(hash).toString(36);
}

export class SearchService {
  private metasearch: MetaSearch;
  private cache: CacheStore;
  private kvStore: KVStore;
  private bangService: BangService;
  private instantService: InstantService;
  private knowledgeService: KnowledgeService;
  private engineSecrets: Record<string, string>;

  constructor(
    metasearch: MetaSearch,
    cache: CacheStore,
    kvStore: KVStore,
    bangService: BangService,
    instantService: InstantService,
    knowledgeService: KnowledgeService,
    engineSecrets?: Record<string, string>,
  ) {
    this.metasearch = metasearch;
    this.cache = cache;
    this.kvStore = kvStore;
    this.bangService = bangService;
    this.instantService = instantService;
    this.knowledgeService = knowledgeService;
    this.engineSecrets = engineSecrets ?? {};
  }

  /**
   * Update engine secrets at runtime (e.g. after resolving KV-stored keys).
   */
  updateEngineSecrets(secrets: Record<string, string>): void {
    this.engineSecrets = secrets;
  }

  /**
   * Main search method. Handles bang redirects, caching, instant answers,
   * knowledge panels, and metasearch aggregation.
   */
  async search(query: string, options: SearchOptions): Promise<SearchResponse> {
    const startTime = Date.now();
    const trimmedQuery = query.trim();

    if (!trimmedQuery) {
      return this.emptyResponse(trimmedQuery, options, 0);
    }

    // 1. Check for bang redirect
    const bangResult = await this.bangService.parse(trimmedQuery);
    if (bangResult.redirect) {
      return {
        ...this.emptyResponse(bangResult.query, options, Date.now() - startTime),
        redirect: bangResult.redirect,
        bang: bangResult.bang?.trigger,
        category: bangResult.category,
      };
    }

    const searchQuery = bangResult.query;
    const cacheHash = hashSearchKey(searchQuery, options);

    // 2. Check cache
    const cachedResponse = await this.cache.getSearch(cacheHash);
    if (cachedResponse) {
      return cachedResponse;
    }

    // 3. Run instant answer detection and metasearch in parallel
    const { category, params } = toEngineParams(options, this.engineSecrets);
    const [instantAnswer, knowledgePanel, metaResult] = await Promise.all([
      this.detectInstantAnswer(searchQuery),
      options.page === 1 ? this.knowledgeService.getPanel(searchQuery) : Promise.resolve(null),
      this.metasearch.search(searchQuery, category, params),
    ]);

    // 4. Convert and paginate results
    const allResults = metaResult.results.map(toSearchResult);
    const filteredResults = await this.applyPostFilters(allResults, options);
    const startIndex = (options.page - 1) * options.per_page;
    const endIndex = startIndex + options.per_page;
    const paginatedResults = filteredResults.slice(startIndex, endIndex);
    const totalResults = filteredResults.length;
    const hasMore = endIndex < totalResults;

    const response: SearchResponse = {
      query: searchQuery,
      corrected_query: metaResult.corrections[0],
      total_results: totalResults,
      results: paginatedResults,
      suggestions: metaResult.suggestions,
      related_searches: metaResult.suggestions,
      instant_answer: instantAnswer ?? undefined,
      knowledge_panel: knowledgePanel ?? undefined,
      search_time_ms: Date.now() - startTime,
      page: options.page,
      per_page: options.per_page,
      has_more: hasMore,
    };

    // 5. Cache the result
    await this.cache.setSearch(cacheHash, response);

    // 6. Add to search history (fire-and-forget, don't await)
    this.addToHistory(searchQuery, totalResults).catch(() => {
      // Silently ignore history errors
    });

    return response;
  }

  /**
   * Search for science/academic results.
   */
  async searchScience(query: string, options: SearchOptions): Promise<SearchResponse> {
    const startTime = Date.now();
    const cacheHash = hashSearchKey(`science:${query}`, options);

    const cachedResponse = await this.cache.getSearch(cacheHash);
    if (cachedResponse) {
      return cachedResponse;
    }

    const params: EngineParams = {
      page: options.page,
      locale: options.language ?? 'en',
      timeRange: parseTimeRange(options.time_range),
      safeSearch: options.safe_search === 'strict' ? 2 : (options.safe_search === 'off' ? 0 : 1),
      engineData: this.engineSecrets,
    };

    const metaResult = await this.metasearch.search(query, 'science', params);
    const allResults = metaResult.results.map(toSearchResult);
    const filteredResults = await this.applyPostFilters(allResults, options);
    const startIndex = (options.page - 1) * options.per_page;
    const endIndex = startIndex + options.per_page;
    const paginatedResults = filteredResults.slice(startIndex, endIndex);
    const totalResults = filteredResults.length;
    const hasMore = endIndex < totalResults;

    const response: SearchResponse = {
      query,
      total_results: totalResults,
      results: paginatedResults,
      suggestions: metaResult.suggestions,
      search_time_ms: Date.now() - startTime,
      page: options.page,
      per_page: options.per_page,
      has_more: hasMore,
    };

    await this.cache.setSearch(cacheHash, response);
    return response;
  }

  /**
   * Search for code/IT results.
   */
  async searchCode(query: string, options: SearchOptions): Promise<SearchResponse> {
    const startTime = Date.now();
    const cacheHash = hashSearchKey(`code:${query}`, options);

    const cachedResponse = await this.cache.getSearch(cacheHash);
    if (cachedResponse) {
      return cachedResponse;
    }

    const params: EngineParams = {
      page: options.page,
      locale: options.language ?? 'en',
      timeRange: parseTimeRange(options.time_range),
      safeSearch: options.safe_search === 'strict' ? 2 : (options.safe_search === 'off' ? 0 : 1),
      engineData: this.engineSecrets,
    };

    const metaResult = await this.metasearch.search(query, 'it', params);
    const allResults = metaResult.results.map(toSearchResult);
    const filteredResults = await this.applyPostFilters(allResults, options);
    const startIndex = (options.page - 1) * options.per_page;
    const endIndex = startIndex + options.per_page;
    const paginatedResults = filteredResults.slice(startIndex, endIndex);
    const totalResults = filteredResults.length;
    const hasMore = endIndex < totalResults;

    const response: SearchResponse = {
      query,
      total_results: totalResults,
      results: paginatedResults,
      suggestions: metaResult.suggestions,
      search_time_ms: Date.now() - startTime,
      page: options.page,
      per_page: options.per_page,
      has_more: hasMore,
    };

    await this.cache.setSearch(cacheHash, response);
    return response;
  }

  /**
   * Search for social results.
   */
  async searchSocial(query: string, options: SearchOptions): Promise<SearchResponse> {
    const startTime = Date.now();
    const cacheHash = hashSearchKey(`social:${query}`, options);

    const cachedResponse = await this.cache.getSearch(cacheHash);
    if (cachedResponse) {
      return cachedResponse;
    }

    const params: EngineParams = {
      page: options.page,
      locale: options.language ?? 'en',
      timeRange: parseTimeRange(options.time_range),
      safeSearch: options.safe_search === 'strict' ? 2 : (options.safe_search === 'off' ? 0 : 1),
      engineData: this.engineSecrets,
    };

    const metaResult = await this.metasearch.search(query, 'social', params);
    const allResults = metaResult.results.map(toSearchResult);
    const filteredResults = await this.applyPostFilters(allResults, options);
    const startIndex = (options.page - 1) * options.per_page;
    const endIndex = startIndex + options.per_page;
    const paginatedResults = filteredResults.slice(startIndex, endIndex);
    const totalResults = filteredResults.length;
    const hasMore = endIndex < totalResults;

    const response: SearchResponse = {
      query,
      total_results: totalResults,
      results: paginatedResults,
      suggestions: metaResult.suggestions,
      search_time_ms: Date.now() - startTime,
      page: options.page,
      per_page: options.per_page,
      has_more: hasMore,
    };

    await this.cache.setSearch(cacheHash, response);
    return response;
  }

  /**
   * Search for music results (music-only engines).
   */
  async searchMusic(query: string, options: SearchOptions): Promise<SearchResponse> {
    const startTime = Date.now();
    const cacheHash = hashSearchKey(`music:${query}`, options);

    const cachedResponse = await this.cache.getSearch(cacheHash);
    if (cachedResponse) {
      return cachedResponse;
    }

    const params: EngineParams = {
      page: options.page,
      locale: options.language ?? 'en',
      timeRange: parseTimeRange(options.time_range),
      safeSearch: options.safe_search === 'strict' ? 2 : (options.safe_search === 'off' ? 0 : 1),
      engineData: this.engineSecrets,
    };

    const metaResult = await this.metasearch.search(query, 'videos', params, {
      engines: ['soundcloud', 'bandcamp', 'genius'],
    });
    const allResults = metaResult.results.map(toSearchResult);
    const filteredResults = await this.applyPostFilters(allResults, options);
    const startIndex = (options.page - 1) * options.per_page;
    const endIndex = startIndex + options.per_page;
    const paginatedResults = filteredResults.slice(startIndex, endIndex);
    const totalResults = filteredResults.length;
    const hasMore = endIndex < totalResults;

    const response: SearchResponse = {
      query,
      total_results: totalResults,
      results: paginatedResults,
      suggestions: metaResult.suggestions,
      search_time_ms: Date.now() - startTime,
      page: options.page,
      per_page: options.per_page,
      has_more: hasMore,
    };

    await this.cache.setSearch(cacheHash, response);
    return response;
  }

  /**
   * Search for map/location results (OpenStreetMap only).
   */
  async searchMaps(query: string, options: SearchOptions): Promise<SearchResponse> {
    const startTime = Date.now();
    const cacheHash = hashSearchKey(`maps:${query}`, options);

    const cachedResponse = await this.cache.getSearch(cacheHash);
    if (cachedResponse) {
      return cachedResponse;
    }

    const params: EngineParams = {
      page: options.page,
      locale: options.language ?? 'en',
      timeRange: '',
      safeSearch: options.safe_search === 'strict' ? 2 : (options.safe_search === 'off' ? 0 : 1),
      engineData: this.engineSecrets,
    };

    const metaResult = await this.metasearch.search(query, 'general', params, {
      engines: ['openstreetmap'],
    });
    const allResults = metaResult.results.map(toSearchResult);
    const filteredResults = await this.applyPostFilters(allResults, options);
    const startIndex = (options.page - 1) * options.per_page;
    const endIndex = startIndex + options.per_page;
    const paginatedResults = filteredResults.slice(startIndex, endIndex);
    const totalResults = filteredResults.length;
    const hasMore = endIndex < totalResults;

    const response: SearchResponse = {
      query,
      total_results: totalResults,
      results: paginatedResults,
      suggestions: metaResult.suggestions,
      search_time_ms: Date.now() - startTime,
      page: options.page,
      per_page: options.per_page,
      has_more: hasMore,
    };

    await this.cache.setSearch(cacheHash, response);
    return response;
  }

  /**
   * Search for images with full filter support.
   */
  async searchImages(query: string, options: ImageSearchOptions): Promise<ImageSearchResponse> {
    const startTime = Date.now();
    const cacheHash = hashSearchKey(`img:${query}`, options);

    const cachedResponse = await this.cache.getImageSearch(cacheHash);
    if (cachedResponse) {
      return cachedResponse;
    }

    const params = toImageEngineParams(options, this.engineSecrets);
    const metaResult = await this.metasearch.search(query, 'images', params);
    const allResults = metaResult.results.map(toImageResult);

    // Default to 30 results per page for images
    const perPage = options.per_page || 30;
    const startIndex = (options.page - 1) * perPage;
    const endIndex = startIndex + perPage;
    const paginatedResults = allResults.slice(startIndex, endIndex);
    const totalResults = allResults.length;
    const hasMore = endIndex < totalResults;

    const response: ImageSearchResponse = {
      query,
      filters: options.filters,
      total_results: totalResults,
      results: paginatedResults,
      related_searches: metaResult.suggestions,
      search_time_ms: Date.now() - startTime,
      page: options.page,
      per_page: perPage,
      has_more: hasMore,
    };

    await this.cache.setImageSearch(cacheHash, response);
    return response;
  }

  /**
   * Reverse image search by URL or base64 image data.
   */
  async reverseImageSearch(
    imageUrl?: string,
    imageData?: string
  ): Promise<ReverseImageSearchResponse> {
    const startTime = Date.now();

    if (!imageUrl && !imageData) {
      throw new Error('Either imageUrl or imageData is required');
    }

    // Use URL if provided, otherwise we'd need to upload the image
    const searchUrl = imageUrl || '';
    if (!searchUrl) {
      // For now, return empty results for base64 uploads
      // Full implementation would upload to a temporary URL
      return {
        query_image: { url: '' },
        exact_matches: [],
        similar_images: [],
        pages_with_image: [],
        search_time_ms: Date.now() - startTime,
      };
    }

    // Get reverse image search engines
    const reverseEngines = getReverseImageEngines();

    // Execute all reverse image engines in parallel
    const params: EngineParams = {
      page: 1,
      locale: 'en',
      safeSearch: 1,
      timeRange: '',
      engineData: this.engineSecrets,
    };

    const promises = reverseEngines.map((engine) =>
      executeEngine(engine, searchUrl, params).catch(() => null)
    );

    const results = await Promise.all(promises);

    // Aggregate results
    const similarImages: ImageResult[] = [];
    const exactMatches: ImageResult[] = [];
    const pagesWithImage: SearchResult[] = [];

    let idx = 0;
    for (const result of results) {
      if (!result) continue;

      for (const r of result.results) {
        if (r.imageUrl) {
          const imgResult = toImageResult(r, idx++);
          // Heuristic: if URL matches exactly, it's an exact match
          if (r.imageUrl === searchUrl || r.url === searchUrl) {
            exactMatches.push(imgResult);
          } else {
            similarImages.push(imgResult);
          }
        }

        // Pages that contain the image
        if (r.url && !r.imageUrl) {
          pagesWithImage.push(toSearchResult(r, idx++));
        }
      }
    }

    return {
      query_image: {
        url: searchUrl,
      },
      exact_matches: exactMatches.slice(0, 10),
      similar_images: similarImages.slice(0, 50),
      pages_with_image: pagesWithImage.slice(0, 20),
      search_time_ms: Date.now() - startTime,
    };
  }

  /**
   * Search for videos with full filter and sort support.
   */
  async searchVideos(query: string, options: VideoSearchOptions): Promise<VideoSearchResponse> {
    const startTime = Date.now();
    const cacheHash = hashSearchKey(`vid:${query}`, options);

    const cachedResponse = await this.cache.getVideoSearch(cacheHash);
    if (cachedResponse) {
      return cachedResponse;
    }

    // Build engine params
    const safeLevel = options.filters?.safe ?? options.safe_search;
    let safeSearch: 0 | 1 | 2 = 1;
    if (safeLevel === 'strict') safeSearch = 2;
    else if (safeLevel === 'off') safeSearch = 0;

    const params: EngineParams = {
      page: options.page,
      locale: options.language ?? 'en',
      timeRange: parseTimeRange(options.filters?.time ?? options.time_range),
      safeSearch,
      engineData: this.engineSecrets,
      videoFilters: options.filters ? {
        duration: options.filters.duration,
        quality: options.filters.quality,
        source: options.filters.source,
        cc: options.filters.cc,
      } : undefined,
    };

    const metaResult = await this.metasearch.search(query, 'videos', params);
    let allResults = metaResult.results.map(toVideoResult);

    // Apply duration filter (client-side)
    const durationFilter = options.filters?.duration as VideoDuration | undefined;
    if (durationFilter && durationFilter !== 'any') {
      allResults = allResults.filter((r) => {
        const seconds = r.duration_seconds ?? 0;
        switch (durationFilter) {
          case 'short':
            return seconds > 0 && seconds < 240; // < 4 minutes
          case 'medium':
            return seconds >= 240 && seconds <= 1200; // 4-20 minutes
          case 'long':
            return seconds > 1200; // > 20 minutes
          default:
            return true;
        }
      });
    }

    // Apply source filter (client-side)
    const sourceFilter = options.filters?.source;
    if (sourceFilter) {
      allResults = allResults.filter((r) => r.source === sourceFilter || r.engine === sourceFilter);
    }

    // Apply sorting
    const sortBy = options.sort ?? 'relevance';
    switch (sortBy) {
      case 'date':
        allResults.sort((a, b) => {
          const dateA = a.published_at ? new Date(a.published_at).getTime() : 0;
          const dateB = b.published_at ? new Date(b.published_at).getTime() : 0;
          return dateB - dateA; // newest first
        });
        break;
      case 'views':
        allResults.sort((a, b) => (b.views ?? 0) - (a.views ?? 0)); // most views first
        break;
      case 'duration':
        allResults.sort((a, b) => (b.duration_seconds ?? 0) - (a.duration_seconds ?? 0)); // longest first
        break;
      // 'relevance' - keep original order (by score)
    }

    // Calculate available sources with result counts
    const sourceCounts = new Map<string, number>();
    for (const r of metaResult.results) {
      const engine = r.engine;
      sourceCounts.set(engine, (sourceCounts.get(engine) || 0) + 1);
    }

    const sourceDisplayNames: Record<string, string> = {
      youtube: 'YouTube',
      vimeo: 'Vimeo',
      dailymotion: 'Dailymotion',
      google_videos: 'Google Videos',
      bing_videos: 'Bing Videos',
      peertube: 'PeerTube',
      '360search': '360 Search',
      sogou: 'Sogou',
      duckduckgo_videos: 'DuckDuckGo',
    };

    const availableSources: VideoSourceInfo[] = Array.from(sourceCounts.entries()).map(([name, count]) => ({
      name,
      display_name: sourceDisplayNames[name] || name,
      icon: getSourceIcon(name),
      result_count: count,
      enabled: !sourceFilter || sourceFilter === name,
    }));

    // Paginate results
    const perPage = options.per_page || 20;
    const startIndex = (options.page - 1) * perPage;
    const endIndex = startIndex + perPage;
    const paginatedResults = allResults.slice(startIndex, endIndex);
    const totalResults = allResults.length;
    const hasMore = endIndex < totalResults;

    const response: VideoSearchResponse = {
      query,
      total_results: totalResults,
      results: paginatedResults,
      filters: options.filters,
      available_sources: availableSources,
      search_time_ms: Date.now() - startTime,
      page: options.page,
      per_page: perPage,
      has_more: hasMore,
    };

    await this.cache.setVideoSearch(cacheHash, response);
    return response;
  }

  /**
   * Search for news.
   */
  async searchNews(query: string, options: SearchOptions): Promise<SearchResponse> {
    const startTime = Date.now();
    const newsOptions: SearchOptions = { ...options, file_type: 'news' };
    const cacheHash = hashSearchKey(`news:${query}`, newsOptions);

    const cachedResponse = await this.cache.getSearch(cacheHash);
    if (cachedResponse) {
      return cachedResponse;
    }

    const { params } = toEngineParams(newsOptions, this.engineSecrets);
    const metaResult = await this.metasearch.search(query, 'news', params);
    const allResults = metaResult.results.map(toSearchResult);
    const filteredResults = await this.applyPostFilters(allResults, options);
    const startIndex = (options.page - 1) * options.per_page;
    const endIndex = startIndex + options.per_page;
    const paginatedResults = filteredResults.slice(startIndex, endIndex);
    const totalResults = filteredResults.length;
    const hasMore = endIndex < totalResults;

    const response: SearchResponse = {
      query,
      total_results: totalResults,
      results: paginatedResults,
      search_time_ms: Date.now() - startTime,
      page: options.page,
      per_page: options.per_page,
      has_more: hasMore,
    };

    await this.cache.setSearch(cacheHash, response);
    return response;
  }

  /**
   * Apply site and lens filters to results.
   */
  private async applyPostFilters(results: SearchResult[], options: SearchOptions): Promise<SearchResult[]> {
    let filtered = results;

    if (options.site) {
      filtered = filtered.filter((r) => matchesDomain(r.url, options.site!));
    }
    if (options.exclude_site) {
      filtered = filtered.filter((r) => !matchesDomain(r.url, options.exclude_site!));
    }

    if (options.lens) {
      const lens = await this.kvStore.getLens(options.lens);
      if (lens) {
        filtered = this.applyLensFilters(filtered, lens);
      }
    }

    return filtered;
  }

  private applyLensFilters(results: SearchResult[], lens: SearchLens): SearchResult[] {
    let filtered = results;

    if (lens.domains && lens.domains.length > 0) {
      filtered = filtered.filter((r) => lens.domains!.some((domain) => matchesDomain(r.url, domain)));
    }
    if (lens.exclude && lens.exclude.length > 0) {
      filtered = filtered.filter((r) => !lens.exclude!.some((domain) => matchesDomain(r.url, domain)));
    }

    if (lens.include_keywords && lens.include_keywords.length > 0) {
      filtered = filtered.filter((r) => matchesKeywords(r, lens.include_keywords!, 'include'));
    }
    if (lens.exclude_keywords && lens.exclude_keywords.length > 0) {
      filtered = filtered.filter((r) => matchesKeywords(r, lens.exclude_keywords!, 'exclude'));
    }

    return filtered;
  }

  /**
   * Detect and compute instant answers based on query patterns.
   */
  private async detectInstantAnswer(query: string): Promise<InstantAnswer | null> {
    try {
      // Calculator: "2 + 3", "sqrt(16)", etc.
      if (CALC_PATTERN.test(query) || FUNC_PATTERN.test(query)) {
        const result = this.instantService.calculate(query);
        return {
          type: 'calculator',
          query,
          result: result.formatted,
          data: result,
        };
      }

      // Unit conversion: "10 km to mi"
      if (UNIT_PATTERN.test(query)) {
        const result = this.instantService.convert(query);
        return {
          type: 'unit_conversion',
          query,
          result: `${result.from_value} ${result.from_unit} = ${result.to_value} ${result.to_unit}`,
          data: result,
        };
      }

      // Currency conversion: "100 usd to eur"
      if (CURRENCY_PATTERN.test(query)) {
        const result = await this.instantService.currency(query);
        return {
          type: 'currency',
          query,
          result: `${result.from_amount} ${result.from_currency} = ${result.to_amount.toFixed(2)} ${result.to_currency}`,
          data: result,
        };
      }

      // Weather: "weather in London"
      const weatherMatch = query.match(WEATHER_PATTERN);
      if (weatherMatch) {
        const location = weatherMatch[2].trim();
        const result = await this.instantService.weather(location);
        return {
          type: 'weather',
          query,
          result: `${result.temperature}${result.unit} ${result.condition} in ${result.location}`,
          data: result,
        };
      }

      // Define: "define serendipity" or "meaning of serendipity"
      const defineMatch = query.match(DEFINE_PATTERN);
      if (defineMatch) {
        const word = defineMatch[1].trim();
        const result = await this.instantService.define(word);
        return {
          type: 'definition',
          query,
          result: result.definitions[0] ?? '',
          data: result,
        };
      }

      // Time: "time in Tokyo" or "what time is it in London"
      const timeMatch = query.match(TIME_PATTERN);
      if (timeMatch) {
        const location = timeMatch[1].trim();
        const result = this.instantService.time(location);
        return {
          type: 'time',
          query,
          result: `${result.time} in ${result.location}`,
          data: result,
        };
      }

      return null;
    } catch {
      // If instant answer detection fails, return null and let the search proceed
      return null;
    }
  }

  /**
   * Add a search query to history.
   */
  private async addToHistory(query: string, totalResults: number): Promise<void> {
    const entry: SearchHistory = {
      id: generateId(),
      query,
      results: totalResults,
      searched_at: new Date().toISOString(),
    };
    await this.kvStore.addHistory(entry);
  }

  /**
   * Build an empty search response.
   */
  private emptyResponse(
    query: string,
    options: SearchOptions,
    timeMs: number,
  ): SearchResponse {
    return {
      query,
      total_results: 0,
      results: [],
      search_time_ms: timeMs,
      page: options.page,
      per_page: options.per_page,
      has_more: false,
    };
  }
}
