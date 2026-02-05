/**
 * Brave Search Engine adapter.
 * Ported from Go: pkg/engine/local/engines/brave.go
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

const timeRangeMap: Record<string, string> = {
  day: 'pd',
  week: 'pw',
  month: 'pm',
  year: 'py',
};

export class BraveEngine implements OnlineEngine {
  name = 'brave';
  shortcut = 'br';
  categories: Category[] = ['general'];
  supportsPaging = true;
  maxPage = 50;
  timeout = 5_000;
  weight = 1.0;
  disabled = false;

  buildRequest(query: string, params: EngineParams): RequestConfig {
    const searchParams = new URLSearchParams();
    searchParams.set('q', query);
    searchParams.set('source', 'web');

    // Pagination
    if (params.page > 1) {
      searchParams.set('offset', (params.page - 1).toString());
    }

    // Time range
    if (params.timeRange && timeRangeMap[params.timeRange]) {
      searchParams.set('tf', timeRangeMap[params.timeRange]);
    }

    // Safe search cookie
    let safeValue = 'moderate';
    if (params.safeSearch === 2) {
      safeValue = 'strict';
    } else if (params.safeSearch === 0) {
      safeValue = 'off';
    }

    return {
      url: `https://search.brave.com/search?${searchParams.toString()}`,
      method: 'GET',
      headers: {
        Accept: 'text/html',
        'Accept-Language': 'en-US,en;q=0.9',
      },
      cookies: [`safesearch=${safeValue}`],
    };
  }

  parseResponse(body: string, _params: EngineParams): EngineResults {
    const results = newEngineResults();

    // Find "snippet" class divs
    const snippetElements = findElements(body, 'div.snippet');

    for (const el of snippetElements) {
      const result = this.parseResult(el);
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

    // Find <a> with href starting with http
    const linkPattern =
      /<a\b[^>]*?\bhref\s*=\s*"(https?:\/\/[^"]+)"[^>]*>([\s\S]*?)<\/a>/gi;
    let linkMatch: RegExpExecArray | null;
    while ((linkMatch = linkPattern.exec(html)) !== null) {
      const href = decodeHtmlEntities(linkMatch[1]);
      if (href.startsWith('http')) {
        url = href;
        title = extractText(linkMatch[2]).trim();
        break;
      }
    }

    if (!url || !title) return null;

    // Find content in div with "content" class
    const contentElements = findElements(html, 'div.content');
    if (contentElements.length > 0) {
      const text = extractText(contentElements[0]).trim();
      if (text.length > content.length) {
        content = text;
      }
    }

    // Also try snippet-description class
    const descElements = findElements(html, 'div.snippet-description');
    if (descElements.length > 0) {
      const text = extractText(descElements[0]).trim();
      if (text.length > content.length) {
        content = text;
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
