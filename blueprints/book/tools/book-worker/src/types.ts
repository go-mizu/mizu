export interface Env {
  DB: D1Database
  KV: KVNamespace
  ENVIRONMENT: string
}

export type HonoEnv = { Bindings: Env }

export interface Book {
  id: number
  ol_key: string
  google_id: string
  title: string
  original_title: string
  subtitle: string
  description: string
  author_names: string
  cover_url: string
  cover_id: number
  isbn10: string
  isbn13: string
  publisher: string
  publish_date: string
  publish_year: number
  page_count: number
  language: string
  edition_language: string
  format: string
  subjects_json: string
  characters_json: string
  settings_json: string
  literary_awards_json: string
  editions_count: number
  average_rating: number
  ratings_count: number
  created_at: string
  updated_at: string
  goodreads_id: string
  goodreads_url: string
  asin: string
  series: string
  reviews_count: number
  currently_reading: number
  want_to_read: number
  rating_dist: string
  first_published: string
  // computed
  user_rating?: number
  user_shelf?: string
  subjects?: string[]
  characters?: string[]
  settings?: string[]
  literary_awards?: string[]
  rating_distribution?: number[]
}

export interface Author {
  id: number
  ol_key: string
  name: string
  bio: string
  photo_url: string
  birth_date: string
  death_date: string
  works_count: number
  created_at: string
  goodreads_id: string
  followers: number
  genres: string
  influences: string
  website: string
}

export interface Shelf {
  id: number
  name: string
  slug: string
  is_exclusive: number
  is_default: number
  sort_order: number
  book_count?: number
  created_at: string
}

export interface ShelfBook {
  id: number
  shelf_id: number
  book_id: number
  date_added: string
  position: number
  date_started: string | null
  date_read: string | null
  read_count: number
  // joined
  book?: Book
}

export interface Review {
  id: number
  book_id: number
  rating: number
  text: string
  is_spoiler: number
  likes_count: number
  comments_count: number
  started_at: string | null
  finished_at: string | null
  created_at: string
  updated_at: string
  reviewer_name: string
  source: string
  book?: Book
}

export interface ReviewComment {
  id: number
  review_id: number
  author_name: string
  text: string
  created_at: string
}

export interface ReadingProgress {
  id: number
  book_id: number
  page: number
  percent: number
  note: string
  created_at: string
}

export interface ReadingChallenge {
  id: number
  year: number
  goal: number
  progress?: number
  created_at: string
}

export interface BookList {
  id: number
  title: string
  description: string
  item_count?: number
  created_at: string
  items?: BookListItem[]
  goodreads_url: string
  voter_count: number
}

export interface BookListItem {
  id: number
  list_id: number
  book_id: number
  position: number
  votes: number
  book?: Book
}

export interface Quote {
  id: number
  book_id: number
  author_name: string
  text: string
  likes_count: number
  created_at: string
  book?: Book
}

export interface BookNote {
  id: number
  book_id: number
  text: string
  created_at: string
  updated_at: string
}

export interface FeedItem {
  id: number
  type: string
  book_id: number
  book_title: string
  data: string
  created_at: string
}

export interface ReadingStats {
  total_books: number
  total_pages: number
  average_rating: number
  books_per_month: Record<string, number>
  pages_per_month: Record<string, number>
  genre_breakdown: Record<string, number>
  rating_distribution: Record<number, number>
  shortest_book?: Book
  longest_book?: Book
  highest_rated?: Book
  most_popular?: Book
}

export interface OLSearchResult {
  key: string
  title: string
  author_name?: string[]
  first_publish_year?: number
  cover_i?: number
  isbn?: string[]
  subject?: string[]
  publisher?: string[]
  language?: string[]
  ratings_average?: number
  ratings_count?: number
  edition_count?: number
}

export interface Genre {
  name: string
  slug: string
  book_count: number
}
