'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import TagInput from '@/components/TagInput';
import TagAutocomplete from '@/components/TagAutocomplete';
import RichTextEditor from '@/components/RichTextEditor';
import EnhancedTagProminenceSelector from '@/components/EnhancedTagProminenceSelector';
import { createWork, CreateWorkRequest } from '@/lib/api';

export default function NewWorkPage() {
  const [formData, setFormData] = useState({
    title: '',
    summary: '',
    notes: '',
    rating: 'Not Rated',
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
  const [tagInputs, setTagInputs] = useState({
    fandom: '',
    character: '',
    relationship: '',
    freeform: ''
  });
  const [showAdvanced, setShowAdvanced] = useState(false);
  const [prominenceTags, setProminenceTags] = useState<Array<{
    tagName: string;
    tagType: string;
    prominence: 'primary' | 'secondary' | 'micro';
    autoSuggested?: boolean;
    canonical?: boolean;
    useCount?: number;
  }>>([]);
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

  const handleTagSelect = (tagType: string, tag: any) => {
    const tagName = tag.name || tag;
    if (!tagName) return;
    
    const fieldMap = {
      fandom: 'fandoms',
      character: 'characters', 
      relationship: 'relationships',
      freeform: 'freeformTags'
    } as const;
    
    const field = fieldMap[tagType as keyof typeof fieldMap];
    if (!field) return;
    
    // Check if tag already exists
    if (!formData[field].includes(tagName)) {
      setFormData(prev => ({ 
        ...prev, 
        [field]: [...prev[field], tagName] 
      }));
      
      // Clear the corresponding input
      setTagInputs(prev => ({
        ...prev,
        [tagType]: ''
      }));
    }
  };

  const removeTag = (tagType: string, index: number) => {
    const fieldMap = {
      fandom: 'fandoms',
      character: 'characters',
      relationship: 'relationships', 
      freeform: 'freeformTags'
    } as const;
    
    const field = fieldMap[tagType as keyof typeof fieldMap];
    if (!field) return;
    
    setFormData(prev => ({
      ...prev,
      [field]: prev[field].filter((_, i) => i !== index)
    }));
  };

  const handleTagsChange = (field: string, tags: string[]) => {
    setFormData(prev => ({ ...prev, [field]: tags }));
  };

  // Enhanced tag prominence handler
  const handleProminenceTagsChange = (tags: typeof prominenceTags) => {
    setProminenceTags(tags);
    
    // Update form data from prominence tags
    const fandoms = tags.filter(t => t.tagType === 'fandom').map(t => t.tagName);
    const characters = tags.filter(t => t.tagType === 'character').map(t => t.tagName);
    const relationships = tags.filter(t => t.tagType === 'relationship').map(t => t.tagName);
    const freeformTags = tags.filter(t => t.tagType === 'freeform').map(t => t.tagName);
    
    setFormData(prev => ({
      ...prev,
      fandoms,
      characters,
      relationships,
      freeformTags
    }));
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
        // Enhanced tag prominence data
        tag_prominence: prominenceTags.map(tag => ({
          tag_name: tag.tagName,
          tag_type: tag.tagType,
          prominence: tag.prominence,
          auto_suggested: tag.autoSuggested
        })),
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
              <p className="text-xs text-slate-500 mb-2">
                Use the toolbar for formatting. You can add headings, lists, links, and more.
              </p>
              <RichTextEditor
                content={formData.chapterContent}
                onChange={(content) => setFormData(prev => ({ ...prev, chapterContent: content }))}
                placeholder="Write your chapter content here..."
                className="mt-1"
              />
            </div>

            {/* Enhanced Tag Selection with Prominence System */}
            <div className="space-y-4">
              <div className="flex items-center justify-between">
                <h3 className="text-md font-medium text-slate-700">Enhanced Tagging System</h3>
                <div className="text-xs text-slate-500 bg-blue-50 px-2 py-1 rounded">
                  ✨ Smart tag prominence • Prevents tag spam
                </div>
              </div>
              
              {/* Required Fandom Selection */}
              <div className="bg-orange-50 border border-orange-200 rounded-lg p-4">
                <h4 className="text-sm font-medium text-orange-800 mb-2">Fandom Selection Required</h4>
                <p className="text-xs text-orange-700 mb-3">
                  Select at least one fandom before adding other tags for better autocomplete suggestions.
                </p>
                
                {/* Basic fandom input for required selection */}
                {formData.fandoms.length > 0 && (
                  <div className="flex flex-wrap gap-2 mb-2">
                    {formData.fandoms.map((fandom, index) => (
                      <span
                        key={index}
                        className="inline-flex items-center px-3 py-1 rounded-full text-sm font-medium bg-orange-100 text-orange-800 border border-orange-200"
                      >
                        {fandom}
                        <button
                          type="button"
                          onClick={() => removeTag('fandom', index)}
                          className="ml-2 text-orange-600 hover:text-orange-800 focus:outline-none"
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
                  value={tagInputs.fandom}
                  onChange={(value) => setTagInputs(prev => ({ ...prev, fandom: value }))}
                  onTagSelect={(tag) => handleTagSelect('fandom', tag)}
                  placeholder="Start typing a fandom name..."
                  tagType="fandom"
                  className="mt-1 block w-full px-3 py-2 border border-orange-300 rounded-md shadow-sm focus:outline-none focus:ring-orange-500 focus:border-orange-500 sm:text-sm"
                />
              </div>

              {/* Enhanced Tag Prominence Selector */}
              <EnhancedTagProminenceSelector
                tags={prominenceTags}
                onTagsChange={handleProminenceTagsChange}
                fandomId={formData.fandoms[0]} // Use first fandom for context
                className="border border-slate-200 rounded-lg p-4"
              />
              
              {/* Guidance Box */}
              <div className="bg-slate-50 border border-slate-200 rounded-lg p-4">
                <h4 className="text-sm font-semibold text-slate-800 mb-2">🎯 Smart Tagging Guidelines</h4>
                <div className="grid grid-cols-1 md:grid-cols-3 gap-4 text-xs text-slate-600">
                  <div>
                    <span className="font-medium text-green-700">Primary:</span> Main focus of your story (2-3 max)
                  </div>
                  <div>
                    <span className="font-medium text-blue-700">Secondary:</span> Important but not central elements
                  </div>
                  <div>
                    <span className="font-medium text-gray-700">Micro:</span> Background mentions, past relationships
                  </div>
                </div>
                <p className="text-xs text-slate-500 mt-2">
                  💡 Use "Background", "Past", or "Minor" before tag names for automatic micro prominence
                </p>
              </div>
            </div>
          </div>

          {/* Basic Work Metadata */}
          <div className="space-y-4">
            <h3 className="text-md font-medium text-slate-700">Work Metadata</h3>
            
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              {/* Rating */}
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
                  <option value="Not Rated">Not Rated</option>
                  <option value="General Audiences">General Audiences</option>
                  <option value="Teen And Up Audiences">Teen And Up Audiences</option>
                  <option value="Mature">Mature</option>
                  <option value="Explicit">Explicit</option>
                </select>
              </div>

              {/* Language */}
              <div>
                <label htmlFor="language" className="block text-sm font-medium text-slate-700">
                  Language *
                </label>
                <select
                  id="language"
                  name="language"
                  required
                  className="mt-1 block w-full px-3 py-2 border border-slate-300 rounded-md shadow-sm focus:outline-none focus:ring-orange-500 focus:border-orange-500 sm:text-sm"
                  value={formData.language}
                  onChange={handleChange}
                >
                  <option value="en">English</option>
                  <option value="es">Español</option>
                  <option value="fr">Français</option>
                  <option value="de">Deutsch</option>
                  <option value="it">Italiano</option>
                  <option value="pt">Português</option>
                  <option value="ru">Русский</option>
                  <option value="ja">日本語</option>
                  <option value="zh">中文</option>
                  <option value="ko">한국어</option>
                  <option value="other">Other</option>
                </select>
              </div>
            </div>

            {/* Archive Warnings */}
            <div>
              <label className="block text-sm font-medium text-slate-700">
                Archive Warnings *
              </label>
              <p className="text-xs text-slate-500 mb-2">
                Choose all that apply to your work
              </p>
              <div className="space-y-2">
                {[
                  { value: 'no_warnings', label: 'No Archive Warnings Apply' },
                  { value: 'creator_chose_not_to_warn', label: 'Creator Chose Not To Use Archive Warnings' },
                  { value: 'graphic_violence', label: 'Graphic Depictions Of Violence' },
                  { value: 'major_character_death', label: 'Major Character Death' },
                  { value: 'rape_noncon', label: 'Rape/Non-Con' },
                  { value: 'underage', label: 'Underage' }
                ].map(warning => (
                  <label key={warning.value} className="flex items-center">
                    <input
                      type="checkbox"
                      className="rounded border-slate-300 text-orange-600 focus:ring-orange-500"
                      checked={formData.warnings.includes(warning.value)}
                      onChange={(e) => {
                        if (e.target.checked) {
                          setFormData(prev => ({ 
                            ...prev, 
                            warnings: [...prev.warnings, warning.value] 
                          }));
                        } else {
                          setFormData(prev => ({ 
                            ...prev, 
                            warnings: prev.warnings.filter(w => w !== warning.value) 
                          }));
                        }
                      }}
                    />
                    <span className="ml-2 text-sm text-slate-700">{warning.label}</span>
                  </label>
                ))}
              </div>
            </div>

            {/* Categories */}
            <div>
              <label className="block text-sm font-medium text-slate-700">
                Categories
              </label>
              <p className="text-xs text-slate-500 mb-2">
                Choose all that apply to your work
              </p>
              <div className="grid grid-cols-2 md:grid-cols-3 gap-2">
                {[
                  { value: 'gen', label: 'Gen' },
                  { value: 'm_m', label: 'M/M' },
                  { value: 'f_f', label: 'F/F' },
                  { value: 'm_f', label: 'M/F' },
                  { value: 'multi', label: 'Multi' },
                  { value: 'other', label: 'Other' }
                ].map(category => (
                  <label key={category.value} className="flex items-center">
                    <input
                      type="checkbox"
                      className="rounded border-slate-300 text-orange-600 focus:ring-orange-500"
                      checked={formData.category.includes(category.value)}
                      onChange={(e) => {
                        if (e.target.checked) {
                          setFormData(prev => ({ 
                            ...prev, 
                            category: [...prev.category, category.value] 
                          }));
                        } else {
                          setFormData(prev => ({ 
                            ...prev, 
                            category: prev.category.filter(c => c !== category.value) 
                          }));
                        }
                      }}
                    />
                    <span className="ml-2 text-sm text-slate-700">{category.label}</span>
                  </label>
                ))}
              </div>
            </div>

            {/* Work Status and Chapter Info */}
            <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
              <div>
                <label htmlFor="status" className="block text-sm font-medium text-slate-700">
                  Work Status
                </label>
                <select
                  id="status"
                  name="status"
                  className="mt-1 block w-full px-3 py-2 border border-slate-300 rounded-md shadow-sm focus:outline-none focus:ring-orange-500 focus:border-orange-500 sm:text-sm"
                  value={formData.status}
                  onChange={handleChange}
                >
                  <option value="draft">Draft</option>
                  <option value="complete">Complete</option>
                  <option value="in_progress">Work in Progress</option>
                </select>
              </div>

              <div>
                <label htmlFor="chapterTitle" className="block text-sm font-medium text-slate-700">
                  Chapter Title
                </label>
                <input
                  type="text"
                  id="chapterTitle"
                  name="chapterTitle"
                  className="mt-1 block w-full px-3 py-2 border border-slate-300 rounded-md shadow-sm focus:outline-none focus:ring-orange-500 focus:border-orange-500 sm:text-sm"
                  placeholder="Chapter 1"
                  value={formData.chapterTitle}
                  onChange={handleChange}
                />
              </div>

              <div>
                <label htmlFor="maxChapters" className="block text-sm font-medium text-slate-700">
                  Total Chapters
                </label>
                <input
                  type="text"
                  id="maxChapters"
                  name="maxChapters"
                  className="mt-1 block w-full px-3 py-2 border border-slate-300 rounded-md shadow-sm focus:outline-none focus:ring-orange-500 focus:border-orange-500 sm:text-sm"
                  placeholder="? or 1 or 5"
                  value={formData.maxChapters}
                  onChange={handleChange}
                />
              </div>
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