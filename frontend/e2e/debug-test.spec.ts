import { test, expect } from '@playwright/test';

test('debug test', async ({ page }) => {
  console.log('Starting test...');
  
  page.on('console', msg => console.log('PAGE LOG:', msg.text()));
  page.on('pageerror', err => console.log('PAGE ERROR:', err.message));
  
  try {
    console.log('About to navigate to homepage...');
    await page.goto('/works', { timeout: 10000 });
    console.log('✅ Successfully navigated to homepage');
    
    const title = await page.title();
    console.log('Page title:', title);
    
    expect(title).toBeTruthy();
  } catch (error) {
    console.error('Navigation failed:', error);
    throw error;
  }
});