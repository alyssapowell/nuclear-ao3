'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import Link from 'next/link';
import { ArrowLeft, Plus, Info, Lock, Users, UserCheck, Eye, EyeOff } from 'lucide-react';
import { createCollection, CreateCollectionRequest } from '@/lib/api';

export default function NewCollectionPage() {
  const router = useRouter();
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  // Form state
  const [formData, setFormData] = useState<CreateCollectionRequest>({
    name: '',
    title: '',
    description: '',
    is_open: true,
    is_moderated: false,
    is_anonymous: false,
    is_unrevealed: false,
  });

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    
    if (!formData.name?.trim() || !formData.title?.trim()) {
      setError('Collection name and title are required');
      return;
    }

    try {
      setLoading(true);
      setError('');
      
      const token = localStorage.getItem('auth_token');
      if (!token) {
        setError('You must be logged in to create a collection');
        return;
      }

      const response = await createCollection(formData, token);
      
      // Redirect to the new collection page
      router.push(`/collections/${response.collection.id}`);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create collection');
    } finally {
      setLoading(false);
    }
  };

  const handleInputChange = (field: keyof CreateCollectionRequest, value: string | boolean) => {
    setFormData(prev => ({
      ...prev,
      [field]: value
    }));
  };

  return (
    <div className="max-w-4xl mx-auto px-4 py-8">
      {/* Header */}
      <div className="flex items-center gap-4 mb-8">
        <Link
          href="/collections"
          className="inline-flex items-center text-orange-600 hover:text-orange-700 transition-colors"
        >
          <ArrowLeft className="w-4 h-4 mr-2" />
          Back to Collections
        </Link>
        <div className="border-l border-gray-300 pl-4">
          <h1 className="text-3xl font-bold text-gray-900">Create New Collection</h1>
          <p className="text-gray-600 mt-1">
            Set up a new collection to organize works around themes, challenges, or events.
          </p>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
        {/* Main Form */}
        <div className="lg:col-span-2">
          <form onSubmit={handleSubmit} className="bg-white rounded-lg shadow-sm border border-gray-200 p-6">
            {error && (
              <div className="mb-6 p-4 bg-red-50 border border-red-200 text-red-700 rounded-md">
                {error}
              </div>
            )}

            {/* Basic Information */}
            <div className="space-y-6">
              <div>
                <label htmlFor="name" className="block text-sm font-medium text-gray-700 mb-2">
                  Collection Name *
                </label>
                <input
                  type="text"
                  id="name"
                  value={formData.name}
                  onChange={(e) => handleInputChange('name', e.target.value)}
                  placeholder="unique_collection_name"
                  className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-orange-500 focus:border-orange-500"
                  required
                />
                <p className="mt-1 text-xs text-gray-500">
                  Unique identifier for your collection. Use lowercase letters, numbers, and underscores only.
                </p>
              </div>

              <div>
                <label htmlFor="title" className="block text-sm font-medium text-gray-700 mb-2">
                  Collection Title *
                </label>
                <input
                  type="text"
                  id="title"
                  value={formData.title}
                  onChange={(e) => handleInputChange('title', e.target.value)}
                  placeholder="My Awesome Collection"
                  className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-orange-500 focus:border-orange-500"
                  required
                />
                <p className="mt-1 text-xs text-gray-500">
                  Display name for your collection that readers will see.
                </p>
              </div>

              <div>
                <label htmlFor="description" className="block text-sm font-medium text-gray-700 mb-2">
                  Description
                </label>
                <textarea
                  id="description"
                  value={formData.description || ''}
                  onChange={(e) => handleInputChange('description', e.target.value)}
                  placeholder="Describe what this collection is about..."
                  rows={4}
                  className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-orange-500 focus:border-orange-500"
                />
                <p className="mt-1 text-xs text-gray-500">
                  Optional description to help readers understand your collection's purpose.
                </p>
              </div>
            </div>

            {/* Collection Settings */}
            <div className="mt-8 pt-6 border-t border-gray-200">
              <h3 className="text-lg font-medium text-gray-900 mb-4">Collection Settings</h3>
              
              <div className="space-y-4">
                <div className="flex items-start">
                  <div className="flex items-center h-5">
                    <input
                      id="is_open"
                      type="checkbox"
                      checked={formData.is_open}
                      onChange={(e) => handleInputChange('is_open', e.target.checked)}
                      className="w-4 h-4 text-orange-600 bg-gray-100 border-gray-300 rounded focus:ring-orange-500 focus:ring-2"
                    />
                  </div>
                  <div className="ml-3 text-sm">
                    <label htmlFor="is_open" className="font-medium text-gray-700 flex items-center">
                      <Users className="w-4 h-4 mr-2" />
                      Open Submissions
                    </label>
                    <p className="text-gray-500">Allow anyone to submit works to this collection</p>
                  </div>
                </div>

                <div className="flex items-start">
                  <div className="flex items-center h-5">
                    <input
                      id="is_moderated"
                      type="checkbox"
                      checked={formData.is_moderated}
                      onChange={(e) => handleInputChange('is_moderated', e.target.checked)}
                      className="w-4 h-4 text-orange-600 bg-gray-100 border-gray-300 rounded focus:ring-orange-500 focus:ring-2"
                    />
                  </div>
                  <div className="ml-3 text-sm">
                    <label htmlFor="is_moderated" className="font-medium text-gray-700 flex items-center">
                      <UserCheck className="w-4 h-4 mr-2" />
                      Moderated Submissions
                    </label>
                    <p className="text-gray-500">Require approval before works appear in the collection</p>
                  </div>
                </div>

                <div className="flex items-start">
                  <div className="flex items-center h-5">
                    <input
                      id="is_anonymous"
                      type="checkbox"
                      checked={formData.is_anonymous}
                      onChange={(e) => handleInputChange('is_anonymous', e.target.checked)}
                      className="w-4 h-4 text-orange-600 bg-gray-100 border-gray-300 rounded focus:ring-orange-500 focus:ring-2"
                    />
                  </div>
                  <div className="ml-3 text-sm">
                    <label htmlFor="is_anonymous" className="font-medium text-gray-700 flex items-center">
                      <EyeOff className="w-4 h-4 mr-2" />
                      Anonymous Collection
                    </label>
                    <p className="text-gray-500">Hide author names for works in this collection</p>
                  </div>
                </div>

                <div className="flex items-start">
                  <div className="flex items-center h-5">
                    <input
                      id="is_unrevealed"
                      type="checkbox"
                      checked={formData.is_unrevealed}
                      onChange={(e) => handleInputChange('is_unrevealed', e.target.checked)}
                      className="w-4 h-4 text-orange-600 bg-gray-100 border-gray-300 rounded focus:ring-orange-500 focus:ring-2"
                    />
                  </div>
                  <div className="ml-3 text-sm">
                    <label htmlFor="is_unrevealed" className="font-medium text-gray-700 flex items-center">
                      <Lock className="w-4 h-4 mr-2" />
                      Unrevealed Collection
                    </label>
                    <p className="text-gray-500">Hide works until you choose to reveal them (for challenges/gifts)</p>
                  </div>
                </div>
              </div>
            </div>

            {/* Submit Button */}
            <div className="mt-8 pt-6 border-t border-gray-200 flex gap-4">
              <button
                type="submit"
                disabled={loading}
                className="flex-1 inline-flex items-center justify-center px-6 py-3 bg-orange-600 text-white text-sm font-medium rounded-md hover:bg-orange-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
              >
                {loading ? (
                  <>
                    <div className="w-4 h-4 border-2 border-white border-t-transparent rounded-full animate-spin mr-2"></div>
                    Creating Collection...
                  </>
                ) : (
                  <>
                    <Plus className="w-4 h-4 mr-2" />
                    Create Collection
                  </>
                )}
              </button>
              
              <Link
                href="/collections"
                className="px-6 py-3 border border-gray-300 text-gray-700 text-sm font-medium rounded-md hover:bg-gray-50 transition-colors"
              >
                Cancel
              </Link>
            </div>
          </form>
        </div>

        {/* Sidebar */}
        <div className="space-y-6">
          <div className="bg-blue-50 border border-blue-200 rounded-lg p-4">
            <div className="flex items-start">
              <Info className="w-5 h-5 text-blue-600 mt-0.5 mr-3 flex-shrink-0" />
              <div>
                <h4 className="text-sm font-medium text-blue-900 mb-2">Collection Guidelines</h4>
                <ul className="text-xs text-blue-800 space-y-1">
                  <li>• Choose a unique collection name that represents your theme</li>
                  <li>• Write a clear description to help authors understand the purpose</li>
                  <li>• Consider using moderation for curated collections</li>
                  <li>• Anonymous collections are great for challenges and exchanges</li>
                </ul>
              </div>
            </div>
          </div>

          <div className="bg-gray-50 border border-gray-200 rounded-lg p-4">
            <h4 className="text-sm font-medium text-gray-900 mb-3">Collection Settings Guide</h4>
            
            <div className="space-y-3 text-xs text-gray-700">
              <div>
                <div className="font-medium flex items-center">
                  <Users className="w-3 h-3 mr-1" />
                  Open Submissions
                </div>
                <p>When enabled, any author can add their works to your collection.</p>
              </div>
              
              <div>
                <div className="font-medium flex items-center">
                  <UserCheck className="w-3 h-3 mr-1" />
                  Moderated
                </div>
                <p>Works will only appear after you approve them. Good for curated collections.</p>
              </div>
              
              <div>
                <div className="font-medium flex items-center">
                  <EyeOff className="w-3 h-3 mr-1" />
                  Anonymous
                </div>
                <p>Author names are hidden. Perfect for blind reviews or gift exchanges.</p>
              </div>
              
              <div>
                <div className="font-medium flex items-center">
                  <Lock className="w-3 h-3 mr-1" />
                  Unrevealed
                </div>
                <p>Works are hidden until you reveal them. Great for timed reveals and surprises.</p>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}