/**
 * Google Search Engine adapters.
 * Ported from Go: pkg/engine/local/engines/google.go
 *
 * Uses SearXNG-style GSA user agents and arc_id bypass techniques.
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

// ========== GSA User Agents ==========

const gsaUserAgents: string[] = [
  'Mozilla/5.0 (iPhone; CPU iPhone OS 17_6_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) GSA/399.2.845414227 Mobile/15E148 Safari/604.1',
  'Mozilla/5.0 (iPhone; CPU iPhone OS 18_3_2 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) GSA/399.2.845414227 Mobile/15E148 Safari/604.1',
  'Mozilla/5.0 (iPhone; CPU iPhone OS 18_5_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) GSA/399.2.845414227 Mobile/15E148 Safari/604.1',
  'Mozilla/5.0 (Linux; Android 14; SM-S928B Build/UP1A.231005.007) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.6099.230 Mobile Safari/537.36 GSA/15.3.36.28.arm64',
  'Mozilla/5.0 (Linux; Android 13; Pixel 7 Pro) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.6099.144 Mobile Safari/537.36 GSA/14.50.15.29.arm64',
];

const ARC_ID_RANGE =
  'abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_-';

// Module-level arc_id cache (regenerates every hour)
let cachedArcId = '';
let arcIdTimestamp = 0;

/**
 * Generate a 23-character random arc_id string (like SearXNG).
 */
function generateArcId(): string {
  const bytes = new Uint8Array(23);
  crypto.getRandomValues(bytes);
  let result = '';
  for (let i = 0; i < 23; i++) {
    result += ARC_ID_RANGE[bytes[i] % ARC_ID_RANGE.length];
  }
  return result;
}

/**
 * Get or regenerate the cached arc_id (changes hourly).
 */
function getArcId(): string {
  const now = Date.now();
  if (!cachedArcId || now - arcIdTimestamp > 3600_000) {
    cachedArcId = generateArcId();
    arcIdTimestamp = now;
  }
  return cachedArcId;
}

/**
 * Format the async parameter for Google's UI request.
 * Format: arc_id:srp_<random_23_chars>_1<start:02>,use_ac:true,_fmt:prog
 */
function uiAsync(start: number): string {
  const startPadded = start.toString().padStart(2, '0');
  const arcId = `arc_id:srp_${getArcId()}_1${startPadded}`;
  return `${arcId},use_ac:true,_fmt:prog`;
}

function getRandomGSAUserAgent(): string {
  const idx = Math.floor(Math.random() * gsaUserAgents.length);
  return gsaUserAgents[idx];
}

// ========== Time Range & Safe Search Maps ==========

const timeRangeMap: Record<string, string> = {
  day: 'd',
  week: 'w',
  month: 'm',
  year: 'y',
};

const safeSearchMap: Record<number, string> = {
  0: 'off',
  1: 'medium',
  2: 'high',
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

// ========== GoogleEngine ==========

export class GoogleEngine implements OnlineEngine {
  name = 'google';
  shortcut = 'g';
  categories: Category[] = ['general'];
  supportsPaging = true;
  maxPage = 50;
  timeout = 10_000;
  weight = 1.0;
  disabled = false;

  buildRequest(query: string, params: EngineParams): RequestConfig {
    const start = (params.page - 1) * 10;
    const asyncParam = uiAsync(start);

    // Derive language/region
    const locale = params.locale || 'en-US';
    const parts = locale.split('-');
    const langCode = parts[0] || 'en';
    const regionCode = parts[1] || 'US';

    const searchParams = new URLSearchParams();
    searchParams.set('q', query);
    searchParams.set('hl', `${langCode}-${regionCode}`);

    if (locale !== 'all') {
      searchParams.set('lr', `lang_${langCode}`);
    }
    if (locale.includes('-')) {
      searchParams.set('cr', `country${regionCode}`);
    }

    searchParams.set('ie', 'utf8');
    searchParams.set('oe', 'utf8');
    searchParams.set('filter', '0');
    searchParams.set('start', start.toString());
    searchParams.set('asearch', 'arc');
    searchParams.set('async', asyncParam);

    // Time range
    if (params.timeRange && timeRangeMap[params.timeRange]) {
      searchParams.set('tbs', `qdr:${timeRangeMap[params.timeRange]}`);
    }

    // Safe search
    const safeValue = safeSearchMap[params.safeSearch];
    if (safeValue) {
      searchParams.set('safe', safeValue);
    }

    return {
      url: `https://www.google.com/search?${searchParams.toString()}`,
      method: 'GET',
      headers: {
        'User-Agent': getRandomGSAUserAgent(),
        Accept: '*/*',
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

    // Parse MjjYud containers (primary result format)
    const mjjYudElements = findElements(body, 'div.MjjYud');
    for (const el of mjjYudElements) {
      const result = this.parseMjjYudResult(el);
      if (result) {
        results.results.push(result);
      }
    }

    // Fallback: parse "g" class containers
    if (results.results.length === 0) {
      const gElements = findElements(body, 'div.g');
      for (const el of gElements) {
        if (el.includes('g-blk')) continue;
        const result = this.parseGResult(el);
        if (result) {
          results.results.push(result);
        }
      }
    }

    // Parse suggestions from ouy7Mc class
    const suggestionElements = findElements(body, 'div.ouy7Mc');
    for (const el of suggestionElements) {
      const linkPattern = /<a\b[^>]*>([^<]*(?:<[^/][^>]*>[^<]*)*)<\/a>/gi;
      let match: RegExpExecArray | null;
      while ((match = linkPattern.exec(el)) !== null) {
        const text = extractText(match[1]).trim();
        if (text) {
          results.suggestions.push(text);
        }
      }
    }

    return results;
  }

  private parseMjjYudResult(
    html: string
  ): EngineResults['results'][number] | null {
    // Find title from div[role="link"]
    let title = '';
    const roleLinkElements = findElements(html, 'div[role=\'link\']');
    if (roleLinkElements.length > 0) {
      title = extractText(roleLinkElements[0]).trim();
    }

    // Find URL from first <a> tag
    let url = '';
    const hrefMatch = html.match(
      /<a\b[^>]*?\bhref\s*=\s*"([^"]+)"/i
    );
    if (hrefMatch) {
      const href = decodeHtmlEntities(hrefMatch[1]);
      if (!href.startsWith('#')) {
        if (href.startsWith('/url?')) {
          url = unwrapGoogleUrl(href);
        } else if (href.startsWith('http://') || href.startsWith('https://')) {
          url = href;
        }
      }
    }

    if (!url || !title) return null;

    // Filter Google internal URLs
    if (url.includes('google.com') && !url.includes('translate.google')) {
      return null;
    }

    // Find content from div[data-sncf="1"]
    let content = '';
    const sncfElements = findElements(html, 'div[data-sncf=\'1\']');
    if (sncfElements.length > 0) {
      // Extract text without scripts
      let cleaned = sncfElements[0].replace(
        /<script[^>]*>[\s\S]*?<\/script>/gi,
        ''
      );
      content = extractText(cleaned).trim();
    }

    return {
      url,
      title,
      content,
      engine: this.name,
      score: this.weight,
      category: 'general',
    };
  }

  private parseGResult(
    html: string
  ): EngineResults['results'][number] | null {
    // Find link with href in <a> tag
    let url = '';
    let title = '';

    const linkMatch = html.match(
      /<a\b[^>]*?\bhref\s*=\s*"(https?:\/\/[^"]+|\/url\?[^"]+)"/i
    );
    if (linkMatch) {
      const href = decodeHtmlEntities(linkMatch[1]);
      if (href.startsWith('/url?')) {
        url = unwrapGoogleUrl(href);
      } else {
        url = href;
      }
    }

    if (!url) return null;

    // Filter Google internal URLs
    if (
      url.includes('google.com/search') ||
      (url.includes('google.com') && !url.includes('translate.google'))
    ) {
      return null;
    }

    // Title from <h3>
    const h3Match = html.match(/<h3[^>]*>([\s\S]*?)<\/h3>/i);
    if (h3Match) {
      title = extractText(h3Match[1]).trim();
    }
    if (!title) {
      // Fallback: text inside the first <a>
      const aContent = html.match(/<a\b[^>]*>([\s\S]*?)<\/a>/i);
      if (aContent) {
        title = extractText(aContent[1]).trim();
      }
    }

    if (!title) return null;

    // Content from snippet classes
    let content = '';
    const snippetPatterns = [
      /class="[^"]*VwiC3b[^"]*"[^>]*>([\s\S]*?)<\/(?:div|span)>/i,
      /class="[^"]*IsZvec[^"]*"[^>]*>([\s\S]*?)<\/(?:div|span)>/i,
      /data-sncf="1"[^>]*>([\s\S]*?)<\/div>/i,
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

    return {
      url,
      title,
      content,
      engine: this.name,
      score: this.weight,
      category: 'general',
    };
  }
}

// ========== Google Image Filter Mappings ==========

const googleSizeMap: Record<string, string> = {
  large: 'l',
  medium: 'm',
  small: 's',
  icon: 'i',
};

const googleColorMap: Record<string, string> = {
  color: 'color',
  gray: 'gray',
  transparent: 'trans',
  red: 'red',
  orange: 'orange',
  yellow: 'yellow',
  green: 'green',
  teal: 'teal',
  blue: 'blue',
  purple: 'purple',
  pink: 'pink',
  white: 'white',
  black: 'black',
  brown: 'brown',
};

const googleTypeMap: Record<string, string> = {
  face: 'face',
  photo: 'photo',
  clipart: 'clipart',
  lineart: 'lineart',
  animated: 'animated',
};

const googleAspectMap: Record<string, string> = {
  tall: 't',
  square: 's',
  wide: 'w',
  panoramic: 'xw',
};

const googleRightsMap: Record<string, string> = {
  creative_commons: 'cl',
  commercial: 'ol',
};

function buildGoogleImageTbs(params: EngineParams): string {
  const tbs: string[] = [];
  const filters = params.imageFilters;

  if (!filters) return '';

  // Size filter
  if (filters.size && filters.size !== 'any' && googleSizeMap[filters.size]) {
    tbs.push(`isz:${googleSizeMap[filters.size]}`);
  }

  // Custom size
  if (filters.minWidth || filters.minHeight) {
    const w = filters.minWidth || 0;
    const h = filters.minHeight || 0;
    tbs.push(`isz:ex,iszw:${w},iszh:${h}`);
  }

  // Color filter
  if (filters.color && filters.color !== 'any') {
    if (filters.color === 'color' || filters.color === 'gray' || filters.color === 'transparent') {
      tbs.push(`ic:${googleColorMap[filters.color]}`);
    } else if (googleColorMap[filters.color]) {
      tbs.push(`ic:specific,isc:${googleColorMap[filters.color]}`);
    }
  }

  // Type filter
  if (filters.type && filters.type !== 'any' && googleTypeMap[filters.type]) {
    tbs.push(`itp:${googleTypeMap[filters.type]}`);
  }

  // Aspect ratio filter
  if (filters.aspect && filters.aspect !== 'any' && googleAspectMap[filters.aspect]) {
    tbs.push(`iar:${googleAspectMap[filters.aspect]}`);
  }

  // Usage rights filter
  if (filters.rights && filters.rights !== 'any' && googleRightsMap[filters.rights]) {
    tbs.push(`sur:${googleRightsMap[filters.rights]}`);
  }

  // Time range
  if (params.timeRange && timeRangeMap[params.timeRange]) {
    tbs.push(`qdr:${timeRangeMap[params.timeRange]}`);
  }

  return tbs.join(',');
}

// ========== GoogleImagesEngine ==========

export class GoogleImagesEngine implements OnlineEngine {
  name = 'google images';
  shortcut = 'gi';
  categories: Category[] = ['images'];
  supportsPaging = true;
  maxPage = 50;
  timeout = 10_000;
  weight = 1.0;
  disabled = false;

  buildRequest(query: string, params: EngineParams): RequestConfig {
    const searchParams = new URLSearchParams();
    searchParams.set('q', query);
    searchParams.set('tbm', 'isch');
    searchParams.set('asearch', 'isch');
    searchParams.set('hl', 'en');

    // Safe search
    const safeValue = safeSearchMap[params.safeSearch];
    if (safeValue) {
      searchParams.set('safe', safeValue);
    } else {
      searchParams.set('safe', 'off');
    }

    // Build tbs parameter for filters
    const tbs = buildGoogleImageTbs(params);
    if (tbs) {
      searchParams.set('tbs', tbs);
    }

    // File type filter
    if (params.imageFilters?.filetype && params.imageFilters.filetype !== 'any') {
      searchParams.set('as_filetype', params.imageFilters.filetype);
    }

    // Zero-based pagination for images
    const ijn = params.page - 1;
    searchParams.set('async', `_fmt:json,p:1,ijn:${ijn}`);

    return {
      url: `https://www.google.com/search?${searchParams.toString()}`,
      method: 'GET',
      headers: {
        'User-Agent': getRandomGSAUserAgent(),
        Accept: '*/*',
      },
      cookies: ['CONSENT=YES+'],
    };
  }

  parseResponse(body: string, params: EngineParams): EngineResults {
    const results = newEngineResults();
    const filters = params.imageFilters;

    // Try to find ischj JSON in the response
    const jsonStart = body.indexOf('{"ischj"');
    if (jsonStart !== -1) {
      let jsonEnd = body.indexOf('\n', jsonStart);
      if (jsonEnd === -1) jsonEnd = body.length;
      const jsonStr = body.slice(jsonStart, jsonEnd);

      try {
        const data = JSON.parse(jsonStr) as {
          ischj?: {
            metadata?: Array<{
              result?: {
                referrer_url?: string;
                page_title?: string;
                site_title?: string;
              };
              original_image?: {
                url?: string;
                width?: number;
                height?: number;
              };
              thumbnail?: { url?: string };
            }>;
          };
        };

        if (data.ischj?.metadata) {
          for (const item of data.ischj.metadata) {
            if (item.original_image?.url) {
              const width = item.original_image.width || 0;
              const height = item.original_image.height || 0;

              // Client-side size filtering for custom dimensions
              if (filters?.maxWidth && width > filters.maxWidth) continue;
              if (filters?.maxHeight && height > filters.maxHeight) continue;

              results.results.push({
                url: item.result?.referrer_url || '',
                title: item.result?.page_title || '',
                content: '',
                engine: this.name,
                score: this.weight,
                category: 'images',
                template: 'images',
                imageUrl: item.original_image.url,
                thumbnailUrl: item.thumbnail?.url || '',
                source: item.result?.site_title || '',
                resolution: `${width}x${height}`,
              });
            }
          }
        }
      } catch {
        // JSON parse failed, fall through to regex
      }
    }

    // Fallback: extract image URLs using regex
    if (results.results.length === 0) {
      const re =
        /\["(https:\/\/[^"]+\.(?:jpg|jpeg|png|gif|webp)[^"]*)",(\d+),(\d+)\]/g;
      let match: RegExpExecArray | null;
      while ((match = re.exec(body)) !== null) {
        const width = parseInt(match[2], 10);
        const height = parseInt(match[3], 10);

        // Client-side size filtering
        if (filters?.maxWidth && width > filters.maxWidth) continue;
        if (filters?.maxHeight && height > filters.maxHeight) continue;

        results.results.push({
          url: match[1],
          title: '',
          content: '',
          engine: this.name,
          score: this.weight,
          category: 'images',
          template: 'images',
          imageUrl: match[1],
          resolution: `${width}x${height}`,
        });
      }
    }

    return results;
  }
}

// ========== GoogleReverseImageEngine ==========

export class GoogleReverseImageEngine implements OnlineEngine {
  name = 'google reverse';
  shortcut = 'gri';
  categories: Category[] = ['images'];
  supportsPaging = false;
  maxPage = 1;
  timeout = 15_000;
  weight = 1.0;
  disabled = false;

  buildRequest(query: string, _params: EngineParams): RequestConfig {
    // query is the image URL for reverse search
    return {
      url: `https://lens.google.com/uploadbyurl?url=${encodeURIComponent(query)}`,
      method: 'GET',
      headers: {
        'User-Agent': getRandomGSAUserAgent(),
        Accept: 'text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8',
      },
      cookies: ['CONSENT=YES+'],
    };
  }

  parseResponse(body: string, _params: EngineParams): EngineResults {
    const results = newEngineResults();

    // Parse visual matches from Lens response
    // The response contains JSON data embedded in the HTML
    const dataMatch = body.match(/AF_initDataCallback\(\{[^}]*data:(\[[\s\S]*?\])\s*,\s*sideChannel/);
    if (dataMatch) {
      try {
        // Extract image results from the nested structure
        const imgPattern = /"(https?:\/\/[^"]+\.(?:jpg|jpeg|png|gif|webp)[^"]*)"/gi;
        let match: RegExpExecArray | null;
        const seen = new Set<string>();

        while ((match = imgPattern.exec(body)) !== null) {
          const url = match[1];
          if (!seen.has(url) && !url.includes('google.com') && !url.includes('gstatic.com')) {
            seen.add(url);
            results.results.push({
              url,
              title: 'Visual match',
              content: '',
              engine: this.name,
              score: this.weight,
              category: 'images',
              template: 'images',
              imageUrl: url,
            });
          }
        }
      } catch {
        // Parse failed
      }
    }

    return results;
  }
}
