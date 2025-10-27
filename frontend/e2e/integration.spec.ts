import { test, expect } from '@playwright/test';

/**
 * REAL End-to-End Integration Tests
 * Tests against actual running Nuclear AO3 services
 * These tests validate the entire system working together
 */

const BASE_URL = 'http://localhost:3001';
const API_BASE = 'http://localhost:8080';

// Test user credentials (should match test data)
const TEST_USER = {
  email: 'testuser30d_v2@example.com',
  password: 'TestPassword123!',
  username: 'testuser30d_v2'
};

test.describe('Nuclear AO3 Integration Tests', () => {
  
  test.beforeEach(async ({ page }) => {
    // Set longer timeout for real API calls
    test.setTimeout(60000);
    
    // Start fresh for each test
    await page.goto(BASE_URL);
  });

  test('Complete User Journey: Browse → Login → Create Work → Publish', async ({ page }) => {
    // 1. Browse works as anonymous user
    await page.goto(`${BASE_URL}/works`);
    await expect(page.getByRole('heading', { name: /works/i })).toBeVisible();
    
    // 2. Navigate to login
    await page.click('a[href="/login"]');
    await expect(page).toHaveURL(`${BASE_URL}/login`);
    
    // 3. Login with test credentials
    await page.fill('input[name="email"]', TEST_USER.email);
    await page.fill('input[name="password"]', TEST_USER.password);
    await page.click('button[type="submit"]');
    
    // Should redirect to dashboard after login
    await expect(page).toHaveURL(`${BASE_URL}/dashboard`);
    await expect(page.locator(`text=${TEST_USER.username}`)).toBeVisible();
    
    // 4. Navigate to create new work
    await page.click('a[href="/works/new"]');
    await expect(page).toHaveURL(`${BASE_URL}/works/new`);
    
    // 5. Fill out work creation form
    const timestamp = Date.now();
    const workTitle = `E2E Test Work ${timestamp}`;
    const workSummary = 'This is an end-to-end test work created by automated testing.';
    const workContent = 'This is the content of our test work. It has multiple paragraphs.\\n\\nThis is the second paragraph to test formatting.';
    
    await page.fill('input[name="title"]', workTitle);
    await page.fill('textarea[name="summary"]', workSummary);
    await page.fill('textarea[name="content"]', workContent);
    
    // Select required tags
    await page.selectOption('select[name="rating"]', 'General Audiences');
    await page.fill('input[name="fandoms"]', 'Test Fandom');
    await page.selectOption('select[name="category"]', 'Gen');
    await page.selectOption('select[name="warnings"]', 'No Archive Warnings Apply');
    
    // Save as draft first
    await page.click('button:has-text("Save Draft")');
    
    // Should redirect to draft work page
    await expect(page.locator(`text=${workTitle}`)).toBeVisible();
    await expect(page.locator('text=Draft')).toBeVisible();
    
    // 6. Publish the work
    await page.click('button:has-text("Publish")');
    
    // Should show published work
    await expect(page.locator('text=Published')).toBeVisible();
    await expect(page.locator(`text=${workTitle}`)).toBeVisible();
    await expect(page.locator(`text=${workContent.split('\\n')[0]}`)).toBeVisible();
  });

  test('Search and Filter Workflow', async ({ page }) => {
    // 1. Go to works page
    await page.goto(`${BASE_URL}/works`);
    
    // 2. Test basic search
    await page.fill('input[placeholder*="Search"]', 'test');
    await page.click('button:has-text("Search")');
    
    // Should show search results
    await expect(page.locator('.work-card, [data-testid="work-card"]')).toBeVisible();
    
    // 3. Test filters
    await page.selectOption('select[name="rating"]', 'General Audiences');
    await page.click('button:has-text("Apply Filters")');
    
    // Should filter results
    await expect(page.locator('text=General Audiences')).toBeVisible();
    
    // 4. Test pagination if available
    const nextButton = page.locator('button:has-text("Next")');
    if (await nextButton.isVisible()) {
      await nextButton.click();
      await expect(page.getByRole('heading', { name: /works/i })).toBeVisible();
    }
  });

  test('Series Creation and Management', async ({ page }) => {
    // Login first
    await page.goto(`${BASE_URL}/login`);
    await page.fill('input[name="email"]', TEST_USER.email);
    await page.fill('input[name="password"]', TEST_USER.password);
    await page.click('button[type="submit"]');
    await expect(page).toHaveURL(`${BASE_URL}/dashboard`);
    
    // Navigate to series page
    await page.goto(`${BASE_URL}/series`);
    await expect(page.getByRole('heading', { name: /series/i })).toBeVisible();
    
    // Go to My Series tab
    await page.click('text=My Series');
    
    // Create new series
    await page.click('button:has-text("Create Series")');
    await expect(page).toHaveURL(`${BASE_URL}/series/new`);
    
    // Fill series form
    const timestamp = Date.now();
    const seriesTitle = `E2E Test Series ${timestamp}`;
    const seriesSummary = 'This is a test series created by end-to-end testing.';
    
    await page.fill('input[id="title"]', seriesTitle);
    await page.fill('textarea[id="summary"]', seriesSummary);
    
    // Add any available works
    const workCheckboxes = page.locator('input[type="checkbox"][id*="work-"]');
    const checkboxCount = await workCheckboxes.count();
    if (checkboxCount > 0) {
      await workCheckboxes.first().check();
    }
    
    // Create series
    await page.click('button:has-text("Create Series")');
    
    // Should redirect to new series page
    await expect(page.locator(`text=${seriesTitle}`)).toBeVisible();
    await expect(page.locator(`text=${seriesSummary}`)).toBeVisible();
  });

  test('Bookmark Workflow', async ({ page }) => {
    // Login first
    await page.goto(`${BASE_URL}/login`);
    await page.fill('input[name="email"]', TEST_USER.email);
    await page.fill('input[name="password"]', TEST_USER.password);
    await page.click('button[type="submit"]');
    await expect(page).toHaveURL(`${BASE_URL}/dashboard`);
    
    // Find a work to bookmark
    await page.goto(`${BASE_URL}/works`);
    const firstWork = page.locator('.work-card, [data-testid="work-card"]').first();
    
    if (await firstWork.isVisible()) {
      await firstWork.click();
      
      // Check if bookmark button exists
      const bookmarkButton = page.locator('button:has-text("Bookmark")');
      if (await bookmarkButton.isVisible()) {
        await bookmarkButton.click();
        
        // Fill bookmark form
        await page.fill('textarea[placeholder*="notes"]', 'Great story! Bookmarked via E2E test.');
        await page.fill('input[placeholder*="tags"]', 'test-bookmark, automated-test');
        
        // Submit bookmark
        await page.click('button:has-text("Add Bookmark")');
        
        // Should show bookmarked state
        await expect(page.locator('button:has-text("Bookmarked")')).toBeVisible();
      }
    }
    
    // Visit bookmarks page
    await page.goto(`${BASE_URL}/bookmarks`);
    await expect(page.getByRole('heading', { name: /bookmarks/i })).toBeVisible();
    
    // Should show our bookmark
    await expect(page.locator('text=Great story!')).toBeVisible();
    await expect(page.locator('text=test-bookmark')).toBeVisible();
  });

  test('Tag System Integration', async ({ page }) => {
    // Test tag autocomplete and hierarchy
    await page.goto(`${BASE_URL}/works/new`);
    
    // Test fandom autocomplete
    await page.fill('input[name="fandoms"]', 'Harry');
    await page.waitForTimeout(500); // Wait for autocomplete
    
    // Check if autocomplete suggestions appear
    const suggestions = page.locator('.autocomplete-suggestion, .suggestion');
    if (await suggestions.isVisible()) {
      await suggestions.first().click();
    }
    
    // Test character tags
    await page.fill('input[name="characters"]', 'Test');
    await page.waitForTimeout(500);
    
    // Type and see if tags are being processed
    await page.fill('input[name="freeform_tags"]', 'fluff, angst, hurt/comfort');
  });

  test('Authentication and Authorization', async ({ page }) => {
    // Test accessing protected pages without auth
    await page.goto(`${BASE_URL}/works/new`);
    
    // Should redirect to login or show auth required message
    const loginRequired = page.locator('text=Please log in');
    const redirectedToLogin = page.url().includes('/login');
    
    expect(await loginRequired.isVisible() || redirectedToLogin).toBeTruthy();
    
    // Test successful login
    if (redirectedToLogin) {
      await page.fill('input[name="email"]', TEST_USER.email);
      await page.fill('input[name="password"]', TEST_USER.password);
      await page.click('button[type="submit"]');
      
      // Should now be able to access protected content
      await page.goto(`${BASE_URL}/works/new`);
      await expect(page.getByRole('heading', { name: /new work/i })).toBeVisible();
    }
    
    // Test logout
    await page.click('button:has-text("Logout"), a:has-text("Logout")');
    
    // Should be logged out
    await expect(page.locator('a[href="/login"]')).toBeVisible();
  });

  test('Error Handling and Edge Cases', async ({ page }) => {
    // Test 404 page
    await page.goto(`${BASE_URL}/works/nonexistent-work-id`);
    await expect(page.locator('text=Not Found')).toBeVisible();
    
    // Test invalid search
    await page.goto(`${BASE_URL}/works?q=<script>alert("xss")</script>`);
    await expect(page.getByRole('heading', { name: /works/i })).toBeVisible();
    
    // Test form validation
    await page.goto(`${BASE_URL}/login`);
    await page.click('button[type="submit"]'); // Submit empty form
    
    // Should show validation errors
    const emailRequired = page.locator('input[name="email"]:invalid');
    const passwordRequired = page.locator('input[name="password"]:invalid');
    
    expect(await emailRequired.isVisible() || await passwordRequired.isVisible()).toBeTruthy();
  });

  test('Performance and Loading States', async ({ page }) => {
    // Test loading states
    await page.goto(`${BASE_URL}/works`);
    
    // Check that content loads within reasonable time
    await expect(page.getByRole('heading', { name: /works/i })).toBeVisible({ timeout: 10000 });
    
    // Test that pagination works smoothly
    const nextButton = page.locator('button:has-text("Next")');
    if (await nextButton.isVisible()) {
      const startTime = Date.now();
      await nextButton.click();
      await expect(page.getByRole('heading', { name: /works/i })).toBeVisible();
      const loadTime = Date.now() - startTime;
      
      // Should load within 5 seconds
      expect(loadTime).toBeLessThan(5000);
    }
  });

  test('Cross-Feature Integration', async ({ page }) => {
    // Test the full flow of creating a work, adding it to a series, and bookmarking it
    
    // Login
    await page.goto(`${BASE_URL}/login`);
    await page.fill('input[name="email"]', TEST_USER.email);
    await page.fill('input[name="password"]', TEST_USER.password);
    await page.click('button[type="submit"]');
    
    // Create a work
    await page.goto(`${BASE_URL}/works/new`);
    const timestamp = Date.now();
    await page.fill('input[name="title"]', `Integration Test Work ${timestamp}`);
    await page.fill('textarea[name="summary"]', 'Cross-feature integration test');
    await page.fill('textarea[name="content"]', 'Test content for integration');
    await page.selectOption('select[name="rating"]', 'General Audiences');
    await page.fill('input[name="fandoms"]', 'Test Fandom');
    
    await page.click('button:has-text("Publish")');
    
    // Work should be created and visible
    await expect(page.locator(`text=Integration Test Work ${timestamp}`)).toBeVisible();
    
    // Bookmark the work we just created
    const bookmarkButton = page.locator('button:has-text("Bookmark")');
    if (await bookmarkButton.isVisible()) {
      await bookmarkButton.click();
      await page.fill('textarea[placeholder*="notes"]', 'Self-bookmark test');
      await page.click('button:has-text("Add Bookmark")');
      await expect(page.locator('button:has-text("Bookmarked")')).toBeVisible();
    }
    
    // Verify the work appears in search
    await page.goto(`${BASE_URL}/works`);
    await page.fill('input[placeholder*="Search"]', `Integration Test Work ${timestamp}`);
    await page.click('button:has-text("Search")');
    await expect(page.locator(`text=Integration Test Work ${timestamp}`)).toBeVisible();
    
    // Verify bookmark appears in bookmarks page
    await page.goto(`${BASE_URL}/bookmarks`);
    await expect(page.locator(`text=Integration Test Work ${timestamp}`)).toBeVisible();
    await expect(page.locator('text=Self-bookmark test')).toBeVisible();
  });

});