import { useState, useEffect, useRef } from 'react'
import { ChevronDown, Server, Cloud, Check } from 'lucide-react'
import type { Provider } from '../../types/ai'

interface ProviderSelectorProps {
  providers: Provider[]
  selected?: string
  onSelect: (providerId: string) => void
  size?: 'sm' | 'md'
}

export function ProviderSelector({
  providers,
  selected,
  onSelect,
  size = 'md',
}: ProviderSelectorProps) {
  const [isOpen, setIsOpen] = useState(false)
  const dropdownRef = useRef<HTMLDivElement>(null)

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

  const currentProvider = providers.find(p => p.id === selected)

  const getIcon = (type: string) => {
    return type === 'cloud' ? Cloud : Server
  }

  return (
    <div className={`provider-selector ${size}`} ref={dropdownRef}>
      <button
        type="button"
        className="provider-selector-button"
        onClick={() => setIsOpen(!isOpen)}
        aria-expanded={isOpen}
        aria-haspopup="listbox"
      >
        <div className="provider-selector-current">
          {currentProvider ? (
            <>
              {(() => {
                const Icon = getIcon(currentProvider.type)
                return <Icon size={14} className="provider-icon" />
              })()}
              <span className="provider-name">{currentProvider.name}</span>
            </>
          ) : (
            <span className="provider-placeholder">Select provider</span>
          )}
        </div>
        <ChevronDown
          size={16}
          className={`provider-selector-chevron ${isOpen ? 'open' : ''}`}
        />
      </button>

      {isOpen && (
        <div className="provider-selector-dropdown" role="listbox">
          {providers.length === 0 ? (
            <div className="provider-selector-empty">No providers available</div>
          ) : (
            providers.map(provider => {
              const Icon = getIcon(provider.type)
              return (
                <button
                  key={provider.id}
                  type="button"
                  className={`provider-option ${provider.id === selected ? 'selected' : ''} ${!provider.available ? 'unavailable' : ''}`}
                  onClick={() => {
                    onSelect(provider.id)
                    setIsOpen(false)
                  }}
                  disabled={!provider.available}
                  role="option"
                  aria-selected={provider.id === selected}
                >
                  <div className="provider-option-info">
                    <Icon size={16} className="provider-option-icon" />
                    <div className="provider-option-details">
                      <span className="provider-option-name">{provider.name}</span>
                      <span className="provider-option-type">
                        {provider.type === 'cloud' ? 'Cloud' : 'Local'}
                      </span>
                    </div>
                    {provider.id === selected && (
                      <Check size={14} className="provider-check" />
                    )}
                  </div>
                  {!provider.available && (
                    <span className="provider-unavailable">Offline</span>
                  )}
                </button>
              )
            })
          )}
        </div>
      )}

      <style>{`
        .provider-selector {
          position: relative;
          display: inline-block;
        }

        .provider-selector.sm {
          font-size: 0.875rem;
        }

        .provider-selector-button {
          display: flex;
          align-items: center;
          gap: 0.5rem;
          padding: 0.5rem 0.75rem;
          background: var(--bg-secondary, #f5f5f5);
          border: 1px solid var(--border-color, #e0e0e0);
          border-radius: 0.5rem;
          cursor: pointer;
          min-width: 120px;
          justify-content: space-between;
        }

        .provider-selector-button:hover {
          background: var(--bg-hover, #ebebeb);
        }

        .provider-selector-current {
          display: flex;
          align-items: center;
          gap: 0.5rem;
        }

        .provider-icon {
          color: var(--text-secondary, #666);
        }

        .provider-name {
          font-weight: 500;
        }

        .provider-placeholder {
          color: var(--text-tertiary, #999);
        }

        .provider-selector-chevron {
          transition: transform 0.2s;
          color: var(--text-secondary, #666);
        }

        .provider-selector-chevron.open {
          transform: rotate(180deg);
        }

        .provider-selector-dropdown {
          position: absolute;
          top: 100%;
          left: 0;
          right: 0;
          margin-top: 0.25rem;
          background: var(--bg-primary, #fff);
          border: 1px solid var(--border-color, #e0e0e0);
          border-radius: 0.5rem;
          box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
          z-index: 100;
          overflow: hidden;
        }

        .provider-option {
          display: flex;
          align-items: center;
          justify-content: space-between;
          width: 100%;
          padding: 0.75rem;
          background: none;
          border: none;
          cursor: pointer;
          text-align: left;
        }

        .provider-option:hover:not(:disabled) {
          background: var(--bg-hover, #f5f5f5);
        }

        .provider-option.selected {
          background: var(--bg-selected, #e3f2fd);
        }

        .provider-option.unavailable {
          opacity: 0.5;
          cursor: not-allowed;
        }

        .provider-option-info {
          display: flex;
          align-items: center;
          gap: 0.75rem;
        }

        .provider-option-icon {
          color: var(--text-secondary, #666);
        }

        .provider-option-details {
          display: flex;
          flex-direction: column;
        }

        .provider-option-name {
          font-weight: 500;
        }

        .provider-option-type {
          font-size: 0.75rem;
          color: var(--text-tertiary, #999);
        }

        .provider-check {
          color: var(--primary, #1976d2);
        }

        .provider-unavailable {
          font-size: 0.75rem;
          color: var(--error, #d32f2f);
        }

        .provider-selector-empty {
          padding: 1rem;
          text-align: center;
          color: var(--text-tertiary, #999);
        }
      `}</style>
    </div>
  )
}
