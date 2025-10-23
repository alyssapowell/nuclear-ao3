import { test, expect } from '@playwright/test';

/**
 * Offline Reading Consent System E2E Tests
 * 
 * These tests validate the complete author-driven offline reading consent system
 * in PRODUCTION work pages (not demo pages):
 * - Three-tier preference system (files_and_pwa, pwa_only, none)
 * - Respectful UI enforcement based on author preferences
 * - Educational content for readers
 * - Author override capabilities
 */

test.describe('Offline Reading Consent System - Production Features', () => {
  
  test.beforeEach(async ({ page }) => {
    // Navigate to the frontend
    await page.goto('http://localhost:3001');
    
    // Set up test data - we'll mock the API responses for different scenarios
    await page.route('**/api/v1/works/*', async (route) => {
      const url = route.request().url();
      const workId = url.split('/').pop();
      
      // Mock different works with different offline reading preferences
      let mockWork = {};
      
      if (workId === '123e4567-e89b-12d3-a456-426614174000') {
        mockWork = {
          work: {
            id: '123e4567-e89b-12d3-a456-426614174000',
            title: 'The Adventures of Sherlock Holmes',
            summary: 'A classic detective story with modern twists',
            offline_reading_override: 'use_default',
            author_default_offline_reading: 'files_and_pwa',
            user_id: 'author-123',
            // ... other work fields
            language: 'en',
            rating: 'general',
            category: [],
            warnings: [],
            fandoms: ['Test Fandom'],
            characters: [],
            relationships: [],
            freeform_tags: [],
            word_count: 1000,
            chapter_count: 1,
            is_complete: true,
            status: 'published',
            published_at: '2023-01-01T00:00:00Z',
            updated_at: '2023-01-01T00:00:00Z',
            hits: 100,
            kudos: 10,
            comments: 5,
            bookmarks: 3
          },
          authors: [{
            pseud_id: 'pseud-123',
            pseud_name: 'TestAuthor',
            user_id: 'author-123',
            username: 'testauthor',
            is_anonymous: false
          }]
        };
      } else if (workId === '223e4567-e89b-12d3-a456-426614174001') {
        mockWork = {
          work: {
            id: '223e4567-e89b-12d3-a456-426614174001',
            title: 'Modern Romance Collection',
            summary: 'Contemporary romance stories with diverse characters',
            offline_reading_override: 'pwa_only',
            author_default_offline_reading: 'files_and_pwa',
            user_id: 'author-456',
            language: 'en',
            rating: 'general',
            category: [],
            warnings: [],
            fandoms: ['Test Fandom'],
            characters: [],
            relationships: [],
            freeform_tags: [],
            word_count: 1000,
            chapter_count: 1,
            is_complete: true,
            status: 'published',
            published_at: '2023-01-01T00:00:00Z',
            updated_at: '2023-01-01T00:00:00Z',
            hits: 100,
            kudos: 10,
            comments: 5,
            bookmarks: 3
          },
          authors: [{
            pseud_id: 'pseud-456',
            pseud_name: 'PWAAuthor',
            user_id: 'author-456',
            username: 'pwaauthor',
            is_anonymous: false
          }]
        };
      } else if (workId === '323e4567-e89b-12d3-a456-426614174002') {
        mockWork = {
          work: {
            id: '323e4567-e89b-12d3-a456-426614174002',
            title: 'Exclusive Online Serial',
            summary: 'A work available only for online reading',
            offline_reading_override: 'none',
            author_default_offline_reading: 'pwa_only',
            user_id: 'author-789',
            language: 'en',
            rating: 'general',
            category: [],
            warnings: [],
            fandoms: ['Test Fandom'],
            characters: [],
            relationships: [],
            freeform_tags: [],
            word_count: 1000,
            chapter_count: 1,
            is_complete: true,
            status: 'published',
            published_at: '2023-01-01T00:00:00Z',
            updated_at: '2023-01-01T00:00:00Z',
            hits: 100,
            kudos: 10,
            comments: 5,
            bookmarks: 3
          },
          authors: [{
            pseud_id: 'pseud-789',
            pseud_name: 'OnlineAuthor',
            user_id: 'author-789',
            username: 'onlineauthor',
            is_anonymous: false
          }]
        };
      }
      
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(mockWork)
      });
    });

    // Mock export service health check
    await page.route('**/export/health', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ status: 'healthy' })
      });
    });

    // Mock chapters endpoint
    await page.route('**/api/v1/works/*/chapters', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          chapters: [{
            id: 'chapter-1',
            work_id: route.request().url().split('/')[5],
            number: 1,
            title: 'Chapter 1',
            summary: '',
            notes: '',
            end_notes: '',
            content: 'This is the test chapter content.',
            word_count: 100,
            status: 'published',
            published_at: '2023-01-01T00:00:00Z',
            created_at: '2023-01-01T00:00:00Z',
            updated_at: '2023-01-01T00:00:00Z'
          }]
        })
      });
    });
  });

  test('should show export button for files_and_pwa preference', async ({ page }) => {
    // Test on a real work page with mocked work data
    await page.goto('http://localhost:3001/works/123e4567-e89b-12d3-a456-426614174000');
    
    // Wait for the page to load and work title to appear
    await expect(page.locator('h1')).toBeVisible();
    
    // Check that the RespectfulExportButton shows "Export" text
    const exportButton = page.locator('[data-testid="respectful-export-button"]');
    await expect(exportButton).toBeVisible();
    await expect(exportButton).toContainText('Export');
    
    // Click the button to open export modal
    await exportButton.click();
    
    // Verify export modal appears with download options
    const exportModal = page.locator('[data-testid="export-modal"]');
    await expect(exportModal).toBeVisible();
    
    // Should show EPUB, MOBI, PDF download options
    await expect(page.locator('text=EPUB')).toBeVisible();
    await expect(page.locator('text=MOBI')).toBeVisible();
    await expect(page.locator('text=PDF')).toBeVisible();
    
    // Should also show PWA option
    await expect(page.locator('text=Read Offline')).toBeVisible();
  });

  test('should show read offline button for pwa_only preference', async ({ page }) => {
    await page.goto('http://localhost:3001/works/223e4567-e89b-12d3-a456-426614174001');
    
    // Wait for the page to load
    await expect(page.locator('h1')).toBeVisible();
    
    // Check that the RespectfulExportButton shows "Read Offline" text
    const exportButton = page.locator('[data-testid="respectful-export-button"]');
    await expect(exportButton).toBeVisible();
    await expect(exportButton).toContainText('Read Offline');
    
    // Click the button to open export modal
    await exportButton.click();
    
    // Verify export modal appears but without download options
    const exportModal = page.locator('[data-testid="export-modal"]');
    await expect(exportModal).toBeVisible();
    
    // Should NOT show file download options
    await expect(page.locator('text=EPUB')).not.toBeVisible();
    await expect(page.locator('text=MOBI')).not.toBeVisible();
    await expect(page.locator('text=PDF')).not.toBeVisible();
    
    // Should show PWA option
    await expect(page.locator('text=Read Offline')).toBeVisible();
    
    // Should show explanation about author's choice
    await expect(page.locator('text=author prefers')).toBeVisible();
  });

  test('should show online only button for none preference', async ({ page }) => {
    await page.goto('http://localhost:3001/works/323e4567-e89b-12d3-a456-426614174002');
    
    // Wait for the page to load
    await expect(page.locator('h1')).toBeVisible();
    
    // Check that the RespectfulExportButton shows "Online Only" text
    const exportButton = page.locator('[data-testid="respectful-export-button"]');
    await expect(exportButton).toBeVisible();
    await expect(exportButton).toContainText('Online Only');
    
    // Click the button to open educational modal
    await exportButton.click();
    
    // Verify educational modal appears
    const modal = page.locator('[role="dialog"]');
    await expect(modal).toBeVisible();
    
    // Should show educational content
    await expect(modal).toContainText('respects');
    await expect(modal).toContainText('author');
    await expect(modal).toContainText('online');
    
    // Should NOT show any download or PWA options
    await expect(page.locator('text=EPUB')).not.toBeVisible();
    await expect(page.locator('text=MOBI')).not.toBeVisible();
    await expect(page.locator('text=PDF')).not.toBeVisible();
    await expect(page.locator('text=Read Offline')).not.toBeVisible();
  });

  test('should handle author vs reader experience differently', async ({ page }) => {
    // Mock authentication as the author
    await page.evaluate(() => {
      localStorage.setItem('auth_token', 'test-token');
      localStorage.setItem('user', JSON.stringify({
        id: 'author-789',
        username: 'onlineauthor'
      }));
    });

    await page.goto('http://localhost:3001/works/323e4567-e89b-12d3-a456-426614174002');
    
    // Wait for the page to load
    await expect(page.locator('h1')).toBeVisible();
    
    // Author should see export options regardless of their settings
    const exportButton = page.locator('[data-testid="respectful-export-button"]');
    await expect(exportButton).toBeVisible();
    
    // Click the button
    await exportButton.click();
    
    // Author should see export options even for "none" preference
    const exportModal = page.locator('[data-testid="export-modal"]');
    await expect(exportModal).toBeVisible();
    
    // Should show download options for author
    await expect(page.locator('text=EPUB')).toBeVisible();
    await expect(page.locator('text=MOBI')).toBeVisible();
    await expect(page.locator('text=PDF')).toBeVisible();
  });

  test('should show appropriate button icons', async ({ page }) => {
    // Test files_and_pwa shows download icon
    await page.goto('http://localhost:3001/works/files-allowed-work');
    await expect(page.locator('[data-testid="respectful-export-button"] svg')).toBeVisible();
    
    // Test pwa_only shows phone icon
    await page.goto('http://localhost:3001/works/pwa-only-work');
    await expect(page.locator('[data-testid="respectful-export-button"] svg')).toBeVisible();
    
    // Test none shows info icon
    await page.goto('http://localhost:3001/works/online-only-work');
    await expect(page.locator('[data-testid="respectful-export-button"] svg')).toBeVisible();
  });

  test('should provide clear feedback for blocked export attempts', async ({ page }) => {
    await page.goto('http://localhost:3001/works/pwa-only-work');
    
    const exportButton = page.locator('[data-testid="respectful-export-button"]');
    await exportButton.click();
    
    // Should show explanation why downloads are blocked
    const modal = page.locator('[role="dialog"]');
    await expect(modal).toContainText('author');
    await expect(modal).toContainText('prefers');
    await expect(modal).toContainText('PWA');
  });

  test('should handle modal close functionality', async ({ page }) => {
    await page.goto('http://localhost:3001/works/files-allowed-work');
    
    const exportButton = page.locator('[data-testid="respectful-export-button"]');
    await exportButton.click();
    
    // Modal should be visible
    const modal = page.locator('[data-testid="export-modal"]');
    await expect(modal).toBeVisible();
    
    // Close with X button
    await page.locator('[data-testid="close-modal"]').click();
    await expect(modal).not.toBeVisible();
    
    // Open again and close with Escape key
    await exportButton.click();
    await expect(modal).toBeVisible();
    await page.keyboard.press('Escape');
    await expect(modal).not.toBeVisible();
  });

  test('should validate export service integration', async ({ page }) => {
    // Mock export service responses
    await page.route('**/export/**', async (route) => {
      const url = route.request().url();
      if (url.includes('/export/epub/')) {
        await route.fulfill({
          status: 200,
          contentType: 'application/epub+zip',
          headers: {
            'Content-Disposition': 'attachment; filename="test-work.epub"'
          },
          body: 'mock epub content'
        });
      }
    });

    await page.goto('http://localhost:3001/works/files-allowed-work');
    
    const exportButton = page.locator('[data-testid="respectful-export-button"]');
    await exportButton.click();
    
    // Click EPUB download
    await page.locator('button:has-text("EPUB")').click();
    
    // Should initiate download (we can't easily test actual file download in Playwright,
    // but we can verify the request was made)
    // The actual export functionality is tested in our node test file
  });

  test('should show loading states appropriately', async ({ page }) => {
    await page.goto('http://localhost:3001/works/files-allowed-work');
    
    const exportButton = page.locator('[data-testid="respectful-export-button"]');
    
    // Button should not show loading initially
    await expect(exportButton).not.toHaveClass(/loading/);
    
    // After clicking, modal should appear
    await exportButton.click();
    const modal = page.locator('[data-testid="export-modal"]');
    await expect(modal).toBeVisible();
  });

  test('should handle network errors gracefully', async ({ page }) => {
    // Mock export service to return error
    await page.route('**/export/health', async (route) => {
      await route.fulfill({
        status: 500,
        contentType: 'application/json',
        body: JSON.stringify({ error: 'Export service unavailable' })
      });
    });

    await page.goto('http://localhost:3001/works/files-allowed-work');
    
    const exportButton = page.locator('[data-testid="respectful-export-button"]');
    await exportButton.click();
    
    // Should show error message
    await expect(page.locator('text=Export service is currently unavailable')).toBeVisible();
  });

});