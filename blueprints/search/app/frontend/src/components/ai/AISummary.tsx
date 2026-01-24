import { useState, useEffect, useCallback } from 'react'
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

  const runQuery = useCallback(async () => {
    if (!query || !aiAvailable) return

    resetStream()
    setLoading(true)
    setError(null)
    setCurrentResponse(null)

    try {
      setStreaming(true)

      const stream = aiApi.queryStreamFetch({
        text: query,
        mode,
      })

      let response: AIResponseType | null = null

      for await (const event of stream) {
        switch (event.type) {
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
          case 'done':
            if (event.response) {
              response = event.response
            }
            break
          case 'error':
            setError(event.error || 'An error occurred')
            break
        }
      }

      if (response) {
        setCurrentResponse(response)
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to get AI response')
    } finally {
      setLoading(false)
      setStreaming(false)
      setHasQueried(true)
    }
  }, [query, mode, aiAvailable])

  // Auto-run query when query changes
  useEffect(() => {
    if (query && aiAvailable && !hasQueried) {
      runQuery()
    }
  }, [query, aiAvailable, hasQueried, runQuery])

  // Reset when query changes
  useEffect(() => {
    setHasQueried(false)
    resetStream()
    setCurrentResponse(null)
  }, [query])

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
