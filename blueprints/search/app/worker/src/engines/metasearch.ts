/**
 * MetaSearch - aggregates results from multiple search engines.
 *
 * Provides engine registry, parallel execution, deduplication,
 * score merging, and pagination.
 */

import type {
  OnlineEngine,
  EngineParams,
  EngineResult,
  EngineResults,
  Category,
} from './engine';
import { executeEngine } from './engine';
import { GoogleEngine, GoogleImagesEngine, GoogleReverseImageEngine } from './google';
import { BingEngine, BingImagesEngine, BingNewsEngine, BingReverseImageEngine } from './bing';
import {
  DuckDuckGoImagesEngine,
  DuckDuckGoVideosEngine,
  DuckDuckGoNewsEngine,
} from './duckduckgo';
import { BraveEngine } from './brave';
import { WikipediaEngine } from './wikipedia';
import { YouTubeEngine } from './youtube';
import { RedditEngine } from './reddit';
import { ArxivEngine } from './arxiv';
import { GitHubEngine } from './github';

// ========== MetaSearch Result ==========

export interface MetaSearchResult {
  results: EngineResult[];
  suggestions: string[];
  corrections: string[];
  totalEngines: number;
  successfulEngines: number;
  failedEngines: string[];
}

// ========== MetaSearch Class ==========

export class MetaSearch {
  private engines: Map<string, OnlineEngine> = new Map();

  /**
   * Register an engine in the registry.
   */
  register(engine: OnlineEngine): void {
    this.engines.set(engine.name, engine);
  }

  /**
   * Get an engine by name.
   */
  get(name: string): OnlineEngine | undefined {
    return this.engines.get(name);
  }

  /**
   * Get all engines for a given category.
   * Only returns enabled engines.
   */
  getByCategory(category: Category): OnlineEngine[] {
    const matched: OnlineEngine[] = [];
    for (const engine of this.engines.values()) {
      if (engine.disabled) continue;
      if (engine.categories.includes(category)) {
        matched.push(engine);
      }
    }
    return matched;
  }

  /**
   * Get all registered engine names.
   */
  listEngines(): string[] {
    return Array.from(this.engines.keys());
  }

  /**
   * Perform a metasearch across all engines in the given category.
   *
   * 1. Gets engines for the category
   * 2. Executes all in parallel with Promise.allSettled
   * 3. Collects results and suggestions
   * 4. Deduplicates by URL (merges scores for duplicates)
   * 5. Sorts by score descending
   * 6. Returns paginated results with metadata
   */
  async search(
    query: string,
    category: Category,
    params: EngineParams
  ): Promise<MetaSearchResult> {
    const engines = this.getByCategory(category);

    if (engines.length === 0) {
      return {
        results: [],
        suggestions: [],
        corrections: [],
        totalEngines: 0,
        successfulEngines: 0,
        failedEngines: [],
      };
    }

    // Execute all engines in parallel
    const promises = engines.map((engine) =>
      executeEngine(engine, query, params).then(
        (result) => ({ engine: engine.name, result, error: null }),
        (error) => ({
          engine: engine.name,
          result: null as EngineResults | null,
          error: error instanceof Error ? error.message : String(error),
        })
      )
    );

    const settled = await Promise.allSettled(promises);

    // Collect results
    const allResults: EngineResult[] = [];
    const allSuggestions: string[] = [];
    const allCorrections: string[] = [];
    const failedEngines: string[] = [];
    let successfulEngines = 0;

    for (const outcome of settled) {
      if (outcome.status === 'rejected') {
        continue;
      }

      const { engine: engineName, result, error } = outcome.value;

      if (error || !result) {
        failedEngines.push(engineName);
        continue;
      }

      successfulEngines++;
      allResults.push(...result.results);
      allSuggestions.push(...result.suggestions);
      allCorrections.push(...result.corrections);
    }

    // Deduplicate by URL (merge scores when duplicate)
    const deduped = this.deduplicateResults(allResults);

    // Sort by score descending
    deduped.sort((a, b) => b.score - a.score);

    // Deduplicate suggestions
    const uniqueSuggestions = [...new Set(allSuggestions)];
    const uniqueCorrections = [...new Set(allCorrections)];

    return {
      results: deduped,
      suggestions: uniqueSuggestions,
      corrections: uniqueCorrections,
      totalEngines: engines.length,
      successfulEngines,
      failedEngines,
    };
  }

  /**
   * Deduplicate results by URL.
   * When two results share the same URL, merge their scores
   * (additive) and keep the one with more content.
   */
  private deduplicateResults(results: EngineResult[]): EngineResult[] {
    const urlMap = new Map<string, EngineResult>();

    for (const result of results) {
      const normalizedUrl = this.normalizeUrl(result.url);

      if (!normalizedUrl) continue;

      const existing = urlMap.get(normalizedUrl);
      if (existing) {
        // Merge scores
        existing.score += result.score;

        // Keep longer content
        if (result.content.length > existing.content.length) {
          existing.content = result.content;
        }

        // Keep title if current is empty
        if (!existing.title && result.title) {
          existing.title = result.title;
        }

        // Merge thumbnail
        if (!existing.thumbnailUrl && result.thumbnailUrl) {
          existing.thumbnailUrl = result.thumbnailUrl;
        }
      } else {
        // Clone the result to avoid mutation
        urlMap.set(normalizedUrl, { ...result });
      }
    }

    return Array.from(urlMap.values());
  }

  /**
   * Normalize a URL for deduplication.
   * Strips trailing slashes, removes www. prefix, lowercases host.
   */
  private normalizeUrl(url: string): string {
    if (!url) return '';

    try {
      const parsed = new URL(url);
      let host = parsed.hostname.toLowerCase();
      if (host.startsWith('www.')) {
        host = host.slice(4);
      }
      let path = parsed.pathname;
      if (path.endsWith('/') && path.length > 1) {
        path = path.slice(0, -1);
      }
      return `${parsed.protocol}//${host}${path}${parsed.search}`;
    } catch {
      return url.toLowerCase();
    }
  }
}

/**
 * Create a MetaSearch instance pre-loaded with all built-in engines.
 */
export function createDefaultMetaSearch(): MetaSearch {
  const ms = new MetaSearch();

  // General search engines
  ms.register(new GoogleEngine());
  ms.register(new BingEngine());
  ms.register(new BraveEngine());
  ms.register(new WikipediaEngine());

  // Image search engines
  ms.register(new GoogleImagesEngine());
  ms.register(new BingImagesEngine());
  ms.register(new DuckDuckGoImagesEngine());

  // Reverse image search engines
  ms.register(new GoogleReverseImageEngine());
  ms.register(new BingReverseImageEngine());

  // Video search engines
  ms.register(new YouTubeEngine());
  ms.register(new DuckDuckGoVideosEngine());

  // News search engines
  ms.register(new BingNewsEngine());
  ms.register(new DuckDuckGoNewsEngine());

  // Specialized engines
  ms.register(new ArxivEngine());
  ms.register(new GitHubEngine());
  ms.register(new RedditEngine());

  return ms;
}

/**
 * Get the reverse image search engines.
 */
export function getReverseImageEngines(): OnlineEngine[] {
  return [
    new GoogleReverseImageEngine(),
    new BingReverseImageEngine(),
  ];
}
