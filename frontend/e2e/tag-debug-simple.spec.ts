import { test, expect } from '@playwright/test';

test('Debug tag suggestions - Simple version', async ({ page }) => {
  console.log('=== Starting simple tag debug test ===');

  // Go to the frontend
  await page.goto('http://localhost:3001');
  await page.waitForLoadState('networkidle');

  // Test the API endpoint directly first
  console.log('1. Testing API endpoint directly...');
  const apiResponse = await page.request.get('http://localhost:8080/api/v1/tags/search?q=flu&limit=10', {
    headers: {
      'Origin': 'http://localhost:3001',
      'Accept': 'application/json'
    }
  });

  console.log('API Status:', apiResponse.status());
  const apiData = await apiResponse.json();
  console.log('API Response:', JSON.stringify(apiData, null, 2));

  expect(apiResponse.status()).toBe(200);
  expect(apiData.tags).toBeTruthy();
  expect(apiData.tags.length).toBeGreaterThan(0);

  // Take a screenshot
  await page.screenshot({ path: '/tmp/tag-debug-frontend.png', fullPage: true });
  console.log('Screenshot saved to /tmp/tag-debug-frontend.png');

  // Look for any input fields
  console.log('2. Looking for input fields...');
  const inputs = await page.locator('input[type="text"], input[type="search"], input:not([type])').all();
  console.log(`Found ${inputs.length} potential input fields`);

  for (let i = 0; i < inputs.length; i++) {
    const input = inputs[i];
    const placeholder = await input.getAttribute('placeholder') || 'no placeholder';
    const name = await input.getAttribute('name') || 'no name';
    const id = await input.getAttribute('id') || 'no id';
    const classes = await input.getAttribute('class') || 'no classes';
    console.log(`Input ${i}: placeholder="${placeholder}", name="${name}", id="${id}", classes="${classes}"`);
  }

  // Look for any elements that might be tag-related
  console.log('3. Looking for tag-related elements...');
  const tagElements = await page.locator('[class*="tag"], [class*="Tag"], [id*="tag"], [id*="Tag"]').all();
  console.log(`Found ${tagElements.length} tag-related elements`);

  for (let i = 0; i < tagElements.length; i++) {
    const element = tagElements[i];
    const tag = await element.evaluate(el => el.tagName);
    const classes = await element.getAttribute('class') || '';
    const text = await element.textContent() || '';
    console.log(`Tag element ${i}: ${tag}, classes="${classes}", text="${text.substring(0, 50)}"`);
  }

  console.log('=== Test completed ===');
});