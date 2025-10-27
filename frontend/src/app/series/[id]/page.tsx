'use client';

import { useState, useEffect } from 'react';
import Link from 'next/link';
import { useParams } from 'next/navigation';
import { getSeries, getSeriesWorks, Series } from '@/lib/api';
import WorkCard from '@/components/WorkCard';

interface Work {
  id: string;
  title: string;
  summary: string;
  rating: string;
  category: string[];
  warnings: string[];
  fandoms: string[];
  characters: string[];
  relationships: string[];
  freeform_tags: string[];
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
  username?: string;
  position?: number;
}

export default function SeriesPage() {
  const params = useParams();
  const seriesId = params.id as string;
  
  const [series, setSeries] = useState<Series | null>(null);
  const [works, setWorks] = useState<Work[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (seriesId) {
      fetchSeries();
    }
  }, [seriesId]);

  const fetchSeries = async () => {
    try {
      setLoading(true);
      
      // Fetch series data and works in parallel
      const [seriesData, worksData] = await Promise.all([
        getSeries(seriesId),
        getSeriesWorks(seriesId)
      ]);
      
      setSeries(seriesData.series);
      setWorks(worksData.works || []);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load series');
    } finally {
      setLoading(false);
    }
  };

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleDateString('en-US', {
      year: 'numeric',
      month: 'long',
      day: 'numeric',
    });
  };

  if (loading) {
    return (
      <div className="max-w-6xl mx-auto px-4 py-8">
        <div className="animate-pulse">
          <div className="h-8 bg-slate-200 rounded mb-4"></div>
          <div className="h-4 bg-slate-200 rounded mb-2"></div>
          <div className="h-4 bg-slate-200 rounded mb-8"></div>
          <div className="space-y-4">
            {[...Array(3)].map((_, i) => (
              <div key={i} className="h-32 bg-slate-200 rounded"></div>
            ))}
          </div>
        </div>
      </div>
    );
  }

  if (error || !series) {
    return (
      <div className="max-w-6xl mx-auto px-4 py-8">
        <div className="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded">
          {error || 'Series not found'}
        </div>
      </div>
    );
  }

  return (
    <div className="max-w-6xl mx-auto px-4 py-8">
      {/* Series Header */}
      <div className="bg-white rounded-lg shadow-sm border border-slate-200 p-6 mb-6">
        <div className="flex justify-between items-start mb-4">
          <div className="flex-1">
            <h1 className="text-3xl font-bold text-slate-900 mb-2">{series.title}</h1>
            <div className="flex items-center gap-2 text-sm text-slate-600 mb-4">
              <span>by</span>
              <Link 
                href={`/users/${series.user_id}`}
                className="text-orange-600 hover:text-orange-700 font-medium"
              >
                {series.username || 'Unknown Author'}
              </Link>
            </div>
          </div>
          {/* Only show edit button if user owns the series */}
          {typeof window !== 'undefined' && localStorage.getItem('user_id') === series.user_id && (
            <Link
              href={`/series/${series.id}/edit`}
              className="px-4 py-2 bg-orange-600 text-white text-sm rounded-lg hover:bg-orange-700 transition-colors"
            >
              Edit Series
            </Link>
          )}
        </div>

        {/* Series Statistics */}
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-6">
          <div className="text-center">
            <div className="text-2xl font-bold text-blue-600">{series.work_count}</div>
            <div className="text-sm text-slate-600">Works</div>
          </div>
          <div className="text-center">
            <div className="text-2xl font-bold text-green-600">
              {series.word_count?.toLocaleString() || '0'}
            </div>
            <div className="text-sm text-slate-600">Words</div>
          </div>
          <div className="text-center">
            <div className={`text-2xl font-bold ${series.is_complete ? 'text-green-600' : 'text-yellow-600'}`}>
              {series.is_complete ? 'Complete' : 'In Progress'}
            </div>
            <div className="text-sm text-slate-600">Status</div>
          </div>
          <div className="text-center">
            <div className="text-2xl font-bold text-purple-600">
              {formatDate(series.updated_at)}
            </div>
            <div className="text-sm text-slate-600">Updated</div>
          </div>
        </div>

        {/* Series Summary */}
        {series.summary && (
          <div className="border-t border-slate-200 pt-4">
            <h3 className="text-sm font-medium text-slate-700 mb-2">Series Summary</h3>
            <div className="prose prose-sm text-slate-600">
              {series.summary.split('\n').map((line, index) => (
                <p key={index}>{line}</p>
              ))}
            </div>
          </div>
        )}

        {/* Series Notes */}
        {series.notes && (
          <div className="border-t border-slate-200 pt-4 mt-4">
            <h3 className="text-sm font-medium text-slate-700 mb-2">Series Notes</h3>
            <div className="prose prose-sm text-slate-600">
              {series.notes.split('\n').map((line, index) => (
                <p key={index}>{line}</p>
              ))}
            </div>
          </div>
        )}
      </div>

      {/* Works in Series */}
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <h2 className="text-2xl font-bold text-slate-900">
            Works in This Series
          </h2>
          <div className="text-sm text-slate-600">
            {works.length} work{works.length !== 1 ? 's' : ''}
          </div>
        </div>

        {works.length === 0 ? (
          <div className="text-center py-16">
            <div className="text-slate-400 mb-4">
              <svg className="w-16 h-16 mx-auto" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1} d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
              </svg>
            </div>
            <p className="text-slate-600">This series doesn&apos;t have any published works yet.</p>
          </div>
        ) : (
          <div className="space-y-6">
            {works.map((work, index) => (
              <div key={work.id} className="bg-white border border-slate-200 rounded-lg overflow-hidden">
                <div className="p-4 bg-slate-50 border-b border-slate-200">
                  <div className="flex items-center gap-2">
                    <span className="inline-flex items-center justify-center w-8 h-8 bg-orange-100 text-orange-800 text-sm font-medium rounded-full">
                      {work.position || index + 1}
                    </span>
                    <span className="text-sm text-slate-600">Part {work.position || index + 1} of the series</span>
                  </div>
                </div>
                <div className="p-6">
                  <WorkCard work={work as any} />
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Navigation */}
      <div className="mt-8 text-center">
        <Link 
          href="/series"
          className="inline-flex items-center text-orange-600 hover:text-orange-700 transition-colors"
        >
          <svg className="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10 19l-7-7m0 0l7-7m-7 7h18" />
          </svg>
          Browse All Series
        </Link>
      </div>
    </div>
  );
}