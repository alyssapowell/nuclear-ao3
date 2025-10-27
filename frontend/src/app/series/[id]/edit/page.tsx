'use client';

import { useState, useEffect } from 'react';
import { useRouter, useParams } from 'next/navigation';
import { getSeries, updateSeries, getSeriesWorks, UpdateSeriesRequest, addWorkToSeries, removeWorkFromSeries, getMyWorks } from '@/lib/api';
import AuthGuard from '@/components/auth/AuthGuard';

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

interface Work {
  id: string;
  title: string;
  summary: string;
  status: string;
  series_id?: string;
  position?: number;
}

export default function EditSeriesPage() {
  const router = useRouter();
  const params = useParams();
  const seriesId = params.id as string;
  
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [series, setSeries] = useState<Series | null>(null);
  const [seriesWorks, setSeriesWorks] = useState<Work[]>([]);
  const [availableWorks, setAvailableWorks] = useState<Work[]>([]);
  
  const [formData, setFormData] = useState<UpdateSeriesRequest>({
    title: '',
    summary: '',
    notes: '',
    is_complete: false,
  });

  useEffect(() => {
    if (seriesId) {
      fetchData();
    }
  }, [seriesId]);

  const fetchData = async () => {
    try {
      setLoading(true);
      setError(null);
      
      const authToken = localStorage.getItem('auth_token');
      if (!authToken) {
        router.push('/auth/login');
        return;
      }

      const [seriesResponse, worksResponse, availableWorksResponse] = await Promise.all([
        getSeries(seriesId, authToken),
        getSeriesWorks(seriesId, authToken),
        getMyWorks(authToken)
      ]);

      const seriesData = seriesResponse.series;
      setSeries(seriesData);
      
      // Set form data from series
      setFormData({
        title: seriesData.title,
        summary: seriesData.summary || '',
        notes: seriesData.notes || '',
        is_complete: seriesData.is_complete,
      });

      // Set current works in series
      setSeriesWorks(worksResponse.works || []);
      
      // Filter available works (not in any series, or in this series)
      const available = availableWorksResponse.works?.filter((work: Work) => 
        !work.series_id || work.series_id === seriesId
      ) || [];
      setAvailableWorks(available);
      
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load series');
    } finally {
      setLoading(false);
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    
    if (!formData.title.trim()) {
      setError('Title is required');
      return;
    }

    try {
      setSaving(true);
      setError(null);
      
      const authToken = localStorage.getItem('auth_token');
      if (!authToken) {
        router.push('/auth/login');
        return;
      }

      await updateSeries(seriesId, formData, authToken);
      
      // Redirect back to series page
      router.push(`/series/${seriesId}`);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to update series');
    } finally {
      setSaving(false);
    }
  };

  const handleAddWork = async (workId: string) => {
    try {
      const authToken = localStorage.getItem('auth_token');
      if (!authToken) return;

      await addWorkToSeries(seriesId, workId, seriesWorks.length + 1, authToken);
      
      // Refresh data
      fetchData();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to add work to series');
    }
  };

  const handleRemoveWork = async (workId: string) => {
    try {
      const authToken = localStorage.getItem('auth_token');
      if (!authToken) return;

      await removeWorkFromSeries(seriesId, workId, authToken);
      
      // Refresh data
      fetchData();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to remove work from series');
    }
  };

  if (loading) {
    return (
      <AuthGuard>
        <div className="max-w-4xl mx-auto px-4 py-8">
          <div className="animate-pulse">
            <div className="h-8 bg-gray-200 rounded mb-4"></div>
            <div className="h-32 bg-gray-200 rounded mb-6"></div>
            <div className="h-64 bg-gray-200 rounded"></div>
          </div>
        </div>
      </AuthGuard>
    );
  }

  if (!series) {
    return (
      <AuthGuard>
        <div className="max-w-4xl mx-auto px-4 py-8">
          <div className="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded-lg">
            {error || 'Series not found'}
          </div>
        </div>
      </AuthGuard>
    );
  }

  return (
    <AuthGuard>
      <div className="max-w-4xl mx-auto px-4 py-8">
        <div className="mb-8">
          <h1 className="text-3xl font-bold text-slate-900">Edit Series</h1>
          <p className="text-slate-600 mt-2">Update your series information and manage works</p>
        </div>

        {error && (
          <div className="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded-lg mb-6">
            {error}
          </div>
        )}

        <form onSubmit={handleSubmit} className="space-y-6">
          {/* Series Information */}
          <div className="bg-white border border-slate-200 rounded-lg p-6">
            <h2 className="text-xl font-semibold text-slate-900 mb-4">Series Information</h2>
            
            <div className="space-y-4">
              {/* Title */}
              <div>
                <label htmlFor="title" className="block text-sm font-medium text-slate-700 mb-1">
                  Title <span className="text-red-500">*</span>
                </label>
                <input
                  type="text"
                  id="title"
                  value={formData.title}
                  onChange={(e) => setFormData(prev => ({ ...prev, title: e.target.value }))}
                  className="w-full px-3 py-2 border border-slate-300 rounded-lg focus:ring-2 focus:ring-orange-500 focus:border-orange-500"
                  required
                />
              </div>

              {/* Summary */}
              <div>
                <label htmlFor="summary" className="block text-sm font-medium text-slate-700 mb-1">
                  Summary
                </label>
                <textarea
                  id="summary"
                  value={formData.summary}
                  onChange={(e) => setFormData(prev => ({ ...prev, summary: e.target.value }))}
                  rows={4}
                  className="w-full px-3 py-2 border border-slate-300 rounded-lg focus:ring-2 focus:ring-orange-500 focus:border-orange-500"
                  placeholder="Describe your series..."
                />
              </div>

              {/* Notes */}
              <div>
                <label htmlFor="notes" className="block text-sm font-medium text-slate-700 mb-1">
                  Notes
                </label>
                <textarea
                  id="notes"
                  value={formData.notes}
                  onChange={(e) => setFormData(prev => ({ ...prev, notes: e.target.value }))}
                  rows={3}
                  className="w-full px-3 py-2 border border-slate-300 rounded-lg focus:ring-2 focus:ring-orange-500 focus:border-orange-500"
                  placeholder="Any additional notes..."
                />
              </div>

              {/* Complete Status */}
              <div className="flex items-center">
                <input
                  type="checkbox"
                  id="is_complete"
                  checked={formData.is_complete}
                  onChange={(e) => setFormData(prev => ({ ...prev, is_complete: e.target.checked }))}
                  className="w-4 h-4 text-orange-600 border-slate-300 rounded focus:ring-orange-500"
                />
                <label htmlFor="is_complete" className="ml-2 text-sm text-slate-700">
                  This series is complete
                </label>
              </div>
            </div>
          </div>

          {/* Works in Series */}
          <div className="bg-white border border-slate-200 rounded-lg p-6">
            <h2 className="text-xl font-semibold text-slate-900 mb-4">Works in Series</h2>
            
            {seriesWorks.length === 0 ? (
              <p className="text-slate-600 mb-4">No works in this series yet.</p>
            ) : (
              <div className="space-y-3 mb-6">
                {seriesWorks.map((work, index) => (
                  <div key={work.id} className="flex items-center justify-between p-3 border border-slate-200 rounded-lg">
                    <div className="flex items-center space-x-3">
                      <span className="w-8 h-8 bg-orange-100 text-orange-800 text-sm font-medium rounded-full flex items-center justify-center">
                        {index + 1}
                      </span>
                      <div>
                        <div className="font-medium text-slate-900">{work.title}</div>
                        <div className="text-sm text-slate-600">Status: {work.status}</div>
                      </div>
                    </div>
                    <button
                      type="button"
                      onClick={() => handleRemoveWork(work.id)}
                      className="text-red-600 hover:text-red-700 text-sm font-medium"
                    >
                      Remove
                    </button>
                  </div>
                ))}
              </div>
            )}

            {/* Add Works */}
            <div>
              <h3 className="text-lg font-medium text-slate-900 mb-3">Add Works</h3>
              {availableWorks.filter(work => !seriesWorks.find(sw => sw.id === work.id)).length === 0 ? (
                <p className="text-slate-600">No additional works available to add.</p>
              ) : (
                <div className="space-y-2">
                  {availableWorks
                    .filter(work => !seriesWorks.find(sw => sw.id === work.id))
                    .map((work) => (
                    <div key={work.id} className="flex items-center justify-between p-3 border border-slate-200 rounded-lg">
                      <div>
                        <div className="font-medium text-slate-900">{work.title}</div>
                        <div className="text-sm text-slate-600">Status: {work.status}</div>
                      </div>
                      <button
                        type="button"
                        onClick={() => handleAddWork(work.id)}
                        className="text-orange-600 hover:text-orange-700 text-sm font-medium"
                      >
                        Add to Series
                      </button>
                    </div>
                  ))}
                </div>
              )}
            </div>
          </div>

          {/* Form Actions */}
          <div className="flex items-center justify-between pt-6 border-t border-slate-200">
            <button
              type="button"
              onClick={() => router.push(`/series/${seriesId}`)}
              className="px-4 py-2 text-slate-600 hover:text-slate-900 transition-colors"
            >
              Cancel
            </button>
            
            <button
              type="submit"
              disabled={saving || !formData.title.trim()}
              className="px-6 py-2 bg-orange-600 text-white rounded-lg hover:bg-orange-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
            >
              {saving ? 'Saving...' : 'Save Changes'}
            </button>
          </div>
        </form>
      </div>
    </AuthGuard>
  );
}