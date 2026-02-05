import type {
  SearchResponse,
  SearchResult,
  SearchOptions,
  SearchHistory,
  InstantAnswer,
} from '../types';
import type { CacheStore } from '../store/cache';
import type { KVStore } from '../store/kv';
import type { BangService } from './bang';
import type { InstantService } from './instant';
import type { KnowledgeService } from './knowledge';
import type { MetaSearch } from '../engines/metasearch';
import type { Category, EngineParams, EngineResult, TimeRange } from '../engines/engine';

/**
 * Convert time range string to TimeRange type.
 */
function parseTimeRange(tr: string | undefined): TimeRange {
  if (tr === 'day' || tr === 'week' || tr === 'month' || tr === 'year') {
    return tr;
  }
  return '';
}

/**
 * Convert SearchOptions to EngineParams and Category for metasearch.
 */
function toEngineParams(options: SearchOptions): { category: Category; params: EngineParams } {
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
      engineData: {},
    },
  };
}

/**
 * Convert EngineResult to SearchResult format.
 */
function toSearchResult(r: EngineResult, index: number): SearchResult {
  return {
    id: `${Date.now().toString(36)}-${index}`,
    url: r.url,
    title: r.title,
    snippet: r.content,
    domain: extractDomain(r.url),
    thumbnail: r.thumbnailUrl ? { url: r.thumbnailUrl } : undefined,
    published: r.publishedAt,
    score: r.score,
    crawled_at: new Date().toISOString(),
    engine: r.engine,
    engines: [r.engine],
  };
}

function extractDomain(url: string): string {
  try {
    return new URL(url).hostname;
  } catch {
    return '';
  }
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

function hashSearchKey(query: string, options: SearchOptions): string {
  const key = `${query}|${options.page}|${options.per_page}|${options.time_range ?? ''}|${options.region ?? ''}|${options.language ?? ''}|${options.safe_search ?? ''}|${options.site ?? ''}|${options.lens ?? ''}`;
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

  constructor(
    metasearch: MetaSearch,
    cache: CacheStore,
    kvStore: KVStore,
    bangService: BangService,
    instantService: InstantService,
    knowledgeService: KnowledgeService,
  ) {
    this.metasearch = metasearch;
    this.cache = cache;
    this.kvStore = kvStore;
    this.bangService = bangService;
    this.instantService = instantService;
    this.knowledgeService = knowledgeService;
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
    const { category, params } = toEngineParams(options);
    const [instantAnswer, knowledgePanel, metaResult] = await Promise.all([
      this.detectInstantAnswer(searchQuery),
      options.page === 1 ? this.knowledgeService.getPanel(searchQuery) : Promise.resolve(null),
      this.metasearch.search(searchQuery, category, params),
    ]);

    // 4. Convert and paginate results
    const allResults = metaResult.results.map(toSearchResult);
    const startIndex = (options.page - 1) * options.per_page;
    const endIndex = startIndex + options.per_page;
    const paginatedResults = allResults.slice(startIndex, endIndex);
    const totalResults = allResults.length;
    const hasMore = endIndex < totalResults;

    const response: SearchResponse = {
      query: searchQuery,
      corrected_query: metaResult.corrections[0],
      total_results: totalResults,
      results: paginatedResults,
      suggestions: metaResult.suggestions,
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
   * Search for images.
   */
  async searchImages(query: string, options: SearchOptions): Promise<SearchResponse> {
    const startTime = Date.now();
    const imageOptions: SearchOptions = { ...options, file_type: 'image' };
    const cacheHash = hashSearchKey(`img:${query}`, imageOptions);

    const cachedResponse = await this.cache.getSearch(cacheHash);
    if (cachedResponse) {
      return cachedResponse;
    }

    const { params } = toEngineParams(imageOptions);
    const metaResult = await this.metasearch.search(query, 'images', params);
    const allResults = metaResult.results.map(toSearchResult);
    const startIndex = (options.page - 1) * options.per_page;
    const endIndex = startIndex + options.per_page;
    const paginatedResults = allResults.slice(startIndex, endIndex);
    const totalResults = allResults.length;
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
   * Search for videos.
   */
  async searchVideos(query: string, options: SearchOptions): Promise<SearchResponse> {
    const startTime = Date.now();
    const videoOptions: SearchOptions = { ...options, file_type: 'video' };
    const cacheHash = hashSearchKey(`vid:${query}`, videoOptions);

    const cachedResponse = await this.cache.getSearch(cacheHash);
    if (cachedResponse) {
      return cachedResponse;
    }

    const { params } = toEngineParams(videoOptions);
    const metaResult = await this.metasearch.search(query, 'videos', params);
    const allResults = metaResult.results.map(toSearchResult);
    const startIndex = (options.page - 1) * options.per_page;
    const endIndex = startIndex + options.per_page;
    const paginatedResults = allResults.slice(startIndex, endIndex);
    const totalResults = allResults.length;
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

    const { params } = toEngineParams(newsOptions);
    const metaResult = await this.metasearch.search(query, 'news', params);
    const allResults = metaResult.results.map(toSearchResult);
    const startIndex = (options.page - 1) * options.per_page;
    const endIndex = startIndex + options.per_page;
    const paginatedResults = allResults.slice(startIndex, endIndex);
    const totalResults = allResults.length;
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
