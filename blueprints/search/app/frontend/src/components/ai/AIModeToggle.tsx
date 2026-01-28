import { Zap, Brain, FlaskConical, FileSearch } from 'lucide-react'
import type { AIMode } from '../../types/ai'
import { useAIStore } from '../../stores/aiStore'

interface AIModeToggleProps {
  size?: 'sm' | 'md'
  onChange?: (mode: AIMode) => void
}

const modes: { id: AIMode; label: string; icon: React.ReactNode; description: string }[] = [
  {
    id: 'quick',
    label: 'Quick',
    icon: <Zap size={16} />,
    description: 'Fast AI summary',
  },
  {
    id: 'deep',
    label: 'Deep',
    icon: <Brain size={16} />,
    description: 'Detailed analysis',
  },
  {
    id: 'research',
    label: 'Research',
    icon: <FlaskConical size={16} />,
    description: 'Multi-step research',
  },
  {
    id: 'deepsearch',
    label: 'Deep Search',
    icon: <FileSearch size={16} />,
    description: 'Comprehensive report from 50+ sources',
  },
]

export function AIModeToggle({ size = 'md', onChange }: AIModeToggleProps) {
  const { mode, setMode, availableModes } = useAIStore()

  const handleModeChange = (newMode: AIMode) => {
    setMode(newMode)
    onChange?.(newMode)
  }

  return (
    <div className={`ai-mode-toggle ${size === 'sm' ? 'gap-1' : 'gap-2'}`}>
      {modes.map((m) => {
        const isAvailable = availableModes.includes(m.id)
        const isActive = mode === m.id

        return (
          <button
            key={m.id}
            type="button"
            disabled={!isAvailable}
            onClick={() => handleModeChange(m.id)}
            className={`
              ai-mode-button
              ${isActive ? 'active' : ''}
              ${!isAvailable ? 'disabled' : ''}
              ${size === 'sm' ? 'px-2 py-1 text-xs' : 'px-3 py-1.5 text-sm'}
            `}
            title={m.description}
          >
            <span className="ai-mode-icon">{m.icon}</span>
            <span className="ai-mode-label">{m.label}</span>
          </button>
        )
      })}
    </div>
  )
}
