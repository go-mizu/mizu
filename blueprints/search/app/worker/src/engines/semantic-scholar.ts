/**
 * Semantic Scholar Academic Search Engine.
 * Uses the Semantic Scholar Academic Graph API.
 * https://api.semanticscholar.org/api-docs/
 */

import type {
  OnlineEngine,
  EngineParams,
  RequestConfig,
  EngineResults,
  Category,
} from './engine';
import { newEngineResults } from './engine';

const SEMANTIC_SCHOLAR_API_URL = 'https://api.semanticscholar.org/graph/v1/paper/search';

const USER_AGENT =
  'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36';

const RESULTS_PER_PAGE = 10;

// Time range mapping for year filter
const timeRangeYears: Record<string, number> = {
  day: 0, // Not really applicable for academic papers
  week: 0,
  month: 0,
  year: 1,
};

// Semantic Scholar API response types
interface S2Author {
  authorId?: string;
  name?: string;
}

interface S2Paper {
  paperId?: string;
  externalIds?: {
    DOI?: string;
    ArXiv?: string;
    PubMed?: string;
    MAG?: string;
    CorpusId?: number;
  };
  url?: string;
  title?: string;
  abstract?: string;
  venue?: string;
  publicationVenue?: {
    id?: string;
    name?: string;
    type?: string;
    alternate_names?: string[];
    issn?: string;
    url?: string;
  };
  year?: number;
  referenceCount?: number;
  citationCount?: number;
  influentialCitationCount?: number;
  isOpenAccess?: boolean;
  openAccessPdf?: {
    url?: string;
    status?: string;
  };
  fieldsOfStudy?: string[];
  s2FieldsOfStudy?: Array<{
    category?: string;
    source?: string;
  }>;
  authors?: S2Author[];
  publicationDate?: string;
  journal?: {
    name?: string;
    volume?: string;
    pages?: string;
  };
}

interface S2SearchResponse {
  total?: number;
  offset?: number;
  next?: number;
  data?: S2Paper[];
}

export class SemanticScholarEngine implements OnlineEngine {
  name = 'semantic scholar';
  shortcut = 's2';
  categories: Category[] = ['science'];
  supportsPaging = true;
  maxPage = 10;
  timeout = 10_000;
  weight = 1.0;
  disabled = false;

  buildRequest(query: string, params: EngineParams): RequestConfig {
    const offset = (params.page - 1) * RESULTS_PER_PAGE;

    const searchParams = new URLSearchParams();
    searchParams.set('query', query);
    searchParams.set('offset', offset.toString());
    searchParams.set('limit', RESULTS_PER_PAGE.toString());

    // Request fields we need
    const fields = [
      'paperId',
      'externalIds',
      'url',
      'title',
      'abstract',
      'venue',
      'publicationVenue',
      'year',
      'citationCount',
      'influentialCitationCount',
      'isOpenAccess',
      'openAccessPdf',
      'fieldsOfStudy',
      'authors',
      'publicationDate',
      'journal',
    ].join(',');
    searchParams.set('fields', fields);

    // Add year filter if time range is specified
    if (params.timeRange === 'year') {
      const currentYear = new Date().getFullYear();
      searchParams.set('year', `${currentYear - 1}-${currentYear}`);
    }

    return {
      url: `${SEMANTIC_SCHOLAR_API_URL}?${searchParams.toString()}`,
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
      const data = JSON.parse(body) as S2SearchResponse;

      if (!data.data || !Array.isArray(data.data)) {
        return results;
      }

      for (const paper of data.data) {
        if (!paper.title) continue;

        // Extract authors
        const authors: string[] = [];
        if (paper.authors) {
          for (const author of paper.authors) {
            if (author.name) {
              authors.push(author.name);
            }
          }
        }

        // Extract DOI
        const doi = paper.externalIds?.DOI || undefined;

        // Build URL (prefer Semantic Scholar URL)
        let url = paper.url || `https://www.semanticscholar.org/paper/${paper.paperId}`;

        // Parse published date
        let publishedAt = '';
        if (paper.publicationDate) {
          try {
            publishedAt = new Date(paper.publicationDate).toISOString();
          } catch {
            publishedAt = paper.publicationDate;
          }
        } else if (paper.year) {
          publishedAt = `${paper.year}`;
        }

        // Build content with abstract snippet
        let content = '';
        if (paper.abstract) {
          content = paper.abstract;
          if (content.length > 300) {
            content = content.slice(0, 297) + '...';
          }
        }

        // Get journal name
        const journal = paper.journal?.name ||
          paper.publicationVenue?.name ||
          paper.venue ||
          undefined;

        // Add citation info to content
        const citationInfo: string[] = [];
        if (paper.citationCount !== undefined && paper.citationCount > 0) {
          citationInfo.push(`${paper.citationCount} citations`);
        }
        if (paper.influentialCitationCount !== undefined && paper.influentialCitationCount > 0) {
          citationInfo.push(`${paper.influentialCitationCount} influential`);
        }
        if (paper.isOpenAccess) {
          citationInfo.push('Open Access');
        }

        if (citationInfo.length > 0) {
          content = content
            ? `${content} | ${citationInfo.join(' | ')}`
            : citationInfo.join(' | ');
        }

        // Add PDF link if available
        if (paper.openAccessPdf?.url) {
          content += ` [PDF: ${paper.openAccessPdf.url}]`;
        }

        results.results.push({
          url,
          title: paper.title,
          content,
          engine: this.name,
          score: this.weight,
          category: 'science',
          template: 'paper',
          authors,
          publishedAt,
          doi,
          journal,
          metadata: {
            paperId: paper.paperId,
            year: paper.year,
            citationCount: paper.citationCount,
            influentialCitationCount: paper.influentialCitationCount,
            isOpenAccess: paper.isOpenAccess,
            openAccessPdfUrl: paper.openAccessPdf?.url,
            fieldsOfStudy: paper.fieldsOfStudy,
            arxivId: paper.externalIds?.ArXiv,
            pubmedId: paper.externalIds?.PubMed,
          },
        });
      }
    } catch {
      // JSON parse failed
    }

    return results;
  }
}
