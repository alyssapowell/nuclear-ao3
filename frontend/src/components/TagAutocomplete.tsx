'use client'

import React, { useState, useEffect, useRef, useId, useCallback, useMemo } from 'react'
import { searchTags } from '@/lib/api'
// Temporarily disable render profiler to fix loading issues
// import { useRenderProfiler, useWhyDidYouUpdate, PerformanceBoundary } from '@/utils/renderProfiler'

interface TagAutocompleteProps {
  id?: string
  value: string
  onChange: (value: string) => void
  onTagSelect?: (tag: any) => void
  placeholder: string
  className?: string
  tagType?: 'fandom' | 'character' | 'relationship' | 'freeform'
  required?: boolean
  'aria-describedby'?: string
  disabled?: boolean
  label?: string
}

interface Tag {
  name: string
  type: string
  use_count?: number
  id?: string
  is_new?: boolean
}

const TagAutocomplete = React.memo(function TagAutocomplete({ 
  id: propId, 
  value, 
  onChange, 
  onTagSelect,
  placeholder, 
  className, 
  tagType, 
  required,
  'aria-describedby': ariaDescribedBy,
  disabled = false,
  label
}: TagAutocompleteProps) {
  // Profiling hooks to track renders
  // Temporarily disable render profiler to fix loading issues
  // useRenderProfiler('TagAutocomplete', {
  //   propId,
  //   value,
  //   onChange,
  //   onTagSelect,
  //   placeholder,
  //   className,
  //   tagType,
  //   required,
  //   ariaDescribedBy,
  //   disabled,
  //   label
  // })
  
  // useWhyDidYouUpdate('TagAutocomplete', {
  //   propId,
  //   value,
  //   onChange,
  //   onTagSelect,
  //   placeholder,
  //   className,
  //   tagType,
  //   required,
  //   ariaDescribedBy,
  //   disabled,
  //   label
  // })

  const [suggestions, setSuggestions] = useState<Tag[]>([])
  const [showSuggestions, setShowSuggestions] = useState(false)
  const [activeSuggestion, setActiveSuggestion] = useState(-1)
  const [announceUpdate, setAnnounceUpdate] = useState('')
  const inputRef = useRef<HTMLInputElement>(null)
  const suggestionsRef = useRef<HTMLDivElement>(null)
  const shouldMaintainFocus = useRef(false)

  // Generate unique IDs for accessibility
  const generatedId = useId()
  const inputId = propId || generatedId
  const listboxId = `${inputId}-listbox`
  const liveRegionId = `${inputId}-live-region`
  const helpId = `${inputId}-help`

  // Get current tag being typed (last incomplete tag)
  const getCurrentTag = useCallback(() => {
    if (!value || typeof value !== 'string') return ''
    const tags = value.split(',')
    const lastTag = tags[tags.length - 1]?.trim() || ''
    return lastTag
  }, [value])

  // Search for suggestions when user types
  useEffect(() => {
    const currentTag = getCurrentTag()
    
    if (currentTag.length >= 2 && !disabled) {
      const searchDelay = setTimeout(async () => {
        try {
          const results = await searchTags(currentTag, tagType)
          
          // Just show existing suggestions - new tags are auto-created on Enter/comma/tab
          setSuggestions(results)
          setShowSuggestions(true)
          setActiveSuggestion(-1)
          setAnnounceUpdate(`${results.length} suggestions available. Use arrow keys to navigate.`)
        } catch (error) {
          setSuggestions([])
          setShowSuggestions(false)
          setAnnounceUpdate('No suggestions found')
        }
      }, 150) // Debounce for 150ms - faster response

      return () => clearTimeout(searchDelay)
    } else {
      setSuggestions([])
      setShowSuggestions(false)
      setAnnounceUpdate('')
    }
  }, [value, tagType, disabled, getCurrentTag])

  // Maintain focus if the user was actively typing
  useEffect(() => {
    if (shouldMaintainFocus.current && inputRef.current) {
      const input = inputRef.current
      const currentFocus = document.activeElement
      if (currentFocus !== input) {
        input.focus()
      }
      shouldMaintainFocus.current = false
    }
  })

  // Handle suggestion selection
  const selectSuggestion = useCallback((tagName: string) => {
    if (!value || typeof value !== 'string') {
      onChange(tagName + ', ')
    } else {
      const tags = value.split(',').map(tag => tag.trim()).filter(tag => tag.length > 0)
      tags[tags.length - 1] = tagName // Replace the current incomplete tag
      onChange(tags.join(', ') + ', ') // Add comma and space for next tag
    }
    setShowSuggestions(false)
    setActiveSuggestion(-1)
    setAnnounceUpdate(`Selected ${tagName}`)
    inputRef.current?.focus()
    
    // Call onTagSelect if provided
    if (onTagSelect) {
      onTagSelect({ name: tagName })
    }
  }, [value, onChange, onTagSelect])

  // Handle keyboard navigation
  const handleKeyDown = useCallback((e: React.KeyboardEvent) => {
    const currentTag = getCurrentTag().trim()

    switch (e.key) {
      case 'ArrowDown':
        if (!showSuggestions || disabled) return
        e.preventDefault()
        const nextIndex = activeSuggestion < suggestions.length - 1 ? activeSuggestion + 1 : activeSuggestion
        setActiveSuggestion(nextIndex)
        if (nextIndex !== activeSuggestion && suggestions[nextIndex]) {
          setAnnounceUpdate(`${suggestions[nextIndex].name} - ${suggestions[nextIndex].use_count} works`)
        }
        break
      case 'ArrowUp':
        if (!showSuggestions || disabled) return
        e.preventDefault()
        const prevIndex = activeSuggestion > 0 ? activeSuggestion - 1 : -1
        setActiveSuggestion(prevIndex)
        if (prevIndex >= 0 && suggestions[prevIndex]) {
          setAnnounceUpdate(`${suggestions[prevIndex].name} - ${suggestions[prevIndex].use_count} works`)
        }
        break
      case 'Enter':
        e.preventDefault()
        e.stopPropagation() // Prevent event bubbling to form
        if (showSuggestions && activeSuggestion >= 0 && suggestions[activeSuggestion]) {
          // Select from suggestions
          selectSuggestion(suggestions[activeSuggestion].name)
        } else if (currentTag && currentTag.length > 0 && tagType !== 'fandom') {
          // Auto-create new tag if user typed something (except fandoms)
          selectSuggestion(currentTag)
        }
        break
      case ',':
        // Auto-create tag on comma (except fandoms)
        if (currentTag && currentTag.length > 0 && tagType !== 'fandom') {
          e.preventDefault()
          selectSuggestion(currentTag)
        }
        break
      case 'Escape':
        e.preventDefault()
        setShowSuggestions(false)
        setActiveSuggestion(-1)
        setAnnounceUpdate('Suggestions closed')
        break
      case 'Tab':
        // Allow normal tab behavior to close suggestions, or auto-create if typing (except fandoms)
        if (currentTag && currentTag.length > 0 && tagType !== 'fandom') {
          e.preventDefault()
          selectSuggestion(currentTag)
        } else {
          setShowSuggestions(false)
          setActiveSuggestion(-1)
        }
        break
    }
  }, [showSuggestions, disabled, activeSuggestion, suggestions, selectSuggestion, getCurrentTag])

  // Hide suggestions when clicking outside
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (
        inputRef.current && 
        !inputRef.current.contains(event.target as Node) &&
        suggestionsRef.current &&
        !suggestionsRef.current.contains(event.target as Node)
      ) {
        setShowSuggestions(false)
      }
    }

    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [])

  return (
    <section className="relative" role="combobox" aria-label="Tag search and autocomplete">
      {/* Live Region for Announcements */}
      <div
        id={liveRegionId}
        aria-live="assertive"
        aria-atomic="true"
        className="sr-only"
      >
        {announceUpdate}
      </div>

      {/* Input Field */}
      <input
        ref={inputRef}
        type="text"
        id={inputId}
        value={value}
        onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
          shouldMaintainFocus.current = true
          onChange(e.target.value)
        }}
        onKeyDown={handleKeyDown}
        onFocus={() => {
          if (suggestions.length > 0 && !disabled) {
            setShowSuggestions(true)
          }
        }}
        className={className}
        placeholder={placeholder}
        required={required}
        disabled={disabled}
        autoComplete="off"
        role="combobox"
        aria-expanded={showSuggestions}
        aria-owns={showSuggestions ? listboxId : undefined}
        aria-activedescendant={useMemo(() => 
          activeSuggestion >= 0 && suggestions[activeSuggestion] 
            ? `${listboxId}-option-${activeSuggestion}` 
            : undefined
        , [activeSuggestion, suggestions, listboxId])}
        aria-describedby={useMemo(() => [
          ariaDescribedBy,
          helpId,
          showSuggestions ? `${listboxId}-status` : undefined
        ].filter(Boolean).join(' '), [ariaDescribedBy, helpId, showSuggestions, listboxId])}
        aria-label={label || `Tag autocomplete input for ${tagType || 'tags'}`}
        aria-autocomplete="list"
      />

      {/* Help Text */}
      <div id={helpId} className="sr-only">
        Type to search for tags. Use arrow keys to navigate suggestions, Enter to select, Escape to close.
      </div>
      
      {/* Suggestions Dropdown */}
      {showSuggestions && suggestions.length > 0 && (
        <aside 
          ref={suggestionsRef}
          id={listboxId}
          role="listbox"
          aria-label={`Tag suggestions for ${tagType || 'tags'}`}
          className="absolute z-50 w-full mt-1 bg-white border border-gray-300 rounded-md shadow-lg max-h-60 overflow-y-auto"
        >
          {/* Status announcement for screen readers */}
          <div id={`${listboxId}-status`} className="sr-only" aria-live="polite">
            {suggestions.length} suggestions available
          </div>

          {suggestions.map((tag, index) => (
            <article
              key={`${tag.name}-${index}`}
              id={`${listboxId}-option-${index}`}
              role="option"
              aria-selected={index === activeSuggestion}
              className={`px-3 py-2 cursor-pointer hover:bg-blue-50 focus:bg-blue-50 ${
                index === activeSuggestion ? 'bg-blue-100' : ''
              }`}
              onClick={() => selectSuggestion(tag.name)}
              onMouseEnter={() => setActiveSuggestion(index)}
              tabIndex={-1}
            >
              <header className="flex items-center justify-between">
                <span className="text-gray-900">{tag.name}</span>
                {tag.use_count && tag.use_count > 0 && (
                  <span className="text-xs text-gray-500" aria-label={`${tag.use_count} works`}>
                    {tag.use_count} works
                  </span>
                )}
              </header>
              {tag.type && (
                <footer className="text-xs text-gray-400 capitalize" aria-label={`Tag type: ${tag.type}`}>
                  {tag.type}
                </footer>
              )}
            </article>
          ))}
          
          {/* Help text for new tag creation */}
          {suggestions.length === 0 && (
            <div className="px-3 py-2 text-xs text-gray-500">
              {tagType === 'fandom' 
                ? 'No fandoms found. Please choose from existing fandoms or contact an admin to add new ones.'
                : 'No existing tags found. Press Enter, comma, or Tab to create a new tag.'
              }
            </div>
          )}
        </aside>
      )}
    </section>
  )
});

export default TagAutocomplete;