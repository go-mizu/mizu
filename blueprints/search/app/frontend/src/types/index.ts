export interface SearchResult {
  id: string
  url: string
  title: string
  snippet: string
  domain: string
  favicon?: string
  score: number
  highlights?: string[]
  sitelinks?: Sitelink[]
  crawled_at: string
}

export interface Sitelink {
  title: string
  url: string
}

export interface SearchResponse {
  query: string
  corrected_query?: string
  total_results: number
  results: SearchResult[]
  suggestions?: string[]
  instant_answer?: InstantAnswer
  knowledge_panel?: KnowledgePanel
  related_searches?: string[]
  search_time_ms: number
  page: number
  per_page: number
}

export interface InstantAnswer {
  type: string
  query: string
  result: string
  data?: unknown
}

export interface KnowledgePanel {
  title: string
  subtitle?: string
  description: string
  image?: string
  facts?: Fact[]
  links?: Link[]
  source?: string
}

export interface Fact {
  label: string
  value: string
}

export interface Link {
  title: string
  url: string
  icon?: string
}

export interface Suggestion {
  text: string
  type: string
  frequency?: number
}

export interface ImageResult {
  id: string
  url: string
  thumbnail_url: string
  title: string
  source_url: string
  source_domain: string
  width: number
  height: number
  file_size: number
  format: string
}

export interface VideoResult {
  id: string
  url: string
  thumbnail_url: string
  title: string
  description: string
  duration_seconds: number
  channel: string
  views: number
  published_at: string
}

export interface NewsResult {
  id: string
  url: string
  title: string
  snippet: string
  source: string
  image_url?: string
  published_at: string
}

export interface SearchHistory {
  id: string
  query: string
  results: number
  clicked_url?: string
  searched_at: string
}

export interface SearchSettings {
  safe_search: string
  results_per_page: number
  region: string
  language: string
  theme: string
  open_in_new_tab: boolean
  show_thumbnails: boolean
  show_instant_answers: boolean
  show_knowledge_panel: boolean
  save_history: boolean
  autocomplete_enabled: boolean
}

export interface SearchLens {
  id: string
  name: string
  description?: string
  domains?: string[]
  exclude?: string[]
  keywords?: string[]
  is_public: boolean
  is_built_in: boolean
}

export interface UserPreference {
  id: string
  domain: string
  action: 'upvote' | 'downvote' | 'block'
  created_at: string
}
