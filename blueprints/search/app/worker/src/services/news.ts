/**
 * News Service - handles news aggregation, personalization, and Full Coverage.
 */

import type {
  NewsArticle,
  NewsCategory,
  NewsHomeResponse,
  NewsCategoryResponse,
  NewsSearchResponse,
  NewsUserPreferences,
  StoryCluster,
  UserLocation,
} from '../types';
import type { EngineParams, EngineResult } from '../engines/engine';
import { executeEngine } from '../engines/engine';
import {
  GoogleNewsRSSEngine,
  GoogleNewsTopicEngine,
} from '../engines/google-news-rss';
import { BingNewsEngine } from '../engines/bing';
import { DuckDuckGoNewsEngine } from '../engines/duckduckgo-news';
import { NewsStore } from '../store/news-store';

// Cache TTL in seconds
const CACHE_TTL_HOME = 300; // 5 minutes
const CACHE_TTL_CATEGORY = 300; // 5 minutes
const CACHE_TTL_SEARCH = 600; // 10 minutes
const CACHE_TTL_STORY = 900; // 15 minutes

/**
 * Convert engine result to NewsArticle.
 */
function toNewsArticle(result: EngineResult, category: NewsCategory): NewsArticle {
  const id = generateArticleId(result.url);

  const metadata = result.metadata as Record<string, string> | undefined;
  return {
    id,
    url: result.url,
    title: result.title,
    snippet: result.content,
    source: result.source || extractDomain(result.url),
    sourceUrl: metadata?.sourceUrl || `https://${extractDomain(result.url)}`,
    sourceIcon: `https://www.google.com/s2/favicons?domain=${extractDomain(result.url)}&sz=32`,
    imageUrl: result.thumbnailUrl || result.imageUrl,
    publishedAt: result.publishedAt || new Date().toISOString(),
    category,
    engines: [result.engine],
    score: result.score,
    isBreaking: false,
    clusterId: metadata?.clusterId,
  };
}

/**
 * Generate article ID from URL.
 */
function generateArticleId(url: string): string {
  let hash = 0;
  for (let i = 0; i < url.length; i++) {
    const char = url.charCodeAt(i);
    hash = ((hash << 5) - hash) + char;
    hash = hash & hash;
  }
  return Math.abs(hash).toString(36);
}

/**
 * Extract domain from URL.
 */
function extractDomain(url: string): string {
  try {
    const parsed = new URL(url);
    return parsed.hostname.replace(/^www\./, '');
  } catch {
    return '';
  }
}

/**
 * Deduplicate articles by URL, merging scores.
 */
function deduplicateArticles(articles: NewsArticle[]): NewsArticle[] {
  const urlMap = new Map<string, NewsArticle>();

  for (const article of articles) {
    const normalizedUrl = normalizeUrl(article.url);
    if (!normalizedUrl) continue;

    const existing = urlMap.get(normalizedUrl);
    if (existing) {
      existing.score += article.score;
      if (!existing.engines.includes(article.engines[0])) {
        existing.engines.push(...article.engines);
      }
      if (article.snippet.length > existing.snippet.length) {
        existing.snippet = article.snippet;
      }
      if (!existing.imageUrl && article.imageUrl) {
        existing.imageUrl = article.imageUrl;
      }
    } else {
      urlMap.set(normalizedUrl, { ...article });
    }
  }

  return Array.from(urlMap.values());
}

/**
 * Normalize URL for deduplication.
 */
function normalizeUrl(url: string): string {
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
    return `${parsed.protocol}//${host}${path}`;
  } catch {
    return url.toLowerCase();
  }
}

/**
 * Sort articles by score and recency.
 */
function sortArticles(articles: NewsArticle[]): NewsArticle[] {
  return articles.sort((a, b) => {
    // Primary sort by score
    const scoreDiff = b.score - a.score;
    if (Math.abs(scoreDiff) > 0.1) return scoreDiff;

    // Secondary sort by recency
    const dateA = new Date(a.publishedAt).getTime();
    const dateB = new Date(b.publishedAt).getTime();
    return dateB - dateA;
  });
}

/**
 * News Service class.
 */
export class NewsService {
  private store: NewsStore;
  private cache: Cache | null = null;

  constructor(kv: KVNamespace) {
    this.store = new NewsStore(kv);
  }

  /**
   * Initialize cache (call once at startup).
   */
  async initCache(): Promise<void> {
    try {
      this.cache = await caches.open('news-cache');
    } catch {
      // Cache API not available (e.g., in tests)
      this.cache = null;
    }
  }

  /**
   * Get cached response or null.
   */
  private async getCached<T>(key: string): Promise<T | null> {
    if (!this.cache) return null;
    try {
      const response = await this.cache.match(new Request(`https://cache/${key}`));
      if (response) {
        return await response.json() as T;
      }
    } catch {
      // Cache miss or error
    }
    return null;
  }

  /**
   * Set cached response.
   */
  private async setCache(key: string, data: unknown, ttl: number): Promise<void> {
    if (!this.cache) return;
    try {
      await this.cache.put(
        new Request(`https://cache/${key}`),
        new Response(JSON.stringify(data), {
          headers: {
            'Content-Type': 'application/json',
            'Cache-Control': `max-age=${ttl}`,
          },
        })
      );
    } catch {
      // Cache write failed
    }
  }

  /**
   * Create engine params from user preferences.
   */
  private createEngineParams(
    prefs: NewsUserPreferences,
    page = 1,
    timeRange?: string
  ): EngineParams {
    return {
      page,
      locale: `${prefs.language}-${prefs.region}`,
      safeSearch: 1,
      timeRange: (timeRange || '') as EngineParams['timeRange'],
      engineData: {},
    };
  }

  /**
   * Fetch news from all engines for a category.
   */
  private async fetchCategoryNews(
    category: NewsCategory,
    params: EngineParams,
    query?: string
  ): Promise<NewsArticle[]> {
    const engines = [
      new GoogleNewsRSSEngine(category),
      new BingNewsEngine(),
      new DuckDuckGoNewsEngine(),
    ];

    const promises = engines.map((engine) =>
      executeEngine(engine, query || '', params)
        .then((result) => result.results)
        .catch(() => [] as EngineResult[])
    );

    const results = await Promise.all(promises);
    const allResults = results.flat();

    const articles = allResults.map((r) => toNewsArticle(r, category));
    const deduped = deduplicateArticles(articles);
    return sortArticles(deduped);
  }

  /**
   * Fetch local news for a location.
   */
  private async fetchLocalNews(
    location: UserLocation,
    params: EngineParams
  ): Promise<NewsArticle[]> {
    const locationQuery = location.state
      ? `${location.city}, ${location.state}`
      : `${location.city}, ${location.country}`;

    const engines = [
      new BingNewsEngine(),
      new DuckDuckGoNewsEngine(),
    ];

    const promises = engines.map((engine) =>
      executeEngine(engine, `${locationQuery} news`, params)
        .then((result) => result.results)
        .catch(() => [] as EngineResult[])
    );

    const results = await Promise.all(promises);
    const allResults = results.flat();

    const articles = allResults.map((r) => toNewsArticle(r, 'top'));
    const deduped = deduplicateArticles(articles);
    return sortArticles(deduped).slice(0, 10);
  }

  /**
   * Get news home feed.
   */
  async getHomeFeed(userId: string): Promise<NewsHomeResponse> {
    const startTime = Date.now();
    const prefs = await this.store.getPreferences(userId);

    // Check cache
    const cacheKey = `home:${userId}:${prefs.language}:${prefs.region}`;
    const cached = await this.getCached<NewsHomeResponse>(cacheKey);
    if (cached) {
      return cached;
    }

    const params = this.createEngineParams(prefs);

    // Fetch in parallel
    const [topStories, worldNews, businessNews, techNews, localNews] = await Promise.all([
      this.fetchCategoryNews('top', params),
      this.fetchCategoryNews('world', params),
      this.fetchCategoryNews('business', params),
      this.fetchCategoryNews('technology', params),
      prefs.location ? this.fetchLocalNews(prefs.location, params) : Promise.resolve([]),
    ]);

    // Apply personalization to create "For You" feed
    const forYou = this.createPersonalizedFeed(
      [...topStories, ...worldNews, ...businessNews, ...techNews],
      prefs
    );

    // Filter out hidden sources
    const filterHidden = (articles: NewsArticle[]) =>
      articles.filter(
        (a) => !prefs.hiddenSources.some(
          (s) => a.source.toLowerCase().includes(s.toLowerCase())
        )
      );

    const response: NewsHomeResponse = {
      topStories: filterHidden(topStories).slice(0, 15),
      forYou: filterHidden(forYou).slice(0, 20),
      localNews: filterHidden(localNews),
      categories: {
        world: filterHidden(worldNews).slice(0, 5),
        business: filterHidden(businessNews).slice(0, 5),
        technology: filterHidden(techNews).slice(0, 5),
      },
      searchTimeMs: Date.now() - startTime,
    };

    // Cache the response
    await this.setCache(cacheKey, response, CACHE_TTL_HOME);

    return response;
  }

  /**
   * Create personalized "For You" feed based on user preferences.
   */
  private createPersonalizedFeed(
    articles: NewsArticle[],
    prefs: NewsUserPreferences
  ): NewsArticle[] {
    // Score articles based on preferences
    const scored = articles.map((article) => ({
      ...article,
      score: this.store.computePersonalizedScore(article, prefs),
    }));

    // Filter out zero-scored (hidden) articles
    const filtered = scored.filter((a) => a.score > 0);

    // Sort by personalized score
    return sortArticles(deduplicateArticles(filtered));
  }

  /**
   * Get category feed.
   */
  async getCategoryFeed(
    userId: string,
    category: NewsCategory,
    page = 1
  ): Promise<NewsCategoryResponse> {
    const startTime = Date.now();
    const prefs = await this.store.getPreferences(userId);

    // Check cache (only cache first page)
    const cacheKey = `category:${category}:${prefs.language}:${prefs.region}:${page}`;
    if (page === 1) {
      const cached = await this.getCached<NewsCategoryResponse>(cacheKey);
      if (cached) {
        return cached;
      }
    }

    const params = this.createEngineParams(prefs, page);
    const articles = await this.fetchCategoryNews(category, params);

    // Filter hidden sources
    const filtered = articles.filter(
      (a) => !prefs.hiddenSources.some(
        (s) => a.source.toLowerCase().includes(s.toLowerCase())
      )
    );

    const response: NewsCategoryResponse = {
      category,
      articles: filtered.slice(0, 30),
      hasMore: filtered.length > 30,
      page,
      searchTimeMs: Date.now() - startTime,
    };

    // Cache first page
    if (page === 1) {
      await this.setCache(cacheKey, response, CACHE_TTL_CATEGORY);
    }

    return response;
  }

  /**
   * Search news.
   */
  async searchNews(
    userId: string,
    query: string,
    options: {
      page?: number;
      timeRange?: string;
      source?: string;
      category?: NewsCategory;
    } = {}
  ): Promise<NewsSearchResponse> {
    const startTime = Date.now();
    const prefs = await this.store.getPreferences(userId);

    const page = options.page || 1;
    const params = this.createEngineParams(prefs, page, options.timeRange);

    // Check cache
    const cacheKey = `search:${query}:${page}:${options.timeRange || ''}:${prefs.language}`;
    const cached = await this.getCached<NewsSearchResponse>(cacheKey);
    if (cached) {
      return cached;
    }

    // Fetch from engines
    const engines = [
      new BingNewsEngine(),
      new DuckDuckGoNewsEngine(),
    ];

    const promises = engines.map((engine) =>
      executeEngine(engine, query, params)
        .then((result) => result.results)
        .catch(() => [] as EngineResult[])
    );

    const results = await Promise.all(promises);
    const allResults = results.flat();

    let articles = allResults.map((r) => toNewsArticle(r, options.category || 'top'));
    articles = deduplicateArticles(articles);

    // Filter by source if specified
    if (options.source) {
      articles = articles.filter((a) =>
        a.source.toLowerCase().includes(options.source!.toLowerCase())
      );
    }

    // Filter hidden sources
    articles = articles.filter(
      (a) => !prefs.hiddenSources.some(
        (s) => a.source.toLowerCase().includes(s.toLowerCase())
      )
    );

    articles = sortArticles(articles);

    const response: NewsSearchResponse = {
      query,
      results: articles.slice(0, 30),
      totalResults: articles.length,
      searchTimeMs: Date.now() - startTime,
      page,
      hasMore: articles.length > 30,
    };

    await this.setCache(cacheKey, response, CACHE_TTL_SEARCH);

    return response;
  }

  /**
   * Get Full Coverage for a story.
   */
  async getFullCoverage(
    userId: string,
    storyId: string
  ): Promise<StoryCluster | null> {
    const prefs = await this.store.getPreferences(userId);

    // Check cache
    const cacheKey = `story:${storyId}:${prefs.language}`;
    const cached = await this.getCached<StoryCluster>(cacheKey);
    if (cached) {
      return cached;
    }

    // Fetch from Google News Topic feed
    const engine = new GoogleNewsTopicEngine(storyId);
    const params = this.createEngineParams(prefs);

    try {
      const result = await executeEngine(engine, '', params);
      if (!result.results.length) {
        return null;
      }

      const articles = result.results.map((r) => toNewsArticle(r, 'top'));
      const deduped = deduplicateArticles(articles);

      // Group by perspective (opinion vs news)
      const opinions = deduped.filter((a) =>
        a.title.toLowerCase().includes('opinion') ||
        a.title.toLowerCase().includes('editorial') ||
        a.snippet.toLowerCase().includes('opinion')
      );

      const news = deduped.filter((a) => !opinions.includes(a));

      const cluster: StoryCluster = {
        id: storyId,
        title: deduped[0]?.title || 'Story',
        summary: deduped[0]?.snippet || '',
        articles: sortArticles(deduped),
        perspectives: [
          { label: 'All coverage', articles: sortArticles(news) },
          ...(opinions.length > 0 ? [{ label: 'Opinion', articles: sortArticles(opinions) }] : []),
        ],
        updatedAt: new Date().toISOString(),
      };

      await this.setCache(cacheKey, cluster, CACHE_TTL_STORY);

      return cluster;
    } catch {
      return null;
    }
  }

  /**
   * Get local news.
   */
  async getLocalNews(
    userId: string,
    location?: UserLocation
  ): Promise<NewsArticle[]> {
    const prefs = await this.store.getPreferences(userId);
    const loc = location || prefs.location;

    if (!loc) {
      return [];
    }

    const params = this.createEngineParams(prefs);
    return this.fetchLocalNews(loc, params);
  }

  /**
   * Get following feed (topics and sources user follows).
   */
  async getFollowingFeed(userId: string): Promise<NewsArticle[]> {
    const prefs = await this.store.getPreferences(userId);

    if (prefs.followedTopics.length === 0 && prefs.followedSources.length === 0) {
      return [];
    }

    const params = this.createEngineParams(prefs);
    const allArticles: NewsArticle[] = [];

    // Fetch for followed topics
    for (const topic of prefs.followedTopics.slice(0, 5)) {
      // Map topic to category if possible
      const category = topic as NewsCategory;
      if (['world', 'business', 'technology', 'science', 'health', 'sports', 'entertainment'].includes(category)) {
        const articles = await this.fetchCategoryNews(category, params);
        allArticles.push(...articles.slice(0, 5));
      }
    }

    // For followed sources, search by source name
    for (const source of prefs.followedSources.slice(0, 3)) {
      const response = await this.searchNews(userId, source, { page: 1 });
      allArticles.push(...response.results.slice(0, 5));
    }

    return sortArticles(deduplicateArticles(allArticles)).slice(0, 30);
  }

  /**
   * Get user preferences.
   */
  async getPreferences(userId: string): Promise<NewsUserPreferences> {
    return this.store.getPreferences(userId);
  }

  /**
   * Update user preferences.
   */
  async updatePreferences(
    userId: string,
    updates: Partial<NewsUserPreferences>
  ): Promise<NewsUserPreferences> {
    return this.store.updatePreferences(userId, updates);
  }

  /**
   * Follow a topic.
   */
  async followTopic(userId: string, topic: string): Promise<void> {
    return this.store.followTopic(userId, topic);
  }

  /**
   * Unfollow a topic.
   */
  async unfollowTopic(userId: string, topic: string): Promise<void> {
    return this.store.unfollowTopic(userId, topic);
  }

  /**
   * Follow a source.
   */
  async followSource(userId: string, source: string): Promise<void> {
    return this.store.followSource(userId, source);
  }

  /**
   * Unfollow a source.
   */
  async unfollowSource(userId: string, source: string): Promise<void> {
    return this.store.unfollowSource(userId, source);
  }

  /**
   * Hide a source.
   */
  async hideSource(userId: string, source: string): Promise<void> {
    return this.store.hideSource(userId, source);
  }

  /**
   * Unhide a source.
   */
  async unhideSource(userId: string, source: string): Promise<void> {
    return this.store.unhideSource(userId, source);
  }

  /**
   * Set user location.
   */
  async setLocation(userId: string, location: UserLocation): Promise<void> {
    return this.store.setLocation(userId, location);
  }

  /**
   * Record article read for personalization.
   */
  async recordRead(
    userId: string,
    article: NewsArticle,
    duration?: number
  ): Promise<void> {
    return this.store.addToReadingHistory(userId, {
      articleId: article.id,
      url: article.url,
      category: article.category,
      source: article.source,
      timestamp: new Date().toISOString(),
      duration,
    });
  }
}
