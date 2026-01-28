import { useState, useEffect, useCallback, useRef } from 'react'
import { Mic, MicOff, Loader2 } from 'lucide-react'

interface VoiceInputProps {
  onTranscript: (text: string) => void
  onInterimTranscript?: (text: string) => void
  disabled?: boolean
  size?: 'sm' | 'md'
}

// Check for browser support
const SpeechRecognition =
  typeof window !== 'undefined'
    ? window.SpeechRecognition || (window as any).webkitSpeechRecognition
    : null

export function VoiceInput({
  onTranscript,
  onInterimTranscript,
  disabled = false,
  size = 'md',
}: VoiceInputProps) {
  const [isListening, setIsListening] = useState(false)
  const [isSupported, setIsSupported] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const recognitionRef = useRef<any>(null)

  useEffect(() => {
    setIsSupported(SpeechRecognition !== null)
  }, [])

  const startListening = useCallback(() => {
    if (!SpeechRecognition || disabled) return

    setError(null)

    const recognition = new SpeechRecognition()
    recognition.continuous = true
    recognition.interimResults = true
    recognition.lang = 'en-US'

    recognition.onresult = (event: any) => {
      let interimTranscript = ''
      let finalTranscript = ''

      for (let i = event.resultIndex; i < event.results.length; i++) {
        const transcript = event.results[i][0].transcript
        if (event.results[i].isFinal) {
          finalTranscript += transcript
        } else {
          interimTranscript += transcript
        }
      }

      if (interimTranscript && onInterimTranscript) {
        onInterimTranscript(interimTranscript)
      }

      if (finalTranscript) {
        onTranscript(finalTranscript)
      }
    }

    recognition.onerror = (event: any) => {
      switch (event.error) {
        case 'no-speech':
          setError('No speech detected')
          break
        case 'audio-capture':
          setError('No microphone found')
          break
        case 'not-allowed':
          setError('Microphone access denied')
          break
        default:
          setError(`Speech recognition error: ${event.error}`)
      }
      setIsListening(false)
    }

    recognition.onend = () => {
      setIsListening(false)
    }

    recognitionRef.current = recognition
    recognition.start()
    setIsListening(true)
  }, [disabled, onTranscript, onInterimTranscript])

  const stopListening = useCallback(() => {
    if (recognitionRef.current) {
      recognitionRef.current.stop()
      recognitionRef.current = null
    }
    setIsListening(false)
  }, [])

  const toggleListening = useCallback(() => {
    if (isListening) {
      stopListening()
    } else {
      startListening()
    }
  }, [isListening, startListening, stopListening])

  // Clean up on unmount
  useEffect(() => {
    return () => {
      if (recognitionRef.current) {
        recognitionRef.current.stop()
      }
    }
  }, [])

  if (!isSupported) {
    return (
      <button
        type="button"
        className={`voice-input-button ${size} unsupported`}
        disabled
        title="Voice input not supported in this browser"
      >
        <MicOff size={size === 'sm' ? 16 : 18} />
      </button>
    )
  }

  return (
    <div className="voice-input">
      <button
        type="button"
        onClick={toggleListening}
        className={`voice-input-button ${size} ${isListening ? 'listening' : ''}`}
        disabled={disabled}
        title={isListening ? 'Stop listening' : 'Start voice input'}
        aria-label={isListening ? 'Stop listening' : 'Start voice input'}
      >
        {isListening ? (
          <div className="voice-input-active">
            <Loader2 size={size === 'sm' ? 16 : 18} className="animate-spin" />
          </div>
        ) : (
          <Mic size={size === 'sm' ? 16 : 18} />
        )}
      </button>

      {error && (
        <div className="voice-input-error" role="alert">
          {error}
        </div>
      )}

      {isListening && (
        <div className="voice-input-indicator">
          <span className="voice-input-dot" />
          <span className="voice-input-label">Listening...</span>
        </div>
      )}
    </div>
  )
}

// Add type declaration for SpeechRecognition
declare global {
  interface Window {
    SpeechRecognition: any
    webkitSpeechRecognition: any
  }
}
