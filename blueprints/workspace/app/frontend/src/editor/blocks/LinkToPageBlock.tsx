import { createReactBlockSpec } from '@blocknote/react'
import { useState, useEffect, useCallback, useRef } from 'react'
import { FileText, ChevronRight, Search, X } from 'lucide-react'
import { motion, AnimatePresence } from 'framer-motion'
import { api } from '../../api/client'

interface PageResult {
  id: string
  title: string
  icon?: string
  parentTitle?: string
}

export const LinkToPageBlock = createReactBlockSpec(
  {
    type: 'linkToPage',
    propSchema: {
      pageId: {
        default: '',
      },
      title: {
        default: '',
      },
      icon: {
        default: '',
      },
    },
    content: 'none',
  },
  {
    render: ({ block, editor }) => {
      const [isSearching, setIsSearching] = useState(!block.props.pageId)
      const [searchQuery, setSearchQuery] = useState('')
      const [searchResults, setSearchResults] = useState<PageResult[]>([])
      const [isLoading, setIsLoading] = useState(false)
      const [selectedIndex, setSelectedIndex] = useState(0)
      const [isHovered, setIsHovered] = useState(false)
      const inputRef = useRef<HTMLInputElement>(null)

      const { pageId, title, icon } = block.props

      // Focus input when searching
      useEffect(() => {
        if (isSearching && inputRef.current) {
          inputRef.current.focus()
        }
      }, [isSearching])

      // Search for pages
      useEffect(() => {
        if (!isSearching) return

        const search = async () => {
          if (!searchQuery.trim()) {
            // Load recent pages when no query
            try {
              const data = await api.get<{ pages: PageResult[] }>('/search/recent')
              setSearchResults(data.pages?.slice(0, 8) || [])
            } catch (err) {
              console.error('Failed to load recent pages:', err)
              setSearchResults([])
            }
            return
          }

          setIsLoading(true)
          try {
            const data = await api.get<{ results: PageResult[] }>(
              `/search/quick?q=${encodeURIComponent(searchQuery)}`
            )
            setSearchResults(data.results || [])
            setSelectedIndex(0)
          } catch (err) {
            console.error('Search failed:', err)
          } finally {
            setIsLoading(false)
          }
        }

        const timeout = setTimeout(search, 150)
        return () => clearTimeout(timeout)
      }, [searchQuery, isSearching])

      const handleSelectPage = useCallback((page: PageResult) => {
        editor.updateBlock(block, {
          props: {
            pageId: page.id,
            title: page.title,
            icon: page.icon || '',
          },
        })
        setIsSearching(false)
        setSearchQuery('')
      }, [block, editor])

      const handleKeyDown = useCallback((e: React.KeyboardEvent) => {
        if (e.key === 'ArrowDown') {
          e.preventDefault()
          setSelectedIndex((prev) => Math.min(prev + 1, searchResults.length - 1))
        } else if (e.key === 'ArrowUp') {
          e.preventDefault()
          setSelectedIndex((prev) => Math.max(prev - 1, 0))
        } else if (e.key === 'Enter') {
          e.preventDefault()
          if (searchResults[selectedIndex]) {
            handleSelectPage(searchResults[selectedIndex])
          }
        } else if (e.key === 'Escape') {
          if (pageId) {
            setIsSearching(false)
            setSearchQuery('')
          } else {
            editor.removeBlocks([block])
          }
        }
      }, [searchResults, selectedIndex, handleSelectPage, pageId, block, editor])

      const handleNavigate = useCallback(() => {
        if (pageId) {
          // Get workspace slug from current URL
          const match = window.location.pathname.match(/\/w\/([^/]+)/)
          const workspaceSlug = match?.[1] || ''
          window.location.href = `/w/${workspaceSlug}/p/${pageId}`
        }
      }, [pageId])

      // Render page picker
      if (isSearching || !pageId) {
        return (
          <div
            className="link-to-page-search"
            style={{
              padding: '8px',
              border: '1px solid var(--border-color, rgba(55, 53, 47, 0.16))',
              borderRadius: '4px',
              background: 'var(--bg-primary, white)',
              margin: '4px 0',
              boxShadow: 'var(--shadow-md)',
            }}
          >
            <div style={{ display: 'flex', alignItems: 'center', gap: '8px', padding: '4px 8px' }}>
              <Search size={14} style={{ color: 'var(--text-tertiary)' }} />
              <input
                ref={inputRef}
                type="text"
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                onKeyDown={handleKeyDown}
                placeholder="Search for a page..."
                style={{
                  flex: 1,
                  padding: '6px 0',
                  border: 'none',
                  outline: 'none',
                  fontSize: '14px',
                  background: 'transparent',
                  color: 'var(--text-primary)',
                }}
              />
              {pageId && (
                <button
                  onClick={() => {
                    setIsSearching(false)
                    setSearchQuery('')
                  }}
                  style={{
                    padding: '4px',
                    background: 'none',
                    border: 'none',
                    cursor: 'pointer',
                    color: 'var(--text-tertiary)',
                    display: 'flex',
                    alignItems: 'center',
                  }}
                >
                  <X size={14} />
                </button>
              )}
            </div>

            <div style={{ borderTop: '1px solid var(--border-color)', marginTop: '4px', paddingTop: '4px' }}>
              {isLoading ? (
                <div style={{ padding: '12px', textAlign: 'center', color: 'var(--text-tertiary)', fontSize: '13px' }}>
                  Searching...
                </div>
              ) : searchResults.length === 0 ? (
                <div style={{ padding: '12px', textAlign: 'center', color: 'var(--text-tertiary)', fontSize: '13px' }}>
                  {searchQuery ? 'No pages found' : 'Start typing to search...'}
                </div>
              ) : (
                <div style={{ maxHeight: '300px', overflowY: 'auto' }}>
                  {searchResults.map((page, index) => (
                    <button
                      key={page.id}
                      onClick={() => handleSelectPage(page)}
                      onMouseEnter={() => setSelectedIndex(index)}
                      style={{
                        width: '100%',
                        display: 'flex',
                        alignItems: 'center',
                        gap: '8px',
                        padding: '8px 12px',
                        border: 'none',
                        background: index === selectedIndex ? 'var(--bg-hover)' : 'transparent',
                        cursor: 'pointer',
                        textAlign: 'left',
                        borderRadius: '4px',
                        fontSize: '14px',
                        color: 'var(--text-primary)',
                      }}
                    >
                      <span style={{ fontSize: '16px' }}>
                        {page.icon || <FileText size={16} style={{ color: 'var(--text-tertiary)' }} />}
                      </span>
                      <span style={{ flex: 1, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                        {page.title || 'Untitled'}
                      </span>
                      {page.parentTitle && (
                        <span style={{ fontSize: '12px', color: 'var(--text-tertiary)' }}>
                          {page.parentTitle}
                        </span>
                      )}
                    </button>
                  ))}
                </div>
              )}
            </div>
          </div>
        )
      }

      // Render linked page
      return (
        <motion.div
          className="link-to-page-block"
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{ duration: 0.15 }}
          style={{
            display: 'flex',
            alignItems: 'center',
            gap: '6px',
            padding: '4px 8px',
            borderRadius: '4px',
            cursor: 'pointer',
            margin: '2px 0',
            transition: 'background 0.1s ease',
            background: isHovered ? 'var(--bg-hover)' : 'transparent',
          }}
          onMouseEnter={() => setIsHovered(true)}
          onMouseLeave={() => setIsHovered(false)}
          onClick={handleNavigate}
        >
          <span style={{ fontSize: '16px', flexShrink: 0 }}>
            {icon || <FileText size={16} style={{ color: 'var(--text-tertiary)' }} />}
          </span>
          <span
            style={{
              flex: 1,
              fontSize: '14px',
              color: 'var(--text-primary)',
              overflow: 'hidden',
              textOverflow: 'ellipsis',
              whiteSpace: 'nowrap',
              textDecoration: 'underline',
              textDecorationColor: 'var(--border-color)',
              textUnderlineOffset: '2px',
            }}
          >
            {title || 'Untitled'}
          </span>
          <ChevronRight
            size={14}
            style={{
              color: 'var(--text-tertiary)',
              opacity: isHovered ? 1 : 0,
              transition: 'opacity 0.1s ease',
            }}
          />

          {/* Edit button on hover */}
          <AnimatePresence>
            {isHovered && (
              <motion.button
                initial={{ opacity: 0, scale: 0.9 }}
                animate={{ opacity: 1, scale: 1 }}
                exit={{ opacity: 0, scale: 0.9 }}
                transition={{ duration: 0.1 }}
                onClick={(e) => {
                  e.stopPropagation()
                  setIsSearching(true)
                }}
                style={{
                  padding: '4px 8px',
                  fontSize: '12px',
                  background: 'var(--bg-primary)',
                  border: '1px solid var(--border-color)',
                  borderRadius: '4px',
                  cursor: 'pointer',
                  color: 'var(--text-secondary)',
                  marginLeft: '4px',
                }}
              >
                Change
              </motion.button>
            )}
          </AnimatePresence>
        </motion.div>
      )
    },
  }
)
