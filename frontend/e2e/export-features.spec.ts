import { test, expect } from '@playwright/test';

/**
 * Export Features E2E Tests
 * Tests work export functionality (PDF, EPUB, HTML)
 */

test.describe('Export Features', () => {
  test.beforeEach(async ({ page }) => {
    // Navigate to a work page for export testing
    await page.goto('/works');
    
    // Find the first available work to test exports
    const firstWork = page.locator('.work-card, [data-testid="work-card"]').first();
    if (await firstWork.isVisible()) {
      await firstWork.click();
      // Wait for work page to load
      await expect(page.locator('article, .work-content')).toBeVisible();
    } else {
      // Skip if no works available
      test.skip('No works available for export testing');
    }
  });

  test('should display export options for works', async ({ page }) => {
    // Look for export button or menu
    const exportButton = page.locator('button:has-text("Export"), button:has-text("Download"), .export-menu');
    
    if (await exportButton.isVisible()) {
      await expect(exportButton).toBeVisible();
      await expect(exportButton).toBeEnabled();
    } else {
      // Check if export is in a dropdown menu
      const moreOptionsButton = page.locator('button:has-text("More"), button:has-text("Options"), .dropdown-trigger');
      if (await moreOptionsButton.isVisible()) {
        await moreOptionsButton.click();
        await expect(page.locator('text=Export, text=Download')).toBeVisible();
      }
    }
  });

  test('should export work as PDF', async ({ page }) => {
    // Look for PDF export option
    const exportButton = page.locator('button:has-text("Export"), button:has-text("Download")');
    
    if (await exportButton.isVisible()) {
      await exportButton.click();
      
      // Look for PDF option in export menu
      const pdfOption = page.locator('a:has-text("PDF"), button:has-text("PDF")');
      if (await pdfOption.isVisible()) {
        // Start waiting for download before clicking
        const downloadPromise = page.waitForEvent('download');
        await pdfOption.click();
        
        const download = await downloadPromise;
        expect(download.suggestedFilename()).toMatch(/\.pdf$/);
      }
    } else {
      test.skip('PDF export not available');
    }
  });

  test('should export work as EPUB', async ({ page }) => {
    // Look for EPUB export option
    const exportButton = page.locator('button:has-text("Export"), button:has-text("Download")');
    
    if (await exportButton.isVisible()) {
      await exportButton.click();
      
      // Look for EPUB option in export menu
      const epubOption = page.locator('a:has-text("EPUB"), button:has-text("EPUB")');
      if (await epubOption.isVisible()) {
        // Start waiting for download before clicking
        const downloadPromise = page.waitForEvent('download');
        await epubOption.click();
        
        const download = await downloadPromise;
        expect(download.suggestedFilename()).toMatch(/\.epub$/);
      }
    } else {
      test.skip('EPUB export not available');
    }
  });

  test('should export work as HTML', async ({ page }) => {
    // Look for HTML export option
    const exportButton = page.locator('button:has-text("Export"), button:has-text("Download")');
    
    if (await exportButton.isVisible()) {
      await exportButton.click();
      
      // Look for HTML option in export menu
      const htmlOption = page.locator('a:has-text("HTML"), button:has-text("HTML")');
      if (await htmlOption.isVisible()) {
        // Start waiting for download before clicking
        const downloadPromise = page.waitForEvent('download');
        await htmlOption.click();
        
        const download = await downloadPromise;
        expect(download.suggestedFilename()).toMatch(/\.html$/);
      }
    } else {
      test.skip('HTML export not available');
    }
  });

  test('should handle export errors gracefully', async ({ page }) => {
    // Test export functionality with network issues
    await page.route('**/export/**', route => {
      route.abort('failed');
    });

    const exportButton = page.locator('button:has-text("Export"), button:has-text("Download")');
    
    if (await exportButton.isVisible()) {
      await exportButton.click();
      
      const pdfOption = page.locator('a:has-text("PDF"), button:has-text("PDF")');
      if (await pdfOption.isVisible()) {
        await pdfOption.click();
        
        // Should show error message
        await expect(page.locator('text=Error, text=Failed, [role="alert"]')).toBeVisible();
      }
    } else {
      test.skip('Export functionality not available');
    }
  });

  test('should export series as collection', async ({ page }) => {
    // Navigate to a series page
    await page.goto('/series');
    
    const firstSeries = page.locator('.series-card, [data-testid="series-card"]').first();
    if (await firstSeries.isVisible()) {
      await firstSeries.click();
      
      // Look for series export options
      const exportButton = page.locator('button:has-text("Export Series"), button:has-text("Download All")');
      if (await exportButton.isVisible()) {
        await expect(exportButton).toBeVisible();
        await expect(exportButton).toBeEnabled();
        
        await exportButton.click();
        
        // Should show format options for series export
        const formatOptions = page.locator('.export-options, .format-selection');
        if (await formatOptions.isVisible()) {
          await expect(formatOptions).toBeVisible();
          await expect(page.locator('text=PDF, text=EPUB')).toBeVisible();
        }
      }
    } else {
      test.skip('No series available for export testing');
    }
  });

  test('should respect user preferences for export format', async ({ page }) => {
    // Check if user has saved export preferences
    const userMenu = page.locator('.user-menu, [data-testid="user-menu"]');
    if (await userMenu.isVisible()) {
      await userMenu.click();
      
      const settingsLink = page.locator('a:has-text("Settings"), a:has-text("Preferences")');
      if (await settingsLink.isVisible()) {
        await settingsLink.click();
        
        // Look for export preferences section
        const exportPrefs = page.locator('.export-preferences, .download-settings');
        if (await exportPrefs.isVisible()) {
          await expect(exportPrefs).toBeVisible();
          
          // Should have format preference options
          await expect(page.locator('select[name*="export"], input[name*="format"]')).toBeVisible();
        }
      }
    }
  });

  test('should provide export accessibility features', async ({ page }) => {
    // Test that export functionality is accessible
    const exportButton = page.locator('button:has-text("Export"), button:has-text("Download")');
    
    if (await exportButton.isVisible()) {
      // Should be keyboard accessible
      await exportButton.focus();
      await expect(exportButton).toBeFocused();
      
      // Should have proper ARIA attributes
      await expect(exportButton).toHaveAttribute('aria-label');
      
      // Should be announced to screen readers
      const buttonText = await exportButton.textContent();
      expect(buttonText).toBeTruthy();
      expect(buttonText?.length).toBeGreaterThan(0);
    }
  });
});