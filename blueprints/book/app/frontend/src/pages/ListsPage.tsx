import { useState, useEffect, useMemo, useCallback } from 'react'
import { Link } from 'react-router-dom'
import { BookOpen, Users, Tag, Search, Loader } from 'lucide-react'
import Header from '../components/Header'
import { booksApi } from '../api/books'
import type { BookList, Genre } from '../types'

export default function ListsPage() {
  const [lists, setLists] = useState<BookList[]>([])
  const [tags, setTags] = useState<string[]>([])
  const [activeTag, setActiveTag] = useState('')
  const [loading, setLoading] = useState(true)
  const [searchQuery, setSearchQuery] = useState('')
  const [searching, setSearching] = useState(false)
  const [searchResults, setSearchResults] = useState<BookList[] | null>(null)
  const [genres, setGenres] = useState<Genre[]>([])

  const fetchLists = (tag?: string) => {
    setLoading(true)
    booksApi.getLists(tag)
      .then(({ lists: l, tags: t }) => { setLists(l); setTags(t) })
      .catch(() => {})
      .finally(() => setLoading(false))
  }

  useEffect(() => {
    fetchLists()
    booksApi.getGenres().then(setGenres).catch(() => {})
  }, [])

  const handleTagFilter = (tag: string) => {
    const next = activeTag === tag ? '' : tag
    setActiveTag(next)
    setSearchQuery('')
    setSearchResults(null)
    fetchLists(next || undefined)
  }

  // Server-side search with debounce
  const doSearch = useCallback((q: string) => {
    if (!q.trim()) {
      setSearchResults(null)
      return
    }
    setSearching(true)
    booksApi.searchLists(q)
      .then(setSearchResults)
      .catch(() => setSearchResults([]))
      .finally(() => setSearching(false))
  }, [])

  useEffect(() => {
    if (!searchQuery.trim()) {
      setSearchResults(null)
      return
    }
    const timer = setTimeout(() => doSearch(searchQuery), 400)
    return () => clearTimeout(timer)
  }, [searchQuery, doSearch])

  // Client-side filter by search query for instant feedback
  const filteredLists = useMemo(() => {
    if (searchResults) return searchResults
    if (!searchQuery.trim()) return lists
    const q = searchQuery.toLowerCase()
    return lists.filter(l =>
      l.title.toLowerCase().includes(q) ||
      (l.description || '').toLowerCase().includes(q) ||
      (l.tag || '').toLowerCase().includes(q)
    )
  }, [lists, searchQuery, searchResults])

  // Group lists by tag for display
  const userLists = filteredLists.filter(l => !l.source_url)
  const seededLists = filteredLists.filter(l => l.source_url)
  const groupedByTag: Record<string, BookList[]> = {}
  for (const list of seededLists) {
    const t = list.tag || 'other'
    if (!groupedByTag[t]) groupedByTag[t] = []
    groupedByTag[t].push(list)
  }
  const tagOrder = Object.keys(groupedByTag).sort()

  const isSearching = !!searchQuery.trim()

  return (
    <>
      <Header />
      <div className="page-container">
        <div className="listopia-header">
          <h1 className="page-title page-title-lg">Listopia</h1>
          <p className="page-subtitle">
            Curated book lists voted by millions of readers
          </p>
          <div className="listopia-search">
            {searching ? (
              <Loader size={16} className="listopia-search-icon spin" />
            ) : (
              <Search size={16} className="listopia-search-icon" />
            )}
            <input
              className="form-input listopia-search-input"
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              placeholder="Search lists (e.g. fantasy, thriller, 2024)..."
            />
          </div>
        </div>

        {/* Tag filter pills */}
        {tags.length > 0 && !isSearching && (
          <div className="list-tag-bar">
            <button
              className={`list-tag-pill ${!activeTag ? 'active' : ''}`}
              onClick={() => handleTagFilter('')}
            >
              All
            </button>
            {tags.map((t) => (
              <button
                key={t}
                className={`list-tag-pill ${activeTag === t ? 'active' : ''}`}
                onClick={() => handleTagFilter(t)}
              >
                {t}
              </button>
            ))}
          </div>
        )}

        {loading ? (
          <div className="loading-spinner"><div className="spinner" /></div>
        ) : filteredLists.length === 0 ? (
          <div className="empty-state">
            {searching ? (
              <>
                <div className="loading-spinner"><div className="spinner" /></div>
                <p>Searching Goodreads for "{searchQuery}"...</p>
              </>
            ) : isSearching ? (
              <>
                <h3>No lists found for "{searchQuery}"</h3>
                <p>Try a different search term like a genre or topic.</p>
              </>
            ) : (
              <>
                <h3>No lists yet</h3>
                <p>Lists are being imported. Please refresh in a moment.</p>
              </>
            )}
          </div>
        ) : (
          <>
            {isSearching && (
              <p className="page-subtitle" style={{ marginBottom: 16 }}>
                {filteredLists.length} list{filteredLists.length !== 1 ? 's' : ''} found
                {searchResults && ' from Goodreads'}
              </p>
            )}

            {/* User-created lists */}
            {userLists.length > 0 && (
              <section className="list-section">
                {!isSearching && <h2 className="list-section-heading">My Lists</h2>}
                <div className="list-grid">
                  {userLists.map((list) => (
                    <ListCard key={list.id} list={list} />
                  ))}
                </div>
              </section>
            )}

            {/* Grouped seeded/imported lists */}
            {isSearching ? (
              seededLists.length > 0 && (
                <section className="list-section">
                  <div className="list-grid">
                    {seededLists.map((list) => (
                      <ListCard key={list.id} list={list} />
                    ))}
                  </div>
                </section>
              )
            ) : activeTag ? (
              seededLists.length > 0 && (
                <section className="list-section">
                  <div className="list-grid">
                    {seededLists.map((list) => (
                      <ListCard key={list.id} list={list} />
                    ))}
                  </div>
                </section>
              )
            ) : (
              tagOrder.map((t) => (
                <section key={t} className="list-section">
                  <h2 className="list-section-heading">{t}</h2>
                  <div className="list-grid">
                    {groupedByTag[t].map((list) => (
                      <ListCard key={list.id} list={list} />
                    ))}
                  </div>
                </section>
              ))
            )}
          </>
        )}

        {/* Browse by Genre */}
        {genres.length > 0 && !isSearching && (
          <section className="list-section" style={{ marginTop: 36 }}>
            <div className="section-header">
              <h2 className="list-section-heading">Browse by Genre</h2>
              <Link to="/browse" className="section-link">View all</Link>
            </div>
            <div className="genre-list">
              {genres.slice(0, 24).map((g) => (
                <Link key={g.slug} to={`/genre/${g.slug}`} className="genre-tag">
                  {g.name}
                  <span className="genre-count">{g.book_count}</span>
                </Link>
              ))}
            </div>
          </section>
        )}
      </div>
    </>
  )
}

function ListCard({ list }: { list: BookList }) {
  const covers = list.cover_urls ? list.cover_urls.split('||').filter(Boolean).slice(0, 5) : []

  return (
    <Link to={`/list/${list.id}`} className="local-list-card">
      {covers.length > 0 && (
        <div className="list-cover-strip">
          {covers.map((url, i) => (
            <img key={i} src={url} alt="" className="list-cover-thumb" loading="lazy" />
          ))}
        </div>
      )}
      <h3 className="local-list-title">{list.title}</h3>
      {list.description && (
        <p className="local-list-description">{list.description}</p>
      )}
      <div className="local-list-meta">
        <span className="source-meta-item">
          <BookOpen size={12} /> {list.item_count} books
        </span>
        {list.voter_count > 0 && (
          <span className="source-meta-item">
            <Users size={12} /> {list.voter_count.toLocaleString()} voters
          </span>
        )}
        {list.tag && (
          <span className="source-meta-item list-tag-badge">
            <Tag size={10} /> {list.tag}
          </span>
        )}
      </div>
    </Link>
  )
}
