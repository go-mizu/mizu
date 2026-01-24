import { useState, useEffect, useRef, KeyboardEvent } from 'react'
import { Search, X, Mic, Camera } from 'lucide-react'
import { useNavigate } from 'react-router-dom'
import { searchApi } from '../api/search'
import { useSearchStore } from '../stores/searchStore'
import type { Suggestion } from '../types'

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
        setSuggestions(results)
      } catch {
        // Ignore errors
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
    const items = suggestions.length > 0 ? suggestions : recentSearches.slice(0, 5).map(text => ({ text, type: 'history' as const, frequency: 0 }))

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

  const displayItems = suggestions.length > 0
    ? suggestions
    : recentSearches.slice(0, 5).map(text => ({ text, type: 'history' as const, frequency: 0 }))

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
              onMouseDown={() => handleSearch(item.text)}
              onMouseEnter={() => setSelectedIndex(index)}
            >
              <Search size={16} className="text-[#9aa0a6] shrink-0" />
              <span className="truncate">{item.text}</span>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
