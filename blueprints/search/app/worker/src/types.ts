/**
 * TypeScript types matching the Go types in the search blueprint.
 */

// ========== Cloudflare Bindings ==========

export interface Env {
  SEARCH_KV: KVNamespace;
  ENVIRONMENT?: string;
}

// ========== Document Types ==========

export interface Sitelink {
  title: string;
  url: string;
}

export interface Thumbnail {
  url: string;
  width?: number;
  height?: number;
}

export interface SearchResult {
  id: string;
  url: string;
  title: string;
  snippet: string;
  domain: string;
  favicon?: string;
  thumbnail?: Thumbnail;
  published?: string;
  score: number;
  highlights?: string[];
  sitelinks?: Sitelink[];
  crawled_at: string;
  engine?: string;
  engines?: string[];
}

export interface ImageResult {
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
}

export interface VideoResult {
  id: string;
  url: string;
  thumbnail_url: string;
  title: string;
  description: string;
  duration_seconds: number;
  channel: string;
  views: number;
  published_at: string;
  embed_url?: string;
  engine?: string;
}

export interface NewsResult {
  id: string;
  url: string;
  title: string;
  snippet: string;
  source: string;
  image_url?: string;
  published_at: string;
  engine?: string;
}

// ========== Search Options & Response ==========

export interface SearchOptions {
  page: number;
  per_page: number;
  time_range?: string;
  region?: string;
  language?: string;
  safe_search?: string;
  safe_level?: number;
  verbatim?: boolean;
  site?: string;
  file_type?: string;
  exclude_site?: string;
  date_before?: string;
  date_after?: string;
  lens?: string;
  refetch?: boolean;
  version?: number;
}

export interface SearchResponse {
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
  bang?: string;
  category?: string;
}

// ========== Suggestion Types ==========

export interface Suggestion {
  text: string;
  type: string;
  frequency?: number;
}

// ========== Instant Answer Types ==========

export interface InstantAnswer {
  type: string;
  query: string;
  result: string;
  data?: unknown;
}

export interface CalculatorResult {
  expression: string;
  result: number;
  formatted: string;
}

export interface UnitConversionResult {
  from_value: number;
  from_unit: string;
  to_value: number;
  to_unit: string;
  category: string;
}

export interface CurrencyResult {
  from_amount: number;
  from_currency: string;
  to_amount: number;
  to_currency: string;
  rate: number;
  updated_at: string;
}

export interface WeatherResult {
  location: string;
  temperature: number;
  unit: string;
  condition: string;
  humidity: number;
  wind_speed: number;
  wind_unit: string;
  icon: string;
}

export interface DefinitionResult {
  word: string;
  phonetic?: string;
  part_of_speech: string;
  definitions: string[];
  synonyms?: string[];
  antonyms?: string[];
  examples?: string[];
}

export interface TimeResult {
  location: string;
  time: string;
  date: string;
  timezone: string;
  offset: string;
}

// ========== Knowledge Panel Types ==========

export interface KnowledgePanel {
  title: string;
  subtitle?: string;
  description: string;
  image?: string;
  facts?: Fact[];
  links?: Link[];
  source?: string;
}

export interface Fact {
  label: string;
  value: string;
}

export interface Link {
  title: string;
  url: string;
  icon?: string;
}

export interface Entity {
  id: string;
  name: string;
  type: string;
  description: string;
  image?: string;
  facts?: Record<string, unknown>;
  links?: Link[];
  created_at: string;
  updated_at: string;
}

// ========== User Preference Types ==========

export const PreferenceBlocked = -2;
export const PreferenceLowered = -1;
export const PreferenceNormal = 0;
export const PreferenceRaised = 1;
export const PreferencePinned = 2;

export interface UserPreference {
  id: string;
  domain: string;
  action: string;
  level: number;
  created_at: string;
  updated_at?: string;
}

export interface SearchLens {
  id: string;
  name: string;
  description?: string;
  domains?: string[];
  exclude?: string[];
  include_keywords?: string[];
  exclude_keywords?: string[];
  keywords?: string[];
  region?: string;
  file_type?: string;
  date_before?: string;
  date_after?: string;
  is_public: boolean;
  is_built_in: boolean;
  is_shared: boolean;
  share_link?: string;
  user_id?: string;
  created_at: string;
  updated_at: string;
}

export interface SearchHistory {
  id: string;
  query: string;
  results: number;
  clicked_url?: string;
  searched_at: string;
}

export interface SearchSettings {
  safe_search: string;
  results_per_page: number;
  region: string;
  language: string;
  theme: string;
  open_in_new_tab: boolean;
  show_thumbnails: boolean;
}

// ========== Bang Types ==========

export interface Bang {
  id: number;
  trigger: string;
  name: string;
  url_template: string;
  category: string;
  is_builtin: boolean;
  user_id?: string;
  created_at: string;
}

// ========== Widget Types ==========

export type WidgetType =
  | 'inline_images'
  | 'inline_videos'
  | 'inline_news'
  | 'inline_discussions'
  | 'interesting_finds'
  | 'listicles'
  | 'inline_maps'
  | 'public_records'
  | 'podcasts'
  | 'quick_peek'
  | 'summary_box'
  | 'cheat_sheet'
  | 'blast_from_past'
  | 'code'
  | 'related_searches'
  | 'wikipedia';

export interface Widget {
  type: WidgetType;
  title?: string;
  position: number;
  content: unknown;
}

export interface CheatSheet {
  language: string;
  title: string;
  sections: CheatSection[];
}

export interface CheatSection {
  name: string;
  items: CheatItem[];
}

export interface CheatItem {
  code: string;
  description: string;
}
