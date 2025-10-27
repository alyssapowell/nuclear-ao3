import { test, expect } from '@playwright/test';

test.describe('Fixed Work Creation Test', () => {
  test('should create work successfully with correct rating format', async ({ page }) => {
    // Navigate to login page
    await page.goto('/auth/login');
    
    // Fill login form
    await page.fill('#email', 'admin@nuclear-ao3.com');
    await page.fill('#password', 'adminpass123');
    
    // Submit login
    await page.click('button[type="submit"]');
    
    // Wait for successful login (should redirect)
    await page.waitForTimeout(3000);
    
    // Navigate to work creation page
    await page.goto('/works/new');
    
    // Wait for the page to load
    await expect(page.getByRole('heading', { name: 'Post New Work' })).toBeVisible();
    
    // Fill required fields
    await page.fill('input[name="title"]', 'Fixed Frontend Test Work');
    await page.fill('textarea[name="chapterContent"]', 'This work tests the fixed frontend with correct rating format.');
    
    // Add a fandom (required)
    const fandomInput = page.getByPlaceholder('Add fandom...');
    await fandomInput.fill('Frontend Testing');
    await fandomInput.press('Enter');
    
    // Select "General Audiences" rating - this should now send the correct format
    await page.check('input[value="General Audiences"]');
    
    // Select a category
    await page.check('input[value="Gen"]');
    
    // Select a warning
    await page.check('input[value="No Archive Warnings Apply"]');
    
    // Listen for network requests to see what's being sent
    let requestData = null;
    page.on('request', request => {
      if (request.url().includes('/api/v1/works') && request.method() === 'POST') {
        requestData = request.postData();
        console.log('Work creation request data:', requestData);
      }
    });
    
    // Listen for responses to see what comes back
    page.on('response', response => {
      if (response.url().includes('/api/v1/works') && response.request().method() === 'POST') {
        console.log(`Work creation response: ${response.status()} ${response.statusText()}`);
      }
    });
    
    // Submit the form - try different approach to avoid click interception
    await page.locator('button[type="submit"]').click({ force: true });
    
    // Wait for response
    await page.waitForTimeout(3000);
    
    // Check for success or error
    const currentUrl = page.url();
    console.log('Current URL after submission:', currentUrl);
    
    // Check if we were redirected to a work page (success)
    const isSuccess = currentUrl.includes('/works/') && !currentUrl.includes('/new');
    
    if (isSuccess) {
      console.log('SUCCESS: Work created successfully!');
      // Verify we're on the work page
      await expect(page.getByRole('heading', { name: /Fixed Frontend Test Work/i })).toBeVisible();
    } else {
      // Check for any error messages
      const errorElement = page.locator('.bg-red-50, .text-red-700, [class*="error"]');
      if (await errorElement.isVisible()) {
        const errorText = await errorElement.textContent();
        console.log('Error message:', errorText);
      }
      
      // Take screenshot for debugging
      await page.screenshot({ path: 'work-creation-error.png' });
      
      console.log('Request data that was sent:', requestData);
    }
    
    // The test should pass if we successfully created the work
    expect(isSuccess).toBe(true);
  });
});