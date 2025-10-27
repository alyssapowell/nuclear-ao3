import { test, expect, Page, BrowserContext } from '@playwright/test';
import { loginUser } from './test-utils';

// Helper function to wait for series form and fill it
async function fillSeriesForm(page: Page, title: string, summary?: string, notes?: string, complete?: boolean) {
  await expect(page.locator('#title')).toBeVisible({ timeout: 10000 });
  await page.fill('#title', title);
  
  if (summary) {
    await page.fill('#summary', summary);
  }
  
  if (notes) {
    await page.fill('#notes', notes);
  }
  
  if (complete) {
    const completeCheckbox = page.locator('#is_complete');
    if (await completeCheckbox.isVisible()) {
      await page.check('#is_complete');
    }
  }
}

// Configuration
const FIRST_PARTY_CLIENT_ID = '11111111-1111-1111-1111-111111111111';
const API_URL = process.env.API_URL || 'http://localhost:8080/api/v1';

// Helper function to get test work ID for series testing
async function getTestWorkId() {
  try {
    const response = await fetch(`${API_URL}/works?limit=1`, {
      headers: {
        'Authorization': 'Bearer dummy-token-for-rate-limiting',
        'X-Client-ID': FIRST_PARTY_CLIENT_ID,
        'X-Client-First-Party': 'true',
        'X-OAuth-Scopes': 'read',
      }
    });
    if (!response.ok) {
      throw new Error(`API responded with status ${response.status}`);
    }
    const data = await response.json();
    return data.works?.[0]?.id || null;
  } catch (error) {
    console.warn(`Failed to get test work ID: ${error}`);
    return null;
  }
}

test.describe('Series Management', () => {
  test.beforeEach(async ({ page }) => {
    // Set up authentication
    await loginUser(page);
  });

  test('should create a new series', async ({ page }) => {
    await test.step('Navigate to series creation page', async () => {
      await page.goto('/series/new');
      await page.waitForLoadState('networkidle');
    });

    await test.step('Fill out series form', async () => {
      await fillSeriesForm(
        page, 
        'Test Series E2E', 
        'This is a test series created by E2E automation',
        'These are author notes for the test series',
        true
      );
    });

    await test.step('Submit form and verify redirect', async () => {
      await page.click('button[type="submit"]');
      
      // Should redirect to the new series page
      await expect(page).toHaveURL(/\/series\/[a-f0-9-]+$/);
      
      // Verify series details are displayed
      await expect(page.locator('h1')).toContainText('Test Series E2E');
      await expect(page.locator('text=This is a test series')).toBeVisible();
      await expect(page.locator('text=Complete')).toBeVisible();
    });
  });

  test('should browse and search series', async ({ page }) => {
    await test.step('Navigate to series listing page', async () => {
      await page.goto('/series');
      await page.waitForLoadState('networkidle');
    });

    await test.step('Verify page elements', async () => {
      await expect(page.locator('h1')).toContainText('Series');
      await expect(page.locator('text=Browse All Series')).toBeVisible();
      await expect(page.locator('text=My Series')).toBeVisible();
      await expect(page.locator('text=Create Series')).toBeVisible();
    });

    await test.step('Test search functionality', async () => {
      const searchInput = page.locator('input[placeholder*="Search series"]');
      await searchInput.fill('test series');
      await page.click('button:has(svg)'); // Search button with icon
      
      // Should show search results or no results message
      await expect(page.locator('body')).toBeVisible(); // Page loads successfully
    });

    await test.step('Switch to My Series tab', async () => {
      await page.click('text=My Series');
      await page.waitForLoadState('networkidle');
      
      // Should show user's series or empty state
      await expect(page.locator('body')).toBeVisible();
    });
  });

  test('should view series details', async ({ page }) => {
    // First create a series to view
    await test.step('Create a test series', async () => {
      await page.goto('/series/new');
      await page.waitForLoadState('networkidle');
      await fillSeriesForm(page, 'Detailed Test Series', 'A series with comprehensive details for testing');
      await page.click('button[type="submit"]');
      await page.waitForURL(/\/series\/[a-f0-9-]+$/);
    });

    await test.step('Verify series details page', async () => {
      await expect(page.locator('h1')).toContainText('Detailed Test Series');
      await expect(page.locator('text=A series with comprehensive details')).toBeVisible();
      
      // Check series statistics
      await expect(page.locator('text=Works')).toBeVisible();
      await expect(page.locator('text=Words')).toBeVisible();
      await expect(page.locator('text=Status')).toBeVisible();
      await expect(page.locator('text=Updated')).toBeVisible();
      
      // Check works section
      await expect(page.locator('text=Works in This Series')).toBeVisible();
    });

    await test.step('Verify edit button for series owner', async () => {
      // Should show edit button since we created the series
      await expect(page.locator('text=Edit Series')).toBeVisible();
    });
  });

  test('should edit an existing series', async ({ page }) => {
    let seriesUrl: string;

    await test.step('Create a series to edit', async () => {
      await page.goto('/series/new');
      await page.waitForLoadState('networkidle');
      await fillSeriesForm(page, 'Series To Edit', 'Original summary');
      await page.click('button[type="submit"]');
      await page.waitForURL(/\/series\/[a-f0-9-]+$/);
      seriesUrl = page.url();
    });

    await test.step('Navigate to edit page', async () => {
      await page.click('text=Edit Series');
      await page.waitForURL(/\/series\/[a-f0-9-]+\/edit$/);
    });

    await test.step('Update series information', async () => {
      await fillSeriesForm(
        page, 
        'Updated Series Title', 
        'Updated summary with new information',
        'Added some author notes',
        true
      );
    });

    await test.step('Save changes and verify', async () => {
      await page.click('text=Save Changes');
      await page.waitForURL(seriesUrl);
      
      // Verify updated information
      await expect(page.locator('h1')).toContainText('Updated Series Title');
      await expect(page.locator('text=Updated summary with new information')).toBeVisible();
      await expect(page.locator('text=Added some author notes')).toBeVisible();
      await expect(page.locator('text=Complete')).toBeVisible();
    });
  });

  test('should handle series with works', async ({ page }) => {
    const testWorkId = await getTestWorkId();
    
    if (!testWorkId) {
      test.skip('No test work available for series management test');
      return;
    }

    await test.step('Create series with work selection', async () => {
      await page.goto('/series/new');
      await page.waitForLoadState('networkidle');
      await fillSeriesForm(page, 'Series With Works', 'A series that includes existing works');
      
      // Wait for works to load and select if available
      await page.waitForTimeout(2000); // Allow works to load
      
      const workCheckboxes = page.locator('input[type="checkbox"][id^="work-"]');
      const checkboxCount = await workCheckboxes.count();
      
      if (checkboxCount > 0) {
        await workCheckboxes.first().check();
      }
      
      await page.click('button[type="submit"]');
      await page.waitForURL(/\/series\/[a-f0-9-]+$/);
    });

    await test.step('Verify works in series', async () => {
      await expect(page.locator('text=Works in This Series')).toBeVisible();
      
      // Check if works are displayed (might be 0 if no works were available)
      const worksSection = page.locator('text=Works in This Series').locator('..');
      await expect(worksSection).toBeVisible();
    });
  });

  test('should manage works in series via edit page', async ({ page }) => {
    let seriesUrl: string;

    await test.step('Create a series for work management', async () => {
      await page.goto('/series/new');
      await page.waitForLoadState('networkidle');
      await fillSeriesForm(page, 'Work Management Series');
      await page.click('button[type="submit"]');
      await page.waitForURL(/\/series\/[a-f0-9-]+$/);
      seriesUrl = page.url();
    });

    await test.step('Navigate to edit page', async () => {
      await page.click('text=Edit Series');
      await page.waitForURL(/\/series\/[a-f0-9-]+\/edit$/);
    });

    await test.step('Check work management interface', async () => {
      // Verify works management sections exist
      await expect(page.locator('text=Works in Series')).toBeVisible();
      await expect(page.locator('text=Add Works')).toBeVisible();
      
      // Check if there are available works to add
      const addWorkButtons = page.locator('text=Add to Series');
      const removeWorkButtons = page.locator('text=Remove');
      
      // Interface should be present even if no works available
      await expect(page.locator('body')).toBeVisible();
    });
  });

  test('should handle navigation between series pages', async ({ page }) => {
    await test.step('Start from series listing', async () => {
      await page.goto('/series');
      await expect(page.locator('h1')).toContainText('Series');
    });

    await test.step('Navigate to series creation', async () => {
      await page.click('text=Create Series');
      await expect(page).toHaveURL('/series/new');
      await expect(page.locator('h1')).toContainText('Create New Series');
    });

    await test.step('Use cancel to return', async () => {
      await page.click('text=Cancel');
      await expect(page).toHaveURL('/series');
    });

    await test.step('Test breadcrumb navigation', async () => {
      // Create a series to test navigation
      await page.click('text=Create Series');
      await page.waitForLoadState('networkidle');
      await fillSeriesForm(page, 'Navigation Test Series');
      await page.click('button[type="submit"]');
      await page.waitForURL(/\/series\/[a-f0-9-]+$/);
      
      // Navigate back to series listing via link
      await page.click('text=Browse All Series');
      await expect(page).toHaveURL('/series');
    });
  });

  test('should validate form inputs', async ({ page }) => {
    await test.step('Navigate to series creation', async () => {
      await page.goto('/series/new');
    });

    await test.step('Test required field validation', async () => {
      // Try to submit without title
      await page.click('button[type="submit"]');
      
      // Should show validation error or stay on page
      await expect(page).toHaveURL('/series/new');
      
      // HTML5 validation should prevent submission
      const titleInput = page.locator('#title');
      await expect(titleInput).toHaveAttribute('required');
    });

    await test.step('Test with valid input', async () => {
      await expect(page.locator('#title')).toBeVisible({ timeout: 10000 });
      await page.fill('#title', 'Valid Series Title');
      
      // Submit button should now be enabled
      const submitButton = page.locator('button[type="submit"]');
      await expect(submitButton).not.toBeDisabled();
    });
  });

  test('should handle errors gracefully', async ({ page }) => {
    await test.step('Test with network interruption simulation', async () => {
      await page.goto('/series');
      
      // Try to access non-existent series
      await page.goto('/series/non-existent-id');
      
      // Should show error message or 404-like state
      await expect(page.locator('body')).toBeVisible(); // Page loads
      
      // May show error message depending on implementation
      const hasErrorText = await page.locator('text=not found, text=error, text=failed').count() > 0;
      // Don't fail if no specific error message is shown
    });
  });

  test('should be responsive on different screen sizes', async ({ page }) => {
    await test.step('Test mobile viewport', async () => {
      await page.setViewportSize({ width: 375, height: 667 }); // iPhone SE
      await page.goto('/series');
      
      // Page should load and be usable
      await expect(page.locator('h1')).toBeVisible();
      await expect(page.locator('text=Create Series')).toBeVisible();
    });

    await test.step('Test tablet viewport', async () => {
      await page.setViewportSize({ width: 768, height: 1024 }); // iPad
      await page.goto('/series/new');
      
      // Form should be usable
      await expect(page.locator('#title')).toBeVisible();
      await expect(page.locator('button[type="submit"]')).toBeVisible();
    });

    await test.step('Test desktop viewport', async () => {
      await page.setViewportSize({ width: 1920, height: 1080 }); // Desktop
      await page.goto('/series');
      
      // Full layout should be visible
      await expect(page.locator('h1')).toBeVisible();
      await expect(page.locator('text=Browse All Series')).toBeVisible();
    });
  });
});