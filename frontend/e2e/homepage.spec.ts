import { test, expect } from '@playwright/test';

test.describe('Homepage', () => {
  test('should display the main navigation and content', async ({ page }) => {
    await page.goto('/');

    // Check that the page loads correctly (title might be empty in dev)
    await page.waitForLoadState('networkidle');

    // Check for main navigation elements (use first match for enhanced design)
    await expect(page.getByRole('link', { name: /works/i }).first()).toBeVisible();
    await expect(page.getByRole('link', { name: /browse/i }).first()).toBeVisible();

    // Check that the main content area exists
    await expect(page.getByRole('main')).toBeVisible();
  });

  test('should navigate to works page', async ({ page }) => {
    await page.goto('/');

    // Click on works link
    await page.getByRole('link', { name: /works/i }).first().click();

    // Should navigate to works page
    await expect(page).toHaveURL('/works');
    await expect(page.getByRole('heading', { name: /works/i }).first()).toBeVisible();
  });

  test('should be responsive on mobile', async ({ page }) => {
    // Set mobile viewport
    await page.setViewportSize({ width: 375, height: 667 });
    await page.goto('/');

    // Main navigation should still be accessible (use first header for enhanced nav)
    await expect(page.locator('header').first()).toBeVisible();
    
    // Content should be visible and readable
    await expect(page.getByRole('main')).toBeVisible();
  });
});