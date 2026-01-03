import { useState, useEffect, useCallback, useRef } from 'react'
import { User, FileText, Calendar, AtSign, Search, Loader2 } from 'lucide-react'
import { motion, AnimatePresence } from 'framer-motion'
import { api } from '../api/client'

interface MentionUser {
  id: string
  name: string
  email: string
  avatarUrl?: string
}

interface MentionPage {
  id: string
  title: string
  icon?: string
}

interface MentionDate {
  type: 'today' | 'tomorrow' | 'yesterday' | 'custom'
  date?: Date
  label: string
}

type MentionType = 'user' | 'page' | 'date'

interface MentionPickerProps {
  isOpen: boolean
  position: { x: number; y: number }
  query: string
  onSelect: (mention: { type: MentionType; id: string; label: string; data?: unknown }) => void
  onClose: () => void
  workspaceId?: string
}

export function MentionPicker({
  isOpen,
  position,
  query,
  onSelect,
  onClose,
  workspaceId,
}: MentionPickerProps) {
  const [activeTab, setActiveTab] = useState<MentionType>('user')
  const [users, setUsers] = useState<MentionUser[]>([])
  const [pages, setPages] = useState<MentionPage[]>([])
  const [isLoading, setIsLoading] = useState(false)
  const [selectedIndex, setSelectedIndex] = useState(0)
  const containerRef = useRef<HTMLDivElement>(null)

  // Date options
  const dateOptions: MentionDate[] = [
    { type: 'today', label: 'Today', date: new Date() },
    { type: 'tomorrow', label: 'Tomorrow', date: new Date(Date.now() + 86400000) },
    { type: 'yesterday', label: 'Yesterday', date: new Date(Date.now() - 86400000) },
    { type: 'custom', label: 'Pick a date...', date: undefined },
  ]

  // Search users
  const searchUsers = useCallback(async (searchQuery: string) => {
    if (!workspaceId) return
    setIsLoading(true)
    try {
      const response = await api.get<{ users: MentionUser[] }>(
        `/workspaces/${workspaceId}/members?q=${encodeURIComponent(searchQuery)}`
      )
      setUsers(response.users || [])
    } catch (err) {
      console.error('Failed to search users:', err)
      setUsers([])
    } finally {
      setIsLoading(false)
    }
  }, [workspaceId])

  // Search pages
  const searchPages = useCallback(async (searchQuery: string) => {
    if (!workspaceId) return
    setIsLoading(true)
    try {
      const response = await api.get<{ results: MentionPage[] }>(
        `/search?q=${encodeURIComponent(searchQuery)}&type=page&limit=10`
      )
      setPages(response.results || [])
    } catch (err) {
      console.error('Failed to search pages:', err)
      setPages([])
    } finally {
      setIsLoading(false)
    }
  }, [workspaceId])

  // Handle query changes
  useEffect(() => {
    if (!isOpen) return

    const trimmedQuery = query.trim()
    if (activeTab === 'user') {
      searchUsers(trimmedQuery)
    } else if (activeTab === 'page') {
      searchPages(trimmedQuery)
    }
  }, [query, activeTab, isOpen, searchUsers, searchPages])

  // Reset selection when results change
  useEffect(() => {
    setSelectedIndex(0)
  }, [users, pages, activeTab])

  // Get current items based on tab
  const getCurrentItems = useCallback(() => {
    switch (activeTab) {
      case 'user':
        return users.map(u => ({ id: u.id, label: u.name, type: 'user' as const, data: u }))
      case 'page':
        return pages.map(p => ({ id: p.id, label: p.title, type: 'page' as const, data: p }))
      case 'date':
        return dateOptions.map(d => ({ id: d.type, label: d.label, type: 'date' as const, data: d }))
      default:
        return []
    }
  }, [activeTab, users, pages, dateOptions])

  // Handle keyboard navigation
  useEffect(() => {
    if (!isOpen) return

    const handleKeyDown = (e: KeyboardEvent) => {
      const items = getCurrentItems()

      switch (e.key) {
        case 'ArrowDown':
          e.preventDefault()
          setSelectedIndex(prev => Math.min(prev + 1, items.length - 1))
          break
        case 'ArrowUp':
          e.preventDefault()
          setSelectedIndex(prev => Math.max(prev - 1, 0))
          break
        case 'Tab':
          e.preventDefault()
          const tabs: MentionType[] = ['user', 'page', 'date']
          const currentIndex = tabs.indexOf(activeTab)
          setActiveTab(tabs[(currentIndex + 1) % tabs.length])
          break
        case 'Enter':
          e.preventDefault()
          const selectedItem = items[selectedIndex]
          if (selectedItem) {
            onSelect({
              type: selectedItem.type,
              id: selectedItem.id,
              label: selectedItem.label,
              data: selectedItem.data,
            })
          }
          break
        case 'Escape':
          e.preventDefault()
          onClose()
          break
      }
    }

    document.addEventListener('keydown', handleKeyDown)
    return () => document.removeEventListener('keydown', handleKeyDown)
  }, [isOpen, activeTab, selectedIndex, getCurrentItems, onSelect, onClose])

  // Handle click outside
  useEffect(() => {
    if (!isOpen) return

    const handleClickOutside = (e: MouseEvent) => {
      if (containerRef.current && !containerRef.current.contains(e.target as Node)) {
        onClose()
      }
    }

    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [isOpen, onClose])

  if (!isOpen) return null

  const items = getCurrentItems()

  return (
    <motion.div
      ref={containerRef}
      initial={{ opacity: 0, y: -8, scale: 0.95 }}
      animate={{ opacity: 1, y: 0, scale: 1 }}
      exit={{ opacity: 0, y: -8, scale: 0.95 }}
      transition={{ duration: 0.15 }}
      style={{
        position: 'fixed',
        left: position.x,
        top: position.y,
        width: '320px',
        maxHeight: '400px',
        background: 'var(--bg-primary)',
        borderRadius: '8px',
        boxShadow: '0 4px 24px rgba(0, 0, 0, 0.15), 0 0 0 1px rgba(0, 0, 0, 0.05)',
        zIndex: 1000,
        overflow: 'hidden',
        display: 'flex',
        flexDirection: 'column',
      }}
    >
      {/* Header with tabs */}
      <div
        style={{
          display: 'flex',
          borderBottom: '1px solid var(--border-color)',
          padding: '8px',
          gap: '4px',
        }}
      >
        <button
          onClick={() => setActiveTab('user')}
          style={{
            flex: 1,
            padding: '6px 12px',
            borderRadius: '4px',
            border: 'none',
            background: activeTab === 'user' ? 'var(--accent-bg)' : 'none',
            color: activeTab === 'user' ? 'var(--accent-color)' : 'var(--text-secondary)',
            fontSize: '13px',
            fontWeight: 500,
            cursor: 'pointer',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            gap: '6px',
            transition: 'background 0.1s',
          }}
        >
          <User size={14} />
          People
        </button>
        <button
          onClick={() => setActiveTab('page')}
          style={{
            flex: 1,
            padding: '6px 12px',
            borderRadius: '4px',
            border: 'none',
            background: activeTab === 'page' ? 'var(--accent-bg)' : 'none',
            color: activeTab === 'page' ? 'var(--accent-color)' : 'var(--text-secondary)',
            fontSize: '13px',
            fontWeight: 500,
            cursor: 'pointer',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            gap: '6px',
            transition: 'background 0.1s',
          }}
        >
          <FileText size={14} />
          Pages
        </button>
        <button
          onClick={() => setActiveTab('date')}
          style={{
            flex: 1,
            padding: '6px 12px',
            borderRadius: '4px',
            border: 'none',
            background: activeTab === 'date' ? 'var(--accent-bg)' : 'none',
            color: activeTab === 'date' ? 'var(--accent-color)' : 'var(--text-secondary)',
            fontSize: '13px',
            fontWeight: 500,
            cursor: 'pointer',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            gap: '6px',
            transition: 'background 0.1s',
          }}
        >
          <Calendar size={14} />
          Date
        </button>
      </div>

      {/* Search input for users/pages */}
      {activeTab !== 'date' && (
        <div
          style={{
            padding: '8px 12px',
            borderBottom: '1px solid var(--border-color)',
          }}
        >
          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: '8px',
              padding: '6px 10px',
              background: 'var(--bg-secondary)',
              borderRadius: '4px',
            }}
          >
            <Search size={14} style={{ color: 'var(--text-tertiary)' }} />
            <span style={{ fontSize: '13px', color: 'var(--text-secondary)' }}>
              {query || `Search ${activeTab === 'user' ? 'people' : 'pages'}...`}
            </span>
          </div>
        </div>
      )}

      {/* Results list */}
      <div
        style={{
          flex: 1,
          overflowY: 'auto',
          padding: '4px',
        }}
      >
        {isLoading ? (
          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              padding: '24px',
              color: 'var(--text-tertiary)',
            }}
          >
            <Loader2 size={20} style={{ animation: 'spin 1s linear infinite' }} />
          </div>
        ) : items.length === 0 ? (
          <div
            style={{
              padding: '24px',
              textAlign: 'center',
              color: 'var(--text-tertiary)',
              fontSize: '13px',
            }}
          >
            {activeTab === 'user' && 'No people found'}
            {activeTab === 'page' && 'No pages found'}
          </div>
        ) : (
          items.map((item, index) => (
            <button
              key={item.id}
              onClick={() => onSelect(item)}
              onMouseEnter={() => setSelectedIndex(index)}
              style={{
                width: '100%',
                display: 'flex',
                alignItems: 'center',
                gap: '10px',
                padding: '8px 10px',
                borderRadius: '4px',
                border: 'none',
                background: index === selectedIndex ? 'var(--bg-hover)' : 'none',
                cursor: 'pointer',
                textAlign: 'left',
                transition: 'background 0.1s',
              }}
            >
              {/* Icon/Avatar */}
              {activeTab === 'user' && (
                <div
                  style={{
                    width: '28px',
                    height: '28px',
                    borderRadius: '50%',
                    background: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    color: 'white',
                    fontSize: '12px',
                    fontWeight: 600,
                    flexShrink: 0,
                  }}
                >
                  {item.label.charAt(0).toUpperCase()}
                </div>
              )}
              {activeTab === 'page' && (
                <span style={{ fontSize: '18px', flexShrink: 0 }}>
                  {(item.data as MentionPage)?.icon || '📄'}
                </span>
              )}
              {activeTab === 'date' && (
                <Calendar size={18} style={{ color: 'var(--text-secondary)', flexShrink: 0 }} />
              )}

              {/* Label */}
              <div style={{ flex: 1, minWidth: 0 }}>
                <div
                  style={{
                    fontSize: '14px',
                    fontWeight: 500,
                    color: 'var(--text-primary)',
                    overflow: 'hidden',
                    textOverflow: 'ellipsis',
                    whiteSpace: 'nowrap',
                  }}
                >
                  {item.label}
                </div>
                {activeTab === 'user' && (item.data as MentionUser)?.email && (
                  <div
                    style={{
                      fontSize: '12px',
                      color: 'var(--text-tertiary)',
                      overflow: 'hidden',
                      textOverflow: 'ellipsis',
                      whiteSpace: 'nowrap',
                    }}
                  >
                    {(item.data as MentionUser).email}
                  </div>
                )}
                {activeTab === 'date' && (item.data as MentionDate)?.date && (
                  <div
                    style={{
                      fontSize: '12px',
                      color: 'var(--text-tertiary)',
                    }}
                  >
                    {(item.data as MentionDate).date?.toLocaleDateString()}
                  </div>
                )}
              </div>
            </button>
          ))
        )}
      </div>

      {/* Footer hint */}
      <div
        style={{
          padding: '8px 12px',
          borderTop: '1px solid var(--border-color)',
          fontSize: '11px',
          color: 'var(--text-tertiary)',
          display: 'flex',
          gap: '12px',
        }}
      >
        <span><kbd style={{ padding: '2px 4px', background: 'var(--bg-secondary)', borderRadius: '2px' }}>Tab</kbd> to switch</span>
        <span><kbd style={{ padding: '2px 4px', background: 'var(--bg-secondary)', borderRadius: '2px' }}>↑↓</kbd> to navigate</span>
        <span><kbd style={{ padding: '2px 4px', background: 'var(--bg-secondary)', borderRadius: '2px' }}>Enter</kbd> to select</span>
      </div>

      <style>{`
        @keyframes spin {
          from { transform: rotate(0deg); }
          to { transform: rotate(360deg); }
        }
      `}</style>
    </motion.div>
  )
}

// Mention chip component for rendering inline mentions
export function MentionChip({
  type,
  label,
  id,
  onClick,
}: {
  type: MentionType
  label: string
  id: string
  onClick?: () => void
}) {
  const getIcon = () => {
    switch (type) {
      case 'user':
        return <AtSign size={12} />
      case 'page':
        return <FileText size={12} />
      case 'date':
        return <Calendar size={12} />
      default:
        return null
    }
  }

  return (
    <span
      onClick={onClick}
      style={{
        display: 'inline-flex',
        alignItems: 'center',
        gap: '4px',
        padding: '1px 6px',
        borderRadius: '4px',
        background: 'var(--accent-bg)',
        color: 'var(--accent-color)',
        fontSize: '14px',
        fontWeight: 500,
        cursor: onClick ? 'pointer' : 'default',
        transition: 'background 0.1s',
      }}
      onMouseEnter={(e) => {
        if (onClick) e.currentTarget.style.background = 'rgba(35, 131, 226, 0.2)'
      }}
      onMouseLeave={(e) => {
        e.currentTarget.style.background = 'var(--accent-bg)'
      }}
    >
      {getIcon()}
      {label}
    </span>
  )
}
