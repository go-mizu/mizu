/**
 * Jina AI Engine adapters.
 *
 * JinaSearchEngine - Web search via s.jina.ai (search API)
 * JinaReaderEngine - Page reading/extraction via r.jina.ai (reader API)
 *
 * Both require a JINA_API_KEY for authentication.
 */

import type {
  EngineParams,
  RequestConfig,
  EngineResults,
  Category,
} from './engine';
import { BaseEngine } from './base';

// ========== Jina Search API Types ==========

interface JinaSearchResult {
  title: string;
  url: string;
  content: string;
  description: string;
}

interface JinaSearchResponse {
  code?: number;
  status?: number;
  data?: JinaSearchResult[];
}

// ========== Jina Reader Types ==========

export interface JinaReaderResult {
  title: string;
  content: string;
  url: string;
  description: string;
  images?: string[];
}

interface JinaReaderResponse {
  code?: number;
  status?: number;
  data?: {
    title?: string;
    content?: string;
    url?: string;
    description?: string;
    images?: string[];
  };
}

// ========== Jina Search Engine ==========

/**
 * Jina Search Engine adapter.
 * Uses the Jina AI search API (s.jina.ai) for high-quality web search results.
 *
 * API key must be provided via params.engineData['jina_api_key'].
 */
export class JinaSearchEngine extends BaseEngine {
  constructor() {
    super({
      name: 'jina',
      shortcut: 'ji',
      categories: ['general'] as Category[],
      supportsPaging: false,
      maxPage: 1,
      timeout: 15_000,
      weight: 1.5,
    });
  }

  buildRequest(query: string, params: EngineParams): RequestConfig {
    const apiKey = params.engineData['jina_api_key'] ?? '';

    const url = `https://s.jina.ai/${encodeURIComponent(query)}`;

    const headers: Record<string, string> = {
      'Accept': 'application/json',
      'X-No-Cache': 'true',
    };

    if (apiKey) {
      headers['Authorization'] = `Bearer ${apiKey}`;
    }

    return this.createJsonRequest({
      url,
      method: 'GET',
      headers,
    });
  }

  parseResponse(body: string, _params: EngineParams): EngineResults {
    const results = this.createResults();

    try {
      const data = JSON.parse(body) as JinaSearchResponse;

      if (!data.data || !Array.isArray(data.data)) {
        return results;
      }

      for (const item of data.data) {
        if (!item.url || !item.title) continue;

        const content = item.content || item.description || '';

        this.addResult(results, {
          url: item.url,
          title: item.title,
          content: this.truncate(this.cleanText(content), 500),
        });
      }
    } catch {
      // JSON parse failed
    }

    return results;
  }
}

// ========== Jina Reader Engine ==========

/**
 * Jina Reader Engine.
 * Uses the Jina AI reader API (r.jina.ai) to extract clean content from web pages.
 *
 * This is NOT an OnlineEngine (not used in metasearch). It is a standalone
 * service class for reading/extracting page content as markdown.
 */
export class JinaReaderEngine {
  /**
   * Read and extract content from a web page using Jina Reader.
   *
   * @param url - The URL of the page to read
   * @param apiKey - Jina API key for authentication
   * @returns Extracted page content as a JinaReaderResult
   */
  async readPage(url: string, apiKey: string): Promise<JinaReaderResult> {
    const readerUrl = `https://r.jina.ai/${url}`;

    const headers: Record<string, string> = {
      'Accept': 'application/json',
      'X-Return-Format': 'markdown',
    };

    if (apiKey) {
      headers['Authorization'] = `Bearer ${apiKey}`;
    }

    const response = await fetch(readerUrl, {
      method: 'GET',
      headers,
    });

    if (!response.ok) {
      throw new Error(
        `Jina Reader: HTTP ${response.status} ${response.statusText}`
      );
    }

    const body = await response.text();
    const data = JSON.parse(body) as JinaReaderResponse;

    if (!data.data) {
      throw new Error('Jina Reader: No data in response');
    }

    return {
      title: data.data.title ?? '',
      content: data.data.content ?? '',
      url: data.data.url ?? url,
      description: data.data.description ?? '',
      images: data.data.images,
    };
  }
}
