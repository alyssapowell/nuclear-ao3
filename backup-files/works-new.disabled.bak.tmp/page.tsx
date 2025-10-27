'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { TagInput } from '@/components/TagInput';
import { EnhancedTagProminenceSelector } from '@/components/EnhancedTagProminenceSelector';
import { createWork, CreateWorkRequest } from '@/lib/api';

export default function NewWorkPage() {
  const [formData, setFormData] = useState({
    title: '',
    summary: '',
    notes: '',
    rating: 'not_rated',
    category: [] as string[],
    warnings: [] as string[],
    fandoms: [] as string[],
    characters: [] as string[],
    relationships: [] as string[],
    freeformTags: [] as string[],
    language: 'en',
    maxChapters: '',
    chapterTitle: '',
    chapterSummary: '',
    chapterNotes: '',
    chapterEndNotes: '',
    chapterContent: '',
    status: 'draft',
    restrictedToUsers: false,
    commentPolicy: 'open',
    moderateComments: false,
    isAnonymous: false,
  });

  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState('');
  const [showAdvanced, setShowAdvanced] = useState(false);
  const router = useRouter();

  const handleChange = (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement>) => {
    const { name, value, type } = e.target;
    if (type === 'checkbox') {
      const checked = (e.target as HTMLInputElement).checked;
      setFormData(prev => ({ ...prev, [name]: checked }));
    } else {
      setFormData(prev => ({ ...prev, [name]: value }));
    }
  };

  const handleTagsChange = (field: string, tags: string[]) => {
    setFormData(prev => ({ ...prev, [field]: tags }));
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsLoading(true);
    setError('');

    try {
      const token = localStorage.getItem('token');
      if (!token) {
        router.push('/auth/login?redirect=/works/new');
        return;
      }

      const payload = {
        title: formData.title,
        summary: formData.summary,
        notes: formData.notes,
        language: formData.language,
        rating: formData.rating,
        category: formData.category,
        warnings: formData.warnings,
        fandoms: formData.fandoms,
        characters: formData.characters,
        relationships: formData.relationships,
        freeform_tags: formData.freeformTags,
        max_chapters: formData.maxChapters ? parseInt(formData.maxChapters) : null,
        chapter_title: formData.chapterTitle,
        chapter_summary: formData.chapterSummary,
        chapter_notes: formData.chapterNotes,
        chapter_end_notes: formData.chapterEndNotes,
        chapter_content: formData.chapterContent,
        status: formData.status,
        restricted_to_users: formData.restrictedToUsers,
        comment_policy: formData.commentPolicy,
        moderate_comments: formData.moderateComments,
        is_anonymous: formData.isAnonymous,
      };

      const data = await createWork(payload as CreateWorkRequest, token);
      router.push(`/works/${data.work.id}`);
    } catch (err) {
      setError('Network error. Please try again.');
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div className="max-w-4xl mx-auto px-4 py-8">
      <div className="bg-white rounded-lg shadow-sm border border-slate-200 p-6">
        <h1 className="text-2xl font-bold text-slate-900 mb-6">Post New Work</h1>
        
        <form onSubmit={handleSubmit} className="space-y-6">
          {/* Basic Work Information */}
          <div className="space-y-4">
            <h2 className="text-lg font-semibold text-slate-800">Work Information</h2>
            
            <div>
              <label htmlFor="title" className="block text-sm font-medium text-slate-700">
                Title *
              </label>
              <input
                type="text"
                id="title"
                name="title"
                required
                className="mt-1 block w-full px-3 py-2 border border-slate-300 rounded-md shadow-sm focus:outline-none focus:ring-orange-500 focus:border-orange-500 sm:text-sm"
                value={formData.title}
                onChange={handleChange}
              />
            </div>

            <div>
              <label htmlFor="fandoms" className="block text-sm font-medium text-slate-700">
                Fandoms *
              </label>
              <TagInput
                value={formData.fandoms}
                onChange={(tags) => handleTagsChange('fandoms', tags)}
                tagType="fandom"
                placeholder="Start typing to search for fandoms..."
                className="mt-1"
              />
              <p className="mt-1 text-xs text-slate-500">
                Type the name of the fandom(s) your work belongs to
              </p>
            </div>

            <div>
              <label htmlFor="summary" className="block text-sm font-medium text-slate-700">
                Summary
              </label>
              <textarea
                id="summary"
                name="summary"
                rows={3}
                className="mt-1 block w-full px-3 py-2 border border-slate-300 rounded-md shadow-sm focus:outline-none focus:ring-orange-500 focus:border-orange-500 sm:text-sm"
                placeholder="Describe what your work is about..."
                value={formData.summary}
                onChange={handleChange}
              />
            </div>

            <div>
              <label htmlFor="notes" className="block text-sm font-medium text-slate-700">
                Notes
              </label>
              <textarea
                id="notes"
                name="notes"
                rows={2}
                className="mt-1 block w-full px-3 py-2 border border-slate-300 rounded-md shadow-sm focus:outline-none focus:ring-orange-500 focus:border-orange-500 sm:text-sm"
                placeholder="Any additional notes for readers..."
                value={formData.notes}
                onChange={handleChange}
              />
            </div>
          </div>

          {/* Rating and Warnings */}
          <div className="space-y-4">
            <h2 className="text-lg font-semibold text-slate-800">Rating & Warnings</h2>
            
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div>
                <label htmlFor="rating" className="block text-sm font-medium text-slate-700">
                  Rating *
                </label>
                <select
                  id="rating"
                  name="rating"
                  required
                  className="mt-1 block w-full px-3 py-2 border border-slate-300 rounded-md shadow-sm focus:outline-none focus:ring-orange-500 focus:border-orange-500 sm:text-sm"
                  value={formData.rating}
                  onChange={handleChange}
                >
                  <option value="not_rated">Not Rated</option>
                  <option value="general">General Audiences</option>
                  <option value="teen">Teen And Up Audiences</option>
                  <option value="mature">Mature</option>
                  <option value="explicit">Explicit</option>
                </select>
              </div>

              <div>
                <label htmlFor="language" className="block text-sm font-medium text-slate-700">
                  Language
                </label>
                <select
                  id="language"
                  name="language"
                  className="mt-1 block w-full px-3 py-2 border border-slate-300 rounded-md shadow-sm focus:outline-none focus:ring-orange-500 focus:border-orange-500 sm:text-sm"
                  value={formData.language}
                  onChange={handleChange}
                >
                  <option value="en">English</option>
                  <option value="es">Español</option>
                  <option value="fr">Français</option>
                  <option value="de">Deutsch</option>
                  <option value="ja">日本語</option>
                  <option value="zh">中文</option>
                </select>
              </div>
            </div>
          </div>

          {/* Tags */}
          <div className="space-y-4">
            <h2 className="text-lg font-semibold text-slate-800">Tags</h2>
            
            <EnhancedTagProminenceSelector
              fandoms={formData.fandoms}
              onTagsChange={(tagType, tags) => {
                if (tagType === 'characters') handleTagsChange('characters', tags);
                else if (tagType === 'relationships') handleTagsChange('relationships', tags);
                else if (tagType === 'freeform') handleTagsChange('freeformTags', tags);
              }}
              initialTags={{
                characters: formData.characters,
                relationships: formData.relationships,
                freeform: formData.freeformTags,
              }}
            />
          </div>

          {/* Chapter Information */}
          <div className="space-y-4">
            <h2 className="text-lg font-semibold text-slate-800">Chapter</h2>
            
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div>
                <label htmlFor="chapterTitle" className="block text-sm font-medium text-slate-700">
                  Chapter Title
                </label>
                <input
                  type="text"
                  id="chapterTitle"
                  name="chapterTitle"
                  className="mt-1 block w-full px-3 py-2 border border-slate-300 rounded-md shadow-sm focus:outline-none focus:ring-orange-500 focus:border-orange-500 sm:text-sm"
                  value={formData.chapterTitle}
                  onChange={handleChange}
                />
              </div>

              <div>
                <label htmlFor="maxChapters" className="block text-sm font-medium text-slate-700">
                  Total Chapters
                </label>
                <input
                  type="number"
                  id="maxChapters"
                  name="maxChapters"
                  min="1"
                  className="mt-1 block w-full px-3 py-2 border border-slate-300 rounded-md shadow-sm focus:outline-none focus:ring-orange-500 focus:border-orange-500 sm:text-sm"
                  placeholder="Leave blank if unknown"
                  value={formData.maxChapters}
                  onChange={handleChange}
                />
              </div>
            </div>

            <div>
              <label htmlFor="chapterContent" className="block text-sm font-medium text-slate-700">
                Chapter Content *
              </label>
              <textarea
                id="chapterContent"
                name="chapterContent"
                required
                rows={12}
                className="mt-1 block w-full px-3 py-2 border border-slate-300 rounded-md shadow-sm focus:outline-none focus:ring-orange-500 focus:border-orange-500 sm:text-sm font-mono"
                placeholder="Write your chapter content here..."
                value={formData.chapterContent}
                onChange={handleChange}
              />
            </div>
          </div>

          {/* Advanced Options */}
          <div className="space-y-4">
            <button
              type="button"
              onClick={() => setShowAdvanced(!showAdvanced)}
              className="flex items-center text-orange-600 hover:text-orange-700 text-sm font-medium"
            >
              {showAdvanced ? '▼' : '▶'} Advanced Options
            </button>
            
            {showAdvanced && (
              <div className="space-y-4 pl-4 border-l-2 border-orange-100">
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  <div>
                    <label htmlFor="commentPolicy" className="block text-sm font-medium text-slate-700">
                      Comment Policy
                    </label>
                    <select
                      id="commentPolicy"
                      name="commentPolicy"
                      className="mt-1 block w-full px-3 py-2 border border-slate-300 rounded-md shadow-sm focus:outline-none focus:ring-orange-500 focus:border-orange-500 sm:text-sm"
                      value={formData.commentPolicy}
                      onChange={handleChange}
                    >
                      <option value="open">Open to all</option>
                      <option value="users_only">Registered users only</option>
                      <option value="disabled">Disabled</option>
                    </select>
                  </div>
                </div>

                <div className="space-y-2">
                  <label className="flex items-center">
                    <input
                      type="checkbox"
                      name="moderateComments"
                      checked={formData.moderateComments}
                      onChange={handleChange}
                      className="h-4 w-4 text-orange-600 focus:ring-orange-500 border-slate-300 rounded"
                    />
                    <span className="ml-2 text-sm text-slate-700">Moderate comments before they appear</span>
                  </label>

                  <label className="flex items-center">
                    <input
                      type="checkbox"
                      name="restrictedToUsers"
                      checked={formData.restrictedToUsers}
                      onChange={handleChange}
                      className="h-4 w-4 text-orange-600 focus:ring-orange-500 border-slate-300 rounded"
                    />
                    <span className="ml-2 text-sm text-slate-700">Only logged-in users can view</span>
                  </label>

                  <label className="flex items-center">
                    <input
                      type="checkbox"
                      name="isAnonymous"
                      checked={formData.isAnonymous}
                      onChange={handleChange}
                      className="h-4 w-4 text-orange-600 focus:ring-orange-500 border-slate-300 rounded"
                    />
                    <span className="ml-2 text-sm text-slate-700">Post anonymously</span>
                  </label>
                </div>
              </div>
            )}
          </div>

          {error && (
            <div className="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded">
              {error}
            </div>
          )}

          {/* Submit Buttons */}
          <div className="flex justify-end space-x-4 pt-6 border-t border-slate-200">
            <button
              type="button"
              onClick={() => setFormData(prev => ({ ...prev, status: 'draft' }))}
              disabled={isLoading}
              className="px-4 py-2 border border-slate-300 text-slate-700 rounded-md hover:bg-slate-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-orange-500 disabled:opacity-50"
            >
              Save as Draft
            </button>
            
            <button
              type="submit"
              onClick={() => setFormData(prev => ({ ...prev, status: 'posted' }))}
              disabled={isLoading || !formData.title || !formData.chapterContent || formData.fandoms.length === 0}
              className="px-4 py-2 bg-orange-600 text-white rounded-md hover:bg-orange-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-orange-500 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {isLoading ? 'Publishing...' : 'Publish Work'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}