/**
 * 360Search Videos Engine adapter.
 *
 * Uses the 360Kan video search API for Chinese video search.
 * No API key required.
 *
 * API URL: https://tv.360kan.com/v1/video/list
 */

import type {
  OnlineEngine,
  EngineParams,
  RequestConfig,
  EngineResults,
  Category,
} from './engine';
import { newEngineResults } from './engine';
import { decodeHtmlEntities } from '../lib/html-parser';

// ========== 360Search Video API Response Types ==========

interface Video360Item {
  title?: string;
  description?: string;
  play_url?: string;
  cover?: string;
  stream_url?: string;
  publish_time?: number;
}

interface Video360ApiResponse {
  data?: {
    list?: Video360Item[];
  };
}

export class Search360VideosEngine implements OnlineEngine {
  name = '360search';
  shortcut = '360v';
  categories: Category[] = ['videos'];
  supportsPaging = true;
  maxPage = 10;
  timeout = 8000;
  weight = 0.6;
  disabled = false;

  buildRequest(query: string, params: EngineParams): RequestConfig {
    const searchParams = new URLSearchParams();
    searchParams.set('q', query);

    // Pagination: offset = (page - 1) * 10
    const offset = (params.page - 1) * 10;
    searchParams.set('start', offset.toString());
    searchParams.set('count', '10');

    return {
      url: `https://tv.360kan.com/v1/video/list?${searchParams.toString()}`,
      method: 'GET',
      headers: {
        Accept: 'application/json',
        'Accept-Language': 'zh-CN,zh;q=0.9,en;q=0.8',
        'User-Agent':
          'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36',
      },
      cookies: [],
    };
  }

  parseResponse(body: string, _params: EngineParams): EngineResults {
    const results = newEngineResults();

    let data: Video360ApiResponse;
    try {
      data = JSON.parse(body);
    } catch {
      return results;
    }

    if (!data.data?.list || !Array.isArray(data.data.list)) {
      return results;
    }

    for (const video of data.data.list) {
      // Skip if no play URL
      if (!video.play_url) continue;

      const title = video.title ? decodeHtmlEntities(video.title) : '';
      const description = video.description
        ? decodeHtmlEntities(video.description)
        : '';
      const url = video.play_url;
      const thumbnailUrl = video.cover || '';
      const embedUrl = video.stream_url || '';

      // Convert unix timestamp to ISO string
      let publishedAt = '';
      if (typeof video.publish_time === 'number') {
        publishedAt = new Date(video.publish_time * 1000).toISOString();
      }

      results.results.push({
        url,
        title,
        content: description,
        engine: this.name,
        score: this.weight,
        category: 'videos',
        template: 'videos',
        thumbnailUrl,
        embedUrl,
        publishedAt,
      });
    }

    return results;
  }
}
