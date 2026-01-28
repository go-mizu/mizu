import { CornerDownRight } from 'lucide-react'
import type { RelatedQuestion } from '../../types/ai'

interface RelatedQuestionsProps {
  questions: RelatedQuestion[]
  onSelect: (question: string) => void
}

export function RelatedQuestions({ questions, onSelect }: RelatedQuestionsProps) {
  if (!questions.length) return null

  return (
    <div className="related-questions">
      <h3 className="related-header">Related</h3>
      <div className="related-list">
        {questions.map((q, i) => (
          <button
            key={i}
            type="button"
            className="related-item"
            onClick={() => onSelect(q.text)}
          >
            <CornerDownRight size={16} className="related-icon" />
            <span>{q.text}</span>
          </button>
        ))}
      </div>
    </div>
  )
}
