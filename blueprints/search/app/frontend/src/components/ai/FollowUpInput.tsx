import { useState, useRef, useCallback } from 'react'
import { Search, Sparkles, Grid3X3, Focus, Paperclip, ArrowUp } from 'lucide-react'
import { VoiceInput } from './VoiceInput'

type InputMode = 'search' | 'ai' | 'collection'

interface FollowUpInputProps {
  onSubmit: (text: string) => void
  disabled?: boolean
  placeholder?: string
}

export function FollowUpInput({ onSubmit, disabled, placeholder = 'Ask a follow-up' }: FollowUpInputProps) {
  const [value, setValue] = useState('')
  const [inputMode, setInputMode] = useState<InputMode>('search')
  const [interimTranscript, setInterimTranscript] = useState('')
  const inputRef = useRef<HTMLInputElement>(null)

  const handleSubmit = (e?: React.FormEvent) => {
    e?.preventDefault()
    if (!value.trim() || disabled) return
    onSubmit(value.trim())
    setValue('')
  }

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      handleSubmit()
    }
  }

  const handleVoiceTranscript = useCallback((text: string) => {
    setValue((prev) => prev + (prev ? ' ' : '') + text)
    setInterimTranscript('')
  }, [])

  const handleInterimTranscript = useCallback((text: string) => {
    setInterimTranscript(text)
  }, [])

  const displayValue = value + (interimTranscript ? ` ${interimTranscript}` : '')

  return (
    <div className="followup-input-container">
      <form onSubmit={handleSubmit} className="followup-input-box">
        {/* Mode toggles */}
        <div className="followup-modes">
          <button
            type="button"
            className={inputMode === 'search' ? 'active' : ''}
            onClick={() => setInputMode('search')}
            title="Search mode"
          >
            <Search size={16} />
          </button>
          <button
            type="button"
            className={inputMode === 'ai' ? 'active' : ''}
            onClick={() => setInputMode('ai')}
            title="AI mode"
          >
            <Sparkles size={16} />
          </button>
          <button
            type="button"
            className={inputMode === 'collection' ? 'active' : ''}
            onClick={() => setInputMode('collection')}
            title="Collection mode"
          >
            <Grid3X3 size={16} />
          </button>
        </div>

        {/* Input field */}
        <input
          ref={inputRef}
          type="text"
          value={displayValue}
          onChange={(e) => setValue(e.target.value)}
          onKeyDown={handleKeyDown}
          placeholder={placeholder}
          disabled={disabled}
          className="followup-input"
        />

        {/* Action buttons */}
        <div className="followup-actions">
          <button
            type="button"
            className="action-btn"
            title="Focus"
            disabled={disabled}
          >
            <Focus size={18} />
          </button>
          <button
            type="button"
            className="action-btn"
            title="Attach file"
            disabled={disabled}
          >
            <Paperclip size={18} />
          </button>
          <VoiceInput
            onTranscript={handleVoiceTranscript}
            onInterimTranscript={handleInterimTranscript}
            disabled={disabled}
            size="sm"
          />
          <button
            type="submit"
            className="submit-btn"
            disabled={!value.trim() || disabled}
            title="Send"
          >
            <ArrowUp size={18} />
          </button>
        </div>
      </form>
    </div>
  )
}
