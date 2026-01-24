import { useEffect, useState } from 'react'
import { useParams, Link } from 'react-router-dom'
import {
  ArrowLeft,
  PanelRightOpen,
  PanelRightClose,
  Sparkles,
  Settings,
} from 'lucide-react'
import { aiApi } from '../api/ai'
import { useAIStore } from '../stores/aiStore'
import { AIChat, Canvas } from '../components/ai'
import type { AISession, Canvas as CanvasType } from '../types/ai'

export default function AISessionPage() {
  const { id } = useParams<{ id: string }>()
  const { setCurrentSessionId, currentCanvas, setCurrentCanvas } = useAIStore()

  const [session, setSession] = useState<AISession | null>(null)
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [showCanvas, setShowCanvas] = useState(true)

  useEffect(() => {
    if (!id) return

    const loadSession = async () => {
      try {
        setIsLoading(true)
        setCurrentSessionId(id)

        // Load session and canvas in parallel
        const [sessionRes, canvasRes] = await Promise.allSettled([
          aiApi.getSession(id),
          aiApi.getCanvas(id),
        ])

        if (sessionRes.status === 'fulfilled') {
          setSession(sessionRes.value.session)
        } else {
          throw new Error('Session not found')
        }

        if (canvasRes.status === 'fulfilled') {
          setCurrentCanvas(canvasRes.value.canvas)
        }
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to load session')
      } finally {
        setIsLoading(false)
      }
    }

    loadSession()

    return () => {
      setCurrentSessionId(null)
      setCurrentCanvas(null)
    }
  }, [id])

  const handleAddToCanvas = async (content: string) => {
    if (!id || !currentCanvas) return

    try {
      const { block } = await aiApi.addBlock(
        id,
        'ai_response',
        content,
        currentCanvas.blocks?.length || 0
      )
      setCurrentCanvas({
        ...currentCanvas,
        blocks: [...(currentCanvas.blocks || []), block],
      })
    } catch (err) {
      console.error('Failed to add to canvas:', err)
    }
  }

  const handleCanvasUpdate = (canvas: CanvasType) => {
    setCurrentCanvas(canvas)
  }

  if (isLoading) {
    return (
      <div className="min-h-screen bg-white flex items-center justify-center">
        <div className="w-8 h-8 border-4 border-[#1a73e8] border-t-transparent rounded-full animate-spin" />
      </div>
    )
  }

  if (error || !session) {
    return (
      <div className="min-h-screen bg-white">
        <header className="sticky top-0 bg-white border-b border-[#e0e0e0] z-50">
          <div className="max-w-7xl mx-auto px-4 py-3">
            <div className="flex items-center gap-4">
              <Link
                to="/ai/sessions"
                className="p-2 text-[#5f6368] hover:bg-[#f1f3f4] rounded-full transition-colors"
              >
                <ArrowLeft size={20} />
              </Link>
              <h1 className="text-lg font-medium text-[#202124]">Error</h1>
            </div>
          </div>
        </header>
        <main className="max-w-4xl mx-auto px-4 py-12 text-center">
          <p className="text-red-600 mb-4">{error || 'Session not found'}</p>
          <Link
            to="/ai/sessions"
            className="text-[#1a73e8] hover:underline"
          >
            Back to sessions
          </Link>
        </main>
      </div>
    )
  }

  return (
    <div className="min-h-screen bg-white flex flex-col">
      {/* Header */}
      <header className="sticky top-0 bg-white border-b border-[#e0e0e0] z-50">
        <div className="px-4 py-3">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-4">
              <Link
                to="/ai/sessions"
                className="p-2 text-[#5f6368] hover:bg-[#f1f3f4] rounded-full transition-colors"
              >
                <ArrowLeft size={20} />
              </Link>
              <div className="flex items-center gap-2">
                <Sparkles size={20} className="text-[#1a73e8]" />
                <h1 className="text-lg font-medium text-[#202124]">
                  {session.title || 'Research Session'}
                </h1>
              </div>
            </div>
            <div className="flex items-center gap-2">
              <button
                type="button"
                onClick={() => setShowCanvas(!showCanvas)}
                className="p-2 text-[#5f6368] hover:bg-[#f1f3f4] rounded-full transition-colors"
                title={showCanvas ? 'Hide canvas' : 'Show canvas'}
              >
                {showCanvas ? <PanelRightClose size={20} /> : <PanelRightOpen size={20} />}
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

      {/* Main content */}
      <main className="flex-1 flex overflow-hidden">
        {/* Chat panel */}
        <div className={`flex-1 flex flex-col ${showCanvas ? 'border-r border-[#e0e0e0]' : ''}`}>
          <AIChat
            sessionId={session.id}
            initialMessages={session.messages || []}
            onAddToCanvas={handleAddToCanvas}
          />
        </div>

        {/* Canvas panel */}
        {showCanvas && currentCanvas && (
          <div className="w-96 flex-shrink-0 overflow-auto bg-[#f8f9fa]">
            <Canvas
              canvas={currentCanvas}
              onUpdate={handleCanvasUpdate}
            />
          </div>
        )}
      </main>
    </div>
  )
}
