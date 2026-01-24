import { useState, useRef, useEffect, useCallback } from 'react'
import { Send, Loader2 } from 'lucide-react'
import { aiApi } from '../../api/ai'
import { useAIStore } from '../../stores/aiStore'
import { AIResponse } from './AIResponse'
import { AIModeToggle } from './AIModeToggle'
import { ModelSelector } from './ModelSelector'
import { FileUploadZone, type UploadedFile } from './FileUploadZone'
import { VoiceInput } from './VoiceInput'
import type { AIMessage, AIResponse as AIResponseType, AIMode } from '../../types/ai'

interface AIChatProps {
  sessionId: string
  initialMessages?: AIMessage[]
}

export function AIChat({ sessionId, initialMessages = [] }: AIChatProps) {
  const {
    mode,
    selectedModelId,
    setSelectedModelId,
    isLoading,
    isStreaming,
    streamingContent,
    streamingThinking,
    setLoading,
    setStreaming,
    appendStreamContent,
    addThinkingStep,
    resetStream,
    setError,
  } = useAIStore()

  const [messages, setMessages] = useState<AIMessage[]>(initialMessages)
  const [input, setInput] = useState('')
  const [files, setFiles] = useState<UploadedFile[]>([])
  const [interimTranscript, setInterimTranscript] = useState('')
  const [currentStreamResponse, setCurrentStreamResponse] = useState<AIResponseType | null>(null)
  const messagesEndRef = useRef<HTMLDivElement>(null)
  const inputRef = useRef<HTMLTextAreaElement>(null)

  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }

  useEffect(() => {
    scrollToBottom()
  }, [messages, streamingContent])

  // Convert files to data URLs for the API
  const getFileUrls = useCallback(async (): Promise<string[]> => {
    const urls: string[] = []
    for (const file of files) {
      if (file.type === 'image' && file.preview) {
        urls.push(file.preview)
      } else {
        // Read file as data URL
        const dataUrl = await new Promise<string>((resolve) => {
          const reader = new FileReader()
          reader.onload = () => resolve(reader.result as string)
          reader.readAsDataURL(file.file)
        })
        urls.push(dataUrl)
      }
    }
    return urls
  }, [files])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!input.trim() || isLoading || isStreaming) return

    const imageUrls = await getFileUrls()

    const userMessage: AIMessage = {
      id: crypto.randomUUID(),
      session_id: sessionId,
      role: 'user',
      content: input.trim(),
      created_at: new Date().toISOString(),
    }

    setMessages((prev) => [...prev, userMessage])
    setInput('')
    setFiles([])
    setInterimTranscript('')
    resetStream()
    setLoading(true)
    setError(null)

    try {
      setStreaming(true)

      const stream = aiApi.queryStreamFetch({
        text: userMessage.content,
        mode,
        model_id: selectedModelId || undefined,
        session_id: sessionId,
        image_urls: imageUrls.length > 0 ? imageUrls : undefined,
      })

      let response: AIResponseType | null = null

      for await (const event of stream) {
        switch (event.type) {
          case 'token':
            if (event.content) {
              appendStreamContent(event.content)
            }
            break
          case 'thinking':
            if (event.thinking) {
              addThinkingStep(event.thinking)
            }
            break
          case 'done':
            if (event.response) {
              response = event.response
              setCurrentStreamResponse(response)
            }
            break
          case 'error':
            setError(event.error || 'An error occurred')
            break
        }
      }

      if (response) {
        const assistantMessage: AIMessage = {
          id: crypto.randomUUID(),
          session_id: sessionId,
          role: 'assistant',
          content: response.text,
          mode: response.mode,
          citations: response.citations,
          created_at: new Date().toISOString(),
        }
        setMessages((prev) => [...prev, assistantMessage])
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to get response')
    } finally {
      setLoading(false)
      setStreaming(false)
      resetStream()
      setCurrentStreamResponse(null)
    }
  }

  const handleFollowUp = (question: string) => {
    setInput(question)
    inputRef.current?.focus()
  }

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      handleSubmit(e)
    }
  }

  const handleVoiceTranscript = useCallback((text: string) => {
    setInput((prev) => prev + (prev ? ' ' : '') + text)
    setInterimTranscript('')
  }, [])

  const handleInterimTranscript = useCallback((text: string) => {
    setInterimTranscript(text)
  }, [])

  return (
    <div className="ai-chat">
      {/* Messages */}
      <div className="ai-chat-messages">
        {messages.map((message) => (
          <div
            key={message.id}
            className={`ai-chat-message ${message.role}`}
          >
            {message.role === 'user' ? (
              <div className="ai-chat-user-message">
                {message.content}
              </div>
            ) : (
              <AIResponse
                response={{
                  text: message.content,
                  mode: (message.mode || 'quick') as AIMode,
                  citations: message.citations || [],
                  follow_ups: [],
                  related_questions: [],
                  images: [],
                  session_id: sessionId,
                  sources_used: message.citations?.length || 0,
                }}
                onFollowUp={handleFollowUp}
              />
            )}
          </div>
        ))}

        {/* Streaming response */}
        {isStreaming && streamingContent && (
          <div className="ai-chat-message assistant">
            <AIResponse
              response={currentStreamResponse || {
                text: streamingContent,
                mode,
                citations: [],
                follow_ups: [],
                related_questions: [],
                images: [],
                session_id: sessionId,
                sources_used: 0,
                thinking_steps: streamingThinking,
              }}
              streamingContent={streamingContent}
              streamingThinking={streamingThinking}
              isStreaming={true}
            />
          </div>
        )}

        <div ref={messagesEndRef} />
      </div>

      {/* Input */}
      <div className="ai-chat-input-container">
        <div className="ai-chat-mode-bar">
          <AIModeToggle size="sm" />
          <ModelSelector
            selectedModel={selectedModelId || undefined}
            onSelectModel={setSelectedModelId}
            size="sm"
          />
        </div>

        <FileUploadZone files={files} onFilesChange={setFiles}>
          <form onSubmit={handleSubmit} className="ai-chat-form">
            <div className="ai-chat-input-wrapper">
              <textarea
                ref={inputRef}
                value={input + (interimTranscript ? ` ${interimTranscript}` : '')}
                onChange={(e) => setInput(e.target.value)}
                onKeyDown={handleKeyDown}
                placeholder="Ask a follow-up question..."
                className="ai-chat-input"
                rows={1}
                disabled={isLoading || isStreaming}
              />
              {interimTranscript && (
                <span className="ai-chat-interim">{interimTranscript}</span>
              )}
            </div>
            <div className="ai-chat-actions">
              <VoiceInput
                onTranscript={handleVoiceTranscript}
                onInterimTranscript={handleInterimTranscript}
                disabled={isLoading || isStreaming}
                size="sm"
              />
              <button
                type="submit"
                disabled={!input.trim() || isLoading || isStreaming}
                className="ai-chat-submit"
              >
                {isLoading || isStreaming ? (
                  <Loader2 size={18} className="animate-spin" />
                ) : (
                  <Send size={18} />
                )}
              </button>
            </div>
          </form>
        </FileUploadZone>
      </div>
    </div>
  )
}
