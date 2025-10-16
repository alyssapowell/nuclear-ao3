'use client';

import { useState, useEffect } from 'react';
import { Search, Plus, User, Calendar, BookOpen, FileText } from 'lucide-react';
import { getMySeries, getUserSeries, createSeries, CreateSeriesRequest } from '@/lib/api';

interface Series {
  id: string;
  title: string;
  summary?: string;
  notes?: string;
  user_id: string;
  username?: string;
  is_complete: boolean;
  work_count: number;
  word_count?: number;
  created_at: string;
  updated_at: string;
}

interface Pagination {
  page: number;
  limit: number;
  total: number;
  total_pages: number;
}

interface SeriesResponse {
  series: Series[];
  pagination: Pagination;
}

export default function SeriesPage() {
  const [activeTab, setActiveTab] = useState<'browse' | 'my'>('browse');
  const [searchQuery, setSearchQuery] = useState('');
  const [series, setSeries] = useState<Series[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [pagination, setPagination] = useState<Pagination>({
    page: 1,
    limit: 20,
    total: 0,
    total_pages: 0
  });
  const [isAuthenticated, setIsAuthenticated] = useState(false);

  useEffect(() => {
    // Check authentication status
    const token = typeof window !== 'undefined' ? localStorage.getItem('auth_token') : null;
    setIsAuthenticated(!!token);
    
    if (activeTab === 'browse') {
      loadSeries();
    } else if (activeTab === 'my' && token) {
      loadMySeries();
    }
  }, [activeTab, pagination.page]);

  const loadSeries = async () => {
    try {
      setLoading(true);
      setError('');
      
      // For now, we'll show empty state as the backend series API might not be fully implemented
      setSeries([]);
      setPagination({
        page: 1,
        limit: 20,
        total: 0,
        total_pages: 0
      });
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load series');
    } finally {
      setLoading(false);
    }
  };

  const loadMySeries = async () => {
    try {
      setLoading(true);
      setError('');
      
      const token = localStorage.getItem('auth_token');
      if (!token) return;

      const response = await getMySeries({
        page: pagination.page,
        limit: pagination.limit
      }, token);

      setSeries(response.series || []);
      setPagination(response.pagination || pagination);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load your series');
    } finally {
      setLoading(false);
    }
  };

  const handleSearch = async () => {
    if (!searchQuery.trim()) {
      loadSeries();
      return;
    }

    try {
      setLoading(true);
      setError('');
      
      // Basic search implementation - for now showing empty results
      setSeries([]);
      setPagination({
        page: 1,
        limit: 20,
        total: 0,
        total_pages: 0
      });
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Search failed');
    } finally {
      setLoading(false);
    }
  };

  const formatDate = (dateString: string) => {
    if (!dateString) {
      return 'Unknown date';
    }
    
    const date = new Date(dateString);
    if (isNaN(date.getTime())) {
      return 'Invalid date';
    }
    
    return date.toLocaleDateString('en-US', {
      year: 'numeric',
      month: 'short',
      day: 'numeric'
    });
  };

  const formatNumber = (num: number) => {
    if (num >= 1000000) return `${(num / 1000000).toFixed(1)}M`;
    if (num >= 1000) return `${(num / 1000).toFixed(1)}K`;
    return num.toString();
  };

  const SeriesCard = ({ series }: { series: Series }) => (
    <div className="bg-white border border-gray-200 rounded-lg p-6 hover:shadow-md transition-shadow">
      <div className="flex justify-between items-start mb-3">
        <div className="flex-1">
          <a 
            href={`/series/${series.id}`}
            className="text-lg font-semibold text-blue-600 hover:text-blue-800"
          >
            {series.title}
          </a>
          <p className="text-sm text-gray-600 flex items-center mt-1">
            <User className="w-4 h-4 mr-1" />
            by {series.username}
          </p>
        </div>
        
        <div className="ml-4 text-right">
          <span className={`px-2 py-1 text-xs rounded-full ${
            series.is_complete 
              ? 'bg-green-100 text-green-800' 
              : 'bg-blue-100 text-blue-800'
          }`}>
            {series.is_complete ? 'Complete' : 'In Progress'}
          </span>
        </div>
      </div>

      {series.summary && (
        <p className="text-sm text-gray-700 mb-3 line-clamp-2">
          {series.summary}
        </p>
      )}

      <div className="flex justify-between items-center text-sm text-gray-500 border-t border-gray-100 pt-3">
        <div className="flex items-center space-x-4">
          <span className="flex items-center">
            <BookOpen className="w-4 h-4 mr-1" />
            {series.work_count} work{series.work_count !== 1 ? 's' : ''}
          </span>
          {series.word_count && (
            <span className="flex items-center">
              <FileText className="w-4 h-4 mr-1" />
              {formatNumber(series.word_count)} words
            </span>
          )}
        </div>
        
        <div className="flex items-center text-xs">
          <Calendar className="w-4 h-4 mr-1" />
          Updated {formatDate(series.updated_at)}
        </div>
      </div>
    </div>
  );

  return (
    <div className="max-w-7xl mx-auto px-4 py-8">
      {/* Header */}
      <div className="mb-8">
        <h1 className="text-3xl font-bold text-gray-900 mb-4">Series</h1>
        <p className="text-gray-600">
          Discover collections of related works organized by their creators.
        </p>
      </div>

      {/* Tab Navigation */}
      <div className="flex items-center justify-between mb-6">
        <div className="flex border-b border-gray-200">
          <button
            onClick={() => setActiveTab('browse')}
            className={`px-4 py-2 font-medium text-sm border-b-2 transition-colors ${
              activeTab === 'browse'
                ? 'border-orange-500 text-orange-600'
                : 'border-transparent text-gray-500 hover:text-gray-700'
            }`}
          >
            Browse All Series
          </button>
          {isAuthenticated && (
            <button
              onClick={() => setActiveTab('my')}
              className={`px-4 py-2 font-medium text-sm border-b-2 transition-colors ${
                activeTab === 'my'
                  ? 'border-orange-500 text-orange-600'
                  : 'border-transparent text-gray-500 hover:text-gray-700'
              }`}
            >
              My Series
            </button>
          )}
        </div>

        {isAuthenticated && (
          <a
            href="/series/new"
            className="inline-flex items-center px-4 py-2 bg-orange-600 text-white text-sm font-medium rounded-md hover:bg-orange-700 transition-colors"
          >
            <Plus className="w-4 h-4 mr-2" />
            Create Series
          </a>
        )}
      </div>

      {/* Search Section */}
      <div className="mb-6">
        <div className="flex gap-2">
          <div className="flex-1">
            <input
              type="text"
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              placeholder="Search series by title, author, or description..."
              className="w-full px-4 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-orange-500 focus:border-orange-500"
              onKeyPress={(e) => e.key === 'Enter' && handleSearch()}
            />
          </div>
          <button
            onClick={handleSearch}
            disabled={loading}
            className="px-6 py-2 bg-orange-600 text-white rounded-md hover:bg-orange-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
          >
            <Search className="w-4 h-4" />
          </button>
        </div>
      </div>

      {/* Content */}
      <div className="min-h-96">
        {loading ? (
          <div className="space-y-4">
            {[...Array(6)].map((_, i) => (
              <div key={i} className="bg-white border border-gray-200 rounded-lg p-6 animate-pulse">
                <div className="h-6 bg-gray-200 rounded w-3/4 mb-2"></div>
                <div className="h-4 bg-gray-200 rounded w-1/2 mb-3"></div>
                <div className="h-4 bg-gray-200 rounded w-full mb-2"></div>
                <div className="h-4 bg-gray-200 rounded w-2/3"></div>
              </div>
            ))}
          </div>
        ) : error ? (
          <div className="text-center py-12 bg-white rounded-lg shadow-sm border">
            <p className="text-red-600 mb-4">{error}</p>
            <button
              onClick={() => activeTab === 'browse' ? loadSeries() : loadMySeries()}
              className="px-4 py-2 bg-orange-600 text-white rounded-md hover:bg-orange-700 transition-colors"
            >
              Try Again
            </button>
          </div>
        ) : series.length > 0 ? (
          <>
            <div className="space-y-4 mb-8">
              {series.map((s) => (
                <SeriesCard key={s.id} series={s} />
              ))}
            </div>

            {/* Pagination */}
            {pagination.total_pages > 1 && (
              <div className="flex justify-center items-center space-x-2">
                <button
                  onClick={() => setPagination(prev => ({ ...prev, page: Math.max(1, prev.page - 1) }))}
                  disabled={pagination.page <= 1}
                  className="px-3 py-2 border border-gray-300 rounded-md disabled:opacity-50 disabled:cursor-not-allowed hover:bg-gray-50"
                >
                  Previous
                </button>
                
                {[...Array(pagination.total_pages)].map((_, i) => {
                  const pageNum = i + 1;
                  if (
                    pageNum === 1 ||
                    pageNum === pagination.total_pages ||
                    Math.abs(pageNum - pagination.page) <= 2
                  ) {
                    return (
                      <button
                        key={pageNum}
                        onClick={() => setPagination(prev => ({ ...prev, page: pageNum }))}
                        className={`px-3 py-2 border rounded-md ${
                          pageNum === pagination.page
                            ? 'bg-orange-600 text-white border-orange-600'
                            : 'border-gray-300 hover:bg-gray-50'
                        }`}
                      >
                        {pageNum}
                      </button>
                    );
                  } else if (
                    pageNum === pagination.page - 3 ||
                    pageNum === pagination.page + 3
                  ) {
                    return <span key={pageNum} className="px-2">...</span>;
                  }
                  return null;
                })}
                
                <button
                  onClick={() => setPagination(prev => ({ ...prev, page: Math.min(prev.total_pages, prev.page + 1) }))}
                  disabled={pagination.page >= pagination.total_pages}
                  className="px-3 py-2 border border-gray-300 rounded-md disabled:opacity-50 disabled:cursor-not-allowed hover:bg-gray-50"
                >
                  Next
                </button>
              </div>
            )}
          </>
        ) : (
          <div className="text-center py-12 bg-white rounded-lg shadow-sm border">
            {searchQuery ? (
              <>
                <h3 className="text-lg font-medium text-gray-900 mb-2">No series found</h3>
                <p className="text-gray-600 mb-4">
                  No series match your search criteria. Try different keywords or browse all series.
                </p>
                <button
                  onClick={() => {
                    setSearchQuery('');
                    loadSeries();
                  }}
                  className="px-4 py-2 bg-orange-600 text-white rounded-md hover:bg-orange-700 transition-colors"
                >
                  Clear Search
                </button>
              </>
            ) : activeTab === 'my' ? (
              <>
                <h3 className="text-lg font-medium text-gray-900 mb-2">You haven't created any series yet</h3>
                <p className="text-gray-600 mb-4">
                  Series allow you to group related works together in a specific order.
                </p>
                <a
                  href="/series/new"
                  className="inline-flex items-center px-4 py-2 bg-orange-600 text-white rounded-md hover:bg-orange-700 transition-colors"
                >
                  <Plus className="w-4 h-4 mr-2" />
                  Create Your First Series
                </a>
              </>
            ) : (
              <>
                <h3 className="text-lg font-medium text-gray-900 mb-2">No series available</h3>
                <p className="text-gray-600">
                  There are currently no series to browse. Check back later!
                </p>
              </>
            )}
          </div>
        )}
      </div>
    </div>
  );
}