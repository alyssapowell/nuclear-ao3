'use client';

import React, { useState, useRef, useId, useEffect } from 'react';
import { 
  Star, 
  Heart, 
  MessageSquare, 
  Eye, 
  Calendar, 
  User, 
  Tag, 
  ChevronDown, 
  ChevronUp,
  ExternalLink,
  AlertTriangle,
  Info
} from 'lucide-react';

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

interface SmartRecommendation {
  type: string;
  title: string;
  description: string;
  suggestions: string[];
  confidence_score: number;
  category: string;
}

interface SearchResultsProps {
  results: Work[];
  recommendations?: SmartRecommendation[];
  loading?: boolean;
  error?: string;
  totalCount?: number;
  currentPage?: number;
  pageSize?: number;
  onPageChange?: (page: number) => void;
  onWorkClick?: (workId: string, workTitle?: string) => void;
  className?: string;
}

const SearchResults: React.FC<SearchResultsProps> = ({
  results,
  recommendations = [],
  loading = false,
  error,
  totalCount = 0,
  currentPage = 1,
  pageSize = 20,
  onPageChange,
  onWorkClick,
  className = ''
}) => {
  const [expandedSummaries, setExpandedSummaries] = useState<Set<string>>(new Set());
  const [expandedRecommendations, setExpandedRecommendations] = useState(false);
  const [announceUpdate, setAnnounceUpdate] = useState('');
  
  // Accessibility IDs
  const resultsId = useId();
  const recommendationsId = useId();
  const statusId = useId();
  const liveRegionId = useId();

  // Refs for focus management
  const resultsHeaderRef = useRef<HTMLHeadingElement>(null);
  const firstResultRef = useRef<HTMLLIElement>(null);

  // Focus management when results change
  useEffect(() => {
    if (results.length > 0 && !loading) {
      // Announce results count and focus results section
      setAnnounceUpdate(`Found ${totalCount || results.length} works. Results loaded.`);
      setTimeout(() => {
        resultsHeaderRef.current?.focus();
      }, 100);
    }
  }, [results, loading, totalCount]);

  // Handle summary expansion
  const toggleSummary = (workId: string) => {
    setExpandedSummaries(prev => {
      const newExpanded = new Set(prev);
      if (newExpanded.has(workId)) {
        newExpanded.delete(workId);
        setAnnounceUpdate('Summary collapsed');
      } else {
        newExpanded.add(workId);
        setAnnounceUpdate('Summary expanded');
      }
      return newExpanded;
    });
  };

  // Handle work click
  const handleWorkClick = (workId: string, workTitle: string) => {
    setAnnounceUpdate(`Opening ${workTitle}`);
    onWorkClick?.(workId, workTitle);
  };

  // Handle keyboard navigation for work items
  const handleWorkKeyDown = (e: React.KeyboardEvent, workId: string, workTitle: string) => {
    if (e.key === 'Enter' || e.key === ' ') {
      e.preventDefault();
      handleWorkClick(workId, workTitle);
    }
  };

  // Format date for display and screen readers
  const formatDate = (dateString: string) => {
    if (!dateString) {
      return { readable: 'Unknown date', iso: '' };
    }
    
    const date = new Date(dateString);
    
    // Check if the date is valid
    if (isNaN(date.getTime())) {
      return { readable: 'Invalid date', iso: '' };
    }
    
    const readable = date.toLocaleDateString('en-US', { 
      year: 'numeric', 
      month: 'long', 
      day: 'numeric',
      timeZone: 'UTC'
    });
    const iso = date.toISOString().split('T')[0];
    return { readable, iso };
  };

  // Format numbers with thousands separators
  const formatNumber = (num?: number) => {
    return num?.toLocaleString() || '0';
  };

  // Get rating color for visual and programmatic identification
  const getRatingColor = (rating: string) => {
    switch (rating.toLowerCase()) {
      case 'general audiences':
        return 'bg-green-100 text-green-800 border-green-200';
      case 'teen and up audiences':
        return 'bg-blue-100 text-blue-800 border-blue-200';
      case 'mature':
        return 'bg-yellow-100 text-yellow-800 border-yellow-200';
      case 'explicit':
        return 'bg-red-100 text-red-800 border-red-200';
      default:
        return 'bg-gray-100 text-gray-800 border-gray-200';
    }
  };

  // Loading state
  if (loading) {
    return (
      <div className={`${className}`}>
        <div
          role="status"
          aria-live="polite"
          aria-label="Loading search results"
          className="flex items-center justify-center py-12"
        >
          <div className="animate-spin h-8 w-8 border-4 border-blue-600 border-t-transparent rounded-full" aria-hidden="true"></div>
          <span className="ml-3 text-lg text-gray-600">Searching the archive...</span>
        </div>
      </div>
    );
  }

  // Error state
  if (error) {
    return (
      <div className={`${className}`}>
        <div
          role="alert"
          className="bg-red-50 border border-red-200 rounded-lg p-6 flex items-start gap-3"
        >
          <AlertTriangle className="h-6 w-6 text-red-500 flex-shrink-0 mt-0.5" aria-hidden="true" />
          <div>
            <h3 className="text-lg font-medium text-red-800 mb-2">Search Error</h3>
            <p className="text-red-700">{error}</p>
            <p className="text-red-600 text-sm mt-2">
              Please try adjusting your search criteria or try again later.
            </p>
          </div>
        </div>
      </div>
    );
  }

  // No results state
  if (results.length === 0) {
    return (
      <div className={`${className}`}>
        <div className="text-center py-12">
          <Info className="h-12 w-12 text-gray-400 mx-auto mb-4" aria-hidden="true" />
          <h3 className="text-lg font-medium text-gray-900 mb-2">No works found</h3>
          <p className="text-gray-600 max-w-md mx-auto">
            Try adjusting your search criteria or browse popular tags to discover new works.
          </p>
        </div>
      </div>
    );
  }

  return (
    <div className={`${className}`} role="region" aria-labelledby={`${resultsId}-heading`}>
      {/* Live Region for Announcements */}
      <div
        id={liveRegionId}
        aria-live="polite"
        aria-atomic="true"
        className="sr-only"
      >
        {announceUpdate}
      </div>

      {/* Results Header */}
      <header className="mb-6">
        <h2 
          id={`${resultsId}-heading`}
          ref={resultsHeaderRef}
          className="text-2xl font-bold text-gray-900 mb-2"
          tabIndex={-1}
        >
          Search Results
        </h2>
        <div
          id={statusId}
          role="status"
          aria-live="polite"
          aria-atomic="true"
          className="text-gray-600"
        >
          {totalCount > 0 ? (
            <>
              Showing {((currentPage - 1) * pageSize) + 1}-{Math.min(currentPage * pageSize, totalCount)} of {formatNumber(totalCount)} works
            </>
          ) : (
            `Found ${results.length} works`
          )}
        </div>
      </header>

      {/* Smart Recommendations */}
      {recommendations.length > 0 && (
        <section 
          className="mb-8 bg-blue-50 border border-blue-200 rounded-lg p-4"
          aria-labelledby={`${recommendationsId}-heading`}
        >
          <div className="flex items-center justify-between mb-3">
            <h3 id={`${recommendationsId}-heading`} className="text-lg font-medium text-blue-900">
              Smart Suggestions
            </h3>
            <button
              type="button"
              onClick={() => {
                setExpandedRecommendations(!expandedRecommendations);
                setAnnounceUpdate(expandedRecommendations ? 'Suggestions collapsed' : 'Suggestions expanded');
              }}
              className="flex items-center gap-1 text-blue-700 hover:text-blue-800 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 rounded-md p-1"
              aria-expanded={expandedRecommendations}
              aria-controls={`${recommendationsId}-content`}
            >
              {expandedRecommendations ? (
                <>
                  <ChevronUp className="h-4 w-4" aria-hidden="true" />
                  Hide
                </>
              ) : (
                <>
                  <ChevronDown className="h-4 w-4" aria-hidden="true" />
                  Show
                </>
              )}
            </button>
          </div>
          
          {expandedRecommendations && (
            <div id={`${recommendationsId}-content`} className="space-y-3">
              {recommendations.map((rec, index) => (
                <div key={index} className="bg-white rounded-md p-3 border border-blue-100">
                  <h4 className="font-medium text-blue-900 mb-1">{rec.title}</h4>
                  <p className="text-blue-800 text-sm mb-2">{rec.description}</p>
                  {rec.suggestions.length > 0 && (
                    <div className="flex flex-wrap gap-1">
                      {rec.suggestions.map((suggestion, sugIndex) => (
                        <span 
                          key={sugIndex}
                          className="inline-block px-2 py-1 bg-blue-100 text-blue-800 text-xs rounded"
                        >
                          {suggestion}
                        </span>
                      ))}
                    </div>
                  )}
                  <div className="text-xs text-blue-600 mt-1">
                    Confidence: {Math.round(rec.confidence_score * 100)}%
                  </div>
                </div>
              ))}
            </div>
          )}
        </section>
      )}

      {/* Results List */}
      <main>
        <h3 className="sr-only">Works matching your search</h3>
        <ol className="space-y-6" role="list">
          {results.map((work, index) => {
            const publishedDate = formatDate(work.published_date);
            const updatedDate = formatDate(work.updated_date);
            const isExpanded = expandedSummaries.has(work.id);
            const workRef = index === 0 ? firstResultRef : undefined;

            return (
              <li
                key={work.id}
                ref={workRef}
                className="bg-white border border-gray-200 rounded-lg p-6 hover:border-blue-300 focus-within:border-blue-300 focus-within:ring-2 focus-within:ring-blue-500 focus-within:ring-offset-2 transition-colors"
                role="article"
                aria-labelledby={`work-${work.id}-title`}
                aria-describedby={`work-${work.id}-meta work-${work.id}-summary`}
              >
                {/* Work Header */}
                <header className="mb-4">
                  <h4 
                    id={`work-${work.id}-title`}
                    className="text-xl font-semibold text-blue-600 hover:text-blue-700 cursor-pointer mb-2"
                    tabIndex={0}
                    onClick={() => handleWorkClick(work.id, work.title)}
                    onKeyDown={(e) => handleWorkKeyDown(e, work.id, work.title)}
                    role="button"
                    aria-describedby={`work-${work.id}-action-help`}
                  >
                    {work.title}
                    <ExternalLink className="inline h-4 w-4 ml-1" aria-hidden="true" />
                  </h4>
                  <div id={`work-${work.id}-action-help`} className="sr-only">
                    Press Enter or Space to open this work
                  </div>

                  <div className="flex items-center gap-2 text-gray-600 mb-2">
                    <User className="h-4 w-4" aria-hidden="true" />
                    <span>by <strong>{work.author}</strong></span>
                  </div>

                  {/* Metadata */}
                  <div 
                    id={`work-${work.id}-meta`}
                    className="flex flex-wrap items-center gap-4 text-sm text-gray-600"
                  >
                    <div className="flex items-center gap-1">
                      <span className={`px-2 py-1 text-xs font-medium rounded border ${getRatingColor(work.rating)}`}>
                        {work.rating}
                      </span>
                      <span className="sr-only">Content rating: {work.rating}</span>
                    </div>

                    <div className="flex items-center gap-1">
                      <Calendar className="h-4 w-4" aria-hidden="true" />
                      <time dateTime={publishedDate.iso} title={`Published ${publishedDate.readable}`}>
                        {publishedDate.readable}
                      </time>
                    </div>

                    {work.updated_date !== work.published_date && (
                      <div className="flex items-center gap-1">
                        <span>Updated:</span>
                        <time dateTime={updatedDate.iso} title={`Updated ${updatedDate.readable}`}>
                          {updatedDate.readable}
                        </time>
                      </div>
                    )}

                    <div>
                      <span className="font-medium">{formatNumber(work.word_count)}</span> words
                    </div>

                    <div>
                      <span className="font-medium">{work.chapter_count}</span>
                      {work.max_chapters ? `/${work.max_chapters}` : '/?'} chapters
                    </div>

                    <div>
                      Status: <span className="font-medium">{work.status}</span>
                    </div>

                    <div>
                      Language: <span className="font-medium">{work.language}</span>
                    </div>
                  </div>
                </header>

                {/* Summary */}
                {work.summary && (
                  <div className="mb-4">
                    <div 
                      id={`work-${work.id}-summary`}
                      className={`text-gray-700 ${!isExpanded ? 'line-clamp-3' : ''}`}
                      dangerouslySetInnerHTML={{ __html: work.summary }}
                    />
                    
                    {work.summary.length > 200 && (
                      <button
                        type="button"
                        onClick={() => toggleSummary(work.id)}
                        className="mt-2 text-blue-600 hover:text-blue-700 text-sm font-medium focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 rounded-md"
                        aria-expanded={isExpanded}
                        aria-controls={`work-${work.id}-summary`}
                      >
                        {isExpanded ? 'Show less' : 'Show more'}
                      </button>
                    )}
                  </div>
                )}

                {/* Tags */}
                <div className="mb-4">
                  <h5 className="sr-only">Tags for this work</h5>
                  
                  {/* Fandoms */}
                  {work.fandoms && work.fandoms.length > 0 && (
                    <div className="mb-2">
                      <span className="text-sm font-medium text-gray-700 mr-2">Fandoms:</span>
                      <div className="inline-flex flex-wrap gap-1">
                        {work.fandoms.map((fandom, index) => (
                          <span 
                            key={index}
                            className="inline-block px-2 py-1 bg-purple-100 text-purple-800 text-xs rounded"
                          >
                            {fandom.name}
                          </span>
                        ))}
                      </div>
                    </div>
                  )}

                  {/* Relationships */}
                  {work.relationships && work.relationships.length > 0 && (
                    <div className="mb-2">
                      <span className="text-sm font-medium text-gray-700 mr-2">Relationships:</span>
                      <div className="inline-flex flex-wrap gap-1">
                        {work.relationships.map((rel, index) => (
                          <span 
                            key={index}
                            className="inline-block px-2 py-1 bg-pink-100 text-pink-800 text-xs rounded"
                          >
                            {rel.name}
                          </span>
                        ))}
                      </div>
                    </div>
                  )}

                  {/* Characters */}
                  {work.characters && work.characters.length > 0 && (
                    <div className="mb-2">
                      <span className="text-sm font-medium text-gray-700 mr-2">Characters:</span>
                      <div className="inline-flex flex-wrap gap-1">
                        {work.characters.map((char, index) => (
                          <span 
                            key={index}
                            className="inline-block px-2 py-1 bg-green-100 text-green-800 text-xs rounded"
                          >
                            {char.name}
                          </span>
                        ))}
                      </div>
                    </div>
                  )}

                  {/* Additional Tags */}
                  {work.freeform_tags && work.freeform_tags.length > 0 && (
                    <div className="mb-2">
                      <span className="text-sm font-medium text-gray-700 mr-2">Additional Tags:</span>
                      <div className="inline-flex flex-wrap gap-1">
                        {work.freeform_tags.map((tag, index) => (
                          <span 
                            key={index}
                            className="inline-block px-2 py-1 bg-gray-100 text-gray-800 text-xs rounded"
                          >
                            {tag.name}
                          </span>
                        ))}
                      </div>
                    </div>
                  )}
                </div>

                {/* Statistics */}
                <footer className="flex items-center justify-between pt-4 border-t border-gray-200">
                  <div className="flex items-center gap-4 text-sm text-gray-600">
                    {work.kudos_count !== undefined && (
                      <div className="flex items-center gap-1" title={`${formatNumber(work.kudos_count)} kudos`}>
                        <Heart className="h-4 w-4" aria-hidden="true" />
                        <span>{formatNumber(work.kudos_count)}</span>
                        <span className="sr-only">kudos</span>
                      </div>
                    )}
                    
                    {work.bookmark_count !== undefined && (
                      <div className="flex items-center gap-1" title={`${formatNumber(work.bookmark_count)} bookmarks`}>
                        <Star className="h-4 w-4" aria-hidden="true" />
                        <span>{formatNumber(work.bookmark_count)}</span>
                        <span className="sr-only">bookmarks</span>
                      </div>
                    )}
                    
                    {work.comment_count !== undefined && (
                      <div className="flex items-center gap-1" title={`${formatNumber(work.comment_count)} comments`}>
                        <MessageSquare className="h-4 w-4" aria-hidden="true" />
                        <span>{formatNumber(work.comment_count)}</span>
                        <span className="sr-only">comments</span>
                      </div>
                    )}
                    
                    {work.hit_count !== undefined && (
                      <div className="flex items-center gap-1" title={`${formatNumber(work.hit_count)} hits`}>
                        <Eye className="h-4 w-4" aria-hidden="true" />
                        <span>{formatNumber(work.hit_count)}</span>
                        <span className="sr-only">hits</span>
                      </div>
                    )}
                  </div>

                  {/* Tag Quality Score */}
                  {work.tag_quality_score !== undefined && (
                    <div className="text-xs text-gray-500">
                      Tag Quality: {Math.round(work.tag_quality_score * 100)}%
                    </div>
                  )}
                </footer>

                {/* Missing Tag Suggestions */}
                {work.missing_tag_suggestions && work.missing_tag_suggestions.length > 0 && (
                  <div className="mt-3 p-3 bg-yellow-50 border border-yellow-200 rounded-md">
                    <h6 className="text-sm font-medium text-yellow-800 mb-1">
                      <Tag className="inline h-4 w-4 mr-1" aria-hidden="true" />
                      Suggested Additional Tags:
                    </h6>
                    <div className="flex flex-wrap gap-1">
                      {work.missing_tag_suggestions.map((suggestion, index) => (
                        <span 
                          key={index}
                          className="inline-block px-2 py-1 bg-yellow-100 text-yellow-800 text-xs rounded"
                        >
                          {suggestion}
                        </span>
                      ))}
                    </div>
                  </div>
                )}
              </li>
            );
          })}
        </ol>
      </main>
    </div>
  );
};

export default SearchResults;