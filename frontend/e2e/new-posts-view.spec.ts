import { test, expect } from '@playwright/test';

// Helper function to authenticate user
async function authenticateUser(page) {
  // Set auth token in localStorage and cookies to simulate logged-in user
  await page.addInitScript(() => {
    localStorage.setItem('auth_token', 'test-token-12345');
    localStorage.setItem('user', JSON.stringify({
      id: 'test-user-123',
      username: 'testuser',
      email: 'test@example.com'
    }));
  });
  
  // Also set the cookie that middleware checks
  await page.context().addCookies([{
    name: 'auth_token',
    value: 'test-token-12345',
    domain: 'localhost',
    path: '/'
  }]);
}

test.describe('New Posts View - Unauthenticated', () => {
  test('should redirect to login when accessing /works/new without authentication', async ({ page }) => {
    test.setTimeout(30000);
    
    await page.goto('/works/new');
    
    // Should be redirected to login page
    await page.waitForSelector('main h1', { timeout: 15000 });
    await expect(page.locator('main h1')).toContainText('Log In');
    
    // Should have redirect parameter
    expect(page.url()).toContain('/auth/login');
    expect(page.url()).toContain('redirect=%2Fworks%2Fnew');
  });
});

test.describe('New Posts View - Authenticated', () => {
  test.beforeEach(async ({ page }) => {
    await authenticateUser(page);
  });

  test('should load new work creation page when authenticated', async ({ page }) => {
    test.setTimeout(30000);
    
    await page.goto('/works/new');
    
    // Wait for the page to load
    await page.waitForSelector('main h1', { timeout: 15000 });
    
    // Check if the page has the expected title
    await expect(page.locator('main h1')).toContainText('Post New Work');
    
    // Check for basic form elements
    await expect(page.locator('input[name="title"]')).toBeVisible();
    await expect(page.locator('textarea[name="summary"]')).toBeVisible();
    
    // Check for submit button
    await expect(page.locator('button[type="submit"]')).toBeVisible();
  });

  test('should show form fields in new work creation', async ({ page }) => {
    test.setTimeout(30000);
    
    await page.goto('/works/new');
    
    // Wait for the page to load
    await page.waitForSelector('main h1', { timeout: 15000 });
    
    // Check for required fields
    await expect(page.locator('input[name="title"]')).toBeVisible();
    await expect(page.locator('textarea[name="summary"]')).toBeVisible();
    await expect(page.locator('textarea[name="chapterContent"]')).toBeVisible();
    
    // Check for TagAutocomplete field
    await expect(page.locator('input[id="fandoms"]')).toBeVisible();
    
    // The current simple form should have these basic elements
    const pageContent = await page.textContent('body');
    expect(pageContent).toContain('Post New Work');
    expect(pageContent).toContain('Fandoms (Test TagAutocomplete)');
  });

  test('should allow typing in TagAutocomplete field', async ({ page }) => {
    test.setTimeout(30000);
    
    await page.goto('/works/new');
    
    // Wait for the page to load
    await page.waitForSelector('input[id="fandoms"]', { timeout: 15000 });
    
    // Type in the TagAutocomplete field
    await page.fill('input[id="fandoms"]', 'Harry Potter');
    
    // Verify the text was entered
    const inputValue = await page.inputValue('input[id="fandoms"]');
    expect(inputValue).toBe('Harry Potter');
  });

  test('should show autocomplete suggestions when typing', async ({ page }) => {
    test.setTimeout(30000);
    
    // Mock the API response to test suggestion functionality
    await page.route('**/api/v1/tags/search*', async route => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify([
          { name: 'Harry Potter - J. K. Rowling', type: 'fandom', use_count: 12000 },
          { name: 'Harry Potter', type: 'fandom', use_count: 8000 },
          { name: 'Harry Potter & the Cursed Child', type: 'fandom', use_count: 1500 }
        ])
      });
    });
    
    await page.goto('/works/new');
    
    // Wait for the page to load
    await page.waitForSelector('input[id="fandoms"]', { timeout: 15000 });
    
    // Type something that should trigger autocomplete
    await page.type('input[id="fandoms"]', 'Harry');
    
    // Wait for API call and debounce (TagAutocomplete has 150ms debounce)
    await page.waitForTimeout(500);
    
    // Look for the suggestions listbox that TagAutocomplete renders
    const suggestionsListbox = page.locator('[role="listbox"]');
    
    // With mocked data, suggestions should appear
    await expect(suggestionsListbox).toBeVisible();
    
    // Look for individual suggestion options
    const suggestionOptions = page.locator('[role="option"]');
    const optionCount = await suggestionOptions.count();
    
    // Should have the 3 mocked suggestions
    expect(optionCount).toBe(3);
    
    // Verify suggestion content
    await expect(suggestionOptions.first()).toContainText('Harry Potter');
    
    console.log(`Found ${optionCount} autocomplete suggestions with mocked data`);
    
    // The input should still work
    const inputValue = await page.inputValue('input[id="fandoms"]');
    expect(inputValue).toBe('Harry');
  });
});

test.describe('Works Browse Page', () => {
  test('should load works browse page', async ({ page }) => {
    test.setTimeout(30000);
    
    await page.goto('/works');
    
    // Wait for the page to load
    await page.waitForSelector('body', { timeout: 15000 });
    
    // Should have some content indicating it's the works page
    const pageContent = await page.textContent('body');
    expect(pageContent).toBeTruthy();
    
    // Should not redirect to login (works browse should be public)
    expect(page.url()).toContain('/works');
    expect(page.url()).not.toContain('/auth/login');
  });
});