'use client';

import { useState, useEffect } from 'react';
import Link from 'next/link';
import { useAuth } from '@/lib/auth';
import { getMyWorks, getUserBookmarks, getUserSubscriptions, deleteWork } from '@/lib/api';
import AuthGuard from '@/components/auth/AuthGuard';

interface Work {
  id: string;
  title: string;
  summary: string;
  rating: string;
  fandoms: string[];
  characters: string[];
  relationships: string[];
  freeform_tags: string[];
  word_count: number;
  chapter_count: number;
  max_chapters?: number;
  is_complete: boolean;
  status: string;
  published_at?: string;
  updated_at: string;
  hits: number;
  kudos: number;
  comments: number;
  bookmarks: number;
}

interface Bookmark {
  id: string;
  work_id: string;
  notes?: string;
  tags?: string[];
  is_private: boolean;
  created_at: string;
  work: Work;
}

interface Subscription {
  id: string;
  type: string;
  target_id: string;
  target_name: string;
  events: string[];
  frequency: string;
  created_at: string;
}

export default function DashboardPage() {
  const { user, token } = useAuth();
  const [loading, setLoading] = useState(true);
  const [activeTab, setActiveTab] = useState<'works' | 'bookmarks' | 'subscriptions'>('works');
  const [works, setWorks] = useState<Work[]>([]);
  const [bookmarks, setBookmarks] = useState<Bookmark[]>([]);
  const [subscriptions, setSubscriptions] = useState<Subscription[]>([]);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (token) {
      fetchDashboardData();
    }
  }, [token]);

  const fetchDashboardData = async () => {
    if (!token) return;
    
    setLoading(true);
    setError(null);
    
    try {
      // Fetch all dashboard data in parallel
      const [worksData, bookmarksData, subscriptionsData] = await Promise.allSettled([
        getMyWorks(token),
        getUserBookmarks(user?.id || '', {}, token),
        getUserSubscriptions(token)
      ]);

      // Handle works data
      if (worksData.status === 'fulfilled') {
        setWorks(worksData.value.works || []);
      }

      // Handle bookmarks data  
      if (bookmarksData.status === 'fulfilled') {
        setBookmarks(bookmarksData.value.bookmarks || []);
      }

      // Handle subscriptions data
      if (subscriptionsData.status === 'fulfilled') {
        setSubscriptions(subscriptionsData.value.subscriptions || []);
      }

    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load dashboard data');
    } finally {
      setLoading(false);
    }
  };

  const handleDeleteWork = async (workId: string) => {
    if (!token) return;
    
    const work = works.find(w => w.id === workId);
    if (!work) return;

    if (!confirm(`Are you sure you want to delete "${work.title}"? This cannot be undone.`)) {
      return;
    }

    try {
      await deleteWork(workId, token);
      setWorks(prev => prev.filter(w => w.id !== workId));
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete work');
    }
  };

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleDateString('en-US', {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
    });
  };

  const getStatusBadge = (status: string) => {
    const statusClasses = {
      'draft': 'bg-yellow-100 text-yellow-800',
      'posted': 'bg-green-100 text-green-800',
      'published': 'bg-green-100 text-green-800',
      'hidden': 'bg-gray-100 text-gray-800',
    };
    
    return (
      <span className={`inline-flex items-center px-2 py-1 rounded-full text-xs font-medium ${statusClasses[status as keyof typeof statusClasses] || statusClasses.draft}`}>
        {status === 'posted' ? 'Published' : status.charAt(0).toUpperCase() + status.slice(1)}
      </span>
    );
  };

  const getRatingBadge = (rating: string) => {
    const ratingClasses = {
      'General Audiences': 'bg-green-100 text-green-800',
      'Teen And Up Audiences': 'bg-blue-100 text-blue-800', 
      'Mature': 'bg-yellow-100 text-yellow-800',
      'Explicit': 'bg-red-100 text-red-800',
      'Not Rated': 'bg-gray-100 text-gray-800',
    };
    
    return (
      <span className={`inline-flex items-center px-2 py-1 rounded-full text-xs font-medium ${ratingClasses[rating as keyof typeof ratingClasses] || ratingClasses['Not Rated']}`}>
        {rating}
      </span>
    );
  };

  if (loading) {
    return (
      <AuthGuard>
        <div className="max-w-7xl mx-auto px-4 py-8">
          <div className="flex items-center justify-center min-h-64">
            <div className="animate-pulse flex items-center">
              <div className="h-8 w-8 bg-slate-200 rounded-full mr-3"></div>
              <div className="text-slate-600">Loading your dashboard...</div>
            </div>
          </div>
        </div>
      </AuthGuard>
    );
  }

  return (
    <AuthGuard>
      <div className="max-w-7xl mx-auto px-4 py-8">
        {/* Header */}
        <div className="mb-8">
          <h1 className="text-3xl font-bold text-slate-900">Dashboard</h1>
          <p className="text-slate-600 mt-2">Welcome back, {user?.username}!</p>
        </div>

        {error && (
          <div className="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded mb-6">
            {error}
          </div>
        )}

        {/* Quick Stats */}
        <div className="grid grid-cols-1 md:grid-cols-3 gap-6 mb-8">
          <div className="bg-white rounded-lg shadow-sm border border-slate-200 p-6">
            <div className="flex items-center">
              <div className="flex-shrink-0">
                <svg className="h-8 w-8 text-orange-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 6.253v13m0-13C10.832 5.477 9.246 5 7.5 5S4.168 5.477 3 6.253v13C4.168 18.477 5.754 18 7.5 18s3.332.477 4.5 1.253m0-13C13.168 5.477 14.754 5 16.5 5c1.746 0 3.332.477 4.5 1.253v13C19.832 18.477 18.246 18 16.5 18c-1.746 0-3.332.477-4.5 1.253" />
                </svg>
              </div>
              <div className="ml-5 w-0 flex-1">
                <dl>
                  <dt className="text-sm font-medium text-slate-500 truncate">Your Works</dt>
                  <dd className="text-2xl font-bold text-slate-900">{works.length}</dd>
                </dl>
              </div>
            </div>
          </div>

          <div className="bg-white rounded-lg shadow-sm border border-slate-200 p-6">
            <div className="flex items-center">
              <div className="flex-shrink-0">
                <svg className="h-8 w-8 text-blue-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 5a2 2 0 012-2h10a2 2 0 012 2v16l-7-3.5L5 21V5z" />
                </svg>
              </div>
              <div className="ml-5 w-0 flex-1">
                <dl>
                  <dt className="text-sm font-medium text-slate-500 truncate">Bookmarks</dt>
                  <dd className="text-2xl font-bold text-slate-900">{bookmarks.length}</dd>
                </dl>
              </div>
            </div>
          </div>

          <div className="bg-white rounded-lg shadow-sm border border-slate-200 p-6">
            <div className="flex items-center">
              <div className="flex-shrink-0">
                <svg className="h-8 w-8 text-green-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 17h5l-5 5v-5zM4.828 7l-.867.867 1.768 1.768L4.828 10.5 8 13.657 12 17l1.414-1.414L8.586 12 5.414 8.828 4.828 7z" />
                </svg>
              </div>
              <div className="ml-5 w-0 flex-1">
                <dl>
                  <dt className="text-sm font-medium text-slate-500 truncate">Subscriptions</dt>
                  <dd className="text-2xl font-bold text-slate-900">{subscriptions.length}</dd>
                </dl>
              </div>
            </div>
          </div>
        </div>

        {/* Tab Navigation */}
        <div className="border-b border-slate-200 mb-8">
          <nav className="-mb-px flex space-x-8">
            <button
              onClick={() => setActiveTab('works')}
              className={`py-2 px-1 border-b-2 font-medium text-sm ${
                activeTab === 'works'
                  ? 'border-orange-500 text-orange-600'
                  : 'border-transparent text-slate-500 hover:text-slate-700 hover:border-slate-300'
              }`}
            >
              My Works ({works.length})
            </button>
            <button
              onClick={() => setActiveTab('bookmarks')}
              className={`py-2 px-1 border-b-2 font-medium text-sm ${
                activeTab === 'bookmarks'
                  ? 'border-orange-500 text-orange-600'
                  : 'border-transparent text-slate-500 hover:text-slate-700 hover:border-slate-300'
              }`}
            >
              Bookmarks ({bookmarks.length})
            </button>
            <button
              onClick={() => setActiveTab('subscriptions')}
              className={`py-2 px-1 border-b-2 font-medium text-sm ${
                activeTab === 'subscriptions'
                  ? 'border-orange-500 text-orange-600'
                  : 'border-transparent text-slate-500 hover:text-slate-700 hover:border-slate-300'
              }`}
            >
              Subscriptions ({subscriptions.length})
            </button>
          </nav>
        </div>

        {/* Content Tabs */}
        {activeTab === 'works' && (
          <div>
            <div className="flex justify-between items-center mb-6">
              <h2 className="text-xl font-semibold text-slate-900">My Works</h2>
              <Link
                href="/works/new"
                className="bg-orange-600 text-white px-4 py-2 rounded-md hover:bg-orange-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-orange-500"
              >
                + Create New Work
              </Link>
            </div>

            {works.length === 0 ? (
              <div className="text-center py-12">
                <svg className="w-16 h-16 text-slate-400 mx-auto mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1} d="M12 6.253v13m0-13C10.832 5.477 9.246 5 7.5 5S4.168 5.477 3 6.253v13C4.168 18.477 5.754 18 7.5 18s3.332.477 4.5 1.253m0-13C13.168 5.477 14.754 5 16.5 5c1.746 0 3.332.477 4.5 1.253v13C19.832 18.477 18.246 18 16.5 18c-1.746 0-3.332.477-4.5 1.253" />
                </svg>
                <h3 className="text-lg font-medium text-slate-900 mb-2">No works yet</h3>
                <p className="text-slate-600 mb-4">Start writing and share your stories with the world!</p>
                <Link
                  href="/works/new"
                  className="bg-orange-600 text-white px-4 py-2 rounded-md hover:bg-orange-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-orange-500"
                >
                  Create Your First Work
                </Link>
              </div>
            ) : (
              <div className="space-y-6">
                {works.map((work) => (
                  <div key={work.id} className="bg-white rounded-lg shadow-sm border border-slate-200 p-6">
                    <div className="flex justify-between items-start mb-4">
                      <div className="flex-1">
                        <div className="flex items-center gap-3 mb-2">
                          <Link
                            href={`/works/${work.id}`}
                            className="text-xl font-bold text-slate-900 hover:text-orange-600"
                          >
                            {work.title}
                          </Link>
                          {getStatusBadge(work.status)}
                          {getRatingBadge(work.rating)}
                        </div>
                        
                        {work.summary && (
                          <p className="text-slate-600 mb-3 line-clamp-3">{work.summary}</p>
                        )}
                        
                        <div className="flex flex-wrap gap-2 mb-3">
                          {work.fandoms.slice(0, 3).map((fandom, index) => (
                            <span key={index} className="inline-flex items-center px-2 py-1 rounded-full text-xs font-medium bg-blue-100 text-blue-800">
                              {fandom}
                            </span>
                          ))}
                          {work.fandoms.length > 3 && (
                            <span className="text-xs text-slate-500">+{work.fandoms.length - 3} more</span>
                          )}
                        </div>
                      </div>
                      
                      <div className="flex items-center gap-2 ml-4">
                        <Link
                          href={`/works/${work.id}/edit`}
                          className="text-orange-600 hover:text-orange-700 text-sm font-medium"
                        >
                          Edit
                        </Link>
                        <button
                          onClick={() => handleDeleteWork(work.id)}
                          className="text-red-600 hover:text-red-700 text-sm font-medium"
                        >
                          Delete
                        </button>
                      </div>
                    </div>

                    <div className="grid grid-cols-2 md:grid-cols-4 gap-4 text-sm">
                      <div>
                        <span className="text-slate-500">Words:</span>
                        <span className="ml-1 font-medium">{work.word_count.toLocaleString()}</span>
                      </div>
                      <div>
                        <span className="text-slate-500">Chapters:</span>
                        <span className="ml-1 font-medium">
                          {work.chapter_count}{work.max_chapters ? `/${work.max_chapters}` : '/?'}
                        </span>
                      </div>
                      <div>
                        <span className="text-slate-500">Kudos:</span>
                        <span className="ml-1 font-medium">{work.kudos}</span>
                      </div>
                      <div>
                        <span className="text-slate-500">Hits:</span>
                        <span className="ml-1 font-medium">{work.hits}</span>
                      </div>
                    </div>

                    {(work.published_at || work.updated_at) && (
                      <div className="mt-4 pt-4 border-t border-slate-200 text-sm text-slate-500">
                        {work.published_at && work.status === 'posted' && (
                          <span>Published: {formatDate(work.published_at)} • </span>
                        )}
                        <span>Updated: {formatDate(work.updated_at)}</span>
                      </div>
                    )}
                  </div>
                ))}
              </div>
            )}
          </div>
        )}

        {activeTab === 'bookmarks' && (
          <div>
            <h2 className="text-xl font-semibold text-slate-900 mb-6">My Bookmarks</h2>
            
            {bookmarks.length === 0 ? (
              <div className="text-center py-12">
                <svg className="w-16 h-16 text-slate-400 mx-auto mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1} d="M5 5a2 2 0 012-2h10a2 2 0 012 2v16l-7-3.5L5 21V5z" />
                </svg>
                <h3 className="text-lg font-medium text-slate-900 mb-2">No bookmarks yet</h3>
                <p className="text-slate-600">Start bookmarking works you want to read later!</p>
              </div>
            ) : (
              <div className="space-y-6">
                {bookmarks.map((bookmark) => (
                  <div key={bookmark.id} className="bg-white rounded-lg shadow-sm border border-slate-200 p-6">
                    <div className="flex justify-between items-start mb-4">
                      <div className="flex-1">
                        <Link
                          href={`/works/${bookmark.work.id}`}
                          className="text-xl font-bold text-slate-900 hover:text-orange-600 block mb-2"
                        >
                          {bookmark.work.title}
                        </Link>
                        
                        {bookmark.work.summary && (
                          <p className="text-slate-600 mb-3 line-clamp-3">{bookmark.work.summary}</p>
                        )}
                        
                        {bookmark.notes && (
                          <div className="bg-slate-50 rounded p-3 mb-3">
                            <p className="text-sm text-slate-700">
                              <span className="font-medium">Your Notes:</span> {bookmark.notes}
                            </p>
                          </div>
                        )}
                      </div>
                      
                      <div className="ml-4">
                        {bookmark.is_private && (
                          <span className="inline-flex items-center px-2 py-1 rounded-full text-xs font-medium bg-gray-100 text-gray-800">
                            Private
                          </span>
                        )}
                      </div>
                    </div>

                    <div className="grid grid-cols-2 md:grid-cols-4 gap-4 text-sm">
                      <div>
                        <span className="text-slate-500">Words:</span>
                        <span className="ml-1 font-medium">{bookmark.work.word_count.toLocaleString()}</span>
                      </div>
                      <div>
                        <span className="text-slate-500">Chapters:</span>
                        <span className="ml-1 font-medium">
                          {bookmark.work.chapter_count}{bookmark.work.max_chapters ? `/${bookmark.work.max_chapters}` : '/?'}
                        </span>
                      </div>
                      <div>
                        <span className="text-slate-500">Rating:</span>
                        <span className="ml-1 font-medium">{bookmark.work.rating}</span>
                      </div>
                      <div>
                        <span className="text-slate-500">Bookmarked:</span>
                        <span className="ml-1 font-medium">{formatDate(bookmark.created_at)}</span>
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        )}

        {activeTab === 'subscriptions' && (
          <div>
            <h2 className="text-xl font-semibold text-slate-900 mb-6">My Subscriptions</h2>
            
            {subscriptions.length === 0 ? (
              <div className="text-center py-12">
                <svg className="w-16 h-16 text-slate-400 mx-auto mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1} d="M15 17h5l-5 5v-5zM4.828 7l-.867.867 1.768 1.768L4.828 10.5 8 13.657 12 17l1.414-1.414L8.586 12 5.414 8.828 4.828 7z" />
                </svg>
                <h3 className="text-lg font-medium text-slate-900 mb-2">No subscriptions yet</h3>
                <p className="text-slate-600">Subscribe to works, authors, or tags to get notifications!</p>
              </div>
            ) : (
              <div className="space-y-6">
                {subscriptions.map((subscription) => (
                  <div key={subscription.id} className="bg-white rounded-lg shadow-sm border border-slate-200 p-6">
                    <div className="flex justify-between items-start">
                      <div className="flex-1">
                        <div className="flex items-center gap-3 mb-2">
                          <h3 className="text-lg font-semibold text-slate-900">{subscription.target_name}</h3>
                          <span className="inline-flex items-center px-2 py-1 rounded-full text-xs font-medium bg-orange-100 text-orange-800">
                            {subscription.type}
                          </span>
                        </div>
                        
                        <div className="text-sm text-slate-600 mb-2">
                          <span className="font-medium">Events:</span> {subscription.events.join(', ')}
                        </div>
                        
                        <div className="text-sm text-slate-600">
                          <span className="font-medium">Frequency:</span> {subscription.frequency}
                        </div>
                      </div>
                      
                      <div className="text-sm text-slate-500">
                        Subscribed: {formatDate(subscription.created_at)}
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        )}
      </div>
    </AuthGuard>
  );
}