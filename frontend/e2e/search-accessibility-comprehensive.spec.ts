import { test, expect } from '@playwright/test';
import AxeBuilder from '@axe-core/playwright';

test.describe('Search Components - Comprehensive Accessibility', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/search');
  });

  test('SearchForm - Complete accessibility audit', async ({ page }) => {
    // Wait for the search form to load
    await page.waitForSelector('[role="search"]');

    // Run axe scan on the search form
    const accessibilityScanResults = await new AxeBuilder({ page })
      .include('[role="search"]')
      .analyze();

    expect(accessibilityScanResults.violations).toEqual([]);

    // Test semantic structure
    await expect(page.locator('[role="search"]')).toBeVisible();
    await expect(page.locator('h2:has-text("Enhanced Search")')).toBeVisible();

    // Test form labeling
    const titleInput = page.locator('#enhanced-search-title');
    await expect(titleInput).toBeVisible();
    await expect(page.locator('label[for="enhanced-search-title"]')).toBeVisible();

    const authorInput = page.locator('#enhanced-search-author');
    await expect(authorInput).toBeVisible();
    await expect(page.locator('label[for="enhanced-search-author"]')).toBeVisible();

    // Test ARIA attributes
    await expect(page.locator('[role="search"]')).toHaveAttribute('aria-labelledby');
    await expect(page.locator('[role="search"]')).toHaveAttribute('aria-describedby');

    // Test live regions
    await expect(page.locator('[aria-live="polite"]')).toBeVisible();

    // Test required field indicators
    const requiredFields = page.locator('[aria-label*="required"]');
    for (const field of await requiredFields.all()) {
      await expect(field).toBeVisible();
    }
  });

  test('SearchForm - Keyboard navigation', async ({ page }) => {
    // Start at the title field
    await page.focus('#enhanced-search-title');
    
    // Tab through all interactive elements
    const interactiveElements = [
      '#enhanced-search-title',
      '#enhanced-search-author',
      '#enhanced-search-relationships',
      '#enhanced-search-characters',
      '#enhanced-search-freeformTags', 
      '#enhanced-search-fandoms',
      '#enhanced-search-rating',
      '#enhanced-search-status',
      '#enhanced-search-exclude-poorly-tagged',
      '#enhanced-search-smart-suggestions',
      'button:has-text("Show Advanced Search")',
      'button:has-text("Clear All")',
      'button[type="submit"]'
    ];

    for (const selector of interactiveElements) {
      const element = page.locator(selector);
      if (await element.isVisible()) {
        await expect(element).toBeFocused({ timeout: 1000 });
        await page.keyboard.press('Tab');
      }
    }
  });

  test('SearchForm - Error handling accessibility', async ({ page }) => {
    // Submit form without any criteria to trigger validation
    await page.click('button[type="submit"]');

    // Check for error announcement
    const errorRegion = page.locator('[role="alert"]');
    await expect(errorRegion).toBeVisible();
    await expect(errorRegion).toContainText('Please enter at least one search criterion');

    // Verify error is announced to screen readers
    await expect(errorRegion).toHaveAttribute('role', 'alert');
    
    // Check focus management - should focus the title input
    await expect(page.locator('#enhanced-search-title')).toBeFocused();
  });

  test('SearchForm - Advanced search accessibility', async ({ page }) => {
    // Toggle advanced search
    const advancedToggle = page.locator('button:has-text("Show Advanced Search")');
    await expect(advancedToggle).toHaveAttribute('aria-expanded', 'false');
    
    await advancedToggle.click();
    
    // Check expanded state
    await expect(advancedToggle).toHaveAttribute('aria-expanded', 'true');
    await expect(advancedToggle).toContainText('Hide Advanced Search');
    
    // Check advanced section is visible and properly labeled
    const advancedSection = page.locator('[role="region"]:has-text("Advanced Search Options")');
    await expect(advancedSection).toBeVisible();
    await expect(advancedSection).toHaveAttribute('aria-labelledby');

    // Test focus management - should focus first advanced field
    await expect(page.locator('#enhanced-search-word-count-min')).toBeFocused();

    // Test all advanced fields have proper labels
    const advancedFields = [
      '#enhanced-search-word-count-min',
      '#enhanced-search-word-count-max',
      '#enhanced-search-language'
    ];

    for (const fieldId of advancedFields) {
      const field = page.locator(fieldId);
      const labelSelector = `label[for="${fieldId.slice(1)}"]`;
      await expect(page.locator(labelSelector)).toBeVisible();
    }
  });

  test('SearchForm - Tag management accessibility', async ({ page }) => {
    // Add a relationship tag
    const relationshipInput = page.locator('#enhanced-search-relationships');
    await relationshipInput.fill('Harry Potter/Draco Malfoy');
    await page.keyboard.press('Enter');

    // Check tag is added with proper accessibility
    const tagGroup = page.locator('[role="group"][aria-label*="Selected relationships"]');
    await expect(tagGroup).toBeVisible();

    const tagButton = page.locator('button[aria-label*="Remove Harry Potter/Draco Malfoy"]');
    await expect(tagButton).toBeVisible();
    await expect(tagButton).toHaveAttribute('title');

    // Test tag removal
    await tagButton.click();
    await expect(tagGroup).not.toBeVisible();

    // Check announcement for tag removal
    const liveRegion = page.locator('[aria-live="polite"]');
    await expect(liveRegion).toContainText('Removed');
  });

  test('SearchAutocomplete - Complete accessibility', async ({ page }) => {
    const autocompleteInput = page.locator('#enhanced-search-relationships');
    
    // Test autocomplete labeling
    await expect(autocompleteInput).toHaveAttribute('aria-describedby');
    
    // Type to trigger autocomplete
    await autocompleteInput.fill('Harry');
    
    // Wait for suggestions to appear
    await page.waitForSelector('[role="listbox"]', { timeout: 3000 });
    
    const listbox = page.locator('[role="listbox"]');
    await expect(listbox).toBeVisible();
    await expect(listbox).toHaveAttribute('aria-label');

    // Test option accessibility
    const options = page.locator('[role="option"]');
    const firstOption = options.first();
    await expect(firstOption).toHaveAttribute('aria-selected');
    
    // Test keyboard navigation in autocomplete
    await page.keyboard.press('ArrowDown');
    await page.keyboard.press('ArrowDown');
    await page.keyboard.press('Enter');
    
    // Verify selection closes autocomplete and announces
    await expect(listbox).not.toBeVisible();
  });

  test('SearchResults - Accessibility when results load', async ({ page }) => {
    // Fill in a search query
    await page.fill('#enhanced-search-title', 'Harry Potter');
    await page.click('button[type="submit"]');

    // Wait for results or loading state
    await page.waitForSelector('[role="status"], [role="main"], .search-results', { timeout: 10000 });

    // Check loading state accessibility
    const loadingStatus = page.locator('[role="status"]');
    if (await loadingStatus.isVisible()) {
      await expect(loadingStatus).toHaveAttribute('aria-live');
      await expect(loadingStatus).toContainText('Searching');
    }

    // If results load, check their accessibility
    const resultsContainer = page.locator('.search-results, [role="main"]');
    if (await resultsContainer.isVisible()) {
      // Run accessibility scan on results
      const resultsScan = await new AxeBuilder({ page })
        .include('.search-results, [role="main"]')
        .analyze();
      
      expect(resultsScan.violations).toEqual([]);

      // Check results announcement
      const liveRegion = page.locator('[aria-live="polite"]');
      await expect(liveRegion).toContainText(/Found \d+ results|Search completed/);
    }
  });

  test('Search page - Overall page structure', async ({ page }) => {
    // Test page landmarks
    await expect(page.locator('main, [role="main"]')).toBeVisible();
    await expect(page.locator('header, [role="banner"]')).toBeVisible();
    
    // Test heading hierarchy
    const headings = page.locator('h1, h2, h3, h4, h5, h6');
    const headingTexts = await headings.allTextContents();
    
    // Should have a logical heading structure
    expect(headingTexts.length).toBeGreaterThan(0);
    
    // Test skip links if present
    const skipLink = page.locator('a:has-text("Skip to main content"), .skip-link');
    if (await skipLink.isVisible()) {
      await expect(skipLink).toHaveAttribute('href');
    }
  });

  test('Search page - Color contrast and visual accessibility', async ({ page }) => {
    // This test focuses on programmatically checkable visual issues
    const accessibilityScan = await new AxeBuilder({ page })
      .withTags(['wcag2a', 'wcag2aa', 'wcag21aa'])
      .analyze();

    // Filter out only color contrast violations
    const colorContrastViolations = accessibilityScan.violations.filter(
      violation => violation.id === 'color-contrast'
    );

    expect(colorContrastViolations).toEqual([]);
  });

  test('Search page - Focus management and indicators', async ({ page }) => {
    // Test that focus is visible on all interactive elements
    const interactiveSelectors = [
      'button',
      'input',
      'select',
      'textarea',
      'a[href]',
      '[tabindex]'
    ];

    for (const selector of interactiveSelectors) {
      const elements = page.locator(selector);
      const count = await elements.count();
      
      for (let i = 0; i < Math.min(count, 5); i++) { // Test first 5 of each type
        const element = elements.nth(i);
        if (await element.isVisible() && await element.isEnabled()) {
          await element.focus();
          
          // Check that element is focused (basic check)
          await expect(element).toBeFocused();
          
          // Check for visible focus indicator (this is basic - real visual testing would be more complex)
          const styles = await element.evaluate((el) => {
            const computed = window.getComputedStyle(el);
            return {
              outline: computed.outline,
              outlineWidth: computed.outlineWidth,
              boxShadow: computed.boxShadow
            };
          });
          
          // Should have some kind of focus indicator
          const hasFocusIndicator = 
            styles.outline !== 'none' || 
            styles.outlineWidth !== '0px' || 
            styles.boxShadow !== 'none';
          
          expect(hasFocusIndicator).toBeTruthy();
        }
      }
    }
  });

  test('Search page - Screen reader announcements', async ({ page }) => {
    // Test that important actions trigger announcements
    
    // Clear all filters
    await page.click('button:has-text("Clear All")');
    const liveRegion = page.locator('[aria-live="polite"]');
    await expect(liveRegion).toContainText('cleared');

    // Toggle advanced search
    await page.click('button:has-text("Show Advanced Search")');
    await expect(liveRegion).toContainText('expanded');

    // Hide advanced search
    await page.click('button:has-text("Hide Advanced Search")');
    await expect(liveRegion).toContainText('collapsed');
  });

  test('Search form - Mobile accessibility', async ({ page, isMobile }) => {
    if (!isMobile) {
      // Set mobile viewport
      await page.setViewportSize({ width: 375, height: 667 });
    }

    // Test that form is usable on mobile
    const searchForm = page.locator('[role="search"]');
    await expect(searchForm).toBeVisible();

    // Test touch targets are appropriately sized
    const buttons = page.locator('button');
    for (const button of await buttons.all()) {
      if (await button.isVisible()) {
        const boundingBox = await button.boundingBox();
        if (boundingBox) {
          // WCAG recommends minimum 44x44px touch targets
          expect(boundingBox.width).toBeGreaterThanOrEqual(44);
          expect(boundingBox.height).toBeGreaterThanOrEqual(44);
        }
      }
    }

    // Test that form scrolls properly and doesn't cause horizontal scroll
    const pageWidth = await page.evaluate(() => document.documentElement.scrollWidth);
    const viewportWidth = await page.evaluate(() => window.innerWidth);
    expect(pageWidth).toBeLessThanOrEqual(viewportWidth + 10); // Allow small tolerance
  });
});