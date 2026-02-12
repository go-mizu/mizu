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
      .then(list => {
        setLists(prev => [...prev, list])
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
      setLists(prev => [...prev, list])
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
          <div style={{ display: 'flex', gap: 8 }}>
            <button className="btn btn-secondary" onClick={handleBrowseSource}>
              <Download size={16} /> {showSource ? 'Hide' : 'Browse'} Source Lists
            </button>
            <button className="btn btn-primary" onClick={() => setShowModal(true)}>
              <Plus size={16} /> Create List
            </button>
          </div>
        </div>

        {/* Source List Browse Section */}
        {showSource && (
          <div style={{ marginBottom: 24 }}>
            <div style={{ display: 'flex', gap: 8, marginBottom: 10, flexWrap: 'wrap' }}>
              <input
                className="form-input"
                value={tag}
                onChange={e => setTag(e.target.value)}
                placeholder="Optional tag (e.g. fantasy)"
                style={{ maxWidth: 260 }}
              />
              <button className="btn btn-secondary btn-sm" onClick={refreshSource} disabled={loadingSource}>
                Refresh
              </button>
            </div>
            <div style={{ display: 'flex', gap: 8, marginBottom: 12, flexWrap: 'wrap' }}>
              <input
                className="form-input"
                value={manualURL}
                onChange={e => setManualURL(e.target.value)}
                placeholder="Paste source list URL"
                style={{ minWidth: 320, flex: 1 }}
              />
              <button
                className="btn btn-secondary btn-sm"
                onClick={() => handleImportList(manualURL)}
                disabled={!manualURL.trim() || !!importingUrl}
              >
                Import URL
              </button>
            </div>
            <h2 style={{
              fontFamily: "'Merriweather', Georgia, serif",
              fontSize: 16,
              color: 'var(--gr-brown)',
              marginBottom: 12,
            }}>
              Popular Source Lists
            </h2>
            {sourceError && (
              <p style={{ color: '#9b1c1c', fontSize: 13, marginBottom: 10 }}>{sourceError}</p>
            )}
            {importError && (
              <p style={{ color: '#9b1c1c', fontSize: 13, marginBottom: 10 }}>{importError}</p>
            )}
            {loadingSource ? (
              <div className="loading-spinner"><div className="spinner" /></div>
            ) : sourceLists.length === 0 ? (
              <p style={{ color: 'var(--gr-light)', fontSize: 14 }}>No lists found.</p>
            ) : (
              <div style={{
                display: 'grid',
                gridTemplateColumns: 'repeat(auto-fill, minmax(300px, 1fr))',
                gap: 12,
              }}>
                {sourceLists.map((gl, i) => (
                  <div
                    key={i}
                    style={{
                      padding: 16,
                      border: '1px solid var(--gr-border)',
                      borderRadius: 8,
                      display: 'flex',
                      justifyContent: 'space-between',
                      alignItems: 'center',
                      gap: 12,
                    }}
                  >
                    <div style={{ minWidth: 0 }}>
                      <div style={{
                        fontWeight: 700,
                        fontSize: 14,
                        color: 'var(--gr-brown)',
                        marginBottom: 4,
                        overflow: 'hidden',
                        textOverflow: 'ellipsis',
                        whiteSpace: 'nowrap',
                      }}>
                        {gl.title}
                      </div>
                      <div style={{ fontSize: 12, color: 'var(--gr-light)', display: 'flex', gap: 12 }}>
                        {gl.tag && (
                          <span>#{gl.tag}</span>
                        )}
                        {gl.book_count > 0 && (
                          <span style={{ display: 'flex', alignItems: 'center', gap: 3 }}>
                            <BookOpen size={11} /> {gl.book_count.toLocaleString()} books
                          </span>
                        )}
                        {gl.voter_count > 0 && (
                          <span style={{ display: 'flex', alignItems: 'center', gap: 3 }}>
                            <Users size={11} /> {gl.voter_count.toLocaleString()} voters
                          </span>
                        )}
                      </div>
                    </div>
                    <button
                      className="btn btn-secondary btn-sm"
                      onClick={() => handleImportList(gl.url)}
                      disabled={importingUrl === gl.url}
                      style={{ flexShrink: 0 }}
                    >
                      {importingUrl === gl.url ? '...' : 'Import'}
                    </button>
                  </div>
                ))}
              </div>
            )}
          </div>
        )}

        {/* Local Lists */}
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
          <div style={{
            display: 'grid',
            gridTemplateColumns: 'repeat(auto-fill, minmax(320px, 1fr))',
            gap: 12,
          }}>
            {lists.map(list => (
              <Link
                key={list.id}
                to={`/list/${list.id}`}
                style={{
                  display: 'block',
                  padding: 16,
                  border: '1px solid var(--gr-border)',
                  borderRadius: 8,
                  textDecoration: 'none',
                  color: 'inherit',
                  transition: 'background 0.15s',
                }}
                onMouseEnter={(e) => e.currentTarget.style.background = 'var(--gr-cream)'}
                onMouseLeave={(e) => e.currentTarget.style.background = ''}
              >
                <h3 style={{
                  fontFamily: "'Merriweather', Georgia, serif",
                  fontWeight: 700,
                  fontSize: 15,
                  color: 'var(--gr-brown)',
                  margin: '0 0 4px',
                }}>
                  {list.title}
                </h3>
                {list.description && (
                  <p style={{
                    fontSize: 13,
                    color: 'var(--gr-light)',
                    marginBottom: 8,
                    overflow: 'hidden',
                    display: '-webkit-box',
                    WebkitLineClamp: 2,
                    WebkitBoxOrient: 'vertical',
                  }}>
                    {list.description}
                  </p>
                )}
                <div style={{ display: 'flex', alignItems: 'center', gap: 12, fontSize: 12, color: 'var(--gr-light)' }}>
                  <span style={{ display: 'flex', alignItems: 'center', gap: 3 }}>
                    <BookOpen size={12} /> {list.item_count} books
                  </span>
                  {list.voter_count > 0 && (
                    <span style={{ display: 'flex', alignItems: 'center', gap: 3 }}>
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
            <div className="modal" onClick={e => e.stopPropagation()}>
              <h2>Create New List</h2>
              <div className="form-group">
                <label className="form-label">Title</label>
                <input
                  className="form-input"
                  value={title}
                  onChange={e => setTitle(e.target.value)}
                  placeholder="Best Books of 2024"
                />
              </div>
              <div className="form-group">
                <label className="form-label">Description</label>
                <textarea
                  className="form-input"
                  value={description}
                  onChange={e => setDescription(e.target.value)}
                  placeholder="A curated list of..."
                />
              </div>
              <div style={{ display: 'flex', gap: 12, justifyContent: 'flex-end' }}>
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
