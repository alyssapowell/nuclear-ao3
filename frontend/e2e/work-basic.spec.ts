import { test, expect } from '@playwright/test';

test.describe('Basic Work Viewing Tests', () => {
  test('should load work page and display content correctly', async ({ page }) => {
    // Visit the specific work that was failing
    await page.goto('http://localhost:3001/works/251989f2-0a32-4d5d-96f0-0c2e2c1487ca');
    
    // Check that the page loads without errors
    await expect(page.locator('h1')).toContainText('Statistics Test');
    
    // Check that author is displayed
    await expect(page.locator('text=by')).toBeVisible();
    await expect(page.locator('text=testuser_')).toBeVisible();
    
    // Check that chapter content is displayed  
    await expect(page.locator('text=Chapter content goes here.')).toBeVisible();
    
    // Check that chapter navigation exists for multi-chapter work
    await expect(page.locator('select')).toBeVisible();
    
    // Verify chapters can be switched
    const chapterSelect = page.locator('select');
    await chapterSelect.selectOption('2');
    
    // Verify we're now on chapter 2
    await expect(page.locator('text=Chapter 2')).toBeVisible();
    
    // Switch back to chapter 1
    await chapterSelect.selectOption('1');
    await expect(page.locator('text=Chapter 1')).toBeVisible();
  });

  test('should display work metadata correctly', async ({ page }) => {
    await page.goto('http://localhost:3001/works/251989f2-0a32-4d5d-96f0-0c2e2c1487ca');
    
    // Check basic metadata
    await expect(page.locator('text=Words:')).toBeVisible();
    await expect(page.locator('text=Chapters:')).toBeVisible();
    await expect(page.locator('text=4/4')).toBeVisible(); // 4 chapters
    
    // Check rating is displayed
    await expect(page.locator('text=Rating:')).toBeVisible();
    await expect(page.locator('text=General')).toBeVisible();
    
    // Check that kudos and hits are displayed
    await expect(page.locator('text=Hits:')).toBeVisible();
    await expect(page.locator('text=Kudos:')).toBeVisible();
  });

  test('should handle non-existent work correctly', async ({ page }) => {
    // Visit a non-existent work
    await page.goto('http://localhost:3001/works/00000000-0000-0000-0000-000000000000');
    
    // Should show error message, not crash
    await expect(page.locator('text=Work not found')).toBeVisible();
  });

  test('should handle invalid work ID correctly', async ({ page }) => {
    // Visit with invalid UUID
    await page.goto('http://localhost:3001/works/invalid-work-id');
    
    // Should show error message
    await expect(page.locator('text=Invalid Work ID')).toBeVisible();
  });
});