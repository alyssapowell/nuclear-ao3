'use client';

import { useState, useEffect } from 'react';
import { offlineReadingManager } from '@/utils/offlineReadingManager';

export default function PWATestPage() {
  const [swStatus, setSWStatus] = useState('Checking...');
  const [cacheStatus, setCacheStatus] = useState('Not tested');
  const [offlineWorks, setOfflineWorks] = useState<any[]>([]);
  const [isOnline, setIsOnline] = useState(true);
  const [testResults, setTestResults] = useState<string[]>([]);
  const [consentLevel, setConsentLevel] = useState('none');
  const [ttlDisplay, setTTLDisplay] = useState('None');

  // Mock work data for testing
  const mockWork = {
    id: 'test-work-123',
    title: 'Test Fanfiction',
    authors: [{ pseud_name: 'TestAuthor', username: 'testuser' }],
    word_count: 5000,
    chapter_count: 1,
    summary: 'A test work for demonstrating consent-aware caching.',
    rating: 'General',
    fandoms: ['Test Fandom']
  };

  useEffect(() => {
    checkServiceWorkerStatus();
    checkOnlineStatus();
    updateConsentDisplay();
    
    // Listen for online/offline events
    const cleanup = offlineReadingManager.onConnectionChange((online) => {
      setIsOnline(online);
      addTestResult(`Network status changed: ${online ? 'Online' : 'Offline'}`);
    });

    return cleanup;
  }, []);

  const updateConsentDisplay = () => {
    const stored = localStorage.getItem('offline-reading-consent-level') || 'none';
    setConsentLevel(stored);
    
    const ttlMapping = {
      'full': '30 days',
      'minimal': '24 hours', 
      'none': 'None'
    };
    setTTLDisplay(ttlMapping[stored as keyof typeof ttlMapping] || 'None');
  };

  const handleConsentChange = (newLevel: string) => {
    localStorage.setItem('offline-reading-consent-level', newLevel);
    localStorage.setItem('offline-reading-consent-timestamp', Date.now().toString());
    setConsentLevel(newLevel);
    updateConsentDisplay();
    
    // Notify service worker of consent change
    if ('serviceWorker' in navigator && navigator.serviceWorker.controller) {
      navigator.serviceWorker.controller.postMessage({
        type: 'CONSENT_LEVEL_CHANGED',
        level: newLevel
      });
    }
    
    addTestResult(`Consent level changed to: ${newLevel}`);
  };

  const addTestResult = (result: string) => {
    setTestResults(prev => [...prev, `${new Date().toLocaleTimeString()}: ${result}`]);
  };

  const checkServiceWorkerStatus = async () => {
    try {
      if ('serviceWorker' in navigator) {
        const registration = await navigator.serviceWorker.ready;
        setSWStatus(`Active: ${registration.active?.scriptURL || 'Unknown'}`);
        addTestResult('Service Worker is active and ready');
        
        // Check if our consent-aware service worker is loaded
        if (registration.active?.scriptURL.includes('sw-consent-aware.js')) {
          addTestResult('✅ Consent-aware service worker detected');
        } else {
          addTestResult('⚠️ Standard service worker detected (not consent-aware)');
        }
      } else {
        setSWStatus('Not supported');
        addTestResult('❌ Service Worker not supported');
      }
    } catch (error) {
      setSWStatus('Error checking');
      addTestResult(`❌ Service Worker check failed: ${error}`);
    }
  };

  const checkOnlineStatus = () => {
    setIsOnline(navigator.onLine);
    addTestResult(`Initial network status: ${navigator.onLine ? 'Online' : 'Offline'}`);
  };

  const testCacheWork = async (consentLevel: 'files_and_pwa' | 'pwa_only') => {
    try {
      addTestResult(`Testing cache with consent level: ${consentLevel}`);
      const success = await offlineReadingManager.cacheWork(
        `${mockWork.id}-${consentLevel}`,
        mockWork,
        consentLevel
      );
      
      if (success) {
        setCacheStatus(`✅ Work cached with ${consentLevel} consent`);
        addTestResult(`✅ Successfully cached work with ${consentLevel} consent`);
      } else {
        setCacheStatus(`❌ Failed to cache work with ${consentLevel} consent`);
        addTestResult(`❌ Failed to cache work with ${consentLevel} consent`);
      }
    } catch (error) {
      setCacheStatus(`❌ Error: ${error}`);
      addTestResult(`❌ Cache error: ${error}`);
    }
  };

  const testGetOfflineWorks = async () => {
    try {
      addTestResult('Getting offline works...');
      const works = await offlineReadingManager.getOfflineWorks();
      setOfflineWorks(works);
      addTestResult(`✅ Retrieved ${works.length} offline works`);
    } catch (error) {
      addTestResult(`❌ Failed to get offline works: ${error}`);
    }
  };

  const testCacheStats = async () => {
    try {
      addTestResult('Getting cache statistics...');
      const stats = await offlineReadingManager.getCacheStats();
      addTestResult(`📊 Cache stats: ${stats.totalWorks} works, ${stats.storageUsed} used`);
    } catch (error) {
      addTestResult(`❌ Failed to get cache stats: ${error}`);
    }
  };

  const testRemoveWork = async (workId: string) => {
    try {
      addTestResult(`Removing work: ${workId}`);
      const success = await offlineReadingManager.removeWork(workId);
      if (success) {
        addTestResult(`✅ Successfully removed work: ${workId}`);
        testGetOfflineWorks(); // Refresh the list
      } else {
        addTestResult(`❌ Failed to remove work: ${workId}`);
      }
    } catch (error) {
      addTestResult(`❌ Remove error: ${error}`);
    }
  };

  const clearTestResults = () => {
    setTestResults([]);
  };

  return (
    <div className="container mx-auto p-6 max-w-4xl">
      <h1 className="text-3xl font-bold mb-6">PWA & Service Worker Test Page</h1>
      
      {/* Status Section */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-8">
        <div className="bg-blue-50 p-4 rounded-lg border">
          <h3 className="font-semibold text-blue-800">Service Worker</h3>
          <p className="text-sm text-blue-600" data-testid="sw-status">{swStatus}</p>
        </div>
        
        <div className="bg-green-50 p-4 rounded-lg border">
          <h3 className="font-semibold text-green-800">Network Status</h3>
          <p className="text-sm text-green-600">{isOnline ? '🟢 Online' : '🔴 Offline'}</p>
        </div>
        
        <div className="bg-purple-50 p-4 rounded-lg border">
          <h3 className="font-semibold text-purple-800">Offline Works</h3>
          <p className="text-sm text-purple-600">{offlineWorks.length} cached</p>
        </div>

        <div className="bg-orange-50 p-4 rounded-lg border">
          <h3 className="font-semibold text-orange-800">Consent TTL</h3>
          <p className="text-sm text-orange-600" data-testid="ttl-display">{ttlDisplay}</p>
        </div>
      </div>

      {/* Consent Controls Section */}
      <div className="bg-white p-6 rounded-lg border shadow-sm mb-8">
        <h3 className="text-lg font-semibold mb-4">Consent Level Testing</h3>
        <div className="flex items-center space-x-4">
          <label htmlFor="consent-select" className="font-medium">Consent Level:</label>
          <select 
            id="consent-select"
            data-testid="consent-level-select"
            value={consentLevel}
            onChange={(e) => handleConsentChange(e.target.value)}
            className="border border-gray-300 rounded px-3 py-2"
          >
            <option value="none">None</option>
            <option value="minimal">Minimal (PWA Only - 24h)</option>
            <option value="full">Full (Files + PWA - 30 days)</option>
          </select>
          <span className="text-sm text-gray-600">Current TTL: <span data-testid="ttl-display-inline">{ttlDisplay}</span></span>
        </div>
      </div>

      {/* Test Actions */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-6 mb-8">
        <div className="bg-white p-6 rounded-lg border shadow-sm">
          <h3 className="text-lg font-semibold mb-4">Consent-Aware Caching Tests</h3>
          <div className="space-y-2">
            <button
              onClick={() => testCacheWork('files_and_pwa')}
              data-testid="test-cache-btn"
              className="w-full bg-green-600 text-white px-4 py-2 rounded hover:bg-green-700 transition-colors"
            >
              Test Caching
            </button>
            
            <button
              onClick={testGetOfflineWorks}
              data-testid="test-storage-btn"
              className="w-full bg-purple-600 text-white px-4 py-2 rounded hover:bg-purple-700 transition-colors"
            >
              Test Storage
            </button>
            
            <button
              onClick={testCacheStats}
              className="w-full bg-indigo-600 text-white px-4 py-2 rounded hover:bg-indigo-700 transition-colors"
            >
              Get Cache Stats
            </button>
          </div>
        </div>

        <div className="bg-white p-6 rounded-lg border shadow-sm">
          <h3 className="text-lg font-semibold mb-4">Cache Management Tests</h3>
          <div className="space-y-2">
            <div data-testid="cache-status" className="text-sm text-gray-600 mt-2">
              Cache Status: {cacheStatus}
            </div>
            
            <button
              onClick={checkServiceWorkerStatus}
              className="w-full bg-gray-600 text-white px-4 py-2 rounded hover:bg-gray-700 transition-colors"
            >
              Refresh SW Status
            </button>
            
            <button
              onClick={clearTestResults}
              className="w-full bg-red-600 text-white px-4 py-2 rounded hover:bg-red-700 transition-colors"
            >
              Clear Test Results
            </button>
          </div>
        </div>
      </div>

      {/* Offline Works List */}
      {offlineWorks.length > 0 && (
        <div className="bg-white p-6 rounded-lg border shadow-sm mb-8">
          <h3 className="text-lg font-semibold mb-4">Cached Works</h3>
          <div className="space-y-2">
            {offlineWorks.map((work) => (
              <div key={work.workId} className="flex items-center justify-between p-3 bg-gray-50 rounded">
                <div>
                  <p className="font-medium">{work.title}</p>
                  <p className="text-sm text-gray-600">
                    Consent: {work.consentLevel} | 
                    Expires: {offlineReadingManager.formatTimeRemaining(work.expiresAt)} |
                    {work.isExpired ? ' ⚠️ Expired' : ' ✅ Valid'}
                  </p>
                </div>
                <button
                  onClick={() => testRemoveWork(work.workId)}
                  className="bg-red-500 text-white px-3 py-1 rounded text-sm hover:bg-red-600 transition-colors"
                >
                  Remove
                </button>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Test Results */}
      <div className="bg-gray-900 text-green-400 p-6 rounded-lg font-mono text-sm">
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-lg font-semibold">Test Console</h3>
          <span className="text-xs text-gray-400">{testResults.length} entries</span>
        </div>
        
        <div className="max-h-64 overflow-y-auto space-y-1">
          {testResults.length === 0 ? (
            <p className="text-gray-500">No test results yet. Click a test button to start.</p>
          ) : (
            testResults.map((result, index) => (
              <div key={index} className="text-xs leading-relaxed">
                {result}
              </div>
            ))
          )}
        </div>
      </div>

      {/* Instructions */}
      <div className="mt-8 bg-yellow-50 p-6 rounded-lg border border-yellow-200">
        <h3 className="text-lg font-semibold text-yellow-800 mb-2">Testing Instructions</h3>
        <ul className="text-sm text-yellow-700 space-y-1">
          <li>1. Check that the Service Worker status shows "consent-aware" is active</li>
          <li>2. Test both consent levels to see different TTL behavior</li>
          <li>3. Use browser DevTools → Application → Storage to see IndexedDB entries</li>
          <li>4. Use DevTools → Network → Offline to test offline functionality</li>
          <li>5. Check DevTools → Application → Service Workers for SW details</li>
          <li>6. Open browser console to see service worker logs</li>
        </ul>
      </div>
    </div>
  );
}