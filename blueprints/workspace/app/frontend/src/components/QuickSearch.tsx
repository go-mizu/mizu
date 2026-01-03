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
  const inputRef = useRef<HTMLInputElement>(null)
  const listRef = useRef<HTMLDivElement>(null)

  // Focus input when opened
  useEffect(() => {
    if (isOpen) {
      setQuery('')
      setResults([])
      setSelectedIndex(0)
      setTimeout(() => inputRef.current?.focus(), 50)
      loadRecentPages()
    }
  }, [isOpen])

  // Load recent pages
  const loadRecentPages = async () => {
    try {
      const data = await api.get<{ pages: SearchResult[] }>('/search/recent')
      setRecentPages(data.pages.slice(0, 5))
    } catch (err) {
      console.error('Failed to load recent pages:', err)
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

  return (
    <>
      <div className="quick-search-overlay" onClick={onClose} />
      <div className="quick-search-modal">
        <div className="search-input-wrapper">
          <Search size={18} className="search-icon" />
          <input
            ref={inputRef}
            type="text"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder="Search pages or type to create..."
            className="search-input"
          />
          {isLoading && <div className="search-spinner" />}
        </div>

        <div className="search-results" ref={listRef}>
          {!query && recentPages.length > 0 && (
            <>
              <div className="results-section-header">
                <Clock size={14} />
                <span>Recent pages</span>
              </div>
              {recentPages.map((page, index) => (
                <button
                  key={page.id}
                  className={`search-result-item ${selectedIndex === index ? 'selected' : ''}`}
                  onClick={() => {
                    onNavigate(page.type, page.id)
                    onClose()
                  }}
                  onMouseEnter={() => setSelectedIndex(index)}
                >
                  <span className="result-icon">
                    {page.icon || (page.type === 'database' ? <Database size={16} /> : <FileText size={16} />)}
                  </span>
                  <span className="result-title">{page.title || 'Untitled'}</span>
                  {page.parentTitle && (
                    <span className="result-parent">{page.parentTitle}</span>
                  )}
                </button>
              ))}
            </>
          )}

          {query && results.length > 0 && (
            <>
              <div className="results-section-header">
                <Search size={14} />
                <span>Search results</span>
              </div>
              {results.map((result, index) => (
                <button
                  key={result.id}
                  className={`search-result-item ${selectedIndex === index ? 'selected' : ''}`}
                  onClick={() => {
                    onNavigate(result.type, result.id)
                    onClose()
                  }}
                  onMouseEnter={() => setSelectedIndex(index)}
                >
                  <span className="result-icon">
                    {result.icon || (result.type === 'database' ? <Database size={16} /> : <FileText size={16} />)}
                  </span>
                  <span className="result-title">{result.title || 'Untitled'}</span>
                  {result.parentTitle && (
                    <span className="result-parent">{result.parentTitle}</span>
                  )}
                </button>
              ))}
            </>
          )}

          {query && results.length === 0 && !isLoading && (
            <div className="no-results">
              <span>No results for "{query}"</span>
            </div>
          )}

          {showCreateOption && (
            <button
              className={`search-result-item create-option ${selectedIndex === displayItems.length ? 'selected' : ''}`}
              onClick={() => {
                onCreatePage?.()
                onClose()
              }}
              onMouseEnter={() => setSelectedIndex(displayItems.length)}
            >
              <span className="result-icon">
                <Plus size={16} />
              </span>
              <span className="result-title">Create page "{query}"</span>
              <ArrowRight size={14} className="result-arrow" />
            </button>
          )}
        </div>

        <div className="search-footer">
          <div className="search-hint">
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
