import { Search, FileText, Lightbulb, CheckCircle } from 'lucide-react'

interface ThinkingStepsProps {
  steps: string[]
  isStreaming?: boolean
}

function getStepIcon(step: string) {
  const lower = step.toLowerCase()
  if (lower.includes('search')) return <Search size={14} />
  if (lower.includes('fetch') || lower.includes('read')) return <FileText size={14} />
  if (lower.includes('answer') || lower.includes('done')) return <CheckCircle size={14} />
  return <Lightbulb size={14} />
}

export function ThinkingSteps({ steps, isStreaming = false }: ThinkingStepsProps) {
  if (!steps.length) return null

  return (
    <div className="thinking-steps">
      <div className="thinking-header">
        {isStreaming && <span className="thinking-indicator" />}
        <span className="thinking-label">Thinking...</span>
      </div>
      <div className="thinking-list">
        {steps.map((step, i) => (
          <div key={i} className="thinking-step">
            <span className="thinking-step-icon">{getStepIcon(step)}</span>
            <span className="thinking-step-text">{step}</span>
          </div>
        ))}
      </div>
    </div>
  )
}
