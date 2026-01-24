import { useState } from 'react'
import { Copy, Check, BookOpen, Volume2, VolumeX } from 'lucide-react'
import type { AIResponse as AIResponseType } from '../../types/ai'
import { MarkdownRenderer } from './MarkdownRenderer'
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
  const [isSpeaking, setIsSpeaking] = useState(false)

  const content = streamingContent || response.text
  const thinkingSteps = streamingThinking.length ? streamingThinking : response.thinking_steps || []

  const handleCopy = async () => {
    await navigator.clipboard.writeText(content)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  const handleCitationClick = (index: number) => {
    setExpandedCitation(expandedCitation === index ? null : index)
  }

  const handleSpeak = () => {
    if (isSpeaking) {
      window.speechSynthesis.cancel()
      setIsSpeaking(false)
      return
    }

    // Strip markdown for TTS
    const plainText = content
      .replace(/\[(\d+)\]/g, '') // Remove citations
      .replace(/```[\s\S]*?```/g, 'code block') // Replace code blocks
      .replace(/`[^`]+`/g, '') // Remove inline code
      .replace(/[#*_~]/g, '') // Remove markdown formatting

    const utterance = new SpeechSynthesisUtterance(plainText)
    utterance.onend = () => setIsSpeaking(false)
    utterance.onerror = () => setIsSpeaking(false)

    window.speechSynthesis.speak(utterance)
    setIsSpeaking(true)
  }

  const getModeLabel = (mode: string) => {
    switch (mode) {
      case 'quick': return 'Quick AI'
      case 'deep': return 'Deep Analysis'
      case 'research': return 'Research'
      case 'deepsearch': return 'Deep Search'
      default: return mode
    }
  }

  return (
    <div className="ai-response">
      {/* Mode badge */}
      <div className="ai-response-header">
        <span className={`ai-mode-badge ${response.mode}`}>
          {getModeLabel(response.mode)}
        </span>
        <span className="ai-sources-count">
          {response.sources_used} sources
        </span>
      </div>

      {/* Thinking steps */}
      {thinkingSteps.length > 0 && (
        <ThinkingSteps steps={thinkingSteps} isStreaming={isStreaming} />
      )}

      {/* Main content with markdown rendering */}
      <div className="ai-response-content">
        <MarkdownRenderer
          content={content}
          citations={response.citations}
          isStreaming={isStreaming}
          onCitationClick={handleCitationClick}
        />
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
        <button
          type="button"
          onClick={handleSpeak}
          className="ai-action-button"
          title={isSpeaking ? 'Stop speaking' : 'Read aloud'}
        >
          {isSpeaking ? <VolumeX size={14} /> : <Volume2 size={14} />}
          {isSpeaking ? 'Stop' : 'Listen'}
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
