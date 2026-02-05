/**
 * Stack Overflow Search Engine adapter.
 * Uses Stack Exchange API to search for questions.
 * https://api.stackexchange.com/docs/search
 */

import type {
  OnlineEngine,
  EngineParams,
  RequestConfig,
  EngineResults,
  Category,
} from './engine';
import { newEngineResults } from './engine';
import { decodeHtmlEntities } from '../lib/html-parser';

// ========== Stack Exchange API Types ==========

interface StackOverflowQuestion {
  question_id: number;
  title: string;
  link: string;
  body_markdown?: string;
  tags?: string[];
  score: number;
  answer_count: number;
  view_count: number;
  is_answered: boolean;
  accepted_answer_id?: number;
  creation_date: number;
  last_activity_date: number;
  owner?: {
    display_name?: string;
    link?: string;
    reputation?: number;
    profile_image?: string;
  };
}

interface StackExchangeResponse {
  items?: StackOverflowQuestion[];
  has_more?: boolean;
  quota_max?: number;
  quota_remaining?: number;
  error_id?: number;
  error_message?: string;
}

const SO_USER_AGENT =
  'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36';

// Time range mapping (seconds)
const timeRangeSeconds: Record<string, number> = {
  day: 86400,
  week: 604800,
  month: 2592000,
  year: 31536000,
};

export class StackOverflowEngine implements OnlineEngine {
  name = 'stack overflow';
  shortcut = 'so';
  categories: Category[] = ['it'];
  supportsPaging = true;
  maxPage = 10;
  timeout = 10_000;
  weight = 0.95;
  disabled = false;

  private site: string;
  private resultsPerPage = 20;

  constructor(options?: { site?: string }) {
    this.site = options?.site || 'stackoverflow';
    if (this.site !== 'stackoverflow') {
      this.name = `stack exchange (${this.site})`;
    }
  }

  buildRequest(query: string, params: EngineParams): RequestConfig {
    const searchParams = new URLSearchParams();
    searchParams.set('order', 'desc');
    searchParams.set('sort', 'relevance');
    searchParams.set('intitle', query);
    searchParams.set('site', this.site);
    searchParams.set('pagesize', String(this.resultsPerPage));
    searchParams.set('page', String(params.page));
    searchParams.set('filter', 'withbody'); // Include body_markdown

    // Apply time range filter
    if (params.timeRange && timeRangeSeconds[params.timeRange]) {
      const cutoff = Math.floor(Date.now() / 1000) - timeRangeSeconds[params.timeRange];
      searchParams.set('fromdate', String(cutoff));
    }

    return {
      url: `https://api.stackexchange.com/2.3/search/advanced?${searchParams.toString()}`,
      method: 'GET',
      headers: {
        'User-Agent': SO_USER_AGENT,
        Accept: 'application/json',
        'Accept-Encoding': 'gzip', // Stack Exchange API requires compression
      },
      cookies: [],
    };
  }

  parseResponse(body: string, _params: EngineParams): EngineResults {
    const results = newEngineResults();

    try {
      const data = JSON.parse(body) as StackExchangeResponse;

      if (!data.items || !Array.isArray(data.items)) return results;

      for (const question of data.items) {
        if (!question.link || !question.title) continue;

        // Build content with snippet and metadata
        let content = '';
        if (question.body_markdown) {
          // Get first 200 chars of body as snippet
          content = question.body_markdown
            .replace(/```[\s\S]*?```/g, '[code]') // Replace code blocks
            .replace(/`[^`]+`/g, '[code]') // Replace inline code
            .replace(/\[([^\]]+)\]\([^)]+\)/g, '$1') // Replace markdown links
            .replace(/[#*_~]/g, '') // Remove markdown formatting
            .slice(0, 200)
            .trim();
          if (question.body_markdown.length > 200) {
            content += '...';
          }
        }

        const meta: string[] = [];
        meta.push(`${question.score} votes`);
        meta.push(`${question.answer_count} answers`);
        if (question.is_answered) {
          meta.push('answered');
        }
        if (question.tags && question.tags.length > 0) {
          meta.push(question.tags.slice(0, 5).join(', '));
        }

        if (meta.length > 0) {
          content = content
            ? `${content} | ${meta.join(' | ')}`
            : meta.join(' | ');
        }

        const publishedAt = new Date(question.creation_date * 1000).toISOString();

        results.results.push({
          url: question.link,
          title: decodeHtmlEntities(question.title),
          content,
          engine: this.name,
          score: this.weight,
          category: 'it',
          template: 'qa',
          thumbnailUrl: question.owner?.profile_image || '',
          publishedAt,
          topics: question.tags || [],
          metadata: {
            questionId: question.question_id,
            votes: question.score,
            answers: question.answer_count,
            views: question.view_count,
            isAnswered: question.is_answered,
            hasAcceptedAnswer: !!question.accepted_answer_id,
            author: question.owner?.display_name,
            authorLink: question.owner?.link,
            authorReputation: question.owner?.reputation,
            lastActivity: new Date(question.last_activity_date * 1000).toISOString(),
          },
        });
      }
    } catch {
      // Parse error
    }

    return results;
  }
}

/**
 * Stack Exchange variant for other sites.
 */
export class StackExchangeEngine extends StackOverflowEngine {
  constructor(site: string) {
    super({ site });
  }
}

/**
 * Server Fault engine.
 */
export class ServerFaultEngine extends StackOverflowEngine {
  constructor() {
    super({ site: 'serverfault' });
    this.name = 'server fault';
    this.shortcut = 'sf';
  }
}

/**
 * Super User engine.
 */
export class SuperUserEngine extends StackOverflowEngine {
  constructor() {
    super({ site: 'superuser' });
    this.name = 'super user';
    this.shortcut = 'su';
  }
}

/**
 * Ask Ubuntu engine.
 */
export class AskUbuntuEngine extends StackOverflowEngine {
  constructor() {
    super({ site: 'askubuntu' });
    this.name = 'ask ubuntu';
    this.shortcut = 'au';
  }
}
