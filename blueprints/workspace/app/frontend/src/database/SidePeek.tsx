import { useState, useEffect, useRef, useCallback, useMemo } from 'react'
import { createPortal } from 'react-dom'
import {
  X,
  ChevronUp,
  ChevronDown,
  MoreHorizontal,
  Trash2,
  Copy,
  ExternalLink,
  Plus,
  ChevronDown as ChevronDownIcon,
  MessageSquare,
  Clock,
  History,
  ChevronsRight,
  ChevronsLeft,
  Maximize2,
  GripVertical,
  Check,
  Square,
  Smile,
} from 'lucide-react'
import { api, Database, DatabaseRow, Property, PropertyType, User } from '../api/client'
import { PropertyCell } from './PropertyCell'
import { ConfirmDialog } from '../components/ConfirmDialog'
import showToast from '../utils/toast'
import { format, parseISO, formatDistanceToNow } from 'date-fns'

// ============================================================
// Types
// ============================================================

export type PeekMode = 'side' | 'center' | 'full'

interface Comment {
  id: string
  row_id: string
  user_id: string
  user_name: string
  user_avatar?: string
  content: string
  created_at: string
  updated_at: string
  resolved: boolean
}

interface ContentBlock {
  id: string
  row_id: string
  parent_id?: string
  type: 'heading_1' | 'heading_2' | 'heading_3' | 'paragraph' | 'bulleted_list' | 'numbered_list' | 'to_do' | 'toggle' | 'code' | 'quote' | 'divider' | 'callout'
  content: string
  checked?: boolean
  language?: string
  order: number
}

interface SidePeekProps {
  row: DatabaseRow
  database: Database
  rows: DatabaseRow[]
  currentIndex: number
  isOpen: boolean
  onClose: () => void
  onNavigate: (direction: 'prev' | 'next') => void
  onUpdate: (row: DatabaseRow) => void
  onDelete?: (rowId: string) => void
  onAddProperty?: (property: Omit<Property, 'id'>) => void | Promise<Property | void>
}

// ============================================================
// Property Types List
// ============================================================

const PROPERTY_TYPES: { type: PropertyType; label: string; icon: string }[] = [
  { type: 'text', label: 'Text', icon: 'Aa' },
  { type: 'number', label: 'Number', icon: '#' },
  { type: 'select', label: 'Select', icon: '○' },
  { type: 'multi_select', label: 'Multi-select', icon: '◎' },
  { type: 'status', label: 'Status', icon: '●' },
  { type: 'date', label: 'Date', icon: '📅' },
  { type: 'person', label: 'Person', icon: '👤' },
  { type: 'checkbox', label: 'Checkbox', icon: '☑' },
  { type: 'url', label: 'URL', icon: '🔗' },
  { type: 'email', label: 'Email', icon: '✉' },
  { type: 'phone', label: 'Phone', icon: '📞' },
  { type: 'files', label: 'Files & media', icon: '📎' },
  { type: 'relation', label: 'Relation', icon: '↔' },
  { type: 'rollup', label: 'Rollup', icon: '∑' },
  { type: 'formula', label: 'Formula', icon: 'ƒ' },
  { type: 'created_time', label: 'Created time', icon: '⏱' },
  { type: 'created_by', label: 'Created by', icon: '👤' },
  { type: 'last_edited_time', label: 'Last edited time', icon: '⏱' },
  { type: 'last_edited_by', label: 'Last edited by', icon: '👤' },
]

// ============================================================
// Main SidePeek Component
// ============================================================

export function SidePeek({
  row,
  database,
  rows,
  currentIndex,
  isOpen,
  onClose,
  onNavigate,
  onUpdate,
  onDelete,
  onAddProperty,
}: SidePeekProps) {
  // Local state
  const [localRow, setLocalRow] = useState(row)
  const [localDatabase, setLocalDatabase] = useState(database)
  const [title, setTitle] = useState(row.properties.title as string || 'Untitled')
  const [icon, setIcon] = useState<string | null>(row.properties.icon as string || null)
  const [isSaving, setIsSaving] = useState(false)
  const [showMenu, setShowMenu] = useState(false)
  const [activeTab, setActiveTab] = useState<'properties' | 'comments' | 'activity'>('properties')
  const [showAddProperty, setShowAddProperty] = useState(false)
  const [newPropertyName, setNewPropertyName] = useState('')
  const [newPropertyType, setNewPropertyType] = useState<PropertyType>('text')
  const [showTypeSelect, setShowTypeSelect] = useState(false)
  const [width, setWidth] = useState(500)
  const [isResizing, setIsResizing] = useState(false)
  const [peekMode, setPeekMode] = useState<PeekMode>('side')
  const [showIconPicker, setShowIconPicker] = useState(false)

  // Comments state
  const [comments, setComments] = useState<Comment[]>([])
  const [newComment, setNewComment] = useState('')
  const [isLoadingComments, setIsLoadingComments] = useState(false)

  // Content blocks state
  const [contentBlocks, setContentBlocks] = useState<ContentBlock[]>([])
  const [isLoadingBlocks, setIsLoadingBlocks] = useState(false)
  const [editingBlockId, setEditingBlockId] = useState<string | null>(null)

  // Delete confirmation dialog
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false)
  const [isDeleting, setIsDeleting] = useState(false)

  // Refs
  const panelRef = useRef<HTMLDivElement>(null)
  const menuRef = useRef<HTMLDivElement>(null)
  const addPropertyRef = useRef<HTMLDivElement>(null)
  const resizeRef = useRef<HTMLDivElement>(null)

  // Update local state when row changes
  useEffect(() => {
    setLocalRow(row)
    setTitle(row.properties.title as string || 'Untitled')
    setIcon(row.properties.icon as string || null)
  }, [row])

  // Update local database when database changes
  useEffect(() => {
    setLocalDatabase(database)
  }, [database])

  // Fetch comments and content blocks when row changes
  useEffect(() => {
    if (isOpen && row.id) {
      fetchComments()
      fetchContentBlocks()
    }
  }, [isOpen, row.id])

  // Keyboard navigation
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (!isOpen) return

      if (e.key === 'Escape') {
        onClose()
        return
      }

      // Navigation with arrows or j/k
      if ((e.key === 'ArrowUp' || e.key === 'k') && !e.metaKey && !e.ctrlKey) {
        const target = e.target as HTMLElement
        if (target.tagName !== 'INPUT' && target.tagName !== 'TEXTAREA') {
          e.preventDefault()
          onNavigate('prev')
        }
      }
      if ((e.key === 'ArrowDown' || e.key === 'j') && !e.metaKey && !e.ctrlKey) {
        const target = e.target as HTMLElement
        if (target.tagName !== 'INPUT' && target.tagName !== 'TEXTAREA') {
          e.preventDefault()
          onNavigate('next')
        }
      }
    }

    document.addEventListener('keydown', handleKeyDown)
    return () => document.removeEventListener('keydown', handleKeyDown)
  }, [isOpen, onClose, onNavigate])

  // Close menu on click outside
  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) {
        setShowMenu(false)
      }
      if (addPropertyRef.current && !addPropertyRef.current.contains(e.target as Node)) {
        setShowAddProperty(false)
        setShowTypeSelect(false)
      }
    }
    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [])

  // Handle resize
  useEffect(() => {
    if (!isResizing) return

    const handleMouseMove = (e: MouseEvent) => {
      const newWidth = window.innerWidth - e.clientX
      setWidth(Math.max(400, Math.min(800, newWidth)))
    }

    const handleMouseUp = () => {
      setIsResizing(false)
    }

    document.addEventListener('mousemove', handleMouseMove)
    document.addEventListener('mouseup', handleMouseUp)
    return () => {
      document.removeEventListener('mousemove', handleMouseMove)
      document.removeEventListener('mouseup', handleMouseUp)
    }
  }, [isResizing])

  // ============================================================
  // API Functions
  // ============================================================

  const fetchComments = async () => {
    setIsLoadingComments(true)
    try {
      const data = await api.get<{ comments: Comment[] }>(`/rows/${row.id}/comments`)
      setComments(data.comments || [])
    } catch (err) {
      console.error('Failed to fetch comments:', err)
      // Initialize with empty for demo
      setComments([])
    } finally {
      setIsLoadingComments(false)
    }
  }

  const fetchContentBlocks = async () => {
    setIsLoadingBlocks(true)
    try {
      const data = await api.get<{ blocks: ContentBlock[] }>(`/rows/${row.id}/blocks`)
      setContentBlocks(data.blocks || [])
    } catch (err) {
      console.error('Failed to fetch content blocks:', err)
      // Initialize with sample blocks for demo
      setContentBlocks([
        {
          id: 'demo-1',
          row_id: row.id,
          type: 'heading_2',
          content: 'Description',
          order: 0,
        },
        {
          id: 'demo-2',
          row_id: row.id,
          type: 'paragraph',
          content: 'Provide an overview of the task and related details.',
          order: 1,
        },
        {
          id: 'demo-3',
          row_id: row.id,
          type: 'heading_2',
          content: 'Sub-tasks',
          order: 2,
        },
        {
          id: 'demo-4',
          row_id: row.id,
          type: 'to_do',
          content: 'To-do item 1',
          checked: false,
          order: 3,
        },
        {
          id: 'demo-5',
          row_id: row.id,
          type: 'to_do',
          content: 'To-do item 2',
          checked: false,
          order: 4,
        },
        {
          id: 'demo-6',
          row_id: row.id,
          type: 'to_do',
          content: 'To-do item 3',
          checked: false,
          order: 5,
        },
      ])
    } finally {
      setIsLoadingBlocks(false)
    }
  }

  // ============================================================
  // Event Handlers
  // ============================================================

  // Update title
  const handleTitleChange = useCallback(async (newTitle: string) => {
    setTitle(newTitle)
    const updated = {
      ...localRow,
      properties: { ...localRow.properties, title: newTitle }
    }
    setLocalRow(updated)

    try {
      setIsSaving(true)
      await api.patch(`/rows/${row.id}`, { properties: updated.properties })
      onUpdate(updated)
    } catch (err) {
      console.error('Failed to update row:', err)
    } finally {
      setIsSaving(false)
    }
  }, [localRow, row.id, onUpdate])

  // Update property
  const handlePropertyChange = useCallback(async (propertyId: string, value: unknown) => {
    const updated = {
      ...localRow,
      properties: { ...localRow.properties, [propertyId]: value }
    }
    setLocalRow(updated)

    try {
      setIsSaving(true)
      await api.patch(`/rows/${row.id}`, { properties: updated.properties })
      onUpdate(updated)
    } catch (err) {
      console.error('Failed to update row:', err)
    } finally {
      setIsSaving(false)
    }
  }, [localRow, row.id, onUpdate])

  // Open delete confirmation dialog
  const handleDeleteClick = useCallback(() => {
    setShowMenu(false)
    setShowDeleteConfirm(true)
  }, [])

  // Confirm delete row
  const handleConfirmDelete = useCallback(async () => {
    setIsDeleting(true)
    try {
      await api.delete(`/rows/${row.id}`)
      showToast.success('Row deleted')
      onDelete?.(row.id)
      onClose()
    } catch (err) {
      console.error('Failed to delete row:', err)
      showToast.error('Failed to delete row')
    } finally {
      setIsDeleting(false)
      setShowDeleteConfirm(false)
    }
  }, [row.id, onDelete, onClose])

  // Duplicate row
  const handleDuplicate = useCallback(async () => {
    try {
      const newRow = await api.post<DatabaseRow>(`/databases/${database.id}/rows`, {
        properties: { ...localRow.properties, title: `${title} (copy)` }
      })
      onUpdate(newRow)
      setShowMenu(false)
      showToast.success('Row duplicated')
    } catch (err) {
      console.error('Failed to duplicate row:', err)
      showToast.error('Failed to duplicate row')
    }
  }, [database.id, localRow.properties, title, onUpdate])

  // Add new property
  const handleAddNewProperty = useCallback(async () => {
    if (!newPropertyName.trim()) return

    try {
      setIsSaving(true)

      const newProperty: Omit<Property, 'id'> = {
        name: newPropertyName.trim(),
        type: newPropertyType,
        options: ['select', 'multi_select', 'status'].includes(newPropertyType) ? [] : undefined,
      }

      if (onAddProperty) {
        await onAddProperty(newProperty)
      } else {
        const addedProperty = await api.post<Property>(`/databases/${database.id}/properties`, newProperty)
        setLocalDatabase({
          ...localDatabase,
          properties: [...localDatabase.properties, addedProperty]
        })
      }

      setNewPropertyName('')
      setNewPropertyType('text')
      setShowAddProperty(false)
      setShowTypeSelect(false)
    } catch (err) {
      console.error('Failed to add property:', err)
    } finally {
      setIsSaving(false)
    }
  }, [newPropertyName, newPropertyType, database.id, localDatabase, onAddProperty])

  // Add comment
  const handleAddComment = useCallback(async () => {
    if (!newComment.trim()) return

    try {
      const comment = await api.post<Comment>(`/rows/${row.id}/comments`, {
        content: newComment.trim()
      })
      setComments(prev => [...prev, comment])
      setNewComment('')
    } catch (err) {
      console.error('Failed to add comment:', err)
      // Demo: add local comment
      const demoComment: Comment = {
        id: `demo-${Date.now()}`,
        row_id: row.id,
        user_id: 'demo-user',
        user_name: 'Demo User',
        content: newComment.trim(),
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
        resolved: false,
      }
      setComments(prev => [...prev, demoComment])
      setNewComment('')
    }
  }, [newComment, row.id])

  // Update content block
  const handleUpdateBlock = useCallback(async (blockId: string, updates: Partial<ContentBlock>) => {
    setContentBlocks(prev =>
      prev.map(block =>
        block.id === blockId ? { ...block, ...updates } : block
      )
    )

    try {
      await api.patch(`/blocks/${blockId}`, updates)
    } catch (err) {
      console.error('Failed to update block:', err)
    }
  }, [])

  // Toggle todo
  const handleToggleTodo = useCallback(async (blockId: string, checked: boolean) => {
    handleUpdateBlock(blockId, { checked })
  }, [handleUpdateBlock])

  // Add new block
  const handleAddBlock = useCallback(async (type: ContentBlock['type'], afterBlockId?: string) => {
    const newBlock: ContentBlock = {
      id: `new-${Date.now()}`,
      row_id: row.id,
      type,
      content: '',
      order: contentBlocks.length,
    }

    if (type === 'to_do') {
      newBlock.checked = false
    }

    try {
      const created = await api.post<ContentBlock>(`/rows/${row.id}/blocks`, {
        type,
        content: '',
        after_id: afterBlockId,
      })
      setContentBlocks(prev => [...prev, created])
      setEditingBlockId(created.id)
    } catch (err) {
      console.error('Failed to add block:', err)
      // Demo: add locally
      setContentBlocks(prev => [...prev, newBlock])
      setEditingBlockId(newBlock.id)
    }
  }, [row.id, contentBlocks.length])

  // Get non-title properties
  const otherProperties = useMemo(() =>
    localDatabase.properties.filter(p => p.name.toLowerCase() !== 'title'),
    [localDatabase.properties]
  )

  // Format dates
  const createdAt = row.created_at ? format(parseISO(row.created_at), 'MMM d, yyyy h:mm a') : ''
  const updatedAt = row.updated_at ? format(parseISO(row.updated_at), 'MMM d, yyyy h:mm a') : ''

  // Navigation info
  const hasPrev = currentIndex > 0
  const hasNext = currentIndex < rows.length - 1

  if (!isOpen) return null

  // ============================================================
  // Render
  // ============================================================

  const panelContent = (
    <div
      className="side-peek-panel"
      ref={panelRef}
      style={{
        position: 'fixed',
        top: 0,
        right: 0,
        bottom: 0,
        width: peekMode === 'side' ? width : peekMode === 'center' ? 700 : '100%',
        maxWidth: peekMode === 'full' ? '100%' : 800,
        background: '#ffffff',
        boxShadow: peekMode === 'full' ? 'none' : '-4px 0 24px rgba(0, 0, 0, 0.08)',
        display: 'flex',
        flexDirection: 'column',
        zIndex: 1000,
        transform: isOpen ? 'translateX(0)' : 'translateX(100%)',
        transition: isResizing ? 'none' : 'transform 0.2s ease-out, width 0.2s ease-out',
        ...(peekMode === 'center' && {
          top: '50%',
          left: '50%',
          right: 'auto',
          bottom: 'auto',
          transform: 'translate(-50%, -50%)',
          maxHeight: '90vh',
          borderRadius: 12,
        }),
      }}
    >
      {/* Resize handle */}
      {peekMode === 'side' && (
        <div
          ref={resizeRef}
          onMouseDown={() => setIsResizing(true)}
          style={{
            position: 'absolute',
            left: 0,
            top: 0,
            bottom: 0,
            width: 6,
            cursor: 'col-resize',
            zIndex: 10,
          }}
        >
          <div
            style={{
              position: 'absolute',
              left: 2,
              top: '50%',
              transform: 'translateY(-50%)',
              opacity: isResizing ? 1 : 0,
              transition: 'opacity 0.15s',
            }}
          >
            <GripVertical size={14} style={{ color: '#9a9a97' }} />
          </div>
        </div>
      )}

      {/* Header */}
      <div
        className="side-peek-header"
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          padding: '8px 12px',
          borderBottom: '1px solid #e9e9e7',
          flexShrink: 0,
        }}
      >
        {/* Left controls */}
        <div style={{ display: 'flex', alignItems: 'center', gap: 2 }}>
          {/* Navigation */}
          <button
            onClick={() => onNavigate('prev')}
            disabled={!hasPrev}
            title="Previous row (↑)"
            style={{
              width: 28,
              height: 28,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              background: 'none',
              border: 'none',
              borderRadius: 4,
              cursor: hasPrev ? 'pointer' : 'not-allowed',
              color: hasPrev ? '#37352f' : '#9a9a97',
              opacity: hasPrev ? 1 : 0.5,
            }}
          >
            <ChevronUp size={16} />
          </button>
          <button
            onClick={() => onNavigate('next')}
            disabled={!hasNext}
            title="Next row (↓)"
            style={{
              width: 28,
              height: 28,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              background: 'none',
              border: 'none',
              borderRadius: 4,
              cursor: hasNext ? 'pointer' : 'not-allowed',
              color: hasNext ? '#37352f' : '#9a9a97',
              opacity: hasNext ? 1 : 0.5,
            }}
          >
            <ChevronDown size={16} />
          </button>

          <div style={{ width: 1, height: 16, background: '#e9e9e7', margin: '0 6px' }} />

          {/* Row position indicator */}
          <span style={{ fontSize: 12, color: '#9a9a97' }}>
            {currentIndex + 1} / {rows.length}
          </span>
        </div>

        {/* Right controls */}
        <div style={{ display: 'flex', alignItems: 'center', gap: 2 }}>
          {isSaving && (
            <span style={{ fontSize: 12, color: '#9a9a97', marginRight: 8 }}>Saving...</span>
          )}

          {/* Peek mode toggle */}
          <button
            onClick={() => {
              const modes: PeekMode[] = ['side', 'center', 'full']
              const currentIdx = modes.indexOf(peekMode)
              setPeekMode(modes[(currentIdx + 1) % modes.length])
            }}
            title={`Switch view mode (current: ${peekMode})`}
            style={{
              width: 28,
              height: 28,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              background: 'none',
              border: 'none',
              borderRadius: 4,
              cursor: 'pointer',
              color: '#9a9a97',
            }}
          >
            {peekMode === 'side' && <ChevronsLeft size={16} />}
            {peekMode === 'center' && <Maximize2 size={16} />}
            {peekMode === 'full' && <ChevronsRight size={16} />}
          </button>

          {/* More options */}
          <div ref={menuRef} style={{ position: 'relative' }}>
            <button
              onClick={() => setShowMenu(!showMenu)}
              title="More options"
              style={{
                width: 28,
                height: 28,
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                background: showMenu ? '#f7f6f3' : 'none',
                border: 'none',
                borderRadius: 4,
                cursor: 'pointer',
                color: '#9a9a97',
              }}
            >
              <MoreHorizontal size={16} />
            </button>
            {showMenu && (
              <div
                style={{
                  position: 'absolute',
                  top: '100%',
                  right: 0,
                  marginTop: 4,
                  background: '#ffffff',
                  border: '1px solid #e9e9e7',
                  borderRadius: 8,
                  boxShadow: '0 4px 16px rgba(0,0,0,0.12)',
                  minWidth: 180,
                  zIndex: 100,
                  padding: '4px 0',
                }}
              >
                <button
                  onClick={handleDuplicate}
                  style={{
                    display: 'flex',
                    alignItems: 'center',
                    gap: 10,
                    width: '100%',
                    padding: '8px 14px',
                    background: 'none',
                    border: 'none',
                    cursor: 'pointer',
                    fontSize: 14,
                    textAlign: 'left',
                    color: '#37352f',
                  }}
                >
                  <Copy size={16} style={{ opacity: 0.7 }} />
                  <span>Duplicate</span>
                </button>
                <button
                  onClick={() => window.open(`/p/${row.id}`, '_blank')}
                  style={{
                    display: 'flex',
                    alignItems: 'center',
                    gap: 10,
                    width: '100%',
                    padding: '8px 14px',
                    background: 'none',
                    border: 'none',
                    cursor: 'pointer',
                    fontSize: 14,
                    textAlign: 'left',
                    color: '#37352f',
                  }}
                >
                  <ExternalLink size={16} style={{ opacity: 0.7 }} />
                  <span>Open as page</span>
                </button>
                <div style={{ height: 1, background: '#e9e9e7', margin: '4px 0' }} />
                <button
                  onClick={handleDeleteClick}
                  style={{
                    display: 'flex',
                    alignItems: 'center',
                    gap: 10,
                    width: '100%',
                    padding: '8px 14px',
                    background: 'none',
                    border: 'none',
                    cursor: 'pointer',
                    fontSize: 14,
                    textAlign: 'left',
                    color: '#eb5757',
                  }}
                >
                  <Trash2 size={16} />
                  <span>Delete</span>
                </button>
              </div>
            )}
          </div>

          {/* Close button */}
          <button
            onClick={onClose}
            title="Close (Escape)"
            style={{
              width: 28,
              height: 28,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              background: 'none',
              border: 'none',
              borderRadius: 4,
              cursor: 'pointer',
              color: '#9a9a97',
            }}
          >
            <X size={16} />
          </button>
        </div>
      </div>

      {/* Content */}
      <div
        className="side-peek-content"
        style={{
          flex: 1,
          overflow: 'auto',
          padding: '20px 48px 40px',
        }}
      >
        {/* Icon */}
        <div
          style={{
            marginBottom: 12,
            position: 'relative',
          }}
        >
          <button
            onClick={() => setShowIconPicker(!showIconPicker)}
            style={{
              width: 72,
              height: 72,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              background: '#f7f6f3',
              border: 'none',
              borderRadius: 8,
              cursor: 'pointer',
              fontSize: 32,
            }}
          >
            {icon || '📄'}
          </button>
          {showIconPicker && (
            <EmojiPicker
              onSelect={(emoji) => {
                setIcon(emoji)
                setShowIconPicker(false)
                handlePropertyChange('icon', emoji)
              }}
              onClose={() => setShowIconPicker(false)}
            />
          )}
        </div>

        {/* Title */}
        <div style={{ marginBottom: 20 }}>
          <input
            type="text"
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            onBlur={(e) => handleTitleChange(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter') {
                e.preventDefault()
                handleTitleChange((e.target as HTMLInputElement).value)
              }
            }}
            placeholder="Untitled"
            style={{
              width: '100%',
              fontSize: 32,
              fontWeight: 700,
              border: 'none',
              outline: 'none',
              background: 'transparent',
              padding: 0,
              color: '#37352f',
              lineHeight: 1.2,
            }}
          />
        </div>

        {/* Properties section */}
        <div style={{ marginBottom: 24 }}>
          {otherProperties.map((property) => (
            <div
              key={property.id}
              style={{
                display: 'flex',
                alignItems: 'flex-start',
                gap: 8,
                padding: '6px 0',
                minHeight: 32,
              }}
            >
              <div
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: 6,
                  minWidth: 140,
                  color: '#9a9a97',
                  fontSize: 14,
                  paddingTop: 4,
                }}
              >
                <PropertyIcon type={property.type} />
                <span>{property.name}</span>
              </div>
              <div style={{ flex: 1, minWidth: 0 }}>
                <PropertyCell
                  property={property}
                  value={localRow.properties[property.id]}
                  onChange={(value) => handlePropertyChange(property.id, value)}
                />
              </div>
            </div>
          ))}

          {/* Add property button */}
          <div ref={addPropertyRef} style={{ position: 'relative', marginTop: 8 }}>
            {showAddProperty ? (
              <AddPropertyForm
                newPropertyName={newPropertyName}
                setNewPropertyName={setNewPropertyName}
                newPropertyType={newPropertyType}
                setNewPropertyType={setNewPropertyType}
                showTypeSelect={showTypeSelect}
                setShowTypeSelect={setShowTypeSelect}
                onAdd={handleAddNewProperty}
                onCancel={() => {
                  setShowAddProperty(false)
                  setNewPropertyName('')
                }}
              />
            ) : (
              <button
                onClick={() => setShowAddProperty(true)}
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: 6,
                  padding: '6px 8px',
                  background: 'none',
                  border: 'none',
                  borderRadius: 4,
                  cursor: 'pointer',
                  fontSize: 14,
                  color: '#9a9a97',
                }}
              >
                <Plus size={14} />
                Add a property
              </button>
            )}
          </div>
        </div>

        {/* Divider */}
        <div style={{ height: 1, background: '#e9e9e7', margin: '16px 0' }} />

        {/* Comments section */}
        <div style={{ marginBottom: 24 }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 12 }}>
            <MessageSquare size={16} style={{ color: '#9a9a97' }} />
            <span style={{ fontSize: 14, fontWeight: 500, color: '#37352f' }}>Comments</span>
          </div>

          {/* Comments list */}
          {comments.length > 0 ? (
            <div style={{ marginBottom: 12 }}>
              {comments.map((comment) => (
                <div
                  key={comment.id}
                  style={{
                    display: 'flex',
                    gap: 10,
                    padding: '8px 0',
                  }}
                >
                  <div
                    style={{
                      width: 28,
                      height: 28,
                      borderRadius: '50%',
                      background: '#e9e9e7',
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                      fontSize: 12,
                      fontWeight: 500,
                      color: '#37352f',
                      flexShrink: 0,
                    }}
                  >
                    {comment.user_avatar ? (
                      <img
                        src={comment.user_avatar}
                        alt=""
                        style={{ width: '100%', height: '100%', borderRadius: '50%' }}
                      />
                    ) : (
                      comment.user_name?.charAt(0).toUpperCase() || 'U'
                    )}
                  </div>
                  <div style={{ flex: 1 }}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 2 }}>
                      <span style={{ fontSize: 13, fontWeight: 500, color: '#37352f' }}>
                        {comment.user_name || 'User'}
                      </span>
                      <span style={{ fontSize: 12, color: '#9a9a97' }}>
                        {formatDistanceToNow(parseISO(comment.created_at), { addSuffix: true })}
                      </span>
                    </div>
                    <p style={{ fontSize: 14, color: '#37352f', margin: 0, lineHeight: 1.5 }}>
                      {comment.content}
                    </p>
                  </div>
                </div>
              ))}
            </div>
          ) : null}

          {/* Add comment input */}
          <div style={{ display: 'flex', gap: 10 }}>
            <div
              style={{
                width: 28,
                height: 28,
                borderRadius: '50%',
                background: '#e9e9e7',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                fontSize: 12,
                fontWeight: 500,
                color: '#37352f',
                flexShrink: 0,
              }}
            >
              U
            </div>
            <input
              type="text"
              value={newComment}
              onChange={(e) => setNewComment(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter' && !e.shiftKey) {
                  e.preventDefault()
                  handleAddComment()
                }
              }}
              placeholder="Add a comment..."
              style={{
                flex: 1,
                padding: '8px 12px',
                border: '1px solid #e9e9e7',
                borderRadius: 6,
                fontSize: 14,
                outline: 'none',
              }}
            />
          </div>
        </div>

        {/* Content blocks */}
        <div>
          {contentBlocks.map((block) => (
            <ContentBlockRenderer
              key={block.id}
              block={block}
              isEditing={editingBlockId === block.id}
              onStartEdit={() => setEditingBlockId(block.id)}
              onEndEdit={() => setEditingBlockId(null)}
              onUpdate={(updates) => handleUpdateBlock(block.id, updates)}
              onToggleTodo={(checked) => handleToggleTodo(block.id, checked)}
            />
          ))}

          {/* Add block button */}
          <button
            onClick={() => handleAddBlock('paragraph')}
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: 6,
              padding: '8px 0',
              background: 'none',
              border: 'none',
              cursor: 'pointer',
              fontSize: 14,
              color: '#9a9a97',
              width: '100%',
              textAlign: 'left',
            }}
          >
            <Plus size={14} />
            Add a block
          </button>
        </div>

        {/* Timestamps */}
        <div style={{ marginTop: 32, paddingTop: 16, borderTop: '1px solid #e9e9e7' }}>
          {createdAt && (
            <div
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: 8,
                fontSize: 12,
                color: '#9a9a97',
                marginBottom: 4,
              }}
            >
              <Clock size={12} />
              <span>Created {createdAt}</span>
            </div>
          )}
          {updatedAt && updatedAt !== createdAt && (
            <div
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: 8,
                fontSize: 12,
                color: '#9a9a97',
              }}
            >
              <Clock size={12} />
              <span>Last edited {updatedAt}</span>
            </div>
          )}
        </div>
      </div>
    </div>
  )

  // Render based on peek mode
  if (peekMode === 'center') {
    return createPortal(
      <>
        <div
          onClick={onClose}
          style={{
            position: 'fixed',
            top: 0,
            left: 0,
            right: 0,
            bottom: 0,
            background: 'rgba(0, 0, 0, 0.5)',
            zIndex: 999,
          }}
        />
        {panelContent}

        {/* Delete confirmation dialog */}
        <ConfirmDialog
          isOpen={showDeleteConfirm}
          onClose={() => setShowDeleteConfirm(false)}
          onConfirm={handleConfirmDelete}
          title="Delete row?"
          message="This action cannot be undone. The row and all its data will be permanently deleted."
          confirmText="Delete"
          cancelText="Cancel"
          variant="danger"
          isLoading={isDeleting}
        />
      </>,
      document.body
    )
  }

  return createPortal(
    <>
      {/* Backdrop for side peek (optional click to close) */}
      <div
        onClick={onClose}
        style={{
          position: 'fixed',
          top: 0,
          left: 0,
          right: 0,
          bottom: 0,
          background: peekMode === 'side' ? 'rgba(0, 0, 0, 0.15)' : 'transparent',
          zIndex: 999,
          pointerEvents: peekMode === 'full' ? 'none' : 'auto',
        }}
      />
      {panelContent}

      {/* Delete confirmation dialog */}
      <ConfirmDialog
        isOpen={showDeleteConfirm}
        onClose={() => setShowDeleteConfirm(false)}
        onConfirm={handleConfirmDelete}
        title="Delete row?"
        message="This action cannot be undone. The row and all its data will be permanently deleted."
        confirmText="Delete"
        cancelText="Cancel"
        variant="danger"
        isLoading={isDeleting}
      />
    </>,
    document.body
  )
}

// ============================================================
// Helper Components
// ============================================================

function PropertyIcon({ type }: { type: string }) {
  const icons: Record<string, string> = {
    text: 'Aa',
    number: '#',
    select: '○',
    multi_select: '◎',
    date: '📅',
    person: '👤',
    checkbox: '☑',
    url: '🔗',
    email: '✉',
    phone: '📞',
    files: '📎',
    relation: '↔',
    rollup: '∑',
    formula: 'ƒ',
    status: '●',
    created_time: '⏱',
    created_by: '👤',
    last_edited_time: '⏱',
    last_edited_by: '👤',
  }

  return <span style={{ fontSize: 14 }}>{icons[type] || '•'}</span>
}

interface AddPropertyFormProps {
  newPropertyName: string
  setNewPropertyName: (name: string) => void
  newPropertyType: PropertyType
  setNewPropertyType: (type: PropertyType) => void
  showTypeSelect: boolean
  setShowTypeSelect: (show: boolean) => void
  onAdd: () => void
  onCancel: () => void
}

function AddPropertyForm({
  newPropertyName,
  setNewPropertyName,
  newPropertyType,
  setNewPropertyType,
  showTypeSelect,
  setShowTypeSelect,
  onAdd,
  onCancel,
}: AddPropertyFormProps) {
  return (
    <div
      style={{
        background: '#f7f6f3',
        borderRadius: 8,
        padding: 12,
        border: '1px solid #e9e9e7',
      }}
    >
      <input
        type="text"
        value={newPropertyName}
        onChange={(e) => setNewPropertyName(e.target.value)}
        placeholder="Property name"
        autoFocus
        onKeyDown={(e) => {
          if (e.key === 'Enter' && newPropertyName.trim()) {
            onAdd()
          }
          if (e.key === 'Escape') {
            onCancel()
          }
        }}
        style={{
          width: '100%',
          padding: '8px 12px',
          border: '1px solid #e9e9e7',
          borderRadius: 4,
          fontSize: 14,
          marginBottom: 8,
          outline: 'none',
          background: '#fff',
        }}
      />

      <div style={{ position: 'relative', marginBottom: 12 }}>
        <button
          onClick={() => setShowTypeSelect(!showTypeSelect)}
          style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            width: '100%',
            padding: '8px 12px',
            background: '#fff',
            border: '1px solid #e9e9e7',
            borderRadius: 4,
            cursor: 'pointer',
            fontSize: 14,
          }}
        >
          <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
            <span>{PROPERTY_TYPES.find((t) => t.type === newPropertyType)?.icon}</span>
            <span>{PROPERTY_TYPES.find((t) => t.type === newPropertyType)?.label}</span>
          </div>
          <ChevronDownIcon size={14} />
        </button>
        {showTypeSelect && (
          <div
            style={{
              position: 'absolute',
              top: '100%',
              left: 0,
              right: 0,
              marginTop: 4,
              background: '#fff',
              border: '1px solid #e9e9e7',
              borderRadius: 8,
              boxShadow: '0 4px 12px rgba(0,0,0,0.1)',
              maxHeight: 300,
              overflowY: 'auto',
              zIndex: 100,
            }}
          >
            {PROPERTY_TYPES.map(({ type, label, icon }) => (
              <button
                key={type}
                onClick={() => {
                  setNewPropertyType(type)
                  setShowTypeSelect(false)
                }}
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: 8,
                  width: '100%',
                  padding: '8px 12px',
                  background: type === newPropertyType ? '#f7f6f3' : 'none',
                  border: 'none',
                  cursor: 'pointer',
                  fontSize: 14,
                  textAlign: 'left',
                }}
              >
                <span style={{ width: 20, textAlign: 'center' }}>{icon}</span>
                <span>{label}</span>
              </button>
            ))}
          </div>
        )}
      </div>

      <div style={{ display: 'flex', gap: 8 }}>
        <button
          onClick={onAdd}
          disabled={!newPropertyName.trim()}
          style={{
            flex: 1,
            padding: '8px 12px',
            background: '#2383e2',
            color: 'white',
            border: 'none',
            borderRadius: 4,
            cursor: newPropertyName.trim() ? 'pointer' : 'not-allowed',
            fontSize: 14,
            opacity: newPropertyName.trim() ? 1 : 0.5,
          }}
        >
          Add property
        </button>
        <button
          onClick={onCancel}
          style={{
            padding: '8px 12px',
            background: 'none',
            border: '1px solid #e9e9e7',
            borderRadius: 4,
            cursor: 'pointer',
            fontSize: 14,
          }}
        >
          Cancel
        </button>
      </div>
    </div>
  )
}

interface ContentBlockRendererProps {
  block: ContentBlock
  isEditing: boolean
  onStartEdit: () => void
  onEndEdit: () => void
  onUpdate: (updates: Partial<ContentBlock>) => void
  onToggleTodo: (checked: boolean) => void
}

function ContentBlockRenderer({
  block,
  isEditing,
  onStartEdit,
  onEndEdit,
  onUpdate,
  onToggleTodo,
}: ContentBlockRendererProps) {
  const [localContent, setLocalContent] = useState(block.content)

  useEffect(() => {
    setLocalContent(block.content)
  }, [block.content])

  const handleBlur = () => {
    if (localContent !== block.content) {
      onUpdate({ content: localContent })
    }
    onEndEdit()
  }

  const baseStyle: React.CSSProperties = {
    width: '100%',
    border: 'none',
    outline: 'none',
    background: 'transparent',
    padding: '4px 0',
    resize: 'none',
    fontFamily: 'inherit',
  }

  switch (block.type) {
    case 'heading_1':
      return (
        <div style={{ marginTop: 24, marginBottom: 4 }}>
          {isEditing ? (
            <input
              type="text"
              value={localContent}
              onChange={(e) => setLocalContent(e.target.value)}
              onBlur={handleBlur}
              autoFocus
              style={{ ...baseStyle, fontSize: 24, fontWeight: 700, color: '#37352f' }}
            />
          ) : (
            <h1
              onClick={onStartEdit}
              style={{
                fontSize: 24,
                fontWeight: 700,
                color: '#37352f',
                margin: 0,
                cursor: 'text',
              }}
            >
              {block.content || 'Heading 1'}
            </h1>
          )}
        </div>
      )

    case 'heading_2':
      return (
        <div style={{ marginTop: 20, marginBottom: 4 }}>
          {isEditing ? (
            <input
              type="text"
              value={localContent}
              onChange={(e) => setLocalContent(e.target.value)}
              onBlur={handleBlur}
              autoFocus
              style={{ ...baseStyle, fontSize: 20, fontWeight: 600, color: '#37352f' }}
            />
          ) : (
            <h2
              onClick={onStartEdit}
              style={{
                fontSize: 20,
                fontWeight: 600,
                color: '#37352f',
                margin: 0,
                cursor: 'text',
              }}
            >
              {block.content || 'Heading 2'}
            </h2>
          )}
        </div>
      )

    case 'heading_3':
      return (
        <div style={{ marginTop: 16, marginBottom: 4 }}>
          {isEditing ? (
            <input
              type="text"
              value={localContent}
              onChange={(e) => setLocalContent(e.target.value)}
              onBlur={handleBlur}
              autoFocus
              style={{ ...baseStyle, fontSize: 16, fontWeight: 600, color: '#37352f' }}
            />
          ) : (
            <h3
              onClick={onStartEdit}
              style={{
                fontSize: 16,
                fontWeight: 600,
                color: '#37352f',
                margin: 0,
                cursor: 'text',
              }}
            >
              {block.content || 'Heading 3'}
            </h3>
          )}
        </div>
      )

    case 'to_do':
      return (
        <div style={{ display: 'flex', alignItems: 'flex-start', gap: 8, padding: '4px 0' }}>
          <button
            onClick={() => onToggleTodo(!block.checked)}
            style={{
              width: 18,
              height: 18,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              background: block.checked ? '#2383e2' : 'none',
              border: block.checked ? 'none' : '2px solid #c4c4c2',
              borderRadius: 3,
              cursor: 'pointer',
              marginTop: 2,
              flexShrink: 0,
            }}
          >
            {block.checked && <Check size={12} color="white" strokeWidth={3} />}
          </button>
          {isEditing ? (
            <input
              type="text"
              value={localContent}
              onChange={(e) => setLocalContent(e.target.value)}
              onBlur={handleBlur}
              autoFocus
              style={{
                ...baseStyle,
                fontSize: 14,
                color: block.checked ? '#9a9a97' : '#37352f',
                textDecoration: block.checked ? 'line-through' : 'none',
                flex: 1,
              }}
            />
          ) : (
            <span
              onClick={onStartEdit}
              style={{
                fontSize: 14,
                color: block.checked ? '#9a9a97' : '#37352f',
                textDecoration: block.checked ? 'line-through' : 'none',
                cursor: 'text',
                flex: 1,
                lineHeight: 1.5,
              }}
            >
              {block.content || 'To-do'}
            </span>
          )}
        </div>
      )

    case 'bulleted_list':
      return (
        <div style={{ display: 'flex', alignItems: 'flex-start', gap: 8, padding: '4px 0' }}>
          <span style={{ color: '#37352f', marginTop: 2 }}>•</span>
          {isEditing ? (
            <input
              type="text"
              value={localContent}
              onChange={(e) => setLocalContent(e.target.value)}
              onBlur={handleBlur}
              autoFocus
              style={{ ...baseStyle, fontSize: 14, color: '#37352f', flex: 1 }}
            />
          ) : (
            <span
              onClick={onStartEdit}
              style={{ fontSize: 14, color: '#37352f', cursor: 'text', flex: 1, lineHeight: 1.5 }}
            >
              {block.content || 'List item'}
            </span>
          )}
        </div>
      )

    case 'numbered_list':
      return (
        <div style={{ display: 'flex', alignItems: 'flex-start', gap: 8, padding: '4px 0' }}>
          <span style={{ color: '#37352f', marginTop: 2, minWidth: 16 }}>{block.order + 1}.</span>
          {isEditing ? (
            <input
              type="text"
              value={localContent}
              onChange={(e) => setLocalContent(e.target.value)}
              onBlur={handleBlur}
              autoFocus
              style={{ ...baseStyle, fontSize: 14, color: '#37352f', flex: 1 }}
            />
          ) : (
            <span
              onClick={onStartEdit}
              style={{ fontSize: 14, color: '#37352f', cursor: 'text', flex: 1, lineHeight: 1.5 }}
            >
              {block.content || 'List item'}
            </span>
          )}
        </div>
      )

    case 'quote':
      return (
        <div
          style={{
            borderLeft: '3px solid #37352f',
            paddingLeft: 16,
            margin: '8px 0',
          }}
        >
          {isEditing ? (
            <textarea
              value={localContent}
              onChange={(e) => setLocalContent(e.target.value)}
              onBlur={handleBlur}
              autoFocus
              style={{
                ...baseStyle,
                fontSize: 14,
                color: '#37352f',
                fontStyle: 'italic',
                minHeight: 40,
              }}
            />
          ) : (
            <p
              onClick={onStartEdit}
              style={{
                fontSize: 14,
                color: '#37352f',
                fontStyle: 'italic',
                margin: 0,
                cursor: 'text',
                lineHeight: 1.5,
              }}
            >
              {block.content || 'Quote'}
            </p>
          )}
        </div>
      )

    case 'divider':
      return <hr style={{ border: 'none', borderTop: '1px solid #e9e9e7', margin: '16px 0' }} />

    case 'callout':
      return (
        <div
          style={{
            display: 'flex',
            alignItems: 'flex-start',
            gap: 12,
            padding: 16,
            background: '#f7f6f3',
            borderRadius: 4,
            margin: '8px 0',
          }}
        >
          <span style={{ fontSize: 20 }}>💡</span>
          {isEditing ? (
            <textarea
              value={localContent}
              onChange={(e) => setLocalContent(e.target.value)}
              onBlur={handleBlur}
              autoFocus
              style={{ ...baseStyle, fontSize: 14, color: '#37352f', flex: 1, minHeight: 40 }}
            />
          ) : (
            <p
              onClick={onStartEdit}
              style={{ fontSize: 14, color: '#37352f', margin: 0, cursor: 'text', flex: 1, lineHeight: 1.5 }}
            >
              {block.content || 'Callout'}
            </p>
          )}
        </div>
      )

    case 'code':
      return (
        <div style={{ margin: '8px 0' }}>
          <pre
            style={{
              background: '#f7f6f3',
              padding: 16,
              borderRadius: 4,
              overflow: 'auto',
              margin: 0,
            }}
          >
            {isEditing ? (
              <textarea
                value={localContent}
                onChange={(e) => setLocalContent(e.target.value)}
                onBlur={handleBlur}
                autoFocus
                style={{
                  ...baseStyle,
                  fontSize: 13,
                  fontFamily: 'ui-monospace, SFMono-Regular, Menlo, Monaco, monospace',
                  color: '#37352f',
                  minHeight: 80,
                  width: '100%',
                }}
              />
            ) : (
              <code
                onClick={onStartEdit}
                style={{
                  fontSize: 13,
                  fontFamily: 'ui-monospace, SFMono-Regular, Menlo, Monaco, monospace',
                  color: '#37352f',
                  cursor: 'text',
                  display: 'block',
                  whiteSpace: 'pre-wrap',
                }}
              >
                {block.content || 'Code'}
              </code>
            )}
          </pre>
        </div>
      )

    case 'paragraph':
    default:
      return (
        <div style={{ padding: '4px 0' }}>
          {isEditing ? (
            <textarea
              value={localContent}
              onChange={(e) => setLocalContent(e.target.value)}
              onBlur={handleBlur}
              autoFocus
              style={{ ...baseStyle, fontSize: 14, color: '#37352f', minHeight: 24 }}
              rows={1}
            />
          ) : (
            <p
              onClick={onStartEdit}
              style={{
                fontSize: 14,
                color: block.content ? '#37352f' : '#9a9a97',
                margin: 0,
                cursor: 'text',
                lineHeight: 1.5,
                minHeight: 21,
              }}
            >
              {block.content || 'Type something...'}
            </p>
          )}
        </div>
      )
  }
}

interface EmojiPickerProps {
  onSelect: (emoji: string) => void
  onClose: () => void
}

function EmojiPicker({ onSelect, onClose }: EmojiPickerProps) {
  const emojis = [
    '📄', '📝', '📋', '📌', '📍', '🎯', '✅', '❌', '⭐', '🔥',
    '💡', '💎', '🎨', '🎬', '🎤', '🎵', '📸', '🎁', '🎉', '🎊',
    '🏠', '🏢', '🏭', '🏫', '🏥', '🚀', '✈️', '🚗', '🚌', '🚲',
    '📱', '💻', '🖥️', '⌨️', '🖱️', '📞', '📧', '💼', '📁', '📂',
    '🔧', '🔨', '⚙️', '🔩', '🔑', '🔒', '🔓', '📐', '📏', '✂️',
    '👤', '👥', '👨‍💻', '👩‍💻', '🤖', '👽', '🐱', '🐶', '🦊', '🐻',
  ]

  return (
    <>
      <div
        onClick={onClose}
        style={{
          position: 'fixed',
          top: 0,
          left: 0,
          right: 0,
          bottom: 0,
          zIndex: 1000,
        }}
      />
      <div
        style={{
          position: 'absolute',
          top: '100%',
          left: 0,
          marginTop: 4,
          background: '#fff',
          border: '1px solid #e9e9e7',
          borderRadius: 8,
          boxShadow: '0 4px 16px rgba(0,0,0,0.12)',
          padding: 8,
          zIndex: 1001,
          width: 280,
        }}
      >
        <div
          style={{
            display: 'grid',
            gridTemplateColumns: 'repeat(10, 1fr)',
            gap: 2,
          }}
        >
          {emojis.map((emoji) => (
            <button
              key={emoji}
              onClick={() => onSelect(emoji)}
              style={{
                width: 24,
                height: 24,
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                background: 'none',
                border: 'none',
                borderRadius: 4,
                cursor: 'pointer',
                fontSize: 16,
              }}
            >
              {emoji}
            </button>
          ))}
        </div>
      </div>
    </>
  )
}

export default SidePeek
