import { useState, useEffect } from 'react'
import { Link } from 'react-router-dom'
import { Plus, BookOpen } from 'lucide-react'
import Header from '../components/Header'
import { booksApi } from '../api/books'
import type { BookList } from '../types'

export default function ListsPage() {
  const [lists, setLists] = useState<BookList[]>([])
  const [loading, setLoading] = useState(true)
  const [showModal, setShowModal] = useState(false)
  const [title, setTitle] = useState('')
  const [description, setDescription] = useState('')

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

  return (
    <>
      <Header />
      <div className="page-container">
        <div className="section-header">
          <h1 className="section-title">Listopia</h1>
          <button className="btn btn-primary" onClick={() => setShowModal(true)}>
            <Plus size={16} /> Create List
          </button>
        </div>

        {loading ? (
          <div className="loading-spinner"><div className="spinner" /></div>
        ) : lists.length === 0 ? (
          <div className="empty-state">
            <h3>No lists yet</h3>
            <p>Create your first book list to get started.</p>
            <button className="btn btn-primary" onClick={() => setShowModal(true)}>
              Create List
            </button>
          </div>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            {lists.map(list => (
              <Link
                key={list.id}
                to={`/list/${list.id}`}
                className="block p-4 border border-gr-border rounded-lg hover:bg-gr-cream transition-colors no-underline"
              >
                <h3 className="font-serif font-bold text-gr-brown mb-1">{list.title}</h3>
                {list.description && (
                  <p className="text-sm text-gr-light mb-3 line-clamp-2">{list.description}</p>
                )}
                <div className="flex items-center gap-4 text-xs text-gr-light">
                  <span className="flex items-center gap-1"><BookOpen size={12} /> {list.item_count} books</span>
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
              <div className="flex gap-3 justify-end">
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
