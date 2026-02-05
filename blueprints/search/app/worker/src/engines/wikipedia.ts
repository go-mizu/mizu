/**
 * Wikipedia Search Engine adapter.
 * Ported from Go: pkg/engine/local/engines/wikipedia.go
 *
 * Uses the MediaWiki API (JSON) with support for 10+ languages.
 */

import type {
  OnlineEngine,
  EngineParams,
  RequestConfig,
  EngineResults,
  Category,
} from './engine';
import { newEngineResults } from './engine';

// Language map: ISO code -> Wikipedia subdomain
const languageMap: Record<string, string> = {
  en: 'en',
  de: 'de',
  fr: 'fr',
  es: 'es',
  it: 'it',
  pt: 'pt',
  ja: 'ja',
  ko: 'ko',
  zh: 'zh',
  ru: 'ru',
  ar: 'ar',
  hi: 'hi',
  nl: 'nl',
  pl: 'pl',
  sv: 'sv',
  vi: 'vi',
  uk: 'uk',
  he: 'he',
  id: 'id',
  cs: 'cs',
  fi: 'fi',
  da: 'da',
  no: 'no',
  hu: 'hu',
  ro: 'ro',
  tr: 'tr',
  th: 'th',
  el: 'el',
  fa: 'fa',
  ca: 'ca',
};

/**
 * Strip HTML tags from a string and decode entities.
 */
function stripHtmlTags(s: string): string {
  let result = s;
  // Remove all HTML tags
  let start = result.indexOf('<');
  while (start !== -1) {
    const end = result.indexOf('>', start);
    if (end === -1) break;
    result = result.slice(0, start) + result.slice(end + 1);
    start = result.indexOf('<');
  }
  // Decode HTML entities
  result = result.replace(/&quot;/g, '"');
  result = result.replace(/&amp;/g, '&');
  result = result.replace(/&lt;/g, '<');
  result = result.replace(/&gt;/g, '>');
  result = result.replace(/&#39;/g, "'");
  result = result.replace(/&nbsp;/g, ' ');
  return result.trim();
}

export class WikipediaEngine implements OnlineEngine {
  name = 'wikipedia';
  shortcut = 'w';
  categories: Category[] = ['general'];
  supportsPaging = true;
  maxPage = 10;
  timeout = 5_000;
  weight = 1.0;
  disabled = false;

  buildRequest(query: string, params: EngineParams): RequestConfig {
    // Determine language from locale
    let lang = 'en';
    if (params.locale) {
      const parts = params.locale.split('-');
      const langCode = parts[0].toLowerCase();
      if (languageMap[langCode]) {
        lang = languageMap[langCode];
      }
    }

    const searchParams = new URLSearchParams();
    searchParams.set('action', 'query');
    searchParams.set('list', 'search');
    searchParams.set('srsearch', query);
    searchParams.set('srwhat', 'text');
    searchParams.set('srlimit', '10');
    searchParams.set('srprop', 'snippet|titlesnippet|timestamp');
    searchParams.set('format', 'json');
    searchParams.set('utf8', '1');

    if (params.page > 1) {
      searchParams.set('sroffset', ((params.page - 1) * 10).toString());
    }

    return {
      url: `https://${lang}.wikipedia.org/w/api.php?${searchParams.toString()}`,
      method: 'GET',
      headers: {
        Accept: 'application/json',
        'User-Agent':
          'MizuSearch/1.0 (https://github.com/go-mizu/mizu; mizu@example.com)',
      },
      cookies: [],
    };
  }

  parseResponse(body: string, _params: EngineParams): EngineResults {
    const results = newEngineResults();

    try {
      const data = JSON.parse(body) as {
        query?: {
          search?: Array<{
            ns?: number;
            title?: string;
            pageid?: number;
            snippet?: string;
            timestamp?: string;
          }>;
          searchinfo?: { totalhits?: number };
        };
      };

      if (!data.query?.search) return results;

      // Extract language from the URL that was used
      // We'll use 'en' as default; the actual lang was encoded in the URL
      const lang = 'en';

      for (const item of data.query.search) {
        if (!item.title) continue;

        const articleUrl = `https://${lang}.wikipedia.org/wiki/${encodeURIComponent(item.title.replace(/ /g, '_'))}`;
        const snippet = stripHtmlTags(item.snippet || '');

        results.results.push({
          url: articleUrl,
          title: item.title,
          content: snippet,
          engine: this.name,
          score: this.weight,
          category: 'general',
        });
      }
    } catch {
      // Parse error
    }

    return results;
  }
}
