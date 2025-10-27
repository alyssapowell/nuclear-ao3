import { test, expect } from '@playwright/test';

test.describe('Enhanced User Dashboard', () => {
  let authToken: string;
  let userId: string;
  let testUser: any;
  
  test.beforeEach(async ({ page }) => {
    // Register and login a test user
    const timestamp = Date.now();
    testUser = {
      username: `testuser_${timestamp}`,
      email: `test_${timestamp}@example.com`,
      password: 'testpassword123'
    };

    // Register user
    const registerResponse = await page.request.post('http://localhost:8080/api/v1/auth/register', {
      data: testUser
    });
    expect(registerResponse.ok()).toBeTruthy();

    // Login user
    const loginResponse = await page.request.post('http://localhost:8080/api/v1/auth/login', {
      data: {
        username: testUser.username,
        password: testUser.password
      }
    });
    expect(loginResponse.ok()).toBeTruthy();
    
    const loginData = await loginResponse.json();
    authToken = loginData.token;
    userId = loginData.user.id;
    
    // Set auth token in browser
    await page.goto('http://localhost:3002');
    await page.evaluate((token) => {
      localStorage.setItem('auth_token', token);
      localStorage.setItem('token', token);
    }, authToken);

    // Create some test works for the dashboard
    await createTestWorks(page, authToken);
  });

  async function createTestWorks(page: any, token: string) {
    // Create a completed work
    await page.request.post('http://localhost:8080/api/v1/works', {
      headers: {
        'Authorization': `Bearer ${token}`,
        'Content-Type': 'application/json',
      },
      data: {
        title: 'Completed Test Work',
        summary: 'A completed work for dashboard testing',
        language: 'en',
        rating: 'general',
        fandoms: ['Test Fandom'],
        chapter_content: 'Content of completed work',
        is_complete: true
      }
    });

    // Create an in-progress work
    await page.request.post('http://localhost:8080/api/v1/works', {
      headers: {
        'Authorization': `Bearer ${token}`,
        'Content-Type': 'application/json',
      },
      data: {
        title: 'In Progress Test Work',
        summary: 'An in-progress work for dashboard testing',
        language: 'en',
        rating: 'teen',
        fandoms: ['Test Fandom'],
        max_chapters: 3,
        chapter_content: 'First chapter content',
        is_complete: false
      }
    });

    // Create a draft work
    await page.request.post('http://localhost:8080/api/v1/works', {
      headers: {
        'Authorization': `Bearer ${token}`,
        'Content-Type': 'application/json',
      },
      data: {
        title: 'Draft Test Work',
        summary: 'A draft work for dashboard testing',
        language: 'en',
        rating: 'mature',
        fandoms: ['Test Fandom'],
        chapter_content: 'Draft content',
        is_draft: true
      }
    });
  }

  test('should display dashboard with user statistics', async ({ page }) => {
    await page.goto('http://localhost:3002/dashboard');
    
    // Verify dashboard loads
    await expect(page.locator('h1')).toContainText('Dashboard');
    await expect(page.locator(`text=Welcome back, ${testUser.username}`)).toBeVisible();
    
    // Verify statistics section
    await expect(page.locator('[data-testid="stats-section"]')).toBeVisible();
    
    // Verify individual stat cards
    await expect(page.locator('[data-testid="total-works-stat"]')).toBeVisible();
    await expect(page.locator('[data-testid="total-kudos-stat"]')).toBeVisible();
    await expect(page.locator('[data-testid="total-bookmarks-stat"]')).toBeVisible();
    await expect(page.locator('[data-testid="total-comments-stat"]')).toBeVisible();
    
    // Verify work counts are displayed correctly (should show 3 works)
    await expect(page.locator('[data-testid="total-works-stat"]')).toContainText('3');
  });

  test('should display recent works section', async ({ page }) => {
    await page.goto('http://localhost:3002/dashboard');
    
    // Verify recent works section
    await expect(page.locator('[data-testid="recent-works-section"]')).toBeVisible();
    await expect(page.locator('text=Recent Works')).toBeVisible();
    
    // Verify test works are displayed
    await expect(page.locator('text=Completed Test Work')).toBeVisible();
    await expect(page.locator('text=In Progress Test Work')).toBeVisible();
    await expect(page.locator('text=Draft Test Work')).toBeVisible();
    
    // Verify work metadata is shown
    await expect(page.locator('text=Complete')).toBeVisible();
    await expect(page.locator('text=In Progress')).toBeVisible();
    await expect(page.locator('text=Draft')).toBeVisible();
  });

  test('should provide quick actions for works', async ({ page }) => {
    await page.goto('http://localhost:3002/dashboard');
    
    // Wait for works to load
    await expect(page.locator('text=Completed Test Work')).toBeVisible();
    
    // Verify edit links are present
    const editLinks = page.locator('a:has-text("Edit")');
    await expect(editLinks).toHaveCount(3); // Should have edit links for all 3 works
    
    // Verify view links are present
    const viewLinks = page.locator('a:has-text("View")');
    await expect(viewLinks).toHaveCount(3); // Should have view links for all 3 works
    
    // Test clicking an edit link
    const firstEditLink = editLinks.first();
    await firstEditLink.click();
    
    // Should navigate to edit page
    await expect(page.url()).toMatch(/\/works\/[a-f0-9-]+\/edit$/);
  });

  test('should show activity feed', async ({ page }) => {
    await page.goto('http://localhost:3002/dashboard');
    
    // Verify activity section exists
    await expect(page.locator('[data-testid="activity-section"]')).toBeVisible();
    await expect(page.locator('text=Recent Activity')).toBeVisible();
    
    // Activity should show work creation events
    await expect(page.locator('text=Published')).toBeVisible();
  });

  test('should display work statistics correctly', async ({ page }) => {
    await page.goto('http://localhost:3002/dashboard');
    
    // Verify stat cards have proper structure
    const totalWorksCard = page.locator('[data-testid="total-works-stat"]');
    await expect(totalWorksCard.locator('.stat-number')).toBeVisible();
    await expect(totalWorksCard.locator('.stat-label')).toContainText('Total Works');
    
    const totalKudosCard = page.locator('[data-testid="total-kudos-stat"]');
    await expect(totalKudosCard.locator('.stat-number')).toBeVisible();
    await expect(totalKudosCard.locator('.stat-label')).toContainText('Total Kudos');
    
    const totalBookmarksCard = page.locator('[data-testid="total-bookmarks-stat"]');
    await expect(totalBookmarksCard.locator('.stat-number')).toBeVisible();
    await expect(totalBookmarksCard.locator('.stat-label')).toContainText('Total Bookmarks');
    
    const totalCommentsCard = page.locator('[data-testid="total-comments-stat"]');
    await expect(totalCommentsCard.locator('.stat-number')).toBeVisible();
    await expect(totalCommentsCard.locator('.stat-label')).toContainText('Total Comments');
  });

  test('should handle empty dashboard state', async ({ page }) => {
    // Create a new user with no works
    const timestamp = Date.now() + 10000;
    const emptyUser = {
      username: `emptyuser_${timestamp}`,
      email: `empty_${timestamp}@example.com`,
      password: 'testpassword123'
    };

    // Register and login empty user
    const registerResponse = await page.request.post('http://localhost:8080/api/v1/auth/register', {
      data: emptyUser
    });
    expect(registerResponse.ok()).toBeTruthy();

    const loginResponse = await page.request.post('http://localhost:8080/api/v1/auth/login', {
      data: {
        username: emptyUser.username,
        password: emptyUser.password
      }
    });
    expect(loginResponse.ok()).toBeTruthy();
    
    const loginData = await loginResponse.json();
    const emptyAuthToken = loginData.token;
    
    // Set auth token for empty user
    await page.evaluate((token) => {
      localStorage.setItem('auth_token', token);
      localStorage.setItem('token', token);
    }, emptyAuthToken);

    await page.goto('http://localhost:3002/dashboard');
    
    // Verify dashboard shows zero statistics
    await expect(page.locator('[data-testid="total-works-stat"]')).toContainText('0');
    await expect(page.locator('[data-testid="total-kudos-stat"]')).toContainText('0');
    await expect(page.locator('[data-testid="total-bookmarks-stat"]')).toContainText('0');
    await expect(page.locator('[data-testid="total-comments-stat"]')).toContainText('0');
    
    // Verify empty state message
    await expect(page.locator('text=No works yet')).toBeVisible();
    await expect(page.locator('a:has-text("Create your first work")')).toBeVisible();
  });

  test('should navigate to work creation from dashboard', async ({ page }) => {
    await page.goto('http://localhost:3002/dashboard');
    
    // Find and click the "Create New Work" button
    const createButton = page.locator('a:has-text("Create New Work")');
    await expect(createButton).toBeVisible();
    await createButton.click();
    
    // Should navigate to work creation page
    await expect(page.url()).toMatch(/\/works\/new$/);
    await expect(page.locator('text=Create New Work')).toBeVisible();
  });

  test('should show work progress for multi-chapter works', async ({ page }) => {
    await page.goto('http://localhost:3002/dashboard');
    
    // Verify the in-progress work shows chapter progress
    const inProgressWork = page.locator('text=In Progress Test Work').locator('..');
    await expect(inProgressWork.locator('text=1/3 chapters')).toBeVisible();
  });

  test('should handle responsive layout', async ({ page }) => {
    // Test mobile viewport
    await page.setViewportSize({ width: 375, height: 667 });
    await page.goto('http://localhost:3002/dashboard');
    
    // Dashboard should still be functional on mobile
    await expect(page.locator('h1')).toContainText('Dashboard');
    await expect(page.locator('[data-testid="stats-section"]')).toBeVisible();
    await expect(page.locator('[data-testid="recent-works-section"]')).toBeVisible();
    
    // Stats should be arranged vertically on mobile
    const statsSection = page.locator('[data-testid="stats-section"]');
    await expect(statsSection).toHaveCSS('flex-direction', 'column');
  });

  test('should update statistics in real-time', async ({ page }) => {
    await page.goto('http://localhost:3002/dashboard');
    
    // Get initial work count
    const initialCount = await page.locator('[data-testid="total-works-stat"] .stat-number').textContent();
    expect(initialCount).toBe('3');
    
    // Create a new work via API
    await page.request.post('http://localhost:8080/api/v1/works', {
      headers: {
        'Authorization': `Bearer ${authToken}`,
        'Content-Type': 'application/json',
      },
      data: {
        title: 'New Dashboard Test Work',
        summary: 'A new work to test real-time updates',
        language: 'en',
        rating: 'general',
        fandoms: ['Test Fandom'],
        chapter_content: 'New work content'
      }
    });
    
    // Refresh the page to see updated count
    await page.reload();
    
    // Work count should have increased
    await expect(page.locator('[data-testid="total-works-stat"] .stat-number')).toContainText('4');
    await expect(page.locator('text=New Dashboard Test Work')).toBeVisible();
  });

  test('should handle authentication requirement', async ({ page }) => {
    // Clear authentication
    await page.evaluate(() => {
      localStorage.removeItem('auth_token');
      localStorage.removeItem('token');
    });
    
    // Try to access dashboard
    await page.goto('http://localhost:3002/dashboard');
    
    // Should redirect to login or show login required message
    const currentUrl = page.url();
    const hasLoginForm = await page.locator('form').count() > 0;
    const hasLoginMessage = await page.locator('text=login').count() > 0;
    
    expect(currentUrl.includes('/login') || hasLoginForm || hasLoginMessage).toBeTruthy();
  });
});