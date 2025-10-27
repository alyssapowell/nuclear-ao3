import { test, expect } from '@playwright/test';

test.describe('Work Creation Flow - Complete End-to-End Test', () => {
  test('should create a work with JWT authentication', async ({ page }) => {
    // Go to login page
    await page.goto('/login');
    
    // Login with our test credentials
    await page.fill('input[name="email"], input[type="email"]', 'admin@nuclear-ao3.com');
    await page.fill('input[name="password"], input[type="password"]', 'adminpass123');
    
    // Submit login form
    await page.click('button[type="submit"], button:has-text("Login"), button:has-text("Sign In")');
    
    // Wait for successful login (should redirect or show success)
    await page.waitForURL(/dashboard|works|profile/, { timeout: 10000 });
    
    // Navigate to work creation page
    await page.goto('/works/new');
    
    // Wait for the page to load
    await expect(page.getByRole('heading', { name: /new work|create work|post new work/i })).toBeVisible();
    
    // Fill out the work creation form
    await page.fill('input[name="title"]', 'E2E Test Work - JWT & UserID Fix');
    
    // Fill summary if available
    const summaryField = page.locator('textarea[name="summary"], input[name="summary"]');
    if (await summaryField.isVisible()) {
      await summaryField.fill('This work was created through Playwright E2E testing to verify the complete JWT persistence and work creation pipeline.');
    }
    
    // Select rating
    const ratingSelect = page.locator('select[name="rating"]');
    if (await ratingSelect.isVisible()) {
      await ratingSelect.selectOption('General Audiences');
    }
    
    // Fill fandoms
    const fandomField = page.locator('input[name="fandoms"], [name*="fandom"]');
    if (await fandomField.first().isVisible()) {
      await fandomField.first().fill('Nuclear AO3 Testing');
    }
    
    // Fill characters if available
    const characterField = page.locator('input[name="characters"], [name*="character"]');
    if (await characterField.first().isVisible()) {
      await characterField.first().fill('Test Admin User');
    }
    
    // Add tags if available
    const tagsField = page.locator('input[name="tags"], input[name="freeform_tags"], [name*="tag"]');
    if (await tagsField.first().isVisible()) {
      await tagsField.first().fill('e2e-test, jwt-persistence, work-creation');
    }
    
    // Fill work content/chapter content
    const contentField = page.locator('textarea[name="content"], textarea[name="chapter_text"], [name*="content"]');
    if (await contentField.isVisible()) {
      await contentField.fill('This is the content of our E2E test work. It demonstrates that the entire JWT authentication and work creation pipeline is working correctly!');
    }
    
    // Submit the form
    const submitButton = page.locator('button[type="submit"], button:has-text("Post"), button:has-text("Create"), button:has-text("Save")');
    await expect(submitButton).toBeVisible();
    await submitButton.click();
    
    // Wait for successful work creation
    // Should either redirect to the work page or show success message
    await page.waitForTimeout(5000); // Give time for the API call
    
    // Check for success indicators
    const successIndicators = [
      page.locator(':has-text("Work created")'),
      page.locator(':has-text("Successfully")'),
      page.locator(':has-text("posted")'),
      page.getByRole('heading', { name: /E2E Test Work/ }),
    ];
    
    let success = false;
    for (const indicator of successIndicators) {
      if (await indicator.isVisible()) {
        success = true;
        break;
      }
    }
    
    // If not redirected to work page, check current URL
    const currentUrl = page.url();
    if (currentUrl.includes('/works/') && !currentUrl.includes('/new')) {
      success = true;
    }
    
    // Assert that work creation was successful
    expect(success).toBe(true);
    
    // Verify no error messages
    await expect(page.locator(':has-text("Error")')).not.toBeVisible();
    await expect(page.locator(':has-text("502")')).not.toBeVisible();
    await expect(page.locator(':has-text("Bad Gateway")')).not.toBeVisible();
  });
  
  test('should handle work creation errors gracefully', async ({ page }) => {
    // Go to login page first
    await page.goto('/login');
    
    // Login
    await page.fill('input[name="email"], input[type="email"]', 'admin@nuclear-ao3.com');
    await page.fill('input[name="password"], input[type="password"]', 'adminpass123');
    await page.click('button[type="submit"], button:has-text("Login"), button:has-text("Sign In")');
    
    // Wait for login
    await page.waitForTimeout(3000);
    
    // Navigate to work creation
    await page.goto('/works/new');
    
    // Try to submit without required fields
    const submitButton = page.locator('button[type="submit"], button:has-text("Post"), button:has-text("Create"), button:has-text("Save")');
    if (await submitButton.isVisible()) {
      await submitButton.click();
      
      // Should show validation errors or stay on the same page
      await expect(page.getByRole('heading', { name: /new work|create work|post new work/i })).toBeVisible();
    }
  });
});