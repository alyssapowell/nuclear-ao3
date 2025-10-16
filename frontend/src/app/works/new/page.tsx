'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import TagInput from '@/components/TagInput';
import TagAutocomplete from '@/components/TagAutocomplete';
import EnhancedTagProminenceSelector from '@/components/EnhancedTagProminenceSelector';
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
  const [fandomInput, setFandomInput] = useState('');
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

  const handleTagSelect = (tag: any) => {
    const tagName = tag.name || tag;
    if (tagName && !formData.fandoms.includes(tagName)) {
      setFormData(prev => ({ 
        ...prev, 
        fandoms: [...prev.fandoms, tagName] 
      }));
      setFandomInput(''); // Clear input after selection
    }
  };

  const removeFandom = (index: number) => {
    setFormData(prev => ({
      ...prev,
      fandoms: prev.fandoms.filter((_, i) => i !== index)
    }));
  };

  const handleTagsChange = (field: string, tags: string[]) => {
    setFormData(prev => ({ ...prev, [field]: tags }));
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsLoading(true);
    setError('');

    try {
      const token = localStorage.getItem('auth_token');
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
      setError('Failed to create work. Please try again.');
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

            {/* Test TagAutocomplete */}
            <div>
              <label htmlFor="fandoms" className="block text-sm font-medium text-slate-700">
                Fandoms (Test TagAutocomplete)
              </label>
              
              {/* Display selected fandoms */}
              {formData.fandoms.length > 0 && (
                <div className="flex flex-wrap gap-2 mb-2">
                  {formData.fandoms.map((fandom, index) => (
                    <span
                      key={index}
                      className="inline-flex items-center px-3 py-1 rounded-full text-sm font-medium bg-blue-100 text-blue-800 border border-blue-200"
                    >
                      {fandom}
                      <button
                        type="button"
                        onClick={() => removeFandom(index)}
                        className="ml-2 text-blue-600 hover:text-blue-800 focus:outline-none"
                        aria-label={`Remove ${fandom}`}
                      >
                        ×
                      </button>
                    </span>
                  ))}
                </div>
              )}
              
              <TagAutocomplete
                id="fandoms"
                value={fandomInput}
                onChange={setFandomInput}
                onTagSelect={handleTagSelect}
                placeholder="Start typing a fandom name..."
                tagType="fandom"
                className="mt-1 block w-full px-3 py-2 border border-slate-300 rounded-md shadow-sm focus:outline-none focus:ring-orange-500 focus:border-orange-500 sm:text-sm"
              />
            </div>
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
              disabled={isLoading}
              className="px-4 py-2 border border-slate-300 text-slate-700 rounded-md hover:bg-slate-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-orange-500 disabled:opacity-50"
            >
              Save as Draft
            </button>
            
            <button
              type="submit"
              disabled={isLoading || !formData.title || !formData.chapterContent}
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