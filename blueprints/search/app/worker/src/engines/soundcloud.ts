/**
 * SoundCloud Search Engine adapter.
 *
 * Uses SoundCloud's public API endpoints for searching tracks.
 * Scrapes the client_id from the SoundCloud page for API access.
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

// ========== SoundCloud API Types ==========

interface SoundCloudUser {
  id: number;
  username: string;
  permalink: string;
  permalink_url: string;
  avatar_url: string;
  full_name?: string;
  followers_count?: number;
  track_count?: number;
  verified?: boolean;
}

interface SoundCloudTrack {
  id: number;
  title: string;
  description?: string;
  permalink: string;
  permalink_url: string;
  uri: string;
  artwork_url?: string;
  waveform_url?: string;
  duration: number;
  genre?: string;
  tag_list?: string;
  playback_count?: number;
  likes_count?: number;
  reposts_count?: number;
  comment_count?: number;
  created_at: string;
  user: SoundCloudUser;
  stream_url?: string;
  downloadable?: boolean;
  embeddable_by?: string;
}

interface SoundCloudSearchResponse {
  collection?: SoundCloudTrack[];
  next_href?: string;
  total_results?: number;
}

const USER_AGENT =
  'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36';

export class SoundCloudEngine implements OnlineEngine {
  name = 'soundcloud';
  shortcut = 'sc';
  categories: Category[] = ['videos']; // Using 'videos' category for audio/media
  supportsPaging = true;
  maxPage = 10;
  timeout = 10_000;
  weight = 0.85;
  disabled = false;

  buildRequest(query: string, params: EngineParams): RequestConfig {
    // Use the widget API for searching without client_id
    // This endpoint returns HTML that we can parse
    const searchParams = new URLSearchParams();
    searchParams.set('q', query);

    // For now, we'll scrape the search page HTML
    return {
      url: `https://soundcloud.com/search/sounds?${searchParams.toString()}`,
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

    // Extract the hydration data from the page
    // SoundCloud embeds JSON data in script tags
    const hydrateMatch = body.match(
      /window\.__sc_hydration\s*=\s*(\[[\s\S]*?\]);/
    );

    if (hydrateMatch) {
      try {
        const hydrateData = JSON.parse(hydrateMatch[1]) as Array<{
          hydratable: string;
          data: unknown;
        }>;

        // Find the search results data
        for (const item of hydrateData) {
          if (
            item.hydratable === 'search' ||
            item.hydratable === 'anonymousId'
          ) {
            continue;
          }

          // Look for collection data
          const data = item.data as { collection?: SoundCloudTrack[] } | undefined;
          if (data?.collection && Array.isArray(data.collection)) {
            for (const track of data.collection) {
              const result = this.parseTrack(track);
              if (result) {
                results.results.push(result);
              }
            }
          }
        }
      } catch {
        // JSON parse failed, fall back to HTML scraping
      }
    }

    // Fallback: Try to extract from HTML structure
    if (results.results.length === 0) {
      this.extractFromHtml(body, results);
    }

    return results;
  }

  private parseTrack(track: SoundCloudTrack): EngineResults['results'][0] | null {
    if (!track.permalink_url || !track.title) return null;

    // Format duration (milliseconds to MM:SS)
    const duration = this.formatDuration(track.duration);

    // Get artwork URL (replace -large with -t500x500 for better quality)
    let thumbnailUrl = track.artwork_url || '';
    if (thumbnailUrl) {
      thumbnailUrl = thumbnailUrl.replace('-large', '-t300x300');
    } else if (track.user?.avatar_url) {
      thumbnailUrl = track.user.avatar_url.replace('-large', '-t300x300');
    }

    // Build content with stats
    let content = track.description
      ? extractText(track.description).slice(0, 200)
      : '';

    const stats: string[] = [];
    if (track.playback_count !== undefined) {
      stats.push(`${this.formatNumber(track.playback_count)} plays`);
    }
    if (track.likes_count !== undefined) {
      stats.push(`${this.formatNumber(track.likes_count)} likes`);
    }
    if (track.genre) {
      stats.push(track.genre);
    }

    if (stats.length > 0) {
      if (content) {
        content += ` | ${stats.join(' | ')}`;
      } else {
        content = stats.join(' | ');
      }
    }

    // Build embed URL
    const embedUrl = `https://w.soundcloud.com/player/?url=${encodeURIComponent(track.permalink_url)}&auto_play=false`;

    return {
      url: track.permalink_url,
      title: decodeHtmlEntities(track.title),
      content,
      engine: this.name,
      score: this.weight,
      category: 'videos',
      template: 'videos',
      duration,
      embedUrl,
      thumbnailUrl,
      channel: track.user?.username || track.user?.full_name || '',
      views: track.playback_count || 0,
      publishedAt: track.created_at,
      metadata: {
        trackId: track.id,
        userId: track.user?.id,
        username: track.user?.username,
        genre: track.genre,
        tags: track.tag_list,
        likesCount: track.likes_count,
        repostsCount: track.reposts_count,
        commentCount: track.comment_count,
        downloadable: track.downloadable,
        waveformUrl: track.waveform_url,
      },
    };
  }

  private extractFromHtml(body: string, results: EngineResults): void {
    // Try to extract track links from HTML
    const trackElements = findElements(body, 'a.soundTitle__title');

    for (const element of trackElements.slice(0, 20)) {
      const hrefMatch = element.match(/href="([^"]+)"/);
      const titleMatch = element.match(/>([^<]+)</);

      if (hrefMatch && titleMatch) {
        let url = hrefMatch[1];
        if (!url.startsWith('http')) {
          url = `https://soundcloud.com${url}`;
        }

        results.results.push({
          url,
          title: decodeHtmlEntities(titleMatch[1].trim()),
          content: '',
          engine: this.name,
          score: this.weight,
          category: 'videos',
          template: 'videos',
        });
      }
    }
  }

  private formatDuration(ms: number): string {
    const totalSeconds = Math.floor(ms / 1000);
    const minutes = Math.floor(totalSeconds / 60);
    const seconds = totalSeconds % 60;
    return `${minutes}:${seconds.toString().padStart(2, '0')}`;
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
