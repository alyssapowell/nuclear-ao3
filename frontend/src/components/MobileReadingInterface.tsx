'use client';

import { useState, useEffect, useRef } from 'react';
import { ChevronLeftIcon, ChevronRightIcon, BookmarkIcon, ShareIcon, AdjustmentsHorizontalIcon, HeartIcon, CloudArrowDownIcon, WifiIcon } from '@heroicons/react/24/outline';
import { BookmarkIcon as BookmarkSolidIcon, HeartIcon as HeartSolidIcon, CloudArrowDownIcon as CloudArrowDownSolidIcon } from '@heroicons/react/24/solid';

interface MobileReadingInterfaceProps {
  work: {
    id: string;
    title: string;
    authors: Array<{ pseud_name: string; username: string }>;
    summary: string;
    word_count: number;
    chapter_count: number;
    is_complete: boolean;
    tags: {
      fandoms: string[];
      relationships: string[];
      characters: string[];
      freeform_tags: string[];
      warnings: string[];
      categories: string[];
      rating: string;
    };
    offline_reading_preference?: 'files_and_pwa' | 'pwa_only' | 'none';
  };
  chapters: Array<{
    id: string;
    number: number;
    title: string;
    content: string;
    word_count: number;
    notes?: string;
    end_notes?: string;
  }>;
  currentChapter: number;
  onChapterChange: (chapterNumber: number) => void;
  isBookmarked?: boolean;
  isKudosed?: boolean;
  onBookmark?: () => void;
  onKudos?: () => void;
  onShare?: () => void;
}

interface ReadingSettings {
  fontSize: 'small' | 'medium' | 'large' | 'extra-large';
  fontFamily: 'serif' | 'sans-serif' | 'dyslexia-friendly';
  theme: 'light' | 'dark' | 'sepia' | 'high-contrast';
  lineHeight: 'compact' | 'normal' | 'relaxed';
  textAlign: 'left' | 'justify';
  margins: 'narrow' | 'normal' | 'wide';
}

interface OfflineWorkStatus {
  workId: string;
  consentLevel: 'files_and_pwa' | 'pwa_only' | 'none';
  cachedAt: number;
  expiresAt: number;
  isExpired: boolean;
}

export default function MobileReadingInterface({
  work,
  chapters,
  currentChapter,
  onChapterChange,
  isBookmarked = false,
  isKudosed = false,
  onBookmark,
  onKudos,
  onShare
}: MobileReadingInterfaceProps) {
  const [showControls, setShowControls] = useState(true);
  const [showSettings, setShowSettings] = useState(false);
  const [readingProgress, setReadingProgress] = useState(0);
  const [readingSettings, setReadingSettings] = useState<ReadingSettings>({
    fontSize: 'medium',
    fontFamily: 'serif',
    theme: 'light',
    lineHeight: 'normal',
    textAlign: 'left',
    margins: 'normal'
  });

  // PWA and offline state
  const [isOnline, setIsOnline] = useState(true);
  const [isWorkCached, setIsWorkCached] = useState(false);
  const [cachingInProgress, setCachingInProgress] = useState(false);
  const [offlineStatus, setOfflineStatus] = useState<OfflineWorkStatus | null>(null);
  const [serviceWorkerReady, setServiceWorkerReady] = useState(false);

  const contentRef = useRef<HTMLDivElement>(null);
  const controlsTimeoutRef = useRef<NodeJS.Timeout>();

  const currentChapterData = chapters.find(c => c.number === currentChapter);

  // Initialize service worker communication and check online status
  useEffect(() => {
    // Online/offline detection
    const updateOnlineStatus = () => setIsOnline(navigator.onLine);
    window.addEventListener('online', updateOnlineStatus);
    window.addEventListener('offline', updateOnlineStatus);

    // Service worker registration and communication
    const initServiceWorker = async () => {
      if ('serviceWorker' in navigator) {
        try {
          const registration = await navigator.serviceWorker.ready;
          setServiceWorkerReady(true);

          // Listen for service worker messages
          navigator.serviceWorker.addEventListener('message', handleServiceWorkerMessage);

          // Check if this work is already cached
          await checkWorkCacheStatus();

        } catch (error) {
          console.error('[PWA] Service worker initialization failed:', error);
        }
      }
    };

    initServiceWorker();

    return () => {
      window.removeEventListener('online', updateOnlineStatus);
      window.removeEventListener('offline', updateOnlineStatus);
      if ('serviceWorker' in navigator) {
        navigator.serviceWorker.removeEventListener('message', handleServiceWorkerMessage);
      }
    };
  }, [work.id]);

  // Handle messages from service worker
  const handleServiceWorkerMessage = (event: MessageEvent) => {
    const { type, workId, consentLevel } = event.data;

    switch (type) {
      case 'WORK_CACHED_WITH_CONSENT':
        if (workId === work.id) {
          setIsWorkCached(true);
          setCachingInProgress(false);
          console.log('[PWA] Work cached successfully with consent level:', consentLevel);
        }
        break;

      case 'WORK_DELETED':
        if (workId === work.id) {
          setIsWorkCached(false);
          setOfflineStatus(null);
          console.log('[PWA] Work removed from cache (author deletion)');
        }
        break;

      default:
        break;
    }
  };

  // Check if current work is cached
  const checkWorkCacheStatus = async () => {
    if (!serviceWorkerReady || !navigator.serviceWorker.controller) return;

    try {
      const messageChannel = new MessageChannel();
      
      const response = await new Promise<OfflineWorkStatus[]>((resolve) => {
        messageChannel.port1.onmessage = (event) => {
          resolve(event.data.works || []);
        };

        navigator.serviceWorker.controller?.postMessage(
          { type: 'GET_OFFLINE_WORKS' },
          [messageChannel.port2]
        );
      });

      const workStatus = response.find(w => w.workId === work.id);
      if (workStatus) {
        setIsWorkCached(true);
        setOfflineStatus(workStatus);
      }

    } catch (error) {
      console.error('[PWA] Failed to check work cache status:', error);
    }
  };

  // Cache work for offline reading (respecting author consent)
  const cacheWorkForOffline = async () => {
    if (!serviceWorkerReady || !navigator.serviceWorker.controller) {
      console.log('[PWA] Service worker not ready');
      return;
    }

    if (!work.offline_reading_preference || work.offline_reading_preference === 'none') {
      alert('This author has chosen not to allow offline reading. We respect this preference.');
      return;
    }

    setCachingInProgress(true);

    try {
      // Prepare work data for caching
      const workData = {
        id: work.id,
        title: work.title,
        authors: work.authors,
        summary: work.summary,
        word_count: work.word_count,
        chapter_count: work.chapter_count,
        is_complete: work.is_complete,
        tags: work.tags,
        chapters: chapters,
        offline_reading_preference: work.offline_reading_preference
      };

      // Send caching request to service worker
      navigator.serviceWorker.controller.postMessage({
        type: 'CACHE_WORK_WITH_CONSENT',
        data: {
          workId: work.id,
          workData: workData,
          consentLevel: work.offline_reading_preference
        }
      });

      console.log('[PWA] Caching request sent for work:', work.id);

    } catch (error) {
      console.error('[PWA] Failed to cache work:', error);
      setCachingInProgress(false);
      alert('Failed to cache work for offline reading. Please try again.');
    }
  };

  // Remove work from offline cache
  const removeWorkFromCache = async () => {
    if (!serviceWorkerReady || !navigator.serviceWorker.controller) return;

    try {
      navigator.serviceWorker.controller.postMessage({
        type: 'CLEAR_CONSENT_CACHE',
        data: { workId: work.id }
      });

      setIsWorkCached(false);
      setOfflineStatus(null);
      console.log('[PWA] Work removed from offline cache');

    } catch (error) {
      console.error('[PWA] Failed to remove work from cache:', error);
    }
  };

  // Auto-hide controls after inactivity
  useEffect(() => {
    const resetControlsTimeout = () => {
      if (controlsTimeoutRef.current) {
        clearTimeout(controlsTimeoutRef.current);
      }
      setShowControls(true);
      controlsTimeoutRef.current = setTimeout(() => {
        setShowControls(false);
      }, 3000);
    };

    const handleUserActivity = () => {
      resetControlsTimeout();
    };

    document.addEventListener('touchstart', handleUserActivity);
    document.addEventListener('scroll', handleUserActivity);

    resetControlsTimeout();

    return () => {
      document.removeEventListener('touchstart', handleUserActivity);
      document.removeEventListener('scroll', handleUserActivity);
      if (controlsTimeoutRef.current) {
        clearTimeout(controlsTimeoutRef.current);
      }
    };
  }, []);

  // Track reading progress
  useEffect(() => {
    const handleScroll = () => {
      if (!contentRef.current) return;
      
      const element = contentRef.current;
      const scrollTop = window.scrollY;
      const scrollHeight = element.scrollHeight - window.innerHeight;
      const progress = Math.min(Math.max(scrollTop / scrollHeight, 0), 1) * 100;
      
      setReadingProgress(progress);
    };

    window.addEventListener('scroll', handleScroll);
    return () => window.removeEventListener('scroll', handleScroll);
  }, [currentChapter]);

  // Load reading settings from localStorage
  useEffect(() => {
    const savedSettings = localStorage.getItem('nuclear-ao3-reading-settings');
    if (savedSettings) {
      setReadingSettings(JSON.parse(savedSettings));
    }
  }, []);

  // Save reading settings to localStorage
  useEffect(() => {
    localStorage.setItem('nuclear-ao3-reading-settings', JSON.stringify(readingSettings));
  }, [readingSettings]);

  const updateSetting = <K extends keyof ReadingSettings>(key: K, value: ReadingSettings[K]) => {
    setReadingSettings(prev => ({ ...prev, [key]: value }));
  };

  const getThemeClasses = () => {
    const baseClasses = 'min-h-screen transition-colors duration-300';
    switch (readingSettings.theme) {
      case 'dark':
        return `${baseClasses} bg-gray-900 text-gray-100`;
      case 'sepia':
        return `${baseClasses} bg-amber-50 text-amber-900`;
      case 'high-contrast':
        return `${baseClasses} bg-black text-white`;
      default:
        return `${baseClasses} bg-white text-gray-900`;
    }
  };

  const getFontClasses = () => {
    const sizeClasses = {
      'small': 'text-sm',
      'medium': 'text-base',
      'large': 'text-lg',
      'extra-large': 'text-xl'
    };

    const familyClasses = {
      'serif': 'font-serif',
      'sans-serif': 'font-sans',
      'dyslexia-friendly': 'font-mono' // In real app, would use OpenDyslexic
    };

    const lineHeightClasses = {
      'compact': 'leading-tight',
      'normal': 'leading-normal',
      'relaxed': 'leading-relaxed'
    };

    const alignClasses = {
      'left': 'text-left',
      'justify': 'text-justify'
    };

    const marginClasses = {
      'narrow': 'px-3',
      'normal': 'px-4',
      'wide': 'px-6'
    };

    return `${sizeClasses[readingSettings.fontSize]} ${familyClasses[readingSettings.fontFamily]} ${lineHeightClasses[readingSettings.lineHeight]} ${alignClasses[readingSettings.textAlign]} ${marginClasses[readingSettings.margins]}`;
  };

  const handlePreviousChapter = () => {
    if (currentChapter > 1) {
      onChapterChange(currentChapter - 1);
      window.scrollTo(0, 0);
    }
  };

  const handleNextChapter = () => {
    if (currentChapter < work.chapter_count) {
      onChapterChange(currentChapter + 1);
      window.scrollTo(0, 0);
    }
  };

  const getOfflineStatusMessage = () => {
    if (!isOnline) {
      return isWorkCached ? '📱 Cached (offline)' : '🌐 Needs connection';
    }

    switch (work.offline_reading_preference) {
      case 'files_and_pwa':
        return isWorkCached ? '📁 Cached for offline' : '📁 Available offline';
      case 'pwa_only':
        return isWorkCached ? '📱 Temporarily cached' : '📱 PWA offline only';
      case 'none':
        return '🌐 Online only';
      default:
        return '';
    }
  };

  const getConnectionStatusColor = () => {
    if (!isOnline) return 'text-red-400';
    if (isWorkCached) return 'text-green-400';
    return 'text-yellow-400';
  };

  return (
    <div className={getThemeClasses()}>
      {/* Progress Bar */}
      <div className="fixed top-0 left-0 right-0 z-50">
        <div className="h-1 bg-gray-200">
          <div 
            className="h-full bg-blue-600 transition-all duration-300"
            style={{ width: `${readingProgress}%` }}
          />
        </div>
      </div>

      {/* Top Controls */}
      <div className={`fixed top-1 left-0 right-0 z-40 transition-opacity duration-300 ${showControls ? 'opacity-100' : 'opacity-0 pointer-events-none'}`}>
        <div className="flex items-center justify-between p-4 bg-gradient-to-b from-black/50 to-transparent">
          <button
            onClick={() => window.history.back()}
            className="flex items-center space-x-2 text-white bg-black/30 rounded-full px-3 py-2"
          >
            <ChevronLeftIcon className="w-5 h-5" />
            <span className="text-sm">Back</span>
          </button>
          
          <div className="text-center text-white">
            <div className="text-sm font-medium truncate max-w-48">
              {work.title}
            </div>
            <div className="text-xs opacity-75">
              Chapter {currentChapter} of {work.chapter_count}
            </div>
          </div>

          <button
            onClick={() => setShowSettings(true)}
            className="text-white bg-black/30 rounded-full p-2"
          >
            <AdjustmentsHorizontalIcon className="w-5 h-5" />
          </button>
        </div>
      </div>

      {/* Main Content */}
      <div ref={contentRef} className="pt-16 pb-20">
        {/* Chapter Header */}
        <div className={`${getFontClasses()} py-6 border-b border-gray-200`}>
          <h1 className="text-xl font-bold mb-2">
            {currentChapterData?.title || `Chapter ${currentChapter}`}
          </h1>
          
          {currentChapterData?.notes && (
            <div className="text-sm opacity-75 mb-4 p-3 bg-gray-100 rounded-lg">
              <div className="font-medium mb-1">Chapter Notes:</div>
              <div dangerouslySetInnerHTML={{ __html: currentChapterData.notes }} />
            </div>
          )}

          <div className="flex items-center justify-between text-xs">
            <span className="opacity-60">{currentChapterData?.word_count?.toLocaleString()} words</span>
            <div className="flex items-center space-x-2">
              <WifiIcon className={`w-3 h-3 ${isOnline ? 'text-green-400' : 'text-red-400'}`} />
              <span className={getConnectionStatusColor()}>{getOfflineStatusMessage()}</span>
            </div>
          </div>
        </div>

        {/* Chapter Content */}
        <div className={`${getFontClasses()} py-6`}>
          <div 
            className="prose prose-sm max-w-none"
            dangerouslySetInnerHTML={{ 
              __html: currentChapterData?.content || 'Chapter content loading...' 
            }}
          />
        </div>

        {/* Chapter End Notes */}
        {currentChapterData?.end_notes && (
          <div className={`${getFontClasses()} py-4 border-t border-gray-200`}>
            <div className="text-sm opacity-75 p-3 bg-gray-100 rounded-lg">
              <div className="font-medium mb-1">End Notes:</div>
              <div dangerouslySetInnerHTML={{ __html: currentChapterData.end_notes }} />
            </div>
          </div>
        )}
      </div>

      {/* Bottom Controls */}
      <div className={`fixed bottom-0 left-0 right-0 z-40 transition-opacity duration-300 ${showControls ? 'opacity-100' : 'opacity-0 pointer-events-none'}`}>
        <div className="bg-gradient-to-t from-black/50 to-transparent p-4">
          {/* Action Buttons */}
          <div className="flex items-center justify-center space-x-4 mb-4">
            <button
              onClick={onBookmark}
              className="flex flex-col items-center space-y-1 text-white"
            >
              {isBookmarked ? (
                <BookmarkSolidIcon className="w-6 h-6 text-yellow-400" />
              ) : (
                <BookmarkIcon className="w-6 h-6" />
              )}
              <span className="text-xs">Bookmark</span>
            </button>

            <button
              onClick={onKudos}
              className="flex flex-col items-center space-y-1 text-white"
            >
              {isKudosed ? (
                <HeartSolidIcon className="w-6 h-6 text-red-400" />
              ) : (
                <HeartIcon className="w-6 h-6" />
              )}
              <span className="text-xs">Kudos</span>
            </button>

            {/* Offline Reading Control */}
            {work.offline_reading_preference && work.offline_reading_preference !== 'none' && (
              <button
                onClick={isWorkCached ? removeWorkFromCache : cacheWorkForOffline}
                disabled={cachingInProgress}
                className="flex flex-col items-center space-y-1 text-white"
              >
                {cachingInProgress ? (
                  <div className="w-6 h-6 border-2 border-white border-t-transparent rounded-full animate-spin" />
                ) : isWorkCached ? (
                  <CloudArrowDownSolidIcon className="w-6 h-6 text-green-400" />
                ) : (
                  <CloudArrowDownIcon className="w-6 h-6" />
                )}
                <span className="text-xs">
                  {cachingInProgress ? 'Caching...' : isWorkCached ? 'Cached' : 'Cache'}
                </span>
              </button>
            )}

            <button
              onClick={onShare}
              className="flex flex-col items-center space-y-1 text-white"
            >
              <ShareIcon className="w-6 h-6" />
              <span className="text-xs">Share</span>
            </button>
          </div>

          {/* Chapter Navigation */}
          <div className="flex items-center justify-between">
            <button
              onClick={handlePreviousChapter}
              disabled={currentChapter <= 1}
              className="flex items-center space-x-2 text-white bg-black/30 rounded-full px-4 py-2 disabled:opacity-30"
            >
              <ChevronLeftIcon className="w-4 h-4" />
              <span className="text-sm">Previous</span>
            </button>

            <div className="text-white text-center">
              <div className="text-xs opacity-75">Chapter</div>
              <div className="text-sm font-medium">
                {currentChapter} / {work.chapter_count}
              </div>
            </div>

            <button
              onClick={handleNextChapter}
              disabled={currentChapter >= work.chapter_count}
              className="flex items-center space-x-2 text-white bg-black/30 rounded-full px-4 py-2 disabled:opacity-30"
            >
              <span className="text-sm">Next</span>
              <ChevronRightIcon className="w-4 h-4" />
            </button>
          </div>
        </div>
      </div>

      {/* Reading Settings Modal */}
      {showSettings && (
        <div className="fixed inset-0 z-50 bg-black/50 flex items-end">
          <div className="w-full bg-white rounded-t-2xl p-6 max-h-[80vh] overflow-y-auto">
            <div className="flex items-center justify-between mb-6">
              <h2 className="text-lg font-semibold">Reading Settings</h2>
              <button
                onClick={() => setShowSettings(false)}
                className="text-gray-500 p-2"
              >
                ✕
              </button>
            </div>

            {/* Font Size */}
            <div className="mb-6">
              <label className="block text-sm font-medium mb-3">Font Size</label>
              <div className="grid grid-cols-4 gap-2">
                {(['small', 'medium', 'large', 'extra-large'] as const).map(size => (
                  <button
                    key={size}
                    onClick={() => updateSetting('fontSize', size)}
                    className={`p-3 rounded-lg border text-center capitalize ${
                      readingSettings.fontSize === size
                        ? 'border-blue-500 bg-blue-50 text-blue-700'
                        : 'border-gray-200'
                    }`}
                  >
                    {size.replace('-', ' ')}
                  </button>
                ))}
              </div>
            </div>

            {/* Font Family */}
            <div className="mb-6">
              <label className="block text-sm font-medium mb-3">Font Family</label>
              <div className="grid grid-cols-3 gap-2">
                {(['serif', 'sans-serif', 'dyslexia-friendly'] as const).map(family => (
                  <button
                    key={family}
                    onClick={() => updateSetting('fontFamily', family)}
                    className={`p-3 rounded-lg border text-center ${
                      readingSettings.fontFamily === family
                        ? 'border-blue-500 bg-blue-50 text-blue-700'
                        : 'border-gray-200'
                    }`}
                  >
                    {family.replace('-', ' ')}
                  </button>
                ))}
              </div>
            </div>

            {/* Theme */}
            <div className="mb-6">
              <label className="block text-sm font-medium mb-3">Theme</label>
              <div className="grid grid-cols-2 gap-2">
                {(['light', 'dark', 'sepia', 'high-contrast'] as const).map(theme => (
                  <button
                    key={theme}
                    onClick={() => updateSetting('theme', theme)}
                    className={`p-3 rounded-lg border text-center capitalize ${
                      readingSettings.theme === theme
                        ? 'border-blue-500 bg-blue-50 text-blue-700'
                        : 'border-gray-200'
                    }`}
                  >
                    {theme.replace('-', ' ')}
                  </button>
                ))}
              </div>
            </div>

            {/* Line Height */}
            <div className="mb-6">
              <label className="block text-sm font-medium mb-3">Line Height</label>
              <div className="grid grid-cols-3 gap-2">
                {(['compact', 'normal', 'relaxed'] as const).map(height => (
                  <button
                    key={height}
                    onClick={() => updateSetting('lineHeight', height)}
                    className={`p-3 rounded-lg border text-center capitalize ${
                      readingSettings.lineHeight === height
                        ? 'border-blue-500 bg-blue-50 text-blue-700'
                        : 'border-gray-200'
                    }`}
                  >
                    {height}
                  </button>
                ))}
              </div>
            </div>

            {/* Additional Settings */}
            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium mb-3">Text Alignment</label>
                <div className="grid grid-cols-2 gap-2">
                  {(['left', 'justify'] as const).map(align => (
                    <button
                      key={align}
                      onClick={() => updateSetting('textAlign', align)}
                      className={`p-3 rounded-lg border text-center capitalize ${
                        readingSettings.textAlign === align
                          ? 'border-blue-500 bg-blue-50 text-blue-700'
                          : 'border-gray-200'
                      }`}
                    >
                      {align}
                    </button>
                  ))}
                </div>
              </div>

              <div>
                <label className="block text-sm font-medium mb-3">Margins</label>
                <div className="grid grid-cols-3 gap-2">
                  {(['narrow', 'normal', 'wide'] as const).map(margin => (
                    <button
                      key={margin}
                      onClick={() => updateSetting('margins', margin)}
                      className={`p-3 rounded-lg border text-center capitalize ${
                        readingSettings.margins === margin
                          ? 'border-blue-500 bg-blue-50 text-blue-700'
                          : 'border-gray-200'
                      }`}
                    >
                      {margin}
                    </button>
                  ))}
                </div>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}