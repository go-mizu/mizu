/**
 * Dailymotion Search Engine adapter.
 *
 * Uses the Dailymotion public REST API for video search.
 * No API key required for basic search.
 *
 * API Documentation: https://developers.dailymotion.com/api/
 */

import type {
  OnlineEngine,
  EngineParams,
  RequestConfig,
  EngineResults,
  Category,
} from './engine';
import { newEngineResults } from './engine';

// ========== Dailymotion API Response Types ==========

interface DailymotionVideo {
  id: string;
  title?: string;
  description?: string;
  duration?: number;
  thumbnail_360_url?: string;
  created_time?: number;
  'owner.screenname'?: string;
  views_total?: number;
  embed_url?: string;
  allow_embed?: boolean;
}

interface DailymotionApiResponse {
  page?: number;
  limit?: number;
  explicit?: boolean;
  total?: number;
  has_more?: boolean;
  list?: DailymotionVideo[];
}

// Time range filter mapping (convert to unix timestamp offset)
const timeRangeSeconds: Record<string, number> = {
  day: 24 * 60 * 60,
  week: 7 * 24 * 60 * 60,
  month: 30 * 24 * 60 * 60,
  year: 365 * 24 * 60 * 60,
};

export class DailymotionEngine implements OnlineEngine {
  name = 'dailymotion';
  shortcut = 'dm';
  categories: Category[] = ['videos'];
  supportsPaging = true;
  maxPage = 10;
  timeout = 8000;
  weight = 0.85;
  disabled = false;

  buildRequest(query: string, params: EngineParams): RequestConfig {
    const searchParams = new URLSearchParams();
    searchParams.set('search', query);
    searchParams.set('page', params.page.toString());
    searchParams.set('limit', '10');
    searchParams.set(
      'fields',
      'id,title,description,duration,thumbnail_360_url,created_time,owner.screenname,views_total,embed_url,allow_embed'
    );

    // Safe search handling
    // 0 = off, 1 = moderate, 2 = strict
    if (params.safeSearch === 0) {
      searchParams.set('family_filter', 'false');
    } else if (params.safeSearch === 1) {
      searchParams.set('family_filter', 'true');
    } else if (params.safeSearch === 2) {
      searchParams.set('family_filter', 'true');
      searchParams.set('is_created_for_kids', 'true');
    }

    // Language/locale filter
    if (params.locale) {
      // Extract language code from locale (e.g., 'en-US' -> 'en')
      const lang = params.locale.split('-')[0];
      if (lang && lang.length === 2) {
        searchParams.set('languages', lang);
      }
    }

    // Time range filter
    if (params.timeRange && timeRangeSeconds[params.timeRange]) {
      const createdAfter = Math.floor(
        Date.now() / 1000 - timeRangeSeconds[params.timeRange]
      );
      searchParams.set('created_after', createdAfter.toString());
    }

    // Exclude private and password-protected videos
    searchParams.set('private', 'false');
    searchParams.set('password_protected', 'false');

    return {
      url: `https://api.dailymotion.com/videos?${searchParams.toString()}`,
      method: 'GET',
      headers: {
        Accept: 'application/json',
      },
      cookies: [],
    };
  }

  parseResponse(body: string, _params: EngineParams): EngineResults {
    const results = newEngineResults();

    let data: DailymotionApiResponse;
    try {
      data = JSON.parse(body);
    } catch {
      return results;
    }

    if (!data.list || !Array.isArray(data.list)) {
      return results;
    }

    for (const video of data.list) {
      if (!video.id) continue;

      const title = video.title || '';
      const description = video.description || '';
      const channel = video['owner.screenname'] || '';
      const views = video.views_total || 0;
      const thumbnailUrl = video.thumbnail_360_url || '';

      // Format duration from seconds
      const duration = video.duration
        ? this.formatSeconds(video.duration)
        : '';

      // Build video URL
      const videoUrl = `https://www.dailymotion.com/video/${video.id}`;

      // Build embed URL (only if embedding is allowed)
      let embedUrl = '';
      if (video.allow_embed !== false) {
        embedUrl = `https://www.dailymotion.com/embed/video/${video.id}`;
      }

      // Format published date
      let publishedAt = '';
      if (video.created_time) {
        publishedAt = new Date(video.created_time * 1000).toISOString();
      }

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
        publishedAt,
      });
    }

    return results;
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
