import { api } from './client'
import type {
  SearchResponse,
  Suggestion,
  ImageResult,
  VideoResult,
  NewsResult,
  SearchHistory,
  SearchSettings,
  SearchLens,
  UserPreference,
  Bang,
  BangResult,
  SummarizeRequest,
  SummarizeResponse,
  EnrichmentResponse,
  CheatSheet,
  WidgetSetting,
} from '../types'

export interface SearchOptions {
  page?: number
  per_page?: number
  time?: string
  region?: string
  lang?: string
  safe?: string
  safe_level?: number // 0=off, 1=moderate, 2=strict
  site?: string
  lens?: string
  verbatim?: boolean
  refetch?: boolean
  version?: number
  before?: string // date filter YYYY-MM-DD
  after?: string // date filter YYYY-MM-DD
  filetype?: string
  // Image filters
  size?: string
  color?: string
  type?: string
}

export interface VideoSearchOptions extends SearchOptions {
  duration?: string // short, medium, long
  source?: string // youtube, vimeo, dailymotion, etc.
  quality?: string // hd
  cc?: boolean // closed captions
}

export const searchApi = {
  // Main search
  search: (query: string, options: SearchOptions = {}): Promise<SearchResponse> => {
    const params = new URLSearchParams({ q: query })
    if (options.page) params.set('page', String(options.page))
    if (options.per_page) params.set('per_page', String(options.per_page))
    if (options.time) params.set('time', options.time)
    if (options.region) params.set('region', options.region)
    if (options.lang) params.set('lang', options.lang)
    if (options.safe) params.set('safe', options.safe)
    if (options.site) params.set('site', options.site)
    if (options.lens) params.set('lens', options.lens)
    if (options.verbatim) params.set('verbatim', 'true')
    if (options.refetch) params.set('refetch', 'true')
    if (options.version) params.set('version', String(options.version))
    return api.get(`/api/search?${params}`)
  },

  // Image search
  searchImages: (query: string, options: SearchOptions = {}): Promise<{ query: string; results: ImageResult[]; total_results?: number }> => {
    const params = new URLSearchParams({ q: query })
    if (options.page) params.set('page', String(options.page))
    if (options.per_page) params.set('per_page', String(options.per_page))
    if (options.refetch) params.set('refetch', 'true')
    if (options.size) params.set('size', options.size)
    if (options.color) params.set('color', options.color)
    if (options.type) params.set('type', options.type)
    return api.get(`/api/search/images?${params}`)
  },

  // Video search
  searchVideos: (query: string, options: VideoSearchOptions = {}): Promise<{ query: string; results: VideoResult[]; total_results?: number }> => {
    const params = new URLSearchParams({ q: query })
    if (options.page) params.set('page', String(options.page))
    if (options.per_page) params.set('per_page', String(options.per_page))
    if (options.refetch) params.set('refetch', 'true')
    if (options.duration) params.set('duration', options.duration)
    if (options.source) params.set('source', options.source)
    if (options.time) params.set('time', options.time)
    if (options.quality) params.set('quality', options.quality)
    if (options.cc) params.set('cc', 'true')
    return api.get(`/api/search/videos?${params}`)
  },

  // Code search
  searchCode: (query: string, options: SearchOptions = {}): Promise<SearchResponse> => {
    const params = new URLSearchParams({ q: query })
    if (options.page) params.set('page', String(options.page))
    if (options.per_page) params.set('per_page', String(options.per_page))
    if (options.time) params.set('time', options.time)
    if (options.refetch) params.set('refetch', 'true')
    return api.get(`/api/search/code?${params}`)
  },

  // Science search
  searchScience: (query: string, options: SearchOptions = {}): Promise<SearchResponse> => {
    const params = new URLSearchParams({ q: query })
    if (options.page) params.set('page', String(options.page))
    if (options.per_page) params.set('per_page', String(options.per_page))
    if (options.time) params.set('time', options.time)
    if (options.refetch) params.set('refetch', 'true')
    return api.get(`/api/search/science?${params}`)
  },

  // Social search
  searchSocial: (query: string, options: SearchOptions = {}): Promise<SearchResponse> => {
    const params = new URLSearchParams({ q: query })
    if (options.page) params.set('page', String(options.page))
    if (options.per_page) params.set('per_page', String(options.per_page))
    if (options.time) params.set('time', options.time)
    if (options.refetch) params.set('refetch', 'true')
    return api.get(`/api/search/social?${params}`)
  },

  // Music search
  searchMusic: (query: string, options: SearchOptions = {}): Promise<SearchResponse> => {
    const params = new URLSearchParams({ q: query })
    if (options.page) params.set('page', String(options.page))
    if (options.per_page) params.set('per_page', String(options.per_page))
    if (options.time) params.set('time', options.time)
    if (options.refetch) params.set('refetch', 'true')
    return api.get(`/api/search/music?${params}`)
  },

  // Maps search
  searchMaps: (query: string, options: SearchOptions = {}): Promise<SearchResponse> => {
    const params = new URLSearchParams({ q: query })
    if (options.page) params.set('page', String(options.page))
    if (options.per_page) params.set('per_page', String(options.per_page))
    if (options.refetch) params.set('refetch', 'true')
    return api.get(`/api/search/maps?${params}`)
  },

  // News search
  searchNews: (query: string, options: SearchOptions = {}): Promise<{ query: string; results: NewsResult[]; total_results: number; has_more: boolean; page: number; per_page: number; search_time_ms: number; cached?: boolean }> => {
    const params = new URLSearchParams({ q: query })
    if (options.page) params.set('page', String(options.page))
    if (options.per_page) params.set('per_page', String(options.per_page))
    if (options.refetch) params.set('refetch', 'true')
    return api.get(`/api/search/news?${params}`)
  },

  // Suggestions
  suggest: (query: string, limit = 10): Promise<Suggestion[]> => {
    const params = new URLSearchParams({ q: query, limit: String(limit) })
    return api.get(`/api/suggest?${params}`)
  },

  // Trending queries
  trending: (limit = 10): Promise<string[]> => {
    return api.get(`/api/suggest/trending?limit=${limit}`)
  },

  // History
  getHistory: (limit = 50, offset = 0): Promise<SearchHistory[]> => {
    return api.get(`/api/history?limit=${limit}&offset=${offset}`)
  },

  clearHistory: (): Promise<void> => {
    return api.delete('/api/history')
  },

  deleteHistoryEntry: (id: string): Promise<void> => {
    return api.delete(`/api/history/${id}`)
  },

  // Settings
  getSettings: (): Promise<SearchSettings> => {
    return api.get('/api/settings')
  },

  updateSettings: (settings: Partial<SearchSettings>): Promise<SearchSettings> => {
    return api.put('/api/settings', settings)
  },

  // Lenses
  getLenses: (): Promise<SearchLens[]> => {
    return api.get('/api/lenses')
  },

  createLens: (lens: Partial<SearchLens>): Promise<SearchLens> => {
    return api.post('/api/lenses', lens)
  },

  updateLens: (id: string, lens: Partial<SearchLens>): Promise<SearchLens> => {
    return api.put(`/api/lenses/${id}`, lens)
  },

  deleteLens: (id: string): Promise<void> => {
    return api.delete(`/api/lenses/${id}`)
  },

  // Preferences
  getPreferences: (): Promise<UserPreference[]> => {
    return api.get('/api/preferences')
  },

  setPreference: (domain: string, action: 'upvote' | 'downvote' | 'block'): Promise<UserPreference> => {
    return api.post('/api/preferences', { domain, action })
  },

  deletePreference: (domain: string): Promise<void> => {
    return api.delete(`/api/preferences/${encodeURIComponent(domain)}`)
  },

  // Bangs
  getBangs: (): Promise<Bang[]> => {
    return api.get('/api/bangs')
  },

  parseBang: (query: string): Promise<BangResult> => {
    return api.get(`/api/bangs/parse?q=${encodeURIComponent(query)}`)
  },

  createBang: (bang: Partial<Bang>): Promise<Bang> => {
    return api.post('/api/bangs', bang)
  },

  deleteBang: (id: number): Promise<void> => {
    return api.delete(`/api/bangs/${id}`)
  },

  // Summarizer
  summarize: (request: SummarizeRequest): Promise<SummarizeResponse> => {
    const params = new URLSearchParams()
    if (request.url) params.set('url', request.url)
    if (request.text) params.set('text', request.text)
    if (request.engine) params.set('engine', request.engine)
    if (request.summary_type) params.set('summary_type', request.summary_type)
    if (request.target_language) params.set('target_language', request.target_language)
    return api.get(`/api/summarize?${params}`)
  },

  // Enrichment (Teclis/TinyGem style small web)
  enrichWeb: (query: string, limit = 10): Promise<EnrichmentResponse> => {
    return api.get(`/api/enrich/web?q=${encodeURIComponent(query)}&limit=${limit}`)
  },

  enrichNews: (query: string, limit = 10): Promise<EnrichmentResponse> => {
    return api.get(`/api/enrich/news?q=${encodeURIComponent(query)}&limit=${limit}`)
  },

  // Widgets
  getWidgetSettings: (): Promise<WidgetSetting[]> => {
    return api.get('/api/widgets')
  },

  updateWidgetSetting: (setting: WidgetSetting): Promise<WidgetSetting> => {
    return api.put('/api/widgets', setting)
  },

  getCheatSheet: (language: string): Promise<CheatSheet> => {
    return api.get(`/api/cheatsheet/${encodeURIComponent(language)}`)
  },

  listCheatSheets: (): Promise<CheatSheet[]> => {
    return api.get('/api/cheatsheets')
  },

  getRelated: (query: string): Promise<{ query: string; related: string[] }> => {
    return api.get(`/api/related?q=${encodeURIComponent(query)}`)
  },

  // Page reader (Jina)
  readPage: (url: string): Promise<{ title: string; url: string; description: string; content: string; images?: string[] }> => {
    return api.get(`/api/read?url=${encodeURIComponent(url)}`)
  },
}
