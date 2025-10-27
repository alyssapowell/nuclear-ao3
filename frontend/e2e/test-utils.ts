import { Page, expect } from '@playwright/test';

// Test configuration
const API_URL = process.env.API_URL || 'http://localhost:8080/api/v1';
const FIRST_PARTY_CLIENT_ID = '11111111-1111-1111-1111-111111111111';

// Test user data
export const TEST_USER = {
  email: `test_user_${Date.now()}@example.com`,
  username: `test_user_${Date.now()}`,
  password: 'TestPassword123!'
};

export async function loginUser(page: Page, userSuffix?: string): Promise<string> {
  // Use real authentication API for E2E tests
  const timestamp = Date.now();
  const suffix = userSuffix || timestamp.toString();
  const uniqueTestUser = {
    email: `e2e_test_${suffix}@example.com`,
    username: `e2e_test_${suffix}`,
    password: 'E2ETestPassword123!'
  };

  // First register the user via API
  try {
    const registerResponse = await fetch(`${API_URL}/auth/register`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'X-Client-ID': FIRST_PARTY_CLIENT_ID,
        'X-Client-First-Party': 'true',
        'X-OAuth-Scopes': 'openid,profile,email,read,write,works:manage,comments:write,bookmarks:manage,collections:manage',
      },
      body: JSON.stringify(uniqueTestUser)
    });

    if (!registerResponse.ok) {
      console.log('Registration failed, user might already exist. Attempting login...');
    }
  } catch (error) {
    console.log('Registration API call failed:', error);
  }

  // Login via OAuth API using first-party client to get proper rate limiting
  const loginResponse = await fetch(`${API_URL}/auth/login`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'X-Client-ID': FIRST_PARTY_CLIENT_ID,
      'X-Client-First-Party': 'true',
      'X-OAuth-Scopes': 'openid,profile,email,read,write,works:manage,comments:write,bookmarks:manage,collections:manage',
    },
    body: JSON.stringify({
      email: uniqueTestUser.email,
      password: uniqueTestUser.password
    })
  });

  if (!loginResponse.ok) {
    throw new Error(`Login failed: ${loginResponse.status} ${loginResponse.statusText}`);
  }

  const loginData = await loginResponse.json();
  const token = loginData.access_token;

  if (!token) {
    throw new Error('No access token received from login');
  }

  // Navigate to the home page to establish localStorage context
  await page.goto('/');
  
  // Set authentication tokens in localStorage
  await page.evaluate(({ authToken, userData }) => {
    localStorage.setItem('auth_token', authToken);
    localStorage.setItem('user', JSON.stringify(userData));
    localStorage.setItem('authenticated', 'true');
    
    // Manually trigger a storage event to notify any components listening
    window.dispatchEvent(new StorageEvent('storage', {
      key: 'auth_token',
      newValue: authToken,
      oldValue: null,
      url: window.location.href
    }));
  }, { authToken: token, userData: loginData.user });

  // Try to trigger a page reload to ensure AuthGuard recognizes the token
  await page.reload();
  await page.waitForLoadState('networkidle');

  console.log('✅ Real authentication completed for E2E tests');
  return token;
}

export async function logoutUser(page: Page) {
  // Clear auth tokens from localStorage
  await page.evaluate(() => {
    localStorage.removeItem('auth_token');
  });
  
  // Try to click logout button if available
  const logoutButton = page.locator('button:has-text("Log out")');
  if (await logoutButton.isVisible()) {
    await logoutButton.click();
    await page.waitForLoadState('networkidle');
  }
  
  console.log('✅ User logged out');
}

export async function waitForCollectionForm(page: Page) {
  // Wait for collection form to be visible
  await expect(page.getByRole('heading', { name: /create.*collection/i })).toBeVisible();
  await expect(page.getByLabel(/collection name/i)).toBeVisible();
  await expect(page.getByLabel(/collection title/i)).toBeVisible();
}