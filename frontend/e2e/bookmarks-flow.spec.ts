import { test, expect } from '@playwright/test';

test.describe('Bookmarks Feature', () => {
  test.beforeEach(async ({ page }) => {
    // Mock API responses for testing
    await page.route('http://localhost:8082/api/v1/works/*/bookmark-status', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ is_bookmarked: false }),
      });
    });

    await page.route('http://localhost:8082/api/v1/works/*/bookmark', async (route) => {
      if (route.request().method() === 'POST') {
        await route.fulfill({
          status: 201,
          contentType: 'application/json',
          body: JSON.stringify({ 
            bookmark: { 
              id: 'test-bookmark-id',
              work_id: 'test-work-id',
              notes: 'Test bookmark',
              tags: ['favorite'],
              is_private: false,
              created_at: new Date().toISOString(),
              updated_at: new Date().toISOString()
            }
          }),
        });
      }
    });

    await page.route('http://localhost:8082/api/v1/bookmarks', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          bookmarks: [
            {
              id: 'bookmark-1',
              work_id: 'work-1',
              notes: 'Amazing story!',
              tags: ['favorite', 'reread'],
              is_private: false,
              created_at: '2024-01-01T00:00:00Z',
              updated_at: '2024-01-01T00:00:00Z',
              work: {
                id: 'work-1',
                title: 'The Great Adventure',
                summary: 'A fantastic journey...',
                rating: 'general',
                fandoms: ['Test Fandom'],
                characters: ['Hero', 'Sidekick'],
                relationships: [],
                freeform_tags: ['adventure', 'friendship'],
                word_count: 5000,
                chapter_count: 3,
                is_complete: true,
                status: 'posted',
                published_at: '2024-01-01T00:00:00Z',
                updated_at: '2024-01-01T00:00:00Z',
                hits: 1200,
                kudos: 89,
                comments: 12,
                bookmarks: 45,
                authors: [
                  { pseud_name: 'TestAuthor', username: 'testauthor', is_anonymous: false }
                ]
              }
            }
          ],
          pagination: {
            page: 1,
            limit: 20,
            total: 1,
            total_pages: 1
          }
        }),
      });
    });

    // Mock localStorage for auth token
    await page.addInitScript(() => {
      localStorage.setItem('auth_token', 'test-auth-token');
    });
  });

  test('bookmark button displays correctly on work page', async ({ page }) => {
    // Mock work API response
    await page.route('http://localhost:8082/api/v1/works/test-work-id', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          work: {
            id: 'test-work-id',
            title: 'Test Work',
            summary: 'A test work for bookmarking',
            rating: 'general',
            fandoms: ['Test Fandom'],
            word_count: 1000,
            chapter_count: 1,
            is_complete: true,
            status: 'posted',
            hits: 100,
            kudos: 10,
            comments: 5,
            bookmarks: 3
          },
          authors: [
            { pseud_name: 'TestAuthor', username: 'testauthor', is_anonymous: false }
          ]
        }),
      });
    });

    await page.route('http://localhost:8082/api/v1/works/test-work-id/chapters', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          chapters: [
            {
              id: 'chapter-1',
              work_id: 'test-work-id',
              number: 1,
              title: 'Chapter 1',
              content: 'Test chapter content...',
              word_count: 1000,
              status: 'posted'
            }
          ]
        }),
      });
    });

    await page.goto('/works/test-work-id');
    
    // Wait for page to load
    await page.waitForSelector('h1');
    
    // Check bookmark button is present
    const bookmarkButton = page.locator('button:has-text("🔖 Bookmark")');
    await expect(bookmarkButton).toBeVisible();
    await expect(bookmarkButton).not.toBeDisabled();
  });

  test('can create a bookmark with notes and tags', async ({ page }) => {
    // Navigate to a work page
    await page.route('http://localhost:8082/api/v1/works/test-work-id', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          work: {
            id: 'test-work-id',
            title: 'Test Work',
            summary: 'A test work for bookmarking',
            rating: 'general',
            fandoms: ['Test Fandom'],
            word_count: 1000,
            chapter_count: 1,
            is_complete: true,
            status: 'posted',
            hits: 100,
            kudos: 10,
            comments: 5,
            bookmarks: 3
          },
          authors: [
            { pseud_name: 'TestAuthor', username: 'testauthor', is_anonymous: false }
          ]
        }),
      });
    });

    await page.route('http://localhost:8082/api/v1/works/test-work-id/chapters', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ chapters: [] }),
      });
    });

    await page.goto('/works/test-work-id');
    
    // Click bookmark button
    await page.click('button:has-text("🔖 Bookmark")');
    
    // Bookmark dialog should appear
    await expect(page.locator('text=Add Bookmark')).toBeVisible();
    
    // Fill in bookmark details
    await page.fill('textarea[placeholder*="Add personal notes"]', 'This is an amazing story!');
    await page.fill('input[placeholder*="tag1, tag2, tag3"]', 'favorite, must-read, excellent');
    await page.check('input[type="checkbox"]:near(:text("Make bookmark private"))');
    
    // Submit bookmark
    await page.click('button:has-text("Add Bookmark")');
    
    // Wait for success - button should change to "Bookmarked"
    await expect(page.locator('button:has-text("🔖 Bookmarked")')).toBeVisible();
    
    // Dialog should close
    await expect(page.locator('text=Add Bookmark')).not.toBeVisible();
  });

  test('can view bookmarks page', async ({ page }) => {
    await page.goto('/bookmarks');
    
    // Page should load with bookmarks
    await expect(page.locator('h1:has-text("My Bookmarks")')).toBeVisible();
    
    // Should show bookmark count
    await expect(page.locator('text=1 bookmark')).toBeVisible();
    
    // Should display the bookmarked work
    await expect(page.locator('text=The Great Adventure')).toBeVisible();
    await expect(page.locator('text=Amazing story!')).toBeVisible();
    
    // Should show bookmark tags
    await expect(page.locator('text=favorite')).toBeVisible();
    await expect(page.locator('text=reread')).toBeVisible();
  });

  test('can search and filter bookmarks', async ({ page }) => {
    await page.goto('/bookmarks');
    
    // Wait for bookmarks to load
    await expect(page.locator('h1:has-text("My Bookmarks")')).toBeVisible();
    
    // Test search functionality
    await page.fill('input[placeholder*="Search in titles"]', 'Adventure');
    await page.click('button:has-text("Search")');
    
    // Should still show the matching bookmark
    await expect(page.locator('text=The Great Adventure')).toBeVisible();
    
    // Test tag filtering
    await page.selectOption('select', 'favorite');
    await page.click('button:has-text("Search")');
    
    // Should show bookmark with the selected tag
    await expect(page.locator('text=The Great Adventure')).toBeVisible();
    
    // Test clear filters
    await page.click('button:has-text("Clear")');
    
    // Should reset to show all bookmarks
    await expect(page.locator('text=The Great Adventure')).toBeVisible();
  });

  test('bookmark button shows disabled state when not authenticated', async ({ page }) => {
    // Clear auth token
    await page.addInitScript(() => {
      localStorage.removeItem('auth_token');
    });

    await page.route('http://localhost:8082/api/v1/works/test-work-id', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          work: {
            id: 'test-work-id',
            title: 'Test Work',
            summary: 'A test work',
            rating: 'general',
            fandoms: ['Test Fandom'],
            word_count: 1000,
            chapter_count: 1,
            is_complete: true,
            status: 'posted',
            hits: 100,
            kudos: 10,
            comments: 5,
            bookmarks: 3
          },
          authors: [
            { pseud_name: 'TestAuthor', username: 'testauthor', is_anonymous: false }
          ]
        }),
      });
    });

    await page.route('http://localhost:8082/api/v1/works/test-work-id/chapters', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ chapters: [] }),
      });
    });

    await page.goto('/works/test-work-id');
    
    // Bookmark button should be disabled
    const bookmarkButton = page.locator('button:has-text("🔖 Bookmark")');
    await expect(bookmarkButton).toBeVisible();
    await expect(bookmarkButton).toBeDisabled();
    await expect(bookmarkButton).toHaveAttribute('title', 'Log in to bookmark works');
  });

  test('bookmarks page shows login prompt when not authenticated', async ({ page }) => {
    // Clear auth token
    await page.addInitScript(() => {
      localStorage.removeItem('auth_token');
    });

    await page.goto('/bookmarks');
    
    // Should show login prompt
    await expect(page.locator('text=Please log in to view your bookmarks')).toBeVisible();
    await expect(page.locator('button:has-text("Log In")')).toBeVisible();
    
    // Should not show bookmarks list
    await expect(page.locator('text=The Great Adventure')).not.toBeVisible();
  });

  test('can cancel bookmark creation', async ({ page }) => {
    await page.route('http://localhost:8082/api/v1/works/test-work-id', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          work: {
            id: 'test-work-id',
            title: 'Test Work',
            summary: 'A test work',
            rating: 'general',
            fandoms: ['Test Fandom'],
            word_count: 1000,
            chapter_count: 1,
            is_complete: true,
            status: 'posted',
            hits: 100,
            kudos: 10,
            comments: 5,
            bookmarks: 3
          },
          authors: [
            { pseud_name: 'TestAuthor', username: 'testauthor', is_anonymous: false }
          ]
        }),
      });
    });

    await page.route('http://localhost:8082/api/v1/works/test-work-id/chapters', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ chapters: [] }),
      });
    });

    await page.goto('/works/test-work-id');
    
    // Click bookmark button
    await page.click('button:has-text("🔖 Bookmark")');
    
    // Dialog should appear
    await expect(page.locator('text=Add Bookmark')).toBeVisible();
    
    // Cancel dialog
    await page.click('button:has-text("Cancel")');
    
    // Dialog should close
    await expect(page.locator('text=Add Bookmark')).not.toBeVisible();
    
    // Button should still show "Bookmark" (unchanged)
    await expect(page.locator('button:has-text("🔖 Bookmark")')).toBeVisible();
  });

  test('shows error when bookmark creation fails', async ({ page }) => {
    // Mock API error response
    await page.route('http://localhost:8082/api/v1/works/test-work-id/bookmark', async (route) => {
      if (route.request().method() === 'POST') {
        await route.fulfill({
          status: 409,
          contentType: 'application/json',
          body: JSON.stringify({ error: 'You have already bookmarked this work' }),
        });
      }
    });

    await page.route('http://localhost:8082/api/v1/works/test-work-id', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          work: {
            id: 'test-work-id',
            title: 'Test Work',
            summary: 'A test work',
            rating: 'general',
            fandoms: ['Test Fandom'],
            word_count: 1000,
            chapter_count: 1,
            is_complete: true,
            status: 'posted',
            hits: 100,
            kudos: 10,
            comments: 5,
            bookmarks: 3
          },
          authors: [
            { pseud_name: 'TestAuthor', username: 'testauthor', is_anonymous: false }
          ]
        }),
      });
    });

    await page.route('http://localhost:8082/api/v1/works/test-work-id/chapters', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ chapters: [] }),
      });
    });

    await page.goto('/works/test-work-id');
    
    // Click bookmark button
    await page.click('button:has-text("🔖 Bookmark")');
    
    // Submit bookmark
    await page.click('button:has-text("Add Bookmark")');
    
    // Error message should appear
    await expect(page.locator('text=You have already bookmarked this work')).toBeVisible();
    
    // Dialog should still be open
    await expect(page.locator('text=Add Bookmark')).toBeVisible();
  });

  test('navigation to bookmarks works correctly', async ({ page }) => {
    await page.goto('/');
    
    // Navigate using header link
    await page.click('a[href="/bookmarks"]');
    
    // Should arrive at bookmarks page
    await expect(page).toHaveURL('/bookmarks');
    await expect(page.locator('h1:has-text("My Bookmarks")')).toBeVisible();
  });

  test('bookmark privacy toggle works correctly', async ({ page }) => {
    // Mock bookmarks with mixed privacy settings
    await page.route('http://localhost:8082/api/v1/bookmarks', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          bookmarks: [
            {
              id: 'bookmark-1',
              work_id: 'work-1',
              notes: 'Public bookmark',
              tags: ['public'],
              is_private: false,
              created_at: '2024-01-01T00:00:00Z',
              updated_at: '2024-01-01T00:00:00Z',
              work: {
                id: 'work-1',
                title: 'Public Work',
                summary: 'A public bookmark...',
                rating: 'general',
                fandoms: ['Test Fandom'],
                characters: [],
                relationships: [],
                freeform_tags: [],
                word_count: 1000,
                chapter_count: 1,
                is_complete: true,
                status: 'posted',
                published_at: '2024-01-01T00:00:00Z',
                updated_at: '2024-01-01T00:00:00Z',
                hits: 100,
                kudos: 10,
                comments: 5,
                bookmarks: 1,
                authors: [
                  { pseud_name: 'TestAuthor', username: 'testauthor', is_anonymous: false }
                ]
              }
            },
            {
              id: 'bookmark-2',
              work_id: 'work-2',
              notes: 'Private bookmark',
              tags: ['private'],
              is_private: true,
              created_at: '2024-01-02T00:00:00Z',
              updated_at: '2024-01-02T00:00:00Z',
              work: {
                id: 'work-2',
                title: 'Private Work',
                summary: 'A private bookmark...',
                rating: 'general',
                fandoms: ['Test Fandom'],
                characters: [],
                relationships: [],
                freeform_tags: [],
                word_count: 2000,
                chapter_count: 1,
                is_complete: true,
                status: 'posted',
                published_at: '2024-01-02T00:00:00Z',
                updated_at: '2024-01-02T00:00:00Z',
                hits: 50,
                kudos: 5,
                comments: 2,
                bookmarks: 1,
                authors: [
                  { pseud_name: 'TestAuthor', username: 'testauthor', is_anonymous: false }
                ]
              }
            }
          ],
          pagination: {
            page: 1,
            limit: 20,
            total: 2,
            total_pages: 1
          }
        }),
      });
    });

    await page.goto('/bookmarks');
    
    // Both bookmarks should be visible initially
    await expect(page.locator('text=Public Work')).toBeVisible();
    await expect(page.locator('text=Private Work')).toBeVisible();
    
    // Uncheck "Show private bookmarks"
    await page.uncheck('input[type="checkbox"]:near(:text("Show private bookmarks"))');
    
    // Only public bookmark should be visible
    await expect(page.locator('text=Public Work')).toBeVisible();
    await expect(page.locator('text=Private Work')).not.toBeVisible();
    
    // Check again to show private bookmarks
    await page.check('input[type="checkbox"]:near(:text("Show private bookmarks"))');
    
    // Both should be visible again
    await expect(page.locator('text=Public Work')).toBeVisible();
    await expect(page.locator('text=Private Work')).toBeVisible();
  });
});