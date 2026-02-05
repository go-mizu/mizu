/**
 * Bing Videos Search Engine adapter.
 *
 * Uses Bing's video search to find video results.
 * Parses HTML response to extract video information from vrhm attributes.
 */

import type {
  OnlineEngine,
  EngineParams,
  RequestConfig,
  EngineResults,
  Category,
} from './engine';
import { newEngineResults } from './engine';
import { extractText, decodeHtmlEntities } from '../lib/html-parser';

// ========== Chrome User Agent ==========

const chromeUserAgent =
  'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36';

// ========== Time Range Map (in minutes) ==========

const timeRangeMap: Record<string, string> = {
  day: 'filterui:videoage-lt1440',     // 24 hours = 1440 minutes
  week: 'filterui:videoage-lt10080',   // 7 days = 10080 minutes
  month: 'filterui:videoage-lt43200',  // 30 days = 43200 minutes
  year: 'filterui:videoage-lt525600',  // 365 days = 525600 minutes
};

// ========== Safe Search Map ==========

const safeSearchMap: Record<number, string> = {
  0: 'off',
  1: 'moderate',
  2: 'strict',
};

// ========== Video Metadata Interface ==========

interface BingVideoMetadata {
  murl?: string;  // Video URL
  vt?: string;    // Video title
  du?: string;    // Duration
  thid?: string;  // Thumbnail ID
  purl?: string;  // Page URL
  desc?: string;  // Description
  vw?: string;    // View count (may be string or number)
  ch?: string;    // Channel name
  pu?: string;    // Publisher URL
}

// ========== Helper Functions ==========

/**
 * Build Bing video thumbnail URL from thumbnail ID.
 */
function buildThumbnailUrl(thid: string): string {
  if (!thid) return '';
  return `https://tse1.mm.bing.net/th?id=${thid}`;
}

/**
 * Parse duration string to human-readable format.
 * Bing returns duration in various formats like "3:45" or milliseconds.
 */
function parseDuration(du: string | undefined): string {
  if (!du) return '';

  // If already in MM:SS or H:MM:SS format, return as-is
  if (/^\d{1,2}:\d{2}(:\d{2})?$/.test(du)) {
    return du;
  }

  // Try to parse as number (milliseconds or seconds)
  const num = parseInt(du, 10);
  if (isNaN(num)) return du;

  // If it's a large number, treat as milliseconds
  const seconds = num > 1000000 ? Math.floor(num / 1000) : num;
  const hours = Math.floor(seconds / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);
  const secs = seconds % 60;

  if (hours > 0) {
    return `${hours}:${minutes.toString().padStart(2, '0')}:${secs.toString().padStart(2, '0')}`;
  }
  return `${minutes}:${secs.toString().padStart(2, '0')}`;
}

/**
 * Parse view count string to number.
 */
function parseViews(vw: string | undefined): number | undefined {
  if (!vw) return undefined;

  // Handle strings like "1.2M views" or "50K"
  const text = vw.toLowerCase().replace(/[,\s]/g, '');
  const match = text.match(/^([\d.]+)([kmb])?/);
  if (!match) return undefined;

  let num = parseFloat(match[1]);
  const suffix = match[2];

  if (suffix === 'k') num *= 1000;
  else if (suffix === 'm') num *= 1000000;
  else if (suffix === 'b') num *= 1000000000;

  return Math.floor(num);
}

// ========== BingVideosEngine ==========

export class BingVideosEngine implements OnlineEngine {
  name = 'bing_videos';
  shortcut = 'biv';
  categories: Category[] = ['videos'];
  supportsPaging = true;
  maxPage = 10;
  timeout = 10_000;
  weight = 0.95;
  disabled = false;

  buildRequest(query: string, params: EngineParams): RequestConfig {
    const searchParams = new URLSearchParams();
    searchParams.set('q', query);
    searchParams.set('count', '35');

    // Pagination: first = (page-1) * 35 + 1
    const first = params.page > 1 ? (params.page - 1) * 35 + 1 : 1;
    searchParams.set('first', first.toString());

    // Time range filter
    if (params.timeRange && timeRangeMap[params.timeRange]) {
      searchParams.set('filters', timeRangeMap[params.timeRange]);
      searchParams.set('form', 'VRFLTR');
    }

    // Build cookies array
    const cookies: string[] = [];

    // Safe search via cookie
    const safeValue = safeSearchMap[params.safeSearch] ?? 'moderate';
    cookies.push(`SRCHHPGUSR=ADLT=${safeValue}`);

    return {
      url: `https://www.bing.com/videos/search?${searchParams.toString()}`,
      method: 'GET',
      headers: {
        'User-Agent': chromeUserAgent,
        Accept: 'text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8',
        'Accept-Language': 'en-US,en;q=0.9',
      },
      cookies,
    };
  }

  parseResponse(body: string, _params: EngineParams): EngineResults {
    const results = newEngineResults();

    // Check for bot detection/sorry pages
    if (body.includes('sorry.bing.com') || body.includes('/sorry/')) {
      return results;
    }

    // Strategy 1: Parse divs with vrhm attribute containing JSON metadata
    const vrhmResults = this.parseVrhmResults(body);
    if (vrhmResults.length > 0) {
      results.results.push(...vrhmResults);
      return results;
    }

    // Strategy 2: Fallback to mc_vtvc link classes
    const fallbackResults = this.parseFallbackResults(body);
    results.results.push(...fallbackResults);

    return results;
  }

  private parseVrhmResults(body: string): EngineResults['results'] {
    const results: EngineResults['results'] = [];

    // Match divs with vrhm attribute containing JSON metadata
    // The vrhm attribute contains URL-encoded JSON data
    const vrhmPattern = /\bvrhm\s*=\s*"([^"]+)"/g;
    let match: RegExpExecArray | null;

    while ((match = vrhmPattern.exec(body)) !== null) {
      try {
        // Decode HTML entities and URL encoding
        let jsonStr = decodeHtmlEntities(match[1]);
        jsonStr = decodeURIComponent(jsonStr);

        const metadata = JSON.parse(jsonStr) as BingVideoMetadata;

        if (metadata.murl) {
          const result = this.buildResultFromMetadata(metadata);
          if (result) {
            results.push(result);
          }
        }
      } catch {
        // JSON parse failed, continue to next match
      }
    }

    // Also try alternative patterns for the metadata
    if (results.length === 0) {
      const altPatterns = [
        // Data attribute pattern
        /data-vrhm\s*=\s*"([^"]+)"/g,
        // Pattern in onclick handlers
        /"vrhm"\s*:\s*"([^"]+)"/g,
      ];

      for (const pattern of altPatterns) {
        let altMatch: RegExpExecArray | null;
        while ((altMatch = pattern.exec(body)) !== null) {
          try {
            let jsonStr = decodeHtmlEntities(altMatch[1]);
            try {
              jsonStr = decodeURIComponent(jsonStr);
            } catch {
              // May already be decoded
            }

            const metadata = JSON.parse(jsonStr) as BingVideoMetadata;
            if (metadata.murl) {
              const result = this.buildResultFromMetadata(metadata);
              if (result) {
                results.push(result);
              }
            }
          } catch {
            // Continue to next match
          }
        }
        if (results.length > 0) break;
      }
    }

    // Try parsing JSON directly from script tags or data attributes
    if (results.length === 0) {
      const murlPatterns = [
        /"murl"\s*:\s*"([^"]+)"/g,
        /murl&quot;:&quot;([^&]+)&quot;/g,
      ];

      for (const pattern of murlPatterns) {
        let murlMatch: RegExpExecArray | null;
        while ((murlMatch = pattern.exec(body)) !== null) {
          let url = murlMatch[1];
          try {
            url = decodeURIComponent(url.replace(/\\u([0-9a-fA-F]{4})/g,
              (_, code) => String.fromCharCode(parseInt(code, 16))));
          } catch {
            // Use as-is
          }

          if (url && url.startsWith('http')) {
            // Try to find associated title near this match
            const context = body.slice(Math.max(0, murlMatch.index - 500), murlMatch.index + 500);
            let title = '';
            const titleMatch = context.match(/"vt"\s*:\s*"([^"]+)"/);
            if (titleMatch) {
              title = decodeHtmlEntities(titleMatch[1]);
            }

            results.push({
              url,
              title: title || 'Video',
              content: '',
              engine: this.name,
              score: this.weight,
              category: 'videos',
              template: 'videos',
            });
          }
        }
        if (results.length > 0) break;
      }
    }

    return results;
  }

  private buildResultFromMetadata(metadata: BingVideoMetadata): EngineResults['results'][number] | null {
    const url = metadata.murl;
    if (!url) return null;

    const title = metadata.vt ? decodeHtmlEntities(metadata.vt) : '';
    if (!title) return null;

    const thumbnailUrl = metadata.thid ? buildThumbnailUrl(metadata.thid) : '';
    const duration = parseDuration(metadata.du);
    const views = parseViews(metadata.vw);
    const channel = metadata.ch ? decodeHtmlEntities(metadata.ch) : '';
    const content = metadata.desc ? decodeHtmlEntities(metadata.desc) : '';

    // Build embed URL for YouTube videos
    let embedUrl = '';
    const ytMatch = url.match(/(?:youtube\.com\/watch\?v=|youtu\.be\/)([a-zA-Z0-9_-]{11})/);
    if (ytMatch) {
      embedUrl = `https://www.youtube-nocookie.com/embed/${ytMatch[1]}`;
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
      views,
      embedUrl,
    };
  }

  private parseFallbackResults(body: string): EngineResults['results'] {
    const results: EngineResults['results'] = [];

    // Look for mc_vtvc video card links
    const vtvcPattern = /<a[^>]*class="[^"]*mc_vtvc[^"]*"[^>]*href="([^"]+)"[^>]*>([\s\S]*?)<\/a>/gi;
    let match: RegExpExecArray | null;

    while ((match = vtvcPattern.exec(body)) !== null) {
      const href = decodeHtmlEntities(match[1]);
      const content = match[2];

      // Skip Bing redirect URLs
      if (!href || href.includes('bing.com/videos/search')) continue;

      // Extract title from content
      let title = '';
      const titleMatch = content.match(/<div[^>]*class="[^"]*mc_vtvc_title[^"]*"[^>]*>([\s\S]*?)<\/div>/i);
      if (titleMatch) {
        title = extractText(titleMatch[1]).trim();
      }
      if (!title) {
        title = extractText(content).trim().slice(0, 100);
      }

      if (!title) continue;

      // Extract thumbnail
      let thumbnailUrl = '';
      const imgMatch = content.match(/<img[^>]*src="([^"]+)"/i);
      if (imgMatch) {
        thumbnailUrl = decodeHtmlEntities(imgMatch[1]);
      }

      // Extract duration
      let duration = '';
      const durMatch = content.match(/(\d{1,2}:\d{2}(?::\d{2})?)/);
      if (durMatch) {
        duration = durMatch[1];
      }

      results.push({
        url: href,
        title,
        content: '',
        engine: this.name,
        score: this.weight,
        category: 'videos',
        template: 'videos',
        thumbnailUrl,
        duration,
      });
    }

    // Alternative: look for video tiles
    if (results.length === 0) {
      const tilePattern = /<div[^>]*class="[^"]*dg_u[^"]*"[^>]*>([\s\S]*?)<\/div>\s*<\/div>/gi;
      let tileMatch: RegExpExecArray | null;

      while ((tileMatch = tilePattern.exec(body)) !== null) {
        const tileContent = tileMatch[1];

        // Find video link
        const linkMatch = tileContent.match(/<a[^>]*href="(https?:\/\/[^"]+)"[^>]*>/i);
        if (!linkMatch) continue;

        const url = decodeHtmlEntities(linkMatch[1]);

        // Skip Bing internal links
        if (url.includes('bing.com')) continue;

        // Extract title
        const titleMatch = tileContent.match(/<div[^>]*class="[^"]*mc_vtvc_title[^"]*"[^>]*>([\s\S]*?)<\/div>/i);
        const title = titleMatch ? extractText(titleMatch[1]).trim() : extractText(tileContent).slice(0, 80);

        if (!title) continue;

        results.push({
          url,
          title,
          content: '',
          engine: this.name,
          score: this.weight,
          category: 'videos',
          template: 'videos',
        });
      }
    }

    return results;
  }
}
