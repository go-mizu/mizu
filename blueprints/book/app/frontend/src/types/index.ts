export interface Book {
  id: number
  title: string
  subtitle?: string
  description?: string
  authors?: Author[]
  author_names: string
  ol_key: string
  google_id?: string
  cover_url?: string
  cover_id?: number
  isbn10?: string
  isbn13?: string
  publisher?: string
  publish_date?: string
  publish_year: number
  page_count: number
  language?: string
  format?: string
  subjects?: string[]
  average_rating: number
  ratings_count: number
  user_rating?: number
  user_shelf?: string
  created_at?: string
  updated_at?: string
  goodreads_id?: string
  asin?: string
  series?: string
  reviews_count: number
  currently_reading: number
  want_to_read: number
  rating_dist: number[]  // [5star, 4star, 3star, 2star, 1star]
  first_published?: string
}

export interface Author {
  id: number
  name: string
  ol_key: string
  photo_url?: string
  bio?: string
  birth_date?: string
  death_date?: string
  works_count: number
  created_at?: string
}

export interface Shelf {
  id: number
  name: string
  slug: string
  is_exclusive: boolean
  is_default: boolean
  sort_order: number
  book_count: number
  created_at?: string
}

export interface Review {
  id: number
  book_id: number
  rating: number
  text?: string
  is_spoiler?: boolean
  likes_count?: number
  started_at?: string
  finished_at?: string
  created_at?: string
  updated_at?: string
  book?: Book
}

export interface ReadingProgress {
  id: number
  book_id: number
  page: number
  percent: number
  note?: string
  created_at?: string
}

export interface ReadingChallenge {
  id: number
  year: number
  goal: number
  progress: number
  created_at?: string
}

export interface BookList {
  id: number
  title: string
  description?: string
  item_count: number
  items?: BookListItem[]
  created_at?: string
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
  book?: Book
  created_at?: string
}

export interface FeedItem {
  id: number
  type: string
  book_id?: number
  book_title?: string
  data: string
  created_at?: string
}

export interface ReadingStats {
  total_books: number
  total_pages: number
  average_rating: number
  books_per_month: Record<string, number>
  pages_per_month: Record<string, number>
  genre_breakdown: Record<string, number>
  rating_distribution: Record<string, number>
  shortest_book: Book | null
  longest_book: Book | null
  highest_rated: Book | null
  most_popular: Book | null
}

export interface SearchResult {
  books: Book[]
  total_count: number
  page: number
  page_size: number
}

export interface Genre {
  name: string
  slug: string
  book_count: number
}
