import type { Bang } from '../types';
import type { KVStore } from '../store/kv';

export interface BangParseResult {
  redirect?: string;
  bang?: { name: string; trigger: string };
  query: string;
  category?: string;
}

const BUILTIN_BANGS: Omit<Bang, 'id' | 'created_at'>[] = [
  { trigger: 'g', name: 'Google', url_template: 'https://www.google.com/search?q={query}', category: 'search', is_builtin: true },
  { trigger: 'ddg', name: 'DuckDuckGo', url_template: 'https://duckduckgo.com/?q={query}', category: 'search', is_builtin: true },
  { trigger: 'b', name: 'Bing', url_template: 'https://www.bing.com/search?q={query}', category: 'search', is_builtin: true },
  { trigger: 'yt', name: 'YouTube', url_template: 'https://www.youtube.com/results?search_query={query}', category: 'video', is_builtin: true },
  { trigger: 'w', name: 'Wikipedia', url_template: 'https://en.wikipedia.org/wiki/Special:Search?search={query}', category: 'reference', is_builtin: true },
  { trigger: 'r', name: 'Reddit', url_template: 'https://www.reddit.com/search/?q={query}', category: 'social', is_builtin: true },
  { trigger: 'gh', name: 'GitHub', url_template: 'https://github.com/search?q={query}', category: 'code', is_builtin: true },
  { trigger: 'so', name: 'Stack Overflow', url_template: 'https://stackoverflow.com/search?q={query}', category: 'code', is_builtin: true },
  { trigger: 'npm', name: 'npm', url_template: 'https://www.npmjs.com/search?q={query}', category: 'code', is_builtin: true },
  { trigger: 'amz', name: 'Amazon', url_template: 'https://www.amazon.com/s?k={query}', category: 'shopping', is_builtin: true },
  { trigger: 'imdb', name: 'IMDb', url_template: 'https://www.imdb.com/find?q={query}', category: 'media', is_builtin: true },
  { trigger: 'mdn', name: 'MDN', url_template: 'https://developer.mozilla.org/en-US/search?q={query}', category: 'code', is_builtin: true },
  { trigger: 'i', name: 'Images', url_template: '/images?q={query}', category: 'internal', is_builtin: true },
  { trigger: 'n', name: 'News', url_template: '/news?q={query}', category: 'internal', is_builtin: true },
  { trigger: 'v', name: 'Videos', url_template: '/videos?q={query}', category: 'internal', is_builtin: true },
];

// Build a lookup map from trigger to bang data for quick access
const BUILTIN_MAP = new Map<string, Omit<Bang, 'id' | 'created_at'>>();
for (const bang of BUILTIN_BANGS) {
  BUILTIN_MAP.set(bang.trigger, bang);
}

export class BangService {
  private kvStore: KVStore;

  constructor(kvStore: KVStore) {
    this.kvStore = kvStore;
  }

  /**
   * Parse a query for bang commands.
   * Bangs can appear at the start (!g query) or end (query !g) of the query.
   * Returns a redirect URL for external bangs, a category for internal bangs,
   * or the cleaned query if no bang matches.
   */
  async parse(query: string): Promise<BangParseResult> {
    const trimmed = query.trim();
    if (!trimmed) {
      return { query: trimmed };
    }

    let trigger: string | null = null;
    let cleanQuery: string;

    // Check for bang at start: !trigger query
    const startMatch = trimmed.match(/^!(\S+)\s*(.*)/);
    if (startMatch) {
      trigger = startMatch[1].toLowerCase();
      cleanQuery = startMatch[2].trim();
    } else {
      // Check for bang at end: query !trigger
      const endMatch = trimmed.match(/(.*)\s+!(\S+)$/);
      if (endMatch) {
        trigger = endMatch[2].toLowerCase();
        cleanQuery = endMatch[1].trim();
      } else {
        return { query: trimmed };
      }
    }

    // Look up the trigger in built-in bangs first
    let bangData = BUILTIN_MAP.get(trigger);

    // If not found in built-ins, check custom bangs from KV
    if (!bangData) {
      const customBang = await this.kvStore.getBang(trigger);
      if (customBang) {
        bangData = customBang;
      }
    }

    if (!bangData) {
      // No matching bang found -- return original query
      return { query: trimmed };
    }

    const encodedQuery = encodeURIComponent(cleanQuery || '');
    const url = bangData.url_template.replace('{query}', encodedQuery);

    // Internal bangs start with / and represent category redirects
    if (url.startsWith('/')) {
      return {
        query: cleanQuery,
        bang: { name: bangData.name, trigger: bangData.trigger },
        category: bangData.category,
        redirect: url,
      };
    }

    // External bang redirect
    return {
      redirect: url,
      bang: { name: bangData.name, trigger: bangData.trigger },
      query: cleanQuery,
    };
  }

  /**
   * List all bangs: built-in bangs combined with custom bangs from KV.
   */
  async listBangs(): Promise<Bang[]> {
    const now = new Date().toISOString();
    const builtins: Bang[] = BUILTIN_BANGS.map((b, idx) => ({
      ...b,
      id: idx + 1,
      created_at: now,
    }));

    const customBangs = await this.kvStore.listBangs();

    return [...builtins, ...customBangs];
  }

  /**
   * Create a custom bang and persist to KV.
   */
  async createBang(bang: Bang): Promise<void> {
    // Prevent overriding built-in bangs
    if (BUILTIN_MAP.has(bang.trigger)) {
      throw new Error(`Cannot override built-in bang: !${bang.trigger}`);
    }
    await this.kvStore.createBang(bang);
  }

  /**
   * Delete a custom bang from KV by trigger.
   */
  async deleteBang(trigger: string): Promise<void> {
    if (BUILTIN_MAP.has(trigger)) {
      throw new Error(`Cannot delete built-in bang: !${trigger}`);
    }
    await this.kvStore.deleteBang(trigger);
  }
}
