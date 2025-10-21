'use client';

import { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import Link from 'next/link';
import { PlusCircle, BookOpen, Heart, MessageCircle, TrendingUp, Clock, Settings, Edit3 } from 'lucide-react';
import { getMyWorks, getMyBookmarks } from '@/lib/api';

interface Work {
  id: string;
  title: string;
  summary?: string;
  word_count: number;
  chapter_count: number;
  max_chapters?: number;
  is_complete: boolean;
  status: string;
  published_at: string;
  updated_at: string;
  hits: number;
  kudos: number;
  comments: number;
  bookmarks: number;
}

interface DashboardStats {
  totalWorks: number;
  totalWordCount: number;
  totalKudos: number;
  totalComments: number;
  totalBookmarks: number;
  totalHits: number;
}

export default function DashboardPage() {
  const [user, setUser] = useState<any>(null);
  const [works, setWorks] = useState<Work[]>([]);
  const [stats, setStats] = useState<DashboardStats>({
    totalWorks: 0,
    totalWordCount: 0,
    totalKudos: 0,
    totalComments: 0,
    totalBookmarks: 0,
    totalHits: 0
  });
  const [loading, setLoading] = useState(true);
  const [recentActivity, setRecentActivity] = useState<any[]>([]);
  const router = useRouter();

  useEffect(() => {
    fetchDashboardData();
  }, [router]);

  const fetchDashboardData = async () => {
    // Check if user is authenticated
    const token = localStorage.getItem('auth_token') || localStorage.getItem('token');
    if (!token) {
      router.replace('/auth/login?redirect=/dashboard');
      return;
    }

    try {
      // Set dummy user data for now
      setUser({
        username: 'Nuclear User',
        email: 'user@nuclear-ao3.com'
      });

      // Fetch user's works
      const worksData = await getMyWorks(token);
      const userWorks = worksData.works || [];
      setWorks(userWorks);

      // Calculate stats
      const calculatedStats = userWorks.reduce((acc: DashboardStats, work: Work) => ({
        totalWorks: acc.totalWorks + 1,
        totalWordCount: acc.totalWordCount + work.word_count,
        totalKudos: acc.totalKudos + (work.kudos || 0),
        totalComments: acc.totalComments + (work.comments || 0),
        totalBookmarks: acc.totalBookmarks + (work.bookmarks || 0),
        totalHits: acc.totalHits + (work.hits || 0)
      }), {
        totalWorks: 0,
        totalWordCount: 0,
        totalKudos: 0,
        totalComments: 0,
        totalBookmarks: 0,
        totalHits: 0
      });

      setStats(calculatedStats);

      // Generate recent activity
      const activity = userWorks
        .slice(0, 5)
        .map(work => ({
          type: 'work_updated',
          work: work,
          timestamp: work.updated_at
        }))
        .sort((a, b) => new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime());

      setRecentActivity(activity);

    } catch (error) {
      console.error('Failed to fetch dashboard data:', error);
    } finally {
      setLoading(false);
    }
  };

  if (loading) {
    return (
      <div className="flex justify-center items-center min-h-screen">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-orange-600"></div>
      </div>
    );
  }

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleDateString('en-US', {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
    });
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'posted': return 'bg-green-100 text-green-800';
      case 'draft': return 'bg-yellow-100 text-yellow-800';
      case 'in_progress': return 'bg-blue-100 text-blue-800';
      default: return 'bg-gray-100 text-gray-800';
    }
  };

  return (
    <div className="max-w-7xl mx-auto px-4 py-8">
      {/* Header */}
      <div className="mb-8">
        <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between">
          <div>
            <h1 className="text-3xl font-bold text-gray-900 mb-2">Dashboard</h1>
            <p className="text-gray-600">Welcome back{user?.username ? `, ${user.username}` : ''}!</p>
          </div>
          <div className="mt-4 sm:mt-0 flex space-x-3">
            <Link 
              href="/works/new"
              className="inline-flex items-center px-4 py-2 bg-orange-600 text-white text-sm font-medium rounded-md hover:bg-orange-700 transition-colors"
            >
              <PlusCircle className="h-4 w-4 mr-2" />
              New Work
            </Link>
            <Link 
              href="/profile"
              className="inline-flex items-center px-4 py-2 border border-gray-300 text-gray-700 text-sm font-medium rounded-md hover:bg-gray-50 transition-colors"
            >
              <Settings className="h-4 w-4 mr-2" />
              Settings
            </Link>
          </div>
        </div>
      </div>

      {/* Stats Overview */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-6 mb-8">
        <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-6">
          <div className="flex items-center">
            <div className="flex-shrink-0">
              <BookOpen className="h-8 w-8 text-blue-600" />
            </div>
            <div className="ml-4">
              <p className="text-sm font-medium text-gray-500">Works</p>
              <p className="text-2xl font-semibold text-gray-900">{stats.totalWorks}</p>
            </div>
          </div>
        </div>

        <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-6">
          <div className="flex items-center">
            <div className="flex-shrink-0">
              <Edit3 className="h-8 w-8 text-green-600" />
            </div>
            <div className="ml-4">
              <p className="text-sm font-medium text-gray-500">Words</p>
              <p className="text-2xl font-semibold text-gray-900">{stats.totalWordCount.toLocaleString()}</p>
            </div>
          </div>
        </div>

        <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-6">
          <div className="flex items-center">
            <div className="flex-shrink-0">
              <Heart className="h-8 w-8 text-red-600" />
            </div>
            <div className="ml-4">
              <p className="text-sm font-medium text-gray-500">Kudos</p>
              <p className="text-2xl font-semibold text-gray-900">{stats.totalKudos}</p>
            </div>
          </div>
        </div>

        <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-6">
          <div className="flex items-center">
            <div className="flex-shrink-0">
              <TrendingUp className="h-8 w-8 text-purple-600" />
            </div>
            <div className="ml-4">
              <p className="text-sm font-medium text-gray-500">Total Hits</p>
              <p className="text-2xl font-semibold text-gray-900">{stats.totalHits.toLocaleString()}</p>
            </div>
          </div>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
        {/* Recent Works */}
        <div className="lg:col-span-2">
          <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-6">
            <div className="flex items-center justify-between mb-6">
              <h2 className="text-xl font-semibold text-gray-900">Your Works</h2>
              <Link 
                href="/works" 
                className="text-sm text-orange-600 hover:text-orange-700 font-medium"
              >
                View All →
              </Link>
            </div>
            
            {works.length > 0 ? (
              <div className="space-y-4">
                {works.slice(0, 5).map((work) => (
                  <div key={work.id} className="border border-gray-200 rounded-lg p-4 hover:shadow-md transition-shadow">
                    <div className="flex items-start justify-between">
                      <div className="flex-1 min-w-0">
                        <Link 
                          href={`/works/${work.id}`}
                          className="text-lg font-medium text-blue-600 hover:text-blue-700 block truncate"
                        >
                          {work.title}
                        </Link>
                        {work.summary && (
                          <p className="text-sm text-gray-600 mt-1 line-clamp-2">{work.summary}</p>
                        )}
                        <div className="flex items-center space-x-4 mt-2 text-sm text-gray-500">
                          <span>{work.word_count.toLocaleString()} words</span>
                          <span>
                            {work.chapter_count}{work.max_chapters ? `/${work.max_chapters}` : '/?'} chapters
                          </span>
                          <span>Updated {formatDate(work.updated_at)}</span>
                        </div>
                      </div>
                      <div className="flex flex-col items-end space-y-2 ml-4">
                        <span className={`px-2 py-1 text-xs font-medium rounded-full ${getStatusColor(work.status)}`}>
                          {work.status}
                        </span>
                        <div className="flex items-center space-x-3 text-sm text-gray-500">
                          <span className="flex items-center">
                            <Heart className="h-3 w-3 mr-1" />
                            {work.kudos || 0}
                          </span>
                          <span className="flex items-center">
                            <MessageCircle className="h-3 w-3 mr-1" />
                            {work.comments || 0}
                          </span>
                        </div>
                        <Link 
                          href={`/works/${work.id}/edit`}
                          className="text-xs text-orange-600 hover:text-orange-700 font-medium"
                        >
                          Edit
                        </Link>
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            ) : (
              <div className="text-center py-8">
                <BookOpen className="mx-auto h-12 w-12 text-gray-400" />
                <h3 className="mt-2 text-sm font-medium text-gray-900">No works yet</h3>
                <p className="mt-1 text-sm text-gray-500">Get started by posting your first work.</p>
                <div className="mt-6">
                  <Link
                    href="/works/new"
                    className="inline-flex items-center px-4 py-2 border border-transparent shadow-sm text-sm font-medium rounded-md text-white bg-orange-600 hover:bg-orange-700"
                  >
                    <PlusCircle className="h-4 w-4 mr-2" />
                    Post New Work
                  </Link>
                </div>
              </div>
            )}
          </div>
        </div>

        {/* Sidebar */}
        <div className="space-y-6">
          {/* Quick Actions */}
          <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-6">
            <h3 className="text-lg font-semibold text-gray-900 mb-4">Quick Actions</h3>
            <div className="space-y-3">
              <Link 
                href="/works/new" 
                className="flex items-center w-full text-left px-4 py-3 bg-orange-50 text-orange-700 rounded-md hover:bg-orange-100 transition-colors"
              >
                <PlusCircle className="h-5 w-5 mr-3" />
                Post New Work
              </Link>
              <Link 
                href="/bookmarks" 
                className="flex items-center w-full text-left px-4 py-3 bg-blue-50 text-blue-700 rounded-md hover:bg-blue-100 transition-colors"
              >
                <BookOpen className="h-5 w-5 mr-3" />
                My Bookmarks
              </Link>
              <Link 
                href="/search" 
                className="flex items-center w-full text-left px-4 py-3 bg-purple-50 text-purple-700 rounded-md hover:bg-purple-100 transition-colors"
              >
                <TrendingUp className="h-5 w-5 mr-3" />
                Discover Works
              </Link>
            </div>
          </div>

          {/* Recent Activity */}
          <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-6">
            <h3 className="text-lg font-semibold text-gray-900 mb-4">Recent Activity</h3>
            {recentActivity.length > 0 ? (
              <div className="space-y-3">
                {recentActivity.map((activity, index) => (
                  <div key={index} className="flex items-start space-x-3">
                    <Clock className="h-4 w-4 text-gray-400 mt-1" />
                    <div className="flex-1 min-w-0">
                      <p className="text-sm text-gray-900">
                        Updated <Link href={`/works/${activity.work.id}`} className="font-medium text-blue-600 hover:text-blue-700">
                          {activity.work.title}
                        </Link>
                      </p>
                      <p className="text-xs text-gray-500">{formatDate(activity.timestamp)}</p>
                    </div>
                  </div>
                ))}
              </div>
            ) : (
              <p className="text-sm text-gray-500">No recent activity.</p>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}