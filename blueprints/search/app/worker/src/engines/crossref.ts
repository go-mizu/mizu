/**
 * Crossref DOI Metadata Search Engine.
 * Uses the Crossref REST API.
 * https://api.crossref.org/swagger-ui/index.html
 */

import type {
  OnlineEngine,
  EngineParams,
  RequestConfig,
  EngineResults,
  Category,
} from './engine';
import { newEngineResults } from './engine';

const CROSSREF_API_URL = 'https://api.crossref.org/works';

const USER_AGENT =
  'MizuSearch/1.0 (https://github.com/mizu-search; mailto:search@example.com)';

const RESULTS_PER_PAGE = 10;

// Time range mapping for filter
const timeRangeDays: Record<string, number> = {
  day: 1,
  week: 7,
  month: 30,
  year: 365,
};

// Crossref API response types
interface CrossrefAuthor {
  given?: string;
  family?: string;
  name?: string;
  sequence?: string;
  affiliation?: Array<{ name?: string }>;
  ORCID?: string;
}

interface CrossrefWork {
  DOI?: string;
  URL?: string;
  title?: string[];
  'container-title'?: string[];
  author?: CrossrefAuthor[];
  issued?: {
    'date-parts'?: number[][];
  };
  published?: {
    'date-parts'?: number[][];
  };
  'published-print'?: {
    'date-parts'?: number[][];
  };
  'published-online'?: {
    'date-parts'?: number[][];
  };
  created?: {
    'date-time'?: string;
    timestamp?: number;
  };
  deposited?: {
    'date-time'?: string;
    timestamp?: number;
  };
  indexed?: {
    'date-time'?: string;
  };
  type?: string;
  publisher?: string;
  volume?: string;
  issue?: string;
  page?: string;
  'is-referenced-by-count'?: number;
  'references-count'?: number;
  subject?: string[];
  abstract?: string;
  license?: Array<{
    URL?: string;
    'content-version'?: string;
    'delay-in-days'?: number;
  }>;
  link?: Array<{
    URL?: string;
    'content-type'?: string;
    'content-version'?: string;
  }>;
  ISSN?: string[];
  ISBN?: string[];
  'short-container-title'?: string[];
  score?: number;
}

interface CrossrefResponse {
  status?: string;
  'message-type'?: string;
  'message-version'?: string;
  message?: {
    'total-results'?: number;
    items?: CrossrefWork[];
    'items-per-page'?: number;
    query?: {
      'start-index'?: number;
      'search-terms'?: string;
    };
  };
}

export class CrossrefEngine implements OnlineEngine {
  name = 'crossref';
  shortcut = 'cr';
  categories: Category[] = ['science'];
  supportsPaging = true;
  maxPage = 10;
  timeout = 15_000; // Crossref can be slow
  weight = 1.0;
  disabled = false;

  buildRequest(query: string, params: EngineParams): RequestConfig {
    const offset = (params.page - 1) * RESULTS_PER_PAGE;

    const searchParams = new URLSearchParams();
    searchParams.set('query', query);
    searchParams.set('offset', offset.toString());
    searchParams.set('rows', RESULTS_PER_PAGE.toString());
    searchParams.set('sort', 'relevance');

    // Add date filter if specified
    if (params.timeRange && timeRangeDays[params.timeRange]) {
      const days = timeRangeDays[params.timeRange];
      const now = new Date();
      const from = new Date(now.getTime() - days * 24 * 60 * 60 * 1000);
      const fromStr = from.toISOString().split('T')[0];
      searchParams.set('filter', `from-pub-date:${fromStr}`);
    }

    return {
      url: `${CROSSREF_API_URL}?${searchParams.toString()}`,
      method: 'GET',
      headers: {
        'User-Agent': USER_AGENT,
        Accept: 'application/json',
      },
      cookies: [],
    };
  }

  parseResponse(body: string, _params: EngineParams): EngineResults {
    const results = newEngineResults();

    try {
      const data = JSON.parse(body) as CrossrefResponse;

      if (!data.message?.items || !Array.isArray(data.message.items)) {
        return results;
      }

      for (const work of data.message.items) {
        // Get title
        const title = work.title?.[0];
        if (!title) continue;

        // Extract authors
        const authors: string[] = [];
        if (work.author) {
          for (const author of work.author) {
            if (author.name) {
              authors.push(author.name);
            } else if (author.family) {
              const name = author.given
                ? `${author.given} ${author.family}`
                : author.family;
              authors.push(name);
            }
          }
        }

        // Build URL (prefer DOI URL)
        const doi = work.DOI;
        const url = work.URL || (doi ? `https://doi.org/${doi}` : '');
        if (!url) continue;

        // Get journal name
        const journal = work['container-title']?.[0] || undefined;

        // Parse published date
        let publishedAt = '';
        const dateSource =
          work.published ||
          work['published-print'] ||
          work['published-online'] ||
          work.issued;

        if (dateSource?.['date-parts']?.[0]) {
          const dateParts = dateSource['date-parts'][0];
          if (dateParts.length >= 1) {
            const year = dateParts[0];
            const month = dateParts[1] || 1;
            const day = dateParts[2] || 1;
            try {
              publishedAt = new Date(year, month - 1, day).toISOString();
            } catch {
              publishedAt = `${year}`;
            }
          }
        }

        // Build content
        let content = '';

        // Add abstract if available
        if (work.abstract) {
          // Strip HTML tags from abstract
          content = work.abstract.replace(/<[^>]+>/g, '').trim();
          if (content.length > 300) {
            content = content.slice(0, 297) + '...';
          }
        }

        // Add type and citation info
        const info: string[] = [];
        if (work.type) {
          info.push(work.type.replace('-', ' '));
        }
        if (work['is-referenced-by-count'] !== undefined && work['is-referenced-by-count'] > 0) {
          info.push(`${work['is-referenced-by-count']} citations`);
        }
        if (work.publisher) {
          info.push(work.publisher);
        }

        if (info.length > 0) {
          content = content
            ? `${content} | ${info.join(' | ')}`
            : info.join(' | ');
        }

        // Add PDF link if available
        if (work.link) {
          const pdfLink = work.link.find(
            (l) => l['content-type'] === 'application/pdf'
          );
          if (pdfLink?.URL) {
            content += ` [PDF: ${pdfLink.URL}]`;
          }
        }

        results.results.push({
          url,
          title,
          content,
          engine: this.name,
          score: this.weight,
          category: 'science',
          template: 'paper',
          authors,
          publishedAt,
          doi: doi || undefined,
          journal,
          metadata: {
            type: work.type,
            publisher: work.publisher,
            volume: work.volume,
            issue: work.issue,
            page: work.page,
            citationCount: work['is-referenced-by-count'],
            referencesCount: work['references-count'],
            subjects: work.subject,
            issn: work.ISSN,
            isbn: work.ISBN,
            license: work.license?.[0]?.URL,
          },
        });
      }
    } catch {
      // JSON parse failed
    }

    return results;
  }
}
