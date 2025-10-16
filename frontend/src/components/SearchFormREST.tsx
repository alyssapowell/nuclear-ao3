'use client';

import { useState } from 'react';
import { searchWorks, getWorks } from '@/lib/api';

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
  onRecommendations: (recommendations: unknown[]) => void;
}

export default function SearchFormREST({ onResults, onRecommendations }: SearchFormProps) {
  const [filters, setFilters] = useState<SearchFilters>({
    relationships: [],
    characters: [],
    freeformTags: [],
    fandoms: [],
    excludePoorlyTagged: false,
    enableSmartSuggestions: true
  });
  
  const [loading, setLoading] = useState(false);

  const handleSearch = async () => {
    setLoading(true);
    try {
      let results;
      
      if (filters.title) {
        // Use search endpoint if there's a query
        const response = await searchWorks(filters.title, {
          limit: 20,
          page: 1
        });
        results = response.works;
      } else {
        // Use works endpoint for browsing
        const response = await getWorks({
          limit: 20,
          page: 1
        });
        results = response.works || [];
      }
      
      // Transform results to match frontend format
      const transformedResults = results.map((work: Record<string, unknown>) => ({
        id: work.id,
        title: work.title,
        author: work.username || work.author,
        summary: work.summary,
        word_count: work.word_count,
        chapter_count: work.chapter_count,
        max_chapters: work.max_chapters,
        rating: work.rating,
        status: work.status,
        language: work.language,
        published_date: work.published_at || work.published_date,
        updated_date: work.updated_at || work.updated_date,
        relationships: work.relationships ? 
          (typeof work.relationships === 'string' ? 
            work.relationships.split(',').map((tag: string) => ({ name: tag.trim(), category: 'relationship' })) :
            work.relationships) : [],
        characters: work.characters ? 
          (typeof work.characters === 'string' ? 
            work.characters.split(',').map((tag: string) => ({ name: tag.trim(), category: 'character' })) :
            work.characters) : [],
        freeform_tags: work.freeform_tags ? 
          (typeof work.freeform_tags === 'string' ? 
            work.freeform_tags.split(',').map((tag: string) => ({ name: tag.trim(), category: 'freeform' })) :
            work.freeform_tags) : [],
        fandoms: work.fandoms ? 
          (typeof work.fandoms === 'string' ? 
            work.fandoms.split(',').map((tag: string) => ({ name: tag.trim(), category: 'fandom' })) :
            work.fandoms) : [],
        kudos_count: work.kudos,
        bookmark_count: work.bookmarks,
        hit_count: work.hits,
        comment_count: work.comments,
      }));
      
      onResults(transformedResults);
      
      // For now, return empty recommendations since search indexing needs work
      onRecommendations([]);
      
    } catch (error) {
      console.error('Search failed:', error);
      // Show message about search indexing
      onResults([]);
      onRecommendations([]);
    } finally {
      setLoading(false);
    }
  };

  const handleInputChange = (field: keyof SearchFilters, value: string | number | boolean | undefined) => {
    setFilters(prev => ({
      ...prev,
      [field]: value
    }));
  };

  return (
    <div className="bg-white p-6 rounded-lg shadow-lg space-y-6">
      <div className="flex items-center justify-between">
        <h2 className="text-xl font-bold text-gray-900">Enhanced Search</h2>
        <div className="flex items-center space-x-4">
          <label className="flex items-center space-x-2">
            <input
              type="checkbox"
              checked={filters.excludePoorlyTagged}
              onChange={(e) => handleInputChange('excludePoorlyTagged', e.target.checked)}
              className="rounded"
            />
            <span className="text-sm text-gray-700">Exclude poorly tagged works</span>
          </label>
          <label className="flex items-center space-x-2">
            <input
              type="checkbox"
              checked={filters.enableSmartSuggestions}
              onChange={(e) => handleInputChange('enableSmartSuggestions', e.target.checked)}
              className="rounded"
            />
            <span className="text-sm text-gray-700">Smart suggestions</span>
          </label>
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        <div className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">Title / Search Query</label>
            <input
              type="text"
              value={filters.title || ''}
              onChange={(e) => handleInputChange('title', e.target.value)}
              placeholder="Search by title or try 'Agatha', 'Rio', 'Harry'..."
              className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">Author</label>
            <input
              type="text"
              value={filters.author || ''}
              onChange={(e) => handleInputChange('author', e.target.value)}
              placeholder="Search by author..."
              className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            />
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">Rating</label>
              <select
                value={filters.rating || ''}
                onChange={(e) => handleInputChange('rating', e.target.value)}
                className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              >
                <option value="">Any Rating</option>
                <option value="General Audiences">General Audiences</option>
                <option value="Teen And Up Audiences">Teen And Up Audiences</option>
                <option value="Mature">Mature</option>
                <option value="Explicit">Explicit</option>
              </select>
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">Status</label>
              <select
                value={filters.status || ''}
                onChange={(e) => handleInputChange('status', e.target.value)}
                className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              >
                <option value="">Any Status</option>
                <option value="complete">Complete</option>
                <option value="published">In Progress</option>
                <option value="hiatus">On Hiatus</option>
              </select>
            </div>
          </div>
        </div>

        <div className="space-y-4">
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">Min Word Count</label>
              <input
                type="number"
                value={filters.wordCountMin || ''}
                onChange={(e) => handleInputChange('wordCountMin', parseInt(e.target.value) || undefined)}
                placeholder="Min words"
                className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">Max Word Count</label>
              <input
                type="number"
                value={filters.wordCountMax || ''}
                onChange={(e) => handleInputChange('wordCountMax', parseInt(e.target.value) || undefined)}
                placeholder="Max words"
                className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              />
            </div>
          </div>

          {/* Note about current limitations */}
          <div className="bg-blue-50 p-4 rounded-lg">
            <h4 className="text-sm font-medium text-blue-900 mb-2">🔧 Development Status</h4>
            <p className="text-sm text-blue-700">
              Currently showing works from database. Search indexing and advanced tag filtering are being developed.
              Try browsing existing works or search by title.
            </p>
          </div>
        </div>
      </div>

      <div className="flex justify-between items-center pt-4 border-t border-gray-200">
        <button
          onClick={() => setFilters({
            relationships: [],
            characters: [],
            freeformTags: [],
            fandoms: [],
            excludePoorlyTagged: false,
            enableSmartSuggestions: true
          })}
          className="px-4 py-2 text-gray-600 hover:text-gray-800 font-medium"
        >
          Clear All
        </button>
        
        <button
          onClick={handleSearch}
          disabled={loading}
          className="px-6 py-2 bg-blue-600 text-white font-medium rounded-lg hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed flex items-center space-x-2"
        >
          {loading && (
            <div className="animate-spin h-4 w-4 border-2 border-white border-t-transparent rounded-full"></div>
          )}
          <span>{loading ? 'Searching...' : 'Search Works'}</span>
        </button>
      </div>
    </div>
  );
}