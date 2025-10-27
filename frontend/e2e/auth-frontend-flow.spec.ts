import { test, expect } from '@playwright/test';
import { loginUser, logoutUser, registerUser, TEST_USER } from './test-utils';

test.describe('Frontend Authentication Flow', () => {
  test.beforeEach(async ({ page }) => {
    // Start fresh for each test
    await page.goto('/');
    await page.waitForLoadState('networkidle');
  });

  test('should redirect unauthenticated users to login', async ({ page }) => {
    // Try to access a protected route
    await page.goto('/dashboard');
    
    // Should be redirected to login page
    await expect(page).toHaveURL(/\/login/);
    
    // Should see login form
    await expect(page.getByLabel(/email/i)).toBeVisible();
    await expect(page.getByLabel(/password/i)).toBeVisible();
  });

  test('should allow user login via frontend form', async ({ page }) => {
    // Navigate to login page
    await page.goto('/login');
    
    // Fill out login form
    await page.getByLabel(/email/i).fill(TEST_USER.email);
    await page.getByLabel(/password/i).fill(TEST_USER.password);
    
    // Submit form
    await page.getByRole('button', { name: /log.*in|sign.*in|submit/i }).click();
    
    // Should redirect to home page
    await expect(page).toHaveURL('/');
    
    // Should see authenticated navigation elements
    await expect(page.getByText(/dashboard|profile|logout/i).first()).toBeVisible({ timeout: 10000 });
  });

  test('should display error for invalid credentials', async ({ page }) => {
    // Navigate to login page
    await page.goto('/login');
    
    // Fill out form with invalid credentials
    await page.getByLabel(/email/i).fill('invalid@example.com');
    await page.getByLabel(/password/i).fill('wrongpassword');
    
    // Submit form
    await page.getByRole('button', { name: /log.*in|sign.*in|submit/i }).click();
    
    // Should see error message
    await expect(page.getByText(/invalid.*credentials|login.*failed|incorrect/i)).toBeVisible({ timeout: 5000 });
    
    // Should still be on login page
    await expect(page).toHaveURL(/\/login/);
  });

  test('should allow access to protected routes after login', async ({ page }) => {
    // Login using frontend flow
    await loginUser(page);
    
    // Should be able to access protected routes
    await page.goto('/dashboard');
    await expect(page).toHaveURL('/dashboard');
    
    await page.goto('/works/new');
    await expect(page).toHaveURL('/works/new');
    
    // Should see authenticated content
    await expect(page.getByText(/dashboard|create.*work|new.*work/i).first()).toBeVisible();
  });

  test('should logout user via frontend', async ({ page }) => {
    // Login first
    await loginUser(page);
    
    // Should see authenticated state
    await expect(page.getByText(/dashboard|profile|logout/i).first()).toBeVisible();
    
    // Logout via frontend
    await logoutUser(page);
    
    // Should be redirected and see unauthenticated state
    await expect(page).toHaveURL(/\/(login|$)/);
    
    // Should not be able to access protected routes
    await page.goto('/dashboard');
    await expect(page).toHaveURL(/\/login/);
  });

  test('should maintain authentication state across page refreshes', async ({ page }) => {
    // Login using frontend flow
    await loginUser(page);
    
    // Navigate to protected route
    await page.goto('/dashboard');
    await expect(page).toHaveURL('/dashboard');
    
    // Refresh the page
    await page.reload();
    await page.waitForLoadState('networkidle');
    
    // Should still be authenticated and on the protected route
    await expect(page).toHaveURL('/dashboard');
    await expect(page.getByText(/dashboard|profile/i).first()).toBeVisible();
  });

  test('should handle token expiration gracefully', async ({ page }) => {
    // Login first
    await loginUser(page);
    
    // Manually expire the token by setting an old timestamp
    await page.evaluate(() => {
      const expiredToken = 'eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJleHAiOjE2MDAwMDAwMDB9.invalid';
      localStorage.setItem('auth_token', expiredToken);
    });
    
    // Try to access a protected route
    await page.goto('/dashboard');
    
    // Should be redirected to login due to expired token
    await expect(page).toHaveURL(/\/login/);
  });

  test('should work with browser back/forward navigation', async ({ page }) => {
    // Start on home page
    await page.goto('/');
    
    // Go to login page
    await page.goto('/login');
    await expect(page).toHaveURL(/\/login/);
    
    // Login
    await page.getByLabel(/email/i).fill(TEST_USER.email);
    await page.getByLabel(/password/i).fill(TEST_USER.password);
    await page.getByRole('button', { name: /log.*in|sign.*in|submit/i }).click();
    
    // Should be redirected to home
    await expect(page).toHaveURL('/');
    
    // Navigate to dashboard
    await page.goto('/dashboard');
    await expect(page).toHaveURL('/dashboard');
    
    // Use browser back button
    await page.goBack();
    await expect(page).toHaveURL('/');
    
    // Use browser forward button
    await page.goForward();
    await expect(page).toHaveURL('/dashboard');
    
    // Should still be authenticated
    await expect(page.getByText(/dashboard|profile/i).first()).toBeVisible();
  });
});

test.describe('Frontend Registration Flow', () => {
  test('should show registration form', async ({ page }) => {
    await page.goto('/register');
    
    // Should see registration form fields
    await expect(page.getByLabel(/username/i)).toBeVisible();
    await expect(page.getByLabel(/email/i)).toBeVisible();
    await expect(page.getByLabel(/password/i)).toBeVisible();
    await expect(page.getByRole('button', { name: /register|sign.*up|create/i })).toBeVisible();
  });

  test('should validate registration form', async ({ page }) => {
    await page.goto('/register');
    
    // Try to submit empty form
    await page.getByRole('button', { name: /register|sign.*up|create/i }).click();
    
    // Should see validation errors
    await expect(page.getByText(/required|field.*required/i).first()).toBeVisible();
  });

  test('should handle registration for existing user', async ({ page }) => {
    await page.goto('/register');
    
    // Try to register with existing user email
    await page.getByLabel(/username/i).fill('existinguser');
    await page.getByLabel(/email/i).fill(TEST_USER.email);
    await page.getByLabel(/display.*name/i).fill('Existing User');
    
    const passwordFields = page.getByLabel(/password/i);
    await passwordFields.first().fill('newpassword123');
    
    await page.getByRole('button', { name: /register|sign.*up|create/i }).click();
    
    // Should see error about existing user
    await expect(page.getByText(/already.*exists|user.*exists|email.*taken/i)).toBeVisible({ timeout: 5000 });
  });
});