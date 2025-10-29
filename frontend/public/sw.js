// Nuclear AO3 Service Worker - Offline Reading Support
// Enables offline reading capabilities and smart caching

const CACHE_NAME = 'nuclear-ao3-v1.0.0';
const STATIC_CACHE = 'nuclear-ao3-static-v1.0.0';
const API_CACHE = 'nuclear-ao3-api-v1.0.0';
const WORKS_CACHE = 'nuclear-ao3-works-v1.0.0';
const OFFLINE_URL = '/offline';

// Critical assets to cache for offline functionality
const STATIC_ASSETS = [
  '/',
  '/works',
  '/search',
  '/bookmarks',
  '/offline',
  '/manifest.json',
  '/_next/static/css/app.css',
  '/_next/static/chunks/main.js',
  '/_next/static/chunks/webpack.js',
  '/icon-192x192.png',
  '/icon-512x512.png'
];

// API endpoints to cache for offline access
const CACHEABLE_APIS = [
  '/api/v1/tags/autocomplete',
  '/api/v1/search/works',
  '/api/v1/works/',
  '/api/v1/tags/popular'
];

// Install event - cache static assets
self.addEventListener('install', (event) => {
  console.log('[SW] Installing service worker...');
  
  event.waitUntil(
    (async () => {
      const staticCache = await caches.open(STATIC_CACHE);
      
      try {
        await staticCache.addAll(STATIC_ASSETS);
        console.log('[SW] Static assets cached successfully');
      } catch (error) {
        console.error('[SW] Failed to cache static assets:', error);
        // Continue installation even if some assets fail
      }
      
      // Skip waiting to activate immediately
      await self.skipWaiting();
    })()
  );
});

// Activate event - cleanup old caches
self.addEventListener('activate', (event) => {
  console.log('[SW] Activating service worker...');
  
  event.waitUntil(
    (async () => {
      // Take control of all clients immediately
      await self.clients.claim();
      
      // Clean up old caches
      const cacheNames = await caches.keys();
      const oldCaches = cacheNames.filter(name => 
        name.startsWith('nuclear-ao3-') && 
        !name.includes('v1.0.0')
      );
      
      await Promise.all(
        oldCaches.map(cacheName => {
          console.log('[SW] Deleting old cache:', cacheName);
          return caches.delete(cacheName);
        })
      );
      
      console.log('[SW] Service worker activated');
    })()
  );
});

// Fetch event - implement caching strategies
self.addEventListener('fetch', (event) => {
  const { request } = event;
  const url = new URL(request.url);
  
  // Skip non-GET requests
  if (request.method !== 'GET') {
    return;
  }
  
  // Skip Chrome extensions and other protocols
  if (!url.protocol.startsWith('http')) {
    return;
  }
  
  event.respondWith(handleFetch(request));
});

// Main fetch handler with different strategies
async function handleFetch(request) {
  const url = new URL(request.url);
  
  try {
    // Strategy 1: Static assets - Cache First
    if (isStaticAsset(url.pathname)) {
      return await cacheFirst(request, STATIC_CACHE);
    }
    
    // Strategy 2: API calls - Network First with fallback
    if (isAPICall(url.pathname)) {
      return await networkFirstWithCache(request, API_CACHE);
    }
    
    // Strategy 3: Work content - Cache with update
    if (isWorkContent(url.pathname)) {
      return await staleWhileRevalidate(request, WORKS_CACHE);
    }
    
    // Strategy 4: Navigation requests - Network with offline fallback
    if (request.mode === 'navigate') {
      return await networkWithOfflineFallback(request);
    }
    
    // Default: Network only
    return await fetch(request);
    
  } catch (error) {
    console.error('[SW] Fetch failed:', error);
    
    // Provide offline fallback for navigation
    if (request.mode === 'navigate') {
      const offlineResponse = await caches.match(OFFLINE_URL);
      return offlineResponse || new Response('Offline', { status: 503 });
    }
    
    throw error;
  }
}

// Cache First strategy - for static assets
async function cacheFirst(request, cacheName) {
  const cache = await caches.open(cacheName);
  const cachedResponse = await cache.match(request);
  
  if (cachedResponse) {
    console.log('[SW] Cache hit:', request.url);
    return cachedResponse;
  }
  
  console.log('[SW] Cache miss, fetching:', request.url);
  const networkResponse = await fetch(request);
  
  if (networkResponse.ok) {
    await cache.put(request, networkResponse.clone());
  }
  
  return networkResponse;
}

// Network First with cache fallback - for API calls
async function networkFirstWithCache(request, cacheName) {
  const cache = await caches.open(cacheName);
  
  try {
    const networkResponse = await fetch(request);
    
    if (networkResponse.ok) {
      // Cache successful API responses
      await cache.put(request, networkResponse.clone());
      console.log('[SW] API cached:', request.url);
    }
    
    return networkResponse;
  } catch (error) {
    console.log('[SW] Network failed, checking cache:', request.url);
    
    const cachedResponse = await cache.match(request);
    if (cachedResponse) {
      console.log('[SW] Serving from cache:', request.url);
      return cachedResponse;
    }
    
    throw error;
  }
}

// Stale While Revalidate - for work content
async function staleWhileRevalidate(request, cacheName) {
  const cache = await caches.open(cacheName);
  const cachedResponse = await cache.match(request);
  
  // Always try to fetch fresh content in background
  const fetchPromise = fetch(request).then(async (response) => {
    if (response.ok) {
      await cache.put(request, response.clone());
      console.log('[SW] Background updated:', request.url);
    }
    return response;
  }).catch(error => {
    console.log('[SW] Background fetch failed:', error);
    return null;
  });
  
  // Return cached version immediately if available
  if (cachedResponse) {
    console.log('[SW] Serving stale content:', request.url);
    return cachedResponse;
  }
  
  // Otherwise wait for network
  console.log('[SW] No cache, waiting for network:', request.url);
  return await fetchPromise;
}

// Network with offline fallback - for navigation
async function networkWithOfflineFallback(request) {
  try {
    const networkResponse = await fetch(request);
    
    // Cache successful navigation responses
    if (networkResponse.ok) {
      const cache = await caches.open(STATIC_CACHE);
      await cache.put(request, networkResponse.clone());
    }
    
    return networkResponse;
  } catch (error) {
    console.log('[SW] Navigation offline, serving offline page');
    
    // Try to serve cached version of the same page
    const cachedResponse = await caches.match(request);
    if (cachedResponse) {
      return cachedResponse;
    }
    
    // Fallback to offline page
    const offlineResponse = await caches.match(OFFLINE_URL);
    return offlineResponse || new Response('Offline', { status: 503 });
  }
}

// Helper functions
function isStaticAsset(pathname) {
  return pathname.startsWith('/_next/') ||
         pathname.endsWith('.css') ||
         pathname.endsWith('.js') ||
         pathname.endsWith('.png') ||
         pathname.endsWith('.svg') ||
         pathname === '/manifest.json';
}

function isAPICall(pathname) {
  return pathname.startsWith('/api/') ||
         CACHEABLE_APIS.some(api => pathname.startsWith(api));
}

function isWorkContent(pathname) {
  return pathname.startsWith('/works/') && 
         !pathname.includes('/edit') &&
         !pathname.includes('/delete');
}

// Message handling for cache management
self.addEventListener('message', (event) => {
  const { type, data } = event.data;
  
  switch (type) {
    case 'CACHE_WORK':
      cacheWork(data.workId, data.workData);
      break;
      
    case 'CLEAR_CACHE':
      clearCache(data.cacheType);
      break;
      
    case 'GET_CACHE_SIZE':
      getCacheSize().then(size => {
        event.ports[0].postMessage({ type: 'CACHE_SIZE', size });
      });
      break;
      
    case 'SKIP_WAITING':
      self.skipWaiting();
      break;
  }
});

// Cache specific work for offline reading
async function cacheWork(workId, workData) {
  try {
    const cache = await caches.open(WORKS_CACHE);
    const workUrl = `/works/${workId}`;
    
    // Cache the work page
    const response = new Response(JSON.stringify(workData), {
      headers: { 'Content-Type': 'application/json' }
    });
    
    await cache.put(workUrl, response);
    console.log('[SW] Work cached for offline reading:', workId);
    
    // Notify all clients
    const clients = await self.clients.matchAll();
    clients.forEach(client => {
      client.postMessage({
        type: 'WORK_CACHED',
        workId: workId
      });
    });
    
  } catch (error) {
    console.error('[SW] Failed to cache work:', error);
  }
}

// Clear specific cache type
async function clearCache(cacheType) {
  try {
    let cacheName;
    
    switch (cacheType) {
      case 'static':
        cacheName = STATIC_CACHE;
        break;
      case 'api':
        cacheName = API_CACHE;
        break;
      case 'works':
        cacheName = WORKS_CACHE;
        break;
      case 'all':
        const cacheNames = await caches.keys();
        await Promise.all(
          cacheNames
            .filter(name => name.startsWith('nuclear-ao3-'))
            .map(name => caches.delete(name))
        );
        console.log('[SW] All caches cleared');
        return;
    }
    
    if (cacheName) {
      await caches.delete(cacheName);
      console.log('[SW] Cache cleared:', cacheName);
    }
  } catch (error) {
    console.error('[SW] Failed to clear cache:', error);
  }
}

// Get total cache size
async function getCacheSize() {
  try {
    const cacheNames = await caches.keys();
    let totalSize = 0;
    
    for (const cacheName of cacheNames) {
      if (cacheName.startsWith('nuclear-ao3-')) {
        const cache = await caches.open(cacheName);
        const requests = await cache.keys();
        
        for (const request of requests) {
          const response = await cache.match(request);
          if (response) {
            const arrayBuffer = await response.clone().arrayBuffer();
            totalSize += arrayBuffer.byteLength;
          }
        }
      }
    }
    
    return totalSize;
  } catch (error) {
    console.error('[SW] Failed to calculate cache size:', error);
    return 0;
  }
}

// Background sync for when connection is restored
self.addEventListener('sync', (event) => {
  console.log('[SW] Background sync:', event.tag);
  
  switch (event.tag) {
    case 'reading-progress':
      event.waitUntil(syncReadingProgress());
      break;
    case 'bookmarks':
      event.waitUntil(syncBookmarks());
      break;
  }
});

// Sync reading progress when back online
async function syncReadingProgress() {
  try {
    // Get stored reading progress from IndexedDB
    // Sync with server when connection restored
    console.log('[SW] Syncing reading progress...');
  } catch (error) {
    console.error('[SW] Failed to sync reading progress:', error);
  }
}

// Sync bookmarks when back online
async function syncBookmarks() {
  try {
    // Get stored bookmark changes from IndexedDB
    // Sync with server when connection restored
    console.log('[SW] Syncing bookmarks...');
  } catch (error) {
    console.error('[SW] Failed to sync bookmarks:', error);
  }
}

console.log('[SW] Nuclear AO3 Service Worker loaded successfully');