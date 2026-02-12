import { useState, useEffect } from 'react'
import { Link } from 'react-router-dom'
import { Plus, BookOpen, Download, Users } from 'lucide-react'
import Header from '../components/Header'
import { booksApi } from '../api/books'
import type { BookList, GoodreadsListSummary } from '../types'

export default function ListsPage() {
  const [lists, setLists] = useState<BookList[]>([])
  const [loading, setLoading] = useState(true)
  const [showModal, setShowModal] = useState(false)
  const [title, setTitle] = useState('')
  const [description, setDescription] = useState('')
  const [grLists, setGrLists] = useState<GoodreadsListSummary[]>([])
  const [loadingGr, setLoadingGr] = useState(false)
  const [showGr, setShowGr] = useState(false)
  const [importingUrl, setImportingUrl] = useState<string | null>(null)

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

  const handleBrowseGoodreads = () => {
    if (grLists.length > 0) {
      setShowGr(!showGr)
      return
    }
    setLoadingGr(true)
    setShowGr(true)
    booksApi.browseGoodreadsLists()
      .then(setGrLists)
      .catch(() => {})
      .finally(() => setLoadingGr(false))
  }

  const handleImportList = async (url: string) => {
    setImportingUrl(url)
    try {
      const list = await booksApi.importGoodreadsList(url)
      setLists(prev => [...prev, list])
    } catch {
      // Silently fail
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
            <button className="btn btn-secondary" onClick={handleBrowseGoodreads}>
              <Download size={16} /> {showGr ? 'Hide' : 'Browse'} Goodreads
            </button>
            <button className="btn btn-primary" onClick={() => setShowModal(true)}>
              <Plus size={16} /> Create List
            </button>
          </div>
        </div>

        {/* Goodreads Browse Section */}
        {showGr && (
          <div style={{ marginBottom: 24 }}>
            <h2 style={{
              fontFamily: "'Merriweather', Georgia, serif",
              fontSize: 16,
              color: 'var(--gr-brown)',
              marginBottom: 12,
            }}>
              Popular Goodreads Lists
            </h2>
            {loadingGr ? (
              <div className="loading-spinner"><div className="spinner" /></div>
            ) : grLists.length === 0 ? (
              <p style={{ color: 'var(--gr-light)', fontSize: 14 }}>No lists found.</p>
            ) : (
              <div style={{
                display: 'grid',
                gridTemplateColumns: 'repeat(auto-fill, minmax(300px, 1fr))',
                gap: 12,
              }}>
                {grLists.map((gl, i) => (
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
            <p>Create your first book list or import from Goodreads.</p>
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
