'use client';

import { useState, useRef, useEffect } from 'react';
import { getTagSuggestions, TagSuggestion } from '@/lib/api';

interface TagInputProps {
  value: string;
  onChange: (value: string) => void;
  onTagAdd?: (tagName: string, suggestion?: TagSuggestion) => void;
  placeholder?: string;
  tagType?: string[];
  fandomId?: string;
  className?: string;
  disabled?: boolean;
}

export default function TagInput({
  value,
  onChange,
  onTagAdd,
  placeholder = "Type to search for tags...",
  tagType,
  fandomId,
  className = '',
  disabled = false
}: TagInputProps) {
  const [suggestions, setSuggestions] = useState<TagSuggestion[]>([]);
  const [showSuggestions, setShowSuggestions] = useState(false);
  const [selectedIndex, setSelectedIndex] = useState(-1);
  const [loading, setLoading] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);
  const suggestionRefs = useRef<(HTMLDivElement | null)[]>([]);

  // Debounced search for suggestions
  useEffect(() => {
    if (!value.trim() || value.endsWith(',') || disabled) {
      setSuggestions([]);
      setShowSuggestions(false);
      return;
    }

    const timeoutId = setTimeout(async () => {
      try {
        setLoading(true);
        const response = await getTagSuggestions({
          query: value.trim(),
          type: tagType,
          fandom_id: fandomId,
          limit: 10,
        });
        setSuggestions(response?.suggestions || []);
        setShowSuggestions((response?.suggestions?.length || 0) > 0);
        setSelectedIndex(-1);
      } catch (error) {
        console.error('Failed to fetch tag suggestions:', error);
        setSuggestions([]);
        setShowSuggestions(false);
      } finally {
        setLoading(false);
      }
    }, 300);

    return () => clearTimeout(timeoutId);
  }, [value, tagType, fandomId, disabled]);

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (!showSuggestions) return;

    switch (e.key) {
      case 'ArrowDown':
        e.preventDefault();
        setSelectedIndex(prev => 
          prev < suggestions.length - 1 ? prev + 1 : 0
        );
        break;
        
      case 'ArrowUp':
        e.preventDefault();
        setSelectedIndex(prev => 
          prev > 0 ? prev - 1 : suggestions.length - 1
        );
        break;
        
      case 'Enter':
        e.preventDefault();
        if (selectedIndex >= 0 && suggestions[selectedIndex]) {
          selectSuggestion(suggestions[selectedIndex]);
        } else if (value.trim()) {
          // Add as new tag if no suggestion selected
          handleTagAdd(value.trim());
        }
        break;
        
      case 'Escape':
        setShowSuggestions(false);
        setSelectedIndex(-1);
        break;
        
      case 'Tab':
        if (selectedIndex >= 0 && suggestions[selectedIndex]) {
          e.preventDefault();
          selectSuggestion(suggestions[selectedIndex]);
        }
        break;
    }
  };

  const selectSuggestion = (suggestion: TagSuggestion) => {
    handleTagAdd(suggestion.name, suggestion);
  };

  const handleTagAdd = (tagName: string, suggestion?: TagSuggestion) => {
    if (onTagAdd) {
      onTagAdd(tagName, suggestion);
    }
    onChange('');
    setShowSuggestions(false);
    setSelectedIndex(-1);
  };

  const handleSuggestionClick = (suggestion: TagSuggestion) => {
    selectSuggestion(suggestion);
  };

  // Scroll selected suggestion into view
  useEffect(() => {
    if (selectedIndex >= 0 && suggestionRefs.current[selectedIndex]) {
      suggestionRefs.current[selectedIndex]?.scrollIntoView({
        block: 'nearest',
        behavior: 'smooth'
      });
    }
  }, [selectedIndex]);

  const getProminenceIndicator = (suggestion: TagSuggestion) => {
    if (suggestion.canonical) {
      return <span className="text-green-600 text-xs font-medium">CANONICAL</span>;
    }
    if (suggestion.use_count > 100) {
      return <span className="text-blue-600 text-xs">POPULAR</span>;
    }
    return null;
  };

  const getTypeColor = (type: string) => {
    switch (type) {
      case 'fandom': return 'text-purple-600';
      case 'character': return 'text-green-600';
      case 'relationship': return 'text-red-600';
      case 'freeform': return 'text-blue-600';
      default: return 'text-slate-600';
    }
  };

  return (
    <div className={`relative ${className}`}>
      <input
        ref={inputRef}
        type="text"
        value={value}
        onChange={(e) => onChange(e.target.value)}
        onKeyDown={handleKeyDown}
        onFocus={() => {
          if (suggestions.length > 0) {
            setShowSuggestions(true);
          }
        }}
        onBlur={() => {
          // Delay hiding to allow click on suggestions
          setTimeout(() => setShowSuggestions(false), 150);
        }}
        placeholder={placeholder}
        disabled={disabled}
        className="w-full px-3 py-2 border border-slate-300 rounded-md focus:outline-none focus:ring-2 focus:ring-orange-500 focus:border-transparent disabled:bg-slate-100 disabled:cursor-not-allowed"
      />
      
      {loading && (
        <div className="absolute right-3 top-1/2 transform -translate-y-1/2">
          <div className="animate-spin h-4 w-4 border-2 border-orange-500 border-t-transparent rounded-full"></div>
        </div>
      )}

      {showSuggestions && suggestions.length > 0 && (
        <div className="absolute z-50 w-full mt-1 bg-white border border-slate-200 rounded-md shadow-lg max-h-60 overflow-y-auto">
          {suggestions.map((suggestion, index) => (
            <div
              key={suggestion.id}
              ref={el => { suggestionRefs.current[index] = el; }}
              onClick={() => handleSuggestionClick(suggestion)}
              className={`px-3 py-2 cursor-pointer border-b border-slate-100 last:border-b-0 ${
                index === selectedIndex 
                  ? 'bg-orange-50 border-orange-200' 
                  : 'hover:bg-slate-50'
              }`}
            >
              <div className="flex items-center justify-between">
                <div className="flex-1 min-w-0">
                  <div className="font-medium text-slate-900 truncate">
                    {suggestion.name}
                  </div>
                  <div className="flex items-center gap-2 mt-1">
                    <span className={`text-xs font-medium uppercase ${getTypeColor(suggestion.type)}`}>
                      {suggestion.type}
                    </span>
                    {suggestion.use_count > 0 && (
                      <span className="text-xs text-slate-500">
                        {suggestion.use_count.toLocaleString()} uses
                      </span>
                    )}
                    {getProminenceIndicator(suggestion)}
                  </div>
                </div>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}