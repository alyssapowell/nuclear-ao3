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