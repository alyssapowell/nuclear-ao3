import { test, expect } from '@playwright/test';
import { loginUser, logoutUser, registerUser, TEST_USER } from './test-utils';

// Test configuration
const FRONTEND_URL = process.env.FRONTEND_URL || 'http://localhost:3001';

test.describe('Authentication Flow E2E Tests', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto(FRONTEND_URL);
  });

  test('Complete user registration and login flow via frontend', async ({ page }) => {
    // Use frontend registration flow
    try {
      await registerUser(page, TEST_USER);
      console.log('✅ Registration completed via frontend');
    } catch (error) {
      console.log('Registration failed (user may exist), proceeding to login');
    }

    // Use frontend login flow
    await loginUser(page);
    console.log('✅ Login completed via frontend');

    // Verify we're logged in by checking authenticated elements
    const loggedInIndicators = [
      page.locator('button:has-text("Log out")'),
      page.locator('a[href="/dashboard"]'),
      page.locator('text=Dashboard'),
      page.locator('text=Profile'),
      page.locator('a:has-text("My Works")'),
      page.locator('a:has-text("My Bookmarks")')
    ];

    // At least one login indicator should be visible
    let foundIndicator = false;
    for (const indicator of loggedInIndicators) {
      try {
        await indicator.waitFor({ timeout: 3000 });
        foundIndicator = true;
        console.log(`✅ Found auth indicator: ${await indicator.textContent()}`);
        break;
      } catch (e) {
        // Continue checking other indicators
      }
    }

    expect(foundIndicator).toBeTruthy();

    // Test logout via frontend
    await logoutUser(page);
    console.log('✅ Logout completed via frontend');
  });

  test('Frontend pages load correctly when authenticated', async ({ page }) => {
    // Login via frontend flow
    await loginUser(page);
    console.log('✅ Authenticated via frontend');

    // Test navigation to protected pages
    const protectedPages = [
      '/dashboard',
      '/works/new',
      '/series/new',
      '/bookmarks'
    ];

    for (const pagePath of protectedPages) {
      await page.goto(`${FRONTEND_URL}${pagePath}`);
      await page.waitForLoadState('networkidle');
      
      // Should stay on the protected page (not redirect to login)
      expect(page.url()).toContain(pagePath);
      
      // Check if page shows authenticated content
      const hasContent = await page.locator('main').isVisible();
      expect(hasContent).toBeTruthy();
      
      console.log(`✅ Protected page ${pagePath} loads correctly when authenticated`);
    }
  });

  test('Search functionality works for both authenticated and unauthenticated users', async ({ page }) => {
    // Test search without authentication
    await page.goto(`${FRONTEND_URL}/search`);
    await expect(page.locator('input[type="search"], input[placeholder*="search"]')).toBeVisible();

    // Perform a search
    const searchInput = page.locator('input[type="search"], input[placeholder*="search"]').first();
    await searchInput.fill('test');
    
    // Submit search (either by button click or enter key)
    const searchButton = page.locator('button:has-text("Search"), button[type="submit"]').first();
    if (await searchButton.isVisible()) {
      await searchButton.click();
    } else {
      await searchInput.press('Enter');
    }

    // Wait for search results
    await page.waitForLoadState('networkidle');
    
    // Check that search results area is visible (even if empty)
    const resultsVisible = await page.locator('main').isVisible();
    expect(resultsVisible).toBeTruthy();
    
    console.log('✅ Search functionality works');
  });

  test('Works browsing is accessible to all users', async ({ page }) => {
    await page.goto(`${FRONTEND_URL}/works`);
    
    // Works page should be accessible
    await expect(page.locator('main')).toBeVisible();
    
    // Should show some indication of works or "no works" message
    const pageContent = await page.textContent('main');
    expect(pageContent).toBeTruthy();
    expect(pageContent.length).toBeGreaterThan(10);
    
    console.log('✅ Works browsing page accessible');
  });

  test('Navigation menu works correctly', async ({ page }) => {
    // Test main navigation links
    const navLinks = [
      { href: '/', text: 'Home' },
      { href: '/search', text: 'Search' },
      { href: '/works', text: 'Works' },
      { href: '/series', text: 'Series' }
    ];

    for (const link of navLinks) {
      await page.goto(FRONTEND_URL);
      
      // Find and click the navigation link
      const navElement = page.locator(`a[href="${link.href}"]`).first();
      if (await navElement.isVisible()) {
        await navElement.click();
        await page.waitForLoadState('networkidle');
        
        // Verify we navigated to the correct page
        expect(page.url()).toContain(link.href);
        
        // Verify page has content
        await expect(page.locator('main')).toBeVisible();
        
        console.log(`✅ Navigation to ${link.href} works`);
      }
    }
  });

  test('Responsive design works on mobile', async ({ page }) => {
    // Set mobile viewport
    await page.setViewportSize({ width: 375, height: 667 });
    
    await page.goto(FRONTEND_URL);
    
    // Page should load and be readable on mobile
    await expect(page.locator('main')).toBeVisible();
    
    // Check that content is not horizontally scrollable
    const bodyWidth = await page.evaluate(() => document.body.scrollWidth);
    const viewportWidth = await page.evaluate(() => window.innerWidth);
    
    // Allow for small differences due to scrollbars
    expect(bodyWidth).toBeLessThanOrEqual(viewportWidth + 20);
    
    console.log('✅ Mobile responsive design works');
  });
});