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

// === Web Search Engines ===
import { GoogleEngine, GoogleImagesEngine, GoogleReverseImageEngine } from './google';
import { BingEngine, BingImagesEngine, BingNewsEngine, BingReverseImageEngine } from './bing';
import {
  DuckDuckGoImagesEngine,
  DuckDuckGoVideosEngine,
  DuckDuckGoNewsEngine,
} from './duckduckgo';
import { BraveEngine } from './brave';
import { YahooEngine } from './yahoo';
import { YandexEngine } from './yandex';
import { MojeekEngine } from './mojeek';
import { StartpageEngine } from './startpage';

// === Reference Engines ===
import { WikipediaEngine } from './wikipedia';

// === Video Engines ===
import { YouTubeEngine } from './youtube';
import { VimeoEngine } from './vimeo';
import { DailymotionEngine } from './dailymotion';
import { GoogleVideosEngine } from './google-videos';
import { BingVideosEngine } from './bing-videos';
import { PeerTubeEngine } from './peertube';
import { Search360VideosEngine } from './360search-videos';
import { SogouVideosEngine } from './sogou-videos';
import { RumbleEngine } from './rumble';
import { OdyseeEngine } from './odysee';
import { BilibiliEngine } from './bilibili';

// === Image Engines ===
import { FlickrEngine } from './flickr';
import { UnsplashEngine } from './unsplash';
import { PixabayEngine } from './pixabay';
import { DeviantArtEngine } from './deviantart';
import { ImgurEngine } from './imgur';

// === News Engines ===
import { YahooNewsEngine } from './yahoo-news';
import { ReutersEngine } from './reuters';
import { HackerNewsEngine } from './hackernews';

// === Academic Engines ===
import { ArxivEngine } from './arxiv';
import { PubMedEngine } from './pubmed';
import { SemanticScholarEngine } from './semantic-scholar';
import { CrossrefEngine } from './crossref';
import { OpenLibraryEngine } from './openlibrary';

// === Code/IT Engines ===
import { GitHubEngine } from './github';
import { GitLabEngine } from './gitlab';
import { StackOverflowEngine } from './stackoverflow';
import { NpmEngine } from './npm';
import { PyPIEngine } from './pypi';
import { CratesEngine } from './crates';
import { PkgGoDevEngine } from './pkg-go-dev';

// === Social Engines ===
import { RedditEngine } from './reddit';
import { MastodonEngine } from './mastodon';
import { LemmyEngine } from './lemmy';

// === Music Engines ===
import { SoundCloudEngine } from './soundcloud';
import { BandcampEngine } from './bandcamp';
import { GeniusEngine } from './genius';

// === Maps Engines ===
import { OpenStreetMapEngine } from './openstreetmap';

// === Entertainment Engines ===
import { IMDbEngine } from './imdb';

// === AI-powered Engines ===
import { JinaSearchEngine } from './jina';

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
    params: EngineParams,
    options?: { engines?: string[]; maxWait?: number }
  ): Promise<MetaSearchResult> {
    const engines = this.getByCategory(category).filter((engine) => {
      if (!options?.engines || options.engines.length === 0) return true;
      return options.engines.includes(engine.name);
    });

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

    // Default deadline: 8 seconds. Fast engines return quickly;
    // slow engines (e.g., Jina) are included only if they respond in time.
    const maxWait = options?.maxWait ?? 8_000;

    type EngineOutcome = {
      engine: string;
      result: EngineResults | null;
      error: string | null;
    };

    // Execute all engines in parallel
    const promises = engines.map((engine) =>
      executeEngine(engine, query, params).then(
        (result): EngineOutcome => ({ engine: engine.name, result, error: null }),
        (error): EngineOutcome => ({
          engine: engine.name,
          result: null,
          error: error instanceof Error ? error.message : String(error),
        })
      )
    );

    // Race: collect results as they arrive, stop after deadline
    const outcomes: EngineOutcome[] = [];
    const remaining = new Set(promises);

    await Promise.race([
      // Resolve when all engines complete
      (async () => {
        for (const p of promises) {
          p.then((outcome) => {
            outcomes.push(outcome);
            remaining.delete(p);
          });
        }
        await Promise.allSettled(promises);
      })(),
      // Or stop collecting after the deadline
      new Promise<void>((resolve) => setTimeout(resolve, maxWait)),
    ]);

    // Collect results from engines that completed in time
    const allResults: EngineResult[] = [];
    const allSuggestions: string[] = [];
    const allCorrections: string[] = [];
    const failedEngines: string[] = [];
    let successfulEngines = 0;

    for (const outcome of outcomes) {
      if (outcome.error || !outcome.result) {
        failedEngines.push(outcome.engine);
        continue;
      }

      successfulEngines++;
      allResults.push(...outcome.result.results);
      allSuggestions.push(...outcome.result.suggestions);
      allCorrections.push(...outcome.result.corrections);
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

  // === General/Web Search Engines ===
  ms.register(new GoogleEngine());
  ms.register(new BingEngine());
  ms.register(new BraveEngine());
  ms.register(new YahooEngine());
  ms.register(new YandexEngine());
  ms.register(new MojeekEngine());
  ms.register(new StartpageEngine());
  ms.register(new WikipediaEngine());

  // === Image Search Engines ===
  ms.register(new GoogleImagesEngine());
  ms.register(new BingImagesEngine());
  ms.register(new DuckDuckGoImagesEngine());
  ms.register(new FlickrEngine());
  ms.register(new UnsplashEngine());
  ms.register(new PixabayEngine());
  ms.register(new DeviantArtEngine());
  ms.register(new ImgurEngine());

  // === Reverse Image Search Engines ===
  ms.register(new GoogleReverseImageEngine());
  ms.register(new BingReverseImageEngine());

  // === Video Search Engines ===
  ms.register(new YouTubeEngine());
  ms.register(new DuckDuckGoVideosEngine());
  ms.register(new VimeoEngine());
  ms.register(new DailymotionEngine());
  ms.register(new GoogleVideosEngine());
  ms.register(new BingVideosEngine());
  ms.register(new PeerTubeEngine());
  ms.register(new Search360VideosEngine());
  ms.register(new SogouVideosEngine());
  ms.register(new RumbleEngine());
  ms.register(new OdyseeEngine());
  ms.register(new BilibiliEngine());

  // === News Search Engines ===
  ms.register(new BingNewsEngine());
  ms.register(new DuckDuckGoNewsEngine());
  ms.register(new YahooNewsEngine());
  ms.register(new ReutersEngine());
  ms.register(new HackerNewsEngine());

  // === Academic/Science Engines ===
  ms.register(new ArxivEngine());
  ms.register(new PubMedEngine());
  ms.register(new SemanticScholarEngine());
  ms.register(new CrossrefEngine());
  ms.register(new OpenLibraryEngine());

  // === Code/IT Engines ===
  ms.register(new GitHubEngine());
  ms.register(new GitLabEngine());
  ms.register(new StackOverflowEngine());
  ms.register(new NpmEngine());
  ms.register(new PyPIEngine());
  ms.register(new CratesEngine());
  ms.register(new PkgGoDevEngine());

  // === Social Engines ===
  ms.register(new RedditEngine());
  ms.register(new MastodonEngine());
  ms.register(new LemmyEngine());

  // === Music Engines ===
  ms.register(new SoundCloudEngine());
  ms.register(new BandcampEngine());
  ms.register(new GeniusEngine());

  // === Maps Engines ===
  ms.register(new OpenStreetMapEngine());

  // === Entertainment Engines ===
  ms.register(new IMDbEngine());

  // === AI-powered Engines ===
  ms.register(new JinaSearchEngine());

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
