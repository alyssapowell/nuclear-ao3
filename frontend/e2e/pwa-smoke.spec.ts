import { test, expect } from '@playwright/test';

test.describe('PWA Smoke Tests', () => {
  test.use({
    serviceWorkers: 'allow',
  });

  test('should load PWA test page successfully', async ({ page }) => {
    await page.goto('/pwa-test');
    
    // Check that the page loads
    await expect(page.getByRole('heading', { name: 'PWA & Service Worker Test Page' })).toBeVisible();
    
    // Check that the consent selector is present
    await expect(page.locator('[data-testid="consent-level-select"]')).toBeVisible();
    
    // Check that service worker status element exists
    await expect(page.locator('[data-testid="sw-status"]')).toBeVisible();
  });

  test('should have PWA manifest accessible', async ({ page, request }) => {
    const manifestResponse = await request.get('/manifest.json');
    expect(manifestResponse.status()).toBe(200);
    
    const manifest = await manifestResponse.json();
    expect(manifest.name).toBeDefined();
    expect(manifest.short_name).toBeDefined();
  });

  test('should have service worker file accessible', async ({ page, request }) => {
    const swResponse = await request.get('/sw-consent-aware.js');
    expect(swResponse.status()).toBe(200);
    
    const swContent = await swResponse.text();
    expect(swContent).toContain('consent-aware');
  });
});