import { useState, useCallback, useRef, useEffect, useMemo } from 'react'
import { motion, AnimatePresence, Reorder } from 'framer-motion'
import {
  ChevronRight,
  FileText,
  Plus,
  MoreHorizontal,
  Trash2,
  Copy,
  ExternalLink,
  Star,
  StarOff,
  Archive,
  Link2,
  Edit2,
  ChevronDown,
} from 'lucide-react'
import { api, Page } from '../api/client'

interface PageTreeProps {
  workspaceSlug: string
  pages: Page[]
  currentPageId?: string
  expandedIds?: Set<string>
  onPageClick?: (page: Page) => void
  onPageCreate?: (parentId: string | null) => void
  onPageDelete?: (pageId: string) => void
  onPageDuplicate?: (pageId: string) => void
  onPageMove?: (pageId: string, newParentId: string | null) => void
  onToggleFavorite?: (pageId: string, isFavorite: boolean) => void
  onRename?: (pageId: string, newTitle: string) => void
  allowDragDrop?: boolean
  showActions?: boolean
  maxDepth?: number
}

interface TreeNode {
  page: Page
  children: TreeNode[]
  depth: number
}

interface ContextMenuState {
  pageId: string
  x: number
  y: number
}

// Build tree structure from flat pages array
function buildTree(pages: Page[], parentId: string | null = null, depth = 0): TreeNode[] {
  return pages
    .filter(p => p.parent_id === parentId || (!p.parent_id && !parentId))
    .map(page => ({
      page,
      children: buildTree(pages, page.id, depth + 1),
      depth,
    }))
    .sort((a, b) => (a.page.position ?? 0) - (b.page.position ?? 0))
}

export function PageTree({
  workspaceSlug,
  pages,
  currentPageId,
  expandedIds: controlledExpandedIds,
  onPageClick,
  onPageCreate,
  onPageDelete,
  onPageDuplicate,
  onPageMove,
  onToggleFavorite,
  onRename,
  allowDragDrop = true,
  showActions = true,
  maxDepth = 10,
}: PageTreeProps) {
  const [internalExpandedIds, setInternalExpandedIds] = useState<Set<string>>(new Set())
  const [contextMenu, setContextMenu] = useState<ContextMenuState | null>(null)
  const [renamingId, setRenamingId] = useState<string | null>(null)
  const [renameValue, setRenameValue] = useState('')
  const [draggedId, setDraggedId] = useState<string | null>(null)
  const [dropTargetId, setDropTargetId] = useState<string | null>(null)
  const contextMenuRef = useRef<HTMLDivElement>(null)
  const renameInputRef = useRef<HTMLInputElement>(null)

  // Use controlled or internal expanded state
  const expandedIds = controlledExpandedIds ?? internalExpandedIds
  const setExpandedIds = controlledExpandedIds ? undefined : setInternalExpandedIds

  // Build the tree structure
  const tree = useMemo(() => buildTree(pages), [pages])

  // Close context menu on outside click
  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (contextMenuRef.current && !contextMenuRef.current.contains(e.target as Node)) {
        setContextMenu(null)
      }
    }
    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [])

  // Focus rename input
  useEffect(() => {
    if (renamingId && renameInputRef.current) {
      renameInputRef.current.focus()
      renameInputRef.current.select()
    }
  }, [renamingId])

  const toggleExpand = useCallback((pageId: string) => {
    if (!setExpandedIds) return
    setExpandedIds(prev => {
      const next = new Set(prev)
      if (next.has(pageId)) {
        next.delete(pageId)
      } else {
        next.add(pageId)
      }
      return next
    })
  }, [setExpandedIds])

  const handleContextMenu = useCallback((e: React.MouseEvent, pageId: string) => {
    e.preventDefault()
    e.stopPropagation()
    setContextMenu({ pageId, x: e.clientX, y: e.clientY })
  }, [])

  const handleRenameStart = useCallback((page: Page) => {
    setRenamingId(page.id)
    setRenameValue(page.title || '')
    setContextMenu(null)
  }, [])

  const handleRenameSubmit = useCallback(() => {
    if (renamingId && renameValue.trim()) {
      onRename?.(renamingId, renameValue.trim())
    }
    setRenamingId(null)
    setRenameValue('')
  }, [renamingId, renameValue, onRename])

  const handleRenameKeyDown = useCallback((e: React.KeyboardEvent) => {
    if (e.key === 'Enter') {
      e.preventDefault()
      handleRenameSubmit()
    } else if (e.key === 'Escape') {
      setRenamingId(null)
      setRenameValue('')
    }
  }, [handleRenameSubmit])

  // Drag and drop handlers
  const handleDragStart = useCallback((pageId: string) => {
    if (!allowDragDrop) return
    setDraggedId(pageId)
  }, [allowDragDrop])

  const handleDragOver = useCallback((e: React.DragEvent, pageId: string) => {
    if (!allowDragDrop || !draggedId || draggedId === pageId) return
    e.preventDefault()
    setDropTargetId(pageId)
  }, [allowDragDrop, draggedId])

  const handleDragLeave = useCallback(() => {
    setDropTargetId(null)
  }, [])

  const handleDrop = useCallback((targetPageId: string | null) => {
    if (!allowDragDrop || !draggedId) return
    if (draggedId !== targetPageId) {
      onPageMove?.(draggedId, targetPageId)
    }
    setDraggedId(null)
    setDropTargetId(null)
  }, [allowDragDrop, draggedId, onPageMove])

  const handleDragEnd = useCallback(() => {
    setDraggedId(null)
    setDropTargetId(null)
  }, [])

  const renderNode = (node: TreeNode): React.ReactNode => {
    if (node.depth >= maxDepth) return null

    const { page, children, depth } = node
    const isExpanded = expandedIds.has(page.id)
    const isActive = page.id === currentPageId
    const hasChildren = children.length > 0
    const isRenaming = renamingId === page.id
    const isDragging = draggedId === page.id
    const isDropTarget = dropTargetId === page.id

    return (
      <div key={page.id} className="page-tree-node">
        <div
          className={`page-tree-item ${isActive ? 'active' : ''} ${isDragging ? 'dragging' : ''} ${isDropTarget ? 'drop-target' : ''}`}
          style={{
            display: 'flex',
            alignItems: 'center',
            gap: 4,
            padding: '4px 8px',
            paddingLeft: 8 + depth * 16,
            borderRadius: 'var(--radius-sm)',
            cursor: 'pointer',
            transition: 'background 0.1s ease',
            background: isActive ? 'var(--accent-bg)' : isDropTarget ? 'var(--accent-bg)' : 'transparent',
            opacity: isDragging ? 0.5 : 1,
            borderLeft: isDropTarget ? '2px solid var(--accent-color)' : '2px solid transparent',
          }}
          onClick={() => onPageClick?.(page)}
          onContextMenu={(e) => handleContextMenu(e, page.id)}
          onMouseEnter={(e) => {
            if (!isActive && !isDropTarget) {
              e.currentTarget.style.background = 'var(--bg-hover)'
            }
          }}
          onMouseLeave={(e) => {
            if (!isActive && !isDropTarget) {
              e.currentTarget.style.background = 'transparent'
            }
          }}
          draggable={allowDragDrop && !isRenaming}
          onDragStart={() => handleDragStart(page.id)}
          onDragOver={(e) => handleDragOver(e, page.id)}
          onDragLeave={handleDragLeave}
          onDrop={() => handleDrop(page.id)}
          onDragEnd={handleDragEnd}
        >
          {/* Expand button */}
          <button
            className="expand-btn"
            onClick={(e) => {
              e.stopPropagation()
              toggleExpand(page.id)
            }}
            style={{
              width: 20,
              height: 20,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              background: 'transparent',
              border: 'none',
              borderRadius: 'var(--radius-sm)',
              cursor: 'pointer',
              padding: 0,
              color: 'var(--text-tertiary)',
              visibility: hasChildren ? 'visible' : 'hidden',
              transition: 'transform 0.15s ease, background 0.1s ease',
              transform: isExpanded ? 'rotate(90deg)' : 'rotate(0deg)',
            }}
            onMouseEnter={(e) => e.currentTarget.style.background = 'var(--bg-active)'}
            onMouseLeave={(e) => e.currentTarget.style.background = 'transparent'}
          >
            <ChevronRight size={14} />
          </button>

          {/* Page icon */}
          <span style={{ fontSize: 16, flexShrink: 0 }}>
            {page.icon || <FileText size={16} style={{ color: 'var(--text-tertiary)' }} />}
          </span>

          {/* Page title or rename input */}
          {isRenaming ? (
            <input
              ref={renameInputRef}
              type="text"
              value={renameValue}
              onChange={(e) => setRenameValue(e.target.value)}
              onBlur={handleRenameSubmit}
              onKeyDown={handleRenameKeyDown}
              onClick={(e) => e.stopPropagation()}
              style={{
                flex: 1,
                padding: '2px 4px',
                border: '1px solid var(--accent-color)',
                borderRadius: 'var(--radius-sm)',
                fontSize: 14,
                background: 'var(--bg-primary)',
                color: 'var(--text-primary)',
                outline: 'none',
              }}
            />
          ) : (
            <span
              style={{
                flex: 1,
                fontSize: 14,
                color: isActive ? 'var(--accent-color)' : 'var(--text-primary)',
                overflow: 'hidden',
                textOverflow: 'ellipsis',
                whiteSpace: 'nowrap',
              }}
            >
              {page.title || 'Untitled'}
            </span>
          )}

          {/* Actions on hover */}
          {showActions && !isRenaming && (
            <div
              className="page-actions"
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: 2,
                opacity: 0,
                transition: 'opacity 0.1s ease',
              }}
              onClick={(e) => e.stopPropagation()}
            >
              <button
                onClick={() => onPageCreate?.(page.id)}
                title="Add subpage"
                style={{
                  width: 20,
                  height: 20,
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  background: 'transparent',
                  border: 'none',
                  borderRadius: 'var(--radius-sm)',
                  cursor: 'pointer',
                  color: 'var(--text-tertiary)',
                }}
                onMouseEnter={(e) => {
                  e.currentTarget.style.background = 'var(--bg-active)'
                  e.currentTarget.style.color = 'var(--text-primary)'
                }}
                onMouseLeave={(e) => {
                  e.currentTarget.style.background = 'transparent'
                  e.currentTarget.style.color = 'var(--text-tertiary)'
                }}
              >
                <Plus size={14} />
              </button>
              <button
                onClick={(e) => handleContextMenu(e, page.id)}
                title="More actions"
                style={{
                  width: 20,
                  height: 20,
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  background: 'transparent',
                  border: 'none',
                  borderRadius: 'var(--radius-sm)',
                  cursor: 'pointer',
                  color: 'var(--text-tertiary)',
                }}
                onMouseEnter={(e) => {
                  e.currentTarget.style.background = 'var(--bg-active)'
                  e.currentTarget.style.color = 'var(--text-primary)'
                }}
                onMouseLeave={(e) => {
                  e.currentTarget.style.background = 'transparent'
                  e.currentTarget.style.color = 'var(--text-tertiary)'
                }}
              >
                <MoreHorizontal size={14} />
              </button>
            </div>
          )}
        </div>

        {/* Children */}
        <AnimatePresence initial={false}>
          {isExpanded && hasChildren && (
            <motion.div
              initial={{ height: 0, opacity: 0 }}
              animate={{ height: 'auto', opacity: 1 }}
              exit={{ height: 0, opacity: 0 }}
              transition={{ duration: 0.15, ease: 'easeOut' }}
              style={{ overflow: 'hidden' }}
            >
              {children.map(renderNode)}
            </motion.div>
          )}
        </AnimatePresence>
      </div>
    )
  }

  // Get page for context menu
  const contextMenuPage = contextMenu ? pages.find(p => p.id === contextMenu.pageId) : null

  return (
    <div
      className="page-tree"
      style={{
        position: 'relative',
      }}
      onDragOver={(e) => {
        e.preventDefault()
        // Allow dropping at root level
        if (draggedId && !dropTargetId) {
          setDropTargetId(null)
        }
      }}
      onDrop={() => handleDrop(null)}
    >
      {/* Root level drop zone */}
      {draggedId && (
        <div
          style={{
            height: 4,
            margin: '2px 8px',
            borderRadius: 2,
            background: dropTargetId === null ? 'var(--accent-color)' : 'transparent',
            transition: 'background 0.1s ease',
          }}
        />
      )}

      {/* Tree nodes */}
      {tree.map(renderNode)}

      {/* Empty state */}
      {tree.length === 0 && (
        <div
          style={{
            padding: '12px 16px',
            textAlign: 'center',
            color: 'var(--text-tertiary)',
            fontSize: 13,
          }}
        >
          No pages yet
        </div>
      )}

      {/* Context menu */}
      <AnimatePresence>
        {contextMenu && contextMenuPage && (
          <motion.div
            ref={contextMenuRef}
            initial={{ opacity: 0, scale: 0.95 }}
            animate={{ opacity: 1, scale: 1 }}
            exit={{ opacity: 0, scale: 0.95 }}
            transition={{ duration: 0.1 }}
            className="page-context-menu"
            style={{
              position: 'fixed',
              left: contextMenu.x,
              top: contextMenu.y,
              zIndex: 1000,
              background: 'var(--bg-primary)',
              border: '1px solid var(--border-color)',
              borderRadius: 'var(--radius-md)',
              boxShadow: 'var(--shadow-lg)',
              padding: 6,
              minWidth: 200,
            }}
          >
            <button
              onClick={() => {
                handleRenameStart(contextMenuPage)
              }}
              style={{
                width: '100%',
                display: 'flex',
                alignItems: 'center',
                gap: 8,
                padding: '8px 12px',
                background: 'transparent',
                border: 'none',
                borderRadius: 'var(--radius-sm)',
                cursor: 'pointer',
                fontSize: 14,
                color: 'var(--text-primary)',
                textAlign: 'left',
              }}
              onMouseEnter={(e) => e.currentTarget.style.background = 'var(--bg-hover)'}
              onMouseLeave={(e) => e.currentTarget.style.background = 'transparent'}
            >
              <Edit2 size={16} style={{ color: 'var(--text-secondary)' }} />
              Rename
            </button>

            <button
              onClick={() => {
                onPageDuplicate?.(contextMenu.pageId)
                setContextMenu(null)
              }}
              style={{
                width: '100%',
                display: 'flex',
                alignItems: 'center',
                gap: 8,
                padding: '8px 12px',
                background: 'transparent',
                border: 'none',
                borderRadius: 'var(--radius-sm)',
                cursor: 'pointer',
                fontSize: 14,
                color: 'var(--text-primary)',
                textAlign: 'left',
              }}
              onMouseEnter={(e) => e.currentTarget.style.background = 'var(--bg-hover)'}
              onMouseLeave={(e) => e.currentTarget.style.background = 'transparent'}
            >
              <Copy size={16} style={{ color: 'var(--text-secondary)' }} />
              Duplicate
            </button>

            <button
              onClick={() => {
                onToggleFavorite?.(contextMenu.pageId, !contextMenuPage.is_favorite)
                setContextMenu(null)
              }}
              style={{
                width: '100%',
                display: 'flex',
                alignItems: 'center',
                gap: 8,
                padding: '8px 12px',
                background: 'transparent',
                border: 'none',
                borderRadius: 'var(--radius-sm)',
                cursor: 'pointer',
                fontSize: 14,
                color: 'var(--text-primary)',
                textAlign: 'left',
              }}
              onMouseEnter={(e) => e.currentTarget.style.background = 'var(--bg-hover)'}
              onMouseLeave={(e) => e.currentTarget.style.background = 'transparent'}
            >
              {contextMenuPage.is_favorite ? (
                <>
                  <StarOff size={16} style={{ color: 'var(--text-secondary)' }} />
                  Remove from favorites
                </>
              ) : (
                <>
                  <Star size={16} style={{ color: 'var(--text-secondary)' }} />
                  Add to favorites
                </>
              )}
            </button>

            <button
              onClick={() => {
                navigator.clipboard.writeText(`${window.location.origin}/w/${workspaceSlug}/p/${contextMenu.pageId}`)
                setContextMenu(null)
              }}
              style={{
                width: '100%',
                display: 'flex',
                alignItems: 'center',
                gap: 8,
                padding: '8px 12px',
                background: 'transparent',
                border: 'none',
                borderRadius: 'var(--radius-sm)',
                cursor: 'pointer',
                fontSize: 14,
                color: 'var(--text-primary)',
                textAlign: 'left',
              }}
              onMouseEnter={(e) => e.currentTarget.style.background = 'var(--bg-hover)'}
              onMouseLeave={(e) => e.currentTarget.style.background = 'transparent'}
            >
              <Link2 size={16} style={{ color: 'var(--text-secondary)' }} />
              Copy link
            </button>

            <button
              onClick={() => {
                window.open(`/w/${workspaceSlug}/p/${contextMenu.pageId}`, '_blank')
                setContextMenu(null)
              }}
              style={{
                width: '100%',
                display: 'flex',
                alignItems: 'center',
                gap: 8,
                padding: '8px 12px',
                background: 'transparent',
                border: 'none',
                borderRadius: 'var(--radius-sm)',
                cursor: 'pointer',
                fontSize: 14,
                color: 'var(--text-primary)',
                textAlign: 'left',
              }}
              onMouseEnter={(e) => e.currentTarget.style.background = 'var(--bg-hover)'}
              onMouseLeave={(e) => e.currentTarget.style.background = 'transparent'}
            >
              <ExternalLink size={16} style={{ color: 'var(--text-secondary)' }} />
              Open in new tab
            </button>

            <div style={{ height: 1, background: 'var(--border-color)', margin: '4px 0' }} />

            <button
              onClick={() => {
                onPageDelete?.(contextMenu.pageId)
                setContextMenu(null)
              }}
              style={{
                width: '100%',
                display: 'flex',
                alignItems: 'center',
                gap: 8,
                padding: '8px 12px',
                background: 'transparent',
                border: 'none',
                borderRadius: 'var(--radius-sm)',
                cursor: 'pointer',
                fontSize: 14,
                color: 'var(--danger-color)',
                textAlign: 'left',
              }}
              onMouseEnter={(e) => e.currentTarget.style.background = 'var(--danger-bg)'}
              onMouseLeave={(e) => e.currentTarget.style.background = 'transparent'}
            >
              <Trash2 size={16} />
              Delete
            </button>
          </motion.div>
        )}
      </AnimatePresence>

      {/* Hover show actions CSS */}
      <style>{`
        .page-tree-item:hover .page-actions {
          opacity: 1 !important;
        }
      `}</style>
    </div>
  )
}
