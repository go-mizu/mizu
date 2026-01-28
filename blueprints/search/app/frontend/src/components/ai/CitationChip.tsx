import { ExternalLink } from 'lucide-react'
import type { Citation } from '../../types/ai'

interface CitationChipProps {
  citation: Citation
  onClick?: () => void
}

export function CitationChip({ citation, onClick }: CitationChipProps) {
  const handleClick = (e: React.MouseEvent) => {
    if (onClick) {
      e.preventDefault()
      onClick()
    }
  }

  return (
    <a
      href={citation.url}
      target="_blank"
      rel="noopener noreferrer"
      onClick={handleClick}
      className="citation-chip group"
      title={citation.title}
    >
      <span className="citation-index">[{citation.index}]</span>
      <span className="citation-title">{citation.title}</span>
      <ExternalLink size={12} className="citation-icon" />
    </a>
  )
}

interface InlineCitationProps {
  index: number
  onClick?: () => void
}

export function InlineCitation({ index, onClick }: InlineCitationProps) {
  return (
    <button
      type="button"
      onClick={onClick}
      className="inline-citation"
    >
      [{index}]
    </button>
  )
}
