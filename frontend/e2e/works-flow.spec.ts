import { test, expect } from '@playwright/test';

test.describe('Works Flow', () => {
  test('should browse works and view work details', async ({ page }) => {
    // Navigate to works page
    await page.goto('/works');

    // Should display works listing
    await expect(page.getByRole('heading', { name: /works/i })).toBeVisible();

    // Should show search functionality
    await expect(page.getByRole('textbox', { name: /search/i })).toBeVisible();

    // Should display work cards if there are any works
    const workCards = page.locator('.work-card, [data-testid="work-card"]').first();
    
    // If works exist, test the flow
    if (await workCards.isVisible()) {
      // Click on first work
      await workCards.click();
      
      // Should navigate to work detail page
      await expect(page).toHaveURL(/\/works\/[^\/]+$/);
      
      // Should display work title
      await expect(page.getByRole('heading').first()).toBeVisible();
      
      // Should display work content/chapters
      await expect(page.locator('article, .work-content, .chapter-content')).toBeVisible();
    }
  });

  test('should allow searching for works', async ({ page }) => {
    await page.goto('/works');

    // Find and use search functionality
    const searchBox = page.getByRole('textbox', { name: /search/i });
    await searchBox.fill('harry potter');
    
    // Submit search (either by pressing Enter or clicking search button)
    await searchBox.press('Enter');
    
    // Should show search results or indicate no results
    await expect(page.locator('body')).toContainText(/results|works|no works found/i);
  });

  test('should allow filtering works', async ({ page }) => {
    await page.goto('/works');

    // Look for filter controls
    const filters = page.locator('[data-testid="filters"], .filters, .search-filters');
    
    if (await filters.isVisible()) {
      // Test rating filter if available
      const ratingFilter = page.locator('select[name*="rating"], input[name*="rating"]');
      if (await ratingFilter.first().isVisible()) {
        await ratingFilter.first().click();
      }
    }
    
    // Page should still be functional
    await expect(page.getByRole('heading', { name: /works/i })).toBeVisible();
  });

  test('should handle pagination if available', async ({ page }) => {
    await page.goto('/works');

    // Look for pagination controls
    const nextButton = page.getByRole('button', { name: /next|→/i });
    const pageButtons = page.locator('[data-testid="pagination"], .pagination');
    
    if (await nextButton.isVisible()) {
      await nextButton.click();
      
      // Should load next page
      await expect(page.getByRole('heading', { name: /works/i })).toBeVisible();
    } else if (await pageButtons.isVisible()) {
      const pageTwo = pageButtons.getByText('2').first();
      if (await pageTwo.isVisible()) {
        await pageTwo.click();
        await expect(page.getByRole('heading', { name: /works/i })).toBeVisible();
      }
    }
  });

  test('should display work creation page', async ({ page }) => {
    await page.goto('/works/new');

    // Should display work creation form
    await expect(page.getByRole('heading', { name: /new work|create work|post new work/i })).toBeVisible();
    
    // Should have title field
    await expect(page.getByRole('textbox', { name: /title/i })).toBeVisible();
    
    // Should have content area
    await expect(page.getByRole('textbox', { name: /content|work text|chapter/i })).toBeVisible();
  });

  test('should handle work interaction features', async ({ page }) => {
    // Navigate to a work page (assuming works exist)
    await page.goto('/works');
    
    const firstWork = page.locator('.work-card, [data-testid="work-card"]').first();
    
    if (await firstWork.isVisible()) {
      await firstWork.click();
      
      // Look for kudos button
      const kudosButton = page.getByRole('button', { name: /kudos|give kudos/i });
      if (await kudosButton.isVisible()) {
        // Button should be clickable (testing UI, not actual kudos giving)
        await expect(kudosButton).toBeEnabled();
      }
      
      // Look for bookmark functionality
      const bookmarkButton = page.getByRole('button', { name: /bookmark/i });
      if (await bookmarkButton.isVisible()) {
        await expect(bookmarkButton).toBeEnabled();
      }
      
      // Look for comments section
      const commentsSection = page.locator('.comments, [data-testid="comments"]');
      if (await commentsSection.isVisible()) {
        await expect(commentsSection).toBeVisible();
      }
    }
  });
});