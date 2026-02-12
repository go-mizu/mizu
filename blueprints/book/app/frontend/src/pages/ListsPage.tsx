import { useState, useEffect } from 'react'
import { Link } from 'react-router-dom'
import { Plus, BookOpen, Download, Users } from 'lucide-react'
import Header from '../components/Header'
import { booksApi } from '../api/books'
import type { BookList, SourceListSummary } from '../types'
import { ApiError } from '../api/client'

export default function ListsPage() {
  const [lists, setLists] = useState<BookList[]>([])
  const [loading, setLoading] = useState(true)
  const [showModal, setShowModal] = useState(false)
  const [title, setTitle] = useState('')
  const [description, setDescription] = useState('')
  const [sourceLists, setSourceLists] = useState<SourceListSummary[]>([])
  const [loadingSource, setLoadingSource] = useState(false)
  const [showSource, setShowSource] = useState(false)
  const [importingUrl, setImportingUrl] = useState<string | null>(null)
  const [sourceError, setSourceError] = useState('')
  const [importError, setImportError] = useState('')
  const [tag, setTag] = useState('')
  const [manualURL, setManualURL] = useState('')

  useEffect(() => {
    booksApi.getLists()
      .then(setLists)
      .catch(() => {})
      .finally(() => setLoading(false))
  }, [])

  const handleCreate = () => {
    if (!title.trim()) return
    booksApi.createList({ title, description })
      .then((list) => {
        setLists((prev) => [...prev, list])
        setShowModal(false)
        setTitle('')
        setDescription('')
      })
      .catch(() => {})
  }

  const handleBrowseSource = () => {
    setSourceError('')
    if (sourceLists.length > 0) {
      setShowSource(!showSource)
      return
    }
    setLoadingSource(true)
    setShowSource(true)
    booksApi.browseSourceLists(tag)
      .then(setSourceLists)
      .catch((err: unknown) => {
        if (err instanceof ApiError) {
          setSourceError(err.message)
        } else {
          setSourceError('Failed to browse source lists')
        }
      })
      .finally(() => setLoadingSource(false))
  }

  const refreshSource = () => {
    setSourceLists([])
    handleBrowseSource()
  }

  const handleImportList = async (url: string) => {
    const normalizedURL = url.trim()
    if (!normalizedURL) return
    setImportError('')
    setImportingUrl(normalizedURL)
    try {
      const list = await booksApi.importSourceList(normalizedURL)
      setLists((prev) => [...prev, list])
      if (manualURL.trim() === normalizedURL) {
        setManualURL('')
      }
    } catch (err: unknown) {
      if (err instanceof ApiError) {
        setImportError(err.message)
      } else {
        setImportError('Failed to import source list')
      }
    } finally {
      setImportingUrl(null)
    }
  }

  return (
    <>
      <Header />
      <div className="page-container">
        <div className="section-header">
          <h1 className="section-title">Listopia</h1>
          <div className="header-actions">
            <button className="btn btn-secondary" onClick={handleBrowseSource}>
              <Download size={16} /> {showSource ? 'Hide' : 'Browse'} Source Lists
            </button>
            <button className="btn btn-primary" onClick={() => setShowModal(true)}>
              <Plus size={16} /> Create List
            </button>
          </div>
        </div>

        {showSource && (
          <section className="source-lists-panel">
            <div className="source-input-row">
              <input
                className="form-input source-tag-input"
                value={tag}
                onChange={(e) => setTag(e.target.value)}
                placeholder="Optional tag (e.g. fantasy)"
              />
              <button className="btn btn-secondary btn-sm" onClick={refreshSource} disabled={loadingSource}>
                Refresh
              </button>
            </div>
            <div className="source-input-row">
              <input
                className="form-input source-url-input"
                value={manualURL}
                onChange={(e) => setManualURL(e.target.value)}
                placeholder="Paste source list URL"
              />
              <button
                className="btn btn-secondary btn-sm"
                onClick={() => handleImportList(manualURL)}
                disabled={!manualURL.trim() || !!importingUrl}
              >
                Import URL
              </button>
            </div>
            <h2 className="source-section-title">Popular Source Lists</h2>
            {sourceError && (
              <p className="form-error">{sourceError}</p>
            )}
            {importError && (
              <p className="form-error">{importError}</p>
            )}
            {loadingSource ? (
              <div className="loading-spinner"><div className="spinner" /></div>
            ) : sourceLists.length === 0 ? (
              <p className="page-subtitle">No lists found.</p>
            ) : (
              <div className="list-grid">
                {sourceLists.map((gl, i) => (
                  <div key={i} className="source-list-card">
                    <div className="source-list-info">
                      <div className="source-list-title">{gl.title}</div>
                      <div className="source-list-meta">
                        {gl.tag && <span>#{gl.tag}</span>}
                        {gl.book_count > 0 && (
                          <span className="source-meta-item">
                            <BookOpen size={11} /> {gl.book_count.toLocaleString()} books
                          </span>
                        )}
                        {gl.voter_count > 0 && (
                          <span className="source-meta-item">
                            <Users size={11} /> {gl.voter_count.toLocaleString()} voters
                          </span>
                        )}
                      </div>
                    </div>
                    <button
                      className="btn btn-secondary btn-sm"
                      onClick={() => handleImportList(gl.url)}
                      disabled={importingUrl === gl.url}
                    >
                      {importingUrl === gl.url ? '...' : 'Import'}
                    </button>
                  </div>
                ))}
              </div>
            )}
          </section>
        )}

        {loading ? (
          <div className="loading-spinner"><div className="spinner" /></div>
        ) : lists.length === 0 ? (
          <div className="empty-state">
            <h3>No lists yet</h3>
            <p>Create your first book list or import from a source list URL.</p>
            <button className="btn btn-primary" onClick={() => setShowModal(true)}>
              Create List
            </button>
          </div>
        ) : (
          <div className="list-grid">
            {lists.map((list) => (
              <Link key={list.id} to={`/list/${list.id}`} className="local-list-card">
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
                </div>
              </Link>
            ))}
          </div>
        )}

        {showModal && (
          <div className="modal-overlay" onClick={() => setShowModal(false)}>
            <div className="modal" onClick={(e) => e.stopPropagation()}>
              <h2>Create New List</h2>
              <div className="form-group">
                <label className="form-label">Title</label>
                <input
                  className="form-input"
                  value={title}
                  onChange={(e) => setTitle(e.target.value)}
                  placeholder="Best Books of 2024"
                />
              </div>
              <div className="form-group">
                <label className="form-label">Description</label>
                <textarea
                  className="form-input"
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                  placeholder="A curated list of..."
                />
              </div>
              <div className="modal-actions">
                <button className="btn btn-secondary" onClick={() => setShowModal(false)}>Cancel</button>
                <button className="btn btn-primary" onClick={handleCreate}>Create</button>
              </div>
            </div>
          </div>
        )}
      </div>
    </>
  )
}
