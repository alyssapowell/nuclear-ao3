'use client';

import { useEffect } from 'react';

/**
 * PWA Initialization Component
 * Handles service worker registration and PWA lifecycle management
 */
export default function PWAInit() {
  useEffect(() => {
    if (typeof window !== 'undefined' && 'serviceWorker' in navigator) {
      initializeServiceWorker();
    }
  }, []);

  const initializeServiceWorker = async () => {
    try {
      // Register the consent-aware service worker
      const registration = await navigator.serviceWorker.register('/sw-consent-aware.js', {
        scope: '/',
        updateViaCache: 'none'
      });

      console.log('[PWA] Service Worker registered:', registration);

      // Handle service worker updates
      registration.addEventListener('updatefound', () => {
        const newWorker = registration.installing;
        if (newWorker) {
          console.log('[PWA] New service worker installing...');
          
          newWorker.addEventListener('statechange', () => {
            if (newWorker.state === 'installed' && navigator.serviceWorker.controller) {
              console.log('[PWA] New service worker available');
              // Could notify user of update here
            }
          });
        }
      });

      // Handle controller change (new SW activated)
      navigator.serviceWorker.addEventListener('controllerchange', () => {
        console.log('[PWA] Service worker controller changed');
        window.location.reload();
      });

      // Listen for messages from service worker
      navigator.serviceWorker.addEventListener('message', (event) => {
        const { type, data } = event.data;
        
        switch (type) {
          case 'WORK_CACHED_WITH_CONSENT':
            console.log('[PWA] Work cached with consent:', data);
            // Dispatch custom event for UI components to listen to
            window.dispatchEvent(new CustomEvent('workCachedWithConsent', { detail: data }));
            break;
            
          case 'WORK_EXPIRED':
            console.log('[PWA] Work expired:', data);
            window.dispatchEvent(new CustomEvent('workExpired', { detail: data }));
            break;
            
          case 'WORK_DELETED':
            console.log('[PWA] Work deleted:', data);
            window.dispatchEvent(new CustomEvent('workDeleted', { detail: data }));
            break;
            
          case 'WORK_CACHE_FAILED':
            console.error('[PWA] Work cache failed:', data);
            window.dispatchEvent(new CustomEvent('workCacheFailed', { detail: data }));
            break;
        }
      });

      // Check for existing service worker and activate immediately if waiting
      if (registration.waiting) {
        registration.waiting.postMessage({ type: 'SKIP_WAITING' });
      }

    } catch (error) {
      console.error('[PWA] Service Worker registration failed:', error);
    }
  };

  return null; // This component doesn't render anything
}