/**
 * Genius Search Engine adapter.
 *
 * Searches Genius for song lyrics, annotations, and artist information.
 * Uses the public search endpoint.
 */

import type {
  OnlineEngine,
  EngineParams,
  RequestConfig,
  EngineResults,
  Category,
} from './engine';
import { newEngineResults } from './engine';
import { decodeHtmlEntities, findElements } from '../lib/html-parser';

// ========== Genius API Types ==========

interface GeniusArtist {
  id: number;
  name: string;
  url: string;
  image_url?: string;
  header_image_url?: string;
  is_verified?: boolean;
}

interface GeniusSong {
  id: number;
  title: string;
  title_with_featured?: string;
  url: string;
  path: string;
  full_title: string;
  song_art_image_url?: string;
  song_art_image_thumbnail_url?: string;
  header_image_url?: string;
  header_image_thumbnail_url?: string;
  release_date_for_display?: string;
  release_date_components?: {
    year?: number;
    month?: number;
    day?: number;
  };
  primary_artist: GeniusArtist;
  featured_artists?: GeniusArtist[];
  stats?: {
    hot?: boolean;
    pageviews?: number;
    unreviewed_annotations?: number;
    verified_annotations?: number;
    concurrents?: number;
  };
  annotation_count?: number;
  pyongs_count?: number;
  lyrics_state?: string;
}

interface GeniusSearchHit {
  type: string;
  index: string;
  result: GeniusSong | GeniusArtist;
}

interface GeniusSearchResponse {
  response?: {
    sections?: Array<{
      type: string;
      hits: GeniusSearchHit[];
    }>;
    hits?: GeniusSearchHit[];
  };
}

const USER_AGENT =
  'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36';

export class GeniusEngine implements OnlineEngine {
  name = 'genius';
  shortcut = 'gn';
  categories: Category[] = ['videos']; // Using 'videos' for music content
  supportsPaging = true;
  maxPage = 5;
  timeout = 10_000;
  weight = 0.85;
  disabled = false;

  buildRequest(query: string, params: EngineParams): RequestConfig {
    const searchParams = new URLSearchParams();
    searchParams.set('q', query);
    searchParams.set('page', params.page.toString());
    searchParams.set('per_page', '20');

    return {
      url: `https://genius.com/api/search/multi?${searchParams.toString()}`,
      method: 'GET',
      headers: {
        Accept: 'application/json',
        'User-Agent': USER_AGENT,
        'Accept-Language': params.locale || 'en-US,en;q=0.9',
      },
      cookies: [],
    };
  }

  parseResponse(body: string, _params: EngineParams): EngineResults {
    const results = newEngineResults();

    try {
      const data: GeniusSearchResponse = JSON.parse(body);

      // Handle multi-section response
      if (data.response?.sections) {
        for (const section of data.response.sections) {
          if (section.type === 'song' || section.type === 'top_hit') {
            for (const hit of section.hits) {
              if (hit.type === 'song') {
                const result = this.parseSong(hit.result as GeniusSong);
                if (result) {
                  results.results.push(result);
                }
              }
            }
          }
        }
      }

      // Handle flat hits array
      if (data.response?.hits) {
        for (const hit of data.response.hits) {
          if (hit.type === 'song') {
            const result = this.parseSong(hit.result as GeniusSong);
            if (result) {
              results.results.push(result);
            }
          }
        }
      }
    } catch {
      // JSON parse failed, try HTML scraping
      this.parseHtmlResults(body, results);
    }

    return results;
  }

  private parseSong(song: GeniusSong): EngineResults['results'][0] | null {
    if (!song.url || !song.title) return null;

    // Build title with artist
    const title = song.title_with_featured || song.full_title || song.title;

    // Get thumbnail
    const thumbnailUrl =
      song.song_art_image_thumbnail_url ||
      song.song_art_image_url ||
      song.header_image_thumbnail_url ||
      '';

    // Build content with metadata
    const contentParts: string[] = [];

    if (song.primary_artist?.name) {
      contentParts.push(`by ${song.primary_artist.name}`);
    }

    if (song.release_date_for_display) {
      contentParts.push(`Released ${song.release_date_for_display}`);
    }

    if (song.stats?.pageviews) {
      contentParts.push(`${this.formatNumber(song.stats.pageviews)} views`);
    }

    if (song.annotation_count && song.annotation_count > 0) {
      contentParts.push(`${song.annotation_count} annotations`);
    }

    if (song.stats?.hot) {
      contentParts.push('Hot');
    }

    const content = contentParts.join(' | ');

    // Format release date as ISO string if available
    let publishedAt: string | undefined;
    if (song.release_date_components?.year) {
      const { year, month, day } = song.release_date_components;
      const dateStr = `${year}-${String(month || 1).padStart(2, '0')}-${String(day || 1).padStart(2, '0')}`;
      publishedAt = new Date(dateStr).toISOString();
    }

    return {
      url: song.url,
      title: decodeHtmlEntities(title),
      content,
      engine: this.name,
      score: this.weight,
      category: 'videos',
      template: thumbnailUrl ? 'images' : undefined,
      thumbnailUrl: thumbnailUrl || undefined,
      channel: song.primary_artist?.name || undefined,
      publishedAt,
      source: 'Genius',
      metadata: {
        songId: song.id,
        artistId: song.primary_artist?.id,
        artistName: song.primary_artist?.name,
        artistUrl: song.primary_artist?.url,
        releaseDate: song.release_date_for_display,
        pageviews: song.stats?.pageviews,
        annotationCount: song.annotation_count,
        pyongsCount: song.pyongs_count,
        isHot: song.stats?.hot,
        lyricsState: song.lyrics_state,
        featuredArtists: song.featured_artists?.map((a) => ({
          id: a.id,
          name: a.name,
        })),
      },
    };
  }

  private parseHtmlResults(body: string, results: EngineResults): void {
    // Parse song cards from HTML
    const songCards = findElements(body, 'div.mini_card');

    for (const card of songCards.slice(0, 20)) {
      // Extract URL
      const urlMatch = card.match(/<a\s+href="(https:\/\/genius\.com\/[^"]+)"/);
      if (!urlMatch) continue;
      const url = urlMatch[1];

      // Extract title
      const titleMatch = card.match(/<div\s+class="mini_card-title"[^>]*>([^<]+)<\/div>/);
      const title = titleMatch ? titleMatch[1].trim() : '';

      if (!title) continue;

      // Extract artist/subtitle
      const subtitleMatch = card.match(/<div\s+class="mini_card-subtitle"[^>]*>([^<]+)<\/div>/);
      const artist = subtitleMatch ? subtitleMatch[1].trim() : '';

      // Extract image
      const imgMatch = card.match(/<img[^>]+src="([^"]+)"/);
      const thumbnailUrl = imgMatch ? imgMatch[1] : '';

      results.results.push({
        url,
        title: decodeHtmlEntities(title),
        content: artist ? `by ${artist}` : 'Song on Genius',
        engine: this.name,
        score: this.weight,
        category: 'videos',
        template: thumbnailUrl ? 'images' : undefined,
        thumbnailUrl: thumbnailUrl || undefined,
        channel: artist || undefined,
        source: 'Genius',
      });
    }
  }

  private formatNumber(num: number): string {
    if (num >= 1_000_000) {
      return `${(num / 1_000_000).toFixed(1)}M`;
    }
    if (num >= 1_000) {
      return `${(num / 1_000).toFixed(1)}K`;
    }
    return num.toString();
  }
}

/**
 * Genius Lyrics Search Engine.
 * Specifically searches for songs with lyrics.
 */
export class GeniusLyricsEngine implements OnlineEngine {
  name = 'genius lyrics';
  shortcut = 'gnl';
  categories: Category[] = ['videos'];
  supportsPaging = true;
  maxPage = 5;
  timeout = 10_000;
  weight = 0.8;
  disabled = false;

  buildRequest(query: string, params: EngineParams): RequestConfig {
    // Search with lyrics focus
    const searchParams = new URLSearchParams();
    searchParams.set('q', `${query} lyrics`);
    searchParams.set('page', params.page.toString());

    return {
      url: `https://genius.com/api/search/song?${searchParams.toString()}`,
      method: 'GET',
      headers: {
        Accept: 'application/json',
        'User-Agent': USER_AGENT,
        'Accept-Language': params.locale || 'en-US,en;q=0.9',
      },
      cookies: [],
    };
  }

  parseResponse(body: string, _params: EngineParams): EngineResults {
    const results = newEngineResults();

    try {
      const data = JSON.parse(body) as {
        response?: {
          sections?: Array<{
            hits?: GeniusSearchHit[];
          }>;
        };
      };

      if (data.response?.sections) {
        for (const section of data.response.sections) {
          if (section.hits) {
            for (const hit of section.hits) {
              if (hit.type === 'song') {
                const song = hit.result as GeniusSong;

                // Only include songs with lyrics
                if (song.lyrics_state !== 'complete') continue;

                if (!song.url || !song.title) continue;

                const title = song.full_title || song.title;
                const thumbnailUrl =
                  song.song_art_image_thumbnail_url || song.header_image_thumbnail_url || '';

                results.results.push({
                  url: song.url,
                  title: decodeHtmlEntities(title),
                  content: `Lyrics by ${song.primary_artist?.name || 'Unknown Artist'}`,
                  engine: this.name,
                  score: this.weight,
                  category: 'videos',
                  template: thumbnailUrl ? 'images' : undefined,
                  thumbnailUrl: thumbnailUrl || undefined,
                  channel: song.primary_artist?.name || undefined,
                  source: 'Genius',
                  metadata: {
                    songId: song.id,
                    artistName: song.primary_artist?.name,
                    lyricsState: song.lyrics_state,
                  },
                });
              }
            }
          }
        }
      }
    } catch {
      // Parse error
    }

    return results;
  }
}
