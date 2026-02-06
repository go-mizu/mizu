/**
 * TypeScript types matching the Go types in the search blueprint.
 */

// ========== Cloudflare Bindings ==========

/**
 * Environment bindings for Cloudflare Workers.
 * This is the single source of truth for Env type across the application.
 */
export interface Env {
  /** KV namespace for search data (cache, settings, history, etc.) */
  SEARCH_KV: KVNamespace;
  /** KV namespace for static assets (frontend files) */
  __STATIC_CONTENT: KVNamespace;
  /** Environment name: development, staging, or production */
  ENVIRONMENT: 'development' | 'staging' | 'production';
}

// Forward declaration for ServiceContainer to avoid circular imports
// These imports are type-only and don't create runtime circular dependencies
import type { CacheStore } from './store/cache';
import type { KVStore } from './store/kv';
import type { MetaSearch } from './engines/metasearch';
import type { SearchService } from './services/search';
import type { BangService } from './services/bang';
import type { InstantService } from './services/instant';
import type { KnowledgeService } from './services/knowledge';
import type { SuggestService } from './services/suggest';

export interface ServiceContainer {
  readonly cache: CacheStore;
  readonly kv: KVStore;
  readonly metasearch: MetaSearch;
  readonly search: SearchService;
  readonly bang: BangService;
  readonly instant: InstantService;
  readonly knowledge: KnowledgeService;
  readonly suggest: SuggestService;
}

/**
 * Hono app type with environment bindings.
 */
export type HonoEnv = {
  Bindings: Env;
  Variables: {
    sessionId?: string;
    services?: ServiceContainer;
  };
};

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
  content?: string;
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
  metadata?: Record<string, unknown>;
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
  color_dominant?: string;
  aspect_ratio?: string;
  license?: string;
  engine?: string;
  score?: number;
}

// ========== Image Search Filter Types ==========

export type ImageSize = 'any' | 'large' | 'medium' | 'small' | 'icon';
export type ImageColor = 'any' | 'color' | 'gray' | 'transparent' | 'red' | 'orange' | 'yellow' | 'green' | 'teal' | 'blue' | 'purple' | 'pink' | 'white' | 'black' | 'brown';
export type ImageType = 'any' | 'face' | 'photo' | 'clipart' | 'lineart' | 'animated';
export type ImageAspect = 'any' | 'tall' | 'square' | 'wide' | 'panoramic';
export type ImageTime = 'any' | 'hour' | 'day' | 'week' | 'month' | 'year';
export type ImageRights = 'any' | 'creative_commons' | 'commercial';
export type ImageFileType = 'any' | 'jpg' | 'png' | 'gif' | 'webp' | 'svg' | 'bmp' | 'ico';
export type SafeSearchLevel = 'off' | 'moderate' | 'strict';

export interface ImageSearchFilters {
  size?: ImageSize;
  color?: ImageColor;
  type?: ImageType;
  aspect?: ImageAspect;
  time?: ImageTime;
  rights?: ImageRights;
  filetype?: ImageFileType;
  safe?: SafeSearchLevel;
  min_width?: number;
  min_height?: number;
  max_width?: number;
  max_height?: number;
}

export interface ImageSearchOptions extends SearchOptions {
  filters?: ImageSearchFilters;
}

export interface ImageSearchResponse {
  query: string;
  filters?: ImageSearchFilters;
  total_results: number;
  results: ImageResult[];
  related_searches?: string[];
  search_time_ms: number;
  page: number;
  per_page: number;
  has_more: boolean;
  cached?: boolean;
}

export interface ReverseImageSearchRequest {
  url?: string;
  image_data?: string; // base64 encoded image
}

export interface ReverseImageSearchResponse {
  query_image: {
    url: string;
    width?: number;
    height?: number;
  };
  exact_matches: ImageResult[];
  similar_images: ImageResult[];
  pages_with_image: SearchResult[];
  search_time_ms: number;
}

// ========== Video Search Filter Types ==========

export type VideoDuration = 'any' | 'short' | 'medium' | 'long';
export type VideoQuality = 'any' | 'hd' | '4k';
export type VideoSort = 'relevance' | 'date' | 'views' | 'duration';

export interface VideoSearchFilters {
  duration?: VideoDuration;
  quality?: VideoQuality;
  time?: ImageTime;
  source?: string;
  cc?: boolean;
  safe?: SafeSearchLevel;
}

export interface VideoSearchOptions extends SearchOptions {
  filters?: VideoSearchFilters;
  sort?: VideoSort;
}

export interface VideoResult {
  id: string;
  url: string;
  title: string;
  description: string;
  thumbnail_url: string;
  thumbnail_width?: number;
  thumbnail_height?: number;
  duration: string;
  duration_seconds?: number;
  channel: string;
  channel_url?: string;
  views?: number;
  views_formatted?: string;
  published_at?: string;
  published_formatted?: string;
  embed_url?: string;
  embed_html?: string;
  source: string;
  source_icon?: string;
  quality?: string;
  has_cc?: boolean;
  is_live?: boolean;
  score: number;
  engines: string[];
  engine?: string;
}

export interface VideoSearchResponse {
  query: string;
  total_results: number;
  results: VideoResult[];
  filters?: VideoSearchFilters;
  available_sources: VideoSourceInfo[];
  search_time_ms: number;
  page: number;
  per_page: number;
  has_more: boolean;
  cached?: boolean;
}

export interface VideoSourceInfo {
  name: string;
  display_name: string;
  icon: string;
  result_count: number;
  enabled: boolean;
}

export interface NewsResult {
  id: string;
  url: string;
  title: string;
  snippet: string;
  source: string;
  source_domain: string;
  author?: string;
  image_url?: string;
  thumbnail_url?: string;
  published_at: string;
  engine: string;
  engines: string[];
  metadata?: Record<string, unknown>;
}

// ========== News Types (Google News Clone) ==========

export type NewsCategory =
  | 'top'
  | 'world'
  | 'nation'
  | 'business'
  | 'technology'
  | 'science'
  | 'health'
  | 'sports'
  | 'entertainment';

export interface NewsArticle {
  id: string;
  url: string;
  title: string;
  snippet: string;
  source: string;
  sourceUrl: string;
  sourceIcon?: string;
  imageUrl?: string;
  publishedAt: string;
  category: NewsCategory;
  engines: string[];
  score: number;
  isBreaking?: boolean;
  relatedArticles?: string[];
  clusterId?: string;
}

export interface UserLocation {
  city: string;
  state?: string;
  country: string;
  lat?: number;
  lng?: number;
}

export interface NewsUserPreferences {
  userId: string;
  location?: UserLocation;
  locations?: UserLocation[];
  followedTopics: string[];
  followedSources: string[];
  hiddenSources: string[];
  interests: string[];
  language: string;
  region: string;
  createdAt: string;
  updatedAt: string;
}

export interface ReadingHistoryEntry {
  articleId: string;
  url: string;
  category: NewsCategory;
  source: string;
  timestamp: string;
  duration?: number;
}

export interface StoryCluster {
  id: string;
  title: string;
  summary: string;
  articles: NewsArticle[];
  timeline?: TimelineEvent[];
  perspectives?: Perspective[];
  updatedAt: string;
}

export interface TimelineEvent {
  timestamp: string;
  title: string;
  description?: string;
}

export interface Perspective {
  label: string;
  articles: NewsArticle[];
}

export interface NewsHomeResponse {
  topStories: NewsArticle[];
  forYou: NewsArticle[];
  localNews: NewsArticle[];
  categories: Partial<Record<NewsCategory, NewsArticle[]>>;
  searchTimeMs: number;
}

export interface NewsCategoryResponse {
  category: NewsCategory;
  articles: NewsArticle[];
  hasMore: boolean;
  page: number;
  searchTimeMs: number;
}

/** Used by the Google News clone service (services/news.ts) */
export interface NewsSearchResponse {
  query: string;
  results: NewsArticle[];
  totalResults: number;
  searchTimeMs: number;
  page: number;
  hasMore: boolean;
}

/** Used by the news tab search (services/search.ts searchNews) */
export interface NewsTabResponse {
  query: string;
  results: NewsResult[];
  total_results: number;
  search_time_ms: number;
  page: number;
  per_page: number;
  has_more: boolean;
  cached?: boolean;
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
  cached?: boolean;
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
