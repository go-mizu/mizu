/**
 * Base engine class providing common functionality for all search engine adapters.
 * Reduces boilerplate across the 90+ engine implementations.
 */

import type {
  OnlineEngine,
  EngineParams,
  RequestConfig,
  EngineResults,
  Category,
} from './engine';
import { newEngineResults } from './engine';

// ========== User Agent Pool ==========

const DESKTOP_USER_AGENTS = [
  'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36',
  'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36',
  'Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:121.0) Gecko/20100101 Firefox/121.0',
  'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.2 Safari/605.1.15',
];

const MOBILE_USER_AGENTS = [
  'Mozilla/5.0 (iPhone; CPU iPhone OS 17_2 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.2 Mobile/15E148 Safari/604.1',
  'Mozilla/5.0 (Linux; Android 14; Pixel 8) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.6099.144 Mobile Safari/537.36',
];

/**
 * Get a random user agent from the pool.
 */
export function getRandomUserAgent(mobile = false): string {
  const pool = mobile ? MOBILE_USER_AGENTS : DESKTOP_USER_AGENTS;
  return pool[Math.floor(Math.random() * pool.length)];
}

// ========== Engine Configuration ==========

export interface EngineConfig {
  /** Unique engine name */
  name: string;
  /** Short identifier for bang commands (e.g., 'g' for google) */
  shortcut: string;
  /** Categories this engine supports */
  categories: Category[];
  /** Whether this engine supports pagination (default: true) */
  supportsPaging?: boolean;
  /** Maximum page number supported (default: 10) */
  maxPage?: number;
  /** Request timeout in milliseconds (default: 10000) */
  timeout?: number;
  /** Result weight for scoring (default: 1.0) */
  weight?: number;
  /** Whether engine is disabled (default: false) */
  disabled?: boolean;
}

// ========== Base Engine Class ==========

/**
 * Abstract base class for all search engine adapters.
 * Provides common functionality and reduces boilerplate.
 *
 * @example
 * ```typescript
 * export class MyEngine extends BaseEngine {
 *   constructor() {
 *     super({
 *       name: 'myengine',
 *       shortcut: 'me',
 *       categories: ['general'],
 *     });
 *   }
 *
 *   buildRequest(query: string, params: EngineParams): RequestConfig {
 *     return this.createRequest({
 *       url: this.buildUrl('https://api.example.com/search', { q: query }),
 *     });
 *   }
 *
 *   parseResponse(body: string, params: EngineParams): EngineResults {
 *     const results = this.createResults();
 *     // Parse body and populate results
 *     return results;
 *   }
 * }
 * ```
 */
export abstract class BaseEngine implements OnlineEngine {
  readonly name: string;
  readonly shortcut: string;
  readonly categories: Category[];
  readonly supportsPaging: boolean;
  readonly maxPage: number;
  readonly timeout: number;
  readonly weight: number;
  readonly disabled: boolean;

  protected constructor(config: EngineConfig) {
    this.name = config.name;
    this.shortcut = config.shortcut;
    this.categories = config.categories;
    this.supportsPaging = config.supportsPaging ?? true;
    this.maxPage = config.maxPage ?? 10;
    this.timeout = config.timeout ?? 10_000;
    this.weight = config.weight ?? 1.0;
    this.disabled = config.disabled ?? false;
  }

  /**
   * Build the HTTP request configuration for a search query.
   * Must be implemented by subclasses.
   */
  abstract buildRequest(query: string, params: EngineParams): RequestConfig;

  /**
   * Parse the HTTP response body and extract results.
   * Must be implemented by subclasses.
   */
  abstract parseResponse(body: string, params: EngineParams): EngineResults;

  // ========== Helper Methods ==========

  /**
   * Create a new empty EngineResults object.
   */
  protected createResults(): EngineResults {
    return newEngineResults();
  }

  /**
   * Create a basic RequestConfig with sensible defaults.
   */
  protected createRequest(
    options: {
      url: string;
      method?: string;
      headers?: Record<string, string>;
      cookies?: string[];
      body?: string;
      mobile?: boolean;
    }
  ): RequestConfig {
    return {
      url: options.url,
      method: options.method ?? 'GET',
      headers: {
        'User-Agent': getRandomUserAgent(options.mobile),
        'Accept': 'text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8',
        'Accept-Language': 'en-US,en;q=0.5',
        ...options.headers,
      },
      cookies: options.cookies ?? [],
      body: options.body,
    };
  }

  /**
   * Create a JSON API request.
   */
  protected createJsonRequest(
    options: {
      url: string;
      method?: string;
      headers?: Record<string, string>;
      body?: unknown;
    }
  ): RequestConfig {
    const headers: Record<string, string> = {
      'Accept': 'application/json',
      ...options.headers,
    };

    let body: string | undefined;
    if (options.body) {
      headers['Content-Type'] = 'application/json';
      body = JSON.stringify(options.body);
    }

    return this.createRequest({
      url: options.url,
      method: options.method ?? (options.body ? 'POST' : 'GET'),
      headers,
      body,
    });
  }

  /**
   * Build a URL with query parameters.
   */
  protected buildUrl(base: string, params: Record<string, string | number | boolean | undefined>): string {
    const url = new URL(base);
    for (const [key, value] of Object.entries(params)) {
      if (value !== undefined && value !== null && value !== '') {
        url.searchParams.set(key, String(value));
      }
    }
    return url.toString();
  }

  /**
   * Add a result to the results collection.
   */
  protected addResult(
    results: EngineResults,
    result: Partial<EngineResults['results'][number]> & { url: string; title: string }
  ): void {
    const { url, title, content, score, category, ...rest } = result;
    results.results.push({
      url,
      title,
      content: content ?? '',
      engine: this.name,
      score: score ?? this.weight,
      category: category ?? this.categories[0],
      ...rest,
    });
  }

  /**
   * Add multiple suggestions to the results.
   */
  protected addSuggestions(results: EngineResults, suggestions: string[]): void {
    results.suggestions.push(...suggestions.filter(Boolean));
  }

  /**
   * Add a correction to the results.
   */
  protected addCorrection(results: EngineResults, correction: string): void {
    if (correction) {
      results.corrections.push(correction);
    }
  }

  /**
   * Calculate the start offset for pagination.
   */
  protected getOffset(params: EngineParams, resultsPerPage = 10): number {
    return (params.page - 1) * resultsPerPage;
  }

  /**
   * Get the language code from locale (e.g., 'en-US' -> 'en').
   */
  protected getLanguage(params: EngineParams): string {
    return params.locale?.split('-')[0] ?? 'en';
  }

  /**
   * Get the region code from locale (e.g., 'en-US' -> 'US').
   */
  protected getRegion(params: EngineParams): string {
    return params.locale?.split('-')[1] ?? 'US';
  }

  /**
   * Get safe search value as a string for common API formats.
   */
  protected getSafeSearch(params: EngineParams): 'off' | 'moderate' | 'strict' {
    switch (params.safeSearch) {
      case 0: return 'off';
      case 2: return 'strict';
      default: return 'moderate';
    }
  }

  /**
   * Truncate text to a maximum length with ellipsis.
   */
  protected truncate(text: string, maxLength: number): string {
    if (text.length <= maxLength) return text;
    return text.slice(0, maxLength - 3) + '...';
  }

  /**
   * Clean whitespace from text.
   */
  protected cleanText(text: string): string {
    return text.replace(/\s+/g, ' ').trim();
  }

  /**
   * Extract domain from a URL.
   */
  protected extractDomain(url: string): string {
    try {
      return new URL(url).hostname;
    } catch {
      return '';
    }
  }

  /**
   * Check if a URL is valid.
   */
  protected isValidUrl(url: string): boolean {
    try {
      new URL(url);
      return true;
    } catch {
      return false;
    }
  }

  /**
   * Parse a duration string to seconds.
   * Supports formats: "1:23", "1:23:45", "PT1H23M45S", "90 seconds", etc.
   */
  protected parseDuration(duration: string | undefined): number {
    if (!duration) return 0;

    // HH:MM:SS or MM:SS format
    const colonMatch = duration.match(/^(\d+):(\d+)(?::(\d+))?$/);
    if (colonMatch) {
      const parts = colonMatch.slice(1).filter(Boolean).map(Number);
      if (parts.length === 3) return parts[0] * 3600 + parts[1] * 60 + parts[2];
      if (parts.length === 2) return parts[0] * 60 + parts[1];
    }

    // ISO 8601 duration (PT1H23M45S)
    const isoMatch = duration.match(/PT(?:(\d+)H)?(?:(\d+)M)?(?:(\d+)S)?/i);
    if (isoMatch) {
      const hours = parseInt(isoMatch[1] ?? '0', 10);
      const minutes = parseInt(isoMatch[2] ?? '0', 10);
      const seconds = parseInt(isoMatch[3] ?? '0', 10);
      return hours * 3600 + minutes * 60 + seconds;
    }

    // Try to parse as plain number
    const num = parseInt(duration, 10);
    return isNaN(num) ? 0 : num;
  }

  /**
   * Format a number with metric suffixes (K, M, B).
   */
  protected formatNumber(num: number | undefined): string {
    if (num === undefined || num === null) return '';
    if (num >= 1_000_000_000) return `${(num / 1_000_000_000).toFixed(1)}B`;
    if (num >= 1_000_000) return `${(num / 1_000_000).toFixed(1)}M`;
    if (num >= 1_000) return `${(num / 1_000).toFixed(1)}K`;
    return String(num);
  }

  /**
   * Format a date string as relative time.
   */
  protected formatRelativeTime(dateStr: string | undefined): string {
    if (!dateStr) return '';

    const date = new Date(dateStr);
    if (isNaN(date.getTime())) return '';

    const now = new Date();
    const diffMs = now.getTime() - date.getTime();
    const diffDays = Math.floor(diffMs / (1000 * 60 * 60 * 24));

    if (diffDays < 1) return 'Today';
    if (diffDays === 1) return 'Yesterday';
    if (diffDays < 7) return `${diffDays} days ago`;
    if (diffDays < 30) return `${Math.floor(diffDays / 7)} weeks ago`;
    if (diffDays < 365) return `${Math.floor(diffDays / 30)} months ago`;
    return `${Math.floor(diffDays / 365)} years ago`;
  }
}
