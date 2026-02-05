/**
 * arXiv Search Engine adapter.
 * Ported from Go: pkg/engine/local/engines/arxiv.go
 *
 * Uses export.arxiv.org/api/query XML (ATOM) endpoint.
 */

import type {
  OnlineEngine,
  EngineParams,
  RequestConfig,
  EngineResults,
  Category,
} from './engine';
import { newEngineResults } from './engine';
import {
  getElementsByTagName,
  getTextContent,
  getElementAttribute,
} from '../lib/xml-parser';

export class ArxivEngine implements OnlineEngine {
  name = 'arxiv';
  shortcut = 'arx';
  categories: Category[] = ['science'];
  supportsPaging = true;
  maxPage = 10;
  timeout = 5_000;
  weight = 1.0;
  disabled = false;

  buildRequest(query: string, params: EngineParams): RequestConfig {
    const maxResults = 10;
    let start = 0;
    if (params.page > 1) {
      start = (params.page - 1) * maxResults;
    }

    const searchParams = new URLSearchParams();
    searchParams.set('search_query', `all:${query}`);
    searchParams.set('start', start.toString());
    searchParams.set('max_results', maxResults.toString());

    return {
      url: `https://export.arxiv.org/api/query?${searchParams.toString()}`,
      method: 'GET',
      headers: {
        Accept: 'application/atom+xml',
      },
      cookies: [],
    };
  }

  parseResponse(body: string, _params: EngineParams): EngineResults {
    const results = newEngineResults();

    // Parse ATOM XML entries
    const entries = getElementsByTagName(body, 'entry');

    for (const entry of entries) {
      // Extract ID (URL)
      const id = getTextContent(entry, 'id').trim();
      if (!id) continue;

      // Extract title
      const title = getTextContent(entry, 'title')
        .replace(/\s+/g, ' ')
        .trim();

      // Extract summary
      const summary = getTextContent(entry, 'summary')
        .replace(/\s+/g, ' ')
        .trim();

      // Extract authors
      const authorNames = getElementsByTagName(entry, 'author');
      const authors: string[] = [];
      for (const authorXml of authorNames) {
        const name = getTextContent(authorXml, 'name').trim();
        if (name) {
          authors.push(name);
        }
      }

      // Extract published date
      const published = getTextContent(entry, 'published').trim();
      let publishedAt = '';
      if (published) {
        try {
          publishedAt = new Date(published).toISOString();
        } catch {
          publishedAt = published;
        }
      }

      // Find PDF link
      const linkHrefs = getElementAttribute(entry, 'link', 'href');
      const linkTitles = getElementAttribute(entry, 'link', 'title');
      let pdfUrl = '';
      for (let i = 0; i < linkTitles.length; i++) {
        if (linkTitles[i] === 'pdf' && linkHrefs[i]) {
          pdfUrl = linkHrefs[i];
          break;
        }
      }

      // Extract DOI
      const doi = getTextContent(entry, 'doi').trim();

      // Extract journal reference
      const journal = getTextContent(entry, 'journal_ref').trim();

      // Build content with PDF link appended
      let content = summary;
      if (pdfUrl) {
        content += ` [PDF: ${pdfUrl}]`;
      }

      results.results.push({
        url: id,
        title,
        content,
        engine: this.name,
        score: this.weight,
        category: 'science',
        template: 'paper',
        authors,
        publishedAt,
        doi: doi || undefined,
        journal: journal || undefined,
      });
    }

    return results;
  }
}
