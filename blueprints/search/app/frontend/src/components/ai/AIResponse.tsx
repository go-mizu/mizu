import { useState } from 'react'
import { Copy, Check, BookOpen } from 'lucide-react'
import type { AIResponse as AIResponseType } from '../../types/ai'
import { CitationChip } from './CitationChip'
import { FollowUpChips } from './FollowUpChips'
import { ThinkingSteps } from './ThinkingSteps'

interface AIResponseProps {
  response: AIResponseType
  streamingContent?: string
  streamingThinking?: string[]
  isStreaming?: boolean
  onFollowUp?: (question: string) => void
  onAddToCanvas?: () => void
}

export function AIResponse({
  response,
  streamingContent,
  streamingThinking = [],
  isStreaming = false,
  onFollowUp,
  onAddToCanvas,
}: AIResponseProps) {
  const [copied, setCopied] = useState(false)
  const [expandedCitation, setExpandedCitation] = useState<number | null>(null)

  const content = streamingContent || response.text
  const thinkingSteps = streamingThinking.length ? streamingThinking : response.thinking_steps || []

  const handleCopy = async () => {
    await navigator.clipboard.writeText(content)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  // Process content to add inline citations
  const processedContent = content.replace(/\[(\d+)\]/g, (match, num) => {
    return `<span class="citation-marker" data-index="${num}">${match}</span>`
  })

  const handleCitationClick = (index: number) => {
    setExpandedCitation(expandedCitation === index ? null : index)
  }

  return (
    <div className="ai-response">
      {/* Mode badge */}
      <div className="ai-response-header">
        <span className={`ai-mode-badge ${response.mode}`}>
          {response.mode === 'quick' && 'Quick AI'}
          {response.mode === 'deep' && 'Deep Analysis'}
          {response.mode === 'research' && 'Research'}
        </span>
        <span className="ai-sources-count">
          {response.sources_used} sources
        </span>
      </div>

      {/* Thinking steps */}
      {thinkingSteps.length > 0 && (
        <ThinkingSteps steps={thinkingSteps} isStreaming={isStreaming} />
      )}

      {/* Main content */}
      <div className="ai-response-content">
        <div
          className="ai-response-text"
          dangerouslySetInnerHTML={{ __html: processedContent }}
        />
        {isStreaming && <span className="typing-cursor" />}
      </div>

      {/* Citations */}
      {response.citations.length > 0 && (
        <div className="ai-citations">
          <div className="ai-citations-header">
            <BookOpen size={14} />
            <span>Sources</span>
          </div>
          <div className="ai-citations-list">
            {response.citations.map((citation) => (
              <div key={citation.index} className="ai-citation-item">
                <CitationChip
                  citation={citation}
                  onClick={() => handleCitationClick(citation.index)}
                />
                {expandedCitation === citation.index && citation.snippet && (
                  <div className="ai-citation-snippet">
                    {citation.snippet}
                  </div>
                )}
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Actions */}
      <div className="ai-response-actions">
        <button
          type="button"
          onClick={handleCopy}
          className="ai-action-button"
        >
          {copied ? <Check size={14} /> : <Copy size={14} />}
          {copied ? 'Copied' : 'Copy'}
        </button>
        {onAddToCanvas && (
          <button
            type="button"
            onClick={onAddToCanvas}
            className="ai-action-button"
          >
            <BookOpen size={14} />
            Add to Canvas
          </button>
        )}
      </div>

      {/* Follow-up suggestions */}
      {!isStreaming && response.follow_ups.length > 0 && onFollowUp && (
        <FollowUpChips
          suggestions={response.follow_ups}
          onSelect={onFollowUp}
        />
      )}
    </div>
  )
}
