/**
 * Rumble Search Engine adapter.
 *
 * Scrapes video data from Rumble search results page.
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

// ========== Rumble JSON Types ==========

interface RumbleVideo {
  id?: number;
  vid?: string;
  url?: string;
  title?: string;
  description?: string;
  duration?: number;
  durationString?: string;
  views?: number;
  thumbnail?: string;
  channelName?: string;
  channelUrl?: string;
  pubDate?: string;
}

interface RumbleSearchResult {
  videos?: RumbleVideo[];
  num_results?: number;
}

export class RumbleEngine implements OnlineEngine {
  name = 'rumble';
  shortcut = 'rb';
  categories: Category[] = ['videos'];
  supportsPaging = true;
  maxPage = 10;
  timeout = 8000;
  weight = 0.7;
  disabled = false;

  buildRequest(query: string, params: EngineParams): RequestConfig {
    const searchParams = new URLSearchParams();
    searchParams.set('q', query);

    // Rumble uses 1-based page numbers
    if (params.page > 1) {
      searchParams.set('page', params.page.toString());
    }

    // Time range filter
    if (params.timeRange) {
      const dateMap: Record<string, string> = {
        day: 'today',
        week: 'this-week',
        month: 'this-month',
        year: 'this-year',
      };
      if (dateMap[params.timeRange]) {
        searchParams.set('date', dateMap[params.timeRange]);
      }
    }

    return {
      url: `https://rumble.com/search/video?${searchParams.toString()}`,
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

    // Strategy 1: Look for JSON data in script tags
    const jsonMatches = body.matchAll(
      /<script[^>]*type="application\/ld\+json"[^>]*>([^<]+)<\/script>/g
    );
    for (const match of jsonMatches) {
      try {
        const data = JSON.parse(match[1]);
        this.extractFromJsonLd(data, results);
      } catch {
        // Continue to next match
      }
    }
    if (results.results.length > 0) return results;

    // Strategy 2: Parse HTML directly for video items
    this.extractFromHtml(body, results);

    return results;
  }

  private extractFromJsonLd(
    data: Record<string, unknown> | unknown[],
    results: EngineResults
  ): void {
    if (Array.isArray(data)) {
      for (const item of data) {
        this.extractFromJsonLd(item as Record<string, unknown>, results);
      }
      return;
    }

    if (data['@type'] === 'VideoObject') {
      const video = data as Record<string, unknown>;
      const url = (video.url as string) || (video.contentUrl as string) || '';
      if (!url) return;

      const videoId = this.extractVideoId(url);

      results.results.push({
        url,
        title: (video.name as string) || '',
        content: (video.description as string) || '',
        engine: this.name,
        score: this.weight,
        category: 'videos',
        template: 'videos',
        duration: this.formatIsoDuration(video.duration as string),
        embedUrl: videoId ? `https://rumble.com/embed/${videoId}/` : '',
        thumbnailUrl: (video.thumbnailUrl as string) || '',
        channel:
          (video.author as { name?: string })?.name ||
          (video.creator as { name?: string })?.name ||
          '',
        views: this.parseViewCount(video.interactionCount),
        publishedAt: (video.uploadDate as string) || (video.datePublished as string) || '',
      });
    }

    // Handle ItemList
    if (data['@type'] === 'ItemList') {
      const items = data.itemListElement as unknown[];
      if (Array.isArray(items)) {
        for (const item of items) {
          const itemObj = item as Record<string, unknown>;
          if (itemObj.item) {
            this.extractFromJsonLd(itemObj.item as Record<string, unknown>, results);
          } else {
            this.extractFromJsonLd(itemObj, results);
          }
        }
      }
    }
  }

  private extractFromHtml(body: string, results: EngineResults): void {
    // Look for video items in HTML structure
    // Rumble uses data attributes and structured HTML for video listings

    // Pattern to find video entries - Rumble uses specific class patterns
    const videoItemRegex =
      /<article[^>]*class="[^"]*video-item[^"]*"[^>]*>[\s\S]*?<\/article>/gi;
    const videoItems = body.matchAll(videoItemRegex);

    for (const match of videoItems) {
      const itemHtml = match[0];
      this.parseVideoItem(itemHtml, results);
    }

    // Alternative pattern: look for video-listing-entry
    if (results.results.length === 0) {
      const listingRegex =
        /<div[^>]*class="[^"]*video-listing-entry[^"]*"[^>]*>[\s\S]*?(?=<div[^>]*class="[^"]*video-listing-entry|$)/gi;
      const listings = body.matchAll(listingRegex);

      for (const match of listings) {
        const itemHtml = match[0];
        this.parseVideoItem(itemHtml, results);
      }
    }

    // Fallback: look for basic video links
    if (results.results.length === 0) {
      const linkRegex =
        /<a[^>]+href="(https?:\/\/rumble\.com\/[^"]*-([a-zA-Z0-9]+)\.html)"[^>]*>[\s\S]*?<\/a>/gi;
      const seen = new Set<string>();

      for (const match of body.matchAll(linkRegex)) {
        const url = match[1];
        const videoId = match[2];

        if (seen.has(videoId) || !videoId || videoId.length < 4) continue;
        seen.add(videoId);

        // Try to extract title from link content or nearby elements
        const titleMatch = match[0].match(/>([^<]+)</);
        const title = titleMatch ? titleMatch[1].trim() : `Video ${videoId}`;

        if (title && !title.includes('rumble.com') && title.length > 2) {
          results.results.push({
            url,
            title,
            content: '',
            engine: this.name,
            score: this.weight,
            category: 'videos',
            template: 'videos',
            embedUrl: `https://rumble.com/embed/${videoId}/`,
            thumbnailUrl: '',
            channel: '',
            views: 0,
          });
        }
      }
    }
  }

  private parseVideoItem(html: string, results: EngineResults): void {
    // Extract URL
    const urlMatch = html.match(
      /href="(https?:\/\/rumble\.com\/[^"]*-([a-zA-Z0-9]+)\.html)"/
    );
    if (!urlMatch) return;

    const url = urlMatch[1];
    const videoId = urlMatch[2];

    // Extract title
    let title = '';
    const titleMatch = html.match(
      /<h3[^>]*class="[^"]*title[^"]*"[^>]*>([^<]+)<\/h3>/i
    );
    if (titleMatch) {
      title = titleMatch[1].trim();
    } else {
      const altTitleMatch = html.match(/title="([^"]+)"/);
      if (altTitleMatch) {
        title = altTitleMatch[1].trim();
      }
    }

    if (!title) return;

    // Extract thumbnail
    let thumbnailUrl = '';
    const thumbMatch = html.match(
      /(?:data-src|src)="(https?:\/\/[^"]*(?:thumb|sp\.rmbl\.ws)[^"]*)"/i
    );
    if (thumbMatch) {
      thumbnailUrl = thumbMatch[1];
    }

    // Extract duration
    let duration = '';
    const durationMatch = html.match(
      /<span[^>]*class="[^"]*duration[^"]*"[^>]*>([^<]+)<\/span>/i
    );
    if (durationMatch) {
      duration = durationMatch[1].trim();
    }

    // Extract channel name
    let channel = '';
    const channelMatch = html.match(
      /<span[^>]*class="[^"]*(?:author|channel)[^"]*"[^>]*>([^<]+)<\/span>/i
    );
    if (channelMatch) {
      channel = channelMatch[1].trim();
    }

    // Extract view count
    let views = 0;
    const viewsMatch = html.match(
      /<span[^>]*class="[^"]*views[^"]*"[^>]*>([^<]+)<\/span>/i
    );
    if (viewsMatch) {
      views = this.parseViewString(viewsMatch[1]);
    }

    results.results.push({
      url,
      title,
      content: '',
      engine: this.name,
      score: this.weight,
      category: 'videos',
      template: 'videos',
      duration,
      embedUrl: `https://rumble.com/embed/${videoId}/`,
      thumbnailUrl,
      channel,
      views,
    });
  }

  private extractVideoId(url: string): string | null {
    // Rumble URLs: https://rumble.com/v1abc2d-title.html
    const match = url.match(/rumble\.com\/([a-zA-Z0-9]+)-[^/]+\.html/);
    if (match) return match[1];

    // Alternative: just the ID segment
    const altMatch = url.match(/\/([a-zA-Z0-9]{6,})-/);
    if (altMatch) return altMatch[1];

    return null;
  }

  private formatIsoDuration(isoDuration: string | undefined): string {
    if (!isoDuration) return '';

    // Parse ISO 8601 duration (e.g., "PT1H2M3S")
    const match = isoDuration.match(/PT(?:(\d+)H)?(?:(\d+)M)?(?:(\d+)S)?/);
    if (!match) return '';

    const hours = parseInt(match[1] || '0', 10);
    const minutes = parseInt(match[2] || '0', 10);
    const seconds = parseInt(match[3] || '0', 10);

    return this.formatSeconds(hours * 3600 + minutes * 60 + seconds);
  }

  private formatSeconds(totalSeconds: number): string {
    const hours = Math.floor(totalSeconds / 3600);
    const minutes = Math.floor((totalSeconds % 3600) / 60);
    const secs = totalSeconds % 60;

    if (hours > 0) {
      return `${hours}:${minutes.toString().padStart(2, '0')}:${secs
        .toString()
        .padStart(2, '0')}`;
    }
    return `${minutes}:${secs.toString().padStart(2, '0')}`;
  }

  private parseViewCount(value: unknown): number {
    if (typeof value === 'number') return value;
    if (typeof value === 'string') {
      return parseInt(value.replace(/[,\s]/g, ''), 10) || 0;
    }
    return 0;
  }

  private parseViewString(str: string): number {
    const cleaned = str.trim().toLowerCase();
    const match = cleaned.match(/([\d.]+)\s*([kmb])?/);
    if (!match) return 0;

    let num = parseFloat(match[1]);
    const suffix = match[2];

    if (suffix === 'k') num *= 1000;
    else if (suffix === 'm') num *= 1000000;
    else if (suffix === 'b') num *= 1000000000;

    return Math.round(num);
  }
}
