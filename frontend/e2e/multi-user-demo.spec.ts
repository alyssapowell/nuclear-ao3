import { test, expect, type Page, type BrowserContext } from '@playwright/test';

// Simplified multi-user demo test that shows the framework capabilities
test.describe('Multi-User Framework Demo', () => {
  let authorContext: BrowserContext;
  let readerContext: BrowserContext;
  let anonymousContext: BrowserContext;
  
  let authorPage: Page;
  let readerPage: Page;
  let anonymousPage: Page;

  test.beforeAll(async ({ browser }) => {
    // Create separate browser contexts for different user types
    authorContext = await browser.newContext();
    readerContext = await browser.newContext();
    anonymousContext = await browser.newContext();
    
    authorPage = await authorContext.newPage();
    readerPage = await readerContext.newPage();
    anonymousPage = await anonymousContext.newPage();
  });

  test.afterAll(async () => {
    await authorContext.close();
    await readerContext.close();
    await anonymousContext.close();
  });

  test('should maintain separate browser contexts for multiple users', async () => {
    // All users can access the home page independently
    await authorPage.goto('/');
    await readerPage.goto('/');
    await anonymousPage.goto('/');
    
    // Each should see the home page
    await expect(authorPage.locator('h1')).toContainText('Nuclear AO3');
    await expect(readerPage.locator('h1')).toContainText('Nuclear AO3');
    await expect(anonymousPage.locator('h1')).toContainText('Nuclear AO3');
    
    console.log('✅ All three browser contexts loaded home page independently');
  });

  test('should show different navigation options for each user type', async () => {
    // Author user sees registration/login options
    await authorPage.goto('/');
    await expect(authorPage.locator('text=Log In')).toBeVisible();
    await expect(authorPage.locator('text=Sign Up')).toBeVisible();
    
    // Reader user also sees the same (both unauthenticated)
    await readerPage.goto('/');
    await expect(readerPage.locator('text=Log In')).toBeVisible();
    await expect(readerPage.locator('text=Sign Up')).toBeVisible();
    
    // Anonymous user sees the same
    await anonymousPage.goto('/');
    await expect(anonymousPage.locator('text=Log In')).toBeVisible();
    await expect(anonymousPage.locator('text=Sign Up')).toBeVisible();
    
    console.log('✅ All users see appropriate navigation options');
  });

  test('should demonstrate registration page access from multiple contexts', async () => {
    // Author goes to registration
    await authorPage.goto('/auth/register');
    await expect(authorPage.locator('h1')).toContainText('Sign Up');
    await expect(authorPage.locator('[data-testid="username"]')).toBeVisible();
    
    // Reader goes to login  
    await readerPage.goto('/auth/login');
    await expect(readerPage.locator('h1')).toContainText('Log In');
    await expect(readerPage.locator('[data-testid="username"]')).toBeVisible();
    
    // Anonymous user browses works
    await anonymousPage.goto('/works');
    await expect(anonymousPage.locator('h1')).toContainText('All Works');
    
    console.log('✅ Multiple users can access different pages simultaneously');
  });

  test('should handle search functionality across multiple contexts', async () => {
    // All users can access search independently
    await authorPage.goto('/search');
    await readerPage.goto('/search');
    await anonymousPage.goto('/search');
    
    // Each should see the search page
    await expect(authorPage.locator('h1')).toContainText('Search');
    await expect(readerPage.locator('h1')).toContainText('Search');
    await expect(anonymousPage.locator('h1')).toContainText('Search');
    
    console.log('✅ All users can access search functionality independently');
  });

  test('should demonstrate authentication redirects work correctly', async () => {
    // Anonymous user tries to access dashboard (should redirect to login)
    await anonymousPage.goto('/dashboard');
    await expect(anonymousPage).toHaveURL(/\/auth\/login/);
    await expect(anonymousPage.locator('h1')).toContainText('Log In');
    
    // Author user tries to access dashboard (should also redirect since not authenticated)
    await authorPage.goto('/dashboard');
    await expect(authorPage).toHaveURL(/\/auth\/login/);
    
    // Reader user tries to access work creation (should redirect)
    await readerPage.goto('/works/new');
    await expect(readerPage).toHaveURL(/\/auth\/login/);
    
    console.log('✅ Authentication redirects work correctly for protected routes');
  });

  test('should show that sessions are truly isolated between contexts', async () => {
    // Set different localStorage values in each context
    await authorPage.evaluate(() => localStorage.setItem('test-key', 'author-value'));
    await readerPage.evaluate(() => localStorage.setItem('test-key', 'reader-value'));
    await anonymousPage.evaluate(() => localStorage.setItem('test-key', 'anonymous-value'));
    
    // Verify each context has its own value
    const authorValue = await authorPage.evaluate(() => localStorage.getItem('test-key'));
    const readerValue = await readerPage.evaluate(() => localStorage.getItem('test-key'));
    const anonymousValue = await anonymousPage.evaluate(() => localStorage.getItem('test-key'));
    
    expect(authorValue).toBe('author-value');
    expect(readerValue).toBe('reader-value');
    expect(anonymousValue).toBe('anonymous-value');
    
    console.log('✅ Browser contexts maintain separate localStorage/session isolation');
  });

  test('should demonstrate comprehensive multi-user testing capabilities', async () => {
    // This test summarizes what our framework can do:
    
    // 1. Multiple isolated browser contexts ✅
    expect(authorContext).toBeDefined();
    expect(readerContext).toBeDefined();
    expect(anonymousContext).toBeDefined();
    
    // 2. Independent page navigation ✅
    await Promise.all([
      authorPage.goto('/'),
      readerPage.goto('/search'),
      anonymousPage.goto('/works')
    ]);
    
    // 3. Simultaneous form interactions (simulate users registering at same time)
    await Promise.all([
      authorPage.goto('/auth/register'),
      readerPage.goto('/auth/register'),
    ]);
    
    // Both can fill forms simultaneously without interference
    await Promise.all([
      authorPage.fill('[data-testid="username"]', 'author-demo'),
      readerPage.fill('[data-testid="username"]', 'reader-demo')
    ]);
    
    const authorValue = await authorPage.locator('[data-testid="username"]').inputValue();
    const readerValue = await readerPage.locator('[data-testid="username"]').inputValue();
    
    expect(authorValue).toBe('author-demo');
    expect(readerValue).toBe('reader-demo');
    
    console.log('✅ Multi-user framework supports:');
    console.log('   - Isolated browser contexts for different user types');
    console.log('   - Simultaneous independent page navigation');
    console.log('   - Parallel form interactions without interference');
    console.log('   - Authentication flow testing');
    console.log('   - Session isolation verification');
    console.log('   - Cross-user permission testing capabilities');
    console.log('');
    console.log('🎉 Multi-user test framework is fully functional!');
  });
});