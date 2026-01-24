import { useState, useRef, useEffect } from 'react'
import { MoreVertical, ThumbsUp, ThumbsDown, Ban } from 'lucide-react'
import type { SearchResult as SearchResultType } from '../types'
import { searchApi } from '../api/search'
import { useSearchStore } from '../stores/searchStore'

interface SearchResultProps {
  result: SearchResultType
}

export function SearchResult({ result }: SearchResultProps) {
  const { settings } = useSearchStore()
  const [showMenu, setShowMenu] = useState(false)
  const menuRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) {
        setShowMenu(false)
      }
    }
    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [])

  const handlePreference = async (action: 'upvote' | 'downvote' | 'block') => {
    try {
      await searchApi.setPreference(result.domain, action)
      setShowMenu(false)
    } catch (error) {
      console.error('Failed to set preference:', error)
    }
  }

  // Parse URL for display
  const displayUrl = (() => {
    try {
      const url = new URL(result.url)
      const pathParts = url.pathname.split('/').filter(Boolean)
      return {
        domain: url.hostname,
        path: pathParts.slice(0, 2).join(' › ')
      }
    } catch {
      return { domain: result.url, path: '' }
    }
  })()

  return (
    <div className="search-result group">
      {/* URL breadcrumb */}
      <div className="search-result-url">
        {result.favicon && (
          <img
            src={result.favicon}
            alt=""
            onError={(e) => {
              (e.target as HTMLImageElement).style.display = 'none'
            }}
          />
        )}
        <span>{displayUrl.domain}</span>
        {displayUrl.path && (
          <span className="search-result-breadcrumb">› {displayUrl.path}</span>
        )}

        {/* Actions menu */}
        <div className="relative ml-auto" ref={menuRef}>
          <button
            type="button"
            onClick={() => setShowMenu(!showMenu)}
            className="opacity-0 group-hover:opacity-100 p-1 text-[#70757a] hover:bg-[#f1f3f4] rounded-full transition-opacity"
          >
            <MoreVertical size={16} />
          </button>

          {showMenu && (
            <div className="dropdown-menu">
              <button
                type="button"
                className="dropdown-item"
                onClick={() => handlePreference('upvote')}
              >
                <ThumbsUp size={16} />
                Boost {result.domain}
              </button>
              <button
                type="button"
                className="dropdown-item"
                onClick={() => handlePreference('downvote')}
              >
                <ThumbsDown size={16} />
                Lower {result.domain}
              </button>
              <div className="dropdown-divider" />
              <button
                type="button"
                className="dropdown-item danger"
                onClick={() => handlePreference('block')}
              >
                <Ban size={16} />
                Block {result.domain}
              </button>
            </div>
          )}
        </div>
      </div>

      {/* Title */}
      <a
        href={result.url}
        target={settings.open_in_new_tab ? '_blank' : '_self'}
        rel="noopener noreferrer"
        className="search-result-title"
      >
        {result.title}
      </a>

      {/* Snippet */}
      <p
        className="search-result-snippet"
        dangerouslySetInnerHTML={{ __html: result.snippet }}
      />

      {/* Sitelinks */}
      {(result.sitelinks?.length ?? 0) > 0 && (
        <div className="flex flex-wrap gap-x-4 gap-y-1 mt-2">
          {(result.sitelinks || []).map((link) => (
            <a
              key={link.url}
              href={link.url}
              target={settings.open_in_new_tab ? '_blank' : '_self'}
              rel="noopener noreferrer"
              className="text-sm text-[#1a0dab] hover:underline"
            >
              {link.title}
            </a>
          ))}
        </div>
      )}
    </div>
  )
}
