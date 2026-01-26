import type { Citation } from '../../types/ai'

interface SourceBadgeProps {
  citation: Citation
  compact?: boolean
  onClick?: () => void
}

export function SourceBadge({ citation, compact, onClick }: SourceBadgeProps) {
  // Extract display name from domain
  const getDisplayName = () => {
    if (citation.domain) {
      return citation.domain.replace('www.', '').split('.')[0]
    }
    try {
      return new URL(citation.url).hostname.replace('www.', '').split('.')[0]
    } catch {
      return 'Source'
    }
  }
  const displayName = getDisplayName()

  return (
    <span
      className={`source-badge ${compact ? 'compact' : ''}`}
      onClick={onClick}
      role={onClick ? 'button' : undefined}
      tabIndex={onClick ? 0 : undefined}
      onKeyDown={onClick ? (e) => {
        if (e.key === 'Enter' || e.key === ' ') {
          onClick()
        }
      } : undefined}
      title={citation.title}
    >
      {citation.favicon && (
        <img
          src={citation.favicon}
          alt=""
          className="source-favicon"
          onError={(e) => {
            // Hide broken favicon
            (e.target as HTMLImageElement).style.display = 'none'
          }}
        />
      )}
      <span className="source-name">{displayName}</span>
      {(citation.other_sources ?? 0) > 0 && (
        <span className="source-more">+{citation.other_sources}</span>
      )}
    </span>
  )
}
