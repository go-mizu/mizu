export interface Book {
  id: number
  title: string
  author_names: string
  author_id: number
  isbn13: string
  isbn10: string
  ol_key: string
  cover_url: string
  description: string
  page_count: number
  publish_year: number
  publisher: string
  language: string
  genres: string
  average_rating: number
  ratings_count: number
  user_rating?: number
  user_shelf?: string
  created_at?: string
  updated_at?: string
}

export interface Author {
  id: number
  name: string
  ol_key: string
  photo_url: string
  bio: string
  birth_date: string
  death_date: string
  book_count: number
  created_at?: string
}

export interface Shelf {
  id: number
  name: string
  slug: string
  description: string
  is_exclusive: boolean
  book_count: number
  created_at?: string
}

export interface Review {
  id: number
  book_id: number
  rating: number
  text: string
  book_title?: string
  book_cover?: string
  started_at?: string
  finished_at?: string
  created_at?: string
  updated_at?: string
}

export interface ReadingProgress {
  id: number
  book_id: number
  current_page: number
  total_pages: number
  percent: number
  note: string
  created_at?: string
}

export interface ReadingChallenge {
  id: number
  year: number
  goal: number
  books_read: number
  created_at?: string
}

export interface BookList {
  id: number
  title: string
  description: string
  book_count: number
  vote_count: number
  user_voted?: boolean
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
  likes: number
  book_title?: string
  created_at?: string
}

export interface FeedItem {
  id: number
  action: string
  book_id: number
  book_title: string
  book_cover: string
  author_name: string
  shelf_name: string
  rating: number
  review_text: string
  created_at?: string
}

export interface ReadingStats {
  total_books: number
  total_pages: number
  avg_rating: number
  avg_pages: number
  books_by_month: Record<string, number>
  rating_distribution: Record<string, number>
  top_authors: { name: string; count: number }[]
  genres: Record<string, number>
  pages_by_month: Record<string, number>
  shortest_book: Book | null
  longest_book: Book | null
  highest_rated: Book | null
}

export interface SearchResult {
  books: Book[]
  total_count: number
}

export interface Genre {
  name: string
  count: number
}
