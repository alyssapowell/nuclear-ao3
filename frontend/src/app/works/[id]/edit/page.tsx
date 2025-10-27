'use client';

import { useState, useEffect } from 'react';
import { useParams, useRouter } from 'next/navigation';
import { getWork, updateWork, getWorkChapters, createChapter, updateChapter, deleteChapter, type UpdateWorkRequest, type Chapter, type CreateChapterRequest, type UpdateChapterRequest } from '@/lib/api';
import Link from 'next/link';
import AuthGuard from '@/components/auth/AuthGuard';

interface WorkData {
  work: {
    id: string;
    title: string;
    summary: string;
    notes?: string;
    language: string;
    rating: string;
    category?: string[];
    warnings?: string[];
    fandoms?: string[];
    characters?: string[];
    relationships?: string[];
    freeform_tags?: string[];
    max_chapters?: number;
    is_complete: boolean;
    status: string;
    word_count: number;
    chapter_count: number;
    // Privacy settings
    restricted_to_users?: boolean;
    restricted_to_adults?: boolean;
    comment_policy?: string;
    moderate_comments?: boolean;
    disable_comments?: boolean;
    is_anonymous?: boolean;
    in_anon_collection?: boolean;
    in_unrevealed_collection?: boolean;
    published_at?: string;
    updated_at: string;
    created_at: string;
  };
  authors?: Array<{
    pseud_id: string;
    pseud_name: string;
    user_id: string;
    username: string;
    is_anonymous: boolean;
  }>;
}

export default function EditWorkPage() {
  const params = useParams();
  const router = useRouter();
  const workId = params.id as string;
  
  const [workData, setWorkData] = useState<WorkData | null>(null);
  const [chapters, setChapters] = useState<Chapter[]>([]);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [activeTab, setActiveTab] = useState<'work' | 'chapters'>('work');
  const [selectedChapter, setSelectedChapter] = useState<Chapter | null>(null);
  const [isCreatingChapter, setIsCreatingChapter] = useState(false);
  
  // Form state for work metadata
  const [workForm, setWorkForm] = useState<UpdateWorkRequest>({});
  
  // Form state for chapter editing
  const [chapterForm, setChapterForm] = useState<UpdateChapterRequest>({
    title: '',
    summary: '',
    notes: '',
    end_notes: '',
    content: '',
    status: 'draft',
  });

  // Form state for new chapter
  const [newChapterForm, setNewChapterForm] = useState<CreateChapterRequest>({
    title: '',
    summary: '',
    notes: '',
    end_notes: '',
    content: '',
  });

  useEffect(() => {
    if (typeof window === 'undefined' || !workId) return;

    const fetchData = async () => {
      try {
        const [workResponse, chaptersResponse] = await Promise.all([
          getWork(workId),
          getWorkChapters(workId)
        ]);
        
        setWorkData(workResponse);
        setChapters(chaptersResponse.chapters || []);
        
        // Initialize work form with current data
        const work = workResponse.work;
        setWorkForm({
          title: work.title,
          summary: work.summary,
          notes: work.notes,
          rating: work.rating,
          category: work.category,
          warnings: work.warnings,
          fandoms: work.fandoms,
          characters: work.characters,
          relationships: work.relationships,
          freeform_tags: work.freeform_tags,
          max_chapters: work.max_chapters,
          is_complete: work.is_complete,
          status: work.status,
          restricted_to_users: work.restricted_to_users,
          restricted_to_adults: work.restricted_to_adults,
          comment_policy: work.comment_policy as 'open' | 'users_only' | 'disabled',
          moderate_comments: work.moderate_comments,
          is_anonymous: work.is_anonymous,
          in_anon_collection: work.in_anon_collection,
          in_unrevealed_collection: work.in_unrevealed_collection,
        });
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Unknown error');
      } finally {
        setLoading(false);
      }
    };

    fetchData();
  }, [workId]);

  const handleWorkInputChange = (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement>) => {
    const { name, value, type } = e.target;
    
    if (type === 'checkbox') {
      const checked = (e.target as HTMLInputElement).checked;
      setWorkForm(prev => ({ ...prev, [name]: checked }));
    } else if (type === 'number') {
      const numValue = value === '' ? undefined : parseInt(value);
      setWorkForm(prev => ({ ...prev, [name]: numValue }));
    } else {
      setWorkForm(prev => ({ ...prev, [name]: value }));
    }
  };

  const handleArrayInputChange = (fieldName: keyof UpdateWorkRequest, value: string) => {
    const tags = value.split(',').map(tag => tag.trim()).filter(tag => tag.length > 0);
    setWorkForm(prev => ({ ...prev, [fieldName]: tags }));
  };

  const handleCategoryChange = (category: string, checked: boolean) => {
    setWorkForm(prev => ({
      ...prev,
      category: checked 
        ? [...(prev.category || []), category]
        : (prev.category || []).filter(c => c !== category)
    }));
  };

  const handleWarningChange = (warning: string, checked: boolean) => {
    setWorkForm(prev => ({
      ...prev,
      warnings: checked 
        ? [...(prev.warnings || []), warning]
        : (prev.warnings || []).filter(w => w !== warning)
    }));
  };

  const handleSaveWork = async () => {
    setSaving(true);
    setError(null);

    try {
      await updateWork(workId, workForm);
      
      // Refresh work data
      const updatedWork = await getWork(workId);
      setWorkData(updatedWork);
      
      // Show success message or redirect
      alert('Work updated successfully!');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
    } finally {
      setSaving(false);
    }
  };

  const handleChapterInputChange = (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement>) => {
    const { name, value } = e.target;
    setChapterForm(prev => ({ ...prev, [name]: value }));
  };

  const handleNewChapterInputChange = (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => {
    const { name, value } = e.target;
    setNewChapterForm(prev => ({ ...prev, [name]: value }));
  };

  const handleSelectChapter = (chapter: Chapter) => {
    setSelectedChapter(chapter);
    setChapterForm({
      title: chapter.title || '',
      summary: chapter.summary || '',
      notes: chapter.notes || '',
      end_notes: chapter.end_notes || '',
      content: chapter.content,
      status: chapter.status,
    });
    setIsCreatingChapter(false);
  };

  const handleSaveChapter = async () => {
    if (!selectedChapter) return;
    
    setSaving(true);
    setError(null);

    try {
      await updateChapter(workId, selectedChapter.id, chapterForm);
      
      // Refresh chapters
      const chaptersResponse = await getWorkChapters(workId);
      setChapters(chaptersResponse.chapters || []);
      
      // Update selected chapter
      const updatedChapter = chaptersResponse.chapters?.find((c: Chapter) => c.id === selectedChapter.id);
      if (updatedChapter) {
        setSelectedChapter(updatedChapter);
      }
      
      alert('Chapter updated successfully!');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
    } finally {
      setSaving(false);
    }
  };

  const handleCreateChapter = async () => {
    setSaving(true);
    setError(null);

    try {
      await createChapter(workId, newChapterForm);
      
      // Refresh chapters
      const chaptersResponse = await getWorkChapters(workId);
      setChapters(chaptersResponse.chapters || []);
      
      // Clear form and exit creation mode
      setNewChapterForm({
        title: '',
        summary: '',
        notes: '',
        end_notes: '',
        content: '',
      });
      setIsCreatingChapter(false);
      
      alert('Chapter created successfully!');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
    } finally {
      setSaving(false);
    }
  };

  const handleDeleteChapter = async (chapter: Chapter) => {
    if (!confirm(`Are you sure you want to delete "${chapter.title || `Chapter ${chapter.number}`}"? This cannot be undone.`)) {
      return;
    }

    setSaving(true);
    setError(null);

    try {
      await deleteChapter(workId, chapter.id);
      
      // Refresh chapters
      const chaptersResponse = await getWorkChapters(workId);
      setChapters(chaptersResponse.chapters || []);
      
      // Clear selection if deleted chapter was selected
      if (selectedChapter?.id === chapter.id) {
        setSelectedChapter(null);
      }
      
      alert('Chapter deleted successfully!');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
    } finally {
      setSaving(false);
    }
  };

  if (loading) {
    return (
      <div className="page-container py-12">
        <div className="flex items-center justify-center">
          <div className="loading-spinner mr-3"></div>
          <span>Loading work...</span>
        </div>
      </div>
    );
  }

  if (error && !workData) {
    return (
      <div className="page-container py-12">
        <div className="error-message">
          Error: {error}
        </div>
      </div>
    );
  }

  if (!workData) {
    return (
      <div className="page-container py-12">
        <div className="text-center">
          <p className="text-accent-600">Work not found</p>
        </div>
      </div>
    );
  }

  const categories = ['F/M', 'M/M', 'F/F', 'Gen', 'Multi', 'Other'];
  const warnings = [
    'Creator Chose Not To Use Archive Warnings',
    'Graphic Depictions Of Violence', 
    'Major Character Death',
    'No Archive Warnings Apply',
    'Rape/Non-Con',
    'Underage'
  ];
  const ratings = ['General Audiences', 'Teen And Up Audiences', 'Mature', 'Explicit'];

  return (
    <AuthGuard>
    <div className="max-w-6xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
      <div className="mb-8">
        <Link 
          href={`/works/${workId}`}
          className="inline-flex items-center text-orange-600 hover:text-orange-700 mb-6 font-medium"
        >
          <svg className="w-4 h-4 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 19l-7-7 7-7" />
          </svg>
          Back to Work
        </Link>
        
        <h1 className="text-3xl font-bold text-slate-900">Edit Work</h1>
        <p className="text-slate-600 mt-2">{workData.work.title}</p>
      </div>

      {error && (
        <div className="error-message mb-6">
          Error: {error}
        </div>
      )}

      {/* Tab Navigation */}
      <div className="border-b border-slate-200 mb-8">
        <nav className="-mb-px flex space-x-8">
          <button
            onClick={() => setActiveTab('work')}
            className={`py-2 px-1 border-b-2 font-medium text-sm ${
              activeTab === 'work'
                ? 'border-orange-500 text-orange-600'
                : 'border-transparent text-slate-500 hover:text-slate-700 hover:border-slate-300'
            }`}
          >
            Work Details
          </button>
          <button
            onClick={() => setActiveTab('chapters')}
            className={`py-2 px-1 border-b-2 font-medium text-sm ${
              activeTab === 'chapters'
                ? 'border-orange-500 text-orange-600'
                : 'border-transparent text-slate-500 hover:text-slate-700 hover:border-slate-300'
            }`}
          >
            Chapters ({chapters.length})
          </button>
        </nav>
      </div>

      {/* Work Details Tab */}
      {activeTab === 'work' && (
        <div className="space-y-8">
          {/* Basic Information */}
          <div className="card">
            <div className="card-header">
              <h2 className="text-xl font-semibold text-slate-900">Work Information</h2>
            </div>
            <div className="card-body space-y-6">
              <div>
                <label htmlFor="title" className="block text-sm font-medium text-slate-700 mb-2">
                  Title *
                </label>
                <input
                  type="text"
                  id="title"
                  name="title"
                  value={workForm.title || ''}
                  onChange={handleWorkInputChange}
                  required
                  className="w-full px-3 py-2 border border-slate-300 rounded-md focus:outline-none focus:ring-2 focus:ring-orange-500 focus:border-transparent"
                />
              </div>

              <div>
                <label htmlFor="summary" className="block text-sm font-medium text-slate-700 mb-2">
                  Summary
                </label>
                <textarea
                  id="summary"
                  name="summary"
                  value={workForm.summary || ''}
                  onChange={handleWorkInputChange}
                  rows={4}
                  className="w-full px-3 py-2 border border-slate-300 rounded-md focus:outline-none focus:ring-2 focus:ring-orange-500 focus:border-transparent"
                />
              </div>

              <div>
                <label htmlFor="notes" className="block text-sm font-medium text-slate-700 mb-2">
                  Author&apos;s Notes
                </label>
                <textarea
                  id="notes"
                  name="notes"
                  value={workForm.notes || ''}
                  onChange={handleWorkInputChange}
                  rows={3}
                  className="w-full px-3 py-2 border border-slate-300 rounded-md focus:outline-none focus:ring-2 focus:ring-orange-500 focus:border-transparent"
                />
              </div>
            </div>
          </div>

          {/* Tags and Classification */}
          <div className="card">
            <div className="card-header">
              <h2 className="text-xl font-semibold text-slate-900">Tags & Classification</h2>
            </div>
            <div className="card-body space-y-6">
              <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                <div>
                  <label htmlFor="rating" className="block text-sm font-medium text-slate-700 mb-2">
                    Rating *
                  </label>
                  <select
                    id="rating"
                    name="rating"
                    value={workForm.rating || ''}
                    onChange={handleWorkInputChange}
                    required
                    className="w-full px-3 py-2 border border-slate-300 rounded-md focus:outline-none focus:ring-2 focus:ring-orange-500 focus:border-transparent"
                  >
                    {ratings.map(rating => (
                      <option key={rating} value={rating}>{rating}</option>
                    ))}
                  </select>
                </div>

                <div>
                  <label htmlFor="status" className="block text-sm font-medium text-slate-700 mb-2">
                    Status
                  </label>
                  <select
                    id="status"
                    name="status"
                    value={workForm.status || ''}
                    onChange={handleWorkInputChange}
                    className="w-full px-3 py-2 border border-slate-300 rounded-md focus:outline-none focus:ring-2 focus:ring-orange-500 focus:border-transparent"
                  >
                    <option value="draft">Draft</option>
                    <option value="posted">Published</option>
                    <option value="hidden">Hidden</option>
                  </select>
                </div>
              </div>

              <div>
                <label className="block text-sm font-medium text-slate-700 mb-2">
                  Category
                </label>
                <div className="grid grid-cols-2 md:grid-cols-3 gap-3">
                  {categories.map(category => (
                    <label key={category} className="flex items-center">
                      <input
                        type="checkbox"
                        checked={workForm.category?.includes(category) || false}
                        onChange={(e) => handleCategoryChange(category, e.target.checked)}
                        className="mr-2 h-4 w-4 text-orange-600 focus:ring-orange-500 border-slate-300 rounded"
                      />
                      <span className="text-sm text-slate-700">{category}</span>
                    </label>
                  ))}
                </div>
              </div>

              <div>
                <label className="block text-sm font-medium text-slate-700 mb-2">
                  Archive Warnings
                </label>
                <div className="space-y-2">
                  {warnings.map(warning => (
                    <label key={warning} className="flex items-center">
                      <input
                        type="checkbox"
                        checked={workForm.warnings?.includes(warning) || false}
                        onChange={(e) => handleWarningChange(warning, e.target.checked)}
                        className="mr-2 h-4 w-4 text-orange-600 focus:ring-orange-500 border-slate-300 rounded"
                      />
                      <span className="text-sm text-slate-700">{warning}</span>
                    </label>
                  ))}
                </div>
              </div>

              <div>
                <label htmlFor="fandoms" className="block text-sm font-medium text-slate-700 mb-2">
                  Fandoms * (separate with commas)
                </label>
                <input
                  type="text"
                  id="fandoms"
                  value={workForm.fandoms?.join(', ') || ''}
                  onChange={(e) => handleArrayInputChange('fandoms', e.target.value)}
                  required
                  className="w-full px-3 py-2 border border-slate-300 rounded-md focus:outline-none focus:ring-2 focus:ring-orange-500 focus:border-transparent"
                />
              </div>

              <div>
                <label htmlFor="characters" className="block text-sm font-medium text-slate-700 mb-2">
                  Characters (separate with commas)
                </label>
                <input
                  type="text"
                  id="characters"
                  value={workForm.characters?.join(', ') || ''}
                  onChange={(e) => handleArrayInputChange('characters', e.target.value)}
                  className="w-full px-3 py-2 border border-slate-300 rounded-md focus:outline-none focus:ring-2 focus:ring-orange-500 focus:border-transparent"
                />
              </div>

              <div>
                <label htmlFor="relationships" className="block text-sm font-medium text-slate-700 mb-2">
                  Relationships (separate with commas)
                </label>
                <input
                  type="text"
                  id="relationships"
                  value={workForm.relationships?.join(', ') || ''}
                  onChange={(e) => handleArrayInputChange('relationships', e.target.value)}
                  className="w-full px-3 py-2 border border-slate-300 rounded-md focus:outline-none focus:ring-2 focus:ring-orange-500 focus:border-transparent"
                />
              </div>

              <div>
                <label htmlFor="freeform_tags" className="block text-sm font-medium text-slate-700 mb-2">
                  Additional Tags (separate with commas)
                </label>
                <input
                  type="text"
                  id="freeform_tags"
                  value={workForm.freeform_tags?.join(', ') || ''}
                  onChange={(e) => handleArrayInputChange('freeform_tags', e.target.value)}
                  className="w-full px-3 py-2 border border-slate-300 rounded-md focus:outline-none focus:ring-2 focus:ring-orange-500 focus:border-transparent"
                />
              </div>

              <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                <div>
                  <label htmlFor="max_chapters" className="block text-sm font-medium text-slate-700 mb-2">
                    Planned Chapters
                  </label>
                  <input
                    type="number"
                    id="max_chapters"
                    name="max_chapters"
                    value={workForm.max_chapters || ''}
                    onChange={handleWorkInputChange}
                    min="1"
                    className="w-full px-3 py-2 border border-slate-300 rounded-md focus:outline-none focus:ring-2 focus:ring-orange-500 focus:border-transparent"
                  />
                </div>

                <div className="flex items-center">
                  <label className="flex items-center">
                    <input
                      type="checkbox"
                      name="is_complete"
                      checked={workForm.is_complete || false}
                      onChange={handleWorkInputChange}
                      className="mr-2 h-4 w-4 text-orange-600 focus:ring-orange-500 border-slate-300 rounded"
                    />
                    <span className="text-sm font-medium text-slate-700">Mark as Complete</span>
                  </label>
                </div>
              </div>
            </div>
          </div>

          {/* Privacy Settings */}
          <div className="card">
            <div className="card-header">
              <h2 className="text-xl font-semibold text-slate-900">Privacy Settings</h2>
            </div>
            <div className="card-body space-y-6">
              <div className="space-y-4">
                <label className="flex items-center">
                  <input
                    type="checkbox"
                    name="restricted_to_users"
                    checked={workForm.restricted_to_users || false}
                    onChange={handleWorkInputChange}
                    className="mr-3 h-4 w-4 text-orange-600 focus:ring-orange-500 border-slate-300 rounded"
                  />
                  <div>
                    <span className="text-sm font-medium text-slate-700">Registered Users Only</span>
                    <p className="text-sm text-slate-500">Only registered users can view this work</p>
                  </div>
                </label>

                <label className="flex items-center">
                  <input
                    type="checkbox"
                    name="restricted_to_adults"
                    checked={workForm.restricted_to_adults || false}
                    onChange={handleWorkInputChange}
                    className="mr-3 h-4 w-4 text-orange-600 focus:ring-orange-500 border-slate-300 rounded"
                  />
                  <div>
                    <span className="text-sm font-medium text-slate-700">Adults Only</span>
                    <p className="text-sm text-slate-500">Only users 18+ can view this work</p>
                  </div>
                </label>

                <label className="flex items-center">
                  <input
                    type="checkbox"
                    name="is_anonymous"
                    checked={workForm.is_anonymous || false}
                    onChange={handleWorkInputChange}
                    className="mr-3 h-4 w-4 text-orange-600 focus:ring-orange-500 border-slate-300 rounded"
                  />
                  <div>
                    <span className="text-sm font-medium text-slate-700">Post Anonymously</span>
                    <p className="text-sm text-slate-500">Hide your username from readers</p>
                  </div>
                </label>
              </div>

              <div>
                <label htmlFor="comment_policy" className="block text-sm font-medium text-slate-700 mb-2">
                  Comment Policy
                </label>
                <select
                  id="comment_policy"
                  name="comment_policy"
                  value={workForm.comment_policy || ''}
                  onChange={handleWorkInputChange}
                  className="w-full px-3 py-2 border border-slate-300 rounded-md focus:outline-none focus:ring-2 focus:ring-orange-500 focus:border-transparent"
                >
                  <option value="open">Anyone can comment</option>
                  <option value="users_only">Only registered users can comment</option>
                  <option value="disabled">Comments disabled</option>
                </select>
              </div>

              <div className="space-y-4">
                <label className="flex items-center">
                  <input
                    type="checkbox"
                    name="moderate_comments"
                    checked={workForm.moderate_comments || false}
                    onChange={handleWorkInputChange}
                    className="mr-3 h-4 w-4 text-orange-600 focus:ring-orange-500 border-slate-300 rounded"
                  />
                  <div>
                    <span className="text-sm font-medium text-slate-700">Moderate Comments</span>
                    <p className="text-sm text-slate-500">Review comments before they appear</p>
                  </div>
                </label>
              </div>
            </div>
          </div>

          {/* Save Button */}
          <div className="flex items-center justify-end gap-4">
            <Link 
              href={`/works/${workId}`}
              className="btn btn-outline"
            >
              Cancel
            </Link>
            <button
              onClick={handleSaveWork}
              disabled={saving}
              className="btn btn-primary disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {saving ? (
                <>
                  <div className="loading-spinner mr-2"></div>
                  Saving...
                </>
              ) : (
                'Save Changes'
              )}
            </button>
          </div>
        </div>
      )}

      {/* Chapters Tab */}
      {activeTab === 'chapters' && (
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
          {/* Chapter List */}
          <div className="lg:col-span-1">
            <div className="card">
              <div className="card-header">
                <div className="flex items-center justify-between">
                  <h2 className="text-lg font-semibold text-slate-900">Chapters</h2>
                  <button
                    onClick={() => {
                      setIsCreatingChapter(true);
                      setSelectedChapter(null);
                    }}
                    className="btn btn-primary btn-sm"
                  >
                    <svg className="w-4 h-4 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
                    </svg>
                    Add Chapter
                  </button>
                </div>
              </div>
              <div className="card-body p-0">
                {chapters.length === 0 ? (
                  <div className="p-6 text-center text-slate-500">
                    <p>No chapters yet.</p>
                    <button
                      onClick={() => {
                        setIsCreatingChapter(true);
                        setSelectedChapter(null);
                      }}
                      className="text-orange-600 hover:text-orange-700 text-sm font-medium mt-2"
                    >
                      Create your first chapter
                    </button>
                  </div>
                ) : (
                  <div className="divide-y divide-slate-200">
                    {chapters.map((chapter) => (
                      <div
                        key={chapter.id}
                        className={`p-4 cursor-pointer hover:bg-slate-50 transition-colors ${
                          selectedChapter?.id === chapter.id ? 'bg-orange-50 border-r-2 border-orange-500' : ''
                        }`}
                        onClick={() => handleSelectChapter(chapter)}
                      >
                        <div className="flex items-start justify-between">
                          <div className="flex-1">
                            <h3 className="font-medium text-slate-900">
                              {chapter.title || `Chapter ${chapter.number}`}
                            </h3>
                            <p className="text-sm text-slate-500 mt-1">
                              {chapter.word_count.toLocaleString()} words
                            </p>
                            <div className="flex items-center gap-2 mt-2">
                              <span className={`text-xs px-2 py-1 rounded ${
                                chapter.status === 'posted' ? 'bg-green-100 text-green-800' : 'bg-yellow-100 text-yellow-800'
                              }`}>
                                {chapter.status === 'posted' ? 'Published' : 'Draft'}
                              </span>
                            </div>
                          </div>
                          <button
                            onClick={(e) => {
                              e.stopPropagation();
                              handleDeleteChapter(chapter);
                            }}
                            className="text-slate-400 hover:text-red-600 transition-colors"
                            title="Delete chapter"
                          >
                            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                            </svg>
                          </button>
                        </div>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            </div>
          </div>

          {/* Chapter Editor */}
          <div className="lg:col-span-2">
            {isCreatingChapter ? (
              <div className="card">
                <div className="card-header">
                  <h2 className="text-lg font-semibold text-slate-900">Create New Chapter</h2>
                </div>
                <div className="card-body space-y-6">
                  <div>
                    <label htmlFor="new_chapter_title" className="block text-sm font-medium text-slate-700 mb-2">
                      Chapter Title
                    </label>
                    <input
                      type="text"
                      id="new_chapter_title"
                      name="title"
                      value={newChapterForm.title}
                      onChange={handleNewChapterInputChange}
                      className="w-full px-3 py-2 border border-slate-300 rounded-md focus:outline-none focus:ring-2 focus:ring-orange-500 focus:border-transparent"
                      placeholder={`Chapter ${chapters.length + 1}`}
                    />
                  </div>

                  <div>
                    <label htmlFor="new_chapter_summary" className="block text-sm font-medium text-slate-700 mb-2">
                      Chapter Summary
                    </label>
                    <textarea
                      id="new_chapter_summary"
                      name="summary"
                      value={newChapterForm.summary}
                      onChange={handleNewChapterInputChange}
                      rows={2}
                      className="w-full px-3 py-2 border border-slate-300 rounded-md focus:outline-none focus:ring-2 focus:ring-orange-500 focus:border-transparent"
                    />
                  </div>

                  <div>
                    <label htmlFor="new_chapter_notes" className="block text-sm font-medium text-slate-700 mb-2">
                      Chapter Notes
                    </label>
                    <textarea
                      id="new_chapter_notes"
                      name="notes"
                      value={newChapterForm.notes}
                      onChange={handleNewChapterInputChange}
                      rows={2}
                      className="w-full px-3 py-2 border border-slate-300 rounded-md focus:outline-none focus:ring-2 focus:ring-orange-500 focus:border-transparent"
                    />
                  </div>

                  <div>
                    <label htmlFor="new_chapter_content" className="block text-sm font-medium text-slate-700 mb-2">
                      Chapter Content *
                    </label>
                    <textarea
                      id="new_chapter_content"
                      name="content"
                      value={newChapterForm.content}
                      onChange={handleNewChapterInputChange}
                      rows={20}
                      required
                      className="w-full px-3 py-2 border border-slate-300 rounded-md focus:outline-none focus:ring-2 focus:ring-orange-500 focus:border-transparent font-mono"
                      placeholder="Write your chapter content here..."
                    />
                    <p className="text-sm text-slate-500 mt-1">
                      Word count: {newChapterForm.content.split(/\s+/).filter(word => word.length > 0).length}
                    </p>
                  </div>

                  <div>
                    <label htmlFor="new_chapter_end_notes" className="block text-sm font-medium text-slate-700 mb-2">
                      End Notes
                    </label>
                    <textarea
                      id="new_chapter_end_notes"
                      name="end_notes"
                      value={newChapterForm.end_notes}
                      onChange={handleNewChapterInputChange}
                      rows={2}
                      className="w-full px-3 py-2 border border-slate-300 rounded-md focus:outline-none focus:ring-2 focus:ring-orange-500 focus:border-transparent"
                    />
                  </div>

                  <div className="flex items-center justify-end gap-4">
                    <button
                      onClick={() => setIsCreatingChapter(false)}
                      className="btn btn-outline"
                    >
                      Cancel
                    </button>
                    <button
                      onClick={handleCreateChapter}
                      disabled={saving || !newChapterForm.content.trim()}
                      className="btn btn-primary disabled:opacity-50 disabled:cursor-not-allowed"
                    >
                      {saving ? (
                        <>
                          <div className="loading-spinner mr-2"></div>
                          Creating...
                        </>
                      ) : (
                        'Create Chapter'
                      )}
                    </button>
                  </div>
                </div>
              </div>
            ) : selectedChapter ? (
              <div className="card">
                <div className="card-header">
                  <h2 className="text-lg font-semibold text-slate-900">
                    Edit: {selectedChapter.title || `Chapter ${selectedChapter.number}`}
                  </h2>
                </div>
                <div className="card-body space-y-6">
                  <div>
                    <label htmlFor="chapter_title" className="block text-sm font-medium text-slate-700 mb-2">
                      Chapter Title
                    </label>
                    <input
                      type="text"
                      id="chapter_title"
                      name="title"
                      value={chapterForm.title}
                      onChange={handleChapterInputChange}
                      className="w-full px-3 py-2 border border-slate-300 rounded-md focus:outline-none focus:ring-2 focus:ring-orange-500 focus:border-transparent"
                    />
                  </div>

                  <div>
                    <label htmlFor="chapter_summary" className="block text-sm font-medium text-slate-700 mb-2">
                      Chapter Summary
                    </label>
                    <textarea
                      id="chapter_summary"
                      name="summary"
                      value={chapterForm.summary}
                      onChange={handleChapterInputChange}
                      rows={2}
                      className="w-full px-3 py-2 border border-slate-300 rounded-md focus:outline-none focus:ring-2 focus:ring-orange-500 focus:border-transparent"
                    />
                  </div>

                  <div>
                    <label htmlFor="chapter_notes" className="block text-sm font-medium text-slate-700 mb-2">
                      Chapter Notes
                    </label>
                    <textarea
                      id="chapter_notes"
                      name="notes"
                      value={chapterForm.notes}
                      onChange={handleChapterInputChange}
                      rows={2}
                      className="w-full px-3 py-2 border border-slate-300 rounded-md focus:outline-none focus:ring-2 focus:ring-orange-500 focus:border-transparent"
                    />
                  </div>

                  <div>
                    <label htmlFor="chapter_status" className="block text-sm font-medium text-slate-700 mb-2">
                      Status
                    </label>
                    <select
                      id="chapter_status"
                      name="status"
                      value={chapterForm.status}
                      onChange={handleChapterInputChange}
                      className="w-full px-3 py-2 border border-slate-300 rounded-md focus:outline-none focus:ring-2 focus:ring-orange-500 focus:border-transparent"
                    >
                      <option value="draft">Draft</option>
                      <option value="posted">Published</option>
                    </select>
                  </div>

                  <div>
                    <label htmlFor="chapter_content" className="block text-sm font-medium text-slate-700 mb-2">
                      Chapter Content *
                    </label>
                    <textarea
                      id="chapter_content"
                      name="content"
                      value={chapterForm.content}
                      onChange={handleChapterInputChange}
                      rows={20}
                      required
                      className="w-full px-3 py-2 border border-slate-300 rounded-md focus:outline-none focus:ring-2 focus:ring-orange-500 focus:border-transparent font-mono"
                    />
                    <p className="text-sm text-slate-500 mt-1">
                      Word count: {chapterForm.content?.split(/\s+/).filter(word => word.length > 0).length || 0}
                    </p>
                  </div>

                  <div>
                    <label htmlFor="chapter_end_notes" className="block text-sm font-medium text-slate-700 mb-2">
                      End Notes
                    </label>
                    <textarea
                      id="chapter_end_notes"
                      name="end_notes"
                      value={chapterForm.end_notes}
                      onChange={handleChapterInputChange}
                      rows={2}
                      className="w-full px-3 py-2 border border-slate-300 rounded-md focus:outline-none focus:ring-2 focus:ring-orange-500 focus:border-transparent"
                    />
                  </div>

                  <div className="flex items-center justify-end gap-4">
                    <button
                      onClick={() => setSelectedChapter(null)}
                      className="btn btn-outline"
                    >
                      Close
                    </button>
                    <button
                      onClick={handleSaveChapter}
                      disabled={saving}
                      className="btn btn-primary disabled:opacity-50 disabled:cursor-not-allowed"
                    >
                      {saving ? (
                        <>
                          <div className="loading-spinner mr-2"></div>
                          Saving...
                        </>
                      ) : (
                        'Save Chapter'
                      )}
                    </button>
                  </div>
                </div>
              </div>
            ) : (
              <div className="card">
                <div className="card-body text-center py-16">
                  <div className="text-slate-400 mb-4">
                    <svg className="w-16 h-16 mx-auto" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1} d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
                    </svg>
                  </div>
                  <p className="text-slate-600 mb-4">Select a chapter to edit or create a new one.</p>
                  <button
                    onClick={() => {
                      setIsCreatingChapter(true);
                      setSelectedChapter(null);
                    }}
                    className="btn btn-primary"
                  >
                    Create New Chapter
                  </button>
                </div>
              </div>
            )}
          </div>
        </div>
      )}
    </div>
    </AuthGuard>
  );
}