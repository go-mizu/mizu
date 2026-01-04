import { useState, useEffect, useRef, useCallback } from 'react'
import { Search, FileText, Database, Clock, Plus, ArrowRight } from 'lucide-react'
import { api } from '../api/client'

interface SearchResult {
  id: string
  type: 'page' | 'database'
  title: string
  icon?: string
  parentTitle?: string
  updatedAt?: string
}

interface QuickSearchProps {
  isOpen: boolean
  onClose: () => void
  workspaceId: string
  onNavigate: (type: 'page' | 'database', id: string) => void
  onCreatePage?: () => void
}

export function QuickSearch({
  isOpen,
  onClose,
  workspaceId,
  onNavigate,
  onCreatePage,
}: QuickSearchProps) {
  const [query, setQuery] = useState('')
  const [results, setResults] = useState<SearchResult[]>([])
  const [recentPages, setRecentPages] = useState<SearchResult[]>([])
  const [isLoading, setIsLoading] = useState(false)
  const [selectedIndex, setSelectedIndex] = useState(0)
  const [isInitialLoad, setIsInitialLoad] = useState(true)
  const inputRef = useRef<HTMLInputElement>(null)
  const listRef = useRef<HTMLDivElement>(null)
  const searchId = useRef(`search-${Math.random().toString(36).slice(2, 9)}`).current

  // Focus input when opened
  useEffect(() => {
    if (isOpen) {
      setQuery('')
      setResults([])
      setSelectedIndex(0)
      setIsInitialLoad(true)
      setTimeout(() => inputRef.current?.focus(), 50)
      loadRecentPages()
    }
  }, [isOpen])

  // Load recent pages
  const loadRecentPages = async () => {
    setIsInitialLoad(true)
    try {
      const data = await api.get<{ pages: SearchResult[] }>('/search/recent')
      setRecentPages(data.pages.slice(0, 5))
    } catch (err) {
      console.error('Failed to load recent pages:', err)
    } finally {
      setIsInitialLoad(false)
    }
  }

  // Search on query change
  useEffect(() => {
    if (!query.trim()) {
      setResults([])
      return
    }

    const searchDebounce = setTimeout(async () => {
      setIsLoading(true)
      try {
        const data = await api.get<{ results: SearchResult[] }>(
          `/search/quick?q=${encodeURIComponent(query)}`
        )
        setResults(data.results)
        setSelectedIndex(0)
      } catch (err) {
        console.error('Search failed:', err)
      } finally {
        setIsLoading(false)
      }
    }, 150)

    return () => clearTimeout(searchDebounce)
  }, [query])

  // Keyboard navigation
  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      const items = query ? results : recentPages

      switch (e.key) {
        case 'ArrowDown':
          e.preventDefault()
          setSelectedIndex((prev) => Math.min(prev + 1, items.length))
          break
        case 'ArrowUp':
          e.preventDefault()
          setSelectedIndex((prev) => Math.max(prev - 1, 0))
          break
        case 'Enter':
          e.preventDefault()
          if (selectedIndex === items.length) {
            // Create new page option
            onCreatePage?.()
            onClose()
          } else if (items[selectedIndex]) {
            const item = items[selectedIndex]
            onNavigate(item.type, item.id)
            onClose()
          }
          break
        case 'Escape':
          e.preventDefault()
          onClose()
          break
      }
    },
    [query, results, recentPages, selectedIndex, onNavigate, onCreatePage, onClose]
  )

  // Scroll selected item into view
  useEffect(() => {
    if (listRef.current) {
      const selected = listRef.current.querySelector('.selected')
      selected?.scrollIntoView({ block: 'nearest' })
    }
  }, [selectedIndex])

  if (!isOpen) return null

  const displayItems = query ? results : recentPages
  const showCreateOption = query.trim().length > 0
  const activeItemId = displayItems[selectedIndex]?.id || (showCreateOption && selectedIndex === displayItems.length ? 'create-option' : undefined)

  // Skeleton loader component
  const SkeletonItem = () => (
    <div className="search-result-item" style={{ pointerEvents: 'none' }}>
      <div className="skeleton skeleton-avatar" style={{ width: 20, height: 20 }} />
      <div className="skeleton skeleton-text" style={{ flex: 1, height: 14 }} />
    </div>
  )

  return (
    <>
      <div
        className="quick-search-overlay"
        onClick={onClose}
        aria-hidden="true"
      />
      <div
        className="quick-search-modal"
        role="dialog"
        aria-modal="true"
        aria-label="Quick search"
      >
        <div className="search-input-wrapper">
          <Search size={18} className="search-icon" aria-hidden="true" />
          <input
            ref={inputRef}
            type="text"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder="Search pages or type to create..."
            className="search-input"
            role="combobox"
            aria-expanded="true"
            aria-controls={`${searchId}-listbox`}
            aria-activedescendant={activeItemId ? `${searchId}-${activeItemId}` : undefined}
            aria-autocomplete="list"
            aria-label="Search pages"
          />
          {isLoading && (
            <div className="search-spinner" role="status" aria-label="Searching..." />
          )}
        </div>

        <div
          className="search-results"
          ref={listRef}
          role="listbox"
          id={`${searchId}-listbox`}
          aria-label={query ? 'Search results' : 'Recent pages'}
        >
          {/* Loading skeletons for initial load */}
          {!query && isInitialLoad && (
            <>
              <div className="results-section-header">
                <Clock size={14} aria-hidden="true" />
                <span>Recent pages</span>
              </div>
              <SkeletonItem />
              <SkeletonItem />
              <SkeletonItem />
            </>
          )}

          {!query && !isInitialLoad && recentPages.length > 0 && (
            <>
              <div className="results-section-header" id={`${searchId}-recent-header`}>
                <Clock size={14} aria-hidden="true" />
                <span>Recent pages</span>
              </div>
              {recentPages.map((page, index) => (
                <button
                  key={page.id}
                  id={`${searchId}-${page.id}`}
                  className={`search-result-item ${selectedIndex === index ? 'selected' : ''}`}
                  onClick={() => {
                    onNavigate(page.type, page.id)
                    onClose()
                  }}
                  onMouseEnter={() => setSelectedIndex(index)}
                  role="option"
                  aria-selected={selectedIndex === index}
                >
                  <span className="result-icon" aria-hidden="true">
                    {page.icon || (page.type === 'database' ? <Database size={16} /> : <FileText size={16} />)}
                  </span>
                  <span className="result-title">{page.title || 'Untitled'}</span>
                  {page.parentTitle && (
                    <span className="result-parent">in {page.parentTitle}</span>
                  )}
                </button>
              ))}
            </>
          )}

          {/* Loading skeletons for search */}
          {query && isLoading && results.length === 0 && (
            <>
              <div className="results-section-header">
                <Search size={14} aria-hidden="true" />
                <span>Searching...</span>
              </div>
              <SkeletonItem />
              <SkeletonItem />
            </>
          )}

          {query && results.length > 0 && (
            <>
              <div className="results-section-header" id={`${searchId}-results-header`}>
                <Search size={14} aria-hidden="true" />
                <span>Search results</span>
              </div>
              {results.map((result, index) => (
                <button
                  key={result.id}
                  id={`${searchId}-${result.id}`}
                  className={`search-result-item ${selectedIndex === index ? 'selected' : ''}`}
                  onClick={() => {
                    onNavigate(result.type, result.id)
                    onClose()
                  }}
                  onMouseEnter={() => setSelectedIndex(index)}
                  role="option"
                  aria-selected={selectedIndex === index}
                >
                  <span className="result-icon" aria-hidden="true">
                    {result.icon || (result.type === 'database' ? <Database size={16} /> : <FileText size={16} />)}
                  </span>
                  <span className="result-title">{result.title || 'Untitled'}</span>
                  {result.parentTitle && (
                    <span className="result-parent">in {result.parentTitle}</span>
                  )}
                </button>
              ))}
            </>
          )}

          {query && results.length === 0 && !isLoading && (
            <div className="no-results" role="status">
              <span>No results for "{query}"</span>
              <p style={{ fontSize: 12, color: 'var(--text-tertiary)', marginTop: 4 }}>
                Try a different search term or create a new page
              </p>
            </div>
          )}

          {showCreateOption && (
            <button
              id={`${searchId}-create-option`}
              className={`search-result-item create-option ${selectedIndex === displayItems.length ? 'selected' : ''}`}
              onClick={() => {
                onCreatePage?.()
                onClose()
              }}
              onMouseEnter={() => setSelectedIndex(displayItems.length)}
              role="option"
              aria-selected={selectedIndex === displayItems.length}
            >
              <span className="result-icon" aria-hidden="true">
                <Plus size={16} />
              </span>
              <span className="result-title">Create page "{query}"</span>
              <ArrowRight size={14} className="result-arrow" aria-hidden="true" />
            </button>
          )}

          {/* Empty state for no recent pages */}
          {!query && !isInitialLoad && recentPages.length === 0 && (
            <div className="empty-state" style={{ padding: 32 }}>
              <Clock size={24} style={{ color: 'var(--text-tertiary)', marginBottom: 8 }} />
              <p style={{ fontSize: 13, color: 'var(--text-secondary)', margin: 0 }}>
                No recent pages
              </p>
              <p style={{ fontSize: 12, color: 'var(--text-tertiary)', margin: '4px 0 0' }}>
                Start typing to search or create a page
              </p>
            </div>
          )}
        </div>

        <div className="search-footer">
          <div className="search-hint" aria-hidden="true">
            <kbd>↑</kbd><kbd>↓</kbd> to navigate
            <kbd>↵</kbd> to select
            <kbd>esc</kbd> to close
          </div>
        </div>
      </div>
    </>
  )
}

// Hook to handle Cmd+K shortcut
export function useQuickSearch() {
  const [isOpen, setIsOpen] = useState(false)

  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
        e.preventDefault()
        setIsOpen((prev) => !prev)
      }
    }

    document.addEventListener('keydown', handleKeyDown)
    return () => document.removeEventListener('keydown', handleKeyDown)
  }, [])

  return {
    isOpen,
    open: () => setIsOpen(true),
    close: () => setIsOpen(false),
    toggle: () => setIsOpen((prev) => !prev),
  }
}
