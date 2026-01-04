import { createReactBlockSpec } from '@blocknote/react'
import React, { useState, useEffect, useCallback } from 'react'
import { Link2, RefreshCw, Unlink, ExternalLink, AlertCircle, Loader2 } from 'lucide-react'
import { motion, AnimatePresence } from 'framer-motion'

interface SyncedBlockData {
  id: string
  content: SyncedBlockContent[]
  pageId: string
  pageName: string
  lastUpdated: string
}

interface SyncedBlockContent {
  id?: string
  type: string
  content?: {
    rich_text?: Array<{ text: string; annotations?: Record<string, boolean | string> }>
    text?: string
    checked?: boolean
    icon?: string
    color?: string
    language?: string
    url?: string
  }
}

// Helper function to render synced block content
function renderSyncedBlock(block: SyncedBlockContent): React.ReactNode {
  const { type, content } = block

  // Extract text from rich_text array
  const getText = (): string => {
    if (content?.rich_text && Array.isArray(content.rich_text)) {
      return content.rich_text.map((rt: { text: string }) => rt.text || '').join('')
    }
    return content?.text || ''
  }

  const text = getText()

  switch (type) {
    case 'paragraph':
      return (
        <p style={{ margin: '2px 0', lineHeight: 1.5 }}>
          {text || <span style={{ color: 'var(--text-placeholder)' }}>Empty paragraph</span>}
        </p>
      )

    case 'heading_1':
      return (
        <h1 style={{ fontSize: '1.875em', fontWeight: 600, margin: '8px 0 4px' }}>
          {text}
        </h1>
      )

    case 'heading_2':
      return (
        <h2 style={{ fontSize: '1.5em', fontWeight: 600, margin: '6px 0 4px' }}>
          {text}
        </h2>
      )

    case 'heading_3':
      return (
        <h3 style={{ fontSize: '1.25em', fontWeight: 600, margin: '4px 0' }}>
          {text}
        </h3>
      )

    case 'bulleted_list':
      return (
        <div style={{ display: 'flex', alignItems: 'flex-start', gap: '4px', margin: '2px 0' }}>
          <span style={{ color: 'var(--text-secondary)' }}>•</span>
          <span>{text}</span>
        </div>
      )

    case 'numbered_list':
      return (
        <div style={{ display: 'flex', alignItems: 'flex-start', gap: '4px', margin: '2px 0' }}>
          <span style={{ color: 'var(--text-secondary)', minWidth: '20px' }}>1.</span>
          <span>{text}</span>
        </div>
      )

    case 'to_do':
      return (
        <div style={{ display: 'flex', alignItems: 'center', gap: '8px', margin: '2px 0' }}>
          <input
            type="checkbox"
            checked={content?.checked || false}
            readOnly
            style={{ width: '16px', height: '16px' }}
          />
          <span style={{ textDecoration: content?.checked ? 'line-through' : 'none' }}>
            {text}
          </span>
        </div>
      )

    case 'quote':
      return (
        <blockquote
          style={{
            borderLeft: '3px solid var(--text-primary)',
            paddingLeft: '14px',
            margin: '4px 0',
            color: 'var(--text-primary)',
          }}
        >
          {text}
        </blockquote>
      )

    case 'callout':
      return (
        <div
          style={{
            display: 'flex',
            alignItems: 'flex-start',
            gap: '8px',
            padding: '12px',
            borderRadius: '4px',
            background: 'var(--bg-secondary)',
            margin: '4px 0',
          }}
        >
          <span style={{ fontSize: '1.2em' }}>{content?.icon || '💡'}</span>
          <span>{text}</span>
        </div>
      )

    case 'code':
      return (
        <pre
          style={{
            background: 'var(--bg-secondary)',
            padding: '12px',
            borderRadius: '4px',
            fontSize: '13px',
            fontFamily: 'var(--font-mono)',
            overflow: 'auto',
            margin: '4px 0',
          }}
        >
          <code>{text}</code>
        </pre>
      )

    case 'divider':
      return (
        <hr
          style={{
            border: 'none',
            borderTop: '1px solid var(--border-color)',
            margin: '8px 0',
          }}
        />
      )

    case 'image':
      return content?.url ? (
        <img
          src={content.url}
          alt=""
          style={{ maxWidth: '100%', borderRadius: '4px', margin: '4px 0' }}
        />
      ) : null

    default:
      // Fallback for unknown block types
      return text ? (
        <div style={{ margin: '2px 0' }}>{text}</div>
      ) : (
        <div style={{ color: 'var(--text-tertiary)', fontStyle: 'italic' }}>
          [{type} block]
        </div>
      )
  }
}

export const SyncedBlock = createReactBlockSpec(
  {
    type: 'syncedBlock',
    propSchema: {
      syncId: {
        default: '',
      },
      originalPageId: {
        default: '',
      },
      originalPageName: {
        default: '',
      },
    },
    content: 'none',
  },
  {
    render: ({ block, editor }) => {
      const [syncedData, setSyncedData] = useState<SyncedBlockData | null>(null)
      const [isLoading, setIsLoading] = useState(true)
      const [error, setError] = useState<string | null>(null)
      const [isHovered, setIsHovered] = useState(false)
      const [showMenu, setShowMenu] = useState(false)
      const [lastSynced, setLastSynced] = useState<Date | null>(null)

      const syncId = block.props.syncId as string
      const originalPageId = block.props.originalPageId as string
      const originalPageName = block.props.originalPageName as string

      // Fetch synced content
      const fetchSyncedContent = useCallback(async () => {
        if (!syncId) {
          setIsLoading(false)
          return
        }

        setIsLoading(true)
        setError(null)

        try {
          const response = await fetch(`/api/v1/synced-blocks/${syncId}`)
          if (!response.ok) {
            throw new Error('Failed to fetch synced content')
          }

          // Check content type before parsing JSON
          const contentType = response.headers.get('content-type')
          if (!contentType || !contentType.includes('application/json')) {
            throw new Error('Invalid response format')
          }

          const data = await response.json()
          setSyncedData(data)
          setLastSynced(new Date())
        } catch (err) {
          console.error('Failed to fetch synced content:', err)
          setError('Unable to load synced content')
        } finally {
          setIsLoading(false)
        }
      }, [syncId])

      // Initial fetch
      useEffect(() => {
        fetchSyncedContent()
      }, [fetchSyncedContent])

      // Unlink synced block (convert to regular content)
      const handleUnlink = useCallback(() => {
        // TODO: Convert synced block to regular blocks
        // This would copy the content and remove the sync reference
        setShowMenu(false)
      }, [])

      // Navigate to original page
      const handleNavigateToOriginal = useCallback(() => {
        if (originalPageId) {
          window.location.href = `/pages/${originalPageId}`
        }
      }, [originalPageId])

      // Empty state - no sync configured
      if (!syncId) {
        return (
          <div
            className="synced-block empty"
            style={{
              display: 'flex',
              flexDirection: 'column',
              alignItems: 'center',
              justifyContent: 'center',
              padding: '32px',
              background: 'var(--bg-secondary)',
              borderRadius: '8px',
              border: '1px dashed var(--border-color)',
              margin: '8px 0',
            }}
          >
            <Link2 size={24} style={{ color: 'var(--text-tertiary)', marginBottom: '12px' }} />
            <p style={{ color: 'var(--text-secondary)', marginBottom: '8px', fontSize: '14px', fontWeight: 500 }}>
              Synced Block
            </p>
            <p style={{ color: 'var(--text-tertiary)', fontSize: '13px', textAlign: 'center', maxWidth: '300px' }}>
              Content synced from another location. Changes made anywhere will be reflected everywhere this block appears.
            </p>
          </div>
        )
      }

      // Loading state
      if (isLoading) {
        return (
          <div
            className="synced-block loading"
            style={{
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              padding: '24px',
              background: 'var(--bg-secondary)',
              borderRadius: '8px',
              margin: '8px 0',
            }}
          >
            <Loader2
              size={20}
              style={{
                color: 'var(--accent-color)',
                animation: 'spin 1s linear infinite',
              }}
            />
            <span style={{ marginLeft: '12px', color: 'var(--text-secondary)', fontSize: '14px' }}>
              Loading synced content...
            </span>
          </div>
        )
      }

      // Error state
      if (error) {
        return (
          <div
            className="synced-block error"
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: '12px',
              padding: '16px',
              background: 'var(--danger-bg)',
              borderRadius: '8px',
              border: '1px solid var(--danger-color)',
              margin: '8px 0',
            }}
          >
            <AlertCircle size={20} style={{ color: 'var(--danger-color)', flexShrink: 0 }} />
            <div style={{ flex: 1 }}>
              <p style={{ color: 'var(--danger-color)', fontSize: '14px', fontWeight: 500, marginBottom: '4px' }}>
                Sync Error
              </p>
              <p style={{ color: 'var(--text-secondary)', fontSize: '13px' }}>
                {error}
              </p>
            </div>
            <button
              onClick={fetchSyncedContent}
              style={{
                padding: '6px 12px',
                background: 'var(--bg-primary)',
                border: '1px solid var(--border-color)',
                borderRadius: '4px',
                fontSize: '13px',
                cursor: 'pointer',
                display: 'flex',
                alignItems: 'center',
                gap: '6px',
              }}
            >
              <RefreshCw size={14} />
              Retry
            </button>
          </div>
        )
      }

      return (
        <div
          className="synced-block"
          onMouseEnter={() => setIsHovered(true)}
          onMouseLeave={() => {
            setIsHovered(false)
            setShowMenu(false)
          }}
          style={{
            position: 'relative',
            margin: '8px 0',
            borderRadius: '8px',
            border: isHovered ? '1px solid var(--accent-color)' : '1px solid transparent',
            transition: 'border-color 0.15s',
          }}
        >
          {/* Sync indicator bar */}
          <AnimatePresence>
            {isHovered && (
              <motion.div
                initial={{ opacity: 0, y: -4 }}
                animate={{ opacity: 1, y: 0 }}
                exit={{ opacity: 0, y: -4 }}
                transition={{ duration: 0.1 }}
                style={{
                  position: 'absolute',
                  top: '-32px',
                  left: '8px',
                  display: 'flex',
                  alignItems: 'center',
                  gap: '8px',
                  padding: '4px 10px',
                  background: 'var(--accent-color)',
                  color: 'white',
                  borderRadius: '4px',
                  fontSize: '12px',
                  fontWeight: 500,
                  zIndex: 10,
                }}
              >
                <Link2 size={12} />
                Synced from {originalPageName || 'another page'}
              </motion.div>
            )}
          </AnimatePresence>

          {/* Toolbar */}
          <AnimatePresence>
            {isHovered && (
              <motion.div
                initial={{ opacity: 0, y: -4 }}
                animate={{ opacity: 1, y: 0 }}
                exit={{ opacity: 0, y: -4 }}
                transition={{ duration: 0.1 }}
                style={{
                  position: 'absolute',
                  top: '-32px',
                  right: '8px',
                  display: 'flex',
                  alignItems: 'center',
                  gap: '4px',
                  zIndex: 10,
                }}
              >
                <button
                  onClick={fetchSyncedContent}
                  title="Refresh synced content"
                  style={{
                    padding: '4px 8px',
                    background: 'var(--bg-primary)',
                    border: '1px solid var(--border-color)',
                    borderRadius: '4px',
                    cursor: 'pointer',
                    display: 'flex',
                    alignItems: 'center',
                    gap: '4px',
                    fontSize: '11px',
                    color: 'var(--text-secondary)',
                    boxShadow: '0 1px 4px rgba(0,0,0,0.1)',
                  }}
                >
                  <RefreshCw size={12} />
                  Sync
                </button>
                <button
                  onClick={handleNavigateToOriginal}
                  title="Go to original"
                  style={{
                    padding: '4px 8px',
                    background: 'var(--bg-primary)',
                    border: '1px solid var(--border-color)',
                    borderRadius: '4px',
                    cursor: 'pointer',
                    display: 'flex',
                    alignItems: 'center',
                    gap: '4px',
                    fontSize: '11px',
                    color: 'var(--text-secondary)',
                    boxShadow: '0 1px 4px rgba(0,0,0,0.1)',
                  }}
                >
                  <ExternalLink size={12} />
                  Original
                </button>
                <button
                  onClick={handleUnlink}
                  title="Unlink (copy content)"
                  style={{
                    padding: '4px 8px',
                    background: 'var(--bg-primary)',
                    border: '1px solid var(--border-color)',
                    borderRadius: '4px',
                    cursor: 'pointer',
                    display: 'flex',
                    alignItems: 'center',
                    gap: '4px',
                    fontSize: '11px',
                    color: 'var(--text-secondary)',
                    boxShadow: '0 1px 4px rgba(0,0,0,0.1)',
                  }}
                >
                  <Unlink size={12} />
                  Unlink
                </button>
              </motion.div>
            )}
          </AnimatePresence>

          {/* Synced content */}
          <div
            className="synced-content"
            style={{
              padding: '16px',
              background: isHovered ? 'rgba(35, 131, 226, 0.04)' : 'transparent',
              borderRadius: '8px',
              minHeight: '60px',
              transition: 'background 0.15s',
            }}
          >
            {syncedData?.content && Array.isArray(syncedData.content) && syncedData.content.length > 0 ? (
              <div className="synced-blocks-container" style={{ color: 'var(--text-primary)', fontSize: '14px' }}>
                {/* Render synced blocks - simplified preview for now */}
                {syncedData.content.map((block: any, index: number) => (
                  <div key={block.id || index} className="synced-block-preview" style={{ marginBottom: '4px' }}>
                    {renderSyncedBlock(block)}
                  </div>
                ))}
              </div>
            ) : (
              <p style={{ color: 'var(--text-tertiary)', fontSize: '14px', fontStyle: 'italic' }}>
                No content synced yet
              </p>
            )}
          </div>

          {/* Last synced indicator */}
          {lastSynced && isHovered && (
            <div
              style={{
                padding: '4px 16px 8px',
                fontSize: '11px',
                color: 'var(--text-tertiary)',
              }}
            >
              Last synced: {lastSynced.toLocaleTimeString()}
            </div>
          )}

          {/* CSS for animation */}
          <style>{`
            @keyframes spin {
              from { transform: rotate(0deg); }
              to { transform: rotate(360deg); }
            }
          `}</style>
        </div>
      )
    },
    // Parse HTML to recreate block when pasting or drag-dropping
    parse: (element: HTMLElement) => {
      if (element.classList.contains('synced-block') || element.hasAttribute('data-sync-id')) {
        return {
          syncId: element.getAttribute('data-sync-id') || '',
          originalPageId: element.getAttribute('data-original-page-id') || '',
          originalPageName: element.getAttribute('data-original-page-name') || '',
        }
      }
      return undefined
    },
    // Convert to external HTML for clipboard/export
    toExternalHTML: ({ block }) => {
      const { syncId, originalPageId, originalPageName } = block.props as Record<string, string>
      return (
        <div
          className="synced-block"
          data-sync-id={syncId}
          data-original-page-id={originalPageId}
          data-original-page-name={originalPageName}
          style={{
            display: 'flex',
            alignItems: 'center',
            gap: '8px',
            padding: '12px 16px',
            background: 'rgba(35, 131, 226, 0.08)',
            borderRadius: '8px',
            border: '1px solid rgba(35, 131, 226, 0.2)',
          }}
        >
          <span>🔗</span>
          <span>Synced from {originalPageName || 'another page'}</span>
        </div>
      )
    },
  }
)
