/**
 * PeerTube Search Engine adapter.
 *
 * Uses Sepia Search - a federated search API for PeerTube instances.
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

// ========== PeerTube JSON Types ==========

interface PeerTubeAccount {
  displayName?: string;
  host?: string;
}

interface PeerTubeChannel {
  displayName?: string;
  host?: string;
}

interface PeerTubeVideo {
  url?: string;
  name?: string;
  description?: string;
  duration?: number;
  views?: number;
  publishedAt?: string;
  thumbnailPath?: string;
  previewPath?: string;
  embedPath?: string;
  account?: PeerTubeAccount;
  channel?: PeerTubeChannel;
}

interface PeerTubeSearchResponse {
  total?: number;
  data?: PeerTubeVideo[];
}

// Time range calculation: returns ISO date string for startDate parameter
function calculateStartDate(timeRange: string): string | null {
  const now = Date.now();
  let msOffset: number;

  switch (timeRange) {
    case 'day':
      msOffset = 24 * 60 * 60 * 1000; // 24 hours
      break;
    case 'week':
      msOffset = 7 * 24 * 60 * 60 * 1000; // 7 days
      break;
    case 'month':
      msOffset = 30 * 24 * 60 * 60 * 1000; // 30 days
      break;
    case 'year':
      msOffset = 365 * 24 * 60 * 60 * 1000; // 365 days
      break;
    default:
      return null;
  }

  return new Date(now - msOffset).toISOString();
}

export class PeerTubeEngine implements OnlineEngine {
  name = 'peertube';
  shortcut = 'ptb';
  categories: Category[] = ['videos'];
  supportsPaging = true;
  maxPage = 10;
  timeout = 10000;
  weight = 0.75;
  disabled = false;

  buildRequest(query: string, params: EngineParams): RequestConfig {
    const searchParams = new URLSearchParams();
    searchParams.set('search', query);
    searchParams.set('start', ((params.page - 1) * 15).toString());
    searchParams.set('count', '15');
    searchParams.set('sort', '-match');
    searchParams.set('searchTarget', 'search-index');

    // Safe search: 0=off (both), 1=moderate (false), 2=strict (false)
    if (params.safeSearch >= 1) {
      searchParams.set('nsfw', 'false');
    } else {
      searchParams.set('nsfw', 'both');
    }

    // Language settings
    const lang = params.locale.split('-')[0] || 'en';
    searchParams.append('languageOneOf[]', lang);
    searchParams.append('boostLanguages[]', lang);

    // Time range filter
    if (params.timeRange) {
      const startDate = calculateStartDate(params.timeRange);
      if (startDate) {
        searchParams.set('startDate', startDate);
      }
    }

    return {
      url: `https://sepiasearch.org/api/v1/search/videos?${searchParams.toString()}`,
      method: 'GET',
      headers: {
        Accept: 'application/json',
        'Accept-Language': `${lang},en;q=0.9`,
      },
      cookies: [],
    };
  }

  parseResponse(body: string, _params: EngineParams): EngineResults {
    const results = newEngineResults();

    let data: PeerTubeSearchResponse;
    try {
      data = JSON.parse(body);
    } catch {
      return results;
    }

    if (!data.data || !Array.isArray(data.data)) {
      return results;
    }

    for (const video of data.data) {
      if (!video.url || !video.name) continue;

      // Extract base URL from video URL
      const baseUrl = this.extractBaseUrl(video.url);
      if (!baseUrl) continue;

      // Build thumbnail URL
      let thumbnailUrl = '';
      if (video.thumbnailPath) {
        thumbnailUrl = `${baseUrl}${video.thumbnailPath}`;
      } else if (video.previewPath) {
        thumbnailUrl = `${baseUrl}${video.previewPath}`;
      }

      // Build embed URL
      let embedUrl = '';
      if (video.embedPath) {
        embedUrl = `${baseUrl}${video.embedPath}`;
      }

      // Format duration
      const duration = video.duration
        ? this.formatSeconds(video.duration)
        : '';

      // Get channel display name with host
      let channel = '';
      if (video.channel?.displayName) {
        const host = video.channel.host || this.extractHost(video.url);
        channel = `${video.channel.displayName}@${host}`;
      } else if (video.account?.displayName) {
        const host = video.account.host || this.extractHost(video.url);
        channel = `${video.account.displayName}@${host}`;
      }

      results.results.push({
        url: video.url,
        title: video.name,
        content: video.description || '',
        engine: this.name,
        score: this.weight,
        category: 'videos',
        template: 'videos',
        duration,
        embedUrl,
        thumbnailUrl,
        channel,
        views: video.views || 0,
        publishedAt: video.publishedAt,
      });
    }

    return results;
  }

  private extractBaseUrl(url: string): string | null {
    try {
      const parsed = new URL(url);
      return `${parsed.protocol}//${parsed.host}`;
    } catch {
      return null;
    }
  }

  private extractHost(url: string): string {
    try {
      return new URL(url).host;
    } catch {
      return '';
    }
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
}
