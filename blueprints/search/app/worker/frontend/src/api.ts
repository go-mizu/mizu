const BASE = '/api';

interface SearchOptions {
  page?: number;
  per_page?: number;
  time_range?: string;
  region?: string;
  language?: string;
  safe_search?: string;
  site?: string;
  exclude_site?: string;
  lens?: string;
}

interface SearchResponse {
  query: string;
  corrected_query?: string;
  total_results: number;
  results: SearchResult[];
  widgets?: Widget[];
  suggestions?: string[];
  instant_answer?: InstantAnswer;
  knowledge_panel?: KnowledgePanel;
  related_searches?: string[];
  search_time_ms: number;
  page: number;
  per_page: number;
  has_more: boolean;
  redirect?: string;
  bang?: { name: string; trigger: string };
  category?: string;
}

interface SearchResult {
  id: string;
  url: string;
  title: string;
  snippet: string;
  domain: string;
  favicon?: string;
  thumbnail?: { url: string; width?: number; height?: number };
  published?: string;
  score: number;
  highlights?: string[];
  sitelinks?: { title: string; url: string }[];
  engine?: string;
  engines?: string[];
}

interface ImageResult {
  id: string;
  url: string;
  thumbnail_url: string;
  title: string;
  source_url: string;
  source_domain: string;
  width: number;
  height: number;
  file_size: number;
  format: string;
  engine?: string;
  score?: number;
  thumbnail?: { url: string };
  domain: string;
}

interface ImageSearchFilters {
  size?: 'any' | 'large' | 'medium' | 'small' | 'icon';
  color?: 'any' | 'color' | 'gray' | 'transparent' | 'red' | 'orange' | 'yellow' | 'green' | 'teal' | 'blue' | 'purple' | 'pink' | 'white' | 'black' | 'brown';
  type?: 'any' | 'face' | 'photo' | 'clipart' | 'lineart' | 'animated';
  aspect?: 'any' | 'tall' | 'square' | 'wide' | 'panoramic';
  time?: 'any' | 'day' | 'week' | 'month' | 'year';
  rights?: 'any' | 'creative_commons' | 'commercial';
  filetype?: 'any' | 'jpg' | 'png' | 'gif' | 'webp' | 'svg' | 'bmp' | 'ico';
  safe?: 'off' | 'moderate' | 'strict';
  page?: number;
  per_page?: number;
}

interface ImageSearchResponse {
  query: string;
  filters?: ImageSearchFilters;
  total_results: number;
  results: ImageResult[];
  related_searches?: string[];
  search_time_ms: number;
  page: number;
  per_page: number;
  has_more: boolean;
}

interface ReverseImageSearchResponse {
  query_image: { url: string; width?: number; height?: number };
  exact_matches: ImageResult[];
  similar_images: ImageResult[];
  pages_with_image: SearchResult[];
  search_time_ms: number;
}

interface VideoResult extends SearchResult {
  duration?: string;
  views?: number;
  channel?: string;
  platform?: string;
}

interface NewsResult extends SearchResult {
  source: string;
  published_date: string;
}

interface InstantAnswer {
  type: string;
  query: string;
  result: string;
  data?: any;
}

interface KnowledgePanel {
  title: string;
  subtitle?: string;
  description: string;
  image?: string;
  facts?: { label: string; value: string }[];
  links?: { title: string; url: string; icon?: string }[];
  source?: string;
}

interface Widget {
  type: string;
  title?: string;
  data: any;
}

interface Suggestion {
  text: string;
  type: string;
  frequency?: number;
}

interface UserPreference {
  id: string;
  domain: string;
  action: string;
  level: number;
  created_at: string;
  updated_at?: string;
}

interface SearchLens {
  id: string;
  name: string;
  description?: string;
  domains?: string[];
  exclude?: string[];
  include_keywords?: string[];
  exclude_keywords?: string[];
  region?: string;
  file_type?: string;
  is_public: boolean;
  is_built_in: boolean;
  created_at: string;
  updated_at: string;
}

interface SearchHistory {
  id: string;
  query: string;
  results: number;
  clicked_url?: string;
  searched_at: string;
}

interface SearchSettings {
  safe_search: string;
  results_per_page: number;
  region: string;
  language: string;
  theme: string;
  open_in_new_tab: boolean;
  show_thumbnails: boolean;
}

interface Bang {
  id: string;
  trigger: string;
  name: string;
  url_template: string;
  category: string;
  is_default: boolean;
}

interface BangParseResult {
  is_bang: boolean;
  bang?: Bang;
  query?: string;
  redirect_url?: string;
}

async function get<T>(path: string, params?: Record<string, string>): Promise<T> {
  let url = `${BASE}${path}`;
  if (params) {
    const qs = new URLSearchParams();
    Object.entries(params).forEach(([k, v]) => {
      if (v !== undefined && v !== '' && v !== null) qs.set(k, v);
    });
    const str = qs.toString();
    if (str) url += `?${str}`;
  }
  const res = await fetch(url);
  if (!res.ok) {
    throw new Error(`API error: ${res.status} ${res.statusText}`);
  }
  return res.json();
}

async function post<T>(path: string, body?: any): Promise<T> {
  const res = await fetch(`${BASE}${path}`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: body ? JSON.stringify(body) : undefined,
  });
  if (!res.ok) {
    throw new Error(`API error: ${res.status} ${res.statusText}`);
  }
  return res.json();
}

async function put<T>(path: string, body: any): Promise<T> {
  const res = await fetch(`${BASE}${path}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  });
  if (!res.ok) {
    throw new Error(`API error: ${res.status} ${res.statusText}`);
  }
  return res.json();
}

async function del<T>(path: string): Promise<T> {
  const res = await fetch(`${BASE}${path}`, { method: 'DELETE' });
  if (!res.ok) {
    throw new Error(`API error: ${res.status} ${res.statusText}`);
  }
  return res.json();
}

function searchParams(query: string, options?: SearchOptions): Record<string, string> {
  const params: Record<string, string> = { q: query };
  if (options) {
    if (options.page !== undefined) params.page = String(options.page);
    if (options.per_page !== undefined) params.per_page = String(options.per_page);
    if (options.time_range) params.time_range = options.time_range;
    if (options.region) params.region = options.region;
    if (options.language) params.language = options.language;
    if (options.safe_search) params.safe_search = options.safe_search;
    if (options.site) params.site = options.site;
    if (options.exclude_site) params.exclude_site = options.exclude_site;
    if (options.lens) params.lens = options.lens;
  }
  return params;
}

export const api = {
  search(query: string, options?: SearchOptions): Promise<SearchResponse> {
    return get<SearchResponse>('/search', searchParams(query, options));
  },

  searchImages(query: string, options?: ImageSearchFilters): Promise<ImageSearchResponse> {
    const params: Record<string, string> = { q: query };
    if (options) {
      if (options.page !== undefined) params.page = String(options.page);
      if (options.per_page !== undefined) params.per_page = String(options.per_page);
      if (options.size && options.size !== 'any') params.size = options.size;
      if (options.color && options.color !== 'any') params.color = options.color;
      if (options.type && options.type !== 'any') params.type = options.type;
      if (options.aspect && options.aspect !== 'any') params.aspect = options.aspect;
      if (options.time && options.time !== 'any') params.time = options.time;
      if (options.rights && options.rights !== 'any') params.rights = options.rights;
      if (options.filetype && options.filetype !== 'any') params.filetype = options.filetype;
      if (options.safe) params.safe = options.safe;
    }
    return get('/search/images', params);
  },

  reverseImageSearch(url: string): Promise<ReverseImageSearchResponse> {
    return post('/search/images/reverse', { url });
  },

  searchVideos(query: string, options?: SearchOptions): Promise<SearchResponse & { results: VideoResult[] }> {
    return get('/search/videos', searchParams(query, options));
  },

  searchNews(query: string, options?: SearchOptions): Promise<SearchResponse & { results: NewsResult[] }> {
    return get('/search/news', searchParams(query, options));
  },

  suggest(query: string): Promise<Suggestion[]> {
    return get<Suggestion[]>('/suggest', { q: query });
  },

  trending(): Promise<Suggestion[]> {
    return get<Suggestion[]>('/suggest/trending');
  },

  calculate(expr: string): Promise<InstantAnswer> {
    return get<InstantAnswer>('/instant/calculate', { q: expr });
  },

  convert(expr: string): Promise<InstantAnswer> {
    return get<InstantAnswer>('/instant/convert', { q: expr });
  },

  currency(expr: string): Promise<InstantAnswer> {
    return get<InstantAnswer>('/instant/currency', { q: expr });
  },

  weather(location: string): Promise<InstantAnswer> {
    return get<InstantAnswer>('/instant/weather', { q: location });
  },

  define(word: string): Promise<InstantAnswer> {
    return get<InstantAnswer>('/instant/define', { q: word });
  },

  time(location: string): Promise<InstantAnswer> {
    return get<InstantAnswer>('/instant/time', { q: location });
  },

  knowledge(query: string): Promise<KnowledgePanel> {
    return get<KnowledgePanel>(`/knowledge/${encodeURIComponent(query)}`);
  },

  getPreferences(): Promise<UserPreference[]> {
    return get<UserPreference[]>('/preferences');
  },

  setPreference(domain: string, action: string): Promise<UserPreference> {
    return post<UserPreference>('/preferences', { domain, action });
  },

  deletePreference(domain: string): Promise<{ ok: boolean }> {
    return del<{ ok: boolean }>(`/preferences/${encodeURIComponent(domain)}`);
  },

  getLenses(): Promise<SearchLens[]> {
    return get<SearchLens[]>('/lenses');
  },

  createLens(lens: Partial<SearchLens>): Promise<SearchLens> {
    return post<SearchLens>('/lenses', lens);
  },

  deleteLens(id: string): Promise<{ ok: boolean }> {
    return del<{ ok: boolean }>(`/lenses/${encodeURIComponent(id)}`);
  },

  getHistory(): Promise<SearchHistory[]> {
    return get<SearchHistory[]>('/history');
  },

  clearHistory(): Promise<{ ok: boolean }> {
    return del<{ ok: boolean }>('/history');
  },

  deleteHistoryItem(id: string): Promise<{ ok: boolean }> {
    return del<{ ok: boolean }>(`/history/${encodeURIComponent(id)}`);
  },

  getSettings(): Promise<SearchSettings> {
    return get<SearchSettings>('/settings');
  },

  updateSettings(settings: Partial<SearchSettings>): Promise<SearchSettings> {
    return put<SearchSettings>('/settings', settings);
  },

  getBangs(): Promise<Bang[]> {
    return get<Bang[]>('/bangs');
  },

  parseBang(query: string): Promise<BangParseResult> {
    return get<BangParseResult>('/bangs/parse', { q: query });
  },

  getRelated(query: string): Promise<string[]> {
    return get<string[]>('/related', { q: query });
  },
};

export type {
  SearchOptions,
  SearchResponse,
  SearchResult,
  ImageResult,
  ImageSearchFilters,
  ImageSearchResponse,
  ReverseImageSearchResponse,
  VideoResult,
  NewsResult,
  InstantAnswer,
  KnowledgePanel,
  Widget,
  Suggestion,
  UserPreference,
  SearchLens,
  SearchHistory,
  SearchSettings,
  Bang,
  BangParseResult,
};
