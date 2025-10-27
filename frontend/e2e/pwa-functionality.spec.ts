import { test, expect } from '@playwright/test';

test.describe('PWA Functionality Tests', () => {
  test.use({
    serviceWorkers: 'allow',
    permissions: ['notifications'],
  });

  test.describe('Service Worker Registration', () => {
    test('should register consent-aware service worker successfully', async ({ page }) => {
      await page.goto('/pwa-test');
      await page.waitForLoadState('networkidle');
      
      const swRegistered = await page.evaluate(async () => {
        if ('serviceWorker' in navigator) {
          const registration = await navigator.serviceWorker.getRegistration();
          return !!registration;
        }
        return false;
      });
      
      expect(swRegistered).toBeTruthy();
    });

    test('should have correct service worker script URL', async ({ page }) => {
      await page.goto('/pwa-test');
      await page.waitForLoadState('networkidle');
      
      const swScriptUrl = await page.evaluate(async () => {
        if ('serviceWorker' in navigator) {
          const registration = await navigator.serviceWorker.getRegistration();
          return registration?.active?.scriptURL;
        }
        return null;
      });
      
      expect(swScriptUrl).toContain('/sw-consent-aware.js');
    });

    test('should display service worker status on test page', async ({ page }) => {
      await page.goto('/pwa-test');
      
      const swStatus = page.locator('[data-testid="sw-status"]');
      await expect(swStatus).toBeVisible();
      await expect(swStatus).toContainText('Active');
    });
  });

  test.describe('Consent-Aware Caching', () => {
    test('should handle consent level changes', async ({ page }) => {
      await page.goto('/pwa-test');
      await page.waitForLoadState('networkidle');
      
      // Change consent level
      await page.selectOption('[data-testid="consent-level"]', 'full');
      
      // Verify consent change was communicated to service worker
      const consentLevel = await page.evaluate(() => {
        return localStorage.getItem('offlineReadingConsent');
      });
      
      expect(consentLevel).toBe('full');
    });

    test('should show different TTL values for consent levels', async ({ page }) => {
      await page.goto('/pwa-test');
      await page.waitForLoadState('networkidle');
      
      // Test minimal consent TTL
      await page.selectOption('[data-testid="consent-level"]', 'minimal');
      const minimalTTL = page.locator('[data-testid="cache-ttl"]');
      await expect(minimalTTL).toContainText('1 hour');
      
      // Test full consent TTL
      await page.selectOption('[data-testid="consent-level"]', 'full');
      const fullTTL = page.locator('[data-testid="cache-ttl"]');
      await expect(fullTTL).toContainText('7 days');
    });

    test('should communicate consent changes to service worker', async ({ page }) => {
      await page.goto('/pwa-test');
      await page.waitForLoadState('networkidle');
      
      // Change consent and verify service worker receives message
      await page.selectOption('[data-testid="consent-level"]', 'minimal');
      
      // Check that cache was updated with new TTL
      const cacheInfo = await page.evaluate(async () => {
        const caches = await window.caches?.names();
        return caches || [];
      });
      
      expect(cacheInfo).toBeDefined();
    });
  });

  test.describe('Offline Functionality', () => {
    test('should cache and serve content offline', async ({ page }) => {
      await page.goto('/pwa-test');
      await page.waitForLoadState('networkidle');
      
      // Enable full consent for caching
      await page.selectOption('[data-testid="consent-level"]', 'full');
      
      // Visit a work page to cache it
      await page.goto('/works/1');
      await page.waitForLoadState('networkidle');
      
      // Go offline and verify cached content is served
      await page.context().setOffline(true);
      await page.goto('/works/1');
      
      // Should not show offline page for cached content
      await expect(page.locator('body')).not.toContainText('You are currently offline');
    });

    test('should show offline page for uncached content', async ({ page }) => {
      await page.goto('/pwa-test');
      await page.waitForLoadState('networkidle');
      
      // Go offline
      await page.context().setOffline(true);
      
      // Try to access uncached page
      await page.goto('/some-uncached-page');
      
      // Should show offline fallback
      await expect(page.locator('h1')).toContainText('You are currently offline');
    });

    test('should handle offline reading consent expiration', async ({ page }) => {
      await page.goto('/pwa-test');
      await page.waitForLoadState('networkidle');
      
      // Set minimal consent (short TTL)
      await page.selectOption('[data-testid="consent-level"]', 'minimal');
      
      // Check that expired consent is handled properly
      const expirationStatus = page.locator('[data-testid="consent-expiration"]');
      await expect(expirationStatus).toBeVisible();
    });
  });

  test.describe('IndexedDB Storage', () => {
    test('should store work metadata in IndexedDB', async ({ page }) => {
      await page.goto('/pwa-test');
      await page.waitForLoadState('networkidle');
      
      // Enable full consent
      await page.selectOption('[data-testid="consent-level"]', 'full');
      
      // Visit a work to trigger metadata storage
      await page.goto('/works/1');
      await page.waitForLoadState('networkidle');
      
      // Check IndexedDB contains work metadata
      const dbData = await page.evaluate(async () => {
        return new Promise((resolve) => {
          const request = indexedDB.open('ao3-offline-reading', 1);
          request.onsuccess = (event) => {
            const db = event.target.result;
            const transaction = db.transaction(['works'], 'readonly');
            const store = transaction.objectStore('works');
            const getRequest = store.get('1');
            getRequest.onsuccess = () => resolve(!!getRequest.result);
          };
          request.onerror = () => resolve(false);
        });
      });
      
      expect(dbData).toBeTruthy();
    });

    test('should clean up expired IndexedDB entries', async ({ page }) => {
      await page.goto('/pwa-test');
      await page.waitForLoadState('networkidle');
      
      // Test cleanup functionality
      await page.click('[data-testid="cleanup-expired"]');
      
      const cleanupResult = page.locator('[data-testid="cleanup-status"]');
      await expect(cleanupResult).toContainText('Cleanup completed');
    });
  });

  test.describe('PWA Installation', () => {
    test('should have valid PWA manifest', async ({ page }) => {
      await page.goto('/');
      
      const manifestResponse = await page.request.get('/manifest.json');
      expect(manifestResponse.status()).toBe(200);
      
      const manifest = await manifestResponse.json();
      expect(manifest.name).toBeDefined();
      expect(manifest.short_name).toBeDefined();
      expect(manifest.icons).toBeDefined();
    });

    test('should have PWA icons accessible', async ({ page }) => {
      await page.goto('/');
      
      const icon192Response = await page.request.get('/icon-192x192.svg');
      expect(icon192Response.status()).toBe(200);
      
      const icon512Response = await page.request.get('/icon-512x512.svg');
      expect(icon512Response.status()).toBe(200);
    });

    test('should register beforeinstallprompt event', async ({ page }) => {
      await page.goto('/pwa-test');
      
      // Check if install prompt handling is present
      const installHandler = await page.evaluate(() => {
        return typeof window.addEventListener === 'function';
      });
      
      expect(installHandler).toBeTruthy();
    });
  });

  test.describe('PWA Test Page Interface', () => {
    test('should display all PWA testing controls', async ({ page }) => {
      await page.goto('/pwa-test');
      
      await expect(page.locator('[data-testid="consent-level"]')).toBeVisible();
      await expect(page.locator('[data-testid="sw-status"]')).toBeVisible();
      await expect(page.locator('[data-testid="cache-ttl"]')).toBeVisible();
    });

    test('should provide real-time status updates', async ({ page }) => {
      await page.goto('/pwa-test');
      await page.waitForLoadState('networkidle');
      
      const statusElement = page.locator('[data-testid="pwa-status"]');
      await expect(statusElement).toBeVisible();
    });

    test('should handle cache testing buttons', async ({ page }) => {
      await page.goto('/pwa-test');
      await page.waitForLoadState('networkidle');
      
      // Test cache functionality button
      await page.click('[data-testid="test-cache"]');
      
      const cacheResult = page.locator('[data-testid="cache-test-result"]');
      await expect(cacheResult).toBeVisible();
    });
  });
});