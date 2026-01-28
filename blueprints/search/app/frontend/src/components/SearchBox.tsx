import { useState, useEffect, useRef, KeyboardEvent, useMemo } from 'react'
import { Search, X, Mic, Camera, Zap } from 'lucide-react'
import { useNavigate } from 'react-router-dom'
import { searchApi } from '../api/search'
import { useSearchStore } from '../stores/searchStore'
import type { Suggestion, Bang } from '../types'

// Common built-in bangs for autocomplete
const BUILTIN_BANGS: Partial<Bang>[] = [
  { trigger: 'g', name: 'Google', category: 'search' },
  { trigger: 'ddg', name: 'DuckDuckGo', category: 'search' },
  { trigger: 'yt', name: 'YouTube', category: 'video' },
  { trigger: 'w', name: 'Wikipedia', category: 'reference' },
  { trigger: 'r', name: 'Reddit', category: 'social' },
  { trigger: 'gh', name: 'GitHub', category: 'code' },
  { trigger: 'so', name: 'Stack Overflow', category: 'code' },
  { trigger: 'amz', name: 'Amazon', category: 'shopping' },
  { trigger: 'imdb', name: 'IMDb', category: 'media' },
  { trigger: 'npm', name: 'npm', category: 'code' },
  { trigger: 'go', name: 'Go Packages', category: 'code' },
  { trigger: 'mdn', name: 'MDN Web Docs', category: 'code' },
  { trigger: 'i', name: 'Images', category: 'internal' },
  { trigger: 'n', name: 'News', category: 'internal' },
  { trigger: 'v', name: 'Videos', category: 'internal' },
  { trigger: 'ai', name: 'AI Mode', category: 'internal' },
  { trigger: 'sum', name: 'Summarize', category: 'internal' },
  { trigger: '24', name: 'Past 24 hours', category: 'time' },
  { trigger: 'week', name: 'Past week', category: 'time' },
  { trigger: 'month', name: 'Past month', category: 'time' },
]

interface SearchBoxProps {
  initialValue?: string
  size?: 'sm' | 'lg'
  autoFocus?: boolean
  onSearch?: (query: string) => void
}

export function SearchBox({
  initialValue = '',
  size = 'lg',
  autoFocus = false,
  onSearch,
}: SearchBoxProps) {
  const navigate = useNavigate()
  const [value, setValue] = useState(initialValue)
  const [suggestions, setSuggestions] = useState<Suggestion[]>([])
  const [showSuggestions, setShowSuggestions] = useState(false)
  const [selectedIndex, setSelectedIndex] = useState(-1)
  const inputRef = useRef<HTMLInputElement>(null)
  const containerRef = useRef<HTMLDivElement>(null)
  const { addRecentSearch, recentSearches } = useSearchStore()

  useEffect(() => {
    setValue(initialValue)
  }, [initialValue])

  useEffect(() => {
    if (value.length < 2) {
      setSuggestions([])
      return
    }

    const timer = setTimeout(async () => {
      try {
        const results = await searchApi.suggest(value)
        setSuggestions(results || [])
      } catch {
        setSuggestions([])
      }
    }, 150)

    return () => clearTimeout(timer)
  }, [value])

  const handleSearch = (query: string) => {
    if (!query.trim()) return

    addRecentSearch(query.trim())
    setShowSuggestions(false)

    if (onSearch) {
      onSearch(query.trim())
    } else {
      navigate(`/search?q=${encodeURIComponent(query.trim())}`)
    }
  }

  const handleKeyDown = (e: KeyboardEvent<HTMLInputElement>) => {
    const items = suggestions.length > 0 ? suggestions : (recentSearches || []).slice(0, 5).map(text => ({ text, type: 'history' as const, frequency: 0 }))

    if (e.key === 'Enter') {
      if (selectedIndex >= 0 && items[selectedIndex]) {
        handleSearch(items[selectedIndex].text)
      } else {
        handleSearch(value)
      }
    } else if (e.key === 'ArrowDown') {
      e.preventDefault()
      setSelectedIndex((prev) => Math.min(prev + 1, items.length - 1))
    } else if (e.key === 'ArrowUp') {
      e.preventDefault()
      setSelectedIndex((prev) => Math.max(prev - 1, -1))
    } else if (e.key === 'Escape') {
      setShowSuggestions(false)
      inputRef.current?.blur()
    }
  }

  // Detect if user is typing a bang
  const bangMatch = useMemo(() => {
    const match = value.match(/^!(\w*)/)
    return match ? match[1] : null
  }, [value])

  // Filter bangs based on typed prefix
  const matchingBangs = useMemo(() => {
    if (bangMatch === null) return []
    const prefix = bangMatch.toLowerCase()
    return BUILTIN_BANGS.filter(b =>
      b.trigger?.toLowerCase().startsWith(prefix)
    ).slice(0, 6)
  }, [bangMatch])

  const displayItems = matchingBangs.length > 0
    ? matchingBangs.map(b => ({ text: `!${b.trigger} `, type: 'bang' as const, bang: b }))
    : suggestions.length > 0
      ? suggestions.map(s => ({ ...s, type: s.type || 'suggestion' as const }))
      : (recentSearches || []).slice(0, 5).map(text => ({ text, type: 'history' as const, frequency: 0 }))

  const showDropdown = showSuggestions && displayItems.length > 0

  return (
    <div
      ref={containerRef}
      className={`search-box-container ${size === 'lg' ? 'search-box-lg' : ''}`}
    >
      <div className="relative">
        {/* Search icon */}
        <div className="absolute left-4 top-1/2 -translate-y-1/2 text-[#9aa0a6] pointer-events-none">
          <Search size={20} />
        </div>

        {/* Input */}
        <input
          ref={inputRef}
          type="text"
          value={value}
          onChange={(e) => {
            setValue(e.target.value)
            setSelectedIndex(-1)
          }}
          onFocus={() => setShowSuggestions(true)}
          onBlur={() => setTimeout(() => setShowSuggestions(false), 200)}
          onKeyDown={handleKeyDown}
          placeholder="Search the web"
          autoFocus={autoFocus}
          className="search-input"
          style={{
            borderRadius: showDropdown ? '24px 24px 0 0' : '24px',
          }}
        />

        {/* Right icons */}
        <div className="absolute right-3 top-1/2 -translate-y-1/2 flex items-center gap-1">
          {value && (
            <button
              type="button"
              onClick={() => {
                setValue('')
                inputRef.current?.focus()
              }}
              className="p-2 text-[#70757a] hover:text-[#202124] rounded-full hover:bg-[#f1f3f4]"
            >
              <X size={18} />
            </button>
          )}
          <div className="w-px h-6 bg-[#dadce0] mx-1" />
          <button
            type="button"
            className="p-2 text-[#4285f4] hover:bg-[#f1f3f4] rounded-full"
            title="Voice search"
          >
            <Mic size={20} />
          </button>
          <button
            type="button"
            className="p-2 text-[#4285f4] hover:bg-[#f1f3f4] rounded-full"
            title="Image search"
          >
            <Camera size={20} />
          </button>
        </div>
      </div>

      {/* Autocomplete dropdown */}
      {showDropdown && (
        <div className="autocomplete-dropdown">
          {displayItems.map((item, index) => (
            <div
              key={item.text}
              className={`autocomplete-item ${index === selectedIndex ? 'active' : ''}`}
              onMouseDown={() => {
                if (item.type === 'bang') {
                  // For bangs, set the value and keep focus
                  setValue(item.text)
                  setShowSuggestions(false)
                  inputRef.current?.focus()
                } else {
                  handleSearch(item.text)
                }
              }}
              onMouseEnter={() => setSelectedIndex(index)}
            >
              {item.type === 'bang' ? (
                <>
                  <Zap size={16} className="text-[#fbbc05] shrink-0" />
                  <span className="font-mono text-[#1a73e8]">{item.text.trim()}</span>
                  <span className="text-[#70757a] ml-2">{(item as any).bang?.name}</span>
                </>
              ) : (
                <>
                  <Search size={16} className="text-[#9aa0a6] shrink-0" />
                  <span className="truncate">{item.text}</span>
                </>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
