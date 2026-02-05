/**
 * Google Videos Search Engine adapter.
 *
 * Uses Google's video search (tbm=vid) to find video results.
 * Parses HTML response to extract video information.
 */

import type {
  OnlineEngine,
  EngineParams,
  RequestConfig,
  EngineResults,
  Category,
} from './engine';
import { newEngineResults } from './engine';
import { extractText, findElements, decodeHtmlEntities } from '../lib/html-parser';

// ========== Chrome User Agent ==========

const chromeUserAgent =
  'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36';

// ========== Time Range & Safe Search Maps ==========

const timeRangeMap: Record<string, string> = {
  day: 'd',
  week: 'w',
  month: 'm',
  year: 'y',
};

const safeSearchMap: Record<number, string> = {
  0: 'off',
  2: 'active',
};

// ========== Video Filter Maps ==========

const durationMap: Record<string, string> = {
  short: 's',  // < 4 min
  medium: 'm', // 4-20 min
  long: 'l',   // > 20 min
};

// ========== URL Unwrapper ==========

/**
 * Unwrap a Google /url?q=... redirect to get the real URL.
 */
function unwrapGoogleUrl(href: string): string {
  if (href.startsWith('/url?')) {
    const match = href.match(/[?&]q=([^&]+)/);
    if (match) {
      let decoded = decodeURIComponent(match[1]);
      const saIdx = decoded.indexOf('&sa=U');
      if (saIdx > 0) {
        decoded = decoded.slice(0, saIdx);
      }
      return decoded;
    }
    const urlMatch = href.match(/[?&]url=([^&]+)/);
    if (urlMatch) {
      return decodeURIComponent(urlMatch[1]);
    }
  }
  return href;
}

/**
 * Extract YouTube video ID from a URL.
 * Returns null if not a YouTube URL or ID not found.
 */
function extractYouTubeVideoId(url: string): string | null {
  // Match youtube.com/watch?v=ID
  const watchMatch = url.match(/youtube\.com\/watch\?v=([a-zA-Z0-9_-]{11})/);
  if (watchMatch) return watchMatch[1];

  // Match youtu.be/ID
  const shortMatch = url.match(/youtu\.be\/([a-zA-Z0-9_-]{11})/);
  if (shortMatch) return shortMatch[1];

  // Match youtube.com/embed/ID
  const embedMatch = url.match(/youtube\.com\/embed\/([a-zA-Z0-9_-]{11})/);
  if (embedMatch) return embedMatch[1];

  return null;
}

/**
 * Get YouTube thumbnail URL from video ID.
 */
function getYouTubeThumbnail(videoId: string): string {
  return `https://img.youtube.com/vi/${videoId}/hqdefault.jpg`;
}

// ========== JSON-LD VideoObject Type ==========

interface VideoObjectJsonLd {
  '@type'?: string;
  name?: string;
  description?: string;
  thumbnailUrl?: string | string[];
  uploadDate?: string;
  duration?: string;
  contentUrl?: string;
  embedUrl?: string;
  author?: {
    '@type'?: string;
    name?: string;
  };
  interactionStatistic?: {
    '@type'?: string;
    interactionCount?: number;
  };
}

// ========== GoogleVideosEngine ==========

export class GoogleVideosEngine implements OnlineEngine {
  name = 'google_videos';
  shortcut = 'gov';
  categories: Category[] = ['videos'];
  supportsPaging = true;
  maxPage = 50;
  timeout = 10_000;
  weight = 1.0;
  disabled = false;

  buildRequest(query: string, params: EngineParams): RequestConfig {
    const searchParams = new URLSearchParams();
    searchParams.set('q', query);
    searchParams.set('tbm', 'vid'); // Video search mode

    // Derive language from locale
    const locale = params.locale || 'en-US';
    const parts = locale.split('-');
    const langCode = parts[0] || 'en';
    searchParams.set('hl', langCode);

    // Pagination (0-based offset, 10 results per page)
    const start = (params.page - 1) * 10;
    if (start > 0) {
      searchParams.set('start', start.toString());
    }

    // Safe search
    const safeValue = safeSearchMap[params.safeSearch];
    if (safeValue) {
      searchParams.set('safe', safeValue);
    }

    // Build tbs parameter for filters
    const tbsParts: string[] = [];

    // Time range filter
    if (params.timeRange && timeRangeMap[params.timeRange]) {
      tbsParts.push(`qdr:${timeRangeMap[params.timeRange]}`);
    }

    // Video filters
    if (params.videoFilters) {
      const filters = params.videoFilters;

      // Duration filter
      if (filters.duration && filters.duration !== 'any' && durationMap[filters.duration]) {
        tbsParts.push(`dur:${durationMap[filters.duration]}`);
      }

      // Quality/HD filter
      if (filters.quality === 'hd' || filters.quality === '4k') {
        tbsParts.push('hq:h');
      }

      // Closed captions filter
      if (filters.cc) {
        tbsParts.push('cc:1');
      }
    }

    if (tbsParts.length > 0) {
      searchParams.set('tbs', tbsParts.join(','));
    }

    return {
      url: `https://www.google.com/search?${searchParams.toString()}`,
      method: 'GET',
      headers: {
        'User-Agent': chromeUserAgent,
        Accept: 'text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8',
        'Accept-Language': 'en-US,en;q=0.9',
      },
      cookies: ['CONSENT=YES+'],
    };
  }

  parseResponse(body: string, _params: EngineParams): EngineResults {
    const results = newEngineResults();

    // Check for CAPTCHA/sorry page
    if (body.includes('sorry.google.com') || body.includes('/sorry/')) {
      return results;
    }

    // Strategy 1: Try to parse JSON-LD VideoObject data
    const jsonLdResults = this.parseJsonLd(body);
    if (jsonLdResults.length > 0) {
      results.results.push(...jsonLdResults);
      return results;
    }

    // Strategy 2: Parse HTML div.g blocks for video results
    const htmlResults = this.parseHtmlResults(body);
    results.results.push(...htmlResults);

    return results;
  }

  private parseJsonLd(body: string): EngineResults['results'] {
    const results: EngineResults['results'] = [];

    // Find all JSON-LD script tags
    const jsonLdPattern = /<script[^>]*type\s*=\s*["']application\/ld\+json["'][^>]*>([\s\S]*?)<\/script>/gi;
    let match: RegExpExecArray | null;

    while ((match = jsonLdPattern.exec(body)) !== null) {
      try {
        const jsonContent = match[1].trim();
        const data = JSON.parse(jsonContent);

        // Handle both single objects and arrays
        const items = Array.isArray(data) ? data : [data];

        for (const item of items) {
          if (item['@type'] === 'VideoObject') {
            const video = item as VideoObjectJsonLd;
            const result = this.parseVideoObject(video);
            if (result) {
              results.push(result);
            }
          }
          // Also check for nested VideoObject in @graph
          if (item['@graph'] && Array.isArray(item['@graph'])) {
            for (const graphItem of item['@graph']) {
              if (graphItem['@type'] === 'VideoObject') {
                const result = this.parseVideoObject(graphItem as VideoObjectJsonLd);
                if (result) {
                  results.push(result);
                }
              }
            }
          }
        }
      } catch {
        // JSON parse failed, continue to next script tag
      }
    }

    return results;
  }

  private parseVideoObject(video: VideoObjectJsonLd): EngineResults['results'][number] | null {
    const url = video.contentUrl || video.embedUrl;
    const title = video.name;

    if (!url || !title) return null;

    // Get thumbnail
    let thumbnailUrl = '';
    if (video.thumbnailUrl) {
      thumbnailUrl = Array.isArray(video.thumbnailUrl)
        ? video.thumbnailUrl[0]
        : video.thumbnailUrl;
    }

    // Fallback to YouTube thumbnail if URL is YouTube
    if (!thumbnailUrl) {
      const videoId = extractYouTubeVideoId(url);
      if (videoId) {
        thumbnailUrl = getYouTubeThumbnail(videoId);
      }
    }

    // Parse ISO 8601 duration (e.g., PT1H2M3S)
    let duration = '';
    if (video.duration) {
      duration = this.parseIsoDuration(video.duration);
    }

    // Get channel/author
    const channel = video.author?.name || '';

    // Get view count
    const views = video.interactionStatistic?.interactionCount || 0;

    // Get embed URL
    let embedUrl = video.embedUrl || '';
    if (!embedUrl) {
      const videoId = extractYouTubeVideoId(url);
      if (videoId) {
        embedUrl = `https://www.youtube-nocookie.com/embed/${videoId}`;
      }
    }

    return {
      url,
      title,
      content: video.description || '',
      engine: this.name,
      score: this.weight,
      category: 'videos',
      template: 'videos',
      thumbnailUrl,
      duration,
      channel,
      views,
      embedUrl,
      publishedAt: video.uploadDate || '',
    };
  }

  private parseHtmlResults(body: string): EngineResults['results'] {
    const results: EngineResults['results'] = [];

    // Find div.g elements (Google's standard result container)
    const gElements = findElements(body, 'div.g');

    for (const el of gElements) {
      const result = this.parseGResult(el);
      if (result) {
        results.push(result);
      }
    }

    return results;
  }

  private parseGResult(html: string): EngineResults['results'][number] | null {
    // Find URL from first <a> tag with href
    let url = '';
    const hrefMatch = html.match(/<a\b[^>]*?\bhref\s*=\s*"([^"]+)"/i);
    if (hrefMatch) {
      const href = decodeHtmlEntities(hrefMatch[1]);
      if (href.startsWith('/url?')) {
        url = unwrapGoogleUrl(href);
      } else if (href.startsWith('http://') || href.startsWith('https://')) {
        url = href;
      }
    }

    if (!url) return null;

    // Filter Google internal URLs (but keep video platforms)
    if (url.includes('google.com/search') || url.includes('google.com/sorry')) {
      return null;
    }

    // Find title from <h3>
    let title = '';
    const h3Match = html.match(/<h3[^>]*>([\s\S]*?)<\/h3>/i);
    if (h3Match) {
      title = extractText(h3Match[1]).trim();
    }
    if (!title) {
      // Fallback: text inside the first <a> that has substantial content
      const aContent = html.match(/<a\b[^>]*>([\s\S]*?)<\/a>/i);
      if (aContent) {
        const text = extractText(aContent[1]).trim();
        if (text.length > 5) {
          title = text;
        }
      }
    }

    if (!title) return null;

    // Find content/description from various snippet classes
    let content = '';
    const snippetPatterns = [
      /class="[^"]*VwiC3b[^"]*"[^>]*>([\s\S]*?)<\/(?:div|span)>/i,
      /class="[^"]*IsZvec[^"]*"[^>]*>([\s\S]*?)<\/(?:div|span)>/i,
      /data-sncf="1"[^>]*>([\s\S]*?)<\/div>/i,
      /<span[^>]*class="[^"]*aCOpRe[^"]*"[^>]*>([\s\S]*?)<\/span>/i,
    ];
    for (const pattern of snippetPatterns) {
      const match = html.match(pattern);
      if (match) {
        const text = extractText(match[1]).trim();
        if (text.length > content.length) {
          content = text;
        }
      }
    }

    // Try to extract thumbnail URL from img tags
    let thumbnailUrl = '';
    const imgMatch = html.match(/<img[^>]*src\s*=\s*"([^"]+)"[^>]*>/i);
    if (imgMatch && !imgMatch[1].includes('data:')) {
      thumbnailUrl = imgMatch[1];
    }

    // Fallback to YouTube thumbnail if URL is YouTube
    if (!thumbnailUrl) {
      const videoId = extractYouTubeVideoId(url);
      if (videoId) {
        thumbnailUrl = getYouTubeThumbnail(videoId);
      }
    }

    // Try to extract duration from text (e.g., "3:45" or "1:23:45")
    let duration = '';
    const durationMatch = html.match(/(?:^|[^\d])(\d{1,2}:\d{2}(?::\d{2})?)(?:[^\d]|$)/);
    if (durationMatch) {
      duration = durationMatch[1];
    }

    // Try to extract channel/source
    let channel = '';
    const channelPatterns = [
      /class="[^"]*NJjxre[^"]*"[^>]*>([\s\S]*?)<\/(?:div|span)>/i,
      /class="[^"]*pcJO7e[^"]*"[^>]*>([\s\S]*?)<\/cite>/i,
    ];
    for (const pattern of channelPatterns) {
      const match = html.match(pattern);
      if (match) {
        const text = extractText(match[1]).trim();
        if (text && !text.includes('http')) {
          channel = text;
          break;
        }
      }
    }

    // Build embed URL for YouTube videos
    let embedUrl = '';
    const videoId = extractYouTubeVideoId(url);
    if (videoId) {
      embedUrl = `https://www.youtube-nocookie.com/embed/${videoId}`;
    }

    return {
      url,
      title,
      content,
      engine: this.name,
      score: this.weight,
      category: 'videos',
      template: 'videos',
      thumbnailUrl,
      duration,
      channel,
      embedUrl,
    };
  }

  /**
   * Parse ISO 8601 duration (PT1H2M3S) to human-readable format (1:02:03)
   */
  private parseIsoDuration(iso: string): string {
    const match = iso.match(/PT(?:(\d+)H)?(?:(\d+)M)?(?:(\d+)S)?/);
    if (!match) return '';

    const hours = parseInt(match[1] || '0', 10);
    const minutes = parseInt(match[2] || '0', 10);
    const seconds = parseInt(match[3] || '0', 10);

    if (hours > 0) {
      return `${hours}:${minutes.toString().padStart(2, '0')}:${seconds.toString().padStart(2, '0')}`;
    }
    return `${minutes}:${seconds.toString().padStart(2, '0')}`;
  }
}
