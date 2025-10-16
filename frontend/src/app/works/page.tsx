'use client';

import { useState, useEffect } from 'react';
import { useRouter } from 'next/navigation';
import SearchResults from '@/components/SearchResults';
import SearchForm from '@/components/SearchForm';
import SmartRecommendations, { SmartRecommendation } from '@/components/SmartRecommendations';
import { getWorks } from '@/lib/api';

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



interface WorksPageState {
  works: Work[];
  loading: boolean;
  error: string;
  pagination: {
    page: number;
    limit: number;
    total: number;
    total_pages: number;
  };
}

export default function WorksPage() {
  const router = useRouter();
  const [state, setState] = useState<WorksPageState>({
    works: [],
    loading: true,
    error: '',
    pagination: { page: 1, limit: 20, total: 0, total_pages: 0 }
  });
  
  const [recommendations, setRecommendations] = useState<SmartRecommendation[]>([]);
  const [filters, setFilters] = useState<{
    rating: string;
    status: string;
    sort: 'title' | 'updated_at' | 'created_at' | 'published_at' | 'word_count' | 'hits' | 'kudos' | 'comments' | 'bookmarks' | 'relevance';
    fandom: string;
    relationship: string;
    character: string;
    tag: string;
  }>({
    rating: '',
    status: '',
    sort: 'updated_at',
    fandom: '',
    relationship: '',
    character: '',
    tag: ''
  });

  const handleWorkClick = (workId: string, workTitle?: string) => {
    router.push(`/works/${workId}`);
  };

  useEffect(() => {
    loadWorks();
  }, [filters, state.pagination.page]);

  const loadWorks = async () => {
    setState(prev => ({ ...prev, loading: true, error: '' }));
    
    try {
      const params = {
        page: state.pagination.page,
        limit: state.pagination.limit,
        sort: filters.sort,
        ...Object.fromEntries(Object.entries(filters).filter(([_, v]) => v))
      };
      
      const response = await getWorks(params);
      setState(prev => ({
        ...prev,
        works: response.works,
        pagination: response.pagination,
        loading: false
      }));
    } catch (error) {
      setState(prev => ({
        ...prev,
        error: error instanceof Error ? error.message : 'Failed to load works',
        loading: false
      }));
    }
  };

  const handleSearch = (searchResults: Work[]) => {
    setState(prev => ({
      ...prev,
      works: searchResults,
      loading: false
    }));
  };

  const handleFilterChange = (field: string, value: string) => {
    setFilters(prev => ({ ...prev, [field]: value }));
    setState(prev => ({ ...prev, pagination: { ...prev.pagination, page: 1 } }));
  };

  const handlePageChange = (newPage: number) => {
    setState(prev => ({ 
      ...prev, 
      pagination: { ...prev.pagination, page: newPage } 
    }));
  };

  return (
    <div className="max-w-7xl mx-auto px-4 py-8">
      {/* Header */}
      <div className="mb-8">
        <h1 className="text-3xl font-bold text-gray-900 mb-4">Browse Works</h1>
        <p className="text-gray-600">
          Discover amazing fanworks from thousands of authors and fandoms.
        </p>
      </div>

      {/* Search Section */}
      <div className="mb-8">
        <SearchForm 
          onResults={handleSearch}
          onRecommendations={setRecommendations}
          className="mb-6"
        />
        
        {/* Quick Filters */}
        <div className="bg-white rounded-lg shadow-sm border p-4">
          <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
            <div>
              <label htmlFor="rating" className="block text-sm font-medium text-gray-700 mb-1">
                Rating
              </label>
              <select
                id="rating"
                value={filters.rating}
                onChange={(e) => handleFilterChange('rating', e.target.value)}
                className="w-full px-3 py-2 border border-gray-300 rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-orange-500"
              >
                <option value="">Any Rating</option>
                <option value="general">General Audiences</option>
                <option value="teen">Teen And Up</option>
                <option value="mature">Mature</option>
                <option value="explicit">Explicit</option>
              </select>
            </div>

            <div>
              <label htmlFor="status" className="block text-sm font-medium text-gray-700 mb-1">
                Status
              </label>
              <select
                id="status"
                value={filters.status}
                onChange={(e) => handleFilterChange('status', e.target.value)}
                className="w-full px-3 py-2 border border-gray-300 rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-orange-500"
              >
                <option value="">Any Status</option>
                <option value="complete">Complete</option>
                <option value="in_progress">Work In Progress</option>
                <option value="hiatus">On Hiatus</option>
              </select>
            </div>

            <div>
              <label htmlFor="sort" className="block text-sm font-medium text-gray-700 mb-1">
                Sort By
              </label>
              <select
                id="sort"
                value={filters.sort}
                onChange={(e) => handleFilterChange('sort', e.target.value)}
                className="w-full px-3 py-2 border border-gray-300 rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-orange-500"
              >
                <option value="updated_at">Date Updated</option>
                <option value="published_at">Date Published</option>
                <option value="kudos">Most Kudos</option>
                <option value="hits">Most Hits</option>
                <option value="bookmarks">Most Bookmarks</option>
                <option value="comments">Most Comments</option>
                <option value="word_count">Word Count</option>
                <option value="title">Title A-Z</option>
              </select>
            </div>

            <div className="flex items-end">
              <button
                onClick={() => {
                  setFilters({
                    rating: '',
                    status: '',
                    sort: 'updated_at',
                    fandom: '',
                    relationship: '',
                    character: '',
                    tag: ''
                  });
                }}
                className="w-full px-3 py-2 text-sm border border-gray-300 rounded-md hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-orange-500"
              >
                Clear Filters
              </button>
            </div>
          </div>
        </div>
      </div>

      {/* Smart Recommendations */}
      {recommendations.length > 0 && (
        <div className="mb-8">
          <SmartRecommendations recommendations={recommendations} />
        </div>
      )}

      {/* Results Header */}
      {!state.loading && state.works.length > 0 && (
        <div className="flex justify-between items-center mb-6">
          <h2 className="text-xl font-semibold text-gray-900">
            {state.pagination.total.toLocaleString()} Works Found
          </h2>
          <div className="text-sm text-gray-500">
            Page {state.pagination.page} of {state.pagination.total_pages}
          </div>
        </div>
      )}

      {/* Loading State */}
      {state.loading && (
        <div className="text-center py-12">
          <div className="inline-block animate-spin rounded-full h-8 w-8 border-b-2 border-orange-600"></div>
          <p className="mt-2 text-gray-600">Loading works...</p>
        </div>
      )}

      {/* Error State */}
      {state.error && (
        <div className="bg-red-50 border border-red-200 rounded-md p-4 mb-6">
          <div className="text-red-800">{state.error}</div>
          <button 
            onClick={loadWorks}
            className="mt-2 text-sm text-red-600 hover:text-red-500"
          >
            Try Again
          </button>
        </div>
      )}

      {/* Works List */}
      {!state.loading && !state.error && (
        <>
          {state.works.length === 0 ? (
            <div className="text-center py-12 bg-white rounded-lg shadow-sm">
              <h3 className="text-lg font-medium text-gray-900 mb-2">No works found</h3>
              <p className="text-gray-600 mb-4">
                Try adjusting your filters or search terms.
              </p>
              <button
                onClick={() => {
                  setFilters({
                    rating: '',
                    status: '',
                    sort: 'updated_at',
                    fandom: '',
                    relationship: '',
                    character: '',
                    tag: ''
                  });
                }}
                className="px-4 py-2 bg-orange-600 text-white rounded-md hover:bg-orange-700 focus:outline-none focus:ring-2 focus:ring-orange-500"
              >
                Clear All Filters
              </button>
            </div>
          ) : (
            <div>
              <SearchResults results={state.works} onWorkClick={handleWorkClick} />
              
              {/* Pagination */}
              {state.pagination.total_pages > 1 && (
                <div className="flex justify-center items-center gap-2 mt-8">
                  <button 
                    onClick={() => handlePageChange(state.pagination.page - 1)}
                    disabled={state.pagination.page <= 1}
                    className="px-3 py-2 text-sm border border-gray-300 rounded-md hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed focus:outline-none focus:ring-2 focus:ring-orange-500"
                  >
                    Previous
                  </button>
                  
                  <span className="px-4 py-2 text-sm text-gray-700">
                    Page {state.pagination.page} of {state.pagination.total_pages}
                  </span>
                  
                  <button 
                    onClick={() => handlePageChange(state.pagination.page + 1)}
                    disabled={state.pagination.page >= state.pagination.total_pages}
                    className="px-3 py-2 text-sm border border-gray-300 rounded-md hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed focus:outline-none focus:ring-2 focus:ring-orange-500"
                  >
                    Next
                  </button>
                </div>
              )}
            </div>
          )}
        </>
      )}
    </div>
  );
}
