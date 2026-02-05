import type { Suggestion } from '../types';
import type { CacheStore } from '../store/cache';

const TRENDING_SUGGESTIONS: Suggestion[] = [
  { text: 'artificial intelligence news', type: 'trending', frequency: 95 },
  { text: 'climate change solutions', type: 'trending', frequency: 88 },
  { text: 'programming tutorials', type: 'trending', frequency: 85 },
  { text: 'space exploration updates', type: 'trending', frequency: 82 },
  { text: 'healthy recipes', type: 'trending', frequency: 78 },
  { text: 'web development frameworks', type: 'trending', frequency: 76 },
  { text: 'machine learning projects', type: 'trending', frequency: 74 },
  { text: 'renewable energy technology', type: 'trending', frequency: 72 },
  { text: 'open source software', type: 'trending', frequency: 70 },
  { text: 'cybersecurity best practices', type: 'trending', frequency: 68 },
];

export class SuggestService {
  private cache: CacheStore;

  constructor(cache: CacheStore) {
    this.cache = cache;
  }

  /**
   * Get search suggestions for a query.
   * Checks cache first, then fetches from Google suggestions API.
   */
  async suggest(query: string): Promise<Suggestion[]> {
    const trimmed = query.trim();
    if (!trimmed) {
      return [];
    }

    // Build cache key from the query
    const hash = this.hashQuery(trimmed);

    // Check cache
    const cached = await this.cache.getSuggest(hash);
    if (cached) {
      return cached;
    }

    // Fetch suggestions from Google
    const encoded = encodeURIComponent(trimmed);
    const url = `https://suggestqueries.google.com/complete/search?client=firefox&q=${encoded}`;

    try {
      const response = await fetch(url, {
        headers: {
          'User-Agent': 'Mozilla/5.0 (compatible; mizu-search/1.0)',
        },
      });

      if (!response.ok) {
        return [];
      }

      // Google returns JSON: [query, [suggestions], [], { "google:suggesttype": [...] }]
      const data = (await response.json()) as [string, string[], ...unknown[]];

      if (!Array.isArray(data) || !Array.isArray(data[1])) {
        return [];
      }

      const suggestions: Suggestion[] = data[1].map((text: string) => ({
        text,
        type: 'query' as const,
      }));

      // Cache the suggestions
      await this.cache.setSuggest(hash, suggestions);

      return suggestions;
    } catch {
      // On network errors, return empty rather than throwing
      return [];
    }
  }

  /**
   * Return a list of trending search suggestions.
   */
  async trending(): Promise<Suggestion[]> {
    return [...TRENDING_SUGGESTIONS];
  }

  /**
   * Simple hash function for cache keys.
   * Uses a basic string hash suitable for KV cache keys.
   */
  private hashQuery(query: string): string {
    let hash = 0;
    for (let i = 0; i < query.length; i++) {
      const char = query.charCodeAt(i);
      hash = ((hash << 5) - hash + char) | 0;
    }
    return Math.abs(hash).toString(36);
  }
}
