import { useState } from 'react'
import { ChevronDown, Coins, Zap, Clock } from 'lucide-react'
import type { TokenUsage } from '../../types/ai'

interface TokenUsageDisplayProps {
  usage: TokenUsage
  provider?: string
  showCost?: boolean
  collapsible?: boolean
}

export function TokenUsageDisplay({
  usage,
  provider,
  showCost = true,
  collapsible = true,
}: TokenUsageDisplayProps) {
  const [isExpanded, setIsExpanded] = useState(!collapsible)

  const isCloudProvider = provider === 'claude' || provider === 'openai'
  const shouldShowCost = showCost && isCloudProvider && usage.cost_usd !== undefined && usage.cost_usd > 0

  // Format cost
  const formatCost = (cost: number) => {
    if (cost < 0.01) {
      return `$${cost.toFixed(6)}`
    }
    return `$${cost.toFixed(4)}`
  }

  // Format tokens
  const formatTokens = (tokens: number | undefined) => {
    if (tokens === undefined || tokens === null) return '0'
    if (tokens >= 1000) {
      return `${(tokens / 1000).toFixed(1)}k`
    }
    return tokens.toString()
  }

  if (!usage || (usage.total_tokens === 0 && !usage.input_tokens && !usage.output_tokens)) {
    return null
  }

  const content = (
    <div className="token-usage-content">
      <div className="token-usage-row">
        <Zap size={12} />
        <span className="token-usage-label">Tokens</span>
        <span className="token-usage-value">
          {formatTokens(usage.input_tokens)} in / {formatTokens(usage.output_tokens)} out
        </span>
      </div>

      {(usage.cache_read_tokens !== undefined && usage.cache_read_tokens > 0) ||
       (usage.cache_write_tokens !== undefined && usage.cache_write_tokens > 0) ? (
        <div className="token-usage-row">
          <span className="token-usage-label">Cache</span>
          <span className="token-usage-value">
            {formatTokens(usage.cache_read_tokens || 0)} read / {formatTokens(usage.cache_write_tokens || 0)} write
          </span>
        </div>
      ) : null}

      {usage.tokens_per_second !== undefined && usage.tokens_per_second > 0 && (
        <div className="token-usage-row">
          <Clock size={12} />
          <span className="token-usage-label">Speed</span>
          <span className="token-usage-value">
            {usage.tokens_per_second.toFixed(1)} tok/s
          </span>
        </div>
      )}

      {shouldShowCost && (
        <div className="token-usage-row cost">
          <Coins size={12} />
          <span className="token-usage-label">Cost</span>
          <span className="token-usage-value cost-value">
            {formatCost(usage.cost_usd!)}
          </span>
        </div>
      )}
    </div>
  )

  if (!collapsible) {
    return (
      <div className="token-usage-display">
        {content}
        <style>{tokenUsageStyles}</style>
      </div>
    )
  }

  return (
    <div className="token-usage-display collapsible">
      <button
        type="button"
        className="token-usage-toggle"
        onClick={() => setIsExpanded(!isExpanded)}
        aria-expanded={isExpanded}
      >
        <Zap size={12} />
        <span className="token-usage-summary">
          {formatTokens(usage.total_tokens)} tokens
          {shouldShowCost && ` (${formatCost(usage.cost_usd!)})`}
        </span>
        <ChevronDown
          size={14}
          className={`token-usage-chevron ${isExpanded ? 'expanded' : ''}`}
        />
      </button>

      {isExpanded && content}

      <style>{tokenUsageStyles}</style>
    </div>
  )
}

const tokenUsageStyles = `
  .token-usage-display {
    font-size: 0.75rem;
    color: var(--text-secondary, #666);
    border-left: 2px solid var(--border-color, #e0e0e0);
    padding-left: 0.75rem;
    margin-top: 0.5rem;
  }

  .token-usage-display.collapsible {
    border-left: none;
    padding-left: 0;
  }

  .token-usage-toggle {
    display: flex;
    align-items: center;
    gap: 0.375rem;
    background: none;
    border: none;
    cursor: pointer;
    padding: 0.25rem 0.5rem;
    border-radius: 0.25rem;
    color: var(--text-secondary, #666);
    font-size: 0.75rem;
  }

  .token-usage-toggle:hover {
    background: var(--bg-hover, #f5f5f5);
  }

  .token-usage-summary {
    font-weight: 500;
  }

  .token-usage-chevron {
    transition: transform 0.2s;
  }

  .token-usage-chevron.expanded {
    transform: rotate(180deg);
  }

  .token-usage-content {
    display: flex;
    flex-direction: column;
    gap: 0.25rem;
    padding: 0.5rem;
    background: var(--bg-secondary, #f9f9f9);
    border-radius: 0.375rem;
    margin-top: 0.25rem;
  }

  .token-usage-row {
    display: flex;
    align-items: center;
    gap: 0.5rem;
  }

  .token-usage-row svg {
    color: var(--text-tertiary, #999);
    flex-shrink: 0;
  }

  .token-usage-label {
    color: var(--text-tertiary, #999);
    min-width: 50px;
  }

  .token-usage-value {
    font-family: var(--font-mono, monospace);
  }

  .token-usage-row.cost {
    margin-top: 0.25rem;
    padding-top: 0.25rem;
    border-top: 1px solid var(--border-color, #e0e0e0);
  }

  .cost-value {
    color: var(--warning, #ed6c02);
    font-weight: 500;
  }
`
