/**
 * Open Library Book Search Engine.
 * Uses the Open Library Search API.
 * https://openlibrary.org/dev/docs/api/search
 */

import type {
  OnlineEngine,
  EngineParams,
  RequestConfig,
  EngineResults,
  Category,
} from './engine';
import { newEngineResults } from './engine';

const OPENLIBRARY_SEARCH_URL = 'https://openlibrary.org/search.json';

const USER_AGENT =
  'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36';

const RESULTS_PER_PAGE = 10;

// Open Library API response types
interface OLBook {
  key?: string;
  title?: string;
  subtitle?: string;
  author_name?: string[];
  author_key?: string[];
  first_publish_year?: number;
  publish_year?: number[];
  publish_date?: string[];
  publisher?: string[];
  isbn?: string[];
  oclc?: string[];
  lccn?: string[];
  number_of_pages_median?: number;
  language?: string[];
  subject?: string[];
  place?: string[];
  person?: string[];
  time?: string[];
  edition_count?: number;
  ebook_access?: string;
  ebook_count_i?: number;
  has_fulltext?: boolean;
  ia?: string[];
  cover_i?: number;
  cover_edition_key?: string;
  first_sentence?: string[];
  ratings_average?: number;
  ratings_count?: number;
  want_to_read_count?: number;
  currently_reading_count?: number;
  already_read_count?: number;
  type?: string;
  seed?: string[];
}

interface OLSearchResponse {
  numFound?: number;
  start?: number;
  numFoundExact?: boolean;
  docs?: OLBook[];
  num_found?: number;
  q?: string;
  offset?: number;
}

export class OpenLibraryEngine implements OnlineEngine {
  name = 'open library';
  shortcut = 'ol';
  categories: Category[] = ['science'];
  supportsPaging = true;
  maxPage = 10;
  timeout = 10_000;
  weight = 0.95;
  disabled = false;

  buildRequest(query: string, params: EngineParams): RequestConfig {
    const offset = (params.page - 1) * RESULTS_PER_PAGE;

    const searchParams = new URLSearchParams();
    searchParams.set('q', query);
    searchParams.set('offset', offset.toString());
    searchParams.set('limit', RESULTS_PER_PAGE.toString());
    searchParams.set('mode', 'everything');

    // Request additional fields
    const fields = [
      'key',
      'title',
      'subtitle',
      'author_name',
      'author_key',
      'first_publish_year',
      'publish_year',
      'publisher',
      'isbn',
      'oclc',
      'lccn',
      'number_of_pages_median',
      'language',
      'subject',
      'edition_count',
      'ebook_access',
      'ebook_count_i',
      'has_fulltext',
      'ia',
      'cover_i',
      'cover_edition_key',
      'first_sentence',
      'ratings_average',
      'ratings_count',
    ].join(',');
    searchParams.set('fields', fields);

    // Add year filter if time range is specified
    if (params.timeRange === 'year') {
      const currentYear = new Date().getFullYear();
      searchParams.set('publish_year', `[${currentYear - 1} TO ${currentYear}]`);
    }

    return {
      url: `${OPENLIBRARY_SEARCH_URL}?${searchParams.toString()}`,
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
      const data = JSON.parse(body) as OLSearchResponse;

      if (!data.docs || !Array.isArray(data.docs)) {
        return results;
      }

      for (const book of data.docs) {
        if (!book.title || !book.key) continue;

        // Extract authors
        const authors: string[] = book.author_name || [];

        // Build URL
        const url = `https://openlibrary.org${book.key}`;

        // Build title (include subtitle if present)
        let title = book.title;
        if (book.subtitle) {
          title += `: ${book.subtitle}`;
        }

        // Get published year
        let publishedAt = '';
        if (book.first_publish_year) {
          publishedAt = `${book.first_publish_year}`;
        } else if (book.publish_year && book.publish_year.length > 0) {
          publishedAt = `${Math.min(...book.publish_year)}`;
        }

        // Build content
        const contentParts: string[] = [];

        // Add first sentence as description if available
        if (book.first_sentence && book.first_sentence.length > 0) {
          let sentence = book.first_sentence[0];
          if (sentence.length > 200) {
            sentence = sentence.slice(0, 197) + '...';
          }
          contentParts.push(sentence);
        }

        // Add publisher
        if (book.publisher && book.publisher.length > 0) {
          contentParts.push(`Publisher: ${book.publisher[0]}`);
        }

        // Add edition info
        if (book.edition_count && book.edition_count > 1) {
          contentParts.push(`${book.edition_count} editions`);
        }

        // Add page count
        if (book.number_of_pages_median) {
          contentParts.push(`${book.number_of_pages_median} pages`);
        }

        // Add ebook availability
        if (book.ebook_access === 'borrowable' || book.ebook_access === 'public') {
          contentParts.push('Available to read');
        }

        // Add rating if available
        if (book.ratings_average && book.ratings_count) {
          contentParts.push(
            `${book.ratings_average.toFixed(1)}/5 (${book.ratings_count} ratings)`
          );
        }

        const content = contentParts.join(' | ');

        // Build thumbnail URL
        let thumbnailUrl: string | undefined;
        if (book.cover_i) {
          thumbnailUrl = `https://covers.openlibrary.org/b/id/${book.cover_i}-M.jpg`;
        } else if (book.cover_edition_key) {
          thumbnailUrl = `https://covers.openlibrary.org/b/olid/${book.cover_edition_key}-M.jpg`;
        }

        // Get ISBN (prefer ISBN-13)
        let isbn: string | undefined;
        if (book.isbn && book.isbn.length > 0) {
          // Prefer ISBN-13 (starts with 978 or 979)
          isbn = book.isbn.find((i) => i.startsWith('978') || i.startsWith('979'));
          if (!isbn) {
            isbn = book.isbn[0];
          }
        }

        results.results.push({
          url,
          title,
          content,
          engine: this.name,
          score: this.weight,
          category: 'science',
          template: 'book',
          authors,
          publishedAt,
          thumbnailUrl,
          metadata: {
            workKey: book.key,
            isbn,
            oclc: book.oclc?.[0],
            lccn: book.lccn?.[0],
            publisher: book.publisher?.[0],
            editionCount: book.edition_count,
            pageCount: book.number_of_pages_median,
            language: book.language,
            subjects: book.subject?.slice(0, 10),
            ebookAccess: book.ebook_access,
            hasFulltext: book.has_fulltext,
            internetArchiveIds: book.ia,
            ratingsAverage: book.ratings_average,
            ratingsCount: book.ratings_count,
          },
        });
      }
    } catch {
      // JSON parse failed
    }

    return results;
  }
}
