'use client';

import { useState, useEffect } from 'react';
import { CloudArrowDownIcon, WifiIcon, ExclamationTriangleIcon } from '@heroicons/react/24/outline';
import { CloudArrowDownIcon as CloudArrowDownSolidIcon } from '@heroicons/react/24/solid';
import { useOfflineReading } from '../utils/offlineReadingManager';
import { usePWAState } from '../hooks/usePWAState';

// Test component for demonstrating the respectful caching system
export default function OfflineReadingTest() {
  const [offlineWorks, setOfflineWorks] = useState<any[]>([]);
  const [loading, setLoading] = useState(false);
  const [cacheStats, setCacheStats] = useState<any>(null);
  const [testWork, setTestWork] = useState({
    id: 'test-work-123',
    title: 'Test Fanfiction: Nuclear Hearts',
    authors: [{ pseud_name: 'TestAuthor', username: 'test_author' }],
    summary: 'A test work to demonstrate ethical offline reading capabilities.',
    word_count: 15000,
    chapter_count: 5,
    is_complete: true,
    tags: {
      fandoms: ['Test Fandom'],
      relationships: ['Character A/Character B'],
      characters: ['Character A', 'Character B'],
      freeform_tags: ['Fluff', 'Happy Ending'],
      warnings: ['No Archive Warnings Apply'],
      categories: ['M/M'],
      rating: 'Teen And Up Audiences'
    },
    offline_reading_preference: 'files_and_pwa' as const,
    chapters: [
      {
        id: 'chapter-1',
        number: 1,
        title: 'Chapter 1: The Beginning',
        content: '<p>This is the first chapter of our test work...</p>',
        word_count: 3000
      }
    ]
  });

  const offlineManager = useOfflineReading();
  const [pwaState, pwaActions] = usePWAState();

  // Load offline works and stats
  const loadData = async () => {
    setLoading(true);
    try {
      const [works, stats] = await Promise.all([
        offlineManager.getOfflineWorks(),
        offlineManager.getCacheStats()
      ]);
      
      setOfflineWorks(works);
      setCacheStats(stats);
    } catch (error) {
      console.error('Failed to load offline data:', error);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadData();
  }, []);

  // Test caching a work
  const handleCacheWork = async () => {
    setLoading(true);
    try {
      const success = await offlineManager.cacheWork(
        testWork.id,
        testWork,
        testWork.offline_reading_preference
      );
      
      if (success) {
        console.log('Test work cached successfully');
        // Wait a moment for service worker to process
        setTimeout(loadData, 1000);
      } else {
        alert('Failed to cache test work');
      }
    } catch (error) {
      console.error('Error caching work:', error);
      alert('Error caching work');
    } finally {
      setLoading(false);
    }
  };

  // Test removing a work
  const handleRemoveWork = async (workId: string) => {
    setLoading(true);
    try {
      const success = await offlineManager.removeWork(workId);
      if (success) {
        console.log('Work removed successfully');
        setTimeout(loadData, 1000);
      }
    } catch (error) {
      console.error('Error removing work:', error);
    } finally {
      setLoading(false);
    }
  };

  // Clear all cached works
  const handleClearAll = async () => {
    if (!confirm('Clear all cached works?')) return;
    
    setLoading(true);
    try {
      await offlineManager.clearAllWorks();
      setTimeout(loadData, 1000);
    } catch (error) {
      console.error('Error clearing works:', error);
    } finally {
      setLoading(false);
    }
  };

  const isWorkCached = offlineWorks.some(w => w.workId === testWork.id);

  return (
    <div className="max-w-4xl mx-auto p-6 space-y-6">
      <div className="bg-white rounded-lg shadow-lg p-6">
        <h1 className="text-2xl font-bold text-gray-900 mb-4">
          Nuclear AO3 - Respectful Caching Test
        </h1>
        
        {/* PWA Status */}
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-6">
          <div className="bg-gray-50 p-4 rounded-lg">
            <div className="flex items-center space-x-2">
              <WifiIcon className={`w-5 h-5 ${pwaState.isOnline ? 'text-green-600' : 'text-red-600'}`} />
              <span className="font-medium">
                {pwaState.isOnline ? 'Online' : 'Offline'}
              </span>
            </div>
          </div>
          
          <div className="bg-gray-50 p-4 rounded-lg">
            <div className="flex items-center space-x-2">
              <CloudArrowDownIcon className={`w-5 h-5 ${pwaState.serviceWorkerReady ? 'text-green-600' : 'text-gray-400'}`} />
              <span className="font-medium">
                Service Worker: {pwaState.serviceWorkerReady ? 'Ready' : 'Not Ready'}
              </span>
            </div>
          </div>
          
          <div className="bg-gray-50 p-4 rounded-lg">
            <div className="flex items-center space-x-2">
              <CloudArrowDownIcon className={`w-5 h-5 ${pwaState.isInstalled ? 'text-blue-600' : 'text-gray-400'}`} />
              <span className="font-medium">
                PWA: {pwaState.isInstalled ? 'Installed' : 'Not Installed'}
              </span>
            </div>
          </div>
        </div>

        {/* Cache Stats */}
        {cacheStats && (
          <div className="bg-blue-50 p-4 rounded-lg mb-6">
            <h3 className="font-medium text-blue-900 mb-2">Cache Statistics</h3>
            <div className="grid grid-cols-2 md:grid-cols-4 gap-4 text-sm">
              <div>
                <span className="text-blue-700">Total Works:</span>
                <span className="ml-2 font-medium">{cacheStats.totalWorks}</span>
              </div>
              <div>
                <span className="text-blue-700">Expired:</span>
                <span className="ml-2 font-medium">{cacheStats.expiredWorks}</span>
              </div>
              <div>
                <span className="text-blue-700">Storage Used:</span>
                <span className="ml-2 font-medium">{cacheStats.storageUsed}</span>
              </div>
              <div>
                <span className="text-blue-700">Utilization:</span>
                <span className="ml-2 font-medium">{cacheStats.utilizationPercentage}%</span>
              </div>
            </div>
          </div>
        )}

        {/* Test Work */}
        <div className="border rounded-lg p-4 mb-6">
          <h3 className="font-medium text-gray-900 mb-2">Test Work</h3>
          <div className="flex items-start justify-between">
            <div>
              <h4 className="font-medium">{testWork.title}</h4>
              <p className="text-sm text-gray-600">
                by {testWork.authors.map(a => a.pseud_name).join(', ')}
              </p>
              <p className="text-xs text-gray-500 mt-1">
                {testWork.word_count.toLocaleString()} words • {testWork.chapter_count} chapters
              </p>
              <div className="flex items-center space-x-2 mt-2">
                <span className="text-xs px-2 py-1 bg-green-100 text-green-800 rounded">
                  📁 Full Offline Access
                </span>
                {isWorkCached && (
                  <span className="text-xs px-2 py-1 bg-blue-100 text-blue-800 rounded">
                    📱 Cached
                  </span>
                )}
              </div>
            </div>
            
            <div className="flex space-x-2">
              {isWorkCached ? (
                <button
                  onClick={() => handleRemoveWork(testWork.id)}
                  disabled={loading}
                  className="px-3 py-1 text-sm bg-red-100 text-red-700 rounded hover:bg-red-200 disabled:opacity-50"
                >
                  Remove
                </button>
              ) : (
                <button
                  onClick={handleCacheWork}
                  disabled={loading}
                  className="px-3 py-1 text-sm bg-blue-100 text-blue-700 rounded hover:bg-blue-200 disabled:opacity-50"
                >
                  Cache for Offline
                </button>
              )}
            </div>
          </div>
        </div>

        {/* Cached Works List */}
        <div className="space-y-4">
          <div className="flex items-center justify-between">
            <h3 className="font-medium text-gray-900">Cached Works ({offlineWorks.length})</h3>
            {offlineWorks.length > 0 && (
              <button
                onClick={handleClearAll}
                disabled={loading}
                className="px-3 py-1 text-sm bg-red-100 text-red-700 rounded hover:bg-red-200 disabled:opacity-50"
              >
                Clear All
              </button>
            )}
          </div>

          {loading ? (
            <div className="flex items-center justify-center py-8">
              <div className="w-8 h-8 border-2 border-blue-600 border-t-transparent rounded-full animate-spin" />
              <span className="ml-3 text-gray-600">Loading...</span>
            </div>
          ) : offlineWorks.length === 0 ? (
            <div className="text-center py-8 text-gray-500">
              <CloudArrowDownIcon className="w-16 h-16 mx-auto mb-4 opacity-50" />
              <p>No works cached for offline reading</p>
              <p className="text-sm">Cache the test work above to see it here</p>
            </div>
          ) : (
            <div className="space-y-3">
              {offlineWorks.map((work) => (
                <div 
                  key={work.workId}
                  className={`p-4 border rounded-lg ${work.isExpired ? 'border-amber-200 bg-amber-50' : 'border-gray-200'}`}
                >
                  <div className="flex items-start justify-between">
                    <div>
                      <h4 className="font-medium">{work.title || `Work ${work.workId}`}</h4>
                      {work.authors && (
                        <p className="text-sm text-gray-600">
                          by {work.authors.map((a: any) => a.pseud_name).join(', ')}
                        </p>
                      )}
                      <div className="flex items-center space-x-2 mt-2">
                        <span className={`text-xs px-2 py-1 rounded ${
                          work.consentLevel === 'files_and_pwa' 
                            ? 'bg-green-100 text-green-800' 
                            : 'bg-yellow-100 text-yellow-800'
                        }`}>
                          {work.consentLevel === 'files_and_pwa' ? '📁 Full Offline' : '📱 PWA Only'}
                        </span>
                        <span className="text-xs text-gray-500">
                          Expires: {offlineManager.formatTimeRemaining(work.expiresAt)}
                        </span>
                        {work.isExpired && (
                          <span className="text-xs px-2 py-1 bg-red-100 text-red-800 rounded">
                            ⚠️ Expired
                          </span>
                        )}
                      </div>
                    </div>
                    
                    <button
                      onClick={() => handleRemoveWork(work.workId)}
                      disabled={loading}
                      className="px-3 py-1 text-sm bg-red-100 text-red-700 rounded hover:bg-red-200 disabled:opacity-50"
                    >
                      Remove
                    </button>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>

        {/* Install PWA Button */}
        {pwaState.isInstallable && !pwaState.isInstalled && (
          <div className="mt-6 p-4 bg-blue-50 border border-blue-200 rounded-lg">
            <div className="flex items-center justify-between">
              <div>
                <h3 className="font-medium text-blue-900">Install Nuclear AO3</h3>
                <p className="text-sm text-blue-700">
                  Install as an app for better offline reading experience
                </p>
              </div>
              <button
                onClick={pwaActions.installPWA}
                className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700"
              >
                Install
              </button>
            </div>
          </div>
        )}

        {/* Warning about service worker */}
        {!pwaState.serviceWorkerReady && (
          <div className="mt-6 p-4 bg-amber-50 border border-amber-200 rounded-lg">
            <div className="flex items-start space-x-3">
              <ExclamationTriangleIcon className="w-5 h-5 text-amber-600 mt-0.5" />
              <div>
                <h3 className="font-medium text-amber-900">Service Worker Not Ready</h3>
                <p className="text-sm text-amber-700">
                  The service worker is required for offline functionality. Please refresh the page or check your browser settings.
                </p>
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}