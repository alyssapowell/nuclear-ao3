import { test, expect } from '@playwright/test';

test.describe('Simple Connectivity Test', () => {
  test('should load the homepage', async ({ page }) => {
    await page.goto('/');
    await expect(page.locator('body')).toBeVisible();
    console.log('✅ Homepage loaded successfully');
  });

  test('should load works page', async ({ page }) => {
    await page.goto('/works');
    await expect(page.locator('body')).toBeVisible();
    console.log('✅ Works page loaded successfully');
  });

  test('should load test components page', async ({ page }) => {
    await page.goto('/test-components');
    await expect(page.locator('body')).toBeVisible();
    console.log('✅ Test components page loaded successfully');
  });
});