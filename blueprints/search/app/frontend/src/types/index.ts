export interface Thumbnail {
  url: string
  width?: number
  height?: number
}

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
  thumbnail?: Thumbnail
  published?: string
  engine?: string
  engines?: string[]
}

export interface Sitelink {
  title: string
  url: string
}

export interface Widget {
  type: WidgetType
  title: string
  position: number
  content: unknown
}

export type WidgetType =
  | 'cheat_sheet'
  | 'related_searches'
  | 'quick_peek'
  | 'weather'
  | 'stock'
  | 'calculator'
  | 'unit_converter'
  | 'currency_converter'
  | 'timer'
  | 'stopwatch'
  | 'color_picker'
  | 'qr_code'
  | 'ip_address'
  | 'package_tracker'
  | 'lyrics'
  | 'recipe'

export interface CheatSheet {
  language: string
  title: string
  description: string
  sections: CheatSection[]
}

export interface CheatSection {
  title: string
  items: CheatItem[]
}

export interface CheatItem {
  code: string
  description: string
  category?: string
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
  widgets?: Widget[]
  has_more?: boolean
  search_time_ms: number
  page: number
  per_page: number
  // Bang redirect
  redirect?: string
  bang?: Bang
  category?: string
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
  embed_url?: string
  source_domain?: string
  duration?: string
  engine?: string
}

export interface NewsResult {
  id: string
  url: string
  title: string
  snippet: string
  source: string
  source_name?: string
  source_domain?: string
  image_url?: string
  thumbnail_url?: string
  published_at: string
  engine?: string
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
  level?: number // -2=blocked, -1=lowered, 0=normal, 1=raised, 2=pinned
  created_at: string
}

// Bang types
export interface Bang {
  id: number
  trigger: string
  name: string
  url_template: string
  category: string
  is_builtin: boolean
  user_id?: string
  created_at: string
}

export interface BangResult {
  bang?: Bang
  query: string
  orig_query: string
  redirect?: string
  internal: boolean
  category?: string
}

// Summarizer types
export type SummaryEngine = 'cecil' | 'agnes' | 'muriel'
export type SummaryType = 'summary' | 'takeaway' | 'key_moments'

export interface SummarizeRequest {
  url?: string
  text?: string
  engine?: SummaryEngine
  summary_type?: SummaryType
  target_language?: string
}

export interface SummarizeResponse {
  output: string
  tokens: number
  cached: boolean
  engine: SummaryEngine
}

// Enrichment types (Teclis/TinyGem style)
export interface EnrichmentResult {
  type: string
  rank: number
  url: string
  title: string
  snippet: string
  published?: string
}

export interface EnrichmentResponse {
  meta: {
    id: string
    node: string
    ms: number
  }
  data: EnrichmentResult[]
}

// Widget settings
export interface WidgetSetting {
  user_id: string
  widget_type: WidgetType
  enabled: boolean
  position: number
}

// Re-export AI types
export * from './ai'
