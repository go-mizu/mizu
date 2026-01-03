import { useState, useEffect, useCallback } from 'react'
import { Trash2, RotateCcw, X, Search, AlertTriangle, FileText, Database } from 'lucide-react'
import { api, Page } from '../api/client'
import { format, parseISO, formatDistanceToNow } from 'date-fns'

interface TrashViewProps {
  workspaceId: string
  onRestore: (page: Page) => void
  onClose: () => void
}

export function TrashView({ workspaceId, onRestore, onClose }: TrashViewProps) {
  const [pages, setPages] = useState<Page[]>([])
  const [loading, setLoading] = useState(true)
  const [search, setSearch] = useState('')
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set())
  const [deleting, setDeleting] = useState<string | null>(null)

  // Load archived pages
  useEffect(() => {
    const loadTrash = async () => {
      setLoading(true)
      try {
        const result = await api.get<{ pages: Page[] }>(`/workspaces/${workspaceId}/trash`)
        setPages(result.pages || [])
      } catch (err) {
        console.error('Failed to load trash:', err)
        setPages([])
      } finally {
        setLoading(false)
      }
    }

    loadTrash()
  }, [workspaceId])

  // Filter pages by search
  const filteredPages = pages.filter(page =>
    page.title.toLowerCase().includes(search.toLowerCase())
  )

  // Restore a page
  const handleRestore = useCallback(async (pageId: string) => {
    try {
      await api.post(`/pages/${pageId}/restore`)
      const page = pages.find(p => p.id === pageId)
      if (page) {
        setPages(pages.filter(p => p.id !== pageId))
        onRestore(page)
      }
    } catch (err) {
      console.error('Failed to restore page:', err)
    }
  }, [pages, onRestore])

  // Delete permanently
  const handleDelete = useCallback(async (pageId: string) => {
    if (!confirm('Are you sure? This action cannot be undone.')) return

    setDeleting(pageId)
    try {
      await api.delete(`/pages/${pageId}?permanent=true`)
      setPages(pages.filter(p => p.id !== pageId))
    } catch (err) {
      console.error('Failed to delete page:', err)
    } finally {
      setDeleting(null)
    }
  }, [pages])

  // Restore selected
  const handleRestoreSelected = useCallback(async () => {
    for (const id of selectedIds) {
      await handleRestore(id)
    }
    setSelectedIds(new Set())
  }, [selectedIds, handleRestore])

  // Delete selected
  const handleDeleteSelected = useCallback(async () => {
    if (!confirm(`Delete ${selectedIds.size} pages permanently? This action cannot be undone.`)) return

    for (const id of selectedIds) {
      try {
        await api.delete(`/pages/${id}?permanent=true`)
      } catch (err) {
        console.error('Failed to delete page:', err)
      }
    }
    setPages(pages.filter(p => !selectedIds.has(p.id)))
    setSelectedIds(new Set())
  }, [selectedIds, pages])

  // Empty trash
  const handleEmptyTrash = useCallback(async () => {
    if (!confirm('Empty trash? All pages will be permanently deleted. This action cannot be undone.')) return

    try {
      await api.delete(`/workspaces/${workspaceId}/trash`)
      setPages([])
    } catch (err) {
      console.error('Failed to empty trash:', err)
    }
  }, [workspaceId])

  // Toggle selection
  const toggleSelection = (id: string) => {
    const next = new Set(selectedIds)
    if (next.has(id)) {
      next.delete(id)
    } else {
      next.add(id)
    }
    setSelectedIds(next)
  }

  // Select all
  const selectAll = () => {
    if (selectedIds.size === filteredPages.length) {
      setSelectedIds(new Set())
    } else {
      setSelectedIds(new Set(filteredPages.map(p => p.id)))
    }
  }

  return (
    <div className="trash-view">
      {/* Header */}
      <div className="trash-header">
        <div className="trash-title">
          <Trash2 size={20} />
          <h2>Trash</h2>
        </div>
        <button className="icon-btn" onClick={onClose}>
          <X size={20} />
        </button>
      </div>

      {/* Toolbar */}
      <div className="trash-toolbar">
        <div className="search-wrapper">
          <Search size={16} />
          <input
            type="text"
            placeholder="Search trash..."
            value={search}
            onChange={e => setSearch(e.target.value)}
          />
        </div>

        <div className="trash-actions">
          {selectedIds.size > 0 && (
            <>
              <button className="btn-secondary" onClick={handleRestoreSelected}>
                <RotateCcw size={14} />
                <span>Restore ({selectedIds.size})</span>
              </button>
              <button className="btn-danger" onClick={handleDeleteSelected}>
                <Trash2 size={14} />
                <span>Delete ({selectedIds.size})</span>
              </button>
            </>
          )}
          {pages.length > 0 && (
            <button className="btn-danger" onClick={handleEmptyTrash}>
              Empty Trash
            </button>
          )}
        </div>
      </div>

      {/* Content */}
      <div className="trash-content">
        {loading ? (
          <div className="loading-state">
            <div className="spinner" />
            <span>Loading...</span>
          </div>
        ) : pages.length === 0 ? (
          <div className="empty-state">
            <Trash2 size={48} />
            <h3>Trash is empty</h3>
            <p>Pages you delete will appear here.</p>
          </div>
        ) : filteredPages.length === 0 ? (
          <div className="empty-state">
            <Search size={48} />
            <h3>No results</h3>
            <p>No pages match your search.</p>
          </div>
        ) : (
          <>
            {/* Select all */}
            <div className="trash-select-all">
              <label>
                <input
                  type="checkbox"
                  checked={selectedIds.size === filteredPages.length && filteredPages.length > 0}
                  onChange={selectAll}
                />
                <span>Select all ({filteredPages.length})</span>
              </label>
            </div>

            {/* List */}
            <div className="trash-list">
              {filteredPages.map(page => (
                <div
                  key={page.id}
                  className={`trash-item ${selectedIds.has(page.id) ? 'selected' : ''}`}
                >
                  <div className="trash-item-checkbox">
                    <input
                      type="checkbox"
                      checked={selectedIds.has(page.id)}
                      onChange={() => toggleSelection(page.id)}
                    />
                  </div>

                  <div className="trash-item-icon">
                    {page.icon || (page.parent_type === 'database' ? <Database size={16} /> : <FileText size={16} />)}
                  </div>

                  <div className="trash-item-info">
                    <span className="trash-item-title">{page.title || 'Untitled'}</span>
                    <span className="trash-item-meta">
                      Deleted {formatDistanceToNow(parseISO(page.updated_at), { addSuffix: true })}
                    </span>
                  </div>

                  <div className="trash-item-actions">
                    <button
                      className="btn-secondary small"
                      onClick={() => handleRestore(page.id)}
                      title="Restore"
                    >
                      <RotateCcw size={14} />
                    </button>
                    <button
                      className="btn-danger small"
                      onClick={() => handleDelete(page.id)}
                      disabled={deleting === page.id}
                      title="Delete permanently"
                    >
                      <Trash2 size={14} />
                    </button>
                  </div>
                </div>
              ))}
            </div>
          </>
        )}
      </div>

      {/* Warning */}
      <div className="trash-warning">
        <AlertTriangle size={14} />
        <span>Pages in trash will be permanently deleted after 30 days.</span>
      </div>
    </div>
  )
}
