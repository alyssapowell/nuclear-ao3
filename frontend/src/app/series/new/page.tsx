'use client';

import { useState, useEffect } from 'react';
import { useRouter } from 'next/navigation';
import { createSeries, getMyWorks, CreateSeriesRequest } from '@/lib/api';
import AuthGuard from '@/components/auth/AuthGuard';

interface Work {
  id: string;
  title: string;
  summary: string;
  status: string;
  series_id?: string;
}

export default function CreateSeriesPage() {
  const router = useRouter();
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [availableWorks, setAvailableWorks] = useState<Work[]>([]);
  const [loadingWorks, setLoadingWorks] = useState(true);
  
  const [formData, setFormData] = useState<CreateSeriesRequest>({
    title: '',
    summary: '',
    notes: '',
    is_complete: false,
    work_ids: [],
  });

  useEffect(() => {
    fetchAvailableWorks();
  }, []);

  const fetchAvailableWorks = async () => {
    try {
      const authToken = localStorage.getItem('auth_token');
      if (!authToken) {
        router.push('/auth/login');
        return;
      }

      const response = await getMyWorks(authToken);
      // Filter works that are not already in a series
      const availableWorks = response.works?.filter((work: Work) => !work.series_id) || [];
      setAvailableWorks(availableWorks);
    } catch (err) {
      console.error('Failed to fetch works:', err);
      setError('Failed to load your works');
    } finally {
      setLoadingWorks(false);
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    
    if (!formData.title.trim()) {
      setError('Title is required');
      return;
    }

    try {
      setLoading(true);
      setError(null);
      
      const authToken = localStorage.getItem('auth_token');
      if (!authToken) {
        router.push('/auth/login');
        return;
      }

      const response = await createSeries(formData, authToken);
      
      // Redirect to the new series page
      router.push(`/series/${response.series.id}`);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create series');
    } finally {
      setLoading(false);
    }
  };

  const handleWorkSelection = (workId: string, selected: boolean) => {
    setFormData(prev => ({
      ...prev,
      work_ids: selected 
        ? [...(prev.work_ids || []), workId]
        : (prev.work_ids || []).filter(id => id !== workId)
    }));
  };

  return (
    <AuthGuard>
    <div className="max-w-4xl mx-auto px-4 py-8">
      <div className="mb-8">
        <h1 className="text-3xl font-bold text-slate-900">Create New Series</h1>
        <p className="text-slate-600 mt-2">Group your related works into a series</p>
      </div>

      {error && (
        <div className="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded-lg mb-6">
          {error}
        </div>
      )}

      <form onSubmit={handleSubmit} className="space-y-6">
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
                placeholder="Enter series title"
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
              <p className="text-sm text-slate-500 mt-1">Optional description of your series</p>
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
                placeholder="Any additional notes about the series..."
              />
              <p className="text-sm text-slate-500 mt-1">Optional author's notes</p>
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

        {/* Works Selection */}
        <div className="bg-white border border-slate-200 rounded-lg p-6">
          <h2 className="text-xl font-semibold text-slate-900 mb-4">Add Works to Series</h2>
          
          {loadingWorks ? (
            <div className="text-center py-8">
              <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-orange-600 mx-auto"></div>
              <p className="text-slate-600 mt-2">Loading your works...</p>
            </div>
          ) : availableWorks.length === 0 ? (
            <div className="text-center py-8">
              <div className="text-slate-400 mb-4">
                <svg className="w-12 h-12 mx-auto" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1} d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
                </svg>
              </div>
              <p className="text-slate-600">No available works to add to series.</p>
              <p className="text-slate-500 text-sm">All your published works are already in series.</p>
            </div>
          ) : (
            <div className="space-y-3">
              <p className="text-sm text-slate-600 mb-4">
                Select works to include in this series (optional - you can add works later):
              </p>
              
              {availableWorks.map((work) => (
                <div key={work.id} className="flex items-start space-x-3 p-3 border border-slate-200 rounded-lg">
                  <input
                    type="checkbox"
                    id={`work-${work.id}`}
                    checked={(formData.work_ids || []).includes(work.id)}
                    onChange={(e) => handleWorkSelection(work.id, e.target.checked)}
                    className="w-4 h-4 text-orange-600 border-slate-300 rounded focus:ring-orange-500 mt-1"
                  />
                  <label htmlFor={`work-${work.id}`} className="flex-1 cursor-pointer">
                    <div className="font-medium text-slate-900">{work.title}</div>
                    {work.summary && (
                      <div className="text-sm text-slate-600 mt-1 line-clamp-2">
                        {work.summary}
                      </div>
                    )}
                    <div className="text-xs text-slate-500 mt-1">
                      Status: {work.status}
                    </div>
                  </label>
                </div>
              ))}
            </div>
          )}
        </div>

        {/* Form Actions */}
        <div className="flex items-center justify-between pt-6 border-t border-slate-200">
          <button
            type="button"
            onClick={() => router.back()}
            className="px-4 py-2 text-slate-600 hover:text-slate-900 transition-colors"
          >
            Cancel
          </button>
          
          <button
            type="submit"
            disabled={loading || !formData.title.trim()}
            className="px-6 py-2 bg-orange-600 text-white rounded-lg hover:bg-orange-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
          >
            {loading ? 'Creating...' : 'Create Series'}
          </button>
        </div>
      </form>
    </div>
    </AuthGuard>
  );
}