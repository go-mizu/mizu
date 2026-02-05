/**
 * YouTube Search Engine adapter.
 * Ported from Go: pkg/engine/local/engines/youtube.go
 *
 * Scrapes ytInitialData JSON from YouTube search results page.
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

// Time range encoding in sp parameter (YouTube's filter encoding)
const timeRangeMap: Record<string, string> = {
  day: 'Ag',
  week: 'Aw',
  month: 'BA',
  year: 'BQ',
};

// ========== YouTube JSON Types ==========

interface YtVideoRenderer {
  videoId?: string;
  thumbnail?: {
    thumbnails?: Array<{
      url?: string;
      width?: number;
      height?: number;
    }>;
  };
  title?: {
    runs?: Array<{ text?: string }>;
  };
  descriptionSnippet?: {
    runs?: Array<{ text?: string }>;
  };
  lengthText?: {
    simpleText?: string;
  };
  ownerText?: {
    runs?: Array<{ text?: string }>;
  };
  viewCountText?: {
    simpleText?: string;
  };
  publishedTimeText?: {
    simpleText?: string;
  };
}

interface YtInitialData {
  contents?: {
    twoColumnSearchResultsRenderer?: {
      primaryContents?: {
        sectionListRenderer?: {
          contents?: Array<{
            itemSectionRenderer?: {
              contents?: Array<{
                videoRenderer?: YtVideoRenderer;
              }>;
            };
          }>;
        };
      };
    };
  };
}

export class YouTubeEngine implements OnlineEngine {
  name = 'youtube';
  shortcut = 'yt';
  categories: Category[] = ['videos'];
  supportsPaging = true;
  maxPage = 5;
  timeout = 5_000;
  weight = 1.0;
  disabled = false;

  buildRequest(query: string, params: EngineParams): RequestConfig {
    const searchParams = new URLSearchParams();
    searchParams.set('search_query', query);

    // Time range filter encoded in sp parameter
    if (params.timeRange && timeRangeMap[params.timeRange]) {
      const sp = `EgIIA${timeRangeMap[params.timeRange]}%3D%3D`;
      searchParams.set('sp', sp);
    }

    return {
      url: `https://www.youtube.com/results?${searchParams.toString()}`,
      method: 'GET',
      headers: {
        Accept: 'text/html',
        'Accept-Language': 'en-US,en;q=0.9',
      },
      cookies: ['CONSENT=YES+'],
    };
  }

  parseResponse(body: string, _params: EngineParams): EngineResults {
    const results = newEngineResults();

    // Extract ytInitialData JSON from HTML
    const match = body.match(
      /var ytInitialData = ({.+?});<\/script>/
    );
    if (!match) return results;

    let data: YtInitialData;
    try {
      data = JSON.parse(match[1]);
    } catch {
      return results;
    }

    // Navigate to video results
    const sectionContents =
      data.contents?.twoColumnSearchResultsRenderer?.primaryContents
        ?.sectionListRenderer?.contents;

    if (!sectionContents) return results;

    for (const section of sectionContents) {
      const items = section.itemSectionRenderer?.contents;
      if (!items) continue;

      for (const item of items) {
        const vr = item.videoRenderer;
        if (!vr?.videoId) continue;

        // Build title from runs
        let title = '';
        if (vr.title?.runs) {
          title = vr.title.runs.map((r) => r.text || '').join('');
        }

        // Build description from runs
        let description = '';
        if (vr.descriptionSnippet?.runs) {
          description = vr.descriptionSnippet.runs
            .map((r) => r.text || '')
            .join('');
        }

        // Get channel name
        let channel = '';
        if (vr.ownerText?.runs && vr.ownerText.runs.length > 0) {
          channel = vr.ownerText.runs[0].text || '';
        }

        // Get thumbnail (last one is usually highest quality)
        let thumbnailUrl = '';
        const thumbnails = vr.thumbnail?.thumbnails;
        if (thumbnails && thumbnails.length > 0) {
          thumbnailUrl = thumbnails[thumbnails.length - 1].url || '';
        }

        // Duration
        const duration = vr.lengthText?.simpleText || '';

        // View count
        let views = 0;
        const viewText = vr.viewCountText?.simpleText || '';
        const viewMatch = viewText.match(/([\d,]+)\s*view/i);
        if (viewMatch) {
          views = parseInt(viewMatch[1].replace(/,/g, ''), 10) || 0;
        }

        const videoUrl = `https://www.youtube.com/watch?v=${vr.videoId}`;
        const embedUrl = `https://www.youtube-nocookie.com/embed/${vr.videoId}`;

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
    }

    return results;
  }
}
