'use client';

import { useState, useEffect } from 'react';
import WorkCard from '@/components/WorkCard';
import { Button } from '@/components/ui/Button';
import { getMyBookmarks, type BookmarkSearchParams, type Bookmark } from '@/lib/api';

interface BookmarksPageState {
  bookmarks: Bookmark[];
  loading: boolean;
  error: string;
  pagination: {
    page: number;
    limit: number;
    total: number;
    total_pages: number;
  };
}

export default function BookmarksPage() {
  const [state, setState] = useState<BookmarksPageState>({
    bookmarks: [],
    loading: true,
    error: '',
    pagination: { page: 1, limit: 20, total: 0, total_pages: 0 }
  });
  
  const [searchQuery, setSearchQuery] = useState('');
  const [selectedTag, setSelectedTag] = useState('');
  const [showPrivate, setShowPrivate] = useState(true);

  // Mock auth token - in real app this would come from auth context
  const authToken = typeof window !== 'undefined' ? localStorage.getItem('auth_token') : null;

  const loadBookmarks = async (params: BookmarkSearchParams = {}) => {
    if (!authToken) {
      setState(prev => ({ 
        ...prev, 
        loading: false, 
        error: 'Please log in to view your bookmarks' 
      }));
      return;
    }

    setState(prev => ({ ...prev, loading: true, error: '' }));

    try {
      const response = await getMyBookmarks(params, authToken);
      setState(prev => ({
        ...prev,
        bookmarks: response.bookmarks,
        pagination: response.pagination,
        loading: false
      }));
    } catch (error) {
      setState(prev => ({
        ...prev,
        error: error instanceof Error ? error.message : 'Failed to load bookmarks',
        loading: false
      }));
    }
  };

  useEffect(() => {
    loadBookmarks({ page: 1, limit: 20 });
  }, [authToken]);

  const handleSearch = () => {
    const params: BookmarkSearchParams = {
      page: 1,
      limit: 20,
    };
    
    if (searchQuery.trim()) {
      params.q = searchQuery.trim();
    }
    
    if (selectedTag.trim()) {
      params.tag = selectedTag.trim();
    }
    
    loadBookmarks(params);
  };

  const handlePageChange = (newPage: number) => {
    const params: BookmarkSearchParams = {
      page: newPage,
      limit: 20,
    };
    
    if (searchQuery.trim()) {
      params.q = searchQuery.trim();
    }
    
    if (selectedTag.trim()) {
      params.tag = selectedTag.trim();
    }
    
    loadBookmarks(params);
  };

  const clearFilters = () => {
    setSearchQuery('');
    setSelectedTag('');
    loadBookmarks({ page: 1, limit: 20 });
  };

  // Get unique tags from bookmarks for filtering
  const allTags = new Set<string>();
  state.bookmarks.forEach(bookmark => {
    bookmark.tags?.forEach(tag => allTags.add(tag));
  });
  const uniqueTags = Array.from(allTags).sort();

  if (!authToken) {
    return (
      <div className="container mx-auto px-4 py-8">
        <div className="text-center">
          <h1 className="text-3xl font-bold mb-4">My Bookmarks</h1>
          <p className="text-gray-600 mb-4">Please log in to view your bookmarks.</p>
          <Button>Log In</Button>
        </div>
      </div>
    );
  }

  return (
    <div className="container mx-auto px-4 py-8">
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-3xl font-bold">My Bookmarks</h1>
        <div className="text-sm text-gray-600">
          {state.pagination.total} bookmark{state.pagination.total !== 1 ? 's' : ''}
        </div>
      </div>

      {/* Search and Filters */}
      <div className="bg-white p-4 rounded-lg shadow mb-6">
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-4">
          <div>
            <label htmlFor="search" className="block text-sm font-medium mb-1">
              Search bookmarks
            </label>
            <input
              type="text"
              id="search"
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="w-full p-2 border border-gray-300 rounded-md"
              placeholder="Search in titles, summaries, or notes..."
              onKeyPress={(e) => e.key === 'Enter' && handleSearch()}
            />
          </div>

          <div>
            <label htmlFor="tag" className="block text-sm font-medium mb-1">
              Filter by tag
            </label>
            <select
              id="tag"
              value={selectedTag}
              onChange={(e) => setSelectedTag(e.target.value)}
              className="w-full p-2 border border-gray-300 rounded-md"
            >
              <option value="">All tags</option>
              {uniqueTags.map(tag => (
                <option key={tag} value={tag}>{tag}</option>
              ))}
            </select>
          </div>

          <div className="flex items-end gap-2">
            <Button onClick={handleSearch} className="flex-1">
              Search
            </Button>
            <Button variant="outline" onClick={clearFilters}>
              Clear
            </Button>
          </div>
        </div>

        <div className="flex items-center">
          <input
            type="checkbox"
            id="showPrivate"
            checked={showPrivate}
            onChange={(e) => setShowPrivate(e.target.checked)}
            className="mr-2"
          />
          <label htmlFor="showPrivate" className="text-sm">
            Show private bookmarks
          </label>
        </div>
      </div>

      {/* Loading State */}
      {state.loading && (
        <div className="text-center py-8">
          <div className="text-gray-600">Loading bookmarks...</div>
        </div>
      )}

      {/* Error State */}
      {state.error && (
        <div className="bg-red-50 border border-red-200 rounded-md p-4 mb-6">
          <div className="text-red-800">{state.error}</div>
        </div>
      )}

      {/* Bookmarks List */}
      {!state.loading && !state.error && (
        <>
          {state.bookmarks.length === 0 ? (
            <div className="text-center py-8">
              <div className="text-gray-600 mb-4">
                {searchQuery || selectedTag 
                  ? 'No bookmarks match your search criteria.' 
                  : 'You haven\'t bookmarked any works yet.'
                }
              </div>
              {(searchQuery || selectedTag) && (
                <Button variant="outline" onClick={clearFilters}>
                  Show all bookmarks
                </Button>
              )}
            </div>
          ) : (
            <div className="space-y-6">
              {state.bookmarks
                .filter(bookmark => showPrivate || !bookmark.is_private)
                .map(bookmark => (
                  <BookmarkCard 
                    key={bookmark.id} 
                    bookmark={bookmark}
                    onUpdate={() => loadBookmarks({
                      page: state.pagination.page,
                      limit: state.pagination.limit,
                      q: searchQuery || undefined,
                      tag: selectedTag || undefined
                    })}
                  />
                ))
              }
            </div>
          )}

          {/* Pagination */}
          {state.pagination.total_pages > 1 && (
            <div className="flex justify-center items-center gap-2 mt-8">
              <Button 
                variant="outline"
                onClick={() => handlePageChange(state.pagination.page - 1)}
                disabled={state.pagination.page <= 1}
              >
                Previous
              </Button>
              
              <span className="px-4 py-2">
                Page {state.pagination.page} of {state.pagination.total_pages}
              </span>
              
              <Button 
                variant="outline"
                onClick={() => handlePageChange(state.pagination.page + 1)}
                disabled={state.pagination.page >= state.pagination.total_pages}
              >
                Next
              </Button>
            </div>
          )}
        </>
      )}
    </div>
  );
}

interface BookmarkCardProps {
  bookmark: Bookmark;
  onUpdate: () => void;
}

function BookmarkCard({ bookmark, onUpdate }: BookmarkCardProps) {
  const [showEditDialog, setShowEditDialog] = useState(false);

  // Convert bookmark.work to WorkCard format
  const convertBookmarkWork = (work: Record<string, unknown>) => ({
    id: String(work.id || ''),
    title: String(work.title || ''),
    author: String(work.author || work.username || ''),
    summary: work.summary ? String(work.summary) : undefined,
    word_count: Number(work.word_count || 0),
    chapter_count: Number(work.chapter_count || 0),
    max_chapters: work.max_chapters ? Number(work.max_chapters) : undefined,
    rating: String(work.rating || ''),
    status: String(work.status || ''),
    language: String(work.language || ''),
    published_date: String(work.published_date || ''),
    updated_date: String(work.updated_date || ''),
    relationships: Array.isArray(work.relationships) ? work.relationships : [],
    characters: Array.isArray(work.characters) ? work.characters : [],
    freeform_tags: Array.isArray(work.freeform_tags) ? work.freeform_tags : [],
    fandoms: Array.isArray(work.fandoms) ? work.fandoms : [],
    kudos_count: work.kudos_count ? Number(work.kudos_count) : undefined,
    bookmark_count: work.bookmark_count ? Number(work.bookmark_count) : undefined,
    hit_count: work.hit_count ? Number(work.hit_count) : undefined,
    comment_count: work.comment_count ? Number(work.comment_count) : undefined,
    tag_quality_score: work.tag_quality_score ? Number(work.tag_quality_score) : undefined,
    missing_tag_suggestions: Array.isArray(work.missing_tag_suggestions) ? work.missing_tag_suggestions : undefined,
  });

  return (
    <div className="bg-white border border-gray-200 rounded-lg p-6">
      <div className="flex justify-between items-start mb-4">
        <div className="flex-1">
          {bookmark.work && <WorkCard work={convertBookmarkWork(bookmark.work)} />}
        </div>
        <div className="ml-4 flex gap-2">
          <Button 
            size="sm" 
            variant="outline"
            onClick={() => setShowEditDialog(true)}
          >
            Edit
          </Button>
          {bookmark.is_private && (
            <span className="px-2 py-1 bg-gray-100 text-gray-700 text-xs rounded">
              Private
            </span>
          )}
        </div>
      </div>

      {/* Bookmark Notes */}
      {bookmark.notes && (
        <div className="mb-4">
          <h4 className="font-medium text-sm text-gray-700 mb-1">My Notes:</h4>
          <div className="text-sm text-gray-600 bg-gray-50 p-3 rounded">
            {bookmark.notes}
          </div>
        </div>
      )}

      {/* Bookmark Tags */}
      {bookmark.tags && bookmark.tags.length > 0 && (
        <div className="mb-4">
          <h4 className="font-medium text-sm text-gray-700 mb-1">My Tags:</h4>
          <div className="flex flex-wrap gap-1">
            {bookmark.tags.map(tag => (
              <span 
                key={tag}
                className="px-2 py-1 bg-blue-100 text-blue-800 text-xs rounded"
              >
                {tag}
              </span>
            ))}
          </div>
        </div>
      )}

      {/* Bookmark Date */}
      <div className="text-xs text-gray-500">
        Bookmarked {new Date(bookmark.created_at).toLocaleDateString()}
        {bookmark.updated_at !== bookmark.created_at && (
          <span> • Updated {new Date(bookmark.updated_at).toLocaleDateString()}</span>
        )}
      </div>

      {/* Edit Dialog would go here */}
      {showEditDialog && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
          <div className="bg-white rounded-lg p-4 max-w-md w-full mx-4">
            <h3 className="font-bold mb-2">Edit Bookmark</h3>
            <p className="text-sm text-gray-600 mb-4">
              Bookmark editing interface would be implemented here.
            </p>
            <Button onClick={() => setShowEditDialog(false)}>
              Close
            </Button>
          </div>
        </div>
      )}
    </div>
  );
}