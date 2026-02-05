/**
 * Bandcamp Search Engine adapter.
 *
 * Searches Bandcamp for tracks, albums, and artists.
 * Scrapes search results from the Bandcamp search page.
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

// ========== Bandcamp Search Types ==========

interface BandcampSearchItem {
  type: 'track' | 'album' | 'artist' | 'label';
  id: number;
  name: string;
  url: string;
  img?: string;
  art_id?: number;
  artist?: string;
  album?: string;
  genre?: string;
  location?: string;
  tags?: string[];
}

interface BandcampSearchData {
  auto: {
    results: BandcampSearchItem[];
  };
}

const USER_AGENT =
  'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36';

// Map search types to Bandcamp item_type parameter
type BandcampItemType = 't' | 'a' | 'b' | 'l'; // track, album, band/artist, label

export class BandcampEngine implements OnlineEngine {
  name = 'bandcamp';
  shortcut = 'bc';
  categories: Category[] = ['videos']; // Using 'videos' category for audio/media
  supportsPaging = true;
  maxPage = 5;
  timeout = 10_000;
  weight = 0.85;
  disabled = false;

  private itemType: BandcampItemType | null;

  constructor(options?: { itemType?: BandcampItemType }) {
    this.itemType = options?.itemType ?? null;

    if (this.itemType) {
      const typeNames: Record<BandcampItemType, string> = {
        t: 'tracks',
        a: 'albums',
        b: 'artists',
        l: 'labels',
      };
      this.name = `bandcamp ${typeNames[this.itemType]}`;
    }
  }

  buildRequest(query: string, params: EngineParams): RequestConfig {
    const searchParams = new URLSearchParams();
    searchParams.set('q', query);
    searchParams.set('page', params.page.toString());

    // Filter by item type if specified
    if (this.itemType) {
      searchParams.set('item_type', this.itemType);
    }

    return {
      url: `https://bandcamp.com/search?${searchParams.toString()}`,
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

    // Try to extract JSON data from script tags first
    const jsonMatch = body.match(/data-search="([^"]+)"/);
    if (jsonMatch) {
      try {
        const decodedJson = decodeHtmlEntities(jsonMatch[1]);
        const searchData: BandcampSearchData = JSON.parse(decodedJson);

        if (searchData.auto?.results) {
          for (const item of searchData.auto.results) {
            const result = this.parseSearchItem(item);
            if (result) {
              results.results.push(result);
            }
          }
        }
      } catch {
        // JSON parse failed, fall back to HTML
      }
    }

    // Parse HTML search results
    if (results.results.length === 0) {
      this.parseHtmlResults(body, results);
    }

    return results;
  }

  private parseSearchItem(item: BandcampSearchItem): EngineResults['results'][0] | null {
    if (!item.url || !item.name) return null;

    // Build thumbnail URL from art_id
    let thumbnailUrl = item.img || '';
    if (item.art_id && !thumbnailUrl) {
      thumbnailUrl = `https://f4.bcbits.com/img/a${item.art_id}_16.jpg`;
    }

    // Build title based on type
    let title = item.name;
    let content = '';

    if (item.type === 'track') {
      if (item.artist) {
        content = `by ${item.artist}`;
      }
      if (item.album) {
        content += content ? ` | from ${item.album}` : `from ${item.album}`;
      }
    } else if (item.type === 'album') {
      if (item.artist) {
        content = `by ${item.artist}`;
      }
    } else if (item.type === 'artist' || item.type === 'label') {
      if (item.location) {
        content = item.location;
      }
    }

    if (item.genre) {
      content += content ? ` | ${item.genre}` : item.genre;
    }

    if (item.tags && item.tags.length > 0) {
      const tagsStr = item.tags.slice(0, 3).join(', ');
      content += content ? ` | ${tagsStr}` : tagsStr;
    }

    return {
      url: item.url,
      title: decodeHtmlEntities(title),
      content: content || `${item.type} on Bandcamp`,
      engine: this.name,
      score: this.weight,
      category: 'videos',
      template: thumbnailUrl ? 'images' : undefined,
      thumbnailUrl: thumbnailUrl || undefined,
      channel: item.artist || undefined,
      source: 'Bandcamp',
      metadata: {
        itemId: item.id,
        itemType: item.type,
        artistName: item.artist,
        albumName: item.album,
        genre: item.genre,
        location: item.location,
        tags: item.tags,
        artId: item.art_id,
      },
    };
  }

  private parseHtmlResults(body: string, results: EngineResults): void {
    // Find search result items
    const resultElements = findElements(body, 'li.searchresult');

    for (const element of resultElements.slice(0, 20)) {
      // Extract the result type
      const typeMatch = element.match(/class="searchresult\s+(\w+)"/);
      const itemType = (typeMatch?.[1] || 'track') as BandcampSearchItem['type'];

      // Extract URL
      const urlMatch = element.match(/<a\s+href="([^"]+)"/);
      if (!urlMatch) continue;
      const url = urlMatch[1];

      // Extract title from heading
      const headingMatch = element.match(/<div\s+class="heading"[^>]*>[\s\S]*?<a[^>]*>([^<]+)<\/a>/);
      const title = headingMatch ? headingMatch[1].trim() : '';

      if (!title) continue;

      // Extract artist/subheading
      const subheadMatch = element.match(/<div\s+class="subhead"[^>]*>([^<]+)<\/div>/);
      const subhead = subheadMatch ? subheadMatch[1].trim() : '';

      // Extract image
      const imgMatch = element.match(/<img[^>]+src="([^"]+)"/);
      const thumbnailUrl = imgMatch ? imgMatch[1] : '';

      // Extract genre from art-tags
      const genreMatch = element.match(/<div\s+class="genre"[^>]*>([^<]+)<\/div>/);
      const genre = genreMatch ? genreMatch[1].trim().replace('genre:', '').trim() : '';

      // Extract tags
      const tagsMatch = element.match(/<div\s+class="tags"[^>]*>([^<]+)<\/div>/);
      const tags = tagsMatch
        ? tagsMatch[1]
            .replace('tags:', '')
            .split(',')
            .map((t) => t.trim())
            .filter(Boolean)
        : [];

      let content = '';
      if (subhead) {
        content = subhead;
      }
      if (genre) {
        content += content ? ` | ${genre}` : genre;
      }
      if (tags.length > 0) {
        content += content ? ` | ${tags.slice(0, 3).join(', ')}` : tags.slice(0, 3).join(', ');
      }

      results.results.push({
        url,
        title: decodeHtmlEntities(title),
        content: content || `${itemType} on Bandcamp`,
        engine: this.name,
        score: this.weight,
        category: 'videos',
        template: thumbnailUrl ? 'images' : undefined,
        thumbnailUrl: thumbnailUrl || undefined,
        source: 'Bandcamp',
        metadata: {
          itemType,
          genre,
          tags,
        },
      });
    }
  }
}

/**
 * Bandcamp Tracks Engine.
 * Searches only for tracks.
 */
export class BandcampTracksEngine extends BandcampEngine {
  constructor() {
    super({ itemType: 't' });
  }
}

/**
 * Bandcamp Albums Engine.
 * Searches only for albums.
 */
export class BandcampAlbumsEngine extends BandcampEngine {
  constructor() {
    super({ itemType: 'a' });
  }
}

/**
 * Bandcamp Artists Engine.
 * Searches only for artists/bands.
 */
export class BandcampArtistsEngine extends BandcampEngine {
  constructor() {
    super({ itemType: 'b' });
  }
}
