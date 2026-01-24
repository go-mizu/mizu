import { ArrowRight } from 'lucide-react'

interface FollowUpChipsProps {
  suggestions: string[]
  onSelect: (suggestion: string) => void
}

export function FollowUpChips({ suggestions, onSelect }: FollowUpChipsProps) {
  if (!suggestions.length) return null

  return (
    <div className="follow-up-chips">
      <p className="follow-up-label">Follow-up questions</p>
      <div className="follow-up-list">
        {suggestions.map((suggestion, i) => (
          <button
            key={i}
            type="button"
            onClick={() => onSelect(suggestion)}
            className="follow-up-chip"
          >
            <span>{suggestion}</span>
            <ArrowRight size={14} className="follow-up-arrow" />
          </button>
        ))}
      </div>
    </div>
  )
}
