/**
 * Bilibili Search Engine adapter.
 *
 * Uses the Bilibili public search API for video search.
 * No API key required for basic search.
 *
 * Note: Bilibili is a Chinese video sharing platform (often called "Chinese YouTube")
 */

import type {
  OnlineEngine,
  EngineParams,
  RequestConfig,
  EngineResults,
  Category,
} from './engine';
import { newEngineResults } from './engine';

// ========== Bilibili API Types ==========

interface BilibiliVideo {
  aid?: number;
  bvid?: string;
  title?: string;
  description?: string;
  pic?: string;
  play?: number;
  video_review?: number;
  favorites?: number;
  tag?: string;
  duration?: string;
  author?: string;
  mid?: number;
  pubdate?: number;
  senddate?: number;
  arcurl?: string;
}

interface BilibiliSearchData {
  result?: BilibiliVideo[];
  numResults?: number;
  numPages?: number;
  page?: number;
  pagesize?: number;
}

interface BilibiliSearchResponse {
  code?: number;
  message?: string;
  data?: BilibiliSearchData;
}

// Order/Sort options
const sortOptions: Record<string, string> = {
  relevance: '',
  date: 'pubdate',
  views: 'click',
  duration: 'duration',
};

// Duration filter options
const durationFilters: Record<string, string> = {
  any: '0',
  short: '1', // 0-10 minutes
  medium: '2', // 10-30 minutes
  long: '3', // 30-60 minutes
  // '4' = 60+ minutes
};

export class BilibiliEngine implements OnlineEngine {
  name = 'bilibili';
  shortcut = 'bili';
  categories: Category[] = ['videos'];
  supportsPaging = true;
  maxPage = 10;
  timeout = 10000;
  weight = 0.65;
  disabled = false;

  buildRequest(query: string, params: EngineParams): RequestConfig {
    const searchParams = new URLSearchParams();
    searchParams.set('keyword', query);
    searchParams.set('search_type', 'video');
    searchParams.set('page', params.page.toString());
    searchParams.set('pagesize', '20');

    // Order/sort - default to relevance
    const sort = params.videoFilters?.source || 'relevance';
    if (sortOptions[sort]) {
      searchParams.set('order', sortOptions[sort]);
    }

    // Duration filter
    const duration = params.videoFilters?.duration || 'any';
    if (durationFilters[duration]) {
      searchParams.set('duration', durationFilters[duration]);
    }

    // Time range filter (tids parameter)
    // Bilibili doesn't have direct time range filter in the public API
    // But we can use order=pubdate for recent content

    return {
      url: `https://api.bilibili.com/x/web-interface/search/type?${searchParams.toString()}`,
      method: 'GET',
      headers: {
        Accept: 'application/json',
        'Accept-Language': 'zh-CN,zh;q=0.9,en;q=0.8',
        Referer: 'https://search.bilibili.com/',
        'User-Agent':
          'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36',
      },
      cookies: [],
    };
  }

  parseResponse(body: string, _params: EngineParams): EngineResults {
    const results = newEngineResults();

    let response: BilibiliSearchResponse;
    try {
      response = JSON.parse(body);
    } catch {
      return results;
    }

    // Check for valid response
    if (response.code !== 0 || !response.data?.result) {
      return results;
    }

    const videos = response.data.result;
    if (!Array.isArray(videos)) {
      return results;
    }

    for (const video of videos) {
      // Use bvid (Bilibili Video ID) or aid (Archive ID)
      const bvid = video.bvid;
      const aid = video.aid;

      if (!bvid && !aid) continue;

      // Build URL - prefer bvid
      const videoUrl = video.arcurl || (bvid
        ? `https://www.bilibili.com/video/${bvid}`
        : `https://www.bilibili.com/video/av${aid}`);

      // Clean title (Bilibili returns HTML-escaped and highlighted titles)
      const title = this.cleanHtml(video.title || '');

      // Clean description
      const description = this.cleanHtml(video.description || '');

      // Get thumbnail - ensure HTTPS and remove size suffix if needed
      let thumbnailUrl = video.pic || '';
      if (thumbnailUrl.startsWith('//')) {
        thumbnailUrl = `https:${thumbnailUrl}`;
      }

      // Format duration (Bilibili returns "MM:SS" or "H:MM:SS" format already)
      const duration = video.duration || '';

      // Get channel/author
      const channel = video.author || '';

      // Get view count
      const views = video.play || 0;

      // Build embed URL
      const embedUrl = bvid
        ? `https://player.bilibili.com/player.html?bvid=${bvid}`
        : `https://player.bilibili.com/player.html?aid=${aid}`;

      // Format published date
      let publishedAt = '';
      if (video.pubdate) {
        publishedAt = new Date(video.pubdate * 1000).toISOString();
      } else if (video.senddate) {
        publishedAt = new Date(video.senddate * 1000).toISOString();
      }

      results.results.push({
        url: videoUrl,
        title,
        content: description,
        engine: this.name,
        score: this.weight,
        category: 'videos',
        template: 'videos',
        duration: this.normalizeDuration(duration),
        embedUrl,
        thumbnailUrl,
        channel,
        views,
        publishedAt,
        metadata: {
          bvid,
          aid,
          favorites: video.favorites,
          danmaku: video.video_review,
          tags: video.tag,
        },
      });
    }

    return results;
  }

  private cleanHtml(html: string): string {
    return html
      .replace(/<em class="keyword">/g, '')
      .replace(/<\/em>/g, '')
      .replace(/&lt;/g, '<')
      .replace(/&gt;/g, '>')
      .replace(/&amp;/g, '&')
      .replace(/&quot;/g, '"')
      .replace(/&#39;/g, "'")
      .replace(/<[^>]+>/g, '')
      .trim();
  }

  private normalizeDuration(duration: string): string {
    // Bilibili already returns duration in M:SS or H:MM:SS format
    // Just ensure proper formatting
    if (!duration) return '';

    const parts = duration.split(':');
    if (parts.length === 2) {
      // M:SS format - ensure proper padding
      const minutes = parseInt(parts[0], 10);
      const seconds = parseInt(parts[1], 10);
      return `${minutes}:${seconds.toString().padStart(2, '0')}`;
    } else if (parts.length === 3) {
      // H:MM:SS format
      const hours = parseInt(parts[0], 10);
      const minutes = parseInt(parts[1], 10);
      const seconds = parseInt(parts[2], 10);
      return `${hours}:${minutes.toString().padStart(2, '0')}:${seconds
        .toString()
        .padStart(2, '0')}`;
    }

    return duration;
  }
}
