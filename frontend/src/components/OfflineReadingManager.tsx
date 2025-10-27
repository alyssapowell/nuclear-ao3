'use client';

import { useState, useEffect } from 'react';
import { 
  CloudArrowDownIcon, 
  WifiIcon, 
  TrashIcon, 
  ClockIcon,
  ExclamationTriangleIcon,
  CheckCircleIcon,
  XCircleIcon
} from '@heroicons/react/24/outline';
import { CloudArrowDownIcon as CloudArrowDownSolidIcon } from '@heroicons/react/24/solid';

interface OfflineWork {
  workId: string;
  title: string;
  authors: Array<{ pseud_name: string; username: string }>;
  consentLevel: 'files_and_pwa' | 'pwa_only' | 'none';
  cachedAt: number;
  expiresAt: number;
  isExpired: boolean;
  word_count?: number;
  chapter_count?: number;
}

interface OfflineReadingManagerProps {
  isOpen: boolean;
  onClose: () => void;
}

export default function OfflineReadingManager({ isOpen, onClose }: OfflineReadingManagerProps) {
  const [offlineWorks, setOfflineWorks] = useState<OfflineWork[]>([]);
  const [isOnline, setIsOnline] = useState(true);
  const [loading, setLoading] = useState(false);
  const [managingWork, setManagingWork] = useState<string | null>(null);
  const [totalCacheSize, setTotalCacheSize] = useState<string>('');

  useEffect(() => {
    // Online/offline detection
    const updateOnlineStatus = () => setIsOnline(navigator.onLine);
    window.addEventListener('online', updateOnlineStatus);
    window.addEventListener('offline', updateOnlineStatus);

    return () => {
      window.removeEventListener('online', updateOnlineStatus);
      window.removeEventListener('offline', updateOnlineStatus);
    };
  }, []);

  useEffect(() => {
    if (isOpen) {
      loadOfflineWorks();
    }
  }, [isOpen]);

  // Load offline works from service worker
  const loadOfflineWorks = async () => {
    if (!('serviceWorker' in navigator)) return;

    setLoading(true);
    try {
      const registration = await navigator.serviceWorker.ready;
      if (!registration.active) return;

      const messageChannel = new MessageChannel();
      
      const response = await new Promise<OfflineWork[]>((resolve) => {
        messageChannel.port1.onmessage = (event) => {
          resolve(event.data.works || []);
        };

        registration.active?.postMessage(
          { type: 'GET_OFFLINE_WORKS' },
          [messageChannel.port2]
        );
      });

      // Get additional work details from localStorage or API
      const enrichedWorks = await Promise.all(
        response.map(async (work) => {
          try {
            // Try to get work details from cache or localStorage
            const workDetails = await getWorkDetails(work.workId);
            return {
              ...work,
              ...workDetails
            };
          } catch (error) {
            console.error('Failed to get work details:', error);
            return {
              ...work,
              title: `Work ${work.workId}`,
              authors: [{ pseud_name: 'Unknown', username: 'unknown' }]
            };
          }
        })
      );

      setOfflineWorks(enrichedWorks);
      
      // Estimate cache size
      estimateCacheSize();

    } catch (error) {
      console.error('[OfflineManager] Failed to load offline works:', error);
    } finally {
      setLoading(false);
    }
  };

  // Get work details from various sources
  const getWorkDetails = async (workId: string): Promise<Partial<OfflineWork>> => {
    // First try localStorage
    const localData = localStorage.getItem(`work-${workId}`);
    if (localData) {
      return JSON.parse(localData);
    }

    // Then try to fetch from cache/API if online
    if (isOnline) {
      try {
        const response = await fetch(`/api/v1/works/${workId}`);
        if (response.ok) {
          const data = await response.json();
          return {
            title: data.title,
            authors: data.authors,
            word_count: data.word_count,
            chapter_count: data.chapter_count
          };
        }
      } catch (error) {
        console.error('Failed to fetch work details:', error);
      }
    }

    return {};
  };

  // Estimate total cache size
  const estimateCacheSize = async () => {
    if ('storage' in navigator && 'estimate' in navigator.storage) {
      try {
        const estimate = await navigator.storage.estimate();
        if (estimate.usage) {
          const sizeInMB = (estimate.usage / (1024 * 1024)).toFixed(1);
          setTotalCacheSize(`~${sizeInMB} MB`);
        }
      } catch (error) {
        console.error('Failed to estimate cache size:', error);
      }
    }
  };

  // Remove specific work from cache
  const removeWorkFromCache = async (workId: string) => {
    if (!('serviceWorker' in navigator)) return;

    setManagingWork(workId);
    try {
      const registration = await navigator.serviceWorker.ready;
      if (!registration.active) return;

      registration.active.postMessage({
        type: 'CLEAR_CONSENT_CACHE',
        data: { workId }
      });

      // Remove from state
      setOfflineWorks(prev => prev.filter(work => work.workId !== workId));

    } catch (error) {
      console.error('[OfflineManager] Failed to remove work:', error);
    } finally {
      setManagingWork(null);
    }
  };

  // Clear all expired works
  const clearExpiredWorks = async () => {
    const expiredWorks = offlineWorks.filter(work => work.isExpired);
    
    for (const work of expiredWorks) {
      await removeWorkFromCache(work.workId);
    }
  };

  // Clear all cached works
  const clearAllCache = async () => {
    if (!confirm('This will remove all offline works. Are you sure?')) return;

    for (const work of offlineWorks) {
      await removeWorkFromCache(work.workId);
    }
  };

  const formatTimeRemaining = (expiresAt: number) => {
    const now = Date.now();
    const remaining = expiresAt - now;

    if (remaining <= 0) return 'Expired';

    const days = Math.floor(remaining / (1000 * 60 * 60 * 24));
    const hours = Math.floor((remaining % (1000 * 60 * 60 * 24)) / (1000 * 60 * 60));

    if (days > 0) return `${days}d ${hours}h`;
    if (hours > 0) return `${hours}h`;
    
    const minutes = Math.floor((remaining % (1000 * 60 * 60)) / (1000 * 60));
    return `${minutes}m`;
  };

  const getConsentLevelInfo = (level: string) => {
    switch (level) {
      case 'files_and_pwa':
        return {
          label: 'Full Offline',
          color: 'text-green-600',
          icon: CheckCircleIcon,
          description: 'Can be cached for extended periods'
        };
      case 'pwa_only':
        return {
          label: 'PWA Only',
          color: 'text-yellow-600',
          icon: ClockIcon,
          description: 'Temporary caching allowed'
        };
      case 'none':
        return {
          label: 'Online Only',
          color: 'text-red-600',
          icon: XCircleIcon,
          description: 'No offline caching allowed'
        };
      default:
        return {
          label: 'Unknown',
          color: 'text-gray-600',
          icon: ExclamationTriangleIcon,
          description: 'Consent level unknown'
        };
    }
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 bg-black/50 flex items-end md:items-center md:justify-center">
      <div className="w-full md:max-w-2xl bg-white rounded-t-2xl md:rounded-2xl max-h-[90vh] flex flex-col">
        {/* Header */}
        <div className="flex items-center justify-between p-6 border-b border-gray-200">
          <div className="flex items-center space-x-3">
            <CloudArrowDownSolidIcon className="w-6 h-6 text-blue-600" />
            <div>
              <h2 className="text-lg font-semibold">Offline Reading</h2>
              <div className="flex items-center space-x-2 text-sm text-gray-500">
                <WifiIcon className={`w-4 h-4 ${isOnline ? 'text-green-500' : 'text-red-500'}`} />
                <span>{isOnline ? 'Online' : 'Offline'}</span>
                {totalCacheSize && <span>• {totalCacheSize} used</span>}
              </div>
            </div>
          </div>
          <button
            onClick={onClose}
            className="text-gray-500 hover:text-gray-700 p-2"
          >
            ✕
          </button>
        </div>

        {/* Content */}
        <div className="flex-1 overflow-y-auto p-6">
          {loading ? (
            <div className="flex items-center justify-center py-12">
              <div className="w-8 h-8 border-2 border-blue-600 border-t-transparent rounded-full animate-spin" />
              <span className="ml-3 text-gray-600">Loading offline works...</span>
            </div>
          ) : offlineWorks.length === 0 ? (
            <div className="text-center py-12 text-gray-500">
              <CloudArrowDownIcon className="w-16 h-16 mx-auto mb-4 opacity-50" />
              <h3 className="text-lg font-medium mb-2">No Offline Works</h3>
              <p className="text-sm">
                Works you cache for offline reading will appear here.
                <br />
                Look for the cache button when reading works that allow it.
              </p>
            </div>
          ) : (
            <div className="space-y-4">
              {/* Cache Management Controls */}
              <div className="flex items-center justify-between p-4 bg-gray-50 rounded-lg">
                <div className="text-sm text-gray-600">
                  {offlineWorks.length} work{offlineWorks.length !== 1 ? 's' : ''} cached
                  {offlineWorks.some(w => w.isExpired) && (
                    <span className="ml-2 text-amber-600">
                      • {offlineWorks.filter(w => w.isExpired).length} expired
                    </span>
                  )}
                </div>
                <div className="flex space-x-2">
                  {offlineWorks.some(w => w.isExpired) && (
                    <button
                      onClick={clearExpiredWorks}
                      className="text-xs px-3 py-1 bg-amber-100 text-amber-700 rounded-md hover:bg-amber-200"
                    >
                      Clear Expired
                    </button>
                  )}
                  <button
                    onClick={clearAllCache}
                    className="text-xs px-3 py-1 bg-red-100 text-red-700 rounded-md hover:bg-red-200"
                  >
                    Clear All
                  </button>
                </div>
              </div>

              {/* Offline Works List */}
              {offlineWorks.map((work) => {
                const consentInfo = getConsentLevelInfo(work.consentLevel);
                const IconComponent = consentInfo.icon;

                return (
                  <div 
                    key={work.workId}
                    className={`p-4 border rounded-lg ${work.isExpired ? 'border-amber-200 bg-amber-50' : 'border-gray-200 hover:border-gray-300'}`}
                  >
                    <div className="flex items-start justify-between">
                      <div className="flex-1 min-w-0">
                        <h3 className="font-medium text-gray-900 truncate">
                          {work.title}
                        </h3>
                        <p className="text-sm text-gray-600 mt-1">
                          by {work.authors.map(a => a.pseud_name).join(', ')}
                        </p>
                        
                        <div className="flex items-center space-x-4 mt-2 text-xs text-gray-500">
                          {work.word_count && (
                            <span>{work.word_count.toLocaleString()} words</span>
                          )}
                          {work.chapter_count && (
                            <span>{work.chapter_count} chapter{work.chapter_count !== 1 ? 's' : ''}</span>
                          )}
                        </div>

                        <div className="flex items-center space-x-4 mt-3">
                          <div className="flex items-center space-x-1">
                            <IconComponent className={`w-4 h-4 ${consentInfo.color}`} />
                            <span className={`text-xs ${consentInfo.color}`}>
                              {consentInfo.label}
                            </span>
                          </div>
                          
                          <div className="flex items-center space-x-1">
                            <ClockIcon className="w-4 h-4 text-gray-400" />
                            <span className={`text-xs ${work.isExpired ? 'text-amber-600' : 'text-gray-500'}`}>
                              {work.isExpired ? 'Expired' : formatTimeRemaining(work.expiresAt)}
                            </span>
                          </div>
                        </div>
                      </div>

                      <button
                        onClick={() => removeWorkFromCache(work.workId)}
                        disabled={managingWork === work.workId}
                        className="ml-4 p-2 text-gray-400 hover:text-red-600 hover:bg-red-50 rounded-md"
                        title="Remove from offline cache"
                      >
                        {managingWork === work.workId ? (
                          <div className="w-4 h-4 border-2 border-gray-400 border-t-transparent rounded-full animate-spin" />
                        ) : (
                          <TrashIcon className="w-4 h-4" />
                        )}
                      </button>
                    </div>

                    {work.isExpired && (
                      <div className="mt-3 p-2 bg-amber-100 border border-amber-200 rounded text-xs text-amber-800">
                        <ExclamationTriangleIcon className="w-4 h-4 inline mr-1" />
                        This work has expired based on the author's consent preferences and may not be available offline.
                      </div>
                    )}
                  </div>
                );
              })}
            </div>
          )}
        </div>

        {/* Footer */}
        <div className="p-6 border-t border-gray-200 bg-gray-50">
          <div className="text-xs text-gray-600 text-center">
            <p className="mb-1">
              <strong>Respectful Offline Reading:</strong> We honor author preferences for offline access.
            </p>
            <p>
              • <strong>Full Offline:</strong> Long-term caching allowed
              • <strong>PWA Only:</strong> Temporary caching for convenience
              • <strong>Online Only:</strong> No offline caching
            </p>
          </div>
        </div>
      </div>
    </div>
  );
}