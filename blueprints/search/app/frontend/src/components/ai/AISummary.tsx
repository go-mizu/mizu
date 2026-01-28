import { useState, useEffect, useCallback, useRef } from 'react'
import { Sparkles, ChevronDown, ChevronUp, RefreshCw } from 'lucide-react'
import { aiApi } from '../../api/ai'
import { useAIStore } from '../../stores/aiStore'
import { AIResponse } from './AIResponse'
import { AIModeToggle } from './AIModeToggle'
import type { AIResponse as AIResponseType } from '../../types/ai'

interface AISummaryProps {
  query: string
  onFollowUp?: (question: string) => void
}

export function AISummary({ query, onFollowUp }: AISummaryProps) {
  const {
    mode,
    aiAvailable,
    isLoading,
    isStreaming,
    streamingContent,
    streamingThinking,
    currentResponse,
    error,
    setLoading,
    setStreaming,
    appendStreamContent,
    addThinkingStep,
    resetStream,
    setCurrentResponse,
    setError,
  } = useAIStore()

  const [isExpanded, setIsExpanded] = useState(true)
  const [hasQueried, setHasQueried] = useState(false)

  // Abort controller for cancelling in-flight requests
  const abortControllerRef = useRef<AbortController | null>(null)
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

  const runQuery = useCallback(async () => {
    if (!query || !aiAvailable) return

    // Cancel any existing request
    cancelCurrentRequest()

    // Increment query version and capture it
    queryVersionRef.current += 1
    const currentVersion = queryVersionRef.current

    // Create new abort controller
    const controller = new AbortController()
    abortControllerRef.current = controller

    // Helper to check if this query is still current
    const isStale = () => queryVersionRef.current !== currentVersion

    resetStream()
    setLoading(true)
    setError(null)
    setCurrentResponse(null)

    try {
      setStreaming(true)

      const stream = aiApi.queryStreamFetch({
        text: query,
        mode,
      }, controller.signal)

      let response: AIResponseType | null = null

      for await (const event of stream) {
        // Ignore events if a newer query has started
        if (isStale()) break

        switch (event.type) {
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
          case 'done':
            if (event.response && !isStale()) {
              response = event.response
            }
            break
          case 'error':
            if (!isStale()) {
              setError(event.error || 'An error occurred')
            }
            break
        }
      }

      if (response && !isStale()) {
        setCurrentResponse(response)
      }
    } catch (err) {
      // Ignore abort errors
      if (err instanceof Error && err.name === 'AbortError') {
        return
      }
      if (!isStale()) {
        setError(err instanceof Error ? err.message : 'Failed to get AI response')
      }
    } finally {
      if (!isStale()) {
        setLoading(false)
        setStreaming(false)
        setHasQueried(true)
      }
      if (abortControllerRef.current === controller) {
        abortControllerRef.current = null
      }
    }
  }, [query, mode, aiAvailable, cancelCurrentRequest])

  // Auto-run query when query changes
  useEffect(() => {
    if (query && aiAvailable && !hasQueried) {
      runQuery()
    }
  }, [query, aiAvailable, hasQueried, runQuery])

  // Reset when query changes
  useEffect(() => {
    cancelCurrentRequest()
    setHasQueried(false)
    resetStream()
    setCurrentResponse(null)
  }, [query, cancelCurrentRequest])

  if (!aiAvailable) {
    return null
  }

  const handleRefresh = () => {
    runQuery()
  }

  const handleModeChange = () => {
    // Re-run query with new mode
    runQuery()
  }

  // Create a temporary response for streaming display
  const displayResponse: AIResponseType = currentResponse || {
    text: streamingContent,
    mode,
    citations: [],
    follow_ups: [],
    session_id: '',
    sources_used: 0,
    thinking_steps: streamingThinking,
  }

  return (
    <div className="ai-summary">
      {/* Header */}
      <div className="ai-summary-header">
        <button
          type="button"
          onClick={() => setIsExpanded(!isExpanded)}
          className="ai-summary-toggle"
        >
          <Sparkles size={18} className="ai-summary-icon" />
          <span className="ai-summary-title">AI Overview</span>
          {isExpanded ? <ChevronUp size={16} /> : <ChevronDown size={16} />}
        </button>

        <div className="ai-summary-controls">
          <AIModeToggle size="sm" onChange={handleModeChange} />
          <button
            type="button"
            onClick={handleRefresh}
            disabled={isLoading || isStreaming}
            className="ai-refresh-button"
            title="Regenerate"
          >
            <RefreshCw size={14} className={isLoading ? 'animate-spin' : ''} />
          </button>
        </div>
      </div>

      {/* Content */}
      {isExpanded && (
        <div className="ai-summary-content">
          {error ? (
            <div className="ai-error">
              <p>{error}</p>
              <button type="button" onClick={handleRefresh}>
                Try again
              </button>
            </div>
          ) : isLoading && !streamingContent ? (
            <div className="ai-loading">
              <div className="ai-loading-dots">
                <span />
                <span />
                <span />
              </div>
              <p>Generating AI overview...</p>
            </div>
          ) : (displayResponse.text || streamingContent) ? (
            <AIResponse
              response={displayResponse}
              streamingContent={isStreaming ? streamingContent : undefined}
              streamingThinking={isStreaming ? streamingThinking : undefined}
              isStreaming={isStreaming}
              onFollowUp={onFollowUp}
            />
          ) : null}
        </div>
      )}
    </div>
  )
}
