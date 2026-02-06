import { useEffect, useState } from 'react'
import { X, ExternalLink, Loader2 } from 'lucide-react'
import ReactMarkdown from 'react-markdown'
import { searchApi } from '../api/search'

interface ReaderViewProps {
  url: string
  onClose: () => void
}

interface ReaderData {
  title: string
  url: string
  description: string
  content: string
}

export function ReaderView({ url, onClose }: ReaderViewProps) {
  const [data, setData] = useState<ReaderData | null>(null)
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    const load = async () => {
      setIsLoading(true)
      setError(null)
      try {
        const result = await searchApi.readPage(url)
        setData(result)
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to load page')
      } finally {
        setIsLoading(false)
      }
    }
    load()
  }, [url])

  // Close on Escape
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose()
    }
    document.addEventListener('keydown', handleKeyDown)
    document.body.style.overflow = 'hidden'
    return () => {
      document.removeEventListener('keydown', handleKeyDown)
      document.body.style.overflow = ''
    }
  }, [onClose])

  const domain = (() => {
    try {
      return new URL(url).hostname
    } catch {
      return url
    }
  })()

  return (
    <>
      {/* Backdrop */}
      <div
        className="fixed inset-0 bg-black/30 z-40"
        onClick={onClose}
      />

      {/* Panel */}
      <div className="fixed inset-y-0 right-0 w-full max-w-2xl bg-white shadow-2xl z-50 flex flex-col">
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-[#e8eaed] flex-shrink-0">
          <div className="min-w-0 flex-1 mr-4">
            {data ? (
              <>
                <h2 className="text-base font-medium text-[#202124] truncate">
                  {data.title}
                </h2>
                <p className="text-xs text-[#70757a] truncate mt-0.5">{domain}</p>
              </>
            ) : (
              <div className="h-5 w-48 bg-[#f1f3f4] rounded animate-pulse" />
            )}
          </div>
          <div className="flex items-center gap-1 flex-shrink-0">
            <a
              href={url}
              target="_blank"
              rel="noopener noreferrer"
              className="p-2 text-[#5f6368] hover:bg-[#f1f3f4] rounded-full transition-colors"
              title="Open original"
            >
              <ExternalLink size={18} />
            </a>
            <button
              type="button"
              onClick={onClose}
              className="p-2 text-[#5f6368] hover:bg-[#f1f3f4] rounded-full transition-colors"
            >
              <X size={18} />
            </button>
          </div>
        </div>

        {/* Content */}
        <div className="flex-1 overflow-y-auto px-6 py-6">
          {isLoading ? (
            <div className="flex flex-col items-center justify-center py-16">
              <Loader2 size={32} className="text-[#1a73e8] animate-spin mb-4" />
              <p className="text-sm text-[#5f6368]">Reading page...</p>
            </div>
          ) : error ? (
            <div className="text-center py-16">
              <p className="text-sm text-[#d93025] mb-4">{error}</p>
              <a
                href={url}
                target="_blank"
                rel="noopener noreferrer"
                className="inline-flex items-center gap-1.5 px-4 py-2 text-sm font-medium bg-[#1a73e8] text-white rounded-lg hover:bg-[#1557b0] transition-colors"
              >
                <ExternalLink size={14} />
                Open original page
              </a>
            </div>
          ) : data ? (
            <article className="prose prose-sm max-w-none prose-headings:text-[#202124] prose-p:text-[#3c4043] prose-a:text-[#1a73e8] prose-strong:text-[#202124] prose-code:text-[#202124] prose-code:bg-[#f1f3f4] prose-code:px-1 prose-code:py-0.5 prose-code:rounded prose-code:text-sm prose-pre:bg-[#f8f9fa] prose-pre:border prose-pre:border-[#e8eaed]">
              <ReactMarkdown>{data.content}</ReactMarkdown>
            </article>
          ) : null}
        </div>
      </div>
    </>
  )
}
