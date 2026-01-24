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
} from '../types'

export interface SearchOptions {
  page?: number
  per_page?: number
  time?: string
  region?: string
  lang?: string
  safe?: string
  site?: string
  lens?: string
  verbatim?: boolean
  refetch?: boolean
  version?: number
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
    return api.get(`/api/search/images?${params}`)
  },

  // Video search
  searchVideos: (query: string, options: SearchOptions = {}): Promise<{ query: string; results: VideoResult[]; total_results?: number }> => {
    const params = new URLSearchParams({ q: query })
    if (options.page) params.set('page', String(options.page))
    if (options.per_page) params.set('per_page', String(options.per_page))
    if (options.refetch) params.set('refetch', 'true')
    return api.get(`/api/search/videos?${params}`)
  },

  // News search
  searchNews: (query: string, options: SearchOptions = {}): Promise<{ query: string; results: NewsResult[]; total_results?: number }> => {
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
}
