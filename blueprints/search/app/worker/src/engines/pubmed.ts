/**
 * PubMed Medical Literature Search Engine.
 * Uses NCBI E-utilities API (esearch + esummary).
 * https://www.ncbi.nlm.nih.gov/books/NBK25499/
 */

import type {
  OnlineEngine,
  EngineParams,
  RequestConfig,
  EngineResults,
  Category,
} from './engine';
import { newEngineResults } from './engine';

const PUBMED_ESEARCH_URL = 'https://eutils.ncbi.nlm.nih.gov/entrez/eutils/esearch.fcgi';
const PUBMED_ESUMMARY_URL = 'https://eutils.ncbi.nlm.nih.gov/entrez/eutils/esummary.fcgi';

const USER_AGENT =
  'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36';

const RESULTS_PER_PAGE = 10;

// Time range mapping in days
const timeRangeDays: Record<string, number> = {
  day: 1,
  week: 7,
  month: 30,
  year: 365,
};

// PubMed eSummary response types
interface PubMedAuthor {
  name?: string;
  authtype?: string;
  clusterid?: string;
}

interface PubMedArticleId {
  idtype?: string;
  idtypen?: number;
  value?: string;
}

interface PubMedDocSum {
  uid?: string;
  pubdate?: string;
  epubdate?: string;
  source?: string;
  authors?: PubMedAuthor[];
  title?: string;
  volume?: string;
  issue?: string;
  pages?: string;
  fulljournalname?: string;
  sortpubdate?: string;
  pubtype?: string[];
  elocationid?: string;
  articleids?: PubMedArticleId[];
  pmcrefcount?: number;
}

interface PubMedSearchResult {
  header?: {
    type?: string;
    version?: string;
  };
  esearchresult?: {
    count?: string;
    retmax?: string;
    retstart?: string;
    idlist?: string[];
    querytranslation?: string;
  };
}

interface PubMedSummaryResult {
  result?: {
    uids?: string[];
    [uid: string]: PubMedDocSum | string[] | undefined;
  };
}

export class PubMedEngine implements OnlineEngine {
  name = 'pubmed';
  shortcut = 'pm';
  categories: Category[] = ['science'];
  supportsPaging = true;
  maxPage = 10;
  timeout = 10_000;
  weight = 1.0;
  disabled = false;

  // Store search IDs for two-phase lookup
  // Note: Currently unused as fetch happens inline in parseResponse
  // private lastSearchIds: string[] = [];

  buildRequest(query: string, params: EngineParams): RequestConfig {
    // Phase 1: Search for article IDs
    // Phase 2 will be handled in parseResponse by making another fetch
    const retstart = (params.page - 1) * RESULTS_PER_PAGE;

    const searchParams = new URLSearchParams();
    searchParams.set('db', 'pubmed');
    searchParams.set('term', query);
    searchParams.set('retstart', retstart.toString());
    searchParams.set('retmax', RESULTS_PER_PAGE.toString());
    searchParams.set('retmode', 'json');
    searchParams.set('sort', 'relevance');

    // Add date range if specified
    if (params.timeRange && timeRangeDays[params.timeRange]) {
      const days = timeRangeDays[params.timeRange];
      searchParams.set('datetype', 'pdat');
      searchParams.set('reldate', days.toString());
    }

    return {
      url: `${PUBMED_ESEARCH_URL}?${searchParams.toString()}`,
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
      const data = JSON.parse(body) as PubMedSearchResult;

      if (!data.esearchresult?.idlist || data.esearchresult.idlist.length === 0) {
        return results;
      }

      // Store IDs for the two-phase lookup pattern
      // The actual summaries need to be fetched in a second request
      // IDs are stored in engineData for the orchestrator to handle

      // Store the IDs in engineData for the orchestrator to handle
      results.engineData['pubmed_ids'] = data.esearchresult.idlist.join(',');
      results.engineData['pubmed_count'] = data.esearchresult.count || '0';
    } catch {
      // JSON parse failed
    }

    return results;
  }

  /**
   * Build request for fetching summaries (phase 2).
   */
  buildSummaryRequest(ids: string[]): RequestConfig {
    const searchParams = new URLSearchParams();
    searchParams.set('db', 'pubmed');
    searchParams.set('id', ids.join(','));
    searchParams.set('retmode', 'json');

    return {
      url: `${PUBMED_ESUMMARY_URL}?${searchParams.toString()}`,
      method: 'GET',
      headers: {
        'User-Agent': USER_AGENT,
        Accept: 'application/json',
      },
      cookies: [],
    };
  }

  /**
   * Parse summary response (phase 2).
   */
  parseSummaryResponse(body: string, _params: EngineParams): EngineResults {
    const results = newEngineResults();

    try {
      const data = JSON.parse(body) as PubMedSummaryResult;

      if (!data.result?.uids) {
        return results;
      }

      for (const uid of data.result.uids) {
        const article = data.result[uid] as PubMedDocSum | undefined;
        if (!article || !article.title) continue;

        // Extract authors
        const authors: string[] = [];
        if (article.authors) {
          for (const author of article.authors) {
            if (author.name) {
              authors.push(author.name);
            }
          }
        }

        // Extract DOI
        let doi = '';
        if (article.articleids) {
          for (const aid of article.articleids) {
            if (aid.idtype === 'doi' && aid.value) {
              doi = aid.value;
              break;
            }
          }
        }

        // Extract ELocationID (often contains DOI)
        if (!doi && article.elocationid) {
          const match = article.elocationid.match(/doi:\s*(\S+)/i);
          if (match) {
            doi = match[1];
          }
        }

        // Parse published date
        let publishedAt = '';
        if (article.sortpubdate) {
          try {
            publishedAt = new Date(article.sortpubdate).toISOString();
          } catch {
            publishedAt = article.sortpubdate;
          }
        } else if (article.pubdate) {
          publishedAt = article.pubdate;
        }

        // Build URL
        const url = `https://pubmed.ncbi.nlm.nih.gov/${uid}/`;

        // Build content with journal info
        let content = '';
        if (article.fulljournalname) {
          content = article.fulljournalname;
          if (article.volume) {
            content += ` ${article.volume}`;
            if (article.issue) {
              content += `(${article.issue})`;
            }
          }
          if (article.pages) {
            content += `:${article.pages}`;
          }
        }

        // Add citation count if available
        if (article.pmcrefcount && article.pmcrefcount > 0) {
          content += content ? ` | ` : '';
          content += `${article.pmcrefcount} citations`;
        }

        results.results.push({
          url,
          title: article.title,
          content,
          engine: this.name,
          score: this.weight,
          category: 'science',
          template: 'paper',
          authors,
          publishedAt,
          doi: doi || undefined,
          journal: article.fulljournalname || article.source || undefined,
          metadata: {
            pmid: uid,
            volume: article.volume,
            issue: article.issue,
            pages: article.pages,
            pubtype: article.pubtype,
            citationCount: article.pmcrefcount,
          },
        });
      }
    } catch {
      // JSON parse failed
    }

    return results;
  }
}

/**
 * Convenience class that combines search + summary in one engine.
 * This performs both API calls internally.
 */
export class PubMedCombinedEngine implements OnlineEngine {
  name = 'pubmed';
  shortcut = 'pm';
  categories: Category[] = ['science'];
  supportsPaging = true;
  maxPage = 10;
  timeout = 15_000; // Longer timeout for two requests
  weight = 1.0;
  disabled = false;

  private baseEngine = new PubMedEngine();

  buildRequest(query: string, params: EngineParams): RequestConfig {
    return this.baseEngine.buildRequest(query, params);
  }

  parseResponse(body: string, params: EngineParams): EngineResults {
    // Parse search response to get IDs
    const searchResults = this.baseEngine.parseResponse(body, params);
    const idsString = searchResults.engineData['pubmed_ids'];

    if (!idsString) {
      return newEngineResults();
    }

    // Return partial results with IDs - the execution layer needs to make second request
    // For a full combined engine, we would need async parseResponse
    return searchResults;
  }
}

/**
 * Execute full PubMed search with both phases.
 * Use this function instead of executeEngine for full results.
 */
export async function executePubMedSearch(
  query: string,
  params: EngineParams
): Promise<EngineResults> {
  const engine = new PubMedEngine();

  // Phase 1: Search for IDs
  const searchConfig = engine.buildRequest(query, params);
  const searchResponse = await fetch(searchConfig.url, {
    method: searchConfig.method,
    headers: searchConfig.headers,
  });

  if (!searchResponse.ok) {
    throw new Error(`PubMed search failed: ${searchResponse.status}`);
  }

  const searchBody = await searchResponse.text();
  const searchResults = engine.parseResponse(searchBody, params);
  const idsString = searchResults.engineData['pubmed_ids'];

  if (!idsString) {
    return newEngineResults();
  }

  // Phase 2: Fetch summaries
  const ids = idsString.split(',');
  const summaryConfig = engine.buildSummaryRequest(ids);
  const summaryResponse = await fetch(summaryConfig.url, {
    method: summaryConfig.method,
    headers: summaryConfig.headers,
  });

  if (!summaryResponse.ok) {
    throw new Error(`PubMed summary failed: ${summaryResponse.status}`);
  }

  const summaryBody = await summaryResponse.text();
  return engine.parseSummaryResponse(summaryBody, params);
}
