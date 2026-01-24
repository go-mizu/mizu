import { useState, useEffect, useRef } from 'react'
import { ChevronDown, Cpu, Eye, Mic, Zap, Check, AlertCircle } from 'lucide-react'
import type { ModelInfo, ModelCapability } from '../../types/ai'
import { aiApi } from '../../api/ai'

interface ModelSelectorProps {
  selectedModel?: string
  onSelectModel: (modelId: string) => void
  requiredCapability?: ModelCapability
  size?: 'sm' | 'md'
}

const capabilityIcons: Record<ModelCapability, typeof Cpu> = {
  text: Cpu,
  vision: Eye,
  voice: Mic,
  embeddings: Zap,
}

const speedLabels: Record<string, { label: string; color: string }> = {
  fast: { label: 'Fast', color: '#34a853' },
  balanced: { label: 'Balanced', color: '#fbbc04' },
  thorough: { label: 'Thorough', color: '#ea4335' },
}

export function ModelSelector({
  selectedModel,
  onSelectModel,
  requiredCapability,
  size = 'md',
}: ModelSelectorProps) {
  const [models, setModels] = useState<ModelInfo[]>([])
  const [isOpen, setIsOpen] = useState(false)
  const [loading, setLoading] = useState(true)
  const dropdownRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    async function fetchModels() {
      try {
        const data = await aiApi.getModels()
        setModels(data)

        // Auto-select default if none selected
        if (!selectedModel) {
          const defaultModel = data.find(m => m.is_default && m.available)
          if (defaultModel) {
            onSelectModel(defaultModel.id)
          }
        }
      } catch (err) {
        console.error('Failed to fetch models:', err)
      } finally {
        setLoading(false)
      }
    }
    fetchModels()
  }, [selectedModel, onSelectModel])

  // Close dropdown on outside click
  useEffect(() => {
    function handleClickOutside(event: MouseEvent) {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target as Node)) {
        setIsOpen(false)
      }
    }
    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [])

  const filteredModels = requiredCapability
    ? models.filter(m => m.capabilities.includes(requiredCapability))
    : models

  const currentModel = models.find(m => m.id === selectedModel)

  if (loading) {
    return (
      <div className={`model-selector ${size}`}>
        <span className="model-selector-loading">Loading models...</span>
      </div>
    )
  }

  return (
    <div className={`model-selector ${size}`} ref={dropdownRef}>
      <button
        type="button"
        className="model-selector-button"
        onClick={() => setIsOpen(!isOpen)}
        aria-expanded={isOpen}
        aria-haspopup="listbox"
      >
        <div className="model-selector-current">
          {currentModel ? (
            <>
              <span className="model-name">{currentModel.name}</span>
              {!currentModel.available && (
                <AlertCircle size={14} className="model-unavailable-icon" />
              )}
            </>
          ) : (
            <span className="model-placeholder">Select model</span>
          )}
        </div>
        <ChevronDown
          size={16}
          className={`model-selector-chevron ${isOpen ? 'open' : ''}`}
        />
      </button>

      {isOpen && (
        <div className="model-selector-dropdown" role="listbox">
          {filteredModels.length === 0 ? (
            <div className="model-selector-empty">No models available</div>
          ) : (
            filteredModels.map(model => (
              <button
                key={model.id}
                type="button"
                className={`model-option ${model.id === selectedModel ? 'selected' : ''} ${!model.available ? 'unavailable' : ''}`}
                onClick={() => {
                  onSelectModel(model.id)
                  setIsOpen(false)
                }}
                disabled={!model.available}
                role="option"
                aria-selected={model.id === selectedModel}
              >
                <div className="model-option-info">
                  <div className="model-option-header">
                    <span className="model-option-name">{model.name}</span>
                    {model.is_default && (
                      <span className="model-default-badge">Default</span>
                    )}
                    {model.id === selectedModel && (
                      <Check size={14} className="model-check" />
                    )}
                  </div>
                  {model.description && (
                    <span className="model-option-description">
                      {model.description}
                    </span>
                  )}
                  <div className="model-option-meta">
                    <span
                      className="model-speed"
                      style={{ color: speedLabels[model.speed]?.color }}
                    >
                      {speedLabels[model.speed]?.label || model.speed}
                    </span>
                    <span className="model-context">
                      {model.context_size.toLocaleString()} ctx
                    </span>
                    <div className="model-capabilities">
                      {model.capabilities.map(cap => {
                        const Icon = capabilityIcons[cap]
                        return (
                          <span
                            key={cap}
                            className="model-capability"
                            title={cap}
                          >
                            <Icon size={12} />
                          </span>
                        )
                      })}
                    </div>
                  </div>
                </div>
                {!model.available && (
                  <span className="model-unavailable">Offline</span>
                )}
              </button>
            ))
          )}
        </div>
      )}
    </div>
  )
}
