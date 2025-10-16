'use client';

import { useState, KeyboardEvent } from 'react';
import { useRouter } from 'next/navigation';
import { createWork, CreateWorkRequest } from '@/lib/api';
import TagAutocomplete from '@/components/TagAutocomplete';
import RichTextEditor from '@/components/RichTextEditor';

// Enhanced tag input component with autocomplete
interface EnhancedTagInputProps {
  tags: string[];
  onChange: (tags: string[]) => void;
  placeholder: string;
  tagType: 'fandom' | 'character' | 'relationship' | 'freeform';
  required?: boolean;
}

// Simple tag input component without autocomplete
interface SimpleTagInputProps {
  tags: string[];
  onChange: (tags: string[]) => void;
  placeholder: string;
}

function EnhancedTagInput({ tags, onChange, placeholder, tagType, required }: EnhancedTagInputProps) {
  const [inputValue, setInputValue] = useState('');

  const handleTagSelect = (tag: any) => {
    const tagName = tag.name || tag;
    if (tagName && !tags.includes(tagName)) {
      onChange([...tags, tagName]);
    }
    setInputValue('');
  };

  const handleInputChange = (value: string) => {
    // Auto-add tag on comma
    if (value.endsWith(',')) {
      const newTag = value.slice(0, -1).trim();
      if (newTag && !tags.includes(newTag)) {
        onChange([...tags, newTag]);
        setInputValue('');
        return;
      }
    }
    setInputValue(value);
  };

  const removeTag = (index: number) => {
    onChange(tags.filter((_, i) => i !== index));
  };

  return (
    <div className="space-y-2">
      {/* Display selected tags */}
      {tags.length > 0 && (
        <div className="flex flex-wrap gap-2">
          {tags.map((tag, index) => (
            <span
              key={index}
              className="inline-flex items-center px-3 py-1 rounded-full text-sm font-medium bg-blue-100 text-blue-800 border border-blue-200"
            >
              {tag}
              <button
                type="button"
                onClick={() => removeTag(index)}
                className="ml-2 text-blue-600 hover:text-blue-800 focus:outline-none"
                aria-label={`Remove ${tag}`}
              >
                ×
              </button>
            </span>
          ))}
        </div>
      )}
      
      {/* Input with autocomplete */}
      <TagAutocomplete
        value={inputValue}
        onChange={handleInputChange}
        onTagSelect={handleTagSelect}
        placeholder={tags.length === 0 ? placeholder : "Add another tag..."}
        tagType={tagType}
        required={required && tags.length === 0}
        className="w-full px-3 py-2 border border-slate-300 rounded-md focus:outline-none focus:ring-2 focus:ring-orange-500 focus:border-orange-500"
      />
      
      <div className="text-xs text-slate-500">
        Type and press Enter, or use comma to add tags. Auto-suggestions will appear as you type.
      </div>
    </div>
  );
}

function SimpleTagInput({ tags, onChange, placeholder }: SimpleTagInputProps) {
  const [inputValue, setInputValue] = useState('');

  const handleKeyDown = (e: KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter' || e.key === ',') {
      e.preventDefault();
      const newTag = inputValue.trim();
      if (newTag && !tags.includes(newTag)) {
        onChange([...tags, newTag]);
        setInputValue('');
      }
    }
    if (e.key === 'Backspace' && inputValue === '' && tags.length > 0) {
      onChange(tags.slice(0, -1));
    }
  };

  const removeTag = (index: number) => {
    onChange(tags.filter((_, i) => i !== index));
  };

  return (
    <div className="border border-slate-300 rounded-md p-2 focus-within:ring-2 focus-within:ring-orange-500 focus-within:border-orange-500">
      <div className="flex flex-wrap gap-1 mb-2">
        {tags.map((tag, index) => (
          <span
            key={index}
            className="inline-flex items-center px-2 py-1 rounded-md text-sm font-medium bg-blue-100 text-blue-800"
          >
            {tag}
            <button
              type="button"
              onClick={() => removeTag(index)}
              className="ml-1 text-blue-600 hover:text-blue-800 focus:outline-none"
            >
              ×
            </button>
          </span>
        ))}
      </div>
      <input
        type="text"
        value={inputValue}
        onChange={(e) => setInputValue(e.target.value)}
        onKeyDown={handleKeyDown}
        placeholder={tags.length === 0 ? placeholder : "Add another tag..."}
        className="w-full border-0 focus:ring-0 focus:outline-none text-sm"
      />
      <div className="text-xs text-gray-500 mt-1">
        Press Enter or comma to add tags
      </div>
    </div>
  );
}

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
    // Add missing chapter header notes
    chapterBeginNotes: '',
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
      // Handle array fields (category, warnings) properly
      if (name === 'category' || name === 'warnings') {
        setFormData(prev => {
          const currentArray = prev[name] as string[];
          if (checked) {
            return { ...prev, [name]: [...currentArray, value] };
          } else {
            return { ...prev, [name]: currentArray.filter(item => item !== value) };
          }
        });
      } else {
        // Handle boolean fields
        setFormData(prev => ({ ...prev, [name]: checked }));
      }
    } else {
      setFormData(prev => ({ ...prev, [name]: value }));
    }
  };

  const handleTagsChange = (field: string, tags: string[]) => {
    setFormData(prev => ({ ...prev, [field]: tags }));
  };

  // Function to auto-convert any remaining text in tag inputs to actual tags before submission
  const autoConvertPendingTags = () => {
    // This will be handled by the enhanced tag inputs themselves
    // Each EnhancedTagInput will auto-convert text on blur/submission
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsLoading(true);
    setError('');

    // Auto-convert any pending text in tag inputs
    autoConvertPendingTags();

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
        chapter_notes: formData.chapterBeginNotes,
        chapter_end_notes: formData.chapterEndNotes,
        chapter_content: formData.chapterContent,
        restricted_to_users: formData.restrictedToUsers,
        comment_policy: formData.commentPolicy,
        moderate_comments: formData.moderateComments,
        is_anonymous: formData.isAnonymous,
      };

      const data = await createWork(payload as CreateWorkRequest, token);
      router.push(`/works/${data.work.id}`);
    } catch (err) {
      console.error('Create work error:', err);
      if (err instanceof Error) {
        setError(`Failed to create work: ${err.message}`);
      } else {
        setError('Failed to create work. Please try again.');
      }
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

          </div>

          {/* Chapter Information */}
          <div className="space-y-4">
            <h2 className="text-lg font-semibold text-slate-800">Chapter Information</h2>
            
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
                  placeholder="Enter chapter title..."
                  value={formData.chapterTitle}
                  onChange={handleChange}
                />
              </div>

              <div>
                <label htmlFor="chapterSummary" className="block text-sm font-medium text-slate-700">
                  Chapter Summary
                </label>
                <textarea
                  id="chapterSummary"
                  name="chapterSummary"
                  rows={2}
                  className="mt-1 block w-full px-3 py-2 border border-slate-300 rounded-md shadow-sm focus:outline-none focus:ring-orange-500 focus:border-orange-500 sm:text-sm"
                  placeholder="Brief summary of this chapter..."
                  value={formData.chapterSummary}
                  onChange={handleChange}
                />
              </div>
            </div>

            <div>
              <label htmlFor="chapterBeginNotes" className="block text-sm font-medium text-slate-700">
                Chapter Notes (Beginning)
              </label>
              <textarea
                id="chapterBeginNotes"
                name="chapterBeginNotes"
                rows={3}
                className="mt-1 block w-full px-3 py-2 border border-slate-300 rounded-md shadow-sm focus:outline-none focus:ring-orange-500 focus:border-orange-500 sm:text-sm"
                placeholder="Notes to display before the chapter content..."
                value={formData.chapterBeginNotes}
                onChange={handleChange}
              />
            </div>

            <div>
              <label htmlFor="chapterContent" className="block text-sm font-medium text-slate-700">
                Chapter Content *
              </label>
              <RichTextEditor
                content={formData.chapterContent}
                onChange={(content) => setFormData(prev => ({ ...prev, chapterContent: content }))}
                placeholder="Write your chapter content here..."
                className="mt-1"
                disabled={isLoading}
              />
              <div className="mt-2 text-sm text-slate-500">
                Use the toolbar for formatting. Bold, italic, headings, lists, and links are supported.
              </div>
            </div>

            <div>
              <label htmlFor="chapterEndNotes" className="block text-sm font-medium text-slate-700">
                Chapter Notes (End)
              </label>
              <textarea
                id="chapterEndNotes"
                name="chapterEndNotes"
                rows={3}
                className="mt-1 block w-full px-3 py-2 border border-slate-300 rounded-md shadow-sm focus:outline-none focus:ring-orange-500 focus:border-orange-500 sm:text-sm"
                placeholder="Notes to display after the chapter content..."
                value={formData.chapterEndNotes}
                onChange={handleChange}
              />
            </div>
          </div>

          {/* Rating and Warnings */}
          <div className="space-y-4">
            <h2 className="text-lg font-semibold text-slate-800">Rating & Warnings</h2>
            
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div>
                <label className="block text-sm font-medium text-slate-700 mb-2">
                  Rating *
                </label>
                <div className="space-y-2">
                    {['General Audiences', 'Teen And Up Audiences', 'Mature', 'Explicit', 'Not Rated'].map((rating) => (
                      <label key={rating} className="flex items-center">
                        <input
                          type="radio"
                          name="rating"
                          value={rating}
                          checked={formData.rating === rating}
                          onChange={handleChange}
                          className="mr-2"
                        />
                        <span className="text-sm">{rating}</span>
                      </label>
                    ))}
                </div>
              </div>

              <div>
                <label className="block text-sm font-medium text-slate-700 mb-2">
                  Archive Warnings
                </label>
                <div className="space-y-2">
                  {[
                    'No Archive Warnings Apply',
                    'Graphic Depictions Of Violence', 
                    'Major Character Death',
                    'Rape/Non-Con',
                    'Underage',
                    'Choose Not To Use Archive Warnings'
                  ].map((warning) => (
                    <label key={warning} className="flex items-center">
                      <input
                        type="checkbox"
                        value={warning}
                        checked={formData.warnings.includes(warning)}
                        onChange={(e) => {
                          const { value, checked } = e.target;
                          setFormData(prev => ({
                            ...prev,
                            warnings: checked 
                              ? [...prev.warnings, value]
                              : prev.warnings.filter(w => w !== value)
                          }));
                        }}
                        className="mr-2"
                      />
                      <span className="text-sm">{warning}</span>
                    </label>
                  ))}
                </div>
              </div>
            </div>

            <div>
              <label className="block text-sm font-medium text-slate-700 mb-2">
                Categories
              </label>
              <div className="flex flex-wrap gap-4">
                {['F/M', 'M/M', 'F/F', 'Gen', 'Multi', 'Other'].map((cat) => (
                  <label key={cat} className="flex items-center">
                    <input
                      type="checkbox"
                      value={cat}
                      checked={formData.category.includes(cat)}
                      onChange={(e) => {
                        const { value, checked } = e.target;
                        setFormData(prev => ({
                          ...prev,
                          category: checked 
                            ? [...prev.category, value]
                            : prev.category.filter(c => c !== value)
                        }));
                      }}
                      className="mr-2"
                    />
                    <span className="text-sm">{cat}</span>
                  </label>
                ))}
              </div>
            </div>
          </div>

          {/* Tags */}
          <div className="space-y-4">
            <h2 className="text-lg font-semibold text-slate-800">Tags</h2>
            
            <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
              <div>
                <label className="block text-sm font-medium text-slate-700 mb-2">
                  Fandoms *
                </label>
                <EnhancedTagInput
                  tags={formData.fandoms}
                  onChange={(tags) => handleTagsChange('fandoms', tags)}
                  placeholder="Start typing a fandom name..."
                  tagType="fandom"
                  required
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-slate-700 mb-2">
                  Characters
                </label>
                <EnhancedTagInput
                  tags={formData.characters}
                  onChange={(tags) => handleTagsChange('characters', tags)}
                  placeholder="Start typing character names..."
                  tagType="character"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-slate-700 mb-2">
                  Relationships
                </label>
                <EnhancedTagInput
                  tags={formData.relationships}
                  onChange={(tags) => handleTagsChange('relationships', tags)}
                  placeholder="Start typing relationships..."
                  tagType="relationship"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-slate-700 mb-2">
                  Additional Tags
                </label>
                <EnhancedTagInput
                  tags={formData.freeformTags}
                  onChange={(tags) => handleTagsChange('freeformTags', tags)}
                  placeholder="Start typing additional tags..."
                  tagType="freeform"
                />
              </div>
            </div>
          </div>

          {/* Additional Work Settings */}
          <div className="space-y-4">
            <h2 className="text-lg font-semibold text-slate-800">Work Settings</h2>
            
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div>
                <label htmlFor="language" className="block text-sm font-medium text-slate-700">
                  Language
                </label>
                <select
                  id="language"
                  name="language"
                  value={formData.language}
                  onChange={handleChange}
                  className="mt-1 block w-full px-3 py-2 border border-slate-300 rounded-md shadow-sm focus:outline-none focus:ring-orange-500 focus:border-orange-500 sm:text-sm"
                >
                  <option value="en">English</option>
                  <option value="es">Español</option>
                  <option value="fr">Français</option>
                  <option value="de">Deutsch</option>
                  <option value="it">Italiano</option>
                  <option value="pt">Português</option>
                  <option value="ja">日本語</option>
                  <option value="ko">한국어</option>
                  <option value="zh">中文</option>
                </select>
              </div>

              <div>
                <label htmlFor="maxChapters" className="block text-sm font-medium text-slate-700">
                  Expected Chapters
                </label>
                <input
                  type="number"
                  id="maxChapters"
                  name="maxChapters"
                  placeholder="? or number"
                  className="mt-1 block w-full px-3 py-2 border border-slate-300 rounded-md shadow-sm focus:outline-none focus:ring-orange-500 focus:border-orange-500 sm:text-sm"
                  value={formData.maxChapters}
                  onChange={handleChange}
                />
              </div>
            </div>

            <div className="flex items-center space-x-6">
              <label className="flex items-center">
                <input
                  type="checkbox"
                  name="restrictedToUsers"
                  checked={formData.restrictedToUsers}
                  onChange={handleChange}
                  className="mr-2"
                />
                <span className="text-sm">Restrict to registered users only</span>
              </label>

              <label className="flex items-center">
                <input
                  type="checkbox"
                  name="isAnonymous"
                  checked={formData.isAnonymous}
                  onChange={handleChange}
                  className="mr-2"
                />
                <span className="text-sm">Post anonymously</span>
              </label>
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