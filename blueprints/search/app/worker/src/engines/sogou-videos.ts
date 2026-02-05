/**
 * Sogou Videos Engine adapter.
 *
 * Uses the Sogou video search API for Chinese video search.
 * No API key required.
 *
 * API URL: https://v.sogou.com/api/video/shortVideoV2
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

// ========== Sogou Video API Response Types ==========

interface SogouVideoItem {
  title?: string;
  url?: string;
  pic?: string;
  duration?: string;
  date?: string;
  site?: string;
}

interface SogouVideoApiResponse {
  data?: {
    listData?: SogouVideoItem[];
  };
}

export class SogouVideosEngine implements OnlineEngine {
  name = 'sogou';
  shortcut = 'sgv';
  categories: Category[] = ['videos'];
  supportsPaging = true;
  maxPage = 10;
  timeout = 8000;
  weight = 0.6;
  disabled = false;

  buildRequest(query: string, params: EngineParams): RequestConfig {
    const searchParams = new URLSearchParams();
    searchParams.set('query', query);
    searchParams.set('page', params.page.toString());
    searchParams.set('pagesize', '10');

    return {
      url: `https://v.sogou.com/api/video/shortVideoV2?${searchParams.toString()}`,
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

    let data: SogouVideoApiResponse;
    try {
      data = JSON.parse(body);
    } catch {
      return results;
    }

    if (!data.data?.listData || !Array.isArray(data.data.listData)) {
      return results;
    }

    for (const video of data.data.listData) {
      // Skip if no URL
      if (!video.url) continue;

      const title = video.title ? decodeHtmlEntities(video.title) : '';

      // URL: if it doesn't start with http, prepend https://v.sogou.com
      let url = video.url;
      if (!url.startsWith('http')) {
        url = `https://v.sogou.com${url.startsWith('/') ? '' : '/'}${url}`;
      }

      const thumbnailUrl = video.pic || '';
      const duration = video.duration || '';
      const publishedAt = video.date || '';
      const content = video.site || ''; // source site name

      results.results.push({
        url,
        title,
        content,
        engine: this.name,
        score: this.weight,
        category: 'videos',
        template: 'videos',
        thumbnailUrl,
        duration,
        publishedAt,
      });
    }

    return results;
  }
}
