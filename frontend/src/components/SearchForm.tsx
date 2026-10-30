import React, { useState, useRef, useId, useCallback, useMemo } from 'react';
import { Search, Filter, X, AlertCircle, Loader2 } from 'lucide-react';
import { searchWorks, type SearchParams } from '@/lib/api';
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
                className="text-blue-700 hover:text-blue-800 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-1 rounded-full p-0.5"
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
      <div id={`${fieldId}-help`} className="text-xs text-gray-600">
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
  // Content filtering
  blockedTags: string[];
  hideIncomplete?: boolean;
  hideCrossovers?: boolean;
  // Date filtering
  updatedWithin?: string; // 'week', 'month', 'year'
  publishedAfter?: string;
  publishedBefore?: string;
  // Engagement filtering
  minKudos?: number;
  minComments?: number;
  minBookmarks?: number;
  hideOrphaned?: boolean;
  // Sorting
  sort?: string;
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
    // Content filtering
    blockedTags: [],
    hideIncomplete: false,
    hideCrossovers: false,
    // Date filtering
    updatedWithin: '',
    publishedAfter: '',
    publishedBefore: '',
    // Engagement filtering
    minKudos: undefined,
    minComments: undefined,
    minBookmarks: undefined,
    hideOrphaned: false,
    ...initialFilters
  });
  
  const [tagInputs, setTagInputs] = useState<Record<string, string>>({
    relationships: '',
    characters: '',
    freeformTags: '',
    fandoms: '',
    blockedTags: ''
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

  // The API sends tags as plain names; the results list shows { name, category }.
  const asTags = (names: string[] | undefined, category: string) =>
    (names || []).map((name) => ({ name, category }));

  // Runs the search against the REST API (lib/api.ts searchWorks).
  const [loading, setLoading] = useState(false);
  const executeSearch = useCallback(async (f: SearchFilters) => {
    setLoading(true);
    try {
      const data = await searchWorks(f.title || '', {
        fandoms: f.fandoms,
        characters: f.characters,
        relationships: f.relationships,
        tags: f.freeformTags,
        blockedTags: f.blockedTags,
        rating: f.rating ? [f.rating] : undefined,
        language: f.language ? [f.language] : undefined,
        wordCountMin: f.wordCountMin,
        wordCountMax: f.wordCountMax,
        kudosMin: f.minKudos,
        commentsMin: f.minComments,
        status: f.hideIncomplete ? 'complete' : f.status,
        hideCrossovers: f.hideCrossovers,
        updatedWithin: f.updatedWithin,
        creator: f.author || undefined,
        sort: (f.sort as SearchParams['sort']) || 'relevance',
      });
      const works: any[] = data.works;

      // The API returns snake_case fields; accept camelCase too.
      const normalizedWorks = works.map((work: any) => ({
        id: work.id ?? work.work_id,
        title: work.title,
        author: work.anonymous || !work.authors?.length ? 'Anonymous' : work.authors.map((a: any) => a.name).join(', '),
        summary: work.summary,
        word_count: work.word_count ?? work.wordCount,
        chapter_count: work.chapter_count ?? work.chapterCount,
        max_chapters: work.max_chapters ?? work.maxChapters,
        rating: work.rating,
        status: work.status ?? work.completion_status ?? (work.is_complete ? 'complete' : 'in_progress'),
        language: work.language,
        published_date: work.published_date ?? work.published_at ?? work.created_at,
        updated_date: work.updated_date ?? work.updated_at,
        relationships: asTags(work.relationships, 'relationship'),
        characters: asTags(work.characters, 'character'),
        freeform_tags: asTags(work.freeform_tags, 'freeform'),
        fandoms: asTags(work.fandoms, 'fandom'),
        kudos_count: work.kudos_count ?? work.kudos,
        bookmark_count: work.bookmark_count ?? work.bookmarks,
        hit_count: work.hit_count ?? work.hits,
        comment_count: work.comment_count ?? work.comments,
      }));

      onResults(normalizedWorks);
      setAnnounceResults(`Search completed. Found ${normalizedWorks.length} results.`);
      setSearchError('');
    } catch (error: any) {
      console.error('Search error:', error);
      setSearchError(error?.message || 'Search failed. Please try again.');
      setAnnounceResults('Search failed. Please try again.');
    } finally {
      setLoading(false);
    }
  }, [onResults]);


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
    
    executeSearch(filters);

  }, [filters, executeSearch]);

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
      blockedTags: [],
      hideIncomplete: false,
      hideCrossovers: false,
      updatedWithin: '',
      publishedAfter: '',
      publishedBefore: '',
      minKudos: undefined,
      minComments: undefined,
      minBookmarks: undefined,
      hideOrphaned: false,
      sort: 'relevance'
    });
    setTagInputs({
      relationships: '',
      characters: '',
      freeformTags: '',
      fandoms: '',
      blockedTags: ''
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
        aria-labelledby={`${formId}-heading`}
        aria-describedby={`${formId}-instructions ${searchError ? errorId : ''}`}
      >
        {/* Form Header */}
        <header className="flex items-center justify-between mb-6">
          <h1 id={`${formId}-heading`} className="text-xl font-bold text-gray-900">
            Enhanced Search
          </h1>
          
        </header>

        {/* Basic Search Fields */}
        <fieldset className="grid grid-cols-1 md:grid-cols-2 gap-6 mb-6">
          <legend className="sr-only">Basic search criteria</legend>
          <section className="space-y-4" aria-labelledby="basic-fields-1">
            <h2 id="basic-fields-1" className="sr-only">Primary search fields</h2>
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
              <div id={`${formId}-title-help`} className="mt-1 text-xs text-gray-600">
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
              <div id={`${formId}-author-help`} className="mt-1 text-xs text-gray-600">
                Enter the creator's pseud
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
            <h2 id="basic-fields-2" className="sr-only">Tag search fields</h2>
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
              <h3 id="quick-filters" className="sr-only">Quick filter options</h3>
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
                <label htmlFor={`${formId}-sort`} className="block text-sm font-medium text-gray-700 mb-2">
                  Sort Results By
                </label>
                <select
                  id={`${formId}-sort`}
                  value={filters.sort || 'relevance'}
                  onChange={(e) => handleInputChange('sort', e.target.value)}
                  className="w-full px-4 py-3 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                >
                  <option value="relevance">Best Match</option>
                  <option value="updated_at">Date Updated</option>
                  <option value="published_at">Date Posted</option>
                  <option value="title">Title</option>
                  <option value="word_count">Word Count</option>
                  <option value="hits">Hits</option>
                  <option value="kudos">Kudos</option>
                  <option value="comments">Comments</option>
                  <option value="bookmarks">Bookmarks</option>
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
            className="flex items-center gap-2 px-4 py-2 text-blue-700 hover:text-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 rounded-md"
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
              <h3 id="advanced-fields" className="sr-only">Advanced search parameters</h3>
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

            {/* Content Filtering Section */}
            <section className="border-t border-gray-200 pt-4 mt-6">
              <h3 className="text-md font-medium text-gray-800 mb-4">🚫 Content Filtering</h3>
              
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div>
                  <label htmlFor={`${formId}-blocked-tags`} className="block text-sm font-medium text-gray-700 mb-2">
                    Blocked Tags
                  </label>
                  <TagAutocomplete
                    id={`${formId}-blocked-tags`}
                    value={tagInputs.blockedTags || ''}
                    onChange={(value) => setTagInputs(prev => ({ ...prev, blockedTags: value }))}
                    onTagSelect={(tag) => addTag('blockedTags', tag.name || tag)}
                    placeholder="Tags to avoid..."
                    className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                  />
                  {filters.blockedTags && filters.blockedTags.length > 0 && (
                    <div className="flex flex-wrap gap-1 mt-2">
                      {filters.blockedTags.map((tag, index) => (
                        <span key={index} className="inline-flex items-center px-2 py-1 bg-red-100 text-red-800 text-xs rounded-full">
                          {tag}
                          <button
                            type="button"
                            onClick={() => removeTag('blockedTags', index, tag)}
                            className="ml-1 text-red-600 hover:text-red-800 focus:outline-none"
                          >
                            ×
                          </button>
                        </span>
                      ))}
                    </div>
                  )}
                  <p className="text-xs text-gray-600 mt-1">Hide works containing these tags</p>
                </div>

                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-2">Content Preferences</label>
                  <div className="space-y-2">
                    <label className="flex items-center">
                      <input
                        type="checkbox"
                        checked={filters.hideIncomplete || false}
                        onChange={(e) => handleInputChange('hideIncomplete', e.target.checked)}
                        className="rounded border-gray-300 text-blue-700 focus:ring-blue-500"
                      />
                      <span className="ml-2 text-sm text-gray-700">Hide incomplete works</span>
                    </label>
                    <label className="flex items-center">
                      <input
                        type="checkbox"
                        checked={filters.hideCrossovers || false}
                        onChange={(e) => handleInputChange('hideCrossovers', e.target.checked)}
                        className="rounded border-gray-300 text-blue-700 focus:ring-blue-500"
                      />
                      <span className="ml-2 text-sm text-gray-700">Hide crossovers</span>
                    </label>
                  </div>
                </div>
              </div>
            </section>

            {/* Date & Engagement Filtering Section */}
            <section className="border-t border-gray-200 pt-4 mt-6">
              <h3 className="text-md font-medium text-gray-800 mb-4">📅 Date & Engagement Filters</h3>
              
              <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                <div>
                  <label htmlFor={`${formId}-updated-within`} className="block text-sm font-medium text-gray-700 mb-2">
                    Updated Within
                  </label>
                  <select
                    id={`${formId}-updated-within`}
                    value={filters.updatedWithin || ''}
                    onChange={(e) => handleInputChange('updatedWithin', e.target.value)}
                    className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                  >
                    <option value="">Any time</option>
                    <option value="week">Past week</option>
                    <option value="month">Past month</option>
                    <option value="3months">Past 3 months</option>
                    <option value="year">Past year</option>
                  </select>
                </div>

                <div>
                  <label htmlFor={`${formId}-min-kudos`} className="block text-sm font-medium text-gray-700 mb-2">
                    Minimum Kudos
                  </label>
                  <input
                    id={`${formId}-min-kudos`}
                    type="number"
                    min="0"
                    value={filters.minKudos || ''}
                    onChange={(e) => handleInputChange('minKudos', parseInt(e.target.value) || undefined)}
                    placeholder="e.g., 10"
                    className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                  />
                </div>

                <div>
                  <label htmlFor={`${formId}-min-comments`} className="block text-sm font-medium text-gray-700 mb-2">
                    Minimum Comments
                  </label>
                  <input
                    id={`${formId}-min-comments`}
                    type="number"
                    min="0"
                    value={filters.minComments || ''}
                    onChange={(e) => handleInputChange('minComments', parseInt(e.target.value) || undefined)}
                    placeholder="e.g., 5"
                    className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                  />
                </div>
              </div>
            </section>
          </fieldset>
        )}

        {/* Form Actions */}
        <footer className="flex justify-between items-center pt-4 border-t border-gray-200">
          <button
            type="button"
            onClick={clearAllFilters}
            className="px-4 py-2 text-gray-700 hover:text-gray-800 font-medium focus:outline-none focus:ring-2 focus:ring-gray-500 focus:ring-offset-2 rounded-md"
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
            className="mt-2 text-center text-sm text-gray-700"
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