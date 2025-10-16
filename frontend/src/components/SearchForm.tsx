'use client';

import React, { useState, useRef, useId, useCallback, useMemo } from 'react';
import { Search, Filter, X, AlertCircle, Loader2 } from 'lucide-react';
import { searchWorks } from '@/lib/api';
import TagAutocomplete from './TagAutocomplete';
import { SmartRecommendation } from './SmartRecommendations';
// Temporarily disable render profiler to fix loading issues
// import { useRenderProfiler, useWhyDidYouUpdate, PerformanceBoundary } from '@/utils/renderProfiler';

// Separate memoized TagField component to prevent recreation
const MemoizedTagField = React.memo(function TagField({ 
  label, 
  field, 
  placeholder, 
  required = false,
  formId,
  tags,
  tagInput,
  onTagInputChange,
  onTagSelect,
  onRemoveTag
}: { 
  label: string; 
  field: string; 
  placeholder: string;
  required?: boolean;
  formId: string;
  tags: string[];
  tagInput: string;
  onTagInputChange: (value: string) => void;
  onTagSelect: (tag: any) => void;
  onRemoveTag: (index: number, tagName: string) => void;
}) {
  const fieldId = `${formId}-${field}`;
  
  return (
    <section className="space-y-2" aria-labelledby={fieldId}>
      <label htmlFor={fieldId} className="block text-sm font-medium text-gray-700">
        {label}
        {required && <span className="text-red-500 ml-1" aria-label="required">*</span>}
      </label>
      
      {/* Selected Tags */}
      {tags && tags.length > 0 && (
        <section 
          className="flex flex-wrap gap-2 mb-2"
          role="group"
          aria-label={`Selected ${label.toLowerCase()}`}
        >
          {tags.map((tag, index) => (
            <span 
              key={index} 
              className="inline-flex items-center px-3 py-1 bg-blue-100 text-blue-800 text-sm rounded-full"
            >
              <span className="mr-2">{tag}</span>
              <button
                type="button"
                onClick={() => onRemoveTag(index, tag)}
                className="text-blue-600 hover:text-blue-800 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-1 rounded-full p-0.5"
                aria-label={`Remove ${tag} from ${label.toLowerCase()}`}
                title={`Remove ${tag}`}
              >
                <X className="h-3 w-3" aria-hidden="true" />
              </button>
            </span>
          ))}
        </section>
      )}
      
      <TagAutocomplete
        key={fieldId}
        id={fieldId}
        value={tagInput}
        onChange={onTagInputChange}
        onTagSelect={onTagSelect}
        placeholder={placeholder}
        className="w-full"
        tagType={field === 'fandoms' ? 'fandom' : field === 'freeformTags' ? 'freeform' : field === 'characters' ? 'character' : field === 'relationships' ? 'relationship' : undefined}
        aria-describedby={`${fieldId}-help`}
      />
      <div id={`${fieldId}-help`} className="text-xs text-gray-500">
        Start typing to see suggestions, then select to add tags
      </div>
    </section>
  );
});

interface SearchFilters {
  title?: string;
  author?: string;
  relationships: string[];
  characters: string[];
  freeformTags: string[];
  fandoms: string[];
  rating?: string;
  wordCountMin?: number;
  wordCountMax?: number;
  language?: string;
  status?: string;
  excludePoorlyTagged?: boolean;
  enableSmartSuggestions?: boolean;
}

interface Work {
  id: string;
  title: string;
  author: string;
  summary?: string;
  word_count: number;
  chapter_count: number;
  max_chapters?: number;
  rating: string;
  status: string;
  language: string;
  published_date: string;
  updated_date: string;
  relationships: Array<{ name: string; category: string }>;
  characters: Array<{ name: string; category: string }>;
  freeform_tags: Array<{ name: string; category: string }>;
  fandoms: Array<{ name: string; category: string }>;
  kudos_count?: number;
  bookmark_count?: number;
  hit_count?: number;
  comment_count?: number;
  tag_quality_score?: number;
  missing_tag_suggestions?: string[];
}



interface SearchFormProps {
  onResults: (results: Work[]) => void;
  onRecommendations: (recommendations: SmartRecommendation[]) => void;
  className?: string;
  initialFilters?: Partial<SearchFilters>;
}

const SearchForm = React.memo(function SearchForm({ 
  onResults, 
  onRecommendations, 
  className = '',
  initialFilters = {}
}: SearchFormProps) {
  // Profiling hooks to track renders
  // Temporarily disable render profiler to fix loading issues
  // useRenderProfiler('SearchForm', {
  //   onResults,
  //   onRecommendations,
  // });
  
  // useWhyDidYouUpdate('SearchForm', {
  //   onResults,
  //   onRecommendations,
  //   className,
  //   initialFilters
  // });

  const [filters, setFilters] = useState<SearchFilters>({
    relationships: [],
    characters: [],
    freeformTags: [],
    fandoms: [],
    excludePoorlyTagged: false,
    enableSmartSuggestions: true,
    ...initialFilters
  });
  
  const [tagInputs, setTagInputs] = useState<Record<string, string>>({
    relationships: '',
    characters: '',
    freeformTags: '',
    fandoms: ''
  });
  const [showAdvanced, setShowAdvanced] = useState(false);
  const [announceResults, setAnnounceResults] = useState('');
  const [searchError, setSearchError] = useState<string>('');

  // Accessibility IDs
  const formId = useId();
  const filtersId = useId();
  const errorId = useId();
  const resultsAnnouncementId = useId();

  // Refs for focus management
  const titleInputRef = useRef<HTMLInputElement>(null);
  const submitButtonRef = useRef<HTMLButtonElement>(null);
  const firstAdvancedFieldRef = useRef<HTMLInputElement>(null);

  const [loading, setLoading] = useState(false);

  const performSearch = async (searchParams: any) => {
    setLoading(true);
    setSearchError('');
    setAnnounceResults('Searching...');
    
    try {
      const query = searchParams.query || '';
      const apiParams = {
        page: 1,
        limit: 20,
        fandoms: searchParams.filters.fandoms,
        characters: searchParams.filters.characters,
        relationships: searchParams.filters.relationships,
        tags: searchParams.filters.freeform_tags,
        rating: searchParams.filters.rating ? [searchParams.filters.rating] : [],
        wordCountMin: searchParams.filters.word_count_min,
        wordCountMax: searchParams.filters.word_count_max,
        language: searchParams.filters.language ? [searchParams.filters.language] : [],
        status: searchParams.filters.status,
      };

      const data = await searchWorks(query, apiParams);
      
      // Normalize the data to match expected structure
      const normalizedWorks = (data.works || []).map((work: any) => ({
        ...work,
        fandoms: work.fandoms || [],
        characters: work.characters || [],
        relationships: work.relationships || [],
        freeform_tags: work.freeform_tags || [],
        // Map snake_case to camelCase for consistency
        wordCount: work.word_count,
        chapterCount: work.chapter_count,
        maxChapters: work.max_chapters,
        publishedAt: work.published_at,
        updatedAt: work.updated_at,
      }));
      
      onResults(normalizedWorks);
      setAnnounceResults(`Search completed. Found ${normalizedWorks.length} results.`);
      setSearchError('');
    } catch (error) {
      console.error('Search error:', error);
      setSearchError(error instanceof Error ? error.message : 'Search failed. Please try again.');
      setAnnounceResults('Search failed. Please try again.');
    } finally {
      setLoading(false);
    }
  };

  const handleSearch = useCallback((e?: React.FormEvent) => {
    if (e) e.preventDefault();
    
    // Basic validation
    if (!filters.title && !filters.author && filters.relationships.length === 0 && 
        filters.characters.length === 0 && filters.freeformTags.length === 0 && 
        filters.fandoms.length === 0) {
      setSearchError('Please enter at least one search criterion');
      setAnnounceResults('Please enter at least one search criterion');
      titleInputRef.current?.focus();
      return;
    }

    setSearchError('');
    setAnnounceResults('Searching...');
    
    const searchParams = {
      query: filters.title || '',
      filters: {
        title: filters.title,
        author: filters.author,
        relationships: filters.relationships,
        characters: filters.characters,
        freeform_tags: filters.freeformTags,
        fandoms: filters.fandoms,
        rating: filters.rating,
        word_count_min: filters.wordCountMin,
        word_count_max: filters.wordCountMax,
        language: filters.language,
        status: filters.status
      },
      options: {
        exclude_poorly_tagged: filters.excludePoorlyTagged,
        enable_smart_suggestions: filters.enableSmartSuggestions,
        limit: 20,
        offset: 0
      }
    };

    performSearch(searchParams);
  }, [filters, searchWorks]);

  const addTag = useCallback((tagField: string, tag: string) => {
    setFilters(prev => ({
      ...prev,
      [tagField]: [...prev[tagField as keyof SearchFilters] as string[], tag]
    }));
    setTagInputs(prev => ({ ...prev, [tagField]: '' }));
    setAnnounceResults(`Added ${tag} to ${tagField}`);
  }, []);

  const removeTag = useCallback((tagField: string, index: number, tagName: string) => {
    setFilters(prev => ({
      ...prev,
      [tagField]: (prev[tagField as keyof SearchFilters] as string[]).filter((_, i) => i !== index)
    }));
    setAnnounceResults(`Removed ${tagName} from ${tagField}`);
  }, []);

  // Create stable callback references for each tag field
  const tagInputHandlers = useMemo(() => ({
    relationships: (value: string) => setTagInputs(prev => ({ ...prev, relationships: value })),
    characters: (value: string) => setTagInputs(prev => ({ ...prev, characters: value })),
    freeformTags: (value: string) => setTagInputs(prev => ({ ...prev, freeformTags: value })),
    fandoms: (value: string) => setTagInputs(prev => ({ ...prev, fandoms: value }))
  }), []);

  const tagSelectHandlers = useMemo(() => ({
    relationships: (tag: any) => addTag('relationships', tag.name),
    characters: (tag: any) => addTag('characters', tag.name),
    freeformTags: (tag: any) => addTag('freeformTags', tag.name),
    fandoms: (tag: any) => addTag('fandoms', tag.name)
  }), [addTag]);

  // Create stable callback references for tag removal
  const removeTagHandlers = useMemo(() => ({
    relationships: (index: number, tagName: string) => removeTag('relationships', index, tagName),
    characters: (index: number, tagName: string) => removeTag('characters', index, tagName),
    freeformTags: (index: number, tagName: string) => removeTag('freeformTags', index, tagName),
    fandoms: (index: number, tagName: string) => removeTag('fandoms', index, tagName)
  }), [removeTag]);

  const handleInputChange = useCallback((field: keyof SearchFilters, value: string | number | boolean | undefined) => {
    setFilters(prev => ({
      ...prev,
      [field]: value
    }));
  }, []);

  const clearAllFilters = useCallback(() => {
    setFilters({
      relationships: [],
      characters: [],
      freeformTags: [],
      fandoms: [],
      excludePoorlyTagged: false,
      enableSmartSuggestions: true
    });
    setAnnounceResults('All search filters cleared');
    titleInputRef.current?.focus();
  }, []);

  const toggleAdvanced = useCallback(() => {
    setShowAdvanced(prev => {
      const newState = !prev;
      setAnnounceResults(
        newState ? 'Advanced search options expanded' : 'Advanced search options collapsed'
      );
      
      // Focus first advanced field when expanding
      if (newState && firstAdvancedFieldRef.current) {
        setTimeout(() => firstAdvancedFieldRef.current?.focus(), 100);
      }
      
      return newState;
    });
  }, []);

  // Enhanced keyboard navigation for form
  const handleFormKeyDown = useCallback((e: React.KeyboardEvent) => {
    // Ctrl+Enter or Cmd+Enter to submit form
    if ((e.ctrlKey || e.metaKey) && e.key === 'Enter') {
      e.preventDefault();
      handleSearch();
    }
    
    // Escape to clear all filters
    if (e.key === 'Escape') {
      clearAllFilters();
    }
  }, [handleSearch, clearAllFilters]);

  return (
    <section 
      role="search" 
      aria-labelledby={`${formId}-heading`}
      className={`bg-white p-6 rounded-lg shadow-lg space-y-6 ${className}`}
    >
      {/* Screen Reader Instructions */}
      <div className="sr-only" id={`${formId}-instructions`}>
        Enhanced search form for finding works in the archive. 
        Use tab to navigate between fields. 
        Form validation will announce any errors.
        {showAdvanced && ' Advanced search options are currently expanded.'}
      </div>

      {/* Error Announcement */}
      {searchError && (
        <div
          role="alert"
          id={errorId}
          className="mb-4 p-3 bg-red-50 border border-red-200 rounded-md text-red-800 flex items-center gap-2"
        >
          <AlertCircle className="h-5 w-5 text-red-500 flex-shrink-0" aria-hidden="true" />
          <span>Search error: {searchError}</span>
        </div>
      )}

      {/* Live Region for Announcements */}
      <div
        id={resultsAnnouncementId}
        aria-live="polite"
        aria-atomic="true"
        className="sr-only"
      >
        {announceResults}
      </div>

      <form
        onSubmit={handleSearch}
        onKeyDown={handleFormKeyDown}
        role="search"
        aria-labelledby={`${formId}-heading`}
        aria-describedby={`${formId}-instructions ${searchError ? errorId : ''}`}
      >
        {/* Form Header */}
        <header className="flex items-center justify-between mb-6">
          <h1 id={`${formId}-heading`} className="text-xl font-bold text-gray-900">
            Enhanced Search
          </h1>
          
          {/* Quick Options */}
          <div className="flex items-center space-x-4">
            <div className="flex items-center space-x-2">
              <input
                id={`${formId}-exclude-poorly-tagged`}
                type="checkbox"
                checked={filters.excludePoorlyTagged}
                onChange={(e) => handleInputChange('excludePoorlyTagged', e.target.checked)}
                className="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
              />
              <label 
                htmlFor={`${formId}-exclude-poorly-tagged`}
                className="text-sm text-gray-700 cursor-pointer"
              >
                Exclude poorly tagged works
              </label>
            </div>
            
            <div className="flex items-center space-x-2">
              <input
                id={`${formId}-smart-suggestions`}
                type="checkbox"
                checked={filters.enableSmartSuggestions}
                onChange={(e) => handleInputChange('enableSmartSuggestions', e.target.checked)}
                className="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
              />
              <label 
                htmlFor={`${formId}-smart-suggestions`}
                className="text-sm text-gray-700 cursor-pointer"
              >
                Smart suggestions
              </label>
            </div>
          </div>
        </header>

        {/* Basic Search Fields */}
        <fieldset className="grid grid-cols-1 md:grid-cols-2 gap-6 mb-6">
          <legend className="sr-only">Basic search criteria</legend>
          <section className="space-y-4" aria-labelledby="basic-fields-1">
            <h3 id="basic-fields-1" className="sr-only">Primary search fields</h3>
            <div>
              <label htmlFor={`${formId}-title`} className="block text-sm font-medium text-gray-700 mb-2">
                Title
              </label>
              <div className="relative">
                <input
                  ref={titleInputRef}
                  id={`${formId}-title`}
                  type="text"
                  value={filters.title || ''}
                  onChange={(e) => handleInputChange('title', e.target.value)}
                  placeholder="Search by title..."
                  className="w-full pl-10 pr-4 py-3 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                  aria-describedby={`${formId}-title-help`}
                />
                <Search className="absolute left-3 top-3.5 h-5 w-5 text-gray-400" aria-hidden="true" />
              </div>
              <div id={`${formId}-title-help`} className="mt-1 text-xs text-gray-500">
                Enter keywords from the work title
              </div>
            </div>

            <div>
              <label htmlFor={`${formId}-author`} className="block text-sm font-medium text-gray-700 mb-2">
                Author
              </label>
              <input
                id={`${formId}-author`}
                type="text"
                value={filters.author || ''}
                onChange={(e) => handleInputChange('author', e.target.value)}
                placeholder="Search by author..."
                className="w-full px-4 py-3 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                aria-describedby={`${formId}-author-help`}
              />
              <div id={`${formId}-author-help`} className="mt-1 text-xs text-gray-500">
                Enter author username or display name
              </div>
            </div>

            <MemoizedTagField
              label="Relationships"
              field="relationships"
              placeholder="Add relationship tag..."
              formId={formId}
              tags={filters.relationships}
              tagInput={tagInputs.relationships}
              onTagInputChange={tagInputHandlers.relationships}
              onTagSelect={tagSelectHandlers.relationships}
              onRemoveTag={removeTagHandlers.relationships}
            />

            <MemoizedTagField
              label="Characters"
              field="characters"
              placeholder="Add character tag..."
              formId={formId}
              tags={filters.characters}
              tagInput={tagInputs.characters}
              onTagInputChange={tagInputHandlers.characters}
              onTagSelect={tagSelectHandlers.characters}
              onRemoveTag={removeTagHandlers.characters}
            />
          </section>

          <section className="space-y-4" aria-labelledby="basic-fields-2">
            <h3 id="basic-fields-2" className="sr-only">Tag search fields</h3>
            <MemoizedTagField
              label="Additional Tags"
              field="freeformTags"
              placeholder="Add additional tag..."
              formId={formId}
              tags={filters.freeformTags}
              tagInput={tagInputs.freeformTags}
              onTagInputChange={tagInputHandlers.freeformTags}
              onTagSelect={tagSelectHandlers.freeformTags}
              onRemoveTag={removeTagHandlers.freeformTags}
            />

            <MemoizedTagField
              label="Fandoms"
              field="fandoms"
              placeholder="Add fandom tag..."
              formId={formId}
              tags={filters.fandoms}
              tagInput={tagInputs.fandoms}
              onTagInputChange={tagInputHandlers.fandoms}
              onTagSelect={tagSelectHandlers.fandoms}
              onRemoveTag={removeTagHandlers.fandoms}
            />

            {/* Quick Filters */}
            <section className="grid grid-cols-2 gap-4" aria-labelledby="quick-filters">
              <h4 id="quick-filters" className="sr-only">Quick filter options</h4>
              <div>
                <label htmlFor={`${formId}-rating`} className="block text-sm font-medium text-gray-700 mb-2">
                  Rating
                </label>
                <select
                  id={`${formId}-rating`}
                  value={filters.rating || ''}
                  onChange={(e) => handleInputChange('rating', e.target.value)}
                  className="w-full px-4 py-3 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                >
                  <option value="">Any Rating</option>
                  <option value="General Audiences">General Audiences</option>
                  <option value="Teen And Up Audiences">Teen And Up Audiences</option>
                  <option value="Mature">Mature</option>
                  <option value="Explicit">Explicit</option>
                </select>
              </div>

              <div>
                <label htmlFor={`${formId}-status`} className="block text-sm font-medium text-gray-700 mb-2">
                  Status
                </label>
                <select
                  id={`${formId}-status`}
                  value={filters.status || ''}
                  onChange={(e) => handleInputChange('status', e.target.value)}
                  className="w-full px-4 py-3 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                >
                  <option value="">Any Status</option>
                  <option value="complete">Complete</option>
                  <option value="in_progress">In Progress</option>
                  <option value="hiatus">On Hiatus</option>
                </select>
              </div>
            </section>
          </section>
        </fieldset>

        {/* Advanced Search Toggle */}
        <div className="flex items-center justify-center mb-4">
          <button
            type="button"
            onClick={toggleAdvanced}
            className="flex items-center gap-2 px-4 py-2 text-blue-600 hover:text-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 rounded-md"
            aria-expanded={showAdvanced}
            aria-controls={filtersId}
          >
            <Filter className="h-4 w-4" aria-hidden="true" />
            {showAdvanced ? 'Hide' : 'Show'} Advanced Search
          </button>
        </div>

        {/* Advanced Search Fields */}
        {showAdvanced && (
          <fieldset
            id={filtersId}
            className="p-4 bg-gray-50 rounded-lg border space-y-4 mb-6"
            aria-labelledby={`${filtersId}-heading`}
          >
            <legend id={`${filtersId}-heading`} className="text-lg font-medium mb-4">
              Advanced Search Options
            </legend>

            <section className="grid grid-cols-1 md:grid-cols-2 gap-4" aria-labelledby="advanced-fields">
              <h4 id="advanced-fields" className="sr-only">Advanced search parameters</h4>
              <div>
                <label htmlFor={`${formId}-word-count-min`} className="block text-sm font-medium text-gray-700 mb-2">
                  Minimum Word Count
                </label>
                <input
                  ref={firstAdvancedFieldRef}
                  id={`${formId}-word-count-min`}
                  type="number"
                  min="0"
                  value={filters.wordCountMin || ''}
                  onChange={(e) => handleInputChange('wordCountMin', parseInt(e.target.value) || undefined)}
                  placeholder="e.g., 1000"
                  className="w-full px-4 py-3 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                />
              </div>

              <div>
                <label htmlFor={`${formId}-word-count-max`} className="block text-sm font-medium text-gray-700 mb-2">
                  Maximum Word Count
                </label>
                <input
                  id={`${formId}-word-count-max`}
                  type="number"
                  min="0"
                  value={filters.wordCountMax || ''}
                  onChange={(e) => handleInputChange('wordCountMax', parseInt(e.target.value) || undefined)}
                  placeholder="e.g., 50000"
                  className="w-full px-4 py-3 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                />
              </div>

              <div>
                <label htmlFor={`${formId}-language`} className="block text-sm font-medium text-gray-700 mb-2">
                  Language
                </label>
                <select
                  id={`${formId}-language`}
                  value={filters.language || ''}
                  onChange={(e) => handleInputChange('language', e.target.value)}
                  className="w-full px-4 py-3 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                >
                  <option value="">Any Language</option>
                  <option value="en">English</option>
                  <option value="es">Spanish</option>
                  <option value="fr">French</option>
                  <option value="de">German</option>
                  <option value="it">Italian</option>
                  <option value="pt">Portuguese</option>
                  <option value="ru">Russian</option>
                  <option value="ja">Japanese</option>
                  <option value="ko">Korean</option>
                  <option value="zh">Chinese</option>
                </select>
              </div>
            </section>
          </fieldset>
        )}

        {/* Form Actions */}
        <footer className="flex justify-between items-center pt-4 border-t border-gray-200">
          <button
            type="button"
            onClick={clearAllFilters}
            className="px-4 py-2 text-gray-600 hover:text-gray-800 font-medium focus:outline-none focus:ring-2 focus:ring-gray-500 focus:ring-offset-2 rounded-md"
          >
            <X className="h-4 w-4 mr-2 inline" aria-hidden="true" />
            Clear All
          </button>
          
          <button
            ref={submitButtonRef}
            type="submit"
            disabled={loading}
            className="px-6 py-3 bg-blue-600 text-white font-medium rounded-lg hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed flex items-center space-x-2 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2"
            aria-describedby={loading ? `${formId}-loading` : undefined}
          >
            {loading ? (
              <>
                <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" />
                <span>Searching...</span>
              </>
            ) : (
              <>
                <Search className="h-4 w-4" aria-hidden="true" />
                <span>Search Works</span>
              </>
            )}
          </button>
        </footer>

        {loading && (
          <div
            id={`${formId}-loading`}
            className="mt-2 text-center text-sm text-gray-600"
            role="status"
            aria-live="polite"
          >
            Searching the archive...
          </div>
        )}
      </form>
    </section>
  );
});

export default SearchForm;