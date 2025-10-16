'use client'

import { useState, useEffect, useRef, useId } from 'react'
import { getSearchSuggestions } from '@/lib/api'

interface Suggestion {
  type: 'work' | 'tag' | 'author'
  value: string
  count?: number
  id?: string
}

interface SearchAutocompleteProps {
  value: string
  onChange: (value: string) => void
  onSearch: (query: string) => void
  placeholder?: string
  className?: string
}

export default function SearchAutocomplete({
  value,
  onChange,
  onSearch,
  placeholder = "Search for works, fandoms, characters, tags...",
  className = ""
}: SearchAutocompleteProps) {
  const [suggestions, setSuggestions] = useState<Suggestion[]>([])
  const [isOpen, setIsOpen] = useState(false)
  const [selectedIndex, setSelectedIndex] = useState(-1)
  const [loading, setLoading] = useState(false)
  const [announceText, setAnnounceText] = useState('')
  const inputRef = useRef<HTMLInputElement>(null)
  const suggestionsRef = useRef<HTMLUListElement>(null)
  
  // Generate stable IDs for accessibility
  const suggestionListId = useId()
  const helpTextId = useId()
  const statusId = useId()

  useEffect(() => {
    const fetchSuggestions = async () => {
      if (value.length < 2) {
        setSuggestions([])
        setIsOpen(false)
        return
      }

      setLoading(true)
      try {
        const data = await getSearchSuggestions(value)
        
        const allSuggestions: Suggestion[] = [
          ...data.works.map((work: any) => ({
            type: 'work' as const,
            value: work.title || work.value,
            id: work.id,
            count: work.kudos
          })),
          ...data.tags.map((tag: any) => ({
            type: 'tag' as const,
            value: tag.name || tag.value,
            count: tag.count
          })),
          ...data.authors.map((author: any) => ({
            type: 'author' as const,
            value: author.name || author.value,
            count: author.work_count
          }))
        ]

        const limitedSuggestions = allSuggestions.slice(0, 8)
        setSuggestions(limitedSuggestions)
        setIsOpen(limitedSuggestions.length > 0)
        setSelectedIndex(-1)
        
        // Announce results to screen readers
        if (limitedSuggestions.length > 0) {
          setAnnounceText(`${limitedSuggestions.length} suggestions available. Use arrow keys to navigate.`)
        } else {
          setAnnounceText('No suggestions found.')
        }
      } catch (error) {
        console.error('Failed to fetch suggestions:', error)
        setSuggestions([])
        setIsOpen(false)
        setAnnounceText('Search suggestions unavailable. Please try again.')
      } finally {
        setLoading(false)
      }
    }

    const timeoutId = setTimeout(fetchSuggestions, 300)
    return () => clearTimeout(timeoutId)
  }, [value])

  const handleKeyDown = (e: React.KeyboardEvent) => {
    switch (e.key) {
      case 'ArrowDown':
        e.preventDefault()
        if (!isOpen && suggestions.length > 0) {
          setIsOpen(true)
          setSelectedIndex(0)
          setAnnounceText(`Suggestions opened. ${suggestions[0].value} selected.`)
        } else if (isOpen && selectedIndex < suggestions.length - 1) {
          const newIndex = selectedIndex + 1
          setSelectedIndex(newIndex)
          setAnnounceText(`${suggestions[newIndex].value} selected.`)
        }
        break
        
      case 'ArrowUp':
        e.preventDefault()
        if (isOpen && selectedIndex > 0) {
          const newIndex = selectedIndex - 1
          setSelectedIndex(newIndex)
          setAnnounceText(`${suggestions[newIndex].value} selected.`)
        } else if (isOpen && selectedIndex === 0) {
          setSelectedIndex(-1)
          setAnnounceText('Back to search input.')
        }
        break
        
      case 'Enter':
        e.preventDefault()
        if (isOpen && selectedIndex >= 0 && selectedIndex < suggestions.length) {
          const suggestion = suggestions[selectedIndex]
          onChange(suggestion.value)
          onSearch(suggestion.value)
          setIsOpen(false)
          setSelectedIndex(-1)
          setAnnounceText(`Selected ${suggestion.value}. Searching...`)
        } else {
          onSearch(value)
          setIsOpen(false)
          setSelectedIndex(-1)
          setAnnounceText(`Searching for ${value}...`)
        }
        break
        
      case 'Escape':
        if (isOpen) {
          e.preventDefault()
          setIsOpen(false)
          setSelectedIndex(-1)
          setAnnounceText('Suggestions closed.')
          inputRef.current?.focus()
        }
        break
    }
  }

  const handleSuggestionClick = (suggestion: Suggestion) => {
    onChange(suggestion.value)
    onSearch(suggestion.value)
    setIsOpen(false)
    setSelectedIndex(-1)
    setAnnounceText(`Selected ${suggestion.value}. Searching...`)
    inputRef.current?.focus()
  }
  
  const getCountLabel = (type: string, count?: number) => {
    if (count === undefined) return ''
    switch (type) {
      case 'work':
        return count === 1 ? 'kudo' : 'kudos'
      case 'author':
        return count === 1 ? 'work' : 'works'
      case 'tag':
        return count === 1 ? 'use' : 'uses'
      default:
        return ''
    }
  }
  
  const getSuggestionDescription = (suggestion: Suggestion) => {
    const typeLabel = getSuggestionTypeLabel(suggestion.type)
    const countLabel = suggestion.count !== undefined 
      ? ` with ${suggestion.count} ${getCountLabel(suggestion.type, suggestion.count)}`
      : ''
    return `${typeLabel}${countLabel}`
  }

  const getSuggestionIcon = (type: string) => {
    switch (type) {
      case 'work':
        return '📚'
      case 'tag':
        return '🏷️'
      case 'author':
        return '👤'
      default:
        return '🔍'
    }
  }

  const getSuggestionTypeLabel = (type: string) => {
    switch (type) {
      case 'work':
        return 'Work'
      case 'tag':
        return 'Tag'
      case 'author':
        return 'Author'
      default:
        return ''
    }
  }

  return (
    <div className={`relative ${className}`}>
      {/* Screen reader announcements */}
      <div 
        id={statusId}
        aria-live="polite" 
        aria-atomic="true"
        className="sr-only"
        role="status"
      >
        {announceText}
      </div>
      
      {/* Loading status announcement */}
      <div 
        aria-live="polite" 
        aria-atomic="true"
        className="sr-only"
        role="status"
      >
        {loading ? 'Searching for suggestions...' : ''}
      </div>
      
      <div className="relative">
        <input
          ref={inputRef}
          type="text"
          role="combobox"
          aria-label={placeholder}
          aria-expanded={isOpen}
          aria-autocomplete="list"
          aria-controls={isOpen ? suggestionListId : undefined}
          aria-describedby={helpTextId}
          aria-activedescendant={
            isOpen && selectedIndex >= 0 
              ? `suggestion-${selectedIndex}`
              : undefined
          }
          value={value}
          onChange={(e) => onChange(e.target.value)}
          onKeyDown={handleKeyDown}
          onFocus={() => suggestions.length > 0 && setIsOpen(true)}
          placeholder={placeholder}
          className="w-full px-4 py-3 text-lg border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 pr-10"
          autoComplete="off"
        />
        
        {/* Search help text */}
        <div id={helpTextId} className="sr-only">
          Type to search for works, fandoms, characters, or tags. Use arrow keys to navigate suggestions, Enter to select, Escape to close.
        </div>
        
        {loading && (
          <div 
            className="absolute right-3 top-1/2 transform -translate-y-1/2"
            aria-hidden="true"
          >
            <div 
              data-testid="loading-spinner" 
              className="animate-spin rounded-full h-5 w-5 border-b-2 border-blue-600"
              role="img"
              aria-label="Loading suggestions"
            ></div>
          </div>
        )}
      </div>

      {isOpen && suggestions.length > 0 && (
        <ul
          ref={suggestionsRef}
          id={suggestionListId}
          role="listbox"
          aria-label={`Search suggestions, ${suggestions.length} available`}
          className="absolute z-50 w-full mt-1 bg-white border border-gray-300 rounded-lg shadow-lg max-h-64 overflow-y-auto"
        >
          {suggestions.map((suggestion, index) => (
            <li
              key={`${suggestion.type}-${suggestion.value}-${index}`}
              id={`suggestion-${index}`}
              role="option"
              aria-selected={index === selectedIndex}
              aria-describedby={`suggestion-desc-${index}`}
              onClick={() => handleSuggestionClick(suggestion)}
              className={`cursor-pointer border-b border-gray-100 last:border-b-0 ${
                index === selectedIndex 
                  ? 'bg-blue-50 border-blue-200' 
                  : 'hover:bg-gray-50'
              }`}
            >
              <div className="w-full px-4 py-3 text-left flex items-center justify-between">
                <div className="flex items-center space-x-3 flex-1 min-w-0">
                  <span 
                    className="text-lg" 
                    aria-hidden="true"
                    role="img"
                    aria-label={getSuggestionTypeLabel(suggestion.type)}
                  >
                    {getSuggestionIcon(suggestion.type)}
                  </span>
                  <div className="flex-1 min-w-0">
                    <div className="font-medium text-gray-900 truncate">
                      {suggestion.value}
                    </div>
                    <div 
                      id={`suggestion-desc-${index}`}
                      className="text-sm text-gray-500"
                    >
                      {getSuggestionDescription(suggestion)}
                    </div>
                  </div>
                </div>
                <div className="text-gray-400 ml-2" aria-hidden="true">
                  <svg 
                    className="w-4 h-4" 
                    fill="none" 
                    stroke="currentColor" 
                    viewBox="0 0 24 24"
                    role="img"
                    aria-label="Select this suggestion"
                  >
                    <path 
                      strokeLinecap="round" 
                      strokeLinejoin="round" 
                      strokeWidth={2} 
                      d="M9 5l7 7-7 7" 
                    />
                  </svg>
                </div>
              </div>
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}