/**
 * Bing Search Engine adapters.
 * Ported from Go: pkg/engine/local/engines/bing.go
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

const BING_USER_AGENT =
  'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36';

// ========== URL Decoding ==========

/**
 * Decode Bing's base64-encoded redirect URLs.
 * Bing wraps URLs as /ck/a?u=a1<base64_encoded_url>&...
 */
function decodeBingUrl(bingUrl: string): string {
  try {
    const parsed = new URL(bingUrl);
    const paramU = parsed.searchParams.get('u');
    if (!paramU) return bingUrl;

    // Remove "a1" prefix (base64 URL encoding marker)
    if (paramU.length > 2 && paramU.startsWith('a1')) {
      let encoded = paramU.slice(2);
      // URL-safe base64 uses - and _ instead of + and /
      encoded = encoded.replace(/-/g, '+').replace(/_/g, '/');
      // Add padding if needed
      const padding = 4 - (encoded.length % 4);
      if (padding < 4) {
        encoded += '='.repeat(padding);
      }
      try {
        const decoded = atob(encoded);
        return decoded;
      } catch {
        return bingUrl;
      }
    }
  } catch {
    // URL parse failed
  }
  return bingUrl;
}

// ========== Time Range Maps ==========

const generalTimeRange: Record<string, string> = {
  day: '1',
  week: '2',
  month: '3',
  year: '5',
};

const imagesTimeRange: Record<string, number> = {
  day: 1440,
  week: 10080,
  month: 44640,
  year: 525600,
};

const newsTimeRange: Record<string, string> = {
  day: '4',
  week: '7',
  month: '9',
};

// ========== BingEngine (General Search) ==========

export class BingEngine implements OnlineEngine {
  name = 'bing';
  shortcut = 'b';
  categories: Category[] = ['general'];
  supportsPaging = true;
  maxPage = 200;
  timeout = 10_000;
  weight = 1.0;
  disabled = false;

  buildRequest(query: string, params: EngineParams): RequestConfig {
    const locale = params.locale || 'en-US';
    const parts = locale.split('-');
    const lang = (parts[0] || 'en').toLowerCase();
    const region = `${lang}-${(parts[1] || 'us').toLowerCase()}`;

    const searchParams = new URLSearchParams();
    searchParams.set('q', query);
    searchParams.set('pq', query);

    // Pagination
    if (params.page > 1) {
      const first = (params.page - 1) * 10 + 1;
      searchParams.set('first', first.toString());
      if (params.page === 2) {
        searchParams.set('FORM', 'PERE');
      } else {
        searchParams.set('FORM', `PERE${params.page - 2}`);
      }
    }

    // Time range
    if (params.timeRange && generalTimeRange[params.timeRange]) {
      const tr = generalTimeRange[params.timeRange];
      if (params.timeRange === 'year') {
        const unixDay = Math.floor(Date.now() / 86400000);
        searchParams.set(
          'filters',
          `ex1:"ez${tr}_${unixDay - 365}_${unixDay}"`
        );
      } else {
        searchParams.set('filters', `ex1:"ez${tr}"`);
      }
    }

    // Cookies for language/region
    const cookies = [
      `_EDGE_CD=m=${region}&u=${lang}`,
      `_EDGE_S=mkt=${region}&ui=${lang}`,
      `SRCHHPGUSR=SRCHLANG=${lang}`,
    ];

    return {
      url: `https://www.bing.com/search?${searchParams.toString()}`,
      method: 'GET',
      headers: {
        'User-Agent': BING_USER_AGENT,
        Accept:
          'text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8',
        'Accept-Language': 'en-US,en;q=0.9',
        DNT: '1',
        'Upgrade-Insecure-Requests': '1',
      },
      cookies,
    };
  }

  parseResponse(body: string, _params: EngineParams): EngineResults {
    const results = newEngineResults();

    // Find li.b_algo elements (Bing result items)
    const algoItems = findElements(body, 'li.b_algo');

    for (const item of algoItems) {
      const result = this.parseResult(item);
      if (result) {
        results.results.push(result);
      }
    }

    return results;
  }

  private parseResult(
    html: string
  ): EngineResults['results'][number] | null {
    let url = '';
    let title = '';
    let content = '';

    // Find h2 > a for title and URL
    const h2Match = html.match(/<h2[^>]*>([\s\S]*?)<\/h2>/i);
    if (h2Match) {
      const h2Content = h2Match[1];
      const linkMatch = h2Content.match(
        /<a\b[^>]*?\bhref\s*=\s*"([^"]+)"[^>]*>([\s\S]*?)<\/a>/i
      );
      if (linkMatch) {
        const href = decodeHtmlEntities(linkMatch[1]);
        if (href.startsWith('https://www.bing.com/ck/a?')) {
          url = decodeBingUrl(href);
        } else if (href.startsWith('http')) {
          url = href;
        }
        title = extractText(linkMatch[2]).trim();
      }
    }

    if (!url || !title) return null;

    // Find content in <p> or b_caption
    const pMatch = html.match(/<p[^>]*>([\s\S]*?)<\/p>/i);
    if (pMatch) {
      content = extractText(pMatch[1]).trim();
    }
    if (!content) {
      const captionElements = findElements(html, 'div.b_caption');
      if (captionElements.length > 0) {
        content = extractText(captionElements[0]).trim();
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

// ========== BingImagesEngine ==========

export class BingImagesEngine implements OnlineEngine {
  name = 'bing images';
  shortcut = 'bi';
  categories: Category[] = ['images'];
  supportsPaging = true;
  maxPage = 50;
  timeout = 10_000;
  weight = 1.0;
  disabled = false;

  buildRequest(query: string, params: EngineParams): RequestConfig {
    const searchParams = new URLSearchParams();
    searchParams.set('q', query);
    searchParams.set('async', '1');
    searchParams.set('count', '35');

    let first = 1;
    if (params.page > 1) {
      first = (params.page - 1) * 35 + 1;
    }
    searchParams.set('first', first.toString());

    // Time range filter
    if (params.timeRange && imagesTimeRange[params.timeRange]) {
      searchParams.set(
        'qft',
        `filterui:age-lt${imagesTimeRange[params.timeRange]}`
      );
    }

    return {
      url: `https://www.bing.com/images/async?${searchParams.toString()}`,
      method: 'GET',
      headers: {
        'User-Agent': BING_USER_AGENT,
        Accept: 'text/html',
      },
      cookies: [],
    };
  }

  parseResponse(body: string, _params: EngineParams): EngineResults {
    const results = newEngineResults();

    // Parse image metadata from JSON in HTML "m" attribute
    const iuscPattern = /class="iusc"[^>]*m="([^"]+)"/g;
    let match: RegExpExecArray | null;

    while ((match = iuscPattern.exec(body)) !== null) {
      // Decode HTML entities in the JSON string
      let jsonStr = match[1]
        .replace(/&quot;/g, '"')
        .replace(/&amp;/g, '&')
        .replace(/&lt;/g, '<')
        .replace(/&gt;/g, '>');

      try {
        const metadata = JSON.parse(jsonStr) as {
          purl?: string;
          murl?: string;
          turl?: string;
          desc?: string;
          t?: string;
        };

        if (metadata.murl) {
          results.results.push({
            url: metadata.purl || '',
            title: metadata.t || '',
            content: metadata.desc || '',
            engine: this.name,
            score: this.weight,
            category: 'images',
            template: 'images',
            imageUrl: metadata.murl,
            thumbnailUrl: metadata.turl || '',
          });
        }
      } catch {
        // JSON parse failed, skip this item
      }
    }

    // Fallback: extract murl from encoded format
    if (results.results.length === 0) {
      const fallbackPattern = /murl&quot;:&quot;([^&]+)&quot;/g;
      let fbMatch: RegExpExecArray | null;
      while ((fbMatch = fallbackPattern.exec(body)) !== null) {
        const imgUrl = decodeURIComponent(fbMatch[1]);
        if (imgUrl) {
          results.results.push({
            url: imgUrl,
            title: '',
            content: '',
            engine: this.name,
            score: this.weight,
            category: 'images',
            template: 'images',
            imageUrl: imgUrl,
          });
        }
      }
    }

    return results;
  }
}

// ========== BingNewsEngine ==========

export class BingNewsEngine implements OnlineEngine {
  name = 'bing news';
  shortcut = 'bn';
  categories: Category[] = ['news'];
  supportsPaging = true;
  maxPage = 50;
  timeout = 10_000;
  weight = 1.0;
  disabled = false;

  buildRequest(query: string, params: EngineParams): RequestConfig {
    const locale = params.locale || 'en-US';
    const parts = locale.split('-');
    const lang = (parts[0] || 'en').toLowerCase();
    const country = (parts[1] || 'us').toLowerCase();

    const searchParams = new URLSearchParams();
    searchParams.set('q', query);
    searchParams.set('InfiniteScroll', '1');
    searchParams.set('form', 'PTFTNR');
    searchParams.set('setlang', lang);
    searchParams.set('cc', country);

    // Pagination
    let first = 1;
    let sfx = 0;
    if (params.page > 1) {
      first = (params.page - 1) * 10 + 1;
      sfx = params.page - 1;
    }
    searchParams.set('first', first.toString());
    searchParams.set('SFX', sfx.toString());

    // Time range
    if (params.timeRange && newsTimeRange[params.timeRange]) {
      searchParams.set(
        'qft',
        `interval="${newsTimeRange[params.timeRange]}"`
      );
    }

    return {
      url: `https://www.bing.com/news/infinitescrollajax?${searchParams.toString()}`,
      method: 'GET',
      headers: {
        'User-Agent': BING_USER_AGENT,
        Accept: 'text/html',
      },
      cookies: [],
    };
  }

  parseResponse(body: string, _params: EngineParams): EngineResults {
    const results = newEngineResults();

    // Find news items
    const newsItems = findElements(body, 'div.newsitem');
    const newsCards = findElements(body, 'div.news-card');
    const allItems = [...newsItems, ...newsCards];

    for (const item of allItems) {
      const result = this.parseNewsResult(item);
      if (result) {
        results.results.push(result);
      }
    }

    return results;
  }

  private parseNewsResult(
    html: string
  ): EngineResults['results'][number] | null {
    let url = '';
    let title = '';
    let content = '';
    let source = '';
    let thumbnailUrl = '';

    // Find title link
    const linkPattern =
      /<a\b[^>]*?\bhref\s*=\s*"(https?:\/\/[^"]+)"[^>]*>([\s\S]*?)<\/a>/gi;
    let linkMatch: RegExpExecArray | null;
    while ((linkMatch = linkPattern.exec(html)) !== null) {
      const linkText = extractText(linkMatch[2]).trim();
      if (linkText && !url) {
        url = decodeHtmlEntities(linkMatch[1]);
        title = linkText;
        break;
      }
    }

    if (!url) return null;

    // Find snippet
    const snippetElements = findElements(html, 'div.snippet');
    const summaryElements = findElements(html, 'div.summary');
    const snippetHtml = snippetElements[0] || summaryElements[0] || '';
    if (snippetHtml) {
      content = extractText(snippetHtml).trim();
    }

    // Find thumbnail
    const imgMatch = html.match(
      /<img\b[^>]*?\bsrc\s*=\s*"([^"]+)"/i
    );
    if (imgMatch && !imgMatch[1].startsWith('data:image')) {
      thumbnailUrl = imgMatch[1];
      if (!thumbnailUrl.startsWith('http')) {
        thumbnailUrl = 'https://www.bing.com' + thumbnailUrl;
      }
    }

    // Find source
    const sourceElements = findElements(html, 'div.source');
    if (sourceElements.length > 0) {
      source = extractText(sourceElements[0]).trim();
    }

    return {
      url,
      title,
      content,
      engine: this.name,
      score: this.weight,
      category: 'news',
      template: 'news',
      thumbnailUrl,
      source,
    };
  }
}
