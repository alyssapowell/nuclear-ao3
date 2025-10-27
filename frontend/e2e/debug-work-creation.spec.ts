import { test, expect } from '@playwright/test';

test.describe('Debug Work Creation', () => {
  test('should debug form submission issues', async ({ page }) => {
    // Navigate to login page
    await page.goto('/auth/login');
    
    // Fill login form
    await page.fill('#email', 'admin@nuclear-ao3.com');
    await page.fill('#password', 'adminpass123');
    
    // Submit login
    await page.click('button[type="submit"]');
    
    // Wait for successful login (should redirect)
    await page.waitForTimeout(3000);
    console.log('After login URL:', page.url());
    
    // Check if auth token is stored
    const authToken = await page.evaluate(() => localStorage.getItem('auth_token'));
    console.log('Auth token present:', !!authToken);
    console.log('Auth token length:', authToken ? authToken.length : 0);
    
    // Navigate to work creation page
    await page.goto('/works/new');
    await page.waitForTimeout(2000);
    
    // Check for JavaScript errors immediately after page load
    const jsErrors = [];
    page.on('pageerror', error => {
      jsErrors.push(error.message);
    });
    
    await page.waitForTimeout(1000);
    console.log('JavaScript errors on page:', jsErrors);
    
    // Wait for the page to load
    await expect(page.getByRole('heading', { name: 'Post New Work' })).toBeVisible();
    console.log('Work creation page loaded');
    
    // Fill only the required fields
    console.log('Filling title field...');
    await page.fill('input[name="title"]', 'Debug Test Work');
    
    console.log('Filling chapter content...');
    await page.fill('textarea[name="chapterContent"]', 'Debug test content.');
    
    console.log('Adding fandom...');
    const fandomInput = page.getByPlaceholder('Add fandom...');
    await fandomInput.fill('Debug Testing');
    await fandomInput.press('Enter');
    await page.waitForTimeout(1000);
    
    console.log('Selecting rating...');
    await page.check('input[value="General Audiences"]');
    
    console.log('Selecting category...');
    await page.check('input[value="Gen"]');
    
    console.log('Selecting warning...');
    await page.check('input[value="No Archive Warnings Apply"]');
    
    // Debug: Check form state before submission
    const formTitle = await page.inputValue('input[name="title"]');
    const formContent = await page.inputValue('textarea[name="chapterContent"]');
    console.log('Form state - Title:', formTitle);
    console.log('Form state - Content:', formContent);
    
    // Check if fandom was added properly
    const fandomTags = await page.locator('.inline-flex.items-center').count();
    console.log('Number of fandom tags:', fandomTags);
    
    // Check form data in browser
    const formDataInBrowser = await page.evaluate(() => {
      const form = document.querySelector('form');
      const formData = new FormData(form);
      const data = Object.fromEntries(formData.entries());
      return data;
    });
    console.log('Browser form data:', formDataInBrowser);
    
    // Check if submit button is enabled
    const submitButton = page.locator('button[type="submit"]');
    const isEnabled = await submitButton.isEnabled();
    console.log('Submit button enabled:', isEnabled);
    
    // Listen for any console logs and errors
    page.on('console', msg => {
      console.log(`Browser console (${msg.type()}):`, msg.text());
    });
    
    // Listen for network requests
    let requestMade = false;
    page.on('request', request => {
      if (request.url().includes('/api/v1/works') && request.method() === 'POST') {
        requestMade = true;
        console.log('Work creation request made!');
        console.log('Request URL:', request.url());
        console.log('Request data:', request.postData());
      }
    });
    
    // Try to submit the form - first try button click, then form submit
    console.log('Clicking submit button...');
    await submitButton.click({ force: true });
    
    await page.waitForTimeout(1000);
    
    // If that didn't work, try submitting the form directly
    console.log('Also trying direct form submission...');
    await page.evaluate(() => {
      const form = document.querySelector('form');
      if (form) {
        form.dispatchEvent(new Event('submit', { cancelable: true, bubbles: true }));
      }
    });
    
    // Check if the handleSubmit function exists
    const hasHandleSubmit = await page.evaluate(() => {
      const form = document.querySelector('form');
      return form && form.onsubmit !== null;
    });
    console.log('Form has onsubmit handler:', hasHandleSubmit);
    
    // Wait and check
    await page.waitForTimeout(5000);
    console.log('Request was made:', requestMade);
    console.log('Final URL:', page.url());
    
    // Just pass for now - we're debugging
    expect(true).toBe(true);
  });
});