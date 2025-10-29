// Consent-Aware Service Worker for Nuclear AO3
// Handles caching based on user consent levels

const CACHE_NAME = 'nuclear-ao3-v1';
const CONSENT_CACHE_NAME = 'nuclear-ao3-consent-v1';

// Default consent level
let consentLevel = 'none';

// Initialize consent level from message
self.addEventListener('message', (event) => {
  if (event.data.type === 'CONSENT_LEVEL_CHANGED') {
    consentLevel = event.data.level;
    console.log('[SW] Consent level updated to:', consentLevel);
  }
});

// Install event
self.addEventListener('install', (event) => {
  console.log('[SW] Installing consent-aware service worker');
  event.waitUntil(
    caches.open(CACHE_NAME).then((cache) => {
      console.log('[SW] Cache opened');
      return cache.addAll([
        '/',
        '/manifest.json'
      ]);
    })
  );
});

// Activate event
self.addEventListener('activate', (event) => {
  console.log('[SW] Activating consent-aware service worker');
  event.waitUntil(
    caches.keys().then((cacheNames) => {
      return Promise.all(
        cacheNames.map((cacheName) => {
          if (cacheName !== CACHE_NAME && cacheName !== CONSENT_CACHE_NAME) {
            console.log('[SW] Deleting old cache:', cacheName);
            return caches.delete(cacheName);
          }
        })
      );
    })
  );
});

// Fetch event with consent-aware caching
self.addEventListener('fetch', (event) => {
  // Only handle GET requests
  if (event.request.method !== 'GET') {
    return;
  }

  // Handle based on consent level
  if (consentLevel === 'none') {
    // No caching for user data
    return;
  }

  event.respondWith(
    caches.match(event.request).then((response) => {
      if (response) {
        console.log('[SW] Serving from cache:', event.request.url);
        return response;
      }

      return fetch(event.request).then((response) => {
        // Don't cache if not successful
        if (!response || response.status !== 200 || response.type !== 'basic') {
          return response;
        }

        // Determine if we should cache based on consent level
        const shouldCache = shouldCacheRequest(event.request.url, consentLevel);
        
        if (shouldCache) {
          const responseToCache = response.clone();
          caches.open(consentLevel === 'minimal' ? CACHE_NAME : CONSENT_CACHE_NAME).then((cache) => {
            console.log('[SW] Caching:', event.request.url);
            cache.put(event.request, responseToCache);
          });
        }

        return response;
      });
    })
  );
});

// Determine if a request should be cached based on consent level
function shouldCacheRequest(url, consent) {
  const urlObj = new URL(url);
  
  // Always cache basic assets
  if (url.match(/\.(js|css|png|jpg|jpeg|gif|svg|ico)$/)) {
    return true;
  }

  // For minimal consent, only cache essential PWA files
  if (consent === 'minimal') {
    return url.includes('/manifest.json') || url.includes('/sw-');
  }

  // For full consent, cache work content too
  if (consent === 'full') {
    return true;
  }

  return false;
}

console.log('[SW] Consent-aware service worker loaded');