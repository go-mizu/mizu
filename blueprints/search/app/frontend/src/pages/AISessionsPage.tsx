import { useEffect, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import {
  MessageSquare,
  Plus,
  Trash2,
  Settings,
  ArrowLeft,
  Clock,
  Sparkles,
} from 'lucide-react'
import { aiApi } from '../api/ai'
import type { AISession } from '../types/ai'

export default function AISessionsPage() {
  const navigate = useNavigate()
  const [sessions, setSessions] = useState<AISession[]>([])
  const [total, setTotal] = useState(0)
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const loadSessions = async () => {
    try {
      setIsLoading(true)
      const { sessions: data, total: count } = await aiApi.listSessions(50, 0)
      setSessions(data)
      setTotal(count)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load sessions')
    } finally {
      setIsLoading(false)
    }
  }

  useEffect(() => {
    loadSessions()
  }, [])

  const handleCreateSession = async () => {
    try {
      const { session } = await aiApi.createSession()
      navigate(`/ai/session/${session.id}`)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create session')
    }
  }

  const handleDeleteSession = async (id: string, e: React.MouseEvent) => {
    e.preventDefault()
    e.stopPropagation()
    if (!confirm('Delete this session?')) return

    try {
      await aiApi.deleteSession(id)
      setSessions((prev) => prev.filter((s) => s.id !== id))
      setTotal((prev) => prev - 1)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete session')
    }
  }

  const formatDate = (dateStr: string) => {
    const date = new Date(dateStr)
    const now = new Date()
    const diff = now.getTime() - date.getTime()
    const days = Math.floor(diff / (1000 * 60 * 60 * 24))

    if (days === 0) {
      return 'Today'
    } else if (days === 1) {
      return 'Yesterday'
    } else if (days < 7) {
      return `${days} days ago`
    } else {
      return date.toLocaleDateString()
    }
  }

  return (
    <div className="min-h-screen bg-white">
      {/* Header */}
      <header className="sticky top-0 bg-white border-b border-[#e0e0e0] z-50">
        <div className="max-w-4xl mx-auto px-4 py-3">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-4">
              <Link
                to="/"
                className="p-2 text-[#5f6368] hover:bg-[#f1f3f4] rounded-full transition-colors"
              >
                <ArrowLeft size={20} />
              </Link>
              <div className="flex items-center gap-2">
                <Sparkles size={24} className="text-[#1a73e8]" />
                <h1 className="text-xl font-medium text-[#202124]">
                  AI Research Sessions
                </h1>
              </div>
            </div>
            <div className="flex items-center gap-2">
              <button
                type="button"
                onClick={handleCreateSession}
                className="flex items-center gap-2 px-4 py-2 bg-[#1a73e8] text-white rounded-full hover:bg-[#1557b0] transition-colors"
              >
                <Plus size={18} />
                New Session
              </button>
              <Link
                to="/settings"
                className="p-2 text-[#5f6368] hover:bg-[#f1f3f4] rounded-full transition-colors"
              >
                <Settings size={20} />
              </Link>
            </div>
          </div>
        </div>
      </header>

      {/* Content */}
      <main className="max-w-4xl mx-auto px-4 py-6">
        {error && (
          <div className="mb-4 p-4 bg-red-50 text-red-700 rounded-lg">
            {error}
          </div>
        )}

        {isLoading ? (
          <div className="flex justify-center py-12">
            <div className="w-8 h-8 border-4 border-[#1a73e8] border-t-transparent rounded-full animate-spin" />
          </div>
        ) : sessions.length === 0 ? (
          <div className="text-center py-12">
            <MessageSquare size={48} className="mx-auto text-[#9aa0a6] mb-4" />
            <h2 className="text-xl font-medium text-[#202124] mb-2">
              No research sessions yet
            </h2>
            <p className="text-[#5f6368] mb-6">
              Start a new AI research session to explore topics in depth
            </p>
            <button
              type="button"
              onClick={handleCreateSession}
              className="inline-flex items-center gap-2 px-6 py-3 bg-[#1a73e8] text-white rounded-full hover:bg-[#1557b0] transition-colors"
            >
              <Plus size={20} />
              Start New Session
            </button>
          </div>
        ) : (
          <>
            <p className="text-sm text-[#5f6368] mb-4">
              {total} session{total !== 1 ? 's' : ''}
            </p>
            <div className="space-y-2">
              {sessions.map((session) => (
                <Link
                  key={session.id}
                  to={`/ai/session/${session.id}`}
                  className="block p-4 bg-white border border-[#e0e0e0] rounded-lg hover:border-[#1a73e8] hover:shadow-sm transition-all group"
                >
                  <div className="flex items-start justify-between">
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-2 mb-1">
                        <MessageSquare size={16} className="text-[#5f6368]" />
                        <h3 className="font-medium text-[#202124] truncate">
                          {session.title || 'Untitled Session'}
                        </h3>
                      </div>
                      <div className="flex items-center gap-2 text-sm text-[#5f6368]">
                        <Clock size={14} />
                        <span>{formatDate(session.updated_at)}</span>
                      </div>
                    </div>
                    <button
                      type="button"
                      onClick={(e) => handleDeleteSession(session.id, e)}
                      className="p-2 text-[#5f6368] hover:text-red-600 hover:bg-red-50 rounded-full opacity-0 group-hover:opacity-100 transition-all"
                    >
                      <Trash2 size={16} />
                    </button>
                  </div>
                </Link>
              ))}
            </div>
          </>
        )}
      </main>
    </div>
  )
}
