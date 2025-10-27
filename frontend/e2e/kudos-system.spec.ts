import { test, expect, Page, BrowserContext } from '@playwright/test';
import { loginUser, logoutUser, TEST_USER } from './test-utils';

// Configuration
const FIRST_PARTY_CLIENT_ID = '11111111-1111-1111-1111-111111111111';
const API_URL = process.env.API_URL || 'http://localhost:8080/api/v1';
const FRONTEND_URL = process.env.FRONTEND_URL || 'http://localhost:3001';

// Helper to get a test work ID
async function getTestWorkId(): Promise<string | null> {
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

// Helper to create a test work
async function createTestWork(authToken: string): Promise<string> {
  const workData = {
    title: `E2E Kudos Test Work ${Date.now()}`,
    summary: 'A test work for kudos system testing',
    content: 'This is test content for the kudos functionality.',
    language: 'en',
    rating: 'General Audiences',
    fandoms: ['Test Fandom'],
    characters: ['Test Character'],
    relationships: [],
    freeform_tags: ['Test Tag'],
    word_count: 50,
    is_complete: true,
    restricted_to_users: false,
    restricted_to_adults: false,
    disable_comments: false,
    moderate_comments: false,
    comment_policy: ''
  };

  const response = await fetch(`${API_URL}/works`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${authToken}`,
      'X-Client-ID': FIRST_PARTY_CLIENT_ID,
      'X-Client-First-Party': 'true',
    },
    body: JSON.stringify(workData)
  });

  if (!response.ok) {
    throw new Error(`Failed to create test work: ${response.status}`);
  }

  const result = await response.json();
  return result.id;
}

// Helper to get auth token from page
async function getAuthToken(page: Page): Promise<string | null> {
  return await page.evaluate(() => {
    return localStorage.getItem('auth_token');
  });
}

// Helper to check kudos count via API
async function getKudosCount(workId: string): Promise<number> {
  try {
    const response = await fetch(`${API_URL}/works/${workId}`);
    if (!response.ok) return 0;
    
    const work = await response.json();
    return work.kudos || 0;
  } catch (error) {
    console.warn('Failed to get kudos count:', error);
    return 0;
  }
}

test.describe('Kudos System E2E Tests', () => {
  let testWorkId: string;

  test.beforeAll(async () => {
    // Try to get existing work first
    testWorkId = await getTestWorkId() || '';
    
    if (!testWorkId) {
      console.warn('No test work found, tests may need to create one');
    }
  });

  test.beforeEach(async ({ page }) => {
    // Set up authentication
    await loginUser(page);
  });

  test('should display kudos button on work page', async ({ page }) => {
    test.skip(!testWorkId, 'No test work available');

    await test.step('Navigate to work page', async () => {
      await page.goto(`/works/${testWorkId}`);
      await page.waitForLoadState('networkidle');
    });

    await test.step('Verify kudos button is visible', async () => {
      // Wait for the kudos button to appear
      const kudosButton = page.locator('button:has-text("Give Kudos"), button:has-text("Kudos Given"), button:has-text("Loading")').first();
      await expect(kudosButton).toBeVisible({ timeout: 10000 });
      
      // Verify kudos count is displayed
      await expect(kudosButton).toContainText(/\(\d+\)/);
    });

    await test.step('Verify kudos button has correct accessibility attributes', async () => {
      const kudosButton = page.locator('button:has-text("Give Kudos"), button:has-text("Kudos Given")').first();
      
      // Check that button has proper title attribute
      const title = await kudosButton.getAttribute('title');
      expect(title).toBeTruthy();
      expect(title).toMatch(/(Leave kudos|already.*kudos)/i);
    });
  });

  test('should give kudos successfully when logged in', async ({ page }) => {
    test.skip(!testWorkId, 'No test work available');

    let initialKudosCount: number;
    let authToken: string | null;

    await test.step('Get initial state', async () => {
      await page.goto(`/works/${testWorkId}`);
      await page.waitForLoadState('networkidle');
      
      // Get auth token
      authToken = await getAuthToken(page);
      expect(authToken).toBeTruthy();
      
      // Get initial kudos count
      initialKudosCount = await getKudosCount(testWorkId);
    });

    await test.step('Click kudos button', async () => {
      // Wait for kudos button to be ready (not in loading state)
      await page.waitForSelector('button:has-text("Give Kudos")', { timeout: 10000 });
      
      const kudosButton = page.locator('button:has-text("Give Kudos")').first();
      await expect(kudosButton).toBeVisible();
      await expect(kudosButton).not.toBeDisabled();
      
      // Click the kudos button
      await kudosButton.click();
    });

    await test.step('Verify kudos was given', async () => {
      // Wait for button text to change
      await expect(page.locator('button:has-text("Kudos Given")').first()).toBeVisible({ timeout: 5000 });
      
      // Verify button is now disabled
      const kudosButton = page.locator('button:has-text("Kudos Given")').first();
      await expect(kudosButton).toBeDisabled();
      
      // Verify kudos count increased
      const newKudosCount = await getKudosCount(testWorkId);
      expect(newKudosCount).toBe(initialKudosCount + 1);
      
      // Verify UI shows updated count
      await expect(kudosButton).toContainText(`(${newKudosCount})`);
    });

    await test.step('Verify kudos cannot be given again', async () => {
      // Refresh the page
      await page.reload();
      await page.waitForLoadState('networkidle');
      
      // Wait for button to load and verify it shows "Kudos Given"
      await expect(page.locator('button:has-text("Kudos Given")').first()).toBeVisible({ timeout: 10000 });
      
      const kudosButton = page.locator('button:has-text("Kudos Given")').first();
      await expect(kudosButton).toBeDisabled();
    });
  });

  test('should handle guest kudos correctly', async ({ page }) => {
    test.skip(!testWorkId, 'No test work available');

    await test.step('Logout to become guest user', async () => {
      await logoutUser(page);
    });

    await test.step('Navigate to work as guest', async () => {
      await page.goto(`/works/${testWorkId}`);
      await page.waitForLoadState('networkidle');
    });

    await test.step('Verify guest can give kudos', async () => {
      // Wait for kudos button (should still be available for guests)
      const kudosButton = page.locator('button:has-text("Give Kudos"), button:has-text("Loading")').first();
      await expect(kudosButton).toBeVisible({ timeout: 10000 });
      
      // If not loading, should be able to give kudos as guest
      if (await kudosButton.textContent() === 'Give Kudos') {
        await kudosButton.click();
        
        // Should show success (either "Kudos Given" or increased count)
        await expect(page.locator('button:has-text("Kudos Given")').first()).toBeVisible({ timeout: 5000 });
      }
    });
  });

  test('should prevent author from giving kudos to own work', async ({ page, browser }) => {
    let workId: string;
    let authToken: string | null;

    await test.step('Create a new work as current user', async () => {
      authToken = await getAuthToken(page);
      expect(authToken).toBeTruthy();
      
      workId = await createTestWork(authToken!);
      expect(workId).toBeTruthy();
    });

    await test.step('Navigate to own work', async () => {
      await page.goto(`/works/${workId}`);
      await page.waitForLoadState('networkidle');
    });

    await test.step('Verify kudos button behavior for author', async () => {
      // Button should either be disabled or show "Cannot give kudos to your own work"
      await page.waitForTimeout(2000); // Wait for API calls to complete
      
      const kudosButton = page.locator('button:has-text("Give Kudos"), button:has-text("Cannot"), button:has-text("Loading")').first();
      await expect(kudosButton).toBeVisible({ timeout: 10000 });
      
      // If button says "Give Kudos", clicking it should show an error
      if (await kudosButton.textContent() === 'Give Kudos') {
        await kudosButton.click();
        
        // Should show error message
        await expect(page.locator('text=Cannot give kudos to your own work, text=your own work').first()).toBeVisible({ timeout: 3000 });
      }
    });

    await test.step('Verify different user can give kudos', async () => {
      // Create a new browser context for a different user
      const newContext = await browser.newContext();
      const newPage = await newContext.newPage();
      
      try {
        // Don't login - test as guest
        await newPage.goto(`/works/${workId}`);
        await newPage.waitForLoadState('networkidle');
        
        // Guest should be able to give kudos
        const guestKudosButton = newPage.locator('button:has-text("Give Kudos")').first();
        if (await guestKudosButton.isVisible()) {
          await guestKudosButton.click();
          await expect(newPage.locator('button:has-text("Kudos Given")').first()).toBeVisible({ timeout: 5000 });
        }
      } finally {
        await newContext.close();
      }
    });
  });

  test('should show correct kudos count and state across page reloads', async ({ page }) => {
    test.skip(!testWorkId, 'No test work available');

    let initialCount: number;

    await test.step('Get initial state', async () => {
      await page.goto(`/works/${testWorkId}`);
      await page.waitForLoadState('networkidle');
      
      // Wait for kudos button to load
      await page.waitForSelector('button:has-text("Give Kudos"), button:has-text("Kudos Given"), button:has-text("Loading")', { timeout: 10000 });
      
      // Extract kudos count from button text
      const kudosButton = page.locator('button:has-text("Give Kudos"), button:has-text("Kudos Given")').first();
      const buttonText = await kudosButton.textContent();
      const countMatch = buttonText?.match(/\((\d+)\)/);
      initialCount = countMatch ? parseInt(countMatch[1]) : 0;
    });

    await test.step('Reload page and verify state persistence', async () => {
      await page.reload();
      await page.waitForLoadState('networkidle');
      
      // Wait for button to load
      const kudosButton = page.locator('button:has-text("Give Kudos"), button:has-text("Kudos Given"), button:has-text("Loading")').first();
      await expect(kudosButton).toBeVisible({ timeout: 10000 });
      
      // Wait for loading to complete
      await expect(page.locator('button:has-text("Loading")')).not.toBeVisible({ timeout: 5000 });
      
      // Verify count is the same
      await expect(kudosButton).toContainText(`(${initialCount})`);
    });

    await test.step('Verify API consistency', async () => {
      const apiCount = await getKudosCount(testWorkId);
      expect(apiCount).toBe(initialCount);
    });
  });

  test('should handle network errors gracefully', async ({ page }) => {
    test.skip(!testWorkId, 'No test work available');

    await test.step('Navigate to work page', async () => {
      await page.goto(`/works/${testWorkId}`);
      await page.waitForLoadState('networkidle');
    });

    await test.step('Mock network failure and test error handling', async () => {
      // Intercept kudos API calls and make them fail
      await page.route(`**/api/v1/works/${testWorkId}/kudos`, route => {
        route.abort('failed');
      });

      // Try to give kudos
      const kudosButton = page.locator('button:has-text("Give Kudos")').first();
      if (await kudosButton.isVisible() && !await kudosButton.isDisabled()) {
        await kudosButton.click();
        
        // Should show error message
        await expect(page.locator('text=Failed, text=error').first()).toBeVisible({ timeout: 5000 });
      }
    });
  });

  test('should maintain accessibility standards', async ({ page }) => {
    test.skip(!testWorkId, 'No test work available');

    await test.step('Navigate to work page', async () => {
      await page.goto(`/works/${testWorkId}`);
      await page.waitForLoadState('networkidle');
    });

    await test.step('Verify keyboard accessibility', async () => {
      const kudosButton = page.locator('button:has-text("Give Kudos"), button:has-text("Kudos Given")').first();
      await expect(kudosButton).toBeVisible({ timeout: 10000 });
      
      // Button should be focusable
      await kudosButton.focus();
      await expect(kudosButton).toBeFocused();
      
      // Should have proper ARIA attributes
      const role = await kudosButton.getAttribute('role');
      expect(role === null || role === 'button').toBeTruthy(); // Default button role is acceptable
      
      // Should have descriptive title
      const title = await kudosButton.getAttribute('title');
      expect(title).toBeTruthy();
    });

    await test.step('Verify screen reader support', async () => {
      const kudosButton = page.locator('button:has-text("Give Kudos"), button:has-text("Kudos Given")').first();
      
      // Button text should be descriptive
      const buttonText = await kudosButton.textContent();
      expect(buttonText).toMatch(/(Give Kudos|Kudos Given|Loading)/);
      
      // Should include kudos count for context
      expect(buttonText).toMatch(/\(\d+\)/);
    });
  });
});

test.describe('Kudos System API Integration', () => {
  let authToken: string;
  let testWorkId: string;

  test.beforeAll(async ({ browser }) => {
    // Get auth token via login
    const page = await browser.newPage();
    await loginUser(page);
    authToken = await getAuthToken(page) || '';
    await page.close();
    
    expect(authToken).toBeTruthy();
    
    // Get or create test work
    testWorkId = await getTestWorkId() || '';
    if (!testWorkId) {
      testWorkId = await createTestWork(authToken);
    }
  });

  test('should handle API kudos endpoints correctly', async () => {
    await test.step('GET kudos endpoint should return correct format', async () => {
      const response = await fetch(`${API_URL}/works/${testWorkId}/kudos`, {
        headers: {
          'Authorization': `Bearer ${authToken}`,
          'X-Client-ID': FIRST_PARTY_CLIENT_ID,
        }
      });
      
      expect(response.ok).toBeTruthy();
      
      const data = await response.json();
      expect(data).toHaveProperty('kudos');
      expect(data).toHaveProperty('has_given_kudos');
      expect(data).toHaveProperty('total_count');
      expect(Array.isArray(data.kudos)).toBeTruthy();
      expect(typeof data.has_given_kudos).toBe('boolean');
      expect(typeof data.total_count).toBe('number');
    });

    await test.step('POST kudos endpoint should work correctly', async () => {
      const initialResponse = await fetch(`${API_URL}/works/${testWorkId}/kudos`, {
        headers: {
          'Authorization': `Bearer ${authToken}`,
          'X-Client-ID': FIRST_PARTY_CLIENT_ID,
        }
      });
      
      const initialData = await initialResponse.json();
      const initialCount = initialData.total_count;
      const hadGivenKudos = initialData.has_given_kudos;
      
      if (!hadGivenKudos) {
        // Give kudos
        const giveResponse = await fetch(`${API_URL}/works/${testWorkId}/kudos`, {
          method: 'POST',
          headers: {
            'Authorization': `Bearer ${authToken}`,
            'X-Client-ID': FIRST_PARTY_CLIENT_ID,
          }
        });
        
        expect(giveResponse.ok).toBeTruthy();
        
        const giveData = await giveResponse.json();
        expect(giveData).toHaveProperty('message');
        
        // Verify count increased
        const afterResponse = await fetch(`${API_URL}/works/${testWorkId}/kudos`, {
          headers: {
            'Authorization': `Bearer ${authToken}`,
            'X-Client-ID': FIRST_PARTY_CLIENT_ID,
          }
        });
        
        const afterData = await afterResponse.json();
        expect(afterData.total_count).toBe(initialCount + 1);
        expect(afterData.has_given_kudos).toBe(true);
      }
    });

    await test.step('Should prevent duplicate kudos', async () => {
      // Try to give kudos again
      const response = await fetch(`${API_URL}/works/${testWorkId}/kudos`, {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${authToken}`,
          'X-Client-ID': FIRST_PARTY_CLIENT_ID,
        }
      });
      
      // Should return conflict or similar
      expect(response.status).toBeGreaterThanOrEqual(400);
      
      const data = await response.json();
      expect(data).toHaveProperty('error');
      expect(data.error).toMatch(/already.*kudos/i);
    });
  });
});