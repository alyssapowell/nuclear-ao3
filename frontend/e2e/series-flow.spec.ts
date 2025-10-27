import { test, expect } from '@playwright/test';

// Helper function to set up authentication
async function setupAuth(page: any) {
  // Mock authentication token
  await page.evaluate(() => {
    localStorage.setItem('auth_token', 'test-auth-token');
  });
}

// Mock API responses
async function mockSeriesAPI(page: any) {
  // Mock series list API
  await page.route('**/api/v1/series?*', async route => {
    const url = new URL(route.request().url());
    const query = url.searchParams.get('q');
    
    if (query === 'test') {
      await route.fulfill({
        json: {
          series: [
            {
              id: 'series-1',
              title: 'Test Series',
              summary: 'A test series for E2E testing',
              user_id: 'user-1',
              username: 'testuser',
              is_complete: false,
              work_count: 2,
              word_count: 15000,
              created_at: '2024-01-01T00:00:00Z',
              updated_at: '2024-01-02T00:00:00Z',
            }
          ],
          pagination: {
            page: 1,
            limit: 20,
            total: 1,
            total_pages: 1,
          }
        }
      });
    } else {
      await route.fulfill({
        json: {
          series: [],
          pagination: {
            page: 1,
            limit: 20,
            total: 0,
            total_pages: 0,
          }
        }
      });
    }
  });

  // Mock individual series API
  await page.route('**/api/v1/series/series-1', async route => {
    await route.fulfill({
      json: {
        series: {
          id: 'series-1',
          title: 'Test Series',
          summary: 'A test series for E2E testing',
          notes: 'These are test notes',
          user_id: 'user-1',
          username: 'testuser',
          is_complete: false,
          work_count: 2,
          word_count: 15000,
          created_at: '2024-01-01T00:00:00Z',
          updated_at: '2024-01-02T00:00:00Z',
        }
      }
    });
  });

  // Mock series works API
  await page.route('**/api/v1/series/series-1/works', async route => {
    await route.fulfill({
      json: {
        works: [
          {
            id: 'work-1',
            title: 'First Work in Series',
            summary: 'First work summary',
            rating: 'General Audiences',
            category: ['Gen'],
            warnings: ['No Archive Warnings Apply'],
            fandoms: ['Test Fandom'],
            characters: ['Character A'],
            relationships: [],
            freeform_tags: ['Test Tag'],
            word_count: 5000,
            chapter_count: 1,
            is_complete: true,
            status: 'published',
            published_at: '2024-01-01T00:00:00Z',
            updated_at: '2024-01-01T00:00:00Z',
            hits: 100,
            kudos: 10,
            comments: 5,
            bookmarks: 3,
            username: 'testuser',
            position: 1,
          },
          {
            id: 'work-2',
            title: 'Second Work in Series',
            summary: 'Second work summary',
            rating: 'Teen And Up Audiences',
            category: ['F/M'],
            warnings: ['No Archive Warnings Apply'],
            fandoms: ['Test Fandom'],
            characters: ['Character A', 'Character B'],
            relationships: ['Character A/Character B'],
            freeform_tags: ['Romance'],
            word_count: 10000,
            chapter_count: 3,
            is_complete: false,
            status: 'published',
            published_at: '2024-01-02T00:00:00Z',
            updated_at: '2024-01-02T00:00:00Z',
            hits: 200,
            kudos: 25,
            comments: 15,
            bookmarks: 8,
            username: 'testuser',
            position: 2,
          }
        ]
      }
    });
  });

  // Mock my series API
  await page.route('**/api/v1/my/series*', async route => {
    await route.fulfill({
      json: {
        series: [
          {
            id: 'my-series-1',
            title: 'My Test Series',
            summary: 'My series for testing',
            user_id: 'current-user',
            username: 'currentuser',
            is_complete: true,
            work_count: 1,
            word_count: 5000,
            created_at: '2024-01-01T00:00:00Z',
            updated_at: '2024-01-01T00:00:00Z',
          }
        ],
        pagination: {
          page: 1,
          limit: 20,
          total: 1,
          total_pages: 1,
        }
      }
    });
  });

  // Mock my works API for series creation
  await page.route('**/api/v1/my/works*', async route => {
    await route.fulfill({
      json: {
        works: [
          {
            id: 'available-work-1',
            title: 'Available Work 1',
            summary: 'Available for series',
            status: 'published',
            series_id: null,
          },
          {
            id: 'available-work-2',
            title: 'Available Work 2',
            summary: 'Also available for series',
            status: 'published',
            series_id: null,
          }
        ]
      }
    });
  });

  // Mock series creation API
  await page.route('**/api/v1/series', async route => {
    if (route.request().method() === 'POST') {
      await route.fulfill({
        json: {
          series: {
            id: 'new-series-id',
            title: 'New Test Series',
            summary: 'Newly created series',
            notes: '',
            user_id: 'current-user',
            username: 'currentuser',
            is_complete: false,
            work_count: 0,
            word_count: 0,
            created_at: '2024-01-03T00:00:00Z',
            updated_at: '2024-01-03T00:00:00Z',
          }
        }
      });
    }
  });
}

test.describe('Series Flow', () => {
  test.beforeEach(async ({ page }) => {
    await mockSeriesAPI(page);
  });

  test('should display series browse page', async ({ page }) => {
    await page.goto('/series');
    
    // Check page title and navigation
    await expect(page.locator('h1')).toContainText('Series');
    await expect(page.locator('text=Discover collections of related works')).toBeVisible();
    
    // Check tabs are present
    await expect(page.locator('text=Browse All Series')).toBeVisible();
    await expect(page.locator('text=My Series')).toBeVisible();
    
    // Check search functionality exists
    await expect(page.locator('input[placeholder*="Search series"]')).toBeVisible();
    await expect(page.locator('button:has-text("Search")')).toBeVisible();
  });

  test('should search for series', async ({ page }) => {
    await page.goto('/series');
    
    // Perform search
    await page.fill('input[placeholder*="Search series"]', 'test');
    await page.click('button:has-text("Search")');
    
    // Verify search results
    await expect(page.locator('text=Test Series')).toBeVisible();
    await expect(page.locator('text=A test series for E2E testing')).toBeVisible();
    await expect(page.locator('text=2 works')).toBeVisible();
    await expect(page.locator('text=15,000 words')).toBeVisible();
  });

  test('should handle empty search results', async ({ page }) => {
    await page.goto('/series');
    
    // Search for non-existent series
    await page.fill('input[placeholder*="Search series"]', 'nonexistent');
    await page.click('button:has-text("Search")');
    
    // Verify empty state
    await expect(page.locator('text=No series found')).toBeVisible();
  });

  test('should display my series when authenticated', async ({ page }) => {
    await setupAuth(page);
    await page.goto('/series');
    
    // Switch to My Series tab
    await page.click('text=My Series');
    
    // Verify my series are displayed
    await expect(page.locator('text=My Test Series')).toBeVisible();
    await expect(page.locator('text=My series for testing')).toBeVisible();
    await expect(page.locator('text=Complete')).toBeVisible();
  });

  test('should show create series button when authenticated', async ({ page }) => {
    await setupAuth(page);
    await page.goto('/series');
    
    // Check create button is visible
    await expect(page.locator('text=Create Series')).toBeVisible();
  });

  test('should navigate to series detail page', async ({ page }) => {
    await page.goto('/series');
    
    // Search for and click on a series
    await page.fill('input[placeholder*="Search series"]', 'test');
    await page.click('button:has-text("Search")');
    await page.click('text=Test Series');
    
    // Verify navigation to detail page
    await expect(page).toHaveURL(/\/series\/series-1/);
  });

  test('should display series detail page', async ({ page }) => {
    await page.goto('/series/series-1');
    
    // Check series information is displayed
    await expect(page.locator('h1')).toContainText('Test Series');
    await expect(page.locator('text=A test series for E2E testing')).toBeVisible();
    await expect(page.locator('text=These are test notes')).toBeVisible();
    await expect(page.locator('text=by testuser')).toBeVisible();
    
    // Check statistics
    await expect(page.locator('text=2').first()).toBeVisible(); // work count
    await expect(page.locator('text=15,000')).toBeVisible(); // word count
    await expect(page.locator('text=In Progress')).toBeVisible(); // status
    
    // Check works are listed
    await expect(page.locator('h2')).toContainText('Works in This Series');
    await expect(page.locator('text=First Work in Series')).toBeVisible();
    await expect(page.locator('text=Second Work in Series')).toBeVisible();
    
    // Check work positions
    await expect(page.locator('text=Part 1 of the series')).toBeVisible();
    await expect(page.locator('text=Part 2 of the series')).toBeVisible();
  });

  test('should display series creation page when authenticated', async ({ page }) => {
    await setupAuth(page);
    await page.goto('/series/new');
    
    // Check form elements
    await expect(page.locator('h1')).toContainText('Create New Series');
    await expect(page.locator('input[id="title"]')).toBeVisible();
    await expect(page.locator('textarea[id="summary"]')).toBeVisible();
    await expect(page.locator('textarea[id="notes"]')).toBeVisible();
    await expect(page.locator('input[id="is_complete"]')).toBeVisible();
    
    // Check available works section
    await expect(page.locator('text=Add Works to Series')).toBeVisible();
    await expect(page.locator('text=Available Work 1')).toBeVisible();
    await expect(page.locator('text=Available Work 2')).toBeVisible();
  });

  test('should create a new series', async ({ page }) => {
    await setupAuth(page);
    await page.goto('/series/new');
    
    // Fill out the form
    await page.fill('input[id="title"]', 'New Test Series');
    await page.fill('textarea[id="summary"]', 'This is a new test series');
    await page.fill('textarea[id="notes"]', 'Some notes about the series');
    
    // Select some works
    await page.check('input[id="work-available-work-1"]');
    await page.check('input[id="work-available-work-2"]');
    
    // Submit the form
    await page.click('button:has-text("Create Series")');
    
    // Should redirect to new series page (mocked)
    await expect(page).toHaveURL(/\/series\/new-series-id/);
  });

  test('should validate series creation form', async ({ page }) => {
    await setupAuth(page);
    await page.goto('/series/new');
    
    // Try to submit without title
    await page.click('button:has-text("Create Series")');
    
    // Check validation
    await expect(page.locator('input[id="title"]:invalid')).toBeVisible();
  });

  test('should require authentication for series creation', async ({ page }) => {
    await page.goto('/series/new');
    
    // Check authentication message
    await expect(page.locator('text=Please log in to create a series')).toBeVisible();
  });

  test('should show empty state for user with no series', async ({ page }) => {
    await setupAuth(page);
    
    // Mock empty my series response
    await page.route('**/api/v1/my/series*', async route => {
      await route.fulfill({
        json: {
          series: [],
          pagination: {
            page: 1,
            limit: 20,
            total: 0,
            total_pages: 0,
          }
        }
      });
    });
    
    await page.goto('/series');
    await page.click('text=My Series');
    
    // Check empty state
    await expect(page.locator('text=You haven\'t created any series yet')).toBeVisible();
    await expect(page.locator('text=Create Your First Series')).toBeVisible();
  });

  test('should handle pagination', async ({ page }) => {
    // Mock paginated response
    await page.route('**/api/v1/series?*', async route => {
      const url = new URL(route.request().url());
      const page_num = url.searchParams.get('page') || '1';
      
      await route.fulfill({
        json: {
          series: [
            {
              id: `series-page-${page_num}`,
              title: `Series Page ${page_num}`,
              summary: `Series from page ${page_num}`,
              user_id: 'user-1',
              username: 'testuser',
              is_complete: false,
              work_count: 1,
              word_count: 5000,
              created_at: '2024-01-01T00:00:00Z',
              updated_at: '2024-01-01T00:00:00Z',
            }
          ],
          pagination: {
            page: parseInt(page_num),
            limit: 20,
            total: 50,
            total_pages: 3,
          }
        }
      });
    });
    
    await page.goto('/series');
    
    // Check pagination exists
    await expect(page.locator('button:has-text("Next")')).toBeVisible();
    await expect(page.locator('button:has-text("2")')).toBeVisible();
    
    // Navigate to next page
    await page.click('button:has-text("2")');
    
    // Verify content changed
    await expect(page.locator('text=Series Page 2')).toBeVisible();
  });

  test('should show navigation breadcrumbs', async ({ page }) => {
    await page.goto('/series/series-1');
    
    // Check back to browse link
    await expect(page.locator('text=Browse All Series')).toBeVisible();
  });

  test('should handle loading states', async ({ page }) => {
    // Delay API response to test loading state
    await page.route('**/api/v1/series/series-1', async route => {
      await new Promise(resolve => setTimeout(resolve, 1000));
      await route.fulfill({
        json: {
          series: {
            id: 'series-1',
            title: 'Test Series',
            summary: 'A test series',
            user_id: 'user-1',
            username: 'testuser',
            is_complete: false,
            work_count: 2,
            word_count: 15000,
            created_at: '2024-01-01T00:00:00Z',
            updated_at: '2024-01-02T00:00:00Z',
          }
        }
      });
    });
    
    await page.goto('/series/series-1');
    
    // Check loading state is shown
    await expect(page.locator('.animate-pulse')).toBeVisible();
    
    // Wait for content to load
    await expect(page.locator('text=Test Series')).toBeVisible();
    await expect(page.locator('.animate-pulse')).not.toBeVisible();
  });

  test('should handle error states', async ({ page }) => {
    // Mock API error
    await page.route('**/api/v1/series/invalid-id', async route => {
      await route.fulfill({
        status: 404,
        json: { error: 'Series not found' }
      });
    });
    
    await page.goto('/series/invalid-id');
    
    // Check error message is displayed
    await expect(page.locator('text=Series not found')).toBeVisible();
  });
});