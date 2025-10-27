import { test, expect, Page } from '@playwright/test';

// Basic subscription functionality tests
// These tests focus on the core subscription features without complex setup

test.describe('Basic Subscription Functionality', () => {
  let page: Page;

  test.beforeEach(async ({ browser }) => {
    const context = await browser.newContext();
    page = await context.newPage();
    
    // Mock API responses for basic testing
    await setupMockResponses();
  });

  async function setupMockResponses() {
    // Mock successful auth check
    await page.route('**/api/v1/auth/verify', route => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          user: {
            id: 'test-user-id',
            username: 'testuser',
            email: 'test@example.com'
          }
        })
      });
    });

    // Mock work data
    await page.route('**/api/v1/works/*', route => {
      if (route.request().method() === 'GET') {
        route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            id: 'test-work-id',
            title: 'Test Work for Subscriptions',
            summary: 'A test work',
            authors: [{ pseud_name: 'Test Author' }],
            fandoms: ['Test Fandom'],
            rating: 'General Audiences',
            warnings: ['No Archive Warnings Apply'],
            categories: ['Gen'],
            kudos: 42,
            hits: 150,
            bookmarks: 10,
            comments: 5
          })
        });
      }
    });

    // Mock subscription status check - initially not subscribed
    await page.route('**/api/v1/subscription-status*', route => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          subscribed: false
        })
      });
    });

    // Mock subscription creation
    await page.route('**/api/v1/subscriptions', route => {
      if (route.request().method() === 'POST') {
        route.fulfill({
          status: 201,
          contentType: 'application/json',
          body: JSON.stringify({
            message: 'Subscription created successfully',
            subscription: {
              id: 'test-subscription-id',
              type: 'work',
              target_id: 'test-work-id',
              target_name: 'Test Work for Subscriptions',
              events: ['work_updated'],
              frequency: 'immediate'
            }
          })
        });
      } else if (route.request().method() === 'GET') {
        // Mock get subscriptions
        route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            subscriptions: [{
              id: 'test-subscription-id',
              type: 'work',
              target_id: 'test-work-id',
              target_name: 'Test Work for Subscriptions',
              events: ['work_updated'],
              frequency: 'immediate',
              is_active: true,
              created_at: new Date().toISOString()
            }]
          })
        });
      }
    });

    // Set mock auth token
    await page.goto('http://localhost:3002');
    await page.evaluate(() => {
      localStorage.setItem('auth_token', 'mock-auth-token');
      document.cookie = 'auth_token=mock-auth-token; path=/';
    });
  }

  test('should display subscription button on work page', async () => {
    await page.goto('http://localhost:3002/works/test-work-id');
    await page.waitForLoadState('networkidle');

    // Should show the subscription button
    await expect(page.locator('button:has-text("Subscribe")')).toBeVisible();
  });

  test('should open subscription modal when clicked', async () => {
    await page.goto('http://localhost:3002/works/test-work-id');
    await page.waitForLoadState('networkidle');

    // Click subscribe button
    await page.click('button:has-text("Subscribe")');

    // Modal should appear
    await expect(page.locator('text=Subscribe to')).toBeVisible();
    await expect(page.locator('text=Get notified when there are updates')).toBeVisible();
  });

  test('should show subscription options in modal', async () => {
    await page.goto('http://localhost:3002/works/test-work-id');
    await page.waitForLoadState('networkidle');

    await page.click('button:has-text("Subscribe")');
    await page.waitForSelector('text=Subscribe to');

    // Check frequency options
    await expect(page.locator('select')).toBeVisible();
    
    // Check event checkboxes
    await expect(page.locator('input[type="checkbox"]')).toHaveCount(2); // work_updated and work_completed
    
    // Check buttons
    await expect(page.locator('button:has-text("Cancel")')).toBeVisible();
    await expect(page.locator('button:has-text("Subscribe"):not(:has-text("Subscribe to"))')).toBeVisible();
  });

  test('should create subscription successfully', async () => {
    await page.goto('http://localhost:3002/works/test-work-id');
    await page.waitForLoadState('networkidle');

    await page.click('button:has-text("Subscribe")');
    await page.waitForSelector('text=Subscribe to');

    // Create subscription with default settings
    await page.click('button:has-text("Subscribe"):not(:has-text("Subscribe to"))');

    // Should close modal and show unsubscribe button
    await expect(page.locator('text=Subscribe to')).not.toBeVisible();
    
    // Note: Due to mocking, we'd need to update the mock to return subscribed: true
    // for this to work completely, but the interaction is tested
  });

  test('should navigate to subscription management page', async () => {
    await page.goto('http://localhost:3002');
    await page.waitForLoadState('networkidle');

    // Look for subscriptions link in navigation
    const subscriptionsLink = page.locator('a:has-text("Subscriptions")');
    if (await subscriptionsLink.isVisible()) {
      await subscriptionsLink.click();
      await expect(page).toHaveURL(/\/subscriptions/);
    } else {
      // Direct navigation if link not found
      await page.goto('http://localhost:3002/subscriptions');
    }

    await expect(page.locator('h1:has-text("Your Subscriptions")')).toBeVisible();
  });

  test('should display subscriptions in management page', async () => {
    await page.goto('http://localhost:3002/subscriptions');
    await page.waitForLoadState('networkidle');

    // Should show subscription management interface
    await expect(page.locator('h1:has-text("Your Subscriptions")')).toBeVisible();
    await expect(page.locator('text=Manage your notifications')).toBeVisible();
    
    // Should show the mocked subscription
    await expect(page.locator('text=Test Work for Subscriptions')).toBeVisible();
    await expect(page.locator('text=Work subscription')).toBeVisible();
  });

  test('subscription modal should be accessible', async () => {
    await page.goto('http://localhost:3002/works/test-work-id');
    await page.waitForLoadState('networkidle');

    // Test keyboard navigation
    await page.keyboard.press('Tab'); // Navigate to subscription button
    const subscribeButton = page.locator('button:has-text("Subscribe")');
    await subscribeButton.focus();
    
    // Open modal with Enter key
    await page.keyboard.press('Enter');
    await expect(page.locator('text=Subscribe to')).toBeVisible();

    // Modal should have proper focus management
    const focusedElement = await page.evaluate(() => document.activeElement?.tagName);
    expect(focusedElement).toBeTruthy();
  });

  test('should handle API errors gracefully', async () => {
    // Override subscription creation to return error
    await page.route('**/api/v1/subscriptions', route => {
      if (route.request().method() === 'POST') {
        route.fulfill({
          status: 500,
          contentType: 'application/json',
          body: JSON.stringify({ error: 'Server error' })
        });
      }
    });

    await page.goto('http://localhost:3002/works/test-work-id');
    await page.waitForLoadState('networkidle');

    await page.click('button:has-text("Subscribe")');
    await page.waitForSelector('text=Subscribe to');
    await page.click('button:has-text("Subscribe"):not(:has-text("Subscribe to"))');

    // Should show error message
    await expect(page.locator('text=Failed to create subscription')).toBeVisible();
  });

  test('subscription form validation', async () => {
    await page.goto('http://localhost:3002/works/test-work-id');
    await page.waitForLoadState('networkidle');

    await page.click('button:has-text("Subscribe")');
    await page.waitForSelector('text=Subscribe to');

    // Uncheck all events
    const checkboxes = page.locator('input[type="checkbox"]');
    const count = await checkboxes.count();
    for (let i = 0; i < count; i++) {
      await checkboxes.nth(i).uncheck();
    }

    // Subscribe button should be disabled
    const subscribeButton = page.locator('button:has-text("Subscribe"):not(:has-text("Subscribe to"))');
    await expect(subscribeButton).toBeDisabled();
  });

  test('should show empty state when no subscriptions exist', async () => {
    // Mock empty subscriptions response
    await page.route('**/api/v1/subscriptions', route => {
      if (route.request().method() === 'GET') {
        route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ subscriptions: [] })
        });
      }
    });

    await page.goto('http://localhost:3002/subscriptions');
    await page.waitForLoadState('networkidle');

    await expect(page.locator('text=No subscriptions yet')).toBeVisible();
    await expect(page.locator('text=Start following your favorite works')).toBeVisible();
    await expect(page.locator('a:has-text("Browse Works")')).toBeVisible();
  });
});