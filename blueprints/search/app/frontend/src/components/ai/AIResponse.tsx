import { useState } from 'react'
import { Copy, Check, BookOpen, Share, Download, RefreshCw, ThumbsUp, ThumbsDown, MoreHorizontal, Zap, Database, Cpu, Cloud } from 'lucide-react'
import type { AIResponse as AIResponseType, TokenUsage } from '../../types/ai'
import { MarkdownRenderer } from './MarkdownRenderer'
import { CitationChip } from './CitationChip'
import { ThinkingSteps } from './ThinkingSteps'
import { ImageCarousel } from './ImageCarousel'
import { RelatedQuestions } from './RelatedQuestions'

// Compact token usage display for AI responses
function TokenUsageBadge({ usage }: { usage: TokenUsage }) {
  const formatTokens = (tokens: number | undefined) => {
    if (tokens === undefined || tokens === null) return '0'
    if (tokens >= 1000) return `${(tokens / 1000).toFixed(1)}k`
    return tokens.toString()
  }

  return (
    <div className="token-usage-badge" title={`${usage.input_tokens ?? 0} in / ${usage.output_tokens ?? 0} out${usage.tokens_per_second ? ` • ${usage.tokens_per_second.toFixed(0)} tok/s` : ''}`}>
      <Zap size={12} />
      <span>{formatTokens(usage.total_tokens)} tokens</span>
      {usage.tokens_per_second && usage.tokens_per_second > 0 && (
        <span className="token-speed">• {usage.tokens_per_second.toFixed(0)} tok/s</span>
      )}
    </div>
  )
}

// Provider and model info badge
function ProviderBadge({ provider, model, fromCache }: { provider?: string; model?: string; fromCache?: boolean }) {
  if (!provider && !model) return null

  const getProviderIcon = () => {
    if (provider === 'claude') return <Cloud size={12} />
    if (provider === 'llamacpp') return <Cpu size={12} />
    return <Cpu size={12} />
  }

  const getProviderLabel = () => {
    if (provider === 'claude') return 'Claude'
    if (provider === 'llamacpp') return 'Local'
    return provider || 'AI'
  }

  const getModelLabel = () => {
    if (!model) return ''
    // Simplify model names for display
    if (model.includes('haiku')) return 'Haiku'
    if (model.includes('sonnet')) return 'Sonnet'
    if (model.includes('opus')) return 'Opus'
    if (model.includes('gemma-3-270m')) return 'Gemma 270M'
    if (model.includes('gemma-3-1b')) return 'Gemma 1B'
    if (model.includes('gemma-3-4b')) return 'Gemma 4B'
    return model
  }

  return (
    <div className="provider-badge" title={`Provider: ${provider || 'unknown'}, Model: ${model || 'unknown'}${fromCache ? ' (cached)' : ''}`}>
      {getProviderIcon()}
      <span>{getProviderLabel()}</span>
      {model && <span className="model-name">{getModelLabel()}</span>}
      {fromCache && (
        <span className="cache-badge" title="Response from cache">
          <Database size={10} />
        </span>
      )}
    </div>
  )
}

interface AIResponseProps {
  response: AIResponseType
  streamingContent?: string
  streamingThinking?: string[]
  isStreaming?: boolean
  onFollowUp?: (question: string) => void
  onRefresh?: () => void
}

export function AIResponse({
  response,
  streamingContent,
  streamingThinking = [],
  isStreaming = false,
  onFollowUp,
  onRefresh,
}: AIResponseProps) {
  const [copied, setCopied] = useState(false)
  const [expandedCitation, setExpandedCitation] = useState<number | null>(null)
  const [showSources, setShowSources] = useState(false)

  const content = streamingContent || response.text
  const thinkingSteps = streamingThinking.length ? streamingThinking : response.thinking_steps || []
  const relatedQuestions = response.related_questions || []
  const images = response.images || []

  const handleCopy = async () => {
    await navigator.clipboard.writeText(content)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  const handleCitationClick = (index: number) => {
    setExpandedCitation(expandedCitation === index ? null : index)
  }

  const handleShare = async () => {
    if (navigator.share) {
      await navigator.share({
        title: 'AI Response',
        text: content,
      })
    }
  }

  const handleDownload = () => {
    const blob = new Blob([content], { type: 'text/markdown' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = 'ai-response.md'
    a.click()
    URL.revokeObjectURL(url)
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
      {/* Mode badge and stats */}
      <div className="ai-response-header">
        <span className={`ai-mode-badge ${response.mode}`}>
          {getModeLabel(response.mode)}
        </span>
        <span className="ai-sources-count">
          {response.sources_used} sources
        </span>
        {!isStreaming && (
          <ProviderBadge
            provider={response.provider}
            model={response.model}
            fromCache={response.from_cache}
          />
        )}
        {!isStreaming && response.usage && (
          <TokenUsageBadge usage={response.usage} />
        )}
      </div>

      {/* Image Carousel */}
      {images.length > 0 && (
        <ImageCarousel
          images={images}
          onImageClick={(img) => window.open(img.source_url || img.url, '_blank')}
        />
      )}

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

      {/* Collapsible Sources */}
      {response.citations.length > 0 && (
        <div className="ai-citations">
          <button
            type="button"
            className="ai-citations-header"
            onClick={() => setShowSources(!showSources)}
          >
            <BookOpen size={14} />
            <span>Sources ({response.citations.length})</span>
          </button>
          {showSources && (
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
          )}
        </div>
      )}

      {/* Action Bar - Perplexity style */}
      <div className="ai-response-actions">
        <button type="button" onClick={handleShare} className="ai-action-button" title="Share">
          <Share size={16} />
        </button>
        <button type="button" onClick={handleDownload} className="ai-action-button" title="Download">
          <Download size={16} />
        </button>
        <button type="button" onClick={handleCopy} className="ai-action-button" title={copied ? 'Copied!' : 'Copy'}>
          {copied ? <Check size={16} /> : <Copy size={16} />}
        </button>
        {onRefresh && (
          <button type="button" onClick={onRefresh} className="ai-action-button" title="Regenerate">
            <RefreshCw size={16} />
          </button>
        )}

        {/* Source icons indicator */}
        <div className="sources-indicator">
          <div className="source-icons">
            {response.citations.slice(0, 4).map((c, i) => (
              c.favicon && (
                <img
                  key={i}
                  src={c.favicon}
                  alt=""
                  onError={(e) => {
                    (e.target as HTMLImageElement).style.display = 'none'
                  }}
                />
              )
            ))}
          </div>
          <span>{response.sources_used} sources</span>
        </div>

        {/* Feedback buttons */}
        <div className="feedback-buttons">
          <button type="button" className="ai-action-button" title="Helpful">
            <ThumbsUp size={16} />
          </button>
          <button type="button" className="ai-action-button" title="Not helpful">
            <ThumbsDown size={16} />
          </button>
          <button type="button" className="ai-action-button" title="More options">
            <MoreHorizontal size={16} />
          </button>
        </div>
      </div>

      {/* Related Questions - Perplexity style */}
      {!isStreaming && relatedQuestions.length > 0 && onFollowUp && (
        <RelatedQuestions
          questions={relatedQuestions}
          onSelect={onFollowUp}
        />
      )}
    </div>
  )
}
