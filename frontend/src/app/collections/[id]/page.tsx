'use client';

import { useState, useEffect } from 'react';
import Link from 'next/link';
import { useParams } from 'next/navigation';
import { getCollection, Collection } from '@/lib/api';

export default function CollectionPage() {
  const params = useParams();
  const collectionId = params.id as string;
  
  const [collection, setCollection] = useState<Collection | null>(null);
  const [maintainer, setMaintainer] = useState<string>('');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (collectionId) {
      fetchCollection();
    }
  }, [collectionId]);

  const fetchCollection = async () => {
    try {
      setLoading(true);
      
      const data = await getCollection(collectionId);
      
      setCollection(data.collection);
      setMaintainer(data.maintainer || 'Unknown');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load collection');
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

  if (error || !collection) {
    return (
      <div className="max-w-6xl mx-auto px-4 py-8">
        <div className="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded">
          {error || 'Collection not found'}
        </div>
      </div>
    );
  }

  return (
    <div className="max-w-6xl mx-auto px-4 py-8">
      {/* Collection Header */}
      <div className="bg-white rounded-lg shadow-sm border border-slate-200 p-6 mb-6">
        <div className="flex justify-between items-start mb-4">
          <div className="flex-1">
            <h1 className="text-3xl font-bold text-slate-900 mb-2">{collection.title}</h1>
            <div className="text-lg text-slate-600 mb-2">@{collection.name}</div>
            <div className="flex items-center gap-2 text-sm text-slate-600 mb-4">
              <span>maintained by</span>
              <Link 
                href={`/users/${collection.user_id}`}
                className="text-orange-600 hover:text-orange-700 font-medium"
              >
                {maintainer}
              </Link>
            </div>
          </div>
          
          {/* Collection Management Actions */}
          <div className="flex flex-col gap-2">
            <Link 
              href={`/collections/${collectionId}/edit`}
              className="inline-flex items-center px-3 py-2 border border-slate-300 text-sm font-medium rounded-md text-slate-700 bg-white hover:bg-slate-50 transition-colors"
            >
              <svg className="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
              </svg>
              Edit Settings
            </Link>
            
            <Link 
              href={`/collections/${collectionId}/manage`}
              className="inline-flex items-center px-3 py-2 border border-slate-300 text-sm font-medium rounded-md text-slate-700 bg-white hover:bg-slate-50 transition-colors"
            >
              <svg className="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5H7a2 2 0 00-2 2v10a2 2 0 002 2h8a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2" />
              </svg>
              Manage Works
            </Link>
            
            {collection.is_moderated && (
              <Link 
                href={`/collections/${collectionId}/moderate`}
                className="inline-flex items-center px-3 py-2 border border-orange-300 text-sm font-medium rounded-md text-orange-700 bg-orange-50 hover:bg-orange-100 transition-colors"
              >
                <svg className="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
                </svg>
                Moderate Queue
              </Link>
            )}
          </div>
        </div>

        {/* Collection Settings */}
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-6">
          <div className="text-center">
            <div className="text-2xl font-bold text-blue-600">{collection.work_count}</div>
            <div className="text-sm text-slate-600">Works</div>
          </div>
          <div className="text-center">
            <div className={`text-2xl font-bold ${collection.is_open ? 'text-green-600' : 'text-red-600'}`}>
              {collection.is_open ? 'Open' : 'Closed'}
            </div>
            <div className="text-sm text-slate-600">Submissions</div>
          </div>
          <div className="text-center">
            <div className={`text-2xl font-bold ${collection.is_moderated ? 'text-yellow-600' : 'text-green-600'}`}>
              {collection.is_moderated ? 'Moderated' : 'Unmoderated'}
            </div>
            <div className="text-sm text-slate-600">Approval</div>
          </div>
          <div className="text-center">
            <div className={`text-2xl font-bold ${collection.is_anonymous ? 'text-purple-600' : 'text-slate-600'}`}>
              {collection.is_anonymous ? 'Anonymous' : 'Named'}
            </div>
            <div className="text-sm text-slate-600">Attribution</div>
          </div>
        </div>

        {/* Collection Description */}
        {collection.description && (
          <div className="border-t border-slate-200 pt-4">
            <h3 className="text-sm font-medium text-slate-700 mb-2">About This Collection</h3>
            <div className="prose prose-sm text-slate-600">
              {collection.description.split('\n').map((line, index) => (
                <p key={index}>{line}</p>
              ))}
            </div>
          </div>
        )}

        {/* Collection Details */}
        <div className="border-t border-slate-200 pt-4 mt-4">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4 text-sm">
            <div>
              <span className="text-slate-600">Created:</span>{' '}
              <span className="text-slate-900">{formatDate(collection.created_at)}</span>
            </div>
            <div>
              <span className="text-slate-600">Last Updated:</span>{' '}
              <span className="text-slate-900">{formatDate(collection.updated_at)}</span>
            </div>
          </div>
        </div>
      </div>

      {/* Collection Rules and Submission Info */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 mb-8">
        <div className="bg-white rounded-lg shadow-sm border border-slate-200 p-6">
          <h2 className="text-xl font-semibold text-slate-900 mb-4">
            <svg className="w-5 h-5 inline mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
            Collection Status
          </h2>
          <div className="space-y-3 text-sm">
            {collection.is_open ? (
              <div className="flex items-center text-green-700">
                <svg className="w-4 h-4 mr-2" fill="currentColor" viewBox="0 0 20 20">
                  <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clipRule="evenodd" />
                </svg>
                This collection is open to new submissions
              </div>
            ) : (
              <div className="flex items-center text-red-700">
                <svg className="w-4 h-4 mr-2" fill="currentColor" viewBox="0 0 20 20">
                  <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clipRule="evenodd" />
                </svg>
                This collection is currently closed to new submissions
              </div>
            )}
            
            {collection.is_moderated && (
              <div className="flex items-center text-yellow-700">
                <svg className="w-4 h-4 mr-2" fill="currentColor" viewBox="0 0 20 20">
                  <path fillRule="evenodd" d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z" clipRule="evenodd" />
                </svg>
                All submissions require moderator approval
              </div>
            )}
            
            {collection.is_anonymous && (
              <div className="flex items-center text-purple-700">
                <svg className="w-4 h-4 mr-2" fill="currentColor" viewBox="0 0 20 20">
                  <path fillRule="evenodd" d="M3.707 2.293a1 1 0 00-1.414 1.414l14 14a1 1 0 001.414-1.414l-1.473-1.473A10.014 10.014 0 0019.542 10C18.268 5.943 14.478 3 10 3a9.958 9.958 0 00-4.512 1.074l-1.78-1.781zm4.261 4.26l1.514 1.515a2.003 2.003 0 012.45 2.45l1.514 1.514a4 4 0 00-5.478-5.478z" clipRule="evenodd" />
                </svg>
                Works in this collection are posted anonymously
              </div>
            )}
          </div>
        </div>

        <div className="bg-white rounded-lg shadow-sm border border-slate-200 p-6">
          <h2 className="text-xl font-semibold text-slate-900 mb-4">
            <svg className="w-5 h-5 inline mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 6v6m0 0v6m0-6h6m-6 0H6" />
            </svg>
            Submit to Collection
          </h2>
          {collection.is_open ? (
            <div className="space-y-3">
              <p className="text-sm text-slate-600">
                To add your work to this collection, you can include the collection name in your work&apos;s associations when posting or editing.
              </p>
              <div className="bg-slate-50 p-3 rounded text-sm">
                <strong>Collection Name:</strong> <code className="bg-white px-2 py-1 rounded">{collection.name}</code>
              </div>
              {collection.is_moderated && (
                <p className="text-xs text-yellow-700">
                  Note: Your submission will need to be approved by the collection maintainer.
                </p>
              )}
            </div>
          ) : (
            <p className="text-sm text-slate-600">
              This collection is not currently accepting new submissions.
            </p>
          )}
        </div>
      </div>

      {/* Works in Collection */}
      <div className="bg-white rounded-lg shadow-sm border border-slate-200 p-6">
        <h2 className="text-2xl font-bold text-slate-900 mb-6">Works in This Collection</h2>
        
        {collection.work_count === 0 ? (
          <div className="text-center py-16">
            <div className="text-slate-400 mb-4">
              <svg className="w-16 h-16 mx-auto" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1} d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
              </svg>
            </div>
            <p className="text-slate-600">This collection doesn&apos;t have any works yet.</p>
            {collection.is_open && (
              <p className="text-sm text-slate-500 mt-2">Be the first to submit a work!</p>
            )}
          </div>
        ) : (
          <div className="text-center py-8">
            <p className="text-slate-600 mb-4">
              This collection contains {collection.work_count} work{collection.work_count !== 1 ? 's' : ''}.
            </p>
            <p className="text-sm text-slate-500">
              Collection works listing coming soon!
            </p>
          </div>
        )}
      </div>

      {/* Navigation */}
      <div className="mt-8 text-center">
        <Link 
          href="/collections"
          className="inline-flex items-center text-orange-600 hover:text-orange-700 transition-colors"
        >
          <svg className="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10 19l-7-7m0 0l7-7m-7 7h18" />
          </svg>
          Browse All Collections
        </Link>
      </div>
    </div>
  );
}