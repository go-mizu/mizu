import { useEffect, useState, useRef, useCallback } from 'react'
import { useSearchParams, Link, useNavigate } from 'react-router-dom'
import { Settings, Image, Video, Newspaper, Sparkles, Loader2, RefreshCw, Search } from 'lucide-react'
import { SearchBox } from '../components/SearchBox'
import { AIResponse } from '../components/ai/AIResponse'
import { AIModeToggle } from '../components/ai/AIModeToggle'
import { ModelSelector } from '../components/ai/ModelSelector'
import { FollowUpInput } from '../components/ai/FollowUpInput'
import { aiApi } from '../api/ai'
import { useAIStore } from '../stores/aiStore'
import type { AIResponse as AIResponseType, Citation, RelatedQuestion, ImageResult } from '../types/ai'

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
  const [relatedQuestions, setRelatedQuestions] = useState<RelatedQuestion[]>([])
  const [images, setImages] = useState<ImageResult[]>([])

  // Track which query has been processed to prevent duplicates
  const processedQueryRef = useRef<string | null>(null)
  // Abort controller for cancelling in-flight requests
  const abortControllerRef = useRef<AbortController | null>(null)
  // Query version to ignore stale events
  const queryVersionRef = useRef(0)

  // Cancel any in-flight request
  const cancelCurrentRequest = useCallback(() => {
    if (abortControllerRef.current) {
      abortControllerRef.current.abort()
      abortControllerRef.current = null
    }
  }, [])

  // Clean up on unmount
  useEffect(() => {
    return () => {
      cancelCurrentRequest()
    }
  }, [cancelCurrentRequest])

  // Run AI query when page loads with a query
  useEffect(() => {
    // Skip if no query or already processed this query
    if (!query || query === processedQueryRef.current) {
      return
    }

    // Mark as processed
    processedQueryRef.current = query

    // Small delay to ensure state is ready
    const timer = setTimeout(() => {
      runAIQuery(query)
    }, 50)

    return () => clearTimeout(timer)
  }, [query])

  const runAIQuery = async (text: string) => {
    if (!text.trim()) return

    // Cancel any existing request
    cancelCurrentRequest()

    // Increment query version and capture it
    queryVersionRef.current += 1
    const currentVersion = queryVersionRef.current

    // Create new abort controller
    const controller = new AbortController()
    abortControllerRef.current = controller

    // Reset state for new query
    resetStream()
    setLoading(true)
    setError(null)
    setResponse(null)
    setCitations([])
    setRelatedQuestions([])
    setImages([])

    // Helper to check if this query is still current
    const isStale = () => queryVersionRef.current !== currentVersion

    try {
      setStreaming(true)

      const stream = aiApi.queryStreamFetch({
        text,
        mode,
        model_id: selectedModelId || undefined,
      }, controller.signal)

      for await (const event of stream) {
        // Ignore events if a newer query has started
        if (isStale()) break

        switch (event.type) {
          case 'start':
            // Stream started
            break
          case 'search':
            // Searching for query
            if (event.query && !isStale()) {
              addThinkingStep(`Searching for: ${event.query}`)
            }
            break
          case 'token':
            if (event.content && !isStale()) {
              appendStreamContent(event.content)
            }
            break
          case 'thinking':
            if (event.thinking && !isStale()) {
              addThinkingStep(event.thinking)
            }
            break
          case 'citation':
            if (event.citation && !isStale()) {
              setCitations((prev) => [...prev, event.citation!])
            }
            break
          case 'done':
            if (event.response && !isStale()) {
              // Map stream response to AIResponse format
              setResponse({
                text: event.response.text,
                mode: event.response.mode,
                citations: event.response.citations || [],
                follow_ups: event.response.follow_ups || [],
                related_questions: event.response.related_questions || [],
                images: event.response.images || [],
                session_id: event.response.session_id,
                sources_used: event.response.sources_used,
                thinking_steps: streamingThinking,
                usage: event.response.usage,
                provider: event.response.provider,
                model: event.response.model,
                from_cache: event.response.from_cache,
              })
              setRelatedQuestions(event.response.related_questions || [])
              setImages(event.response.images || [])
            }
            break
          case 'error':
            if (!isStale()) {
              setError(event.error || 'An error occurred')
            }
            break
        }
      }
    } catch (err) {
      // Ignore abort errors - they're expected when cancelling
      if (err instanceof Error && err.name === 'AbortError') {
        return
      }
      if (!isStale()) {
        setError(err instanceof Error ? err.message : 'Failed to get AI response')
      }
    } finally {
      // Only update loading state if this is still the current query
      if (!isStale()) {
        setLoading(false)
        setStreaming(false)
      }
      // Clear abort controller if it's still ours
      if (abortControllerRef.current === controller) {
        abortControllerRef.current = null
      }
    }
  }

  const handleSearch = (newQuery: string) => {
    // Cancel any in-flight request
    cancelCurrentRequest()
    // Reset processed query to allow the new query to run
    processedQueryRef.current = null
    setResponse(null)
    setError(null)
    setCitations([])
    setRelatedQuestions([])
    setImages([])
    resetStream()
    setLoading(false)
    setStreaming(false)
    navigate(`/ai?q=${encodeURIComponent(newQuery)}`)
  }

  const handleFollowUp = (question: string) => {
    handleSearch(question)
  }

  const handleRefresh = () => {
    if (query) {
      // Cancel any in-flight request first
      cancelCurrentRequest()
      // Reset to allow re-running the same query
      processedQueryRef.current = null
      setResponse(null)
      setError(null)
      setCitations([])
      setRelatedQuestions([])
      setImages([])
      resetStream()
      runAIQuery(query)
    }
  }

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
                related_questions: relatedQuestions,
                images: images,
                session_id: '',
                sources_used: displayCitations.length,
                thinking_steps: displayThinking,
              }}
              streamingContent={isStreaming ? streamingContent : undefined}
              streamingThinking={isStreaming ? streamingThinking : undefined}
              isStreaming={isStreaming}
              onFollowUp={handleFollowUp}
              onRefresh={handleRefresh}
            />
          </div>
        )}

        {/* Follow-up Input - Perplexity style */}
        {(response || displayContent) && !isStreaming && (
          <FollowUpInput
            onSubmit={handleFollowUp}
            disabled={isLoading || isStreaming}
          />
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
