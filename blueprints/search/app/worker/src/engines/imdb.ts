/**
 * IMDb Search Engine adapter.
 *
 * Searches IMDb for movies, TV shows, and celebrities.
 * Uses the public IMDb suggestion API.
 */

import type {
  OnlineEngine,
  EngineParams,
  RequestConfig,
  EngineResults,
  Category,
} from './engine';
import { newEngineResults } from './engine';
import { decodeHtmlEntities, extractText, findElements } from '../lib/html-parser';

// ========== IMDb API Types ==========

interface IMDbSuggestionItem {
  id: string;
  l: string; // Title/Name
  s?: string; // Secondary info (actors for movies, or description)
  y?: number; // Year
  yr?: string; // Year range (for TV shows: "2019-2022")
  q?: string; // Type: "feature", "TV series", "video", "TV movie", etc.
  rank?: number; // Popularity rank
  i?: {
    // Image info
    imageUrl: string;
    width: number;
    height: number;
  };
  v?: Array<{
    // Videos
    id: string;
    l: string;
    s: string;
  }>;
  vt?: number; // Video type
}

interface IMDbSuggestionResponse {
  d?: IMDbSuggestionItem[];
  q: string;
  v: number;
}

const USER_AGENT =
  'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36';

// Map IMDb type codes to human-readable labels
const typeLabels: Record<string, string> = {
  feature: 'Movie',
  'TV series': 'TV Series',
  'TV mini-series': 'TV Mini-Series',
  'TV movie': 'TV Movie',
  'TV episode': 'TV Episode',
  'TV short': 'TV Short',
  video: 'Video',
  short: 'Short Film',
  'video game': 'Video Game',
  podcast: 'Podcast',
  'podcast episode': 'Podcast Episode',
  music: 'Music Video',
  actor: 'Actor',
  actress: 'Actress',
  director: 'Director',
  writer: 'Writer',
  producer: 'Producer',
};

export class IMDbEngine implements OnlineEngine {
  name = 'imdb';
  shortcut = 'imdb';
  categories: Category[] = ['general', 'videos'];
  supportsPaging = false;
  maxPage = 1;
  timeout = 10_000;
  weight = 0.9;
  disabled = false;

  buildRequest(query: string, _params: EngineParams): RequestConfig {
    // Use the IMDb suggestion API
    // This returns up to 8 suggestions, but we can also try the search page
    const encodedQuery = encodeURIComponent(query);

    return {
      url: `https://v3.sg.media-imdb.com/suggestion/x/${encodedQuery}.json`,
      method: 'GET',
      headers: {
        Accept: 'application/json',
        'User-Agent': USER_AGENT,
        Origin: 'https://www.imdb.com',
        Referer: 'https://www.imdb.com/',
      },
      cookies: [],
    };
  }

  parseResponse(body: string, _params: EngineParams): EngineResults {
    const results = newEngineResults();

    try {
      const data: IMDbSuggestionResponse = JSON.parse(body);

      if (data.d && Array.isArray(data.d)) {
        for (const item of data.d) {
          const result = this.parseItem(item);
          if (result) {
            results.results.push(result);
          }
        }
      }
    } catch {
      // JSON parse failed
    }

    return results;
  }

  private parseItem(item: IMDbSuggestionItem): EngineResults['results'][0] | null {
    if (!item.id || !item.l) return null;

    // Determine if this is a title or name (person)
    const isTitle = item.id.startsWith('tt');
    const isPerson = item.id.startsWith('nm');

    // Build URL
    let url: string;
    if (isTitle) {
      url = `https://www.imdb.com/title/${item.id}/`;
    } else if (isPerson) {
      url = `https://www.imdb.com/name/${item.id}/`;
    } else {
      url = `https://www.imdb.com/${item.id}/`;
    }

    // Get type label
    const typeLabel = item.q ? (typeLabels[item.q] || item.q) : (isTitle ? 'Title' : 'Person');

    // Build content
    const contentParts: string[] = [];

    contentParts.push(typeLabel);

    if (item.y) {
      contentParts.push(item.y.toString());
    } else if (item.yr) {
      contentParts.push(item.yr);
    }

    if (item.s) {
      contentParts.push(item.s);
    }

    if (item.rank) {
      contentParts.push(`Rank #${item.rank}`);
    }

    const content = contentParts.join(' | ');

    // Get thumbnail
    let thumbnailUrl = '';
    if (item.i?.imageUrl) {
      // Modify URL to get smaller image
      thumbnailUrl = item.i.imageUrl.replace(/\._V1_.*\./, '._V1_UX300.');
    }

    return {
      url,
      title: decodeHtmlEntities(item.l),
      content,
      engine: this.name,
      score: this.weight,
      category: 'general',
      template: thumbnailUrl ? 'images' : undefined,
      thumbnailUrl: thumbnailUrl || undefined,
      channel: item.s || undefined,
      publishedAt: item.y ? `${item.y}-01-01T00:00:00Z` : undefined,
      source: 'IMDb',
      metadata: {
        imdbId: item.id,
        type: item.q,
        year: item.y,
        yearRange: item.yr,
        rank: item.rank,
        isTitle,
        isPerson,
        cast: item.s,
        hasVideos: item.v && item.v.length > 0,
        imageWidth: item.i?.width,
        imageHeight: item.i?.height,
      },
    };
  }
}

/**
 * IMDb Advanced Search Engine.
 * Uses the full search page for more comprehensive results.
 */
export class IMDbAdvancedEngine implements OnlineEngine {
  name = 'imdb advanced';
  shortcut = 'imdba';
  categories: Category[] = ['general', 'videos'];
  supportsPaging = true;
  maxPage = 5;
  timeout = 15_000;
  weight = 0.85;
  disabled = false;

  private searchType: 'all' | 'tt' | 'nm' | 'kw' | 'co';

  constructor(options?: { searchType?: 'all' | 'tt' | 'nm' | 'kw' | 'co' }) {
    this.searchType = options?.searchType ?? 'all';

    if (this.searchType !== 'all') {
      const typeNames: Record<string, string> = {
        tt: 'titles',
        nm: 'names',
        kw: 'keywords',
        co: 'companies',
      };
      this.name = `imdb ${typeNames[this.searchType]}`;
    }
  }

  buildRequest(query: string, params: EngineParams): RequestConfig {
    const searchParams = new URLSearchParams();
    searchParams.set('q', query);

    // Set search type if not 'all'
    if (this.searchType !== 'all') {
      searchParams.set('s', this.searchType);
    }

    // Pagination - IMDb uses ref for pagination
    if (params.page > 1) {
      searchParams.set('exact', 'true');
    }

    return {
      url: `https://www.imdb.com/find/?${searchParams.toString()}`,
      method: 'GET',
      headers: {
        Accept: 'text/html,application/xhtml+xml',
        'User-Agent': USER_AGENT,
        'Accept-Language': params.locale || 'en-US,en;q=0.9',
      },
      cookies: [],
    };
  }

  parseResponse(body: string, _params: EngineParams): EngineResults {
    const results = newEngineResults();

    // Parse JSON-LD data if available
    const jsonLdMatch = body.match(
      /<script type="application\/ld\+json">([\s\S]*?)<\/script>/
    );

    if (jsonLdMatch) {
      try {
        JSON.parse(jsonLdMatch[1]);
        // JSON-LD data available for future structured data extraction
      } catch {
        // Continue with HTML parsing
      }
    }

    // Parse search results from HTML
    this.parseHtmlResults(body, results);

    return results;
  }

  private parseHtmlResults(body: string, results: EngineResults): void {
    // Find result list items
    const resultItems = findElements(body, 'li.find-result-item');

    for (const item of resultItems.slice(0, 30)) {
      // Extract URL and ID
      const urlMatch = item.match(/href="(\/(?:title|name)\/([a-z]{2}\d+)\/[^"]*)"/);
      if (!urlMatch) continue;

      const path = urlMatch[1];
      const id = urlMatch[2];
      const url = `https://www.imdb.com${path}`;

      // Determine type
      const isTitle = id.startsWith('tt');

      // Extract title/name
      const titleMatch = item.match(/<a[^>]*class="[^"]*ipc-metadata-list-summary-item__t[^"]*"[^>]*>([^<]+)<\/a>/);
      const title = titleMatch ? titleMatch[1].trim() : '';

      if (!title) continue;

      // Extract year
      const yearMatch = item.match(/<span[^>]*class="[^"]*ipc-metadata-list-summary-item__li[^"]*"[^>]*>(\d{4})/);
      const year = yearMatch ? yearMatch[1] : '';

      // Extract type label
      const typeMatch = item.match(/<span[^>]*class="[^"]*ipc-metadata-list-summary-item__li[^"]*"[^>]*>([^<]+)<\/span>/);
      const typeLabel = typeMatch ? typeMatch[1].trim() : '';

      // Extract image
      const imgMatch = item.match(/<img[^>]+src="([^"]+)"/);
      let thumbnailUrl = imgMatch ? imgMatch[1] : '';
      if (thumbnailUrl) {
        thumbnailUrl = thumbnailUrl.replace(/\._V1_.*\./, '._V1_UX300.');
      }

      // Extract description/cast
      const descMatch = item.match(/<ul[^>]*class="[^"]*ipc-metadata-list-summary-item__stl[^"]*"[^>]*>([\s\S]*?)<\/ul>/);
      let description = '';
      if (descMatch) {
        description = extractText(descMatch[1]);
      }

      // Build content
      const contentParts: string[] = [];
      if (typeLabel && typeLabel !== year) {
        contentParts.push(typeLabel);
      }
      if (year) {
        contentParts.push(year);
      }
      if (description) {
        contentParts.push(description);
      }

      results.results.push({
        url,
        title: decodeHtmlEntities(title),
        content: contentParts.join(' | '),
        engine: this.name,
        score: this.weight,
        category: 'general',
        template: thumbnailUrl ? 'images' : undefined,
        thumbnailUrl: thumbnailUrl || undefined,
        publishedAt: year ? `${year}-01-01T00:00:00Z` : undefined,
        source: 'IMDb',
        metadata: {
          imdbId: id,
          isTitle,
          isPerson: !isTitle,
          year,
          typeLabel,
        },
      });
    }
  }
}

/**
 * IMDb Title Search Engine.
 * Searches only for movies and TV shows.
 */
export class IMDbTitleEngine extends IMDbAdvancedEngine {
  constructor() {
    super({ searchType: 'tt' });
  }
}

/**
 * IMDb Name Search Engine.
 * Searches only for people (actors, directors, etc.).
 */
export class IMDbNameEngine extends IMDbAdvancedEngine {
  constructor() {
    super({ searchType: 'nm' });
  }
}
