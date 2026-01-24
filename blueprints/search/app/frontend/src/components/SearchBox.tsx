import { useState, useEffect, useRef, KeyboardEvent } from 'react'
import { TextInput, Paper, Text, Group, ActionIcon, Kbd } from '@mantine/core'
import { IconSearch, IconX, IconMicrophone, IconCamera } from '@tabler/icons-react'
import { useNavigate } from 'react-router-dom'
import { searchApi } from '../api/search'
import { useSearchStore } from '../stores/searchStore'
import type { Suggestion } from '../types'

interface SearchBoxProps {
  initialValue?: string
  size?: 'sm' | 'md' | 'lg'
  showLogo?: boolean
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
    if (e.key === 'Enter') {
      if (selectedIndex >= 0 && suggestions[selectedIndex]) {
        handleSearch(suggestions[selectedIndex].text)
      } else {
        handleSearch(value)
      }
    } else if (e.key === 'ArrowDown') {
      e.preventDefault()
      setSelectedIndex((prev) => Math.min(prev + 1, suggestions.length - 1))
    } else if (e.key === 'ArrowUp') {
      e.preventDefault()
      setSelectedIndex((prev) => Math.max(prev - 1, -1))
    } else if (e.key === 'Escape') {
      setShowSuggestions(false)
      inputRef.current?.blur()
    }
  }

  const displaySuggestions = showSuggestions && (suggestions.length > 0 || recentSearches.length > 0)

  return (
    <div className="relative w-full max-w-2xl">
      <TextInput
        ref={inputRef}
        value={value}
        onChange={(e) => {
          setValue(e.target.value)
          setSelectedIndex(-1)
        }}
        onFocus={() => setShowSuggestions(true)}
        onBlur={() => setTimeout(() => setShowSuggestions(false), 200)}
        onKeyDown={handleKeyDown}
        placeholder="Search the web..."
        size={size}
        autoFocus={autoFocus}
        leftSection={<IconSearch size={20} className="text-gray-400" />}
        rightSection={
          <Group gap={4}>
            {value && (
              <ActionIcon
                variant="subtle"
                color="gray"
                size="sm"
                onClick={() => setValue('')}
              >
                <IconX size={16} />
              </ActionIcon>
            )}
            <ActionIcon variant="subtle" color="gray" size="sm">
              <IconMicrophone size={16} />
            </ActionIcon>
            <ActionIcon variant="subtle" color="gray" size="sm">
              <IconCamera size={16} />
            </ActionIcon>
          </Group>
        }
        rightSectionWidth={100}
        styles={{
          input: {
            borderRadius: '24px',
            border: '1px solid #dfe1e5',
            boxShadow: '0 1px 6px rgba(32,33,36,.08)',
            '&:hover': {
              boxShadow: '0 1px 6px rgba(32,33,36,.28)',
            },
            '&:focus': {
              boxShadow: '0 1px 6px rgba(32,33,36,.28)',
              borderColor: 'transparent',
            },
          },
        }}
      />

      {displaySuggestions && (
        <Paper
          shadow="md"
          className="absolute top-full left-0 right-0 mt-1 py-2 z-50 rounded-2xl overflow-hidden"
        >
          {suggestions.length > 0 ? (
            suggestions.map((suggestion, index) => (
              <div
                key={suggestion.text}
                className={`autocomplete-item flex items-center gap-3 ${
                  index === selectedIndex ? 'active' : ''
                }`}
                onMouseDown={() => handleSearch(suggestion.text)}
                onMouseEnter={() => setSelectedIndex(index)}
              >
                <IconSearch size={16} className="text-gray-400" />
                <Text size="sm">{suggestion.text}</Text>
              </div>
            ))
          ) : (
            recentSearches.slice(0, 5).map((query, index) => (
              <div
                key={query}
                className={`autocomplete-item flex items-center gap-3 ${
                  index === selectedIndex ? 'active' : ''
                }`}
                onMouseDown={() => handleSearch(query)}
                onMouseEnter={() => setSelectedIndex(index)}
              >
                <IconSearch size={16} className="text-gray-400" />
                <Text size="sm" c="dimmed">
                  {query}
                </Text>
              </div>
            ))
          )}
          <div className="px-4 py-2 border-t flex justify-end gap-2">
            <Text size="xs" c="dimmed">
              <Kbd size="xs">Enter</Kbd> to search
            </Text>
          </div>
        </Paper>
      )}
    </div>
  )
}
