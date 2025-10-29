// Nuclear AO3 Enhanced Service Worker - Ethical Offline Reading
// Implements author-driven consent controls for respectful offline access

const CACHE_VERSION = 'v2.0.0';
const CACHE_NAME = `nuclear-ao3-${CACHE_VERSION}`;
const STATIC_CACHE = `nuclear-ao3-static-${CACHE_VERSION}`;
const API_CACHE = `nuclear-ao3-api-${CACHE_VERSION}`;
const WORKS_CACHE = `nuclear-ao3-works-${CACHE_VERSION}`;
const CONSENT_CACHE = `nuclear-ao3-consent-${CACHE_VERSION}`;
const OFFLINE_URL = '/offline';

// Enhanced caching strategies for ethical offline reading
const CACHE_STRATEGIES = {
  NETWORK_FIRST: 'network-first',      // User data, real-time content
  CACHE_FIRST: 'cache-first',          // Static assets, UI components
  STALE_WHILE_REVALIDATE: 'swr',       // Work content (with consent)
  CACHE_ONLY: 'cache-only',            // Offline-approved content
  NETWORK_ONLY: 'network-only'         // Author preference checks
};

// Author consent levels for offline reading
const CONSENT_LEVELS = {
  FILES_AND_PWA: 'files_and_pwa',      // Full offline access
  PWA_ONLY: 'pwa_only',                // Temporary PWA caching only
  NONE: 'none'                         // Online only, no caching
};

// Critical assets for offline functionality
const STATIC_ASSETS = [
  '/',
  '/works',
  '/search',
  '/dashboard',
  '/offline',
  '/manifest.json',
  '/_next/static/css/app.css',
  '/_next/static/chunks/main.js',
  '/_next/static/chunks/webpack.js',
  '/icon-192x192.png',
  '/icon-512x512.png'
];

// API endpoints that respect author consent
const CONSENT_AWARE_APIS = [
  '/api/v1/works/',
  '/api/v1/works/*/chapters',
  '/api/v1/works/*/consent'
];

// TTL settings for different consent levels (in milliseconds)
const TTL_SETTINGS = {
  [CONSENT_LEVELS.FILES_AND_PWA]: 30 * 24 * 60 * 60 * 1000, // 30 days
  [CONSENT_LEVELS.PWA_ONLY]: 7 * 24 * 60 * 60 * 1000,       // 7 days
  [CONSENT_LEVELS.NONE]: 0                                    // No caching
};

// Install event - enhanced with consent system
self.addEventListener('install', (event) => {
  console.log('[SW] Installing enhanced service worker with consent controls...');
  
  event.waitUntil(
    (async () => {
      // Cache static assets
      const staticCache = await caches.open(STATIC_CACHE);
      
      try {
        await staticCache.addAll(STATIC_ASSETS);
        console.log('[SW] Static assets cached successfully');
      } catch (error) {
        console.error('[SW] Failed to cache static assets:', error);
      }
      
      // Initialize consent database
      await initializeConsentDatabase();
      
      // Skip waiting to activate immediately
      await self.skipWaiting();
    })()
  );
});

// Activate event - cleanup and claim clients
self.addEventListener('activate', (event) => {
  console.log('[SW] Activating enhanced service worker...');
  
  event.waitUntil(
    (async () => {
      // Take control of all clients immediately
      await self.clients.claim();
      
      // Clean up old caches
      await cleanupOldCaches();
      
      // Clean up expired consent-based content
      await cleanupExpiredContent();
      
      console.log('[SW] Enhanced service worker activated');
    })()
  );
});

// Enhanced fetch handler with author consent awareness
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
  
  event.respondWith(handleConsentAwareFetch(request));
});

// Main fetch handler with consent awareness
async function handleConsentAwareFetch(request) {
  const url = new URL(request.url);
  
  try {
    // Strategy 1: Static assets - Cache First
    if (isStaticAsset(url.pathname)) {
      return await cacheFirst(request, STATIC_CACHE);
    }
    
    // Strategy 2: Work content - Consent-aware caching
    if (isWorkContent(url.pathname)) {
      return await consentAwareWorkFetch(request);
    }
    
    // Strategy 3: API calls - Network First with consent check
    if (isAPICall(url.pathname)) {
      return await consentAwareAPIFetch(request);
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
      return offlineResponse || createOfflineResponse();
    }
    
    throw error;
  }
}

// Consent-aware work content fetching
async function consentAwareWorkFetch(request) {
  const workId = extractWorkId(request.url);
  if (!workId) {
    return await fetch(request);
  }
  
  // Get author's consent level for this work
  const consent = await getWorkConsentLevel(workId);
  
  switch (consent) {
    case CONSENT_LEVELS.FILES_AND_PWA:
      // Full caching allowed
      return await staleWhileRevalidateWithTTL(request, WORKS_CACHE, TTL_SETTINGS[consent]);
      
    case CONSENT_LEVELS.PWA_ONLY:
      // Temporary caching only
      return await temporaryCacheWithTTL(request, WORKS_CACHE, TTL_SETTINGS[consent]);
      
    case CONSENT_LEVELS.NONE:
      // No caching - always fetch from network
      return await networkOnlyWithEducationalMessage(request, workId);
      
    default:
      // Unknown consent - err on the side of caution
      return await fetch(request);
  }
}

// Get work consent level from cache or API
async function getWorkConsentLevel(workId) {
  try {
    // First check cache
    const cache = await caches.open(CONSENT_CACHE);
    const consentRequest = new Request(`/api/v1/works/${workId}/consent`);
    const cachedConsent = await cache.match(consentRequest);
    
    if (cachedConsent) {
      const data = await cachedConsent.json();
      return data.offline_reading_preference;
    }
    
    // Fetch from network
    const response = await fetch(consentRequest);
    if (response.ok) {
      const data = await response.json();
      
      // Cache consent decision (short TTL)
      const consentResponse = new Response(JSON.stringify(data), {
        headers: { 
          'Content-Type': 'application/json',
          'Cache-Control': 'max-age=300' // 5 minutes
        }
      });
      await cache.put(consentRequest, consentResponse);
      
      return data.offline_reading_preference;
    }
    
    // Default to most restrictive if we can't determine consent
    return CONSENT_LEVELS.NONE;
    
  } catch (error) {
    console.error('[SW] Failed to get consent level:', error);
    return CONSENT_LEVELS.NONE;
  }
}

// Temporary cache with TTL for PWA-only content
async function temporaryCacheWithTTL(request, cacheName, ttl) {
  const cache = await caches.open(cacheName);
  const cachedResponse = await cache.match(request);
  
  // Check if cached content is still valid
  if (cachedResponse) {
    const cachedTime = cachedResponse.headers.get('sw-cached-time');
    if (cachedTime && (Date.now() - parseInt(cachedTime)) < ttl) {
      console.log('[SW] Serving temporary cached content:', request.url);
      return cachedResponse;
    } else {
      // Expired - remove from cache
      await cache.delete(request);
      console.log('[SW] Temporary cache expired, removed:', request.url);
    }
  }
  
  // Fetch fresh content
  try {
    const response = await fetch(request);
    
    if (response.ok) {
      // Add cache timestamp
      const responseClone = response.clone();
      const headers = new Headers(responseClone.headers);
      headers.set('sw-cached-time', Date.now().toString());
      headers.set('sw-consent-level', CONSENT_LEVELS.PWA_ONLY);
      
      const modifiedResponse = new Response(responseClone.body, {
        status: responseClone.status,
        statusText: responseClone.statusText,
        headers: headers
      });
      
      await cache.put(request, modifiedResponse.clone());
      console.log('[SW] Content temporarily cached with TTL:', request.url);
      
      return modifiedResponse;
    }
    
    return response;
  } catch (error) {
    // If network fails and we had expired cache, serve it anyway
    if (cachedResponse) {
      console.log('[SW] Network failed, serving expired temporary cache:', request.url);
      return cachedResponse;
    }
    throw error;
  }
}

// Network-only with educational message for restricted content
async function networkOnlyWithEducationalMessage(request, workId) {
  try {
    const response = await fetch(request);
    
    // Don't cache anything for NONE consent level
    console.log('[SW] Serving online-only content (no caching):', request.url);
    
    return response;
  } catch (error) {
    // Provide educational offline message
    return createEducationalOfflineResponse(workId);
  }
}

// Stale while revalidate with TTL for full access content
async function staleWhileRevalidateWithTTL(request, cacheName, ttl) {
  const cache = await caches.open(cacheName);
  const cachedResponse = await cache.match(request);
  
  // Check if we have valid cached content
  let serveCached = false;
  if (cachedResponse) {
    const cachedTime = cachedResponse.headers.get('sw-cached-time');
    if (cachedTime && (Date.now() - parseInt(cachedTime)) < ttl) {
      serveCached = true;
    }
  }
  
  // Always try to fetch fresh content in background
  const fetchPromise = fetch(request).then(async (response) => {
    if (response.ok) {
      const headers = new Headers(response.headers);
      headers.set('sw-cached-time', Date.now().toString());
      headers.set('sw-consent-level', CONSENT_LEVELS.FILES_AND_PWA);
      
      const modifiedResponse = new Response(response.body, {
        status: response.status,
        statusText: response.statusText,
        headers: headers
      });
      
      await cache.put(request, modifiedResponse.clone());
      console.log('[SW] Content updated in long-term cache:', request.url);
      return modifiedResponse;
    }
    return response;
  }).catch(error => {
    console.log('[SW] Background fetch failed:', error);
    return null;
  });
  
  // Return cached version immediately if available and valid
  if (serveCached) {
    console.log('[SW] Serving cached content:', request.url);
    return cachedResponse;
  }
  
  // Otherwise wait for network
  console.log('[SW] No valid cache, waiting for network:', request.url);
  return await fetchPromise;
}

// Consent-aware API fetching
async function consentAwareAPIFetch(request) {
  // For consent-checking APIs, always use network first
  if (isConsentAwareAPI(request.url)) {
    return await networkFirstWithShortCache(request, API_CACHE);
  }
  
  // For other APIs, use normal network first
  return await networkFirstWithCache(request, API_CACHE);
}

// Helper functions
function isWorkContent(pathname) {
  return pathname.startsWith('/works/') && 
         !pathname.includes('/edit') &&
         !pathname.includes('/delete') &&
         !pathname.includes('/consent');
}

function isConsentAwareAPI(url) {
  return CONSENT_AWARE_APIS.some(api => url.includes(api));
}

function extractWorkId(url) {
  const match = url.match(/\/works\/([^\/\?]+)/);
  return match ? match[1] : null;
}

function isStaticAsset(pathname) {
  return pathname.startsWith('/_next/') ||
         pathname.endsWith('.css') ||
         pathname.endsWith('.js') ||
         pathname.endsWith('.png') ||
         pathname.endsWith('.svg') ||
         pathname === '/manifest.json';
}

function isAPICall(pathname) {
  return pathname.startsWith('/api/');
}

// Create educational offline response for restricted content
function createEducationalOfflineResponse(workId) {
  const html = `
    <!DOCTYPE html>
    <html>
    <head>
      <title>Respecting Author Preferences</title>
      <meta name="viewport" content="width=device-width, initial-scale=1">
      <style>
        body { font-family: system-ui; padding: 2rem; max-width: 600px; margin: 0 auto; }
        .message { background: #f3f4f6; padding: 1.5rem; border-radius: 8px; margin: 1rem 0; }
        .icon { font-size: 2rem; margin-bottom: 1rem; }
      </style>
    </head>
    <body>
      <div class="message">
        <div class="icon">🔒</div>
        <h2>Respecting Author Preferences</h2>
        <p>This work's author has chosen online-only reading. Nuclear AO3 respects this preference even when you're offline.</p>
        <p>This approach ensures authors maintain control over how their creative works are accessed.</p>
        <p><strong>Please connect to the internet to read this work.</strong></p>
      </div>
    </body>
    </html>
  `;
  
  return new Response(html, {
    headers: { 'Content-Type': 'text/html' },
    status: 503
  });
}

// Create generic offline response
function createOfflineResponse() {
  const html = `
    <!DOCTYPE html>
    <html>
    <head>
      <title>Nuclear AO3 - Offline</title>
      <meta name="viewport" content="width=device-width, initial-scale=1">
      <style>
        body { font-family: system-ui; padding: 2rem; max-width: 600px; margin: 0 auto; }
        .message { background: #e0f2fe; padding: 1.5rem; border-radius: 8px; margin: 1rem 0; }
        .icon { font-size: 2rem; margin-bottom: 1rem; }
      </style>
    </head>
    <body>
      <div class="message">
        <div class="icon">📱</div>
        <h2>Nuclear AO3 - Offline Mode</h2>
        <p>You're currently offline. Some content may not be available based on author preferences.</p>
        <p>Nuclear AO3 respects authors' choices about offline reading access.</p>
        <p><strong>Connect to the internet for full access.</strong></p>
      </div>
    </body>
    </html>
  `;
  
  return new Response(html, {
    headers: { 'Content-Type': 'text/html' },
    status: 503
  });
}

// Initialize consent database
async function initializeConsentDatabase() {
  try {
    const cache = await caches.open(CONSENT_CACHE);
    console.log('[SW] Consent cache initialized');
  } catch (error) {
    console.error('[SW] Failed to initialize consent database:', error);
  }
}

// Clean up old caches
async function cleanupOldCaches() {
  const cacheNames = await caches.keys();
  const oldCaches = cacheNames.filter(name => 
    name.startsWith('nuclear-ao3-') && 
    !name.includes(CACHE_VERSION)
  );
  
  await Promise.all(
    oldCaches.map(cacheName => {
      console.log('[SW] Deleting old cache:', cacheName);
      return caches.delete(cacheName);
    })
  );
}

// Clean up expired content based on consent TTL
async function cleanupExpiredContent() {
  try {
    const cache = await caches.open(WORKS_CACHE);
    const requests = await cache.keys();
    
    for (const request of requests) {
      const response = await cache.match(request);
      if (response) {
        const cachedTime = response.headers.get('sw-cached-time');
        const consentLevel = response.headers.get('sw-consent-level');
        
        if (cachedTime && consentLevel) {
          const age = Date.now() - parseInt(cachedTime);
          const ttl = TTL_SETTINGS[consentLevel] || 0;
          
          if (age > ttl) {
            await cache.delete(request);
            console.log('[SW] Expired consent-based content removed:', request.url);
          }
        }
      }
    }
  } catch (error) {
    console.error('[SW] Failed to cleanup expired content:', error);
  }
}

// Network first with short cache for API calls
async function networkFirstWithShortCache(request, cacheName) {
  const cache = await caches.open(cacheName);
  
  try {
    const networkResponse = await fetch(request);
    
    if (networkResponse.ok) {
      // Cache with short TTL for consent-related APIs
      const headers = new Headers(networkResponse.headers);
      headers.set('sw-cached-time', Date.now().toString());
      headers.set('Cache-Control', 'max-age=300'); // 5 minutes
      
      const modifiedResponse = new Response(networkResponse.body, {
        status: networkResponse.status,
        statusText: networkResponse.statusText,
        headers: headers
      });
      
      await cache.put(request, modifiedResponse.clone());
      return modifiedResponse;
    }
    
    return networkResponse;
  } catch (error) {
    const cachedResponse = await cache.match(request);
    if (cachedResponse) {
      // Check if cache is still fresh
      const cachedTime = cachedResponse.headers.get('sw-cached-time');
      if (cachedTime && (Date.now() - parseInt(cachedTime)) < 300000) { // 5 minutes
        return cachedResponse;
      }
    }
    throw error;
  }
}

// Standard implementations for non-consent features
async function cacheFirst(request, cacheName) {
  const cache = await caches.open(cacheName);
  const cachedResponse = await cache.match(request);
  
  if (cachedResponse) {
    return cachedResponse;
  }
  
  const networkResponse = await fetch(request);
  if (networkResponse.ok) {
    await cache.put(request, networkResponse.clone());
  }
  
  return networkResponse;
}

async function networkFirstWithCache(request, cacheName) {
  const cache = await caches.open(cacheName);
  
  try {
    const networkResponse = await fetch(request);
    if (networkResponse.ok) {
      await cache.put(request, networkResponse.clone());
    }
    return networkResponse;
  } catch (error) {
    const cachedResponse = await cache.match(request);
    if (cachedResponse) {
      return cachedResponse;
    }
    throw error;
  }
}

async function networkWithOfflineFallback(request) {
  try {
    const networkResponse = await fetch(request);
    if (networkResponse.ok) {
      const cache = await caches.open(STATIC_CACHE);
      await cache.put(request, networkResponse.clone());
    }
    return networkResponse;
  } catch (error) {
    const cachedResponse = await caches.match(request);
    if (cachedResponse) {
      return cachedResponse;
    }
    
    const offlineResponse = await caches.match(OFFLINE_URL);
    return offlineResponse || createOfflineResponse();
  }
}

// Enhanced message handling for consent operations
self.addEventListener('message', (event) => {
  const { type, data } = event.data;
  
  switch (type) {
    case 'CACHE_WORK_WITH_CONSENT':
      cacheWorkWithConsent(data.workId, data.workData, data.consentLevel);
      break;
      
    case 'UPDATE_CONSENT_LEVEL':
      updateWorkConsentLevel(data.workId, data.consentLevel);
      break;
      
    case 'CLEAR_CONSENT_CACHE':
      clearConsentCache(data.workId);
      break;
      
    case 'GET_OFFLINE_WORKS':
      getOfflineWorks().then(works => {
        event.ports[0].postMessage({ type: 'OFFLINE_WORKS', works });
      });
      break;
      
    case 'RESPECT_AUTHOR_DELETION':
      respectAuthorDeletion(data.workId);
      break;
      
    default:
      // Handle legacy message types
      break;
  }
});

// Cache work with author consent
async function cacheWorkWithConsent(workId, workData, consentLevel) {
  if (consentLevel === CONSENT_LEVELS.NONE) {
    console.log('[SW] Author does not allow caching for work:', workId);
    return;
  }
  
  try {
    const cache = await caches.open(WORKS_CACHE);
    const workUrl = `/works/${workId}`;
    
    const headers = new Headers({
      'Content-Type': 'application/json',
      'sw-cached-time': Date.now().toString(),
      'sw-consent-level': consentLevel
    });
    
    const response = new Response(JSON.stringify(workData), { headers });
    await cache.put(workUrl, response);
    
    console.log('[SW] Work cached with consent level:', workId, consentLevel);
    
    // Notify clients
    const clients = await self.clients.matchAll();
    clients.forEach(client => {
      client.postMessage({
        type: 'WORK_CACHED_WITH_CONSENT',
        workId: workId,
        consentLevel: consentLevel
      });
    });
    
  } catch (error) {
    console.error('[SW] Failed to cache work with consent:', error);
  }
}

// Update work consent level
async function updateWorkConsentLevel(workId, newConsentLevel) {
  try {
    // Update consent cache
    const consentCache = await caches.open(CONSENT_CACHE);
    const consentRequest = new Request(`/api/v1/works/${workId}/consent`);
    
    const consentData = { offline_reading_preference: newConsentLevel };
    const consentResponse = new Response(JSON.stringify(consentData), {
      headers: { 'Content-Type': 'application/json' }
    });
    
    await consentCache.put(consentRequest, consentResponse);
    
    // Handle content based on new consent level
    if (newConsentLevel === CONSENT_LEVELS.NONE) {
      // Remove from cache if author revokes consent
      const worksCache = await caches.open(WORKS_CACHE);
      await worksCache.delete(`/works/${workId}`);
      console.log('[SW] Work removed from cache due to consent change:', workId);
    }
    
  } catch (error) {
    console.error('[SW] Failed to update consent level:', error);
  }
}

// Respect author deletion
async function respectAuthorDeletion(workId) {
  try {
    const cache = await caches.open(WORKS_CACHE);
    await cache.delete(`/works/${workId}`);
    
    const consentCache = await caches.open(CONSENT_CACHE);
    await consentCache.delete(`/api/v1/works/${workId}/consent`);
    
    console.log('[SW] Respected author deletion, removed work:', workId);
    
    // Notify all clients
    const clients = await self.clients.matchAll();
    clients.forEach(client => {
      client.postMessage({
        type: 'WORK_DELETED',
        workId: workId
      });
    });
    
  } catch (error) {
    console.error('[SW] Failed to respect author deletion:', error);
  }
}

// Get list of offline-available works
async function getOfflineWorks() {
  try {
    const cache = await caches.open(WORKS_CACHE);
    const requests = await cache.keys();
    const works = [];
    
    for (const request of requests) {
      const response = await cache.match(request);
      if (response) {
        const consentLevel = response.headers.get('sw-consent-level');
        const cachedTime = response.headers.get('sw-cached-time');
        
        if (consentLevel && cachedTime) {
          const workId = extractWorkId(request.url);
          const ttl = TTL_SETTINGS[consentLevel];
          const expiresAt = parseInt(cachedTime) + ttl;
          
          works.push({
            workId,
            consentLevel,
            cachedAt: parseInt(cachedTime),
            expiresAt,
            isExpired: Date.now() > expiresAt
          });
        }
      }
    }
    
    return works;
  } catch (error) {
    console.error('[SW] Failed to get offline works:', error);
    return [];
  }
}

console.log('[SW] Nuclear AO3 Enhanced Service Worker with Ethical Consent Controls loaded successfully');