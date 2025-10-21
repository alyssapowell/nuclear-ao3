import { test, expect } from '@playwright/test';

test.describe('Enhanced Frontend Features', () => {
  test('should display homepage with enhanced features', async ({ page }) => {
    await page.goto('/');
    
    // Verify enhanced navigation is present (use specific unique text)
    await expect(page.getByRole('heading', { name: 'Welcome to Nuclear AO3' })).toBeVisible();
    await expect(page.getByRole('heading', { name: 'Enhanced Search', exact: true })).toBeVisible();
    
    // Verify search form is present with enhanced features
    await expect(page.locator('select[id*="sort"]')).toBeVisible();
    await expect(page.getByText('Sort Results By')).toBeVisible();
    
    // Verify footer is present (use the main footer)
    await expect(page.getByRole('contentinfo')).toBeVisible();
    await expect(page.getByText('A modern, fast, and user-friendly archive')).toBeVisible();
    
    // Verify responsive design elements
    await expect(page.locator('.max-w-7xl').first()).toBeVisible();
  });

  test('should have working navigation links', async ({ page }) => {
    await page.goto('/');
    
    // Test navigation to works page
    await page.click('a[href="/works"]');
    await expect(page).toHaveURL('/works');
    
    // Verify works page loads with enhanced content
    await expect(page.getByRole('heading', { name: 'Browse Works' })).toBeVisible();
  });

  test('should be mobile responsive', async ({ page }) => {
    // Test mobile viewport
    await page.setViewportSize({ width: 375, height: 667 });
    await page.goto('/');
    
    // Mobile navigation should be present (at least one button should be visible)
    await expect(page.locator('button').first()).toBeVisible();
    
    // Content should still be accessible (use specific heading)
    await expect(page.getByRole('heading', { name: 'Welcome to Nuclear AO3' })).toBeVisible();
    await expect(page.getByRole('heading', { name: 'Enhanced Search', exact: true })).toBeVisible();
  });

  test('should display service status', async ({ page }) => {
    await page.goto('/');
    
    // Service status indicator should be present
    await expect(page.getByText('Services')).toBeVisible();
  });
});