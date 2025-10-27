import { test, expect, Page, BrowserContext } from '@playwright/test';

// Test configuration
const TEST_BASE_URL = process.env.TEST_BASE_URL || 'http://localhost:3002';
const API_BASE_URL = process.env.API_BASE_URL || 'http://localhost:8080';

// Test data
const TEST_USER = {
  email: 'subscription-test@example.com',
  password: 'testpassword123',
  username: 'subscriptiontester',
  displayName: 'Subscription Tester'
};

const TEST_WORK = {
  title: 'Test Work for Subscriptions',
  summary: 'A test work to verify subscription functionality',
  fandom: 'Test Fandom'
};

test.describe('Subscription System', () => {
  let context: BrowserContext;
  let page: Page;
  let authToken: string;
  let userId: string;
  let workId: string;

  test.beforeAll(async ({ browser }) => {
    context = await browser.newContext();
    page = await context.newPage();
    
    // Setup: Create test user and work
    await setupTestData();
  });

  test.afterAll(async () => {
    // Cleanup: Remove test data
    await cleanupTestData();
    await context.close();
  });

  async function setupTestData() {
    // Register test user
    const registerResponse = await page.request.post(`${API_BASE_URL}/api/v1/auth/register`, {
      data: {
        username: TEST_USER.username,
        email: TEST_USER.email,
        password: TEST_USER.password,
        display_name: TEST_USER.displayName
      }
    });
    
    if (registerResponse.ok()) {
      const registerData = await registerResponse.json();
      authToken = registerData.access_token;
      userId = registerData.user.id;
    } else {
      // User might already exist, try to login
      const loginResponse = await page.request.post(`${API_BASE_URL}/api/v1/auth/login`, {
        data: {
          email: TEST_USER.email,
          password: TEST_USER.password
        }
      });
      
      if (loginResponse.ok()) {
        const loginData = await loginResponse.json();
        authToken = loginData.access_token;
        userId = loginData.user.id;
      } else {
        throw new Error('Failed to setup test user');
      }
    }

    // Create test work
    const workResponse = await page.request.post(`${API_BASE_URL}/api/v1/works`, {
      headers: {
        'Authorization': `Bearer ${authToken}`,
        'Content-Type': 'application/json'
      },
      data: {
        title: TEST_WORK.title,
        summary: TEST_WORK.summary,
        fandoms: [TEST_WORK.fandom],
        rating: 'General Audiences',
        warnings: ['No Archive Warnings Apply'],
        categories: ['Gen'],
        status: 'posted',
        first_chapter: {
          title: 'Chapter 1',
          content: 'Test chapter content for subscription testing.'
        }
      }
    });

    if (workResponse.ok()) {
      const workData = await workResponse.json();
      workId = workData.work.id;
    } else {
      throw new Error('Failed to create test work');
    }

    // Set auth token in browser for frontend tests
    await page.goto(TEST_BASE_URL);
    await page.evaluate((token) => {
      localStorage.setItem('auth_token', token);
      document.cookie = `auth_token=${token}; path=/`;
    }, authToken);
  }

  async function cleanupTestData() {
    if (workId && authToken) {
      // Delete test work
      await page.request.delete(`${API_BASE_URL}/api/v1/works/${workId}`, {
        headers: { 'Authorization': `Bearer ${authToken}` }
      });
    }
    
    // Clean up any test subscriptions
    if (authToken) {
      const subscriptionsResponse = await page.request.get(`${API_BASE_URL}/api/v1/subscriptions`, {
        headers: { 'Authorization': `Bearer ${authToken}` }
      });
      
      if (subscriptionsResponse.ok()) {
        const subscriptionsData = await subscriptionsResponse.json();
        const subscriptions = subscriptionsData.subscriptions || [];
        
        for (const subscription of subscriptions) {
          await page.request.delete(`${API_BASE_URL}/api/v1/subscriptions/${subscription.id}`, {
            headers: { 'Authorization': `Bearer ${authToken}` }
          });
        }
      }
    }
  }

  test.describe('Subscription Button on Work Page', () => {
    test.beforeEach(async () => {
      await page.goto(`${TEST_BASE_URL}/works/${workId}`);
      await page.waitForLoadState('networkidle');
    });

    test('should display subscription button for authenticated users', async () => {
      await expect(page.locator('button:has-text("Subscribe")')).toBeVisible();
    });

    test('should show subscription modal when clicked', async () => {
      await page.click('button:has-text("Subscribe")');
      
      await expect(page.locator('text=Subscribe to')).toBeVisible();
      await expect(page.locator('text=Get notified when there are updates')).toBeVisible();
      await expect(page.locator('select', { hasText: 'Immediate' })).toBeVisible();
    });

    test('should allow creating a subscription with default settings', async () => {
      await page.click('button:has-text("Subscribe")');
      await page.waitForSelector('text=Subscribe to');
      
      // Verify default frequency is selected
      await expect(page.locator('select[value="immediate"]')).toBeVisible();
      
      // Verify events are pre-selected
      await expect(page.locator('input[type="checkbox"]:checked')).toHaveCount(2);
      
      // Create subscription
      await page.click('button:has-text("Subscribe"):not(:has-text("Subscribe to"))');
      
      // Verify subscription was created
      await expect(page.locator('button:has-text("Unsubscribe")')).toBeVisible();
    });

    test('should allow customizing subscription settings', async () => {
      await page.click('button:has-text("Subscribe")');
      await page.waitForSelector('text=Subscribe to');
      
      // Change frequency to daily
      await page.selectOption('select', 'daily');
      
      // Uncheck one event
      const checkboxes = page.locator('input[type="checkbox"]');
      await checkboxes.first().uncheck();
      
      // Create subscription
      await page.click('button:has-text("Subscribe"):not(:has-text("Subscribe to"))');
      
      // Verify subscription was created
      await expect(page.locator('button:has-text("Unsubscribe")')).toBeVisible();
    });

    test('should show unsubscribe button for existing subscriptions', async () => {
      // First create a subscription
      await page.click('button:has-text("Subscribe")');
      await page.waitForSelector('text=Subscribe to');
      await page.click('button:has-text("Subscribe"):not(:has-text("Subscribe to"))');
      await expect(page.locator('button:has-text("Unsubscribe")')).toBeVisible();
      
      // Refresh page and verify subscription persists
      await page.reload();
      await page.waitForLoadState('networkidle');
      await expect(page.locator('button:has-text("Unsubscribe")')).toBeVisible();
    });

    test('should allow unsubscribing from work', async () => {
      // First create a subscription
      await page.click('button:has-text("Subscribe")');
      await page.waitForSelector('text=Subscribe to');
      await page.click('button:has-text("Subscribe"):not(:has-text("Subscribe to"))');
      await expect(page.locator('button:has-text("Unsubscribe")')).toBeVisible();
      
      // Now unsubscribe
      await page.click('button:has-text("Unsubscribe")');
      
      // Verify subscription was removed
      await expect(page.locator('button:has-text("Subscribe")')).toBeVisible();
      await expect(page.locator('button:has-text("Unsubscribe")')).not.toBeVisible();
    });

    test('should validate subscription form', async () => {
      await page.click('button:has-text("Subscribe")');
      await page.waitForSelector('text=Subscribe to');
      
      // Uncheck all events
      const checkboxes = page.locator('input[type="checkbox"]');
      const count = await checkboxes.count();
      for (let i = 0; i < count; i++) {
        await checkboxes.nth(i).uncheck();
      }
      
      // Try to submit with no events selected
      const subscribeButton = page.locator('button:has-text("Subscribe"):not(:has-text("Subscribe to"))');
      await expect(subscribeButton).toBeDisabled();
    });
  });

  test.describe('Subscription Management Dashboard', () => {
    test.beforeEach(async () => {
      // Create a test subscription first
      await page.goto(`${TEST_BASE_URL}/works/${workId}`);
      await page.waitForLoadState('networkidle');
      
      // Only create subscription if it doesn't exist
      const subscribeButton = page.locator('button:has-text("Subscribe")');
      if (await subscribeButton.isVisible()) {
        await subscribeButton.click();
        await page.waitForSelector('text=Subscribe to');
        await page.click('button:has-text("Subscribe"):not(:has-text("Subscribe to"))');
        await expect(page.locator('button:has-text("Unsubscribe")')).toBeVisible();
      }
      
      // Navigate to subscriptions page
      await page.goto(`${TEST_BASE_URL}/subscriptions`);
      await page.waitForLoadState('networkidle');
    });

    test('should display subscriptions dashboard', async () => {
      await expect(page.locator('h1:has-text("Your Subscriptions")')).toBeVisible();
      await expect(page.locator('text=Manage your notifications for works, authors, and more.')).toBeVisible();
    });

    test('should show existing subscriptions', async () => {
      await expect(page.locator(`text=${TEST_WORK.title}`)).toBeVisible();
      await expect(page.locator('text=Work subscription')).toBeVisible();
      await expect(page.locator('text=immediate', { exact: false })).toBeVisible();
    });

    test('should allow editing subscription preferences', async () => {
      // Click edit button
      await page.click('button:has-text("Edit")');
      
      // Verify edit modal opens
      await expect(page.locator('text=Edit Subscription')).toBeVisible();
      await expect(page.locator(`text=${TEST_WORK.title}`)).toBeVisible();
      
      // Change frequency to weekly
      await page.selectOption('select', 'weekly');
      
      // Update subscription
      await page.click('button:has-text("Update")');
      
      // Verify changes were saved
      await expect(page.locator('text=weekly', { exact: false })).toBeVisible();
    });

    test('should allow deleting subscriptions', async () => {
      // Click delete button
      page.on('dialog', async dialog => {
        expect(dialog.message()).toContain('Are you sure you want to delete this subscription?');
        await dialog.accept();
      });
      
      await page.click('button:has-text("Delete")');
      
      // Verify subscription was deleted
      await expect(page.locator('text=No subscriptions yet')).toBeVisible();
      await expect(page.locator(`text=${TEST_WORK.title}`)).not.toBeVisible();
    });

    test('should show empty state when no subscriptions exist', async () => {
      // Delete all subscriptions first
      while (await page.locator('button:has-text("Delete")').count() > 0) {
        page.on('dialog', async dialog => {
          await dialog.accept();
        });
        await page.click('button:has-text("Delete")');
        await page.waitForTimeout(1000); // Wait for deletion to complete
      }
      
      await expect(page.locator('text=No subscriptions yet')).toBeVisible();
      await expect(page.locator('text=Start following your favorite works')).toBeVisible();
      await expect(page.locator('a:has-text("Browse Works")')).toBeVisible();
    });

    test('should navigate to works from empty state', async () => {
      // First ensure we're in empty state
      while (await page.locator('button:has-text("Delete")').count() > 0) {
        page.on('dialog', async dialog => {
          await dialog.accept();
        });
        await page.click('button:has-text("Delete")');
        await page.waitForTimeout(1000);
      }
      
      await expect(page.locator('text=No subscriptions yet')).toBeVisible();
      
      // Click browse works link
      await page.click('a:has-text("Browse Works")');
      
      // Verify navigation
      await expect(page).toHaveURL(/\/works/);
    });
  });

  test.describe('Navigation Integration', () => {
    test('should show subscriptions link in navigation for authenticated users', async () => {
      await page.goto(TEST_BASE_URL);
      await page.waitForLoadState('networkidle');
      
      // Look for user menu or navigation
      const subscriptionsLink = page.locator('a:has-text("Subscriptions"), [href="/subscriptions"]');
      await expect(subscriptionsLink).toBeVisible();
    });

    test('should navigate to subscriptions page from navigation', async () => {
      await page.goto(TEST_BASE_URL);
      await page.waitForLoadState('networkidle');
      
      await page.click('a:has-text("Subscriptions")');
      await expect(page).toHaveURL(/\/subscriptions/);
      await expect(page.locator('h1:has-text("Your Subscriptions")')).toBeVisible();
    });
  });

  test.describe('API Error Handling', () => {
    test('should handle subscription creation errors gracefully', async () => {
      await page.goto(`${TEST_BASE_URL}/works/${workId}`);
      await page.waitForLoadState('networkidle');
      
      // Mock API failure
      await page.route(`${API_BASE_URL}/api/v1/subscriptions`, route => {
        route.fulfill({
          status: 500,
          contentType: 'application/json',
          body: JSON.stringify({ error: 'Internal server error' })
        });
      });
      
      await page.click('button:has-text("Subscribe")');
      await page.waitForSelector('text=Subscribe to');
      await page.click('button:has-text("Subscribe"):not(:has-text("Subscribe to"))');
      
      // Should show error message
      await expect(page.locator('text=Failed to create subscription')).toBeVisible();
    });

    test('should handle subscription loading errors', async () => {
      // Mock API failure for getting subscriptions
      await page.route(`${API_BASE_URL}/api/v1/subscriptions`, route => {
        route.fulfill({
          status: 500,
          contentType: 'application/json',
          body: JSON.stringify({ error: 'Internal server error' })
        });
      });
      
      await page.goto(`${TEST_BASE_URL}/subscriptions`);
      await page.waitForLoadState('networkidle');
      
      // Should show error message
      await expect(page.locator('text=Failed to load subscriptions')).toBeVisible();
    });
  });

  test.describe('Accessibility', () => {
    test('subscription button should be keyboard accessible', async () => {
      await page.goto(`${TEST_BASE_URL}/works/${workId}`);
      await page.waitForLoadState('networkidle');
      
      // Focus the subscription button using keyboard
      await page.keyboard.press('Tab');
      const subscribeButton = page.locator('button:has-text("Subscribe")');
      
      // Verify button can be activated with keyboard
      await subscribeButton.focus();
      await page.keyboard.press('Enter');
      
      await expect(page.locator('text=Subscribe to')).toBeVisible();
    });

    test('subscription modal should trap focus', async () => {
      await page.goto(`${TEST_BASE_URL}/works/${workId}`);
      await page.waitForLoadState('networkidle');
      
      await page.click('button:has-text("Subscribe")');
      await page.waitForSelector('text=Subscribe to');
      
      // Tab through modal elements - focus should stay within modal
      await page.keyboard.press('Tab');
      await page.keyboard.press('Tab');
      await page.keyboard.press('Tab');
      
      // Focus should be within the modal
      const focusedElement = await page.evaluate(() => document.activeElement?.closest('[role="dialog"]'));
      expect(focusedElement).toBeTruthy();
    });

    test('subscription management page should have proper headings', async () => {
      await page.goto(`${TEST_BASE_URL}/subscriptions`);
      await page.waitForLoadState('networkidle');
      
      // Check heading hierarchy
      await expect(page.locator('h1')).toHaveText('Your Subscriptions');
      
      // Check for proper ARIA labels and roles
      await expect(page.locator('[role="main"]')).toBeVisible();
    });
  });

  test.describe('Performance', () => {
    test('subscription button should load quickly', async () => {
      const startTime = Date.now();
      
      await page.goto(`${TEST_BASE_URL}/works/${workId}`);
      await page.waitForSelector('button:has-text("Subscribe")');
      
      const loadTime = Date.now() - startTime;
      expect(loadTime).toBeLessThan(5000); // Should load within 5 seconds
    });

    test('subscription status should be checked efficiently', async () => {
      await page.goto(`${TEST_BASE_URL}/works/${workId}`);
      
      // Monitor network requests
      const subscriptionStatusRequests = [];
      page.on('request', request => {
        if (request.url().includes('/subscription-status')) {
          subscriptionStatusRequests.push(request);
        }
      });
      
      await page.waitForLoadState('networkidle');
      
      // Should only make one status check request per page load
      expect(subscriptionStatusRequests.length).toBeLessThanOrEqual(1);
    });
  });
});