/**
 * Vimeo Search Engine adapter.
 *
 * Scrapes JSON data from Vimeo search results page.
 * Looks for video data in data-search-data attributes or embedded JSON.
 * No API key required.
 */

import type {
  OnlineEngine,
  EngineParams,
  RequestConfig,
  EngineResults,
  Category,
} from './engine';
import { newEngineResults } from './engine';

// ========== Vimeo JSON Types ==========

interface VimeoClip {
  clip_id?: number;
  id?: number;
  link?: string;
  uri?: string;
  name?: string;
  title?: string;
  description?: string;
  pictures?: {
    sizes?: Array<{
      link?: string;
      width?: number;
      height?: number;
    }>;
  };
  thumbnail?: {
    base_link?: string;
  };
  duration?: number;
  user?: {
    name?: string;
    link?: string;
  };
  owner?: {
    name?: string;
  };
  stats?: {
    plays?: number;
  };
  plays?: number;
  created_time?: string;
}

interface VimeoSearchData {
  filtered?: {
    data?: VimeoClip[];
  };
  data?: VimeoClip[];
  clips?: VimeoClip[];
}

export class VimeoEngine implements OnlineEngine {
  name = 'vimeo';
  shortcut = 'vm';
  categories: Category[] = ['videos'];
  supportsPaging = true;
  maxPage = 10;
  timeout = 8000;
  weight = 0.9;
  disabled = false;

  buildRequest(query: string, params: EngineParams): RequestConfig {
    const searchParams = new URLSearchParams();
    searchParams.set('q', query);

    // Vimeo uses 1-based page numbers
    if (params.page > 1) {
      searchParams.set('page', params.page.toString());
    }

    return {
      url: `https://vimeo.com/search?${searchParams.toString()}`,
      method: 'GET',
      headers: {
        Accept: 'text/html,application/xhtml+xml',
        'Accept-Language': 'en-US,en;q=0.9',
        'User-Agent':
          'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36',
      },
      cookies: [],
    };
  }

  parseResponse(body: string, _params: EngineParams): EngineResults {
    const results = newEngineResults();

    // Try multiple extraction strategies

    // Strategy 1: Look for data-search-data attribute
    const searchDataMatch = body.match(/data-search-data="([^"]+)"/);
    if (searchDataMatch) {
      try {
        const decoded = searchDataMatch[1]
          .replace(/&quot;/g, '"')
          .replace(/&amp;/g, '&')
          .replace(/&lt;/g, '<')
          .replace(/&gt;/g, '>')
          .replace(/&#39;/g, "'");
        const data: VimeoSearchData = JSON.parse(decoded);
        this.extractFromSearchData(data, results);
        if (results.results.length > 0) return results;
      } catch {
        // Continue to next strategy
      }
    }

    // Strategy 2: Look for window.vimeo.config or similar embedded JSON
    const configMatches = body.matchAll(
      /window\.vimeo\s*=\s*window\.vimeo\s*\|\|\s*\{\};\s*window\.vimeo\.[\w_]+\s*=\s*(\{.+?\});/g
    );
    for (const match of configMatches) {
      try {
        const data = JSON.parse(match[1]);
        if (data.clips || data.data || data.filtered) {
          this.extractFromSearchData(data as VimeoSearchData, results);
          if (results.results.length > 0) return results;
        }
      } catch {
        // Continue
      }
    }

    // Strategy 3: Look for __INITIAL_STATE__ or similar React/SSR hydration data
    const initialStateMatch = body.match(
      /<script[^>]*>\s*window\.__INITIAL_STATE__\s*=\s*(\{.+?\});\s*<\/script>/s
    );
    if (initialStateMatch) {
      try {
        const state = JSON.parse(initialStateMatch[1]);
        this.extractFromInitialState(state, results);
        if (results.results.length > 0) return results;
      } catch {
        // Continue
      }
    }

    // Strategy 4: Look for JSON-LD structured data
    const jsonLdMatches = body.matchAll(
      /<script[^>]*type="application\/ld\+json"[^>]*>([^<]+)<\/script>/g
    );
    for (const match of jsonLdMatches) {
      try {
        const data = JSON.parse(match[1]);
        this.extractFromJsonLd(data, results);
      } catch {
        // Continue
      }
    }
    if (results.results.length > 0) return results;

    // Strategy 5: Look for clip data in script tags
    const scriptMatches = body.matchAll(
      /<script[^>]*>\s*(\{[^<]*"clip"[^<]*\})\s*<\/script>/g
    );
    for (const match of scriptMatches) {
      try {
        const data = JSON.parse(match[1]);
        if (data.clip) {
          this.extractClip(data.clip, results);
        }
      } catch {
        // Continue
      }
    }

    // Strategy 6: Parse HTML directly for video links
    this.extractFromHtml(body, results);

    return results;
  }

  private extractFromSearchData(
    data: VimeoSearchData,
    results: EngineResults
  ): void {
    const clips =
      data.filtered?.data || data.data || data.clips || [];
    for (const clip of clips) {
      this.extractClip(clip, results);
    }
  }

  private extractFromInitialState(
    state: Record<string, unknown>,
    results: EngineResults
  ): void {
    // Traverse the state looking for clip arrays
    const traverse = (obj: unknown): void => {
      if (!obj || typeof obj !== 'object') return;

      if (Array.isArray(obj)) {
        for (const item of obj) {
          if (
            item &&
            typeof item === 'object' &&
            ('clip_id' in item || 'uri' in item)
          ) {
            this.extractClip(item as VimeoClip, results);
          } else {
            traverse(item);
          }
        }
      } else {
        for (const value of Object.values(obj)) {
          traverse(value);
        }
      }
    };
    traverse(state);
  }

  private extractFromJsonLd(
    data: Record<string, unknown> | unknown[],
    results: EngineResults
  ): void {
    if (Array.isArray(data)) {
      for (const item of data) {
        this.extractFromJsonLd(
          item as Record<string, unknown>,
          results
        );
      }
      return;
    }

    if (data['@type'] === 'VideoObject') {
      const video = data as Record<string, unknown>;
      const url = video.url as string;
      const videoId = this.extractVideoId(url);
      if (!videoId) return;

      results.results.push({
        url: url || `https://vimeo.com/${videoId}`,
        title: (video.name as string) || '',
        content: (video.description as string) || '',
        engine: this.name,
        score: this.weight,
        category: 'videos',
        template: 'videos',
        duration: this.formatDuration(video.duration as string),
        embedUrl: `https://player.vimeo.com/video/${videoId}`,
        thumbnailUrl: (video.thumbnailUrl as string) || '',
        channel: (video.author as { name?: string })?.name || '',
        views: 0,
      });
    }
  }

  private extractClip(clip: VimeoClip, results: EngineResults): void {
    const videoId = clip.clip_id || clip.id;
    if (!videoId) {
      // Try to extract from uri or link
      const uri = clip.uri || clip.link || '';
      const idMatch = uri.match(/\/videos?\/(\d+)/);
      if (!idMatch) return;
    }

    const id =
      videoId ||
      this.extractVideoId(clip.uri || clip.link || '');
    if (!id) return;

    const title = clip.name || clip.title || '';
    const description = clip.description || '';

    // Get best thumbnail
    let thumbnailUrl = '';
    if (clip.pictures?.sizes && clip.pictures.sizes.length > 0) {
      // Get largest thumbnail
      const sorted = [...clip.pictures.sizes].sort(
        (a, b) => (b.width || 0) - (a.width || 0)
      );
      thumbnailUrl = sorted[0].link || '';
    } else if (clip.thumbnail?.base_link) {
      thumbnailUrl = clip.thumbnail.base_link;
    }

    // Format duration
    const duration = clip.duration
      ? this.formatSeconds(clip.duration)
      : '';

    // Get user/channel name
    const channel = clip.user?.name || clip.owner?.name || '';

    // Get play count
    const views = clip.stats?.plays || clip.plays || 0;

    const videoUrl = clip.link || `https://vimeo.com/${id}`;
    const embedUrl = `https://player.vimeo.com/video/${id}`;

    results.results.push({
      url: videoUrl,
      title,
      content: description,
      engine: this.name,
      score: this.weight,
      category: 'videos',
      template: 'videos',
      duration,
      embedUrl,
      thumbnailUrl,
      channel,
      views,
    });
  }

  private extractFromHtml(body: string, results: EngineResults): void {
    // Look for video links in the HTML
    const videoLinkRegex =
      /<a[^>]+href="(https?:\/\/vimeo\.com\/(\d+))"[^>]*>/g;
    const seen = new Set<string>();

    for (const match of body.matchAll(videoLinkRegex)) {
      const url = match[1];
      const videoId = match[2];

      if (seen.has(videoId)) continue;
      seen.add(videoId);

      // Try to find associated title
      const titleMatch = body.match(
        new RegExp(
          `<a[^>]+href="${url.replace(
            /[.*+?^${}()|[\]\\]/g,
            '\\$&'
          )}"[^>]*>([^<]+)</a>`
        )
      );
      const title = titleMatch ? titleMatch[1].trim() : `Video ${videoId}`;

      results.results.push({
        url,
        title,
        content: '',
        engine: this.name,
        score: this.weight,
        category: 'videos',
        template: 'videos',
        embedUrl: `https://player.vimeo.com/video/${videoId}`,
        thumbnailUrl: `https://i.vimeocdn.com/video/${videoId}_640.jpg`,
        channel: '',
        views: 0,
      });
    }
  }

  private extractVideoId(url: string): number | null {
    const match = url.match(/\/videos?\/(\d+)/);
    if (match) return parseInt(match[1], 10);

    const simpleMatch = url.match(/vimeo\.com\/(\d+)/);
    if (simpleMatch) return parseInt(simpleMatch[1], 10);

    return null;
  }

  private formatSeconds(seconds: number): string {
    const hours = Math.floor(seconds / 3600);
    const minutes = Math.floor((seconds % 3600) / 60);
    const secs = seconds % 60;

    if (hours > 0) {
      return `${hours}:${minutes.toString().padStart(2, '0')}:${secs
        .toString()
        .padStart(2, '0')}`;
    }
    return `${minutes}:${secs.toString().padStart(2, '0')}`;
  }

  private formatDuration(isoDuration: string | undefined): string {
    if (!isoDuration) return '';

    // Parse ISO 8601 duration (e.g., "PT1H2M3S")
    const match = isoDuration.match(
      /PT(?:(\d+)H)?(?:(\d+)M)?(?:(\d+)S)?/
    );
    if (!match) return '';

    const hours = parseInt(match[1] || '0', 10);
    const minutes = parseInt(match[2] || '0', 10);
    const seconds = parseInt(match[3] || '0', 10);

    return this.formatSeconds(hours * 3600 + minutes * 60 + seconds);
  }
}
