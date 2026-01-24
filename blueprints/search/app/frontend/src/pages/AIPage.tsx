import { useEffect, useState, useRef, useCallback } from 'react'
import { useSearchParams, Link, useNavigate } from 'react-router-dom'
import { Settings, Image, Video, Newspaper, Sparkles, Loader2, Send, RefreshCw, Search } from 'lucide-react'
import { SearchBox } from '../components/SearchBox'
import { AIResponse } from '../components/ai/AIResponse'
import { AIModeToggle } from '../components/ai/AIModeToggle'
import { ModelSelector } from '../components/ai/ModelSelector'
import { FileUploadZone, type UploadedFile } from '../components/ai/FileUploadZone'
import { VoiceInput } from '../components/ai/VoiceInput'
import { aiApi } from '../api/ai'
import { useAIStore } from '../stores/aiStore'
import type { AIResponse as AIResponseType, Citation } from '../types/ai'

export default function AIPage() {
  const [searchParams] = useSearchParams()
  const navigate = useNavigate()
  const query = searchParams.get('q') || ''

  const {
    mode,
    selectedModelId,
    setSelectedModelId,
    isLoading,
    isStreaming,
    streamingContent,
    streamingThinking,
    setLoading,
    setStreaming,
    appendStreamContent,
    addThinkingStep,
    resetStream,
    error,
    setError,
  } = useAIStore()

  const [response, setResponse] = useState<AIResponseType | null>(null)
  const [citations, setCitations] = useState<Citation[]>([])
  const [files, setFiles] = useState<UploadedFile[]>([])
  const [followUpInput, setFollowUpInput] = useState('')
  const [interimTranscript, setInterimTranscript] = useState('')
  const followUpRef = useRef<HTMLTextAreaElement>(null)

  // Run AI query when page loads with a query
  useEffect(() => {
    if (query && !isLoading && !isStreaming && !response) {
      runAIQuery(query)
    }
  }, [query])

  const runAIQuery = async (text: string) => {
    if (!text.trim() || isLoading || isStreaming) return

    resetStream()
    setLoading(true)
    setError(null)
    setResponse(null)
    setCitations([])

    try {
      setStreaming(true)

      const stream = aiApi.queryStreamFetch({
        text,
        mode,
        model_id: selectedModelId || undefined,
      })

      for await (const event of stream) {
        switch (event.type) {
          case 'start':
            // Stream started
            break
          case 'search':
            // Searching for query
            if (event.query) {
              addThinkingStep(`Searching for: ${event.query}`)
            }
            break
          case 'token':
            if (event.content) {
              appendStreamContent(event.content)
            }
            break
          case 'thinking':
            if (event.thinking) {
              addThinkingStep(event.thinking)
            }
            break
          case 'citation':
            if (event.citation) {
              setCitations((prev) => [...prev, event.citation!])
            }
            break
          case 'done':
            if (event.response) {
              // Map stream response to AIResponse format
              setResponse({
                text: event.response.text,
                mode: event.response.mode,
                citations: event.response.citations || [],
                follow_ups: event.response.follow_ups || [],
                session_id: event.response.session_id,
                sources_used: event.response.sources_used,
                thinking_steps: streamingThinking,
              })
            }
            break
          case 'error':
            setError(event.error || 'An error occurred')
            break
        }
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to get AI response')
    } finally {
      setLoading(false)
      setStreaming(false)
    }
  }

  const handleSearch = (newQuery: string) => {
    navigate(`/ai?q=${encodeURIComponent(newQuery)}`)
    setResponse(null)
    resetStream()
  }

  const handleFollowUp = (question: string) => {
    handleSearch(question)
  }

  const handleFollowUpSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    if (!followUpInput.trim()) return
    handleSearch(followUpInput)
    setFollowUpInput('')
  }

  const handleRefresh = () => {
    if (query) {
      setResponse(null)
      resetStream()
      runAIQuery(query)
    }
  }

  const handleVoiceTranscript = useCallback((text: string) => {
    setFollowUpInput((prev) => prev + (prev ? ' ' : '') + text)
    setInterimTranscript('')
  }, [])

  const handleInterimTranscript = useCallback((text: string) => {
    setInterimTranscript(text)
  }, [])

  const displayContent = streamingContent || response?.text || ''
  const displayThinking = streamingThinking.length > 0 ? streamingThinking : response?.thinking_steps || []
  const displayCitations = response?.citations || citations

  return (
    <div className="min-h-screen bg-white">
      {/* Header */}
      <header className="sticky top-0 bg-white z-50 border-b border-gray-200">
        <div className="max-w-7xl mx-auto px-4 py-3">
          <div className="flex items-center gap-6">
            {/* Logo */}
            <Link to="/">
              <span
                className="text-3xl font-bold"
                style={{
                  background: 'linear-gradient(90deg, #4285F4, #EA4335, #FBBC05, #34A853)',
                  WebkitBackgroundClip: 'text',
                  WebkitTextFillColor: 'transparent',
                }}
              >
                Search
              </span>
            </Link>

            {/* Search box */}
            <div className="flex-1 max-w-xl">
              <SearchBox
                initialValue={query}
                size="sm"
                onSearch={handleSearch}
              />
            </div>

            {/* AI Sessions */}
            <Link
              to="/ai/sessions"
              className="p-2 text-[#5f6368] hover:bg-[#f1f3f4] rounded-full transition-colors"
              title="AI Research Sessions"
            >
              <Sparkles size={20} />
            </Link>

            {/* Settings */}
            <Link
              to="/settings"
              className="p-2 text-[#5f6368] hover:bg-[#f1f3f4] rounded-full transition-colors"
            >
              <Settings size={20} />
            </Link>
          </div>

          {/* Tabs */}
          <div className="search-tabs mt-2" style={{ paddingLeft: 0 }}>
            <button
              type="button"
              className="search-tab"
              onClick={() => navigate(`/search?q=${encodeURIComponent(query)}`)}
            >
              <Search size={16} />
              All
            </button>
            <button
              type="button"
              className="search-tab active"
            >
              <Sparkles size={16} />
              AI
            </button>
            <button
              type="button"
              className="search-tab"
              onClick={() => navigate(`/images?q=${encodeURIComponent(query)}`)}
            >
              <Image size={16} />
              Images
            </button>
            <button
              type="button"
              className="search-tab"
              onClick={() => navigate(`/videos?q=${encodeURIComponent(query)}`)}
            >
              <Video size={16} />
              Videos
            </button>
            <button
              type="button"
              className="search-tab"
              onClick={() => navigate(`/news?q=${encodeURIComponent(query)}`)}
            >
              <Newspaper size={16} />
              News
            </button>
          </div>
        </div>
      </header>

      {/* Main Content */}
      <main className="max-w-4xl mx-auto px-4 py-6">
        {/* Mode and Model Selection */}
        <div className="flex items-center gap-4 mb-6">
          <AIModeToggle />
          <ModelSelector
            selectedModel={selectedModelId || undefined}
            onSelectModel={setSelectedModelId}
          />
          {(response || isStreaming) && (
            <button
              type="button"
              onClick={handleRefresh}
              className="ai-action-button ml-auto"
              disabled={isLoading || isStreaming}
            >
              <RefreshCw size={14} className={isLoading ? 'animate-spin' : ''} />
              Regenerate
            </button>
          )}
        </div>

        {/* Query Display */}
        {query && (
          <div className="ai-query-display mb-6">
            <h1 className="text-2xl font-medium text-gray-900">{query}</h1>
          </div>
        )}

        {/* Error */}
        {error && (
          <div className="ai-error-banner mb-6">
            <span className="text-red-600">{error}</span>
            <button
              type="button"
              onClick={handleRefresh}
              className="text-blue-600 hover:underline ml-2"
            >
              Try again
            </button>
          </div>
        )}

        {/* Loading State */}
        {isLoading && !displayContent && (
          <div className="ai-loading-state">
            <Loader2 size={32} className="animate-spin text-blue-500 mb-4" />
            <p className="text-gray-600">Analyzing with AI...</p>
          </div>
        )}

        {/* AI Response */}
        {(displayContent || response) && (
          <div className="ai-main-response">
            <AIResponse
              response={response ?? {
                text: displayContent,
                mode: mode,
                citations: displayCitations,
                follow_ups: [],
                session_id: '',
                sources_used: displayCitations.length,
                thinking_steps: displayThinking,
              }}
              streamingContent={isStreaming ? streamingContent : undefined}
              streamingThinking={isStreaming ? streamingThinking : undefined}
              isStreaming={isStreaming}
              onFollowUp={handleFollowUp}
            />
          </div>
        )}

        {/* Follow-up Input */}
        {(response || displayContent) && !isStreaming && (
          <div className="ai-followup-section mt-8">
            <h3 className="text-sm font-medium text-gray-700 mb-3">Ask a follow-up question</h3>
            <FileUploadZone files={files} onFilesChange={setFiles}>
              <form onSubmit={handleFollowUpSubmit} className="ai-followup-form">
                <div className="ai-followup-input-wrapper">
                  <textarea
                    ref={followUpRef}
                    value={followUpInput + (interimTranscript ? ` ${interimTranscript}` : '')}
                    onChange={(e) => setFollowUpInput(e.target.value)}
                    placeholder="Ask a follow-up question..."
                    className="ai-followup-input"
                    rows={1}
                  />
                </div>
                <div className="ai-followup-actions">
                  <VoiceInput
                    onTranscript={handleVoiceTranscript}
                    onInterimTranscript={handleInterimTranscript}
                    size="sm"
                  />
                  <button
                    type="submit"
                    disabled={!followUpInput.trim()}
                    className="ai-followup-submit"
                  >
                    <Send size={18} />
                  </button>
                </div>
              </form>
            </FileUploadZone>
          </div>
        )}

        {/* Empty State */}
        {!query && (
          <div className="ai-empty-state">
            <Sparkles size={48} className="text-blue-500 mb-4" />
            <h2 className="text-xl font-medium text-gray-900 mb-2">AI-Powered Search</h2>
            <p className="text-gray-600 mb-6">
              Get comprehensive, AI-generated answers with citations from multiple sources.
            </p>
            <div className="ai-suggestion-chips">
              <button
                type="button"
                onClick={() => handleSearch('What are the latest developments in AI?')}
                className="ai-suggestion-chip"
              >
                Latest AI developments
              </button>
              <button
                type="button"
                onClick={() => handleSearch('Compare React vs Vue for web development')}
                className="ai-suggestion-chip"
              >
                React vs Vue comparison
              </button>
              <button
                type="button"
                onClick={() => handleSearch('Best practices for Go backend development')}
                className="ai-suggestion-chip"
              >
                Go backend best practices
              </button>
            </div>
          </div>
        )}
      </main>
    </div>
  )
}
