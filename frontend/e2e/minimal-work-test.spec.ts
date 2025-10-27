import { test, expect } from '@playwright/test';

test.describe('Minimal Work Creation Test', () => {
  test('should create a work via API', async ({ page }) => {
    // First, login to get a token
    await page.goto('http://localhost:3001/login');
    
    // Fill login form
    await page.fill('input[type="email"]', 'admin@nuclear-ao3.com');
    await page.fill('input[type="password"]', 'adminpass123');
    
    // Submit login
    await page.click('button[type="submit"]');
    
    // Wait for redirect or success
    await page.waitForTimeout(3000);
    
    // Get current URL to verify login
    const currentUrl = page.url();
    console.log('After login, current URL:', currentUrl);
    
    // Try to navigate to works/new page
    await page.goto('http://localhost:3001/works/new');
    
    // Wait for page to load
    await page.waitForTimeout(2000);
    
    // Check if we can see the work creation form
    const titleField = page.locator('input[name="title"]');
    const hasTitle = await titleField.isVisible();
    console.log('Title field visible:', hasTitle);
    
    if (hasTitle) {
      // Fill the form
      await titleField.fill('Playwright Test Work');
      
      // Find and fill other required fields
      const summaryField = page.locator('textarea[name="summary"]');
      if (await summaryField.isVisible()) {
        await summaryField.fill('This is a test work created by Playwright');
      }
      
      // Submit the form
      const submitButton = page.locator('button[type="submit"]');
      if (await submitButton.isVisible()) {
        console.log('Clicking submit button...');
        
        // Listen for network requests
        page.on('response', response => {
          if (response.url().includes('/api/v1/works')) {
            console.log(`Work creation response: ${response.status()} ${response.statusText()}`);
          }
        });
        
        await submitButton.click();
        
        // Wait for response
        await page.waitForTimeout(5000);
        
        // Check current page
        console.log('After submit, current URL:', page.url());
      }
    }
    
    // Take a screenshot for debugging
    await page.screenshot({ path: 'work-creation-debug.png' });
  });
});